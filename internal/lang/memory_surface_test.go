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
