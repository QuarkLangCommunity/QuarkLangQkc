package main

// macros_test.go — the compiler's macro step must hand cgen a **token stream**, never source text it
// rebuilt from one. A token carries a literal's value, not its spelling: a string token's Text is the
// decoded bytes with the quotes and escapes already resolved (qkparser/token.go, lexString), so printing
// the expanded stream back to source turned `io.println("x")` into `io.println(x)` — `qkc` then reported
// `ParseError: expected ')', got identifier`, and the committed examples/macro.qk printed nothing.
// The interpreter has always parsed the stream (internal/lang/compile.go); these tests pin that the
// compiler does the same and that a macro error reaches the user as a macro error.

import (
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/compiler/internal/cgen"
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// idMacroSource is a macro call whose argument is a string literal — the shape that used to lose its quotes.
const idMacroSource = `#macro id (x) {
    #return x
}

fn main(IOStream io) {
    io.println(id("hello world"));
}
`

// escapedMacroSource covers the escapes and the raw-string spelling the printed form used to mangle.
const escapedMacroSource = "#macro id (x) {\n    #return x\n}\n\nfn main(IOStream io) {\n" +
	"    io.println(id(\"tab\\there \\\"quoted\\\" back\\\\slash\"));\n" +
	"    io.println(id(`raw, no escapes`));\n}\n"

// TestExpandMacrosKeepsStringLiteralTokens checks that expansion returns the literal's own token: the same
// kind and the same value the lexer produced, so no later step can spell it as an identifier.
func TestExpandMacrosKeepsStringLiteralTokens(t *testing.T) {
	lexed, err := lang.Lex(idMacroSource)
	if err != nil {
		t.Fatal(err)
	}
	var want lang.Token
	found := false
	for _, tok := range lexed {
		if tok.Kind == lang.TStr {
			want, found = tok, true
			break
		}
	}
	if !found {
		t.Fatal("test source has no string literal to compare against")
	}
	toks, err := expandMacros(idMacroSource, "compile")
	if err != nil {
		t.Fatalf("expandMacros: %v", err)
	}
	for _, tok := range toks {
		if tok.Kind == lang.TStr && tok.Text == want.Text {
			return // the literal survived as a string token
		}
	}
	t.Fatalf("the expanded stream lost the string literal %q (kind %s): %d tokens", want.Text, want.Kind, len(toks))
}

// TestTranspileTokensKeepsMacroStringLiterals compiles the expanded stream and checks the literal reaches
// the IR intact; the printed-source path could not compile these programs at all (a ParseError, or a
// LexError on the escapes).
func TestTranspileTokensKeepsMacroStringLiterals(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		want      []string
	}{
		{"plain string argument", idMacroSource, []string{"hello world"}},
		{"escapes and a raw string", escapedMacroSource, []string{"here", "quoted", "slash", "raw, no escapes"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			toks, err := expandMacros(tc.src, "compile")
			if err != nil {
				t.Fatalf("expandMacros: %v", err)
			}
			ir, err := cgen.TranspileTokens(toks, tc.src, "t.qk")
			if err != nil {
				t.Fatalf("TranspileTokens: %v", err)
			}
			for _, w := range tc.want {
				if !strings.Contains(ir, w) {
					t.Fatalf("the IR does not carry %q — the literal was rewritten on the way to the parser", w)
				}
			}
		})
	}
}

// TestExpandMacrosReportsMacroErrors checks that every macro failure is returned as itself: the old version
// fell back to the unexpanded source, so the user saw whatever the parser made of the text (a ParseError
// about a stray ')') instead of the macro error.
func TestExpandMacrosReportsMacroErrors(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"#error directive", "#macro boom () {\n    #error(\"bad thing\")\n}\n\nfn main(IOStream io) {\n    boom();\n}\n", "bad thing"},
		{"wrong argument count", "#macro add (a, b) {\n    #return a + b\n}\n\nfn main(IOStream io) {\n    io.println(add(1));\n}\n", "add"},
		{"malformed definition", "#macro (x) {\n}\n\nfn main(IOStream io) {\n}\n", "#macro"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			toks, err := expandMacros(tc.src, "compile")
			if err == nil {
				t.Fatalf("a macro failure must be an error, got %d tokens and no error", len(toks))
			}
			if toks != nil {
				t.Fatalf("a failed expansion must not produce a stream, got %d tokens", len(toks))
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "expected ')'") {
				t.Errorf("error %q is the parse error about ')' the macro error used to be hidden behind", err)
			}
		})
	}
}

// TestExpandMacrosLeavesMacroFreeSourceUntouched checks the no-macro path: the stream is the file, token for
// token, so the compiler's fast path and the interpreter's agree about macro-free programs.
func TestExpandMacrosLeavesMacroFreeSourceUntouched(t *testing.T) {
	const src = "fn main(IOStream io) {\n    io.println(\"plain\");\n}\n"
	ref, err := lang.Lex(src)
	if err != nil {
		t.Fatal(err)
	}
	toks, err := expandMacros(src, "compile")
	if err != nil {
		t.Fatalf("expandMacros on macro-free source: %v", err)
	}
	if len(toks) != len(ref) {
		t.Fatalf("token count changed: %d → %d", len(ref), len(toks))
	}
	for i := range ref {
		if toks[i] != ref[i] {
			t.Fatalf("token %d changed: %+v → %+v", i, ref[i], toks[i])
		}
	}
}
