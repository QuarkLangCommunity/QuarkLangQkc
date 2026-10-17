package lang

import (
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

// TestExpandRunAreNotImplemented pins the other two §7.2 directives the standard names and neither
// engine implements. They are recorded as gaps (docs/ARCHITECTURE.md §6.8, the generated status
// report), not given an invented meaning: the program must be refused, not accepted partially.
func TestExpandRunAreNotImplemented(t *testing.T) {
	withLocalizer(t, i18n.New(i18n.EN))
	cases := []struct{ name, src string }{
		{"run", "#run main\nfn main(IOStream io) { io.println(1); }\n"},
		{"expand", "#expand main in program\nfn main(IOStream io) { io.println(1); }\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runSrc(t, tc.src)
			if err == nil || !strings.Contains(err.Error(), "expected a top-level declaration") {
				t.Fatalf("got %v", err)
			}
		})
	}
}
