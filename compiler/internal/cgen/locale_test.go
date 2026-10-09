// locale_test.go pins the locale used by this package's tests.
//
// The diagnostic-wording assertions here (feature_test.go, cgen_test.go, parity_test.go)
// compare against the Chinese strings in the i18n table, while the runtime default follows
// the system locale (English on CI runners). The pin must run before m.Run: the process default is
// resolved once at package init and this package caches its own localizer, so t.Setenv has no effect.

package cgen

import (
	"os"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

func TestMain(m *testing.M) {
	// This package renders through its own localizer, so pin that (and the process default, for
	// messages produced by other packages).
	i18n.SetLocale(i18n.ZH)
	SetLocalizer(i18n.New(nil, i18n.ZH))
	os.Exit(m.Run())
}
