package main

// Front-end cost breakdown benchmark: how much time lexing (Lex) and parsing (Parse) each take -- used to decide which services are worth hosting in C.
// Measured on a 1500-line synthetic file: Lex ~0.48ms, Parse ~0.61ms; on this evidence the
// C lexer proposal was evaluated and **rejected** (cgo boundary + rebuilding tokens on the Go side eats the C scanning gain, no net end-to-end win).

import (
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
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
