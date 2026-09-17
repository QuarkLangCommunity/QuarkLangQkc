package main

import (
	"os"
	"path/filepath"
	"quarklang/internal/lang"
	"testing"
)

func BenchmarkPhaseTypecheck(b *testing.B) {
	src := bigSource(120)
	p := filepath.Join(b.TempDir(), "big.qk")
	os.WriteFile(p, []byte(src), 0o644)
	prog, _ := lang.ParseSource(src)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lang.Typecheck(prog)
	}
}
