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

// memoryRefusalCases lists the audited forms: one per declaration kind and one per memory operation.
func memoryRefusalCases() []memoryRefusal {
	return append(deleteRefusalCases(), memoryCallRefusalCases()...)
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
			name: "delete a List<String> local",
			src:  "fn main(IOStream io) {\n    List<String> s = [\"a\"];\n    delete s;\n}\n",
			id:   i18n.MsgCompilerDeleteQNotSupportedCompilerHandles,
			args: []any{"s"},
			line: 3,
		},
		{
			name: "delete a struct",
			src:  "type struct { int n; } P;\nfn main(IOStream io) {\n    P p;\n    delete p;\n}\n",
			id:   i18n.MsgCompilerDeleteQNotSupportedCompilerHandles,
			args: []any{"p"},
			line: 4,
		},
	}
}

func memoryCallRefusalCases() []memoryRefusal {
	return []memoryRefusal{
		{
			name: "memory.clear",
			src:  "fn main(IOStream io) {\n    memory.clear();\n}\n",
			id:   i18n.MsgCompilerMemoryPrimitiveSNotLoweredStage1,
			args: []any{"clear"},
			line: 2,
		},
		{
			name: "memory.compact",
			src:  "fn main(IOStream io) {\n    memory.compact();\n}\n",
			id:   i18n.MsgCompilerMemoryPrimitiveSNotLoweredStage1,
			args: []any{"compact"},
			line: 2,
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
