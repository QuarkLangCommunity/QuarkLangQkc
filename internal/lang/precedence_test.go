package lang

import (
	"fmt"
	"testing"
)

// binShape renders a binary-expression tree as text, so a test can state which operand an operator takes.
func binShape(e Expr) string {
	switch x := e.(type) {
	case *BinOp:
		return "(" + binShape(x.L) + " " + x.Op + " " + binShape(x.R) + ")"
	case *UnOp:
		return "(" + x.Op + binShape(x.X) + ")"
	case *IntLit:
		return fmt.Sprintf("%d", x.V)
	case *Ident:
		return x.Name
	}
	return fmt.Sprintf("%T", e)
}

// declShape returns the shape of the initializer of the first declaration in the source's first function.
func declShape(t *testing.T, src string) string {
	t.Helper()
	p, err := ParseSource(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range p.Funcs {
		for _, st := range f.Body.Stmts {
			if d, ok := st.(*DeclStmt); ok {
				return binShape(d.Init)
			}
		}
	}
	return "(no declaration)"
}

// TestShiftGroupingComesFromTheSyntaxLayer records why §11.3's shift divergence cannot be corrected in this
// repository: the pinned syntax layer erases parentheses (qkparser/parser.go, the TLParen case returns the
// inner expression), so `1 + 2 << 3` and `1 + (2 << 3)` reach every engine as one and the same tree — and a
// source that the standard reads as 24 and 17 respectively can only be told apart while the parentheses are
// still there, which is inside the syntax layer.
func TestShiftGroupingComesFromTheSyntaxLayer(t *testing.T) {
	unparenthesized := declShape(t, "fn f() int { int a = 1 + 2 << 3; return a; }")
	parenthesized := declShape(t, "fn f() int { int a = 1 + (2 << 3); return a; }")
	if unparenthesized != parenthesized {
		t.Fatalf("the syntax layer now keeps the two spellings apart (%s vs %s), so §11.3's fix can land in this repository: update this test and the three marked values in TestPrecedenceMatchesTheStandard", unparenthesized, parenthesized)
	}
	if unparenthesized != "(1 + (2 << 3))" {
		t.Fatalf("the pinned grouping changed to %s: re-read spec §5 and §11.3 before changing this test", unparenthesized)
	}
}

// TestPrecedenceMatchesTheStandardEverywhereItCan pins every level of §5's precedence table. The three
// shift expressions carry the values the pinned syntax layer produces today (17/7/9); §5's order makes them
// 24/32/4, and §11.3 records that gap together with the two-line fix it needs, so a failure here means the
// grouping moved and a number has to be re-read rather than re-derived.
func TestPrecedenceMatchesTheStandardEverywhereItCan(t *testing.T) {
	got := runCase(t, "fn main(IOStream io) {\n"+
		"    io.println(true || false && false);\n"+
		"    io.println(1 | 2 == 3);\n"+
		"    io.println(1 | 2 ^ 3);\n"+
		"    io.println(4 ^ 1 & 1);\n"+
		"    io.println(3 & 1 << 2);\n"+
		"    io.println(1 + 2 << 3);\n"+
		"    io.println(1 << 2 + 3);\n"+
		"    io.println(10 - 2 >> 1);\n"+
		"    io.println(1 + (2 << 3));\n"+
		"    io.println((1 + 2) << 3);\n"+
		"    io.println(1 + 2 * 3);\n"+
		"    io.println(1 - 2 - 3);\n"+
		"    io.println(-1 + 2);\n"+
		"    io.println(~0 + 1);\n"+
		"}")
	want := "true\ntrue\n1\n5\n0\n17\n7\n9\n17\n24\n7\n-4\n1\n0\n"
	if got != want {
		t.Fatalf("precedence moved:\n got %q\nwant %q\n§11.3 marks the shift level: 17/7/9 become 24/32/4 once the syntax layer is corrected.", got, want)
	}
}
