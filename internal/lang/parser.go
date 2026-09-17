package lang

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// ParseError reports a syntax error with source position.
type ParseError struct {
	Msg  string
	Line int
	Col  int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("ParseError: %s at line %d, col %d", e.Msg, e.Line, e.Col)
}

type parser struct {
	toks []Token
	i    int

	prog      *Program          // 当前程序（匿名 struct/interface 作类型标注时合成实名类型登记）
	anonByKey map[string]string // 匿名类型结构键 → 合成实名（同一结构复用同一个名字）
	anonSeq   int
}

// Parse builds the AST from tokens (spec §2).
func Parse(toks []Token) (*Program, error) {
	p := &parser{toks: toks}
	return p.parseProgram()
}

// Compile lexes, parses, and statically type-checks source code
// (spec §11.1: strict checks run at compile time).
// compileCache：进程内增量编译缓存（源码 sha256 → 已编译 Program）。
var compileCache sync.Map

// walkCallRefs 遍历语句/表达式，解析 CallExpr.FnIdx（Ident 且命中函数表）。
func walkCallRefs(s interface{}, idx map[string]int) {
	walkNode(s, func(e Expr) {
		if c, ok := e.(*CallExpr); ok {
			if id, ok := c.Fn.(*Ident); ok {
				c.FnIdx = -1
				if i, ok := idx[id.Name]; ok {
					c.FnIdx = i
				}
			}
		}
	})
}

func walkNode(n interface{}, visit func(Expr)) {
	// 判 nil（含具体类型 nil 指针，如 (*Block)(nil)）
	if n == nil {
		return
	}
	if rv := reflect.ValueOf(n); rv.Kind() == reflect.Ptr && rv.IsNil() {
		return
	}
	if rv := reflect.ValueOf(n); rv.Kind() == reflect.Ptr && rv.IsNil() {
		return
	}
	switch t := n.(type) {
	case Expr:
		visit(t)
		if c, ok := t.(*CallExpr); ok {
			for _, a := range c.Args {
				walkNode(a, visit)
			}
			walkNode(c.Fn, visit)
		}
		if b, ok := t.(*BinOp); ok {
			walkNode(b.L, visit)
			walkNode(b.R, visit)
		}
	case *Block:
		for _, s := range t.Stmts {
			walkNode(s, visit)
		}
	case *DeclStmt:
		walkNode(t.Init, visit)
	case *AssignStmt:
		walkNode(t.Target, visit)
		walkNode(t.X, visit)
	case *ReturnStmt:
		walkNode(t.X, visit)
	case *ExprStmt:
		walkNode(t.X, visit)
	case *IfStmt:
		walkNode(t.Cond, visit)
		walkNode(t.Then, visit)
		walkNode(t.Else, visit)
	case *WhileStmt:
		walkNode(t.Cond, visit)
		walkNode(t.Body, visit)
	case *ForStmt:
		walkNode(t.Iter, visit)
		walkNode(t.Body, visit)
	case *TryStmt:
		for _, s := range t.Try.Stmts {
			walkNode(s, visit)
		}
		for _, s := range t.Catch.Stmts {
			walkNode(s, visit)
		}
	}
}

// mainBody 返回主函数体（无 main 则空）。
func mainBody(prog *Program) *Block {
	for _, f := range prog.Funcs {
		if f.Name == "main" {
			return f.Body
		}
	}
	return &Block{}
}

func Compile(src string) (*Program, error) {
	h := sha256.Sum256([]byte(src))
	key := string(h[:])
	if p, ok := compileCache.Load(key); ok {
		return p.(*Program), nil
	}
	prog, err := compileSlow(src)
	if err != nil {
		return nil, err
	}
	compileCache.Store(key, prog)
	return prog, nil
}

func compileSlow(src string) (*Program, error) {
	toks, err := Lex(src)
	if err != nil {
		return nil, err
	}
	// 宏系统：切出 macro 定义，token 级展开（解释器为 run 态），再解析
	macros, rest, err := SplitMacroDefs(toks)
	if err != nil {
		return nil, err
	}
	if len(macros) > 0 {
		rest, err = ExpandMacros(rest, macros, "explain") // 解释器 = explain 操作时（xmind §操作时）
		if err != nil {
			return nil, err
		}
	}
	prog, err := Parse(rest)
	if err != nil {
		return nil, err
	}
	prog.Src = src
	if err := Typecheck(prog); err != nil {
		return nil, err
	}
	// 编译期解析变量槽位（解释器免线性扫名；编译器路径忽略该字段）
	resolveSlots(prog)
	// 编译期解析函数调用索引（eval 免 map 查找）
	prog.FnList = make([]*FuncDecl, 0, len(prog.Funcs))
	prog.FnIndex = map[string]int{}
	for i, f := range prog.Funcs {
		prog.FnList = append(prog.FnList, f)
		prog.FnIndex[f.Name] = i
	}
	for _, f := range prog.Funcs {
		walkCallRefs(f.Body, prog.FnIndex)
	}
	walkCallRefs(mainBody(prog), prog.FnIndex)
	return prog, nil
}

func (p *parser) cur() Token { return p.toks[p.i] }
func (p *parser) peekTextIs(t string) bool {
	return p.i+1 < len(p.toks) && p.toks[p.i+1].Kind != TEOF && p.toks[p.i+1].Text == t
}

func (p *parser) peekIs(k TokenKind) bool { return p.i+1 < len(p.toks) && p.toks[p.i+1].Kind == k }

func (p *parser) peek() Token {
	if p.i+1 < len(p.toks) {
		return p.toks[p.i+1]
	}
	return p.toks[len(p.toks)-1]
}

func (p *parser) advance() Token {
	t := p.toks[p.i]
	if p.i < len(p.toks)-1 {
		p.i++
	}
	return t
}

func (p *parser) curIs(k TokenKind) bool { return p.cur().Kind == k }

func (p *parser) errf(tok Token, format string, args ...interface{}) error {
	return &ParseError{Msg: fmt.Sprintf(format, args...), Line: tok.Line, Col: tok.Col}
}

func (p *parser) expect(k TokenKind, what string) (Token, error) {
	if !p.curIs(k) {
		return Token{}, p.errf(p.cur(), "expected %s, got %s", what, p.cur().Kind)
	}
	return p.advance(), nil
}

func (p *parser) expectIdent(what string) (Token, error) {
	if !p.curIs(TIdent) {
		return Token{}, p.errf(p.cur(), "expected %s, got %s", what, p.cur().Kind)
	}
	return p.advance(), nil
}

// expectMemberName accepts a member/field name; keywords (like in/out/log)
// are valid names in member position.
func (p *parser) expectMemberName(what string) (Token, error) {
	if isWordToken(p.cur().Kind) {
		return p.advance(), nil
	}
	return Token{}, p.errf(p.cur(), "expected %s, got %s", what, p.cur().Kind)
}

// isBuiltinTypeName 判断是否为内建类型名（type-first 声明用）。
func isBuiltinTypeName(s string) bool {
	switch s {
	case "int", "float", "bool", "String", "void", "List", "HashTable", "Channel", "thread", "Task", "memorize", "memory", "IOStream", "Copyd",
		"istream", "ostream", "ifstream", "ofstream", "iofstream", "InputStream", "OutputStream":
		return true
	}
	return false
}

// isWordToken 判断是否为标识符或关键字（非标点/字面量）。
func isWordToken(k TokenKind) bool {
	return k == TIdent || (k > TStr && k < TLParen)
}

func (p *parser) parseProgram() (*Program, error) {
	prog := &Program{}
	p.prog = prog
	if p.anonByKey == nil {
		p.anonByKey = map[string]string{}
	}
	for !p.curIs(TEOF) {
		switch p.cur().Kind {
		case TFunc:
			fn, err := p.parseFunc()
			if err != nil {
				return nil, err
			}
			prog.Funcs = append(prog.Funcs, fn)
		case TStruct:
			sd, err := p.parseStruct()
			if err != nil {
				return nil, err
			}
			if sd.Name != "" {
				return nil, p.errf(p.cur(), "实名结构体必须写 type：type struct { ... } %s;", sd.Name)
			}
			prog.Structs = append(prog.Structs, sd)
		case TInterface:
			id, err := p.parseInterface()
			if err != nil {
				return nil, err
			}
			if id.Name != "" {
				return nil, p.errf(p.cur(), "实名接口必须写 type：type interface { ... } %s;", id.Name)
			}
			prog.Interfaces = append(prog.Interfaces, id)
		case TImpl:
			im, err := p.parseImpl()
			if err != nil {
				return nil, err
			}
			prog.Impls = append(prog.Impls, im)
		default:
			if p.cur().Kind == TIdent {
				switch p.cur().Text {
				case "type":
					// xmind：type interface<T>{...}(Name) / type struct<T>{...}(Name)
					p.advance()
					switch p.cur().Kind {
					case TInterface:
						id, err := p.parseInterface()
						if err != nil {
							return nil, err
						}
						prog.Interfaces = append(prog.Interfaces, id)
					case TStruct:
						sd, err := p.parseStruct()
						if err != nil {
							return nil, err
						}
						prog.Structs = append(prog.Structs, sd)
					default:
						// 通用类型别名：type <类型表达式> 名字;
						typ, err := p.parseType()
						if err != nil {
							return nil, err
						}
						n, err := p.expectIdent("type alias name")
						if err != nil {
							return nil, err
						}
						if _, err := p.expect(TSemi, "';'"); err != nil {
							return nil, err
						}
						prog.TypeAliases = append(prog.TypeAliases, &TypeAlias{Name: n.Text, Type: typ, Pos: Pos{Line: n.Line, Col: n.Col}})
					}
					continue
				case "space":
					// space { ... } name;  自我实现空间（xmind 写作 space {...} (name);，(name) 表示名字槽位）
					p.advance()
					if _, err := p.expect(TLBrace, "'{'"); err != nil {
						return nil, err
					}
					im := &ImplDecl{Pos: Pos{Line: p.cur().Line, Col: p.cur().Col}}
					for !p.curIs(TRBrace) {
						if p.curIs(TEOF) {
							return nil, p.errf(p.cur(), "unterminated space body")
						}
						m, err := p.parseFunc()
						if err != nil {
							return nil, err
						}
						im.Methods = append(im.Methods, m)
					}
					p.advance() // '}'
					name, err := p.expectIdent("space name")
					if err != nil {
						return nil, err
					}
					if _, err := p.expect(TSemi, "';'"); err != nil {
						return nil, err
					}
					im.Type = name.Text
					prog.Impls = append(prog.Impls, im)
					continue
				case "library":
					// library 系统库绑定：library <Ident 或 "字符串"> { fn 签名; ... }; / library X;
					libPos := Pos{Line: p.cur().Line, Col: p.cur().Col}
					p.advance()
					var libName string
					switch p.cur().Kind {
					case TStr:
						libName = p.cur().Text
						p.advance()
					default:
						n, err := p.expectIdent("library name")
						if err != nil {
							return nil, err
						}
						libName = n.Text
					}
					ld := &LibraryDecl{Name: libName, Lib: libName, Pos: libPos}
					if p.curIs(TLBrace) {
						p.advance()
						for !p.curIs(TRBrace) {
							if p.curIs(TEOF) {
								return nil, p.errf(p.cur(), "unterminated library body")
							}
							if !p.curIs(TFunc) {
								return nil, p.errf(p.cur(), "library 体内只能有 fn 符号签名")
							}
							kw := p.advance()
							name, err := p.expectIdent("function symbol name")
							if err != nil {
								return nil, err
							}
							if _, err := p.expect(TLParen, "'('"); err != nil {
								return nil, err
							}
							params, err := p.parseParamList()
							if err != nil {
								return nil, err
							}
							ret := "void"
							if p.curIs(TIdent) || p.curIs(TInterface) || p.curIs(TStruct) {
								ret, err = p.parseType()
								if err != nil {
									return nil, err
								}
							}
							if _, err := p.expect(TSemi, "';'"); err != nil {
								return nil, err
							}
							ld.Methods = append(ld.Methods, &Func{Name: name.Text, Params: params, Ret: ret, Pos: Pos{Line: kw.Line, Col: kw.Col}})
						}
						p.advance() // '}'
						if p.curIs(TSemi) {
							p.advance() // ';' 可选
						}
					} else {
						if _, err := p.expect(TSemi, "';'"); err != nil {
							return nil, err
						}
					}
					prog.Libraries = append(prog.Libraries, ld)
					continue
				case "program":
					// 预制宏：program main; / program library;
					p.advance()
					kind, err := p.expectIdent("program kind (main|library)")
					if err != nil {
						return nil, err
					}
					if kind.Text != "main" && kind.Text != "library" && kind.Text != "lib" {
						return nil, p.errf(kind, "program 声明只支持 main / library（lib）")
					}
					if kind.Text == "lib" {
						kind.Text = "library"
					}
					if _, err := p.expect(TSemi, "';'"); err != nil {
						return nil, err
					}
					if prog.Kind != "" {
						return nil, p.errf(kind, "重复的 program 预制宏")
					}
					prog.Kind = kind.Text
					prog.kindSet = true
					continue
				case "import":
					// 预制宏：import path;（同目录默认在搜索范围）
					kw := p.cur()
					p.advance()
					var path string
					if p.cur().Kind == TStr {
						path = p.cur().Text
						p.advance()
					} else {
						n, err := p.expectIdent("import path")
						if err != nil {
							return nil, err
						}
						path = n.Text
					}
					if _, err := p.expect(TSemi, "';'"); err != nil {
						return nil, err
					}
					prog.Imports = append(prog.Imports, path)
					prog.ImportPos = append(prog.ImportPos, Pos{Line: kw.Line, Col: kw.Col})
					continue
				case "pub":
					// 预制宏：pub 前缀，公开下一个顶层符号（fn / type struct / type interface）
					p.advance()
					if p.curIs(TIdent) && p.cur().Text == "type" {
						p.advance()
						switch p.cur().Kind {
						case TStruct:
							sd, err := p.parseStruct()
							if err != nil {
								return nil, err
							}
							if sd.Name == "" {
								return nil, p.errf(p.cur(), "pub 的匿名结构体没有名字可导出：pub type struct { ... } Name;")
							}
							prog.Pub = append(prog.Pub, sd.Name)
							prog.Structs = append(prog.Structs, sd)
						case TInterface:
							id, err := p.parseInterface()
							if err != nil {
								return nil, err
							}
							if id.Name == "" {
								return nil, p.errf(p.cur(), "pub 的匿名接口没有名字可导出：pub type interface { ... } Name;")
							}
							prog.Pub = append(prog.Pub, id.Name)
							prog.Interfaces = append(prog.Interfaces, id)
						default:
							return nil, p.errf(p.cur(), "pub type 只支持 struct / interface")
						}
						continue
					}
					if p.cur().Kind != TFunc {
						return nil, p.errf(p.cur(), "pub 只能前缀于 fn 或 type struct/interface（实名类型必须写 type）")
					}
					fn, err := p.parseFunc()
					if err != nil {
						return nil, err
					}
					prog.Pub = append(prog.Pub, fn.Name)
					prog.Funcs = append(prog.Funcs, fn)
					continue
				}
			}
			if p.curIs(TIdent) && p.cur().Text == "func" {
				return nil, p.errf(p.cur(), "函数声明关键字是 fn：func main(...) 应写为 fn main(...)")
			}
			return nil, p.errf(p.cur(), "expected a top-level declaration (fn/struct/impl/interface/program/import/pub), got %s", p.cur().Kind)
		}
	}
	return prog, nil
}

func (p *parser) parseFunc() (*FuncDecl, error) {
	kw := p.cur()
	if !p.curIs(TFunc) {
		return nil, p.errf(p.cur(), "expected 'fn'")
	}
	p.advance()
	var typeParams []string
	if p.curIs(TLt) {
		// 泛型参数 <T, ...>（xmind §函数：泛型可有可无）
		p.advance()
		for {
			tp, err := p.expectIdent("type parameter")
			if err != nil {
				return nil, err
			}
			typeParams = append(typeParams, tp.Text)
			if p.curIs(TComma) {
				p.advance()
				continue
			}
			break
		}
		if _, err := p.expect(TGt, "'>'"); err != nil {
			return nil, err
		}
	}
	name, err := p.expectIdent("function name")
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TLParen, "'('"); err != nil {
		return nil, err
	}
	fn := &FuncDecl{Name: name.Text, TypeParams: typeParams, Pos: Pos{Line: kw.Line, Col: kw.Col}}
	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}
	fn.Params = params
	// 返回类型注解必填（新模型：函数必须声明返回类型；main 入口可省略，视为 void）
	if p.curIs(TIdent) || p.curIs(TInterface) || p.curIs(TStruct) {
		typ, err := p.parseType()
		if err != nil {
			return nil, err
		}
		fn.Ret = typ
	} else if name.Text == "main" {
		fn.Ret = "void"
	} else {
		return nil, p.errf(p.cur(), "函数必须声明返回类型：fn %s(...) 返回类型 { ... }", name.Text)
	}
	startTok := p.cur() // parseBlock 前：'{' 之后第一个 token（起始行）
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	fn.Body = body
	fn.BodyStart = Pos{Line: startTok.Line, Col: startTok.Col}
	if p.i > 0 {
		fn.BodyEnd = Pos{Line: p.toks[p.i-1].Line, Col: p.toks[p.i-1].Col} // 结束 '}'
	}
	return fn, nil
}

// parseParamList parses "(<修饰> <类型> <名字>, ...)"（形参与声明同一通式：
// 修饰 const/copyd 写在类型前，如 fn f(copyd int a)）。
func (p *parser) parseParamList() ([]Param, error) {
	var params []Param
	if !p.curIs(TRParen) {
		for {
			decor := ""
			// 修饰消歧：cur 是 const/copyd 且下一个 token 是类型（TIdent/TInterface）时按修饰解析
			if p.curIs(TIdent) && (p.cur().Text == "const" || p.cur().Text == "copyd") &&
				(p.peekIs(TIdent) || p.peekIs(TInterface) || p.peekIs(TStruct)) {
				decor = p.cur().Text
				p.advance()
			}
			typ, err := p.parseType()
			if err != nil {
				return nil, err
			}
			ptok, err := p.expectIdent("parameter name")
			if err != nil {
				return nil, err
			}
			params = append(params, Param{
				Name:  ptok.Text,
				Type:  typ,
				Decor: decor,
				Pos:   Pos{Line: ptok.Line, Col: ptok.Col},
			})
			if p.curIs(TComma) {
				p.advance()
				continue
			}
			break
		}
	}
	if _, err := p.expect(TRParen, "')'"); err != nil {
		return nil, err
	}
	return params, nil
}

// parseTypeParams parses "<T, U, ...>".
func (p *parser) parseTypeParams() ([]string, error) {
	if _, err := p.expect(TLt, "'<'"); err != nil {
		return nil, err
	}
	var params []string
	for {
		tok, err := p.expectIdent("type parameter")
		if err != nil {
			return nil, err
		}
		params = append(params, tok.Text)
		if p.curIs(TComma) {
			p.advance()
			continue
		}
		break
	}
	if _, err := p.expect(TGt, "'>'"); err != nil {
		return nil, err
	}
	return params, nil
}

// parseStruct parses "struct { members } [Name];".
func (p *parser) parseStruct() (*StructDecl, error) {
	kw, err := p.expect(TStruct, "'struct'")
	if err != nil {
		return nil, err
	}
	sd, err := p.parseStructBody(kw)
	if err != nil {
		return nil, err
	}
	if err := p.parseStructName(sd); err != nil {
		return nil, err
	}
	if _, err := p.expect(TSemi, "';'"); err != nil {
		return nil, err
	}
	return sd, nil
}

// parseStructBody 解析 struct[<T,...>] { 成员… }（不含名字与 ';'，实名/匿名共用一个解析体）。
func (p *parser) parseStructBody(kw Token) (*StructDecl, error) {
	sd := &StructDecl{Pos: Pos{Line: kw.Line, Col: kw.Col}}
	// 可选泛型参数：struct<T, U> { ... }
	if p.curIs(TLt) {
		params, err := p.parseTypeParams()
		if err != nil {
			return nil, err
		}
		sd.TypeParams = params
	}
	if _, err := p.expect(TLBrace, "'{"); err != nil {
		return nil, err
	}
	for !p.curIs(TRBrace) {
		if p.curIs(TEOF) {
			return nil, p.errf(p.cur(), "unterminated struct body (missing '}')")
		}
		typ, err := p.parseType()
		if err != nil {
			return nil, err
		}
		name, err := p.expectIdent("member name")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TSemi, "';'"); err != nil {
			return nil, err
		}
		sd.Members = append(sd.Members, Member{
			Name: name.Text,
			Type: typ,
			Pos:  Pos{Line: name.Line, Col: name.Col},
		})
	}
	p.advance() // '}'
	return sd, nil
}

// parseStructName 取结构体名字：} Name;（匿名结构体则无名）。
func (p *parser) parseStructName(sd *StructDecl) error {
	if p.curIs(TIdent) {
		sd.Name = p.advance().Text
	}
	return nil
}

// parseMethodSig parses an interface method signature: "fn name(params) Ret;".
func (p *parser) parseMethodSig() (MethodSig, error) {
	dyn := false
	if p.curIs(TIdent) && p.cur().Text == "dynamic" {
		p.advance()
		dyn = true
	}
	kw, err := p.expect(TFunc, "'fn'")
	if err != nil {
		return MethodSig{}, err
	}
	name, err := p.expectIdent("method name")
	if err != nil {
		return MethodSig{}, err
	}
	if _, err := p.expect(TLParen, "'('"); err != nil {
		return MethodSig{}, err
	}
	params, err := p.parseParamList()
	if err != nil {
		return MethodSig{}, err
	}
	sig := MethodSig{Name: name.Text, Params: params, Dynamic: dyn, Pos: Pos{Line: kw.Line, Col: kw.Col}}
	if p.curIs(TIdent) || p.curIs(TInterface) || p.curIs(TStruct) {
		typ, err := p.parseType()
		if err != nil {
			return MethodSig{}, err
		}
		sig.Ret = typ
	}
	if _, err := p.expect(TSemi, "';'"); err != nil {
		return MethodSig{}, err
	}
	return sig, nil
}

// parseInterface parses "interface { sigs } [Name<...>];".
func (p *parser) parseInterface() (*InterfaceDecl, error) {
	kw, err := p.expect(TInterface, "'interface'")
	if err != nil {
		return nil, err
	}
	id, err := p.parseInterfaceBody(kw)
	if err != nil {
		return nil, err
	}
	if p.curIs(TIdent) {
		id.Name = p.advance().Text
	}
	if _, err := p.expect(TSemi, "';'"); err != nil {
		return nil, err
	}
	return id, nil
}

// parseInterfaceBody 解析 interface[<T,...>] { 签名… }（不含名字与 ';'）。
func (p *parser) parseInterfaceBody(kw Token) (*InterfaceDecl, error) {
	id := &InterfaceDecl{Pos: Pos{Line: kw.Line, Col: kw.Col}}
	if p.curIs(TLt) { // 泛型接口：interface<T, ...> { ... } Name;
		tps, err := p.parseTypeParams()
		if err != nil {
			return nil, err
		}
		id.TypeParams = tps
	}
	if _, err := p.expect(TLBrace, "'{"); err != nil {
		return nil, err
	}
	for !p.curIs(TRBrace) {
		if p.curIs(TEOF) {
			return nil, p.errf(p.cur(), "unterminated interface body (missing '}')")
		}
		// expand interface Name; 组合接口行（xmind §接口；可带 dynamic 前缀）
		if p.curIs(TIdent) && (p.cur().Text == "expand" || (p.cur().Text == "dynamic" && p.peekTextIs("expand"))) {
			if p.cur().Text == "dynamic" {
				p.advance()
			}
			p.advance()
			if _, err := p.expect(TInterface, "'interface'"); err != nil {
				return nil, err
			}
			n, err := p.expectIdent("expanded interface name")
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TSemi, "';'"); err != nil {
				return nil, err
			}
			id.Expands = append(id.Expands, n.Text)
			continue
		}
		sig, err := p.parseMethodSig()
		if err != nil {
			return nil, err
		}
		id.Methods = append(id.Methods, sig)
	}
	p.advance() // '}'
	return id, nil
}

// parseAnonTypeName 解析匿名 struct { ... } / interface { ... } 类型标注：
// 合成实名类型（__anon_struct_N / __anon_iface_N）登记进 Program（解释器/编译器按实名类型处理），
// 返回该合成名。字段/方法集合等价的匿名类型复用同一个合成名。
func (p *parser) parseAnonTypeName() (string, error) {
	kw := p.advance() // struct | interface
	if kw.Kind == TStruct {
		sd, err := p.parseStructBody(kw)
		if err != nil {
			return "", err
		}
		if len(sd.TypeParams) > 0 {
			return "", p.errf(kw, "匿名 struct 类型不支持泛型参数（请写 type struct<T> { ... } Name;）")
		}
		key := anonStructKey(sd)
		if n, ok := p.anonByKey[key]; ok {
			return n, nil
		}
		p.anonSeq++
		name := fmt.Sprintf("__anon_struct_%d", p.anonSeq)
		sd.Name = name
		if p.prog != nil {
			p.prog.Structs = append(p.prog.Structs, sd)
		}
		p.anonByKey[key] = name
		return name, nil
	}
	id, err := p.parseInterfaceBody(kw)
	if err != nil {
		return "", err
	}
	if len(id.TypeParams) > 0 {
		return "", p.errf(kw, "匿名 interface 类型不支持泛型参数（请写 type interface<T> { ... } Name;）")
	}
	if len(id.Methods) == 0 && len(id.Expands) == 0 {
		return "interface{}", nil // 空接口 = void（既有语义不变）
	}
	key := anonIfaceKey(id)
	if n, ok := p.anonByKey[key]; ok {
		return n, nil
	}
	p.anonSeq++
	name := fmt.Sprintf("__anon_iface_%d", p.anonSeq)
	id.Name = name
	if p.prog != nil {
		p.prog.Interfaces = append(p.prog.Interfaces, id)
	}
	p.anonByKey[key] = name
	return name, nil
}

// anonStructKey 匿名 struct 的结构键（字段顺序 + 名字 + 类型）。
func anonStructKey(sd *StructDecl) string {
	var sb strings.Builder
	for _, m := range sd.Members {
		sb.WriteString(m.Name)
		sb.WriteByte(':')
		sb.WriteString(m.Type)
		sb.WriteByte(';')
	}
	return sb.String()
}

// anonIfaceKey 匿名 interface 的结构键（方法 + expand 组合；顺序无关）。
func anonIfaceKey(id *InterfaceDecl) string {
	parts := make([]string, 0, len(id.Methods))
	for _, m := range id.Methods {
		var sb strings.Builder
		sb.WriteString(m.Name)
		if m.Dynamic {
			sb.WriteString("!dynamic")
		}
		sb.WriteByte('(')
		for _, prm := range m.Params {
			sb.WriteString(prm.Decor)
			sb.WriteByte(' ')
			sb.WriteString(prm.Type)
			sb.WriteByte(',')
		}
		sb.WriteString(")->")
		sb.WriteString(m.Ret)
		parts = append(parts, sb.String())
	}
	sort.Strings(parts)
	out := strings.Join(parts, "|")
	if len(id.Expands) > 0 {
		ex := append([]string(nil), id.Expands...)
		sort.Strings(ex)
		out += " expand:" + strings.Join(ex, ",")
	}
	return out
}

// parseImpl parses "impl [Iface] { funcs } Type;".
func (p *parser) parseImpl() (*ImplDecl, error) {
	kw, err := p.expect(TImpl, "'impl'")
	if err != nil {
		return nil, err
	}
	im := &ImplDecl{Pos: Pos{Line: kw.Line, Col: kw.Col}}
	// 可选泛型参数：impl<T> [Iface] { ... }
	if p.curIs(TLt) {
		params, err := p.parseTypeParams()
		if err != nil {
			return nil, err
		}
		im.TypeParams = params
	}
	if _, err := p.expect(TLBrace, "'{"); err != nil {
		return nil, err
	}
	for !p.curIs(TRBrace) {
		if p.curIs(TEOF) {
			return nil, p.errf(p.cur(), "unterminated impl body (missing '}')")
		}
		fn, err := p.parseFunc()
		if err != nil {
			return nil, err
		}
		im.Methods = append(im.Methods, fn)
	}
	p.advance() // '}'
	// impl<T, ...> { ... } Name;  名字必须在块后（xmind §类）
	name, err := p.expectIdent("impl type name")
	if err != nil {
		return nil, err
	}
	im.Type = name.Text
	if _, err := p.expect(TSemi, "';'"); err != nil {
		return nil, err
	}
	return im, nil
}

// parseType reads a type annotation: 类型基名 ( "<" ... ">" )* ( "[" ... "]" )? ( "&" )?。
// 类型基名可以是标识符、interface{}（空接口）、匿名 struct { ... } / interface { ... }
// （匿名类型合成实名类型登记进 Program，等价结构复用同一个名字）。
func (p *parser) parseType() (string, error) {
	if p.curIs(TInterface) || p.curIs(TStruct) {
		name, err := p.parseAnonTypeName()
		if err != nil {
			return "", err
		}
		if name == "interface{}" {
			return name, nil // 空接口 = void（无后缀可言）
		}
		if p.curIs(TLBracket) && !p.peekIs(TInt) && !p.peekIs(TMinus) {
			p.advance()
			if p.curIs(TRBracket) {
				p.advance()
				name += "[]"
			} else {
				inner, err := p.expectIdent("type suffix (e.g. Copyd)")
				if err != nil {
					return "", err
				}
				if _, err := p.expect(TRBracket, "']'"); err != nil {
					return "", err
				}
				name += "[" + inner.Text + "]"
			}
		}
		if p.curIs(TAmper) {
			p.advance()
			name += "&"
		}
		return name, nil
	}
	tok, err := p.expectIdent("type name")
	if err != nil {
		return "", err
	}
	name := tok.Text
	if name == "function" && p.curIs(TLt) {
		// 函数类型：function<ret, p1, p2, ...>（函数引用）
		name += "<"
		p.advance()
		depth := 1
		for depth > 0 {
			if p.curIs(TEOF) {
				return "", p.errf(p.cur(), "unterminated function type")
			}
			if p.curIs(TLt) {
				depth++
			}
			if p.curIs(TGt) {
				depth--
			}
			name += p.cur().Text
			p.advance()
		}
		return name, nil
	}
	if name == "pointer" {
		// pointer 修饰：pointer <type>（等价 T&）；裸 pointer = 不透明句柄（FFI void*，可空、可往返）
		if !p.curIs(TIdent) && !p.curIs(TInterface) && !p.curIs(TStruct) {
			return "pointer", nil
		}
		// 消歧：pointer p（p 后紧跟 , ) ; =）是「裸 pointer + 参数/变量名」，不是「pointer 指向 p」
		switch p.peek().Kind {
		case TComma, TRParen, TSemi, TAssign:
			return "pointer", nil
		}
		rest, err := p.parseType()
		if err != nil {
			return "", err
		}
		return "pointer " + rest, nil
	}
	for p.curIs(TLt) {
		name += "<"
		p.advance()
		depth := 1
		for depth > 0 {
			if p.curIs(TEOF) {
				return "", p.errf(p.cur(), "unterminated type arguments in %q", name)
			}
			if p.curIs(TLt) {
				depth++
			}
			if p.curIs(TGt) {
				depth--
			}
			if p.curIs(TShr) {
				depth -= 2
				name += ">>"
				p.advance()
				continue
			}
			name += p.cur().Text
			p.advance()
		}
	}
	if p.curIs(TLBracket) && !p.peekIs(TInt) && !p.peekIs(TMinus) {
		// 类型后缀 [Copyd]/[]：数字开头的 [ 是 new <type>[size] 的 size，不消费
		p.advance()
		if p.curIs(TRBracket) {
			p.advance()
			name += "[]"
		} else {
			inner, err := p.expectIdent("type suffix (e.g. Copyd)")
			if err != nil {
				return "", err
			}
			if _, err := p.expect(TRBracket, "']'"); err != nil {
				return "", err
			}
			name += "[" + inner.Text + "]"
		}
	}
	// & 后缀：指针类型（如 node<T>&）
	if p.curIs(TAmper) {
		p.advance()
		name += "&"
	}
	return name, nil
}

func (p *parser) parseBlock() (*Block, error) {
	if _, err := p.expect(TLBrace, "'{'"); err != nil {
		return nil, err
	}
	b := &Block{}
	for !p.curIs(TRBrace) {
		if p.curIs(TEOF) {
			return nil, p.errf(p.cur(), "unterminated block (missing '}')")
		}
		st, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		b.Stmts = append(b.Stmts, st)
	}
	p.advance() // '}'
	return b, nil
}

func (p *parser) parseStmt() (Stmt, error) {
	// break; 跳出循环（while/for 体内）
	if p.curIs(TIdent) && p.cur().Text == "break" {
		bp := Pos{Line: p.cur().Line, Col: p.cur().Col}
		p.advance()
		if p.curIs(TSemi) {
			p.advance()
		}
		return &BreakStmt{Pos: bp}, nil
	}
	// delete variable; 语句（xmind 内存：回收内存于 __delete__()）
	if p.curIs(TIdent) && p.cur().Text == "delete" {
		kw := p.advance()
		x, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TSemi, "';'"); err != nil {
			return nil, err
		}
		return &DeleteStmt{X: x, Pos: Pos{Line: kw.Line, Col: kw.Col}}, nil
	}
	switch p.cur().Kind {
	case TTry:
		kw := p.advance()
		tryB, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TCatch, "'catch'"); err != nil {
			return nil, err
		}
		if _, err := p.expect(TLParen, "'('"); err != nil {
			return nil, err
		}
		ct, err := p.parseType()
		if err != nil {
			return nil, err
		}
		cv, err := p.expectIdent("catch variable")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TRParen, "')'"); err != nil {
			return nil, err
		}
		catchB, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &TryStmt{Try: tryB, CatchVar: cv.Text, CatchVarType: ct, Catch: catchB, Pos: Pos{Line: kw.Line, Col: kw.Col}}, nil
	case TLog:
		kw := p.advance()
		x, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TSemi, "';'"); err != nil {
			return nil, err
		}
		return &LogStmt{X: x, Pos: Pos{Line: kw.Line, Col: kw.Col}}, nil
	case TReturn:
		kw := p.advance()
		pos := Pos{Line: kw.Line, Col: kw.Col}
		if p.curIs(TSemi) {
			p.advance()
			return &ReturnStmt{Pos: pos}, nil
		}
		x, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TSemi, "';'"); err != nil {
			return nil, err
		}
		return &ReturnStmt{X: x, Pos: pos}, nil
	case TIf:
		p.advance()
		if _, err := p.expect(TLParen, "'('"); err != nil {
			return nil, err
		}
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TRParen, "')'"); err != nil {
			return nil, err
		}
		thenB, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		st := &IfStmt{Cond: cond, Then: thenB}
		if p.curIs(TElse) {
			p.advance()
			if p.curIs(TIf) {
				// else if 链：嵌套 if 包装为 else 块（语义一致）
				nested, err := p.parseStmt()
				if err != nil {
					return nil, err
				}
				st.Else = &Block{Stmts: []Stmt{nested}}
				return st, nil
			}
			elseB, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			st.Else = elseB
		}
		return st, nil
	case TWhile:
		p.advance()
		if _, err := p.expect(TLParen, "'('"); err != nil {
			return nil, err
		}
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TRParen, "')'"); err != nil {
			return nil, err
		}
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &WhileStmt{Cond: cond, Body: body}, nil
	case TFor:
		kw := p.advance()
		if _, err := p.expect(TLParen, "'('"); err != nil {
			return nil, err
		}
		// 迭代形式：for (<type> <name> : <expr>) { ... }
		save := p.i
		typ, terr := p.parseType()
		if terr == nil && p.curIs(TIdent) {
			v := p.advance()
			if p.curIs(TColon) {
				p.advance()
				iter, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				if _, err := p.expect(TRParen, "')'"); err != nil {
					return nil, err
				}
				body, err := p.parseBlock()
				if err != nil {
					return nil, err
				}
				return &ForStmt{Var: v.Text, Type: typ, Iter: iter, Body: body, Pos: Pos{Line: kw.Line, Col: kw.Col}}, nil
			}
		}
		p.i = save
		// C 风格：for (<init>; <cond>; <step>) { ... }
		init, err := p.parseForInit()
		if err != nil {
			return nil, err
		}
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TSemi, "';'"); err != nil {
			return nil, err
		}
		step, err := p.parseForStep()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TRParen, "')'"); err != nil {
			return nil, err
		}
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return &ForCStmt{Init: init, Cond: cond, Step: step, Body: body, Pos: Pos{Line: kw.Line, Col: kw.Col}}, nil
	}
	// 变量声明：<修饰> <类型> <名字> [= 初值];（xmind §变量：类型在前，唯一形态）
	if st, ok, err := p.tryParseDecl(); err != nil {
		return nil, err
	} else if ok {
		return st, nil
	}
	x, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.curIs(TAssign) {
		eq := p.advance()
		v, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TSemi, "';'"); err != nil {
			return nil, err
		}
		return &AssignStmt{Target: x, X: v, Pos: Pos{Line: eq.Line, Col: eq.Col}}, nil
	}
	if _, err := p.expect(TSemi, "';'"); err != nil {
		return nil, err
	}
	return &ExprStmt{X: x}, nil
}

// parseDecl parses "name Type [= init];".
// tryParseDecl 解析「<修饰> <类型> <名字> [= 初值];」；不是声明则回退（ok=false），由调用方按表达式解析。
func (p *parser) tryParseDecl() (Stmt, bool, error) {
	save := p.i
	decor := ""
	if p.curIs(TIdent) && (p.cur().Text == "const" || p.cur().Text == "copyd") {
		decor = p.cur().Text
		p.advance()
	}
	typ, err := p.parseType()
	if err != nil {
		p.i = save
		return nil, false, nil
	}
	if !p.curIs(TIdent) {
		p.i = save
		return nil, false, nil
	}
	name := p.advance()
	st := &DeclStmt{Name: name.Text, Type: typ, Decor: decor, Pos: Pos{Line: name.Line, Col: name.Col}}
	if p.curIs(TAssign) {
		p.advance()
		init, err := p.parseExpr()
		if err != nil {
			return nil, false, err
		}
		st.Init = init
	}
	if !p.curIs(TSemi) {
		p.i = save
		return nil, false, nil
	}
	p.advance()
	return st, true, nil
}

// parseForInit 解析 C 风格 for 的初始化：<类型> <名字> [= 初值]; 或赋值/表达式 + ';'。
func (p *parser) parseForInit() (Stmt, error) {
	if st, ok, err := p.tryParseDecl(); err != nil {
		return nil, err
	} else if ok {
		return st, nil
	}
	x, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	var st Stmt
	if p.curIs(TAssign) {
		eq := p.advance()
		v, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		st = &AssignStmt{Target: x, X: v, Pos: Pos{Line: eq.Line, Col: eq.Col}}
	} else {
		st = &ExprStmt{X: x}
	}
	if _, err := p.expect(TSemi, "';'"); err != nil {
		return nil, err
	}
	return st, nil
}

// parseForStep 解析 C 风格 for 的步进（可为空）。
func (p *parser) parseForStep() (Stmt, error) {
	if p.curIs(TRParen) {
		return nil, nil
	}
	x, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.curIs(TAssign) {
		eq := p.advance()
		v, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &AssignStmt{Target: x, X: v, Pos: Pos{Line: eq.Line, Col: eq.Col}}, nil
	}
	return &ExprStmt{X: x}, nil
}

// ---- expressions ----

func (p *parser) parseExpr() (Expr, error) { return p.parseOr() }

func (p *parser) parseOr() (Expr, error)  { return p.parseBin(p.parseAnd, TOr) }
func (p *parser) parseAnd() (Expr, error) { return p.parseBin(p.parseCmp, TAnd) }
func (p *parser) parseCmp() (Expr, error) {
	return p.parseBin(p.parseAdd, TEq, TNe, TLt, TLe, TGt, TGe)
}
func (p *parser) parseAdd() (Expr, error)   { return p.parseBin(p.parseShift, TPlus, TMinus) }
func (p *parser) parseMul() (Expr, error)   { return p.parseBin(p.parseUnary, TStar, TSlash, TPercent) }
func (p *parser) parseShift() (Expr, error) { return p.parseBin(p.parseMul, TShl, TShr) }

func (p *parser) parseBin(left func() (Expr, error), kinds ...TokenKind) (Expr, error) {
	x, err := left()
	if err != nil {
		return nil, err
	}
	for {
		matched := false
		for _, k := range kinds {
			if p.curIs(k) {
				matched = true
				break
			}
		}
		if !matched {
			return x, nil
		}
		op := p.advance()
		r, err := left()
		if err != nil {
			return nil, err
		}
		x = &BinOp{Op: op.Text, L: x, R: r, Pos: Pos{Line: op.Line, Col: op.Col}}
	}
}

func (p *parser) parseUnary() (Expr, error) {
	if p.curIs(TBang) || p.curIs(TMinus) || p.curIs(TStar) {
		tok := p.advance()
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnOp{Op: tok.Text, X: x, Pos: Pos{Line: tok.Line, Col: tok.Col}}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Expr, error) {
	x, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch {
		case p.curIs(TDot):
			p.advance()
			name, err := p.expectMemberName("member name")
			if err != nil {
				return nil, err
			}
			pos := Pos{Line: name.Line, Col: name.Col}
			if p.curIs(TLParen) {
				args, err := p.parseArgs()
				if err != nil {
					return nil, err
				}
				x = &CallExpr{Fn: &MemberExpr{X: x, Name: name.Text, Pos: pos}, Args: args, Pos: pos, FnIdx: -1}
			} else {
				x = &MemberExpr{X: x, Name: name.Text, Pos: pos}
			}
		case p.curIs(TLParen):
			lp := p.cur()
			args, err := p.parseArgs()
			if err != nil {
				return nil, err
			}
			x = &CallExpr{Fn: x, Args: args, Pos: Pos{Line: lp.Line, Col: lp.Col}, FnIdx: -1}
		case p.curIs(TLBracket):
			p.advance()
			idx, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			rt, err := p.expect(TRBracket, "']'")
			if err != nil {
				return nil, err
			}
			x = &IndexExpr{X: x, Idx: idx, Pos: Pos{Line: rt.Line, Col: rt.Col}}
		case p.curIs(TScope):
			p.advance()
			name, err := p.expectMemberName("method name")
			if err != nil {
				return nil, err
			}
			args, err := p.parseArgs()
			if err != nil {
				return nil, err
			}
			id, ok := x.(*Ident)
			if !ok {
				return nil, p.errf(name, "'::' requires an identifier on the left")
			}
			x = &ScopeCall{Scope: id.Name, Name: name.Text, Args: args, Pos: Pos{Line: name.Line, Col: name.Col}}
		case p.curIs(TAt):
			p.advance()
			sname, err := p.expectIdent("signature name")
			if err != nil {
				return nil, err
			}
			sargs, err := p.parseArgs()
			if err != nil {
				return nil, err
			}
			call, ok := x.(*CallExpr)
			if !ok {
				return nil, p.errf(sname, "@ signature must follow a function call")
			}
			if call.Sign != nil {
				return nil, p.errf(sname, "duplicate signature on one call")
			}
			call.Sign = &SignCall{Name: sname.Text, Args: sargs}
		default:
			return x, nil
		}
	}
}

func (p *parser) parseArgs() ([]Expr, error) {
	if _, err := p.expect(TLParen, "'('"); err != nil {
		return nil, err
	}
	var args []Expr
	if !p.curIs(TRParen) {
		for {
			a, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			args = append(args, a)
			if p.curIs(TComma) {
				p.advance()
				continue
			}
			break
		}
	}
	if _, err := p.expect(TRParen, "')'"); err != nil {
		return nil, err
	}
	return args, nil
}

func (p *parser) parsePrimary() (Expr, error) {
	tok := p.cur()
	pos := Pos{Line: tok.Line, Col: tok.Col}
	switch tok.Kind {
	case TInt:
		p.advance()
		if tok.Int > 2147483647 || tok.Int < -2147483648 {
			return nil, p.errf(tok, "int literal %d out of 32-bit range (int is 32-bit, like C)", tok.Int)
		}
		return &IntLit{V: tok.Int, Pos: pos}, nil
	case TFloat:
		p.advance()
		return &FloatLit{V: tok.Flt, Pos: pos}, nil
	case TStr:
		p.advance()
		return &StrLit{V: tok.Text, Pos: pos}, nil
	case TTrue:
		p.advance()
		return &BoolLit{V: true, Pos: pos}, nil
	case TFalse:
		p.advance()
		return &BoolLit{V: false, Pos: pos}, nil
	case TNull:
		p.advance()
		return &NullLit{Pos: pos}, nil
	case TDot:
		// 结构体字面量/展开式：.{ field: expr, ... }
		p.advance()
		if _, err := p.expect(TLBrace, "'{'"); err != nil {
			return nil, err
		}
		lit := &StructLit{Pos: pos}
		if !p.curIs(TRBrace) {
			for {
				// 位置形式 .{v1, v2}（与编译器一致）：冒号后跟值的是命名形式，否则位置
				if p.peekIs(TColon) {
					name := p.advance()
					p.advance() // :
					v, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					lit.Fields = append(lit.Fields, StructLitField{Name: name.Text, X: v})
				} else {
					v, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					lit.Fields = append(lit.Fields, StructLitField{Name: "", X: v})
				}
				if p.curIs(TComma) {
					p.advance()
					continue
				}
				break
			}
		}
		if _, err := p.expect(TRBrace, "'}'"); err != nil {
			return nil, err
		}
		return lit, nil
	case TIdent:
		if tok.Text == "new" {
			// new <type>[size] —— 堆上直接申请内存（失败 badAlloc）
			p.advance()
			typ, err := p.parseType()
			if err != nil {
				return nil, err
			}
			ne := &NewExpr{Typ: typ, Pos: pos}
			if p.curIs(TLBracket) {
				p.advance()
				sz, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				if _, err := p.expect(TRBracket, "']'"); err != nil {
					return nil, err
				}
				ne.Size = sz
			}
			return ne, nil
		}
		p.advance()
		return &Ident{Name: tok.Text, Pos: pos}, nil
	case TLBracket:
		p.advance()
		l := &ListLit{}
		if !p.curIs(TRBracket) {
			for {
				it, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				l.Items = append(l.Items, it)
				if p.curIs(TComma) {
					p.advance()
					continue
				}
				break
			}
		}
		if _, err := p.expect(TRBracket, "']'"); err != nil {
			return nil, err
		}
		return l, nil
	case TLParen:
		p.advance()
		x, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TRParen, "')'"); err != nil {
			return nil, err
		}
		return x, nil
	}
	return nil, p.errf(tok, "unexpected token %s in expression", tok.Kind)
}
