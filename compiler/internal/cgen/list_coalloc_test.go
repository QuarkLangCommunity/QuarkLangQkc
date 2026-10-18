package cgen

import (
	"strings"
	"testing"
)

// listAllocCall / listCallocCall are the allocation entry points the emitter uses on this platform.
// The list-literal shape (one block vs. two) must be visible in the IR, so these tests count calls
// rather than trusting a comment.
func listAllocCall() string {
	if hostAllocatorFallback {
		return "call i8* @malloc("
	}
	return "call i8* @ql_alloc("
}

func listCallocCall() string {
	if hostAllocatorFallback {
		return "call i8* @calloc("
	}
	return "call i8* @ql_calloc("
}

// TestSmallListLiteralIsOneBlock pins the co-allocation budget for the three list layouts. A literal
// below the budget allocates its header and elements in one counted-slab block; a literal above it
// keeps the header and buffer as two allocations. The budget only ever picks the shape that does not
// reserve more slot bytes, and every element is still written through the returned buffer pointer.
func TestSmallListLiteralIsOneBlock(t *testing.T) {
	cases := []struct {
		name      string
		src       string
		wantAlloc int
		wantCall  int
	}{
		{
			name:      "int3_one_block",
			src:       "fn main(IOStream io) {\n    List<int> l = [1, 2, 3];\n    io.println(l.size());\n}\n",
			wantAlloc: 1, wantCall: 0,
		},
		{
			name:      "int13_spills",
			src:       "fn main(IOStream io) {\n    List<int> l = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13];\n    io.println(l.size());\n}\n",
			wantAlloc: 1, wantCall: 1,
		},
		{
			name:      "string2_one_block",
			src:       "fn main(IOStream io) {\n    List<String> l = [\"a\", \"b\"];\n    io.println(l.size());\n}\n",
			wantAlloc: 0, wantCall: 1,
		},
		{
			name:      "string7_spills",
			src:       "fn main(IOStream io) {\n    List<String> l = [\"a\", \"b\", \"c\", \"d\", \"e\", \"f\", \"g\"];\n    io.println(l.size());\n}\n",
			wantAlloc: 0, wantCall: 2,
		},
		{
			name:      "float2_one_block",
			src:       "fn main(IOStream io) {\n    List<float> l = [1.5, 2.5];\n    io.println(l.size());\n}\n",
			wantAlloc: 0, wantCall: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ir := transpile(t, c.src)
			if got := strings.Count(ir, listAllocCall()); got != c.wantAlloc {
				t.Fatalf("IR has %d %q call(s), want %d:\n%s", got, listAllocCall(), c.wantAlloc, ir)
			}
			if got := strings.Count(ir, listCallocCall()); got != c.wantCall {
				t.Fatalf("IR has %d %q call(s), want %d:\n%s", got, listCallocCall(), c.wantCall, ir)
			}
		})
	}
}

// TestListAppendUsesOneGrowHelper: every append/appendAll site goes through the generated
// @ql_list_grow, which migrates an inline buffer before reallocating. The helper is emitted once,
// whatever the element layout or the number of call sites.
func TestListAppendUsesOneGrowHelper(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    List<int> a = [1, 2, 3];\n"+
		"    a.append(4);\n"+
		"    List<String> s = [\"a\"];\n"+
		"    s.append(\"b\");\n"+
		"    List<float> f = [1.5];\n"+
		"    f.append(2.5);\n"+
		"    io.println(a.size(), s.size(), f.size());\n"+
		"}\n")
	if got := strings.Count(ir, "define internal i8* @ql_list_grow("); got != 1 {
		t.Fatalf("IR defines @ql_list_grow %d time(s), want 1:\n%s", got, ir)
	}
	if got := strings.Count(ir, "call i8* @ql_list_grow("); got != 3 {
		t.Fatalf("IR calls @ql_list_grow %d time(s), want 3 (one per append):\n%s", got, ir)
	}
}

// TestListLiteralBudgetNeverReservesMore checks the choice against the runtime's slot classes for
// every literal length and element width the compiler can emit: a co-allocated shape must never
// reserve more slot bytes than the two-allocation shape it replaces.
func TestListLiteralBudgetNeverReservesMore(t *testing.T) {
	for _, elem := range []int{4, 8} {
		for n := 0; n <= 64; n++ {
			if !listLiteralCoallocates(n, elem) {
				continue
			}
			cap := n
			if cap < 1 {
				cap = 1
			}
			combined := qlAllocatorSlotBytes(qlListHeaderBytes + cap*elem)
			separate := qlAllocatorSlotBytes(qlListHeaderBytes) + qlAllocatorSlotBytes(cap*elem)
			if combined == 0 || combined > separate {
				t.Fatalf("elem=%d n=%d co-allocates but reserves %d bytes vs %d separate", elem, n, combined, separate)
			}
		}
	}
}

// TestCoallocatedListGrowsAndReleases runs the generated helpers through lli with the test runtime's
// libc-backed allocator stubs: a small literal starts as one block, append migrates the inline buffer,
// the elements survive the move, and delete releases the list once. (End-to-end parity with the real
// counted-slab runtime is exercised by cases_run/z_list_coalloc.kq and z_delete_churn.kq.)
func TestCoallocatedListGrowsAndReleases(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    List<int> l = [1, 2, 3];\n"+
		"    l.append(4);\n"+
		"    l.append(5);\n"+
		"    io.println(l.size(), l[0], l[4]);\n"+
		"    delete l;\n"+
		"    List<int> e = [];\n"+
		"    e.append(7);\n"+
		"    io.println(e.size(), e[0]);\n"+
		"    delete e;\n"+
		"}\n")
	if got, want := lliRun(t, ir), "5 1 5\n1 7\n"; got != want {
		t.Fatalf("got %q want %q\nIR:\n%s", got, want, ir)
	}
}
