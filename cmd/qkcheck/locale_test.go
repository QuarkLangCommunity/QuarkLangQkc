// locale_test.go pins the locale used by this package's tests.
//
// The CLI's output wording follows the locale: message text comes from the i18n catalog through the
// process default and through internal/lang's injected localizer, both of which read QK_LANG/LANG. The
// assertions here compare English wording (TestCLITypeErrorReported looks for "cannot assign"), so on a
// machine whose LANG is Chinese the test would otherwise assert against the Chinese rendering — and it
// used to pass there only because that message had no Chinese catalog value at all.
//
// The pin must run before m.Run: the process default is resolved once, so t.Setenv has no effect.
// The sibling packages pin the same way, each in the language its assertions are written in —
// cmd/qkrepl and compiler/internal/cgen assert Chinese wording and pin ZH; this package's assertions
// are English, so the pin is EN.
package main

import (
	"os"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

func TestMain(m *testing.M) {
	i18n.SetLocale(i18n.EN)
	lang.SetLocalizer(i18n.New(nil, i18n.EN)) // interpreter/linter diagnostics follow the same language
	os.Exit(m.Run())
}
