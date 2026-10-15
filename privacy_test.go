package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/ratchet"
)

// This file is the standing ratchet against personal data in the repository. A public repository is a
// published artefact: a home directory in a tracked file tells every reader the account name, the
// directory layout and the machine of whoever generated it, and anyone on the network can read it
// without authenticating. The rules below are the classes of that data that were actually found in
// this tree — an account name in a home directory, a Windows home directory, an absolute path into a
// machine-local cache, and the owner's machine name in prose — and they are checked against every
// tracked file, because a scratch file that is not tracked is not published and a tracked one is.

// privacySpec is this gate: the classes of personal data, the occurrences not removed yet, and the proof
// cases that show the rules still match.
var privacySpec = ratchet.Spec{
	Name:      "personal-data-in-tracked-files",
	Reason:    "a public repository must not name a machine or a directory layout",
	Allowlist: "internal/ratchet/privacy-allowlist.tsv",
	Proofs:    "internal/ratchet/testdata/privacy-rules.tsv",
	Rules:     personalRules(),
}

// personalRule is one class of personal data, and where in a file it can appear.
type personalRule struct {
	name string
	// pattern matches one occurrence of the rule in a line. When it has a capture group, the first
	// non-empty group is the account or host the hit is about, and a conventional placeholder is not a
	// hit.
	pattern *regexp.Regexp
	// proseOnly limits the rule to prose files: a Go identifier or a test fixture may legitimately
	// contain the same letters as a machine name, so code is only read for the path-shaped rules.
	proseOnly bool
	// placeholder is true when a documented example may use a conventional account name, which the
	// rule then skips.
	placeholder bool
	// placeholderGroups is true when a rule can capture a field that is not a name, such as the drive
	// letter of a Windows path, and every captured group therefore has to be checked.
	placeholderGroups bool
	// underHomeOnly is true for a rule whose occurrences inside a home directory belong to the
	// home-directory rules instead, so that one occurrence belongs to exactly one rule.
	underHomeOnly bool
}

// personalRules are the classes this repository is held to. They are patterns rather than a list of
// forbidden strings, so renaming the account or the machine does not silently empty the check, and
// they are disjoint: one occurrence belongs to exactly one rule, so a count in the allowlist says
// what it is counting.
func personalRules() []ratchet.Rule {
	return []ratchet.Rule{
		// A home directory under a `Users` directory, which is how macOS and Windows spell it: `C:\Users\`,
		// `C:/Users/` and `/C:/Users/`. RE2 has no lookahead, so the alternatives are written out — a URL
		// and a bare `myUsers/` cannot open this rule, because at a slash the next character must not be a
		// slash or a dot. A Windows spelling captures the drive letter first, which the placeholder check
		// is given as well as the account, so `C:/Users/user/` stays quiet.
		personalRule{name: "windows-home-dir", pattern: regexp.MustCompile(`(?i)([A-Za-z]:|[^A-Za-z0-9:/]|^)[\\/]Users[\\/]([A-Za-z0-9._-]+)[\\/]`), placeholder: true, placeholderGroups: true},
		// An account name in a POSIX home directory. The boundary excludes a colon and a slash, so
		// `C:/home/account/` and a `//home/account/` UNC path are not counted here as well as by the rule
		// above, and one occurrence belongs to exactly one rule.
		personalRule{name: "home-dir", pattern: regexp.MustCompile(`(?:^|[^A-Za-z0-9:/])/home/([A-Za-z0-9._-]+)/`), placeholder: true},
		// An absolute path into a machine-local cache that is not already inside a counted home directory:
		// the cache directory is chosen by the build tool of the machine that ran it, so the path names
		// that machine's layout. A path under a home directory is left to the rules above, so that one
		// occurrence is counted once and a documented `/home/account/.cache/...` example stays quiet. A
		// script that derives its scratch directory from `$HOME` names no machine at all, and neither does
		// `/tmp`, which is the same directory on every POSIX host — which is why CI configuration uses it.
		personalRule{name: "abs-cache", pattern: regexp.MustCompile(`(?:^|[^A-Za-z0-9_$/])(/(?:[A-Za-z0-9._-]+/)*\.cache(?:/[A-Za-z0-9._-]+)*)`), underHomeOnly: true},
		// The owner's machine name, in prose only. The maintainer's public handle is attribution rather
		// than private data, so the handle is not this rule. The rule cannot be trusted in code, where the
		// same letters can be an identifier.
		personalRule{name: "machine-name", pattern: regexp.MustCompile(`(?i)\benoch\b`), proseOnly: true},
	}
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

// homeDrivePath is a home directory written with a Windows drive letter.
var homeDrivePath = regexp.MustCompile(`(?i)^/[a-z]:/users/`)

func (r personalRule) Name() string { return r.name }

// Reads reports that a prose-only rule reads prose files, and every other rule reads every file: a home
// directory is a home directory wherever it is written.
func (r personalRule) Reads(path string) bool {
	return !r.proseOnly || proseExt[strings.ToLower(filepath.Ext(path))]
}

// Scan returns the rule's name once per occurrence in the file, skipping the occurrences the rules agree
// to leave alone.
func (r personalRule) Scan(f *ratchet.File) []ratchet.Hit {
	if f.Raw == nil {
		return nil
	}
	var out []ratchet.Hit
	for _, line := range f.Lines() {
		for range r.hitsOnLine(line) {
			out = append(out, ratchet.Hit{Rule: r.name, Path: f.Path})
		}
	}
	return out
}

// hitsOnLine returns the texts one rule matches on a line, which is how a rule is proved as well as
// counted.
func (r personalRule) hitsOnLine(line string) []string {
	keep := line
	if handle, ok := handlePatterns[r.name]; ok {
		keep = handle.ReplaceAllString(keep, "")
	}
	var out []string
	for _, m := range r.pattern.FindAllStringSubmatch(keep, -1) {
		if !r.skip(m) {
			out = append(out, m[0])
		}
	}
	return out
}

// skip is the agreement between the rules: a documented placeholder account is not a machine, and a
// cache path under a home directory belongs to the home-directory rule alone.
func (r personalRule) skip(m []string) bool {
	if r.placeholder {
		if r.placeholderGroups {
			// Every group is a name-like field, and one of them being an account is enough: the drive
			// letter of a Windows path is one of them and is not a name.
			for _, g := range m[1:] {
				if placeholderAccount[strings.ToLower(g)] {
					return true
				}
			}
		} else if len(m) > 1 && placeholderAccount[strings.ToLower(accountOf(m))] {
			return true
		}
	}
	return r.underHomeOnly && underHome(accountOf(m))
}

// underHome reports whether a matched path already names a home directory.
func underHome(path string) bool {
	return strings.HasPrefix(path, "/home/") || strings.HasPrefix(path, "/Users/") || homeDrivePath.MatchString(path)
}

// accountOf picks the account a path-shaped match is about out of its capture groups.
func accountOf(m []string) string {
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return ""
}

// TestNoPersonalDataInTrackedFiles is the ratchet itself: every rule over every tracked file, against
// the counts the allowlist permits.
func TestNoPersonalDataInTrackedFiles(t *testing.T) { ratchet.Verify(t, privacySpec) }

// TestPersonalDataRulesMatch is the proof that the ratchet is not vacuous: every rule has to match the
// lines it is written to catch, and to leave a documented placeholder, a public handle and a
// `$HOME`-relative default alone, so a pattern that was mistyped fails here instead of reporting zero
// occurrences forever.
func TestPersonalDataRulesMatch(t *testing.T) {
	ratchet.ProveRules(t, privacySpec)
}

// TestPersonalDataAllowlistIsWellFormed keeps the ratchet's own input honest: an entry naming a path
// that is no longer tracked, or a rule that does not exist, is a stale permission.
func TestPersonalDataAllowlistIsWellFormed(t *testing.T) { ratchet.CheckAllowlist(t, privacySpec) }
