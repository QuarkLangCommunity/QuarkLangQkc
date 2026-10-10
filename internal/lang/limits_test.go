package lang

// Regression tests for the two explicit depth limits (see limits.go). Both limits exist so that an
// over-deep input is answered with a positioned diagnostic instead of a dead process: the front end
// refuses an expression nested deeper than MaxExprDepth, and a call chain deeper than MaxCallDepth is
// refused by the interpreter at the call site. The compiler emits the same call-depth counter into the
// code it generates, so the same source must produce the same bytes from both engines.

import (
	"errors"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

// chainSource builds a program whose single expression is a left-deep chain of n additions, on line 3.
func chainSource(n int) string {
	var b strings.Builder
	b.WriteString("// a deep expression chain\n\n")
	b.WriteString("fn main(IOStream io) { io.println(0")
	for i := 0; i < n; i++ {
		b.WriteString("+1")
	}
	b.WriteString("); }\n")
	return b.String()
}

// downSource builds a non-tail recursion n levels deep; the recursive call sits on line 3.
func downSource(n int) string {
	return "fn down(int n) int {\n  if (n == 0) { return 0; }\n  return n + down(n - 1);\n}\n" +
		"\nfn main(IOStream io) {\n  io.println(down(" + itoaTest(n) + "));\n}\n"
}

// itoaTest spells a small non-negative integer without importing strconv into this file.
func itoaTest(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// TestOverDeepExpressionIsRefusedWithPosition checks that the loudest finding of the stress suite is a
// diagnostic rather than a crash: the front end refuses the expression and says where it is. The
// wording is asserted in English, so the language is pinned instead of taken from the environment.
func TestOverDeepExpressionIsRefusedWithPosition(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	_, err := Compile(chainSource(2 * MaxExprDepth))
	if err == nil {
		t.Fatal("an expression nested past MaxExprDepth must be refused")
	}
	var ce *CheckError
	if !errors.As(err, &ce) {
		t.Fatalf("expected a positioned check error, got %T: %v", err, err)
	}
	if ce.Pos.Line != 3 {
		t.Errorf("diagnostic line = %d, want 3 (the line the expression is on)", ce.Pos.Line)
	}
	if !strings.Contains(ce.Msg, "expression nesting is deeper than") {
		t.Errorf("diagnostic does not name the limit: %q", ce.Msg)
	}
}

// TestExpressionUnderTheDepthLimitIsAccepted checks that the limit refuses only what is over it: half
// the budget must still compile, because a false refusal would break working programs.
func TestExpressionUnderTheDepthLimitIsAccepted(t *testing.T) {
	if _, err := Compile(chainSource(MaxExprDepth / 2)); err != nil {
		t.Fatalf("an expression well under MaxExprDepth must compile: %v", err)
	}
}

// TestOverDeepNestingIsRefusedByTheParser checks the shape the type checker never sees: a nest of
// parentheses (or blocks) is consumed by the parser's recursive descent, so the parser carries the same
// limit — a 300 000-deep nest used to exhaust the Go stack before the checker was ever reached. As
// above, the language is pinned because the assertion is about the English wording.
func TestOverDeepNestingIsRefusedByTheParser(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	deep := strings.Repeat("(", 2*MaxExprDepth)
	src := "fn main(IOStream io) {\n  io.println(" + deep + "1" + strings.Repeat(")", 2*MaxExprDepth) + ");\n}\n"
	_, err := Compile(src)
	if err == nil {
		t.Fatal("a nest past MaxExprDepth must be refused")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected a positioned parse error, got %T: %v", err, err)
	}
	if pe.Line != 2 {
		t.Errorf("diagnostic line = %d, want 2 (the line the nest is on)", pe.Line)
	}
	if !strings.Contains(err.Error(), "nesting is deeper than") {
		t.Errorf("diagnostic does not name the limit: %v", err)
	}
}

// TestNestingUnderTheLimitIsAccepted checks that the parser limit leaves real nesting alone: the corpus
// case is ten thousand nested parentheses, well inside the budget.
func TestNestingUnderTheLimitIsAccepted(t *testing.T) {
	deep := strings.Repeat("(", 10000)
	src := "fn main(IOStream io) {\n  io.println(" + deep + "1" + strings.Repeat(")", 10000) + ");\n}\n"
	if _, err := Compile(src); err != nil {
		t.Fatalf("ten thousand nested parentheses must compile: %v", err)
	}
}

// TestRecursionAtTheCallDepthLimitRuns checks the accepted side of the call-depth boundary.
func TestRecursionAtTheCallDepthLimitRuns(t *testing.T) {
	out, err := runSrc(t, downSource(MaxCallDepth-1))
	if err != nil {
		t.Fatalf("recursion at the cap must run: %v", err)
	}
	if out != "33550336\n" { // 1 + 2 + … + 8191
		t.Errorf("output = %q, want the sum of 1..8191", out)
	}
}

// TestRecursionPastTheCallDepthLimitIsRefused checks the refused side: one level past the cap is an
// error carrying the call site's line, which is what the compiler's generated counter must reproduce.
func TestRecursionPastTheCallDepthLimitIsRefused(t *testing.T) {
	_, err := runSrc(t, downSource(MaxCallDepth))
	if err == nil {
		t.Fatal("recursion past the cap must be refused")
	}
	if !strings.Contains(err.Error(), MaxCallDepthMessage) {
		t.Errorf("error %q does not carry the shared message %q", err, MaxCallDepthMessage)
	}
	if !strings.Contains(err.Error(), "at line 3") {
		t.Errorf("error %q does not carry the call site's line", err)
	}
}
