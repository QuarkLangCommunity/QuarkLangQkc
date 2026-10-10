package lang

// syntax.go re-exports the QuarkLang syntax layer (package qkparser, module
// github.com/QuarkLangCommunity/QuarkLangQkparser) so that the type checker, linter, doc renderer,
// evaluator, the compiler's cgen and the cmd/* tools keep the names they already use.
//
// These are ALIASES, not copies: lang.Expr IS qkparser.Expr. Exactly one parser and one AST exist in
// the build, so nothing can drift, and the syntax layer no longer needs to live in this module (a
// module whose import path is not under QuarkLangQkc/ cannot import our internal packages).
import "github.com/QuarkLangCommunity/QuarkLangQkparser"

// ---- AST and lexer types ----
type (
	// AssignStmt is qkparser.AssignStmt.
	AssignStmt = qkparser.AssignStmt
	// BinOp is qkparser.BinOp.
	BinOp = qkparser.BinOp
	// Block is qkparser.Block.
	Block = qkparser.Block
	// BoolLit is qkparser.BoolLit.
	BoolLit = qkparser.BoolLit
	// BreakStmt is qkparser.BreakStmt.
	BreakStmt = qkparser.BreakStmt
	// CallExpr is qkparser.CallExpr.
	CallExpr = qkparser.CallExpr
	// Comment is qkparser.Comment.
	Comment = qkparser.Comment
	// DeclStmt is qkparser.DeclStmt.
	DeclStmt = qkparser.DeclStmt
	// DeleteStmt is qkparser.DeleteStmt.
	DeleteStmt = qkparser.DeleteStmt
	// Expr is qkparser.Expr.
	Expr = qkparser.Expr
	// ExprStmt is qkparser.ExprStmt.
	ExprStmt = qkparser.ExprStmt
	// FloatLit is qkparser.FloatLit.
	FloatLit = qkparser.FloatLit
	// ForCStmt is qkparser.ForCStmt.
	ForCStmt = qkparser.ForCStmt
	// ForStmt is qkparser.ForStmt.
	ForStmt = qkparser.ForStmt
	// FuncDecl is qkparser.FuncDecl.
	FuncDecl = qkparser.FuncDecl
	// Ident is qkparser.Ident.
	Ident = qkparser.Ident
	// IfStmt is qkparser.IfStmt.
	IfStmt = qkparser.IfStmt
	// ImplDecl is qkparser.ImplDecl.
	ImplDecl = qkparser.ImplDecl
	// IncExpr is qkparser.IncExpr.
	IncExpr = qkparser.IncExpr
	// IndexExpr is qkparser.IndexExpr.
	IndexExpr = qkparser.IndexExpr
	// IntLit is qkparser.IntLit.
	IntLit = qkparser.IntLit
	// InterfaceDecl is qkparser.InterfaceDecl.
	InterfaceDecl = qkparser.InterfaceDecl
	// LexError is qkparser.LexError.
	LexError = qkparser.LexError
	// LibraryDecl is qkparser.LibraryDecl.
	LibraryDecl = qkparser.LibraryDecl
	// ListLit is qkparser.ListLit.
	ListLit = qkparser.ListLit
	// LogStmt is qkparser.LogStmt.
	LogStmt = qkparser.LogStmt
	// MacroDef is qkparser.MacroDef.
	MacroDef = qkparser.MacroDef
	// Member is qkparser.Member.
	Member = qkparser.Member
	// MemberExpr is qkparser.MemberExpr.
	MemberExpr = qkparser.MemberExpr
	// MethodSig is qkparser.MethodSig.
	MethodSig = qkparser.MethodSig
	// NewExpr is qkparser.NewExpr.
	NewExpr = qkparser.NewExpr
	// NullLit is qkparser.NullLit.
	NullLit = qkparser.NullLit
	// Param is qkparser.Param.
	Param = qkparser.Param
	// ParseError is qkparser.ParseError.
	ParseError = qkparser.ParseError
	// Pos is qkparser.Pos.
	Pos = qkparser.Pos
	// Program is qkparser.Program.
	Program = qkparser.Program
	// ReturnStmt is qkparser.ReturnStmt.
	ReturnStmt = qkparser.ReturnStmt
	// ScopeCall is qkparser.ScopeCall.
	ScopeCall = qkparser.ScopeCall
	// SignCall is qkparser.SignCall.
	SignCall = qkparser.SignCall
	// Stmt is qkparser.Stmt.
	Stmt = qkparser.Stmt
	// StrLit is qkparser.StrLit.
	StrLit = qkparser.StrLit
	// StructDecl is qkparser.StructDecl.
	StructDecl = qkparser.StructDecl
	// StructLit is qkparser.StructLit.
	StructLit = qkparser.StructLit
	// StructLitField is qkparser.StructLitField.
	StructLitField = qkparser.StructLitField
	// Token is qkparser.Token.
	Token = qkparser.Token
	// TokenKind is qkparser.TokenKind.
	TokenKind = qkparser.TokenKind
	// TryStmt is qkparser.TryStmt.
	TryStmt = qkparser.TryStmt
	// TypeAlias is qkparser.TypeAlias.
	TypeAlias = qkparser.TypeAlias
	// UnOp is qkparser.UnOp.
	UnOp = qkparser.UnOp
	// WhileStmt is qkparser.WhileStmt.
	WhileStmt = qkparser.WhileStmt
)

// ---- token kinds ----
const (
	TAmper     = qkparser.TAmper
	TAnd       = qkparser.TAnd
	TAssign    = qkparser.TAssign
	TAt        = qkparser.TAt
	TBang      = qkparser.TBang
	TCaret     = qkparser.TCaret
	TCatch     = qkparser.TCatch
	TColon     = qkparser.TColon
	TComma     = qkparser.TComma
	TDot       = qkparser.TDot
	TEOF       = qkparser.TEOF
	TElse      = qkparser.TElse
	TEq        = qkparser.TEq
	TEqStrict  = qkparser.TEqStrict
	TFalse     = qkparser.TFalse
	TFloat     = qkparser.TFloat
	TFor       = qkparser.TFor
	TFunc      = qkparser.TFunc
	TGe        = qkparser.TGe
	TGt        = qkparser.TGt
	TIdent     = qkparser.TIdent
	TIf        = qkparser.TIf
	TImpl      = qkparser.TImpl
	TInc       = qkparser.TInc
	TInt       = qkparser.TInt
	TInterface = qkparser.TInterface
	TLBrace    = qkparser.TLBrace
	TLBracket  = qkparser.TLBracket
	TLParen    = qkparser.TLParen
	TLe        = qkparser.TLe
	TLog       = qkparser.TLog
	TLt        = qkparser.TLt
	TMacro     = qkparser.TMacro
	TMinus     = qkparser.TMinus
	TNe        = qkparser.TNe
	TNull      = qkparser.TNull
	TOr        = qkparser.TOr
	TPercent   = qkparser.TPercent
	TPipe      = qkparser.TPipe
	TPlus      = qkparser.TPlus
	TRBrace    = qkparser.TRBrace
	TRBracket  = qkparser.TRBracket
	TRParen    = qkparser.TRParen
	TReturn    = qkparser.TReturn
	TScope     = qkparser.TScope
	TSemi      = qkparser.TSemi
	TSharp     = qkparser.TSharp
	TShl       = qkparser.TShl
	TShr       = qkparser.TShr
	TSlash     = qkparser.TSlash
	TStar      = qkparser.TStar
	TStr       = qkparser.TStr
	TStruct    = qkparser.TStruct
	TTilde     = qkparser.TTilde
	TTrue      = qkparser.TTrue
	TTry       = qkparser.TTry
	TWhile     = qkparser.TWhile
)

// ---- frontend entry points ----

// Lex tokenizes source without collecting comments (spec §2).
func Lex(src string) ([]Token, error) { return qkparser.Lex(src) }

// LexWithComments tokenizes source and returns its comments in order of appearance.
func LexWithComments(src string) ([]Token, []Comment, error) { return qkparser.LexWithComments(src) }

// Parse builds the AST from tokens (spec §2), refusing a nest deeper than MaxExprDepth (see nesting.go).
func Parse(toks []Token) (*Program, error) {
	if err := refuseOverDeepNesting(toks); err != nil {
		return nil, err
	}
	return qkparser.Parse(toks)
}

// ParseSource does the frontend only — lexing, macro expansion, parsing — with no import resolution and no type checking.
func ParseSource(src string) (*Program, error) {
	if err := refuseOverDeepNestingInSource(src); err != nil {
		return nil, err
	}
	return qkparser.ParseSource(src)
}

// ParseSourceWithComments is ParseSource plus the comments in the source, in order of appearance.
func ParseSourceWithComments(src string) (*Program, []Comment, error) {
	if err := refuseOverDeepNestingInSource(src); err != nil {
		return nil, nil, err
	}
	return qkparser.ParseSourceWithComments(src)
}

// ParseSourceAll is ParseSourceWithComments plus the #macro definitions, which are cut out before parsing and so are absent from the AST.
func ParseSourceAll(src string) (*Program, []Comment, []*MacroDef, error) {
	if err := refuseOverDeepNestingInSource(src); err != nil {
		return nil, nil, nil, err
	}
	return qkparser.ParseSourceAll(src)
}

// SplitMacroDefs splits every #macro definition out of a token stream, returning the macro list and the remaining tokens.
func SplitMacroDefs(toks []Token) ([]*MacroDef, []Token, error) { return qkparser.SplitMacroDefs(toks) }

// ExpandMacros rewrites macro calls in a token stream, with mode selecting the #when operation time.
func ExpandMacros(toks []Token, macros []*MacroDef, mode string) ([]Token, error) {
	return qkparser.ExpandMacros(toks, macros, mode)
}
