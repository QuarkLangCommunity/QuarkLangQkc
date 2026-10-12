package i18ngen

import (
	"strings"
	"testing"
)

// testLangs is the smallest set a table parses with: one source language and one translation.
var testLangs = []Language{{Code: "en", Direction: "source"}, {Code: "de", Direction: "translation"}}

// oneRow returns a table holding a single row whose ctx column is ctx and whose other columns are valid.
func oneRow(ctx string) []byte {
	return []byte("probe.one_row\tlocal\t" + ctx + "\tprobe note\t\"hello\"\t\"hello\"\t\"hallo\"\n")
}

// TestParseMessagesRefusesAbsoluteSite is the refusal this defect needs: a ctx column naming the site by
// an absolute path must not parse, so the machine the table was authored on cannot be committed back in.
func TestParseMessagesRefusesAbsoluteSite(t *testing.T) {
	for _, ctx := range []string{
		"/home/jack/.cache/qk-i18n-s5/QuarkLang/internal/lang/eval.go:3563",
		`C:\Users\jack\QuarkLang\internal\lang\eval.go:3563`,
		`\\build-server\share\QuarkLang\internal\lang\eval.go:3563`,
		"file:///home/jack/QuarkLang/internal/lang/eval.go:3563",
		"../other-checkout/internal/lang/eval.go:3563",
	} {
		_, err := ParseMessages("fields.tsv", oneRow(ctx), testLangs)
		if err == nil {
			t.Errorf("ctx %q parsed; a site outside this repository must be refused", ctx)
			continue
		}
		if !strings.Contains(err.Error(), "relative to the repository root") {
			t.Errorf("ctx %q: error %q does not say how to write the site", ctx, err)
		}
	}
}

// TestParseMessagesAcceptsRepositoryRelativeSite is the other half: the relative form every table now
// uses, and a short context label that is not a site at all, must keep parsing.
func TestParseMessagesAcceptsRepositoryRelativeSite(t *testing.T) {
	for _, ctx := range []string{
		"internal/lang/eval.go:3563",
		"compiler/internal/cgen/lower.go:139",
		"lint",
		"the /help path of the REPL",
	} {
		if _, err := ParseMessages("fields.tsv", oneRow(ctx), testLangs); err != nil {
			t.Errorf("ctx %q was refused: %v", ctx, err)
		}
	}
}
