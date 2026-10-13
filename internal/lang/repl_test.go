package lang

import (
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
	"strings"
	"testing"
)

// newTestREPL builds a session with the output written to a string buffer.
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
		t.Errorf("the expression should echo 7, got %q", got)
	}
}

func TestREPLPersistentVariables(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if out, err := s.Eval("int x = 21;"); err != nil || out != "" {
		t.Fatalf("declaring a variable: out=%q err=%v", out, err)
	}
	out, err := s.Eval("x * 2")
	if err != nil {
		t.Fatalf("second Eval segment: %v", err)
	}
	if strings.TrimSpace(out) != "42" {
		t.Errorf("a variable should survive across segments, got %q", out)
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
		t.Errorf("the assignment should persist, got %q err=%v", out, err)
	}
}

func TestREPLFunctionDefinitionAndCall(t *testing.T) {
	withLocalizer(t, i18n.New(i18n.ZH)) // asserts the Chinese wording
	s, _ := newTestREPL(t, "")
	out, err := s.Eval("fn double(int n) int {\n    return n * 2;\n}")
	if err != nil {
		t.Fatalf("defining a function: %v", err)
	}
	if !strings.Contains(out, "已定义 fn double(int n) int") {
		t.Errorf("it should echo the definition summary, got %q", out)
	}
	got, err := s.Eval("double(21)")
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if strings.TrimSpace(got) != "42" {
		t.Errorf("the call should yield 42, got %q", got)
	}
}

// A cross-chunk call must be dispatched by name (FnIdx=-1), otherwise it would wrongly call the function registered last time.
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
		t.Errorf("caller should call bar and get 2, got %q", got)
	}
	// after foo is redefined it is still resolved by name
	if _, err := s.Eval("fn foo(int n) int {\n    return 111;\n}"); err != nil {
		t.Fatal(err)
	}
	if got, err = s.Eval("foo(0)"); err != nil || strings.TrimSpace(got) != "111" {
		t.Errorf("redefinition should yield 111, got %q err=%v", got, err)
	}
}

func TestREPLMultiStatementChunk(t *testing.T) {
	s, _ := newTestREPL(t, "")
	out, err := s.Eval("int i = 0;\nwhile (i < 3) {\n    i = i + 1;\n}\ni")
	if err != nil {
		t.Fatalf("several statements: %v", err)
	}
	if strings.TrimSpace(out) != "3" {
		t.Errorf("the multi-line block should evaluate to 3, got %q", out)
	}
}

func TestREPLStructAndImpl(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if _, err := s.Eval("type struct {\n    int v;\n} P;"); err != nil {
		t.Fatalf("defining a struct: %v", err)
	}
	if _, err := s.Eval("impl {\n    fn twice(P self) int {\n        return self.v * 2;\n    }\n} P;"); err != nil {
		t.Fatalf("defining an impl: %v", err)
	}
	out, err := s.Eval("P p = .{v: 21};\np.twice()")
	if err != nil {
		t.Fatalf("struct evaluation: %v", err)
	}
	if strings.TrimSpace(out) != "42" {
		t.Errorf("the method call should yield 42, got %q", out)
	}
}

func TestREPLRuntimeErrorKeepsSessionAlive(t *testing.T) {
	s, _ := newTestREPL(t, "")
	if _, err := s.Eval("int a = 1;"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Eval("int z = 1 / 0;"); err == nil {
		t.Error("division by zero should return an error")
	}
	out, err := s.Eval("a + 41")
	if err != nil {
		t.Fatalf("the session should survive an error: %v", err)
	}
	if strings.TrimSpace(out) != "42" {
		t.Errorf("evaluation after the error should be 42, got %q", out)
	}
}

func TestREPLParseErrorMessageLine(t *testing.T) {
	s, _ := newTestREPL(t, "")
	_, err := s.Eval("int a = 1;\nint b = ;")
	if err == nil {
		t.Fatal("a syntax error should return an error")
	}
	// the wrapper introduces a +1 line offset, which must be converted back to the user's input line number (line 2)
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("the error line should be translated to line 2, got %v", err)
	}
}

func TestREPLUndefinedIdentifier(t *testing.T) {
	s, _ := newTestREPL(t, "")
	_, err := s.Eval("nope + 1")
	if err == nil {
		t.Error("an undeclared identifier should be an error")
	}
}

func TestREPLIoAvailable(t *testing.T) {
	var out strings.Builder
	s, err := NewREPLSession(strings.NewReader(""), &out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Eval(`io.println("hello", 42)`); err != nil {
		t.Fatalf("io should be usable: %v", err)
	}
	if !strings.Contains(out.String(), "hello 42") {
		t.Errorf("io.println output should go to the stream, got %q", out.String())
	}
}

func TestREPLLogStatement(t *testing.T) {
	s, _ := newTestREPL(t, "")
	out, err := s.Eval(`log 6 * 7;`)
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("log should echo the recorded value, got %q", out)
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
		t.Fatalf("defining an interface: %v", err)
	}
	out, err := s.Eval("type struct {\n    int v;\n} Q;\nimpl {\n    fn get(Q self) int {\n        return self.v;\n    }\n} Q;")
	if err != nil {
		t.Fatalf("defining a type and an impl: %v", err)
	}
	if out == "" {
		t.Error("it should echo the definition summary")
	}
	got, err := s.Eval("Q q = .{v: 7};\nGetter g = q;\ng.get()")
	if err != nil {
		t.Fatalf("interface variable: %v", err)
	}
	if strings.TrimSpace(got) != "7" {
		t.Errorf("the interface method should return 7, got %q", got)
	}
}
