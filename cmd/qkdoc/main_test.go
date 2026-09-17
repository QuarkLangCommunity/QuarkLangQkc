package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCapture(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return out.String(), errb.String(), code
}

const libSrc = `/* 数学小库。 */
program library;

// 翻倍。
pub fn double(int n) int {
    return n * 2;
}

// 内部函数。
fn hidden(int n) int {
    return n;
}

/* 点。 */
pub type struct {
    int x; // 横坐标
    int y; // 纵坐标
} Point;
`

func writeLib(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "mathlib.qk")
	if err := os.WriteFile(p, []byte(libSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestQkdocMarkdownStdout(t *testing.T) {
	p := writeLib(t, t.TempDir())
	out, errOut, code := runCapture(t, p)
	if code != 0 {
		t.Fatalf("应退出 0，got %d（stderr %s）", code, errOut)
	}
	for _, want := range []string{
		"# " + p,
		"数学小库。",
		"fn double(int n) int",
		"翻倍。",
		"type struct Point;",
		"| `x` | `int` | 横坐标 |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Markdown 缺少 %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "hidden") {
		t.Error("默认应过滤非 pub 符号")
	}
}

func TestQkdocAllFlag(t *testing.T) {
	p := writeLib(t, t.TempDir())
	out, _, _ := runCapture(t, "-all", p)
	if !strings.Contains(out, "hidden") {
		t.Errorf("-all 应包含非 pub 符号:\n%s", out)
	}
}

func TestQkdocHTML(t *testing.T) {
	p := writeLib(t, t.TempDir())
	out, _, code := runCapture(t, "-html", p)
	if code != 0 {
		t.Fatalf("应退出 0，got %d", code)
	}
	for _, want := range []string{"<!doctype html>", "<title>" + p + "</title>", "<pre><code>fn double(int n) int</code></pre>", "<table>"} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML 缺少 %q", want)
		}
	}
	if strings.Contains(out, "hidden") {
		t.Error("HTML 默认应过滤非 pub 符号")
	}
}

func TestQkdocOutputFile(t *testing.T) {
	dir := t.TempDir()
	p := writeLib(t, dir)
	outFile := filepath.Join(dir, "API.md")
	out, _, code := runCapture(t, "-o", outFile, p)
	if code != 0 {
		t.Fatalf("应退出 0，got %d（%s）", code, out)
	}
	if !strings.Contains(out, "已写入") {
		t.Errorf("应报告写入: %q", out)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "fn double(int n) int") {
		t.Errorf("产物内容不对:\n%s", data)
	}
}

func TestQkdocMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	p1 := writeLib(t, dir)
	p2 := filepath.Join(dir, "other.qk")
	if err := os.WriteFile(p2, []byte("// 另一函数。\npub fn other(int a) int {\n    return a;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code := runCapture(t, p1, p2)
	if code != 0 {
		t.Fatalf("应退出 0，got %d", code)
	}
	if !strings.Contains(out, "fn double(int n) int") || !strings.Contains(out, "fn other(int a) int") {
		t.Errorf("多文件应合并输出:\n%s", out)
	}
	if !strings.Contains(out, "\n---\n") {
		t.Error("多文件之间应有分隔")
	}
}

func TestQkdocTitle(t *testing.T) {
	p := writeLib(t, t.TempDir())
	out, _, _ := runCapture(t, "-title", "数学库 API", p)
	if !strings.Contains(out, "# 数学库 API") {
		t.Errorf("自定义标题未生效:\n%s", out)
	}
}

func TestQkdocParseError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.qk")
	if err := os.WriteFile(p, []byte("fn main(IOStream io) {\n    int x = ;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runCapture(t, p)
	if code != 1 {
		t.Fatalf("解析失败应退出 1，got %d", code)
	}
	if !strings.Contains(errOut, "ParseError") {
		t.Errorf("stderr 应含 ParseError: %q", errOut)
	}
	if out != "" {
		t.Errorf("失败时不应输出文档: %q", out)
	}
}

func TestQkdocUsage(t *testing.T) {
	if _, _, code := runCapture(t); code != 2 {
		t.Errorf("无参数应退出 2")
	}
	if _, _, code := runCapture(t, "-o"); code != 2 {
		t.Errorf("-o 缺参数应退出 2")
	}
	if _, _, code := runCapture(t, "-bogus", "x.qk"); code != 2 {
		t.Errorf("未知参数应退出 2")
	}
	if out, _, code := runCapture(t, "--version"); code != 0 || !strings.Contains(out, "qkdoc") {
		t.Errorf("--version 输出不对: %d %q", code, out)
	}
	if out, _, code := runCapture(t, "-h"); code != 0 || !strings.Contains(out, "usage: qkdoc") {
		t.Errorf("-h 输出不对: %d %q", code, out)
	}
}

// TestQkdocRealLibrarySmoke 对真实库做冒烟：能生成、含关键符号、HTML 结构闭合。
func TestQkdocRealLibrarySmoke(t *testing.T) {
	lib := filepath.Join("..", "..", "compiler", "testdata", "mathlib.qk")
	if _, err := os.Stat(lib); err != nil {
		t.Skipf("缺少真实库样本: %v", err)
	}
	out, errOut, code := runCapture(t, lib)
	if code != 0 {
		t.Fatalf("真实库应生成成功，got %d（%s）", code, errOut)
	}
	for _, want := range []string{"fn double(int n) int", "fn square(int n) int"} {
		if !strings.Contains(out, want) {
			t.Errorf("真实库文档缺少 %q:\n%s", want, out)
		}
	}
	h, _, code := runCapture(t, "-html", lib)
	if code != 0 || !strings.Contains(h, "</html>") {
		t.Errorf("HTML 生成异常: code=%d", code)
	}
}
