// Package cgen 的 lowering 层：正典 AST（internal/lang）→ cgen IR。
//
// 本文件是编译器唯一的前端入口：cgen.go 只保留 LLVM IR 发射器与 IR 数据结构，
// 词法/语法/类型检查全部来自 internal/lang（单一语法源，与解释器同源）。
//
// 语义基准：**解释器**。同一份源码在 /tmp/quark（解释器）与 qkc -run（编译器）
// 下输出必须逐字节一致；lowering 的每条规则都以 internal/lang 的 Typecheck 与
// eval 语义为准（不另造规则）。
//
// 值语义（与解释器对齐）：
//   - List / struct 是**引用**（解释器里是 *List / *Struct）：赋值、传参复制引用，
//     字段/元素修改对所有别名可见（q = p; q.a = 1 也会改到 p.a）。
//   - int → float 是唯一隐式转换（与 typecheck.assignable 一致）。
//
// 已 lower 的构造（阶段 A/B/C）：
//
//   - 语句：声明（int/bool/float/String/List<int>/struct/接口）、赋值（变量/List 下标/
//     struct 字段）、if/else、while、for（C 风格 + 迭代 for-in）、break、try/catch(void)、
//     return、log、delete List、io.println/io.print、表达式语句
//
//   - 表达式：int/float/bool/String 字面量、算术/比较/短路逻辑、String 拼接、
//     int/float/bool.toString()、函数调用/递归、List size/get/append/[i]、
//     struct 字面量（命名+位置）、字段读写、实例方法/静态方法、space 调用、
//     Operation 运算符重载（__add__ 等）、泛型函数/泛型 struct/泛型 impl 单态化、
//     接口装箱与 vtable dynamic 分发、内置 sum/clock
//
//   - 顶层：fn、main(IOStream io)、import（lang.CompileWithImports 递归合并）、
//     type struct、impl、space
//
//   - 顶层：library（FFI：LLVM declare + C ABI 直调 + ; qkc-link 链接标记）、
//     taskm（spawn/merge/block/done/channel，对接 qthreads 运行时）
//
// 仍未 lower（一律返回带位置的明确错误，绝不静默错编）：
// String/List 内建方法（substring/size/split/toString 等，除 int/float/bool.toString）、
// 指针/new/HashTable、打印 struct/接口值、copyd、签名调用 @sign、匿名 struct、
// interface{}（tAny）、long 与 FFI pointer、float 取模、含 log 函数的返回值被使用、
// merge 多参数/非 int 参数。
package cgen

import (
	"fmt"
	"strings"

	"quarklang/internal/lang"
)

// ---------- 带位置的诊断 ----------

// diagError 是带源码位置与源码行的编译期诊断。
type diagError struct {
	msg   string
	file  string
	line  int
	col   int
	srcLn string
	note  string
}

func (e *diagError) Error() string {
	var sb strings.Builder
	sb.WriteString("compiler: ")
	sb.WriteString(e.msg)
	if e.line > 0 {
		fmt.Fprintf(&sb, "\n  --> %s:%d:%d", e.file, e.line, e.col)
	}
	if e.note != "" {
		sb.WriteString(" " + e.note)
	}
	if e.srcLn != "" {
		disp, caret := expandTabs(e.srcLn, e.col)
		sb.WriteString("\n   |\n")
		fmt.Fprintf(&sb, "%3d| %s\n", e.line, disp)
		sb.WriteString("   | " + strings.Repeat(" ", max(0, caret-1)) + "^")
	}
	return sb.String()
}

// expandTabs 把源码行中的制表符展开为 4 空格，并把列号换算到展开后的位置。
func expandTabs(line string, col int) (string, int) {
	idx := col - 1
	if idx < 0 {
		idx = 0
	}
	if idx > len(line) {
		idx = len(line)
	}
	pad := 0
	for _, ch := range line[:idx] {
		if ch == '\t' {
			pad += 3
		}
	}
	return strings.ReplaceAll(line, "\t", "    "), col + pad
}

// exprPos 返回表达式位置；缺失时回退到 fallback。
func exprPos(x lang.Expr, fallback lang.Pos) lang.Pos {
	switch t := x.(type) {
	case *lang.IntLit:
		return t.Pos
	case *lang.FloatLit:
		return t.Pos
	case *lang.StrLit:
		return t.Pos
	case *lang.BoolLit:
		return t.Pos
	case *lang.NullLit:
		return t.Pos
	case *lang.Ident:
		return t.Pos
	case *lang.StructLit:
		return t.Pos
	case *lang.NewExpr:
		return t.Pos
	case *lang.BinOp:
		return t.Pos
	case *lang.UnOp:
		return t.Pos
	case *lang.CallExpr:
		return t.Pos
	case *lang.MemberExpr:
		return t.Pos
	case *lang.ScopeCall:
		return t.Pos
	case *lang.IndexExpr:
		return t.Pos
	}
	return fallback
}

// ---------- 声明表 ----------

// methodInfo 是 impl/space 里的一个方法。
type methodInfo struct {
	fn      *lang.FuncDecl
	impl    *lang.ImplDecl
	selfTyp string            // receiver 类型（"Self" 已替换为具体类型）
	isSelf  bool              // 有 self 首参（实例方法）
	irName  string            // IR 函数名（Point_sum / math_max）
	subst   map[string]string // impl 类型参数 → 具体类型
}

// lowerer 保存整个程序的符号信息与诊断上下文。
type lowerer struct {
	prog      *lang.Program
	file      string
	lines     []string // prog.Src（import 合并后源码）按行切分
	mainLines int      // 主文件行数（import 合并前的 src 行数）
	imported  bool

	fns      map[string]*lang.FuncDecl // 非泛型、非 main 的普通函数
	fnOrder  []string
	fnRet    map[string]string // lang 函数名 → 返回类型
	generics map[string]*lang.FuncDecl
	structs  map[string]*lang.StructDecl // 具名 struct（含泛型模板）
	ifaces   map[string]*lang.InterfaceDecl
	libs     map[string]*lang.LibraryDecl
	methods  map[string]map[string]*methodInfo // 类型/space 名 → 方法名 → 信息
	spaces   map[string]bool                   // space 名（impl.Type 无对应 struct）

	tys       []structDef // 需要发射的 struct 类型（含泛型实例）
	tyEmit    map[string]bool
	insts     map[string]string // 泛型实例 → 已实例化标记
	instMi    map[string]*methodInfo
	ifaceTbl  map[string][]ifaceM
	vtables   map[string]*vtableDef
	extSeen   map[string]bool
	runnerIdx map[string]string
	sigs      map[string]*funcDef
	nilFns    map[string]bool // 含 log 的函数（返回值可能为 nil，仅解释器可用）
	lowering  map[string]bool // 正在 lower 的函数（防递归实例化死循环）

	out *lowered
}

// lowerProgram 把正典 AST 降为 cgen IR。任何后端暂未 lower 的构造都会返回
// 带位置的明确错误，绝不静默生成错误代码。
// src 为主文件原始源码（用于诊断行号范围判断）。
func lowerProgram(prog *lang.Program, file, src string) (*lowered, error) {
	l := &lowerer{
		prog:      prog,
		file:      file,
		lines:     strings.Split(prog.Src, "\n"),
		mainLines: len(strings.Split(src, "\n")),
		imported:  len(prog.Imports) > 0,
		fns:       map[string]*lang.FuncDecl{},
		fnRet:     map[string]string{},
		generics:  map[string]*lang.FuncDecl{},
		structs:   map[string]*lang.StructDecl{},
		ifaces:    map[string]*lang.InterfaceDecl{},
		libs:      map[string]*lang.LibraryDecl{},
		methods:   map[string]map[string]*methodInfo{},
		spaces:    map[string]bool{},
		tyEmit:    map[string]bool{},
		insts:     map[string]string{},
		instMi:    map[string]*methodInfo{},
		ifaceTbl:  map[string][]ifaceM{},
		vtables:   map[string]*vtableDef{},
		extSeen:   map[string]bool{},
		runnerIdx: map[string]string{},
		sigs:      map[string]*funcDef{},
		nilFns:    map[string]bool{},
		lowering:  map[string]bool{},
		out:       &lowered{},
	}
	if l.file == "" {
		l.file = "<src>"
	}
	if err := l.collect(); err != nil {
		return nil, err
	}
	l.out.structs = l.tys
	for sym, v := range l.vtables {
		_ = sym
		l.out.vtables = append(l.out.vtables, v)
	}
	l.out.sortVTables()
	for name := range l.ifaces {
		l.out.ifaces = append(l.out.ifaces, name)
	}
	return l.out, nil
}

// errf 构造带位置的诊断。
func (l *lowerer) errf(pos lang.Pos, format string, args ...interface{}) error {
	d := &diagError{msg: fmt.Sprintf(format, args...), file: l.file, line: pos.Line, col: pos.Col}
	if d.line <= 0 {
		d.line, d.col = 1, 1
	}
	if d.col <= 0 {
		d.col = 1
	}
	if d.line <= len(l.lines) {
		d.srcLn = l.lines[d.line-1]
	}
	if l.imported && d.line > l.mainLines {
		d.note = "（import 合并后行号）"
	}
	return d
}

// errAt 构造固定提示位置的诊断（无 AST 位置可用的场景，如缺 main）。
func (l *lowerer) errAt(format string, args ...interface{}) error {
	return l.errf(lang.Pos{Line: 1, Col: 1}, format, args...)
}

// ---------- 顶层收集 ----------

func (l *lowerer) collect() error {
	if l.prog.Kind == "library" {
		return l.errAt("暂未支持编译 program library（库形态由解释器/导出流程处理）")
	}
	for _, sd := range l.prog.Structs {
		if sd.Name == "" {
			return l.errf(sd.Pos, "暂未支持匿名 struct 类型（编译器后端未实现）")
		}
		l.structs[sd.Name] = sd
	}
	for _, id := range l.prog.Interfaces {
		if id.Name != "" {
			l.ifaces[id.Name] = id
		}
	}
	for _, im := range l.prog.Impls {
		if err := l.registerImpl(im); err != nil {
			return err
		}
	}
	for _, ld := range l.prog.Libraries {
		l.libs[ld.Name] = ld
	}

	// 函数名收集（重载/重名 → 编译器无法区分，明确报错）
	seen := map[string]lang.Pos{}
	var mainFn *lang.FuncDecl
	for _, f := range l.prog.Funcs {
		if f.Name == "sum" || f.Name == "clock" {
			return l.errf(f.Pos, "暂未支持重定义内置函数 %q（编译器将 %s 视为内建）", f.Name, f.Name)
		}
		if p, dup := seen[f.Name]; dup {
			return l.errf(f.Pos, "暂未支持函数重载 %q（首次定义于 %d:%d，编译器后端未实现）", f.Name, p.Line, p.Col)
		}
		seen[f.Name] = f.Pos
		if len(f.TypeParams) > 0 {
			l.generics[f.Name] = f
			continue
		}
		if f.Name == "main" {
			mainFn = f
			continue
		}
		ret := f.Ret
		if ret == "" {
			ret = "void"
		}
		l.fns[f.Name] = f
		l.fnRet[f.Name] = ret
		l.fnOrder = append(l.fnOrder, f.Name)
	}
	if mainFn == nil {
		return l.errAt("未找到 main 函数（正典入口：fn main(IOStream io) { ... }）")
	}

	// 先 lower 普通函数（签名/语句/表达式的支持性检查都在这里完成）
	for _, name := range l.fnOrder {
		fd, err := l.lowerFunc(l.fns[name], name, "", "", nil)
		if err != nil {
			return err
		}
		l.out.funcs = append(l.out.funcs, fd)
	}
	// 再 lower impl/space 方法（非泛型 impl；泛型 impl 在调用点单态化）
	for _, im := range l.prog.Impls {
		if len(im.TypeParams) > 0 {
			continue
		}
		for _, m := range im.Methods {
			mi, ok := l.methods[im.Type][m.Name]
			if !ok {
				continue
			}
			selfTyp, selfName := "", ""
			if mi.isSelf {
				selfTyp = im.Type
				if len(m.Params) > 0 {
					selfName = m.Params[0].Name
				}
			}
			fd, err := l.lowerFunc(m, mi.irName, selfTyp, selfName, nil)
			if err != nil {
				return err
			}
			l.out.funcs = append(l.out.funcs, fd)
		}
	}
	stmts, err := l.lowerMain(mainFn)
	if err != nil {
		return err
	}
	l.out.mainStmts = stmts
	return nil
}

// registerImpl 登记 impl/space 的方法表（含方法重名检查）。
func (l *lowerer) registerImpl(im *lang.ImplDecl) error {
	typ := im.Type
	if _, isStruct := l.structs[typ]; !isStruct {
		l.spaces[typ] = true
	}
	if l.methods[typ] == nil {
		l.methods[typ] = map[string]*methodInfo{}
	}
	for _, m := range im.Methods {
		if _, dup := l.methods[typ][m.Name]; dup {
			return l.errf(m.Pos, "暂未支持重载/重定义方法 %s.%s（编译器按名字修饰生成函数，无法区分重载）", typ, m.Name)
		}
		mi := &methodInfo{fn: m, impl: im, irName: typ + "_" + m.Name, subst: map[string]string{}}
		for i, tp := range im.TypeParams {
			_ = i
			mi.subst[tp] = tp
		}
		if len(m.Params) > 0 && (isSelfParam(m.Params[0]) || m.Params[0].Type == typ) {
			mi.isSelf = true
			mi.selfTyp = typ
			if isSelfParam(m.Params[0]) {
				mi.selfTyp = typ
			}
		}
		l.methods[typ][m.Name] = mi
	}
	return nil
}

// isSelfParam 判断参数是否是 self 接收者（类型为 Self 或参数名 self）。
func isSelfParam(p lang.Param) bool {
	return strings.TrimSpace(p.Type) == "Self" || (p.Name == "self" && p.Type != "int" && p.Type != "float" && p.Type != "bool" && p.Type != "String")
}

// ---------- 类型工具 ----------

// splitGeneric 拆 "Box<int>" → ("Box", ["int"])；非泛型返回 (t, nil)。
func splitGeneric(t string) (string, []string) {
	t = strings.TrimSpace(t)
	i := strings.Index(t, "<")
	if i < 0 || !strings.HasSuffix(t, ">") {
		return t, nil
	}
	base := strings.TrimSpace(t[:i])
	inner := t[i+1 : len(t)-1]
	var args []string
	depth := 0
	start := 0
	for j := 0; j < len(inner); j++ {
		switch inner[j] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(inner[start:j]))
				start = j + 1
			}
		}
	}
	args = append(args, strings.TrimSpace(inner[start:]))
	return base, args
}

// substType 用类型参数替换表展开类型串（T → int，Box<T> → Box<int>）。
func substType(t string, sub map[string]string) string {
	if len(sub) == 0 {
		return t
	}
	if v, ok := sub[strings.TrimSpace(t)]; ok {
		return v
	}
	base, args := splitGeneric(t)
	if args == nil {
		return t
	}
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = substType(a, sub)
	}
	return base + "<" + strings.Join(out, ",") + ">"
}

// structSubst 返回 struct 实例的类型参数替换表（Point → {}；Box<int> → {T:int}）。
func (l *lowerer) structSubst(t string) (*lang.StructDecl, map[string]string) {
	base, args := splitGeneric(t)
	sd, ok := l.structs[base]
	if !ok {
		return nil, nil
	}
	sub := map[string]string{}
	for i, tp := range sd.TypeParams {
		if i < len(args) {
			sub[tp] = args[i]
		}
	}
	return sd, sub
}

// fieldType 返回 struct 字段类型（含泛型替换）；不存在返回 ""。
func (l *lowerer) fieldType(structType, field string) (string, bool) {
	sd, sub := l.structSubst(structType)
	if sd == nil {
		return "", false
	}
	for _, m := range sd.Members {
		if m.Name == field {
			return substType(m.Type, sub), true
		}
	}
	return "", false
}

// ensureStructTy 登记需要发射的 struct 类型定义（含泛型实例）。
func (l *lowerer) ensureStructTy(t string) {
	if l.tyEmit[t] {
		return
	}
	sd, sub := l.structSubst(t)
	if sd == nil {
		return
	}
	l.tyEmit[t] = true
	def := structDef{name: t}
	for _, m := range sd.Members {
		def.fields = append(def.fields, m.Name)
		def.fieldTypes = append(def.fieldTypes, substType(m.Type, sub))
	}
	// 依赖先发射：LLVM 命名类型必须先定义后使用
	for _, ft := range def.fieldTypes {
		if l.isStructType(ft) {
			l.ensureStructTy(ft)
		}
	}
	l.tys = append(l.tys, def)
}

// listElem 解析 List<T> 的元素类型。
func listElem(t string) (string, bool) {
	base, args := splitGeneric(t)
	if base != "List" || len(args) != 1 {
		return "", false
	}
	return args[0], true
}

// isStructType 判断类型是否是本程序声明的 struct（含泛型实例）。
func (l *lowerer) isStructType(t string) bool {
	base, _ := splitGeneric(t)
	_, ok := l.structs[base]
	return ok
}

// isIfaceType 判断类型是否是接口。
func (l *lowerer) isIfaceType(t string) bool {
	_, ok := l.ifaces[t]
	return ok
}

// checkType 校验类型可用性（不可 lower 的类型给出明确诊断）。
func (l *lowerer) checkType(t string, pos lang.Pos, what string) error {
	t = strings.TrimSpace(t)
	switch t {
	case "int", "bool", "float", "String", "List<int>":
		return nil
	case "void":
		return l.errf(pos, "暂未支持 void 类型 %s", what)
	}
	if elem, ok := listElem(t); ok {
		return l.errf(pos, "暂未支持 %s 类型 %q（编译器暂只支持 List<int>，元素 %s）", what, t, elem)
	}
	if l.isStructType(t) {
		return nil
	}
	if l.isIfaceType(t) {
		return nil // 接口类型：vtable 分发（Phase C）
	}
	switch t {
	case "long", "char":
		return l.errf(pos, "暂未支持 %s 类型 %q（编译器暂未 lower）", what, t)
	case "thread", "Task", "Channel", "channel":
		return nil // taskm：thread = pid(i32)，Channel = i8* 句柄
	case "IOStream":
		return l.errf(pos, "暂未支持 IOStream %s（编译器仅在 main 入口绑定 io）", what)
	case "interface{}":
		return l.errf(pos, "暂未支持 interface{} 类型 %s（dynamic 分发仅解释器可用）", what)
	}
	if strings.HasPrefix(t, "List") {
		return l.errf(pos, "暂未支持 %s 类型 %q（编译器仅支持 List<int>）", what, t)
	}
	if strings.HasPrefix(t, "HashTable") {
		return l.errf(pos, "暂未支持 %s 类型 %q（HashTable 仅解释器可用）", what, t)
	}
	if strings.HasPrefix(t, "pointer ") || strings.HasSuffix(t, "&") || strings.Contains(t, "[Copyd]") || strings.HasSuffix(t, "[]") {
		return l.errf(pos, "暂未支持指针/传时复制类型 %q（解释器可用）", t)
	}
	return l.errf(pos, "未知类型 %q（%s 声明）", t, what)
}

// retOK 判断返回类型是否可 lower。
func (l *lowerer) retOK(t string) bool {
	switch t {
	case "int", "bool", "float", "String", "void":
		return true
	}
	return l.isStructType(t) || l.isIfaceType(t)
}

// ---------- 作用域 ----------

// scope 是块级作用域（用于变量类型查询与重名/遮蔽检查）。
type scope struct {
	vars   map[string]string
	parent *scope
}

// funcCtx 是单个函数 lowering 的上下文。
type funcCtx struct {
	l      *lowerer
	fn     string // IR 函数名
	ret    string
	ioName string // main 的 IOStream 形参名
	top    *scope
	logged bool // 函数体内出现 log（返回值可能为 nil）

	// discarded 是当前「值被丢弃」的调用（表达式语句的顶层调用）：
	// 含 log 的函数返回值可能是 nil，只允许在这个位置调用。
	discarded *lang.CallExpr

	// expectT 是当前表达式的期望类型（用于补全泛型 struct 字面量的类型实参：
	// typecheck 只把 .{...} 的名字填成基名 "Box"，实参在声明类型里）。
	expectT string

	// subst 是当前泛型实例的类型参数替换表（T → int）。
	subst map[string]string
}

// resolveT 展开当前实例的类型参数（T → int；Box<T> → Box<int>）。
func (fc *funcCtx) resolveT(t string) string {
	if len(fc.subst) == 0 {
		return t
	}
	return substType(t, fc.subst)
}

// typeOfAs 在「期望类型 want」的上下文里推断表达式类型。
func (fc *funcCtx) typeOfAs(x lang.Expr, want string) string {
	old := fc.expectT
	fc.expectT = want
	t := fc.typeOf(x)
	fc.expectT = old
	return t
}

// exprAs 在「期望类型 want」的上下文里 lower 表达式（含接口装箱）。
func (fc *funcCtx) exprAs(x lang.Expr, want string) (*expr, error) {
	old := fc.expectT
	fc.expectT = want
	defer func() { fc.expectT = old }()
	e, err := fc.expr(x)
	if err != nil {
		return nil, err
	}
	return fc.boxTo(e, want, exprPos(x, lang.Pos{Line: 1, Col: 1}))
}

func (l *lowerer) newCtx(fn, ret string) *funcCtx {
	return &funcCtx{l: l, fn: fn, ret: ret, top: &scope{vars: map[string]string{}}}
}

func (fc *funcCtx) push() { fc.top = &scope{vars: map[string]string{}, parent: fc.top} }
func (fc *funcCtx) pop()  { fc.top = fc.top.parent }

// lookup 查变量类型（沿作用域链）。
func (fc *funcCtx) lookup(name string) (string, bool) {
	for s := fc.top; s != nil; s = s.parent {
		if t, ok := s.vars[name]; ok {
			return t, true
		}
	}
	return "", false
}

// declare 登记变量；重名或遮蔽外层变量时报明确错误（emitter 的变量表是平坦的，
// 遮蔽会静默错编，因此这里必须拦截）。
func (fc *funcCtx) declare(name, typ string, pos lang.Pos) error {
	if _, ok := fc.lookup(name); ok {
		return fc.l.errf(pos, "暂未支持变量重名/遮蔽 %q（编译器变量表不支持同名变量，请改名）", name)
	}
	fc.top.vars[name] = typ
	return nil
}

// ---------- 函数 lowering ----------

// lowerFunc lower 一个普通函数/方法。subst 为 impl 泛型参数替换表。
func (l *lowerer) lowerFunc(f *lang.FuncDecl, irName, selfTyp, selfParam string, subst map[string]string) (*funcDef, error) {
	ret := f.Ret
	if ret == "" {
		ret = "void"
	}
	ret = substType(ret, subst)
	if !l.retOK(ret) {
		return nil, l.errf(f.Pos, "暂未支持返回类型 %q（编译器支持 int/bool/float/String/void/struct）", f.Ret)
	}
	fc := l.newCtx(irName, ret)
	fc.subst = subst
	params := make([]funcParam, 0, len(f.Params))
	start := 0
	if selfTyp != "" {
		if len(f.Params) == 0 {
			return nil, l.errf(f.Pos, "impl 实例方法 %s 缺少 self 首参", f.Name)
		}
		if err := fc.declare(selfParam, selfTyp, f.Params[0].Pos); err != nil {
			return nil, err
		}
		start = 1
	}
	for _, p := range f.Params[start:] {
		pt := substType(p.Type, subst)
		if err := l.checkType(pt, p.Pos, "参数"); err != nil {
			return nil, err
		}
		if err := fc.declare(p.Name, pt, p.Pos); err != nil {
			return nil, err
		}
		params = append(params, funcParam{name: p.Name, typ: pt})
	}
	body, err := fc.block(f.Body)
	if err != nil {
		return nil, err
	}
	fd := &funcDef{name: irName, params: params, ret: ret, body: body, selfTyp: selfTyp, selfParam: selfParam}
	if fc.logged && ret != "void" {
		l.nilFns[irName] = true
	}
	if l.sigs[irName] == nil {
		l.sigs[irName] = fd
	}
	return fd, nil
}

// lowerMain lower 程序入口 fn main(IOStream io)。
func (l *lowerer) lowerMain(f *lang.FuncDecl) ([]stmt, error) {
	if len(f.TypeParams) > 0 {
		return nil, l.errf(f.Pos, "暂未支持泛型 main 函数")
	}
	if f.Ret != "" && f.Ret != "void" {
		return nil, l.errf(f.Pos, "main 不能有返回值（got %q）", f.Ret)
	}
	if len(f.Params) == 0 || f.Params[0].Type != "IOStream" {
		return nil, l.errf(f.Pos, "main 的第一个参数必须是 IOStream（正典：fn main(IOStream io) { ... }）")
	}
	if len(f.Params) > 1 {
		return nil, l.errf(f.Params[1].Pos, "暂未支持 main 的 %q 参数（编译器只绑定 IOStream io）", f.Params[1].Name)
	}
	fc := l.newCtx("main", "void")
	fc.ioName = f.Params[0].Name
	if err := fc.declare(fc.ioName, "IOStream", f.Params[0].Pos); err != nil {
		return nil, err
	}
	return fc.block(f.Body)
}

// ---------- 语句 lowering ----------

func (fc *funcCtx) block(b *lang.Block) ([]stmt, error) {
	fc.push()
	defer fc.pop()
	out := make([]stmt, 0, len(b.Stmts))
	for _, s := range b.Stmts {
		st, err := fc.stmt(s)
		if err != nil {
			return nil, err
		}
		if st != nil { // 仅登记、不产生代码的语句
			out = append(out, st)
		}
	}
	return out, nil
}

func (fc *funcCtx) stmt(s lang.Stmt) (stmt, error) {
	l := fc.l
	switch st := s.(type) {
	case *lang.DeclStmt:
		if st.Decor == "copyd" {
			return nil, l.errf(st.Pos, "暂未支持 copyd 修饰（传时复制仅解释器可用）")
		}
		return fc.declStmt(st)

	case *lang.AssignStmt:
		return fc.assignStmt(st)

	case *lang.ExprStmt:
		// io.println(...) / io.print(...) 语句
		if call, ok := st.X.(*lang.CallExpr); ok {
			if me, ok := call.Fn.(*lang.MemberExpr); ok {
				if id, ok := me.X.(*lang.Ident); ok && fc.ioName != "" && id.Name == fc.ioName {
					switch me.Name {
					case "println":
						args, err := fc.printArgs(call)
						if err != nil {
							return nil, err
						}
						return &printlnStmt{args: args}, nil
					case "print":
						args, err := fc.printArgs(call)
						if err != nil {
							return nil, err
						}
						return &printStmt{args: args}, nil
					default:
						return nil, l.errf(call.Pos, "暂未支持 IOStream 方法 %q（编译器仅支持 io.println/io.print）", me.Name)
					}
				}
			}
		}
		// 表达式语句的顶层调用：值被丢弃，含 log 的函数也允许调用
		if call, ok := st.X.(*lang.CallExpr); ok {
			old := fc.discarded
			fc.discarded = call
			x, err := fc.expr(st.X)
			fc.discarded = old
			if err != nil {
				return nil, err
			}
			return &exprStmt{x: x}, nil
		}
		x, err := fc.expr(st.X)
		if err != nil {
			return nil, err
		}
		return &exprStmt{x: x}, nil

	case *lang.ReturnStmt:
		if st.X == nil {
			return &returnStmt{x: nil}, nil
		}
		if fc.ret == "void" {
			return nil, l.errf(st.Pos, "void 函数不能 return 值（解释器会忽略返回值，编译器不静默忽略）")
		}
		t := fc.typeOfAs(st.X, fc.ret)
		if !fc.assignable(t, fc.ret) {
			return nil, l.errf(exprPos(st.X, st.Pos), "暂未支持从 %s 函数返回 %s 值", fc.ret, t)
		}
		x, err := fc.exprAs(st.X, fc.ret)
		if err != nil {
			return nil, err
		}
		return &returnStmt{x: x}, nil

	case *lang.IfStmt:
		cond, err := fc.cond(st.Cond)
		if err != nil {
			return nil, err
		}
		thenB, err := fc.block(st.Then)
		if err != nil {
			return nil, err
		}
		var els []stmt
		if st.Else != nil {
			if els, err = fc.block(st.Else); err != nil {
				return nil, err
			}
		}
		return &ifStmt{cond: cond, then: thenB, els: els}, nil

	case *lang.WhileStmt:
		cond, err := fc.cond(st.Cond)
		if err != nil {
			return nil, err
		}
		body, err := fc.block(st.Body)
		if err != nil {
			return nil, err
		}
		return &whileStmt{cond: cond, body: body}, nil

	case *lang.ForCStmt:
		fc.push()
		defer fc.pop()
		var init stmt
		var err error
		if st.Init != nil {
			if init, err = fc.stmt(st.Init); err != nil {
				return nil, err
			}
		}
		cond, err := fc.cond(st.Cond)
		if err != nil {
			return nil, err
		}
		var step stmt
		if st.Step != nil {
			if step, err = fc.stmt(st.Step); err != nil {
				return nil, err
			}
		}
		body, err := fc.block(st.Body)
		if err != nil {
			return nil, err
		}
		return &forStmt{init: init, cond: cond, step: step, body: body}, nil

	case *lang.ForStmt:
		return fc.forIn(st)

	case *lang.BreakStmt:
		return &breakStmt{}, nil

	case *lang.LogStmt:
		x, err := fc.expr(st.X)
		if err != nil {
			return nil, err
		}
		fc.logged = true
		return &logStmt{x: x}, nil

	case *lang.TryStmt:
		if st.CatchVarType != "" && st.CatchVarType != "void" {
			return nil, l.errf(st.Pos, "暂未支持 catch (%s %s)（编译器仅支持 catch (void e)）", st.CatchVarType, st.CatchVar)
		}
		if st.CatchVar != "" && blockUsesIdent(st.Catch, st.CatchVar) {
			return nil, l.errf(st.Pos, "暂未支持在 catch 体内使用 %q（错误值传递仅解释器可用）", st.CatchVar)
		}
		thenB, err := fc.block(st.Try)
		if err != nil {
			return nil, err
		}
		catchB, err := fc.block(st.Catch)
		if err != nil {
			return nil, err
		}
		return &tryStmt{then: thenB, catch: catchB}, nil

	case *lang.DeleteStmt:
		id, ok := st.X.(*lang.Ident)
		if !ok {
			return nil, l.errf(st.Pos, "暂未支持 delete 非变量表达式（编译器仅支持 delete <List 变量>）")
		}
		if t, _ := fc.lookup(id.Name); t != "List<int>" {
			return nil, l.errf(st.Pos, "暂未支持 delete %q（编译器仅支持 delete List 变量）", id.Name)
		}
		return &deleteStmt{name: id.Name}, nil
	}
	return nil, l.errf(lang.Pos{Line: 1, Col: 1}, "compiler 内部错误：未知语句类型 %T", s)
}

// forIn lower 迭代 for：for (<T> x : list)。
func (fc *funcCtx) forIn(st *lang.ForStmt) (stmt, error) {
	l := fc.l
	lt := fc.typeOf(st.Iter)
	base, args := splitGeneric(lt)
	if base != "List" || len(args) != 1 {
		return nil, l.errf(st.Pos, "暂未支持对 %s 的 for 迭代（编译器支持 List<int>）", lt)
	}
	elem := args[0]
	if elem != "int" {
		return nil, l.errf(st.Pos, "暂未支持迭代 List<%s>（编译器暂只支持 List<int>）", elem)
	}
	id, ok := st.Iter.(*lang.Ident)
	if !ok {
		return nil, l.errf(st.Pos, "暂未支持对非变量列表的 for 迭代（编译器要求 List 变量）")
	}
	vt, _ := fc.lookup(id.Name)
	if vt != "List<int>" {
		return nil, l.errf(st.Pos, "暂未支持对 %s 的 for 迭代（编译器支持 List<int>）", vt)
	}
	fc.push()
	defer fc.pop()
	if err := fc.declare(st.Var, elem, st.Pos); err != nil {
		return nil, err
	}
	body, err := fc.block(st.Body)
	if err != nil {
		return nil, err
	}
	return &forInStmt{name: st.Var, typ: elem, list: id.Name, body: body}, nil
}

// ---------- 变量声明 ----------

func (fc *funcCtx) declStmt(st *lang.DeclStmt) (stmt, error) {
	l := fc.l
	t := fc.resolveT(st.Type)
	if err := l.checkType(t, st.Pos, "变量"); err != nil {
		return nil, err
	}
	switch t {
	case "int", "bool", "float", "String", "thread", "Channel", "channel":
		if st.Init == nil {
			if err := fc.declare(st.Name, t, st.Pos); err != nil {
				return nil, err
			}
			return &declStmt{name: st.Name, typ: t}, nil
		}
		it := fc.typeOf(st.Init)
		if !fc.assignable(it, t) {
			return nil, l.errf(exprPos(st.Init, st.Pos), "暂未支持用 %s 初始化 %s 变量 %q", it, t, st.Name)
		}
		x, err := fc.expr(st.Init)
		if err != nil {
			return nil, err
		}
		if err := fc.declare(st.Name, t, st.Pos); err != nil {
			return nil, err
		}
		return &declStmt{name: st.Name, typ: t, init: x}, nil

	case "List<int>":
		if st.Init == nil {
			return nil, l.errf(st.Pos, "暂未支持无初值的 List<int> %q（解释器零值为空列表，编译器后端未 lower）", st.Name)
		}
		lit, ok := st.Init.(*lang.ListLit)
		if !ok {
			it := fc.typeOf(st.Init)
			if it != "List<int>" {
				return nil, l.errf(exprPos(st.Init, st.Pos), "暂未支持 List<int> 的这种初始化（编译器仅支持 [a, b, ...] 字面量）")
			}
			x, err := fc.expr(st.Init)
			if err != nil {
				return nil, err
			}
			if err := fc.declare(st.Name, t, st.Pos); err != nil {
				return nil, err
			}
			return &declStmt{name: st.Name, typ: t, init: x}, nil
		}
		items := make([]*expr, 0, len(lit.Items))
		for _, it := range lit.Items {
			et := fc.typeOf(it)
			if !fc.assignable(et, "int") {
				return nil, l.errf(exprPos(it, st.Pos), "暂未支持用 %s 元素初始化 List<int>", et)
			}
			x, err := fc.expr(it)
			if err != nil {
				return nil, err
			}
			items = append(items, x)
		}
		if err := fc.declare(st.Name, t, st.Pos); err != nil {
			return nil, err
		}
		return &declStmt{name: st.Name, typ: t, init: &expr{kind: kList, typ: t, lst: &listLit{items: items}}}, nil
	}

	// 接口类型（vtable 装箱）
	if l.isIfaceType(t) {
		if st.Init == nil {
			if err := fc.declare(st.Name, t, st.Pos); err != nil {
				return nil, err
			}
			return &declStmt{name: st.Name, typ: t}, nil
		}
		it := fc.typeOfAs(st.Init, t)
		if !fc.assignable(it, t) {
			return nil, l.errf(exprPos(st.Init, st.Pos), "暂未支持用 %s 初始化接口 %s 变量 %q", it, t, st.Name)
		}
		x, err := fc.exprAs(st.Init, t)
		if err != nil {
			return nil, err
		}
		if err := fc.declare(st.Name, t, st.Pos); err != nil {
			return nil, err
		}
		return &declStmt{name: st.Name, typ: t, init: x}, nil
	}
	// struct 类型
	l.ensureStructTy(t)
	if st.Init == nil {
		if err := fc.declare(st.Name, t, st.Pos); err != nil {
			return nil, err
		}
		return &declStmt{name: st.Name, typ: t}, nil
	}
	it := fc.typeOfAs(st.Init, t)
	if it != t {
		return nil, l.errf(exprPos(st.Init, st.Pos), "暂未支持用 %s 初始化 %s 变量 %q", it, t, st.Name)
	}
	x, err := fc.exprAs(st.Init, t)
	if err != nil {
		return nil, err
	}
	if err := fc.declare(st.Name, t, st.Pos); err != nil {
		return nil, err
	}
	return &declStmt{name: st.Name, typ: t, init: x}, nil
}

// printArgs lower io.println/io.print 的实参。
func (fc *funcCtx) printArgs(call *lang.CallExpr) ([]*expr, error) {
	args := make([]*expr, 0, len(call.Args))
	for _, a := range call.Args {
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		switch t := fc.typeOf(a); t {
		case "int", "String", "bool", "float", "?":
		default:
			return nil, fc.l.errf(exprPos(a, call.Pos), "暂未支持打印 %s 类型的值（编译器支持 int/float/bool/String）", t)
		}
		args = append(args, x)
	}
	return args, nil
}

// cond lower 条件表达式（typecheck 已保证 bool）。
func (fc *funcCtx) cond(x lang.Expr) (*expr, error) {
	switch t := fc.typeOf(x); t {
	case "bool", "?":
	default:
		return nil, fc.l.errf(exprPos(x, lang.Pos{Line: 1, Col: 1}), "暂未支持 %s 类型的条件（需要 bool）", t)
	}
	return fc.expr(x)
}

// ---------- 赋值 ----------

func (fc *funcCtx) assignStmt(st *lang.AssignStmt) (stmt, error) {
	l := fc.l
	switch tgt := st.Target.(type) {
	case *lang.Ident:
		vt, ok := fc.lookup(tgt.Name)
		if !ok {
			return nil, l.errf(tgt.Pos, "赋值目标 %q 未声明", tgt.Name)
		}
		if !fc.assignable(fc.typeOfAs(st.X, vt), vt) {
			return nil, l.errf(exprPos(st.X, st.Pos), "暂未支持用 %s 给 %s 变量 %q 赋值", fc.typeOf(st.X), vt, tgt.Name)
		}
		x, err := fc.exprAs(st.X, vt)
		if err != nil {
			return nil, err
		}
		return &assignStmt{name: tgt.Name, x: x}, nil

	case *lang.IndexExpr:
		rt := fc.typeOf(tgt.X)
		if _, ok := listElem(rt); !ok {
			return nil, l.errf(st.Pos, "暂未支持该下标赋值（编译器仅支持 List<int>）")
		}
		if t := fc.typeOf(tgt.Idx); t != "int" && t != "?" {
			return nil, l.errf(exprPos(tgt.Idx, st.Pos), "暂未支持下标为 %s（需要 int）", t)
		}
		id, ok := tgt.X.(*lang.Ident)
		if !ok {
			return nil, l.errf(st.Pos, "暂未支持该下标赋值（编译器仅支持 <List 变量>[i] = v）")
		}
		if t := fc.typeOf(st.X); !fc.assignable(t, "int") {
			return nil, l.errf(exprPos(st.X, st.Pos), "暂未支持用 %s 赋值给 List<int> 元素", t)
		}
		idx, err := fc.expr(tgt.Idx)
		if err != nil {
			return nil, err
		}
		x, err := fc.expr(st.X)
		if err != nil {
			return nil, err
		}
		return &indexAssignStmt{name: id.Name, idx: idx, x: x}, nil

	case *lang.MemberExpr:
		rt := fc.typeOf(tgt.X)
		if l.isStructType(rt) {
			ft, ok := l.fieldType(rt, tgt.Name)
			if !ok {
				return nil, l.errf(tgt.Pos, "struct %s 没有字段 %q", rt, tgt.Name)
			}
			if !fc.assignable(fc.typeOf(st.X), ft) {
				return nil, l.errf(exprPos(st.X, st.Pos), "暂未支持用 %s 给 %s.%s（%s）赋值", fc.typeOf(st.X), rt, tgt.Name, ft)
			}
			recv, err := fc.expr(tgt.X)
			if err != nil {
				return nil, err
			}
			x, err := fc.exprAs(st.X, ft)
			if err != nil {
				return nil, err
			}
			return &fieldAssignStmt{recv: recv, field: tgt.Name, x: x}, nil
		}
		return nil, l.errf(tgt.Pos, "暂未支持成员赋值（接口分发仅解释器可用）")
	}
	return nil, l.errf(st.Pos, "暂未支持该赋值目标（编译器仅支持变量、List 下标与 struct 字段）")
}

// ---------- 表达式 ----------

func (fc *funcCtx) expr(x lang.Expr) (*expr, error) {
	l := fc.l
	switch e := x.(type) {
	case *lang.IntLit:
		return &expr{kind: kInt, typ: "int", i: e.V, line: e.Pos.Line}, nil
	case *lang.FloatLit:
		return &expr{kind: kFloat, typ: "float", f: e.V, line: e.Pos.Line}, nil
	case *lang.StrLit:
		return &expr{kind: kString, typ: "String", s: e.V}, nil
	case *lang.BoolLit:
		return &expr{kind: kBool, typ: "bool", b: e.V}, nil
	case *lang.NullLit:
		return nil, l.errf(e.Pos, "暂未支持 null（编译器暂只支持标量/struct/List<int>）")

	case *lang.Ident:
		t, ok := fc.lookup(e.Name)
		if !ok {
			if _, isLib := l.libs[e.Name]; isLib {
				return nil, l.errf(e.Pos, "暂未支持 library 库对象 %q 作为值（FFI 仅解释器可用）", e.Name)
			}
			return nil, l.errf(e.Pos, "未声明的标识符 %q（编译器不支持函数引用/全局名）", e.Name)
		}
		if t == "IOStream" {
			return nil, l.errf(e.Pos, "暂未支持 IOStream 值 %q（编译器仅在 main 入口绑定 io）", e.Name)
		}
		return &expr{kind: kIdent, typ: t, s: e.Name}, nil

	case *lang.BinOp:
		return fc.binOp(e)
	case *lang.UnOp:
		return fc.unOp(e)
	case *lang.CallExpr:
		return fc.call(e)
	case *lang.IndexExpr:
		rt := fc.typeOf(e.X)
		elem, ok := listElem(rt)
		if !ok {
			return nil, l.errf(e.Pos, "暂未支持该下标访问（编译器仅支持 List<int>）")
		}
		id, ok := e.X.(*lang.Ident)
		if !ok {
			return nil, l.errf(e.Pos, "暂未支持该下标访问（编译器仅支持 <List 变量>[i]）")
		}
		if t := fc.typeOf(e.Idx); t != "int" && t != "?" {
			return nil, l.errf(exprPos(e.Idx, e.Pos), "暂未支持下标为 %s（需要 int）", t)
		}
		idx, err := fc.expr(e.Idx)
		if err != nil {
			return nil, err
		}
		return &expr{kind: kIndex, typ: elem, line: e.Pos.Line, idx: &indexExpr{name: id.Name, i: idx}}, nil

	case *lang.StructLit:
		return fc.structLit(e, fc.expectT)

	case *lang.NewExpr:
		return nil, l.errf(e.Pos, "暂未支持 new %s（堆分配仅解释器可用）", e.Typ)
	case *lang.ListLit:
		return nil, l.errf(exprPos(e, lang.Pos{Line: 1, Col: 1}), "暂未支持该处的列表字面量（编译器仅支持 List<int> 变量初始化的 [a, b, ...]）")
	case *lang.ScopeCall:
		return fc.scopeCall(e)
	case *lang.MemberExpr:
		return fc.member(e)
	}
	return nil, l.errf(lang.Pos{Line: 1, Col: 1}, "compiler 内部错误：未知表达式类型 %T", x)
}

// binOp lower 二元运算（算术/比较/逻辑/String 拼接/运算符重载）。
func (fc *funcCtx) binOp(e *lang.BinOp) (*expr, error) {
	l := fc.l
	lt, rt := fc.typeOf(e.L), fc.typeOf(e.R)
	pos := e.Pos
	// 运算符重载：同 struct 类型且 impl 定义了 __add__ 等
	if lt == rt && l.isStructType(lt) {
		if op := opMethodFor(e.Op); op != "" {
			if mi := l.lookupSelf(lt, op); mi != nil && len(mi.fn.Params) == 2 {
				imi, err := l.instantiateFor(mi, lt, []lang.Expr{e.L, e.R}, fc, e.Pos)
				if err != nil {
					return nil, err
				}
				return fc.methodCall(imi, lt, e.L, []lang.Expr{e.R}, e.Pos)
			}
		}
	}
	switch e.Op {
	case "+":
		if lt == "String" || rt == "String" {
			if lt != "String" || rt != "String" {
				return nil, l.errf(pos, "暂未支持 %s 与 %s 相加（String 拼接要求两侧都是 String）", lt, rt)
			}
			x, err := fc.binary(e, "+")
			if err != nil {
				return nil, err
			}
			x.typ = "String"
			x.strcat = true
			return x, nil
		}
		if !numLike(lt) || !numLike(rt) {
			return nil, l.errf(pos, "暂未支持对 %s / %s 使用 %q（Operation 运算符重载仅同类型 struct 可用）", lt, rt, e.Op)
		}
		x, err := fc.binary(e, "+")
		if err != nil {
			return nil, err
		}
		x.typ = numResult(lt, rt)
		return x, nil
	case "-", "*", "/":
		if !numLike(lt) || !numLike(rt) {
			return nil, l.errf(pos, "暂未支持对 %s / %s 使用 %q（Operation 运算符重载仅同类型 struct 可用）", lt, rt, e.Op)
		}
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.typ = numResult(lt, rt)
		x.line = pos.Line
		return x, nil
	case "%":
		if lt == "float" || rt == "float" {
			return nil, l.errf(pos, "暂未支持 float 取模（解释器在运行期报 TypeError: '%%' requires int operands）")
		}
		if !numLike(lt) || !numLike(rt) {
			return nil, l.errf(pos, "暂未支持对 %s / %s 使用 %q（Operation 运算符重载仅同类型 struct 可用）", lt, rt, e.Op)
		}
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.typ = "int"
		x.line = pos.Line
		return x, nil
	case "<<", ">>":
		if lt != "int" || rt != "int" {
			return nil, l.errf(pos, "暂未支持对 %s / %s 使用 %q（位移需要 int 操作数）", lt, rt, e.Op)
		}
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.typ = "int"
		return x, nil
	case "==", "!=", "<", "<=", ">", ">=":
		if l.isStructType(lt) && lt == rt {
			// __eq__ 等已在上方重载分支处理；其余情况 = 引用比较（解释器 Value 语义）
			x, err := fc.binary(e, e.Op)
			if err != nil {
				return nil, err
			}
			x.kind = kCmp
			x.typ = "bool"
			return x, nil
		}
		if lt == "String" && rt == "String" {
			x, err := fc.binary(e, e.Op)
			if err != nil {
				return nil, err
			}
			x.kind = kCmp
			x.typ = "bool"
			return x, nil
		}
		if lt == "bool" && rt == "bool" {
			if e.Op != "==" && e.Op != "!=" {
				return nil, l.errf(pos, "暂未支持对 bool 使用 %q（解释器只允许 == / !=）", e.Op)
			}
			x, err := fc.binary(e, e.Op)
			if err != nil {
				return nil, err
			}
			x.kind = kCmp
			x.typ = "bool"
			return x, nil
		}
		if !numLike(lt) || !numLike(rt) {
			return nil, l.errf(pos, "暂未支持对 %s / %s 使用 %q（String/Operation 比较仅同类型可用）", lt, rt, e.Op)
		}
		// 顺序比较：int 与 float 都按 float64 比较（解释器 ordCmp）
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.kind = kCmp
		x.typ = "bool"
		return x, nil
	case "&&", "||":
		for _, t := range []string{lt, rt} {
			if t != "bool" && t != "?" {
				return nil, l.errf(pos, "暂未支持对 %s 使用 %q（需要 bool）", t, e.Op)
			}
		}
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.kind = kAndOr
		x.typ = "bool"
		return x, nil
	}
	return nil, l.errf(pos, "暂未支持运算符 %q（解释器可用）", e.Op)
}

// opMethodFor 运算符 → Operation 协议方法名（与 internal/lang/typecheck 一致）。
func opMethodFor(op string) string {
	switch op {
	case "+":
		return "__add__"
	case "-":
		return "__sub__"
	case "*":
		return "__mul__"
	case "/":
		return "__div__"
	case "%":
		return "__mod__"
	case "==":
		return "__eq__"
	case "!=":
		return "__ne__"
	case "<":
		return "__lt__"
	case "<=":
		return "__le__"
	case ">":
		return "__gt__"
	case ">=":
		return "__ge__"
	}
	return ""
}

// binary lower 左右操作数并构造运算表达式。
func (fc *funcCtx) binary(e *lang.BinOp, op string) (*expr, error) {
	x, err := fc.expr(e.L)
	if err != nil {
		return nil, err
	}
	y, err := fc.expr(e.R)
	if err != nil {
		return nil, err
	}
	kind := kBin
	if op == "==" || op == "!=" || op == "<" || op == "<=" || op == ">" || op == ">=" {
		kind = kCmp
	}
	if op == "&&" || op == "||" {
		kind = kAndOr
	}
	return &expr{kind: kind, op: op, l: x, r: y, line: e.Pos.Line}, nil
}

// unOp lower 一元运算：- 展开为 0 - x（float 用 fneg 语义），! 走逻辑非。
func (fc *funcCtx) unOp(e *lang.UnOp) (*expr, error) {
	l := fc.l
	t := fc.typeOf(e.X)
	switch e.Op {
	case "-":
		if l.isStructType(t) {
			if mi := l.lookupSelf(t, "__neg__"); mi != nil {
				imi, err := l.instantiateFor(mi, t, []lang.Expr{e.X}, fc, e.Pos)
				if err != nil {
					return nil, err
				}
				return fc.methodCall(imi, t, e.X, nil, e.Pos)
			}
		}
		if !numLike(t) {
			return nil, l.errf(e.Pos, "暂未支持对 %s 取负（Operation 运算符重载仅同类型 struct 可用）", t)
		}
		inner, err := fc.expr(e.X)
		if err != nil {
			return nil, err
		}
		if t == "float" {
			r := &expr{kind: kBin, op: "-", typ: "float", l: &expr{kind: kFloat, typ: "float"}, r: inner}
			return r, nil
		}
		return &expr{kind: kBin, op: "-", typ: "int", l: &expr{kind: kInt, typ: "int"}, r: inner}, nil
	case "!":
		if t != "bool" && t != "?" {
			return nil, l.errf(e.Pos, "暂未支持对 %s 取反（需要 bool）", t)
		}
		inner, err := fc.expr(e.X)
		if err != nil {
			return nil, err
		}
		return &expr{kind: kAndOr, op: "!", typ: "bool", l: inner}, nil
	case "*":
		return nil, l.errf(e.Pos, "暂未支持解引用 *（指针仅解释器可用）")
	}
	return nil, l.errf(e.Pos, "暂未支持一元运算符 %q", e.Op)
}

// ---------- 调用 ----------

func (fc *funcCtx) call(c *lang.CallExpr) (*expr, error) {
	l := fc.l
	if c.Sign != nil {
		return nil, l.errf(c.Pos, "暂未支持签名调用 %s @%s()（解释器可用）", identifierName(c.Fn), c.Sign.Name)
	}
	switch fn := c.Fn.(type) {
	case *lang.Ident:
		return fc.callNamed(fn.Name, fn.Pos, c)
	case *lang.MemberExpr:
		return fc.callMethod(c, fn)
	case *lang.ScopeCall:
		return fc.scopeCallCall(c, fn)
	}
	return nil, l.errf(c.Pos, "暂未支持该调用形式（编译器仅支持 f(...)、obj.m(...) 与 T::m(...)）")
}

// callNamed lower 具名函数调用（含内建 sum/clock 与泛型实例化）。
func (fc *funcCtx) callNamed(name string, pos lang.Pos, c *lang.CallExpr) (*expr, error) {
	l := fc.l
	switch name {
	case "sum":
		return fc.callSum(c, pos)
	case "clock":
		if len(c.Args) != 0 {
			return nil, l.errf(pos, "clock() 不接受参数，got %d", len(c.Args))
		}
		return &expr{kind: kCall, typ: "int", call: &callExpr{name: "clock"}}, nil
	}
	if _, isGeneric := l.generics[name]; isGeneric {
		return fc.callGeneric(name, c, pos)
	}
	fd, ok := l.fns[name]
	if !ok {
		return nil, l.errf(pos, "未知函数 %q（编译器只支持同一程序内定义的函数）", name)
	}
	if len(c.Args) != len(fd.Params) {
		return nil, l.errf(pos, "函数 %s 需要 %d 个参数，got %d", name, len(fd.Params), len(c.Args))
	}
	if l.nilFns[name] && fc.discarded != c {
		return nil, l.errf(pos, "暂未支持在表达式中使用含 log 的函数 %s 的返回值（解释器返回 nil）", name)
	}
	args, err := fc.callArgs(c, fd.Params)
	if err != nil {
		return nil, err
	}
	return &expr{kind: kCall, typ: l.fnRet[name], line: pos.Line, call: &callExpr{name: name, args: args}}, nil
}

// callArgs lower 实参列表并做可赋值性检查（int → float 允许）。
func (fc *funcCtx) callArgs(c *lang.CallExpr, params []lang.Param) ([]*expr, error) {
	args := make([]*expr, 0, len(c.Args))
	for i, a := range c.Args {
		want := "?"
		if i < len(params) {
			want = params[i].Type
		}
		at := fc.typeOfAs(a, want)
		if want != "?" && !fc.assignable(at, want) {
			return nil, fc.l.errf(exprPos(a, c.Pos), "暂未支持向 %s 传递 %s 参数（需要 %s）", identifierName(c.Fn), at, want)
		}
		x, err := fc.exprAs(a, want)
		if err != nil {
			return nil, err
		}
		args = append(args, x)
	}
	return args, nil
}

// callSum lower 内建 sum(生成器, begin, stop[, step])。
func (fc *funcCtx) callSum(c *lang.CallExpr, pos lang.Pos) (*expr, error) {
	l := fc.l
	if len(c.Args) != 3 && len(c.Args) != 4 {
		return nil, l.errf(pos, "sum(generate, begin, stop[, step]) 需要 3 或 4 个参数，got %d", len(c.Args))
	}
	gid, ok := c.Args[0].(*lang.Ident)
	if !ok {
		return nil, l.errf(exprPos(c.Args[0], pos), "暂未支持该 sum 生成器（编译器要求生成器是具名函数）")
	}
	if _, isGen := l.generics[gid.Name]; isGen {
		return nil, l.errf(gid.Pos, "暂未支持泛型实例化 %s（解释器可用）", gid.Name)
	}
	gfn, ok := l.fns[gid.Name]
	if !ok {
		return nil, l.errf(gid.Pos, "未知生成器函数 %q（编译器只支持同一程序内定义且无泛型的函数）", gid.Name)
	}
	if len(gfn.Params) != 1 || gfn.Params[0].Type != "int" || l.fnRet[gid.Name] != "int" {
		return nil, l.errf(gid.Pos, "暂未支持生成器 %q（编译器要求 int %s(int)）", gid.Name, gid.Name)
	}
	args := []*expr{{kind: kIdent, typ: "function", s: gid.Name}}
	for _, a := range c.Args[1:] {
		if t := fc.typeOf(a); t != "int" && t != "?" {
			return nil, l.errf(exprPos(a, pos), "暂未支持 sum 的 %s 参数（需要 int）", t)
		}
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		args = append(args, x)
	}
	return &expr{kind: kCall, typ: "int", call: &callExpr{name: "sum", args: args}}, nil
}

// callMethod lower obj.m(...)。
func (fc *funcCtx) callMethod(c *lang.CallExpr, me *lang.MemberExpr) (*expr, error) {
	l := fc.l
	rt := fc.typeOf(me.X)
	// 内建 toString()
	if me.Name == "toString" {
		switch rt {
		case "int", "bool", "float", "String":
			if len(c.Args) != 0 {
				return nil, l.errf(me.Pos, "toString() 不接受参数")
			}
			recv, err := fc.expr(me.X)
			if err != nil {
				return nil, err
			}
			return &expr{kind: kToString, typ: "String", l: recv}, nil
		case "List<int>":
			return nil, l.errf(me.Pos, "暂未支持 List.toString()（后端未 lower 列表转字符串，解释器可用）")
		}
	}
	// List 内建方法
	if elem, ok := listElem(rt); ok && elem == "int" {
		id, ok := me.X.(*lang.Ident)
		if !ok {
			return nil, l.errf(me.Pos, "暂未支持该形式的方法调用（接收者必须是变量）")
		}
		args := make([]*expr, 0, len(c.Args))
		for _, a := range c.Args {
			x, err := fc.expr(a)
			if err != nil {
				return nil, err
			}
			args = append(args, x)
		}
		switch me.Name {
		case "size":
			if len(c.Args) != 0 {
				return nil, l.errf(me.Pos, "List.size() 不接受参数")
			}
			return &expr{kind: kMethod, typ: "int", method: &methodExpr{recv: &expr{kind: kIdent, typ: rt, s: id.Name}, name: "size"}}, nil
		case "get":
			if len(c.Args) != 1 {
				return nil, l.errf(me.Pos, "List.get(i) 需要 1 个参数")
			}
			if t := fc.typeOf(c.Args[0]); t != "int" && t != "?" {
				return nil, l.errf(exprPos(c.Args[0], me.Pos), "暂未支持 List.get 的 %s 下标（需要 int）", t)
			}
			return &expr{kind: kMethod, typ: "int", line: me.Pos.Line, method: &methodExpr{recv: &expr{kind: kIdent, typ: rt, s: id.Name}, name: "get", args: args}}, nil
		case "append":
			if len(c.Args) != 1 {
				return nil, l.errf(me.Pos, "List.append(v) 需要 1 个参数")
			}
			if t := fc.typeOf(c.Args[0]); !fc.assignable(t, "int") {
				return nil, l.errf(exprPos(c.Args[0], me.Pos), "暂未支持 List<int>.append 的 %s 参数（需要 int）", t)
			}
			return &expr{kind: kMethod, typ: "void", method: &methodExpr{recv: &expr{kind: kIdent, typ: rt, s: id.Name}, name: "append", args: args}}, nil
		case "head", "tail", "next", "reset", "appendAll", "__sort__":
			return nil, l.errf(me.Pos, "暂未支持 List.%s()（编译器后端未 lower，解释器可用）", me.Name)
		}
		return nil, l.errf(me.Pos, "暂未支持 List 方法 %q（编译器支持 size()/get(i)/append(v)）", me.Name)
	}
	// taskm 全局 / thread 变量 / Channel 变量
	if id, ok := me.X.(*lang.Ident); ok {
		if id.Name == "taskm" {
			return fc.taskmCall(c, me)
		}
		if t, found := fc.lookup(id.Name); found {
			switch t {
			case "thread":
				return fc.threadCall(c, me, &expr{kind: kIdent, typ: "thread", s: id.Name})
			case "Channel", "channel":
				return fc.channelCall(c, me, &expr{kind: kIdent, typ: t, s: id.Name})
			}
		}
	}
	// library FFI（外部符号：直接 call C 符号，参数按 C ABI 转换）
	if id, ok := me.X.(*lang.Ident); ok {
		if _, isLib := l.libs[id.Name]; isLib {
			return fc.libCall(c, me, id.Name)
		}
	}
	// 接口方法（vtable 分发）
	if l.isIfaceType(rt) {
		return fc.ifaceCall(c, me, rt)
	}
	// struct 实例方法
	if l.isStructType(rt) {
		if mi := l.lookupSelf(rt, me.Name); mi != nil {
			if len(mi.fn.Params)-1 != len(c.Args) {
				return nil, l.errf(me.Pos, "方法 %s.%s 需要 %d 个参数，got %d", rt, me.Name, len(mi.fn.Params)-1, len(c.Args))
			}
			imi, err := l.instantiateFor(mi, rt, c.Args, fc, me.Pos)
			if err != nil {
				return nil, err
			}
			return fc.methodCall(imi, rt, me.X, c.Args, me.Pos)
		}
		if l.hasMethod(rt, me.Name) {
			return nil, l.errf(me.Pos, "暂未支持静态方法以实例方式调用 %s.%s（正典写法 %s::%s(...)）", rt, me.Name, rt, me.Name)
		}
		return nil, l.errf(me.Pos, "struct %s 没有方法 %q（解释器在类型检查期报错）", rt, me.Name)
	}
	// 内建对象：taskm / library / io
	if id, ok := me.X.(*lang.Ident); ok {
		if id.Name == "taskm" {
			return nil, l.errf(me.Pos, "暂未支持 taskm 线程/通道（%s，解释器可用）", me.Name)
		}
		if _, isLib := l.libs[id.Name]; isLib {
			return nil, l.errf(me.Pos, "暂未支持 library FFI 调用 %s.%s（解释器可用）", id.Name, me.Name)
		}
		if fc.ioName != "" && id.Name == fc.ioName {
			return nil, l.errf(me.Pos, "io.%s 无返回值，不能用于表达式（请作为独立语句调用）", me.Name)
		}
	}
	return nil, l.errf(me.Pos, "暂未支持对 %s 调用方法 %s（解释器可用）", rt, me.Name)
}

// methodCall lower 一次实例方法调用（含运算符重载）。
func (fc *funcCtx) methodCall(mi *methodInfo, recvTyp string, recv lang.Expr, extra []lang.Expr, pos lang.Pos) (*expr, error) {
	l := fc.l
	rx, err := fc.expr(recv)
	if err != nil {
		return nil, err
	}
	ret := substType(mi.fn.Ret, mi.subst)
	if ret == "" {
		ret = "void"
	}
	args := make([]*expr, 0, len(extra))
	params := mi.fn.Params[1:]
	for i, a := range extra {
		at := fc.typeOf(a)
		want := "?"
		if i < len(params) {
			want = substType(params[i].Type, mi.subst)
		}
		if want != "?" && !fc.assignable(at, want) {
			return nil, l.errf(exprPos(a, pos), "暂未支持向 %s.%s 传递 %s 参数（需要 %s）", recvTyp, mi.fn.Name, at, want)
		}
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		args = append(args, x)
	}
	return &expr{kind: kMethod, typ: ret, line: pos.Line, method: &methodExpr{
		recv: rx, name: mi.fn.Name, args: args, sig: mi.irName, isSelf: true,
	}}, nil
}

// scopeCallCall lower T::m(...) / space::f(...) 调用形式。
func (fc *funcCtx) scopeCallCall(c *lang.CallExpr, sc *lang.ScopeCall) (*expr, error) {
	l := fc.l
	mi, err := l.staticMethodFor(sc.Scope, sc.Name, sc.Args, fc, sc.Pos)
	if err != nil {
		return nil, err
	}
	ret := substType(mi.fn.Ret, mi.subst)
	if ret == "" {
		ret = "void"
	}
	if l.nilFns[mi.irName] {
		return nil, l.errf(sc.Pos, "暂未支持在表达式中使用含 log 的函数 %s 的返回值（解释器返回 nil）", mi.irName)
	}
	args := make([]*expr, 0, len(sc.Args))
	for i, a := range sc.Args {
		at := fc.typeOf(a)
		want := "?"
		if i < len(mi.fn.Params) {
			want = substType(mi.fn.Params[i].Type, mi.subst)
		}
		if want != "?" && !fc.assignable(at, want) {
			return nil, l.errf(exprPos(a, sc.Pos), "暂未支持向 %s::%s 传递 %s 参数（需要 %s）", sc.Scope, sc.Name, at, want)
		}
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		args = append(args, x)
	}
	return &expr{kind: kCall, typ: ret, line: sc.Pos.Line, call: &callExpr{name: mi.irName, args: args}}, nil
}

// scopeCall lower 表达式位置的 space/T:: 调用。
func (fc *funcCtx) scopeCall(sc *lang.ScopeCall) (*expr, error) {
	return fc.scopeCallCall(&lang.CallExpr{Fn: sc, Pos: sc.Pos}, sc)
}

// member lower 成员访问 p.x / obj.m（非调用）。
func (fc *funcCtx) member(e *lang.MemberExpr) (*expr, error) {
	l := fc.l
	rt := fc.typeOf(e.X)
	if l.isStructType(rt) {
		ft, ok := l.fieldType(rt, e.Name)
		if !ok {
			return nil, l.errf(e.Pos, "struct %s 没有字段 %q", rt, e.Name)
		}
		recv, err := fc.expr(e.X)
		if err != nil {
			return nil, err
		}
		return &expr{kind: kField, typ: ft, field: &fieldExpr{recv: recv, name: e.Name, typ: rt}}, nil
	}
	if l.isIfaceType(rt) {
		return nil, l.errf(e.Pos, "暂未支持接口成员访问 %s.%s（dynamic 分发仅解释器可用）", rt, e.Name)
	}
	if id, ok := e.X.(*lang.Ident); ok {
		if _, isLib := l.libs[id.Name]; isLib {
			return nil, l.errf(e.Pos, "暂未支持 library 成员访问 %s.%s（FFI 仅解释器可用）", id.Name, e.Name)
		}
	}
	return nil, l.errf(e.Pos, "暂未支持成员访问 %s.%s（解释器可用）", rt, e.Name)
}

// structLit lower .{...} 字面量（target 为空时取 typecheck 填充的 Name）。
func (fc *funcCtx) structLit(e *lang.StructLit, target string) (*expr, error) {
	l := fc.l
	typ := e.Name
	if typ == "" {
		typ = target
	}
	base, targs := splitGeneric(typ)
	if sd0, ok := l.structs[base]; ok && len(sd0.TypeParams) > 0 && len(targs) == 0 {
		// 泛型 struct 字面量：typecheck 只填基名，实例类型由期望类型补全
		tb, ta := splitGeneric(target)
		if tb == base && len(ta) > 0 {
			typ = target
		} else {
			return nil, l.errf(e.Pos, "暂未支持无法推断类型实参的泛型 struct 字面量 .{...}（%s，解释器可用）", base)
		}
	}
	sd, sub := l.structSubst(typ)
	if sd == nil {
		return nil, l.errf(e.Pos, "暂未支持匿名 struct 字面量 .{...}（编译器要求有名 struct）")
	}
	l.ensureStructTy(typ)
	// 按声明顺序收集字段值（命名 + 位置两种写法）
	values := make([]*expr, len(sd.Members))
	seen := make([]bool, len(sd.Members))
	for i, f := range e.Fields {
		idx := i
		if f.Name != "" {
			idx = -1
			for j, m := range sd.Members {
				if m.Name == f.Name {
					idx = j
					break
				}
			}
			if idx < 0 {
				return nil, l.errf(e.Pos, "struct %s 没有字段 %q", typ, f.Name)
			}
		}
		if idx >= len(sd.Members) {
			return nil, l.errf(e.Pos, "struct %s 字面量字段过多", typ)
		}
		ft := substType(sd.Members[idx].Type, sub)
		vt := fc.typeOfAs(f.X, ft)
		if !fc.assignable(vt, ft) {
			return nil, l.errf(exprPos(f.X, e.Pos), "暂未支持用 %s 初始化字段 %s.%s（%s）", vt, typ, sd.Members[idx].Name, ft)
		}
		x, err := fc.exprAs(f.X, ft)
		if err != nil {
			return nil, err
		}
		values[idx] = x
		seen[idx] = true
	}
	for i := range values {
		if !seen[i] {
			values[i] = &expr{kind: kInt, typ: "int"} // 未给出的字段 = 零值（解释器同）
			if sd.Members[i].Type == "String" {
				values[i] = &expr{kind: kString, typ: "String"}
			} else if sd.Members[i].Type == "float" {
				values[i] = &expr{kind: kFloat, typ: "float"}
			} else if sd.Members[i].Type == "bool" {
				values[i] = &expr{kind: kBool, typ: "bool"}
			}
		}
	}
	return &expr{kind: kStructLit, typ: typ, sl: &structLit{typ: typ, values: values}}, nil
}

// ---------- 方法表查询 ----------

// hasMethod 判断类型（基类型）是否声明了该方法名。
func (l *lowerer) hasMethod(typ, name string) bool {
	base, _ := splitGeneric(typ)
	m, ok := l.methods[base]
	if !ok {
		return false
	}
	_, ok = m[name]
	return ok
}

// lookupSelf 查实例方法（有 self 首参）的**基类型**声明，未单态化。
func (l *lowerer) lookupSelf(typ, name string) *methodInfo {
	base, _ := splitGeneric(typ)
	if m, ok := l.methods[base]; ok {
		if mi, ok := m[name]; ok && mi.isSelf {
			return mi
		}
	}
	return nil
}

// lookupStatic 查静态方法的基类型声明，未单态化。
func (l *lowerer) lookupStatic(scope, name string) *methodInfo {
	if m, ok := l.methods[scope]; ok {
		if mi, ok := m[name]; ok && !mi.isSelf {
			return mi
		}
	}
	return nil
}

// inferStaticSubst 从静态方法调用实参推断 impl 类型参数。
func (l *lowerer) inferStaticSubst(mi *methodInfo, args []lang.Expr, fc *funcCtx) map[string]string {
	sub := map[string]string{}
	for i, p := range mi.fn.Params {
		if i >= len(args) || fc == nil {
			break
		}
		at := fc.typeOf(args[i])
		for _, tp := range mi.impl.TypeParams {
			if _, ok := sub[tp]; ok {
				continue
			}
			if strings.Contains(p.Type, tp) && at != "?" && at != "function" {
				sub[tp] = at
			}
		}
	}
	return sub
}

// staticMethodFor 查静态方法并完成单态化（T::m / space::f）。
func (l *lowerer) staticMethodFor(scope, name string, args []lang.Expr, fc *funcCtx, pos lang.Pos) (*methodInfo, error) {
	mi := l.lookupStatic(scope, name)
	if mi == nil {
		if _, isStruct := l.structs[scope]; isStruct {
			return nil, l.errf(pos, "暂未支持 struct %s 的静态方法 %s（该类型未声明 impl）", scope, name)
		}
		if _, ok := l.methods[scope]; !ok {
			return nil, l.errf(pos, "暂未支持 space 调用 %s::%s（解释器可用；编译器只支持同一程序内声明的 space/impl）", scope, name)
		}
		if l.hasMethod(scope, name) {
			return nil, l.errf(pos, "%s::%s 是实例方法（需要 self 接收者）", scope, name)
		}
		return nil, l.errf(pos, "暂未支持 %s::%s（该 space/类型未声明此方法）", scope, name)
	}
	return l.instantiateFor(mi, scope, args, fc, pos)
}

// methodRetType 方法在 receiver 类型下的返回类型（不实例化函数体，用于类型推断）。
func (l *lowerer) methodRetType(mi *methodInfo, recvType string) string {
	sub := mi.subst
	if len(mi.impl.TypeParams) > 0 {
		sub = l.recvSubst(mi, recvType)
	}
	ret := substType(mi.fn.Ret, sub)
	if ret == "" {
		return "void"
	}
	return ret
}

// callGeneric / lowerGeneric 由 lower_generic.go 实现（泛型单态化）。

// ---------- 静态类型推断 ----------

// assignable 判定 from 是否可赋给 to（与 internal/lang/typecheck.assignable 对齐）。
func (fc *funcCtx) assignable(from, to string) bool {
	if from == to || from == "?" || to == "?" || to == "" {
		return true
	}
	if from == "int" && to == "float" {
		return true
	}
	// 接口：具体 struct / 接口 → 接口（结构化满足由 typecheck 在赋值/传参处校验）
	if fc.l.isIfaceType(to) && (fc.l.isStructType(from) || fc.l.isIfaceType(from)) {
		return true
	}
	// null 只在指针/引用位置合法（编译器暂不支持 null 字面量）
	return false
}

func numLike(t string) bool { return t == "int" || t == "float" || t == "?" }

// numResult 返回数值运算结果类型（有 float 则 float）。
func numResult(a, b string) string {
	if a == "float" || b == "float" {
		return "float"
	}
	if a == "?" || b == "?" {
		return "?"
	}
	return "int"
}

// typeOf 尽力推断表达式的语言类型；无法判定返回 "?"。
func (fc *funcCtx) typeOf(x lang.Expr) string {
	l := fc.l
	switch e := x.(type) {
	case *lang.IntLit:
		return "int"
	case *lang.FloatLit:
		return "float"
	case *lang.StrLit:
		return "String"
	case *lang.BoolLit:
		return "bool"
	case *lang.NullLit:
		return "null"
	case *lang.Ident:
		if t, ok := fc.lookup(e.Name); ok {
			return t
		}
		if _, ok := l.fns[e.Name]; ok {
			return "function"
		}
		if _, ok := l.generics[e.Name]; ok {
			return "function"
		}
		return "?"
	case *lang.StructLit:
		if e.Name != "" {
			base, targs := splitGeneric(e.Name)
			if sd, ok := l.structs[base]; ok && len(sd.TypeParams) > 0 && len(targs) == 0 {
				if tb, ta := splitGeneric(fc.expectT); tb == base && len(ta) > 0 {
					return fc.expectT
				}
			}
			return e.Name
		}
		return "?"
	case *lang.MemberExpr:
		rt := fc.typeOf(e.X)
		if l.isStructType(rt) {
			if ft, ok := l.fieldType(rt, e.Name); ok {
				return ft
			}
		}
		return "?"
	case *lang.IndexExpr:
		if elem, ok := listElem(fc.typeOf(e.X)); ok {
			return elem
		}
		return "?"
	case *lang.BinOp:
		lt, rt := fc.typeOf(e.L), fc.typeOf(e.R)
		switch e.Op {
		case "+":
			if lt == "String" || rt == "String" {
				return "String"
			}
			if lt == rt && l.isStructType(lt) {
				if mi := l.lookupSelf(lt, "__add__"); mi != nil {
					return l.methodRetType(mi, lt)
				}
			}
			if numLike(lt) && numLike(rt) {
				return numResult(lt, rt)
			}
			return "?"
		case "-", "*", "/":
			if lt == rt && l.isStructType(lt) {
				if mi := l.lookupSelf(lt, opMethodFor(e.Op)); mi != nil {
					return l.methodRetType(mi, lt)
				}
			}
			if numLike(lt) && numLike(rt) {
				return numResult(lt, rt)
			}
			return "?"
		case "%":
			return "int"
		case "<<", ">>":
			return "int"
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||":
			return "bool"
		}
		return "?"
	case *lang.UnOp:
		switch e.Op {
		case "-":
			t := fc.typeOf(e.X)
			if l.isStructType(t) {
				if mi := l.lookupSelf(t, "__neg__"); mi != nil {
					return l.methodRetType(mi, t)
				}
			}
			return t
		case "!":
			return "bool"
		}
		return "?"
	case *lang.CallExpr:
		return fc.callType(e)
	case *lang.ScopeCall:
		mi := l.lookupStatic(e.Scope, e.Name)
		if mi == nil {
			return "?"
		}
		sub := mi.subst
		if len(mi.impl.TypeParams) > 0 {
			sub = l.inferStaticSubst(mi, e.Args, fc)
		}
		ret := substType(mi.fn.Ret, sub)
		if ret == "" {
			return "void"
		}
		return ret
	}
	return "?"
}

// callType 推断调用表达式的返回类型。
func (fc *funcCtx) callType(e *lang.CallExpr) string {
	l := fc.l
	if me, ok := e.Fn.(*lang.MemberExpr); ok {
		rt := fc.typeOf(me.X)
		if fc.ioName != "" {
			if id, ok := me.X.(*lang.Ident); ok && id.Name == fc.ioName {
				return "void"
			}
		}
		if me.Name == "toString" {
			switch rt {
			case "int", "bool", "float", "String":
				return "String"
			}
		}
		if elem, ok := listElem(rt); ok && elem == "int" {
			switch me.Name {
			case "size", "get":
				return "int"
			case "append":
				return "void"
			}
		}
		if id, ok := me.X.(*lang.Ident); ok {
			if _, isLib := l.libs[id.Name]; isLib {
				return l.libRetType(id.Name, me.Name)
			}
			if id.Name == "taskm" {
				switch me.Name {
				case "spawn":
					return "thread"
				case "channel":
					return "Channel"
				case "done":
					return "bool"
				case "block", "merge":
					return "void"
				}
				return "?"
			}
			if t, found := fc.lookup(id.Name); found {
				switch t {
				case "thread":
					switch me.Name {
					case "pid":
						return "int"
					case "merge":
						return "void"
					}
				case "Channel", "channel":
					switch me.Name {
					case "send":
						return "void"
					case "recv":
						return "int"
					}
				}
			}
		}
		if l.isIfaceType(rt) {
			for _, m := range l.ifaceMethodList(rt) {
				if m.name == me.Name {
					ret := m.ret
					if ret == "Self" {
						return rt
					}
					if ret == "" {
						return "void"
					}
					return ret
				}
			}
			return "?"
		}
		if l.isStructType(rt) {
			if mi := l.lookupSelf(rt, me.Name); mi != nil {
				return l.methodRetType(mi, rt)
			}
		}
		return "?"
	}
	id, ok := e.Fn.(*lang.Ident)
	if !ok {
		return "?"
	}
	switch id.Name {
	case "sum", "clock":
		return "int"
	}
	if _, isGen := l.generics[id.Name]; isGen {
		fn := l.generics[id.Name]
		sub := inferGenSubst(fn, e.Args, fc)
		ret := substType(fn.Ret, sub)
		if ret == "" {
			return "void"
		}
		return ret
	}
	if ret, ok := l.fnRet[id.Name]; ok {
		return ret
	}
	return "?"
}

// inferGenSubst 从实参推断泛型函数的类型参数（与 typecheck.inferGenSubst 同规则）。
func inferGenSubst(fn *lang.FuncDecl, args []lang.Expr, fc *funcCtx) map[string]string {
	sub := map[string]string{}
	for i, p := range fn.Params {
		if i >= len(args) {
			break
		}
		for _, tp := range fn.TypeParams {
			if _, ok := sub[tp]; ok {
				continue
			}
			if strings.Contains(p.Type, tp) {
				at := fc.typeOf(args[i])
				if at != "?" && at != "function" {
					sub[tp] = at
				}
			}
		}
	}
	return sub
}

// identifierName 取调用目标的标识符名（诊断用）。
func identifierName(fn lang.Expr) string {
	if id, ok := fn.(*lang.Ident); ok {
		return id.Name
	}
	if me, ok := fn.(*lang.MemberExpr); ok {
		return identifierName(me.X) + "." + me.Name
	}
	if sc, ok := fn.(*lang.ScopeCall); ok {
		return sc.Scope + "::" + sc.Name
	}
	return "?"
}

// ---------- library FFI ----------

// ffiLangType 把 library 声明里的 C 类型映射为 cgen 内部类型：
// int/char → int；bool → cbool（C int）；float/double → double；f32 → f32；String → String。
func (l *lowerer) ffiLangType(t string, pos lang.Pos) (string, error) {
	switch t {
	case "", "void":
		return "void", nil
	case "int", "char":
		return "int", nil
	case "bool":
		return "cbool", nil
	case "float", "double":
		return "double", nil
	case "f32":
		return "f32", nil
	case "String":
		return "String", nil
	}
	if strings.TrimSpace(t) == "pointer" {
		return "", l.errf(pos, "暂未支持 library FFI 的 pointer 参数/返回（不透明句柄仅解释器可用）")
	}
	if t == "long" {
		return "", l.errf(pos, "暂未支持 library FFI 的 long 参数/返回（解释器为 64 位，编译器未 lower）")
	}
	return "", l.errf(pos, "暂未支持 library FFI 的类型 %q", t)
}

// libRetType 返回 library 调用在语言层的可见类型。
func (l *lowerer) libRetType(libName, method string) string {
	ld, ok := l.libs[libName]
	if !ok {
		return "?"
	}
	for _, fn := range ld.Methods {
		if fn.Name != method {
			continue
		}
		t, err := l.ffiLangType(fn.Ret, fn.Pos)
		if err != nil {
			return "?"
		}
		switch t {
		case "cbool":
			return "bool"
		case "f32":
			return "float"
		case "double":
			return "float"
		case "void":
			return "void"
		}
		return t
	}
	return "?"
}

// libCall lower library FFI 调用 m.f(args)（LLVM declare + C ABI 直调）。
func (fc *funcCtx) libCall(c *lang.CallExpr, me *lang.MemberExpr, libName string) (*expr, error) {
	l := fc.l
	ld := l.libs[libName]
	var fn *lang.Func
	for _, f := range ld.Methods {
		if f.Name == me.Name {
			fn = f
			break
		}
	}
	if fn == nil {
		return nil, l.errf(me.Pos, "library %s 没有声明函数 %q", libName, me.Name)
	}
	if len(c.Args) != len(fn.Params) {
		return nil, l.errf(me.Pos, "library 函数 %s.%s 需要 %d 个参数，got %d", libName, me.Name, len(fn.Params), len(c.Args))
	}
	ret, err := l.ffiLangType(fn.Ret, fn.Pos)
	if err != nil {
		return nil, err
	}
	params := make([]funcParam, 0, len(fn.Params))
	for i, p := range fn.Params {
		pt, err := l.ffiLangType(p.Type, p.Pos)
		if err != nil {
			return nil, err
		}
		if pt == "void" {
			return nil, l.errf(p.Pos, "library 函数参数不能是 void")
		}
		if !fc.ffiArgOK(pt) {
			return nil, l.errf(p.Pos, "暂未支持 library FFI 参数类型 %q（编译器支持 int/bool/f32/float/double/String）", p.Type)
		}
		params = append(params, funcParam{name: p.Name, typ: pt})
		// 实参可赋值性：int → double/f32、bool → cbool、float → f32 允许
		at := fc.typeOf(c.Args[i])
		if !fc.ffiAssignable(at, pt) {
			return nil, l.errf(exprPos(c.Args[i], me.Pos), "暂未支持向 %s.%s 传递 %s 参数（需要 %s）", libName, me.Name, at, p.Type)
		}
	}
	args := make([]*expr, 0, len(c.Args))
	for _, a := range c.Args {
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		args = append(args, x)
	}
	l.addExtern(fn.Name, ld.Lib, params, ret)
	vis := ret
	switch ret {
	case "cbool":
		vis = "bool"
	case "f32":
		vis = "float"
	case "double":
		vis = "float"
	}
	if vis == "void" {
		return &expr{kind: kCall, typ: "void", call: &callExpr{name: fn.Name, args: args}}, nil
	}
	return &expr{kind: kCall, typ: vis, line: me.Pos.Line, call: &callExpr{name: fn.Name, args: args}}, nil
}

// ffiArgOK 判断 FFI 形参类型是否可 lower。
func (fc *funcCtx) ffiArgOK(t string) bool {
	switch t {
	case "int", "cbool", "f32", "double", "String":
		return true
	}
	return false
}

// ffiAssignable 判断语言类型实参能否传给 FFI 形参。
func (fc *funcCtx) ffiAssignable(from, to string) bool {
	if from == "?" {
		return true
	}
	switch to {
	case "int":
		return from == "int" || from == "long"
	case "cbool":
		return from == "bool" || from == "int"
	case "f32", "double":
		return from == "int" || from == "float" || from == "double" || from == "f32"
	case "String":
		return from == "String"
	}
	return false
}

// addExtern 登记外部符号（去重）。
func (l *lowerer) addExtern(name, lib string, params []funcParam, ret string) {
	if _, ok := l.extSeen[name]; ok {
		return
	}
	l.extSeen[name] = true
	l.out.externs = append(l.out.externs, externDef{name: name, params: params, ret: ret, lib: lib})
}

// ---------- catch 变量使用检测 ----------

func blockUsesIdent(b *lang.Block, name string) bool {
	if b == nil {
		return false
	}
	for _, s := range b.Stmts {
		if stmtUsesIdent(s, name) {
			return true
		}
	}
	return false
}

func stmtUsesIdent(s lang.Stmt, name string) bool {
	switch st := s.(type) {
	case *lang.ExprStmt:
		return exprUsesIdent(st.X, name)
	case *lang.LogStmt:
		return exprUsesIdent(st.X, name)
	case *lang.DeleteStmt:
		return exprUsesIdent(st.X, name)
	case *lang.ReturnStmt:
		return st.X != nil && exprUsesIdent(st.X, name)
	case *lang.DeclStmt:
		return st.Init != nil && exprUsesIdent(st.Init, name)
	case *lang.AssignStmt:
		return exprUsesIdent(st.Target, name) || exprUsesIdent(st.X, name)
	case *lang.IfStmt:
		return exprUsesIdent(st.Cond, name) || blockUsesIdent(st.Then, name) || blockUsesIdent(st.Else, name)
	case *lang.WhileStmt:
		return exprUsesIdent(st.Cond, name) || blockUsesIdent(st.Body, name)
	case *lang.ForStmt:
		return exprUsesIdent(st.Iter, name) || blockUsesIdent(st.Body, name)
	case *lang.ForCStmt:
		return stmtUsesIdent(st.Init, name) || exprUsesIdent(st.Cond, name) ||
			(st.Step != nil && stmtUsesIdent(st.Step, name)) || blockUsesIdent(st.Body, name)
	case *lang.TryStmt:
		return blockUsesIdent(st.Try, name) || blockUsesIdent(st.Catch, name)
	}
	return false
}

func exprUsesIdent(x lang.Expr, name string) bool {
	if x == nil {
		return false
	}
	switch e := x.(type) {
	case *lang.Ident:
		return e.Name == name
	case *lang.BinOp:
		return exprUsesIdent(e.L, name) || exprUsesIdent(e.R, name)
	case *lang.UnOp:
		return exprUsesIdent(e.X, name)
	case *lang.CallExpr:
		if exprUsesIdent(e.Fn, name) {
			return true
		}
		for _, a := range e.Args {
			if exprUsesIdent(a, name) {
				return true
			}
		}
		return false
	case *lang.MemberExpr:
		return exprUsesIdent(e.X, name)
	case *lang.IndexExpr:
		return exprUsesIdent(e.X, name) || exprUsesIdent(e.Idx, name)
	case *lang.ListLit:
		for _, it := range e.Items {
			if exprUsesIdent(it, name) {
				return true
			}
		}
		return false
	case *lang.StructLit:
		for _, f := range e.Fields {
			if exprUsesIdent(f.X, name) {
				return true
			}
		}
		return false
	case *lang.NewExpr:
		return e.Size != nil && exprUsesIdent(e.Size, name)
	case *lang.ScopeCall:
		for _, a := range e.Args {
			if exprUsesIdent(a, name) {
				return true
			}
		}
		return false
	}
	return false
}
