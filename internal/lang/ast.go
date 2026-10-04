package lang

// Pos is a source position.
type Pos struct {
	Line int
	Col  int
}

// Program is a parsed QuarkLang program.
type TypeAlias struct {
	Name string
	Type string
	Pos  Pos
}

type Program struct {
	FnList      []*FuncDecl // function table (eval indexes it directly, avoiding a map)
	FnIndex     map[string]int
	Funcs       []*FuncDecl
	Structs     []*StructDecl
	Libraries   []*LibraryDecl
	Interfaces  []*InterfaceDecl
	Impls       []*ImplDecl
	TypeAliases []*TypeAlias
	Kind        string // "main" (default) | "library" (program macro)
	kindSet     bool
	Imports     []string
	ImportPos   []Pos    // one-to-one with Imports: position of the `import` keyword (used by tools to locate errors)
	Pub         []string // pub macro: names of the symbols exported from the library
	Src         string   // original source (sliced by function-body line range when exporting a library)
}

// MacroDef is a `#macro name (params...) { body }` definition (named-parameter macro; parameters are substituted by name).
type MacroDef struct {
	Name   string
	Params []string
	Body   []Token
	Pos    Pos
}

// FuncDecl is a top-level function declaration. Ret is the optional return
// type annotation: functions WITHOUT Ret yield a FuncBuffer (out -> tail),
// functions WITH Ret yield the value of `return expr;` directly.
// LibraryDecl is a system library binding declaration:
//
//	library gl3 { fn ClearColor(x f32, y f32, z f32, w f32) void; ... }
//
// Essentially `gl3 ... = library("gl3")` -- a library object; the fn entries in its body are exported symbol signatures (ABI declarations).
type LibraryDecl struct {
	Name    string
	Lib     string // system library name (for dlopen: "libGL.so.1" / "opengl32")
	Methods []*Func
	Pos     Pos
}

type FuncDecl struct {
	Name       string
	TypeParams []string // generic function func<T, ...> (xmind §functions)
	Params     []Param
	Ret        string
	Body       *Block
	BodyStart  Pos // source line range of the function body (used for library export)
	BodyEnd    Pos
	Pos        Pos
}

// Param is a function parameter ("<modifier> <type> <name>").
// Decor is the declaration modifier ("" | "const" | "copyd", same table as DeclStmt.Decor):
// a copyd parameter is deep-copied when bound (copy on pass); a const parameter cannot be assigned inside the callee.
type Param struct {
	Name  string
	Type  string
	Decor string
	Perm  string // permission stack r/w/m for a reference parameter ("&rw u int p"), "" when absent
	Scope string // follow layer u/f/a/t for a reference parameter, "" when absent
	Pos   Pos
}

// Member is a struct member declaration ("name Type;").
type Member struct {
	Name string
	Type string
	Pos  Pos
}

// StructDecl is a struct declaration: "struct<T> { ... } Name;".
type StructDecl struct {
	Name       string
	TypeParams []string
	Members    []Member
	Pos        Pos
}

// MethodSig is an interface method signature (no body).
type MethodSig struct {
	Name    string
	Params  []Param
	Ret     string
	Dynamic bool // interface method with the `dynamic` modifier: dispatched dynamically at runtime (protocols such as Operation rely on this)
	Pos     Pos
}

// InterfaceDecl is an interface declaration.
type InterfaceDecl struct {
	Name       string
	TypeParams []string // generic interface: interface<T, ...> { ... } Name;
	Methods    []MethodSig
	Expands    []string // expand interface composes interfaces (xmind §interfaces)
	Pos        Pos
}

// ImplDecl is an impl declaration: "impl<T> [Iface] { funcs } Type;".
// Rule: when a struct has generic parameters, the impl must introduce the same parameters.
type ImplDecl struct {
	Iface      string
	Type       string
	TypeParams []string
	Methods    []*FuncDecl
	Pos        Pos
}

// Block is a brace-delimited statement list.
type Block struct {
	Stmts []Stmt
}

// Stmt is any statement.
type Stmt interface{ isStmt() }

// Expr is any expression.
type Expr interface{ isExpr() }

// ---- statements ----

type ExprStmt struct{ X Expr }
type LogStmt struct {
	X   Expr
	Pos Pos
}

// DeleteStmt: `delete variable;` -- reclaims the memory in __delete__(); essentially appends an elimination record to that block's log.
type DeleteStmt struct {
	X   Expr
	Pos Pos
}

// BreakStmt breaks out (only inside a while/for loop body).
type BreakStmt struct {
	Pos Pos
}

func (b *BreakStmt) isStmt() {}

type TryStmt struct {
	Try          *Block
	CatchVar     string
	CatchVarType string
	Catch        *Block
	Pos          Pos
}
type ReturnStmt struct {
	X   Expr // nil X = bare return
	Pos Pos
}
type IfStmt struct {
	Cond Expr
	Then *Block
	Else *Block // nil when absent
}
type WhileStmt struct {
	Cond Expr
	Body *Block
}
type ForStmt struct {
	Var  string
	Type string // iteration variable type (type first): for (<type> <name> : <expr>)
	Iter Expr
	Body *Block
	Pos  Pos
}

// ForCStmt is a C-style for: for (<init>; <cond>; <step>) { ... }
type ForCStmt struct {
	Init Stmt // declaration (type first) or assignment/expression
	Cond Expr
	Step Stmt // assignment/expression, may be nil
	Body *Block
	Pos  Pos
}
type DeclStmt struct {
	Name  string
	Type  string
	Init  Expr   // nil = uninitialized
	Decor string // "" | "const" | "copyd" (xmind §variable modifiers)
	Perm  string // permission stack r/w/m for a reference declaration ("&rw u int p"), "" when absent
	Scope string // follow layer u/f/a/t for a reference declaration, "" when absent
	Pos   Pos
}
type AssignStmt struct {
	Target Expr
	X      Expr
	Pos    Pos
}

func (*ExprStmt) isStmt()   {}
func (*LogStmt) isStmt()    {}
func (*DeleteStmt) isStmt() {}
func (*TryStmt) isStmt()    {}
func (*ReturnStmt) isStmt() {}
func (*IfStmt) isStmt()     {}
func (*WhileStmt) isStmt()  {}
func (*ForStmt) isStmt()    {}
func (*ForCStmt) isStmt()   {}
func (*DeclStmt) isStmt()   {}
func (*AssignStmt) isStmt() {}

// ---- expressions ----

type IntLit struct {
	V   int64
	Pos Pos
}
type FloatLit struct {
	V   float64
	Pos Pos
}
type StrLit struct {
	V   string
	Pos Pos
}
type BoolLit struct {
	V   bool
	Pos Pos
}
type NullLit struct{ Pos Pos }
type Ident struct {
	Name string
	Pos  Pos
	// Slot is the scope slot resolved at compile time (0 = unresolved).
	// Written by slots.go where the index is provably stable; at runtime the name is still verified before use (safety fallback).
	Slot int32
}
type ListLit struct {
	Items []Expr
}
type StructLit struct {
	Fields []StructLitField
	Name   string // target named struct (filled in by typecheck); eval uses it to build the type
	Pos    Pos
}

// NewExpr: `new <type>[size]` -- allocates memory directly on the heap (returns badAlloc on failure).
type NewExpr struct {
	Typ  string
	Size Expr // nil = single element
	Pos  Pos
}
type StructLitField struct {
	Name string
	X    Expr
}
type BinOp struct {
	Op   string
	L, R Expr
	Pos  Pos
}
type UnOp struct {
	Op  string
	X   Expr
	Pos Pos
}
type CallExpr struct {
	Fn    Expr
	Args  []Expr
	Sign  *SignCall
	Pos   Pos
	FnIdx int // function index resolved at compile time (-1 = unresolved/variable call); eval indexes FnList directly, avoiding a map

	// Argument classification, computed once on the first evaluation (superinstruction-style bookkeeping):
	// when no argument can form an lvalue cell, the argument list is evaluated without the per-argument
	// lvalue probe that evalArg performs. Classification is stable for a given AST node.
	argsClassified bool
	plainArgs      bool
}
type SignCall struct {
	Name string
	Args []Expr
}
type MemberExpr struct {
	X    Expr
	Name string
	Pos  Pos
}
type ScopeCall struct {
	Scope string
	Name  string
	Args  []Expr
	Pos   Pos
}
type IndexExpr struct {
	X   Expr
	Idx Expr
	Pos Pos
}

func (*IntLit) isExpr()     {}
func (*FloatLit) isExpr()   {}
func (*StrLit) isExpr()     {}
func (*BoolLit) isExpr()    {}
func (*NullLit) isExpr()    {}
func (*Ident) isExpr()      {}
func (*ListLit) isExpr()    {}
func (*StructLit) isExpr()  {}
func (*NewExpr) isExpr()    {}
func (*BinOp) isExpr()      {}
func (*UnOp) isExpr()       {}
func (*CallExpr) isExpr()   {}
func (*MemberExpr) isExpr() {}
func (*ScopeCall) isExpr()  {}
func (*IndexExpr) isExpr()  {}
