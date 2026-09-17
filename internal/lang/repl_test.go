package lang

import (
	"strings"
	"testing"
)

// newTestREPL 建会话，输出写入字符串缓冲。
func newTestREPL(t *testing.T, stdin string) (*REPLSession, *strings.Builder) {
	t.Helper()
	var out strings.Builder
	s, err := NewREPLSession(strings.NewReader(stdin), &out)
	if err != nil {
		t.Fatalf("NewREPLSession: %v", err)
	}
	return s, &out
}

func TestREPLExpressionEcho(t *testing.T) {
	s, _ := newTestREPL(t, "")
	got, err := s.Eval("1 + 2 * 3")
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if strings.TrimSpace(got) != "7" {
		t.Errorf("表达式回显应为 7，got %q", got)
	}
}

func TestREPLPersistentVariables(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if out, err := s.Eval("int x = 21;"); err != nil || out != "" {
		t.Fatalf("声明变量: out=%q err=%v", out, err)
	}
	out, err := s.Eval("x * 2")
	if err != nil {
		t.Fatalf("Eval 第二段: %v", err)
	}
	if strings.TrimSpace(out) != "42" {
		t.Errorf("跨段变量应存活，got %q", out)
	}
}

func TestREPLAssignmentPersists(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if _, err := s.Eval("int a = 1;"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Eval("a = a + 41;"); err != nil {
		t.Fatal(err)
	}
	out, err := s.Eval("a")
	if err != nil || strings.TrimSpace(out) != "42" {
		t.Errorf("赋值应持久，got %q err=%v", out, err)
	}
}

func TestREPLFunctionDefinitionAndCall(t *testing.T) {
	s, _ := newTestREPL(t, "")
	out, err := s.Eval("fn double(int n) int {\n    return n * 2;\n}")
	if err != nil {
		t.Fatalf("定义函数: %v", err)
	}
	if !strings.Contains(out, "已定义 fn double(int n) int") {
		t.Errorf("应回显定义摘要，got %q", out)
	}
	got, err := s.Eval("double(21)")
	if err != nil {
		t.Fatalf("调用: %v", err)
	}
	if strings.TrimSpace(got) != "42" {
		t.Errorf("调用结果应为 42，got %q", got)
	}
}

// 跨段调用必须是按名字派发（FnIdx=-1），否则会错调到上一次登记的函数。
func TestREPLCrossChunkDispatch(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if _, err := s.Eval("fn foo(int n) int {\n    return 1;\n}"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Eval("fn bar(int n) int {\n    return 2;\n}"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Eval("fn caller(int n) int {\n    return bar(n);\n}"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Eval("caller(0)")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "2" {
		t.Errorf("caller 应调用 bar 得 2，got %q", got)
	}
	// 重新定义 foo 后仍按名字解析
	if _, err := s.Eval("fn foo(int n) int {\n    return 111;\n}"); err != nil {
		t.Fatal(err)
	}
	if got, err = s.Eval("foo(0)"); err != nil || strings.TrimSpace(got) != "111" {
		t.Errorf("重定义后应得 111，got %q err=%v", got, err)
	}
}

func TestREPLMultiStatementChunk(t *testing.T) {
	s, _ := newTestREPL(t, "")
	out, err := s.Eval("int i = 0;\nwhile (i < 3) {\n    i = i + 1;\n}\ni")
	if err != nil {
		t.Fatalf("多语句: %v", err)
	}
	if strings.TrimSpace(out) != "3" {
		t.Errorf("多行块求值结果应为 3，got %q", out)
	}
}

func TestREPLStructAndImpl(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if _, err := s.Eval("type struct {\n    int v;\n} P;"); err != nil {
		t.Fatalf("定义结构体: %v", err)
	}
	if _, err := s.Eval("impl {\n    fn twice(P self) int {\n        return self.v * 2;\n    }\n} P;"); err != nil {
		t.Fatalf("定义 impl: %v", err)
	}
	out, err := s.Eval("P p = .{v: 21};\np.twice()")
	if err != nil {
		t.Fatalf("结构体求值: %v", err)
	}
	if strings.TrimSpace(out) != "42" {
		t.Errorf("方法调用应为 42，got %q", out)
	}
}

func TestREPLRuntimeErrorKeepsSessionAlive(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if _, err := s.Eval("int a = 1;"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Eval("int z = 1 / 0;"); err == nil {
		t.Error("除零应返回错误")
	}
	out, err := s.Eval("a + 41")
	if err != nil {
		t.Fatalf("出错后会话应存活: %v", err)
	}
	if strings.TrimSpace(out) != "42" {
		t.Errorf("出错后求值应为 42，got %q", out)
	}
}

func TestREPLParseErrorMessageLine(t *testing.T) {
	s, _ := newTestREPL(t, "")
	_, err := s.Eval("int a = 1;\nint b = ;")
	if err == nil {
		t.Fatal("语法错误应返回错误")
	}
	// 包装带来 +1 行偏移，必须换算回用户输入的行号（第 2 行）
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("错误行号应换算为第 2 行，got %v", err)
	}
}

func TestREPLUndefinedIdentifier(t *testing.T) {
	s, _ := newTestREPL(t, "")
	_, err := s.Eval("nope + 1")
	if err == nil {
		t.Error("未定义标识符应报错")
	}
}

func TestREPLIoAvailable(t *testing.T) {
	var out strings.Builder
	s, err := NewREPLSession(strings.NewReader(""), &out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Eval(`io.println("hello", 42)`); err != nil {
		t.Fatalf("io 应可用: %v", err)
	}
	if !strings.Contains(out.String(), "hello 42") {
		t.Errorf("io.println 输出应写入流，got %q", out.String())
	}
}

func TestREPLLogStatement(t *testing.T) {
	s, _ := newTestREPL(t, "")
	out, err := s.Eval(`log 6 * 7;`)
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("log 应回显记录值，got %q", out)
	}
}

func TestREPLIncomplete(t *testing.T) {
	s, _ := newTestREPL(t, "")
	cases := []struct {
		src  string
		want bool
	}{
		{"if (true) {", true},
		{"if (true) {\n    int a = 1;\n", true},
		{"fn f(int a) int {", true},
		{`String s = "abc`, true},
		{"if (true) {\n    int a = 1;\n}", false},
		{"int a = 1;", false},
		{"fn f(int a) int {\n    return a;\n}", false},
		{"1 + 2", false},
		{"", false},
	}
	for _, c := range cases {
		if got := s.Incomplete(c.src); got != c.want {
			t.Errorf("Incomplete(%q) = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestREPLInterfaceDef(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if _, err := s.Eval("type interface {\n    fn get(Self self) int;\n} Getter;"); err != nil {
		t.Fatalf("定义接口: %v", err)
	}
	out, err := s.Eval("type struct {\n    int v;\n} Q;\nimpl {\n    fn get(Q self) int {\n        return self.v;\n    }\n} Q;")
	if err != nil {
		t.Fatalf("定义类型+impl: %v", err)
	}
	if out == "" {
		t.Error("应回显定义摘要")
	}
	got, err := s.Eval("Q q = .{v: 7};\nGetter g = q;\ng.get()")
	if err != nil {
		t.Fatalf("接口变量: %v", err)
	}
	if strings.TrimSpace(got) != "7" {
		t.Errorf("接口方法应为 7，got %q", got)
	}
}
