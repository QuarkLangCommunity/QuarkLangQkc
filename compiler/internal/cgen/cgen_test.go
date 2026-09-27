package cgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// transpile compiles canonical-syntax source (filename fixed to test.qk, no import resolution).
func transpile(t *testing.T, src string) string {
	t.Helper()
	ir, err := Transpile(src, "test.qk")
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	return ir
}

// testRuntime is the minimal ql_strcat impl used by the lli unit tests (qkc -run gets it from qthreads.c;
// llvm-as/lli does not link it, so the tests add a definition via libc strlen/memcpy/malloc).
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

// withTestRuntime strips the module's ql_strcat declaration and appends the test runtime definition.
func withTestRuntime(ir string) string {
	ir = strings.Replace(ir, "declare i8* @ql_strcat(i8*, i8*)\n", "", 1)
	return ir + "\n" + testRuntime
}

// lliRun runs IR with lli and returns stdout (skips the test when lli is missing).
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
	// Constant folding: 1 + 2 * 3 → 7 (computed at compile time, no arithmetic instructions in the IR)
	if strings.Contains(ir, "mul i32") || strings.Contains(ir, "add i32") {
		t.Fatalf("constant folding failed:\n%s", ir)
	}
	if !strings.Contains(ir, "@.str1") {
		t.Fatalf("missing format string:\n%s", ir)
	}
}

// Full pipeline: QuarkLang → LLVM IR → lli execution (when the LLVM toolchain is available)
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

// llvm-as syntax check: the generated IR must pass LLVM's official syntax check
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
		// Parameter assignment: LLVM SSA parameter registers are not writable, so they must be promoted to alloca slots
		"fn f(int n) void {\n    while (n > 0) { n = n - 1; }\n}\nfn main(IOStream io) { f(3); io.println(1); }\n",
		// List methods/indexing: must not emit load i8* for {i32*, i32}*
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

// Unary minus + modulo (full lli pipeline)
func TestUnaryMinusAndModulo(t *testing.T) {
	got := lliRun(t, transpile(t, "fn main(IOStream io) {\n    io.println(-5 + 3, 7 % 3);\n}\n"))
	if got != "-2 1\n" {
		t.Fatalf("got %q", got)
	}
}

// String escapes (\n, \") → IR escapes → restored at runtime
func TestStringEscapes(t *testing.T) {
	got := lliRun(t, transpile(t, "fn main(IOStream io) {\n    io.println(\"a\\nb\", \"q\\\"q\");\n}\n"))
	if got != "a\nb q\"q\n" {
		t.Fatalf("got %q", got)
	}
}

// Variables + if/else + while + comparison + bool + List (full lli pipeline, canonical syntax)
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

// Multiple functions + recursion (full lli pipeline: the compiled path targets C-level performance)
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

// Assigned parameter: must be promoted to alloca, with behavior matching the interpreter
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

// Declaration without initializer + later assignment (canonical K3: the initializer may be omitted)
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

// String-returning functions + int/bool.toString(): i8* signature/return, concatenation and printing on the LLVM side
func TestStringReturnAndToString(t *testing.T) {
	src := "fn value(int n) String { return n.toString(); }\n" +
		"fn label(int n) String { return \"x = \" + value(n); }\n" +
		"fn tail(int n) String { return value(n); }\n" + // String tail call
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

// Parent-task repro 1): int.toString() in String concatenation (type inference must recognize String-returning builtin methods)
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

// import: same-directory .qk (including nested imports), with the same recursive merge semantics as the interpreter
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

// Non-canonical syntax (name first) must report an explicit error instead of being silently accepted
func TestRejectsLegacySyntax(t *testing.T) {
	_, err := Transpile("fn main(io IOStream) {\n    io.println(1);\n}\n", "legacy.qk")
	if err == nil {
		t.Fatal("legacy name-first syntax must fail")
	}
	// Rejected directly by the language frontend (typecheck): "io" is taken as a type name → unknown type IOStream
	if !strings.Contains(err.Error(), "IOStream") && !strings.Contains(err.Error(), "unknown type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
