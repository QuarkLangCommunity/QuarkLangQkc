package lang

// ============ qkcheck：静态检查（只读 AST，不改变任何语义） ============
//
// 设计原则：
//   - 与 typecheck 的作用域规则**逐条对齐**（if/while/for-C 体共用外层作用域；
//     for-in / catch 引入嵌套作用域），否则会产生误报；
//   - 只报编译器**不报**的问题（语法/类型硬错误由 typecheck 负责，CLI 另行呈现）；
//   - 诊断带行号列号，可被 CI 与编辑器消费。

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// 诊断码（稳定标识，供 CI / 编辑器过滤）。
const (
	CodeUnusedVar    = "QK101" // 未使用局部变量（声明后从未读取）
	CodeUnusedParam  = "QK102" // 未使用形参（-params 开启）
	CodeUnreachable  = "QK103" // 不可达代码
	CodeShadow       = "QK104" // 遮蔽外层变量（编译器未拦截的三种作用域）
	CodeMissingRet   = "QK105" // 缺返回：声明了返回类型，但存在不产生返回值的路径
	CodeIfaceMissing = "QK106" // 接口未实现（近失配：实现了部分方法，缺其余）
	CodeVoidAsValue  = "QK107" // void 函数的返回值被当作值使用
	CodeSelfAssign   = "QK108" // 自赋值：x = x
	CodeConstCond    = "QK109" // 常量条件 / while(true) 无出口
	CodeDivZero      = "QK110" // 常量除零（try 之外）
	CodeUnusedImport = "QK111" // 未使用的 import
	CodeShadowGlobal = "QK112" // 局部变量遮蔽全局符号（函数/类型/空间/宏）
	CodeDeadStore    = "QK113" // 死存储：赋值被后续赋值覆盖且其间未读取
	CodeSelfCompare  = "QK114" // 自身比较：x == x / x != x
	CodeUnusedFunc   = "QK115" // 未被调用的函数（仅 program main；库文件里的函数是 API）
)

// Diag 是一条静态检查诊断。
type Diag struct {
	Pos  Pos
	Code string
	Sev  string // "warning"
	Msg  string
}

func (d Diag) String() string {
	return fmt.Sprintf("%d:%d: %s: %s [%s]", d.Pos.Line, d.Pos.Col, d.Sev, d.Msg, d.Code)
}

// LintOptions 控制可选检查项。
type LintOptions struct {
	// Params 同时检查未使用形参。默认关闭：接口实现常有忽略的形参。
	Params bool
	// File 当前文件路径（import 解析与位置相关检查用；可空）。
	File string
	// LibDirs 额外 import 搜索目录（与 qkcheck -L 一致；可空 = 只看同目录）。
	LibDirs []string
}

// Lint 对已解析的程序做静态检查，返回按位置排序的诊断。
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
			l.globals[im.Type] = "space" // impl 目标未声明为 struct → 视为 space
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

// LintSource 便捷入口：解析源码并静态检查（解析失败返回错误与已得诊断）。
func LintSource(src string, opts LintOptions) ([]Diag, error) {
	prog, err := ParseSource(src)
	if err != nil {
		return nil, err
	}
	return Lint(prog, opts), nil
}

// ---------- 作用域模型（与 typecheck 对齐） ----------

type lvar struct {
	name     string
	pos      Pos
	kind     string // local | param | forvar | catch
	typeName string // 声明类型串（引用类型判定用）
	reads    int
	writes   int
	outer    *lvar // 遮蔽时指向外层同名变量
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

	// 新增检查的状态
	globals   map[string]string // 全局符号名 → 类别（fn/type/space/library/macro/alias）
	pending   map[string]Pos    // 变量 → 最近一次「尚未被读取」的赋值位置（QK113）
	tryDepth  int               // try 块深度（QK110 豁免：try 内的除零是刻意错误处理）
	inFn      bool              // 是否处于函数体内（QK112 只在函数内报遮蔽全局）
	usesNames map[string]bool   // 本文件出现过的标识符（QK111 导入使用判定）
	loopDepth int               // 循环体深度（QK113 豁免：循环体内赋值会被下一轮读取）
	refVars   map[string]bool   // 引用类型变量（T& / pointer T：`p = v` 是写穿，不是重新绑定）
}

func (l *linter) warn(pos Pos, code, format string, args ...interface{}) {
	l.diags = append(l.diags, Diag{Pos: pos, Code: code, Sev: "warning", Msg: fmt.Sprintf(format, args...)})
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

// declare 登记一个声明。同名外层变量存在且当前是嵌套作用域时记为遮蔽。
func (l *linter) declare(name string, pos Pos, kind string, typeName ...string) *lvar {
	if name == "_" || strings.HasPrefix(name, "_") {
		return nil // _ 前缀 = 显式忽略
	}
	v := &lvar{name: name, pos: pos, kind: kind}
	if len(typeName) > 0 {
		v.typeName = typeName[0]
	}
	if _, dup := l.scope.vars[name]; dup { // 当前作用域重名：typecheck 已报 duplicate，不重复报
		return nil
	}
	if outer := l.lookupOuter(name); outer != nil {
		v.outer = outer
		l.warn(pos, CodeShadow, "%s %s 遮蔽外层同名变量（外层声明于第 %d 行）", lintKindName(kind), name, outer.pos.Line)
	}
	// QK112：函数内声明遮蔽全局符号（函数/类型/空间/库/宏）——调用点会突然指向局部变量
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
		return "形参"
	case "forvar":
		return "循环变量"
	case "catch":
		return "catch 变量"
	default:
		return "变量"
	}
}

// use 记一次读取；读取后清除「未读赋值」记录（QK113 只有从未读取的赋值才算死存储）。
func (l *linter) use(name string) {
	if v := l.lookup(name); v != nil {
		v.reads++
	}
	delete(l.pending, name)
}

// lintGlobalName 全局符号类别的中文名。
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

// write 记一次赋值。
func (l *linter) write(name string) {
	if v := l.lookup(name); v != nil {
		v.writes++
	}
}

// ---------- 函数检查 ----------

func (l *linter) checkFunc(f *FuncDecl) {
	l.checkBody(f, f.Body, "")
}

// checkMethod 检查 impl / space 中的方法；implType 为所属类型（用于 void 调用解析）。
func (l *linter) checkMethod(f *FuncDecl, implType string) {
	l.checkBody(f, f.Body, implType)
}

// checkLibFuncs 库绑定（library 声明）只有签名，无体；此处不做检查。
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
	// 未使用变量/形参（函数级汇报）
	for _, v := range l.fnVars {
		if v.reads > 0 {
			continue
		}
		if v.kind == "param" && !l.opts.Params {
			continue
		}
		if v.kind == "catch" {
			continue // catch 变量默认不查：忽略错误对象是惯用法
		}
		if v.writes > 0 {
			l.warn(v.pos, codeForKind(v.kind), "%s %s 只被赋值、从未读取", lintKindName(v.kind), v.name)
		} else {
			l.warn(v.pos, codeForKind(v.kind), "%s %s 声明后从未使用", lintKindName(v.kind), v.name)
		}
	}
	// 缺返回：声明了非 void 返回类型，但存在不产生返回值的路径（运行期得到 nil）
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

// ---------- 语句遍历 ----------

// block 遍历语句块，返回（是否有终止语句, 终止是否全部为 return）。
// 终止 = return / log / break；不可达语句只报第一条。
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

// stmt 遍历一条语句，返回该语句是否终止（return/log/break）以及是否以 return 终止。
func (l *linter) stmt(s Stmt) (term bool, allRet bool) {
	switch t := s.(type) {
	case *ExprStmt:
		l.expr(t.X, false, "")
		return false, false
	case *LogStmt:
		l.expr(t.X, true, "")
		return true, false // log = 记录并结束，但不产生返回值
	case *DeleteStmt:
		l.expr(t.X, true, "") // delete x 视为对 x 的使用（生命周期操作）
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
		l.declare(t.Name, t.Pos, "local", t.Type) // 先算初值再登记（与 typecheck 一致）
		if isRefType(t.Type) {
			l.refVars[t.Name] = true // 引用类型：`p = v` 写穿指针目标，不算重新绑定
		}
		if t.Init != nil && l.loopDepth == 0 && !l.refVars[t.Name] {
			l.pending[t.Name] = t.Pos // QK113：初值若被后续赋值覆盖且其间未读取 → 死存储
		}
		return false, false
	case *AssignStmt:
		// QK108：自赋值 x = x（含 p.x / l[i] 的结构等价形式）
		if sameExpr(t.Target, t.X) {
			l.warn(t.Pos, CodeSelfAssign, "自赋值：%s 赋值给自身（无效果）", exprText(t.Target))
		}
		// 先求右值与目标子表达式：`x = x + 1` 里的读取会清除「未读赋值」记录，
		// 否则会把自己读自己的场景误判成死存储（QK113 误报源）。
		l.expr(t.X, true, "")
		switch tg := t.Target.(type) {
		case *Ident:
			// QK113：上一次对同一变量的赋值（含声明初值）若从未被读取 → 死存储。
			// 两类豁免（宁漏报不误报）：引用类型写穿；循环体内赋值会被下一轮读取。
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
		// 两分支都终止才算终止；无 else 时 else 隐含不终止
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
		return false, false // 循环可能一次都不执行
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

// ---------- 表达式遍历 ----------

// expr 遍历表达式。valueCtx=true 表示该表达式的值被使用（用于 void 误用检查）。
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
		// QK110：常量除零/取模零（try 内豁免：那是刻意的错误处理）
		if (t.Op == "/" || t.Op == "%") && l.tryDepth == 0 {
			if n, ok := t.R.(*IntLit); ok && n.V == 0 {
				l.warn(t.Pos, CodeDivZero, "常量除零：%s 0（运行期报错；若为刻意错误处理请放进 try 块）", t.Op)
			}
		}
		// QK114：自身比较 x == x / x != x（只对可寻址形态：标识符/成员/下标）
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
		if t.Sign != nil { // 签名调用 f(args) @mb(args)：签名实例名与额外实参都算使用
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

// singleFunc 返回唯一的同名函数（无重载）；重载无法静态判定返回类型时返回 nil。
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

// ---------- 接口近失配（接口未实现） ----------

// checkInterfaces 对每个接口，找出「实现了部分方法」的具体类型，报出缺失的方法。
// 结构满足在赋值点由 typecheck 报错；这里补的是「意图实现但漏了方法」的场景。
func (l *linter) checkInterfaces(prog *Program) {
	if len(prog.Interfaces) == 0 {
		return
	}
	structs := map[string]bool{}
	for _, s := range prog.Structs {
		structs[s.Name] = true
	}
	// 类型的实例方法集（impl 内首参为本类型/Self 的方法）
	methodsOf := map[string]map[string]bool{}
	typePos := map[string]Pos{}
	for _, im := range prog.Impls {
		if !structs[im.Type] {
			continue // 只对已声明 struct 做推断，避免 space / 外部类型误报
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

// ---------- 辅助：表达式结构键 / 出口判定 / 分支清除 / 导入检查 ----------

// isRefType 判断类型串是否为引用类型：T& 或 pointer …（写穿语义，不是重新绑定）。
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

// declTypeOf 取某变量的声明类型（QK112/QK113 辅助；未登记返回 ""）。
func declTypeOf(l *linter, name string) string {
	if v := l.lookup(name); v != nil {
		return v.typeName
	}
	return ""
}

// sameExpr 结构等价比较：只认简单形态（标识符 / 成员 / 下标 / 字面量），
// 其余（调用、运算等）一律视为不相等 → 不报，宁漏不误。
// 全部走指针/值比较，零分配（旧实现用 fmt.Sprintf 造键，是 lint 阶段的主要分配来源）。
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

// isRefLike 是否可寻址形态（自赋值/自身比较只对这些形态报告）。
func isRefLike(e Expr) bool {
	switch e.(type) {
	case *Ident, *MemberExpr, *IndexExpr:
		return true
	}
	return false
}

// exprText 生成诊断里展示的表达式文本（仅在实际报错时调用 → 慢路径无所谓分配）。
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
		b.WriteString("表达式")
	}
}

// checkConstCond 常量条件：if (true/false)、while (false)。
// while (true) 由调用点单独判定（有 break/return 出口时不报）。
func (l *linter) checkConstCond(cond Expr, pos Pos, kw string) {
	bl, ok := cond.(*BoolLit)
	if !ok {
		return
	}
	if kw == "while" && bl.V {
		return // 交给 while(true) 的出口分析
	}
	if bl.V {
		l.warn(pos, CodeConstCond, "%s (true)：条件恒为真，分支永远执行", kw)
	} else {
		l.warn(pos, CodeConstCond, "%s (false)：条件恒为假，分支永不执行", kw)
	}
}

// branch 进入嵌套块/分支前清除未读赋值记录：
// 分支内可能有读取，保守清空 → 宁漏报不误报（QK113）。
func (l *linter) branch(b *Block) {
	if b == nil || len(l.pending) == 0 {
		return // 已经是空表：无需清（常见情形，省 clear 开销）
	}
	clear(l.pending) // 复用同一张表，避免每个分支新建 map
}

// hasBreak 判断块内是否存在 break（不进入嵌套循环：嵌套循环里的 break 不外逃）。
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

// hasReturn 判断块内是否存在 return / log（函数级出口）。
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

// collectUses 收集本文件出现过的标识符名与空间名（QK111 判定「导入是否被使用」）。
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

// collectTypeNames 把类型串里的标识符记为「已使用」：`LibPoint p` 的类型标注不是 Ident，
// 只导入库的类型（或泛型实参里的库类型）同样属于「用到该导入」。
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

// checkImports 未使用的 import（QK111）：
// 载入被导入文件 → 收集其对外符号（pub 函数/类型 + 全部宏 + space/library 名），
// 若本文件从未出现其中任何名字 → 报「未使用导入」。
// 库无法解析、或符号集为空（.qlib 等）时**跳过**——宁可漏报也不误报。
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

// checkUnusedFuncs 未被调用的函数：
// 只对 program main 报（库文件里的函数就是对外 API），且只在整文件都没出现过该名字时报。
func (l *linter) checkUnusedFuncs(prog *Program) {
	// 只对「有 main 且没有任何 pub 符号」的程序报：
	//   - program library; → 库，函数是 API；
	//   - 有 pub（即使漏写 program library;，如 compiler/testdata/mathlib.qk）→ 同样是库；
	//   - 没有 main → 片段文件，判断依据不足。
	if prog.Kind == "library" || len(prog.Pub) > 0 || !hasMainFunc(prog) {
		return
	}
	for _, f := range prog.Funcs {
		if f.Name == "main" || l.usesNames[f.Name] {
			continue
		}
		l.warn(f.Pos, CodeUnusedFunc, "函数 %s 从未被调用（死代码；库文件中的函数不报）", f.Name)
	}
}

// hasMainFunc 判断程序是否定义了 main。
func hasMainFunc(prog *Program) bool {
	for _, f := range prog.Funcs {
		if f.Name == "main" {
			return true
		}
	}
	return false
}

// libPublicSymbols 收集库对外可见的符号名（pub 函数/类型、space 与其方法、FFI 库、宏）。
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
