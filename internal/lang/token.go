// Package lang implements the QuarkLang v0.1 interpreter:
// lexer, parser, evaluator, and built-in runtime (spec: docs/spec.md).
package lang

import (
	"fmt"
	"strconv"
)

// TokenKind enumerates all lexer token kinds.
type TokenKind int

const (
	TEOF TokenKind = iota
	TIdent
	TInt
	TFloat
	TStr
	TFunc
	TStruct
	TImpl
	TInterface
	TReturn
	TIf
	TElse
	TWhile
	TFor
	TTrue
	TFalse
	TLParen
	TRParen
	TLBrace
	TRBrace
	TLBracket
	TRBracket
	TSemi
	TComma
	TDot
	TColon
	TScope
	TAt
	TAssign
	TPlus
	TMinus
	TStar
	TSlash
	TPercent
	TBang
	TEq
	TNe
	TLt
	TLe
	TGt
	TGe
	TAnd
	TOr
	TShl
	TShr
	TLog
	TAmper
	TNull
	TTry
	TCatch
	TSharp
	TMacro
)

var tokenNames = [...]string{
	"end of file", "identifier", "int literal", "float literal", "string literal",
	"'fn'", "'struct'", "'impl'", "'interface'",
	"'return'", "'if'", "'else'",
	"'while'", "'for'", "'true'", "'false'",
	"'('", "')'", "'{'", "'}'", "'['", "']'", "';'", "','", "'.'", "':'", "'::'", "'@'",
	"'='", "'+'", "'-'", "'*'", "'/'", "'%'", "'!'", "'=='", "'!='", "'<'", "'<='", "'>'", "'>='", "'&&'", "'||'",
	"'<<'", "'>>'", "'log'", "'&'", "'null'", "'try'", "'catch'", "'#'", "'macro'",
}

func (k TokenKind) String() string {
	if int(k) >= 0 && int(k) < len(tokenNames) {
		return tokenNames[k]
	}
	return "unknown token"
}

// Token is a single lexical token.
type Token struct {
	Kind TokenKind
	Text string
	Int  int64
	Flt  float64
	Line int
	Col  int
}

// LexError reports a lexical error with source position.
type LexError struct {
	Msg  string
	Line int
	Col  int
}

func (e *LexError) Error() string {
	return fmt.Sprintf("LexError: %s at line %d, col %d", e.Msg, e.Line, e.Col)
}

var keywords = map[string]TokenKind{
	"fn":        TFunc,
	"struct":    TStruct,
	"impl":      TImpl,
	"interface": TInterface,
	"return":    TReturn,
	"if":        TIf,
	"else":      TElse,
	"while":     TWhile,
	"for":       TFor,
	"true":      TTrue,
	"false":     TFalse,
	"log":       TLog,
	"null":      TNull,
	"try":       TTry,
	"catch":     TCatch,
	"macro":     TMacro,
}

type lexer struct {
	src  string
	pos  int
	line int
	col  int

	collectComments bool
	comments        []Comment
}

// singleCharText 返回单字符 token 的文本：预建表，避免每个标点都分配一个 1 字节字符串
// （实测每文件数千个标点 token，是词法层最大的分配来源之一）。
var singleCharTable = func() [256]string {
	var t [256]string
	for i := 0; i < 256; i++ {
		t[i] = string(rune(i))
	}
	return t
}()

func singleCharText(c byte) string { return singleCharTable[c] }

// Comment 是源码注释（qkdoc / qklsp 等工具用；常规编译流程丢弃注释）。
type Comment struct {
	Text    string // 原文（含 `//` 或 `/* */` 定界符）
	Pos     Pos    // 起始位置
	EndLine int    // 结束行（行注释 = 起始行；块注释 = `*/` 所在行）
	Block   bool   // true = /* */，false = //
}

func (lx *lexer) addComment(c Comment) {
	if lx.collectComments {
		lx.comments = append(lx.comments, c)
	}
}

// Lex tokenizes QuarkLang source code.
func Lex(src string) ([]Token, error) {
	toks, _, err := lex(src, false)
	return toks, err
}

// LexWithComments 与 Lex 相同，但额外按出现顺序收集注释。
func LexWithComments(src string) ([]Token, []Comment, error) {
	return lex(src, true)
}

func lex(src string, collectComments bool) ([]Token, []Comment, error) {
	lx := &lexer{src: src, line: 1, col: 1, collectComments: collectComments}
	// 预分配 token 容量：实测每 token 约 3–4 字节源码，避免 append 反复扩容复制
	toks := make([]Token, 0, len(src)/3+8)
	for {
		tok, err := lx.next()
		if err != nil {
			return nil, nil, err
		}
		toks = append(toks, tok)
		if tok.Kind == TEOF {
			return toks, lx.comments, nil
		}
	}
}

func (lx *lexer) errf(line, col int, format string, args ...interface{}) error {
	return &LexError{Msg: fmt.Sprintf(format, args...), Line: line, Col: col}
}

func (lx *lexer) peekByte() byte {
	if lx.pos >= len(lx.src) {
		return 0
	}
	return lx.src[lx.pos]
}

func (lx *lexer) peekByteAt(off int) byte {
	if lx.pos+off >= len(lx.src) {
		return 0
	}
	return lx.src[lx.pos+off]
}

func (lx *lexer) advance() {
	if lx.pos < len(lx.src) {
		if lx.src[lx.pos] == '\n' {
			lx.line++
			lx.col = 1
		} else {
			lx.col++
		}
		lx.pos++
	}
}

// skipSpace consumes whitespace and comments.
func (lx *lexer) skipSpace() error {
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			lx.advance()
		case c == '/' && lx.peekByteAt(1) == '/':
			startLine, startCol, start := lx.line, lx.col, lx.pos
			for lx.pos < len(lx.src) && lx.src[lx.pos] != '\n' {
				lx.advance()
			}
			lx.addComment(Comment{
				Text: lx.src[start:lx.pos], Pos: Pos{Line: startLine, Col: startCol},
				EndLine: startLine, Block: false,
			})
		case c == '/' && lx.peekByteAt(1) == '*':
			line, col := lx.line, lx.col
			start := lx.pos
			lx.advance()
			lx.advance()
			closed := false
			for lx.pos < len(lx.src) {
				if lx.src[lx.pos] == '*' && lx.peekByteAt(1) == '/' {
					lx.advance()
					lx.advance()
					closed = true
					break
				}
				lx.advance()
			}
			if !closed {
				return lx.errf(line, col, "unterminated block comment")
			}
			lx.addComment(Comment{
				Text: lx.src[start:lx.pos], Pos: Pos{Line: line, Col: col},
				EndLine: lx.line, Block: true,
			})
		default:
			return nil
		}
	}
	return nil
}

func (lx *lexer) next() (Token, error) {
	if err := lx.skipSpace(); err != nil {
		return Token{}, err
	}
	if lx.pos >= len(lx.src) {
		return Token{Kind: TEOF, Line: lx.line, Col: lx.col}, nil
	}
	line, col := lx.line, lx.col
	c := lx.src[lx.pos]
	switch {
	case c >= '0' && c <= '9':
		return lx.lexNumber(line, col)
	case c == '"':
		return lx.lexString(line, col)
	case c == '`':
		return lx.lexRawString(line, col)
	case isIdentStart(c):
		return lx.lexIdent(line, col)
	}

	one := func(k TokenKind) (Token, error) {
		lx.advance()
		return Token{Kind: k, Text: singleCharText(c), Line: line, Col: col}, nil
	}
	switch c {
	case '#':
		return one(TSharp)
	case '(':
		return one(TLParen)
	case ')':
		return one(TRParen)
	case '{':
		return one(TLBrace)
	case '}':
		return one(TRBrace)
	case '[':
		return one(TLBracket)
	case ']':
		return one(TRBracket)
	case ';':
		return one(TSemi)
	case ',':
		return one(TComma)
	case '.':
		return one(TDot)
	case '@':
		return one(TAt)
	case '+':
		return one(TPlus)
	case '-':
		return one(TMinus)
	case '*':
		return one(TStar)
	case '/':
		return one(TSlash)
	case '%':
		return one(TPercent)
	case '=':
		if lx.peekByteAt(1) == '=' {
			lx.advance()
			lx.advance()
			return Token{Kind: TEq, Text: "==", Line: line, Col: col}, nil
		}
		return one(TAssign)
	case '!':
		if lx.peekByteAt(1) == '=' {
			lx.advance()
			lx.advance()
			return Token{Kind: TNe, Text: "!=", Line: line, Col: col}, nil
		}
		return one(TBang)
	case '<':
		if lx.peekByteAt(1) == '=' {
			lx.advance()
			lx.advance()
			return Token{Kind: TLe, Text: "<=", Line: line, Col: col}, nil
		}
		if lx.peekByteAt(1) == '<' {
			lx.advance()
			lx.advance()
			return Token{Kind: TShl, Text: "<<", Line: line, Col: col}, nil
		}
		return one(TLt)
	case '>':
		if lx.peekByteAt(1) == '=' {
			lx.advance()
			lx.advance()
			return Token{Kind: TGe, Text: ">=", Line: line, Col: col}, nil
		}
		if lx.peekByteAt(1) == '>' {
			lx.advance()
			lx.advance()
			return Token{Kind: TShr, Text: ">>", Line: line, Col: col}, nil
		}
		return one(TGt)
	case ':':
		if lx.peekByteAt(1) == ':' {
			lx.advance()
			lx.advance()
			return Token{Kind: TScope, Text: "::", Line: line, Col: col}, nil
		}
		return one(TColon)
	case '&':
		if lx.peekByteAt(1) == '&' {
			lx.advance()
			lx.advance()
			return Token{Kind: TAnd, Text: "&&", Line: line, Col: col}, nil
		}
		return one(TAmper)
	case '|':
		if lx.peekByteAt(1) == '|' {
			lx.advance()
			lx.advance()
			return Token{Kind: TOr, Text: "||", Line: line, Col: col}, nil
		}
		return Token{}, lx.errf(line, col, "unexpected character '|' (did you mean '||'?)")
	}
	return Token{}, lx.errf(line, col, "unexpected character %q", string(rune(c)))
}

func (lx *lexer) lexNumber(line, col int) (Token, error) {
	start := lx.pos
	for isDigit(lx.peekByte()) {
		lx.advance()
	}
	isFloat := false
	if lx.peekByte() == '.' && isDigit(lx.peekByteAt(1)) {
		isFloat = true
		lx.advance()
		for isDigit(lx.peekByte()) {
			lx.advance()
		}
	}
	text := lx.src[start:lx.pos]
	if isFloat {
		f, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return Token{}, lx.errf(line, col, "invalid float literal %q", text)
		}
		return Token{Kind: TFloat, Text: text, Flt: f, Line: line, Col: col}, nil
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return Token{}, lx.errf(line, col, "invalid int literal %q", text)
	}
	return Token{Kind: TInt, Text: text, Int: n, Line: line, Col: col}, nil
}

// lexRawString 反引号原始字符串（同 Go：无转义、可多行、换行计入行号）。
// 注意：与 Go 的差异是 \r 原样保留在字符串值中（本实现不做丢弃）。
func (lx *lexer) lexRawString(line, col int) (Token, error) {
	lx.advance() // `
	start := lx.pos
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		if c == '\x60' {
			text := lx.src[start:lx.pos]
			lx.advance()
			return Token{Kind: TStr, Text: text, Line: line, Col: col}, nil
		}
		// 行/列推进统一交给 advance()：此前这里额外自增 line 会造成
		// 多行原始字符串之后所有 token 的行号偏移（每个换行多算 1 行）。
		lx.advance()
	}
	return Token{}, lx.errf(line, col, "unterminated raw string")
}

func (lx *lexer) lexString(line, col int) (Token, error) {
	lx.advance() // opening quote
	var sb []byte
	for {
		if lx.pos >= len(lx.src) {
			return Token{}, lx.errf(line, col, "unterminated string literal")
		}
		c := lx.src[lx.pos]
		if c == '"' {
			lx.advance()
			return Token{Kind: TStr, Text: string(sb), Line: line, Col: col}, nil
		}
		if c == '\\' {
			lx.advance()
			if lx.pos >= len(lx.src) {
				return Token{}, lx.errf(line, col, "unterminated escape sequence")
			}
			e := lx.src[lx.pos]
			switch e {
			case 'n':
				sb = append(sb, '\n')
			case 't':
				sb = append(sb, '\t')
			case 'r':
				sb = append(sb, '\r')
			case '"':
				sb = append(sb, '"')
			case '\\':
				sb = append(sb, '\\')
			default:
				// 未知转义保留字面（纯文本语义：仅标准转义生效）
				sb = append(sb, '\\', e)
			}
			lx.advance()
			continue
		}
		sb = append(sb, c)
		lx.advance()
	}
}

func (lx *lexer) lexIdent(line, col int) (Token, error) {
	start := lx.pos
	for isIdentPart(lx.peekByte()) {
		lx.advance()
	}
	text := lx.src[start:lx.pos]
	if kw, ok := keywords[text]; ok {
		return Token{Kind: kw, Text: text, Line: line, Col: col}, nil
	}
	return Token{Kind: TIdent, Text: text, Line: line, Col: col}, nil
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || isDigit(c)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
