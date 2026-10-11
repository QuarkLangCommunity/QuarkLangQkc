// qkcheck: QuarkLang static checker (toolchain member)
//
// Usage: qkcheck [flags] files...
//
//	-json          Emit diagnostics as JSON (consumed by CI / editors)
//	-params        Also check unused parameters (off by default: interface implementations often have ignored parameters)
//	-no-typecheck  Syntax + static checks only, no type checking
//	-L dir         Append an import search directory (repeatable; same directory takes priority)
//	-exit0         Exit 0 even with diagnostics (report only, never blocks)
//	-fix           Apply every mechanical correction in place, then re-check and report what remains
//	-check         Write nothing and exit non-zero when a correction is pending (the CI form; -dry-run is an alias)
//	--lang         Output language zh|en (QK_LANG, then LC_ALL/LC_MESSAGES/LANG; English if none is set)
//	--version      Print version
//
// A diagnostic that has a mechanical correction prints it: the corrected line, not only what is
// wrong. The corrections and the reasons a diagnostic has none are inventoried in README.md.
//
// Exit codes: 0 = no diagnostics; 1 = errors or warnings; 2 = usage error; 3 = -check found a pending
// correction and wrote nothing.
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

// exitFixPending is the -check exit code: corrections are pending and nothing was written.
const exitFixPending = 3

type finding struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
	Sev  string `json:"severity"` // error | warning
	Code string `json:"code,omitempty"`
	Msg  string `json:"message"`
	// Fix is the mechanical correction, when this diagnostic has one: the corrected line, how it is
	// applied, and why it is right. A diagnostic without a fix is one whose correction needs a human
	// decision (see README.md).
	Fix *lang.Fix `json:"fix,omitempty"`
	// NoFix says why no correction could be applied to a diagnostic that is mechanically fixable in
	// principle (for example a declaration whose initializer is a dead store).
	NoFix string `json:"no_fix,omitempty"`
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
	fmt.Fprintln(w, msgID(i18n.MsgCliCheckUsage))
	fmt.Fprintln(w, msgID(i18n.MsgCliQuarklangStaticAnalysisQk101Qk115Unused))
	fmt.Fprintln(w, msgID(i18n.MsgCliInterfaceNearMissVoidMisuseSelf))
}

func run(args []string, stdout, stderr io.Writer) int {
	jsonOut, params, typecheck, exit0, fixMode, checkMode := false, false, true, false, false, false
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
		case "-fix", "--fix":
			fixMode = true
		case "-check", "--check", "-dry-run", "--dry-run":
			checkMode = true
		case "-L":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, msgID(i18n.MsgCliCheckLNeedsDirectory))
				return 2
			}
			i++
			libDirs = append(libDirs, args[i])
		case "--lang", "-lang":
			if i+1 < len(args) {
				i++
				if l, ok := i18n.ParseLang(args[i]); ok {
					i18n.SetLocale(l)
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
				fmt.Fprintf(stderr, msgID(i18n.MsgCliCheckUnknownArgument), a)
				usage(stderr)
				return 2
			}
			files = append(files, a)
		}
	}
	// -fix applies, -check only reports what -fix would do: asking for both is a usage error rather
	// than a silent choice between them.
	if fixMode && checkMode {
		fmt.Fprint(stderr, msgID(i18n.MsgCliCheckFixFlagConflict))
		return 2
	}
	if len(files) == 0 {
		usage(stderr)
		return 2
	}

	files, err := expandArgs(files)
	if err != nil {
		fmt.Fprintln(stderr, "qkcheck:", err)
		return 2
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, msgID(i18n.MsgCliCheckNoFiles))
		return 2
	}

	opts := checkOptions{params: params, typecheck: typecheck, libDirs: libDirs}
	var all []finding
	pending, wrote := 0, false
	for _, f := range files {
		var fs []finding
		var code int
		if fixMode {
			fs, code = fixFile(f, opts, stdout, stderr)
			if code != 0 {
				return code
			}
		} else {
			fs, code = checkFile(f, opts, stderr)
			if code != 0 {
				return code
			}
		}
		for _, x := range fs {
			if x.Fix != nil {
				pending++
			}
		}
		all = append(all, fs...)
	}
	sortFindings(all)

	report(all, jsonOut, checkMode, stdout)

	if len(all) == 0 {
		return 0
	}
	if checkMode && pending > 0 {
		fmt.Fprintf(stdout, msgID(i18n.MsgCliCheckFixPending), pending)
		return exitFixPending
	}
	if exit0 {
		return 0
	}
	if !jsonOut {
		printSummary(stdout, all, wrote)
	}
	return 1
}

// sortFindings puts the findings of every file in report order (file, then position, then message).
func sortFindings(all []finding) {
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
}

// report prints the findings: as JSON, or line by line with the proposed correction under each one
// that has a correction, and the reason under each that has none but is mechanically fixable.
// checkMode only adds the note that nothing was written.
func report(all []finding, jsonOut, checkMode bool, stdout io.Writer) {
	if jsonOut {
		if all == nil {
			all = []finding{}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(all); err != nil {
			fmt.Fprintln(stdout, "qkcheck:", err)
		}
		return
	}
	for _, f := range all {
		fmt.Fprintln(stdout, f.String())
		switch {
		case f.Fix != nil && f.Fix.Delete:
			fmt.Fprintf(stdout, msgID(i18n.MsgCliCheckFixDelete), f.Fix.Why)
		case f.Fix != nil:
			fmt.Fprintf(stdout, msgID(i18n.MsgCliCheckFixReplace), f.Fix.Replacement(), f.Fix.Why)
		case f.NoFix != "":
			fmt.Fprintf(stdout, msgID(i18n.MsgCliCheckFixSkipped), f.NoFix)
		}
		if checkMode && f.Fix != nil {
			fmt.Fprint(stdout, msgID(i18n.MsgCliCheckFixFlagged))
		}
	}
}

// printSummary prints the counts line, and - after a run that rewrote something - how many findings no
// fix applies to. Nothing is ever dropped silently: those findings were printed above.
func printSummary(stdout io.Writer, all []finding, wrote bool) {
	errs, warns, unfixed := 0, 0, 0
	for _, f := range all {
		if f.Sev == "error" {
			errs++
		} else {
			warns++
		}
		if f.Fix == nil {
			unfixed++
		}
	}
	fmt.Fprintf(stdout, msgID(i18n.MsgCliCheckSummary), len(all), errs, warns)
	if wrote && unfixed > 0 {
		fmt.Fprintf(stdout, msgID(i18n.MsgCliCheckFixRemaining), unfixed)
	}
}

type checkOptions struct {
	params    bool
	typecheck bool
	libDirs   []string
}

// checkFile checks one file and returns its findings. A correction is attached to every finding that
// has one; a finding that is mechanically fixable in principle but cannot be corrected in this
// position carries the reason instead (NoFix), so a -fix run can never silently drop it.
func checkFile(path string, opts checkOptions, stderr io.Writer) ([]finding, int) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(stderr, "qkcheck:", err)
		return nil, 2
	}
	src := string(data)
	return checkSource(path, src, opts), 0
}

// checkSource is the check itself, over source the caller already holds.
func checkSource(path, src string, opts checkOptions) []finding {
	var out []finding

	// 1) Single-file front end: positions are file line numbers, static checks run in that coordinate space.
	// The correction is computed against src, the text the author wrote and a -fix run rewrites.
	prog, perr := lang.ParseSource(src)
	if perr != nil {
		return append(out, compileFinding(path, perr))
	}
	lintOpts := lang.LintOptions{Params: opts.params, File: path, LibDirs: opts.libDirs, OrigSrc: src}
	res := lang.LintReport(prog, lintOpts)
	out = append(out, lintFindings(path, res.Diags, res.Plan)...)

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
	return out
}

// fixFile applies every correction the checks propose to path, then checks the result again and
// returns what remains. It repeats while a correction still applies, because one correction can expose
// another (deleting `x = x;` between two writes can make the write it carried dead as well); the loop
// is bounded, and reaching the bound is reported rather than looped on.
func fixFile(path string, opts checkOptions, stdout, stderr io.Writer) ([]finding, int) {
	const maxPasses = 8
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(stderr, "qkcheck:", err)
		return nil, 2
	}
	src, lines := string(data), 0
	for pass := 0; pass < maxPasses; pass++ {
		fixed, n := fixOnce(path, src, opts)
		if n == 0 {
			break
		}
		if fixed == src {
			break // a pass that changes nothing must not become a loop
		}
		src, lines = fixed, lines+n
	}
	final := checkSource(path, src, opts)
	if lines > 0 {
		// Write only when something changed, and keep the file's own permissions: -fix rewrites the
		// bytes the checks rewrote, nothing else.
		if err := writePreservingMode(path, src); err != nil {
			fmt.Fprintln(stderr, "qkcheck:", err)
			return nil, 2
		}
		fmt.Fprintf(stdout, msgID(i18n.MsgCliCheckFixWrote), path, lines)
	}
	return final, 0
}

// fixOnce computes the corrections of one pass over src: the corrected source and how many lines it
// rewrote. The findings of that pass are not reported - the caller re-checks the result and reports
// that, which is the report the user's file now deserves.
func fixOnce(path, src string, opts checkOptions) (string, int) {
	prog, perr := lang.ParseSource(src)
	if perr != nil {
		return src, 0 // a file that does not parse has no mechanical correction
	}
	res := lang.LintReport(prog, lang.LintOptions{
		Params: opts.params, File: path, LibDirs: opts.libDirs, OrigSrc: src,
	})
	if res.Plan == nil || res.Plan.Pending() == 0 {
		return src, 0
	}
	return res.Fixed, res.Plan.Pending()
}

// writePreservingMode rewrites a file's bytes without changing its permissions.
func writePreservingMode(path, src string) error {
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	return os.WriteFile(path, []byte(src), mode)
}

// lintFindings turns the checker's diagnostics into findings, carrying each correction and the reason
// a diagnostic that could have had one does not (plan.Skip). A diagnostic is never dropped by the
// conversion: a missing correction adds NoFix to it, it does not remove it.
func lintFindings(file string, diags []lang.Diag, plan *lang.FixPlan) []finding {
	out := make([]finding, 0, len(diags))
	for _, d := range diags {
		out = append(out, finding{
			File: file, Line: d.Pos.Line, Col: d.Pos.Col,
			Sev: d.Sev, Code: d.Code, Msg: d.Msg, Fix: d.Fix,
		})
	}
	if plan == nil {
		return out
	}
	// Match a skip to its diagnostic by code and line: SkipReason carries no column, and two
	// diagnostics of one code on one line is not a shape the linter produces.
	for _, sk := range plan.Skip {
		for i := range out {
			if out[i].Code == sk.Code && out[i].Line == sk.Line && out[i].Fix == nil && out[i].NoFix == "" {
				out[i].NoFix = sk.Reason
				break
			}
		}
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
