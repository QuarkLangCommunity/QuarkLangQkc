package lang

// compile.go holds the parts of the old parser.go that are not the syntax layer: the cached
// Compile entry point, its slow path, and the AST walkers used to resolve call references.

import (
	"crypto/sha256"
	"reflect"
	"sync"
)

// compileCache is an in-process incremental compilation cache (source sha256 -> compiled Program).
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

// Compile lexes, parses and statically type-checks source code (spec §11.1: strict checks run at compile time).
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

// compileSlow is Compile's uncached path: lex, expand macros, parse, type-check and resolve slots.
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
