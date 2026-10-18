package benchgen

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// loadRecord loads the checked-in record so a test can mutate one part of it.
func loadRecord(t *testing.T) *Record {
	t.Helper()
	rec, err := Load(repoRoot(t), filepath.Join(repoRoot(t), DefaultTSV))
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// TestCanonicalRecordIsAccepted checks that the checked-in record declares the canonical shape and passes.
func TestCanonicalRecordIsAccepted(t *testing.T) {
	rec := loadRecord(t)
	if rec.Meta["rounds"] != "100000000" || rec.Meta["runs"] != "5" {
		t.Fatalf("the record must declare # rounds 100000000 and # runs 5, got rounds=%q runs=%q", rec.Meta["rounds"], rec.Meta["runs"])
	}
	if err := rec.Validate(); err != nil {
		t.Fatalf("the canonical record was rejected: %v", err)
	}
}

// TestRecordRejectsIncompleteOrNonCanonicalParameters checks that the metadata half names the contract.
func TestRecordRejectsIncompleteOrNonCanonicalParameters(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Record)
	}{
		{"missing rounds", func(rec *Record) { delete(rec.Meta, "rounds") }},
		{"missing runs", func(rec *Record) { delete(rec.Meta, "runs") }},
		{"probe rounds", func(rec *Record) { rec.Meta["rounds"] = "1000000" }},
		{"probe runs", func(rec *Record) { rec.Meta["runs"] = "3" }},
		{"non-integer rounds", func(rec *Record) { rec.Meta["rounds"] = "many" }},
	}
	for _, c := range cases {
		rec := loadRecord(t)
		c.mutate(rec)
		err := rec.Validate()
		if err == nil {
			t.Errorf("%s: a non-canonical record was accepted", c.name)
			continue
		}
		if msg := err.Error(); !strings.Contains(msg, "canonical") || !strings.Contains(msg, "100000000") {
			t.Errorf("%s: the rejection does not state the canonical contract: %v", c.name, err)
		}
	}
}

// TestRecordRejectsProbeTimingRows checks that canonical metadata cannot disguise a probe row.
func TestRecordRejectsProbeTimingRows(t *testing.T) {
	for _, probe := range []struct{ rounds, runs int64 }{{1_000_000, 3}, {1_000_000, 5}, {100_000_000, 3}} {
		rec := loadRecord(t)
		changed := false
		for i := range rec.Rows {
			if rec.Rows[i].Kind == "timing" {
				rec.Rows[i].Rounds, rec.Rows[i].Runs = probe.rounds, probe.runs
				changed = true
			}
		}
		if !changed {
			t.Fatal("the record has no timing row to mutate")
		}
		err := rec.Validate()
		if err == nil {
			t.Errorf("a timing row with %d rounds and %d runs was accepted", probe.rounds, probe.runs)
			continue
		}
		if !strings.Contains(err.Error(), "canonical timing row") {
			t.Errorf("the row rejection does not name the canonical shape: %v", err)
		}
	}
}

// tideScript returns the harness path, skipping on platforms where bash cannot run it.
func tideScript(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the tide harness is a bash script")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed")
	}
	return filepath.Join(repoRoot(t), "bench", "tide", "run.sh")
}

// runTide runs the harness in the checkout and returns its combined output and exit code.
func runTide(t *testing.T, script string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("run %s: %v", script, err)
	}
	return string(out), exitErr.ExitCode()
}

// fileHash returns the SHA-256 of a file, which is how the tests prove it was not rewritten.
func fileHash(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

// TestRecordHarnessRefusesProbesAndAcceptsCanonical is the both-directions proof for the producer guard.
func TestRecordHarnessRefusesProbesAndAcceptsCanonical(t *testing.T) {
	script := tideScript(t)
	record := filepath.Join(repoRoot(t), DefaultTSV)
	before := fileHash(t, record)
	probes := [][]string{
		{"--record", "--dry-run", "--n", "1000000", "--runs", "3"},
		{"--record", "--dry-run", "--quick"},
		{"--record", "--dry-run", "--n", "100000000", "--runs", "3"},
	}
	for _, probe := range probes {
		out, code := runTide(t, script, probe...)
		if code == 0 {
			t.Errorf("%v was accepted as a record: %s", probe, out)
		}
		if !strings.Contains(out, "canonical") || !strings.Contains(out, "spec/bench/tide.tsv") {
			t.Errorf("%v was refused without naming the canonical contract: %s", probe, out)
		}
	}
	if got := fileHash(t, record); got != before {
		t.Errorf("a refused probe changed %s", DefaultTSV)
	}
	out, code := runTide(t, script, "--record", "--dry-run")
	if code != 0 {
		t.Fatalf("the canonical dry run was refused (exit %d): %s", code, out)
	}
	for _, want := range []string{"100000000", "5", "no file written"} {
		if !strings.Contains(out, want) {
			t.Errorf("the canonical dry run does not report %q: %s", want, out)
		}
	}
	if got := fileHash(t, record); got != before {
		t.Errorf("the canonical dry run changed %s", DefaultTSV)
	}
}
