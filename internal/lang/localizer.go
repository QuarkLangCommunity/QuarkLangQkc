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

// parserMessages renders the syntax layer's messages out of this repository's generated catalog. The
// module owns its message IDs and its English source text; this repository owns the values, so the
// bridge is a type conversion and nothing else — there is no text to look up and no registry to keep
// in sync. The generated catalog's Msg numbering follows the module's table order, which is what makes
// the conversion mean the same message on both sides (see cmd/qki18n and internal/i18n/msg_gen.go).
type parserMessages struct{ L i18n.Localizer }

// Msg renders one of the module's messages in the localizer's language.
func (p parserMessages) Msg(id qkparser.ID, args ...any) string {
	return p.L.Msg(i18n.Msg(id), args...)
}

// SetLocalizer installs the localizer used for messages produced by this package and by the syntax
// layer it embeds, so one call switches the whole toolchain's language.
func SetLocalizer(L i18n.Localizer) i18n.Localizer {
	prev := langLocalizer
	langLocalizer = L
	qkparser.SetLocalizer(parserMessages{L: L})
	return prev
}

// Localizer reports the localizer currently in use.
func Localizer() i18n.Localizer { return langLocalizer }

// msgID renders a message of this repository by its stable ID: the ID survives a rephrasing, a
// translation is attached to the message rather than to its current spelling, and the same text in two
// contexts stays two messages. It is the only way a message is written here, because step 5 deleted the
// text-keyed msg and the catalog that made a wording into a key.
func msgID(id i18n.Msg, args ...any) string { return langLocalizer.Msg(id, args...) }

// The syntax layer renders its messages through the same localizer as the rest of this package, so
// one installed localizer covers the parser's generated IDs and this package's catalog keys alike.
func init() { qkparser.SetLocalizer(parserMessages{L: langLocalizer}) }
