package main

// 前端成本拆分基准：词法（Lex）与语法（Parse）各占多少——用于判断「哪些服务值得托管给 C」。
// 实测（1500 行合成文件）：Lex ≈ 0.48ms、Parse ≈ 0.61ms；曾按此结论评估并**否决**了
// C 词法器方案（cgo 边界 + Go 侧重建 token 的成本吃掉 C 扫描收益，端到端无净收益）。

import (
	"testing"

	"quarklang/internal/lang"
)

func BenchmarkLexOnly(b *testing.B) {
	src := bigSource(120)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := lang.Lex(src); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkLexWithComments(b *testing.B) {
	src := bigSource(120)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toks, comments, err := lang.LexWithComments(src)
		if err != nil {
			b.Fatal(err)
		}
		_ = toks
		_ = comments
	}
}
