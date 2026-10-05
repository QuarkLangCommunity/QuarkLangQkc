package lang

import (
	"strings"
	"testing"
)

// Infinite recursion: a clean error instead of a crash
func TestRecursionLimitError(t *testing.T) {
	_, err := runSrc(t, "fn f(int n) int { return f(n + 1); }\nfn main(IOStream io) { io.println(f(0)); }")
	if err == nil || !strings.Contains(err.Error(), "StackOverflowError") {
		t.Fatalf("got %v", err)
	}
}

// channel capacity limit: guards against OOM attacks
func TestChannelCapLimit(t *testing.T) {
	_, err := runSrc(t, "fn main(IOStream io) { Channel c = taskm.channel(2000000000); }")
	if err == nil || !strings.Contains(err.Error(), "requires 0 < n") {
		t.Fatalf("got %v", err)
	}
}

// Tightened new limit (1<<23 entries)
func TestNewCapLimit(t *testing.T) {
	_, err := runSrc(t, "fn main(IOStream io) { pointer int p = new int[66000000]; }")
	if err == nil || !strings.Contains(err.Error(), "badAlloc") {
		t.Fatalf("got %v", err)
	}
}

// Nested generic >> lexical merging: HashTable<String,List<int>>
func TestNestedGenericGtGt(t *testing.T) {
	out, err := runSrc(t, "fn main(IOStream io) {\n HashTable<String,List<int>> h = HashTable::new();\n List<int> l = [1];\n h.put(\"a\", l);\n io.println(h.get(\"a\").size());\n}")
	if err != nil {
		t.Fatal(err)
	}
	if out != "1\n" {
		t.Fatalf("got %q", out)
	}
}
