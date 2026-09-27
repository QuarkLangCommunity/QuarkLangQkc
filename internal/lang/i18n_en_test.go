package lang

// Actual-render verification of English mode (QK_LANG=en / SetLocale(en)): no Chinese may remain in the output of real scenarios.
// Covers the three paths most easily missed: type-check errors, parse errors, static-check diagnostics.

import (
	"regexp"
	"strings"
	"testing"

	"quarklang/internal/i18n"
)

var hanRe = regexp.MustCompile(`[\p{Han}]`)

func TestEnglishModeRendersEnglish(t *testing.T) {
	defer i18n.SetLocale(i18n.ZH)
	i18n.SetLocale(i18n.EN)

	t.Run("类型检查错误", func(t *testing.T) {
		_, err := Compile("fn main(IOStream io) void {\n    int x = \"s\";\n}\n")
		if err == nil {
			t.Fatal("期望类型错误")
		}
		if hanRe.MatchString(err.Error()) {
			t.Errorf("英文模式下错误信息仍是中文：%s", err.Error())
		}
	})
	t.Run("解析错误", func(t *testing.T) {
		_, err := ParseSource("fn main(IOStream io) void {\n    int x = ;\n}\n")
		if err == nil {
			t.Fatal("期望解析错误")
		}
		if hanRe.MatchString(err.Error()) {
			t.Errorf("英文模式下解析错误仍是中文：%s", err.Error())
		}
	})
	t.Run("静态检查诊断", func(t *testing.T) {
		prog, err := ParseSource("fn main(IOStream io) void {\n    int unused = 1;\n    io.println(\"ok\");\n}\n")
		if err != nil {
			t.Fatal(err)
		}
		diags := Lint(prog, LintOptions{})
		if len(diags) == 0 {
			t.Fatal("期望至少一条诊断（未使用变量）")
		}
		for _, d := range diags {
			if hanRe.MatchString(d.Msg) {
				t.Errorf("英文模式下诊断仍是中文：%s %s", d.Code, d.Msg)
			}
		}
		t.Logf("诊断：%s %s", diags[0].Code, diags[0].Msg)
	})
}

// The default language must be Chinese (without changing existing behaviour), and QK_LANG is decided only once at process start.
func TestDefaultLocaleZh(t *testing.T) {
	if i18n.Locale() != i18n.ZH {
		t.Fatalf("默认语言应为 zh，实际 %s", i18n.Locale())
	}
	if got := i18n.T("变量 %s 声明后从未使用", "x"); !strings.Contains(got, "变量") {
		t.Errorf("中文模式不该翻译：%q", got)
	}
}
