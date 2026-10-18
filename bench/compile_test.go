package main

// The checked-in C controls are measurement tools, not language surface, and a tool that no longer
// builds is a silent gate: stage 2 changed the string ABI and the attribution control kept compiling
// its own stale declarations until someone next tried to measure. This test builds every control with
// warnings fatal and never runs a measurement loop, so a stale prototype is an ordinary red test.

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCControlsCompile builds every checked-in C benchmark and control without running it.
func TestCControlsCompile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the tide controls link the POSIX compiled runtime, which Windows does not carry")
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not available")
	}
	root := repoRoot(t)
	dir := t.TempDir()
	t.Run("tide controls", func(t *testing.T) { compileTideControls(t, root, dir) })
	t.Run("claims controls", func(t *testing.T) { compileClaimsControls(t, root, dir) })
	t.Run("allocator benchmark", func(t *testing.T) { compileAllocBench(t, root, dir) })
	t.Run("C control", func(t *testing.T) {
		runClang(t, root, "-O2", "-Wall", "-Wextra", "-Werror",
			"bench/c/fib.c", "-o", filepath.Join(dir, "fib"))
	})
}

// repoRoot returns the checkout root from the package directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// compileTideControls runs the harness's compile-only mode, which builds the sink, the C controls and
// the attribution control with warnings fatal.
func compileTideControls(t *testing.T, root, dir string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	cache := filepath.Join(dir, "cache")
	if err := os.Mkdir(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", filepath.Join(root, "bench/tide/run.sh"), "--compile-only")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "XDG_CACHE_HOME="+cache)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tide compile-only failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "compiled the checked-in C controls") {
		t.Fatalf("tide compile-only did not report a successful build:\n%s", out)
	}
}

// compileClaimsControls runs the claims harness's compile-only mode, which builds the fib and latency
// C controls with warnings fatal.
func compileClaimsControls(t *testing.T, root, dir string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	cache := filepath.Join(dir, "claims-cache")
	if err := os.Mkdir(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", filepath.Join(root, "bench/claims/run.sh"), "--compile-only")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "XDG_CACHE_HOME="+cache)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("claims compile-only failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "compiled the checked-in C controls") {
		t.Fatalf("claims compile-only did not report a successful build:\n%s", out)
	}
}

// compileAllocBench builds the allocator benchmark against libc and against the routed runtime.
func compileAllocBench(t *testing.T, root, dir string) {
	t.Helper()
	src := filepath.Join(root, "bench/memory/alloc_bench.c")
	runClang(t, root, "-O2", "-Wall", "-Wextra", "-Werror", "-DUSE_QL_ALLOC=0",
		src, "-o", filepath.Join(dir, "alloc-host"))
	runtimeSrc := filepath.Join(dir, "runtime.c")
	msg, err := os.ReadFile(filepath.Join(root, "compiler/internal/rtcat/msg_c.h"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeBody, err := os.ReadFile(filepath.Join(root, "compiler/qthreads.c.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimeSrc, append(msg, runtimeBody...), 0o644); err != nil {
		t.Fatal(err)
	}
	obj := filepath.Join(dir, "alloc-ql.o")
	runClang(t, root, "-O2", "-Wall", "-Wextra", "-Werror", "-DUSE_QL_ALLOC=1", "-c",
		src, "-o", obj)
	runtimeObj := filepath.Join(dir, "runtime.o")
	runClang(t, root, "-O2", "-pthread", "-c", runtimeSrc, "-o", runtimeObj)
	runClang(t, root, obj, runtimeObj, "-pthread", "-o", filepath.Join(dir, "alloc-ql"))
}

// runClang runs clang in dir and reports the compiler output when it exits non-zero.
func runClang(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("clang", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clang %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
