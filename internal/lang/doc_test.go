package lang

import (
	"strings"
	"testing"
)

const docFixture = `/* QuarkLang 数学小库
   提供整数运算。 */
program library;

import "helper";

// 双精度翻倍。
// 第二行说明。
pub fn double(int n) int {
    return n * 2;
}

// 内部工具，不导出。
fn hidden(int n) int {
    return n;
}

/* 平面点。
   字段均为 int。 */
pub type struct {
    int x; // 横坐标
    int y; // 纵坐标
} Point;

// 可求值对象。
pub type interface {
    fn sum(Self self) int; // 求和
} Summable;

type int Number;

impl {
    fn sum(Point self) int { return self.x + self.y; }
} Point;

space {
    fn max(int a, int b) int { return a; } // 取大
} math;

library gl {
    fn ClearColor(float r, float g, float b, float a) void;
}

fn main(IOStream io) {
    io.println(double(2));
}
`

func buildFixtureDoc(t *testing.T) (*Doc, []Comment) {
	t.Helper()
	prog, comments, err := ParseSourceWithComments(docFixture)
	if err != nil {
		t.Fatalf("ParseSourceWithComments: %v", err)
	}
	return BuildDoc(prog, comments), comments
}

func TestDocModelExtraction(t *testing.T) {
	d, _ := buildFixtureDoc(t)
	if d.Kind != "library" {
		t.Errorf("Kind 应为 library，got %q", d.Kind)
	}
	if len(d.Imports) != 1 || d.Imports[0] != "helper" {
		t.Errorf("Imports 不对: %v", d.Imports)
	}
	if !strings.Contains(d.FileDoc, "QuarkLang 数学小库") || !strings.Contains(d.FileDoc, "提供整数运算") {
		t.Errorf("文件头注释不对: %q", d.FileDoc)
	}

	byName := map[string]*DocItem{}
	for _, it := range d.Types {
		byName[it.Name] = it
	}
	for _, it := range d.Funcs {
		byName[it.Name] = it
	}
	fn := byName["double"]
	if fn == nil {
		t.Fatal("未提取到 double")
	}
	if fn.Signature != "fn double(int n) int" {
		t.Errorf("签名不对: %q", fn.Signature)
	}
	if !strings.Contains(fn.Doc, "双精度翻倍") || !strings.Contains(fn.Doc, "第二行说明") {
		t.Errorf("函数文档不对: %q", fn.Doc)
	}
	if !fn.Pub {
		t.Error("double 应为 pub")
	}
	if h := byName["hidden"]; h == nil || h.Pub {
		t.Error("hidden 应存在且非 pub")
	}

	pt := byName["Point"]
	if pt == nil || pt.Kind != DocStruct {
		t.Fatalf("Point 应为 struct，got %+v", pt)
	}
	if !strings.Contains(pt.Doc, "平面点") || !strings.Contains(pt.Doc, "字段均为 int") {
		t.Errorf("结构体文档不对: %q", pt.Doc)
	}
	if len(pt.Fields) != 2 || pt.Fields[0].Name != "x" || pt.Fields[0].Type != "int" {
		t.Fatalf("字段提取不对: %+v", pt.Fields)
	}
	if !strings.Contains(pt.Fields[0].Doc, "横坐标") {
		t.Errorf("字段注释应紧邻字段上方/行尾: %q", pt.Fields[0].Doc)
	}
	if pt.Signature != "type struct Point;" {
		t.Errorf("结构体签名不对: %q", pt.Signature)
	}

	iface := byName["Summable"]
	if iface == nil || iface.Kind != DocInterface || len(iface.Fields) != 1 {
		t.Fatalf("接口提取不对: %+v", iface)
	}
	if !strings.Contains(iface.Fields[0].Signature, "fn sum(Self self) int;") {
		t.Errorf("接口方法签名不对: %q", iface.Fields[0].Signature)
	}
	if !strings.Contains(iface.Fields[0].Doc, "求和") {
		t.Errorf("接口方法行尾注释应为文档: %q", iface.Fields[0].Doc)
	}

	var impl *DocItem
	for _, it := range d.Impls {
		if it.Kind == DocImpl {
			impl = it
		}
	}
	if impl == nil || impl.Signature != "impl Point;" || len(impl.Fields) != 1 {
		t.Fatalf("impl 提取不对: %+v", impl)
	}
	if impl.Fields[0].Signature != "fn sum(Point self) int" {
		t.Errorf("impl 方法签名不对: %q", impl.Fields[0].Signature)
	}

	var space *DocItem
	for _, it := range d.Impls {
		if it.Kind == DocSpace {
			space = it
		}
	}
	if space == nil || space.Signature != "space math;" || space.Fields[0].Signature != "fn max(int a, int b) int" {
		t.Fatalf("space 提取不对: %+v", space)
	}

	if len(d.Libraries) != 1 || d.Libraries[0].Signature != "library gl;" {
		t.Fatalf("library 提取不对: %+v", d.Libraries)
	}
	if got := d.Libraries[0].Fields[0].Signature; got != "fn ClearColor(float r, float g, float b, float a) void" {
		t.Errorf("库符号签名不对: %q", got)
	}
	if alias := byName["Number"]; alias == nil || alias.Kind != DocAlias {
		t.Errorf("类型别名提取不对: %+v", alias)
	}
}

func TestDocMarkdownPubFilter(t *testing.T) {
	d, _ := buildFixtureDoc(t)
	md := d.Markdown(DocOptions{Path: "mathlib.qk"})
	if !strings.Contains(md, "# mathlib.qk") {
		t.Errorf("缺少标题: %s", md)
	}
	if !strings.Contains(md, "fn double(int n) int") {
		t.Error("缺少 pub 函数签名")
	}
	if strings.Contains(md, "hidden") {
		t.Error("默认应过滤掉非 pub 符号")
	}
	if !strings.Contains(md, "| 类别 | 名称 | 签名 | 摘要 |") {
		t.Error("缺少概览表")
	}
	if !strings.Contains(md, "双精度翻倍") {
		t.Error("缺少文档正文")
	}
	if !strings.Contains(md, "| `x` | `int` | 横坐标 |") {
		t.Errorf("字段表不对:\n%s", md)
	}
	if !strings.Contains(md, "space math;") || !strings.Contains(md, "library gl;") {
		t.Error("缺少 space / library 章节内容")
	}

	mdAll := d.Markdown(DocOptions{All: true})
	if !strings.Contains(mdAll, "hidden") {
		t.Error("-all 语义下应包含非 pub 符号")
	}
}

// 文件没有 pub 时（program main）默认渲染全部，避免生成空文档。
func TestDocMarkdownNoPubRendersAll(t *testing.T) {
	src := "fn helper(int a) int {\n    return a;\n}\nfn main(IOStream io) {\n    io.println(helper(1));\n}\n"
	prog, comments, err := ParseSourceWithComments(src)
	if err != nil {
		t.Fatal(err)
	}
	md := BuildDoc(prog, comments).Markdown(DocOptions{})
	if !strings.Contains(md, "helper") || !strings.Contains(md, "main") {
		t.Errorf("无 pub 文件应渲染全部符号:\n%s", md)
	}
}

func TestDocHTML(t *testing.T) {
	d, _ := buildFixtureDoc(t)
	h := d.HTML(DocOptions{Path: "mathlib.qk"})
	for _, want := range []string{"<!doctype html>", "<title>mathlib.qk</title>", "<table>", "<pre><code>fn double(int n) int</code></pre>", "<h2>函数</h2>"} {
		if !strings.Contains(h, want) {
			t.Errorf("HTML 缺少 %q", want)
		}
	}
	if strings.Contains(h, "hidden") {
		t.Error("HTML 默认应过滤非 pub 符号")
	}
	if strings.Count(h, "<table>") != strings.Count(h, "</table>") {
		t.Error("table 标签不配对")
	}
}

func TestDocCommentCleaning(t *testing.T) {
	c := Comment{Text: "// 第一行\n//   第二行\n//\n// 第三行", Block: false}
	got := cleanComment(c)
	want := "第一行\n  第二行\n\n第三行\n"
	if got != want {
		t.Errorf("行注释清洗不对:\n got %q\nwant %q", got, want)
	}
	b := Comment{Text: "/* 甲\n * 乙\n */", Block: true}
	if got := cleanComment(b); !strings.Contains(got, "甲") || !strings.Contains(got, "乙") {
		t.Errorf("块注释清洗不对: %q", got)
	}
}

func TestDocSummary(t *testing.T) {
	if got := Summary("\n\n第一行\n第二行\n"); got != "第一行" {
		t.Errorf("Summary 不对: %q", got)
	}
	if got := Summary(""); got != "" {
		t.Errorf("空文档 Summary 应为空，got %q", got)
	}
}

// 回归：上一行的行尾注释不能泄漏为下一行声明的文档注释。
func TestDocTrailingCommentNotLeakedToNextDecl(t *testing.T) {
	src := "type struct {\n    int kind; // 0=无 1=linear\n    int angle;\n} Grad;\n"
	prog, comments, err := ParseSourceWithComments(src)
	if err != nil {
		t.Fatal(err)
	}
	d := BuildDoc(prog, comments)
	var st *DocItem
	for _, it := range d.Types {
		if it.Name == "Grad" {
			st = it
		}
	}
	if st == nil || len(st.Fields) != 2 {
		t.Fatalf("结构体提取不对: %+v", st)
	}
	if st.Fields[0].Doc != "0=无 1=linear" {
		t.Errorf("kind 应取同行行尾注释，got %q", st.Fields[0].Doc)
	}
	if st.Fields[1].Doc != "" {
		t.Errorf("angle 无注释，不应继承上一行行尾注释，got %q", st.Fields[1].Doc)
	}
}
