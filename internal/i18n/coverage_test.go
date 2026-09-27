package i18n

// The two gates for message internationalization:
//  1. TestTableCoversWiredTemplates: every "wired" Chinese template in the source must have English in the table
//     (adding Chinese text but forgetting to register the translation -> this fails immediately and lists what is missing);
//   (the actual rendering in English mode is covered by TestEnglishModeRendersEnglish in internal/lang --
//     the i18n package cannot import lang back, that would be an import cycle.)

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var cjkRe = regexp.MustCompile(`[\p{Han}]`)

// wiredTemplates scans the repository source and returns the set of "wired Chinese templates".
// It covers three kinds of wiring sites: i18n.T("…"), the central helpers' format strings (errf/warn/errAt/replyErr), and Msg: "…" fields.
func wiredTemplates(t *testing.T) map[string][]string {
	t.Helper()
	root := filepath.Join("..", "..")
	out := map[string][]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		slash := filepath.ToSlash(path)
		if strings.HasSuffix(path, "_test.go") || strings.Contains(slash, "/internal/i18n/") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		// a Windows checkout may be CRLF: normalize it, otherwise multi-line template keys do not match the table
		src := strings.ReplaceAll(string(data), "\r\n", "\n")
		rel, _ := filepath.Rel(root, path)
		add := func(lit string) {
			if lit != "" && cjkRe.MatchString(lit) {
				out[lit] = append(out[lit], rel)
			}
		}
		// i18n.T("…") and i18n.T(`…`)
		for _, m := range regexp.MustCompile("i18n\\.T\\(\"").FindAllStringIndex(src, -1) {
			i := m[1]
			j := i
			for j < len(src) {
				if src[j] == '\\' {
					j += 2
					continue
				}
				if src[j] == '"' {
					break
				}
				j++
			}
			add(unescapeLit(src[i:j]))
		}
		for _, m := range regexp.MustCompile("i18n\\.T\\(`").FindAllStringIndex(src, -1) {
			i := m[1]
			j := strings.Index(src[i:], "`")
			if j > 0 {
				add(src[i : i+j])
			}
		}
		// central helper format strings + Msg: "…"
		for _, re := range []string{
			`\.(?:errf|warn|errAt|replyErr)\([^,()]*,\s*"((?:[^"\\]|\\.)*)"`,
			`Msg:\s*"((?:[^"\\]|\\.)*)"`,
		} {
			for _, m := range regexp.MustCompile(re).FindAllStringSubmatch(src, -1) {
				add(unescapeLit(m[1]))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func unescapeLit(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i+1])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestTableCoversWiredTemplates(t *testing.T) {
	wired := wiredTemplates(t)
	missing := Untranslated(keysOf(wired))
	if len(missing) > 0 {
		t.Errorf("%d 条已接线文案缺英文译文（请在 table.go 登记）：", len(missing))
		for i, m := range missing {
			if i >= 25 {
				t.Errorf("  …（还有 %d 条）", len(missing)-i)
				break
			}
			t.Errorf("  %q   ← %s", m, strings.Join(wired[m], ", "))
		}
	}
	t.Logf("已接线中文模板 %d 条，已登记英文 %d 条", len(wired), len(table))
}

func keysOf(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
