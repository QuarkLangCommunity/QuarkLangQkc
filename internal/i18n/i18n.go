// Package i18n renders user-facing messages in the user's language.
//
// Design (deliberately small, immutable and injectable):
//
//   - Lang is a plain value and Detect is a pure function over the environment: no hidden
//     per-call state.
//   - Catalog is immutable data assembled from the per-domain table_*.go files (runtime,
//     typecheck, lint, lexer, macro, docgen, repl, cli, compiler). Keys are the exact source
//     templates, so a call site and its translation cannot drift apart silently —
//     TestTemplatesRegistered fails when they do.
//   - Localizer is a value type (catalog + language). Pass it down instead of reaching for a
//     package-level variable: `L := i18n.Default()` once per tool interpreter, then `L.T(...)`.
//     An unknown template under EN is counted (Misses) rather than silently ignored, so tests
//     can assert "no silent fallback" instead of trusting a source scan.
//
// The package-level T/SetLocale/Locale functions exist for call sites written before the
// Localizer existed; they delegate to Default. New code should carry a Localizer.
package i18n

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

// Lang is a supported output language.
type Lang string

// Supported languages.
const (
	ZH Lang = "zh"
	EN Lang = "en"
)

// ParseLang maps a locale-ish string ("en_US.UTF-8", "zh", "EN") to a Lang.
func ParseLang(s string) (Lang, bool) {
	v := strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.HasPrefix(v, string(ZH)):
		return ZH, true
	case strings.HasPrefix(v, string(EN)):
		return EN, true
	}
	return "", false
}

func (l Lang) String() string { return string(l) }

// Detect resolves the language from the environment: QK_LANG, then LC_ALL / LC_MESSAGES / LANG,
// then English. It keeps no state, so a Chinese machine still gets Chinese output while an
// English machine needs no configuration.
func Detect() Lang {
	if l, ok := ParseLang(os.Getenv("QK_LANG")); ok {
		return l
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if l, ok := ParseLang(os.Getenv(k)); ok {
			return l
		}
	}
	return EN // global default: the documentation is English-first
}

// Catalog holds the translations, grouped by the domain that owns them.
type Catalog struct {
	domains map[string]map[string]string // domain -> template -> translation
	byKey   map[string]map[Lang]string   // template -> lang -> translation (lookup index)
	keys    []string                     // sorted, for diagnostics and tests
}

// Load assembles the catalog from the per-domain tables compiled into the binary. Each table maps the
// template written at the call site to the same message in the other language, so the direction is a
// property of the key: the legacy entries are Chinese-source and carry their English translation, the
// migrated ones are English-source and carry their Chinese translation. Reading the direction off the
// key is what keeps an English-source message English under EN — indexing every table as if its key
// were the Chinese text rendered the Chinese value of exactly the messages that were already migrated.
// The result is immutable; callers may share it freely.
func Load() *Catalog {
	c := &Catalog{
		domains: make(map[string]map[string]string, 10),
		byKey:   make(map[string]map[Lang]string, 512),
	}
	for domain, table := range domainTables() {
		c.domains[domain] = table
		for template, translated := range table {
			m, ok := c.byKey[template]
			if !ok {
				m = make(map[Lang]string, 2)
				c.byKey[template] = m
				c.keys = append(c.keys, template)
			}
			if needsTranslation(template) { // Chinese source: the value is its English translation
				m[EN] = translated
				m[ZH] = template
			} else { // English source: the value is its Chinese translation
				m[EN] = template
				m[ZH] = translated
			}
		}
	}
	sort.Strings(c.keys)
	return c
}

// Domains returns the domain names, sorted.
func (c *Catalog) Domains() []string {
	out := make([]string, 0, len(c.domains))
	for d := range c.domains {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// Domain returns one domain's table (never nil).
func (c *Catalog) Domain(name string) map[string]string { return c.domains[name] }

// Keys returns every known template, sorted.
func (c *Catalog) Keys() []string { return append([]string(nil), c.keys...) }

// Lookup returns the translation of a template. ok is false when the template is unknown.
func (c *Catalog) Lookup(template string, l Lang) (string, bool) {
	m, ok := c.byKey[template]
	if !ok {
		return "", false
	}
	s, ok := m[l]
	return s, ok
}

// Has reports whether the template is registered (in any language).
func (c *Catalog) Has(template string) bool {
	_, ok := c.byKey[template]
	return ok
}

var stdCatalog = Load()

// Std returns the process-wide immutable catalog.
func Std() *Catalog { return stdCatalog }

// domainTables collects the generated per-domain tables. Keeping the list here (rather than in
// the table files) makes adding a domain a one-line change and the merge explicit.
func domainTables() map[string]map[string]string {
	return map[string]map[string]string{
		"cli":       cliTable,
		"compiler":  compilerTable,
		"docgen":    docgenTable,
		"lexer":     lexerTable,
		"lint":      lintTable,
		"macro":     macroTable,
		"repl":      replTable,
		"runtime":   runtimeTable,
		"shared":    sharedTable,
		"typecheck": typecheckTable,
	}
}

// ---- Localizer: a value, not a global ----

// Localizer renders messages in one language from one catalog. It is a value type: copy it
// freely, keep one per tool or interpreter, and never mutate shared state to change language.
type Localizer struct {
	cat  *Catalog
	lang Lang
}

// New builds a localizer for an explicit catalog and language (nil catalog = the standard one).
func New(cat *Catalog, l Lang) Localizer {
	if cat == nil {
		cat = stdCatalog
	}
	return Localizer{cat: cat, lang: l}
}

// WithLang returns a copy that renders in another language; the receiver is unchanged.
func (L Localizer) WithLang(l Lang) Localizer {
	L.lang = l
	return L
}

// Lang reports the language this localizer renders in.
func (L Localizer) Lang() Lang {
	if L.lang == "" {
		return defaultLang()
	}
	return L.lang
}

// T renders a template: under EN it uses the registered translation, otherwise the template
// itself. An unknown template under EN is counted (see Misses) so the fallback is visible.
func (L Localizer) T(template string, args ...any) string {
	format := template
	if L.Lang() == EN {
		if translated, ok := L.catalog().Lookup(template, EN); ok {
			format = translated
		} else if needsTranslation(template) {
			// A template that is already English needs no entry; only a Chinese one that fell
			// back to Chinese counts as a missing translation.
			countMiss(template)
		}
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// Has reports whether a template is registered.
func (L Localizer) Has(template string) bool { return L.catalog().Has(template) }

// needsTranslation reports whether a template contains Han characters, i.e. whether rendering it
// under English without a catalog entry would leak Chinese to the user.
func needsTranslation(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func (L Localizer) catalog() *Catalog {
	if L.cat == nil {
		return stdCatalog
	}
	return L.cat
}

// ---- process default: resolved once, replaceable only by tests and --lang ----

var (
	langOnce  sync.Once
	langValue atomic.Value // stores Lang
)

func defaultLang() Lang {
	langOnce.Do(func() { langValue.Store(Detect()) })
	l, _ := langValue.Load().(Lang)
	if l == "" {
		return EN
	}
	return l
}

// Default returns the process default localizer. The environment is read once; afterwards the
// language changes only through SetLocale (tests, --lang) or WithLang.
func Default() Localizer { return New(stdCatalog, defaultLang()) }

// SetLocale overrides the process default language. Intended for tests and the tools'
// --lang flag; elsewhere pass a Localizer around instead of mutating process state.
func SetLocale(l Lang) {
	langOnce.Do(func() {}) // mark resolved so Detect cannot overwrite the explicit choice
	langValue.Store(l)
}

// Locale returns the process default language.
func Locale() Lang { return defaultLang() }

// T renders a template with the process default localizer. Prefer holding a Localizer.
func T(template string, args ...any) string { return Default().T(template, args...) }

// Has reports whether a template is registered in the standard catalog.
func Has(template string) bool { return stdCatalog.Has(template) }

// Untranslated filters templates that are not registered.
func Untranslated(templates []string) []string {
	var out []string
	for _, t := range templates {
		if !stdCatalog.Has(t) {
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}
