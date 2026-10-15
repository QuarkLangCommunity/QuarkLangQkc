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

// TestShiftGroupingComesFromTheSyntaxLayer pins where the shift level is decided: the syntax layer, whose
// parser now threads `parseBitAnd → parseShift → parseAdd → parseMul`. It erases the parenthesis node
// (qkparser/parser.go, the TLParen case returns the inner expression), so a later pass receives a tree
// whose operands are already grouped and cannot re-group them; what it must not do is conflate the two
// readings, and this test holds that line. §11.3 recorded the divergence and the fix while the code was
// wrong; the divergence is closed, and this is the shape that closed it.
func TestShiftGroupingComesFromTheSyntaxLayer(t *testing.T) {
	unparenthesized := declShape(t, "fn f() int { int a = 1 + 2 << 3; return a; }")
	shiftParenthesized := declShape(t, "fn f() int { int a = 1 + (2 << 3); return a; }")
	addParenthesized := declShape(t, "fn f() int { int a = (1 + 2) << 3; return a; }")
	if unparenthesized != "((1 + 2) << 3)" {
		t.Fatalf("`1 + 2 << 3` groups as %s, want ((1 + 2) << 3) — spec §5 puts `<<` looser than `+`", unparenthesized)
	}
	if shiftParenthesized != "(1 + (2 << 3))" {
		t.Fatalf("`1 + (2 << 3)` groups as %s, want (1 + (2 << 3)) — an explicit parenthesis must survive as grouping", shiftParenthesized)
	}
	if unparenthesized == shiftParenthesized {
		t.Fatalf("the syntax layer conflates `1 + 2 << 3` and `1 + (2 << 3)` (%s): §11.3's divergence is back", unparenthesized)
	}
	if addParenthesized != unparenthesized {
		t.Fatalf("`(1 + 2) << 3` groups as %s but `1 + 2 << 3` as %s: §5 reads them alike", addParenthesized, unparenthesized)
	}
}

// TestPrecedenceMatchesTheStandard pins every level of §5's precedence table, one row per adjacency, and
// the three shift expressions carry the values §5 states (24/32/4). A failure here means the grouping
// moved: re-read spec §5 and §11.3 before re-deriving a number, because this test exists to notice a
// precedence change rather than to describe one.
func TestPrecedenceMatchesTheStandard(t *testing.T) {
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
		"    io.println(2 * 3 % 4);\n"+
		"    io.println(1 - 2 - 3);\n"+
		"    io.println(-1 + 2);\n"+
		"    io.println(~0 + 1);\n"+
		"}")
	want := "true\ntrue\n1\n5\n0\n24\n32\n4\n17\n24\n7\n2\n-4\n1\n0\n"
	if got != want {
		t.Fatalf("precedence moved:\n got %q\nwant %q\n§5's order makes the shift rows 24/32/4; §11.3 records that this level was the one divergence and that it is closed.", got, want)
	}
}
