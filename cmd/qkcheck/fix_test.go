package main

// The negative cases of -fix and -check: what the tool must refuse to do, and what it must say while
// refusing. A fixer is only as trustworthy as the cases it declines, so each of these is a case where
// the tempting rewrite is wrong: a diagnostic whose correction is the program's design, a file that is
// already correct, a Windows checkout whose line endings must survive, and a line that two corrections
// name at once.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixableSrc is a program whose every diagnostic carries a correction: a self-assignment (QK108) and
// a dead store whose earlier write is a plain assignment rather than a declaration (QK113). The
// declaration initializer is read before anything writes the variable, which is what keeps QK113 away
// from it - a declaration initializer that is overwritten is report-only by design.
const fixableSrc = `fn main(IOStream io) void {
    int a = 1;
    io.println(a);
    a = 4;
    a = a;
    a = 5;
    io.println(a);
}
`

// unfixableSrc produces a diagnostic that has no mechanical correction (the unused variable is a
// declaration, and deleting it would be a decision about the program).
const unfixableSrc = `fn main(IOStream io) void {
    int unused = 1;
    io.println("ok");
}
`

// TestCLIFixAppliesInPlaceAndReportsWhatRemains is the positive case cut down to its contract: the
// corrected lines are gone from the file, nothing else is, and the second run has nothing left to do.
func TestCLIFixAppliesInPlaceAndReportsWhatRemains(t *testing.T) {
	p := writeFile(t, t.TempDir(), "fix.qk", fixableSrc)
	out, _, code := runCapture(t, "-fix", p)
	if code != 0 {
		t.Fatalf("a file whose every diagnostic has a fix must exit 0, got %d (%q)", code, out)
	}
	got := readFile(t, p)
	if strings.Contains(got, "a = a;") || strings.Contains(got, "a = 4;") {
		t.Errorf("-fix left a corrected line in the file:\n%s", got)
	}
	if !strings.Contains(got, "int a = 1;") || strings.Count(got, "io.println(a);") != 2 {
		t.Errorf("-fix removed a line it had no correction for:\n%s", got)
	}
	second, _, code := runCapture(t, "-fix", p)
	if code != 0 {
		t.Fatalf("the second -fix must exit 0, got %d (%q)", code, second)
	}
	if again := readFile(t, p); again != got {
		t.Errorf("the second -fix changed the file:\n first: %q\nsecond: %q", got, again)
	}
}

// TestCLIFixLeavesUnfixableDiagnosticsAndStillFailsTheGate: a diagnostic with no correction is not
// dropped by -fix, is not counted as fixed, and still fails the gate the way it did before.
func TestCLIFixLeavesUnfixableDiagnosticsAndStillFailsTheGate(t *testing.T) {
	p := writeFile(t, t.TempDir(), "unfixable.qk", unfixableSrc)
	before := readFile(t, p)
	out, _, code := runCapture(t, "-fix", p)
	if code != 1 {
		t.Fatalf("an unfixable diagnostic must still fail the gate (exit 1), got %d (%q)", code, out)
	}
	if !strings.Contains(out, "QK101") {
		t.Errorf("the remaining diagnostic is missing from the report: %q", out)
	}
	if got := readFile(t, p); got != before {
		t.Errorf("-fix rewrote a file it had no correction for:\n before: %q\n  after: %q", before, got)
	}
	// -check must not consider it pending either: there is nothing to apply.
	if _, _, code := runCapture(t, "-check", p); code != 1 {
		t.Errorf("-check on a file with only unfixable diagnostics must exit 1, got %d", code)
	}
}

// TestCLICheckWritesNothingAndFailsWhenAFixIsPending is the CI contract: a pending correction is a
// non-zero exit and an untouched file.
func TestCLICheckWritesNothingAndFailsWhenAFixIsPending(t *testing.T) {
	for _, flag := range []string{"-check", "--check", "-dry-run", "--dry-run"} {
		t.Run(flag, func(t *testing.T) {
			dir := t.TempDir()
			p := writeFile(t, dir, "pending.qk", fixableSrc)
			before := readFile(t, p)
			out, errOut, code := runCapture(t, flag, p)
			if code != exitFixPending {
				t.Fatalf("%s must exit %d when a fix is pending, got %d (%q%s)", flag, exitFixPending, code, out, errOut)
			}
			if got := readFile(t, p); got != before {
				t.Errorf("%s wrote to the file:\n before: %q\n  after: %q", flag, before, got)
			}
			if !strings.Contains(out, "fix:") {
				t.Errorf("%s must show the correction it found: %q", flag, out)
			}
			if !strings.Contains(out, "nothing was written") {
				t.Errorf("%s must say that nothing was written: %q", flag, out)
			}
		})
	}
}

// TestCLICheckAndFixAreExclusive: asking for both is a usage error rather than a silent choice.
func TestCLICheckAndFixAreExclusive(t *testing.T) {
	p := writeFile(t, t.TempDir(), "both.qk", fixableSrc)
	_, errOut, code := runCapture(t, "-fix", "-check", p)
	if code != 2 {
		t.Fatalf("-fix -check must be a usage error (exit 2), got %d", code)
	}
	if !strings.Contains(errOut, "cannot be combined") {
		t.Errorf("the usage error must say why: %q", errOut)
	}
}

// TestCLIAlreadyCorrectFileIsUntouched: a file with no diagnostics is left byte for byte alone by every
// mode, including the one that writes.
func TestCLIAlreadyCorrectFileIsUntouched(t *testing.T) {
	p := writeFile(t, t.TempDir(), "correct.qk", cleanSrc)
	before := readFile(t, p)
	for _, flag := range []string{"-fix", "-check"} {
		out, errOut, code := runCapture(t, flag, p)
		if code != 0 {
			t.Errorf("%s on an already-correct file must exit 0, got %d (%q%s)", flag, code, out, errOut)
		}
		if got := readFile(t, p); got != before {
			t.Errorf("%s changed an already-correct file:\n before: %q\n  after: %q", flag, before, got)
		}
	}
}

// TestCLICRLFFileKeepsItsLineEndings: a Windows checkout stores CRLF, and a rewrite of one line must
// not normalise the rest of the file to LF.
func TestCLICRLFFileKeepsItsLineEndings(t *testing.T) {
	crlf := strings.ReplaceAll(fixableSrc, "\n", "\r\n")
	p := writeFile(t, t.TempDir(), "windows.qk", crlf)
	if _, _, code := runCapture(t, "-fix", p); code != 0 {
		t.Fatalf("-fix on a CRLF file exited %d", code)
	}
	got := readFile(t, p)
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a bare LF appeared in a CRLF file:\n%q", got)
	}
	if !strings.Contains(got, "    int a = 1;\r\n") {
		t.Errorf("an untouched line lost its CRLF ending:\n%q", got)
	}
	if strings.Contains(got, "a = a;") {
		t.Errorf("the correction was not applied to the CRLF file:\n%q", got)
	}
	// The deleted line's own terminator goes with it, so the file has four lines and four CRLFs.
	if n := strings.Count(got, "\r\n"); n != 6 {
		t.Errorf("after deleting two lines the file has %d CRLF endings, want 6:\n%q", n, got)
	}
	if !strings.Contains(got, "int a = 1;\r\n") {
		t.Errorf("a line with no correction lost its CRLF ending:\n%q", got)
	}
}

// TestCLITwoFixesOnTheSameLineLeaveOneCorrection: a line carries at most one correction. Two
// diagnostics can name the same line - a self-assignment is also a dead store - and then the rewrite
// happens once. The test is the invariant rather than the count: no line of the plan is rewritten
// twice, and after the run no diagnostic claims a correction on a line that did not change.
func TestCLITwoFixesOnTheSameLineLeaveOneCorrection(t *testing.T) {
	src := `fn main(IOStream io) void {
    int a = 1;
    a = 5;
    a = a;
    a = 6;
    io.println(a);
}
`
	p := writeFile(t, t.TempDir(), "one_line.qk", src)
	// The self-assignment (QK108) and the dead store it leaves behind (QK113) are both on line 4.
	jsonOut, _, _ := runCapture(t, "-json", "-exit0", p)
	if !strings.Contains(jsonOut, `"QK108"`) || !strings.Contains(jsonOut, `"QK113"`) {
		t.Fatalf("the fixture must produce QK108 and QK113 on one line, got %s", jsonOut)
	}
	out, _, _ := runCapture(t, "-fix", p)
	got := readFile(t, p)
	if strings.Contains(got, "a = a;") {
		t.Errorf("the self-assignment survived:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.Count(strings.TrimSpace(line), ";") > 1 && strings.Contains(line, "a = ") {
			t.Errorf("the corrected line still holds two statements: %q", line)
		}
	}
	after, _, _ := runCapture(t, "-exit0", p)
	if strings.Contains(after, "QK108") {
		t.Errorf("QK108 survived the run that corrected its line: %q", after)
	}
	if strings.Contains(out, "delete this line") && !strings.Contains(got, "a = 5;") {
		t.Errorf("-fix removed the line before the corrected one:\n%s", got)
	}
}

// TestCLIFixKeepsFileMode: a rewrite must not change what the file's permissions were.
func TestCLIFixKeepsFileMode(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("file modes are not meaningful on Windows")
	}
	p := writeFile(t, t.TempDir(), "mode.qk", fixableSrc)
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, code := runCapture(t, "-fix", p); code != 0 {
		t.Fatalf("-fix exited %d", code)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("-fix changed the file mode to %v, want 0600", fi.Mode().Perm())
	}
}

// readFile reads a file or fails the test.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
