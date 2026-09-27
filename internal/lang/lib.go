package lang

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"quarklang/internal/i18n"
	"strconv"
	"strings"
)

// ============ Binary library interop (cross-platform) ============
// program library; + pub-exported symbols are packaged as .qlib (gob binary, platform-independent).

// ExportLibrary exports a program library's pub symbols as a .qlib binary library.
func ExportLibrary(prog *Program, outPath string) error {
	if prog.Kind != "library" {
		return errors.New(i18n.T("ExportLibrary: 只有 program library; 才能导出库（当前 Kind=%q）", prog.Kind))
	}
	syms := map[string]string{}
	for _, fn := range prog.Funcs {
		if !containsStr(prog.Pub, fn.Name) {
			continue
		}
		body := extractBody(prog.Src, fn)
		params := make([]string, 0, len(fn.Params))
		for _, p := range fn.Params {
			params = append(params, p.Type+" "+p.Name) // 正典：类型在前
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

// LoadImport finds an import by compile/run options: a same-directory .qk (source) or .qlib (library).
func LoadImport(dir, path string) (string, error) {
	src, _, err := LoadImportIn([]string{dir}, path)
	return src, err
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
	return "", "", errors.New(i18n.T("ImportError: 找不到 %q（搜索目录 %s 下的 %s.qk 或 %s.qlib）",
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
	file string // 源文件路径（主文件为传入的 filename）
	line int    // 该行在源文件中的行号（1 基）
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
	idx  map[string]int // (文件\x00行号) → 合并源码行号（1 基）
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
			break // 末尾换行不产生新行
		}
		srcLine := i + 1
		if srcLines != nil && i < len(srcLines) {
			srcLine = srcLines[i]
		}
		m.segs = append(m.segs, srcLoc{file: file, line: srcLine})
		key := file + "\x00" + strconv.Itoa(srcLine)
		if _, ok := m.idx[key]; !ok { // 同名文件重复导入时保留首次
			m.idx[key] = len(m.segs)
		}
	}
}

// mergedLine converts (file, line inside that file) into a merged-source line number; 0 when it does not match.
func (m *mapBuilder) mergedLine(file string, line int) int {
	return m.idx[file+"\x00"+strconv.Itoa(line)]
}

func (m *mapBuilder) String() string { return m.b.String() }

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
	dir := "."
	if filename != "" {
		dir = filepath.Dir(filename)
	}
	var merged mapBuilder
	merged.append(src+"\n", filename)
	visited := map[string]bool{}
	// Recursive merge: handle imports in the main file and in each library file (visited guards against cycles)
	var collect func(text, base, file string) error
	collect = func(text, base, file string) error {
		toks, err := Lex(text)
		if err != nil {
			return err
		}
		macros, rest, err := SplitMacroDefs(toks)
		if err != nil {
			return err
		}
		if len(macros) > 0 {
			rest, err = ExpandMacros(rest, macros, "explain")
			if err != nil {
				return err
			}
		}
		prog, err := Parse(rest)
		if err != nil {
			return err
		}
		for i, imp := range prog.Imports {
			base0 := base
			if base0 == "" {
				base0 = dir
			}
			dirs := append([]string{base0}, extraPaths...)
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
					pos.Line = merged.mergedLine(file, pos.Line) // 换算到合并源码坐标，供 CLI 反查真实文件
				}
				return &CheckError{Msg: err.Error(), Pos: pos}
			}
			stripped, srcLines := stripProgramDeclMapped(imported)
			merged.appendMapped(stripped+"\n", impFile, srcLines)
			if err := collect(imported, base0, impFile); err != nil {
				return err
			}
		}
		return nil
	}
	if err := collect(src, dir, filename); err != nil {
		return nil, merged.srcMap(), err
	}
	prog, err := Compile(merged.String())
	if err != nil {
		return nil, merged.srcMap(), err
	}
	return prog, merged.srcMap(), nil
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
