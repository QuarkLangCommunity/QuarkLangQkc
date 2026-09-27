package lang

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"quarklang/internal/i18n"
	"strings"
	"testing"
)

func runSrc(t *testing.T, src string, args ...string) (string, error) {
	t.Helper()
	prog, err := Compile(src)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = Run(prog, "test.qk", args, strings.NewReader(""), &out)
	return out.String(), err
}

// ============ v2 core semantics ============

func TestHello(t *testing.T) {
	out, err := runSrc(t, `fn main(IOStream io) {
    io.println("Hello World!");
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello World!\n" {
		t.Fatalf("got %q", out)
	}
}

// Functions must carry a return type; return yields the real value and ends the function
func TestReturnValue(t *testing.T) {
	out, err := runSrc(t, `
fn sq(int n) int {
    return n * n;
}

fn main(IOStream io) {
    int x = sq(7);
    io.println(x);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "49\n" {
		t.Fatalf("got %q", out)
	}
}

// log records and ends the function (return is no longer executed)
func TestLogEndsFunction(t *testing.T) {
	out, err := runSrc(t, `
fn f() int {
    log "done";
    return 999;
}

fn main(IOStream io) {
    int x = f();
    io.println(x);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "nil\n" {
		t.Fatalf("got %q", out)
	}
}

// try/catch (name + type)
func TestTryCatch(t *testing.T) {
	out, err := runSrc(t, `
fn main(IOStream io) {
    try {
        int y = 1 / 0;
        io.println(y);
    } catch (void e) {
        io.println("caught: " + e);
    }
    io.println("after");
}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "caught: DivisionByZeroError") || !strings.HasSuffix(out, "after\n") {
		t.Fatalf("got %q", out)
	}
}

// .{...} anonymous struct literal (field names may be the keywords in/out)
func TestStructLiteral(t *testing.T) {
	out, err := runSrc(t, `
fn main(IOStream io) {
    io.println(.{in: 1, out: 2}.out);
    io.println(.{in: 1, out: 2}.in);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "2\n1\n" {
		t.Fatalf("got %q", out)
	}
}

// Rolling List: * takes the head, next rolls forward, for syntactic sugar, exhaustion, reset
func TestRollingList(t *testing.T) {
	out, err := runSrc(t, `
fn main(IOStream io) {
    List<int> l = [10, 20, 30];
    io.println(*l);
    io.println(l.next());
    for (int x : l) {
        io.println(x);
    }
    io.println(l.head() == l.tail());
    l.reset();
    io.println(*l);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "10\n10\n20\n30\ntrue\n10\n" {
		t.Fatalf("got %q", out)
	}
}

func TestListExhausted(t *testing.T) {
	_, err := runSrc(t, `
fn main(IOStream io) {
    List<int> l = [1];
    l.next();
    l.next();
}`)
	if err == nil || !strings.Contains(err.Error(), "ListExhaustedError") {
		t.Fatalf("got %v", err)
	}
}

// ============ signature (v2): f(args) @instance() ============

func TestMemorizeSignature(t *testing.T) {
	out, err := runSrc(t, `
fn expensive(int n) int {
    return n * n;
}

fn main(IOStream io) {
    memorize mb = memorize::new();
    int x = expensive(41) @mb();
    io.println(x);
    int y = expensive(41) @mb();
    io.println(y);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "1681\n1681\n" {
		t.Fatalf("got %q", out)
	}
}

// ============ taskm thread model ============

func TestTaskmThreads(t *testing.T) {
	out, err := runSrc(t, `
fn add(int a, int b) int {
    return a + b;
}

fn main(IOStream io) {
    thread t = taskm.spawn();
    io.println(t.pid() > 0);
    t.merge(add, 3, 4);
    Channel ch = taskm.channel();
    t.talk(ch);
    io.println("ok");
}`)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 || lines[0] != "true" || lines[1] != "ok" {
		t.Fatalf("got %q", out)
	}
}

// ============ struct / impl / interface ============

func TestStructAndMethods(t *testing.T) {
	out, err := runSrc(t, `
type struct {
    int x;
    int y;
} Point;

impl {
    fn translate(Point self, int dx, int dy) void {
        self.x = self.x + dx;
        self.y = self.y + dy;
    }
    fn sum(Point self) int {
        return self.x + self.y;
    }
    fn new() Point {
        Point p;
        p.x = 3;
        p.y = 4;
        return p;
    }
} Point;

fn main(IOStream io) {
    Point p = Point::new();
    p.translate(1, 1);
    io.println(p.x);
    io.println(p.y);
    io.println(p.sum());
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "4\n5\n9\n" {
		t.Fatalf("got %q", out)
	}
}

func TestImplConformanceError(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.ZH)) // asserts the Chinese wording; English rendering is covered by this package's i18n tests
	src := `
type interface {
    fn call(void prefix, void rec) void;
} Sign;

type struct {
    int x;
} Bad;

impl {
    fn new() Bad {
        Bad b;
        return b;
    }
} Bad;

fn need(Sign s) void {
}

fn main(IOStream io) {
    Bad b = Bad::new();
    need(b);
}`
	_, err := runSrc(t, src)
	if err == nil || !strings.Contains(err.Error(), "未实现接口方法") {
		t.Fatalf("got %v", err)
	}
}

// ============ generics / pointers / Copyd ============

func TestGenericNode(t *testing.T) {
	out, err := runSrc(t, `
type struct<T> {
    T val;
    node<T>& next;
} node;

impl<T> {
    fn set(node<T> self, T v) void {
        self.val = v;
    }
    fn new() node<T> {
        node<T> n;
        return n;
    }
} node;

fn main(IOStream io) {
    node<int> a = node::new();
    a.set(42);
    node<int> b = node::new();
    b.set(7);
    a.next = b;
    io.println(a.next.val);
    a.next = null;
    io.println(a.next == null);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "7\ntrue\n" {
		t.Fatalf("got %q", out)
	}
}

func TestNullPointerDeref(t *testing.T) {
	_, err := runSrc(t, `
type struct<T> {
    T val;
    node<T>& next;
} node;

impl<T> {
    fn new() node<T> {
        node<T> n;
        return n;
    }
} node;

fn main(IOStream io) {
    node<int> a = node::new();
    a.next = null;
    io.println(a.next.val);
}`)
	if err == nil || !strings.Contains(err.Error(), "NullPointerError") {
		t.Fatalf("got %v", err)
	}
}

func TestCopydPtr(t *testing.T) {
	out, err := runSrc(t, `
fn f(IOStream io, int[Copyd] a) void {
    a.append(99);
    List<int> b = a.ptr();
    io.println(b.size());
}

fn main(IOStream io) {
    List<int> l = [1, 2];
    f(io, l);
    io.println(l.size());
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "3\n2\n" {
		t.Fatalf("got %q", out)
	}
}

// ============ real memory system ============

func TestMemoryCompactReclaims(t *testing.T) {
	src := `
fn worker(Channel ch) void {
    ch.send(1);
}

fn main(IOStream io) {
    thread t = taskm.spawn();
    Channel ch = taskm.channel();
    t.talk(ch);
    t.merge(worker, ch);
    void x = ch.recv();
    GlobalMemory.compact();
}`
	prog, err := Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	in, err := runWithInterp(prog, "test.qk", nil, strings.NewReader(""), &out)
	if err != nil {
		t.Fatal(err)
	}
	// the function block of merge (owner=pid) has been reclaimed by compact; the thread's own persistent block (owner=0) is kept → 1 remains
	if n := in.mem.BlockCount(); n != 1 {
		t.Fatalf("expected 1 block (persistent thread block) after compact, got %d", n)
	}
}

// ============ strict checks ============

func TestTypeCheckErrors(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.ZH)) // asserts the Chinese wording; English rendering is covered by this package's i18n tests
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"badarith", `fn main(IOStream io) {
    io.println("s" - 1);
}`, "arithmetic requires numbers"},
		{"star_nonlist", `fn main(IOStream io) {
    int n = 5;
    io.println(*n);
}`, "requires a List"},
		{"cond_not_bool", `fn main(IOStream io) {
    if (1) {
        io.println("x");
    }
}`, "condition must be bool"},
		{"arg_count", `fn f(int a) int {
    return a;
}
fn main(IOStream io) {
    f(1, 2, 3);
}`, "expects 1 args"},
		{"arg_type", `fn f(String s) String {
    return s;
}
fn main(IOStream io) {
    f(42);
}`, "cannot assign int to String"},
		{"undeclared", `fn main(IOStream io) {
    x = 5;
}`, "undeclared"},
		{"duplicate", `fn main(IOStream io) {
    List<int> l = [1];
    List<int> l = [2];
}`, "duplicate declaration"},
		{"use_before_init", `fn main(IOStream io) {
    List<int> l;
    io.println(l.size());
}`, "used before initialization"},
		{"unknown_member", `fn main(IOStream io) {
    int n = 5;
    n.append(1);
}`, "no method"},
		{"missing_ret", `fn f() {
}`, "必须声明返回类型"},
	}
	for _, c := range cases {
		_, err := runSrc(t, c.src)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: got %v, want contains %q", c.name, err, c.want)
		}
	}
}

// ============ edge unit tests ============

func TestArithmeticEdges(t *testing.T) {
	out, err := runSrc(t, `
fn main(IOStream io) {
    io.println(7 % 3);
    io.println(-7 % 3);
    io.println(-5 + 3);
    io.println(1 + 2 + 3);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "1\n-1\n-2\n6\n" {
		t.Fatalf("got %q", out)
	}
}

func TestDivByZero(t *testing.T) {
	_, err := runSrc(t, `fn main(IOStream io) {
    io.println(1 / 0);
}`)
	if err == nil || !strings.Contains(err.Error(), "DivisionByZeroError") {
		t.Fatalf("got %v", err)
	}
}

func TestStringConcat(t *testing.T) {
	out, err := runSrc(t, `
fn main(IOStream io) {
    io.println("a" + "b" + 3);
    io.println(1 + 2 + "x");
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ab3\n3x\n" {
		t.Fatalf("got %q", out)
	}
}

func TestBoolOps(t *testing.T) {
	out, err := runSrc(t, `
fn main(IOStream io) {
    io.println(true && false);
    io.println(true || false);
    io.println(!true);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "false\ntrue\nfalse\n" {
		t.Fatalf("got %q", out)
	}
}

func TestHashTableOps(t *testing.T) {
	out, err := runSrc(t, `
fn main(IOStream io) {
    HashTable<String, int> h = HashTable::new();
    h.put("a", 10);
    h.put("b", 20);
    io.println(h.get("a"));
    io.println(h.contains("b"));
    io.println(h.size());
    h.remove("a");
    io.println(h.size());
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "10\ntrue\n2\n1\n" {
		t.Fatalf("got %q", out)
	}
}

func TestInt32Wrap(t *testing.T) {
	out, err := runSrc(t, `
fn main(IOStream io) {
    io.println(2147483647 + 1);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "-2147483648\n" {
		t.Fatalf("got %q", out)
	}
}

func TestIORedirect(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	// explicit close: on Windows TempDir cleanup fails while the process still holds the handle (cross-system difference)
	src := fmt.Sprintf(`fn main(IOStream io) {
    OutputStream f = FileOutputStream("%s");
    io.setOut(f);
    io.println("redirected");
    f.close();
}`, filepath.ToSlash(path))
	prog, err := Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(prog, "test.qk", nil, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "redirected\n" {
		t.Fatalf("got %q", b)
	}
}

// ============ macro system ============

// Named macros: #macro name (parameters) { body }, called as name(args), parameters are substituted by name
func TestMacroNamedParams(t *testing.T) {
	out, err := runSrc(t, `#macro emit (expr) {
    #when (run) {
        #return expr
    }
}

fn main(IOStream io) {
    emit(io.println("hello from macro"));
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello from macro\n" {
		t.Fatalf("got %q", out)
	}
}

// #when(compile) blocks are dropped at run time
func TestMacroWhenCompileDropped(t *testing.T) {
	out, err := runSrc(t, `#macro only (x) {
    #when (compile) { #return io.println("COMPILE-ONLY"); }
    #when (run) {
        #return x
    }
}

fn main(IOStream io) {
    only(io.println("run-line"));
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "run-line\n" {
		t.Fatalf("got %q", out)
	}
}

// #error raises an error directly in the selected preprocessor branch
func TestMacroErrorDirective(t *testing.T) {
	_, err := runSrc(t, `#macro bad (x) {
    #when (run) {
        #error("cannot do this at run time")
    }
}

fn main(IOStream io) {
    bad(io.println("x"));
}`)
	if err == nil || !strings.Contains(err.Error(), "cannot do this at run time") {
		t.Fatalf("got %v", err)
	}
}

func TestProgramLibraryNotRunnable(t *testing.T) {
	_, err := runSrc(t, `fn main(IOStream io) {
    io.println("never");
}

program library;`)
	if err == nil || !strings.Contains(err.Error(), "cannot run a library") {
		t.Fatalf("got %v", err)
	}
}

// program main; runs normally
func TestProgramMain(t *testing.T) {
	out, err := runSrc(t, `fn main(IOStream io) {
    io.println("ok");
}

program main;`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok\n" {
		t.Fatalf("got %q", out)
	}
}

// pub / import preprocessor macros: recorded by the parse layer
func TestPubAndImportParse(t *testing.T) {
	prog, err := Compile(`import "util";
program library;

pub fn add(int a, int b) int {
    return a + b;
}

pub type struct {
    int x;
} Box;
`)
	if err != nil {
		t.Fatal(err)
	}
	if prog.Kind != "library" {
		t.Fatalf("Kind = %q", prog.Kind)
	}
	if len(prog.Imports) != 1 || prog.Imports[0] != "util" {
		t.Fatalf("Imports = %v", prog.Imports)
	}
	if len(prog.Pub) != 2 || prog.Pub[0] != "add" || prog.Pub[1] != "Box" {
		t.Fatalf("Pub = %v", prog.Pub)
	}
}

// Parameter-list and call delimiters () [] {} are interchangeable; the parameter count is unlimited but the call must match
func TestMacroDelimitersAndArity(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.ZH)) // asserts the Chinese wording; English rendering is covered by this package's i18n tests
	out, err := runSrc(t, `#macro add [a, b] {
    #when (run) {
        #return a + b
    }
}

fn main(IOStream io) {
    io.println(add(10, 32));
    io.println(add{1, 2});
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "42\n3\n" {
		t.Fatalf("got %q", out)
	}
	_, err = runSrc(t, `#macro two (a, b) {
    #return a
}

fn main(IOStream io) {
    two(1);
}`)
	if err == nil || !strings.Contains(err.Error(), "需要 2 个参数") {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteReclaimsBlock(t *testing.T) {
	prog, err := Compile(`fn main(IOStream io) {
    List<int> l = [1, 2, 3];
    io.println(l.size());
    delete l;
    GlobalMemory.compact();
}`)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	in, err := runWithInterp(prog, "test.qk", nil, strings.NewReader(""), &out)
	if err != nil {
		t.Fatal(err)
	}
	if n := in.mem.BlockCount(); n != 0 {
		t.Fatalf("expected 0 blocks after delete+compact, got %d", n)
	}
}

// pointer modifier + new <type>[size] (allocated on the heap, failure gives badAlloc)
func TestPointerAndNew(t *testing.T) {
	out, err := runSrc(t, `fn main(IOStream io) {
    pointer List<int> l = new int[10];
    io.println("ok");
    delete l;
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok\n" {
		t.Fatalf("got %q", out)
	}
}

// new with an illegal size → badAlloc
func TestNewBadAlloc(t *testing.T) {
	_, err := runSrc(t, `fn main(IOStream io) {
    try {
        pointer List<int> bad = new int[-1];
    } catch (void e) {
        io.println("badalloc");
    }
}`)
	if err != nil {
		t.Fatal(err)
	}
}

// program declaration position: either before or after (user confirmed: program main; before/after makes no difference)
func TestProgramPlacementAnywhere(t *testing.T) {
	out, err := runSrc(t, `program main;
fn main(IOStream io) {
    io.println("front-ok");
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "front-ok\n" {
		t.Fatalf("got %q", out)
	}
	out, err = runSrc(t, `fn main(IOStream io) {
    io.println("back-ok");
}
program main;`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "back-ok\n" {
		t.Fatalf("got %q", out)
	}
}

// String text-processing built-in method set
func TestStringMethods(t *testing.T) {
	out, err := runSrc(t, `fn main(IOStream io) {
    String s = "  Hello, QuarkLang World  ";
    io.println(s.trim().size());
    io.println(s.trim().toLower());
    io.println(s.trim().startsWith("Hello"));
    io.println(s.trim().endsWith("World"));
    io.println(s.trim().indexOf("Quark"));
    io.println(s.trim().substring(7, 14));
    List<String> parts = "a,b,c".split(",");
    io.println(parts.size());
    io.println("hello".replace("l", "L"));
    io.println("42".toInt());
    io.println("3.5".toFloat());
    io.println("字".charAt(0));
    io.println("x".contains(""));
}`)
	if err != nil {
		t.Fatal(err)
	}
	want := "22\nhello, quarklang world\ntrue\ntrue\n7\nQuarkLa\n3\nheLLo\n42\n3.5\n字\ntrue\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

// String out-of-bounds access raises an error
func TestStringBoundsError(t *testing.T) {
	_, err := runSrc(t, `fn main(IOStream io) {
    io.println("abc".charAt(5));
}`)
	if err == nil || !strings.Contains(err.Error(), "StringIndexOutOfBoundsError") {
		t.Fatalf("got %v", err)
	}
}

// Redeclaring a variable inside a loop must update its slot (once left the old indexOf value behind → substring out of bounds)
func TestLoopRedeclareUpdatesSlot(t *testing.T) {
	out, err := runSrc(t, `fn main(IOStream io) {
    String html = "AAA/xxxxB/yyyyyC/zzzzzz";
    int i = 0;
    while (i < 3) {
        int p = html.indexOf("/");
        io.println(p);
        html = html.substring(p + 1);
        i = i + 1;
    }
    io.println(html);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "3\n5\n6\nzzzzzz\n" {
		t.Fatalf("got %q", out)
	}
}
