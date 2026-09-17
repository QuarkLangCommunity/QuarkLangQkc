package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCapture(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return out.String(), errb.String(), code
}

func TestQkreplEvalFlag(t *testing.T) {
	out, errOut, code := runCapture(t, "", "-e", "1 + 2 * 3")
	if code != 0 {
		t.Fatalf("应退出 0，got %d（%s）", code, errOut)
	}
	if strings.TrimSpace(out) != "7" {
		t.Errorf("应回显 7，got %q", out)
	}
}

func TestQkreplEvalFlagMultiple(t *testing.T) {
	out, _, code := runCapture(t, "", "-e", "int x = 5;", "-e", "x * 8")
	if code != 0 {
		t.Fatalf("应退出 0，got %d", code)
	}
	if !strings.Contains(out, "40") {
		t.Errorf("多段 -e 应共享环境，got %q", out)
	}
}

func TestQkreplEvalFlagError(t *testing.T) {
	_, errOut, code := runCapture(t, "", "-e", "1 / 0")
	if code != 1 {
		t.Fatalf("出错应退出 1，got %d", code)
	}
	if !strings.Contains(errOut, "error:") {
		t.Errorf("stderr 应有错误信息，got %q", errOut)
	}
}

func TestQkreplBatchMode(t *testing.T) {
	in := "int x = 21;\nx * 2\n"
	out, errOut, code := runCapture(t, in)
	if code != 0 {
		t.Fatalf("应退出 0，got %d（%s）", code, errOut)
	}
	if strings.Contains(out, "qk>") {
		t.Errorf("批处理不应打印提示符，got %q", out)
	}
	if strings.TrimSpace(out) != "42" {
		t.Errorf("应输出 42，got %q", out)
	}
}

func TestQkreplBatchMultiLineBlock(t *testing.T) {
	in := "fn double(int n) int {\n    return n * 2;\n}\ndouble(21)\n"
	out, errOut, code := runCapture(t, in)
	if code != 0 {
		t.Fatalf("应退出 0，got %d（%s）", code, errOut)
	}
	if !strings.Contains(out, "已定义 fn double(int n) int") {
		t.Errorf("应回显定义摘要，got %q", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("应输出 42，got %q", out)
	}
}

func TestQkreplBatchControlFlow(t *testing.T) {
	in := "int i = 0;\nwhile (i < 3) {\n    i = i + 1;\n}\ni\n"
	out, errOut, code := runCapture(t, in)
	if code != 0 {
		t.Fatalf("应退出 0，got %d（%s）", code, errOut)
	}
	if strings.TrimSpace(out) != "3" {
		t.Errorf("应输出 3，got %q", out)
	}
}

func TestQkreplBatchErrorContinues(t *testing.T) {
	in := "int a = 1;\nnope + 1\na + 41\n"
	out, errOut, code := runCapture(t, in)
	if code != 1 {
		t.Fatalf("有错误应退出 1，got %d", code)
	}
	if !strings.Contains(errOut, "error:") {
		t.Errorf("stderr 应有错误，got %q", errOut)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("出错后应继续求值，got %q", out)
	}
}

func TestQkreplIoOutput(t *testing.T) {
	out, errOut, code := runCapture(t, `io.println("hello", 42)`+"\n")
	if code != 0 {
		t.Fatalf("应退出 0，got %d（%s）", code, errOut)
	}
	if !strings.Contains(out, "hello 42") {
		t.Errorf("io.println 应写入 stdout，got %q", out)
	}
}

func TestQkreplHelpAndQuit(t *testing.T) {
	out, _, code := runCapture(t, ":help\n")
	if code != 0 || !strings.Contains(out, ":load") {
		t.Errorf(":help 应打印帮助，got code=%d %q", code, out)
	}
	out, _, code = runCapture(t, "int x = 1;\n:quit\nnope\n")
	if code != 0 {
		t.Errorf(":quit 后应正常退出，got %d", code)
	}
	if !strings.Contains(out, "qkrepl") == false && strings.Contains(out, "nope") {
		t.Errorf(":quit 之后的行不应求值，got %q", out)
	}
}

func TestQkreplUnknownCommand(t *testing.T) {
	_, errOut, code := runCapture(t, ":nope\n")
	if code != 1 {
		t.Fatalf("未知命令应退出 1，got %d", code)
	}
	if !strings.Contains(errOut, "未知命令") {
		t.Errorf("stderr 应说明未知命令，got %q", errOut)
	}
}

func TestQkreplLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "lib.qk")
	src := "fn triple(int n) int {\n    return n * 3;\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	in := ":load " + p + "\ntriple(14)\n"
	out, errOut, code := runCapture(t, in)
	if code != 0 {
		t.Fatalf("应退出 0，got %d（%s）", code, errOut)
	}
	if !strings.Contains(out, "已定义 fn triple(int n) int") {
		t.Errorf(":load 应登记定义，got %q", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("加载后应能调用，got %q", out)
	}
}

func TestQkreplLoadMissingFile(t *testing.T) {
	_, errOut, code := runCapture(t, ":load /nonexistent/x.qk\n")
	if code != 1 || !strings.Contains(errOut, "error:") {
		t.Errorf("缺文件应报错退出 1，got code=%d %q", code, errOut)
	}
}

func TestQkreplUsage(t *testing.T) {
	if _, _, code := runCapture(t, "", "-e"); code != 2 {
		t.Errorf("-e 缺参数应退出 2")
	}
	if _, _, code := runCapture(t, "", "-bogus"); code != 2 {
		t.Errorf("未知参数应退出 2")
	}
	if out, _, code := runCapture(t, "", "--version"); code != 0 || !strings.Contains(out, "qkrepl") {
		t.Errorf("--version 输出不对: %d %q", code, out)
	}
	if out, _, code := runCapture(t, "", "-h"); code != 0 || !strings.Contains(out, "usage: qkrepl") {
		t.Errorf("-h 输出不对: %d %q", code, out)
	}
}

func TestQkreplQuietFlag(t *testing.T) {
	out, _, code := runCapture(t, "1 + 1\n", "-q")
	if code != 0 {
		t.Fatalf("应退出 0，got %d", code)
	}
	if strings.Contains(out, "qkrepl") {
		t.Errorf("-q 不应打印横幅，got %q", out)
	}
}
