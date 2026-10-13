package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// runCapture runs the CLI and captures stdout/stderr.
func runCapture(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const cleanSrc = `fn main(IOStream io) {
    io.println("ok");
}
`

func TestCLICleanExitZero(t *testing.T) {
	p := writeFile(t, t.TempDir(), "clean.qk", cleanSrc)
	out, _, code := runCapture(t, p)
	if code != 0 {
		t.Fatalf("a clean file should exit 0, got %d (output %q)", code, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("a clean file should produce no output, got %q", out)
	}
}

func TestCLIFindingsExitOne(t *testing.T) {
	src := `fn main(IOStream io) {
    int unused = 1;
    io.println("ok");
}
`
	p := writeFile(t, t.TempDir(), "unused.qk", src)
	out, _, code := runCapture(t, p)
	if code != 1 {
		t.Fatalf("a file with diagnostics should exit 1, got %d", code)
	}
	if !strings.Contains(out, "QK101") {
		t.Errorf("the output should contain QK101: %q", out)
	}
}

func TestCLIExit0Flag(t *testing.T) {
	p := writeFile(t, t.TempDir(), "unused.qk", "fn main(IOStream io) {\n    int unused = 1;\n    io.println(\"ok\");\n}\n")
	_, _, code := runCapture(t, "-exit0", p)
	if code != 0 {
		t.Fatalf("-exit0 should exit 0, got %d", code)
	}
}

func TestCLIJSONOutput(t *testing.T) {
	p := writeFile(t, t.TempDir(), "unused.qk", "fn main(IOStream io) {\n    int unused = 1;\n    io.println(\"ok\");\n}\n")
	out, _, _ := runCapture(t, "-json", p)
	var got []finding
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("JSON parsing failed: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].Code != "QK101" || got[0].Line != 2 {
		t.Fatalf("unexpected JSON content: %+v", got)
	}
	if got[0].File != p {
		t.Errorf("the JSON file field should be %s, got %s", p, got[0].File)
	}
}

func TestCLIJSONCleanIsEmptyArray(t *testing.T) {
	p := writeFile(t, t.TempDir(), "clean.qk", cleanSrc)
	out, _, _ := runCapture(t, "-json", p)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("a clean file should produce the JSON value [], got %q", out)
	}
}

func TestCLIParseErrorReported(t *testing.T) {
	p := writeFile(t, t.TempDir(), "bad.qk", "fn main(IOStream io) {\n    int x = ;\n}\n")
	out, _, code := runCapture(t, p)
	if code != 1 {
		t.Fatalf("a syntax error should exit 1, got %d", code)
	}
	if !strings.Contains(out, "ParseError") || !strings.Contains(out, ":2:") {
		t.Errorf("a ParseError with a line and a column was expected: %q", out)
	}
}

func TestCLITypeErrorReported(t *testing.T) {
	p := writeFile(t, t.TempDir(), "type.qk", "fn main(IOStream io) {\n    int x = \"s\";\n    io.println(x);\n}\n")
	out, _, code := runCapture(t, p)
	if code != 1 {
		t.Fatalf("a type error should exit 1, got %d", code)
	}
	if !strings.Contains(out, "error:") || !strings.Contains(out, "cannot assign") {
		t.Errorf("a type error was expected: %q", out)
	}
}

func TestCLIUsageErrors(t *testing.T) {
	if _, _, code := runCapture(t); code != 2 {
		t.Errorf("no arguments should exit 2, got %d", code)
	}
	if _, _, code := runCapture(t, "-nope", "x.qk"); code != 2 {
		t.Errorf("an unknown argument should exit 2, got %d", code)
	}
	if _, _, code := runCapture(t, "-L"); code != 2 {
		t.Errorf("-L with a missing argument should exit 2, got %d", code)
	}
	if out, _, code := runCapture(t, "--version"); code != 0 || !strings.Contains(out, "qkcheck") {
		t.Errorf("--version should exit 0 and print the version, got %d %q", code, out)
	}
	if out, _, code := runCapture(t, "-h"); code != 0 || !strings.Contains(out, "usage: qkcheck") {
		t.Errorf("-h should print usage, got %d %q", code, out)
	}
}

// TestCLIImportPath verifies -L: a cross-directory import resolves once the search directory is given;
// and the error position maps back to the line of the import statement.
func TestCLIImportPath(t *testing.T) {
	root := t.TempDir()
	libDir := filepath.Join(root, "libs")
	appDir := filepath.Join(root, "app")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, libDir, "mathlib.qk", "program library;\npub fn double(int n) int {\n    return n * 2;\n}\n")
	app := writeFile(t, appDir, "main.qk", "program main;\nimport \"mathlib\";\n\nfn main(IOStream io) {\n    io.println(double(21));\n}\n")

	out, _, code := runCapture(t, app)
	if code != 1 || !strings.Contains(out, "ImportError") {
		t.Fatalf("without -L an ImportError and exit 1 were expected, got code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "main.qk:2:1") {
		t.Errorf("ImportError should point at the import statement (line 2), got %q", out)
	}

	out, _, code = runCapture(t, "-L", libDir, app)
	if code != 0 {
		t.Fatalf("with -L the check should pass, got code=%d out=%q", code, out)
	}
}

// TestCLICorpusNoFalsePositives is the whole-repo corpus gate:
//   - no error of any kind is allowed (syntax/type/import all pass);
//   - warnings must match the manually reviewed snapshot entry by entry (a new warning = new false positive, needs review).
func TestCLICorpusNoFalsePositives(t *testing.T) {
	root := filepath.Join("..", "..")
	var files []string
	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if skipNestedRepo(p, fi, root) {
			return filepath.SkipDir
		}
		if err != nil {
			return err
		}
		if fi.IsDir() {
			switch fi.Name() {
			case ".git", "dist", "dist-ci":
				return filepath.SkipDir
			case "stress-out":
				// Generated stress inputs (cmd/qkstress gen): adversarial sources by construction, so
				// linting them says nothing about this corpus — and walking untracked build output at all
				// is what let a 1 MiB single-line case overflow the type checker's stack here.
				return filepath.SkipDir
			case "lintbench", "bench":
				// Benchmark/performance fixtures: lintbench is gated by TestLintBenchmark against its own manifest;
				// bench/tools holds benchmark inputs (including third-party library copies) and is not part of corpus review.
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".qk") || strings.HasSuffix(p, ".kq") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 10 {
		t.Fatalf("too few corpus files (%d): the test proves nothing", len(files))
	}
	sort.Strings(files)

	out, errOut, code := runCapture(t, append([]string{"-json"}, files...)...)
	if code != 1 && code != 0 {
		t.Fatalf("the corpus check exited abnormally with %d: %s", code, errOut)
	}
	var got []finding
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	// 1) no errors allowed
	for _, f := range got {
		if f.Sev == "error" {
			t.Errorf("the corpus produced an error-level diagnostic (there should be none): %s", f)
		}
	}

	// 2) warning snapshot: all are manually reviewed true findings (see the "measurements" section of the README)
	wantWarnings := map[string]bool{
		"compiler/testdata/cases/a_for_break.kq:2:1 QK105":  true, // log n; ends a function that declares an int return but yields nil
		"compiler/testdata/cases/a_for_break.kq:4:5 QK103":  true, // the return after log is unreachable
		"compiler/testdata/cases_run/t_table.kq:45:5 QK101": true, // the loop variable of for (String k : sk) is unused
		"compiler/testdata/demo.qk:28:13 QK101":             true, // int bad = 10 / 0; only there to trigger an exception
		"examples/trycatch.qk:3:13 QK101":                   true, // int a = 10 / 0; same as above
		// The next 6 entries are hits on **test fixtures** after QK108-QK114 were strengthened, each reviewed as a true finding:
		// the fixtures deliberately demonstrate these shapes (used to align with the qkc dual path), so they are not false positives.
		"compiler/testdata/cases/a_scalars.kq:13:34 QK114":    true, // b == b compares a value with itself (demonstrates == semantics)
		"compiler/testdata/cases/b_struct.kq:42:26 QK114":     true, // a == a (struct value comparison)
		"compiler/testdata/cases/b_struct_ref.kq:21:36 QK114": true, // w == w (reference comparison)
		"compiler/testdata/cases/y_edge.kq:7:23 QK108":        true, // x = x self-assignment (edge case)
		"compiler/testdata/cases/y_edge.kq:19:13 QK113":       true, // a List literal overwrites the initial value (demonstrates assignment semantics)
		"compiler/testdata/cases_run/z_any.kq:22:5 QK113":     true, // a void variable assigned twice; the first assignment is a dead store
		"compiler/testdata/cases_run/v_sign.kq:12:1 QK115":    true, // fn mix is never called anywhere in the file (dead function in the fixture)
	}
	gotWarnings := map[string]bool{}
	for _, f := range got {
		rel, err := filepath.Rel(root, f.File)
		if err != nil {
			rel = f.File
		}
		// The snapshot table uses /: on Windows filepath.Rel returns \, so normalize before comparing (consistent across systems)
		rel = filepath.ToSlash(rel)
		gotWarnings[rel+":"+itoa(f.Line)+":"+itoa(f.Col)+" "+f.Code] = true
	}
	for k := range gotWarnings {
		if !wantWarnings[k] {
			t.Errorf("an unreviewed warning appeared (possibly a new false positive): %s", k)
		}
	}
	for k := range wantWarnings {
		if !gotWarnings[k] {
			t.Errorf("a warning in the snapshot disappeared (a regression in the check?): %s", k)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// skipNestedRepo reports whether a walked directory belongs to a different repository. The QuarkLang
// family keeps its sibling projects inside this workspace, each with its own .git, and another
// repository's files are not this project's files: walking into them would let their sources decide
// this project's gates.
func skipNestedRepo(path string, info os.FileInfo, root string) bool {
	if info == nil || !info.IsDir() || path == root {
		return false
	}
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
