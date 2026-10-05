package lang

// Tests for generic interfaces: `type interface<T> { ... } Index;` used as `Index<Vec<int>>`.
//
// This is what lets a container type be described by an interface that talks about its element type —
// the shape the Vec/Index standard-library work needs: Vec<T> implements Index with T substituted, and
// a value of type Index<Vec<int>> then answers `size()` and `get(i)` with int.

import (
	"strings"
	"testing"

	"quarklang/internal/i18n"
)

const genericIndexProgram = `program main;
type interface<T> { fn get(Self self, int i) T; fn size(Self self) int; } Index;
type struct<T> { List<T> data; } Vec;
impl<T> {
    fn get(Vec<T> self, int i) T { return self.data[i]; }
    fn size(Vec<T> self) int { return self.data.size(); }
} Vec;
`

func TestGenericInterfaceInstantiation(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    string
		wantErr string
	}{
		{
			name: "generic struct implements the instantiated interface",
			src: genericIndexProgram + `fn main(IOStream io) {
    Vec<int> v = .{[10, 20, 30]};
    Index<Vec<int>> x = v;
    io.println(x.size());
    io.println(x.get(1));
}`,
			want: "3\n20\n",
		},
		{
			// Unambiguous direction check: the interface's parameter is the *container* type, so a
			// method returning T returns the container, and the compiler must type it that way rather
			// than falling back to a concrete impl signature.
			name: "the type argument substitutes into the method signature",
			src: `program main;
type interface<T> { fn self(Self self) T; } Same;
type struct<T> { List<T> data; } Vec;
impl<T> { fn self(Vec<T> self) Vec<T> { return self; } } Vec;
fn main(IOStream io) {
    Vec<int> v = .{[1, 2, 3]};
    Same<Vec<int>> s = v;
    Vec<int> got = s.self();
    io.println(got.data.size());
}`,
			want: "3\n",
		},
		{
			name:    "wrong number of type arguments",
			src:     `program main;` + "\n" + `type interface<T> { fn get(Self self) T; } Index;` + "\n" + `fn main(IOStream io) { Index<int, int> bad = null; }`,
			wantErr: "Index takes 1 type argument(s), got 2",
		},
		{
			name:    "type arguments on a non-generic interface",
			src:     `program main;` + "\n" + `type interface { fn n(Self self) int; } Plain;` + "\n" + `fn main(IOStream io) { Plain<int> bad = null; }`,
			wantErr: "interface Plain is not generic",
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
