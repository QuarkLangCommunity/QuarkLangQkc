package main

// lib_test.go —— 库制品：导出面 ABI 规范化 / 容器读写 / 清单
// （这些用例锁住"库与调用方 ABI 一致"这一关键不变量：曾经导致运行时解引用 0x15 段错误）

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
	// 导出函数：参数应为**值**
	if !strings.Contains(out, "define i32 @vfc_room_cap(i32 %p0_val)") {
		t.Fatalf("导出函数参数未转值：\n%s", firstLines(out, 6))
	}
	// 入口处应有 alloca + store，保证函数体原有 %p0 用法仍然有效
	if !strings.Contains(out, "%p0 = alloca i32") || !strings.Contains(out, "store i32 %p0_val, i32* %p0") {
		t.Fatalf("缺少入口桥接（alloca/store）：\n%s", firstLines(out, 10))
	}
	// 函数体未被破坏
	if !strings.Contains(out, "load i32, i32* %p0") {
		t.Fatal("函数体被破坏")
	}
	// 非导出函数保持原样（内部约定不变）
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
	// 篡改变体 → sha256 校验必须失败
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

// 读写辅助：直接用 os（测试文件内自足）
func readFileBytes(p string) ([]byte, error)  { return os.ReadFile(p) }
func writeFileBytes(p string, b []byte) error { return os.WriteFile(p, b, 0o644) }
