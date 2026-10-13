package lang

// qkcheck corpus perturbation invariance: after semantics-preserving rewrites of **real source files**, the diagnostic set must stay identical.
//
// Two real risks are covered:
//  1. Position robustness: after inserting comments/blank lines the diagnostic line numbers must shift exactly (both editors and CI rely on this to locate them);
//  2. Cross-system: CRLF (Windows line endings) and trailing whitespace must not change any diagnostic - the core test case for cross-platform consistency.
//
// Complementary to lintbench (hand-labelled): that side checks that what should be reported is reported and what should not be reported is not,
// this side checks that the same code written differently gives the same verdict.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type diagKey struct {
	code string
	line int
}

func diagSet(t *testing.T, src string) []diagKey {
	t.Helper()
	prog, err := ParseSource(src)
	if err != nil {
		t.Fatalf("parsing failed: %v", err)
	}
	var out []diagKey
	for _, d := range Lint(prog, LintOptions{}) {
		out = append(out, diagKey{d.Code, d.Pos.Line})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].line != out[j].line {
			return out[i].line < out[j].line
		}
		return out[i].code < out[j].code
	})
	return out
}

func sameDiags(a, b []diagKey) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func corpusFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	root := filepath.Join("..", "..")
	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if skipNestedRepo(p, fi, root) {
			return filepath.SkipDir
		}
		if err != nil {
			return err
		}
		if fi.IsDir() {
			switch fi.Name() {
			case ".git", "dist", "dist-ci", "node_modules", "lintbench", "bench", "stress-out":
				// lintbench is the hand-labelled benchmark (covered by the lintstat/lintbench tests);
				// bench/tools is the performance fixture (contains copies of third-party libraries);
				// stress-out is the generated extreme-input corpus (cmd/qkstress gen), written by the stress
				// gate into an untracked tree: it holds adversarial sources by construction, and descending
				// into it made this walker depend on whether the stress gate had run first in the same
				// checkout (a generated multi-megabyte case overflows the parser's stack).
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".qk") || strings.HasSuffix(p, ".kq") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func TestLintCorpusPerturbationInvariance(t *testing.T) {
	files := corpusFiles(t)
	if len(files) < 5 {
		t.Skipf("too few corpus files (%d)", len(files))
	}
	checked := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		if _, err := ParseSource(src); err != nil {
			t.Logf("skipping the unparsable file %s: %v", f, err)
			continue
		}
		base := diagSet(t, src)

		// 1) CRLF (Windows line endings): diagnostics must be identical item by item
		if got := diagSet(t, strings.ReplaceAll(src, "\n", "\r\n")); !sameDiags(got, base) {
			t.Errorf("%s: the diagnostics changed under CRLF\n base=%v\n  crlf=%v", f, base, got)
		}
		// 2) Trailing comments: change no diagnostic (comments are stripped by the lexer, positions unmoved)
		var withComments []string
		for _, ln := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
			if strings.TrimSpace(ln) == "" || strings.Contains(ln, "//") || strings.Contains(ln, "/*") {
				withComments = append(withComments, ln)
				continue
			}
			withComments = append(withComments, ln+" // 扰动注释")
		}
		if got := diagSet(t, strings.Join(withComments, "\n")+"\n"); !sameDiags(got, base) {
			t.Errorf("%s: the diagnostics changed after adding a trailing comment\n base=%v\n  pert=%v", f, base, got)
		}
		// 3) Inserting 3 comment lines at the head: every diagnostic line number shifts by +3
		head := "// 扰动头部 1\n// 扰动头部 2\n// 扰动头部 3\n" + src
		want := make([]diagKey, len(base))
		for i, d := range base {
			want[i] = diagKey{d.code, d.line + 3}
		}
		if got := diagSet(t, head); !sameDiags(got, want) {
			t.Errorf("%s: the line numbers did not shift as expected after inserting a comment at the top\n want=%v\n  got=%v", f, want, got)
		}
		// 4) Trailing whitespace + blank lines at the end of the file: diagnostics unchanged
		var padded []string
		for _, ln := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
			padded = append(padded, ln+"   \t")
		}
		if got := diagSet(t, strings.Join(padded, "\n")+"\n\n\n"); !sameDiags(got, base) {
			t.Errorf("%s: the diagnostics changed after trailing whitespace or a final blank line\n base=%v\n  pert=%v", f, base, got)
		}
		checked++
	}
	t.Logf("corpus perturbation invariance holds: %d files x 4 perturbations (CRLF / trailing comment / inserted line at the top / trailing whitespace)", checked)
	if checked < 5 {
		t.Errorf("too few files were actually verified: %d", checked)
	}
}

// TestLintCRLFPositions single-point check: under CRLF the token line/column numbers are exactly the same as under LF (cross-system consistency).
func TestLintCRLFPositions(t *testing.T) {
	src := "program library;\n\npub fn f(int a) int {\n    int unused = 1;\n    return a;\n}\n"
	lf := diagSet(t, src)
	crlf := diagSet(t, strings.ReplaceAll(src, "\n", "\r\n"))
	if !sameDiags(lf, crlf) {
		t.Fatalf("the diagnostics differ between CRLF and LF: lf=%v crlf=%v", lf, crlf)
	}
	if len(lf) != 1 || lf[0].code != CodeUnusedVar || lf[0].line != 4 {
		t.Fatalf("QK101 was expected on line 4, got %v", lf)
	}
	// Also check the lexer layer: CRLF produces no extra tokens and line/column numbers match
	toksLF, err := Lex(src)
	if err != nil {
		t.Fatal(err)
	}
	toksCRLF, err := Lex(strings.ReplaceAll(src, "\n", "\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(toksLF) != len(toksCRLF) {
		t.Fatalf("the token count differs: LF=%d CRLF=%d", len(toksLF), len(toksCRLF))
	}
	for i := range toksLF {
		if toksLF[i].Kind != toksCRLF[i].Kind || toksLF[i].Line != toksCRLF[i].Line || toksLF[i].Col != toksCRLF[i].Col {
			t.Fatalf("token %d differs under CRLF: LF=%+v CRLF=%+v", i, toksLF[i], toksCRLF[i])
		}
	}
}

// skipNestedRepo reports whether a walked directory belongs to a different repository. The QuarkLang
// family keeps its sibling projects inside this workspace, each with its own .git, and another
// repository's files are not this project's files: walking into them would let their sources decide
// this project's gates.
func skipNestedRepo(path string, info os.FileInfo, root string) bool {
	if info == nil || !info.IsDir() || path == root {
		return false
	}
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// TestCorpusWalkSkipsGeneratedTrees pins the walker's skip list against the tree the stress gate writes:
// a program planted in stress-out/ must not be reached, because otherwise the corpus this test measures
// would depend on whether the stress gate ran first in the same checkout.
func TestCorpusWalkSkipsGeneratedTrees(t *testing.T) {
	root := filepath.Join("..", "..")
	probeDir := filepath.Join(root, "stress-out", "corpus-walk-probe")
	probe := filepath.Join(probeDir, "probe.kq")
	if err := os.MkdirAll(probeDir, 0o755); err != nil {
		t.Fatalf("cannot plant the probe: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(probeDir)
		os.Remove(filepath.Join(root, "stress-out")) // only succeeds while the generated tree stayed empty
	})
	if err := os.WriteFile(probe, []byte("program library;\n\nfn main() int { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("cannot write the probe: %v", err)
	}
	for _, f := range corpusFiles(t) {
		if filepath.ToSlash(f) == filepath.ToSlash(probe) {
			t.Fatalf("the walk reached a program under stress-out/: %s", f)
		}
	}
}
