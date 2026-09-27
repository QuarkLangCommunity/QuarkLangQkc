package main

// lib_test.go — library artifacts: exported-surface ABI normalization / container read-write / manifest
// (these cases lock down the key invariant "library and caller share the same ABI": it once caused a runtime dereference segfault at 0x15)

import (
	"os"
	"strings"
	"testing"
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
		t.Fatalf("应识别 2 个顶层函数，得到 %d", len(exports))
	}
	out, sigs := normalizeExportedABI(sampleLibIR, []QKExport{{Name: "vfc_room_cap"}})
	// Exported function: parameters must be **values**
	if !strings.Contains(out, "define i32 @vfc_room_cap(i32 %p0_val)") {
		t.Fatalf("导出函数参数未转值：\n%s", firstLines(out, 6))
	}
	// The entry must have alloca + store, so the body's existing %p0 usage still works
	if !strings.Contains(out, "%p0 = alloca i32") || !strings.Contains(out, "store i32 %p0_val, i32* %p0") {
		t.Fatalf("缺少入口桥接（alloca/store）：\n%s", firstLines(out, 10))
	}
	// Body not broken
	if !strings.Contains(out, "load i32, i32* %p0") {
		t.Fatal("函数体被破坏")
	}
	// Non-exported functions stay as-is (internal convention unchanged)
	if !strings.Contains(out, "define i32 @internal_helper(i32* noundef %x)") {
		t.Fatal("非导出函数不应被改写")
	}
	if sigs["vfc_room_cap"] != "(i32)" {
		t.Fatalf("规范化签名应为 (i32)，得到 %q", sigs["vfc_room_cap"])
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
		t.Fatalf("清单往返异常：%+v", got)
	}
	if len(got.Exports) != 1 || got.Exports[0].QLSig == "" {
		t.Fatalf("导出签名丢失：%+v", got.Exports)
	}
	if len(got.Targets) != 2 || len(vs) != 2 {
		t.Fatalf("变体数量异常：%v / %d", got.Targets, len(vs))
	}
	name, ir, exact := pickVariant(vs, "windows-x86_64")
	if !exact || name != "windows-x86_64" || !strings.Contains(ir, "99") {
		t.Fatalf("变体选择异常：%s exact=%v", name, exact)
	}
	name2, _, exact2 := pickVariant(vs, "windows-arm64")
	if exact2 || name2 == "" {
		t.Fatalf("回退逻辑异常：%s exact=%v", name2, exact2)
	}
	// Tamper with a variant → sha256 check must fail
	raw, _ := readFileBytes(path)
	raw[len(raw)-3] ^= 0xFF
	badPath := t.TempDir() + "/bad.qklib"
	if werr := writeFileBytes(badPath, raw); werr != nil {
		t.Fatal(werr)
	}
	if _, _, rerr := readQKLib(badPath); rerr == nil {
		t.Fatal("变体被篡改应校验失败")
	}
}

func TestDeclBlockFor(t *testing.T) {
	name, blk := DeclBlockFor(QKLibManifest{Name: "vfc-ext", Exports: []QKExport{
		{Name: "vfc_room_cap", QLSig: "fn vfc_room_cap(int base) int"}}})
	if name != "vfc-ext" && name != "vfc_ext" {
		t.Fatalf("库名清洗异常：%q", name)
	}
	if !strings.Contains(blk, "library ") || !strings.Contains(blk, "fn vfc_room_cap(int base) int;") {
		t.Fatalf("声明块生成异常：\n%s", blk)
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
