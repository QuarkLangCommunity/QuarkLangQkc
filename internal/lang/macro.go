package lang

import (
	"errors"
	"fmt"
	"quarklang/internal/i18n"
	"strings"
)

// SplitMacroDefs splits every #macro name (params...) { body } definition out of a token stream,
// returning the macro list and the remaining tokens.
//
// Syntax (settled by the user):
//
//	#macro name (p1, p2, ...) { body }
//	#macro name [p1, p2] { body }   // the parameter-list delimiter may be () [] or {}
//	#macro name {p1, p2} { body }
//
// Parameters are names, comma-separated, with no limit on the count; inside the body they are replaced by the call's argument tokens.
// Invocation (mirrors the definition; the delimiter is likewise free):
//
//	name(args) / name[args] / name{args}
func SplitMacroDefs(toks []Token) ([]*MacroDef, []Token, error) {
	// Fast path: when the whole file has no token starting with `#` (macro/preprocessor directives) there is nothing to split →
	// return the original slice (zero copy, zero allocation). The vast majority of source files take this path.
	hasSharp := false
	for i := range toks {
		if toks[i].Kind == TSharp {
			hasSharp = true
			break
		}
	}
	if !hasSharp {
		return nil, toks, nil
	}
	var macros []*MacroDef
	rest := make([]Token, 0, len(toks))
	i := 0
	for i < len(toks) {
		t := toks[i]
		if t.Kind != TSharp || i+1 >= len(toks) || toks[i+1].Kind != TMacro {
			rest = append(rest, t)
			i++
			continue
		}
		pos := Pos{Line: t.Line, Col: t.Col}
		i += 2 // consume '# macro'
		if i >= len(toks) || toks[i].Kind != TIdent {
			return nil, nil, errors.New(i18n.T("ParseError: #macro 需要名字，第 %d 行", t.Line))
		}
		name := toks[i].Text
		i++
		if i >= len(toks) {
			return nil, nil, errors.New(i18n.T("ParseError: #macro %s 缺少参数列表，第 %d 行", name, t.Line))
		}
		closeK, ok := closeOf(toks[i].Kind)
		if !ok {
			return nil, nil, errors.New(i18n.T("ParseError: #macro %s 参数列表需要 () [] {} 之一，第 %d 行", name, toks[i].Line))
		}
		params, ni, err := takeBalancedPair(toks, i, closeK)
		if err != nil {
			return nil, nil, err
		}
		i = ni
		var pnames []string
		for _, pt := range splitTop(params) {
			if len(pt) == 0 {
				continue
			}
			if len(pt) != 1 || pt[0].Kind != TIdent {
				return nil, nil, errors.New(i18n.T("ParseError: #macro %s 参数必须是名字（逗号分隔，个数不限），第 %d 行", name, t.Line))
			}
			pnames = append(pnames, pt[0].Text)
		}
		if i >= len(toks) {
			return nil, nil, errors.New(i18n.T("ParseError: #macro %s 缺少主体，第 %d 行", name, t.Line))
		}
		closeB, ok := closeOf(toks[i].Kind)
		if !ok {
			return nil, nil, errors.New(i18n.T("ParseError: #macro %s 主体需要 () [] {} 之一，第 %d 行", name, toks[i].Line))
		}
		body, ni, err := takeBalancedPair(toks, i, closeB)
		if err != nil {
			return nil, nil, err
		}
		i = ni
		macros = append(macros, &MacroDef{Name: name, Params: pnames, Body: body, Pos: pos})
	}
	return macros, rest, nil
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
					return nil, 0, errors.New(i18n.T("ParseError: 分隔符不配对，第 %d 行", toks[i].Line))
				}
				return toks[i+1 : j], j + 1, nil
			}
		}
	}
	return nil, 0, errors.New(i18n.T("ParseError: 分隔符不配对，第 %d 行", toks[i].Line))
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
						return nil, errors.New(i18n.T("ParseError: 宏 %s 需要 %d 个参数，调用给了 %d 个（第 %d 行）", m.Name, len(m.Params), len(callArgs), t.Line))
					}
					subst := make(map[string][]Token, len(callArgs))
					for p, pname := range m.Params {
						subst[pname] = callArgs[p]
					}
					body, _, err := expandBody(m.Body, subst, mode, m.Params)
					if err != nil {
						return nil, errors.New(i18n.T("宏 %s 展开失败：%v", m.Name, err))
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

// expandBody expands a macro body: parameters are replaced by name; #when (compile|run) { ... } selects a block by state;
// #return <token...> is the macro's return value (the expansion result is the tokens after the return directive, with parameters substituted) and terminates the whole expansion immediately;
// #error("msg") reports an error directly; #insert/#execute/#ast were removed with the pattern syntax.
// Returns (output, whether a #return terminated it).
func expandBody(body []Token, subst map[string][]Token, mode string, paramOrder []string) ([]Token, bool, error) {
	var out []Token
	i := 0
	for i < len(body) {
		t := body[i]
		if t.Kind != TSharp {
			if sub, ok := subst[t.Text]; ok && t.Kind == TIdent {
				out = append(out, sub...)
			} else {
				out = append(out, t)
			}
			i++
			continue
		}
		if i+1 >= len(body) || (body[i+1].Kind != TIdent && body[i+1].Kind != TReturn) {
			return nil, false, errors.New(i18n.T("第 %d 行：# 后必须是预处理命令（when/return/error）", t.Line))
		}
		cmd := body[i+1].Text
		i += 2
		if cmd == "return" {
			// #return <expr>: the shown result is everything after the return directive (parameters substituted + later directives still processed),
			// e.g. #return #insert(#ast(a)); and it terminates the whole macro expansion.
			sub, _, err := expandBody(body[i:], subst, mode, paramOrder)
			if err != nil {
				return nil, false, err
			}
			out = append(out, sub...)
			return out, true, nil
		}
		if i >= len(body) {
			return nil, false, errors.New(i18n.T("第 %d 行：#%s 需要 ( ... )", t.Line, cmd))
		}
		closeK, ok := closeOf(body[i].Kind)
		if !ok {
			return nil, false, errors.New(i18n.T("第 %d 行：#%s 需要 ( ... )", t.Line, cmd))
		}
		args, ni, err := takeBalancedPair(body, i, closeK)
		if err != nil {
			return nil, false, err
		}
		i = ni
		switch cmd {
		case "when":
			if len(args) < 1 || args[0].Kind != TIdent {
				return nil, false, errors.New(i18n.T("第 %d 行：#when 需要 (compile|run)", t.Line))
			}
			if i >= len(body) || body[i].Kind != TLBrace {
				return nil, false, errors.New(i18n.T("第 %d 行：#when 需要 { ... } 块", t.Line))
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
				return nil, false, errors.New(i18n.T("第 %d 行：#error 需要 (\"消息\")", t.Line))
			}
			return nil, false, errors.New(i18n.T("第 %d 行：预处理错误 #error(%s)", t.Line, args[0].Text))
		case "insert":
			// #insert(#ast(name)): splice in the tokens of parameter name; #insert(#ast(...)): splice in all parameters in order
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
					return nil, false, errors.New(i18n.T("第 %d 行：#insert(#ast(%s))：%s 不是宏参数名", t.Line, inner, inner))
				}
				out = append(out, captured...)
			}
		case "execute":
			// #execute(name): splice in an identifier token (original semantics: a name built by concatenation)
			if len(args) < 1 || args[0].Kind != TIdent {
				return nil, false, errors.New(i18n.T("第 %d 行：#execute 需要 (名字)", t.Line))
			}
			out = append(out, Token{Kind: TIdent, Text: args[0].Text, Line: t.Line, Col: t.Col})
		default:
			return nil, false, errors.New(i18n.T("第 %d 行：未知预处理命令 #%s", t.Line, cmd))
		}
	}
	return out, false, nil
}

// parseAstArg parses an (#ast(name)) argument and returns the name; the name may be a parameter name or ... (all parameters).
func parseAstArg(args []Token) (string, error) {
	if len(args) < 4 || args[0].Kind != TSharp || args[1].Kind != TIdent || args[1].Text != "ast" ||
		args[2].Kind != TLParen || args[len(args)-1].Kind != TRParen {
		return "", errors.New(i18n.T("#insert 需要 (#ast(名字)) 形式"))
	}
	if len(args) >= 5 && args[3].Kind == TDot && args[4].Kind == TDot && len(args) >= 6 && args[5].Kind == TDot {
		return "...", nil
	}
	if args[3].Kind == TIdent {
		return args[3].Text, nil
	}
	return "", errors.New(i18n.T("#insert 需要 (#ast(名字)) 形式"))
}

// String renders it for error messages.
func (m *MacroDef) String() string {
	return fmt.Sprintf("#macro %s (%s)", m.Name, strings.Join(m.Params, ", "))
}
