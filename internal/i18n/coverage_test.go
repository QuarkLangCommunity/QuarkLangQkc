package i18n

// 文案国际化的两道门禁：
//  1. TestTableCoversWiredTemplates：源码里所有「已接线」的中文模板都必须在表里有英文
//     （新增中文文案却忘了登记译文 → 这里直接失败并列出缺哪些）；
//   （英文模式的实际渲染由 internal/lang 的 TestEnglishModeRendersEnglish 覆盖——
//     i18n 包不能反向导入 lang，否则成环。）

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var cjkRe = regexp.MustCompile(`[\p{Han}]`)

// wiredTemplates 扫描仓库源码，返回「已接线的中文模板」集合。
// 覆盖三类接线点：i18n.T("…")、中心助手的格式串（errf/warn/errAt/replyErr）、Msg: "…" 字段。
func wiredTemplates(t *testing.T) map[string][]string {
	t.Helper()
	root := filepath.Join("..", "..")
	out := map[string][]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/internal/i18n/") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		src := string(data)
		rel, _ := filepath.Rel(root, path)
		add := func(lit string) {
			if lit != "" && cjkRe.MatchString(lit) {
				out[lit] = append(out[lit], rel)
			}
		}
		// i18n.T("…") 与 i18n.T(`…`)
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
		// 中心助手格式串 + Msg: "…"
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
