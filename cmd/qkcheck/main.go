// qkcheck: QuarkLang static checker (toolchain member)
//
// Usage: qkcheck [flags] files...
//
//	-json          Emit diagnostics as JSON (consumed by CI / editors)
//	-params        Also check unused parameters (off by default: interface implementations often have ignored parameters)
//	-no-typecheck  Syntax + static checks only, no type checking
//	-L dir         Append an import search directory (repeatable; same directory takes priority)
//	-exit0         Exit 0 even with diagnostics (report only, never blocks)
//	--lang           Output language zh|en (default Chinese; QK_LANG also works)
//	--version      Print version
//
// Exit codes: 0 = no diagnostics; 1 = errors or warnings; 2 = usage error.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// version is the release version, injected at build time with -ldflags "-X main.version=vX.Y.Z" (see scripts/build-release.sh)
var version = "dev"

type finding struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
	Sev  string `json:"severity"` // error | warning
	Code string `json:"code,omitempty"`
	Msg  string `json:"message"`
}

func (f finding) String() string {
	code := ""
	if f.Code != "" {
		code = " [" + f.Code + "]"
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s%s", f.File, f.Line, f.Col, f.Sev, f.Msg, code)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: qkcheck [-json] [-params] [-no-typecheck] [-L dir] [-exit0] [--version] files...")
	fmt.Fprintln(w, i18n.T("  QuarkLang 静态检查（QK101–QK115）：未使用变量/形参/导入/函数 · 不可达代码 · 遮蔽 · 缺返回 ·"))
	fmt.Fprintln(w, i18n.T("                           接口近失配 · void 误用 · 自赋值 · 常量条件 · 常量除零 · 死存储 · 自身比较"))
}

func run(args []string, stdout, stderr io.Writer) int {
	jsonOut, params, typecheck, exit0 := false, false, true, false
	var files, libDirs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-json":
			jsonOut = true
		case "-params":
			params = true
		case "-no-typecheck":
			typecheck = false
		case "-exit0":
			exit0 = true
		case "-L":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, i18n.T("qkcheck: -L 需要一个目录参数"))
				return 2
			}
			i++
			libDirs = append(libDirs, args[i])
		case "--lang", "-lang":
			if i+1 < len(args) {
				i++
				if l, ok := i18n.ParseLang(args[i]); ok {
					i18n.SetLocale(l)
					lang.SetLocalizer(i18n.New(nil, l)) // interpreter diagnostics follow the same language
				} // zh | en; unknown values keep the current locale
			}
		case "--version", "-V":
			fmt.Fprintln(stdout, "qkcheck", version)
			return 0
		case "-h", "--help":
			usage(stdout)
			return 0
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				fmt.Fprintf(stderr, i18n.T("qkcheck: 未知参数 %s\n"), a)
				usage(stderr)
				return 2
			}
			files = append(files, a)
		}
	}
	if len(files) == 0 {
		usage(stderr)
		return 2
	}

	var all []finding
	files, err := expandArgs(files)
	if err != nil {
		fmt.Fprintln(stderr, "qkcheck:", err)
		return 2
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, i18n.T("qkcheck: 没有可检查的 .qk 文件"))
		return 2
	}
	for _, f := range files {
		fs, code := checkFile(f, checkOptions{params: params, typecheck: typecheck, libDirs: libDirs}, stderr)
		if code != 0 {
			return code
		}
		all = append(all, fs...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		if all[i].Line != all[j].Line {
			return all[i].Line < all[j].Line
		}
		if all[i].Col != all[j].Col {
			return all[i].Col < all[j].Col
		}
		return all[i].Msg < all[j].Msg
	})

	if jsonOut {
		if all == nil {
			all = []finding{}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(all); err != nil {
			fmt.Fprintln(stderr, "qkcheck:", err)
			return 2
		}
	} else {
		for _, f := range all {
			fmt.Fprintln(stdout, f.String())
		}
		if n := len(all); n > 0 {
			errs, warns := 0, 0
			for _, f := range all {
				if f.Sev == "error" {
					errs++
				} else {
					warns++
				}
			}
			fmt.Fprintf(stdout, i18n.T("qkcheck: %d 个问题（%d 错误 / %d 警告）\n"), n, errs, warns)
		}
	}
	if exit0 || len(all) == 0 {
		return 0
	}
	return 1
}

type checkOptions struct {
	params    bool
	typecheck bool
	libDirs   []string
}

func checkFile(path string, opts checkOptions, stderr io.Writer) ([]finding, int) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(stderr, "qkcheck:", err)
		return nil, 2
	}
	src := string(data)
	var out []finding

	// 1) Single-file front end: positions are file line numbers, static checks run in that coordinate space
	prog, perr := lang.ParseSource(src)
	if perr != nil {
		return append(out, compileFinding(path, perr)), 0
	}
	out = append(out, lintFindings(path, lang.Lint(prog, lang.LintOptions{Params: opts.params}))...)

	// 2) Type checking
	//
	// No imports (the vast majority of files): type-check the **same already-parsed AST** directly --
	// positions are already file coordinates, and one full lex+parse is saved (measured at ~30% of single-file checking).
	// With imports: the text must be merged and compiled (library symbols come from the merged source), then SrcMap restores the positions.
	if opts.typecheck {
		if len(prog.Imports) == 0 {
			if terr := lang.Typecheck(prog); terr != nil {
				out = append(out, compileFinding(path, terr))
			}
		} else if _, sm, cerr := lang.CompileWithImportPaths(src, path, opts.libDirs); cerr != nil {
			out = append(out, mappedCompileFinding(path, sm, cerr))
		}
	}
	return out, 0
}

func lintFindings(file string, diags []lang.Diag) []finding {
	out := make([]finding, 0, len(diags))
	for _, d := range diags {
		out = append(out, finding{
			File: file, Line: d.Pos.Line, Col: d.Pos.Col,
			Sev: d.Sev, Code: d.Code, Msg: d.Msg,
		})
	}
	return out
}

// compileFinding turns a single-file front-end error into a diagnostic (the position is already a file line; "at line N" in the message is carried by the position field).
func compileFinding(file string, err error) finding {
	return finding{
		File: file, Line: errLine(err), Col: errCol(err), Sev: "error",
		Msg: lineSuffix.ReplaceAllString(err.Error(), ""),
	}
}

var lineSuffix = regexp.MustCompile(` at line \d+$`)

// mappedCompileFinding maps merged-compilation errors back to original file lines.
func mappedCompileFinding(mainFile string, sm *lang.SrcMap, err error) finding {
	line, col := errLine(err), errCol(err)
	file, fline := mainFile, line
	if sm != nil {
		if f, l := sm.Map(line); f != "" {
			file, fline = f, l
		}
	}
	msg := lineSuffix.ReplaceAllString(err.Error(), "")
	return finding{File: file, Line: fline, Col: col, Sev: "error", Msg: msg}
}

// errLine/errCol extract the line and column of each kind of front-end error (0 = unknown).
func errLine(err error) int {
	switch e := err.(type) {
	case *lang.LexError:
		return e.Line
	case *lang.ParseError:
		return e.Line
	case *lang.CheckError:
		return e.Pos.Line
	case *lang.RunError:
		return e.Pos.Line
	}
	if m := regexp.MustCompile(`at line (\d+)`).FindStringSubmatch(err.Error()); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func errCol(err error) int {
	switch e := err.(type) {
	case *lang.LexError:
		return e.Col
	case *lang.ParseError:
		return e.Col
	case *lang.CheckError:
		return e.Pos.Col
	case *lang.RunError:
		return e.Pos.Col
	}
	return 0
}

// expandArgs expands directory arguments into the .qk files they contain (sorted by path); other arguments pass through unchanged.
func expandArgs(args []string) ([]string, error) {
	var out []string
	for _, a := range args {
		info, err := os.Stat(a)
		if err != nil || !info.IsDir() {
			out = append(out, a)
			continue
		}
		var found []string
		err = filepath.Walk(a, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fi.IsDir() && (strings.HasSuffix(p, ".qk") || strings.HasSuffix(p, ".kq")) {
				found = append(found, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		sort.Strings(found)
		out = append(out, found...)
	}
	return out, nil
}
