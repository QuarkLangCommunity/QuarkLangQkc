package benchgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// TestPerformanceClaimsMatchMeasurement compares the single claims document with the record and fails
// when any other document embeds a second copy of the block.
func TestPerformanceClaimsMatchMeasurement(t *testing.T) {
	root := repoRoot(t)
	rec, err := Load(root, filepath.Join(root, ClaimsTSV))
	if err != nil {
		t.Fatal(err)
	}
	want, err := ClaimsBlock(rec)
	if err != nil {
		t.Fatal(err)
	}
	if got := performanceClaimsBlock(t, filepath.Join(root, ClaimsOut)); got != strings.TrimRight(want, "\n") {
		t.Errorf("%s: the claims block differs from the record; run `go generate ./internal/benchgen`", ClaimsOut)
	}
	for _, path := range []string{"README.md", "spec/STANDARD.md"} {
		text := readText(t, filepath.Join(root, path))
		if strings.Contains(text, ClaimsBlockStart) || strings.Contains(text, ClaimsBlockEnd) {
			t.Errorf("%s embeds a second claims block; link %s instead", path, ClaimsOut)
		}
		if strings.Contains(text, "| Workload | Language | Rounds | Runs | Median |") {
			t.Errorf("%s copies the claims measurement table; link %s instead", path, ClaimsOut)
		}
		if !strings.Contains(text, ClaimsOut) {
			t.Errorf("%s does not link the single claims document %s", path, ClaimsOut)
		}
	}
}

// TestPerformanceClaimsWithdrawManualNumbers fails when a withdrawn figure returns to live text, and
// fails when the provenance section loses an earlier figure instead of keeping it as history.
func TestPerformanceClaimsWithdrawManualNumbers(t *testing.T) {
	root := repoRoot(t)
	for _, path := range []string{"README.md", "spec/STANDARD.md", "docs/ARCHITECTURE.md", "compiler/README.md", "CONTRIBUTING.md"} {
		text := readText(t, filepath.Join(root, path))
		if path == "spec/STANDARD.md" {
			// The revision log quotes superseded figures as history; the live text above it must not.
			text = beforeRevisionLog(t, text)
		}
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
				t.Errorf("%s still contains the withdrawn claim %q", path, stale)
			}
		}
	}
	provenance := readText(t, filepath.Join(root, ClaimsOut))
	for _, kept := range []string{"`fib(35)` in 21 ms", "P99 (fib20×1000)", "223 ms", "73.95", "1517 ms"} {
		if !strings.Contains(provenance, kept) {
			t.Errorf("%s no longer records the earlier figure %q; provenance is content, not duplication", ClaimsOut, kept)
		}
	}
}

// beforeRevisionLog returns the live standard text, excluding the historical revision log.
func beforeRevisionLog(t *testing.T, text string) string {
	t.Helper()
	if at := strings.Index(text, "## 12. Change policy and revision log"); at >= 0 {
		return text[:at]
	}
	t.Fatal("spec/STANDARD.md has no revision log section")
	return ""
}

// claimsProbe is one mutation of a valid claims record and the text its rejection must contain.
type claimsProbe struct {
	name   string
	mutate func(*Record)
	want   string
}

// claimsProbeCases lists the non-canonical shapes the validator must refuse.
func claimsProbeCases() []claimsProbe {
	return []claimsProbe{
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
	for _, c := range claimsProbeCases() {
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

// TestTimingNumbersStayInAuthoritativeReports fails when a timing value appears outside the single
// report that owns it; history, the generator inputs and the stress measurement records are the only
// allowed exceptions, each named here so a new copy fails instead of joining an implicit allow-list.
func TestTimingNumbersStayInAuthoritativeReports(t *testing.T) {
	root := repoRoot(t)
	allowed := map[string]bool{
		"docs/tide-benchmark.md":        true,
		"docs/performance-claims.md":    true,
		"docs/compiled-memory.md":       true, // historical figures under the banner
		"CHANGELOG.md":                  true, // release history
		"stress/NOTES.md":               true,
		"stress/REPORT.md":              true,
		"spec/bench/claims-template.md": true, // generator input of the claims report
		"spec/bench/tide-template.md":   true, // generator input of the tide report
	}
	timing := regexp.MustCompile(`[0-9]+(\.[0-9]+)? ?(ms|µs|ns)\b`)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if allowed[rel] {
			return nil
		}
		text := readText(t, path)
		if rel == "spec/STANDARD.md" {
			text = beforeRevisionLog(t, text)
		}
		for n, line := range strings.Split(text, "\n") {
			if timing.MatchString(line) {
				t.Errorf("%s line %d has a timing value outside the single reports: %s", rel, n+1, line)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestSingleCopyLayout enforces the consolidation: the old files are gone, every surviving topic has
// exactly one document, and the two reports carry the only embedded blocks.
func TestSingleCopyLayout(t *testing.T) {
	root := repoRoot(t)
	for _, gone := range []string{
		"docs/compiled-memory-design.md",
		"docs/compiled-memory-stage1.md",
		"docs/compiled-memory-stage2.md",
		"docs/performance-claims-inventory.md",
	} {
		if _, err := os.Stat(filepath.Join(root, gone)); !os.IsNotExist(err) {
			t.Errorf("%s still exists; its content belongs to the single document that owns the topic", gone)
		}
	}
	for _, path := range []string{"README.md", "spec/STANDARD.md"} {
		text := readText(t, filepath.Join(root, path))
		for _, doc := range []string{DefaultOut, ClaimsOut} {
			if !strings.Contains(text, doc) {
				t.Errorf("%s does not link %s", path, doc)
			}
		}
		for _, marker := range []string{BlockStart, ClaimsBlockStart} {
			if strings.Contains(text, marker) {
				t.Errorf("%s embeds the copy marker %q", path, marker)
			}
		}
	}
	merged := readText(t, filepath.Join(root, "docs/compiled-memory.md"))
	for _, want := range []string{"## Part I", "## Part II", "## Part III", "**Historical measurements.**"} {
		if !strings.Contains(merged, want) {
			t.Errorf("docs/compiled-memory.md lacks %q", want)
		}
	}
	if strings.Count(merged, "**Historical measurements.**") != 1 {
		t.Errorf("docs/compiled-memory.md must carry one historical-measurement banner")
	}
	if !strings.Contains(readText(t, filepath.Join(root, ClaimsOut)), "## Provenance: every earlier performance claim") {
		t.Errorf("%s does not own the claim provenance section", ClaimsOut)
	}
}
