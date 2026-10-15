package lang

// The two safety properties of a correction, checked over the repository's own corpus rather than
// asserted: applying a correction is idempotent, and it does not change what the program does.
//
// The design (cmd/qkcheck/README.md) argues both properties from what each rewrite removes, and the
// argument is what decides which diagnostics may carry a correction at all. This file is the half that
// does not depend on the argument being right: it rewrites every runnable file of the corpus, runs the
// result, and compares it with the original byte for byte.
//
// A corpus file that does not compile or does not run is not evidence, and is skipped - but the count
// of files that were compared is asserted, so a change that quietly stops the corpus from running
// fails here instead of turning this file into a test that proves nothing.

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fixCorpusFiles are the trees the corrections are exercised over: the examples a reader starts from,
// the compiler's own cases, and the linter's benchmark corpus (which carries one hand-written file per
// fixable code, so every correction path is reached by something).
func fixCorpusFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, root := range []string{"../../examples", "../../compiler/testdata", "testdata/lintbench"} {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".qk") || strings.HasSuffix(path, ".kq") {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
	sort.Strings(out)
	return out
}

// runProgramForFix compiles and runs src as if it were the file at path, and returns everything a
// reader of the program could observe: its output, and whether it finished without an error. Imports
// resolve against the file's own directory, so a corpus file that imports a sibling still works.
func runProgramForFix(t *testing.T, path, src string) (string, bool) {
	t.Helper()
	prog, err := CompileWithImports(src, path)
	if err != nil {
		return "", false
	}
	var out bytes.Buffer
	runErr := Run(prog, path, nil, strings.NewReader(""), &out)
	return out.String(), runErr == nil
}

// TestLintFixIsIdempotentOverTheCorpus: the second application of the corrections is a no-op, byte for
// byte. A rewrite that kept proposing itself, or that produced a file its own diagnostics describe
// differently, would fail here.
func TestLintFixIsIdempotentOverTheCorpus(t *testing.T) {
	files := fixCorpusFiles(t)
	if len(files) < 50 {
		t.Fatalf("the corpus shrank to %d files: this test would pass without checking anything", len(files))
	}
	fixed, rewritten := 0, 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		first, err := LintSourceReport(src, LintOptions{File: path, OrigSrc: src})
		if err != nil {
			continue // a file that does not parse has no corrections
		}
		if first.Fixed == src {
			continue
		}
		rewritten++
		second, err := LintSourceReport(first.Fixed, LintOptions{File: path, OrigSrc: first.Fixed})
		if err != nil {
			t.Errorf("%s: the corrected file no longer parses: %v", path, err)
			continue
		}
		if second.Fixed != first.Fixed {
			t.Errorf("%s: applying the corrections a second time changes the file\n first: %q\nsecond: %q",
				path, first.Fixed, second.Fixed)
			continue
		}
		fixed++
	}
	t.Logf("corpus %d files, %d rewritten, %d proved idempotent", len(files), rewritten, fixed)
	if rewritten < 3 {
		t.Errorf("only %d corpus files were rewritten: the corpus no longer exercises the corrections", rewritten)
	}
}

// TestLintFixPreservesBehaviourOverTheCorpus: a corrected program produces the same output and the
// same success or failure as the original. This is the property that makes a correction applicable at
// all, so it is checked against the interpreter rather than argued from the rewrite's shape.
func TestLintFixPreservesBehaviourOverTheCorpus(t *testing.T) {
	files := fixCorpusFiles(t)
	compared, rewritten := 0, 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		res, err := LintSourceReport(src, LintOptions{File: path, OrigSrc: src})
		if err != nil || res.Fixed == src {
			continue
		}
		rewritten++
		wantOut, wantOK := runProgramForFix(t, path, src)
		if !wantOK {
			continue // a file that does not run is not evidence about what running it does
		}
		gotOut, gotOK := runProgramForFix(t, path, res.Fixed)
		if !gotOK {
			t.Errorf("%s: the corrected program no longer runs", path)
			continue
		}
		if gotOut != wantOut {
			t.Errorf("%s: the corrected program prints something else\n   before: %q\n    after: %q",
				path, wantOut, gotOut)
			continue
		}
		compared++
	}
	t.Logf("%d runnable corpus files rewritten, %d compared byte for byte", rewritten, compared)
	if compared < 2 {
		t.Errorf("only %d rewritten files actually ran: behaviour preservation is not being checked", compared)
	}
}

// TestLintFixKeepsEveryLineOtherwiseUntouched: a correction rewrites the line it names and leaves
// every other byte of the file alone, line endings included. A Windows checkout stores CRLF, so the
// file's own ending is the one a rewrite must keep - it may never normalise the file to LF.
func TestLintFixKeepsEveryLineOtherwiseUntouched(t *testing.T) {
	files := fixCorpusFiles(t)
	checked := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		for _, tt := range []struct{ name, src string }{
			{"as written", src},
			{"CRLF", crlfSource(src)},
		} {
			res, err := LintSourceReport(tt.src, LintOptions{File: path, OrigSrc: tt.src})
			if err != nil || res.Fixed == tt.src {
				continue
			}
			checked++
			// A deletion takes its own line's terminator with it, so the file loses exactly the
			// separators of the lines the plan removes; every other separator survives unchanged.
			deleted := deletedLines(res)
			wantCRLF := countLines(tt.src, "\r\n")
			if wantCRLF > 0 {
				wantCRLF -= deleted // the file is CRLF, so the lines that went away were CRLF ones
			}
			if got := countLines(res.Fixed, "\r\n"); got != wantCRLF {
				t.Errorf("%s (%s): the rewrite left %d CRLF endings, want %d", path, tt.name, got, wantCRLF)
			}
			// The deleted lines' separators are gone from both counts, so what is left of each kind
			// has to be exactly what the original had of that kind, minus the deletions if that is
			// the kind they were written with.
			wantBare := countLines(tt.src, "\n") - countLines(tt.src, "\r\n")
			if countLines(tt.src, "\r\n") == 0 {
				wantBare -= deleted
			}
			if got := countLines(res.Fixed, "\n") - countLines(res.Fixed, "\r\n"); got != wantBare {
				t.Errorf("%s (%s): the rewrite left %d bare LF endings, want %d", path, tt.name, got, wantBare)
			}
			if got, want := countLines(res.Fixed, "\n"), countLines(tt.src, "\n")-deleted; got != want {
				t.Errorf("%s (%s): the rewritten file has %d line endings, want %d", path, tt.name, got, want)
			}
		}
	}
	t.Logf("%d rewrites checked for line-ending preservation", checked)
	if checked < 3 {
		t.Errorf("only %d rewrites were checked: the corpus no longer exercises this", checked)
	}
}

// countLines counts the occurrences of a separator, so a file's own endings can be compared with the
// endings of the file a rewrite produces.
func countLines(src, sep string) int { return strings.Count(src, sep) }

// crlfSource converts a source's separators to CRLF, which is how a Windows checkout stores the file.
func crlfSource(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	return strings.ReplaceAll(src, "\n", "\r\n")
}

// deletedLines counts the lines the plan removes, whose separators go with them.
func deletedLines(res LintResult) int {
	n := 0
	for _, c := range res.Plan.Lines {
		if c.Delete {
			n++
		}
	}
	return n
}
