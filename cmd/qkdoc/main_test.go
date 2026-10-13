package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lang_i18n "github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

func runCapture(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return out.String(), errb.String(), code
}

const libSrc = `/* Math micro-library. */
program library;

// Doubles a number.
pub fn double(int n) int {
    return n * 2;
}

// Internal function.
fn hidden(int n) int {
    return n;
}

/* A point. */
pub type struct {
    int x; // x coordinate
    int y; // y coordinate
} Point;
`

func writeLib(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "mathlib.qk")
	if err := os.WriteFile(p, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestQkdocMarkdownStdout(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	p := writeLib(t, t.TempDir())
	out, errOut, code := runCapture(t, p)
	if code != 0 {
		t.Fatalf("it should exit 0, got %d (stderr %s)", code, errOut)
	}
	for _, want := range []string{
		"# " + p,
		"Math micro-library.",
		"fn double(int n) int",
		"Doubles a number.",
		"type struct Point;",
		"| `x` | `int` | x coordinate |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the Markdown is missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "hidden") {
		t.Error("non-pub symbols should be filtered by default")
	}
}

func TestQkdocAllFlag(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	p := writeLib(t, t.TempDir())
	out, _, _ := runCapture(t, "-all", p)
	if !strings.Contains(out, "hidden") {
		t.Errorf("-all should include non-pub symbols:\n%s", out)
	}
}

func TestQkdocHTML(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	p := writeLib(t, t.TempDir())
	out, _, code := runCapture(t, "-html", p)
	if code != 0 {
		t.Fatalf("it should exit 0, got %d", code)
	}
	for _, want := range []string{"<!doctype html>", "<title>" + p + "</title>", "<pre><code>fn double(int n) int</code></pre>", "<table>"} {
		if !strings.Contains(out, want) {
			t.Errorf("the HTML is missing %q", want)
		}
	}
	if strings.Contains(out, "hidden") {
		t.Error("the HTML should filter non-pub symbols by default")
	}
}

func TestQkdocOutputFile(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	t.Setenv("QK_LANG", "zh")         // this test asserts Chinese output
	dir := t.TempDir()
	p := writeLib(t, dir)
	outFile := filepath.Join(dir, "API.md")
	out, _, code := runCapture(t, "-o", outFile, p)
	if code != 0 {
		t.Fatalf("it should exit 0, got %d (%s)", code, out)
	}
	if !strings.Contains(out, "已写入") {
		t.Errorf("it should report the write: %q", out)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "fn double(int n) int") {
		t.Errorf("wrong artifact content:\n%s", data)
	}
}

func TestQkdocMultipleFiles(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	dir := t.TempDir()
	p1 := writeLib(t, dir)
	p2 := filepath.Join(dir, "other.qk")
	if err := os.WriteFile(p2, []byte("// Another function.\npub fn other(int a) int {\n    return a;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCapture(t, p1, p2)
	if code != 0 {
		t.Fatalf("it should exit 0, got %d", code)
	}
	if !strings.Contains(out, "fn double(int n) int") || !strings.Contains(out, "fn other(int a) int") {
		t.Errorf("several files should be merged into one document:\n%s", out)
	}
	if !strings.Contains(out, "\n---\n") {
		t.Error("several files should be separated")
	}
}

func TestQkdocTitle(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	p := writeLib(t, t.TempDir())
	out, _, _ := runCapture(t, "-title", "Math API", p)
	if !strings.Contains(out, "# Math API") {
		t.Errorf("the custom title had no effect:\n%s", out)
	}
}

func TestQkdocParseError(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	p := filepath.Join(t.TempDir(), "bad.qk")
	if err := os.WriteFile(p, []byte("fn main(IOStream io) {\n    int x = ;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runCapture(t, p)
	if code != 1 {
		t.Fatalf("a parse failure should exit 1, got %d", code)
	}
	if !strings.Contains(errOut, "ParseError") {
		t.Errorf("stderr should contain ParseError: %q", errOut)
	}
	if out != "" {
		t.Errorf("a failure should produce no document: %q", out)
	}
}

func TestQkdocUsage(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	if _, _, code := runCapture(t); code != 2 {
		t.Errorf("no arguments should exit 2")
	}
	if _, _, code := runCapture(t, "-o"); code != 2 {
		t.Errorf("-o with a missing argument should exit 2")
	}
	if _, _, code := runCapture(t, "-bogus", "x.qk"); code != 2 {
		t.Errorf("an unknown argument should exit 2")
	}
	if out, _, code := runCapture(t, "--version"); code != 0 || !strings.Contains(out, "qkdoc") {
		t.Errorf("wrong --version output: %d %q", code, out)
	}
	if out, _, code := runCapture(t, "-h"); code != 0 || !strings.Contains(out, "usage: qkdoc") {
		t.Errorf("wrong -h output: %d %q", code, out)
	}
}

// TestQkdocRealLibrarySmoke smoke-tests a real library: the docs generate, contain the key symbols, and the HTML structure is closed.
func TestQkdocRealLibrarySmoke(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	lib := filepath.Join("..", "..", "compiler", "testdata", "mathlib.qk")
	if _, err := os.Stat(lib); err != nil {
		t.Skipf("missing the real library sample: %v", err)
	}
	out, errOut, code := runCapture(t, lib)
	if code != 0 {
		t.Fatalf("a real library should generate successfully, got %d (%s)", code, errOut)
	}
	for _, want := range []string{"fn double(int n) int", "fn square(int n) int"} {
		if !strings.Contains(out, want) {
			t.Errorf("the real library's document is missing %q:\n%s", want, out)
		}
	}
	h, _, code := runCapture(t, "-html", lib)
	if code != 0 || !strings.Contains(h, "</html>") {
		t.Errorf("HTML generation failed: code=%d", code)
	}
}
