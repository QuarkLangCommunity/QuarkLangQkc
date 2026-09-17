package lang

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ============ 二进制库互调（跨系统） ============
// program library; + pub 导出的符号打包为 .qlib（gob 二进制，与平台无关）。

// ExportLibrary 把 program library 的 pub 符号导出为 .qlib 二进制库。
func ExportLibrary(prog *Program, outPath string) error {
	if prog.Kind != "library" {
		return fmt.Errorf("ExportLibrary: 只有 program library; 才能导出库（当前 Kind=%q）", prog.Kind)
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

// ImportLibrary 读取 .qlib 二进制库，返回导出符号源码文本表。
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

// LoadImport 按编译/运行选项寻找 import：同目录 .qk（源码）或 .qlib（库）。
func LoadImport(dir, path string) (string, error) {
	src, _, err := LoadImportIn([]string{dir}, path)
	return src, err
}

// LoadImportIn 在多个目录中依次查找 import（同目录优先，其次额外搜索路径）。
// 返回源码文本与实际命中的文件路径（.qlib 时文件路径为该库文件）。
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
	return "", "", fmt.Errorf("ImportError: 找不到 %q（搜索目录 %s 下的 %s.qk 或 %s.qlib）",
		path, strings.Join(dirs, ", "), path, path)
}

// stripProgramDecl 去掉导入源中的 program library;/program main; 声明行（库形态由主程序决定）。
func stripProgramDecl(src string) string {
	out, _ := stripProgramDeclMapped(src)
	return out
}

// stripProgramDeclMapped 同 stripProgramDecl，另返回剥离后每行对应的**原文件行号**（1 基）。
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

// ============ 源码位置映射（合并 import 后仍能回到原文件） ============
//
// CompileWithImports 把主文件与各导入库**文本合并**后编译，合并源码的行号与
// 任一原文件都不对应。工具（qkcheck / qklsp）要把诊断指回真实文件，需要这张表。

// SrcMap 记录合并源码每一行的来源：合并第 i 行 ← lines[i-1]。
type SrcMap struct {
	lines []srcLoc
}

type srcLoc struct {
	file string // 源文件路径（主文件为传入的 filename）
	line int    // 该行在源文件中的行号（1 基）
}

// Map 把合并源码的行号映射回（文件, 文件内行号）。行号非法时返回 ("", 0)。
func (m *SrcMap) Map(line int) (string, int) {
	if m == nil || line < 1 || line > len(m.lines) {
		return "", 0
	}
	l := m.lines[line-1]
	return l.file, l.line
}

// mapBuilder 累积合并源码并同时记录每行来源。
type mapBuilder struct {
	b    strings.Builder
	segs []srcLoc
	idx  map[string]int // (文件\x00行号) → 合并源码行号（1 基）
}

// append 追加一段源码（text 末尾的换行由调用方保证与旧实现一致）。
func (m *mapBuilder) append(text, file string) {
	m.appendMapped(text, file, nil)
}

// appendMapped 追加一段源码，srcLines[i] 给出该段第 i 行对应的原文件行号（nil = 顺序一致）。
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

// mergedLine 把（文件, 文件内行号）换算成合并源码行号；未命中返回 0。
func (m *mapBuilder) mergedLine(file string, line int) int {
	return m.idx[file+"\x00"+strconv.Itoa(line)]
}

func (m *mapBuilder) String() string { return m.b.String() }

func (m *mapBuilder) srcMap() *SrcMap { return &SrcMap{lines: m.segs} }

// CompileWithImports 编译源码并解析 import（同目录默认搜索范围；v1 单层）。
func CompileWithImports(src, filename string) (*Program, error) {
	prog, _, err := CompileWithImportsMapped(src, filename)
	return prog, err
}

// CompileWithImportsMapped 同 CompileWithImports，另返回源码位置映射表。
func CompileWithImportsMapped(src, filename string) (*Program, *SrcMap, error) {
	return CompileWithImportPaths(src, filename, nil)
}

// CompileWithImportPaths 同 CompileWithImportsMapped，另支持额外 import 搜索目录
// （工具用：qkcheck -L / qklsp 工作区；语言本身仍以同目录为准，额外目录仅为兜底）。
func CompileWithImportPaths(src, filename string, extraPaths []string) (*Program, *SrcMap, error) {
	dir := "."
	if filename != "" {
		dir = filepath.Dir(filename)
	}
	var merged mapBuilder
	merged.append(src+"\n", filename)
	visited := map[string]bool{}
	// 递归合并：处理主文件与各库文件中的 import（visited 防环）
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

// extractBody 按函数体行区间从源码提取函数体文本（含大括号）。
func extractBody(src string, fn *FuncDecl) string {
	if fn.BodyStart.Line <= 0 {
		return ""
	}
	lines := strings.Split(src, "\n")
	// '{' 的下一行起、'}' 的前一行止（索引为 0 基；单行函数体为空）
	start := fn.BodyStart.Line
	end := fn.BodyEnd.Line - 2
	if start <= 0 || end < start || end >= len(lines) {
		return ""
	}
	return strings.Join(lines[start:end+1], "\n") + "\n"
}
