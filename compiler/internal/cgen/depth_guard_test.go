package cgen

// Regression tests for the call-depth guard of the generated code: the two engines must refuse the
// same call, with the same message, at the same line and with the same exit status. The interpreter
// counts every call context it creates, so the compiled binary counts the same levels with the
// thread-local counter below, and a self tail call must be counted too — the optimisation that reuses
// the native frame would otherwise let a tail-recursive program run past the shared limit.

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// downTestSrc builds a non-tail recursion that goes n levels below main; the recursive call is line 3.
func downTestSrc(n string) string {
	return "fn down(int n) int {\n  if (n == 0) { return 0; }\n  return n + down(n - 1);\n}\n" +
		"\nfn main(IOStream io) {\n  io.println(down(" + n + "));\n}\n"
}

// tailTestSrc builds a self tail call that goes n levels below main; the recursive call is line 3.
func tailTestSrc(n string) string {
	return "fn up(int n, int acc) int {\n  if (n == 0) { return acc; }\n  return up(n - 1, acc + 1);\n}\n" +
		"\nfn main(IOStream io) {\n  io.println(up(" + n + ", 0));\n}\n"
}

// lliStatus runs IR with lli and returns its merged output and exit status, skipping when lli is absent.
func lliStatus(t *testing.T, ir string) (string, int) {
	t.Helper()
	lli, err := exec.LookPath("lli")
	if err != nil {
		t.Skip("lli not available")
	}
	f, err := os.CreateTemp("", "quark-depth-*.ll")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.WriteString(withTestRuntime(ir)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	out, err := exec.Command(lli, name).CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("lli: %v\n%s", err, out)
	}
	return string(out), exitErr.ExitCode()
}

// TestDepthGuardShapeIsInTheGeneratedIR checks the three properties the guard needs: the shared
// counter with its definition, the call site's line stored before the call, and no tail call.
func TestDepthGuardShapeIsInTheGeneratedIR(t *testing.T) {
	ir := transpile(t, tailTestSrc("9000"))
	for _, want := range []string{
		"@qk_call_depth = thread_local global i32 0",
		"@qk_call_line = thread_local global i32 0",
		"define void @ql_depth_enter()",
		"define void @ql_depth_leave()",
		"store i32 3, i32* @qk_call_line",
	} {
		if !strings.Contains(ir, want) {
			t.Errorf("generated IR does not contain %q", want)
		}
	}
	if strings.Contains(ir, "tail call") {
		t.Error("a tail call would reuse the frame and bypass the depth counter")
	}
}

// TestGeneratedCallDepthBoundary runs the generated code at the shared limit, one level past it, and
// for a self tail call past it: the interpreter answers 33550336 / the shared error / the shared error.
func TestGeneratedCallDepthBoundary(t *testing.T) {
	want := "error: " + lang.MaxCallDepthMessage + " at line 3\n"
	cases := []struct {
		name     string
		src      string
		wantOut  string
		wantExit int
	}{
		{"at the cap", downTestSrc("8191"), "33550336\n", 0},
		{"past the cap", downTestSrc("8192"), want, 1},
		{"tail call past the cap", tailTestSrc("9000"), want, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, code := lliStatus(t, transpile(t, tc.src))
			if code != tc.wantExit {
				t.Errorf("exit status = %d, want %d (output %q)", code, tc.wantExit, out)
			}
			if out != tc.wantOut {
				t.Errorf("output = %q, want %q", out, tc.wantOut)
			}
		})
	}
}
