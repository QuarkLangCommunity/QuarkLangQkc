package benchgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot returns the checkout root from the package directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestGeneratedTideBenchmarkIsCurrent is the standing drift gate: the committed report must be
// exactly what the record and template render, so a record or template edit without regenerating
// fails the ordinary go test run.
func TestGeneratedTideBenchmarkIsCurrent(t *testing.T) {
	if err := Run(Options{Root: repoRoot(t), Check: true}); err != nil {
		t.Fatal(err)
	}
}

// measuredBlock extracts the marked block from one file.
func measuredBlock(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := normalize(string(b))
	start := strings.Index(text, BlockStart)
	end := strings.Index(text, BlockEnd)
	if start < 0 || end < 0 || end < start {
		t.Fatalf("%s has no complete tide-measurements block", path)
	}
	if strings.Count(text, BlockStart) != 1 || strings.Count(text, BlockEnd) != 1 {
		t.Fatalf("%s has more than one tide-measurements block", path)
	}
	return text[start : end+len(BlockEnd)]
}

// TestTideClaimsMatchMeasurement fails when README or §10 shows a number that the generated record
// does not contain, which is the drift this task exists to stop.
func TestTideClaimsMatchMeasurement(t *testing.T) {
	root := repoRoot(t)
	doc := measuredBlock(t, filepath.Join(root, DefaultOut))
	for _, path := range []string{"README.md", "spec/STANDARD.md"} {
		got := measuredBlock(t, filepath.Join(root, path))
		if got != doc {
			t.Errorf("%s: the tide block differs from %s; run `go generate ./internal/benchgen` and copy the generated block", path, DefaultOut)
		}
	}
	for _, path := range []string{"README.md", "spec/STANDARD.md"} {
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		if path == "spec/STANDARD.md" {
			// The revision log quotes the superseded figures as history; only §10 is a live claim.
			text = between(t, text, "## 10. Memory and lifetime", "## 11. Implementation status")
		}
		for _, stale := range []string{"| Tide, 1e8 rounds |", "99.96% reuse", "0.195% fragmentation"} {
			if strings.Contains(text, stale) {
				t.Errorf("%s still contains the stale claim %q", path, stale)
			}
		}
	}
}

// between returns the text between two markers, or fails when either is missing.
func between(t *testing.T, text, start, end string) string {
	t.Helper()
	i := strings.Index(text, start)
	j := strings.Index(text, end)
	if i < 0 || j < 0 || j <= i {
		t.Fatalf("missing section between %q and %q", start, end)
	}
	return text[i:j]
}

// TestValidateRequiresExplicitSkip checks that a missing Rust or Go control is accepted only when the
// record names the missing toolchain, which is the "skip explicitly, never invent a number" rule.
func TestValidateRequiresExplicitSkip(t *testing.T) {
	rec, err := Load(repoRoot(t), filepath.Join(repoRoot(t), DefaultTSV))
	if err != nil {
		t.Fatal(err)
	}
	var rows []Row
	for _, row := range rec.Rows {
		if !strings.HasPrefix(row.Key, "rust_") {
			rows = append(rows, row)
		}
	}
	rec.Rows = rows
	rec.Meta["skipped"] = "rust: rustc not found"
	if err := rec.Validate(); err != nil {
		t.Fatalf("an explicitly skipped Rust control was rejected: %v", err)
	}
	rec.Meta["skipped"] = "none"
	if err := rec.Validate(); err == nil {
		t.Fatal("a missing Rust control with no skip note was accepted")
	}
}

// TestValidateRequiresLoadMetadata checks that a record without the load it was taken under is refused.
func TestValidateRequiresLoadMetadata(t *testing.T) {
	rec, err := Load(repoRoot(t), filepath.Join(repoRoot(t), DefaultTSV))
	if err != nil {
		t.Fatal(err)
	}
	delete(rec.Meta, "load")
	if err := rec.Validate(); err == nil {
		t.Fatal("a record without load metadata was accepted")
	}
}

// TestSummaryBlockReportsLoad checks that the recorded load reaches the block the README and §10 embed.
func TestSummaryBlockReportsLoad(t *testing.T) {
	rec, err := Load(repoRoot(t), filepath.Join(repoRoot(t), DefaultTSV))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Meta["load"] == "" {
		t.Fatal("the record has no load metadata")
	}
	block, err := SummaryBlock(rec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(block, rec.Meta["load"]) {
		t.Fatalf("the generated block does not carry the recorded load %q", rec.Meta["load"])
	}
}

// TestMeasurementRecordCoversControls checks that the record still carries the observable language
// workload, the C control, the dead-code floor and the policy counters, and that the controls did
// the same work.
func TestMeasurementRecordCoversControls(t *testing.T) {
	rec, err := Load(repoRoot(t), filepath.Join(repoRoot(t), DefaultTSV))
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, row := range rec.Rows {
		keys[row.Key] = true
		if row.Kind == "timing" && row.Runs < 3 {
			t.Errorf("%s: %d runs is too few to report variance", row.Key, row.Runs)
		}
		if row.Kind == "timing" && row.Key != "legacy_floor" && row.Checksum == "-" {
			t.Errorf("%s: a measured row has no checksum", row.Key)
		}
	}
	for _, key := range []string{"compiled_observable", "c_observable", "compiled_floor", "legacy_floor", "policy_reuse", "policy_fragmentation"} {
		if !keys[key] {
			t.Errorf("the record is missing %s", key)
		}
	}
	if !strings.Contains(rec.Meta["evidence"], "malloc") {
		t.Errorf("the record does not show that the observable binary references the allocator: %q", rec.Meta["evidence"])
	}
}
