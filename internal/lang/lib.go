package lang

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
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
	srcPath := path
	if !strings.HasSuffix(srcPath, ".qk") {
		srcPath += ".qk"
	}
	if !filepath.IsAbs(srcPath) {
		srcPath = filepath.Join(dir, srcPath)
	}
	if b, err := os.ReadFile(srcPath); err == nil {
		return string(b), nil
	}
	libPath := path
	if !strings.HasSuffix(libPath, ".qlib") {
		libPath += ".qlib"
	}
	if !filepath.IsAbs(libPath) {
		libPath = filepath.Join(dir, libPath)
	}
	syms, err := ImportLibrary(libPath)
	if err != nil {
		return "", fmt.Errorf("ImportError: 找不到 %q（同目录 %s.qk 或 %s.qlib）", path, path, path)
	}
	var sb strings.Builder
	for _, src := range syms {
		sb.WriteString(src)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// CompileWithImports 编译源码并解析 import（同目录默认搜索范围；v1 单层）。
// stripProgramDecl 去掉导入源中的 program library;/program main; 声明行（库形态由主程序决定）。
func stripProgramDecl(src string) string {
	var sb strings.Builder
	for _, ln := range strings.Split(src, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "program ") && strings.HasSuffix(t, ";") {
			continue
		}
		sb.WriteString(ln)
		sb.WriteString("\n")
	}
	return sb.String()
}

func CompileWithImports(src, filename string) (*Program, error) {
	dir := "."
	if filename != "" {
		dir = filepath.Dir(filename)
	}
	var merged strings.Builder
	merged.WriteString(src)
	merged.WriteString("\n")
	visited := map[string]bool{}
	// 递归合并：处理主文件与各库文件中的 import（visited 防环）
	var collect func(text, base string) error
	collect = func(text, base string) error {
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
		for _, imp := range prog.Imports {
			key := base + "|" + imp
			if visited[key] {
				continue
			}
			visited[key] = true
			imported, err := LoadImport(base, imp)
			if err != nil {
				return err
			}
			merged.WriteString(stripProgramDecl(imported))
			merged.WriteString("\n")
			if err := collect(imported, base); err != nil {
				return err
			}
		}
		return nil
	}
	if err := collect(src, dir); err != nil {
		return nil, err
	}
	return Compile(merged.String())
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
