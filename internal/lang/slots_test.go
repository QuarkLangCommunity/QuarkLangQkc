package lang

// 槽位预解析（slots.go）的语义回归：优化把「按名字查找」换成「下标直取」，
// 一旦推导与实际作用域不符就会静默读错变量。这里用真实执行结果钉住各类边界。
// 用例覆盖：前缀顶层声明、嵌套块内声明、循环体内声明、for-in 遮蔽、catch 作用域、
//          参数与局部同名、递归、引用参数写穿。

import (
	"bytes"
	"strings"
	"testing"
)

func runSlotSrc(t *testing.T, src string) string {
	t.Helper()
	prog, err := Compile(src)
	if err != nil {
		t.Fatalf("编译失败: %v\n%s", err, src)
	}
	var out bytes.Buffer
	if err := Run(prog, "slot_test.qk", nil, strings.NewReader(""), &out); err != nil {
		t.Fatalf("运行失败: %v\n%s", err, src)
	}
	return out.String()
}

func TestSlotResolutionSemantics(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			"前缀顶层声明 + while 内读写",
			`fn main(IOStream io) void {
    int i = 0;
    int sum = 0;
    while (i < 3) {
        sum = sum + i;
        i = i + 1;
    }
    io.println(sum);
}`,
			"3\n",
		},
		{
			"嵌套块内声明（槽位序号不稳定）",
			`fn main(IOStream io) void {
    int a = 1;
    if (a > 0) {
        int b = 2;
        a = a + b;
    }
    int c = 3;
    io.println(a + c);
}`,
			"6\n",
		},
		{
			"循环体内声明早于其后顶层声明",
			`fn main(IOStream io) void {
    int i = 0;
    while (i < 2) {
        int t = i;
        i = i + 1;
    }
    int y = 7;
    io.println(y);
}`,
			"7\n",
		},
		{
			"循环体不执行时其后顶层声明",
			`fn main(IOStream io) void {
    int i = 5;
    while (i < 0) {
        int t = 1;
        i = i + t;
    }
    int y = 9;
    io.println(y);
}`,
			"9\n",
		},
		{
			"for-in 循环变量遮蔽外层同名",
			`fn main(IOStream io) void {
    int x = 9;
    List<int> l = [1, 2];
    for (int x : l) {
        io.print(x);
    }
    io.println(x);
}`,
			"129\n",
		},
		{
			"catch 变量在新作用域（与同名外层共存）",
			`fn main(IOStream io) void {
    String e = "outer";
    try {
        io.println(1 / 0);
    } catch (String e) {
        io.println("caught");
    }
    io.println(e);
}`,
			"caught\nouter\n",
		},
		{
			"C 风格 for 的初始化声明在子作用域",
			`fn main(IOStream io) void {
    int i = 100;
    for (int i = 0; i < 2; i = i + 1) {
        io.print(i);
    }
    io.println(i);
}`,
			"01100\n",
		},
		{
			"参数与局部同名遮蔽",
			`fn f(int n) int {
    int r = n;
    int n2 = r + 1;
    return n2;
}
fn main(IOStream io) void {
    io.println(f(41));
}`,
			"42\n",
		},
		{
			"递归 + 参数槽位",
			`fn fib(int n) int {
    if (n <= 1) {
        return n;
    }
    return fib(n - 1) + fib(n - 2);
}
fn main(IOStream io) void {
    io.println(fib(10));
}`,
			"55\n",
		},
		{
			"引用参数写穿（槽位 vs 引用语义）",
			`fn setRef(int& p, int v) void { p = v; }
fn main(IOStream io) void {
    int& p = new int;
    p = 1;
    setRef(p, 42);
    io.println(p);
}`,
			"42\n",
		},
		{
			"多函数各自槽位互不干扰",
			`fn a(int x) int {
    int y = x + 1;
    return y;
}
fn b(int x) int {
    int y = x * 2;
    int z = y + 1;
    return z;
}
fn main(IOStream io) void {
    io.println(a(1), b(3));
}`,
			"2 7\n",
		},
		{
			"泛型/重载调用后槽位仍正确",
			`fn id(int x) int { return x; }
fn id(String s) int { return s.size(); }
fn main(IOStream io) void {
    int a = id(7);
    int b = id("abcd");
    io.println(a + b);
}`,
			"11\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := runSlotSrc(t, c.src); got != c.want {
				t.Errorf("输出不符\n got %q\nwant %q\n源码:\n%s", got, c.want, c.src)
			}
		})
	}
}

// 槽位标记本身要能通过名字校验（防止把标记写到错误的变量上）。
func TestSlotAnnotationNameMatches(t *testing.T) {
	src := `fn main(IOStream io) void {
    int alpha = 1;
    int beta = 2;
    io.println(alpha + beta);
}
`
	prog, err := Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	fn := prog.Funcs[0]
	slots := []string{"io"} // 第 0 槽位是 io 参数
	var check func(e Expr, names []string, ok *bool)
	ok := true
	check = func(e Expr, names []string, ok *bool) {
		if e == nil {
			return
		}
		switch x := e.(type) {
		case *Ident:
			if x.Slot > 0 {
				i := int(x.Slot) - 1
				if i >= len(names) || names[i] != x.Name {
					t.Errorf("Ident %q 的槽位 %d 与实际名字 %v 不符", x.Name, i, names)
					*ok = false
				}
			}
		case *BinOp:
			check(x.L, names, ok)
			check(x.R, names, ok)
		case *CallExpr:
			for _, a := range x.Args {
				check(a, names, ok)
			}
		}
	}
	names := slots
	for _, st := range fn.Body.Stmts {
		if ds, isDecl := st.(*DeclStmt); isDecl {
			names = append(names, ds.Name)
			continue
		}
		if es, isExpr := st.(*ExprStmt); isExpr {
			check(es.X, names, &ok)
		}
	}
	if !ok {
		t.Fatal("槽位标记与实际槽位不符（会静默读错变量）")
	}
}
