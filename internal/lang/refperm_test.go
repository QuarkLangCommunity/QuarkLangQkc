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

	"quarklang/internal/i18n"
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
