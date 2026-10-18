package cgen

// The compiled memory surface, pinned as refusals rather than as silent no-ops.
//
// `delete` is lowered for one shape only: an owned local List<int> variable. A by-reference parameter
// names the caller's list, so 0.8.5 lowered it and emitted nothing — neither a release nor an error.
// Lowering now refuses it; the other memory-surface shapes below were already hard errors, and these
// tests keep them hard in both shipped languages. The expected wording is read from the generated
// catalog, never spelled here, so the test cannot drift from the table and stays English-only.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

// memoryRefusal is one unsupported memory-surface form and the message ID it must raise.
type memoryRefusal struct {
	name string
	src  string
	id   i18n.Msg
	args []any
	line int
}

// memoryRefusalCases lists the audited forms: one per declaration kind. The memory rows themselves are
// covered by memory_surface_test.go now that stage 2 lowers them.
func memoryRefusalCases() []memoryRefusal {
	return deleteRefusalCases()
}

func deleteRefusalCases() []memoryRefusal {
	return []memoryRefusal{
		{
			name: "delete a by-reference List<int> parameter",
			src:  "fn drop(List<int> xs) void {\n    delete xs;\n}\nfn main(IOStream io) { io.println(1); }\n",
			id:   i18n.MsgCompilerDeleteOnByReferenceParameterQNotSupported,
			args: []any{"xs"},
			line: 2,
		},
		{
			name: "delete a by-reference struct parameter",
			src:  "type struct { int n; } P;\nfn drop(P p) void {\n    delete p;\n}\nfn main(IOStream io) { io.println(1); }\n",
			id:   i18n.MsgCompilerDeleteOnByReferenceParameterQNotSupported,
			args: []any{"p"},
			line: 3,
		},
		{
			name: "delete a struct field",
			src:  "type struct { List<int> xs; } Box;\nfn main(IOStream io) {\n    Box b;\n    delete b.xs;\n}\n",
			id:   i18n.MsgCompilerDeleteOnNonVariableExpressionNot,
			line: 4,
		},
		{
			name: "delete a new[] result expression",
			src:  "fn main(IOStream io) {\n    delete new int[5];\n}\n",
			id:   i18n.MsgCompilerDeleteOnNonVariableExpressionNot,
			line: 2,
		},
		{
			name: "delete a scalar local",
			src:  "fn main(IOStream io) {\n    int x = 1;\n    delete x;\n}\n",
			id:   i18n.MsgCompilerDeleteQNotSupportedCompilerHandles,
			args: []any{"x"},
			line: 3,
		},
		{
			name: "delete a String local",
			src:  "fn main(IOStream io) {\n    String s = \"a\";\n    delete s;\n}\n",
			id:   i18n.MsgCompilerDeleteQNotSupportedCompilerHandles,
			args: []any{"s"},
			line: 3,
		},
	}
}

// TestCompiledMemorySurfaceRefusesBothLocales lowers every audited form in English and Chinese and
// requires the same catalog entry's rendering in each: one message, two languages, never a silent
// acceptance.
func TestCompiledMemorySurfaceRefusesBothLocales(t *testing.T) {
	for _, tc := range memoryRefusalCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, lang := range []i18n.Lang{i18n.EN, i18n.ZH} {
				got := refusalIn(t, lang, tc.src)
				want := i18n.New(lang).Msg(tc.id, tc.args...)
				if !strings.Contains(got, want) {
					t.Errorf("%s: error does not carry the %s rendering %q:\n%s", tc.name, lang, want, got)
				}
				if pos := fmt.Sprintf("test.qk:%d:", tc.line); !strings.Contains(got, pos) {
					t.Errorf("%s: error lacks the source position %q:\n%s", tc.name, pos, got)
				}
			}
		})
	}
}

// refusalIn lowers src under one language and returns the refusal. The package pins Chinese in
// TestMain, so the localizer is swapped per call and restored before the next one.
func refusalIn(t *testing.T, lang i18n.Lang, src string) string {
	t.Helper()
	prev := SetLocalizer(i18n.New(lang))
	defer SetLocalizer(prev)
	return lowerErr(t, src).Error()
}

// TestDeleteOnOwnedListIntStillLowers is the positive half: the supported form keeps its two frees.
func TestDeleteOnOwnedListIntStillLowers(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    List<int> xs = [1, 2, 3];\n"+
		"    io.println(xs.size());\n"+
		"    delete xs;\n"+
		"    io.println(9);\n"+
		"}\n")
	freeCall := "call void @ql_free(i8*"
	if hostAllocatorFallback {
		freeCall = "call void @free(i8*"
	}
	if got := strings.Count(ir, freeCall); got != 2 {
		t.Fatalf("delete on a local List<int> emits %d %s call(s), want 2 (buffer and header):\n%s", got, freeCall, ir)
	}
}

// TestDeleteOnCopydListIntParameterStillLowers keeps the refusal narrow: a copyd parameter is a
// callee-owned deep copy, so releasing it is supported and must not be caught by the new refusal.
func TestDeleteOnCopydListIntParameterStillLowers(t *testing.T) {
	ir := transpile(t, "fn drop(copyd List<int> xs) void {\n"+
		"    delete xs;\n"+
		"}\n"+
		"fn main(IOStream io) {\n"+
		"    List<int> xs = [1];\n"+
		"    drop(xs);\n"+
		"    io.println(xs.size());\n"+
		"}\n")
	freeCall := "call void @ql_free(i8*"
	if hostAllocatorFallback {
		freeCall = "call void @free(i8*"
	}
	if got := strings.Count(ir, freeCall); got != 2 {
		t.Fatalf("delete on a copyd List<int> parameter emits %d %s call(s), want 2:\n%s", got, freeCall, ir)
	}
}

// deleteFreeCall is the release symbol the emitter uses on the host platform (Windows keeps libc).
func deleteFreeCall() string {
	if hostAllocatorFallback {
		return "call void @free(i8*"
	}
	return "call void @ql_free(i8*"
}

// TestDeleteOnListStringAndFloatLowers covers the other two list layouts the compiler emits: every one
// is a header plus a heap buffer, so release is two frees of the same pair, not a special case.
func TestDeleteOnListStringAndFloatLowers(t *testing.T) {
	ir := transpile(t, "fn main(IOStream io) {\n"+
		"    List<String> s = [\"a\", \"b\"];\n"+
		"    io.println(s.size());\n"+
		"    delete s;\n"+
		"    List<float> f = [1.5, 2.5];\n"+
		"    io.println(f.get(1));\n"+
		"    delete f;\n"+
		"}\n")
	if got := strings.Count(ir, deleteFreeCall()); got != 4 {
		t.Fatalf("delete on List<String>/List<float> emits %d release call(s), want 4 (two per list):\n%s", got, ir)
	}
}

// TestDeleteOnStructRunsDeleteMethodThenReleases pins §10's order: __delete__() first, then the
// storage. The call is the ordinary self-method call, so it has the same receiver convention a direct
// call has.
func TestDeleteOnStructRunsDeleteMethodThenReleases(t *testing.T) {
	ir := transpile(t, "type struct { int n; } P;\n"+
		"impl {\n"+
		"    fn __delete__(P self) void { self.n = 0; }\n"+
		"} P;\n"+
		"fn main(IOStream io) {\n"+
		"    P p = .{n: 1};\n"+
		"    delete p;\n"+
		"}\n")
	call := strings.Index(ir, "@P___delete__(")
	free := strings.Index(ir, deleteFreeCall())
	if call < 0 {
		t.Fatalf("delete on a struct does not call __delete__:\n%s", ir)
	}
	if free < 0 || free < call {
		t.Fatalf("the release does not follow __delete__ (call=%d free=%d):\n%s", call, free, ir)
	}
}

// TestDeleteOnStructWithoutDeleteMethodReleases: a struct that declares no __delete__ still owns its
// cell, and releasing it must not be a silent no-op.
func TestDeleteOnStructWithoutDeleteMethodReleases(t *testing.T) {
	ir := transpile(t, "type struct { int n; } P;\n"+
		"fn main(IOStream io) {\n"+
		"    P p = .{n: 1};\n"+
		"    delete p;\n"+
		"}\n")
	if got := strings.Count(ir, deleteFreeCall()); got != 1 {
		t.Fatalf("delete on a struct without __delete__ emits %d release call(s), want 1:\n%s", got, ir)
	}
}

// TestDeleteOnCopydStructParameterLowers: a copyd struct parameter is a callee-owned deep copy, so it
// is releasable; the by-reference refusal must not swallow it.
func TestDeleteOnCopydStructParameterLowers(t *testing.T) {
	ir := transpile(t, "type struct { int n; } P;\n"+
		"fn drop(copyd P p) void { delete p; }\n"+
		"fn main(IOStream io) { P p = .{n: 1}; drop(p); io.println(p.n); }\n")
	if !strings.Contains(ir, deleteFreeCall()) {
		t.Fatalf("delete on a copyd struct parameter emits no release:\n%s", ir)
	}
}
