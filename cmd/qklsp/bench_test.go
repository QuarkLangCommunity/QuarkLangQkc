package main

// 编辑器延迟基准：每次编辑后的重新分析（parse + lint + 类型检查）与查询（补全/跳转/悬停）。

import (
	"fmt"
	"strings"
	"testing"
)

func bigDocSource(n int) string {
	var b strings.Builder
	b.WriteString("program library;\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `// 函数 f%d 的说明。
pub fn f%d(int n, String s) int {
    int total = 0;
    int i = 0;
    while (i < n) {
        if (i %% 2 == 0) {
            total = total + i;
        } else {
            total = total - i;
        }
        i = i + 1;
    }
    return total + s.size();
}

`, i, i)
	}
	return b.String()
}

func BenchmarkAnalyzeDocument(b *testing.B) {
	src := bigDocSource(120)
	uri := docURI("bench.qk")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := newDocument(uri, src)
		_ = d
	}
}

func BenchmarkAnalyzeDocumentNoTypecheck(b *testing.B) {
	src := bigDocSource(120)
	uri := docURI("bench.qk")
	d := newDocument(uri, src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.diagnostics(nil)
	}
}

func BenchmarkCompletion(b *testing.B) {
	src := bigDocSource(120)
	d := newDocument(docURI("bench.qk"), src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.completionCandidates(1200)
	}
}

func BenchmarkDefinitionLookup(b *testing.B) {
	src := bigDocSource(120)
	d := newDocument(docURI("bench.qk"), src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.lookup("f60", 1200)
	}
}
