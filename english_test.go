package main

import (
	"go/token"
	"strings"
	"testing"
	"unicode"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/ratchet"
)

// This file is the standing ratchet against Chinese traces in Go source. The repository's prose is
// English and its user-visible words live in the message catalog, so a Han rune in a comment, in a
// test's failure message or in a literal that ships is a trace of the language the code used to be
// written in rather than a fact about the program. Two things are data and are not traces: the
// generated catalog, whose text is the message table's own values, and the test fixtures, which are
// programs, paths and expected renderings in the language under test.

// generatedCatalog is the generated message catalog: its literals are the table's data, so only its
// comments are read, and its text is carved out rather than allowlisted.
const generatedCatalog = "internal/i18n/msg_gen.go"

// traceSpec is this gate: the three shapes a trace takes in a Go file, the occurrences not removed yet,
// and the proof cases that show the rules still match.
var traceSpec = ratchet.Spec{
	Name: "chinese-traces-in-go-source",
	Reason: "a Chinese trace in code is prose the repository does not read, " +
		"or a user-visible word that belongs in internal/i18n/source/*.tsv",
	Allowlist: "internal/ratchet/english-allowlist.tsv",
	Proofs:    "internal/ratchet/testdata/english-rules.tsv",
	Rules: []ratchet.Rule{
		// Han or full-width punctuation inside a comment.
		newTraceRule("han-comment", readComment, true),
		// The same runes inside a string literal passed to a testing helper, which is the message a
		// failing run prints.
		newTraceRule("han-test-prose", readTestProse, true),
		// The same runes in any other string literal. This is where data legitimately lives, which is
		// why it is the only rule the allowlist permits.
		newTraceRule("han-literal", readLiteral, false),
	},
}

// traceRule is one of the three shapes a Chinese trace takes in Go source: a reader, and whether the
// generated catalog is read at all.
type traceRule struct {
	name        string
	read        func(*ratchet.GoFile) []string
	readCatalog bool
}

// newTraceRule names one shape and the reader that finds it. readCatalog is false for the one rule whose
// hits in the generated catalog would be the message table's own data.
func newTraceRule(name string, read func(*ratchet.GoFile) []string, readCatalog bool) ratchet.Rule {
	return traceRule{name: name, read: read, readCatalog: readCatalog}
}

func (r traceRule) Name() string { return r.name }

func (r traceRule) Reads(path string) bool {
	return strings.HasSuffix(path, ".go") && (r.readCatalog || path != generatedCatalog)
}

func (r traceRule) Scan(f *ratchet.File) []ratchet.Hit {
	if f.Go == nil {
		return nil
	}
	var out []ratchet.Hit
	for range r.read(f.Go) {
		out = append(out, ratchet.Hit{Rule: r.name, Path: f.Path})
	}
	return out
}

// readComment returns the file's comment text when any of it carries a trace. A comment group is read
// whole because a trace anywhere in it is one occurrence.
func readComment(f *ratchet.GoFile) []string {
	for _, group := range f.File.Comments {
		if text := group.Text(); carriesTrace(text) {
			return []string{text}
		}
	}
	return nil
}

func readTestProse(f *ratchet.GoFile) []string { return tracedLiterals(f, f.TestProse) }

func readLiteral(f *ratchet.GoFile) []string { return tracedLiterals(f, nil) }

// tracedLiterals returns the file's unquoted string literals that carry a trace, either because they sit
// at one of the wanted positions or because wanted is nil and they do not.
func tracedLiterals(f *ratchet.GoFile, wanted map[token.Pos]bool) []string {
	var out []string
	for pos, value := range f.Unquote {
		if wanted == nil {
			if f.TestProse[pos] {
				continue
			}
		} else if !wanted[pos] {
			continue
		}
		if carriesTrace(value) {
			out = append(out, value)
		}
	}
	return out
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
func TestNoChineseTracesInTrackedGoFiles(t *testing.T) { ratchet.Verify(t, traceSpec) }

// TestChineseTraceRulesMatch is the proof that the ratchet is not vacuous: every rule has to match the
// shape it is written to catch, and to leave ASCII, a Latin accent and the generated catalog alone, so a
// classifier that stopped working fails here instead of reporting zero occurrences forever.
func TestChineseTraceRulesMatch(t *testing.T) {
	ratchet.ProveRules(t, traceSpec)
}

// TestChineseTraceAllowlistIsWellFormed keeps the ratchet's own input honest: an entry naming a path
// that is no longer tracked, or a rule that does not exist, is a stale permission.
func TestChineseTraceAllowlistIsWellFormed(t *testing.T) { ratchet.CheckAllowlist(t, traceSpec) }
