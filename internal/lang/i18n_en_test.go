package lang

// End-to-end language checks. Since the evaluator/type checker/linter render through the package
// localizer (not through process state), these tests inject one with SetLocalizer — the same single
// injection point a tool or embedder would use — and restore it afterwards.

import (
	"regexp"
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

// The Chinese path must stay untouched: injecting a Chinese localizer renders the templates
// verbatim, with no translation and no misses.
func TestChineseLocalizerRendersTemplates(t *testing.T) {
	withLocalizer(t, i18n.New(nil, i18n.ZH))
	if misses := i18n.Misses(); len(misses) != 0 {
		t.Fatalf("unexpected misses before the test: %v", misses)
	}
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
	if misses := i18n.Misses(); len(misses) != 0 {
		t.Errorf("Chinese rendering must not need translations, got misses: %v", misses)
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
