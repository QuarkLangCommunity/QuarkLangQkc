package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// This file is the standing ratchet against personal data in the repository. A public repository is a
// published artefact: a home directory in a tracked file tells every reader the account name, the
// directory layout and the machine of whoever generated it, and anyone on the network can read it
// without authenticating. The rules below are the classes of that data that were actually found in
// this tree — an account name in a home directory, a Windows home directory, an absolute path into a
// machine-local cache, and the owner's machine name in prose — and they are checked against every
// tracked file, because a scratch file that is not tracked is not published and a tracked one is.
//
// The check is a ratchet, not a clean sweep: internal/privacy/allowlist.tsv names the occurrences the
// sweep has not removed yet, with a count per rule and path. A path may carry at most that many, so
// the numbers can only be lowered and a new occurrence anywhere fails the test. Deleting an entry is
// a deliberate two-step: the count goes to zero first, the entry is removed second.

// personalRule is one class of personal data, and where in a file it can appear.
type personalRule struct {
	name string
	// pattern matches one occurrence of the rule in a line. When it has a capture group, the first
	// non-empty group is the account or host the hit is about, and a conventional placeholder is not a
	// hit.
	pattern *regexp.Regexp
	// handle is a public identifier the rule must not fire inside, empty when the rule has none.
	handle string
	// proseOnly limits the rule to prose files: a Go identifier or a test fixture may legitimately
	// contain the same letters as a machine name, so code is only read for the path-shaped rules.
	proseOnly bool
	// placeholder is true when a documented example may use a conventional account name, which the
	// rule then skips.
	placeholder bool
}

// personalRules are the classes this repository is held to. They are patterns rather than a list of
// forbidden strings, so renaming the account or the machine does not silently empty the check, and
// they are disjoint: one occurrence belongs to exactly one rule, so a count in the allowlist says
// what it is counting.
var personalRules = []personalRule{
	// The home directory of a Windows path, which is written with a drive letter and either separator,
	// or — as this tree writes it — with a leading slash before the drive letter. This rule is first
	// because it owns `/Users/`, so that a Windows path and a macOS one are counted by one rule and the
	// POSIX rule below never sees either of them twice.
	{name: "windows-home-dir", pattern: regexp.MustCompile(`(?i)(?:[A-Za-z]:|/)[\\/]Users[\\/]([A-Za-z0-9._-]+)[\\/]`), placeholder: true},
	// An account name in a home directory. The boundary keeps a drive-letter path out of it, so that
	// `C:/Users/account/...` is counted once, by the rule above.
	{name: "home-dir", pattern: regexp.MustCompile(`(?:^|[^A-Za-z0-9:])/home/([A-Za-z0-9._-]+)/`), placeholder: true},
	// An absolute path into a machine-local cache that is not already inside a counted home directory:
	// the cache directory is chosen by the build tool of the machine that ran it, so the path names
	// that machine's layout. A path under a home directory is left to the rules above, so that one
	// occurrence is counted once and a documented `/home/account/.cache/...` example stays quiet. A
	// script that derives its scratch directory from `$HOME` names no machine at all, and neither does
	// `/tmp`, which is the same directory on every POSIX host — which is why CI configuration uses it.
	{name: "abs-cache", pattern: regexp.MustCompile(`(?:^|[^A-Za-z0-9_$/])(/(?:[A-Za-z0-9._-]+/)*\.cache(?:/[A-Za-z0-9._-]+)*)`)},
	// The owner's machine name, in prose only. The maintainer's public handle is attribution rather
	// than private data, so the handle is not this rule. The rule cannot be trusted in code, where the
	// same letters can be an identifier.
	{name: "machine-name", pattern: regexp.MustCompile(`(?i)\benoch\b`), proseOnly: true, handle: `\benoch-199811\b`},
}

// handlePatterns keep a public handle out of a rule: the handle names a maintainer, which is
// attribution rather than private data, and a rule that fires on it would be noise.
var handlePatterns = map[string]*regexp.Regexp{
	"machine-name": regexp.MustCompile(`(?i)\benoch-199811\b`),
}

// placeholderAccount are the account names a document may use when it has to show a home directory or
// a cache path in an example. They are the conventional stand-ins rather than anybody's account, so a
// document that uses one is showing the shape of a path, not a machine.
var placeholderAccount = map[string]bool{
	"user": true, "you": true, "youruser": true, "username": true, "name": true,
	"example": true, "me": true, "someone": true, "account": true, "owner": true,
}

// proseExt are the file kinds a prose-only rule is looked for in.
var proseExt = map[string]bool{".md": true, ".txt": true, ".tsv": true, ".sh": true, ".yml": true, ".yaml": true}

// allowlistPath is the tracked file naming the occurrences that are not removed yet.
const allowlistPath = "internal/privacy/allowlist.tsv"

// allowKey identifies one permission: a rule applied to one path.
type allowKey struct {
	rule string
	path string
}

// allowEntry is one line of allowlist.tsv: how many occurrences of one rule one path may carry.
type allowEntry struct {
	allowKey
	count int
}

// TestNoPersonalDataInTrackedFiles is the ratchet itself: every rule over every tracked file, against
// the counts the allowlist permits.
func TestNoPersonalDataInTrackedFiles(t *testing.T) {
	root := repoRoot(t)
	allowed := readAllowlist(t, root)
	actual := map[allowKey]int{}
	for _, path := range trackedFiles(t, root) {
		for _, rule := range hitsInFile(root, path) {
			actual[allowKey{rule: rule, path: path}]++
		}
	}
	for key, n := range actual {
		if n > allowed[key] {
			t.Errorf("%s: %s carries %d occurrence(s) and %s allows %d — a public repository must not name a machine or a directory layout",
				key.path, key.rule, n, allowlistPath, allowed[key])
		}
	}
	// An entry whose occurrences are gone is a number that guards nothing, and leaving it behind would
	// make the allowlist the last place in the tree that names the machine.
	for key, permit := range allowed {
		if actual[key] == 0 {
			t.Errorf("%s: %s allows %d occurrence(s) of %s and the file now has none — delete the entry, because this ratchet only shrinks",
				allowlistPath, key.path, permit, key.rule)
		}
	}
}

// TestPersonalDataRulesMatch is the proof that the ratchet is not vacuous: every rule has to match a
// line of the class it is written to catch, and to leave a documented placeholder, a public handle and
// a `$HOME`-relative default alone, so a pattern that was mistyped fails here instead of reporting
// zero occurrences forever.
func TestPersonalDataRulesMatch(t *testing.T) {
	catches := map[string][]string{
		"home-dir":         {"site: /home/alice/projects/quark/internal/lang/eval.go:12"},
		"windows-home-dir": {`{"C:\Users\alice\a.qk", "/C:/Users/alice/a.qk"}, worktree: /Users/alice/src/quark/eval.go:12`},
		"abs-cache":        {"cache: /mnt/disk/.cache/qk-proof/baseline/bin/qkc"},
		"machine-name":     {"the report was produced on enoch by the stress runner"},
	}
	misses := map[string][]string{
		"home-dir": {
			"the example path is /home/user/a.qk and the prose writes /home/account/a.qk",
			"a relative site column: internal/lang/eval.go:12",
		},
		"windows-home-dir": {`the example uses C:\Users\user\a.qk, and the prose writes /Users/account/a.qk`},
		"abs-cache": {
			`work="${QK_PROOF_WORK:-$HOME/.cache/qk-i18n/proof}"`,
			"the example directory is /home/user/.cache/qk-proof",
			"https://example.com/foo/.cache/bar",
		},
		"machine-name": {
			"thanks to [@Enoch-199811](https://github.com/Enoch-199811), the maintainer",
			"run under CI, where the runner's host name is not the developer's",
		},
	}
	byName := map[string]personalRule{}
	for _, r := range personalRules {
		byName[r.name] = r
	}
	if len(byName) != len(catches) {
		t.Fatalf("personalRules defines %d rule(s) and the proof covers %d, so a rule could be added without a proof", len(byName), len(catches))
	}
	for name, lines := range catches {
		r, ok := byName[name]
		if !ok {
			t.Errorf("the proof covers rule %q, which personalRules does not define", name)
			continue
		}
		for _, line := range lines {
			if !hitsOnLine(r, line) {
				t.Errorf("rule %s does not match %q, so it reports no occurrences for data it is written to catch", name, line)
			}
		}
	}
	for name, lines := range misses {
		r, ok := byName[name]
		if !ok {
			continue
		}
		for _, line := range lines {
			if hitsOnLine(r, line) {
				t.Errorf("rule %s matches %q, which is not personal data — a rule that fires on a public handle, a documented placeholder or a `$HOME` default is noise, and noise gets the whole check switched off", name, line)
			}
		}
	}
}

// TestPersonalDataAllowlistIsWellFormed keeps the ratchet's own input honest: an entry naming a path
// that is no longer tracked, or a rule that does not exist, is a stale permission.
func TestPersonalDataAllowlistIsWellFormed(t *testing.T) {
	root := repoRoot(t)
	tracked := map[string]bool{}
	for _, path := range trackedFiles(t, root) {
		tracked[path] = true
	}
	known := map[string]bool{}
	for _, r := range personalRules {
		known[r.name] = true
	}
	for _, entry := range readAllowlistEntries(t, root) {
		if !known[entry.rule] {
			t.Errorf("%s: %q is not a rule this file checks", allowlistPath, entry.rule)
		}
		if !tracked[entry.path] {
			t.Errorf("%s: %q is not a tracked file, so its permission guards nothing", allowlistPath, entry.path)
		}
		if entry.count <= 0 {
			t.Errorf("%s: %q allows %d occurrence(s), and a permission of zero is an entry that should be deleted",
				allowlistPath, entry.path, entry.count)
		}
	}
}

// ---- reading the tree ----

// repoRoot is the checkout the test runs in. A linked worktree has a .git file rather than a
// directory, so the root is asked of git instead of being inferred from the layout.
func repoRoot(t *testing.T) string {
	t.Helper()
	root := gitOutput(t, ".", "rev-parse", "--show-toplevel")
	if root == "" {
		t.Fatal("git reported no worktree root, so the ratchet cannot tell what is published")
	}
	return filepath.FromSlash(root)
}

// trackedFiles is every file git would publish, which is the tree this ratchet is about.
func trackedFiles(t *testing.T, root string) []string {
	t.Helper()
	out := gitOutput(t, root, "ls-files", "-z")
	var paths []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			paths = append(paths, filepath.ToSlash(p))
		}
	}
	if len(paths) == 0 {
		t.Fatal("git listed no tracked files, so the ratchet would pass without reading anything")
	}
	return paths
}

// hitsInFile returns the name of each rule that matches one file, once per occurrence, or nothing
// when the file is not text or is not there.
func hitsInFile(root, path string) []string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return nil
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return nil
	}
	prose := proseExt[strings.ToLower(filepath.Ext(path))]
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		for _, r := range personalRules {
			if r.proseOnly && !prose {
				continue
			}
			out = append(out, ruleHits(r, line)...)
		}
	}
	return out
}

// ruleHits returns the name of one rule once per occurrence it finds on a line, skipping the
// occurrences the rules agree to leave alone.
func ruleHits(r personalRule, line string) []string {
	keep := line
	if h, ok := handlePatterns[r.name]; ok {
		keep = h.ReplaceAllString(keep, "")
	}
	var out []string
	for _, m := range r.pattern.FindAllStringSubmatch(keep, -1) {
		if skipMatch(r, m) {
			continue
		}
		out = append(out, r.name)
	}
	return out
}

// skipMatch is the agreement between the rules: a documented placeholder account is not a machine,
// and a cache path under a home directory belongs to the home-directory rule alone.
func skipMatch(r personalRule, m []string) bool {
	if r.placeholder && len(m) > 1 && placeholderAccount[strings.ToLower(accountOf(m))] {
		return true
	}
	return r.name == "abs-cache" && underHome(accountOf(m))
}

// underHome reports whether a matched path is already inside a home directory.
func underHome(path string) bool {
	return strings.HasPrefix(path, "/home/") || strings.HasPrefix(path, "/Users/") || homeDrivePath.MatchString(path)
}

// homeDrivePath is a home directory written with a Windows drive letter.
var homeDrivePath = regexp.MustCompile(`(?i)^/[a-z]:/users/`)

// hitsOnLine reports whether one rule matches a single line, which is how the rules are proved.
func hitsOnLine(r personalRule, line string) bool { return len(ruleHits(r, line)) > 0 }

// accountOf picks the account a path-shaped match is about out of its capture groups.
func accountOf(m []string) string {
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return ""
}

// readAllowlist maps each permission to its count.
func readAllowlist(t *testing.T, root string) map[allowKey]int {
	t.Helper()
	out := map[allowKey]int{}
	for _, e := range readAllowlistEntries(t, root) {
		out[e.allowKey] = e.count
	}
	return out
}

// readAllowlistEntries parses allowlist.tsv: rule, tab, path, tab, count, with # for comments.
func readAllowlistEntries(t *testing.T, root string) []allowEntry {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(allowlistPath)))
	if err != nil {
		t.Fatalf("reading %s: %v", allowlistPath, err)
	}
	var out []allowEntry
	for i, line := range strings.Split(string(raw), "\n") {
		text := strings.TrimSpace(line)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Split(text, "\t")
		if len(fields) != 3 {
			t.Errorf("%s:%d: want a rule, a path and a count separated by tabs, got %d field(s)", allowlistPath, i+1, len(fields))
			continue
		}
		count, cerr := strconv.Atoi(strings.TrimSpace(fields[2]))
		if cerr != nil {
			t.Errorf("%s:%d: %q is not a count", allowlistPath, i+1, fields[2])
			continue
		}
		out = append(out, allowEntry{
			allowKey: allowKey{rule: strings.TrimSpace(fields[0]), path: filepath.ToSlash(strings.TrimSpace(fields[1]))},
			count:    count,
		})
	}
	if len(out) == 0 {
		t.Errorf("%s lists no entries; when the sweep is complete, delete the file and this test with it rather than leaving an empty permission list", allowlistPath)
	}
	return out
}

// gitOutput runs one git command in dir and returns its trimmed stdout, failing the test when git
// cannot answer.
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
