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

	prog      *Program          // the current program (anonymous struct/interface used as a type annotation is registered as a synthesized named type)
	anonByKey map[string]string // anonymous type structural key → synthesized name (the same structure reuses the same name)
	anonSeq   int
}

// Parse builds the AST from tokens (spec §2).
func Parse(toks []Token) (*Program, error) {
	p := &parser{toks: toks}
	return p.parseProgram()
}

// Compile lexes, parses, and statically type-checks source code
// (spec §11.1: strict checks run at compile time).
// compileCache: an in-process incremental compilation cache (source sha256 → compiled Program).
var compileCache sync.Map

// walkCallRefs walks statements/expressions and resolves CallExpr.FnIdx (an Ident that hits the function table).
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
	// nil test (including a typed nil pointer such as (*Block)(nil))
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

// mainBody returns the main function body (empty when there is no main).
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
	// Macro system: split out macro definitions, expand at token level (run state for the interpreter), then parse
	macros, rest, err := SplitMacroDefs(toks)
	if err != nil {
		return nil, err
	}
	if len(macros) > 0 {
		rest, err = ExpandMacros(rest, macros, "explain") // interpreter = at the explain operation time (xmind §operation time)
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
	// Resolve variable slots at compile time (the interpreter avoids linear name scans; the compiler path ignores the field)
	resolveSlots(prog)
	// Resolve function-call indices at compile time (eval avoids map lookups)
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

// peekAt returns the token n positions ahead (n = 0 is the current one), clamped to the last token.
func (p *parser) peekAt(n int) Token {
	if p.i+n < len(p.toks) {
		return p.toks[p.i+n]
	}
	return p.toks[len(p.toks)-1]
}

func (p *parser) peekIsAt(n int, k TokenKind) bool {
	return p.i+n < len(p.toks) && p.toks[p.i+n].Kind == k
}

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
	return &ParseError{Msg: msg(format, args...), Line: tok.Line, Col: tok.Col}
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

// isBuiltinTypeName reports whether a name is a builtin type (used by type-first declarations).
func isBuiltinTypeName(s string) bool {
	switch s {
	case "int", "float", "bool", "String", "void", "List", "HashTable", "Channel", "thread", "Task", "memorize", "memory", "IOStream", "Copyd",
		"istream", "ostream", "ifstream", "ofstream", "iofstream", "InputStream", "OutputStream":
		return true
	}
	return false
}

// isWordToken reports whether a token is an identifier or keyword (not punctuation/literal).
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
					// xmind: type interface<T>{...}(Name) / type struct<T>{...}(Name)
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
						// Generic type alias: type <type expression> Name;
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
					// space { ... } name;  self-impl space (xmind writes space {...} (name); where (name) marks the name slot)
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
					// library system binding: library <Ident or "string"> { fn signature; ... }; / library X;
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
							p.advance() // ';' is optional
						}
					} else {
						if _, err := p.expect(TSemi, "';'"); err != nil {
							return nil, err
						}
					}
					prog.Libraries = append(prog.Libraries, ld)
					continue
				case "program":
					// Predefined macro: program main; / program library;
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
					// Predefined macro: import path; (the same directory is searched by default)
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
					// Predefined macro: the pub prefix publishes the next top-level symbol (fn / type struct / type interface)
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
		// Generic parameters <T, ...> (xmind §functions: generics are optional)
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
	// The return type annotation is required (new model: functions must declare a return type; the main entry may omit it, treated as void)
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
	startTok := p.cur() // before parseBlock: the first token after '{' (start line)
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	fn.Body = body
	fn.BodyStart = Pos{Line: startTok.Line, Col: startTok.Col}
	if p.i > 0 {
		fn.BodyEnd = Pos{Line: p.toks[p.i-1].Line, Col: p.toks[p.i-1].Col} // closing '}'
	}
	return fn, nil
}

// parseParamList parses "(<modifier> <type> <name>, ...)" (parameters and declarations share one form:
// the const/copyd modifiers precede the type, e.g. fn f(copyd int a)).
func (p *parser) parseParamList() ([]Param, error) {
	var params []Param
	if !p.curIs(TRParen) {
		for {
			decor := ""
			perm, scope, hasPerm := p.parseRefPermission()
			if hasPerm {
				decor = "ref"
			}
			// Modifier disambiguation: when cur is const/copyd and the next token is a type (TIdent/TInterface), parse it as a modifier
			if !hasPerm && p.curIs(TIdent) && (p.cur().Text == "const" || p.cur().Text == "copyd") &&
				(p.peekIs(TIdent) || p.peekIs(TInterface) || p.peekIs(TStruct)) {
				decor = p.cur().Text
				p.advance()
			}
			typ, err := p.parseType()
			if err != nil {
				return nil, err
			}
			if hasPerm {
				typ = refBaseType(typ)
			}
			ptok, err := p.expectIdent("parameter name")
			if err != nil {
				return nil, err
			}
			params = append(params, Param{
				Name:  ptok.Text,
				Type:  typ,
				Decor: decor,
				Perm:  perm,
				Scope: scope,
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

// parseStructBody parses struct[<T,...>] { members… } (without the name and ';'; named and anonymous share one body parser).
func (p *parser) parseStructBody(kw Token) (*StructDecl, error) {
	sd := &StructDecl{Pos: Pos{Line: kw.Line, Col: kw.Col}}
	// Optional generic parameters: struct<T, U> { ... }
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

// parseStructName takes the struct name: } Name; (an anonymous struct has none).
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

// parseInterfaceBody parses interface[<T,...>] { signatures… } (without the name and ';').
func (p *parser) parseInterfaceBody(kw Token) (*InterfaceDecl, error) {
	id := &InterfaceDecl{Pos: Pos{Line: kw.Line, Col: kw.Col}}
	if p.curIs(TLt) { // generic interface: interface<T, ...> { ... } Name;
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
		// expand interface Name;  composes interfaces (xmind §interfaces; may carry the dynamic prefix)
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

// parseAnonTypeName parses an anonymous struct { ... } / interface { ... } type annotation:
// it synthesizes a named type (__anon_struct_N / __anon_iface_N) registered in the Program (the interpreter/compiler treat it as named),
// and returns that synthesized name. Structurally equivalent anonymous types reuse the same synthesized name.
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
		return "interface{}", nil // empty interface = void (existing semantics unchanged)
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

// anonStructKey is the structural key of an anonymous struct (field order + names + types).
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

// anonIfaceKey is the structural key of an anonymous interface (methods + expand composition; order-independent).
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
	// Optional generic parameters: impl<T> [Iface] { ... }
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
	// impl<T, ...> { ... } Name;  the name must come after the block (xmind §classes)
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

// parseType reads a type annotation: base name ( "<" ... ">" )* ( "[" ... "]" )? ( "&" )?.
// The base name may be an identifier, interface{} (the empty interface), or an anonymous struct { ... } / interface { ... }
// (anonymous types are synthesized into named types registered in the Program, and equivalent structures reuse one name).
func (p *parser) parseType() (string, error) {
	if p.curIs(TInterface) || p.curIs(TStruct) {
		name, err := p.parseAnonTypeName()
		if err != nil {
			return "", err
		}
		if name == "interface{}" {
			return name, nil // empty interface = void (no suffix to speak of)
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
		// Function type: function<ret, p1, p2, ...> (function references)
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
		// pointer modifier: pointer <type> (equivalent to T&); bare pointer = an opaque handle (FFI void*, nullable, round-trippable)
		if !p.curIs(TIdent) && !p.curIs(TInterface) && !p.curIs(TStruct) {
			return "pointer", nil
		}
		// Disambiguation: pointer p (p followed by , ) ; =) is "bare pointer + a parameter/variable name", not "pointer to p"
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
		// Type suffixes [Copyd]/[]: a '[' followed by a digit is the size of new <type>[size] and is not consumed here
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
	// & suffix: pointer type (such as node<T>&)
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
	// break;  leaves a loop (inside a while/for body)
	if p.curIs(TIdent) && p.cur().Text == "break" {
		bp := Pos{Line: p.cur().Line, Col: p.cur().Col}
		p.advance()
		if p.curIs(TSemi) {
			p.advance()
		}
		return &BreakStmt{Pos: bp}, nil
	}
	// delete variable;  statement (xmind §memory: reclaim memory in __delete__())
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
				// else if chain: a nested if is wrapped in an else block (same semantics)
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
		// Iteration form: for (<type> <name> : <expr>) { ... }
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
		// C style: for (<init>; <cond>; <step>) { ... }
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
	// Variable declaration: <modifier> <type> <name> [= initializer]; (xmind §variables: type-first, the only form)
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
// parseRefPermission parses the leading "&<perm> <scope>" of a reference declaration, e.g.
// "&rw u int p = &x;" or "&rm f int q = &a[1];". perm is a stack of r/w/m, scope is u/f/a/t
// (BioLang's typed smart references). It reports ok=false when this is not such a form, so the
// caller can fall back to parsing an ordinary declaration or an expression.
func (p *parser) parseRefPermission() (perm, scope string, ok bool) {
	if !p.curIs(TAmper) {
		return "", "", false
	}
	// Lookahead only: an address-of expression such as &x must not be mistaken for a declaration.
	if !p.peekIs(TIdent) || !p.peekIsAt(2, TIdent) {
		return "", "", false
	}
	pm := p.peekAt(1).Text
	if !validRefPerm(pm) {
		return "", "", false
	}
	sc := p.peekAt(2).Text
	if !validRefScope(sc) {
		return "", "", false
	}
	// The token after the scope must start a type, otherwise this is not a declaration.
	if !(p.peekIsAt(3, TIdent) || p.peekIsAt(3, TInterface) || p.peekIsAt(3, TStruct) || p.peekIsAt(3, TStar)) {
		return "", "", false
	}
	p.advance() // &
	p.advance() // perm
	p.advance() // scope
	return pm, sc, true
}

// tryParseDecl parses "<modifier> <type> <name> [= initializer];"; when it is not a declaration it falls back (ok=false) and the caller parses an expression.
func (p *parser) tryParseDecl() (Stmt, bool, error) {
	save := p.i
	decor := ""
	perm, scope, hasPerm := p.parseRefPermission()
	if hasPerm {
		decor = "ref"
	}
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
	if hasPerm {
		typ = refBaseType(typ)
	}
	st := &DeclStmt{Name: name.Text, Type: typ, Decor: decor, Perm: perm, Scope: scope, Pos: Pos{Line: name.Line, Col: name.Col}}
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

// parseForInit parses a C-style for initializer: <type> <name> [= initializer]; or an assignment/expression + ';'.
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

// parseForStep parses a C-style for step (may be empty).
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
	return p.parseBin(p.parseAdd, TEqStrict, TEq, TNe, TLt, TLe, TGt, TGe)
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
	// & takes an lvalue's address (BioLang-style references). Declaration forms such as
	// "&rw u int p = &x;" are recognised by tryParseDecl before expression parsing ever sees them,
	// so an & reaching this point is an address-of expression.
	if p.curIs(TBang) || p.curIs(TMinus) || p.curIs(TStar) || p.curIs(TAmper) {
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
		// Struct literal / spread form: .{ field: expr, ... }
		p.advance()
		if _, err := p.expect(TLBrace, "'{'"); err != nil {
			return nil, err
		}
		lit := &StructLit{Pos: pos}
		if !p.curIs(TRBrace) {
			for {
				// Positional form .{v1, v2} (same as the compiler): a value after a colon means the named form, otherwise positional
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
			// new <type>[size] — allocate memory directly on the heap (badAlloc on failure)
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
