package cgen

import (
	"strings"
	"testing"
)

// lowerErr 编译正典语法源码并返回错误（用于「必须明确报错」的用例）。
func lowerErr(t *testing.T, src string) error {
	t.Helper()
	_, err := Transpile(src, "test.qk")
	if err == nil {
		t.Fatalf("expected explicit error, got none for:\n%s", src)
	}
	return err
}

func TestTryCatchCompile(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    try {\n"+
		"        int a = 10 / 0;\n"+
		"    } catch (void e) {\n"+
		"        io.println(1);\n"+
		"    }\n"+
		"}\n")
	if !strings.Contains(ir, "define i32 @main()") {
		t.Fatalf("empty ir:\n%s", ir)
	}
	if got := lliRun(t, ir); got != "1\n" {
		t.Fatalf("got %q want %q", got, "1\n")
	}
}

func TestStringConcatCompile(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    String s = \"a\";\n"+
		"    io.println(s + \"b\");\n"+
		"}\n")
	if !strings.Contains(ir, "call i8* @ql_strcat") {
		t.Fatalf("missing strcat call:\n%s", ir)
	}
	// 拼接走运行时 ql_strcat（lli 无该符号，端到端由 qkc -run + qthreads.c 覆盖）
	if strings.Contains(ir, "memory(none)") {
		t.Fatalf("函数带 IO/strcat 副作用，不应标 memory(none):\n%s", ir)
	}
}

// Phase A 端到端：for/break/log/float/bool/io.print/String 比较/String 形参。
// want 全部取自解释器（compiler/testdata/compare.sh 会复核两条路径逐字节一致）。
func TestPhaseACompile(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "for-c-style-and-break",
			src: "fn main(IOStream io) {\n" +
				"    for (int i = 0; i < 3; i = i + 1) { io.print(i, \"-\"); }\n" +
				"    io.println(\"\");\n" +
				"    int i = 0;\n" +
				"    while (true) { i = i + 1; if (i > 2) { break; } io.println(i); }\n" +
				"}\n",
			want: "0 -1 -2 -\n1\n2\n",
		},
		{
			name: "for-in-consumes-list",
			src: "fn main(IOStream io) {\n" +
				"    List<int> l = [10, 20, 30];\n" +
				"    for (int x : l) { io.print(x); }\n" +
				"    io.println(\"\");\n" +
				"    io.println(l.size());\n" +
				"}\n",
			want: "102030\n0\n",
		},
		{
			name: "break-in-nested-for",
			src: "fn main(IOStream io) {\n" +
				"    for (int k = 0; k < 2; k = k + 1) {\n" +
				"        for (int j = 0; j < 2; j = j + 1) {\n" +
				"            if (j == 1) { break; }\n" +
				"            io.print(k, j);\n" +
				"        }\n" +
				"    }\n" +
				"    io.println(\"\");\n" +
				"}\n",
			want: "0 01 0\n",
		},
		{
			name: "log-returns-nil-and-ends-function",
			src: "fn sink(int n) int {\n" +
				"    log n;\n" +
				"    return n;\n" +
				"}\n" +
				"fn main(IOStream io) {\n" +
				"    io.println(\"before\");\n" +
				"    sink(7);\n" +
				"    io.println(\"after\");\n" +
				"}\n",
			want: "before\nafter\n",
		},
		{
			name: "float-and-bool",
			src: "fn main(IOStream io) {\n" +
				"    float x = 1.5;\n" +
				"    float y = 2.0;\n" +
				"    io.println(x + y, x - y, x * y, x / y, -x);\n" +
				"    io.println(1.0 / 3.0, 0.1 + 0.2);\n" +
				"    io.println(x > y, x < y, x == 1.5, x != y);\n" +
				"    io.println(x.toString());\n" +
				"    bool b = true;\n" +
				"    bool c = false;\n" +
				"    io.println(b, c, b && !c, b || c, !b, b == c, b != c, b.toString());\n" +
				"    float f = 3;\n" +
				"    io.println(f, f.toString());\n" +
				"}\n",
			want: "3.5 -0.5 3 0.75 -1.5\n0.3333333333333333 0.30000000000000004\nfalse true true true\n1.5\ntrue false true true false false true true\n3 3\n",
		},
		{
			name: "io-print-no-newline",
			src: "fn main(IOStream io) {\n" +
				"    io.print(\"p\", 1, 1.5, true);\n" +
				"    io.print(\"q\");\n" +
				"    io.println(\"\");\n" +
				"}\n",
			want: "p 1 1.5 trueq\n",
		},
		{
			name: "string-compare",
			src: "fn main(IOStream io) {\n" +
				"    String s = \"abc\";\n" +
				"    String t = \"abd\";\n" +
				"    io.println(s == t, s != t, s == \"abc\", s != \"abc\");\n" +
				"    io.println(s < t, s > t, s <= t, s >= t);\n" +
				"    io.println(s < \"ab\", \"b\" > s);\n" +
				"}\n",
			want: "false true true false\ntrue false true false\nfalse true\n",
		},
		{
			name: "string-params",
			src: "fn greet(String s) String { return \"hi \" + s; }\n" +
				"fn main(IOStream io) {\n" +
				"    io.println(greet(\"bob\"));\n" +
				"    String x = \"abcd\";\n" +
				"    io.println(greet(x));\n" +
				"    io.println(greet(greet(\"z\")));\n" +
				"}\n",
			want: "hi bob\nhi abcd\nhi hi z\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := lliRun(t, withTestRuntime(transpile(t, c.src)))
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

// Phase B 端到端：struct / impl / space / 运算符重载（引用语义与解释器一致）。
func TestPhaseBStructCompile(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "struct-fields-methods-static-space-ops",
			src: "type struct { int a; String name; float f; bool ok; } Rec;\n" +
				"impl {\n" +
				"    fn new(int a, String n) Rec {\n" +
				"        Rec r;\n" +
				"        r.a = a;\n" +
				"        r.name = n;\n" +
				"        r.f = 1.5;\n" +
				"        r.ok = true;\n" +
				"        return r;\n" +
				"    }\n" +
				"    fn describe(Rec self) String { return self.name + \"=\" + self.a.toString(); }\n" +
				"    fn bump(Rec self, int d) void { self.a = self.a + d; }\n" +
				"} Rec;\n" +
				"type struct { Rec r; int tag; } Wrap;\n" +
				"impl { fn show(Wrap self) String { return self.r.describe() + \"/\" + self.tag.toString(); } } Wrap;\n" +
				"space {\n" +
				"    fn max(int a, int b) int { if (a > b) { return a; } return b; }\n" +
				"    fn twice(int a) int { return math::max(a, a) + a; }\n" +
				"} math;\n" +
				"fn main(IOStream io) {\n" +
				"    Rec r = Rec::new(3, \"abc\");\n" +
				"    io.println(r.describe(), r.a, r.name, r.f, r.ok);\n" +
				"    r.bump(4);\n" +
				"    io.println(r.a);\n" +
				"    Wrap w = .{r: r, tag: 9};\n" +
				"    io.println(w.show());\n" +
				"    io.println(math::max(3, 7), math::twice(5));\n" +
				"    Rec z;\n" +
				"    io.println(z.a, z.name, z.f, z.ok, z.name == \"\");\n" +
				"}\n",
			want: "abc=3 3 abc 1.5 true\n7\nabc=7/9\n7 10\n0  0 false true\n",
		},
		{
			name: "struct-reference-semantics",
			src: "type struct { int a; int b; } Point;\n" +
				"impl {\n" +
				"    fn seta(Point self, int v) void { self.a = v; }\n" +
				"    fn sum(Point self) int { return self.a + self.b; }\n" +
				"} Point;\n" +
				"fn touch(Point p) void { p.a = 42; }\n" +
				"fn main(IOStream io) {\n" +
				"    Point p = .{a: 1, b: 2};\n" +
				"    Point q = p;\n" +
				"    q.a = 99;\n" +
				"    io.println(p.a, q.a);\n" +
				"    p.seta(7);\n" +
				"    touch(p);\n" +
				"    io.println(p.a, q.a, p.sum());\n" +
				"    Point z;\n" +
				"    io.println(z.a, z.b);\n" +
				"    Point w = .{1, 2};\n" +
				"    io.println(w.a, w.b, w == p, w == w);\n" +
				"}\n",
			want: "99 99\n42 42 44\n0 0\n1 2 false true\n",
		},
		{
			name: "operator-overload",
			src: "type struct { int x; int y; } Vec;\n" +
				"impl {\n" +
				"    fn __add__(Vec self, Vec o) Vec { Vec r; r.x = self.x + o.x; r.y = self.y + o.y; return r; }\n" +
				"    fn __eq__(Vec self, Vec o) bool { return self.x == o.x && self.y == o.y; }\n" +
				"    fn __lt__(Vec self, Vec o) bool { return self.x < o.x; }\n" +
				"    fn __neg__(Vec self) Vec { Vec r; r.x = 0 - self.x; r.y = 0 - self.y; return r; }\n" +
				"} Vec;\n" +
				"fn main(IOStream io) {\n" +
				"    Vec a = .{x: 1, y: 2};\n" +
				"    Vec b = .{x: 10, y: 20};\n" +
				"    Vec c = a + b;\n" +
				"    io.println(c.x, c.y);\n" +
				"    io.println(a == b, a == a, a < b, b < a);\n" +
				"    Vec d = -a;\n" +
				"    io.println(d.x, d.y);\n" +
				"}\n",
			want: "11 22\nfalse true true false\n-1 -2\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := lliRun(t, withTestRuntime(transpile(t, c.src)))
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

// 后端暂未 lower 的构造必须给出带位置的明确「暂未支持」错误，
// 而不是 parse 错误，更不允许静默错编。
func TestUnsupportedConstructs(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "接口参数（dynamic 分发）",
			src: "type interface { fn sum(Self self) int; } Iface;\n" +
				"fn need(Iface s) int { return 0; }\n" +
				"fn main(IOStream io) { io.println(1); }\n",
			want: "暂未支持接口类型",
		},
		{
			name: "泛型实例化",
			src: "fn<T> id(T v) T { return v; }\n" +
				"fn main(IOStream io) { io.println(id(5)); }\n",
			want: "暂未支持泛型实例化 id",
		},
		{
			name: "library FFI",
			src: "library m {\n" +
				"    fn sqrt(double x) double;\n" +
				"}\n" +
				"fn main(IOStream io) { io.println(m.sqrt(16.0)); }\n",
			want: "暂未支持 library FFI 调用 m.sqrt",
		},
		{
			name: "taskm 线程",
			src:  "fn main(IOStream io) { thread t = taskm.spawn(); }\n",
			want: "暂未支持 taskm",
		},
		{
			name: "taskm 通道",
			src:  "fn main(IOStream io) { Channel c = taskm.channel(); }\n",
			want: "暂未支持 taskm",
		},
		{
			name: "catch 变量使用",
			src: "fn main(IOStream io) {\n" +
				"    try { int a = 1 / 0; } catch (void e) { io.println(\"caught: \" + e); }\n" +
				"}\n",
			want: "暂未支持在 catch 体内使用",
		},
		{
			name: "函数重载",
			src: "fn add(int a, int b) int { return a + b; }\n" +
				"fn add(String a, String b) String { return a + b; }\n" +
				"fn main(IOStream io) { io.println(add(1, 2)); }\n",
			want: "暂未支持函数重载",
		},
		{
			name: "IOStream 形参（仅 main 入口绑定）",
			src: "fn f(int n, IOStream io) void { io.println(n); }\n" +
				"fn main(IOStream io) { f(1, io); }\n",
			want: "暂未支持 IOStream",
		},
		{
			name: "String 内建方法未 lower（substring 等）",
			src: "fn main(IOStream io) {\n" +
				"    String s = \"abc\";\n" +
				"    io.println(s.substring(0, 1));\n" +
				"}\n",
			want: "暂未支持对 String 调用方法 substring",
		},
		{
			name: "List.toString 未 lower",
			src: "fn main(IOStream io) {\n" +
				"    List<int> l = [1, 2];\n" +
				"    io.println(l.toString());\n" +
				"}\n",
			want: "暂未支持 List.toString()",
		},
		{
			name: "非标量返回类型",
			src: "fn f(int n) List<String> {\n" +
				"    List<String> l = [];\n" +
				"    return l;\n" +
				"}\n" +
				"fn main(IOStream io) { io.println(1); }\n",
			want: "暂未支持返回类型",
		},
		{
			name: "List<String> 变量",
			src:  "fn main(IOStream io) { List<String> l = [\"a\"]; io.println(1); }\n",
			want: "暂未支持",
		},
		{
			name: "指针/new",
			src:  "fn main(IOStream io) { int& p = new int; io.println(1); }\n",
			want: "暂未支持指针/传时复制类型",
		},
		{
			name: "打印 struct 值（解释器字段序不确定）",
			src: "type struct { int a; } P;\n" +
				"fn main(IOStream io) { P p; io.println(p); }\n",
			want: "暂未支持打印",
		},
		{
			name: "float 取模",
			src:  "fn main(IOStream io) { float a = 5.0; float b = 2.0; io.println(a % b); }\n",
			want: "暂未支持 float 取模",
		},
		{
			name: "含 log 的函数返回值被使用",
			src: "fn f(int n) int { log n; return n; }\n" +
				"fn main(IOStream io) { io.println(f(7)); }\n",
			want: "暂未支持在表达式中使用含 log 的函数",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := lowerErr(t, c.src)
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), c.want)
			}
			if strings.Contains(err.Error(), "ParseError") {
				t.Fatalf("expected unsupported-construct error, got parse error: %v", err)
			}
			// 必须带位置（--> file:line:col）
			if !strings.Contains(err.Error(), "test.qk:") {
				t.Fatalf("error lacks source position: %v", err)
			}
		})
	}
}
