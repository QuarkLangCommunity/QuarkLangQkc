package lang

import (
	"errors"
	"fmt"
	"strings"
)

// SplitMacroDefs splits every standard macro definition out of a token stream, returning the macro list and the remaining tokens.
//
// Standard spelling (spec/STANDARD.md 6.6):
//
//	macro name($a $b) { body }
//
// `macro` is the keyword, the parameters are one or more `$name` (the sigil is what marks them) and the body is a braced block.
// Invocation keeps the three call forms name(args) / name[args] / name{args}.
func SplitMacroDefs(toks []Token) ([]*MacroDef, []Token, error) {
	// Fast path: without a `macro` keyword token there is nothing to split → return the original slice
	// (zero copy, zero allocation). The vast majority of source files take this path.
	hasMacro := false
	for i := range toks {
		if toks[i].Kind == TMacro {
			hasMacro = true
			break
		}
	}
	if !hasMacro {
		return nil, toks, nil
	}
	var macros []*MacroDef
	rest := make([]Token, 0, len(toks))
	i := 0
	for i < len(toks) {
		t := toks[i]
		if t.Kind == TSharp && i+1 < len(toks) && toks[i+1].Kind == TMacro {
			return nil, nil, errors.New(msg("ParseError: #macro is not the standard spelling (STANDARD 6.6): write macro name($a $b) { ... }, line %d", t.Line))
		}
		if t.Kind != TMacro {
			rest = append(rest, t)
			i++
			continue
		}
		m, ni, err := parseMacroDef(toks, i)
		if err != nil {
			return nil, nil, err
		}
		macros = append(macros, m)
		i = ni
	}
	return macros, rest, nil
}

// parseMacroDef parses one standard definition, macro name($a $b) { body }, starting at the `macro` keyword toks[i]
// and returning it with the index just past the body.
func parseMacroDef(toks []Token, i int) (*MacroDef, int, error) {
	pos := Pos{Line: toks[i].Line, Col: toks[i].Col}
	i++
	if i >= len(toks) || toks[i].Kind != TIdent {
		return nil, 0, errors.New(msg("ParseError: macro needs a name, line %d", pos.Line))
	}
	name := toks[i].Text
	i++
	if i >= len(toks) || toks[i].Kind != TLParen {
		return nil, 0, errors.New(msg("ParseError: macro %s is missing its ($name) parameter list, line %d", name, pos.Line))
	}
	params, i, err := macroParams(toks, i, name)
	if err != nil {
		return nil, 0, err
	}
	if i >= len(toks) || toks[i].Kind != TLBrace {
		return nil, 0, errors.New(msg("ParseError: macro %s body must be a { ... } block, line %d", name, pos.Line))
	}
	body, i, err := takeBalancedPair(toks, i, TRBrace)
	if err != nil {
		return nil, 0, err
	}
	return &MacroDef{Name: name, Params: params, Body: body, Pos: pos}, i, nil
}

// macroParams reads the ($a $b) parameter list of macro name at toks[i] (the opening parenthesis)
// and returns the parameter names with the index just past the closing parenthesis.
func macroParams(toks []Token, i int, name string) ([]string, int, error) {
	inner, next, err := takeBalancedPair(toks, i, TRParen)
	if err != nil {
		return nil, 0, err
	}
	var params []string
	for j := 0; j < len(inner); {
		if inner[j].Kind != TDollar || j+1 >= len(inner) || inner[j+1].Kind != TIdent {
			return nil, 0, errors.New(msg("ParseError: macro %s parameters are written $name (one or more, space-separated), line %d", name, inner[j].Line))
		}
		params = append(params, inner[j+1].Text)
		j += 2
	}
	if len(params) == 0 {
		return nil, 0, errors.New(msg("ParseError: macro %s needs at least one $name parameter, line %d", name, toks[i].Line))
	}
	return params, next, nil
}

// closeOf returns the closing delimiter matching an opening one.
func closeOf(k TokenKind) (TokenKind, bool) {
	switch k {
	case TLParen:
		return TRParen, true
	case TLBracket:
		return TRBracket, true
	case TLBrace:
		return TRBrace, true
	}
	return 0, false
}

// takeBalancedPair takes a balanced block starting at toks[i] (an opening delimiter), counting mixed ()[]{} pairs,
// and returns the tokens inside plus the end position; the closing delimiter must correspond to openK.
func takeBalancedPair(toks []Token, i int, closeK TokenKind) ([]Token, int, error) {
	depth := 0
	for j := i; j < len(toks); j++ {
		switch toks[j].Kind {
		case TLParen, TLBracket, TLBrace:
			depth++
		case TRParen, TRBracket, TRBrace:
			depth--
			if depth == 0 {
				if toks[j].Kind != closeK {
					return nil, 0, errors.New(msg("ParseError: 分隔符不配对，第 %d 行", toks[i].Line))
				}
				return toks[i+1 : j], j + 1, nil
			}
		}
	}
	return nil, 0, errors.New(msg("ParseError: 分隔符不配对，第 %d 行", toks[i].Line))
}

// splitTop splits a token sequence on top-level commas (ignoring commas inside ()[]{}).
func splitTop(toks []Token) [][]Token {
	var parts [][]Token
	depth := 0
	start := 0
	for i, t := range toks {
		switch t.Kind {
		case TLParen, TLBracket, TLBrace:
			depth++
		case TRParen, TRBracket, TRBrace:
			depth--
		case TComma:
			if depth == 0 {
				parts = append(parts, toks[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, toks[start:])
	return parts
}

// ExpandMacros expands named macro calls in a token stream: name(args)/name[args]/name{args}.
// mode is the dynamic preprocessing state ("run"/"compile"/"explain"); bodies are not re-expanded recursively.
func ExpandMacros(toks []Token, macros []*MacroDef, mode string) ([]Token, error) {
	if len(macros) == 0 {
		return toks, nil
	}
	byName := make(map[string]*MacroDef, len(macros))
	for _, m := range macros {
		byName[m.Name] = m
	}
	var out []Token
	i := 0
	for i < len(toks) {
		t := toks[i]
		if t.Kind == TIdent && i+1 < len(toks) {
			if m, ok := byName[t.Text]; ok {
				if closeK, okc := closeOf(toks[i+1].Kind); okc && !macroCallExcluded(i, toks) {
					argsToks, ni, err := takeBalancedPair(toks, i+1, closeK)
					if err != nil {
						return nil, err
					}
					callArgs := splitTop(argsToks)
					if len(callArgs) == 1 && len(callArgs[0]) == 0 {
						callArgs = nil
					}
					if len(callArgs) != len(m.Params) {
						return nil, errors.New(msg("ParseError: 宏 %s 需要 %d 个参数，调用给了 %d 个（第 %d 行）", m.Name, len(m.Params), len(callArgs), t.Line))
					}
					subst := make(map[string][]Token, len(callArgs))
					for p, pname := range m.Params {
						subst[pname] = callArgs[p]
					}
					body, _, err := expandBody(m.Body, subst, mode, m.Params)
					if err != nil {
						return nil, errors.New(msg("宏 %s 展开失败：%v", m.Name, err))
					}
					out = append(out, body...)
					i = ni
					continue
				}
			}
		}
		out = append(out, t)
		i++
	}
	return out, nil
}

// macroCallExcluded: macros are not expanded at declaration positions, member accesses, scoped calls or preprocessor commands.
func macroCallExcluded(i int, toks []Token) bool {
	if i == 0 {
		return false
	}
	switch toks[i-1].Kind {
	case TFunc, TStruct, TImpl, TInterface, TMacro, TDot, TScope, TSharp:
		return true
	}
	return false
}

// substParam splices the argument tokens bound to the `$name` parameter reference at body[i],
// returning them with the index just past the reference.
func substParam(body []Token, i int, subst map[string][]Token) ([]Token, int, error) {
	if i+1 >= len(body) || body[i+1].Kind != TIdent {
		return nil, 0, errors.New(msg("ParseError: $ must be followed by a parameter name, line %d", body[i].Line))
	}
	name := body[i+1].Text
	arg, ok := subst[name]
	if !ok {
		return nil, 0, errors.New(msg("ParseError: $%s is not a parameter of this macro, line %d", name, body[i].Line))
	}
	return arg, i + 2, nil
}

// expandBody expands a macro body: a `$name` reference is replaced by the argument bound to that parameter;
// #when (compile|run) { ... } selects a block by state;
// #return <token...> is the macro's return value (the expansion result is the tokens after the return directive, with parameters substituted) and terminates the whole expansion immediately;
// #error("msg") reports an error directly; #insert/#execute/#ast come from the removed pattern syntax.
// Returns (output, whether a #return terminated it).
func expandBody(body []Token, subst map[string][]Token, mode string, paramOrder []string) ([]Token, bool, error) {
	var out []Token
	i := 0
	for i < len(body) {
		t := body[i]
		if t.Kind == TDollar {
			arg, ni, err := substParam(body, i, subst)
			if err != nil {
				return nil, false, err
			}
			out = append(out, arg...)
			i = ni
			continue
		}
		if t.Kind != TSharp {
			out = append(out, t)
			i++
			continue
		}
		if i+1 >= len(body) || (body[i+1].Kind != TIdent && body[i+1].Kind != TReturn) {
			return nil, false, errors.New(msg("第 %d 行：# 后必须是预处理命令（when/return/error）", t.Line))
		}
		cmd := body[i+1].Text
		i += 2
		if cmd == "return" {
			// #return <expr>: the shown result is everything after the return directive (parameters substituted + later directives still processed),
			// e.g. #return #insert(#ast($a)); and it terminates the whole macro expansion.
			sub, _, err := expandBody(body[i:], subst, mode, paramOrder)
			if err != nil {
				return nil, false, err
			}
			out = append(out, sub...)
			return out, true, nil
		}
		if i >= len(body) {
			return nil, false, errors.New(msg("第 %d 行：#%s 需要 ( ... )", t.Line, cmd))
		}
		closeK, ok := closeOf(body[i].Kind)
		if !ok {
			return nil, false, errors.New(msg("第 %d 行：#%s 需要 ( ... )", t.Line, cmd))
		}
		args, ni, err := takeBalancedPair(body, i, closeK)
		if err != nil {
			return nil, false, err
		}
		i = ni
		switch cmd {
		case "when":
			if len(args) < 1 || args[0].Kind != TIdent {
				return nil, false, errors.New(msg("第 %d 行：#when 需要 (compile|run)", t.Line))
			}
			if i >= len(body) || body[i].Kind != TLBrace {
				return nil, false, errors.New(msg("第 %d 行：#when 需要 { ... } 块", t.Line))
			}
			blk, ni, err := takeBalancedPair(body, i, TRBrace)
			if err != nil {
				return nil, false, err
			}
			i = ni
			if args[0].Text == mode || (mode == "explain" && args[0].Text == "run") {
				sub, stop, err := expandBody(blk, subst, mode, paramOrder)
				if err != nil {
					return nil, false, err
				}
				out = append(out, sub...)
				if stop {
					return out, true, nil
				}
			}
		case "error":
			if len(args) < 1 || args[0].Kind != TStr {
				return nil, false, errors.New(msg("第 %d 行：#error 需要 (\"消息\")", t.Line))
			}
			return nil, false, errors.New(msg("第 %d 行：预处理错误 #error(%s)", t.Line, args[0].Text))
		case "insert":
			// #insert(#ast($name)): splice in the tokens of parameter name (a bare name is still accepted); #insert(#ast(...)): splice in all parameters in order
			inner, err := parseAstArg(args)
			if err != nil {
				return nil, false, err
			}
			if inner == "..." {
				// All parameters are spliced in declaration order (comma-joined, forwarding the call form f(#insert(#ast(...))))
				for pi, pname := range paramOrder {
					if pi > 0 {
						out = append(out, Token{Kind: TComma, Text: ",", Line: t.Line, Col: t.Col})
					}
					out = append(out, subst[pname]...)
				}
			} else {
				captured, ok := subst[inner]
				if !ok {
					return nil, false, errors.New(msg("第 %d 行：#insert(#ast(%s))：%s 不是宏参数名", t.Line, inner, inner))
				}
				out = append(out, captured...)
			}
		case "execute":
			// #execute(name): splice in an identifier token (original semantics: a name built by concatenation)
			if len(args) < 1 || args[0].Kind != TIdent {
				return nil, false, errors.New(msg("第 %d 行：#execute 需要 (名字)", t.Line))
			}
			out = append(out, Token{Kind: TIdent, Text: args[0].Text, Line: t.Line, Col: t.Col})
		default:
			return nil, false, errors.New(msg("第 %d 行：未知预处理命令 #%s", t.Line, cmd))
		}
	}
	return out, false, nil
}

// parseAstArg parses an (#ast($name)) argument and returns the name; `$name` is the parameter sigil and a
// bare name is still accepted, and the name may be ... (all parameters).
func parseAstArg(args []Token) (string, error) {
	if len(args) < 4 || args[0].Kind != TSharp || args[1].Kind != TIdent || args[1].Text != "ast" ||
		args[2].Kind != TLParen || args[len(args)-1].Kind != TRParen {
		return "", errors.New(msg("#insert 需要 (#ast($名字)) 形式"))
	}
	if len(args) >= 5 && args[3].Kind == TDot && args[4].Kind == TDot && len(args) >= 6 && args[5].Kind == TDot {
		return "...", nil
	}
	if args[3].Kind == TDollar && len(args) >= 6 && args[4].Kind == TIdent && args[5].Kind == TRParen {
		return args[4].Text, nil
	}
	if args[3].Kind == TIdent {
		return args[3].Text, nil
	}
	return "", errors.New(msg("#insert 需要 (#ast($名字)) 形式"))
}

// Signature renders the macro's standard declaration form: macro name($a $b).
func (m *MacroDef) Signature() string {
	params := make([]string, len(m.Params))
	for i, p := range m.Params {
		params[i] = "$" + p
	}
	return fmt.Sprintf("macro %s(%s)", m.Name, strings.Join(params, " "))
}

// String renders the definition in the standard spelling for error messages.
func (m *MacroDef) String() string {
	return m.Signature()
}
