package benchgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestGeneratedPerformanceClaimsIsCurrent is the standing drift gate for the claims report.
func TestGeneratedPerformanceClaimsIsCurrent(t *testing.T) {
	if err := RunSuite(repoRoot(t), ClaimsSuite, true); err != nil {
		t.Fatal(err)
	}
}

// performanceClaimsBlock extracts the claims block from one file.
func performanceClaimsBlock(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := normalize(string(b))
	start := strings.Index(text, ClaimsBlockStart)
	end := strings.Index(text, ClaimsBlockEnd)
	if start < 0 || end < 0 || end < start {
		t.Fatalf("%s has no complete performance-claims block", path)
	}
	if strings.Count(text, ClaimsBlockStart) != 1 || strings.Count(text, ClaimsBlockEnd) != 1 {
		t.Fatalf("%s has more than one performance-claims block", path)
	}
	return text[start : end+len(ClaimsBlockEnd)]
}

// TestPerformanceClaimsMatchMeasurement fails when the README's copy differs from the generated one.
func TestPerformanceClaimsMatchMeasurement(t *testing.T) {
	root := repoRoot(t)
	doc := performanceClaimsBlock(t, filepath.Join(root, ClaimsOut))
	got := performanceClaimsBlock(t, filepath.Join(root, "README.md"))
	if got != doc {
		t.Errorf("README.md: the claims block differs from %s; run `go generate ./internal/benchgen` and copy the generated block", ClaimsOut)
	}
}

// TestPerformanceClaimsWithdrawManualNumbers fails when a withdrawn figure returns to the README.
func TestPerformanceClaimsWithdrawManualNumbers(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, stale := range []string{
		"| fib(30) | **3 ms**",
		"| fib(35) | **21 ms**",
		"P99 (fib20×1000)",
		"223 ms",
		"16× faster rebuilds",
		"22 ns/call",
		"1517 ms",
		"6.6×",
		"fill4K 2.5 ms",
		"8 ms to generate",
		"284 ms",
		"73.95",
		"51.09",
		"parity with C",
	} {
		if strings.Contains(text, stale) {
			t.Errorf("README.md still contains the withdrawn claim %q", stale)
		}
	}
}

// TestClaimsRecordRejectsProbeShapes checks the consumer half of the canonical-shape guard.
func TestClaimsRecordRejectsProbeShapes(t *testing.T) {
	root := repoRoot(t)
	rec, err := Load(root, filepath.Join(root, ClaimsTSV))
	if err != nil {
		t.Fatal(err)
	}
	if err := rec.ValidateClaims(root); err != nil {
		t.Fatalf("the checked-in claims record was rejected: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*Record)
		want   string
	}{
		{"probe rounds", func(rec *Record) {
			for i := range rec.Rows {
				if rec.Rows[i].Key == "fib30_qk" {
					rec.Rows[i].Rounds = 2
				}
			}
		}, "canonical shape"},
		{"probe runs", func(rec *Record) {
			rec.Meta["runs"] = "3"
			for i := range rec.Rows {
				rec.Rows[i].Runs = 3
			}
		}, "canonical shape is 5"},
		{"checksum drift in a family", func(rec *Record) {
			for i := range rec.Rows {
				if rec.Rows[i].Key == "fib35_c" {
					rec.Rows[i].Checksum = "1"
				}
			}
		}, "does not match"},
		{"missing row", func(rec *Record) {
			var rows []Row
			for _, row := range rec.Rows {
				if row.Key != "fib30_erl" {
					rows = append(rows, row)
				}
			}
			rec.Rows = rows
			rec.Meta["skipped"] = "none"
		}, "missing \"fib30_erl\""},
		{"missing load", func(rec *Record) { delete(rec.Meta, "load") }, "missing \"load\""},
	}
	for _, c := range cases {
		probe, err := Load(root, filepath.Join(root, ClaimsTSV))
		if err != nil {
			t.Fatal(err)
		}
		c.mutate(probe)
		err = probe.ValidateClaims(root)
		if err == nil {
			t.Errorf("%s: a non-canonical claims record was accepted", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: the rejection does not say %q: %v", c.name, c.want, err)
		}
	}
}

// TestClaimsRecordMatchesShapeTable checks that every required row has a shape and a checksum.
func TestClaimsRecordMatchesShapeTable(t *testing.T) {
	root := repoRoot(t)
	rec, err := Load(root, filepath.Join(root, ClaimsTSV))
	if err != nil {
		t.Fatal(err)
	}
	shapes, err := loadClaimShapes(root)
	if err != nil {
		t.Fatal(err)
	}
	for family, languages := range claimRequired {
		shape, ok := shapes[family]
		if !ok {
			t.Errorf("family %q has no shape row", family)
			continue
		}
		for _, language := range languages {
			key := family + "_" + language
			found := false
			for _, row := range rec.Rows {
				if row.Key != key {
					continue
				}
				found = true
				if row.Rounds != shape.rounds || row.Runs != shape.runs {
					t.Errorf("%s: got %d rounds and %d runs, want %d and %d", key, row.Rounds, row.Runs, shape.rounds, shape.runs)
				}
			}
			if !found && !strings.Contains(strings.ToLower(rec.Meta["skipped"]), claimSkipName(language)) {
				t.Errorf("%s is missing and skipped does not name %s", key, claimSkipName(language))
			}
		}
	}
}

// claimsScript returns the claims harness path, skipping on platforms where bash cannot run it.
func claimsScript(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the claims harness is a bash script")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed")
	}
	return filepath.Join(repoRoot(t), "bench", "claims", "run.sh")
}

// TestClaimsHarnessRefusesProbesAndAcceptsCanonical is the producer half of the shape guard.
func TestClaimsHarnessRefusesProbesAndAcceptsCanonical(t *testing.T) {
	script := claimsScript(t)
	record := filepath.Join(repoRoot(t), ClaimsTSV)
	before := fileHash(t, record)
	for _, probe := range [][]string{
		{"--record", "--dry-run", "--quick"},
		{"--record", "--dry-run", "--runs", "3"},
	} {
		out, code := runTide(t, script, probe...)
		if code == 0 {
			t.Errorf("%v was accepted as a record: %s", probe, out)
		}
		if !strings.Contains(out, "canonical") || !strings.Contains(out, "claims.tsv") {
			t.Errorf("%v was refused without naming the canonical contract: %s", probe, out)
		}
	}
	if got := fileHash(t, record); got != before {
		t.Errorf("a refused probe changed %s", ClaimsTSV)
	}
	out, code := runTide(t, script, "--record", "--dry-run")
	if code != 0 {
		t.Fatalf("the canonical dry run was refused (exit %d): %s", code, out)
	}
	for _, want := range []string{"canonical", "no file written"} {
		if !strings.Contains(out, want) {
			t.Errorf("the canonical dry run does not report %q: %s", want, out)
		}
	}
	if got := fileHash(t, record); got != before {
		t.Errorf("the canonical dry run changed %s", ClaimsTSV)
	}
}
