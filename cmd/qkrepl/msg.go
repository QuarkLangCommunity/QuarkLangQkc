package main

import "github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"

// localizer is the localizer this tool renders through; installed means SetLocalizer put it there.
// Without one, rendering goes through i18n.Default(), which is what i18n.T did — so a test or a caller
// that pins the language with i18n.SetLocale, or sets QK_LANG before the first render, is obeyed.
var (
	localizer i18n.Localizer
	installed bool
)

// SetLocalizer installs the localizer used for this tool's messages and returns the previous one, so a
// test can restore it instead of mutating process state.
func SetLocalizer(L i18n.Localizer) i18n.Localizer {
	prev := localizer
	prevSet := installed
	localizer, installed = L, true
	if prevSet {
		return prev
	}
	return i18n.Default()
}

// msgID renders a message of this repository by its stable ID: the ID survives a rephrasing, and the
// same text in two contexts stays two messages.
func msgID(id i18n.Msg, args ...any) string {
	if installed {
		return localizer.Msg(id, args...)
	}
	return i18n.Default().Msg(id, args...)
}
