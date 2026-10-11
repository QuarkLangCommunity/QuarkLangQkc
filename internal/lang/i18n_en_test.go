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
	withLocalizer(t, i18n.New(i18n.EN))

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

// The Chinese path renders the Chinese the catalog holds, English-source messages included. It used to
// assert the opposite — that Chinese rendering needed no translation and produced no misses — which is
// the behaviour that printed English under QK_LANG=zh; the miss counter it leaned on is gone with the
// text-keyed catalog (step 5), so what is left is the rendering itself.
func TestChineseLocalizerRendersTemplates(t *testing.T) {
	withLocalizer(t, i18n.New(i18n.ZH))

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

	// Both nesting refusals are generated now — the parser's and the type checker's — so the assertions
	// name IDs rather than repeating the wording, which is the point of the migration: this test used to
	// break whenever the English text was corrected. The last one was a catalog key until the typecheck
	// domain migrated; the message and its Chinese are unchanged, only where they live.
	for _, msg := range []i18n.Msg{
		i18n.MsgParserNestingTooDeep,
		i18n.MsgParserPpUnclosedIf,
		i18n.MsgTypecheckCompileerrorExpressionNestingDeeperThanD,
	} {
		got := i18n.New(i18n.ZH).Msg(msg, MaxExprDepth)
		if !hanRe.MatchString(got) {
			t.Errorf("Chinese mode did not render Chinese for %s: %q", i18n.MsgKey(msg), got)
		}
	}
	got := i18n.New(i18n.ZH).Msg(i18n.MsgTypecheckCompileerrorExpressionNestingDeeperThanD, MaxExprDepth)
	if !strings.Contains(got, "嵌套深度超过本实现的") {
		t.Errorf("Chinese mode did not render the catalog's Chinese: %q", got)
	}
	if strings.Contains(got, "nesting is deeper than") {
		t.Errorf("Chinese mode leaked the English template: %q", got)
	}
}

// TestEnglishRenderingHasNoMisses is deleted, not weakened: it asserted that rendering under English
// counted no fallback to Chinese, and the counter it read existed to tolerate exactly the defect this
// migration removed. With IDs there is no fallback to count — a message with no English value fails
// generation (TestProofMissingLanguageFails) rather than falling back at run time — so the test has no
// subject. Step 5 of docs/i18n-design.md deletes Misses/miss.go with it.

// withLocalizer swaps the package localizer for the duration of one test and restores it after.
func withLocalizer(t *testing.T, L i18n.Localizer) {
	t.Helper()
	prev := SetLocalizer(L)
	t.Cleanup(func() { SetLocalizer(prev) })
}
