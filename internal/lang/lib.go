package lang

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ============ Binary library interop (cross-platform) ============
// program library; + pub-exported symbols are packaged as .qlib (gob binary, platform-independent).

// ExportLibrary exports a program library's pub symbols as a .qlib binary library.
func ExportLibrary(prog *Program, outPath string) error {
	if prog.Kind != "library" {
		return errors.New(msg("ExportLibrary: 只有 program library; 才能导出库（当前 Kind=%q）", prog.Kind))
	}
	syms := map[string]string{}
	for _, fn := range prog.Funcs {
		if !containsStr(prog.Pub, fn.Name) {
			continue
		}
		body := extractBody(prog.Src, fn)
		params := make([]string, 0, len(fn.Params))
		for _, p := range fn.Params {
			params = append(params, p.Type+" "+p.Name) // canonical form: type first
		}
		var sb strings.Builder
		sb.WriteString("fn ")
		if len(fn.TypeParams) > 0 {
			sb.WriteString("<" + strings.Join(fn.TypeParams, ", ") + "> ")
		}
		sb.WriteString(fn.Name)
		sb.WriteString("(" + strings.Join(params, ", ") + ")")
		if fn.Ret != "" {
			sb.WriteString(" " + fn.Ret)
		}
		sb.WriteString(" {" + "\n")
		sb.WriteString(body)
		sb.WriteString("}" + "\n")
		syms[fn.Name] = sb.String()
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(syms); err != nil {
		return err
	}
	return os.WriteFile(outPath, buf.Bytes(), 0o644)
}

// ImportLibrary reads a .qlib binary library and returns the table of exported symbol source text.
func ImportLibrary(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var syms map[string]string
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&syms); err != nil {
		return nil, err
	}
	return syms, nil
}

// LoadImport finds an import: a same-directory .qk (source) or .qlib (library), then the standard
// library. The stdlib location is discovered by walking up from the importing file for a `stdlib`
// directory, or from QK_STDLIB when set, so `import "vec";` works from anywhere inside a checkout —
// the standard library is ordinary library code, not a special case in the resolver.
func LoadImport(dir, path string) (string, error) {
	dirs := []string{dir}
	if std := stdlibDir(dir); std != "" {
		dirs = append(dirs, std)
	}
	src, _, err := LoadImportIn(dirs, path)
	return src, err
}

// stdlibDir locates the standard library for a program directory: QK_STDLIB wins, otherwise the
// nearest ancestor holding a stdlib/vec.qk (the canonical entry point of the collection library).
func stdlibDir(dir string) string {
	if env := os.Getenv("QK_STDLIB"); env != "" {
		return env
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		cand := filepath.Join(abs, "stdlib")
		if _, err := os.Stat(filepath.Join(cand, "vec.qk")); err == nil {
			return cand
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

// LoadImportIn searches several directories in turn for an import (same directory first, then extra search paths).
// It returns the source text and the path actually hit (for a .qlib, the path is that library file).
func LoadImportIn(dirs []string, path string) (string, string, error) {
	srcName := path
	if !strings.HasSuffix(srcName, ".qk") {
		srcName += ".qk"
	}
	for _, dir := range dirs {
		p := srcName
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		if b, err := os.ReadFile(p); err == nil {
			return string(b), p, nil
		}
	}
	libName := path
	if !strings.HasSuffix(libName, ".qlib") {
		libName += ".qlib"
	}
	for _, dir := range dirs {
		p := libName
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		syms, err := ImportLibrary(p)
		if err != nil {
			continue
		}
		var sb strings.Builder
		for _, src := range syms {
			sb.WriteString(src)
			sb.WriteString("\n")
		}
		return sb.String(), p, nil
	}
	return "", "", errors.New(msg("ImportError: 找不到 %q（搜索目录 %s 下的 %s.qk 或 %s.qlib）",
		path, strings.Join(dirs, ", "), path, path))
}

// stripProgramDecl removes program library;/program main; declaration lines from imported source (the library form is decided by the main program).
func stripProgramDecl(src string) string {
	out, _ := stripProgramDeclMapped(src)
	return out
}

// stripProgramDeclMapped is stripProgramDecl plus the **original line number** (1-based) each stripped line maps to.
func stripProgramDeclMapped(src string) (string, []int) {
	var sb strings.Builder
	var lines []int
	for i, ln := range strings.Split(src, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "program ") && strings.HasSuffix(t, ";") {
			continue
		}
		sb.WriteString(ln)
		sb.WriteString("\n")
		lines = append(lines, i+1)
	}
	return sb.String(), lines
}

// ============ Source position mapping (still resolvable after imports are merged) ============
//
// CompileWithImports compiles the main file and its imports after **merging their text**; the merged source's line numbers
// match no original file. Tools (qkcheck / qklsp) need this table to point diagnostics back at real files.

// SrcMap records where each line of the merged source came from: merged line i ← lines[i-1].
type SrcMap struct {
	lines []srcLoc
}

type srcLoc struct {
	file string // source file path (for the main file, the filename passed in)
	line int    // line number of this line in the source file (1-based)
}

// Map maps a merged-source line number back to (file, line inside that file). Invalid line numbers return ("", 0).
func (m *SrcMap) Map(line int) (string, int) {
	if m == nil || line < 1 || line > len(m.lines) {
		return "", 0
	}
	l := m.lines[line-1]
	return l.file, l.line
}

// mapBuilder accumulates the merged source while recording each line's origin.
type mapBuilder struct {
	b    strings.Builder
	segs []srcLoc
	idx  map[string]int // (file\x00line number) → merged source line number (1-based)
}

// append appends a chunk of source (the caller guarantees the trailing newline matches the old behaviour).
func (m *mapBuilder) append(text, file string) {
	m.appendMapped(text, file, nil)
}

// appendMapped appends a chunk of source; srcLines[i] gives the original line number of that chunk's i-th line (nil = sequential).
func (m *mapBuilder) appendMapped(text, file string, srcLines []int) {
	if m.idx == nil {
		m.idx = map[string]int{}
	}
	m.b.WriteString(text)
	lines := strings.Split(text, "\n")
	for i, ln := range lines {
		if i == len(lines)-1 && ln == "" {
			break // the trailing newline does not produce a new line
		}
		srcLine := i + 1
		if srcLines != nil && i < len(srcLines) {
			srcLine = srcLines[i]
		}
		m.segs = append(m.segs, srcLoc{file: file, line: srcLine})
		key := file + "\x00" + strconv.Itoa(srcLine)
		if _, ok := m.idx[key]; !ok { // keep the first occurrence when the same file is imported repeatedly
			m.idx[key] = len(m.segs)
		}
	}
}

// mergedLine converts (file, line inside that file) into a merged-source line number; 0 when it does not match.
func (m *mapBuilder) mergedLine(file string, line int) int {
	return m.idx[file+"\x00"+strconv.Itoa(line)]
}

func (m *mapBuilder) String() string { return m.b.String() }

// lineCount is the number of merged-source lines written so far (a token line is shifted by it).
func (m *mapBuilder) lineCount() int { return len(m.segs) }

func (m *mapBuilder) srcMap() *SrcMap { return &SrcMap{lines: m.segs} }

// CompileWithImports compiles source and resolves imports (the same directory is searched by default; v1 is single-level).
func CompileWithImports(src, filename string) (*Program, error) {
	prog, _, err := CompileWithImportsMapped(src, filename)
	return prog, err
}

// CompileWithImportsMapped is CompileWithImports plus the source position mapping table.
func CompileWithImportsMapped(src, filename string) (*Program, *SrcMap, error) {
	return CompileWithImportPaths(src, filename, nil)
}

// CompileWithImportPaths is CompileWithImportsMapped plus extra import search directories
// (for tools: qkcheck -L / the qklsp workspace; the language itself still resolves relative to the same directory, the extra directories are only a fallback).
func CompileWithImportPaths(src, filename string, extraPaths []string) (*Program, *SrcMap, error) {
	return compileWithImportPaths(src, filename, extraPaths, nil)
}

// CompileTokensWithImports compiles a main file that arrives as an **already macro-expanded token stream**
// together with the files it imports; src is the source that stream was lexed from and is kept for
// diagnostics only.
//
// It exists so that the compiler can parse the very tokens its macro step produced instead of printing
// them back to source text. A token carries a literal's **value**, not its spelling — a string token holds
// the decoded bytes, quotes excluded (qkparser/token.go, lexString) — so a token→text→token round trip
// rewrites `io.println("x")` into `io.println(x)`. The interpreter has always parsed the stream
// (compileTokens); this entry point is how the compiler reaches the same tail.
func CompileTokensWithImports(toks []Token, src, filename string) (*Program, error) {
	prog, _, err := compileWithImportPaths(src, filename, nil, toks)
	return prog, err
}

// compileWithImportPaths is the single import-merge implementation: it walks the main file and every file
// it imports (that file's directory first, then extraPaths, then the standard library), appends each file's
// text to the merged source used for positions, and — when mainToks is non-nil — each file's tokens to a
// merged stream that compileTokens parses instead of the merged text being lexed again.
// mainToks must already have gone through the macro step (the compiler's caller does that for the main file).
func compileWithImportPaths(src, filename string, extraPaths []string, mainToks []Token) (*Program, *SrcMap, error) {
	dir := "."
	if filename != "" {
		dir = filepath.Dir(filename)
	}
	var merged mapBuilder
	merged.append(src+"\n", filename)
	// Token mode: the caller expanded the main file's macros already, so the merged stream replaces the
	// re-lex of the merged text. Text mode (mainToks == nil) is what the tools and the interpreter use.
	tokenMode := mainToks != nil
	var acc []Token
	if tokenMode {
		acc = append(acc, streamIn(mainToks, nil, 0)...)
	}
	visited := map[string]bool{}
	// Recursive merge: handle imports in the main file and in each library file (visited guards against cycles)
	var collect func(text, base, file string, toks []Token) error
	collect = func(text, base, file string, toks []Token) error {
		if toks == nil {
			var err error
			toks, err = lexExpanded(text)
			if err != nil {
				return err
			}
		}
		prog, err := Parse(toks)
		if err != nil {
			return err
		}
		for i, imp := range prog.Imports {
			base0 := base
			if base0 == "" {
				base0 = dir
			}
			dirs := append([]string{base0}, extraPaths...)
			if std := stdlibDir(base0); std != "" {
				dirs = append(dirs, std) // the standard library is importable from anywhere in a checkout
			}
			key := imp
			if filepath.IsAbs(imp) {
				key = imp
			} else {
				key = base0 + "|" + imp
			}
			if visited[key] {
				continue
			}
			visited[key] = true
			imported, impFile, err := LoadImportIn(dirs, imp)
			if err != nil {
				pos := Pos{}
				if i < len(prog.ImportPos) {
					pos = prog.ImportPos[i]
					pos.Line = merged.mergedLine(file, pos.Line) // convert to merged-source coordinates, so the CLI can map back to the real file
				}
				return &CheckError{Msg: err.Error(), Pos: pos}
			}
			stripped, srcLines := stripProgramDeclMapped(imported)
			lineOff := merged.lineCount()
			merged.appendMapped(stripped+"\n", impFile, srcLines)
			var itoks []Token
			if tokenMode {
				if itoks, err = lexExpanded(imported); err != nil {
					return err
				}
				acc = append(acc, streamIn(itoks, srcLineMap(srcLines), lineOff)...)
			}
			if err := collect(imported, base0, impFile, itoks); err != nil {
				return err
			}
		}
		return nil
	}
	if err := collect(src, dir, filename, mainToks); err != nil {
		return nil, merged.srcMap(), err
	}
	if !tokenMode {
		prog, err := Compile(merged.String())
		if err != nil {
			return nil, merged.srcMap(), err
		}
		return prog, merged.srcMap(), nil
	}
	// One end-of-file marker closes the merged stream, at the position the merged text's lexer would give it.
	mergedToks := append(acc, Token{Kind: TEOF, Line: merged.lineCount() + 1, Col: 1})
	prog, err := compileTokens(mergedToks, merged.String())
	if err != nil {
		return nil, merged.srcMap(), err
	}
	return prog, merged.srcMap(), nil
}

// lexExpanded lexes source and applies the macro step the interpreter uses: split the definitions out and
// expand every call in the explain state ("explain" also fires #when (run) branches).
func lexExpanded(text string) ([]Token, error) {
	toks, err := Lex(text)
	if err != nil {
		return nil, err
	}
	macros, rest, err := SplitMacroDefs(toks)
	if err != nil {
		return nil, err
	}
	if len(macros) == 0 {
		return rest, nil
	}
	return ExpandMacros(rest, macros, "explain")
}

// streamIn copies a file's tokens into merged-source coordinates: the end-of-file marker is dropped (the
// merged stream carries exactly one, at its end), a line missing from lines is dropped as well (the text
// merge strips those lines too), and the kept lines are renumbered by off via lines.
// lines is nil for the main file, whose text is appended unshifted.
func streamIn(toks []Token, lines map[int]int, off int) []Token {
	out := make([]Token, 0, len(toks))
	for _, t := range toks {
		if t.Kind == TEOF {
			continue
		}
		if lines != nil {
			ln, ok := lines[t.Line]
			if t.Line == 0 {
				t.Line = off + 1 // a token the expander synthesised without a source line has none to map
			} else if !ok {
				continue // a line the text merge dropped (a program declaration): absent from the merged source
			} else {
				t.Line = off + ln
			}
		} else if off != 0 {
			t.Line += off
		}
		out = append(out, t)
	}
	return out
}

// srcLineMap maps each kept line of a stripped import to its line number in the merged source (srcLines
// holds the original line number of each kept line, in order).
func srcLineMap(srcLines []int) map[int]int {
	m := make(map[int]int, len(srcLines))
	for i, ln := range srcLines {
		m[ln] = i + 1
	}
	return m
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// extractBody extracts a function body's text (braces included) from the source by line range.
func extractBody(src string, fn *FuncDecl) string {
	if fn.BodyStart.Line <= 0 {
		return ""
	}
	lines := strings.Split(src, "\n")
	// From the line after '{' to the line before '}' (0-based indices; a single-line body is empty)
	start := fn.BodyStart.Line
	end := fn.BodyEnd.Line - 2
	if start <= 0 || end < start || end >= len(lines) {
		return ""
	}
	return strings.Join(lines[start:end+1], "\n") + "\n"
}
