package lang

import (
	"strings"
	"testing"
)

func lintSrc(t *testing.T, src string, opts LintOptions) []Diag {
	t.Helper()
	prog, err := ParseSource(src)
	if err != nil {
		t.Fatalf("ParseSource: %v\n源码:\n%s", err, src)
	}
	return Lint(prog, opts)
}

// wantCodes 断言诊断码集合（顺序无关，含出现次数）。
func wantCodes(t *testing.T, diags []Diag, want map[string]int) {
	t.Helper()
	got := map[string]int{}
	for _, d := range diags {
		got[d.Code]++
	}
	for code, n := range want {
		if got[code] != n {
			t.Errorf("诊断码 %s: got %d 条, want %d 条；实际诊断：", code, got[code], n)
			for _, d := range diags {
				t.Logf("  %s", d)
			}
			return
		}
	}
	for code, n := range got {
		if _, ok := want[code]; !ok {
			t.Errorf("出现未预期的诊断码 %s（%d 条）：", code, n)
			for _, d := range diags {
				t.Logf("  %s", d)
			}
		}
	}
}

func TestLintUnusedVar(t *testing.T) {
	src := `fn main(IOStream io) {
    int unused1 = 1;
    int used = 2;
    io.println(used);
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{CodeUnusedVar: 1})
}

func TestLintUnusedVarAssignOnly(t *testing.T) {
	src := `fn main(IOStream io) {
    int acc = 0;
    acc = 5;
    io.println("x");
}
`
	d := lintSrc(t, src, LintOptions{})
	// 声明初值 0 从未被读取、随后被 5 覆盖 → QK101（未使用）+ QK113（死存储）各一条
	wantCodes(t, d, map[string]int{CodeUnusedVar: 1, CodeDeadStore: 1})
	var sawAssignOnly, sawDeadStore bool
	for _, x := range d {
		if x.Code == CodeUnusedVar && strings.Contains(x.Msg, "只被赋值") {
			sawAssignOnly = true
		}
		if x.Code == CodeDeadStore {
			sawDeadStore = true
		}
	}
	if !sawAssignOnly {
		t.Errorf("应含「只被赋值」文案: %+v", d)
	}
	if !sawDeadStore {
		t.Errorf("应含死存储诊断: %+v", d)
	}
}

func TestLintUsedVarNoDiag(t *testing.T) {
	src := `fn main(IOStream io) {
    int a = 1;
    int b = a + 1;
    log b;
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{})
}

func TestLintDeleteCountsAsUse(t *testing.T) {
	src := `fn main(IOStream io) {
    List<int> l = [1, 2];
    delete l;
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{})
}

func TestLintUnderscoreSkipped(t *testing.T) {
	src := `fn main(IOStream io) {
    int _ignored = 1;
    int _ = 2;
    io.println("ok");
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{})
}

func TestLintUnusedParam(t *testing.T) {
	src := `fn helper(int a, int b) int {
    return a;
}
fn main(IOStream io) {
    io.println(helper(1, 2));
}
`
	// 默认不查形参
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{})
	// -params 打开后报 b
	wantCodes(t, lintSrc(t, src, LintOptions{Params: true}), map[string]int{CodeUnusedParam: 1})
}

func TestLintUnreachableAfterReturn(t *testing.T) {
	src := `fn f(int x) int {
    return 1;
    int dead = x + 1;
    return dead;
}
fn main(IOStream io) {
    io.println(f(1));
}
`
	d := lintSrc(t, src, LintOptions{})
	wantCodes(t, d, map[string]int{CodeUnreachable: 1})
	if d[0].Pos.Line != 3 {
		t.Errorf("不可达诊断应在第 3 行，got %d", d[0].Pos.Line)
	}
}

func TestLintUnreachableAfterIfElseBothReturn(t *testing.T) {
	src := `fn f(int x) int {
    if (x > 0) {
        return 1;
    } else {
        return 2;
    }
    return 3;
}
fn main(IOStream io) {
    io.println(f(1));
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{CodeUnreachable: 1})
}

func TestLintUnreachableInLoopBody(t *testing.T) {
	src := `fn main(IOStream io) {
    int i = 0;
    while (i < 3) {
        break;
        io.println("dead");
    }
    io.println("live");
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{CodeUnreachable: 1})
}

func TestLintShadowForC(t *testing.T) {
	src := `fn main(IOStream io) {
    int i = 9;
    for (int i = 0; i < 2; i = i + 1) {
        io.println(i);
    }
    io.println(i);
}
`
	d := lintSrc(t, src, LintOptions{})
	wantCodes(t, d, map[string]int{CodeShadow: 1})
	if !strings.Contains(d[0].Msg, "第 2 行") {
		t.Errorf("遮蔽诊断应指认外层第 2 行，got %q", d[0].Msg)
	}
}

func TestLintShadowForIn(t *testing.T) {
	src := `fn main(IOStream io) {
    List<int> x = [1, 2];
    for (int x : x) {
        io.println(x);
    }
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{CodeShadow: 1})
}

func TestLintShadowCatch(t *testing.T) {
	src := `fn main(IOStream io) {
    String e = "outer";
    try {
        io.println(e);
    } catch (void e) {
        io.println(e);
    }
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{CodeShadow: 1})
}

// if/while 体与函数体共用作用域（typecheck 语义）→ 内层重名是 duplicate 硬错误，
// qkcheck 不重复报遮蔽（避免与编译器文案冲突）。
func TestLintNoShadowForFlatScope(t *testing.T) {
	src := `fn f(int x) int {
    if (x > 0) {
        return 1;
    }
    return 2;
}
fn main(IOStream io) {
    io.println(f(1));
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{})
}

func TestLintMissingReturnFallthrough(t *testing.T) {
	src := `fn f(int x) int {
    if (x > 0) {
        return 1;
    }
}
fn main(IOStream io) {
    io.println(f(5));
}
`
	d := lintSrc(t, src, LintOptions{})
	wantCodes(t, d, map[string]int{CodeMissingRet: 1})
	if !strings.Contains(d[0].Msg, "执行到函数末尾") {
		t.Errorf("文案应指出执行到函数末尾，got %q", d[0].Msg)
	}
}

func TestLintMissingReturnViaLog(t *testing.T) {
	src := `fn f(int x) int {
    log x;
}
fn main(IOStream io) {
    io.println(f(7));
}
`
	d := lintSrc(t, src, LintOptions{})
	wantCodes(t, d, map[string]int{CodeMissingRet: 1})
	if !strings.Contains(d[0].Msg, "log") {
		t.Errorf("文案应指出 log 路径，got %q", d[0].Msg)
	}
}

func TestLintReturnCompleteNoDiag(t *testing.T) {
	src := `fn f(int x) int {
    if (x > 0) {
        return 1;
    } else {
        return 2;
    }
}
fn g(int x) int {
    while (x > 0) {
        return x;
    }
    return 0;
}
fn v(int x) void {
    if (x > 0) {
        return;
    }
}
fn main(IOStream io) {
    io.println(f(1) + g(2));
    v(3);
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{})
}

func TestLintIfaceNearMiss(t *testing.T) {
	src := `type interface { fn foo(Self self) int; fn bar(Self self) int; } I;
type struct { int v; } P;
impl { fn foo(P self) int { return self.v; } } P;
fn main(IOStream io) {
    P p = .{v: 3};
    io.println(p.foo());
}
`
	d := lintSrc(t, src, LintOptions{})
	wantCodes(t, d, map[string]int{CodeIfaceMissing: 1})
	if !strings.Contains(d[0].Msg, "bar") {
		t.Errorf("诊断应列出缺失方法 bar，got %q", d[0].Msg)
	}
}

func TestLintIfaceCompleteNoDiag(t *testing.T) {
	src := `type interface { fn foo(Self self) int; fn bar(Self self) int; } I;
type struct { int v; } P;
impl {
    fn foo(P self) int { return self.v; }
    fn bar(P self) int { return self.v + 1; }
} P;
fn main(IOStream io) {
    P p = .{v: 3};
    io.println(p.foo());
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{})
}

func TestLintVoidAsValue(t *testing.T) {
	src := `fn noret(int x) void {
    if (x > 0) {
        return;
    }
}
fn main(IOStream io) {
    int y = noret(1);
    io.println(y);
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{CodeVoidAsValue: 1})
}

func TestLintVoidCallStatementNoDiag(t *testing.T) {
	src := `fn noret(int x) void {
    if (x > 0) {
        return;
    }
}
fn main(IOStream io) {
    noret(1);
    io.println("ok");
}
`
	wantCodes(t, lintSrc(t, src, LintOptions{}), map[string]int{})
}

func TestLintCodesUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range []string{CodeUnusedVar, CodeUnusedParam, CodeUnreachable, CodeShadow,
		CodeMissingRet, CodeIfaceMissing, CodeVoidAsValue} {
		if seen[c] {
			t.Fatalf("诊断码重复: %s", c)
		}
		seen[c] = true
		if !strings.HasPrefix(c, "QK") {
			t.Errorf("诊断码格式应为 QKxxx: %s", c)
		}
	}
}
