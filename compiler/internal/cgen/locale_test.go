// locale_test.go pins the locale used by this package's tests.
//
// The diagnostic-wording assertions here (feature_test.go, cgen_test.go, parity_test.go)
// compare against the Chinese strings in the i18n table, while the runtime default follows
// the system locale (English on CI runners). SetLocale must run before m.Run because i18n
// resolves the locale once at package init, so t.Setenv("QK_LANG", ...) has no effect.

package cgen

import (
	"os"
	"testing"

	"quarklang/internal/i18n"
)

func TestMain(m *testing.M) {
	i18n.SetLocale(i18n.ZH)
	os.Exit(m.Run())
}
