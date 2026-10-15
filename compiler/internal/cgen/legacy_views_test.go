package cgen

import (
	"strings"
	"testing"
)

// TestLegacySpellingsFoldOntoTheViews pins §9's legacy spellings onto the views this backend lowers: `char`
// is the int view and `double` is the 64-bit float view, in a declaration, a parameter, a return, a struct
// field and a List element, because the interpreter accepts all of them and an engine difference is not a
// licence for a hard error (spec §1.4, §11.3).
func TestLegacySpellingsFoldOntoTheViews(t *testing.T) {
	ir := transpile(t, `type struct { char c; double d; } Pair;
fn bumpc(char c) char { return c + 1; }
fn main(IOStream io) {
    char c = 65;
    double d = 1.5;
    io.println(c);
    io.println(d);
    io.println(char(66));
    io.println(bumpc(c));
    Pair p = .{c: 1, d: 2.5};
    io.println(p.c);
    io.println(p.d);
    List<char> l = [1, 2];
    io.println(l[0]);
}
`)
	if !strings.Contains(ir, "define i32 @main()") {
		t.Fatalf("empty ir:\n%s", ir)
	}
	if got, want := lliRun(t, withTestRuntime(ir)), "65\n1.5\n66\n66\n1\n2.5\n1\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// TestListDoubleIsRefusedWhileTheInterpreterAcceptsIt pins the one engine difference the legacy spellings
// keep — `List<char>` folds to `List<int>` and is lowered, while `List<double>` would need a `List<float>`
// this backend does not have (§11.2) — by the folded type name, which no locale translates, so the
// divergence fails loudly here instead of staying invisible in the parity gate.
func TestListDoubleIsRefusedWhileTheInterpreterAcceptsIt(t *testing.T) {
	err := lowerErr(t, "fn main(IOStream io) {\n"+
		"    List<double> l = [1.5];\n"+
		"    io.println(l[0]);\n"+
		"}\n")
	if !strings.Contains(err.Error(), "List<float>") {
		t.Fatalf("the refusal changed: %v", err)
	}
}
