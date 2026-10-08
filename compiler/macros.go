package main

import (
	"strings"

	"quarklang/internal/lang"
)

// Macro expansion engineering: the compiler reuses the interpreter's token-level macro system (one logic, two users).
// Flow: Lex → SplitMacroDefs → ExpandMacros(mode) → tokens rebuilt into source → cgen parses.

// quoteString re-quotes a decoded string token using exactly the escapes the lexer decodes
// (`"` `\` and the three line escapes), so every other byte survives the rebuild verbatim.
func quoteString(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			sb.WriteByte(s[i])
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// joinTokens rebuilds the token stream into source text (space between identifiers/numbers/strings, punctuation stays tight).
func joinTokens(toks []lang.Token) string {
	var sb strings.Builder
	prevSpace := true
	for _, t := range toks {
		if t.Kind == lang.TEOF {
			break
		}
		text := t.Text
		if t.Kind == lang.TStr {
			// A string token carries the decoded value (no quotes, no raw backquotes), so the rebuilt
			// source needs them back: without this, an argument like "qu" reaches cgen as the identifier qu.
			text = quoteString(text)
		}
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
// Both engines share this frontend, so the standard spelling macro name($a $b) { body } is the one it accepts.
func expandMacros(src string, mode string) (string, error) {
	// Fast path: skip tokenize + expansion when the source has no macro keyword (saves Lex cost)
	if !strings.Contains(src, "macro") {
		return src, nil
	}
	toks, err := lang.Lex(src)
	if err != nil {
		return src, nil // Lex failure falls back to as-is (let cgen report the error)
	}
	macros, rest, err := lang.SplitMacroDefs(toks)
	if err != nil {
		// A macro definition the frontend rejects (the removed #macro spelling included) is a hard error:
		// falling back to the raw source would let cgen report a bare "unexpected '#'" instead of the reason.
		return "", err
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
