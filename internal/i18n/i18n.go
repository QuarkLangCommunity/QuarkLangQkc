// Package i18n renders user-facing messages in the user's language.
//
// The design is deliberately small, and after step 5 of docs/i18n-design.md it is smaller than it was:
// there is exactly one way to write a message and one way to render it.
//
//   - A message is an ID. The IDs, their per-language values and the blob they are packed into are
//     generated from the authored tables under source/ by internal/i18ngen and committed (msg_gen.go),
//     so a build never needs the generator to run and a missing language is a missing column at
//     generation time rather than a fallback at run time.
//   - Lang is a plain value and Detect is a pure function over the environment: no hidden per-call
//     state.
//   - Localizer is a value type (language only). Pass it down instead of reaching for a package-level
//     variable: `L := i18n.Default()` once per tool, then `L.Msg(...)`.
//
// There is no text-keyed lookup, no catalog, no direction sniff and no miss counter. Those existed to
// tolerate the defect this migration removed: a message whose identity was its own text, which could be
// rendered in the wrong language without anything failing. A text-keyed entry point is what made that
// unfixable, so step 5 deleted it, and TestDecouplingRatchet now asserts that none comes back.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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

// Localizer renders messages in one language. It is a value type: copy it freely, keep one per tool or
// interpreter, and never mutate shared state to change language.
type Localizer struct {
	lang Lang
}

// New builds a localizer for an explicit language.
func New(l Lang) Localizer { return Localizer{lang: l} }

// WithLang returns a copy that renders in another language; the receiver is unchanged.

// Lang reports the language this localizer renders in.
func (L Localizer) Lang() Lang {
	if L.lang == "" {
		return defaultLang()
	}
	return L.lang
}

// Msg renders a generated message in the localizer's language. The message is an ID and the value is
// read out of the generated blob by index (msg_gen.go): one index and one slice, no map keyed by the
// text, no direction sniff, no fallback to count. It is also the interface the syntax layer renders its
// messages through (qkparser.Localizer), so the module's messages are localized without the module
// knowing anything about it.
func (L Localizer) Msg(id Msg, args ...any) string {
	text := msgText(id, langBlock(string(L.Lang())))
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
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
func Default() Localizer { return New(defaultLang()) }

// Render is the text of one message ID, through the localizer the process is set to. A tool with its
// own injection point installs a Localizer and renders through that, instead of keeping a copy of this.
func Render(id Msg, args ...any) string { return Default().Msg(id, args...) }

// SetLocale overrides the process default language. Intended for tests and the tools'
// --lang flag; elsewhere pass a Localizer around instead of mutating process state.
func SetLocale(l Lang) {
	langOnce.Do(func() {}) // mark resolved so Detect cannot overwrite the explicit choice
	langValue.Store(l)
}

func Locale() Lang { return defaultLang() }
