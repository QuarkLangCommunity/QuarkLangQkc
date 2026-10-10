//go:build unix

package main

import (
	"testing"
	"time"
)

// TestRunBoundedReportsExitAndOutput checks the bounded executor on a command that finishes by itself.
// The exit status and the two streams are asserted everywhere; the peak figure is asserted where the
// platform can measure it, and the -1 "unavailable" sentinel where it cannot.
func TestRunBoundedReportsExitAndOutput(t *testing.T) {
	res := runBounded([]string{"/bin/sh", "-c", "echo out; echo err 1>&2; exit 3"}, childEnv(nil), 5*time.Second, 512*kibPerMiB)
	if res.Exit != 3 {
		t.Errorf("exit = %d, want 3", res.Exit)
	}
	if res.Stdout != "out\n" || res.Stderr != "err\n" {
		t.Errorf("stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}
	if res.TimedOut || res.MemCapped || res.Panicked {
		t.Errorf("a clean run reported a bound: %+v", res)
	}
	if rssProbeSupported {
		if res.PeakRSSKB <= 0 {
			t.Errorf("no peak RSS was sampled: %+v", res)
		}
	} else if res.PeakRSSKB != -1 {
		t.Errorf("without an RSS probe the peak must stay the -1 sentinel, got %d", res.PeakRSSKB)
	}
}

// TestRunBoundedKillsOnTimeout checks that the wall-clock bound is enforced on the whole process group,
// which is what stops the compiler's clang children from outliving a killed case.
func TestRunBoundedKillsOnTimeout(t *testing.T) {
	start := time.Now()
	res := runBounded([]string{"/bin/sh", "-c", "sleep 30"}, childEnv(nil), 300*time.Millisecond, 512*kibPerMiB)
	if !res.TimedOut {
		t.Fatalf("the bound did not fire: %+v", res)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("killing took %s: the group survived the kill", elapsed)
	}
}

// TestRunBoundedKillsOnMemoryCeiling checks the memory bound (the ceiling is in kibibytes, as is the
// kernel's resident-set figure it is compared against) with a process whose resident set grows
// steadily, so the supervisor has to notice the growth rather than a single huge allocation. Where the
// platform has no RSS probe the ceiling is unmeasurable by construction, and what is asserted instead
// is the documented degradation: the run is still stopped by the wall clock, still reports an exit
// status, and still marks the figure it could not take as unavailable.
func TestRunBoundedKillsOnMemoryCeiling(t *testing.T) {
	script := `x=; while [ ${#x} -lt 200000000 ]; do x="$x$(printf %0100000d 0)"; sleep 0.01; done; sleep 30`
	timeout := 20 * time.Second
	if !rssProbeSupported {
		timeout = 2 * time.Second
	}
	res := runBounded([]string{"/bin/sh", "-c", script}, childEnv(nil), timeout, 8*kibPerMiB)
	if !rssProbeSupported {
		if !res.TimedOut {
			t.Fatalf("the wall-clock bound did not stop a run the ceiling cannot bound: %+v", res)
		}
		if res.MemCapped {
			t.Errorf("a platform without an RSS probe cannot have measured the ceiling: %+v", res)
		}
		if res.Exit == 0 {
			t.Errorf("the killed run must still report an exit status, got 0: %+v", res)
		}
		if res.PeakRSSKB != -1 {
			t.Errorf("without an RSS probe the peak must stay the -1 sentinel, got %d", res.PeakRSSKB)
		}
		return
	}
	if !res.MemCapped {
		t.Fatalf("the memory ceiling did not fire: %+v", res)
	}
}
