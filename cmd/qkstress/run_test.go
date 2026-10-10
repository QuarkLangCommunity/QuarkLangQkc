package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang/stressgen"
)

// TestVerdictPrecedence checks that the classifier reports the most serious outcome first, whatever the
// later phases did: a panic outranks a timeout, a bound outranks a parity difference.
func TestVerdictPrecedence(t *testing.T) {
	accepted := engineResult{Front: probeResult{Exit: 0}, Run: probeResult{Exit: 0}, Accepted: true}
	rejected := engineResult{Front: probeResult{Exit: 1}, Run: probeResult{Exit: 1}, Accepted: false}

	cases := []struct {
		name string
		r    caseResult
		want string
	}{
		{"both accept and agree", caseResult{Interp: accepted, Compiler: accepted}, VerdictAgree},
		{"both reject identically", caseResult{Interp: rejected, Compiler: rejected}, VerdictRejectAgree},
		{"one accepts", caseResult{Interp: accepted, Compiler: rejected}, VerdictAcceptMismatch},
		{"both accept, outputs differ", caseResult{
			Interp:   accepted,
			Compiler: engineResult{Front: probeResult{Exit: 0}, Run: probeResult{Exit: 1}, Accepted: true},
		}, VerdictDiverge},
		{"panic wins over everything", caseResult{
			Interp:   engineResult{Front: probeResult{Exit: 0}, Run: probeResult{Exit: 2, Panicked: true, TimedOut: true}, Accepted: true},
			Compiler: rejected,
		}, VerdictPanic},
		{"timeout wins over a mismatch", caseResult{
			Interp:   engineResult{Front: probeResult{Exit: 0}, Run: probeResult{Exit: -1, TimedOut: true}, Accepted: true},
			Compiler: rejected,
		}, VerdictTimeout},
		{"memory ceiling wins over a divergence", caseResult{
			Interp:   accepted,
			Compiler: engineResult{Front: probeResult{Exit: 0, MemCapped: true}, Run: probeResult{Exit: 1}, Accepted: true},
		}, VerdictMemCap},
	}
	for _, tc := range cases {
		if got := verdictOf(tc.r); got != tc.want {
			t.Errorf("%s: verdict = %s, want %s", tc.name, got, tc.want)
		}
	}
}

// TestRejectDiffIsQuiet checks that two engines rejecting the same input in different words stays a
// note rather than a failure, which is what keeps the suite's signal-to-noise ratio usable.
func TestRejectDiffIsQuiet(t *testing.T) {
	r := caseResult{
		Interp:   engineResult{Front: probeResult{Exit: 1, Stderr: "error: LexError: unterminated string literal at line 1\n"}, Accepted: false},
		Compiler: engineResult{Front: probeResult{Exit: 1, Stderr: "error: compiler: nope\n"}, Accepted: false},
	}
	r.Verdict = verdictOf(r)
	if r.Verdict != VerdictRejectDiff {
		t.Fatalf("verdict = %s, want %s", r.Verdict, VerdictRejectDiff)
	}
	if r.loud() {
		t.Fatal("a diagnostic wording difference must not fail the run")
	}
}

// TestSameOutcomeUsesTheParityRule checks that agreement is decided the way the project's own parity
// script decides it: exit status plus the merged stdout and stderr streams.
func TestSameOutcomeUsesTheParityRule(t *testing.T) {
	base := probeResult{Exit: 0, Stdout: "7\n"}
	if !sameOutcome(base, probeResult{Exit: 0, Stdout: "7\n"}) {
		t.Error("identical runs must agree")
	}
	if sameOutcome(base, probeResult{Exit: 0, Stdout: "8\n"}) {
		t.Error("different output must not agree")
	}
	if sameOutcome(base, probeResult{Exit: 1, Stdout: "7\n"}) {
		t.Error("different exit status must not agree")
	}
	if !sameOutcome(base, probeResult{Exit: 0, Stderr: "", Stdout: "7\n"}) {
		t.Error("an empty stderr must not change the outcome")
	}
}

// TestSameOutcomeFoldsThePlatformLineEnding checks the comparison across the two runtimes: the
// interpreter writes LF and a linked case writes through the C runtime, which turns every LF into CRLF
// in text mode on Windows. The same text must therefore agree across both spellings, while every real
// difference — an extra byte, a missing line, a lone CR — is still a divergence.
func TestSameOutcomeFoldsThePlatformLineEnding(t *testing.T) {
	lf := probeResult{Exit: 0, Stdout: "7\nhello\n"}
	if !sameOutcome(lf, probeResult{Exit: 0, Stdout: "7\r\nhello\r\n"}) {
		t.Error("the same text with Windows line endings must agree")
	}
	if !sameOutcome(
		probeResult{Exit: 0, Stdout: "7\n", Stderr: "warning\n"},
		probeResult{Exit: 0, Stdout: "7\r\n", Stderr: "warning\r\n"},
	) {
		t.Error("both streams must be folded, not just stdout")
	}
	if !sameOutcome(lf, probeResult{Exit: 0, Stdout: "7\r\nhello\n"}) {
		t.Error("a stream mixing both endings must fold to the same content")
	}
	if sameOutcome(lf, probeResult{Exit: 0, Stdout: "7\nhello"}) {
		t.Error("a missing trailing newline is a content difference and must not agree")
	}
	if sameOutcome(lf, probeResult{Exit: 0, Stdout: "7\rhello\r"}) {
		t.Error("a lone CR is content, not a line ending, and must not agree")
	}
	if sameOutcome(lf, probeResult{Exit: 0, Stdout: "7\nhello!\n"}) {
		t.Error("an extra byte must not agree")
	}
	if sameOutcome(lf, probeResult{Exit: 1, Stdout: "7\r\nhello\r\n"}) {
		t.Error("a different exit status must not agree, whatever the streams say")
	}
}

// TestChildPathsUseForwardSlashes checks that the paths handed to the engines are spelled the same way
// everywhere, so a diagnostic quoting one reads stress-out/cases/x.kq and not stress-out\cases\x.kq.
func TestChildPathsUseForwardSlashes(t *testing.T) {
	got := childPath(filepath.Join("stress-out", "cases", "bad_unbalanced_ifdef.kq"))
	if got != "stress-out/cases/bad_unbalanced_ifdef.kq" {
		t.Errorf("child path = %q, want stress-out/cases/bad_unbalanced_ifdef.kq", got)
	}
	if strings.Contains(got, `\`) {
		t.Errorf("child path carries the host separator: %q", got)
	}
}

// TestPositionDetection checks both engines' position spellings, since the report answers "does the
// diagnostic tell the user where the problem is?" from exactly this.
func TestPositionDetection(t *testing.T) {
	for _, s := range []string{
		"error: LexError: unterminated string literal at line 1, col 35",
		"error: CompilerError: nope\n  --> stress-out/cases/x.kq:1:1\n",
	} {
		if !hasPosition(s) {
			t.Errorf("no position found in %q", s)
		}
	}
	if !hasPosition("error: StackOverflowError: recursion depth exceeded 8192 at line 1") {
		t.Error("a run-time diagnostic without a column still carries a line number")
	}
	if hasPosition("hello\n") {
		t.Error("program output must not look like a position")
	}
}

// TestPanicDetection checks the Go crash markers the runner treats as PANIC.
func TestPanicDetection(t *testing.T) {
	if !looksLikePanic("panic: runtime error: index out of range [1] with length 0\n\ngoroutine 1 [running]:\n") {
		t.Error("a Go panic must be recognised")
	}
	if !looksLikePanic("runtime: goroutine stack exceeds 1000000000-byte limit\nfatal error: stack overflow\n") {
		t.Error("a Go stack overflow must be recognised")
	}
	if looksLikePanic("error: CompileError: no main function found\n") {
		t.Error("an ordinary diagnostic is not a panic")
	}
}

// TestLoadKnownParsesTheDocumentedFormat checks the list's format: comments, blank lines, tab or space
// separated reasons, and a refusal of entries without one.
func TestLoadKnownParsesTheDocumentedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known.txt")
	body := "# a comment\n\nscale_ident_1m\tinterpreter runs out of memory\nbad_bom lexical rejection, accepted as designed\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, text, err := loadKnown(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("%d entries, want 2: %+v", len(entries), entries)
	}
	if entries[0].Case != "scale_ident_1m" || entries[0].Reason != "interpreter runs out of memory" {
		t.Errorf("tab-separated entry parsed as %+v", entries[0])
	}
	if entries[1].Case != "bad_bom" || !strings.HasPrefix(entries[1].Reason, "lexical rejection") {
		t.Errorf("space-separated entry parsed as %+v", entries[1])
	}
	if !strings.Contains(text, "a comment") {
		t.Error("the raw text must be kept for the report")
	}
	if err := os.WriteFile(path, []byte("scale_ident_1m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadKnown(path); err == nil {
		t.Error("an entry without a reason must be refused")
	}
}

// TestLoadKnownMissingFileIsEmpty checks that a repository without the list still runs the suite.
func TestLoadKnownMissingFileIsEmpty(t *testing.T) {
	entries, text, err := loadKnown(filepath.Join(t.TempDir(), "absent.txt"))
	if err != nil || len(entries) != 0 || text != "" {
		t.Fatalf("missing list: entries=%v text=%q err=%v", entries, text, err)
	}
}

// TestCheckKnownNamesRejectsTypos checks that a list naming an unknown case is an error, since such an
// entry would silently allow-list nothing.
func TestCheckKnownNamesRejectsTypos(t *testing.T) {
	if err := checkKnownNames([]knownEntry{{Case: "scale_ident_1m", Reason: "x"}}); err != nil {
		t.Errorf("a real case name was rejected: %v", err)
	}
	if err := checkKnownNames([]knownEntry{{Case: "scale_ident_1M_typo", Reason: "x"}}); err == nil {
		t.Error("a typo must be refused")
	}
}

// TestUndocumentedFailureNamesEveryNewFinding checks the message a human sees at the end of a run.
func TestUndocumentedFailureNamesEveryNewFinding(t *testing.T) {
	rep := runReport{Cases: []caseResult{
		{Name: "known_case", Verdict: VerdictDiverge, Known: true},
		{Name: "new_case", Verdict: VerdictTimeout},
		{Name: "clean_case", Verdict: VerdictAgree},
	}}
	err := undocumentedFailure(rep)
	if err == nil {
		t.Fatal("a new finding must fail the run")
	}
	if !strings.Contains(err.Error(), "new_case") {
		t.Errorf("the failure does not name the case: %v", err)
	}
	if strings.Contains(err.Error(), "known_case") || strings.Contains(err.Error(), "clean_case") {
		t.Errorf("the failure names cases that are not new findings: %v", err)
	}
	if err := undocumentedFailure(runReport{Cases: rep.Cases[:1]}); err != nil {
		t.Errorf("a fully documented run must pass: %v", err)
	}
}

// TestSummarySeparatesNonFatalDisagreements pins the summary's reporting contract: the cases that can
// fail a run and the wording-only disagreements are printed under their own headings, with truthful
// counts, so a run whose only difference is wording cannot read as one that found defects.
func TestSummarySeparatesNonFatalDisagreements(t *testing.T) {
	rep := runReport{Cases: []caseResult{
		{Name: "clean_case", Verdict: VerdictAgree},
		{Name: "panic_case", Doc: "a deliberate panic", Verdict: VerdictPanic,
			Interp:   engineResult{Front: probeResult{Exit: 0}, Run: probeResult{Exit: 2, Panicked: true, Stderr: "panic: deliberate\n"}, Accepted: true},
			Compiler: engineResult{Front: probeResult{Exit: 0}, Run: probeResult{Exit: 0}, Accepted: true}},
		{Name: "known_case", Doc: "a documented timeout", Verdict: VerdictTimeout, Known: true},
		{Name: "reject_diff_case", Doc: "wording only", Verdict: VerdictRejectDiff,
			Interp:   engineResult{Front: probeResult{Exit: 1, Stderr: "error: LexError: nope at line 1\n"}},
			Compiler: engineResult{Front: probeResult{Exit: 1, Stderr: "error: compiler: nope\n"}}},
	}}
	var b strings.Builder
	printSummary(&b, rep)
	got := b.String()

	if !strings.Contains(got, "4 cases: AGREE 1, PANIC 1, REJECT_DIFF 1, TIMEOUT 1") {
		t.Errorf("the verdict histogram is not truthful:\n%s", got)
	}
	loudAt := strings.Index(got, "2 loud findings:")
	nonFatalAt := strings.Index(got, "1 non-fatal disagreement (both engines refused, wording differs):")
	if loudAt < 0 || nonFatalAt < 0 {
		t.Fatalf("the summary does not carry both headings:\n%s", got)
	}
	if loudAt > nonFatalAt {
		t.Errorf("the loud findings must be printed before the disagreements:\n%s", got)
	}
	if loud := got[loudAt:nonFatalAt]; strings.Contains(loud, "reject_diff_case") {
		t.Errorf("a REJECT_DIFF case is still reported as a loud finding:\n%s", loud)
	}
	if loud := got[loudAt:nonFatalAt]; !strings.Contains(loud, "panic_case") ||
		!strings.Contains(loud, "panic: deliberate") || !strings.Contains(loud, "TIMEOUT (known)") {
		t.Errorf("the loud block lost a case, its diagnostic or its known label:\n%s", loud)
	}
	if nonFatal := got[nonFatalAt:]; !strings.Contains(nonFatal, "reject_diff_case") ||
		!strings.Contains(nonFatal, "error: compiler: nope") {
		t.Errorf("the non-fatal block does not show the disagreement it reports:\n%s", nonFatal)
	}

	// A run whose only result is a wording difference must not announce findings at all.
	var quiet strings.Builder
	printSummary(&quiet, runReport{Cases: []caseResult{rep.Cases[0], rep.Cases[3]}})
	quietText := quiet.String()
	if !strings.Contains(quietText, "no loud findings: every case either agreed") {
		t.Errorf("a run with no loud case must say so:\n%s", quietText)
	}
	if !strings.Contains(quietText, "1 non-fatal disagreement (both engines refused, wording differs):") {
		t.Errorf("a single disagreement must be counted in the singular:\n%s", quietText)
	}
	if strings.Contains(quietText, "loud finding:") {
		t.Errorf("a run with no loud case still prints a loud-findings heading:\n%s", quietText)
	}
}

// TestStaleKnownFailuresAreListed checks that a fixed case is reported instead of silently staying on.
func TestStaleKnownFailuresAreListed(t *testing.T) {
	rep := assembleReport(options{timeout: time.Second, memCapMB: 1024}, []caseResult{
		{Name: "scale_ident_1m", Verdict: VerdictAgree},
	}, []knownEntry{{Case: "scale_ident_1m", Reason: "used to fail"}}, "")
	if len(rep.Stale) != 1 || rep.Stale[0] != "scale_ident_1m" {
		t.Fatalf("stale = %v, want [scale_ident_1m]", rep.Stale)
	}
	if rep.Cases[0].Known {
		t.Error("a case that no longer fails must not be labelled known")
	}
}

// TestCappedBufferKeepsThePrefix checks the output cap and the truncation flag the JSON results carry.
func TestCappedBufferKeepsThePrefix(t *testing.T) {
	var buf cappedBuffer
	chunk := strings.Repeat("x", 4096)
	for i := 0; i < outputCap/len(chunk)+3; i++ {
		if _, err := buf.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(buf.String()); got != outputCap {
		t.Errorf("kept %d bytes, want %d", got, outputCap)
	}
	if !buf.cut() {
		t.Error("the buffer must record that it dropped output")
	}
	var small cappedBuffer
	if _, err := small.Write([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	if small.cut() || small.String() != "hi" {
		t.Error("a small stream must be kept whole and unflagged")
	}
}

// TestChildEnvOverridesInheritedValues checks that an override replaces the inherited variable instead
// of being appended next to it, which would leave the choice to the child's libc.
func TestChildEnvOverridesInheritedValues(t *testing.T) {
	t.Setenv("QK_STRESS_PROBE", "inherited")
	env := childEnv(map[string]string{"QK_STRESS_PROBE": "overridden"})
	count := 0
	for _, kv := range env {
		if strings.HasPrefix(kv, "QK_STRESS_PROBE=") {
			count++
			if kv != "QK_STRESS_PROBE=overridden" {
				t.Errorf("kept the inherited value: %s", kv)
			}
		}
	}
	if count != 1 {
		t.Errorf("QK_STRESS_PROBE appears %d times, want 1", count)
	}
}

// TestSkipSelectsABoundedSubset checks the mechanism a CI job uses to keep its cost predictable: the
// skipped names are absent, everything else is still selected, and a selection that skips everything
// is an error rather than a silently empty run.
func TestSkipSelectsABoundedSubset(t *testing.T) {
	dir := t.TempDir()
	for _, c := range stressgen.All() {
		if err := os.WriteFile(filepath.Join(dir, c.Name+".kq"), c.Source, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	all, err := selectCases(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	subset, err := selectCases(dir, "", "path_fn_10k_params|res_few_mb")
	if err != nil {
		t.Fatal(err)
	}
	if len(subset) != len(all)-2 {
		t.Fatalf("subset has %d cases, want %d", len(subset), len(all)-2)
	}
	for _, c := range subset {
		if c.Name == "path_fn_10k_params" || c.Name == "res_few_mb" {
			t.Errorf("%s was skipped but is still selected", c.Name)
		}
	}
	if _, err := selectCases(dir, "", ".*"); err == nil {
		t.Error("skipping every case must fail rather than report a clean run")
	}
	if _, err := selectCases(dir, "", "["); err == nil {
		t.Error("an invalid -skip pattern must be refused")
	}
}
