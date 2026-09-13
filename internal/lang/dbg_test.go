package lang

import (
	"strings"
	"testing"
)

func TestDbgRunQkexec(t *testing.T) {
	prog, err := Compile(`fn main(IOStream io) { int code = qkexec("echo hi"); }`)
	if err != nil {
		t.Fatal("compile:", err)
	}
	var b strings.Builder
	if err := Run(prog, "t.qk", nil, strings.NewReader(""), &b); err != nil {
		t.Fatal("run:", err)
	}
	t.Logf("out=%q", b.String())
}
