package lang

import (
	"path/filepath"
	"strings"
	"testing"
)

// .qlib library round trip: serializing a pub function must use the fn header (regression guard for the func spelling),
// after import it can be compiled and run (the LoadImport → CompileWithImports path).
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
