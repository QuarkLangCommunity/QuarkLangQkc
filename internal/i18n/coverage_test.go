package i18n

// Engineering gates for the message catalog. They are data-driven on purpose: an earlier version
// scanned sources with regular expressions and missed trailing comments, CRLF checkouts and
// helper-passed format strings — this one parses Go with go/ast and inspects the catalog itself.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var hanRe = regexp.MustCompile(`[\p{Han}]`)

// formatVerbs returns the multiset of fmt verbs in a template ("%s" -> {"s": 1}).
var verbRe = regexp.MustCompile(`%[-+# 0-9.]*([a-zA-Z%])`)

func formatVerbs(s string) map[string]int {
	out := map[string]int{}
	for _, m := range verbRe.FindAllStringSubmatch(s, -1) {
		out[m[1]]++
	}
	return out
}

// templateIndex maps a called name to the argument position carrying the format template.
// -1 means "any argument": the first Chinese string literal wins.
var templateIndex = map[string]int{
	"T": -1, "Msgf": -1, "msg": -1, // msg is the per-package facade name
	"errf": 1, "warn": 2, "errAt": 0, "replyErr": 2,
}

// wiredTemplates parses every Go source in the repository (this package excluded) and returns the
// Chinese templates handed to an i18n boundary, keyed by template with the files that use them.
func wiredTemplates(t *testing.T) map[string][]string {
	t.Helper()
	root := filepath.Join("..", "..")
	out := map[string][]string{}
	fset := token.NewFileSet()

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		slash := filepath.ToSlash(path)
		if !strings.HasSuffix(slash, ".go") || strings.HasSuffix(slash, "_test.go") ||
			strings.Contains(slash, "/internal/i18n/") || strings.Contains(slash, "/.git/") {
			return nil // test files hold fixtures and assertion wording, not wired messages
		}
		rel, _ := filepath.Rel(root, slash)
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Errorf("parse %s: %v", rel, perr)
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			idx, known := calleeTemplateIndex(call.Fun)
			if !known {
				return true
			}
			for i, arg := range call.Args {
				if idx >= 0 && i != idx {
					continue
				}
				lit, ok := arg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				raw := lit.Value
				if len(raw) > 0 && raw[0] == '"' {
					if unq, uerr := strconv.Unquote(raw); uerr == nil {
						raw = unq
					}
				} else if len(raw) >= 2 && raw[0] == '`' {
					raw = raw[1 : len(raw)-1] // raw strings carry no escapes
				}
				if hanRe.MatchString(raw) {
					out[raw] = append(out[raw], rel)
				}
				if idx == -1 {
					return true
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// calleeTemplateIndex resolves a called name ("pkg.T", "x.errf", "errf") to the argument position
// that carries the template; known=false means the call is not an i18n boundary.
func calleeTemplateIndex(fun ast.Expr) (idx int, known bool) {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		pos, ok := templateIndex[f.Sel.Name]
		return pos, ok
	case *ast.Ident:
		pos, ok := templateIndex[f.Name]
		return pos, ok
	}
	return 0, false
}

// TestTemplatesRegistered: every Chinese template reaching an i18n boundary must exist in the
// catalog — the "no Chinese leaking through an unwired message" gate.
func TestTemplatesRegistered(t *testing.T) {
	wired := wiredTemplates(t)
	missing := Untranslated(sortedKeys(wired))
	if len(missing) > 0 {
		t.Errorf("%d wired templates are missing from the catalog:", len(missing))
		for i, m := range missing {
			if i == 25 {
				t.Errorf("  … and %d more", len(missing)-i)
				break
			}
			t.Errorf("  %q   <- %s", m, strings.Join(wired[m], ", "))
		}
	}
	t.Logf("wired templates: %d, catalog entries: %d", len(wired), len(stdCatalog.Keys()))
}

func sortedKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestCatalogIntegrity: no empty entries, no template owned by two domains, and no translation
// that is a verbatim copy of its Chinese source (a classic copy-paste bug).
func TestCatalogIntegrity(t *testing.T) {
	seen := map[string]string{}
	for _, domain := range stdCatalog.Domains() {
		for template, translated := range stdCatalog.Domain(domain) {
			if strings.TrimSpace(template) == "" || strings.TrimSpace(translated) == "" {
				t.Errorf("%s: empty template or translation (%q -> %q)", domain, template, translated)
			}
			if prev, dup := seen[template]; dup {
				t.Errorf("template owned by two domains: %s and %s (%q)", prev, domain, truncate(template))
			}
			seen[template] = domain
			if translated == template && hanRe.MatchString(template) {
				t.Errorf("%s: translation identical to the Chinese source: %q", domain, truncate(template))
			}
		}
	}
}

// TestVerbSetsMatch: a translation must consume exactly the same fmt verbs as its source, which is
// what silently breaks when a placeholder is dropped or reordered while translating.
func TestVerbSetsMatch(t *testing.T) {
	for _, domain := range stdCatalog.Domains() {
		for template, translated := range stdCatalog.Domain(domain) {
			want, got := formatVerbs(template), formatVerbs(translated)
			if len(want) != len(got) {
				t.Errorf("%s: verb kinds differ\n  zh: %q -> %v\n  en: %q -> %v", domain, truncate(template), want, truncate(translated), got)
				continue
			}
			for verb, n := range want {
				if got[verb] != n {
					t.Errorf("%s: verb %%%s count differs (%d vs %d)\n  zh: %q\n  en: %q",
						domain, verb, n, got[verb], truncate(template), truncate(translated))
				}
			}
		}
	}
}

// TestNoSilentFallback: every catalog entry must resolve under EN, and an unknown template must be
// counted — otherwise the miss counter cannot be trusted as an engineering gate.
func TestNoSilentFallback(t *testing.T) {
	L := New(stdCatalog, EN)
	Misses() // clear
	for _, key := range stdCatalog.Keys() {
		if !L.Has(key) {
			t.Fatalf("catalog key not resolvable: %q", truncate(key))
		}
		L.T(key)
	}
	if misses := Misses(); len(misses) != 0 {
		t.Fatalf("rendering the whole catalog produced %d misses: %v", len(misses), misses)
	}
	L.T("这条模板故意没有登记：%s", "x") // negative control: a Chinese template with no entry
	if n := MissTotal(); n != 1 {
		t.Fatalf("miss counter did not record the unknown Chinese template (got %d)", n)
	}
	Misses()
}

// TestDecouplingRatchet freezes the number of direct package-level T() calls: new code must carry a
// Localizer or go through a choke point, so the count may only go down.
func TestDecouplingRatchet(t *testing.T) {
	const allowed = 29 // lowered from 152 after the Localizer migration (internal/lang, compiler, cgen)
	n := 0
	root := filepath.Join("..", "..")
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		slash := filepath.ToSlash(path)
		if err != nil || info.IsDir() || !strings.HasSuffix(slash, ".go") ||
			strings.Contains(slash, "/internal/i18n/") || strings.Contains(slash, "/.git/") {
			return nil
		}
		if src, rerr := os.ReadFile(path); rerr == nil {
			n += strings.Count(string(src), "i18n.T(")
		}
		return nil
	})
	if n > allowed {
		t.Errorf("direct i18n.T( call sites grew to %d (limit %d): carry a Localizer instead", n, allowed)
	}
	t.Logf("direct i18n.T( call sites: %d (limit %d)", n, allowed)
}

// TestEnglishSourceRatchet freezes the number of Chinese-source wired templates.
//
// English is the source language for an internationalized project: a new message is written in English
// and the catalog carries the Chinese translation (so `QK_LANG=zh` still renders Chinese). The legacy
// set was written the other way round, which also made Chinese the *default* rendering; the count may
// only go down as those messages are migrated.
func TestEnglishSourceRatchet(t *testing.T) {
	const allowed = 391 // measured by the AST scan when the ratchet was added; migrate downwards, never up
	wired := wiredTemplates(t)
	n := 0
	var sample []string
	for tpl, files := range wired {
		if !hanRe.MatchString(tpl) {
			continue
		}
		n++
		if len(sample) < 5 {
			sample = append(sample, truncate(tpl)+"  <- "+strings.Join(files, ", "))
		}
	}
	if n > allowed {
		t.Errorf("Chinese-source wired templates grew to %d (limit %d): write new messages in English and put the Chinese in the catalog", n, allowed)
		for _, s := range sample {
			t.Errorf("  %s", s)
		}
	}
	t.Logf("Chinese-source wired templates: %d (limit %d)", n, allowed)
}

func truncate(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 90 {
		return s[:90] + "…"
	}
	return s
}
