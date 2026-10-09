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

// msg renders a message template through this package's localizer.
func msg(template string, args ...any) string { return cgenLocalizer.T(template, args...) }
