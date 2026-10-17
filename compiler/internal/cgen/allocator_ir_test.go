package cgen

import (
	"strings"
	"testing"
)

// TestRoutedAllocationsUseRuntimeAllocator is the IR-shape half of the allocator gate (design T2).
// It covers every allocation shape cgen can emit at once: a List literal (header + buffer), append
// growth (realloc), a struct (zeroed block), an interface{} box (4/8-byte cell), int.toString (the
// fixed helper body) and delete (per-object release). On Windows the compiler keeps the libc path
// because the runtime is not linked there; the same test then asserts that fallback instead.
func TestRoutedAllocationsUseRuntimeAllocator(t *testing.T) {
	ir := transpile(t, `type struct { int a; } S;
fn main(IOStream io) {
    S s;
    s.a = 1;
    List<int> l = [1, 2, 3];
    l.append(4);
    interface{} boxed = 42;
    int seven = 7;
    io.println(s.a, l.toString(), boxed, seven.toString());
    delete l;
}
`)
	routed := []string{
		"call i8* @ql_alloc(",
		"call i8* @ql_calloc(",
		"call i8* @ql_realloc(",
		"call void @ql_free(",
	}
	host := []string{
		"call i8* @malloc(",
		"call i8* @calloc(",
		"call i8* @realloc(",
		"call void @free(",
	}
	if hostAllocatorFallback {
		for _, want := range host {
			if !strings.Contains(ir, want) {
				t.Fatalf("Windows fallback IR missing %q:\n%s", want, ir)
			}
		}
		for _, unwanted := range routed {
			if strings.Contains(ir, unwanted) {
				t.Fatalf("Windows fallback IR still routes %q:\n%s", unwanted, ir)
			}
		}
		return
	}
	for _, want := range routed {
		if !strings.Contains(ir, want) {
			t.Fatalf("routed IR missing %q:\n%s", want, ir)
		}
	}
	for _, unwanted := range host {
		if strings.Contains(ir, unwanted) {
			t.Fatalf("routed IR still has a raw allocation call %q:\n%s", unwanted, ir)
		}
	}
}
