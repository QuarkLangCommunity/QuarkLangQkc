package lang

import (
	"os"
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

// TestCompileTokensWithImportsMatchesTextPath pins the compiler's entry point against the text path: one
// main file with a macro and a string literal plus an imported file carrying a program declaration must
// merge to the same source and produce the same output. The token path only skips the token→text→token
// round trip, so import resolution, the program-declaration strip and the line mapping must not move.
func TestCompileTokensWithImportsMatchesTextPath(t *testing.T) {
	dir := t.TempDir()
	lib := "program library;\n\npub fn shout(String s) String {\n    return s;\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "greet.qk"), []byte(lib), 0o644); err != nil {
		t.Fatal(err)
	}
	const mainSrc = "import \"greet\";\n\n#macro id (x) {\n    #return x\n}\n\n" +
		"fn main(IOStream io) {\n    io.println(id(\"hello world\"));\n    io.println(shout(\"from the import\"));\n}\n"
	mainPath := filepath.Join(dir, "main.qk")

	textProg, err := CompileWithImports(mainSrc, mainPath)
	if err != nil {
		t.Fatalf("CompileWithImports: %v", err)
	}
	toks, err := Lex(mainSrc)
	if err != nil {
		t.Fatal(err)
	}
	macros, rest, err := SplitMacroDefs(toks)
	if err != nil {
		t.Fatal(err)
	}
	expanded := rest
	if len(macros) > 0 {
		if expanded, err = ExpandMacros(rest, macros, "compile"); err != nil {
			t.Fatalf("ExpandMacros: %v", err)
		}
	}
	tokenProg, err := CompileTokensWithImports(expanded, mainSrc, mainPath)
	if err != nil {
		t.Fatalf("CompileTokensWithImports: %v", err)
	}
	if textProg.Src != tokenProg.Src {
		t.Fatalf("merged source differs\n--- text ---\n%s\n--- tokens ---\n%s", textProg.Src, tokenProg.Src)
	}
	var textOut, tokenOut strings.Builder
	if err := Run(textProg, "main.qk", nil, strings.NewReader(""), &textOut); err != nil {
		t.Fatal(err)
	}
	if err := Run(tokenProg, "main.qk", nil, strings.NewReader(""), &tokenOut); err != nil {
		t.Fatal(err)
	}
	if textOut.String() != tokenOut.String() {
		t.Fatalf("output differs: text %q, tokens %q", textOut.String(), tokenOut.String())
	}
	if want := "hello world\nfrom the import\n"; textOut.String() != want {
		t.Fatalf("got %q, want %q", textOut.String(), want)
	}
}
