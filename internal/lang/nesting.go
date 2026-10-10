package lang

// nesting.go keeps the front end's nesting limit in this repository. The recursive descent that the
// limit protects now lives in qkparser (see syntax.go), and that dependency is pinned to one published
// revision, so the counter that refuses an over-deep descent has to sit at the boundary this repository
// owns: its own front-end entry points.
//
// The counter covers exactly the shape the type checker's own counter cannot see. A nest of
// parentheses, brackets or blocks costs the parser one recursion level per delimiter yet collapses into
// a single AST node, so `((((1))))` never reaches the checker's per-node depth check at all; a deep
// chain of operator or call nodes is the other way round, and the checker refuses that one (limits.go,
// typecheck.go). Both counters read the same MaxExprDepth, so the interpreter and the compiler answer
// an over-deep input with the same positioned diagnostic on every path a user can write.

import "strings"

// refuseOverDeepNesting returns the positioned refusal for the first token that opens a nest deeper
// than MaxExprDepth, or nil when the whole token stream stays inside the limit.
func refuseOverDeepNesting(toks []Token) error {
	depth := 0
	for _, tok := range toks {
		switch tok.Kind {
		case TLParen, TLBracket, TLBrace:
			depth++
			if depth > MaxExprDepth {
				return &ParseError{
					Msg:  msg("nesting is deeper than %d levels, the limit of this implementation: split the expression into smaller statements", MaxExprDepth),
					Line: tok.Line,
					Col:  tok.Col,
				}
			}
		case TRParen, TRBracket, TRBrace:
			if depth > 0 {
				depth--
			}
		}
	}
	return nil
}

// refuseOverDeepNestingInSource refuses an over-deep nest in a source string, whose parsing happens
// inside the module: the opening delimiters bound the nesting, so the token pass runs only for a source
// that could possibly be too deep, and never replaces the error the parser itself would report.
func refuseOverDeepNestingInSource(src string) error {
	if strings.Count(src, "(")+strings.Count(src, "[")+strings.Count(src, "{") <= MaxExprDepth {
		return nil
	}
	toks, err := Lex(src)
	if err != nil {
		return nil
	}
	return refuseOverDeepNesting(toks)
}
