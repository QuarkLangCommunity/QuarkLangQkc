package i18n

// The C catalog's half of the standing gate. TestGeneratedCatalogIsCurrent proves msg_gen.go is what the
// generator writes today; the compiled path reads a second artefact, compiler/internal/rtcat/msg_c.h,
// and an artefact no ordinary test checks is an artefact that drifts. It is the same table, the same
// generator and the same run: only the reader differs (clang, not Go), which is why the check belongs
// here rather than in the compiler package — the tables and the list are this package's.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18ngen"
)

// cOutPath is where the generator writes the C catalog, relative to this package.
func cOutPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "compiler", "internal", "rtcat", "msg_c.h")
}

func readCHeader(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(cOutPath(t))
	if err != nil {
		t.Fatalf("the generated C catalog: %v", err)
	}
	return string(raw)
}

// TestGeneratedCCatalogIsCurrent is the compiled path's staleness gate: the committed C header must be
// exactly what the authored tables produce today. It runs the whole generator, so the C header, the Go
// catalog and the review document are checked by one command, which is what `go generate` writes.
func TestGeneratedCCatalogIsCurrent(t *testing.T) {
	opts := generatedOptions(t, true)
	opts.COutFile = cOutPath(t)
	if problems := i18ngen.Run(opts); len(problems) > 0 {
		t.Fatalf("the committed C catalog is not current:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCCatalogCarriesEveryListedMessage: the header must carry a row per id in source/c_runtime.txt, in
// every shipped language, and it must carry nothing else. It is the check the runtime cannot make — an
// id with no macro is a C compile error at qkc's next build, but an id whose Chinese column went missing
// would render English to a Chinese reader, and only this side can see the difference. The rendering is
// compared as the generator escapes it, through the generator's own escaper, so this test cannot pass
// while disagreeing with the file's format.
func TestCCatalogCarriesEveryListedMessage(t *testing.T) {
	src := loadAuthored(t)
	if len(src.CRuntimeIDs) == 0 {
		t.Fatal("source/c_runtime.txt listed no messages, so the C runtime has no catalog")
	}
	header := readCHeader(t)
	rows, err := src.CRuntimeRows()
	if err != nil {
		t.Fatalf("the runtime's message list: %v", err)
	}
	for _, m := range rows {
		if !strings.Contains(header, i18ngen.CString(m.ID)) {
			t.Errorf("%s is listed in source/c_runtime.txt but the C catalog does not carry its id", m.ID)
		}
		for li, l := range src.Table.Languages {
			if li >= len(m.Values) {
				t.Errorf("%s has no %s value", m.ID, l.Code)
				continue
			}
			if !strings.Contains(header, i18ngen.CString(m.Values[li])) {
				t.Errorf("%s: the C catalog does not carry its %s rendering %q", m.ID, l.Code, truncate(m.Values[li]))
			}
		}
	}
	// The header is generated from the list, so it must carry a macro per listed message and no macro the
	// list does not name: a row nothing can print would be a string compiled into every binary for
	// nothing, and a row the list names but the header dropped would be a C compile error at qkc's next
	// build rather than here.
	if got, want := strings.Count(header, "QK_MSG_RUNTIME_"), len(rows); got != want {
		t.Errorf("the C catalog carries %d message macro(s), want %d (one per listed message)", got, want)
	}
	// The list is the header's only input for ids, so an id appearing twice would number two macros the
	// same and silently make one message unreachable.
	for _, m := range rows {
		macro := i18ngen.CMsgMacro(m.ID)
		if got := strings.Count(header, macro+" ="); got != 1 {
			t.Errorf("%s: the macro %s is defined %d times, want once", m.ID, macro, got)
		}
	}
}

// TestCRuntimeListIsReachableFromTheTables: every id in source/c_runtime.txt must be a message the
// authored tables carry, with a value in every shipped language. Generation refuses otherwise, so this
// asserts the refusal is reachable rather than trusting it — a list that names a message nobody authored
// is the mistake a hand-written list makes, and it would otherwise surface as a stale-header failure
// whose message points at the wrong file.
func TestCRuntimeListIsReachableFromTheTables(t *testing.T) {
	src := loadAuthored(t)
	if len(src.CRuntimeIDs) == 0 {
		t.Fatal("source/c_runtime.txt listed no messages")
	}
	rows, err := src.CRuntimeRows()
	if err != nil {
		t.Fatalf("the runtime's message list: %v", err)
	}
	if len(rows) != len(src.CRuntimeIDs) {
		t.Fatalf("the list has %d id(s) but %d row(s) resolved", len(src.CRuntimeIDs), len(rows))
	}
	for _, m := range rows {
		if len(m.Values) != len(src.Table.Languages) {
			t.Errorf("%s has %d value(s), want one per shipped language", m.ID, len(m.Values))
		}
		for li, l := range src.Table.Languages {
			if li < len(m.Values) && strings.TrimSpace(m.Values[li]) == "" {
				t.Errorf("%s: empty %s value", m.ID, l.Code)
			}
		}
	}
	// A mutation the generator must refuse: the list naming an id no table carries.
	broken := src.Clone()
	broken.CRuntimeIDs = append(broken.CRuntimeIDs, "runtime.no_such_message")
	if _, err := i18ngen.GenerateC(broken); err == nil {
		t.Error("generation accepted a listed id that no table carries, so the list is not checked")
	}
}
