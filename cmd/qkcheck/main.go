// qkcheck：QuarkLang 静态检查器（工具链成员）
//
// 用法: qkcheck [flags] files...
//
//	-json          以 JSON 输出诊断（CI / 编辑器消费）
//	-params        同时检查未使用形参（默认关闭：接口实现常有忽略的形参）
//	-no-typecheck  只做语法 + 静态检查，不做类型检查
//	-L dir         追加 import 搜索目录（可重复；同目录优先）
//	-exit0         有诊断也退出 0（只报告，不阻断）
//	--version      打印版本
//
// 退出码：0 = 无诊断；1 = 有错误或警告；2 = 用法错误。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"quarklang/internal/lang"
)

// version 发布版本：构建时用 -ldflags "-X main.version=vX.Y.Z" 注入（见 scripts/build-release.sh）
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
	fmt.Fprintln(w, "  QuarkLang 静态检查（QK101–QK115）：未使用变量/形参/导入/函数 · 不可达代码 · 遮蔽 · 缺返回 ·")
	fmt.Fprintln(w, "                           接口近失配 · void 误用 · 自赋值 · 常量条件 · 常量除零 · 死存储 · 自身比较")
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
				fmt.Fprintln(stderr, "qkcheck: -L 需要一个目录参数")
				return 2
			}
			i++
			libDirs = append(libDirs, args[i])
		case "--version", "-V":
			fmt.Fprintln(stdout, "qkcheck", version)
			return 0
		case "-h", "--help":
			usage(stdout)
			return 0
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				fmt.Fprintf(stderr, "qkcheck: 未知参数 %s\n", a)
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
		fmt.Fprintln(stderr, "qkcheck: 没有可检查的 .qk 文件")
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
			fmt.Fprintf(stdout, "qkcheck: %d 个问题（%d 错误 / %d 警告）\n", n, errs, warns)
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

	// 1) 单文件前端：位置即文件行号，静态检查在此坐标下进行
	prog, perr := lang.ParseSource(src)
	if perr != nil {
		return append(out, compileFinding(path, perr)), 0
	}
	out = append(out, lintFindings(path, lang.Lint(prog, lang.LintOptions{Params: opts.params}))...)

	// 2) 类型检查：合并 import 后编译，再用源码映射把位置还原到真实文件
	if opts.typecheck {
		_, sm, cerr := lang.CompileWithImportPaths(src, path, opts.libDirs)
		if cerr != nil {
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

// compileFinding 把单文件前端错误转成诊断（位置已是文件行号；消息里的「at line N」由位置字段承载）。
func compileFinding(file string, err error) finding {
	return finding{
		File: file, Line: errLine(err), Col: errCol(err), Sev: "error",
		Msg: lineSuffix.ReplaceAllString(err.Error(), ""),
	}
}

var lineSuffix = regexp.MustCompile(` at line \d+$`)

// mappedCompileFinding 把合并编译错误映射回原文件行号。
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

// errLine/errCol 提取各类前端错误的行号列号（0 = 未知）。
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

// expandArgs 展开目录参数为其中的 .qk 文件（按路径排序），其余原样返回。
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
