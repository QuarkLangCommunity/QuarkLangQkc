package lang

import (
	"runtime"
	"strings"
	"testing"
)

// FFI：library 绑定系统库（POSIX: libm+libc；Windows: CRT(msvcrt)），
// 覆盖 libffi（POSIX）与自研 wrapper（Windows）两条实现。
func TestLibraryFFI(t *testing.T) {
	var src string
	if runtime.GOOS == "windows" {
		// Windows 没有 libm/libc：数学与 C 运行时都在 msvcrt 里（同名库只能声明一次）
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
