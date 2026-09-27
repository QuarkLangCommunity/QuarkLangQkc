package lang

import (
	"strings"
	"testing"
)

func runCase(t *testing.T, body string) string {
	t.Helper()
	out, err := runSrc(t, body)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Equality/comparison semantics edge cases (full regression after Value was tagged)
func TestEqualityEdges(t *testing.T) {
	v := runCase(t, "fn main(IOStream io) {\n"+
		"    io.println(1 == 2.0);\n"+
		"    io.println(1.5 == 1.5);\n"+
		"    io.println(5 != 5.0);\n"+
		"    io.println(null == null);\n"+
		"    io.println(\"abc\" == \"abc\");\n"+
		"    io.println(3 < 3.5);\n"+
		"    io.println(2.0 <= 2.0);\n"+
		"    io.println(true == true);\n"+
		"}")
	want := strings.Join([]string{"false", "true", "false", "true", "true", "true", "true", "true"}, "\n") + "\n"
	if v != want {
		t.Fatalf("got %q want %q", v, want)
	}
}

// Floating-point division by zero: an error by language design (DivisionByZeroError); the error path is verified explicitly
func TestFloatDivZeroErrors(t *testing.T) {
	_, err := runSrc(t, "fn main(IOStream io) {\n"+
		"    float x = 0.0 / 0.0;\n"+
		"    io.println(x);\n"+
		"}")
	if err == nil || !strings.Contains(err.Error(), "DivisionByZeroError") {
		t.Fatalf("expected DivisionByZeroError, got %v", err)
	}
}

// Design semantics: List does not support == (class types have no value-equality semantics; the type checker rejects it explicitly)
func TestListComparisonRejected(t *testing.T) {
	_, err := runSrc(t, "fn main(IOStream io) {\n"+
		"    List<int> l = [1, 2];\n"+
		"    io.println(l == l);\n"+
		"}")
	if err == nil || !strings.Contains(err.Error(), "cannot compare") {
		t.Fatalf("expected comparison rejection, got %v", err)
	}
}

// List aliasing shares the reference: after l2=l, append is shared (which distinguishes it from deepCopy)
func TestListAliasSharing(t *testing.T) {
	v := runCase(t, "fn main(IOStream io) {\n"+
		"    List<int> l = [1, 2, 3];\n"+
		"    List<int> l2 = l;\n"+
		"    l2.append(4);\n"+
		"    io.println(l.size());\n"+
		"}")
	if v != "4\n" {
		t.Fatalf("got %q", v)
	}
}

// Correctness of mixed float and int comparison (cross-type comparison between numbers is by numeric value)
func TestMixedNumericCmp(t *testing.T) {
	v := runCase(t, "fn main(IOStream io) {\n"+
		"    io.println(7 == 7.0);\n"+
		"    io.println(7.0 == 7);\n"+
		"    io.println(8 > 7.9);\n"+
		"}")
	if v != "true\ntrue\ntrue\n" {
		t.Fatalf("got %q", v)
	}
}
