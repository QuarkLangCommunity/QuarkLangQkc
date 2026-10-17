package cgen

import (
	"strings"
	"testing"
)

// TestMemoryPrimitivesAreRefusedExceptListDelete pins the compiler's half of §10: every row but a
// List-variable delete is refused explicitly, and that one is lowered.
func TestMemoryPrimitivesAreRefusedExceptListDelete(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"delete int", "fn main(IOStream io) { int x = 1; delete x; }\n", "delete"},
		{"setBlock", "fn main(IOStream io) { memory.setBlock(1024); }\n", "setBlock"},
		{"compact", "fn main(IOStream io) { memory.compact(); }\n", "compact"},
		{"Fragmentation", "fn main(IOStream io) { io.println(memory.Fragmentation()); }\n", "Fragmentation"},
		{"clear", "fn main(IOStream io) { memory.clear(); }\n", "clear"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := lowerErr(t, tc.src)
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not name %q", err.Error(), tc.want)
			}
		})
	}
	transpile(t, "fn main(IOStream io) { List<int> l = [1]; delete l; }\n")
}
