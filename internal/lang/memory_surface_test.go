package lang

import (
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

// TestMemoryPrimitivesRun pins the interpreter's half of §10: setBlock, compact and the derived clear
// run, while Fragmentation is named by the standard but has no member to call.
func TestMemoryPrimitivesRun(t *testing.T) {
	withLocalizer(t, i18n.New(i18n.EN))
	out, err := runSrc(t, `fn main(IOStream io) {
    memory.setBlock(1024);
    memory.compact();
    memory.clear();
    io.println(1);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "1\n" {
		t.Fatalf("got %q", out)
	}
	_, err = runSrc(t, `fn main(IOStream io) { io.println(memory.Fragmentation()); }`)
	if err == nil || !strings.Contains(err.Error(), "Fragmentation") {
		t.Fatalf("got %v", err)
	}
}

// TestFragmentationRefusalNamesTheSituation pins the shared type checker's message for the one
// standard primitive neither engine exposes. Stage 1 gives the compiled runtime an internal
// fragmentation accessor, but exposing it (and making the two engines agree) is still D8.
func TestFragmentationRefusalNamesTheSituation(t *testing.T) {
	for _, L := range []i18n.Lang{i18n.EN, i18n.ZH} {
		withLocalizer(t, i18n.New(L))
		_, err := runSrc(t, `fn main(IOStream io) { io.println(memory.Fragmentation()); }`)
		if err == nil {
			t.Fatalf("%s: expected a refusal", L)
		}
		want := i18n.New(L).Msg(i18n.MsgTypecheckMemoryFragmentationNotExposedYet)
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: error %q does not carry %q", L, err.Error(), want)
		}
	}
}
