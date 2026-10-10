package main

import (
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// Macro expansion engineering: the compiler reuses the interpreter's token-level macro system (one logic, two users).
// Flow: Lex → SplitMacroDefs → ExpandMacros(mode) → cgen parses the expanded **tokens**.
//
// The compiler may not print that stream back to source text. A token carries a literal's value, not its
// spelling: a string token's Text is the decoded byte sequence, quotes and escapes excluded
// (qkparser/token.go, lexString), so `io.println("x")` came back as `io.println(x)` — a ParseError for a
// macro plus a string, and two empty lines for the committed examples/macro.qk. The interpreter has always
// parsed the tokens (internal/lang/compile.go); cgen.TranspileTokens is how the compiler now does the same.

// expandMacros lexes already-preprocessed source and returns it as one token stream with every macro call
// expanded in the given macro state ("compile" for this engine); the macro definitions themselves are
// dropped, exactly as the interpreter's compile step drops them.
func expandMacros(src string, mode string) ([]lang.Token, error) {
	toks, err := lang.Lex(src)
	if err != nil {
		return nil, err // not a macro error: cgen would report this same lex failure, so report it once here
	}
	macros, rest, err := lang.SplitMacroDefs(toks)
	if err != nil {
		return nil, err // a #macro definition is malformed: a macro error, not a parse error about ')'
	}
	if len(macros) == 0 {
		return rest, nil // no macro in this file: the stream is the file
	}
	exp, err := lang.ExpandMacros(rest, macros, mode)
	if err != nil {
		return nil, err // a macro call could not be expanded: a macro error, not a parse error about ')'
	}
	return exp, nil
}
