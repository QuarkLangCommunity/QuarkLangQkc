package cgen

import (
	"strings"
	"testing"
)

// TestMemoryPrimitivesAreRefusedExceptLoweredRows pins the compiler's half of §10: the rows the
// lowering layer does not implement are refused explicitly (never dropped), and the rows stage 2 has
// wired are lowered to their runtime entry points.
func TestMemoryPrimitivesAreRefusedExceptLoweredRows(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"delete int", "fn main(IOStream io) { int x = 1; delete x; }\n", "delete"},
		{"Fragmentation", "fn main(IOStream io) { io.println(memory.Fragmentation()); }\n", "Fragmentation"},
		{"mode", "fn main(IOStream io) { memory.mode(1); }\n", "mode"},
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

// TestMemorySetBlockLowersToRuntimeEntryPoint is the positive half of row 1: the accepted range is the
// interpreter's (1 … 2^20), so lowering emits the runtime call and a branch to a positioned runtime
// error carrying the interpreter's message for a value outside it. The call is opaque to the optimizer,
// so the block granularity the interpreter accepts is the one the compiled path honours.
func TestMemorySetBlockLowersToRuntimeEntryPoint(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    memory.setBlock(1024);\n"+
		"    io.println(1);\n"+
		"}\n")
	if !strings.Contains(ir, "call i32 @ql_mem_set_block(i32 1024)") {
		t.Fatalf("memory.setBlock does not call the runtime entry point:\n%s", ir)
	}
	if !strings.Contains(ir, "call void @ql_panic(") {
		t.Fatalf("memory.setBlock has no positioned error for an out-of-range n:\n%s", ir)
	}
	if !strings.Contains(ir, "setBlock(n) requires a positive int block size") {
		t.Fatalf("the range error does not carry the interpreter's message:\n%s", ir)
	}
}

// TestMemorySetBlockErrorJoinsTry pins the control-flow half: the interpreter raises a RunError, so a
// try/catch around the call catches it, and the compiled path must branch to the catch block rather
// than terminating the process.
func TestMemorySetBlockErrorJoinsTry(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    try { memory.setBlock(0); } catch (void e) { io.println(7); }\n"+
		"    io.println(8);\n"+
		"}\n")
	if strings.Contains(ir, "call void @ql_panic(") {
		t.Fatalf("a setBlock call inside try emits an uncatchable panic:\n%s", ir)
	}
	if !strings.Contains(ir, "br i1 %") {
		t.Fatalf("the range check does not branch (no catch edge):\n%s", ir)
	}
}

// TestMemoryCompactLowersAndReturns pins D6 from the interpreter's behaviour: eval.go calls
// MemoryManager.Compact() and returns normally (the comment "never returns at all" is not what the
// code does), so the compiled call returns too. A terminating lowering would be a language-visible
// control-flow primitive the standard does not describe.
func TestMemoryCompactLowersAndReturns(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    memory.compact();\n"+
		"    io.println(1);\n"+
		"}\n")
	if !strings.Contains(ir, "call void @ql_mem_compact()") {
		t.Fatalf("memory.compact does not call the runtime entry point:\n%s", ir)
	}
	// The call is followed by the println in the same function body: control continues, as in the
	// interpreter (a terminating lowering would have no reachable statement after it).
	call := strings.Index(ir, "call void @ql_mem_compact()")
	print := strings.Index(ir[call:], "@printf")
	if call < 0 || print < 0 {
		t.Fatalf("memory.compact does not return control to the next statement:\n%s", ir)
	}
}

// TestMemoryClearLowersToTheDerivedWipe pins the call: §10 defines clear as release plus zero-fill, so
// the compiled path emits the runtime wipe. The observable half (free payloads stay wiped and live
// payloads do not) is in compiler/testdata/memory/qthreads_mem_test.c, because the compiler has no
// language surface that can read a released region after a delete.
func TestMemoryClearLowersToTheDerivedWipe(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    memory.clear();\n"+
		"    io.println(1);\n"+
		"}\n")
	if !strings.Contains(ir, "call void @ql_mem_clear()") {
		t.Fatalf("memory.clear does not call the runtime wipe:\n%s", ir)
	}
}
