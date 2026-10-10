package lang

// End-to-end language checks. Since the evaluator/type checker/linter render through the package
// localizer (not through process state), these tests inject one with SetLocalizer — the same single
// injection point a tool or embedder would use — and restore it afterwards.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

var hanRe = regexp.MustCompile(`[\p{Han}]`)

func TestEnglishModeRendersEnglish(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))

	t.Run("type check error", func(t *testing.T) {
		_, err := Compile("fn main(IOStream io) void {\n    int x = \"s\";\n}\n")
		if err == nil {
			t.Fatal("expected a type error")
		}
		if hanRe.MatchString(err.Error()) {
			t.Errorf("English mode still rendered Chinese: %s", err.Error())
		}
	})
	t.Run("parse error", func(t *testing.T) {
		_, err := ParseSource("fn main(IOStream io) void {\n    int x = ;\n}\n")
		if err == nil {
			t.Fatal("expected a parse error")
		}
		if hanRe.MatchString(err.Error()) {
			t.Errorf("English mode still rendered Chinese: %s", err.Error())
		}
	})
	t.Run("lint diagnostic", func(t *testing.T) {
		prog, err := ParseSource("fn main(IOStream io) void {\n    int unused = 1;\n    io.println(\"ok\");\n}\n")
		if err != nil {
			t.Fatal(err)
		}
		diags := Lint(prog, LintOptions{})
		if len(diags) == 0 {
			t.Fatal("expected at least one diagnostic (unused variable)")
		}
		for _, d := range diags {
			if hanRe.MatchString(d.Msg) {
				t.Errorf("English mode still rendered Chinese: %s %s", d.Code, d.Msg)
			}
		}
		t.Logf("diagnostic: %s %s", diags[0].Code, diags[0].Msg)
	})
}

// The Chinese path renders the catalog's Chinese, English-source messages included: a message migrated to an
// English template carries its Chinese in the catalog, so a Chinese localizer has to consult it instead of
// assuming the call-site template is already Chinese. This test used to assert the opposite — that Chinese
// rendering needed no translation and produced no misses — which is the behaviour that printed English under
// QK_LANG=zh. The catalog-wide check lives in internal/i18n (TestTRendersTheCatalogForTheRequestedLanguage);
// the pair below is the one the bug was reported on, asserted end to end through this package's localizer.
func TestChineseLocalizerRendersTemplates(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.ZH))
	i18n.Misses() // the counters are process-wide; other tests share them

	prog, err := ParseSource("fn main(IOStream io) void {\n    int unused = 1;\n    io.println(\"ok\");\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Lint(prog, LintOptions{})
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic")
	}
	if !hanRe.MatchString(diags[0].Msg) {
		t.Errorf("Chinese mode should render the Chinese template, got %q", diags[0].Msg)
	}

	for _, template := range []string{
		"nesting is deeper than %d levels, the limit of this implementation: split the expression into smaller statements",
		"CompileError: expression nesting is deeper than %d levels, the limit of this implementation: split the expression into smaller statements",
	} {
		got := i18n.New(nil, i18n.ZH).T(template, MaxExprDepth)
		if !strings.Contains(got, "嵌套深度超过本实现的") {
			t.Errorf("Chinese mode did not render the catalog's Chinese for %q: %q", template, got)
		}
		if strings.Contains(got, "nesting is deeper than") {
			t.Errorf("Chinese mode leaked the English template for %q: %q", template, got)
		}
	}

	if misses := i18n.Misses(); len(misses) != 0 {
		t.Errorf("every message above resolves through the catalog, so rendering them must not count misses: %v", misses)
	}
}

// English rendering must resolve every message it touches: a miss means a template is missing from
// the catalog, which used to fall back to Chinese silently.
func TestEnglishRenderingHasNoMisses(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.EN))
	i18n.Misses() // clear

	_, _ = Compile("fn main(IOStream io) void {\n    int x = \"s\";\n}\n")
	_, _ = ParseSource("fn main(IOStream io) void {\n    int x = ;\n}\n")
	if prog, err := ParseSource("fn main(IOStream io) void {\n    int unused = 1;\n    io.println(\"ok\");\n}\n"); err == nil {
		_ = Lint(prog, LintOptions{})
	}

	if misses := i18n.Misses(); len(misses) != 0 {
		t.Errorf("%d templates rendered their Chinese fallback under English mode: %v", len(misses), misses)
	}
}

// withLocalizer swaps the package localizer for the duration of one test and restores it after.
func withLocalizer(t *testing.T, L i18n.Localizer) {
	t.Helper()
	prev := SetLocalizer(L)
	t.Cleanup(func() { SetLocalizer(prev) })
}
