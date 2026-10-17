//go:build cgo

package lang

// The end-to-end arity regression: the interpreter loads a real shared library and calls C functions
// with the argument counts the old Windows shim could not express.
//
// The probe is built with the host C compiler at test time, so the same test runs on Linux, macOS and
// Windows; on Windows it is the proof that the Microsoft x64 thunk (not just the System V twin used by
// the local shim test) reaches a real sixth and seventeenth argument. The six-argument function has the
// shape Cleg's cleg_image has (String, then five ints); the eighteen-argument one crosses both register
// files and leaves ten arguments on the stack on Windows.

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const ffiArityProbeC = `
#include <string.h>

int qk_image(const char* path, int x, int y, int w, int h, int repeat) {
    return (int)strlen(path) + x + y + w + h + repeat;
}

long long qk_sum17(int a1, int a2, int a3, int a4, int a5, int a6, int a7, int a8, int a9,
                   int a10, int a11, int a12, int a13, int a14, int a15, int a16, int a17) {
    return a1 + 2*a2 + 3*a3 + 4*a4 + 5*a5 + 6*a6 + 7*a7 + 8*a8 + 9*a9 + 10*a10 +
           11*a11 + 12*a12 + 13*a13 + 14*a14 + 15*a15 + 16*a16 + 17*a17;
}

double qk_mix(double a, int b, float c, double d, int e, float f, double g, int h,
              double i, int j, float k, double l, int m, int n, int o, int p, int q, double r) {
    return a + b + c + d + e + f + g + h + i + j + k + l + m + n + o + p + q + r;
}
`

const ffiArityProbeQL = `library qkffiprobe {
    fn qk_image(String path, int x, int y, int w, int h, int repeat) int;
    fn qk_sum17(int a1, int a2, int a3, int a4, int a5, int a6, int a7, int a8, int a9,
                int a10, int a11, int a12, int a13, int a14, int a15, int a16, int a17) long;
    fn qk_mix(double a, int b, f32 c, double d, int e, f32 f, double g, int h, double i,
              int j, f32 k, double l, int m, int n, int o, int p, int q, double r) double;
}

fn main(IOStream io) {
    io.println(qkffiprobe.qk_image("abcd", 1, 2, 3, 4, 5));
    io.println(qkffiprobe.qk_sum17(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17));
    io.println(qkffiprobe.qk_mix(1.0, 2, 3.0, 4.0, 5, 6.0, 7.0, 8, 9.0, 10, 11.0, 12.0,
                                 13, 14, 15, 16, 17, 18.0));
}`

func TestLibraryFFIArity(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH != "amd64" {
		t.Skip("the Windows shim implements the Microsoft x64 convention")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		if cc, err = exec.LookPath("gcc"); err != nil {
			t.Skip("no C compiler (cc/gcc), skipping the FFI arity probe")
		}
	}
	// The Windows loader keeps a DLL mapped until the process exits, so the directory is removed on a
	// best-effort basis rather than through t.TempDir, whose cleanup treats a locked DLL as a failure.
	dir, err := os.MkdirTemp("", "qkffi-arity-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	cf := filepath.Join(dir, "qkffiprobe.c")
	if err := os.WriteFile(cf, []byte(ffiArityProbeC), 0o644); err != nil {
		t.Fatal(err)
	}
	var lib string
	var args []string
	switch runtime.GOOS {
	case "windows":
		lib = "qkffiprobe.dll"
		args = []string{"-shared", "-static-libgcc", "-O1", "-o"}
	case "darwin":
		lib = "libqkffiprobe.dylib"
		args = []string{"-dynamiclib", "-O1", "-o"}
	default:
		lib = "libqkffiprobe.so"
		args = []string{"-shared", "-fPIC", "-O1", "-o"}
	}
	outName := filepath.Join(dir, lib)
	args = append(args, outName, cf)
	if out, err := exec.Command(cc, args...).CombinedOutput(); err != nil {
		t.Fatalf("building the FFI probe library failed: %v\n%s", err, out)
	}
	t.Chdir(dir)
	out, err := runSrc(t, ffiArityProbeQL)
	if err != nil {
		t.Fatalf("the FFI arity program failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	want := []string{"19", "1785", "171"}
	if len(lines) != len(want) {
		t.Fatalf("got %q, want %q", out, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d is %q, want %q (whole output %q)", i+1, lines[i], want[i], out)
		}
	}
}
