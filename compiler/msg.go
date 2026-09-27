package main

import "quarklang/internal/i18n"

// The compiler renders its diagnostics through one localizer instead of calling i18n at each site.
// SetLocalizer is the single injection point (tools and tests use it; nothing else mutates state).
var compilerLocalizer = i18n.Default()

// SetLocalizer installs the localizer used for compiler messages.
func SetLocalizer(L i18n.Localizer) i18n.Localizer {
	prev := compilerLocalizer
	compilerLocalizer = L
	return prev
}

// msg renders a message template through the compiler's localizer.
func msg(template string, args ...any) string { return compilerLocalizer.T(template, args...) }
