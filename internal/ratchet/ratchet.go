// Package ratchet is the one implementation behind this repository's shrink-only gates.
//
// A ratchet is a set of rules over every tracked file, plus an allowlist that names the occurrences not
// removed yet with a count per rule and path. A path may carry at most that many, so the numbers can
// only be lowered, and an entry whose occurrences are gone is an error rather than a stale line. Two
// gates use it — no Chinese trace in Go source, and no personal data anywhere — and they are two rule
// sets rather than two implementations for the same reason a linter has one engine and many checks.
package ratchet

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Rule is one class of forbidden content. Rules within a set are disjoint: one site belongs to exactly
// one rule, so a count in the allowlist says what it is counting.
type Rule interface {
	// Name is the identifier an allowlist entry and a proof case refer to.
	Name() string
	// Reads reports whether this rule reads the file at path at all. A rule set that read every file
	// with every rule would parse Go source once per rule instead of once.
	Reads(path string) bool
	// Scan returns one Hit per occurrence in one file.
	Scan(f *File) []Hit
}

// Hit is one occurrence: the rule that found it, and where.
type Hit struct {
	Rule string
	Path string
}

// File is one tracked file presented to a rule set, with the result of parsing it as Go source.
type File struct {
	// Path is where the file sits, relative to the repository root and slash-separated.
	Path string
	// Raw is the file's text, or nil when it is not there or is binary.
	Raw []byte
	// Go is the parsed file and its token set, and ParseErr says why it is not there. The two travel
	// together because a rule set that reads syntax has to distinguish "not Go source" from "Go source
	// this build cannot read": a file a gate cannot read is a file it cannot clear.
	Go       *GoFile
	ParseErr error
}

// GoFile is a parsed Go source, read once for every rule that has something to ask of it: its token
// set, its string literals already unquoted, and the positions of the testing helper arguments whose
// text a reader of a failure message sees.
type GoFile struct {
	File       *ast.File
	Fset       *token.FileSet
	Unquote    map[token.Pos]string
	TestProse  map[token.Pos]bool
	HasComment bool
}

// NewFile reads raw as Go source when path names one, keeping a parse failure for the rule set to
// report rather than dropping the file.
func NewFile(path string, raw []byte) *File {
	f := &File{Path: path, Raw: raw}
	if raw == nil || !strings.HasSuffix(path, ".go") {
		return f
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, raw, parser.ParseComments)
	if err != nil {
		f.ParseErr = err
		return f
	}
	f.Go = &GoFile{
		File:       parsed,
		Fset:       fset,
		Unquote:    map[token.Pos]string{},
		TestProse:  map[token.Pos]bool{},
		HasComment: len(parsed.Comments) > 0,
	}
	ast.Inspect(parsed, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.BasicLit:
			if t.Kind == token.STRING {
				if value, err := strconv.Unquote(t.Value); err == nil {
					f.Go.Unquote[t.Pos()] = value
				}
			}
		case *ast.CallExpr:
			sel, ok := t.Fun.(*ast.SelectorExpr)
			if !ok || !testingHelpers[sel.Sel.Name] {
				return true
			}
			recv, ok := sel.X.(*ast.Ident)
			if !ok || !testingReceivers[recv.Name] {
				return true
			}
			for _, a := range t.Args {
				if lit, ok := a.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					f.Go.TestProse[lit.Pos()] = true
				}
			}
		}
		return true
	})
	return f
}

// testingReceivers are the receiver names a Go testing helper is called on.
var testingReceivers = map[string]bool{"t": true, "b": true, "tb": true}

// testingHelpers are the helpers whose string arguments are prose for a reader of a failure.
var testingHelpers = map[string]bool{
	"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true, "Skip": true, "Skipf": true,
	"Log": true, "Logf": true, "Fail": true, "FailNow": true,
}

// Lines splits the file's text into lines. A rule set that reads prose rather than syntax works here.
func (f *File) Lines() []string { return strings.Split(string(f.Raw), "\n") }

// Spec is one gate: its name for failure messages, why it exists, the rules, the allowlist holding the
// occurrences that are not removed yet, and the proof cases that show the rules still match.
type Spec struct {
	Name      string
	Reason    string
	Allowlist string
	Proofs    string
	Rules     []Rule
}

// Entry is one line of an allowlist: how many occurrences of one rule one path may carry.
type Entry struct {
	Rule  string
	Path  string
	Count int
}

// Verify applies every rule of the spec to every tracked file and fails the test when a path carries
// more occurrences than its allowlist entry permits, or when an entry guards nothing.
func Verify(t *testing.T, spec Spec) {
	t.Helper()
	root := RepoRoot(t)
	allowed := ReadAllowlist(t, spec.Allowlist)
	var hits []Hit
	for _, path := range TrackedFiles(t, root) {
		file := NewFile(path, ReadText(root, path))
		for _, rule := range spec.Rules {
			if !rule.Reads(path) {
				continue
			}
			// A file the rule set reads as syntax and this build cannot parse is a file the gate cannot
			// clear, so it is reported rather than skipped.
			if file.Go == nil && file.ParseErr != nil {
				t.Errorf("%s: %v", path, file.ParseErr)
			}
			hits = append(hits, rule.Scan(file)...)
		}
	}
	actual := Tally(hits)
	for _, key := range sortedKeys(actual) {
		if actual[key] > allowed[key] {
			t.Errorf("%s: %s carries %d occurrence(s) and %s allows %d — %s",
				key.Path, key.Rule, actual[key], spec.Allowlist, allowed[key], spec.Reason)
		}
	}
	// An entry whose occurrences are gone is a number that guards nothing, and leaving it behind would
	// describe a boundary the tree no longer has.
	for _, key := range sortedKeys(allowed) {
		if actual[key] == 0 {
			t.Errorf("%s: %s allows %d occurrence(s) of %s and the file now has none — delete the entry, because this ratchet only shrinks",
				spec.Allowlist, key.Path, allowed[key], key.Rule)
		}
	}
}

// ProofCase is one line of testdata/rules.tsv: a rule, whether it must match, and the text to try.
type ProofCase struct {
	Rule string
	Want string
	Text string
}

// ProveRules fails the test unless every rule matches the shape it is written to catch and leaves the
// texts it must leave alone, and unless every rule has a case. A rule that matches nothing would report
// zero occurrences forever, which is how a gate turns green for a tree it never read.
func ProveRules(t *testing.T, spec Spec, apply func(Rule, *File) bool) {
	t.Helper()
	byName := map[string]Rule{}
	for _, r := range spec.Rules {
		byName[r.Name()] = r
	}
	proved := map[string]bool{}
	for _, c := range ReadProofCases(t, spec.Proofs) {
		r, ok := byName[c.Rule]
		if !ok {
			t.Errorf("the proof covers rule %q, which this rule set does not define", c.Rule)
			continue
		}
		proved[c.Rule] = true
		// The proof case says where the shape sits: a `// path: <file> ` header names the file the text
		// is read as, and a case without one is Go source, which is the stricter reading. A rule that
		// does not read that file has nothing to say about it, which is how the deliberate exemptions
		// are proved as well as the matches.
		path := proofPath(c.Text)
		file := NewFile(path, []byte(proofText(c.Text)))
		if !r.Reads(path) {
			// A rule that does not read the file makes no hit, which is the exemption an `ignores` case
			// is there to prove. A `catches` case it does not read proves nothing at all.
			if c.Want == "catches" {
				t.Errorf("rule %s does not read %s, so the case for %q proves nothing", c.Rule, path, c.Text)
			}
			continue
		}
		switch c.Want {
		case "catches":
			if !apply(r, file) {
				t.Errorf("rule %s does not match %q, so it reports no occurrences for content it is written to catch", c.Rule, c.Text)
			}
		case "ignores":
			if apply(r, file) {
				t.Errorf("rule %s matches %q, which is not what it forbids — a rule that fires on an agreement is noise, and noise is how a check like this gets switched off", c.Rule, c.Text)
			}
		default:
			t.Errorf("%s: %q is neither catches nor ignores", spec.Proofs, c.Want)
		}
	}
	for name := range byName {
		if !proved[name] {
			t.Errorf("rule %s has no case in %s, so nothing proves it does anything", name, spec.Proofs)
		}
	}
}

// proofPath is the file a proof case is tried as: a header `// path: a.go` names one, and a case without
// one is Go source, which is the stricter reading.
func proofPath(text string) string {
	const marker = "// path: "
	if after, found := strings.CutPrefix(text, marker); found {
		if path, _, ok := strings.Cut(after, " "); ok {
			return path
		}
		return after
	}
	return "a.go"
}

// proofText strips the header a proof case may carry.
func proofText(text string) string {
	const marker = "// path: "
	if after, found := strings.CutPrefix(text, marker); found {
		if _, rest, ok := strings.Cut(after, " "); ok {
			return rest
		}
	}
	return text
}

// CheckAllowlist keeps the gate's own input honest: an entry naming a path that is no longer tracked, or
// a rule that does not exist, is a stale permission.
func CheckAllowlist(t *testing.T, spec Spec) {
	t.Helper()
	tracked := map[string]bool{}
	for _, path := range TrackedFiles(t, RepoRoot(t)) {
		tracked[path] = true
	}
	known := map[string]bool{}
	for _, r := range spec.Rules {
		known[r.Name()] = true
	}
	for _, entry := range ReadAllowlistEntries(t, spec.Allowlist) {
		if !known[entry.Rule] {
			t.Errorf("%s: %q is not a rule this gate checks", spec.Allowlist, entry.Rule)
		}
		if !tracked[entry.Path] {
			t.Errorf("%s: %q is not a tracked file, so its permission guards nothing", spec.Allowlist, entry.Path)
		}
		if entry.Count <= 0 {
			t.Errorf("%s: %q allows %d occurrence(s), and a permission of zero is an entry that should be deleted",
				spec.Allowlist, entry.Path, entry.Count)
		}
	}
}

// RepoRoot is the checkout the test runs in. A linked worktree has a .git file rather than a directory,
// so the root is asked of git instead of being inferred from the layout.
func RepoRoot(t *testing.T) string {
	t.Helper()
	root := gitOutput(t, ".", "rev-parse", "--show-toplevel")
	if root == "" {
		t.Fatal("git reported no worktree root, so the ratchet cannot tell what is published")
	}
	return filepath.FromSlash(root)
}

// TrackedFiles is every file git would publish, which is the tree a ratchet is about, less its own
// fixtures: a proof case has to spell out the shape it detects, so a fixture directory is read by the
// proof and by nothing else.
func TrackedFiles(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	for _, p := range strings.Split(gitOutput(t, root, "ls-files", "-z"), "\x00") {
		if p == "" || strings.Contains("/"+p, "/testdata/") {
			continue
		}
		paths = append(paths, filepath.ToSlash(p))
	}
	if len(paths) == 0 {
		t.Fatal("git listed no tracked files, so the ratchet would pass without reading anything")
	}
	return paths
}

// ReadAllowlist maps each permission to its count.
func ReadAllowlist(t *testing.T, path string) map[Entry]int {
	t.Helper()
	out := map[Entry]int{}
	for _, e := range ReadAllowlistEntries(t, path) {
		out[Entry{Rule: e.Rule, Path: e.Path}] = e.Count
	}
	return out
}

// ReadAllowlistEntries parses an allowlist: rule, tab, path, tab, count, with # for comments.
func ReadAllowlistEntries(t *testing.T, path string) []Entry {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(RepoRoot(t), filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var out []Entry
	for i, line := range strings.Split(string(raw), "\n") {
		text := strings.TrimSpace(line)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Split(text, "\t")
		if len(fields) != 3 {
			t.Errorf("%s:%d: want a rule, a path and a count separated by tabs, got %d field(s)", path, i+1, len(fields))
			continue
		}
		count, cerr := strconv.Atoi(strings.TrimSpace(fields[2]))
		if cerr != nil {
			t.Errorf("%s:%d: %q is not a count", path, i+1, fields[2])
			continue
		}
		out = append(out, Entry{
			Rule:  strings.TrimSpace(fields[0]),
			Path:  filepath.ToSlash(strings.TrimSpace(fields[1])),
			Count: count,
		})
	}
	return out
}

// ReadProofCases parses the rule proofs, which are tab-separated: rule, expectation, text. A case is one
// line, so a newline inside one is written `\n` and a tab `\t`, and the text is the last field.
func ReadProofCases(t *testing.T, path string) []ProofCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(RepoRoot(t), filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("reading the rule proof cases: %v", err)
	}
	var out []ProofCase
	for i, line := range strings.Split(string(raw), "\n") {
		if text := strings.TrimSpace(line); text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		parts := strings.SplitN(strings.TrimRight(line, "\r"), "\t", 3)
		if len(parts) != 3 {
			t.Errorf("%s:%d: want rule, expectation and text separated by tabs, got %d field(s)", path, i+1, len(parts))
			continue
		}
		out = append(out, ProofCase{Rule: parts[0], Want: parts[1], Text: unescape(parts[2])})
	}
	if len(out) == 0 {
		t.Fatal("the proof cases are empty, so nothing proves the rules match anything")
	}
	return out
}

// unescape turns the line breaks and tabs a proof case writes back into characters.
func unescape(s string) string {
	return strings.NewReplacer(`\n`, "\n", `\t`, "\t").Replace(s)
}

// ReadText returns one tracked file's content, or nil when it is not there or holds a NUL byte — a
// binary file is not prose and has no lines to read.
func ReadText(root, path string) []byte {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil || bytes.IndexByte(raw, 0) >= 0 {
		return nil
	}
	return raw
}

// Tally counts hits per rule.
func Tally(hits []Hit) map[Entry]int {
	out := map[Entry]int{}
	for _, h := range hits {
		out[Entry{Rule: h.Rule, Path: h.Path}]++
	}
	return out
}

// sortedKeys returns the entries of a count map in a stable order, so a failure reads the same every run.
func sortedKeys[V any](m map[Entry]V) []Entry {
	out := make([]Entry, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Rule < out[j].Rule
	})
	return out
}

// gitOutput runs one git command in dir and returns its trimmed stdout, failing the test when git cannot
// answer.
func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String())
}
