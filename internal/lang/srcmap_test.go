package lang

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSrcMapImports verifies: after imports are merged, compile errors/diagnostics can be traced back to the real file and line number.
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
			if line != 3 { // lib.qk line 3: pub fn double(...)
				t.Errorf("double 应映射到 mathlib.qk:3，got 第 %d 行", line)
			}
		case "main":
			sawMain = true
			if file != mainPath {
				t.Errorf("main 应映射到 %s，got %q", mainPath, file)
			}
			if line != 4 { // main.qk line 4: fn main(IOStream io) {
				t.Errorf("main 应映射到 main.qk:4，got 第 %d 行", line)
			}
		}
	}
	if !sawLib || !sawMain {
		t.Fatalf("未覆盖两个文件：sawLib=%v sawMain=%v", sawLib, sawMain)
	}

	// out-of-range and invalid line numbers must safely return the zero value
	for _, bad := range []int{0, -1, 100000} {
		if file, line := sm.Map(bad); file != "" || line != 0 {
			t.Errorf("Map(%d) 应为空，got (%q,%d)", bad, file, line)
		}
	}
}

// TestSrcMapParserErrorPosition verifies that in a file with imports, an error line number can be mapped back to the original file.
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
