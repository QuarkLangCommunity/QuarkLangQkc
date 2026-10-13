package main

import (
	"bytes"
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lang_i18n "github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

func runCapture(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return out.String(), errb.String(), code
}

func TestQkreplEvalFlag(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	t.Setenv("QK_LANG", "zh")         // this file asserts Chinese echo output
	out, errOut, code := runCapture(t, "", "-e", "1 + 2 * 3")
	if code != 0 {
		t.Fatalf("it should exit 0, got %d (%s)", code, errOut)
	}
	if strings.TrimSpace(out) != "7" {
		t.Errorf("it should echo 7, got %q", out)
	}
}

func TestQkreplEvalFlagMultiple(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	out, _, code := runCapture(t, "", "-e", "int x = 5;", "-e", "x * 8")
	if code != 0 {
		t.Fatalf("it should exit 0, got %d", code)
	}
	if !strings.Contains(out, "40") {
		t.Errorf("several -e segments should share one environment, got %q", out)
	}
}

func TestQkreplEvalFlagError(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	_, errOut, code := runCapture(t, "", "-e", "1 / 0")
	if code != 1 {
		t.Fatalf("an error should exit 1, got %d", code)
	}
	if !strings.Contains(errOut, "error:") {
		t.Errorf("stderr should carry an error message, got %q", errOut)
	}
}

func TestQkreplBatchMode(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	in := "int x = 21;\nx * 2\n"
	out, errOut, code := runCapture(t, in)
	if code != 0 {
		t.Fatalf("it should exit 0, got %d (%s)", code, errOut)
	}
	if strings.Contains(out, "qk>") {
		t.Errorf("batch mode should not print a prompt, got %q", out)
	}
	if strings.TrimSpace(out) != "42" {
		t.Errorf("it should print 42, got %q", out)
	}
}

func TestQkreplBatchMultiLineBlock(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	in := "fn double(int n) int {\n    return n * 2;\n}\ndouble(21)\n"
	out, errOut, code := runCapture(t, in)
	if code != 0 {
		t.Fatalf("it should exit 0, got %d (%s)", code, errOut)
	}
	if !strings.Contains(out, "已定义 fn double(int n) int") {
		t.Errorf("it should echo the definition summary, got %q", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("it should print 42, got %q", out)
	}
}

func TestQkreplBatchControlFlow(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	in := "int i = 0;\nwhile (i < 3) {\n    i = i + 1;\n}\ni\n"
	out, errOut, code := runCapture(t, in)
	if code != 0 {
		t.Fatalf("it should exit 0, got %d (%s)", code, errOut)
	}
	if strings.TrimSpace(out) != "3" {
		t.Errorf("it should print 3, got %q", out)
	}
}

func TestQkreplBatchErrorContinues(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	in := "int a = 1;\nnope + 1\na + 41\n"
	out, errOut, code := runCapture(t, in)
	if code != 1 {
		t.Fatalf("an error should exit 1, got %d", code)
	}
	if !strings.Contains(errOut, "error:") {
		t.Errorf("stderr should carry an error, got %q", errOut)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("evaluation should continue after an error, got %q", out)
	}
}

func TestQkreplIoOutput(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	out, errOut, code := runCapture(t, `io.println("hello", 42)`+"\n")
	if code != 0 {
		t.Fatalf("it should exit 0, got %d (%s)", code, errOut)
	}
	if !strings.Contains(out, "hello 42") {
		t.Errorf("io.println should write to stdout, got %q", out)
	}
}

func TestQkreplHelpAndQuit(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	out, _, code := runCapture(t, ":help\n")
	if code != 0 || !strings.Contains(out, ":load") {
		t.Errorf(":help should print help, got code=%d %q", code, out)
	}
	out, _, code = runCapture(t, "int x = 1;\n:quit\nnope\n")
	if code != 0 {
		t.Errorf("the session should exit cleanly after :quit, got %d", code)
	}
	if !strings.Contains(out, "qkrepl") == false && strings.Contains(out, "nope") {
		t.Errorf("lines after :quit should not be evaluated, got %q", out)
	}
}

func TestQkreplUnknownCommand(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	_, errOut, code := runCapture(t, ":nope\n")
	if code != 1 {
		t.Fatalf("an unknown command should exit 1, got %d", code)
	}
	if !strings.Contains(errOut, "未知命令") {
		t.Errorf("stderr should explain the unknown command, got %q", errOut)
	}
}

func TestQkreplLoad(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	dir := t.TempDir()
	p := filepath.Join(dir, "lib.qk")
	src := "fn triple(int n) int {\n    return n * 3;\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	in := ":load " + p + "\ntriple(14)\n"
	out, errOut, code := runCapture(t, in)
	if code != 0 {
		t.Fatalf("it should exit 0, got %d (%s)", code, errOut)
	}
	if !strings.Contains(out, "已定义 fn triple(int n) int") {
		t.Errorf(":load should register the definition, got %q", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("it should be callable after loading, got %q", out)
	}
}

func TestQkreplLoadMissingFile(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	_, errOut, code := runCapture(t, ":load /nonexistent/x.qk\n")
	if code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("a missing file should fail and exit 1, got code=%d %q", code, errOut)
	}
}

func TestQkreplUsage(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	if _, _, code := runCapture(t, "", "-e"); code != 2 {
		t.Errorf("-e with a missing argument should exit 2")
	}
	if _, _, code := runCapture(t, "", "-bogus"); code != 2 {
		t.Errorf("an unknown argument should exit 2")
	}
	if out, _, code := runCapture(t, "", "--version"); code != 0 || !strings.Contains(out, "qkrepl") {
		t.Errorf("wrong --version output: %d %q", code, out)
	}
	if out, _, code := runCapture(t, "", "-h"); code != 0 || !strings.Contains(out, "usage: qkrepl") {
		t.Errorf("wrong -h output: %d %q", code, out)
	}
}

func TestQkreplQuietFlag(t *testing.T) {
	defer lang_i18n.SetLocale(lang_i18n.Locale())
	lang_i18n.SetLocale(lang_i18n.ZH) // this test asserts Chinese output
	out, _, code := runCapture(t, "1 + 1\n", "-q")
	if code != 0 {
		t.Fatalf("it should exit 0, got %d", code)
	}
	if strings.Contains(out, "qkrepl") {
		t.Errorf("-q should not print the banner, got %q", out)
	}
}

// TestMain pins the language for the whole package: these tests assert Chinese wording, and the
// REPL renders both through the process default and through internal/lang's injected localizer.
func TestMain(m *testing.M) {
	lang_i18n.SetLocale(lang_i18n.ZH)
	lang.SetLocalizer(lang_i18n.New(lang_i18n.ZH))
	os.Exit(m.Run())
}
