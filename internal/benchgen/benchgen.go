// Package benchgen renders the measured reports and is the drift gate for their claims.
//
// Each suite has a measured record (spec/bench/*.tsv), an authored prose template
// (spec/bench/*-template.md) and a committed generated report (docs/*.md). A marked block from the
// report is embedded in README.md (and, for the tide suite, spec/STANDARD.md §10), so a hand-edited
// number fails a claim test instead of drifting away from the record.
package benchgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// DefaultTSV is the measured tide record, relative to the checkout root.
	DefaultTSV = "spec/bench/tide.tsv"
	// DefaultTemplate is the authored tide prose, relative to the checkout root.
	DefaultTemplate = "spec/bench/tide-template.md"
	// DefaultOut is the committed tide report, relative to the checkout root.
	DefaultOut = "docs/tide-benchmark.md"
	// BlockStart and BlockEnd delimit the tide block the README and the standard embed.
	BlockStart = "<!-- tide-measurements -->"
	BlockEnd   = "<!-- /tide-measurements -->"
	// ClaimsTSV is the measured performance-claims record, relative to the checkout root.
	ClaimsTSV = "spec/bench/claims.tsv"
	// ClaimsTemplate is the authored claims prose, relative to the checkout root.
	ClaimsTemplate = "spec/bench/claims-template.md"
	// ClaimsOut is the committed claims report, relative to the checkout root.
	ClaimsOut = "docs/performance-claims.md"
	// ClaimsShapes is the canonical-shape table the claims harness and this package agree on.
	ClaimsShapes = "spec/bench/claims-shapes.tsv"
	// ClaimsRecord is the record format the claims validator accepts.
	ClaimsRecord = "claims-v1"
	// ClaimsBlockStart and ClaimsBlockEnd delimit the block README embeds.
	ClaimsBlockStart = "<!-- performance-claims -->"
	ClaimsBlockEnd   = "<!-- /performance-claims -->"
)

// canonicalRounds is the round count every timing row in the authoritative tide record must carry.
const canonicalRounds int64 = 100_000_000

// canonicalRuns is the timed-run count every timing row in the authoritative tide record must carry.
const canonicalRuns int64 = 5

// Options locates the tide inputs and chooses whether to write or to check the report.
type Options struct {
	Root     string // checkout root (default ".")
	TSV      string // measured record, relative to Root (default DefaultTSV)
	Template string // authored prose template, relative to Root (default DefaultTemplate)
	Out      string // generated report, relative to Root (default DefaultOut)
	Check    bool   // compare with Out instead of writing it
}

// Suite bundles one report's paths, its record validation and its block renderer.
type Suite struct {
	Name        string                        // report name, e.g. tide
	TSV         string                        // measured record, relative to the checkout root
	Template    string                        // authored prose template, relative to the checkout root
	Out         string                        // committed generated report, relative to the checkout root
	BlockStart  string                        // marker that opens the embedded block
	BlockEnd    string                        // marker that closes the embedded block
	Validate    func(*Record, string) error   // record validation, with the checkout root for shape tables
	RenderBlock func(*Record) (string, error) // renders the embedded block
}

// TideSuite is the tide report; its block is embedded in README.md and spec/STANDARD.md §10.
var TideSuite = Suite{
	Name:        "tide",
	TSV:         DefaultTSV,
	Template:    DefaultTemplate,
	Out:         DefaultOut,
	BlockStart:  BlockStart,
	BlockEnd:    BlockEnd,
	Validate:    func(rec *Record, _ string) error { return rec.Validate() },
	RenderBlock: SummaryBlock,
}

// ClaimsSuite is the fib/sum/latency/backend report; its block is embedded in README.md.
var ClaimsSuite = Suite{
	Name:        "claims",
	TSV:         ClaimsTSV,
	Template:    ClaimsTemplate,
	Out:         ClaimsOut,
	BlockStart:  ClaimsBlockStart,
	BlockEnd:    ClaimsBlockEnd,
	Validate:    func(rec *Record, root string) error { return rec.ValidateClaims(root) },
	RenderBlock: ClaimsBlock,
}

// Suites lists every generated report.
func Suites() []Suite { return []Suite{TideSuite, ClaimsSuite} }

// Row is one measured workload: a timing (microseconds) or a counter (a ratio or a block count).
type Row struct {
	Kind     string  // timing, counter
	Key      string  // stable identity, e.g. compiled_observable
	Label    string  // human label
	Language string  // QuarkLang, C, Rust, Go, interpreter
	Rounds   int64   // loop rounds (timing) or waves (counter)
	Runs     int64   // timed runs (timing) or allocation calls (counter)
	Median   float64 // microseconds for timing, a ratio or block count for a counter
	Unit     string  // us, ratio, blocks
	Min      float64 // minimum for timing
	Max      float64 // maximum for timing
	Stdev    float64 // sample standard deviation for timing
	Spread   float64 // (max-min)/median as a percentage
	Checksum string  // shared checksum, or "-" when the workload has no checksum
	Note     string  // what the row counts
}

// Record is the parsed measurement record: metadata plus rows.
type Record struct {
	Meta map[string]string
	Rows []Row
}

// Run validates the tide record and either writes the report or fails when it is stale.
func Run(opts Options) error {
	root := opts.Root
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	return runReport(absRoot, TideSuite, opts.TSV, opts.Template, opts.Out, opts.Check)
}

// RunSuite validates one suite's record and either writes its report or fails when it is stale.
func RunSuite(root string, s Suite, check bool) error {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	return runReport(absRoot, s, "", "", "", check)
}

// runReport loads and validates the record, renders the suite's report and writes or checks it.
func runReport(absRoot string, s Suite, tsv, template, out string, check bool) error {
	rec, err := Load(absRoot, pathFor(absRoot, tsv, s.TSV))
	if err != nil {
		return err
	}
	if err := s.Validate(rec, absRoot); err != nil {
		return err
	}
	tmplPath := pathFor(absRoot, template, s.Template)
	tmpl, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	rendered, err := renderReport(rec, tmpl, s)
	if err != nil {
		return err
	}
	outPath := pathFor(absRoot, out, s.Out)
	if check {
		old, err := os.ReadFile(outPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", display(absRoot, outPath), err)
		}
		if normalize(string(old)) != string(rendered) {
			return fmt.Errorf("%s is out of date: run `go generate ./internal/benchgen`", display(absRoot, outPath))
		}
		return nil
	}
	if err := os.WriteFile(outPath, rendered, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", display(absRoot, outPath), err)
	}
	return nil
}

// Load parses the measurement record; comments of the form "# key<TAB>value" are metadata.
func Load(root, path string) (*Record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", display(root, path), err)
	}
	rec := &Record{Meta: map[string]string{}}
	for n, line := range strings.Split(normalize(string(b)), "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if !strings.HasPrefix(line, "# ") {
				continue
			}
			parts := strings.SplitN(strings.TrimPrefix(line, "# "), "\t", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("line %d: metadata needs a key and a value", n+1)
			}
			rec.Meta[parts[0]] = parts[1]
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 14 {
			return nil, fmt.Errorf("line %d: expected 14 fields, got %d", n+1, len(fields))
		}
		row, err := parseRow(fields)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n+1, err)
		}
		rec.Rows = append(rec.Rows, row)
	}
	if len(rec.Rows) == 0 {
		return nil, fmt.Errorf("%s has no measurement rows", display(root, path))
	}
	return rec, nil
}

// parseRow converts one tab-separated row into a Row.
func parseRow(fields []string) (Row, error) {
	row := Row{
		Kind:     fields[0],
		Key:      fields[1],
		Label:    fields[2],
		Language: fields[3],
		Unit:     fields[7],
		Checksum: fields[12],
		Note:     fields[13],
	}
	rounds, err := strconv.ParseInt(fields[4], 10, 64)
	if err != nil {
		return row, fmt.Errorf("rounds %q: %w", fields[4], err)
	}
	runs, err := strconv.ParseInt(fields[5], 10, 64)
	if err != nil {
		return row, fmt.Errorf("runs %q: %w", fields[5], err)
	}
	row.Rounds, row.Runs = rounds, runs
	if row.Median, err = parseNumber(fields[6]); err != nil {
		return row, fmt.Errorf("median: %w", err)
	}
	if row.Kind == "timing" {
		if row.Min, err = parseNumber(fields[8]); err != nil {
			return row, fmt.Errorf("min: %w", err)
		}
		if row.Max, err = parseNumber(fields[9]); err != nil {
			return row, fmt.Errorf("max: %w", err)
		}
		if row.Stdev, err = parseNumber(fields[10]); err != nil {
			return row, fmt.Errorf("stdev: %w", err)
		}
		if row.Spread, err = parseNumber(fields[11]); err != nil {
			return row, fmt.Errorf("spread: %w", err)
		}
	}
	return row, nil
}

// parseNumber accepts a non-negative decimal number, with "-" meaning "not applicable".
func parseNumber(s string) (float64, error) {
	if s == "-" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, fmt.Errorf("%q is negative", s)
	}
	return v, nil
}

// Validate checks the record's required rows, its arithmetic and its platform neutrality.
func (r *Record) Validate() error {
	if err := validateMeta(r.Meta); err != nil {
		return err
	}
	seen := map[string]bool{}
	checksums := map[string]bool{}
	for _, row := range r.Rows {
		if err := validateRow(row, seen, checksums); err != nil {
			return err
		}
	}
	if len(checksums) > 1 {
		return fmt.Errorf("the controls disagree on the checksum: %v", checksums)
	}
	return validateCoverage(seen, r.Meta["skipped"])
}

// validateMeta checks the required metadata keys, the canonical shape and their platform neutrality.
func validateMeta(meta map[string]string) error {
	for _, key := range []string{"class", "load", "toolchain", "flags", "counted", "skipped", "evidence"} {
		if strings.TrimSpace(meta[key]) == "" {
			return fmt.Errorf("metadata is missing %q", key)
		}
	}
	if err := validateCanonicalShape(meta); err != nil {
		return err
	}
	for _, value := range meta {
		if machineSpecific(value) {
			return fmt.Errorf("metadata %q is machine-specific", value)
		}
	}
	return nil
}

// validateCanonicalShape refuses metadata that does not declare the canonical 1e8-round, 5-run shape.
func validateCanonicalShape(meta map[string]string) error {
	for _, param := range []struct {
		key  string
		want int64
	}{{"rounds", canonicalRounds}, {"runs", canonicalRuns}} {
		raw := strings.TrimSpace(meta[param.key])
		if raw == "" {
			return fmt.Errorf("metadata is missing %q: a canonical record must declare %q and %q (the 1e8-round, 5-run measurement that bench/tide/run.sh --record writes)", param.key, "# rounds 100000000", "# runs 5")
		}
		got, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("metadata %q is %q, not an integer: a canonical record must declare %q and %q", param.key, raw, "# rounds 100000000", "# runs 5")
		}
		if got != param.want {
			return fmt.Errorf("metadata %q is %d, but a canonical record must have %d rounds and %d timed runs; a probe is not a record", param.key, got, canonicalRounds, canonicalRuns)
		}
	}
	return nil
}

// validateRow checks one row and records its key and checksum.
func validateRow(row Row, seen, checksums map[string]bool) error {
	if row.Key == "" || seen[row.Key] {
		return fmt.Errorf("row key %q is empty or duplicated", row.Key)
	}
	seen[row.Key] = true
	if row.Label == "" || row.Language == "" || row.Note == "" {
		return fmt.Errorf("%s: label, language and note are required", row.Key)
	}
	for _, s := range []string{row.Key, row.Label, row.Language, row.Unit, row.Checksum, row.Note} {
		if machineSpecific(s) {
			return fmt.Errorf("%s: %q is machine-specific", row.Key, s)
		}
	}
	switch row.Kind {
	case "timing":
		return validateTiming(row, checksums)
	case "counter":
		return validateCounter(row)
	default:
		return fmt.Errorf("%s: unknown kind %q", row.Key, row.Kind)
	}
}

// validateTiming checks a timing row and records its checksum.
func validateTiming(row Row, checksums map[string]bool) error {
	if row.Unit != "us" {
		return fmt.Errorf("%s: timing unit must be us, got %q", row.Key, row.Unit)
	}
	if row.Runs < 1 || row.Rounds < 1 {
		return fmt.Errorf("%s: rounds and runs must be positive", row.Key)
	}
	if row.Rounds != canonicalRounds || row.Runs != canonicalRuns {
		return fmt.Errorf("%s: a canonical timing row must carry %d rounds and %d timed runs, got %d rounds and %d runs", row.Key, canonicalRounds, canonicalRuns, row.Rounds, row.Runs)
	}
	if row.Min > row.Median || row.Median > row.Max {
		return fmt.Errorf("%s: median %v is outside %v..%v", row.Key, row.Median, row.Min, row.Max)
	}
	if row.Stdev < 0 || row.Spread < 0 {
		return fmt.Errorf("%s: stdev and spread must be non-negative", row.Key)
	}
	if row.Checksum == "-" {
		return nil
	}
	if _, err := strconv.ParseInt(row.Checksum, 10, 64); err != nil {
		return fmt.Errorf("%s: checksum %q is not an integer", row.Key, row.Checksum)
	}
	checksums[row.Checksum] = true
	return nil
}

// validateCounter checks a counter row's unit and range.
func validateCounter(row Row) error {
	if row.Unit != "ratio" && row.Unit != "blocks" {
		return fmt.Errorf("%s: counter unit must be ratio or blocks, got %q", row.Key, row.Unit)
	}
	if row.Unit == "ratio" && row.Median > 1 {
		return fmt.Errorf("%s: ratio %v is above 1", row.Key, row.Median)
	}
	if row.Unit == "blocks" && row.Median != float64(int64(row.Median)) {
		return fmt.Errorf("%s: block count %v is not an integer", row.Key, row.Median)
	}
	return nil
}

// validateCoverage checks that every required row is present and that a missing control names its skip.
func validateCoverage(seen map[string]bool, skipped string) error {
	for _, key := range []string{"compiled_observable", "c_observable", "compiled_floor", "legacy_floor", "policy_reuse", "policy_fragmentation"} {
		if !seen[key] {
			return fmt.Errorf("the record is missing the required row %q", key)
		}
	}
	for _, key := range []string{"rust_observable", "go_observable"} {
		language := strings.SplitN(key, "_", 2)[0]
		if !seen[key] && !strings.Contains(strings.ToLower(skipped), language) {
			return fmt.Errorf("%s is missing and the skipped metadata does not name %s", key, language)
		}
	}
	return nil
}

// Render fills the tide template's summary placeholder and marks the result as generated.
func Render(rec *Record, template []byte) ([]byte, error) {
	return renderReport(rec, template, TideSuite)
}

// renderReport fills one suite's summary placeholder and marks the result as generated.
func renderReport(rec *Record, template []byte, s Suite) ([]byte, error) {
	block, err := s.RenderBlock(rec)
	if err != nil {
		return nil, err
	}
	text := normalize(string(template))
	if !strings.Contains(text, "{{summary}}") {
		return nil, fmt.Errorf("template has no {{summary}} placeholder")
	}
	text = strings.Replace(text, "{{summary}}", block, 1)
	note := fmt.Sprintf("<!-- Generated by cmd/qkbench from %s and %s. Do not edit: run `go generate ./internal/benchgen`. -->", s.TSV, s.Template)
	if !strings.HasPrefix(text, note) {
		text = note + "\n\n" + text
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return []byte(text), nil
}

// SummaryBlock renders the marked block that is embedded in the report, README.md and §10.
func SummaryBlock(rec *Record) (string, error) {
	var b strings.Builder
	b.WriteString(BlockStart + "\n")
	fmt.Fprintf(&b, "Measured on %s. Load: %s. Toolchain: %s. Flags: %s. Skipped controls: %s.\n\n", rec.Meta["class"], rec.Meta["load"], rec.Meta["toolchain"], rec.Meta["flags"], rec.Meta["skipped"])
	fmt.Fprintf(&b, "What is counted: %s. Evidence: %s.\n\n", rec.Meta["counted"], rec.Meta["evidence"])

	b.WriteString("| Workload | Rounds | Runs | Median | Min–max | Spread | Checksum | What is counted |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---|\n")
	for _, row := range rec.Rows {
		if row.Kind != "timing" || !strings.HasSuffix(row.Key, "_observable") {
			continue
		}
		writeTimingRow(&b, row)
	}

	b.WriteString("\nThe same loops without the sink. The checksum keeps the loop alive; on the pre-stage-1 libc path the legacy row removed the use entirely. Since stage 1 the QuarkLang rows call the opaque counted-slab allocator, so they measure allocation too; the C/Rust/Go floor rows still measure their own optimizer floors.\n\n")
	b.WriteString("| Workload | Rounds | Runs | Median | Min–max | Spread | Checksum | What is counted |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---|\n")
	for _, row := range rec.Rows {
		if row.Kind != "timing" || strings.HasSuffix(row.Key, "_observable") {
			continue
		}
		writeTimingRow(&b, row)
	}
	b.WriteString("\nThe historical `1 ms` claim measured the legacy row on the pre-stage-1 libc path, where LLVM removed the loop; that is what `0 µs` meant. On the stage-1 tree the same row is no longer erased (the allocator call is opaque), so the recorded value is a real allocation measurement and the label no longer calls it a dead-code floor.\n\n")

	b.WriteString("| Policy measurement | Waves | Allocations | Value | Note |\n")
	b.WriteString("|---|---:|---:|---:|---|\n")
	for _, row := range rec.Rows {
		if row.Kind != "counter" {
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", row.Label, formatCount(row.Rounds), formatCount(row.Runs), formatCounter(row), row.Note)
	}
	b.WriteString(BlockEnd + "\n")
	return b.String(), nil
}

// writeTimingRow renders one timing row.
func writeTimingRow(b *strings.Builder, row Row) {
	fmt.Fprintf(b, "| %s | %s | %d | %s | %s | %s | %s | %s |\n",
		row.Label, formatRounds(row.Rounds), row.Runs, formatTiming(row.Median), formatRange(row.Min, row.Max, row.Median), formatPct(row.Spread), formatChecksum(row.Checksum), row.Note)
}

// formatTiming renders microseconds in the largest unit that keeps two decimals.
func formatTiming(us float64) string {
	switch {
	case us >= 1_000_000:
		return fmt.Sprintf("%.2f s", us/1_000_000)
	case us >= 1_000:
		return fmt.Sprintf("%.2f ms", us/1_000)
	default:
		return fmt.Sprintf("%.0f µs", us)
	}
}

// formatRange renders a minimum and maximum in the median's unit.
func formatRange(min, max, median float64) string {
	_, unit := timingParts(median)
	switch unit {
	case "s":
		return fmt.Sprintf("%.2f–%.2f s", min/1_000_000, max/1_000_000)
	case "ms":
		return fmt.Sprintf("%.2f–%.2f ms", min/1_000, max/1_000)
	default:
		return fmt.Sprintf("%.0f–%.0f µs", min, max)
	}
}

// timingParts returns the value and unit formatTiming would use.
func timingParts(us float64) (float64, string) {
	switch {
	case us >= 1_000_000:
		return us / 1_000_000, "s"
	case us >= 1_000:
		return us / 1_000, "ms"
	default:
		return us, "µs"
	}
}

// formatPct renders a percentage with two decimals.
func formatPct(v float64) string { return fmt.Sprintf("%.2f%%", v) }

// formatRounds renders a round count compactly.
func formatRounds(n int64) string {
	switch n {
	case 100_000_000:
		return "1e8"
	case 1_000_000:
		return "1e6"
	}
	return formatCount(n)
}

// formatCount renders an integer with thousands separators.
func formatCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ",")
	if neg {
		out = "-" + out
	}
	return out
}

// formatChecksum renders an integer checksum, or "-" when the row has none.
func formatChecksum(s string) string {
	if s == "-" {
		return "-"
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	return formatCount(n)
}

// formatCounter renders a ratio as a percentage or a block count as an integer.
func formatCounter(row Row) string {
	if row.Unit == "blocks" {
		return formatCount(int64(row.Median))
	}
	return fmt.Sprintf("%.4f%%", row.Median*100)
}

// machineSpecific reports whether a field carries an absolute path or a host path separator.
func machineSpecific(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "/") || strings.Contains(t, "/home/") || strings.Contains(t, "/Users/") || strings.Contains(t, "C:\\")
}

// normalize converts CRLF to LF so a Windows checkout compares byte for byte.
func normalize(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

// pathFor resolves value against root, falling back to def when value is empty.
func pathFor(root, value, def string) string {
	if value == "" {
		value = def
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}

// display renders a path relative to root when it is inside it, for stable error messages.
func display(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return filepath.ToSlash(rel)
}

//go:generate go run ../../cmd/qkbench -root ../..

// claimShape is the canonical rounds and timed runs one claims family records.
type claimShape struct {
	rounds int64
	runs   int64
}

// claimRequired lists the rows every claims record must carry, by family.
var claimRequired = map[string][]string{
	"fib30":       {"qk", "c", "rust", "go", "erl"},
	"fib35":       {"qk", "c", "rust", "go"},
	"sum":         {"qk", "go"},
	"p99_p50":     {"qk", "c"},
	"p99_p99":     {"qk", "c"},
	"back_fib30":  {"qk", "native"},
	"back_loop":   {"qk", "native"},
	"back_list":   {"qk", "native"},
	"back_struct": {"qk", "native"},
	"back_hash":   {"qk", "native"},
	"back_str":    {"qk", "native"},
}

// loadClaimShapes reads the canonical-shape table shared with bench/claims/run.sh.
func loadClaimShapes(root string) (map[string]claimShape, error) {
	b, err := os.ReadFile(filepath.Join(root, ClaimsShapes))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ClaimsShapes, err)
	}
	shapes := map[string]claimShape{}
	for n, line := range strings.Split(normalize(string(b)), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			return nil, fmt.Errorf("%s line %d: expected three fields, got %d", ClaimsShapes, n+1, len(fields))
		}
		rounds, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || rounds < 1 {
			return nil, fmt.Errorf("%s line %d: rounds %q is not a positive integer", ClaimsShapes, n+1, fields[1])
		}
		runs, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || runs < 1 {
			return nil, fmt.Errorf("%s line %d: runs %q is not a positive integer", ClaimsShapes, n+1, fields[2])
		}
		shapes[fields[0]] = claimShape{rounds: rounds, runs: runs}
	}
	if len(shapes) == 0 {
		return nil, fmt.Errorf("%s has no shape rows", ClaimsShapes)
	}
	return shapes, nil
}

// claimFamily returns the longest family prefix that owns one row key.
func claimFamily(key string, shapes map[string]claimShape) (string, claimShape, bool) {
	best := ""
	for family := range shapes {
		if key == family || strings.HasPrefix(key, family+"_") {
			if len(family) > len(best) {
				best = family
			}
		}
	}
	if best == "" {
		return "", claimShape{}, false
	}
	return best, shapes[best], true
}

// claimSkipName maps a row suffix to the name the skipped metadata uses.
func claimSkipName(language string) string {
	switch language {
	case "rust":
		return "rust"
	case "erl":
		return "erlang"
	}
	return language
}

// ValidateClaims checks the claims record's metadata, canonical shape, checksum agreement and coverage.
func (r *Record) ValidateClaims(root string) error {
	if err := validateClaimsMeta(r.Meta); err != nil {
		return err
	}
	shapes, err := loadClaimShapes(root)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	familyChecksum := map[string]string{}
	for _, row := range r.Rows {
		if err := validateClaimsRow(row, shapes, seen, familyChecksum); err != nil {
			return err
		}
	}
	return validateClaimsCoverage(seen, r.Meta["skipped"])
}

// validateClaimsMeta checks the metadata the claims record and its generator agree on.
func validateClaimsMeta(meta map[string]string) error {
	for _, key := range []string{"record", "runs", "class", "load", "toolchain", "flags", "counted", "skipped", "evidence"} {
		if strings.TrimSpace(meta[key]) == "" {
			return fmt.Errorf("claims metadata is missing %q", key)
		}
	}
	if meta["record"] != ClaimsRecord {
		return fmt.Errorf("claims record is %q, not %q", meta["record"], ClaimsRecord)
	}
	if meta["runs"] != "5" {
		return fmt.Errorf("claims record declares %q timed runs, but the canonical shape is 5", meta["runs"])
	}
	for _, value := range meta {
		if machineSpecific(value) {
			return fmt.Errorf("claims metadata %q is machine-specific", value)
		}
	}
	return nil
}

// validateClaimsRow checks one timing row against its family's canonical shape and checksum.
func validateClaimsRow(row Row, shapes map[string]claimShape, seen map[string]bool, familyChecksum map[string]string) error {
	if row.Kind != "timing" {
		return fmt.Errorf("%s: the claims record has no counter rows, got %q", row.Key, row.Kind)
	}
	if !claimAllowed(row.Key) {
		return fmt.Errorf("%s: no such claims row", row.Key)
	}
	if seen[row.Key] {
		return fmt.Errorf("%s: duplicate claims row", row.Key)
	}
	seen[row.Key] = true
	if row.Label == "" || row.Language == "" || row.Note == "" {
		return fmt.Errorf("%s: label, language and note are required", row.Key)
	}
	for _, value := range []string{row.Key, row.Label, row.Language, row.Unit, row.Checksum, row.Note} {
		if machineSpecific(value) {
			return fmt.Errorf("%s: %q is machine-specific", row.Key, value)
		}
	}
	family, shape, ok := claimFamily(row.Key, shapes)
	if !ok {
		return fmt.Errorf("%s: no canonical shape row owns this key", row.Key)
	}
	if row.Rounds != shape.rounds || row.Runs != shape.runs {
		return fmt.Errorf("%s: canonical shape is %d rounds and %d timed runs, got %d and %d", row.Key, shape.rounds, shape.runs, row.Rounds, row.Runs)
	}
	if row.Unit != "us" {
		return fmt.Errorf("%s: timing unit must be us, got %q", row.Key, row.Unit)
	}
	if row.Min > row.Median || row.Median > row.Max {
		return fmt.Errorf("%s: median %v is outside %v..%v", row.Key, row.Median, row.Min, row.Max)
	}
	if row.Stdev < 0 || row.Spread < 0 {
		return fmt.Errorf("%s: stdev and spread must be non-negative", row.Key)
	}
	if _, err := strconv.ParseInt(row.Checksum, 10, 64); err != nil {
		return fmt.Errorf("%s: checksum %q is not an integer; every claims row must print its result", row.Key, row.Checksum)
	}
	if previous := familyChecksum[family]; previous != "" && previous != row.Checksum {
		return fmt.Errorf("%s: checksum %s does not match %s in family %s", row.Key, row.Checksum, previous, family)
	}
	familyChecksum[family] = row.Checksum
	return nil
}

// claimAllowed reports whether a row key is one the claims record requires.
func claimAllowed(key string) bool {
	for family, languages := range claimRequired {
		for _, language := range languages {
			if key == family+"_"+language {
				return true
			}
		}
	}
	return false
}

// validateClaimsCoverage checks that every required row is present or explicitly skipped.
func validateClaimsCoverage(seen map[string]bool, skipped string) error {
	for family, languages := range claimRequired {
		for _, language := range languages {
			key := family + "_" + language
			if seen[key] {
				continue
			}
			if strings.Contains(strings.ToLower(skipped), claimSkipName(language)) {
				continue
			}
			return fmt.Errorf("claims: the record is missing %q and skipped does not name %s", key, claimSkipName(language))
		}
	}
	return nil
}

// ClaimsBlock renders the marked block the README's performance section embeds.
func ClaimsBlock(rec *Record) (string, error) {
	var b strings.Builder
	b.WriteString(ClaimsBlockStart + "\n")
	fmt.Fprintf(&b, "Measured on %s. Load: %s. Toolchain: %s. Flags: %s. Skipped controls: %s.\n\n", rec.Meta["class"], rec.Meta["load"], rec.Meta["toolchain"], rec.Meta["flags"], rec.Meta["skipped"])
	fmt.Fprintf(&b, "What is counted: %s. Evidence: %s.\n\n", rec.Meta["counted"], rec.Meta["evidence"])
	b.WriteString("| Workload | Language | Rounds | Runs | Median | Min–max | Spread | Checksum | What is counted |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---|\n")
	for _, row := range rec.Rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %s | %s | %s | %s | %s |\n",
			row.Label, row.Language, formatRounds(row.Rounds), row.Runs, formatTiming(row.Median),
			formatRange(row.Min, row.Max, row.Median), formatPct(row.Spread), formatChecksum(row.Checksum), row.Note)
	}
	b.WriteString(ClaimsBlockEnd + "\n")
	return b.String(), nil
}
