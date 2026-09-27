package lang

// ============ qkcheck: static analysis (read-only over the AST; changes no semantics) ============
//
// Design principles:
//   - scope rules are aligned **item by item** with typecheck (if/while/for-C bodies share the outer scope;
//     for-in / catch introduce nested scopes) — otherwise false positives appear;
//   - only report what the compiler does **not** report (syntax/type hard errors belong to typecheck and are surfaced by the CLI);
//   - diagnostics carry line and column so CI and editors can consume them.

import (
	"fmt"
	"path/filepath"
	"quarklang/internal/i18n"
	"sort"
	"strconv"
	"strings"
)

// Diagnostic codes (stable identifiers for CI / editor filtering).
const (
	CodeUnusedVar    = "QK101" // unused local variable (declared but never read)
	CodeUnusedParam  = "QK102" // unused parameter (with -params enabled)
	CodeUnreachable  = "QK103" // unreachable code
	CodeShadow       = "QK104" // shadows an outer variable (the three scopes the compiler does not catch)
	CodeMissingRet   = "QK105" // missing return: a return type is declared but some path produces no value
	CodeIfaceMissing = "QK106" // interface not implemented (near miss: some methods implemented, the rest missing)
	CodeVoidAsValue  = "QK107" // the result of a void function is used as a value
	CodeSelfAssign   = "QK108" // self assignment: x = x
	CodeConstCond    = "QK109" // constant condition / while(true) with no exit
	CodeDivZero      = "QK110" // constant division by zero (outside try)
	CodeUnusedImport = "QK111" // unused import
	CodeShadowGlobal = "QK112" // a local variable shadows a global symbol (function/type/space/macro)
	CodeDeadStore    = "QK113" // dead store: an assignment overwritten by a later assignment with no read in between
	CodeSelfCompare  = "QK114" // self comparison: x == x / x != x
	CodeUnusedFunc   = "QK115" // uncalled function (program main only; functions in library files are API)
)

// Diag is one static-analysis diagnostic.
type Diag struct {
	Pos  Pos
	Code string
	Sev  string // "warning"
	Msg  string
}

func (d Diag) String() string {
	return fmt.Sprintf("%d:%d: %s: %s [%s]", d.Pos.Line, d.Pos.Col, d.Sev, d.Msg, d.Code)
}

// LintOptions controls the optional checks.
type LintOptions struct {
	// Params also checks unused parameters. Off by default: interface implementations often ignore parameters.
	Params bool
	// File is the current file path (used for import resolution and position-sensitive checks; may be empty).
	File string
	// LibDirs are extra import search directories (same as qkcheck -L; empty = the same directory only).
	LibDirs []string
}

// Lint statically checks a parsed program and returns diagnostics sorted by position.
func Lint(prog *Program, opts LintOptions) []Diag {
	l := &linter{
		opts:      opts,
		funcs:     map[string][]*FuncDecl{},
		spaces:    map[string]map[string]*FuncDecl{},
		globals:   map[string]string{},
		pending:   map[string]Pos{},
		usesNames: map[string]bool{},
		refVars:   map[string]bool{},
	}
	for _, f := range prog.Funcs {
		l.funcs[f.Name] = append(l.funcs[f.Name], f)
	}
	for _, f := range prog.Funcs {
		l.globals[f.Name] = "fn"
	}
	for _, s := range prog.Structs {
		if s.Name != "" && !strings.HasPrefix(s.Name, "__anon_") {
			l.globals[s.Name] = "type"
		}
	}
	for _, i := range prog.Interfaces {
		if i.Name != "" && !strings.HasPrefix(i.Name, "__anon_") {
			l.globals[i.Name] = "type"
		}
	}
	for _, ta := range prog.TypeAliases {
		l.globals[ta.Name] = "alias"
	}
	for _, lb := range prog.Libraries {
		l.globals[lb.Name] = "library"
	}
	for _, im := range prog.Impls {
		if l.spaces[im.Type] == nil {
			l.spaces[im.Type] = map[string]*FuncDecl{}
		}
		for _, m := range im.Methods {
			l.spaces[im.Type][m.Name] = m
		}
		if _, isStruct := l.globals[im.Type]; !isStruct {
			l.globals[im.Type] = "space" // impl target not declared as a struct → treated as a space
		}
	}

	for _, f := range prog.Funcs {
		l.checkFunc(f)
	}
	for _, im := range prog.Impls {
		for _, m := range im.Methods {
			l.checkMethod(m, im.Type)
		}
	}
	l.checkInterfaces(prog)
	l.checkLibFuncs(prog)
	l.collectUses(prog)
	l.checkImports(prog)
	l.checkUnusedFuncs(prog)

	sort.SliceStable(l.diags, func(i, j int) bool {
		a, b := l.diags[i], l.diags[j]
		if a.Pos.Line != b.Pos.Line {
			return a.Pos.Line < b.Pos.Line
		}
		if a.Pos.Col != b.Pos.Col {
			return a.Pos.Col < b.Pos.Col
		}
		return a.Code < b.Code
	})
	return l.diags
}

// LintSource is a convenience entry point: parse the source and lint it (a parse failure returns the error plus whatever diagnostics were found).
func LintSource(src string, opts LintOptions) ([]Diag, error) {
	prog, err := ParseSource(src)
	if err != nil {
		return nil, err
	}
	return Lint(prog, opts), nil
}

// ---------- Scope model (aligned with typecheck) ----------

type lvar struct {
	name     string
	pos      Pos
	kind     string // local | param | forvar | catch
	typeName string // declared type string (used for reference-type detection)
	reads    int
	writes   int
	outer    *lvar // points at the outer variable of the same name when shadowing
}

type lscope struct {
	outer *lscope
	vars  map[string]*lvar
}

type linter struct {
	opts   LintOptions
	diags  []Diag
	scope  *lscope
	fnVars []*lvar

	funcs  map[string][]*FuncDecl
	spaces map[string]map[string]*FuncDecl

	curFn *FuncDecl

	// State for the newly added checks
	globals   map[string]string // global symbol name → kind (fn/type/space/library/macro/alias)
	pending   map[string]Pos    // variable → the most recent assignment position not yet read (QK113)
	tryDepth  int               // try block depth (QK110 exemption: a division by zero inside try is deliberate error handling)
	inFn      bool              // whether we are inside a function body (QK112 reports shadowing of a global only inside a function)
	usesNames map[string]bool   // identifiers that appear in this file (QK111 import-usage decision)
	loopDepth int               // loop body depth (QK113 exemption: an assignment in a loop body is read by the next iteration)
	refVars   map[string]bool   // reference-type variables (T& / pointer T: `p = v` writes through, it is not a rebinding)
}

func (l *linter) warn(pos Pos, code, format string, args ...interface{}) {
	l.diags = append(l.diags, Diag{Pos: pos, Code: code, Sev: "warning", Msg: i18n.T(format, args...)})
}

func (l *linter) push() {
	l.scope = &lscope{outer: l.scope, vars: map[string]*lvar{}}
}

func (l *linter) pop() {
	if l.scope != nil {
		l.scope = l.scope.outer
	}
}

func (l *linter) lookup(name string) *lvar {
	for s := l.scope; s != nil; s = s.outer {
		if v, ok := s.vars[name]; ok {
			return v
		}
	}
	return nil
}

func (l *linter) lookupOuter(name string) *lvar {
	if l.scope == nil {
		return nil
	}
	for s := l.scope.outer; s != nil; s = s.outer {
		if v, ok := s.vars[name]; ok {
			return v
		}
	}
	return nil
}

// declare registers a declaration. An outer variable with the same name in a nested scope is recorded as shadowing.
func (l *linter) declare(name string, pos Pos, kind string, typeName ...string) *lvar {
	if name == "_" || strings.HasPrefix(name, "_") {
		return nil // _ prefix = explicitly ignored
	}
	v := &lvar{name: name, pos: pos, kind: kind}
	if len(typeName) > 0 {
		v.typeName = typeName[0]
	}
	if _, dup := l.scope.vars[name]; dup { // already declared in the current scope: typecheck already reports duplicate, do not report it again
		return nil
	}
	if outer := l.lookupOuter(name); outer != nil {
		v.outer = outer
		l.warn(pos, CodeShadow, "%s %s 遮蔽外层同名变量（外层声明于第 %d 行）", lintKindName(kind), name, outer.pos.Line)
	}
	// QK112: a declaration inside a function shadows a global symbol (function/type/space/library/macro) — call sites would suddenly resolve to the local
	if isRefType(declTypeOf(l, name)) {
		l.refVars[name] = true
	}
	if l.inFn {
		if cat, isGlobal := l.globals[name]; isGlobal && cat == "fn" {
			l.warn(pos, CodeShadowGlobal, "%s %s 遮蔽全局函数（同名调用会被解析为该局部变量）", lintKindName(kind), name)
		}
	}
	l.scope.vars[name] = v
	l.fnVars = append(l.fnVars, v)
	return v
}

func lintKindName(kind string) string {
	switch kind {
	case "param":
		return i18n.T("形参")
	case "forvar":
		return i18n.T("循环变量")
	case "catch":
		return i18n.T("catch 变量")
	default:
		return i18n.T("变量")
	}
}

// use records a read; it clears the "assigned but unread" record (QK113 only counts never-read assignments as dead stores).
func (l *linter) use(name string) {
	if v := l.lookup(name); v != nil {
		v.reads++
	}
	delete(l.pending, name)
}

// lintGlobalName is the display name of a global symbol category.
func lintGlobalName(cat string) string {
	switch cat {
	case "fn":
		return "函数"
	case "type":
		return "类型"
	case "space":
		return "空间"
	case "library":
		return "系统库"
	case "macro":
		return "宏"
	case "alias":
		return "类型别名"
	}
	return "符号"
}

// write records an assignment.
func (l *linter) write(name string) {
	if v := l.lookup(name); v != nil {
		v.writes++
	}
}

// ---------- Function checks ----------

func (l *linter) checkFunc(f *FuncDecl) {
	l.checkBody(f, f.Body, "")
}

// checkMethod checks methods in an impl / space; implType is the owning type (used to resolve void calls).
func (l *linter) checkMethod(f *FuncDecl, implType string) {
	l.checkBody(f, f.Body, implType)
}

// checkLibFuncs: library bindings (library declarations) only have signatures and no body, so nothing is checked here.
func (l *linter) checkLibFuncs(prog *Program) {}

func (l *linter) checkBody(f *FuncDecl, body *Block, implType string) {
	if body == nil {
		return
	}
	prevFn, prevVars, prevScope := l.curFn, l.fnVars, l.scope
	prevInFn, prevPending := l.inFn, l.pending
	l.curFn, l.fnVars, l.scope = f, nil, nil
	l.inFn, l.pending = true, map[string]Pos{}
	l.push()
	for _, p := range f.Params {
		l.declare(p.Name, p.Pos, "param", p.Type)
	}
	term, allRet := l.block(body)
	// Unused variables/parameters (reported per function)
	for _, v := range l.fnVars {
		if v.reads > 0 {
			continue
		}
		if v.kind == "param" && !l.opts.Params {
			continue
		}
		if v.kind == "catch" {
			continue // catch variables are not checked by default: ignoring the error object is idiomatic
		}
		if v.writes > 0 {
			l.warn(v.pos, codeForKind(v.kind), "%s %s 只被赋值、从未读取", lintKindName(v.kind), v.name)
		} else {
			l.warn(v.pos, codeForKind(v.kind), "%s %s 声明后从未使用", lintKindName(v.kind), v.name)
		}
	}
	// Missing return: a non-void return type is declared but some path produces no value (yields nil at runtime)
	if ret := strings.TrimSpace(f.Ret); ret != "" && ret != "void" && f.Name != "main" {
		switch {
		case !term:
			l.warn(f.Pos, CodeMissingRet, "函数 %s 声明返回类型 %s，但存在执行到函数末尾的路径（运行期得到 nil）", f.Name, ret)
		case !allRet:
			l.warn(f.Pos, CodeMissingRet, "函数 %s 声明返回类型 %s，但终止路径以 log 结束、不产生返回值（运行期得到 nil）", f.Name, ret)
		}
	}
	l.curFn, l.fnVars, l.scope = prevFn, prevVars, prevScope
	l.inFn, l.pending = prevInFn, prevPending
}

func codeForKind(kind string) string {
	if kind == "param" {
		return CodeUnusedParam
	}
	return CodeUnusedVar
}

// ---------- Statement walk ----------

// block walks a statement block and returns (does it terminate, do all terminators return).
// Terminators = return / log / break; only the first unreachable statement is reported.
func (l *linter) block(b *Block) (term bool, allRet bool) {
	if b == nil {
		return false, false
	}
	dead := false
	for i, st := range b.Stmts {
		if dead {
			l.warn(posOfStmt(st), CodeUnreachable, "不可达代码（第 %d 行之后的语句不会执行）", posOfStmt(b.Stmts[i-1]).Line)
			break
		}
		t, r := l.stmt(st)
		if t {
			term, allRet = t, r
			dead = true
		}
	}
	if dead {
		return term, allRet
	}
	return false, false
}

func posOfStmt(s Stmt) Pos {
	switch t := s.(type) {
	case *ExprStmt:
		return posOf(t.X)
	case *LogStmt:
		return t.Pos
	case *DeleteStmt:
		return t.Pos
	case *BreakStmt:
		return t.Pos
	case *TryStmt:
		return t.Pos
	case *ReturnStmt:
		return t.Pos
	case *IfStmt:
		return posOf(t.Cond)
	case *WhileStmt:
		return posOf(t.Cond)
	case *ForStmt:
		return t.Pos
	case *ForCStmt:
		return t.Pos
	case *DeclStmt:
		return t.Pos
	case *AssignStmt:
		return t.Pos
	}
	return Pos{}
}

// stmt walks one statement and reports whether it terminates (return/log/break) and whether it ends with return.
func (l *linter) stmt(s Stmt) (term bool, allRet bool) {
	switch t := s.(type) {
	case *ExprStmt:
		l.expr(t.X, false, "")
		return false, false
	case *LogStmt:
		l.expr(t.X, true, "")
		return true, false // log = record and terminate, but produce no return value
	case *DeleteStmt:
		l.expr(t.X, true, "") // delete x counts as a use of x (a lifetime operation)
		return false, false
	case *BreakStmt:
		return true, false
	case *ReturnStmt:
		if t.X != nil {
			l.expr(t.X, true, "")
		}
		return true, true
	case *DeclStmt:
		if t.Init != nil {
			l.expr(t.Init, true, "")
		}
		l.declare(t.Name, t.Pos, "local", t.Type) // evaluate the initial value before registering (consistent with typecheck)
		if isRefType(t.Type) {
			l.refVars[t.Name] = true // reference type: `p = v` writes through to the pointer target, not a rebinding
		}
		if t.Init != nil && l.loopDepth == 0 && !l.refVars[t.Name] {
			l.pending[t.Name] = t.Pos // QK113: if the initial value is overwritten by a later assignment with no read in between → dead store
		}
		return false, false
	case *AssignStmt:
		// QK108: self assignment x = x (including the structurally equivalent p.x / l[i] forms)
		if sameExpr(t.Target, t.X) {
			l.warn(t.Pos, CodeSelfAssign, "自赋值：%s 赋值给自身（无效果）", exprText(t.Target))
		}
		// Evaluate the right-hand side and the target subexpressions first: a read in `x = x + 1` clears the "assigned but unread" record,
		// otherwise reading a variable in terms of itself would be misjudged as a dead store (a QK113 false-positive source).
		l.expr(t.X, true, "")
		switch tg := t.Target.(type) {
		case *Ident:
			// QK113: the previous assignment to the same variable (including its declaration initializer) was never read → dead store.
			// Two exemptions (prefer a missed report over a false one): write-through of reference types; assignments inside loops are read on the next round.
			if prev, ok := l.pending[tg.Name]; ok && !l.refVars[tg.Name] && l.loopDepth == 0 {
				l.warn(prev, CodeDeadStore, "对 %s 的赋值被第 %d 行的赋值覆盖（其间未读取）", tg.Name, t.Pos.Line)
			}
			l.write(tg.Name)
			if l.loopDepth == 0 && !l.refVars[tg.Name] {
				l.pending[tg.Name] = t.Pos
			} else {
				delete(l.pending, tg.Name)
			}
		case *MemberExpr:
			l.expr(tg.X, true, "")
		case *IndexExpr:
			l.expr(tg.X, true, "")
			l.expr(tg.Idx, true, "")
		default:
			l.expr(t.Target, true, "")
		}
		return false, false
	case *IfStmt:
		l.expr(t.Cond, true, "")
		l.checkConstCond(t.Cond, posOf(t.Cond), "if")
		l.branch(t.Then)
		t1, r1 := l.block(t.Then)
		l.branch(t.Else)
		t2, r2 := false, false
		if t.Else != nil {
			t2, r2 = l.block(t.Else)
		}
		// Both branches must terminate for the statement to terminate; without an else the else branch implicitly does not terminate
		return t1 && t2, r1 && r2
	case *WhileStmt:
		l.expr(t.Cond, true, "")
		l.checkConstCond(t.Cond, posOf(t.Cond), "while")
		if bl, ok := t.Cond.(*BoolLit); ok && bl.V && !hasBreak(t.Body) && !hasReturn(t.Body) {
			l.warn(posOf(t.Cond), CodeConstCond, "while (true) 且循环体内无 break/return：可能的死循环")
		}
		l.branch(t.Body)
		l.loopDepth++
		l.block(t.Body)
		l.loopDepth--
		return false, false // the loop may execute zero times
	case *ForStmt:
		l.expr(t.Iter, true, "")
		l.push()
		l.declare(t.Var, t.Pos, "forvar", t.Type)
		l.branch(t.Body)
		l.loopDepth++
		l.block(t.Body)
		l.loopDepth--
		l.pop()
		return false, false
	case *ForCStmt:
		l.push()
		if t.Init != nil {
			l.stmt(t.Init)
		}
		if t.Cond != nil {
			l.expr(t.Cond, true, "")
		}
		if t.Step != nil {
			l.stmt(t.Step)
		}
		l.branch(t.Body)
		l.loopDepth++
		l.block(t.Body)
		l.loopDepth--
		l.pop()
		return false, false
	case *TryStmt:
		l.tryDepth++
		t1, r1 := l.block(t.Try)
		l.tryDepth--
		l.branch(t.Catch)
		l.push()
		l.declare(t.CatchVar, t.Pos, "catch", t.CatchVarType)
		t2, r2 := l.block(t.Catch)
		l.pop()
		if t.Catch == nil {
			return false, false
		}
		return t1 && t2, r1 && r2
	}
	return false, false
}

// ---------- Expression walk ----------

// expr walks an expression. valueCtx=true means the expression's value is used (for the void-misuse check).
func (l *linter) expr(e Expr, valueCtx bool, implType string) {
	switch t := e.(type) {
	case nil:
		return
	case *Ident:
		l.use(t.Name)
	case *IntLit, *FloatLit, *StrLit, *BoolLit, *NullLit:
	case *ListLit:
		for _, it := range t.Items {
			l.expr(it, true, implType)
		}
	case *StructLit:
		for _, f := range t.Fields {
			l.expr(f.X, true, implType)
		}
	case *NewExpr:
		if t.Size != nil {
			l.expr(t.Size, true, implType)
		}
	case *BinOp:
		l.expr(t.L, true, implType)
		l.expr(t.R, true, implType)
		// QK110: division/modulo by a constant zero (exempt inside try: that is deliberate error handling)
		if (t.Op == "/" || t.Op == "%") && l.tryDepth == 0 {
			if n, ok := t.R.(*IntLit); ok && n.V == 0 {
				l.warn(t.Pos, CodeDivZero, "常量除零：%s 0（运行期报错；若为刻意错误处理请放进 try 块）", t.Op)
			}
		}
		// QK114: self comparison x == x / x != x (only for addressable forms: identifier/member/index)
		if (t.Op == "==" || t.Op == "!=") && isRefLike(t.L) && sameExpr(t.L, t.R) {
			name := exprText(t.L)
			if t.Op == "==" {
				l.warn(t.Pos, CodeSelfCompare, "自身比较：%s == %s 恒为真", name, name)
			} else {
				l.warn(t.Pos, CodeSelfCompare, "自身比较：%s != %s 恒为假", name, name)
			}
		}
	case *UnOp:
		l.expr(t.X, true, implType)
	case *MemberExpr:
		l.expr(t.X, true, implType)
	case *IndexExpr:
		l.expr(t.X, true, implType)
		l.expr(t.Idx, true, implType)
	case *ScopeCall:
		for _, a := range t.Args {
			l.expr(a, true, implType)
		}
		if valueCtx {
			if fn := l.spaceFunc(t.Scope, t.Name); fn != nil && isVoidRet(fn.Ret) && !l.overloaded(t.Scope, t.Name) {
				l.warn(t.Pos, CodeVoidAsValue, "%s::%s 返回 void，其调用结果被当作值使用（运行期为 nil）", t.Scope, t.Name)
			}
		}
	case *CallExpr:
		for _, a := range t.Args {
			l.expr(a, true, implType)
		}
		if t.Sign != nil { // signature call f(args) @mb(args): both the signature instance name and the extra arguments count as uses
			l.use(t.Sign.Name)
			for _, a := range t.Sign.Args {
				l.expr(a, true, implType)
			}
		}
		l.expr(t.Fn, false, implType)
		if valueCtx {
			if id, ok := t.Fn.(*Ident); ok {
				if fn := l.singleFunc(id.Name); fn != nil && isVoidRet(fn.Ret) {
					l.warn(t.Pos, CodeVoidAsValue, "函数 %s 返回 void，其调用结果被当作值使用（运行期为 nil）", id.Name)
				}
			}
		}
	}
}

func isVoidRet(ret string) bool { return strings.TrimSpace(ret) == "void" }

// singleFunc returns the unique function with that name (no overloads); nil when overloads make the return type statically undecidable.
func (l *linter) singleFunc(name string) *FuncDecl {
	defs := l.funcs[name]
	if len(defs) != 1 {
		return nil
	}
	return defs[0]
}

func (l *linter) overloaded(space, name string) bool {
	if space == "" {
		return len(l.funcs[name]) != 1
	}
	n := 0
	for _, m := range l.spaces {
		if _, ok := m[name]; ok {
			n++
		}
	}
	return n != 1
}

func (l *linter) spaceFunc(space, name string) *FuncDecl {
	if m, ok := l.spaces[space]; ok {
		return m[name]
	}
	return nil
}

// ---------- Interface near-miss (interface not implemented) ----------

// checkInterfaces finds, for each interface, the concrete types that implement **some** methods and reports the missing ones.
// Structural satisfaction is reported by typecheck at the assignment site; this covers the "meant to implement but missed a method" case.
func (l *linter) checkInterfaces(prog *Program) {
	if len(prog.Interfaces) == 0 {
		return
	}
	structs := map[string]bool{}
	for _, s := range prog.Structs {
		structs[s.Name] = true
	}
	// The type's instance-method set (methods in impl whose first parameter is this type/Self)
	methodsOf := map[string]map[string]bool{}
	typePos := map[string]Pos{}
	for _, im := range prog.Impls {
		if !structs[im.Type] {
			continue // infer only for declared structs, to avoid false positives on space / external types
		}
		if methodsOf[im.Type] == nil {
			methodsOf[im.Type] = map[string]bool{}
			typePos[im.Type] = im.Pos
		}
		for _, m := range im.Methods {
			if len(m.Params) > 0 && isRecvParam(im.Type, &m.Params[0]) {
				methodsOf[im.Type][m.Name] = true
			}
		}
	}
	for _, iface := range prog.Interfaces {
		want := map[string]MethodSig{}
		l.collectIfaceMethods(prog, iface, want, map[string]bool{})
		if len(want) == 0 {
			continue
		}
		for typ, have := range methodsOf {
			var missing, present []string
			for name := range want {
				if have[name] {
					present = append(present, name)
				} else {
					missing = append(missing, name)
				}
			}
			if len(present) == 0 || len(missing) == 0 {
				continue
			}
			sort.Strings(missing)
			sort.Strings(present)
			l.warn(typePos[typ], CodeIfaceMissing,
				"类型 %s 疑似实现接口 %s，但缺少方法 %s（已实现 %s）",
				typ, iface.Name, strings.Join(missing, ", "), strings.Join(present, ", "))
		}
	}
}

func (l *linter) collectIfaceMethods(prog *Program, iface *InterfaceDecl, out map[string]MethodSig, seen map[string]bool) {
	if iface == nil || seen[iface.Name] {
		return
	}
	seen[iface.Name] = true
	for _, m := range iface.Methods {
		if _, ok := out[m.Name]; !ok {
			out[m.Name] = m
		}
	}
	for _, ex := range iface.Expands {
		for _, other := range prog.Interfaces {
			if other.Name == ex {
				l.collectIfaceMethods(prog, other, out, seen)
			}
		}
	}
}

// ---------- Helpers: expression structural keys / exit tests / branch clearing / import checks ----------

// isRefType reports whether a type string is a reference type: T& or pointer … (write-through semantics, not rebinding).
func isRefType(t string) bool {
	t = strings.TrimSpace(t)
	if t == "" {
		return false
	}
	if strings.HasSuffix(t, "&") || strings.HasPrefix(t, "pointer") {
		return true
	}
	return false
}

// declTypeOf returns a variable's declared type (helpers for QK112/QK113; "" when unregistered).
func declTypeOf(l *linter, name string) string {
	if v := l.lookup(name); v != nil {
		return v.typeName
	}
	return ""
}

// sameExpr: structural equality over simple forms only (identifier / member / index / literal);
// everything else (calls, operations, …) counts as unequal → do not report; prefer a miss over a false positive.
// All comparisons are pointer/value based and allocation-free (the old implementation built keys with fmt.Sprintf, the main allocation source in linting).
func sameExpr(a, b Expr) bool {
	switch x := a.(type) {
	case *Ident:
		y, ok := b.(*Ident)
		return ok && x.Name == y.Name
	case *MemberExpr:
		y, ok := b.(*MemberExpr)
		return ok && x.Name == y.Name && sameExpr(x.X, y.X)
	case *IndexExpr:
		y, ok := b.(*IndexExpr)
		return ok && sameExpr(x.X, y.X) && sameExpr(x.Idx, y.Idx)
	case *IntLit:
		y, ok := b.(*IntLit)
		return ok && x.V == y.V
	case *FloatLit:
		y, ok := b.(*FloatLit)
		return ok && x.V == y.V
	case *StrLit:
		y, ok := b.(*StrLit)
		return ok && x.V == y.V
	case *BoolLit:
		y, ok := b.(*BoolLit)
		return ok && x.V == y.V
	case *NullLit:
		_, ok := b.(*NullLit)
		return ok
	}
	return false
}

// isRefLike reports whether a form is addressable (self assignment/self comparison are only reported for those).
func isRefLike(e Expr) bool {
	switch e.(type) {
	case *Ident, *MemberExpr, *IndexExpr:
		return true
	}
	return false
}

// exprText renders the expression text shown in a diagnostic (only called when actually reporting → allocation on the slow path is fine).
func exprText(e Expr) string {
	var b strings.Builder
	writeExprText(&b, e)
	return b.String()
}

func writeExprText(b *strings.Builder, e Expr) {
	switch t := e.(type) {
	case *Ident:
		b.WriteString(t.Name)
	case *MemberExpr:
		writeExprText(b, t.X)
		b.WriteByte('.')
		b.WriteString(t.Name)
	case *IndexExpr:
		writeExprText(b, t.X)
		b.WriteByte('[')
		writeExprText(b, t.Idx)
		b.WriteByte(']')
	case *IntLit:
		b.WriteString(strconv.FormatInt(t.V, 10))
	case *StrLit:
		b.WriteString(strconv.Quote(t.V))
	case *BoolLit:
		b.WriteString(strconv.FormatBool(t.V))
	default:
		b.WriteString(i18n.T("表达式"))
	}
}

// checkConstCond: constant conditions such as if (true/false), while (false).
// while (true) is judged separately at the call site (not reported when a break/return exit exists).
func (l *linter) checkConstCond(cond Expr, pos Pos, kw string) {
	bl, ok := cond.(*BoolLit)
	if !ok {
		return
	}
	if kw == "while" && bl.V {
		return // left to the while(true) exit analysis
	}
	if bl.V {
		l.warn(pos, CodeConstCond, "%s (true)：条件恒为真，分支永远执行", kw)
	} else {
		l.warn(pos, CodeConstCond, "%s (false)：条件恒为假，分支永不执行", kw)
	}
}

// branch clears the unread-assignment records before entering a nested block or branch:
// a read may happen inside, so clearing conservatively avoids false positives (QK113).
func (l *linter) branch(b *Block) {
	if b == nil || len(l.pending) == 0 {
		return // already an empty map: nothing to clear (the common case, saves the clear cost)
	}
	clear(l.pending) // reuse the same map instead of allocating a new one per branch
}

// hasBreak reports whether a block contains a break (it does not descend into nested loops: a break inside them does not escape).
func hasBreak(b *Block) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		switch t := st.(type) {
		case *BreakStmt:
			return true
		case *IfStmt:
			if hasBreak(t.Then) || hasBreak(t.Else) {
				return true
			}
		case *TryStmt:
			if hasBreak(t.Try) || hasBreak(t.Catch) {
				return true
			}
		}
	}
	return false
}

// hasReturn reports whether a block contains return / log (function-level exits).
func hasReturn(b *Block) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		switch t := st.(type) {
		case *ReturnStmt, *LogStmt:
			return true
		case *IfStmt:
			if hasReturn(t.Then) || hasReturn(t.Else) {
				return true
			}
		case *TryStmt:
			if hasReturn(t.Try) || hasReturn(t.Catch) {
				return true
			}
		case *WhileStmt:
			if hasReturn(t.Body) {
				return true
			}
		case *ForStmt:
			if hasReturn(t.Body) {
				return true
			}
		case *ForCStmt:
			if hasReturn(t.Body) {
				return true
			}
		}
	}
	return false
}

// collectUses collects identifier and space names that appear in this file (used by QK111 to decide whether an import is used).
func (l *linter) collectUses(prog *Program) {
	var walkBlock func(b *Block)
	var walkExpr func(e Expr)
	walkExpr = func(e Expr) {
		switch t := e.(type) {
		case nil:
		case *Ident:
			l.usesNames[t.Name] = true
		case *ListLit:
			for _, it := range t.Items {
				walkExpr(it)
			}
		case *StructLit:
			for _, f := range t.Fields {
				walkExpr(f.X)
			}
		case *NewExpr:
			l.collectTypeNames(t.Typ)
			walkExpr(t.Size)
		case *BinOp:
			walkExpr(t.L)
			walkExpr(t.R)
		case *UnOp:
			walkExpr(t.X)
		case *MemberExpr:
			l.usesNames[t.Name] = true
			walkExpr(t.X)
		case *IndexExpr:
			walkExpr(t.X)
			walkExpr(t.Idx)
		case *ScopeCall:
			l.usesNames[t.Scope] = true
			l.usesNames[t.Name] = true
			l.usesNames[t.Scope+"::"+t.Name] = true
			for _, a := range t.Args {
				walkExpr(a)
			}
		case *CallExpr:
			walkExpr(t.Fn)
			for _, a := range t.Args {
				walkExpr(a)
			}
			if t.Sign != nil {
				for _, a := range t.Sign.Args {
					walkExpr(a)
				}
			}
		}
	}
	var walkStmt func(st Stmt)
	walkBlock = func(b *Block) {
		if b == nil {
			return
		}
		for _, st := range b.Stmts {
			walkStmt(st)
		}
	}
	walkStmt = func(st Stmt) {
		switch t := st.(type) {
		case *ExprStmt:
			walkExpr(t.X)
		case *LogStmt:
			walkExpr(t.X)
		case *DeleteStmt:
			walkExpr(t.X)
		case *ReturnStmt:
			walkExpr(t.X)
		case *DeclStmt:
			l.collectTypeNames(t.Type)
			walkExpr(t.Init)
		case *AssignStmt:
			walkExpr(t.Target)
			walkExpr(t.X)
		case *IfStmt:
			walkExpr(t.Cond)
			walkBlock(t.Then)
			walkBlock(t.Else)
		case *WhileStmt:
			walkExpr(t.Cond)
			walkBlock(t.Body)
		case *ForStmt:
			l.collectTypeNames(t.Type)
			walkExpr(t.Iter)
			walkBlock(t.Body)
		case *ForCStmt:
			walkStmt(t.Init)
			walkExpr(t.Cond)
			walkStmt(t.Step)
			walkBlock(t.Body)
		case *TryStmt:
			l.collectTypeNames(t.CatchVarType)
			walkBlock(t.Try)
			walkBlock(t.Catch)
		}
	}
	for _, f := range prog.Funcs {
		l.collectTypeNames(f.Ret)
		for _, p := range f.Params {
			l.collectTypeNames(p.Type)
		}
		walkBlock(f.Body)
	}
	for _, im := range prog.Impls {
		for _, m := range im.Methods {
			l.collectTypeNames(m.Ret)
			for _, p := range m.Params {
				l.collectTypeNames(p.Type)
			}
			walkBlock(m.Body)
		}
	}
	for _, sd := range prog.Structs {
		for _, mem := range sd.Members {
			l.collectTypeNames(mem.Type)
		}
	}
	for _, id := range prog.Interfaces {
		for _, m := range id.Methods {
			l.collectTypeNames(m.Ret)
			for _, p := range m.Params {
				l.collectTypeNames(p.Type)
			}
		}
	}
	for _, ta := range prog.TypeAliases {
		l.collectTypeNames(ta.Type)
	}
}

// collectTypeNames marks identifiers inside type strings as used: a type annotation like `LibPoint p` is not an Ident,
// and importing only a library's types (or a library type inside generic arguments) still counts as using that import.
func (l *linter) collectTypeNames(t string) {
	if t == "" {
		return
	}
	start := -1
	for i, r := range t {
		isName := r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') || r > 0x7f
		if isName {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			l.usesNames[t[start:i]] = true
			start = -1
		}
	}
	if start >= 0 {
		l.usesNames[t[start:]] = true
	}
}

// checkImports: unused imports (QK111):
// load the imported file → collect its public symbols (pub functions/types + all macros + space/library names),
// and if none of those names ever appears in this file → report "unused import".
// Skip when the library cannot be resolved or its symbol set is empty (.qlib etc.) — prefer a miss over a false positive.
func (l *linter) checkImports(prog *Program) {
	if len(prog.Imports) == 0 {
		return
	}
	dirs := []string{"."}
	if l.opts.File != "" {
		dirs = []string{filepath.Dir(l.opts.File)}
	}
	dirs = append(dirs, l.opts.LibDirs...)
	for i, imp := range prog.Imports {
		src, _, err := LoadImportIn(dirs, imp)
		if err != nil {
			continue
		}
		libProg, _, libMacros, err := ParseSourceAll(src)
		if err != nil {
			continue
		}
		syms := libPublicSymbols(libProg, libMacros)
		if len(syms) == 0 {
			continue
		}
		used := false
		for _, name := range syms {
			if l.usesNames[name] {
				used = true
				break
			}
		}
		if used {
			continue
		}
		pos := Pos{}
		if i < len(prog.ImportPos) {
			pos = prog.ImportPos[i]
		}
		l.warn(pos, CodeUnusedImport, "导入了 %q 但未使用其任何符号（可删除该 import）", imp)
	}
}

// checkUnusedFuncs: functions that are never called:
// only reported for program main (functions in a library file are the public API), and only when the name never appears in the whole file.
func (l *linter) checkUnusedFuncs(prog *Program) {
	// Only reported for a program **with a main** (dead-code judgement only holds for an executable program):
	//   - program library; → a library, its functions are the API, not reported;
	//   - a file without main (such as compiler/testdata/mathlib.qk, a library missing `program library;`) → not enough evidence, not reported;
	//   - program main; is treated as executable even when pub appears (pub has no effect in a main program) → reported.
	// The rule is pinned by the statistical benchmark (lintstat_test.go) together with the corpus snapshot.
	if prog.Kind == "library" || !hasMainFunc(prog) {
		return
	}
	for _, f := range prog.Funcs {
		if f.Name == "main" || l.usesNames[f.Name] {
			continue
		}
		l.warn(f.Pos, CodeUnusedFunc, "函数 %s 从未被调用（死代码；库文件中的函数不报）", f.Name)
	}
}

// hasMainFunc reports whether the program defines main.
func hasMainFunc(prog *Program) bool {
	for _, f := range prog.Funcs {
		if f.Name == "main" {
			return true
		}
	}
	return false
}

// libPublicSymbols collects a library's publicly visible symbol names (pub functions/types, spaces and their methods, FFI libraries, macros).
func libPublicSymbols(lib *Program, macros []*MacroDef) []string {
	var out []string
	for _, m := range macros {
		out = append(out, m.Name)
	}
	pub := map[string]bool{}
	for _, p := range lib.Pub {
		pub[p] = true
	}
	for _, f := range lib.Funcs {
		if pub[f.Name] {
			out = append(out, f.Name)
		}
	}
	for _, s := range lib.Structs {
		if pub[s.Name] {
			out = append(out, s.Name)
		}
	}
	for _, i := range lib.Interfaces {
		if pub[i.Name] {
			out = append(out, i.Name)
		}
	}
	for _, im := range lib.Impls {
		out = append(out, im.Type)
		for _, m := range im.Methods {
			out = append(out, m.Name)
		}
	}
	for _, lb := range lib.Libraries {
		out = append(out, lb.Name)
	}
	return out
}
