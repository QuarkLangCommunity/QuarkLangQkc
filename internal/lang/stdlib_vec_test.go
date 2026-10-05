package lang

// Tests for the Vec standard library (stdlib/vec.qk).
//
// Vec is a library type, not a language capability: the core provides raw blocks (new T[n] →
// pointer T), references, structs and generics, and the dynamic array is written in QuarkLang on top
// of them. These tests exercise it the way a program does — through `import "vec"` — so growth,
// element access and the capacity contract are checked end to end.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quarklang/internal/i18n"
)

// runWithStdlib copies the repository's stdlib into a temp dir alongside the program and runs it.
func runWithStdlib(t *testing.T, src string) (string, error) {
	t.Helper()
	withLocalizer(t, i18n.New(nil, i18n.EN))
	dir := t.TempDir()
	lib, err := os.ReadFile(filepath.Join("..", "..", "stdlib", "vec.qk"))
	if err != nil {
		t.Fatalf("read stdlib/vec.qk: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vec.qk"), lib, 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.qk")
	if err := os.WriteFile(mainPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	prog, _, err := CompileWithImportsMapped(src, mainPath)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = Run(prog, mainPath, nil, strings.NewReader(""), &out)
	return out.String(), err
}

func TestStdlibVec(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    string
		wantErr string
	}{
		{
			name: "push grows the block and preserves the elements",
			src: `program main;
import "vec";
fn main(IOStream io) {
    Vec<int> v = vecNew(0, 2);   // capacity 2
    v.push(10);
    v.push(20);
    io.println(v.capacity());
    v.push(30);                  // forces growth
    io.println(v.capacity());
    io.println(v.size());
    io.println(v.get(0));
    io.println(v.get(1));
    io.println(v.get(2));
}`,
			want: "2\n4\n3\n10\n20\n30\n",
		},
		{
			name: "set and get address individual elements",
			src: `program main;
import "vec";
fn main(IOStream io) {
    Vec<int> v = vecNew(0, 4);
    v.push(1); v.push(2); v.push(3);
    v.set(1, 99);
    io.println(v.get(1));
    io.println(v.size());
}`,
			want: "99\n3\n",
		},
		{
			name: "pop returns the last element and shrinks the length",
			src: `program main;
import "vec";
fn main(IOStream io) {
    Vec<int> v = vecNew(0, 2);
    v.push(7); v.push(8);
    io.println(v.pop());
    io.println(v.size());
    io.println(v.capacity());   // capacity is not given back on pop
}`,
			want: "8\n1\n2\n",
		},
		{
			name: "clear keeps the storage",
			src: `program main;
import "vec";
fn main(IOStream io) {
    Vec<int> v = vecNew(0, 2);
    v.push(1); v.push(2);
    v.clear();
    io.println(v.size());
    v.push(5);
    io.println(v.get(0));
}`,
			want: "0\n5\n",
		},
		{
			name: "iteration follows the size()/get() protocol",
			src: `program main;
import "vec";
fn main(IOStream io) {
    Vec<int> v = vecNew(0, 2);
    v.push(10); v.push(20); v.push(30);
    int sum = 0;
    for (int x : v) { sum = sum + x; }
    io.println(sum);
}`,
			want: "60\n",
		},
		{
			name: "break leaves the loop early",
			src: `program main;
import "vec";
fn main(IOStream io) {
    Vec<int> v = vecNew(0, 4);
    v.push(1); v.push(2); v.push(3); v.push(4);
    int seen = 0;
    for (int x : v) {
        if (x == 3) { break; }
        seen = seen + 1;
    }
    io.println(seen);
}`,
			want: "2\n",
		},
		{
			name: "elements survive many growth steps",
			src: `program main;
import "vec";
fn main(IOStream io) {
    Vec<int> v = vecNew(0, 1);
    int i = 0;
    while (i < 50) { v.push(i); i = i + 1; }
    io.println(v.size());
    int sum = 0;
    for (int j = 0; j < v.size(); j = j + 1) { sum = sum + v.get(j); }
    io.println(sum);
}`,
			want: "50\n1225\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runWithStdlib(t, tc.src)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got output %q", tc.wantErr, got)
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
