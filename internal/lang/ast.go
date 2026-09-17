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
	FnList      []*FuncDecl // 函数表（eval 直取索引免 map）
	FnIndex     map[string]int
	Funcs       []*FuncDecl
	Structs     []*StructDecl
	Libraries   []*LibraryDecl
	Interfaces  []*InterfaceDecl
	Impls       []*ImplDecl
	TypeAliases []*TypeAlias
	Kind        string // "main"（默认）| "library"（program 宏）
	kindSet     bool
	Imports     []string
	ImportPos   []Pos    // 与 Imports 一一对应：import 关键字位置（工具报错定位用）
	Pub         []string // pub 宏：库中公开的符号名
	Src         string   // 原始源码（库导出时按函数体行区间切片）
}

// MacroDef 是 #macro name (参数...) { 主体 } 定义（命名参数宏，参数按名替换）。
type MacroDef struct {
	Name   string
	Params []string
	Body   []Token
	Pos    Pos
}

// FuncDecl is a top-level function declaration. Ret is the optional return
// type annotation: functions WITHOUT Ret yield a FuncBuffer (out -> tail),
// functions WITH Ret yield the value of `return expr;` directly.
// LibraryDecl 是系统库绑定声明：
//
//	library gl3 { fn ClearColor(x f32, y f32, z f32, w f32) void; ... }
//
// 本质：gl3 ... = library("gl3")——库对象；体内 fn 为导出符号签名（ABI 声明）。
type LibraryDecl struct {
	Name    string
	Lib     string // 系统库名（dlopen 用："libGL.so.1" / "opengl32"）
	Methods []*Func
	Pos     Pos
}

type FuncDecl struct {
	Name       string
	TypeParams []string // 泛型函数 func<T, ...>（xmind §函数）
	Params     []Param
	Ret        string
	Body       *Block
	BodyStart  Pos // 函数体源码行区间（库导出用）
	BodyEnd    Pos
	Pos        Pos
}

// Param is a function parameter ("<修饰> <类型> <名字>")。
// Decor 是声明修饰（"" | "const" | "copyd"，与 DeclStmt.Decor 同一张表）：
// copyd 形参在绑定时深拷贝（传时复制），const 形参不可在 callee 内赋值。
type Param struct {
	Name  string
	Type  string
	Decor string
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
	Dynamic bool // 接口方法 dynamic 修饰：运行时动态分发（Operation 等协议基于此）
	Pos     Pos
}

// InterfaceDecl is an interface declaration.
type InterfaceDecl struct {
	Name       string
	TypeParams []string // 泛型接口：interface<T, ...> { ... } Name;
	Methods    []MethodSig
	Expands    []string // expand interface 组合接口（xmind §接口）
	Pos        Pos
}

// ImplDecl is an impl declaration: "impl<T> [Iface] { funcs } Type;".
// 规则：struct 有泛型参数时，impl 必须引入同样的参数。
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

// DeleteStmt：delete variable; —— 回收内存于 __delete__()，本质是给对应 block 的日志加消除记录。
type DeleteStmt struct {
	X   Expr
	Pos Pos
}

// BreakStmt 跳出（仅限 while/for 循环体内）。
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
	Type string // 迭代变量类型（类型在前）：for (<type> <name> : <expr>)
	Iter Expr
	Body *Block
	Pos  Pos
}

// ForCStmt is a C-style for: for (<init>; <cond>; <step>) { ... }
type ForCStmt struct {
	Init Stmt // 声明（类型在前）或赋值/表达式
	Cond Expr
	Step Stmt // 赋值/表达式，可为 nil
	Body *Block
	Pos  Pos
}
type DeclStmt struct {
	Name  string
	Type  string
	Init  Expr   // nil = uninitialized
	Decor string // "" | "const" | "copyd"（xmind §变量修饰表）
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
	// Slot 是编译期解析出的作用域槽位（0 = 未解析）。
	// 由 slots.go 在「下标可证明稳定」处写入，运行时还会校验名字后才使用（安全兜底）。
	Slot int32
}
type ListLit struct {
	Items []Expr
}
type StructLit struct {
	Fields []StructLitField
	Name   string // 目标有名结构体（typecheck 填充），eval 用它建类型
	Pos    Pos
}

// NewExpr：new <type>[size] —— 在堆上直接申请内存（失败返回 badAlloc）。
type NewExpr struct {
	Typ  string
	Size Expr // nil = 单元素
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
	FnIdx int // 编译期解析的函数索引（-1 = 未解析/变量调用）；eval 直取 FnList 免 map
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
