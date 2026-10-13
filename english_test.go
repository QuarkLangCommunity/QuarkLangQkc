package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// This file is the standing ratchet against Chinese traces in Go source. The repository's prose is
// English and its user-visible words live in the message catalog, so a Han rune in a comment, in a test's
// failure message or in a literal that ships is a trace of the language the code used to be written in
// rather than a fact about the program. Two things are data and are not traces: the generated catalog,
// whose text is the message table's own values, and the test fixtures, which are programs, paths and
// expected renderings in the language under test.
//
// The check is a ratchet, not a clean sweep: internal/english/allowlist.tsv names the occurrences the
// sweep has not removed yet, with a count per rule and path. A path may carry at most that many, so the
// numbers can only be lowered and a new trace anywhere fails the test. Deleting an entry is a deliberate
// two-step: the count goes to zero first, the entry is removed second.

// traceRule is one class of Chinese trace, and where in a Go file it can appear.
type traceRule struct {
	name string
	// prose is true for a rule that reads comments, false for one that reads string literals.
	prose bool
}

// traceRules are the classes this repository is held to. They are disjoint: one site belongs to exactly
// one rule, so a count in the allowlist says what it is counting.
var traceRules = []traceRule{
	// Han or full-width punctuation inside a comment.
	{name: "han-comment", prose: true},
	// The same runes inside a string literal passed to a testing helper, which is the message a failing
	// run prints.
	{name: "han-test-prose"},
	// The same runes in any other string literal. This is where data legitimately lives, which is why it
	// is the only rule the allowlist permits.
	{name: "han-literal"},
}

// allowlistPathZH names the tracked file holding the occurrences that are not removed yet.
const allowlistPathZH = "internal/english/allowlist.tsv"

// generatedCatalog is the generated message catalog: its literals are the table's data, so only its
// comments are read, and its text is carved out rather than allowlisted.
const generatedCatalog = "internal/i18n/msg_gen.go"

// testingRecv are the receiver names a testing helper is called on in this tree.
var testingRecv = map[string]bool{"t": true, "b": true, "tb": true}

// failureCalls are the testing helpers whose string arguments are prose for a reader of a failure.
var failureCalls = map[string]bool{
	"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true, "Skip": true, "Skipf": true,
	"Log": true, "Logf": true, "Fail": true, "FailNow": true,
}

// carriesTrace reports whether one string holds a Han ideograph or a full-width punctuation mark, which
// are the two shapes a Chinese trace shows up as.
func carriesTrace(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
		if (r >= 0x3000 && r <= 0x303f) || (r >= 0xff01 && r <= 0xff60) || (r >= 0xffe0 && r <= 0xffe6) {
			return true
		}
	}
	return false
}

// TestNoChineseTracesInTrackedGoFiles is the ratchet itself: every rule over every tracked Go file,
// against the counts the allowlist permits.
func TestNoChineseTracesInTrackedGoFiles(t *testing.T) {
	root := repoRoot(t)
	allowed := readTraceAllowlist(t, root)
	actual := map[allowKey]int{}
	for _, path := range trackedFiles(t, root) {
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		hits, err := traceHitsInGoFile(root, path)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		for _, rule := range hits {
			actual[allowKey{rule: rule, path: path}]++
		}
	}
	for key, n := range actual {
		if n > allowed[key] {
			t.Errorf("%s: %s carries %d occurrence(s) and %s allows %d — a Chinese trace in code is prose the repository does not read, or a user-visible word that belongs in internal/i18n/source/*.tsv",
				key.path, key.rule, n, allowlistPathZH, allowed[key])
		}
	}
	// An entry whose occurrences are gone is a number that guards nothing, and leaving it behind would
	// describe a boundary the tree no longer has.
	for key, permit := range allowed {
		if actual[key] == 0 {
			t.Errorf("%s: %s allows %d occurrence(s) of %s and the file now has none — delete the entry, because this ratchet only shrinks",
				allowlistPathZH, key.path, permit, key.rule)
		}
	}
}

// TestChineseTraceRulesMatch is the proof that the ratchet is not vacuous: every rule has to match the
// shape it is written to catch, and to leave ASCII, a Latin-1 accent and the generated catalog alone, so
// a classifier that stopped working fails here instead of reporting zero occurrences forever. The runes
// are written as escapes on purpose: a literal Han rune in this file would itself be a trace.
func TestChineseTraceRulesMatch(t *testing.T) {
	const han = "\u4e00"
	const fullWidthColon = "\uff1a"
	cases := []struct {
		name string
		path string
		src  string
		want []string
	}{
		{"a comment", "a.go", "package p\n\n// " + han + fullWidthColon + " trailing\nfunc f() {}\n", []string{"han-comment"}},
		{"a failure message", "a_test.go", "package p\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {\n\tt.Errorf(\"" + han + "\")\n}\n", []string{"han-test-prose"}},
		{"a fixture", "a_test.go", "package p\n\nvar src = \"io.println(\\\"" + han + "\\\")\"\n", []string{"han-literal"}},
		{"a shipped literal", "a.go", "package p\n\nvar msg = \"" + han + "\"\n", []string{"han-literal"}},
		{"the catalog's own data", generatedCatalog, "package i18n\n\nvar blob = \"" + han + "\"\n", nil},
		{"ascii", "a.go", "package p\n\n// plain english\nvar msg = \"ok\"\n", nil},
		{"a latin accent", "a.go", "package p\n\n// cafe\u0301 latin\nvar msg = \"caf\u00e9\"\n", nil},
	}
	known := map[string]bool{}
	for _, r := range traceRules {
		known[r.name] = true
	}
	proved := map[string]bool{}
	for _, c := range cases {
		hits, err := traceHitsInSource(c.path, []byte(c.src))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		for _, h := range hits {
			if !known[h] {
				t.Errorf("the proof produced rule %q, which traceRules does not define", h)
			}
			proved[h] = true
		}
		if strings.Join(hits, ",") != strings.Join(c.want, ",") {
			t.Errorf("%s: the rules reported %v for %q, want %v", c.name, hits, c.src, c.want)
		}
	}
	for name := range known {
		if !proved[name] {
			t.Errorf("rule %s has no case here, so nothing proves it does anything", name)
		}
	}
}

// TestChineseTraceAllowlistIsWellFormed keeps the ratchet's own input honest: an entry naming a path that
// is no longer tracked, or a rule that does not exist, is a stale permission.
func TestChineseTraceAllowlistIsWellFormed(t *testing.T) {
	root := repoRoot(t)
	tracked := map[string]bool{}
	for _, path := range trackedFiles(t, root) {
		tracked[path] = true
	}
	known := map[string]bool{}
	for _, r := range traceRules {
		known[r.name] = true
	}
	for _, entry := range readTraceAllowlistEntries(t, root) {
		if !known[entry.rule] {
			t.Errorf("%s: %q is not a rule this file checks", allowlistPathZH, entry.rule)
		}
		if !tracked[entry.path] {
			t.Errorf("%s: %q is not a tracked file, so its permission guards nothing", allowlistPathZH, entry.path)
		}
		if entry.count <= 0 {
			t.Errorf("%s: %q allows %d occurrence(s), and a permission of zero is an entry that should be deleted",
				allowlistPathZH, entry.path, entry.count)
		}
	}
}

// traceHitsInGoFile returns the rule of each Chinese site in one tracked Go file, or nothing when the
// file is not there.
func traceHitsInGoFile(root, path string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return nil, nil
	}
	return traceHitsInSource(path, raw)
}

// traceHitsInSource classifies every comment and string literal of one Go source, one entry per site, and
// fails when the source does not parse because a file this ratchet cannot read is a file it cannot clear.
func traceHitsInSource(path string, raw []byte) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, raw, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, cg := range f.Comments {
		if carriesTrace(cg.Text()) {
			out = append(out, "han-comment")
		}
	}
	// The generated catalog's literals are the message table's data rather than text written here.
	if path == generatedCatalog {
		return out, nil
	}
	owned := map[token.Pos]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !failureCalls[sel.Sel.Name] {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok || !testingRecv[recv.Name] {
			return true
		}
		for _, a := range call.Args {
			if bl, ok := a.(*ast.BasicLit); ok && bl.Kind == token.STRING {
				owned[bl.Pos()] = true
			}
		}
		return true
	})
	ast.Inspect(f, func(n ast.Node) bool {
		bl, ok := n.(*ast.BasicLit)
		if !ok || bl.Kind != token.STRING {
			return true
		}
		value, uerr := strconv.Unquote(bl.Value)
		if uerr != nil || !carriesTrace(value) {
			return true
		}
		if owned[bl.Pos()] {
			out = append(out, "han-test-prose")
		} else {
			out = append(out, "han-literal")
		}
		return true
	})
	return out, nil
}

// readTraceAllowlist maps each permission to its count.
func readTraceAllowlist(t *testing.T, root string) map[allowKey]int {
	t.Helper()
	out := map[allowKey]int{}
	for _, e := range readTraceAllowlistEntries(t, root) {
		out[e.allowKey] = e.count
	}
	return out
}

// readTraceAllowlistEntries parses the allowlist: rule, tab, path, tab, count, with # for comments.
func readTraceAllowlistEntries(t *testing.T, root string) []allowEntry {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(allowlistPathZH)))
	if err != nil {
		t.Fatalf("reading %s: %v", allowlistPathZH, err)
	}
	var out []allowEntry
	for i, line := range strings.Split(string(raw), "\n") {
		text := strings.TrimSpace(line)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Split(text, "\t")
		if len(fields) != 3 {
			t.Errorf("%s:%d: want a rule, a path and a count separated by tabs, got %d field(s)", allowlistPathZH, i+1, len(fields))
			continue
		}
		count, cerr := strconv.Atoi(strings.TrimSpace(fields[2]))
		if cerr != nil {
			t.Errorf("%s:%d: %q is not a count", allowlistPathZH, i+1, fields[2])
			continue
		}
		out = append(out, allowEntry{
			allowKey: allowKey{rule: strings.TrimSpace(fields[0]), path: filepath.ToSlash(strings.TrimSpace(fields[1]))},
			count:    count,
		})
	}
	return out
}
