package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang/stressgen"
)

// Verdicts of a measured case; the classifier tests them in the order they are declared here.
const (
	VerdictPanic          = "PANIC"           // a Go runtime crash surfaced in one of the engines
	VerdictTimeout        = "TIMEOUT"         // a phase outlived the wall-clock bound
	VerdictMemCap         = "MEMCAP"          // a phase outlived the memory ceiling
	VerdictAcceptMismatch = "ACCEPT_MISMATCH" // one front end accepted the source, the other rejected it
	VerdictDiverge        = "DIVERGE"         // both accepted it, and the run outcomes differ
	VerdictAgree          = "AGREE"           // both accepted it, with identical output and exit status
	VerdictRejectAgree    = "REJECT_AGREE"    // both rejected it, with identical diagnostics
	VerdictRejectDiff     = "REJECT_DIFF"     // both rejected it, with different wording
)

// errNoCases reports an empty case selection, which is always a mistake rather than a clean run.
var errNoCases = errors.New("no cases selected: check -cases, -filter, -skip and `qkstress gen`")

// loudVerdicts are the outcomes that fail a run unless the case is a documented known failure.
var loudVerdicts = map[string]bool{
	VerdictPanic:          true,
	VerdictTimeout:        true,
	VerdictMemCap:         true,
	VerdictAcceptMismatch: true,
	VerdictDiverge:        true,
}

// options carries the runner's paths, bounds and case selection.
type options struct {
	cases    string
	interp   string
	compiler string
	work     string
	timeout  time.Duration
	memCapMB int64
	known    string
	jsonPath string
	report   string
	notes    string
	filter   string
	skip     string
}

// engineResult is the two bounded phases one engine ran on a case.
type engineResult struct {
	Front    probeResult `json:"front"`
	Run      probeResult `json:"run"`
	Accepted bool        `json:"accepted"`
}

// caseResult is everything the report needs to know about one measured case.
type caseResult struct {
	Name     string       `json:"name"`
	Category string       `json:"category"`
	Doc      string       `json:"doc"`
	Bytes    int          `json:"bytes"`
	Interp   engineResult `json:"interpreter"`
	Compiler engineResult `json:"compiler"`
	Verdict  string       `json:"verdict"`
	Known    bool         `json:"known_failure"`
	Reason   string       `json:"known_reason,omitempty"`
}

// knownEntry is one documented known failure: the case it covers and why it is not a regression.
type knownEntry struct {
	Case   string
	Reason string
}

// cmdRun executes every selected case through both engines, writes the machine-readable results and the
// Markdown report, prints the summary and fails on any finding that is not a documented known failure.
func cmdRun(args []string) error {
	fs := newFlagSet("run")
	var opt options
	fs.StringVar(&opt.cases, "cases", "stress-out/cases", "directory holding the generated cases")
	fs.StringVar(&opt.interp, "interp", "stress-out/bin/quark", "interpreter binary")
	fs.StringVar(&opt.compiler, "compiler", "stress-out/bin/qkc", "compiler binary")
	fs.StringVar(&opt.work, "work", "stress-out", "scratch directory for temp files and caches")
	fs.DurationVar(&opt.timeout, "timeout", 10*time.Second, "wall-clock bound of one phase")
	fs.Int64Var(&opt.memCapMB, "mem", 2048, "memory ceiling of one phase, in MiB")
	fs.StringVar(&opt.known, "known", "stress/known-failures.txt", "known-failure list (a missing file is an empty list)")
	fs.StringVar(&opt.jsonPath, "json", "stress-out/results.json", "machine-readable results")
	fs.StringVar(&opt.report, "report", "stress-out/report.md", "generated Markdown report")
	fs.StringVar(&opt.notes, "notes", "", "optional Markdown file appended to the report as a notes section")
	fs.StringVar(&opt.filter, "filter", "", "only cases whose name matches this regular expression")
	fs.StringVar(&opt.skip, "skip", "", "drop cases whose name matches this regular expression (for a bounded CI subset)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if opt.timeout <= 0 {
		return errors.New("-timeout must be positive")
	}
	if opt.memCapMB <= 0 {
		return errors.New("-mem must be positive")
	}
	for _, bin := range []string{opt.interp, opt.compiler} {
		if _, err := os.Stat(bin); err != nil {
			return fmt.Errorf("%s is not there: build it first (see stress/README.md)", bin)
		}
	}
	if opt.notes != "" { // read it before the measurements: a missing notes file must not waste a full run
		if _, err := os.ReadFile(opt.notes); err != nil {
			return err
		}
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	for _, dir := range []string{"tmp", "ir", "cache"} {
		if err := os.MkdirAll(filepath.Join(opt.work, dir), 0o755); err != nil {
			return err
		}
	}
	corpus, err := selectCases(opt.cases, opt.filter, opt.skip)
	if err != nil {
		return err
	}
	known, knownText, err := loadKnown(opt.known)
	if err != nil {
		return err
	}
	// A list that names a case the corpus does not have would silently allow-list nothing, so it is
	// refused before any measurement is taken.
	if err := checkKnownNames(known); err != nil {
		return err
	}

	results := make([]caseResult, 0, len(corpus))
	for _, c := range corpus {
		r := runCase(opt, self, c)
		results = append(results, r)
		fmt.Fprintf(os.Stderr, "%-24s %-15s %s\n", r.Name, r.Verdict, r.summaryLine())
	}
	rep := assembleReport(opt, results, known, knownText)
	if err := writeJSON(opt.jsonPath, rep); err != nil {
		return err
	}
	if err := os.WriteFile(opt.report, []byte(renderMarkdown(rep)), 0o644); err != nil {
		return err
	}
	printSummary(os.Stdout, rep)
	return undocumentedFailure(rep)
}

// selectCases resolves the corpus on disk, refusing a stale or missing case directory; -filter and
// -skip select a subset by name, which is how a bounded CI job keeps its cost predictable.
func selectCases(dir, filter, skip string) ([]stressgen.Case, error) {
	var pattern *regexp.Regexp
	if filter != "" {
		compiled, err := regexp.Compile(filter)
		if err != nil {
			return nil, fmt.Errorf("-filter %q: %w", filter, err)
		}
		pattern = compiled
	}
	var skipped *regexp.Regexp
	if skip != "" {
		compiled, err := regexp.Compile(skip)
		if err != nil {
			return nil, fmt.Errorf("-skip %q: %w", skip, err)
		}
		skipped = compiled
	}
	var out []stressgen.Case
	for _, c := range stressgen.All() {
		if pattern != nil && !pattern.MatchString(c.Name) {
			continue
		}
		if skipped != nil && skipped.MatchString(c.Name) {
			continue
		}
		info, err := os.Stat(filepath.Join(dir, c.Name+".kq"))
		if err != nil {
			return nil, fmt.Errorf("%s: %w (run `qkstress gen -out %s` first)", c.Name, err, dir)
		}
		if info.Size() != int64(len(c.Source)) {
			return nil, fmt.Errorf("%s on disk is %d bytes, the generator builds %d: regenerate the corpus", c.Name, info.Size(), len(c.Source))
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, errNoCases
	}
	return out, nil
}

// runCase probes one case: interpreter acceptance, interpreter run, compiler code generation, compiler run.
func runCase(opt options, self string, c stressgen.Case) caseResult {
	path := filepath.Join(opt.cases, c.Name+".kq")
	irPath := filepath.Join(opt.work, "ir", c.Name+".ll")
	cacheDir := filepath.Join(opt.work, "cache", c.Name)
	_ = os.RemoveAll(cacheDir) // a cold cache: the measured compiler time is a full cold compile
	_ = os.MkdirAll(cacheDir, 0o755)
	_ = os.Remove(irPath)
	tmp := filepath.Join(opt.work, "tmp")
	memCapKB := opt.memCapMB * kibPerMiB
	// The children get slash-separated paths: they quote them back in their diagnostics, which must read
	// the same on every platform for the report to be comparable across the three jobs.
	runEnv := childEnv(map[string]string{"TMPDIR": childPath(tmp), "QK_LANG": "en"})
	compEnv := childEnv(map[string]string{"TMPDIR": childPath(tmp), "QUARK_CACHE": childPath(cacheDir), "QK_LANG": "en"})

	r := caseResult{Name: c.Name, Category: c.Category, Doc: c.Doc, Bytes: len(c.Source)}
	r.Interp.Front = runBounded([]string{self, "frontend", childPath(path)}, runEnv, opt.timeout, memCapKB)
	r.Interp.Run = runBounded([]string{opt.interp, childPath(path)}, runEnv, opt.timeout, memCapKB)
	r.Compiler.Front = runBounded([]string{opt.compiler, childPath(path), "-o", childPath(irPath)}, compEnv, opt.timeout, memCapKB)
	r.Compiler.Run = runBounded([]string{opt.compiler, "-run", childPath(path)}, compEnv, opt.timeout, memCapKB)
	r.Interp.Accepted = r.Interp.Front.Exit == 0
	r.Compiler.Accepted = r.Compiler.Front.Exit == 0
	r.Verdict = verdictOf(r)
	return r
}

// verdictOf classifies a measured case: bounds and crashes first, then front-end parity, then the
// agreement of the two run outcomes.
func verdictOf(r caseResult) string {
	switch {
	case r.anyPanic():
		return VerdictPanic
	case r.anyTimeout():
		return VerdictTimeout
	case r.anyMemCap():
		return VerdictMemCap
	case r.Interp.Accepted != r.Compiler.Accepted:
		return VerdictAcceptMismatch
	case !r.Interp.Accepted:
		if sameOutcome(r.Interp.Front, r.Compiler.Front) {
			return VerdictRejectAgree
		}
		return VerdictRejectDiff
	case sameOutcome(r.Interp.Run, r.Compiler.Run):
		return VerdictAgree
	default:
		return VerdictDiverge
	}
}

// sameOutcome compares two phases the way the project's own parity script does: exit status plus the
// merged stdout and stderr streams, with the platform's line ending folded away. The whole content is
// still compared, so any real divergence — a different word, a missing line, an extra byte — fails.
func sameOutcome(a, b probeResult) bool {
	return a.Exit == b.Exit && a.platformNormal() == b.platformNormal()
}

// anyPanic reports whether either engine crashed with a Go runtime error.
func (r caseResult) anyPanic() bool {
	return r.Interp.Front.Panicked || r.Interp.Run.Panicked || r.Compiler.Front.Panicked || r.Compiler.Run.Panicked
}

// anyTimeout reports whether a phase outlived the wall-clock bound.
func (r caseResult) anyTimeout() bool {
	return r.Interp.Front.TimedOut || r.Interp.Run.TimedOut || r.Compiler.Front.TimedOut || r.Compiler.Run.TimedOut
}

// anyMemCap reports whether a phase outlived the memory ceiling.
func (r caseResult) anyMemCap() bool {
	return r.Interp.Front.MemCapped || r.Interp.Run.MemCapped || r.Compiler.Front.MemCapped || r.Compiler.Run.MemCapped
}

// loud reports whether the verdict fails a run that has no matching known-failure entry.
func (r caseResult) loud() bool { return loudVerdicts[r.Verdict] }

// exitPair renders both engines' run exit statuses for the table and the progress line.
func (r caseResult) exitPair() string {
	return fmt.Sprintf("%d/%d", r.Interp.Run.Exit, r.Compiler.Run.Exit)
}

// summaryLine is the one-line description printed while the run proceeds.
func (r caseResult) summaryLine() string {
	parts := []string{fmt.Sprintf("interp %ss exit %d", r.Interp.Run.seconds(), r.Interp.Run.Exit),
		fmt.Sprintf("qkc gen %ss, run %ss exit %d", r.Compiler.Front.seconds(), r.Compiler.Run.seconds(), r.Compiler.Run.Exit)}
	if r.loud() {
		parts = append(parts, "first diagnostic: "+quoteOf(firstLine(r.Interp.Run.Stderr+r.Compiler.Run.Stderr)))
	}
	return strings.Join(parts, " | ")
}

// quoteOf renders a diagnostic for a one-line summary, keeping it short and single-line.
func quoteOf(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if s == "" {
		return "(none)"
	}
	if len(s) > positionSample {
		return s[:positionSample] + "…"
	}
	return s
}

// loadKnown parses the known-failure list: "<case-name> <reason>" per line, '#' starts a comment, and a
// missing file is an empty list rather than an error.
func loadKnown(path string) ([]knownEntry, string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	text := string(data)
	var out []knownEntry
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		name, reason, _ := strings.Cut(trimmed, "\t")
		if reason == "" {
			name, reason, _ = strings.Cut(trimmed, " ")
		}
		name = strings.TrimSpace(name)
		reason = strings.TrimSpace(reason)
		if name == "" {
			return nil, "", fmt.Errorf("%s:%d: no case name", path, i+1)
		}
		if reason == "" {
			return nil, "", fmt.Errorf("%s:%d: %s needs a reason", path, i+1, name)
		}
		out = append(out, knownEntry{Case: name, Reason: reason})
	}
	return out, text, nil
}

// checkKnownNames refuses a list that names a case the corpus does not have, which would silently
// allow-list nothing at all.
func checkKnownNames(entries []knownEntry) error {
	names := map[string]bool{}
	for _, c := range stressgen.All() {
		names[c.Name] = true
	}
	for _, e := range entries {
		if !names[e.Case] {
			return fmt.Errorf("known-failure entry %q is not a corpus case", e.Case)
		}
	}
	return nil
}

// assembleReport pairs the measurements with the known-failure list and the machine description.
func assembleReport(opt options, results []caseResult, known []knownEntry, knownText string) runReport {
	notes := ""
	if data, err := os.ReadFile(opt.notes); err == nil {
		notes = string(data)
	}
	byName := map[string]string{}
	for _, e := range known {
		byName[e.Case] = e.Reason
	}
	loudSeen := map[string]bool{}
	for i := range results {
		if results[i].loud() {
			loudSeen[results[i].Name] = true
			if reason, ok := byName[results[i].Name]; ok {
				results[i].Known = true // a listed case that no longer fails is reported as stale, not as known
				results[i].Reason = reason
			}
		}
	}
	var stale []string
	for _, e := range known {
		if !loudSeen[e.Case] {
			stale = append(stale, e.Case)
		}
	}
	sort.Strings(stale)
	return runReport{
		Command:   strings.Join(os.Args, " "),
		Commit:    gitRevision(),
		Date:      time.Now().Format(time.RFC3339),
		TimeoutMS: opt.timeout.Milliseconds(),
		MemCapMiB: opt.memCapMB,
		Locale:    "QK_LANG=en",
		Machine:   machineInfo(),
		Cases:     results,
		Known:     known,
		KnownPath: opt.known,
		KnownText: knownText,
		Notes:     notes,
		Stale:     stale,
	}
}

// undocumentedFailure returns the error that fails the run, naming every finding a human must look at.
func undocumentedFailure(rep runReport) error {
	var names []string
	for _, r := range rep.Cases {
		if r.loud() && !r.Known {
			names = append(names, r.Name+" ("+r.Verdict+")")
		}
	}
	if len(names) == 0 {
		return nil
	}
	return fmt.Errorf("%d undocumented robustness finding(s): %s — document them in stress/known-failures.txt or fix them", len(names), strings.Join(names, ", "))
}

// printSummary writes the run's roll-up: the per-case table, the verdict counts and the findings.
func printSummary(w io.Writer, rep runReport) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "case\tcategory\tbytes\tinterp s\tqkc gen s\tqkc run s\tpeak RSS MiB\texits i/c\tstderr pos i/c\tverdict")
	for _, r := range rep.Cases {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\t%s/%s\t%s\t%s/%s\t%s\n",
			r.Name, r.Category, r.Bytes,
			r.Interp.Run.seconds(), r.Compiler.Front.seconds(), r.Compiler.Run.seconds(),
			r.Interp.Run.peakMiB(), r.Compiler.Run.peakMiB(),
			r.exitPair(), positionMark(r.Interp.Run.Stderr), positionMark(r.Compiler.Run.Stderr),
			verdictLabel(r))
	}
	_ = tw.Flush()
	printCounts(w, rep)
	printFindings(w, rep)
}

// positionMark renders whether a run's stderr carried a source position.
func positionMark(stderr string) string {
	if hasPosition(stderr) {
		return "yes"
	}
	return "no"
}

// verdictLabel tags a known finding so the summary cannot be mistaken for a regression.
func verdictLabel(r caseResult) string {
	if r.Known {
		return r.Verdict + " (known)"
	}
	return r.Verdict
}

// printCounts writes the verdict histogram and the stale known-failure entries.
func printCounts(w io.Writer, rep runReport) {
	counts := map[string]int{}
	for _, r := range rep.Cases {
		counts[r.Verdict]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", k, counts[k]))
	}
	fmt.Fprintf(w, "\n%d cases: %s\n", len(rep.Cases), strings.Join(parts, ", "))
	if len(rep.Stale) > 0 {
		fmt.Fprintf(w, "stale known-failure entries (the case no longer fails, remove them): %s\n", strings.Join(rep.Stale, ", "))
	}
}

// printFindings writes one block per loud or informative case, so a failure is readable without the report.
func printFindings(w io.Writer, rep runReport) {
	interesting := make([]caseResult, 0, len(rep.Cases))
	for _, r := range rep.Cases {
		if r.loud() || r.Verdict == VerdictRejectDiff {
			interesting = append(interesting, r)
		}
	}
	if len(interesting) == 0 {
		fmt.Fprintln(w, "\nno findings: every case agreed, or was rejected cleanly by both engines")
		return
	}
	fmt.Fprintf(w, "\n%d findings:\n", len(interesting))
	for _, r := range interesting {
		fmt.Fprintf(w, "  %-15s %-24s %s\n", verdictLabel(r), r.Name, r.Doc)
		fmt.Fprintf(w, "      interp: exit %d in %ss\tqkc: exit %d in %ss (gen %ss)\n",
			r.Interp.Run.Exit, r.Interp.Run.seconds(), r.Compiler.Run.Exit, r.Compiler.Run.seconds(), r.Compiler.Front.seconds())
		if line := firstLine(r.Interp.Run.Stderr); line != "" {
			fmt.Fprintf(w, "      interp says: %s\n", quoteOf(line))
		}
		if line := firstLine(r.Compiler.Run.Stderr); line != "" {
			fmt.Fprintf(w, "      qkc says:    %s\n", quoteOf(line))
		}
	}
}
