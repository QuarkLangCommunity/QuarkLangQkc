package lang

import (
	"fmt"
	"strings"
)

// SplitMacroDefs 从 token 流中切分出所有 #macro name (参数...) { 主体 } 定义，
// 返回宏列表与剩余 token。
//
// 语法（用户定案）：
//
//	#macro name (p1, p2, ...) { 主体 }
//	#macro name [p1, p2] { 主体 }   // 参数列表分隔符 () [] {} 任意
//	#macro name {p1, p2} { 主体 }
//
// 参数均为名字、逗号分隔、个数不限；主体内按名替换为调用实参 token。
// 调用形式（与定义同形，分隔符同样任意）：
//
//	name(args) / name[args] / name{args}
func SplitMacroDefs(toks []Token) ([]*MacroDef, []Token, error) {
	// 快路径：全文没有 `#` 开头的 token（宏/预处理指令），即无宏可切 →
	// 直接返回原切片（零拷贝、零分配）。绝大多数源文件走这条路。
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
		i += 2 // 吃掉 # macro
		if i >= len(toks) || toks[i].Kind != TIdent {
			return nil, nil, fmt.Errorf("ParseError: #macro 需要名字，第 %d 行", t.Line)
		}
		name := toks[i].Text
		i++
		if i >= len(toks) {
			return nil, nil, fmt.Errorf("ParseError: #macro %s 缺少参数列表，第 %d 行", name, t.Line)
		}
		closeK, ok := closeOf(toks[i].Kind)
		if !ok {
			return nil, nil, fmt.Errorf("ParseError: #macro %s 参数列表需要 () [] {} 之一，第 %d 行", name, toks[i].Line)
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
				return nil, nil, fmt.Errorf("ParseError: #macro %s 参数必须是名字（逗号分隔，个数不限），第 %d 行", name, t.Line)
			}
			pnames = append(pnames, pt[0].Text)
		}
		if i >= len(toks) {
			return nil, nil, fmt.Errorf("ParseError: #macro %s 缺少主体，第 %d 行", name, t.Line)
		}
		closeB, ok := closeOf(toks[i].Kind)
		if !ok {
			return nil, nil, fmt.Errorf("ParseError: #macro %s 主体需要 () [] {} 之一，第 %d 行", name, toks[i].Line)
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

// closeOf 返回左分隔符对应的右分隔符。
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

// takeBalancedPair 从 toks[i]（左分隔符）开始取平衡块（()[]{} 混合配对计数），
// 返回块内 token 与结束位置；块闭合符必须与 openK 对应。
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
					return nil, 0, fmt.Errorf("ParseError: 分隔符不配对，第 %d 行", toks[i].Line)
				}
				return toks[i+1 : j], j + 1, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("ParseError: 分隔符不配对，第 %d 行", toks[i].Line)
}

// splitTop 把 token 序列按顶层逗号切分（忽略 ()[]{} 内部的逗号）。
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

// ExpandMacros 在 token 流上展开命名宏调用：name(args)/name[args]/name{args}。
// mode 为动态预处理态（"run"/"compile"/"explain"）；主体不递归再展开。
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
						return nil, fmt.Errorf("ParseError: 宏 %s 需要 %d 个参数，调用给了 %d 个（第 %d 行）", m.Name, len(m.Params), len(callArgs), t.Line)
					}
					subst := make(map[string][]Token, len(callArgs))
					for p, pname := range m.Params {
						subst[pname] = callArgs[p]
					}
					body, _, err := expandBody(m.Body, subst, mode, m.Params)
					if err != nil {
						return nil, fmt.Errorf("宏 %s 展开失败：%v", m.Name, err)
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

// macroCallExcluded：声明位置/成员访问/作用域调用/预处理命令处不展开宏。
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

// expandBody 展开宏主体：参数按名替换；#when (compile|run) { ... } 按态选块；
// #return <token...> 是宏的返回值（展开结果 = 返回指令后的 token，参数替换后），立即终止整个展开；
// #error("msg") 直接报错；#insert/#execute/#ast 已随 pattern 语法移除。
// 返回 (输出, 是否被 #return 终止)。
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
			return nil, false, fmt.Errorf("第 %d 行：# 后必须是预处理命令（when/return/error）", t.Line)
		}
		cmd := body[i+1].Text
		i += 2
		if cmd == "return" {
			// #return <expr>：显示结果 = 返回指令后的全部内容（参数替换 + 后续指令继续处理），
			// 例如 #return #insert(#ast(a))；并终止整个宏展开。
			sub, _, err := expandBody(body[i:], subst, mode, paramOrder)
			if err != nil {
				return nil, false, err
			}
			out = append(out, sub...)
			return out, true, nil
		}
		if i >= len(body) {
			return nil, false, fmt.Errorf("第 %d 行：#%s 需要 ( ... )", t.Line, cmd)
		}
		closeK, ok := closeOf(body[i].Kind)
		if !ok {
			return nil, false, fmt.Errorf("第 %d 行：#%s 需要 ( ... )", t.Line, cmd)
		}
		args, ni, err := takeBalancedPair(body, i, closeK)
		if err != nil {
			return nil, false, err
		}
		i = ni
		switch cmd {
		case "when":
			if len(args) < 1 || args[0].Kind != TIdent {
				return nil, false, fmt.Errorf("第 %d 行：#when 需要 (compile|run)", t.Line)
			}
			if i >= len(body) || body[i].Kind != TLBrace {
				return nil, false, fmt.Errorf("第 %d 行：#when 需要 { ... } 块", t.Line)
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
				return nil, false, fmt.Errorf("第 %d 行：#error 需要 (\"消息\")", t.Line)
			}
			return nil, false, fmt.Errorf("第 %d 行：预处理错误 #error(%s)", t.Line, args[0].Text)
		case "insert":
			// #insert(#ast(name))：直插参数 name 的 token；#insert(#ast(...))：按序直插全部参数
			inner, err := parseAstArg(args)
			if err != nil {
				return nil, false, err
			}
			if inner == "..." {
				// 全部参数按声明顺序直插（逗号连接，转发调用形式 f(#insert(#ast(...)))）
				for pi, pname := range paramOrder {
					if pi > 0 {
						out = append(out, Token{Kind: TComma, Text: ",", Line: t.Line, Col: t.Col})
					}
					out = append(out, subst[pname]...)
				}
			} else {
				captured, ok := subst[inner]
				if !ok {
					return nil, false, fmt.Errorf("第 %d 行：#insert(#ast(%s))：%s 不是宏参数名", t.Line, inner, inner)
				}
				out = append(out, captured...)
			}
		case "execute":
			// #execute(name)：直插一个标识符 token（原语义：拼接生成的名字）
			if len(args) < 1 || args[0].Kind != TIdent {
				return nil, false, fmt.Errorf("第 %d 行：#execute 需要 (名字)", t.Line)
			}
			out = append(out, Token{Kind: TIdent, Text: args[0].Text, Line: t.Line, Col: t.Col})
		default:
			return nil, false, fmt.Errorf("第 %d 行：未知预处理命令 #%s", t.Line, cmd)
		}
	}
	return out, false, nil
}

// parseAstArg 解析 (#ast(名字)) 参数，返回名字；名字可为参数名或 ...（全部参数）。
func parseAstArg(args []Token) (string, error) {
	if len(args) < 4 || args[0].Kind != TSharp || args[1].Kind != TIdent || args[1].Text != "ast" ||
		args[2].Kind != TLParen || args[len(args)-1].Kind != TRParen {
		return "", fmt.Errorf("#insert 需要 (#ast(名字)) 形式")
	}
	if len(args) >= 5 && args[3].Kind == TDot && args[4].Kind == TDot && len(args) >= 6 && args[5].Kind == TDot {
		return "...", nil
	}
	if args[3].Kind == TIdent {
		return args[3].Text, nil
	}
	return "", fmt.Errorf("#insert 需要 (#ast(名字)) 形式")
}

// String 便于报错展示。
func (m *MacroDef) String() string {
	return fmt.Sprintf("#macro %s (%s)", m.Name, strings.Join(m.Params, ", "))
}
