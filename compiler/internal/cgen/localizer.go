package cgen

import "github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"

// Lowering diagnostics are rendered through one localizer, so the lowering code itself does not
// depend on the i18n package and a single call site decides the language.
var cgenLocalizer = i18n.Default()

// SetLocalizer installs the localizer used for lowering diagnostics.
func SetLocalizer(L i18n.Localizer) i18n.Localizer {
	prev := cgenLocalizer
	cgenLocalizer = L
	return prev
}

// msgID renders a generated message by ID: one index into the blob, in the localizer's language. It is
// the only way the lowering layer renders a message — the text-keyed msg it replaced went at step 5.
func msgID(id i18n.Msg, args ...any) string { return cgenLocalizer.Msg(id, args...) }
