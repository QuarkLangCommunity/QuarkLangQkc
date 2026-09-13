package lang

import (
	"strings"
	"testing"
)

// 无限递归：干净报错而非崩溃
func TestRecursionLimitError(t *testing.T) {
	_, err := runSrc(t, "fn f(int n) int { return f(n + 1); }\nfn main(IOStream io) { io.println(f(0)); }")
	if err == nil || !strings.Contains(err.Error(), "StackOverflowError") {
		t.Fatalf("got %v", err)
	}
}

// channel 容量上限：防 OOM 攻击
func TestChannelCapLimit(t *testing.T) {
	_, err := runSrc(t, "fn main(IOStream io) { Channel c = taskm.channel(2000000000); }")
	if err == nil || !strings.Contains(err.Error(), "requires 0 < n") {
		t.Fatalf("got %v", err)
	}
}

// new 上限收窄（1<<23 条目）
func TestNewCapLimit(t *testing.T) {
	_, err := runSrc(t, "fn main(IOStream io) { pointer List<int> p = new int[66000000]; }")
	if err == nil || !strings.Contains(err.Error(), "badAlloc") {
		t.Fatalf("got %v", err)
	}
}

// 嵌套泛型 >> 词法合并：HashTable<String,List<int>>
func TestNestedGenericGtGt(t *testing.T) {
	out, err := runSrc(t, "fn main(IOStream io) {\n HashTable<String,List<int>> h = HashTable::new();\n List<int> l = [1];\n h.put(\"a\", l);\n io.println(h.get(\"a\").size());\n}")
	if err != nil {
		t.Fatal(err)
	}
	if out != "1\n" {
		t.Fatalf("got %q", out)
	}
}
