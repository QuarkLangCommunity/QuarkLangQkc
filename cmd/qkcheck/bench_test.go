package main

// 工具级基准：单文件检查（lint + 类型检查）耗时分解，供性能回归跟踪。
// 用合成大文件（约 1500 行 / 120 个函数），避免依赖外部库。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quarklang/internal/lang"
)

// bigSource 生成 n 个函数的合成源码（含局部变量、循环、分支、字符串、列表）。
func bigSource(n int) string {
	var b strings.Builder
	b.WriteString("program library;\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `pub fn f%d(int n, String s) int {
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
    List<int> l = [1, 2, 3];
    for (int v : l) {
        total = total + v;
    }
    String t = s.trim();
    if (t.size() > 3) {
        total = total + t.size();
    }
    return total;
}

`, i)
	}
	return b.String()
}

func writeBig(tb testing.TB, n int) string {
	tb.Helper()
	p := filepath.Join(tb.TempDir(), "big.qk")
	if err := os.WriteFile(p, []byte(bigSource(n)), 0o644); err != nil {
		tb.Fatal(err)
	}
	return p
}

func BenchmarkLintOnly(b *testing.B) {
	src := bigSource(120)
	prog, err := lang.ParseSource(src)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lang.Lint(prog, lang.LintOptions{File: "big.qk"})
	}
}

func BenchmarkParseOnly(b *testing.B) {
	src := bigSource(120)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := lang.ParseSource(src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCheckFile(b *testing.B) {
	src := bigSource(120)
	path := filepath.Join(b.TempDir(), "big.qk")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, code := checkFile(path, checkOptions{typecheck: true}, os.Stderr); code != 0 {
			b.Fatalf("checkFile code=%d", code)
		}
	}
}

func BenchmarkCheckFileNoTypecheck(b *testing.B) {
	src := bigSource(120)
	path := filepath.Join(b.TempDir(), "big.qk")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, code := checkFile(path, checkOptions{typecheck: false}, os.Stderr); code != 0 {
			b.Fatalf("checkFile code=%d", code)
		}
	}
}
