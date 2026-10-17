package cgen

import (
	"strings"
	"testing"
)

// TestIndexErrorNamesUseDistinctRuntimeHelpers pins the compiler's half of §8's distinction: a dynamic
// container subscript calls the IndexOutOfBoundsError helper, while a dynamic bit subscript calls the
// IndexOutOfRangeError helper. The two messages are different conditions and must not share a helper.
func TestIndexErrorNamesUseDistinctRuntimeHelpers(t *testing.T) {
	listIR := transpile(t, "fn main(IOStream io) { List<int> l = [1, 2]; int i = 5; io.println(l[i]); }\n")
	if !strings.Contains(listIR, "call void @ql_panic_index") {
		t.Fatalf("container subscript does not call the IndexOutOfBoundsError helper:\n%s", listIR)
	}
	if strings.Contains(listIR, "call void @ql_panic_bits") {
		t.Fatalf("container subscript calls the bit helper too:\n%s", listIR)
	}
	bitIR := transpile(t, "fn main(IOStream io) { bits<8> b = 1; int i = 9; io.println(b[i]); }\n")
	if !strings.Contains(bitIR, "call void @ql_panic_bits") {
		t.Fatalf("bit subscript does not call the IndexOutOfRangeError helper:\n%s", bitIR)
	}
	if strings.Contains(bitIR, "call void @ql_panic_index") {
		t.Fatalf("bit subscript calls the container helper too:\n%s", bitIR)
	}
}
