package lang

import (
	"strings"
	"testing"
)

func runCase(t *testing.T, body string) string {
	t.Helper()
	out, err := runSrc(t, body)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// 相等/比较语义边界（Value 标签化后的完整回归）
func TestEqualityEdges(t *testing.T) {
	v := runCase(t, "fn main(IOStream io) {\n"+
		"    io.println(1 == 2.0);\n"+
		"    io.println(1.5 == 1.5);\n"+
		"    io.println(5 != 5.0);\n"+
		"    io.println(null == null);\n"+
		"    io.println(\"abc\" == \"abc\");\n"+
		"    io.println(3 < 3.5);\n"+
		"    io.println(2.0 <= 2.0);\n"+
		"    io.println(true == true);\n"+
		"}")
	want := strings.Join([]string{"false", "true", "false", "true", "true", "true", "true", "true"}, "\n") + "\n"
	if v != want {
		t.Fatalf("got %q want %q", v, want)
	}
}

// 浮点除零：按语言设计报错（DivisionByZeroError），错误路径显式验证
func TestFloatDivZeroErrors(t *testing.T) {
	_, err := runSrc(t, "fn main(IOStream io) {\n"+
		"    float x = 0.0 / 0.0;\n"+
		"    io.println(x);\n"+
		"}")
	if err == nil || !strings.Contains(err.Error(), "DivisionByZeroError") {
		t.Fatalf("expected DivisionByZeroError, got %v", err)
	}
}

// 设计语义：List 不支持 ==（类类型无值相等语义，类型检查显式拒绝）
func TestListComparisonRejected(t *testing.T) {
	_, err := runSrc(t, "fn main(IOStream io) {\n"+
		"    List<int> l = [1, 2];\n"+
		"    io.println(l == l);\n"+
		"}")
	if err == nil || !strings.Contains(err.Error(), "cannot compare") {
		t.Fatalf("expected comparison rejection, got %v", err)
	}
}

// 列表别名同引用：l2=l 后 append 共享（与 deepCopy 区分）
func TestListAliasSharing(t *testing.T) {
	v := runCase(t, "fn main(IOStream io) {\n"+
		"    List<int> l = [1, 2, 3];\n"+
		"    List<int> l2 = l;\n"+
		"    l2.append(4);\n"+
		"    io.println(l.size());\n"+
		"}")
	if v != "4\n" {
		t.Fatalf("got %q", v)
	}
}

// float 与 int 混比正确性（数字间跨类型比较按数值）
func TestMixedNumericCmp(t *testing.T) {
	v := runCase(t, "fn main(IOStream io) {\n"+
		"    io.println(7 == 7.0);\n"+
		"    io.println(7.0 == 7);\n"+
		"    io.println(8 > 7.9);\n"+
		"}")
	if v != "true\ntrue\ntrue\n" {
		t.Fatalf("got %q", v)
	}
}
