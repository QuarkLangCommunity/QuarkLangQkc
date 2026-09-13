package cgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// transpile 编译正典语法源码（filename 固定 test.qk，无 import 解析）。
func transpile(t *testing.T, src string) string {
	t.Helper()
	ir, err := Transpile(src, "test.qk")
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	return ir
}

// testRuntime 是 lli 单测用的最小 ql_strcat 实现（qkc -run 由 qthreads.c 提供；
// llvm-as/lli 不会链接它，故测试时用 libc 的 strlen/memcpy/malloc 补一个定义）。
const testRuntime = `declare i64 @strlen(i8*)
declare i8* @memcpy(i8*, i8*, i64)
define i8* @ql_strcat(i8* %a, i8* %b) {
entry:
  %la = call i64 @strlen(i8* %a)
  %lb = call i64 @strlen(i8* %b)
  %n = add i64 %la, %lb
  %n1 = add i64 %n, 1
  %buf = call i8* @malloc(i64 %n1)
  %x = call i8* @memcpy(i8* %buf, i8* %a, i64 %la)
  %p = getelementptr inbounds i8, i8* %buf, i64 %la
  %l1 = add i64 %lb, 1
  %y = call i8* @memcpy(i8* %p, i8* %b, i64 %l1)
  ret i8* %buf
}
`

// withTestRuntime 去掉模块里的 ql_strcat 声明并追加测试运行时定义。
func withTestRuntime(ir string) string {
	ir = strings.Replace(ir, "declare i8* @ql_strcat(i8*, i8*)\n", "", 1)
	return ir + "\n" + testRuntime
}

// lliRun 用 lli 执行 IR 并返回 stdout（无 lli 时跳过测试）。
func lliRun(t *testing.T, ir string) string {
	t.Helper()
	lli, err := exec.LookPath("lli")
	if err != nil {
		t.Skip("lli not available")
	}
	f, err := os.CreateTemp("", "quark-test-*.ll")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	if _, err := f.WriteString(ir); err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(name)
	out, err := exec.Command(lli, name).CombinedOutput()
	if err != nil {
		t.Fatalf("lli: %v\nIR:\n%s", err, ir)
	}
	return string(out)
}

func TestHelloIR(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n    io.println(\"Hello World!\");\n}\n")
	if !strings.Contains(ir, "define i32 @main()") {
		t.Fatalf("missing main:\n%s", ir)
	}
	if !strings.Contains(ir, "declare i32 @printf") {
		t.Fatalf("missing printf decl:\n%s", ir)
	}
	if !strings.Contains(ir, "call i32 (i8*, ...) @printf") {
		t.Fatalf("missing printf call:\n%s", ir)
	}
	if !strings.Contains(ir, "c\"Hello World!\\00\"") {
		t.Fatalf("missing string constant:\n%s", ir)
	}
}

func TestArithmeticIR(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n    io.println(1 + 2 * 3, \"=\");\n}\n")
	// 常量折叠：1 + 2 * 3 → 7（编译期算掉，IR 无算术指令）
	if strings.Contains(ir, "mul i32") || strings.Contains(ir, "add i32") {
		t.Fatalf("constant folding failed:\n%s", ir)
	}
	if !strings.Contains(ir, "@.str1") {
		t.Fatalf("missing format string:\n%s", ir)
	}
}

// 全链路：QuarkLang → LLVM IR → lli 执行（本机有 LLVM 工具链时）
func TestLLIPipeline(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"fn main(IOStream io) {\n    io.println(\"Hello World!\");\n}\n", "Hello World!\n"},
		{"fn main(IOStream io) {\n    io.println(1 + 2 * 3, \"=\");\n}\n", "7 =\n"},
	}
	for _, c := range cases {
		got := lliRun(t, transpile(t, c.src))
		if got != c.want {
			t.Fatalf("got %q want %q", got, c.want)
		}
	}
}

// llvm-as 语法校验：生成的 IR 必须通过 LLVM 官方语法检查
func TestIRSyntax(t *testing.T) {
	llvmAs, err := exec.LookPath("llvm-as")
	if err != nil {
		t.Skip("llvm-as not available")
	}
	progs := []string{
		"fn main(IOStream io) {\n    io.println(\"Hello World!\");\n}\n",
		"fn main(IOStream io) {\n    io.println(1 + 2 * 3, \"=\");\n}\n",
		"fn main(IOStream io) {\n    io.println(-5 + 3, 7 % 3);\n}\n",
		"fn main(IOStream io) {\n    io.println(\"a\\nb\", \"q\\\"q\");\n}\n",
		// 参数赋值：LLVM SSA 参数寄存器不可写，必须提升为 alloca 槽
		"fn f(int n) void {\n    while (n > 0) { n = n - 1; }\n}\nfn main(IOStream io) { f(3); io.println(1); }\n",
		// List 方法/下标：不能对 {i32*, i32}* 误发 load i8*
		"fn main(IOStream io) {\n    List<int> l = [1, 2, 3];\n    l.append(4);\n    l[0] = 9;\n    io.println(l.size(), l[2]);\n}\n",
	}
	for _, src := range progs {
		ir := transpile(t, src)
		f, err := os.CreateTemp("", "quark-syn-*.ll")
		if err != nil {
			t.Fatal(err)
		}
		name := f.Name()
		if _, err := f.WriteString(ir); err != nil {
			t.Fatal(err)
		}
		f.Close()
		defer os.Remove(name)
		if out, err := exec.Command(llvmAs, name, "-o", os.DevNull).CombinedOutput(); err != nil {
			t.Fatalf("llvm-as rejected IR: %v\n%s\nIR:\n%s", err, out, ir)
		}
	}
}

// 一元负号 + 取模（lli 全链路）
func TestUnaryMinusAndModulo(t *testing.T) {
	got := lliRun(t, transpile(t, "fn main(IOStream io) {\n    io.println(-5 + 3, 7 % 3);\n}\n"))
	if got != "-2 1\n" {
		t.Fatalf("got %q", got)
	}
}

// 字符串转义（\n、\"）→ IR 转义 → 运行时还原
func TestStringEscapes(t *testing.T) {
	got := lliRun(t, transpile(t, "fn main(IOStream io) {\n    io.println(\"a\\nb\", \"q\\\"q\");\n}\n"))
	if got != "a\nb q\"q\n" {
		t.Fatalf("got %q", got)
	}
}

// 变量 + if/else + while + 比较 + 布尔 + List（lli 全链路，正典语法）
func TestVariablesAndControlFlow(t *testing.T) {
	src := "fn main(IOStream io) {\n" +
		"    int x = 5;\n" +
		"    int y = x * 2 + 1;\n" +
		"    io.println(y);\n" +
		"    if (y > 10) {\n" +
		"        io.println(\"big\");\n" +
		"    } else {\n" +
		"        io.println(\"small\");\n" +
		"    }\n" +
		"    int n = 0;\n" +
		"    int i = 1;\n" +
		"    while (i <= 5) {\n" +
		"        n = n + i;\n" +
		"        i = i + 1;\n" +
		"    }\n" +
		"    io.println(n);\n" +
		"    io.println(3 == 3, 3 != 4, 2 < 1);\n" +
		"    io.println(true && false, true || false, !true);\n" +
		"    String s = \"hi\";\n" +
		"    io.println(s);\n" +
		"    List<int> l = [1, 2, 3];\n" +
		"    io.println(l.size(), l[2]);\n" +
		"    l.append(4);\n" +
		"    l[0] = 9;\n" +
		"    io.println(l.size(), l[0]);\n" +
		"}\n"
	got := lliRun(t, transpile(t, src))
	want := "11\nbig\n15\ntrue true false\nfalse true false\nhi\n3 3\n4 9\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// 多函数 + 递归调用（lli 全链路：编译路径性能对标 C）
func TestFunctionCallAndRecursion(t *testing.T) {
	src := "fn fib(int n) int {\n" +
		"    if (n < 2) {\n" +
		"        return n;\n" +
		"    }\n" +
		"    return fib(n - 1) + fib(n - 2);\n" +
		"}\n" +
		"\n" +
		"fn main(IOStream io) {\n" +
		"    io.println(fib(10));\n" +
		"}\n"
	got := lliRun(t, transpile(t, src))
	if got != "55\n" {
		t.Fatalf("got %q", got)
	}
}

// 参数被赋值：必须提升为 alloca，并且行为与解释器一致
func TestParamAssignment(t *testing.T) {
	src := "fn down(int n) int {\n" +
		"    int acc = 0;\n" +
		"    while (n > 0) {\n" +
		"        acc = acc + n;\n" +
		"        n = n - 1;\n" +
		"    }\n" +
		"    return acc;\n" +
		"}\n" +
		"fn main(IOStream io) {\n" +
		"    io.println(down(5));\n" +
		"}\n"
	got := lliRun(t, transpile(t, src))
	if got != "15\n" {
		t.Fatalf("got %q", got)
	}
}

// 无初值声明 + 后续赋值（正典 K3：初始化可省略）
func TestDeclareWithoutInit(t *testing.T) {
	src := "fn main(IOStream io) {\n" +
		"    int x;\n" +
		"    x = 3;\n" +
		"    String s;\n" +
		"    s = \"ok\";\n" +
		"    io.println(x, s);\n" +
		"}\n"
	got := lliRun(t, transpile(t, src))
	if got != "3 ok\n" {
		t.Fatalf("got %q want %q", got, "3 ok\n")
	}
}

// String 返回函数 + int/bool.toString()：LLVM 侧 i8* 签名/返回、拼接、打印
func TestStringReturnAndToString(t *testing.T) {
	src := "fn value(int n) String { return n.toString(); }\n" +
		"fn label(int n) String { return \"x = \" + value(n); }\n" +
		"fn tail(int n) String { return value(n); }\n" + // String 尾调用
		"fn main(IOStream io) {\n" +
		"    io.println(label(5));\n" +
		"    io.println(tail(9));\n" +
		"    String s = label(1) + label(2);\n" +
		"    io.println(s);\n" +
		"    io.println((-42).toString());\n" +
		"}\n"
	ir := transpile(t, src)
	for _, want := range []string{"define i8* @value(", "define i8* @label(", "ret i8* ", "call i8* @ql_int_to_str"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %q in IR:\n%s", want, ir)
		}
	}
	want := "x = 5\n9\nx = 1x = 2\n-42\n"
	if got := lliRun(t, withTestRuntime(ir)); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// 父任务复现 1)：int.toString() 参与 String 拼接（类型推断必须识别 String 返回内建方法）
func TestToStringConcatRepro(t *testing.T) {
	src := "program main;\nfn main(IOStream io) { int x = 5; io.println(\"x = \" + x.toString()); }\n"
	ir := transpile(t, src)
	if !strings.Contains(ir, "call i8* @ql_int_to_str") || !strings.Contains(ir, "call i8* @ql_strcat") {
		t.Fatalf("missing toString/strcat call:\n%s", ir)
	}
	if got := lliRun(t, withTestRuntime(ir)); got != "x = 5\n" {
		t.Fatalf("got %q want %q", got, "x = 5\n")
	}
}

// import：同目录 .qk（含嵌套 import），与解释器同一套递归合并语义
func TestImportCompile(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("util.qk", "pub fn triple(int n) int {\n    return n * 3;\n}\n")
	write("mathlib.qk", "import \"util\";\npub fn double(int n) int {\n    return n * 2;\n}\n")
	src := "import \"mathlib\";\nfn main(IOStream io) {\n    io.println(double(21), triple(14));\n}\n"
	main := filepath.Join(dir, "main.qk")
	ir, err := Transpile(src, main)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	for _, want := range []string{"define i32 @double", "define i32 @triple", "define i32 @main()"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %q in IR:\n%s", want, ir)
		}
	}
	if got := lliRun(t, ir); got != "42 42\n" {
		t.Fatalf("got %q want %q", got, "42 42\n")
	}
}

// 非正典写法（名字在前）必须报明确错误，而不是被静默接受
func TestRejectsLegacySyntax(t *testing.T) {
	_, err := Transpile("fn main(io IOStream) {\n    io.println(1);\n}\n", "legacy.qk")
	if err == nil {
		t.Fatal("legacy name-first syntax must fail")
	}
	// 由语言前端（typecheck）直接拒绝："io" 被当成类型名 → 未知类型 IOStream
	if !strings.Contains(err.Error(), "IOStream") && !strings.Contains(err.Error(), "unknown type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
