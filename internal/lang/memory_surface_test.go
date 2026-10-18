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

// TestMemoryCompactReturnsControl pins D6's answer: compact() returns no value and the next statement runs.
func TestMemoryCompactReturnsControl(t *testing.T) {
	withLocalizer(t, i18n.New(i18n.EN))
	out, err := runSrc(t, `fn main(IOStream io) {
    memory.compact();
    io.println(7);
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "7\n" {
		t.Fatalf("got %q", out)
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

// TestMemoryManagerClearDropsEmptyBlocks pins the interpreter's half of the derived wipe: clear is the
// operation that removes the storage nobody occupies, and it must not touch a block with a live amount
// in it. The compiled runtime's zero-fill half is exercised by
// compiler/testdata/memory/qthreads_mem_test.c: the interpreter's blocks are logical (Go values), so it
// has no released bytes to zero, and the status report records that gap rather than inventing a wipe.
func TestMemoryManagerClearDropsEmptyBlocks(t *testing.T) {
	m := NewMemoryManager()
	id := m.Alloc(100, 0)
	if m.BlockCount() != 1 {
		t.Fatalf("Alloc made %d blocks, want 1", m.BlockCount())
	}
	m.Delete(id)
	m.Clear()
	if m.BlockCount() != 0 {
		t.Fatalf("Clear kept an empty block: %d blocks remain", m.BlockCount())
	}
	live := m.Alloc(100, 0)
	if live == 0 {
		t.Fatal("Alloc after Clear failed")
	}
	m.Clear()
	if m.BlockCount() != 1 {
		t.Fatalf("Clear dropped a block with a live allocation: %d blocks remain", m.BlockCount())
	}
}
