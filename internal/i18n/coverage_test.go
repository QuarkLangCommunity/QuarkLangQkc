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
		if skipNestedRepo(path, info, root) {
			return filepath.SkipDir
		}
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
				// Every wired template is collected, English-source ones included: they need a Chinese
				// catalog value for QK_LANG=zh, so they cannot be skipped by a Han filter.
				out[raw] = append(out[raw], rel)
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
	// Every wired template must have a translation, including the English-source ones: English is the
	// source language, so a message written in English needs a Chinese catalog value for QK_LANG=zh.
	missing := missingEntries(sortedKeys(wired))
	// Ratchet, not a cliff: every message needs a Chinese catalog value (English is the source
	// language, so QK_LANG=zh renders the translation). The scanner used to skip English templates —
	// that is how this backlog was found — so the count may only go down from here.
	const allowedMissing = 124
	if len(missing) > allowedMissing {
		t.Errorf("%d wired templates have no translation (limit %d): an English message needs a Chinese catalog value for QK_LANG=zh", len(missing), allowedMissing)
		for i, m := range missing {
			if i == 25 {
				t.Errorf("  … and %d more", len(missing)-i)
				break
			}
			t.Errorf("  %q   <- %s", m, strings.Join(wired[m], ", "))
		}
	}
	t.Logf("wired templates: %d, without a translation: %d (limit %d)", len(wired), len(missing), allowedMissing)
}

// missingEntries reports wired templates with no catalog entry at all. The Han filter used to live
// here, which let English-source messages through without a Chinese translation — they then stayed
// English under QK_LANG=zh, which is exactly what an internationalized catalog must avoid.
func missingEntries(keys []string) []string {
	var out []string
	for _, k := range keys {
		if !stdCatalog.Has(k) {
			out = append(out, k)
		}
	}
	return out
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
// counted — otherwise the miss counter cannot be trusted as an engineering gate. The counter follows
// the requested language in both directions: a template only falls back visibly when the text it falls
// back to is written in the language the other one asked for.
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
	// The mirror image: English is the source language for migrated messages, so an English template
	// with no entry renders English to a reader who asked for Chinese.
	New(stdCatalog, ZH).T("this template is deliberately not registered: %s", "x")
	if n := MissTotal(); n != 1 {
		t.Fatalf("miss counter did not record the unknown English template under ZH (got %d)", n)
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
	const allowed = 318 // lowered by the English-source batches (compiler front end migrated)
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

// TestTRendersTheCatalogForTheRequestedLanguage: T must consult the catalog for the localizer's own
// language. The lookup used to be gated on EN, so every English-source entry — whose Chinese text is
// the catalog value — rendered its English template under QK_LANG=zh while Lookup(key, ZH) answered
// correctly: the translation existed and was simply never reached.
func TestTRendersTheCatalogForTheRequestedLanguage(t *testing.T) {
	renderers := map[Lang]Localizer{EN: New(stdCatalog, EN), ZH: New(stdCatalog, ZH)}
	checked := 0
	for _, domain := range stdCatalog.Domains() {
		for template := range stdCatalog.Domain(domain) {
			for lang, L := range renderers {
				want, ok := stdCatalog.Lookup(template, lang)
				if !ok {
					t.Fatalf("%s: %q has no %s rendering", domain, truncate(template), lang)
				}
				if got := L.T(template); got != want {
					t.Errorf("%s: T(%s) of %q rendered %q, want the catalog's %q",
						domain, lang, truncate(template), truncate(got), truncate(want))
				}
				checked++
			}
		}
	}
	t.Logf("rendered %d catalog entries through T() in both languages", checked)
}

// TestCatalogDirectionFollowsTheSourceLanguage pins the per-language index to the direction of each
// table entry. A table maps the call-site template to the other language, so the entry's direction is
// a property of its key; indexing every table as if the key were Chinese put the Chinese value behind
// the English one, which is what made an English-source diagnostic render Chinese on the runners.
func TestCatalogDirectionFollowsTheSourceLanguage(t *testing.T) {
	for _, domain := range stdCatalog.Domains() {
		for template, translated := range stdCatalog.Domain(domain) {
			en, _ := stdCatalog.Lookup(template, EN)
			zh, _ := stdCatalog.Lookup(template, ZH)
			if hanRe.MatchString(template) { // legacy Chinese source: EN carries the translation
				if en != translated {
					t.Errorf("%s: %q renders %q under EN, want its translation %q", domain, truncate(template), truncate(en), truncate(translated))
				}
				if zh != template {
					t.Errorf("%s: %q renders %q under ZH, want the template itself", domain, truncate(template), truncate(zh))
				}
				continue
			}
			if en != template { // English source: EN is the template, ZH carries the translation
				t.Errorf("%s: English-source %q renders %q under EN, want the template itself", domain, truncate(template), truncate(en))
			}
			if zh != translated {
				t.Errorf("%s: English-source %q renders %q under ZH, want its translation %q", domain, truncate(template), truncate(zh), truncate(translated))
			}
		}
	}
}

func truncate(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 90 {
		return s[:90] + "…"
	}
	return s
}

// skipNestedRepo reports whether a walked directory belongs to a different repository. The QuarkLang
// family keeps its sibling projects inside this workspace, each with its own .git, and another
// repository's files are not this project's files: walking into them would let their sources decide
// this project's gates.
func skipNestedRepo(path string, info os.FileInfo, root string) bool {
	if info == nil || !info.IsDir() || path == root {
		return false
	}
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
