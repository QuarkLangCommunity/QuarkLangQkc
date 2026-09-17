package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// runCapture 执行 CLI 并捕获 stdout/stderr。
func runCapture(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const cleanSrc = `fn main(IOStream io) {
    io.println("ok");
}
`

func TestCLICleanExitZero(t *testing.T) {
	p := writeFile(t, t.TempDir(), "clean.qk", cleanSrc)
	out, _, code := runCapture(t, p)
	if code != 0 {
		t.Fatalf("干净文件应退出 0，got %d（输出 %q）", code, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("干净文件不应有输出，got %q", out)
	}
}

func TestCLIFindingsExitOne(t *testing.T) {
	src := `fn main(IOStream io) {
    int unused = 1;
    io.println("ok");
}
`
	p := writeFile(t, t.TempDir(), "unused.qk", src)
	out, _, code := runCapture(t, p)
	if code != 1 {
		t.Fatalf("有诊断应退出 1，got %d", code)
	}
	if !strings.Contains(out, "QK101") {
		t.Errorf("输出应包含 QK101：%q", out)
	}
}

func TestCLIExit0Flag(t *testing.T) {
	p := writeFile(t, t.TempDir(), "unused.qk", "fn main(IOStream io) {\n    int unused = 1;\n    io.println(\"ok\");\n}\n")
	_, _, code := runCapture(t, "-exit0", p)
	if code != 0 {
		t.Fatalf("-exit0 应退出 0，got %d", code)
	}
}

func TestCLIJSONOutput(t *testing.T) {
	p := writeFile(t, t.TempDir(), "unused.qk", "fn main(IOStream io) {\n    int unused = 1;\n    io.println(\"ok\");\n}\n")
	out, _, _ := runCapture(t, "-json", p)
	var got []finding
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("JSON 解析失败: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].Code != "QK101" || got[0].Line != 2 {
		t.Fatalf("JSON 内容不符: %+v", got)
	}
	if got[0].File != p {
		t.Errorf("JSON file 应为 %s，got %s", p, got[0].File)
	}
}

func TestCLIJSONCleanIsEmptyArray(t *testing.T) {
	p := writeFile(t, t.TempDir(), "clean.qk", cleanSrc)
	out, _, _ := runCapture(t, "-json", p)
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("干净文件 JSON 应为 []，got %q", out)
	}
}

func TestCLIParseErrorReported(t *testing.T) {
	p := writeFile(t, t.TempDir(), "bad.qk", "fn main(IOStream io) {\n    int x = ;\n}\n")
	out, _, code := runCapture(t, p)
	if code != 1 {
		t.Fatalf("语法错误应退出 1，got %d", code)
	}
	if !strings.Contains(out, "ParseError") || !strings.Contains(out, ":2:") {
		t.Errorf("应报 ParseError 且带行列：%q", out)
	}
}

func TestCLITypeErrorReported(t *testing.T) {
	p := writeFile(t, t.TempDir(), "type.qk", "fn main(IOStream io) {\n    int x = \"s\";\n    io.println(x);\n}\n")
	out, _, code := runCapture(t, p)
	if code != 1 {
		t.Fatalf("类型错误应退出 1，got %d", code)
	}
	if !strings.Contains(out, "error:") || !strings.Contains(out, "cannot assign") {
		t.Errorf("应报类型错误：%q", out)
	}
}

func TestCLIUsageErrors(t *testing.T) {
	if _, _, code := runCapture(t); code != 2 {
		t.Errorf("无参数应退出 2，got %d", code)
	}
	if _, _, code := runCapture(t, "-nope", "x.qk"); code != 2 {
		t.Errorf("未知参数应退出 2，got %d", code)
	}
	if _, _, code := runCapture(t, "-L"); code != 2 {
		t.Errorf("-L 缺参数应退出 2，got %d", code)
	}
	if out, _, code := runCapture(t, "--version"); code != 0 || !strings.Contains(out, "qkcheck") {
		t.Errorf("--version 应退出 0 且打印版本，got %d %q", code, out)
	}
	if out, _, code := runCapture(t, "-h"); code != 0 || !strings.Contains(out, "usage: qkcheck") {
		t.Errorf("-h 应打印用法，got %d %q", code, out)
	}
}

// TestCLIImportPath 验证 -L：跨目录 import 在给定搜索目录后可解析；
// 且错误位置能映射回导入语句所在行。
func TestCLIImportPath(t *testing.T) {
	root := t.TempDir()
	libDir := filepath.Join(root, "libs")
	appDir := filepath.Join(root, "app")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, libDir, "mathlib.qk", "program library;\npub fn double(int n) int {\n    return n * 2;\n}\n")
	app := writeFile(t, appDir, "main.qk", "program main;\nimport \"mathlib\";\n\nfn main(IOStream io) {\n    io.println(double(21));\n}\n")

	out, _, code := runCapture(t, app)
	if code != 1 || !strings.Contains(out, "ImportError") {
		t.Fatalf("无 -L 时应报 ImportError 且退出 1，got code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "main.qk:2:1") {
		t.Errorf("ImportError 应定位到 import 语句（第 2 行），got %q", out)
	}

	out, _, code = runCapture(t, "-L", libDir, app)
	if code != 0 {
		t.Fatalf("带 -L 应检查通过，got code=%d out=%q", code, out)
	}
}

// TestCLICorpusNoFalsePositives 全仓语料门禁：
//   - 不允许任何 error（语法/类型/import 全部通过）；
//   - 警告必须与已人工复核的快照逐条一致（新增警告 = 新误报，需要复核）。
func TestCLICorpusNoFalsePositives(t *testing.T) {
	root := filepath.Join("..", "..")
	var files []string
	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if fi.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".qk") || strings.HasSuffix(p, ".kq") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 10 {
		t.Fatalf("语料过少（%d 个文件），测试无效", len(files))
	}
	sort.Strings(files)

	out, errOut, code := runCapture(t, append([]string{"-json"}, files...)...)
	if code != 1 && code != 0 {
		t.Fatalf("语料检查异常退出 %d：%s", code, errOut)
	}
	var got []finding
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}

	// 1) 不允许 error
	for _, f := range got {
		if f.Sev == "error" {
			t.Errorf("语料出现错误级诊断（不应存在）：%s", f)
		}
	}

	// 2) 警告快照：均为人工复核过的真问题（见 README「实测」一节）
	wantWarnings := map[string]bool{
		"compiler/testdata/cases/a_for_break.kq:2:1 QK105":  true, // log n; 结束函数，声明 int 返回却给 nil
		"compiler/testdata/cases/a_for_break.kq:4:5 QK103":  true, // log 之后的 return 不可达
		"compiler/testdata/cases_run/t_table.kq:45:5 QK101": true, // for (String k : sk) 循环变量未使用
		"compiler/testdata/demo.qk:28:13 QK101":             true, // int bad = 10 / 0; 只用于触发异常
		"examples/trycatch.qk:3:13 QK101":                   true, // int a = 10 / 0; 同上
	}
	gotWarnings := map[string]bool{}
	for _, f := range got {
		rel, err := filepath.Rel(root, f.File)
		if err != nil {
			rel = f.File
		}
		gotWarnings[rel+":"+itoa(f.Line)+":"+itoa(f.Col)+" "+f.Code] = true
	}
	for k := range gotWarnings {
		if !wantWarnings[k] {
			t.Errorf("出现未复核的警告（可能为新误报）：%s", k)
		}
	}
	for k := range wantWarnings {
		if !gotWarnings[k] {
			t.Errorf("快照中的警告消失（检查能力回归？）：%s", k)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
