package lang

import (
	"strings"
	"testing"
)

// sum mathematical optimization: the closed-form result must agree with the language's int=32-bit semantics (wrap); both paths yield the same value
func TestSumClosedFormWrap(t *testing.T) {
	out, err := runSrc(t, `fn ident(int n) int {
    return n * 3 + 1;
}
fn main(IOStream io) {
    io.println(sum(ident, 0, 100000000));
}`)
	if err != nil {
		t.Fatal(err)
	}
	rd := strings.NewReader("")
	if out != expectedWrapV() {
		t.Fatalf("got %q", out)
	}
	_ = rd
}

func expectedWrapV() string {
	// 14999999950000000 mod 2^32 (int = 32-bit two's complement)
	// pinned by the compiler-path baseline value: -1532588160
	const want = "-1532588160\n"
	return want
}
