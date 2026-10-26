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

// TestTideClaimsMatchMeasurement compares the single tide document with the record and fails when any
// other document embeds a second copy of the block.
func TestTideClaimsMatchMeasurement(t *testing.T) {
	root := repoRoot(t)
	rec, err := Load(root, filepath.Join(root, DefaultTSV))
	if err != nil {
		t.Fatal(err)
	}
	want, err := SummaryBlock(rec)
	if err != nil {
		t.Fatal(err)
	}
	if got := measuredBlock(t, filepath.Join(root, DefaultOut)); got != strings.TrimRight(want, "\n") {
		t.Errorf("%s: the tide block differs from the record; run `go generate ./internal/benchgen`", DefaultOut)
	}
	for _, path := range []string{"README.md", "spec/STANDARD.md"} {
		text := readText(t, filepath.Join(root, path))
		if strings.Contains(text, BlockStart) || strings.Contains(text, BlockEnd) {
			t.Errorf("%s embeds a second tide block; link %s instead", path, DefaultOut)
		}
		if strings.Contains(text, "| QuarkLang compiled, observable |") {
			t.Errorf("%s copies the tide measurement table; link %s instead", path, DefaultOut)
		}
		if !strings.Contains(text, DefaultOut) {
			t.Errorf("%s does not link the single tide document %s", path, DefaultOut)
		}
	}
}

// readText returns one file's normalized contents.
func readText(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return normalize(string(b))
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

// TestSummaryBlockReportsLoad checks that the recorded load reaches the single tide report.
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
