package lang

import (
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
	"strings"
	"testing"
)

const docFixture = `/* QuarkLang math micro-library
   Provides integer arithmetic. */
program library;

import "helper";

// Doubles a double.
// A second line of description.
pub fn double(int n) int {
    return n * 2;
}

// Internal helper, not exported.
fn hidden(int n) int {
    return n;
}

/* A planar point.
   Both fields are int. */
pub type struct {
    int x; // x coordinate
    int y; // y coordinate
} Point;

// Something that can be summed.
pub type interface {
    fn sum(Self self) int; // sums the two coordinates
} Summable;

type int Number;

impl {
    fn sum(Point self) int { return self.x + self.y; }
} Point;

space {
    fn max(int a, int b) int { return a; } // the larger of the two
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
		t.Errorf("Kind should be library, got %q", d.Kind)
	}
	if len(d.Imports) != 1 || d.Imports[0] != "helper" {
		t.Errorf("wrong Imports: %v", d.Imports)
	}
	if !strings.Contains(d.FileDoc, "QuarkLang math micro-library") || !strings.Contains(d.FileDoc, "Provides integer arithmetic") {
		t.Errorf("wrong file header comment: %q", d.FileDoc)
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
		t.Fatal("double was not extracted")
	}
	if fn.Signature != "fn double(int n) int" {
		t.Errorf("wrong signature: %q", fn.Signature)
	}
	if !strings.Contains(fn.Doc, "Doubles a double") || !strings.Contains(fn.Doc, "A second line of description") {
		t.Errorf("wrong function doc: %q", fn.Doc)
	}
	if !fn.Pub {
		t.Error("double should be pub")
	}
	if h := byName["hidden"]; h == nil || h.Pub {
		t.Error("hidden should exist and not be pub")
	}

	pt := byName["Point"]
	if pt == nil || pt.Kind != DocStruct {
		t.Fatalf("Point should be a struct, got %+v", pt)
	}
	if !strings.Contains(pt.Doc, "A planar point") || !strings.Contains(pt.Doc, "Both fields are int") {
		t.Errorf("wrong struct doc: %q", pt.Doc)
	}
	if len(pt.Fields) != 2 || pt.Fields[0].Name != "x" || pt.Fields[0].Type != "int" {
		t.Fatalf("wrong field extraction: %+v", pt.Fields)
	}
	if !strings.Contains(pt.Fields[0].Doc, "x coordinate") {
		t.Errorf("a field comment should sit directly above the field or at its end of line: %q", pt.Fields[0].Doc)
	}
	if pt.Signature != "type struct Point;" {
		t.Errorf("wrong struct signature: %q", pt.Signature)
	}

	iface := byName["Summable"]
	if iface == nil || iface.Kind != DocInterface || len(iface.Fields) != 1 {
		t.Fatalf("wrong interface extraction: %+v", iface)
	}
	if !strings.Contains(iface.Fields[0].Signature, "fn sum(Self self) int;") {
		t.Errorf("wrong interface method signature: %q", iface.Fields[0].Signature)
	}
	if !strings.Contains(iface.Fields[0].Doc, "sums the two coordinates") {
		t.Errorf("an interface method's trailing comment should be its doc: %q", iface.Fields[0].Doc)
	}

	var impl *DocItem
	for _, it := range d.Impls {
		if it.Kind == DocImpl {
			impl = it
		}
	}
	if impl == nil || impl.Signature != "impl Point;" || len(impl.Fields) != 1 {
		t.Fatalf("wrong impl extraction: %+v", impl)
	}
	if impl.Fields[0].Signature != "fn sum(Point self) int" {
		t.Errorf("wrong impl method signature: %q", impl.Fields[0].Signature)
	}

	var space *DocItem
	for _, it := range d.Impls {
		if it.Kind == DocSpace {
			space = it
		}
	}
	if space == nil || space.Signature != "space math;" || space.Fields[0].Signature != "fn max(int a, int b) int" {
		t.Fatalf("wrong space extraction: %+v", space)
	}

	if len(d.Libraries) != 1 || d.Libraries[0].Signature != "library gl;" {
		t.Fatalf("wrong library extraction: %+v", d.Libraries)
	}
	if got := d.Libraries[0].Fields[0].Signature; got != "fn ClearColor(float r, float g, float b, float a) void" {
		t.Errorf("wrong library symbol signature: %q", got)
	}
	if alias := byName["Number"]; alias == nil || alias.Kind != DocAlias {
		t.Errorf("wrong type alias extraction: %+v", alias)
	}
}

func TestDocMarkdownPubFilter(t *testing.T) {
	withLocalizer(t, i18n.New(i18n.ZH)) // asserts the Chinese wording; English rendering is covered by this package's i18n tests
	d, _ := buildFixtureDoc(t)
	md := d.Markdown(DocOptions{Path: "mathlib.qk"})
	if !strings.Contains(md, "# mathlib.qk") {
		t.Errorf("missing the title: %s", md)
	}
	if !strings.Contains(md, "fn double(int n) int") {
		t.Error("missing the pub function signature")
	}
	if strings.Contains(md, "hidden") {
		t.Error("non-pub symbols should be filtered by default")
	}
	if !strings.Contains(md, "| 类别 | 名称 | 签名 | 摘要 |") {
		t.Error("missing the overview table")
	}
	if !strings.Contains(md, "Doubles a double") {
		t.Error("missing the document body")
	}
	if !strings.Contains(md, "| `x` | `int` | x coordinate |") {
		t.Errorf("wrong field table:\n%s", md)
	}
	if !strings.Contains(md, "space math;") || !strings.Contains(md, "library gl;") {
		t.Error("missing the space / library section content")
	}

	mdAll := d.Markdown(DocOptions{All: true})
	if !strings.Contains(mdAll, "hidden") {
		t.Error("under -all the document should include non-pub symbols")
	}
}

// With no pub symbols in the file (program main) everything is rendered by default, so no empty document is produced.
func TestDocMarkdownNoPubRendersAll(t *testing.T) {
	src := "fn helper(int a) int {\n    return a;\n}\nfn main(IOStream io) {\n    io.println(helper(1));\n}\n"
	prog, comments, err := ParseSourceWithComments(src)
	if err != nil {
		t.Fatal(err)
	}
	md := BuildDoc(prog, comments).Markdown(DocOptions{})
	if !strings.Contains(md, "helper") || !strings.Contains(md, "main") {
		t.Errorf("a file with no pub symbols should render every symbol:\n%s", md)
	}
}

func TestDocHTML(t *testing.T) {
	// The assertions below are the Chinese wording, so the language is pinned rather than inherited from
	// the environment. They used to pass under any process language, because the kind labels and section
	// titles were Chinese string literals in the renderer; now that they are messages, a test that asserts
	// Chinese has to say so, the way TestDocMarkdownPubFilter already does.
	withLocalizer(t, i18n.New(i18n.ZH))
	d, _ := buildFixtureDoc(t)
	h := d.HTML(DocOptions{Path: "mathlib.qk"})
	for _, want := range []string{"<!doctype html>", "<title>mathlib.qk</title>", "<table>", "<pre><code>fn double(int n) int</code></pre>", "<h2>函数</h2>"} {
		if !strings.Contains(h, want) {
			t.Errorf("the HTML is missing %q", want)
		}
	}
	if strings.Contains(h, "hidden") {
		t.Error("the HTML should filter non-pub symbols by default")
	}
	if strings.Count(h, "<table>") != strings.Count(h, "</table>") {
		t.Error("unbalanced table tags")
	}
}

func TestDocCommentCleaning(t *testing.T) {
	c := Comment{Text: "// first line\n//   second line\n//\n// third line", Block: false}
	got := cleanComment(c)
	want := "first line\n  second line\n\nthird line\n"
	if got != want {
		t.Errorf("wrong line comment sanitizing:\n got %q\nwant %q", got, want)
	}
	b := Comment{Text: "/* alpha\n * beta\n */", Block: true}
	if got := cleanComment(b); !strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Errorf("wrong block comment sanitizing: %q", got)
	}
}

func TestDocSummary(t *testing.T) {
	if got := Summary("\n\nfirst line\nsecond line\n"); got != "first line" {
		t.Errorf("wrong Summary: %q", got)
	}
	if got := Summary(""); got != "" {
		t.Errorf("an empty document's Summary should be empty, got %q", got)
	}
}

// Regression: a trailing comment on the previous line must not leak into the doc comment of the next declaration.
func TestDocTrailingCommentNotLeakedToNextDecl(t *testing.T) {
	src := "type struct {\n    int kind; // 0=none 1=linear\n    int angle;\n} Grad;\n"
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
		t.Fatalf("wrong struct extraction: %+v", st)
	}
	if st.Fields[0].Doc != "0=none 1=linear" {
		t.Errorf("kind should take the trailing comment on the same line, got %q", st.Fields[0].Doc)
	}
	if st.Fields[1].Doc != "" {
		t.Errorf("angle has no comment of its own and should not inherit the previous line's trailing comment, got %q", st.Fields[1].Doc)
	}
}

// Macro definitions are cut out of the AST before parsing, so the doc model must collect them explicitly (BuildDocWithMacros).
func TestDocMacros(t *testing.T) {
	// Asserts the Chinese section title and kind label, so it pins the language for the same reason
	// TestDocHTML does: the labels are messages now, not literals.
	withLocalizer(t, i18n.New(i18n.ZH))
	src := `/* Utility macro library. */
program library;

// Doubling macro.
#macro twice (a) {
    #return a + a
}

// An ordinary function.
pub fn dbl(int n) int {
    return n * 2;
}
`
	prog, comments, macros, err := ParseSourceAll(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(macros) != 1 || macros[0].Name != "twice" {
		t.Fatalf("wrong macro definition extraction: %+v", macros)
	}
	d := BuildDocWithMacros(prog, comments, macros)
	if len(d.Macros) != 1 {
		t.Fatalf("the document model should carry 1 macro, got %d", len(d.Macros))
	}
	m := d.Macros[0]
	if m.Signature != "#macro twice (a)" {
		t.Errorf("wrong macro signature: %q", m.Signature)
	}
	if !strings.Contains(m.Doc, "Doubling macro") {
		t.Errorf("wrong macro doc comment: %q", m.Doc)
	}
	md := d.Markdown(DocOptions{})
	if !strings.Contains(md, "## 宏") || !strings.Contains(md, "#macro twice (a)") {
		t.Errorf("the Markdown should carry a macro section:\n%s", md)
	}
	if !strings.Contains(md, "| 宏 | `twice` |") {
		t.Errorf("the overview table should carry the macro entry:\n%s", md)
	}
	// Should work normally with no macros (nil safe)
	if got := BuildDocWithMacros(prog, comments, nil); len(got.Macros) != 0 {
		t.Error("a nil macros field should produce no macro entry")
	}
}
