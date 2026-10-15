package main

// qkdoc generation pipeline benchmark: parsing + doc model + Markdown rendering.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

func docBigSource(n int) string {
	var b strings.Builder
	b.WriteString("/* 基准文档库。 */\nprogram library;\n\n")
	for i := 0; i < n; i++ {
		b.WriteString("// 函数说明。\n")
		b.WriteString("pub fn f")
		b.WriteString(strings.Repeat("x", i%7))
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`(int n, String s) int {
    int total = 0;
    while (n > 0) {
        total = total + n;
        n = n - 1;
    }
    return total + s.size();
}

`)
	}
	return b.String()
}

func BenchmarkDocPipeline(b *testing.B) {
	src := docBigSource(120)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prog, comments, macros, err := lang.ParseSourceAll(src)
		if err != nil {
			b.Fatal(err)
		}
		doc := lang.BuildDocWithMacros(prog, comments, macros)
		_ = doc.Markdown(lang.DocOptions{Path: "big.qk"})
	}
}

func BenchmarkDocRenderOnly(b *testing.B) {
	src := docBigSource(120)
	prog, comments, macros, err := lang.ParseSourceAll(src)
	if err != nil {
		b.Fatal(err)
	}
	doc := lang.BuildDocWithMacros(prog, comments, macros)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = doc.Markdown(lang.DocOptions{Path: "big.qk"})
	}
}
