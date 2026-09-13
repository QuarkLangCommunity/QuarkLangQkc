package lang

import (
	"strings"
	"testing"
)

// sum 数学优化：闭式结果必须与语言 int=32 位语义一致（wrap），双路径同值
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
	// 14999999950000000 mod 2^32（int=32 位补码）
	// 由编译器路径基准值锁定：-1532588160
	const want = "-1532588160\n"
	return want
}
