package main

// D1's audit gate. The allocator lives in qthreads.c.txt, and so do the runtime's own allocations
// (strings, split lists, hash tables, channels and tasks). Stage 2 routes all of them through the same
// counted-slab API the generated program uses, so occupancy and fragmentation describe the whole
// compiled artifact. The only libc allocation left is the allocator's own host block storage: a block
// is the unit the metric reports on, so its host bookkeeping is the one thing that cannot be counted
// inside it. This test fails if a new runtime helper reaches for malloc/calloc/realloc/free directly.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// rawAllocation matches a libc allocation call but not a routed one: the identifier before the call
// may not be part of a longer name, which is what keeps `ql_free(` and `ql_calloc(` out.
var rawAllocation = regexp.MustCompile(`(^|[^A-Za-z0-9_])(malloc|calloc|realloc|free)\s*\(`)

// hostAllocatorLines are the two lines the allocator itself owns: the host block and its release.
var hostAllocatorLines = map[string]bool{
	"void* raw = malloc(size + QL_MEM_ALIGN + sizeof(void*));": true,
	"if (p) free(((void**)p)[-1]);":                            true,
}

func TestRuntimeAllocationsStayRouted(t *testing.T) {
	src, err := os.ReadFile("qthreads.c.txt")
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(src), "\n") {
		code := line
		if j := strings.Index(code, "//"); j >= 0 {
			code = code[:j]
		}
		if !rawAllocation.MatchString(code) {
			continue
		}
		if hostAllocatorLines[strings.TrimSpace(code)] {
			continue
		}
		t.Errorf("qthreads.c.txt:%d reaches the host allocator outside ql_mem_host_alloc/free; route it through ql_alloc/ql_calloc/ql_realloc/ql_free: %s", i+1, strings.TrimSpace(code))
	}

	// The audit would also pass if routing were deleted wholesale, so pin the shapes the runtime's own
	// helpers use: the task constructor, the string builder, and the hash-table growth release.
	for _, want := range []string{
		"ql_calloc(1, sizeof(ql_task), 0)",
		"ql_alloc(la + lb + 1, 0)",
		"ql_free(t->b)",
	} {
		if !strings.Contains(string(src), want) {
			t.Errorf("qthreads.c.txt does not route %q through the counted-slab API", want)
		}
	}
}
