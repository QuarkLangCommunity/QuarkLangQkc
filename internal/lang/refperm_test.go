package lang

// Tests for permission-carrying references (R2, modelled on BioLang's typed smart references).
//
// The point of the feature is that the permission is enforced, not documented: a read-only reference
// cannot be written through, a write-only reference cannot be read, and a function-scoped reference
// cannot be parked in a program-scoped variable. These tests pin the static enforcement, and the
// compiler front end applies the same rules (it rejects the syntax outright for now, which is the
// project's policy for constructs it cannot lower — never a silently different answer).

import (
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

func TestRefPermissionEnforcement(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    string // exact stdout when the program is expected to run
		wantErr string // substring of the expected error when it must be rejected
	}{
		{
			name: "rw reference writes through and reads back",
			src: `program main;
fn main(IOStream io) { int x = 5; &rw u int p = &x; p = 7; io.println(x); io.println(p); }`,
			want: "7\n7\n",
		},
		{
			name: "read-only reference rejects a write",
			src: `program main;
fn main(IOStream io) { int x = 5; &r u int p = &x; p = 7; }`,
			wantErr: `cannot be written through (w is missing)`,
		},
		{
			name: "write-only reference rejects a read",
			src: `program main;
fn main(IOStream io) { int x = 5; &w u int p = &x; io.println(p); }`,
			wantErr: `cannot be read (r is missing)`,
		},
		{
			name: "list element reference writes through",
			src: `program main;
fn main(IOStream io) { List<int> a = [1, 2, 3]; &rw u int p = &a[1]; p = 42; io.println(a[1]); }`,
			want: "42\n",
		},
		{
			name: "struct field reference writes through",
			src: `program main;
type struct { int v; } S;
fn main(IOStream io) { S s = .{5}; &rw u int f = &s.v; f = 9; io.println(s.v); }`,
			want: "9\n",
		},
		{
			name: "bare reference keeps full permission",
			src: `program main;
fn main(IOStream io) { int& p = new int; p = 3; io.println(p); }`,
			want: "3\n",
		},
		{
			name: "address of a non-lvalue is rejected",
			src: `program main;
fn main(IOStream io) { &rw u int p = &(1 + 2); io.println(p); }`,
			wantErr: "cannot take the address",
		},
		{
			name: "function-scoped reference cannot be stored in a program-scoped variable",
			src: `program main;
fn main(IOStream io) { int x = 5; &rw f int a = &x; &rw u int b = a; io.println(b); }`,
			wantErr: "would outlive the reference",
		},
		{
			name: "a more restrictive reference cannot be widened on assignment",
			src: `program main;
fn main(IOStream io) { int x = 5; &r u int a = &x; &rw u int b = a; io.println(b); }`,
			wantErr: "the source is more restrictive",
		},
		{
			name: "permission parameters are enforced inside the callee",
			src: `program main;
fn touch(&rw u int p) void { p = 1; }
fn main(IOStream io) { int x = 0; touch(x); io.println(x); }`,
			want: "1\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withLocalizer(t, i18n.New(nil, i18n.EN))
			got, err := runSrc(t, tc.src)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, program ran and printed %q", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRefPermissionMatrix pins the parsing of the 7x4 permission/scope matrix and the helpers behind it.
func TestRefPermissionMatrix(t *testing.T) {
	for _, perm := range []string{"r", "w", "m", "rw", "rm", "wm", "rwm"} {
		if !validRefPerm(perm) {
			t.Errorf("perm %q must be valid", perm)
		}
	}
	for _, bad := range []string{"", "x", "rr", "wr", "rmw"} {
		if bad != "rmw" && validRefPerm(bad) {
			t.Errorf("perm %q must be invalid", bad)
		}
	}
	for _, scope := range []string{"u", "f", "a", "t"} {
		if !validRefScope(scope) {
			t.Errorf("scope %q must be valid", scope)
		}
	}
	if validRefScope("z") {
		t.Error(`scope "z" must be invalid`)
	}
	// A wider-scoped reference may be stored in a narrower variable, never the other way round.
	if refScopeRank("u") <= refScopeRank("f") {
		t.Error("program scope must rank wider than function scope")
	}
	if !refPermCovers("rwm", "rw") || refPermCovers("r", "rw") {
		t.Error("permission coverage is wrong")
	}
	if refBaseType("int") != "int&" || refBaseType("int&") != "int&" {
		t.Error("refBaseType must produce a bare reference type")
	}
}

// TestCopydIsOnlyAModifier pins that copyd is a modifier: the language has no Copyd<T> type and no
// T[Copyd] form, while the modifier keeps its deep-copy-on-pass semantics.
func TestCopydIsOnlyAModifier(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	cases := []struct {
		name    string
		src     string
		want    string
		wantErr string
	}{
		{
			name: "modifier deep-copies a list on pass",
			src: `program main;
fn take(copyd List<int> a) int { a.append(9); return a.size(); }
fn main(IOStream io) { List<int> l = [1, 2]; io.println(take(l)); io.println(l.size()); }`,
			want: "3\n2\n",
		},
		{
			name: "modifier deep-copies a scalar on pass",
			src: `program main;
fn take(copyd int a) int { a = a + 1; return a; }
fn main(IOStream io) { int x = 5; io.println(take(x)); io.println(x); }`,
			want: "6\n5\n",
		},
		{
			name:    "Copyd<T> is not a type",
			src:     `program main;` + "\n" + `fn main(IOStream io) { Copyd<int> c = null; io.println(1); }`,
			wantErr: "Copyd is a modifier, not a type",
		},
		{
			name:    "Copyd<T> as a parameter type is not a type",
			src:     `program main;` + "\n" + `fn f(Copyd<int> a) int { return 1; }` + "\n" + `fn main(IOStream io) { io.println(1); }`,
			wantErr: "Copyd is a modifier, not a type",
		},
		{
			name:    "the [Copyd] suffix is not accepted either",
			src:     `program main;` + "\n" + `fn f(int[Copyd] a) int { return 1; }` + "\n" + `fn main(IOStream io) { io.println(1); }`,
			wantErr: "ParseError",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runSrc(t, tc.src)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, program ran and printed %q", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestForInProtocol pins that for-in accepts a library container: any type with size() int and
// get(int) T can be iterated, which is what lets the standard-library Vec work in a loop while the
// core knows nothing about it.
func TestForInProtocol(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	cases := []struct {
		name    string
		src     string
		want    string
		wantErr string
	}{
		{
			name: "user type with size()/get() is iterable",
			src: `program main;
type struct { int n; } Squares;
impl { fn size(Squares self) int { return self.n; } fn get(Squares self, int i) int { return i * i; } } Squares;
fn main(IOStream io) { Squares s = .{3}; int sum = 0; for (int v : s) { sum = sum + v; } io.println(sum); }`,
			want: "5\n",
		},
		{
			name: "loop variable type is checked against get()",
			src: `program main;
type struct { int n; } Squares;
impl { fn size(Squares self) int { return self.n; } fn get(Squares self, int i) int { return i * i; } } Squares;
fn main(IOStream io) { Squares s = .{3}; for (String v : s) { io.println(v); } }`,
			wantErr: "does not match element type",
		},
		{
			name:    "a type without the protocol is rejected",
			src:     `program main;` + "\n" + `type struct { int n; } Plain;` + "\n" + `fn main(IOStream io) { Plain p = .{3}; for (int v : p) { io.println(v); } }`,
			wantErr: "for-in requires a List or a container with size() int and get(int)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runSrc(t, tc.src)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, program ran and printed %q", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestLiteralProtocol pins that [...] is an overloadable notation: a type declaring a static
// __literal__() (an empty container) plus __element__(Self, T) is built from a literal, and the core
// knows only those two protocol names — not the container.
func TestLiteralProtocol(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	cases := []struct {
		name    string
		src     string
		want    string
		wantErr string
	}{
		{
			name: "a user type with the protocol is built from a literal",
			src: `program main;
type struct { int sum; int n; } Tally;
impl {
    fn __literal__() Tally { Tally t = .{0, 0}; return t; }
    fn __element__(Tally self, int v) void { self.sum = self.sum + v; self.n = self.n + 1; }
    fn total(Tally self) int { return self.sum; }
    fn count(Tally self) int { return self.n; }
} Tally;
fn main(IOStream io) { Tally t = [1, 2, 3]; io.println(t.count()); io.println(t.total()); }`,
			want: "3\n6\n",
		},
		{
			name: "a wrong element type is rejected",
			src: `program main;
type struct { int sum; } Tally;
impl {
    fn __literal__() Tally { Tally t = .{0}; return t; }
    fn __element__(Tally self, int v) void { self.sum = self.sum + v; }
} Tally;
fn main(IOStream io) { Tally t = ["x"]; io.println(t.sum); }`,
			wantErr: "literal element is",
		},
		{
			name: "__literal__ without __element__ is rejected",
			src: `program main;
type struct { int n; } Half;
impl { fn __literal__() Half { Half h = .{0}; return h; } } Half;
fn main(IOStream io) { Half h = [1]; io.println(h.n); }`,
			wantErr: "no __element__",
		},
		{
			name: "a type without the protocol keeps the plain literal",
			src: `program main;
type struct { int n; } Plain;
fn main(IOStream io) { Plain p = .{1}; List<int> l = [1, 2]; io.println(l.size()); io.println(p.n); }`,
			want: "2\n1\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runSrc(t, tc.src)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, program ran and printed %q", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
