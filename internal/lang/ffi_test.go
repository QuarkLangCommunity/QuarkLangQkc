package lang

import (
	"strings"
	"testing"
)

// FFI：library 绑定系统库（libm/libc 真是符），跨平台 libffi 调用
func TestLibraryFFI(t *testing.T) {
	out, err := runSrc(t, `library m {
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
}`)
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
