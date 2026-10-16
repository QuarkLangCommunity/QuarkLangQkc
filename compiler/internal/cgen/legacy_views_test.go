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

// TestListDoubleIsLoweredLikeListInt pins the lowering §11.2 used to record as missing: `List<double>` folds
// onto `List<float>`, whose elements live in i64 slots each carrying the element as a bit-cast double — the
// same bit-cast the other float paths use — so the legacy spelling reaches the values the interpreter
// produces. The refusal that stood here was the one engine difference the parity gate could not see; it is
// closed, and this test fails loudly if the lowering goes away or the slots stop carrying the element's bits.
func TestListDoubleIsLoweredLikeListInt(t *testing.T) {
	ir := transpile(t, `fn main(IOStream io) {
    List<double> l = [1.5, -2.25, 0.5];
    io.println(l[0]);
    l[1] = 4.5;
    io.println(l[1]);
    l.append(0.125);
    io.println(l.size());
    io.println(l.get(2));
    List<float> m = [1.0];
    m.append(2.5);
    io.println(m[1]);
    float total = 0.0;
    for (double x : l) { total = total + x; }
    io.println(total);
}
`)
	if !strings.Contains(ir, "%ListF = type { i64*, i32, i32 }") {
		t.Fatalf("the float list no longer lowers onto an i64 slot buffer:\n%s", ir)
	}
	if !strings.Contains(ir, "bitcast double") || !strings.Contains(ir, "bitcast i64") {
		t.Fatalf("an element no longer travels as a bit-cast double:\n%s", ir)
	}
	if got, want := lliRun(t, withTestRuntime(ir)), "1.5\n4.5\n4\n0.5\n2.5\n6.625\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
