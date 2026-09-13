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

// Phase C：泛型函数 / 泛型 struct / 泛型 impl 的按调用点单态化。
func TestPhaseCGenerics(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "generic-function",
			src: "fn<T> id(T v) T { return v; }\n" +
				"fn<T, U> pair(T a, U b) T { return a; }\n" +
				"fn<T> pass(T v) T { T x = v; return x; }\n" +
				"fn main(IOStream io) {\n" +
				"    io.println(id(5), id(\"ab\"), id(1.5), id(true));\n" +
				"    io.println(id(pass(21)), pass(\"xy\"));\n" +
				"    io.println(id(id(3)));\n" +
				"    io.println(pair(7, \"x\"), pair(\"y\", 1.5));\n" +
				"}\n",
			want: "5 ab 1.5 true\n21 xy\n3\n7 y\n",
		},
		{
			name: "generic-struct-and-impl",
			src: "type struct<T> { T v; } Box;\n" +
				"impl<T> {\n" +
				"    fn new(T x) Box<T> { Box<T> b; b.v = x; return b; }\n" +
				"    fn get(Box<T> self) T { return self.v; }\n" +
				"    fn put(Box<T> self, T x) void { self.v = x; }\n" +
				"} Box;\n" +
				"fn main(IOStream io) {\n" +
				"    Box<int> a = Box::new(3);\n" +
				"    io.println(a.get(), a.v);\n" +
				"    a.put(8);\n" +
				"    io.println(a.get());\n" +
				"    Box<String> s = Box::new(\"hey\");\n" +
				"    io.println(s.get());\n" +
				"    Box<int> lit = .{v: 42};\n" +
				"    io.println(lit.get());\n" +
				"    Box<Box<int>> nest = Box::new(a);\n" +
				"    io.println(nest.get().get());\n" +
				"}\n",
			want: "3 3\n8\nhey\n42\n8\n",
		},
		{
			name: "interface-vtable-dispatch",
			src: "type interface { fn sum(Self self) int; fn tag(Self self) String; } Iface;\n" +
				"type struct { int a; } A;\n" +
				"impl {\n" +
				"    fn sum(A self) int { return self.a + 1; }\n" +
				"    fn tag(A self) String { return \"A\"; }\n" +
				"} A;\n" +
				"type struct { int b; int c; } B;\n" +
				"impl {\n" +
				"    fn sum(B self) int { return self.b + self.c; }\n" +
				"    fn tag(B self) String { return \"B\"; }\n" +
				"} B;\n" +
				"fn call(Iface x) int { return x.sum(); }\n" +
				"type struct { Iface x; int tag; } Holder;\n" +
				"fn main(IOStream io) {\n" +
				"    A a = .{a: 5};\n" +
				"    Iface x = a;\n" +
				"    io.println(x.sum(), x.tag());\n" +
				"    B b = .{b: 1, c: 2};\n" +
				"    x = b;\n" +
				"    io.println(x.sum(), x.tag());\n" +
				"    io.println(call(a), call(b));\n" +
				"    Holder h = .{x: a, tag: 7};\n" +
				"    io.println(h.x.sum(), h.tag);\n" +
				"}\n",
			want: "6 A\n3 B\n6 3\n6 7\n",
		},
		{
			name: "interface-self-return",
			src: "type interface { fn __add__(Self self, Self o) Self; fn show(Self self) String; } Addable;\n" +
				"type struct { int v; } N;\n" +
				"impl {\n" +
				"    fn __add__(N self, N o) N { N r; r.v = self.v + o.v; return r; }\n" +
				"    fn show(N self) String { return \"N\" + self.v.toString(); }\n" +
				"} N;\n" +
				"fn main(IOStream io) {\n" +
				"    N a = .{v: 1};\n" +
				"    N b = .{v: 2};\n" +
				"    Addable x = a;\n" +
				"    Addable y = b;\n" +
				"    Addable z = x.__add__(y);\n" +
				"    io.println(z.show());\n" +
				"}\n",
			want: "N3\n",
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

// String / List 内建方法：语义运行时在 qthreads.c（UTF-8 感知），lli 无法解析，
// 这里校验 IR 里确实调用了对应助手；端到端字节对齐由 compare.sh 的
// cases_run/s_string_methods.kq、t_list_methods.kq 覆盖。
func TestBuiltinMethodsIR(t *testing.T) {
	str := transpile(t, "fn main(IOStream io) {\n"+
		"    String s = \"abc\";\n"+
		"    io.println(s.size(), s.contains(\"b\"), s.startsWith(\"a\"), s.endsWith(\"c\"));\n"+
		"    io.println(s.indexOf(\"c\"), s.substring(1), s.substring(0, 2));\n"+
		"    io.println(s.trim(), s.trimLeft(), s.trimRight(), s.toLower(), s.toUpper());\n"+
		"    io.println(s.replace(\"a\", \"z\"), s.charAt(0), \"12\".toInt(), \"1.5\".toFloat());\n"+
		"}\n")
	for _, want := range []string{
		"call i32 @ql_str_size(", "call i32 @ql_str_contains(", "call i32 @ql_str_startswith(",
		"call i32 @ql_str_endswith(", "call i32 @ql_str_indexof(", "call i8* @ql_str_sub(",
		"call i8* @ql_str_trim(", "call i8* @ql_str_lower(", "call i8* @ql_str_upper(",
		"call i8* @ql_str_replace(", "call i8* @ql_str_charat(", "call i32 @ql_str_toint(",
		"call double @ql_str_tofloat(",
	} {
		if !strings.Contains(str, want) {
			t.Fatalf("String method IR missing %q:\n%s", want, str)
		}
	}
	lst := transpile(t, "fn main(IOStream io) {\n"+
		"    List<int> l = [3, 1, 2];\n"+
		"    l.__sort__();\n"+
		"    io.println(l.toString(), l.head(), l.tail(), l.next());\n"+
		"    l.reset();\n"+
		"    List<int> m = [9];\n"+
		"    l.appendAll(m);\n"+
		"    io.println(*l, l.size());\n"+
		"}\n")
	for _, want := range []string{
		"call void @ql_list_int_sort(", "call i8* @ql_list_int_str(",
		"@realloc(", "ListExhaustedError",
	} {
		if !strings.Contains(lst, want) {
			t.Fatalf("List method IR missing %q:\n%s", want, lst)
		}
	}
	// split 需要 List<String>：明确报错
	if _, err := Transpile("fn main(IOStream io) { String s = \"a,b\"; io.println(s.split(\",\")); }\n", "test.qk"); err == nil {
		t.Fatal("String.split must report unsupported (List<String>)")
	}
}

// 函数重载：按参数个数+类型解析（bestMatch 语义：同 kind +10 / 可赋值 +5）。
func TestOverloads(t *testing.T) {
	src := "fn add(int a, int b) int { return a + b; }\n" +
		"fn add(String a, String b) String { return a + \"<\" + b; }\n" +
		"fn add(float a, float b) float { return a + b; }\n" +
		"fn pick(int a) String { return \"int\"; }\n" +
		"fn pick(String s) String { return \"str\"; }\n" +
		"fn pick(float f) String { return \"float\"; }\n" +
		"fn zero() int { return 0; }\n" +
		"fn zero(String s) int { return 1; }\n" +
		"fn main(IOStream io) {\n" +
		"    io.println(add(1, 2));\n" +
		"    io.println(add(\"a\", \"b\"));\n" +
		"    io.println(add(1.5, 2.5));\n" +
		"    io.println(pick(1), pick(\"x\"), pick(2.5));\n" +
		"    io.println(add(1, 2.5));\n" +
		"    io.println(zero(), zero(\"s\"));\n" +
		"    io.println(add(1, 2).toString() + \"!\");\n" +
		"}\n"
	ir := transpile(t, src)
	for _, want := range []string{"@add$int$int", "@add$String$String", "@add$float$float", "@pick$int", "@zero$void"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("overload IR missing %q:\n%s", want, ir)
		}
	}
	got := lliRun(t, withTestRuntime(ir))
	want := "3\na<b\n4\nint str float\n3.5\n0 1\n3!\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// 没有匹配重载 → 明确报错
	if _, err := Transpile("fn f(int a) int { return a; }\nfn main(IOStream io) { io.println(f(\"s\")); }\n", "test.qk"); err == nil {
		t.Fatal("mismatched overload call must fail")
	}
}

// 形参按引用传递：callee 写形参穿透到调用方；copyd 传时深拷贝；非左值实参是临时单元。
func TestByRefParams(t *testing.T) {
	src := "type struct { int v; } S;\n" +
		"fn bump(int a) int { a = a + 1; return a; }\n" +
		"fn rebind(S s) int { S o; o.v = 99; s = o; return s.v; }\n" +
		"fn deep(copyd S s) int { s.v = 100; return s.v; }\n" +
		"fn chain(int a) int { return bump(a); }\n" +
		"fn main(IOStream io) {\n" +
		"    int x = 5;\n" +
		"    io.println(bump(x), x);\n" + // 6 6（写回）
		"    chain(x);\n" +
		"    io.println(x);\n" + // 7（引用链）
		"    io.println(bump(x + 1), x);\n" + // 9 7（非左值不回写）
		"    S s; s.v = 1;\n" +
		"    io.println(rebind(s), s.v);\n" + // 99 99（形参重绑定写回）
		"    S c; c.v = 7;\n" +
		"    io.println(deep(c), c.v);\n" + // 100 7（copyd 深拷贝）
		"}\n"
	ir := transpile(t, src)
	for _, want := range []string{"define i32 @bump(i32* noundef %p0)", "define i32 @rebind(%S** noundef %p0)", "call %S* @ql_copyd_struct_S"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("by-ref IR missing %q:\n%s", want, ir)
		}
	}
	if got := lliRun(t, withTestRuntime(ir)); got != "6 6\n7\n9 7\n99 99\n100 7\n" {
		t.Fatalf("got %q", got)
	}
}

// 签名 memorize（f(args) @mb()）：IR 必须包含初始化、查表、调用与写回（端到端由
// testdata/cases_run/v_sign.kq 的 compare.sh 对比覆盖：需要 clang 链接 qthreads 运行时）。
func TestMemoizeSignatureIR(t *testing.T) {
	ir := transpile(t, "fn sq(int n) int { return n * n; }\n"+
		"fn main(IOStream io) {\n"+
		"    memorize mb = memorize::new();\n"+
		"    io.println(sq(41) @mb());\n"+
		"}\n")
	for _, want := range []string{
		"declare i8* @ql_memo_new()",
		"call i8* @ql_memo_new()",
		"declare i32 @ql_memo_get(i8*, i32, i32*, i32*)",
		"call i32 @ql_memo_get(i8*",
		"call void @ql_memo_put(i8*",
		"call i32 @sq(i32*",
		"phi i32",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("memory IR missing %q:\n%s", want, ir)
		}
	}
	// 非 int 键 / 返回值：明确报错
	if _, err := Transpile("fn f(String s) String { return s; }\nfn main(IOStream io) { memorize mb = memorize::new(); io.println(f(\"a\") @mb()); }\n", "test.qk"); err == nil {
		t.Fatal("memorize with String keys must report unsupported")
	}
	if _, err := Transpile("fn main(IOStream io) { int x = 1; io.println(1 @x()); }\n", "test.qk"); err == nil {
		t.Fatal("non-memorize sign instance must report unsupported")
	}
}

// interface{}（tAny）：装箱 + RTTI 描述符 + 打印/相等/拆箱 IR 形状；拆箱到 struct/List 明确报错。
func TestAnyInterfaceIR(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    interface{} a = 7;\n"+
		"    a = \"s\";\n"+
		"    io.println(a);\n"+
		"    io.println(a == 7);\n"+
		"    int n = a;\n"+
		"    io.println(n);\n"+
		"}\n")
	for _, want := range []string{
		"%RT = type { i8* (i8*)*, i8*, i32 }",
		"@rt$int = private constant %RT",
		"@rt$String = private constant %RT",
		"declare i8* @ql_any_str(i8*, i8**)",
		"call i8* @ql_any_str(i8*",
		"call i32 @ql_any_eq(",
		"call i32 @ql_any_int(",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("interface{} IR missing %q:\n%s", want, ir)
		}
	}
	for _, src := range []string{
		"type struct { int v; } P;\nfn main(IOStream io) { interface{} a = 1; P p = a; io.println(p.v); }\n",
		"fn main(IOStream io) { interface{} a = 1; List<int> l = a; io.println(l.size()); }\n",
	} {
		if _, err := Transpile(src, "test.qk"); err == nil {
			t.Fatalf("unboxing interface{} to struct/List must report unsupported: %s", src)
		}
	}
	if _, err := Transpile("fn main(IOStream io) { pointer p; interface{} a = p; io.println(1); }\n", "test.qk"); err == nil {
		t.Fatal("boxing pointer into interface{} must report unsupported")
	}
}

// Phase D：library FFI（LLVM declare + C ABI 直调）与 taskm（qthreads 运行时）。
// 这两类需要 clang 链接（-lm / qthreads.c），lli 单测只校验 IR 形状；
// 端到端输出对齐由 compiler/testdata/compare.sh 覆盖（cases_run/e_ffi、f_taskm）。
func TestPhaseDFFIAndTaskm(t *testing.T) {
	ffi := transpile(t, "library m {\n"+
		"    fn sqrt(double x) double;\n"+
		"    fn sqrtf(f32 x) f32;\n"+
		"}\n"+
		"fn main(IOStream io) { io.println(m.sqrt(16.0), m.sqrtf(9.0)); }\n")
	for _, want := range []string{
		"declare double @sqrt(double)",
		"declare float @sqrtf(float)",
		"call double @sqrt(",
		"fpext float",
		"; qkc-link: m => -lm",
	} {
		if !strings.Contains(ffi, want) {
			t.Fatalf("FFI IR missing %q:\n%s", want, ffi)
		}
	}
	// 链接名映射：libc 默认已链接（-lc），m → -lm；IR 里以「库名 => 候选参数」标记
	links := transpile(t, "library libc {\n"+
		"    fn abs(int v) int;\n"+
		"}\n"+
		"library m {\n"+
		"    fn sqrt(float x) float;\n"+
		"}\n"+
		"fn main(IOStream io) { io.println(libc.abs(0 - 7), m.sqrt(16.0)); }\n")
	for _, want := range []string{
		"; qkc-link: libc => -lc",
		"; qkc-link: m => -lm",
		"declare i32 @abs(i32)",
		"declare double @sqrt(double)",
	} {
		if !strings.Contains(links, want) {
			t.Fatalf("link mapping IR missing %q:\n%s", want, links)
		}
	}
	if strings.Contains(links, "-llibc") {
		t.Fatalf("libc 不应映射为 -llibc:\n%s", links)
	}
	// long（i64）与 pointer（i8* 不透明句柄，可空、可往返）
	lp := transpile(t, "library libc {\n"+
		"    fn malloc(long n) pointer;\n"+
		"    fn free(pointer p) void;\n"+
		"    fn memset(pointer p, int c, long n) pointer;\n"+
		"    fn strlen(String s) long;\n"+
		"}\n"+
		"fn main(IOStream io) {\n"+
		"    pointer p = libc.malloc(16);\n"+
		"    pointer q = libc.memset(p, 65, 4);\n"+
		"    long n = libc.strlen(\"hello\");\n"+
		"    pointer z;\n"+
		"    io.println(n, p == q, p != null, z == null);\n"+
		"    libc.free(p);\n"+
		"}\n")
	for _, want := range []string{
		"declare i8* @malloc(i64)",
		"declare i64 @strlen(i8*)",
		"declare i8* @memset(i8*, i32, i64)",
		"call i8* @malloc(i64",
		"call i64 @strlen(",
		"icmp eq i8*",
		"sext i32", // int → long（malloc 实参）
		"%lld",     // long 打印
	} {
		if !strings.Contains(lp, want) {
			t.Fatalf("long/pointer IR missing %q:\n%s", want, lp)
		}
	}
	taskm := transpile(t, "fn work(int n) int { return n; }\n"+
		"fn main(IOStream io) {\n"+
		"    thread t = taskm.spawn();\n"+
		"    t.merge(work, 1);\n"+
		"    taskm.block(t.pid());\n"+
		"    io.println(taskm.done(t.pid()));\n"+
		"    Channel c = taskm.channel();\n"+
		"    c.send(3);\n"+
		"    io.println(c.recv());\n"+
		"}\n")
	for _, want := range []string{
		"declare i32 @ql_spawn()",
		"declare void @ql_merge(i32, i8*, i64, i64, i64, i64)",
		"define void @runner_0(i64 %a0)",
		"call void @ql_merge(i32",
		"call void @ql_block(i32",
		"call i32 @ql_send(i8*",
		"call i32 @ql_recv(i8*",
	} {
		if !strings.Contains(taskm, want) {
			t.Fatalf("taskm IR missing %q:\n%s", want, taskm)
		}
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
			name: "catch 变量使用",
			src: "fn main(IOStream io) {\n" +
				"    try { int a = 1 / 0; } catch (void e) { io.println(\"caught: \" + e); }\n" +
				"}\n",
			want: "暂未支持在 catch 体内使用",
		},
		{
			name: "IOStream 形参（仅 main 入口绑定）",
			src: "fn f(int n, IOStream io) void { io.println(n); }\n" +
				"fn main(IOStream io) { f(1, io); }\n",
			want: "暂未支持 IOStream",
		},
		// String 内建方法已 lower（见 testdata/cases_run/s_string_methods.kq 的解析器/编译器对比）
		// List.toString 已 lower（见 testdata/cases_run/t_list_methods.kq）
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
		// float 取模已 lower
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
