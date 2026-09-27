package lang

import (
	"runtime"
	"strings"
	"testing"
)

// FFI: library binds system libraries (POSIX: libm+libc; Windows: CRT(msvcrt)),
// covering both implementations: libffi (POSIX) and the in-house wrapper (Windows).
func TestLibraryFFI(t *testing.T) {
	var src string
	if runtime.GOOS == "windows" {
		// Windows has no libm/libc: both the math and the C runtime live in msvcrt (a library with the same name can only be declared once)
		src = `library msvcrt {
    fn sqrt(double x) double;
    fn pow(double x, double y) double;
    fn strlen(String s) long;
    fn rand() int;
}

fn main(IOStream io) {
    io.println(msvcrt.sqrt(16.0));
    io.println(msvcrt.pow(2.0, 10.0));
    io.println(msvcrt.strlen("abcdef"));
    io.println(msvcrt.rand());
}`
	} else {
		src = `library m {
    fn sqrt(double x) double;
    fn pow(double x, double y) double;
}
library c {
    fn strlen(String s) long;
    fn rand() int;
}

fn main(IOStream io) {
    io.println(m.sqrt(16.0));
    io.println(m.pow(2.0, 10.0));
    io.println(c.strlen("abcdef"));
    io.println(c.rand());
}`
	}
	out, err := runSrc(t, src)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if !strings.HasPrefix(lines[0], "4") || !strings.HasPrefix(lines[1], "1024") || lines[2] != "6" {
		t.Fatalf("got %q", out)
	}
	if lines[3] == "" {
		t.Fatalf("rand empty: %q", out)
	}
}
