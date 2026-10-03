package lang

// Differential test for the bytecode VM (vm.go).
//
// The VM is an optimization, never a second semantics: every program here must produce exactly the
// same stdout and the same error (or success) with the VM enabled and with it disabled. QUARK_NO_VM=1
// is the switch the test flips, so this is the guard that keeps the two engines from drifting — the
// same idea as compiler/testdata/compare.sh, applied inside the interpreter.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quarklang/internal/i18n"
)

// vmDiffPrograms are inline programs covering the shapes the VM compiles plus the shapes that must
// make it deoptimize or refuse to compile at all.
var vmDiffPrograms = []struct {
	name string
	src  string
}{
	{"recursion", `program main;
fn fib(int n) int { if (n <= 1) { return n; } return fib(n - 1) + fib(n - 2); }
fn main(IOStream io) { io.println(fib(18)); }`},
	{"while_loop_slots", `program main;
fn main(IOStream io) {
    int total = 0;
    int i = 0;
    while (i < 1000) { total = total + i; i = i + 1; }
    io.println(total);
}`},
	{"for_c_loop", `program main;
fn main(IOStream io) {
    int s = 0;
    for (int i = 0; i < 100; i = i + 1) { s = s + i * 2; }
    io.println(s);
}`},
	{"byref_write_through", `program main;
fn bump(int a) int { a = a + 1; return a; }
fn main(IOStream io) {
    int x = 5;
    io.println(bump(x));
    io.println(x);
}`},
	{"mixed_int_float_promotion", `program main;
fn add(int a, int b) int { return a + b; }
fn add(float a, float b) float { return a + b; }
fn main(IOStream io) { io.println(add(1, 2)); io.println(add(1, 2.5)); }`},
	{"bool_conditions_and_short_circuit", `program main;
fn main(IOStream io) {
    int a = 3;
    bool t = a > 1 && a < 10;
    bool f = a > 100 || a == 3;
    if (t && f) { io.println("both"); } else { io.println("not both"); }
}`},
	{"division_errors", `program main;
fn div(int a, int b) int { return a / b; }
fn main(IOStream io) { io.println(div(7, 2)); io.println(div(1, 0)); }`},
	{"log_ends_function", `program main;
fn f(int n) void { log "value=" + n.toString(); }
fn main(IOStream io) { f(4); io.println("after"); }`},
	{"string_ops_fall_back", `program main;
fn main(IOStream io) {
    String s = "ab";
    io.println(s + "cd");
    io.println(s.size());
}`},
	{"nested_block_declaration_falls_back", `program main;
fn main(IOStream io) {
    int x = 1;
    if (x == 1) { int y = 2; io.println(x + y); }
}`},
	{"overload_by_arity", `program main;
fn zero() int { return 0; }
fn zero(String s) int { return 1; }
fn main(IOStream io) { io.println(zero(), zero("s")); }`},
	{"taskm_spawn_falls_back", `program main;
fn work(int n) int { int s = 0; for (int i = 0; i < n; i = i + 1) { s = s + i; } return s; }
fn main(IOStream io) {
    thread t = taskm.spawn();
    t.merge(work, 1000);
    taskm.block(t.pid());
    io.println("done");
}`},
}

// runWithVM runs src with the VM enabled or disabled and returns stdout plus a one-line outcome.
func runWithVM(t *testing.T, src string, vmEnabled bool) (string, string) {
	t.Helper()
	if vmEnabled {
		os.Unsetenv("QUARK_NO_VM")
	} else {
		os.Setenv("QUARK_NO_VM", "1")
	}
	defer os.Unsetenv("QUARK_NO_VM")
	out, err := runSrc(t, src)
	if err != nil {
		return out, "error: " + vmFirstLine(err.Error())
	}
	return out, "ok"
}

func vmFirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TestVMEqualsTreeWalker is the core guarantee: identical output and identical error behaviour.
func TestVMEqualsTreeWalker(t *testing.T) {
	for _, tc := range vmDiffPrograms {
		t.Run(tc.name, func(t *testing.T) {
			withLocalizer(t, i18n.New(nil, i18n.EN))
			vmOut, vmRes := runWithVM(t, tc.src, true)
			twOut, twRes := runWithVM(t, tc.src, false)
			if vmRes != twRes {
				t.Fatalf("outcome differs:\n  vm   = %s\n  walk = %s", vmRes, twRes)
			}
			if vmOut != twOut {
				t.Fatalf("stdout differs:\n  vm   = %q\n  walk = %q", vmOut, twOut)
			}
		})
	}
}

// TestVMCompilesHotShapes checks that the VM is actually used (a silent permanent fallback would make
// the optimization pointless while every semantic test above still passed).
func TestVMCompilesHotShapes(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	cases := map[string]bool{
		"recursion":            true,
		"while_loop_slots":     true,
		"for_c_loop":           true,
		"string_ops_fall_back": false,
	}
	for _, tc := range vmDiffPrograms {
		want, listed := cases[tc.name]
		if !listed {
			continue
		}
		before := vmExecutions.Load()
		os.Unsetenv("QUARK_NO_VM")
		if _, err := runSrc(t, tc.src); err != nil {
			t.Fatalf("%s: run failed: %v", tc.name, err)
		}
		used := vmExecutions.Load() > before
		if used != want {
			t.Errorf("%s: bytecode used = %v, want %v", tc.name, used, want)
		}
	}
}

// TestVMEqualsTreeWalkerOnCorpus runs the real programs in the repository through both engines.
func TestVMEqualsTreeWalkerOnCorpus(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	roots := []string{"../../examples", "../../compiler/testdata/cases"}
	found := 0
	for _, root := range roots {
		matches, err := filepath.Glob(filepath.Join(root, "*.qk"))
		if err != nil {
			t.Fatal(err)
		}
		more, _ := filepath.Glob(filepath.Join(root, "*.kq"))
		for _, path := range append(matches, more...) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			// Programs that need stdin, FFI libraries or network access are covered by their own
			// tests; here the goal is the interpreter's two engines agreeing.
			if strings.Contains(string(src), "sys::") || strings.Contains(string(src), "setIn") {
				continue
			}
			name := filepath.Base(path)
			t.Run(name, func(t *testing.T) {
				vmOut, vmRes := runWithVM(t, string(src), true)
				twOut, twRes := runWithVM(t, string(src), false)
				if vmRes != twRes || vmOut != twOut {
					t.Fatalf("engines differ for %s:\n  vm   = %s / %q\n  walk = %s / %q", name, vmRes, vmOut, twRes, twOut)
				}
			})
			found++
		}
	}
	if found == 0 {
		t.Fatal("no corpus programs found — the differential test would be vacuous")
	}
	t.Logf("compared %d corpus programs between the VM and the tree-walker", found)
}
