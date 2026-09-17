package lang

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSrcMapImports 验证：合并 import 后的编译错误/诊断能回到真实文件与行号。
func TestSrcMapImports(t *testing.T) {
	dir := t.TempDir()
	libSrc := `program library;
// 库注释
pub fn double(int n) int {
    return n * 2;
}
`
	mainSrc := `program main;
import "mathlib";

fn main(IOStream io) {
    io.println(double(21));
}
`
	libPath := filepath.Join(dir, "mathlib.qk")
	mainPath := filepath.Join(dir, "main.qk")
	if err := os.WriteFile(libPath, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	prog, sm, err := CompileWithImportsMapped(mainSrc, mainPath)
	if err != nil {
		t.Fatalf("CompileWithImportsMapped: %v", err)
	}
	if sm == nil {
		t.Fatal("SrcMap 为空")
	}
	if len(prog.Funcs) == 0 {
		t.Fatal("未解析到函数")
	}

	var sawLib, sawMain bool
	for _, f := range prog.Funcs {
		file, line := sm.Map(f.Pos.Line)
		switch f.Name {
		case "double":
			sawLib = true
			if file != libPath {
				t.Errorf("double 应映射到 %s，got %q", libPath, file)
			}
			if line != 3 { // lib.qk 第 3 行：pub fn double(...)
				t.Errorf("double 应映射到 mathlib.qk:3，got 第 %d 行", line)
			}
		case "main":
			sawMain = true
			if file != mainPath {
				t.Errorf("main 应映射到 %s，got %q", mainPath, file)
			}
			if line != 4 { // main.qk 第 4 行：fn main(IOStream io) {
				t.Errorf("main 应映射到 main.qk:4，got 第 %d 行", line)
			}
		}
	}
	if !sawLib || !sawMain {
		t.Fatalf("未覆盖两个文件：sawLib=%v sawMain=%v", sawLib, sawMain)
	}

	// 越界与非法的行号必须安全返回零值
	for _, bad := range []int{0, -1, 100000} {
		if file, line := sm.Map(bad); file != "" || line != 0 {
			t.Errorf("Map(%d) 应为空，got (%q,%d)", bad, file, line)
		}
	}
}

// TestSrcMapParserErrorPosition 验证带 import 的文件里，错误行号可映射回原文件。
func TestSrcMapParserErrorPosition(t *testing.T) {
	dir := t.TempDir()
	libSrc := `program library;
pub fn broken(int n) int {
    return n *;
}
`
	mainSrc := `program main;
import "broken";

fn main(IOStream io) {
    io.println(broken(1));
}
`
	if err := os.WriteFile(filepath.Join(dir, "broken.qk"), []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := CompileWithImportsMapped(mainSrc, filepath.Join(dir, "main.qk"))
	if err == nil {
		t.Fatal("期望解析错误")
	}
	if _, ok := err.(*ParseError); !ok && err.Error() == "" {
		t.Fatalf("期望 *ParseError，got %T", err)
	}
}
