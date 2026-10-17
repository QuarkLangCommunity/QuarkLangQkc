package main

// The allocator's assertion and stress gate. compiler/qthreads.c.txt is the translation unit qkc
// embeds and links into every compiled program; this test compiles that exact source (through
// runtimeSrc(), which prepends the generated message catalog) with the checked-in C harness and
// runs the T4 shape: delete/reallocate under load, a tide, the large path, compact and the full
// ql_mem_check() invariant walk.
//
// The debug build defines QL_MEM_DEBUG=1, so the allocator's hot-path assertions are active. The
// release build defines it 0; ql_mem_check() still runs, which is what keeps the two builds from
// drifting. Windows is skipped: compiler/main.go's runtimeSrc returns no source there, so the
// allocator is not linked into Windows programs at all (docs/compiled-memory-stage1.md §5).

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCompiledAllocatorAssertionsUnderStress(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the compiled runtime is not linked on Windows; stage 1's allocator is absent there")
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang not available")
	}
	rt := runtimeSrc()
	if rt == "" {
		t.Skip("runtimeSource returned no translation unit")
	}
	defer os.Remove(rt)

	harness, err := os.ReadFile(filepath.Join("testdata", "memory", "qthreads_mem_test.c"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "qthreads_mem_test.c")
	if err := os.WriteFile(harnessPath, harness, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ name, debug string }{
		{"debug", "1"},
		{"release", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := filepath.Join(dir, "qthreads_mem_test-"+tc.name)
			args := []string{
				"-std=c11", "-O1", "-g", "-pthread",
				"-DQL_MEM_DEBUG=" + tc.debug,
				rt, harnessPath, "-o", bin,
			}
			buildCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			if out, err := exec.CommandContext(buildCtx, clang, args...).CombinedOutput(); err != nil {
				t.Fatalf("clang: %v\n%s", err, out)
			}
			runCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			out, err := exec.CommandContext(runCtx, bin).CombinedOutput()
			if err != nil {
				t.Fatalf("allocator stress (%s) failed: %v\n%s", tc.name, err, out)
			}
			t.Logf("%s: %s", tc.name, out)
		})
	}
}
