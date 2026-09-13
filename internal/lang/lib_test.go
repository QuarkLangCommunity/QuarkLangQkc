package lang

import (
	"path/filepath"
	"strings"
	"testing"
)

// .qlib 库往返：pub 函数序列化必须用 fn 头（func 拼写回归守卫），
// 导入后再编译可运行（LoadImport → CompileWithImports 路径）。
func TestLibraryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	libSrc := "program library;\n\npub fn add(int a, int b) int {\n    return a + b;\n}\n"
	prog, err := Compile(libSrc)
	if err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(dir, "mylib.qlib")
	if err := ExportLibrary(prog, lib); err != nil {
		t.Fatal(err)
	}
	imported, err := LoadImport(dir, "mylib")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(imported, "fn add(") {
		t.Fatalf("exported source must use fn keyword, got:\n%s", imported)
	}
	mainSrc := "import \"mylib\";\n\nfn main(IOStream io) {\n    io.println(add(19, 23));\n}\n"
	p2, err := CompileWithImports(mainSrc, filepath.Join(dir, "t.qk"))
	if err != nil {
		t.Fatalf("recompile with import: %v", err)
	}
	var buf strings.Builder
	if err := Run(p2, "t.qk", nil, strings.NewReader(""), &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "42\n" {
		t.Fatalf("got %q", buf.String())
	}
}
