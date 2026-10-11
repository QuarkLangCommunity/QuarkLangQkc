package main

import "github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"

// The compiler renders its diagnostics through one localizer instead of calling i18n at each site.
// SetLocalizer is the single injection point (tools and tests use it; nothing else mutates state).
var compilerLocalizer = i18n.Default()

// SetLocalizer installs the localizer used for compiler messages.
func SetLocalizer(L i18n.Localizer) i18n.Localizer {
	prev := compilerLocalizer
	compilerLocalizer = L
	return prev
}

// msgID renders a message of this repository by its stable ID. It is what a migrated call site uses
// instead of msg: the ID survives a rephrasing, and the same text in two contexts stays two messages.
// msg and msgID both disappear at step 5, when the text-keyed catalog does.
func msgID(id i18n.Msg, args ...any) string { return compilerLocalizer.Msg(id, args...) }
