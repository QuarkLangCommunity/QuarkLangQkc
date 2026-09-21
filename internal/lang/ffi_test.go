package lang

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// FFI：library 绑定系统库（POSIX: libm/libc；Windows: CRT(msvcrt)），跨平台 libffi/自研 wrapper 调用
func TestLibraryFFI(t *testing.T) {
	// Windows 没有 libm/libc（C 运行时在 msvcrt 里），按平台选库名——同时覆盖 Windows 的 FFI 实现
	mathLib, cLib := "m", "c"
	if runtime.GOOS == "windows" {
		mathLib, cLib = "msvcrt", "msvcrt"
	}
	out, err := runSrc(t, fmt.Sprintf(`library %s {
    fn sqrt(double x) double;
    fn pow(double x, double y) double;
}
library %s {
    fn strlen(String s) long;
    fn rand() int;
}

fn main(IOStream io) {
    io.println(%s.sqrt(16.0));
    io.println(%s.pow(2.0, 10.0));
    io.println(%s.strlen("abcdef"));
    io.println(%s.rand());
}`, mathLib, cLib, mathLib, mathLib, cLib, cLib))
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
