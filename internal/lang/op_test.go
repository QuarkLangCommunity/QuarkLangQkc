package lang

import (
	"strings"
	"testing"
)

// Operation 接口族：内置 dynamic 协议；impl 方法聚合 + 运算符重载
func TestOperationOverload(t *testing.T) {
	out, err := runSrc(t, `type struct {
    int x;
    int y;
} Vec2;

impl {
    fn __add__(Vec2 self, Vec2 o) Vec2 {
        Vec2 r; r.x = self.x + o.x; r.y = self.y + o.y; return r;
    }
    fn __sub__(Vec2 self, Vec2 o) Vec2 {
        Vec2 r; r.x = self.x - o.x; r.y = self.y - o.y; return r;
    }
    fn __eq__(Vec2 self, Vec2 o) bool {
        return self.x == o.x && self.y == o.y;
    }
    fn __neg__(Vec2 self) Vec2 {
        Vec2 r; r.x = -self.x; r.y = -self.y; return r;
    }
} Vec2;

impl {
} Vec2;

fn main(IOStream io) {
    Vec2 a; a.x = 1; a.y = 2;
    Vec2 b; b.x = 3; b.y = 4;
    Vec2 c = a + b;
    io.println(c.x);
    io.println(c.y);
    Vec2 d = a - b;
    io.println(d.x);
    Vec2 e = -c;
    io.println(e.y);
    io.println(a == b);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "4\n6\n-2\n-6\nfalse\n" {
		t.Fatalf("got %q", out)
	}
}

// dynamic expand 组合接口解析
func TestDynamicExpandParse(t *testing.T) {
	_, err := Compile(`type interface {
    dynamic expand interface Operation;
} Everything;
fn main(IOStream io) { io.println(1); }`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(`type interface {
    width String;
} X;`); err == nil || !strings.Contains(err.Error(), "fn") {
		t.Fatalf("expected method-sig error, got %v", err)
	}
}

// 函数重载：同名多签名（参数数据/品种）按实参最优匹配
func TestFunctionOverload(t *testing.T) {
	out, err := runSrc(t, `fn add(int a, int b) int { return a + b; }
fn add(float a, float b) float { return a + b; }
fn add(String a, String b) String { return a + b; }
fn add(int a, int b, int c) int { return a + b + c; }

fn main(IOStream io) {
    io.println(add(1, 2));
    io.println(add(1.5, 2.5));
    io.println(add("A", "B"));
    io.println(add(1, 2, 3));
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "3\n4\nAB\n6\n" {
		t.Fatalf("got %q", out)
	}
}

// 反引号原始字符串（Go 语义）
func TestRawString(t *testing.T) {
	out, err := runSrc(t, `fn main(IOStream io) {
    String s = `+"`"+`line1
line2 "q" \n raw`+"`"+`;
    io.println(s.size());
    io.println(s);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "22\nline1\nline2 \"q\" \\n raw\n" {
		t.Fatalf("got %q", out)
	}
}

// expand interface X; 是接口体内的语句（而非常规声明）；dynamic 前缀；组合递归 + 结构化满足接口
func TestExpandStatementSyntax(t *testing.T) {
	out, err := runSrc(t, `type interface {
    dynamic expand interface AddOperation;
    dynamic fn ping(Self self) void;
} Everything;

type struct {
    int x;
} Thing;

impl {
    fn __add__(Thing self, Thing o) Thing {
        Thing r; r.x = self.x + o.x; return r;
    }
} Thing;

impl {
    fn ping(Thing self) void {
    }
} Thing;

fn needEverything(Everything e) void {
}

fn main(IOStream io) {
    Thing a; a.x = 5;
    Thing b; b.x = 7;
    io.println((a + b).x);
    io.println(a.ping());
    needEverything(a);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "12\n") {
		t.Fatalf("got %q", out)
	}
}

// break 语句：while/for 循环内跳出；循环外编译报错
func TestBreakStatement(t *testing.T) {
	out, err := runSrc(t, `fn main(IOStream io) {
    int i = 0;
    while (i < 1000) {
        if (i == 3) { break; }
        i = i + 1;
    }
    io.println(i);
    List<int> items = [1,2,3,4,5];
    for (int x : items) {
        if (x == 4) { break; }
        io.println(x);
    }
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "3\n1\n2\n3\n" {
		t.Fatalf("got %q", out)
	}
	if _, err := runSrc(t, `fn main(IOStream io) { break; }`); err == nil ||
		!strings.Contains(err.Error(), "break 只能在") {
		t.Fatalf("outside-break should error, got %v", err)
	}
}

// 字符串纯文本语义：双引号内字面（算式不解析）；\W 等未知转义保留字面；反引号含换行
func TestStringLiteralSemantics(t *testing.T) {
	out, err := runSrc(t, "fn main(IOStream io) {\n"+
		"    String a = \"1 + 2\";\n"+
		"    io.println(a);\n"+
		"    String b = \"C:\\\\Windows\\\\Fonts\";\n"+
		"    io.println(b);\n"+
		"    String c = `x\\ny`;\n"+
		"    io.println(c.size());\n"+
		"}")
	if err != nil {
		t.Fatal(err)
	}
	if out != "1 + 2\nC:\\Windows\\Fonts\n4\n" {
		t.Fatalf("got %q", out)
	}
}
