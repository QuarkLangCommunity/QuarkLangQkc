package lang

// Tests for the === storage-identity primitive.
//
// === compares where two operands live, never what they hold. Defining properties:
//   x === x            true   (the same storage)
//   x === y            false  (different storage, even when the values are equal)
//   two pointers to the same target: == is true, === is false (the pointers themselves live in
//                                   different places — the reason === exists next to ==)
//   literals/temporaries: false (each evaluation has its own temporary storage)
// It is a primitive, so it cannot be overloaded and it accepts operands of any type.

import (
	"strings"
	"testing"

	"quarklang/internal/i18n"
)

func TestStrictEqStorageIdentity(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"same variable", `program main;
fn main(IOStream io) { int x = 5; io.println(x === x); }`, "true\n"},
		{"distinct variables with equal values", `program main;
fn main(IOStream io) { int x = 5; int y = 5; io.println(x == y); io.println(x === y); }`, "true\nfalse\n"},
		{"same list element", `program main;
fn main(IOStream io) { List<int> a = [1, 2]; io.println(a[0] === a[0]); }`, "true\n"},
		{"different list elements", `program main;
fn main(IOStream io) { List<int> a = [1, 2]; io.println(a[0] === a[1]); }`, "false\n"},
		{"two handles to one list are different storage", `program main;
fn main(IOStream io) { List<int> a = [1, 2]; List<int> b = a; io.println(a === b); io.println(a === a); }`, "false\ntrue\n"},
		{"literals are temporaries", `program main;
fn main(IOStream io) { io.println(5 === 5); }`, "false\n"},
		{"accepted for any type pair", `program main;
fn main(IOStream io) { int x = 1; String s = "a"; io.println(x === s); }`, "false\n"},
		{"struct field", `program main;
type struct { int v; } S;
fn main(IOStream io) { S a = .{1}; S b = .{1}; io.println(a.v === a.v); io.println(a.v === b.v); }`, "true\nfalse\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withLocalizer(t, i18n.New(nil, i18n.EN))
			got, err := runSrc(t, tc.src)
			if err != nil {
				t.Fatalf("run error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestStrictEqNotOverloadable pins the primitive's second defining property: a user-defined __eq__
// changes == but never ===.
func TestStrictEqNotOverloadable(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	src := `program main;
type struct { int v; } S;
impl {
    fn eq(Self a, S b) bool { return true; }
    fn __eq__(Self a, S b) bool { return true; }
} S;
fn main(IOStream io) { S a = .{1}; S b = .{2}; io.println(a === b); }`
	got, err := runSrc(t, src)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if got != "false\n" {
		t.Fatalf("=== must stay false regardless of user operator methods, got %q", got)
	}
	if m := opMethodFor("==="); m != "" {
		t.Fatalf("opMethodFor(===) must be empty so no user method can hook it, got %q", m)
	}
}

// TestStrictEqLexingIsNotSplit guards the lexer ordering: === must never lex as == then =.
func TestStrictEqLexingIsNotSplit(t *testing.T) {
	toks, err := Lex(`a === b`)
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}
	var kinds []string
	for _, tk := range toks {
		kinds = append(kinds, tk.Text)
	}
	joined := strings.Join(kinds, " ")
	if !strings.Contains(joined, "===") {
		t.Fatalf("expected a === token, got %q", joined)
	}
	if strings.Contains(joined, "== =") {
		t.Fatalf("=== was split into == and =: %q", joined)
	}
}
