package main

// lib_test.go — library artifacts: exported-surface ABI normalization / container read-write / manifest
// (these cases lock down the key invariant "library and caller share the same ABI": it once caused a runtime dereference segfault at 0x15)

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

const sampleLibIR = `source_filename = "ext.qk"
define i32 @vfc_room_cap(i32* noundef %p0) norecurse
{
entry:
  %1 = load i32, i32* %p0
  %2 = add i32 %1, 37
  ret i32 %2
}

define i32 @internal_helper(i32* noundef %x)
{
entry:
  %1 = load i32, i32* %x
  ret i32 %1
}
`

func TestNormalizeExportedABI(t *testing.T) {
	exports := collectExports(sampleLibIR)
	if len(exports) != 2 {
		t.Fatalf("2 top-level functions should be recognized, got %d", len(exports))
	}
	out, sigs := normalizeExportedABI(sampleLibIR, []QKExport{{Name: "vfc_room_cap"}})
	// Exported function: parameters must be **values**
	if !strings.Contains(out, "define i32 @vfc_room_cap(i32 %p0_val)") {
		t.Fatalf("an exported function's parameters were not converted:\n%s", firstLines(out, 6))
	}
	// The entry must have alloca + store, so the body's existing %p0 usage still works
	if !strings.Contains(out, "%p0 = alloca i32") || !strings.Contains(out, "store i32 %p0_val, i32* %p0") {
		t.Fatalf("missing the entry bridge (alloca/store):\n%s", firstLines(out, 10))
	}
	// Body not broken
	if !strings.Contains(out, "load i32, i32* %p0") {
		t.Fatal("the function body was mangled")
	}
	// Non-exported functions stay as-is (internal convention unchanged)
	if !strings.Contains(out, "define i32 @internal_helper(i32* noundef %x)") {
		t.Fatal("a non-exported function should not be rewritten")
	}
	if sigs["vfc_room_cap"] != "(i32)" {
		t.Fatalf("the canonical signature should be (i32), got %q", sigs["vfc_room_cap"])
	}
}

func TestQKLibRoundTrip(t *testing.T) {
	man := QKLibManifest{Name: "vfc-ext", Version: "0.1.0",
		Exports: []QKExport{{Name: "vfc_room_cap", Sig: "(i32)", QLSig: "fn vfc_room_cap(int base) int"}}}
	variants := map[string]string{
		"linux-x86_64":   `define i32 @vfc_room_cap(i32 %v) { ret i32 %v }`,
		"windows-x86_64": `define i32 @vfc_room_cap(i32 %v) { ret i32 99 }`,
	}
	path := t.TempDir() + "/x.qklib"
	if err := writeQKLib(path, man, variants); err != nil {
		t.Fatal(err)
	}
	got, vs, err := readQKLib(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "vfc-ext" || got.Version != "0.1.0" || got.Payload != "ir" {
		t.Fatalf("wrong manifest round trip: %+v", got)
	}
	if got.Allocator != qklibAllocatorABI {
		t.Fatalf("a freshly written artifact carries allocator stamp %q, want %q", got.Allocator, qklibAllocatorABI)
	}
	if len(got.Exports) != 1 || got.Exports[0].QLSig == "" {
		t.Fatalf("the export signatures were lost: %+v", got.Exports)
	}
	if len(got.Targets) != 2 || len(vs) != 2 {
		t.Fatalf("wrong variant count: %v / %d", got.Targets, len(vs))
	}
	name, ir, exact := pickVariant(vs, "windows-x86_64")
	if !exact || name != "windows-x86_64" || !strings.Contains(ir, "99") {
		t.Fatalf("wrong variant selection: %s exact=%v", name, exact)
	}
	name2, _, exact2 := pickVariant(vs, "windows-arm64")
	if exact2 || name2 == "" {
		t.Fatalf("wrong fallback: %s exact=%v", name2, exact2)
	}
	// Tamper with a variant → sha256 check must fail
	raw, _ := readFileBytes(path)
	raw[len(raw)-3] ^= 0xFF
	badPath := t.TempDir() + "/bad.qklib"
	if werr := writeFileBytes(badPath, raw); werr != nil {
		t.Fatal(werr)
	}
	if _, _, rerr := readQKLib(badPath); rerr == nil {
		t.Fatal("a tampered variant should fail verification")
	}
}

// writeLegacyQKLib writes an artifact in the pre-stage-1 shape, whose manifest has no allocator field.
func writeLegacyQKLib(t *testing.T, path string, variants map[string]string) {
	t.Helper()
	hashes := map[string]string{}
	for target, ir := range variants {
		sum := sha256.Sum256([]byte(ir))
		hashes[target] = hex.EncodeToString(sum[:])
	}
	manifest := map[string]any{
		"name": "legacy-ext", "version": "0.1.0", "abi": "qk-ir-1", "payload": "ir",
		"targets": []string{"linux-x86_64"}, "sha256": hashes,
	}
	body, err := json.Marshal(map[string]any{"manifest": manifest, "variants": variants})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append([]byte(qklibMagic), body...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestQKLibWithoutAllocatorStampIsRefusedBothLocales pins the refusal of a pre-stage-1 artifact in English and Chinese.
func TestQKLibWithoutAllocatorStampIsRefusedBothLocales(t *testing.T) {
	path := t.TempDir() + "/legacy.qklib"
	writeLegacyQKLib(t, path, map[string]string{"linux-x86_64": `define i32 @f() { ret i32 1 }`})
	wantEN := fmt.Sprintf(`library %s has no compiled-memory allocator stamp: this engine requires %q because the allocator changed from libc malloc/free to counted slabs in compiled-memory stage 1; the library must be rebuilt with this qkc`, path, qklibAllocatorABI)
	wantZH := i18n.New(i18n.ZH).Msg(i18n.MsgCompilerLibraryNoAllocatorStampRebuildS, path, qklibAllocatorABI)
	if wantZH == wantEN {
		t.Fatal("the Chinese rendering is the English one, so the locale check would be vacuous")
	}
	for _, lang := range []i18n.Lang{i18n.EN, i18n.ZH} {
		t.Run(lang.String(), func(t *testing.T) {
			previous := SetLocalizer(i18n.New(lang))
			defer SetLocalizer(previous)
			_, _, err := readQKLib(path)
			if err == nil {
				t.Fatal("a library with no allocator stamp was accepted")
			}
			want := wantEN
			if lang == i18n.ZH {
				want = wantZH
			}
			if got := err.Error(); got != want {
				t.Fatalf("refusal rendered %q, want %q", got, want)
			}
		})
	}
}

func TestDeclBlockFor(t *testing.T) {
	name, blk := DeclBlockFor(QKLibManifest{Name: "vfc-ext", Exports: []QKExport{
		{Name: "vfc_room_cap", QLSig: "fn vfc_room_cap(int base) int"}}})
	if name != "vfc-ext" && name != "vfc_ext" {
		t.Fatalf("wrong library name sanitizing: %q", name)
	}
	if !strings.Contains(blk, "library ") || !strings.Contains(blk, "fn vfc_room_cap(int base) int;") {
		t.Fatalf("wrong declaration block:\n%s", blk)
	}
}

func firstLines(s string, n int) string {
	parts := strings.Split(s, "\n")
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, "\n")
}

// read/write helpers: use os directly (self-contained in the test file)
func readFileBytes(p string) ([]byte, error)  { return os.ReadFile(p) }
func writeFileBytes(p string, b []byte) error { return os.WriteFile(p, b, 0o644) }
