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
	"sort"
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
}

// Lint 对已解析的程序做静态检查，返回按位置排序的诊断。
func Lint(prog *Program, opts LintOptions) []Diag {
	l := &linter{
		opts:   opts,
		funcs:  map[string][]*FuncDecl{},
		spaces: map[string]map[string]*FuncDecl{},
	}
	for _, f := range prog.Funcs {
		l.funcs[f.Name] = append(l.funcs[f.Name], f)
	}
	for _, im := range prog.Impls {
		if l.spaces[im.Type] == nil {
			l.spaces[im.Type] = map[string]*FuncDecl{}
		}
		for _, m := range im.Methods {
			l.spaces[im.Type][m.Name] = m
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
	name   string
	pos    Pos
	kind   string // local | param | forvar | catch
	reads  int
	writes int
	outer  *lvar // 遮蔽时指向外层同名变量
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
func (l *linter) declare(name string, pos Pos, kind string) *lvar {
	if name == "_" || strings.HasPrefix(name, "_") {
		return nil // _ 前缀 = 显式忽略
	}
	v := &lvar{name: name, pos: pos, kind: kind}
	if _, dup := l.scope.vars[name]; dup { // 当前作用域重名：typecheck 已报 duplicate，不重复报
		return nil
	}
	if outer := l.lookupOuter(name); outer != nil {
		v.outer = outer
		l.warn(pos, CodeShadow, "%s %s 遮蔽外层同名变量（外层声明于第 %d 行）", lintKindName(kind), name, outer.pos.Line)
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

// use 记一次读取。
func (l *linter) use(name string) {
	if v := l.lookup(name); v != nil {
		v.reads++
	}
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
	l.curFn, l.fnVars, l.scope = f, nil, nil
	l.push()
	for _, p := range f.Params {
		l.declare(p.Name, p.Pos, "param")
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
		l.declare(t.Name, t.Pos, "local") // 先算初值再登记（与 typecheck 一致）
		return false, false
	case *AssignStmt:
		switch tg := t.Target.(type) {
		case *Ident:
			l.write(tg.Name)
		case *MemberExpr:
			l.expr(tg.X, true, "")
		case *IndexExpr:
			l.expr(tg.X, true, "")
			l.expr(tg.Idx, true, "")
		default:
			l.expr(t.Target, true, "")
		}
		l.expr(t.X, true, "")
		return false, false
	case *IfStmt:
		l.expr(t.Cond, true, "")
		t1, r1 := l.block(t.Then)
		t2, r2 := false, false
		if t.Else != nil {
			t2, r2 = l.block(t.Else)
		}
		// 两分支都终止才算终止；无 else 时 else 隐含不终止
		return t1 && t2, r1 && r2
	case *WhileStmt:
		l.expr(t.Cond, true, "")
		l.block(t.Body)
		return false, false // 循环可能一次都不执行
	case *ForStmt:
		l.expr(t.Iter, true, "")
		l.push()
		l.declare(t.Var, t.Pos, "forvar")
		l.block(t.Body)
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
		l.block(t.Body)
		l.pop()
		return false, false
	case *TryStmt:
		t1, r1 := l.block(t.Try)
		l.push()
		l.declare(t.CatchVar, t.Pos, "catch")
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
