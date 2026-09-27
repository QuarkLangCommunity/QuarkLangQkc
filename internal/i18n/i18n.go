// Package i18n provides multilingual support for user-visible text (Chinese by default; QK_LANG / LC_ALL / LANG select the language).
//
// Usage: just hand the Chinese template to T -- if the table has English it renders English, otherwise the Chinese is used as-is (**never lose information**):
//
//	&RunError{Msg: i18n.T("类型 %s 不能赋给 %s", a, b)}   // the Chinese template is the lookup key (see table.go)
//
// Coverage is enforced by TestCoverage in internal/i18n: every Chinese template appearing in the source must be registered in the table.
package i18n

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// Language identifiers: currently "zh" (default) and "en" are supported.
const (
	ZH = "zh"
	EN = "en"
)

var locale = detect()

// detect decides the initial language:
//
//	QK_LANG (explicit) > LC_ALL / LC_MESSAGES / LANG prefix > English.
//
// Locale-based detection means an English machine reports English and a Chinese machine Chinese,
// with no configuration; QK_LANG always wins for scripts and tests.
func detect() string {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("QK_LANG"))); v != "" {
		if strings.HasPrefix(v, ZH) {
			return ZH
		}
		if strings.HasPrefix(v, EN) {
			return EN
		}
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
		if v == "" {
			continue
		}
		if strings.HasPrefix(v, ZH) {
			return ZH
		}
		if strings.HasPrefix(v, EN) {
			return EN
		}
	}
	return EN // global default: the docs are English-primary
}

// SetLocale sets the language explicitly (used by the tools' --lang CLI flag).
func SetLocale(l string) {
	l = strings.ToLower(strings.TrimSpace(l))
	if l == EN || l == ZH {
		locale = l
	}
}

// Locale returns the current language.
func Locale() string { return locale }

// T renders a message: in English mode it looks up the table, and returns the Chinese as-is when the entry is not registered (so that no information is lost).
func T(format string, args ...any) string {
	if locale == EN {
		if en, ok := table[format]; ok {
			format = en
		}
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// Has reports whether a template already has an English translation (used for coverage statistics).
func Has(format string) bool {
	_, ok := table[format]
	return ok
}

// Untranslated returns the entries of the given template set that are not translated yet (in lexicographic order).
func Untranslated(formats []string) []string {
	var out []string
	for _, f := range formats {
		if !Has(f) {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}
