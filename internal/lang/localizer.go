package lang

import (
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
	"github.com/QuarkLangCommunity/QuarkLangQkparser"
)

// The interpreter renders every user-facing message through one localizer instead of reaching for
// the i18n package at ~70 call sites. That keeps presentation out of the evaluator, type checker and
// linter, and lets tests or embedders inject a fixed language without touching global state.
//
// SetLocalizer is the single injection point; it returns the previous localizer so callers can
// restore it (tests use that instead of mutating a package-level language).
var langLocalizer = i18n.Default()

// SetLocalizer installs the localizer used for messages produced by this package and by the syntax
// layer it embeds, so one call switches the whole toolchain's language.
func SetLocalizer(L i18n.Localizer) i18n.Localizer {
	prev := langLocalizer
	langLocalizer = L
	qkparser.SetLocalizer(L)
	return prev
}

// Localizer reports the localizer currently in use.
func Localizer() i18n.Localizer { return langLocalizer }

// msg renders a message template through the package localizer. It is the only place in this
// package that talks to i18n, so switching languages stays a single, testable decision.
func msg(template string, args ...any) string { return langLocalizer.T(template, args...) }

// The syntax layer renders its messages through the same localizer as the rest of this package, so
// the parser templates keep resolving against the catalog even though they now live in qkparser.
func init() { qkparser.SetLocalizer(langLocalizer) }
