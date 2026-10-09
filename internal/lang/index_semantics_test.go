package lang

// Tests for the two language-level pieces an Index<Container> handle needs, per the design decision:
//
//   - `==` on handles (interface-typed values) is the same underlying block — the runtime already
//     compares object identity, and the checker now accepts interface operands;
//   - `*handle` is defined as calling get() on it, which is what makes a cursor read like a pointer.
//
// Container and index are ordinary user types here: the point is that neither Vec nor Index is a
// language-level capability — they are built from interfaces, structs and handles.

import (
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

const cursorProgram = `program main;
type interface<C> { fn get(Self self) int; fn advance(Self self) void; } Index;
type struct { List<int> data; int pos; } Cursor;
impl {
    fn get(Cursor self) int { return self.data[self.pos]; }
    fn advance(Cursor self) void { self.pos = self.pos + 1; }
} Cursor;
`

func TestIndexHandleSemantics(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    string
		wantErr string
	}{
		{
			name: "two handles to one block are equal",
			src: cursorProgram + `fn main(IOStream io) {
    Cursor a = .{[10, 20, 30], 0};
    Cursor b = a;                    // same block, not a copy
    Index<Cursor> i1 = a;
    Index<Cursor> i2 = b;
    io.println(i1 == i2);
}`,
			want: "true\n",
		},
		{
			name: "handles to different blocks are not equal",
			src: cursorProgram + `fn main(IOStream io) {
    Cursor a = .{[10, 20], 0};
    Cursor c = .{[10, 20], 0};
    Index<Cursor> i1 = a;
    Index<Cursor> i3 = c;
    io.println(i1 == i3);
}`,
			want: "false\n",
		},
		{
			name: "deref is get()",
			src: cursorProgram + `fn main(IOStream io) {
    Cursor a = .{[10, 20, 30], 0};
    Index<Cursor> i1 = a;
    io.println(*i1);
    i1.advance();
    io.println(*i1);
}`,
			want: "10\n20\n",
		},
		{
			name: "deref and get agree on the same value",
			src: cursorProgram + `fn main(IOStream io) {
    Cursor a = .{[7, 8], 1};
    Index<Cursor> i = a;
    io.println(*i == i.get());
}`,
			want: "true\n",
		},
		{
			name: "storage identity still separates the handles themselves",
			src: cursorProgram + `fn main(IOStream io) {
    Cursor a = .{[1, 2], 0};
    Cursor b = a;
    Index<Cursor> i1 = a;
    Index<Cursor> i2 = b;
    io.println(i1 == i2);   // same block
    io.println(i1 === i2);  // but the handle variables are different storage
}`,
			want: "true\nfalse\n",
		},
		{
			name:    "deref on something without get() is rejected",
			src:     `program main;` + "\n" + `fn main(IOStream io) { int x = 1; io.println(*x); }`,
			wantErr: "'*' requires a List or a container handle with get()",
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
