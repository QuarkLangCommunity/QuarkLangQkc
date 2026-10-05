package lang

// Tests for the move permission (m) and its syntax p++ — BioLang's "move the pointer".
//
// A movable reference walks a container: p++ advances it one element and rebinds the variable that
// holds it. Only element references can move, moving past the end is an error rather than a dangling
// reference, and the m permission is required — a &rw reference refuses to move, exactly like a &r
// reference refuses to be written through.

import (
	"strings"
	"testing"

	"quarklang/internal/i18n"
)

func TestReferenceMove(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    string
		wantErr string
	}{
		{
			name: "walking a container",
			src: `program main;
fn main(IOStream io) { List<int> a = [10, 20, 30]; &rm u int q = &a[0]; io.println(q); q++; io.println(q); q++; io.println(q); }`,
			want: "10\n20\n30\n",
		},
		{
			name: "writing through a moved reference",
			src: `program main;
fn main(IOStream io) { List<int> a = [10, 20, 30]; &rwm u int q = &a[0]; q++; q = 99; io.println(a[1]); }`,
			want: "99\n",
		},
		{
			name: "move requires the m permission",
			src: `program main;
fn main(IOStream io) { List<int> a = [10, 20]; &rw u int q = &a[0]; q++; }`,
			wantErr: "cannot move (m is missing)",
		},
		{
			name: "moving past the end is an error",
			src: `program main;
fn main(IOStream io) { List<int> a = [10, 20]; &rm u int q = &a[1]; q++; }`,
			wantErr: "past the end of the container",
		},
		{
			name: "only element references move",
			src: `program main;
fn main(IOStream io) { int x = 1; &rm u int p = &x; p++; }`,
			wantErr: "only moves element references",
		},
		{
			name: "a helper moving a reference moves the caller's variable",
			src: `program main;
fn advance(&rm u int q) void { q++; }
fn main(IOStream io) { List<int> a = [7, 8, 9]; &rm u int p = &a[0]; io.println(p); advance(p); io.println(p); }`,
			want: "7\n8\n",
		},
		{
			name: "moving a non-reference is rejected",
			src: `program main;
fn main(IOStream io) { int x = 1; int y = 2; y++; }`,
			wantErr: "++ needs a reference",
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
