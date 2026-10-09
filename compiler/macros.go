package main

import (
	"strings"

	"quarklang/internal/lang"
)

// Macro expansion engineering: the compiler reuses the interpreter's token-level macro system (one logic, two users).
// Flow: Lex → SplitMacroDefs → ExpandMacros(mode) → tokens rebuilt into source → cgen parses.

// joinTokens rebuilds the token stream into source text (space between identifiers/numbers/strings, punctuation stays tight).
func joinTokens(toks []lang.Token) string {
	var sb strings.Builder
	prevSpace := true
	for _, t := range toks {
		if t.Kind == lang.TEOF {
			break
		}
		text := t.Text
		if text == "" {
			continue
		}
		// Whether a leading space is needed: the previous token did not end as a symbol and the current one is not a symbol
		curSym := isSymStart(text)
		if !prevSpace && !curSym && !strings.HasSuffix(sb.String(), " ") {
			sb.WriteByte(' ')
		}
		sb.WriteString(text)
		prevSpace = curSym
	}
	return sb.String()
}

func isSymStart(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '"')
}

// expandMacros performs token-level macro expansion on source (runtime branch; returns as-is when there is no macro).
func expandMacros(src string, mode string) (string, error) {
	// Fast path: skip tokenize + expansion when the source has no macro keyword (saves Lex cost)
	if !strings.Contains(src, "#macro") {
		return src, nil
	}
	toks, err := lang.Lex(src)
	if err != nil {
		return src, nil // Lex failure falls back to as-is (let cgen report the error)
	}
	macros, rest, err := lang.SplitMacroDefs(toks)
	if err != nil {
		return src, nil
	}
	if len(macros) == 0 {
		return src, nil
	}
	exp, err := lang.ExpandMacros(rest, macros, mode)
	if err != nil {
		return src, nil
	}
	return joinTokens(exp), nil
}
