// Package cgen 将 QuarkLang 编译为 LLVM IR（跨系统编译器，LLVM 后端）。
//
// 前端只有一套语法源：internal/lang 的正典 AST（与解释器同源）。本包不再自带
// 词法/语法分析，流程为：
//
//	lang.CompileWithImports(src, filename)  // 词法/语法/类型检查 + import 递归合并
//	  → lowerProgram（lower.go）            // 正典 AST → cgen IR（不支持即报错）
//	  → emitter（本文件）                    // cgen IR → LLVM IR
//
// 支持与不支持的构造清单见 lower.go 顶部说明；后端未 lower 的构造会返回带
// 源码位置的明确错误，绝不静默错编。
//
// 值模型（与解释器语义对齐）：
//   - int → i32，bool → i1，float → double，String → i8*
//   - List<T> / struct 都是**引用**（堆对象指针），与解释器的 *List / *Struct
//     别名语义一致：赋值/传参复制的是引用，字段/元素修改对所有别名可见
//   - 结构体字段按声明顺序布局（LLVM 字面/命名结构体），零值 = calloc 清零
package cgen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"quarklang/internal/lang"
)

// Transpile 把 QuarkLang 源码编译为 LLVM IR。
//
// src 需已完成编译器侧宏展开（见 compiler/main.go 的 expandMacros）；
// filename 用于 import 解析（同目录 .qk/.qlib，与解释器同一套递归合并语义）
// 与诊断定位，可为 ""。
func Transpile(src, filename string) (string, error) {
	prog, err := lang.CompileWithImports(src, filename)
	if err != nil {
		return "", err
	}
	lp, err := lowerProgram(prog, filename, src)
	if err != nil {
		return "", err
	}
	e := newEmitter(lp)
	return e.emitProgram(lp), nil
}

// ---------- cgen IR ----------

type exprKind int

const (
	kInt exprKind = iota
	kFloat
	kString
	kBool
	kIdent
	kBin
	kCmp
	kAndOr
	kCall
	kList
	kIndex
	kMethod
	kStructLit
	kField
	kToString
	kRunner // taskm.merge 的 runner 函数引用（i8* 函数指针）
)

// expr 是 cgen IR 表达式。typ 由 lowering 填入语言类型（"?" = 未判定）。
type expr struct {
	kind exprKind
	i    int64
	f    float64
	b    bool
	s    string
	op   string
	typ  string
	l, r *expr

	call   *callExpr   // kCall
	lst    *listLit    // kList
	idx    *indexExpr  // kIndex
	method *methodExpr // kMethod
	sl     *structLit  // kStructLit
	field  *fieldExpr  // kField

	sc     strConst // 预注册的字符串常量（kString）
	strcat bool     // kBin "+" 且为 String 拼接（有 ql_strcat 副作用）
	line   int      // 运行期错误定位（除零等）

	ifaceBox string // 非空：把具体 struct 值装箱成接口值（值为 vtable 符号）
}

type indexExpr struct {
	name string // List 变量名
	i    *expr
}

type methodExpr struct {
	recv   *expr
	name   string // 方法名
	args   []*expr
	sig    string // IR 函数名（lowering 解析后填入）
	isSelf bool   // 实例方法（需要 receiver 实参）

	iface    string   // 非空：接口方法调用（vtable 分发）
	idx      int      // 方法在接口派发表中的槽位
	ifaceSig *funcSig // 调用签名（Self 形参以 "Self" 标记）
}

type fieldExpr struct {
	recv *expr
	name string
	typ  string // 接收者的 struct 类型名
}

type structLit struct {
	typ    string // 目标 struct 类型名
	values []*expr
}

type stmt interface{}

type exprStmt struct{ x *expr }
type returnStmt struct{ x *expr }
type printlnStmt struct{ args []*expr }
type printStmt struct{ args []*expr }

type declStmt struct {
	name string
	typ  string
	init *expr
}

type assignStmt struct {
	name string
	x    *expr
}

type indexAssignStmt struct {
	name string
	idx  *expr
	x    *expr
}

type fieldAssignStmt struct {
	recv  *expr
	field string
	x     *expr
}

type ifStmt struct {
	cond *expr
	then []stmt
	els  []stmt
}

type whileStmt struct {
	cond *expr
	body []stmt
}

type forStmt struct {
	init stmt
	cond *expr
	step stmt
	body []stmt
}

type forInStmt struct {
	name string
	typ  string
	list string
	body []stmt
}

type breakStmt struct{}

type tryStmt struct {
	then  []stmt
	catch []stmt
}

type deleteStmt struct{ name string }

type logStmt struct{ x *expr }

type listLit struct {
	items []*expr
}

type callExpr struct {
	name string // IR 函数名
	args []*expr
}

type funcParam struct {
	name string
	typ  string
}

type funcDef struct {
	name      string // IR 函数名（已修饰：Point_sum / math_max / …）
	params    []funcParam
	ret       string
	body      []stmt
	selfTyp   string // impl 方法 receiver 的 struct 类型名
	selfParam string // receiver 参数名（self）
}

type structDef struct {
	name       string
	fields     []string
	fieldTypes []string
}

// lowered 是正典 AST 降到 cgen IR 后的程序。
type lowered struct {
	funcs     []*funcDef // 非 main 函数
	mainStmts []stmt     // fn main 的函数体
	structs   []structDef
	vtables   []*vtableDef
	ifaces    []string
	externs   []externDef
	runners   []runnerDef
}

// sortVTables 让 vtable 发射顺序稳定（便于回归对比）。
func (lp *lowered) sortVTables() {
	sort.Slice(lp.vtables, func(i, j int) bool { return lp.vtables[i].sym < lp.vtables[j].sym })
}

// runnerDef 是 taskm.merge 的执行器（qthreads.c 以 void (*)(int) 调用）。
type runnerDef struct {
	fn     string // 目标函数 IR 名
	hasArg bool   // 目标函数是否有 1 个 int 参数
}

// emitRunners 生成 taskm.merge 的 runner：调目标函数并丢弃返回值。
func emitRunners(rs []runnerDef) string {
	if len(rs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, r := range rs {
		sig := ""
		call := "call void @" + r.fn + "()"
		if r.hasArg {
			sig = "i32 %a"
			call = "call void @" + r.fn + "(i32 %a)"
		}
		fmt.Fprintf(&sb, "define void @runner_%d(%s) {\nentry:\n  %s\n  ret void\n}\n", i, sig, call)
	}
	return sb.String()
}

// externDef 是 library FFI 外部符号声明（LLVM declare + C ABI 调用）。
type externDef struct {
	name   string // C 符号名
	params []funcParam
	ret    string // cgen 类型（cbool/f32/double/…）
	lib    string // 链接库名（qkc-link 标记）
}

// vtableDef 是 (具体类型, 接口) 的派发表。
type vtableDef struct {
	sym    string // @vt$A$Iface
	iface  string
	typ    string
	thunks []*ifaceThunk
}

// ifaceThunk 把接口调用还原为具体类型方法调用。
type ifaceThunk struct {
	iface   string
	typ     string
	method  string
	irName  string   // 具体方法 IR 名
	ret     string   // 具体返回类型（"void" = 无返回）
	params  []string // 具体参数类型（不含 self）
	selfIdx []int    // 接口声明为 Self 的参数下标（thunk 收 i8*）
	boxRet  bool     // 接口返回 Self → thunk 装箱后返回 %Iface
	vtSym   string   // 装箱用的 vtable 符号
}

// ---------- LLVM IR 发射器 ----------

type strConst struct {
	name string
	size int
}

// varSlot 是变量的存储：alloca 槽（reg 是槽指针）或 SSA 直通/参数（reg 是值）。
type varSlot struct {
	reg    string
	typ    string // 语言类型
	param  bool   // 函数参数：值寄存器
	direct bool   // SSA 直通（单赋值标量）
}

type structField struct {
	name string
	typ  string
}

type funcSig struct {
	name   string
	params []funcParam
	ret    string
}

type emitter struct {
	fnMeta map[string]*fnMeta

	types   strings.Builder // %Point = type { ... } / %List = type { ... }
	globals strings.Builder // 字符串常量
	decls   strings.Builder // declare（外部符号）
	helpers strings.Builder // 运行期助手（按需）
	bodies  strings.Builder // 函数定义

	body strings.Builder // 当前函数体
	cur  string

	blockCount int
	regCount   int
	strs       map[string]strConst
	strCount   int

	vars    map[string]varSlot
	structs map[string][]structField
	sigs    map[string]*funcSig
	curRet  string
	curTry  string
	breaks  []string

	funcReturned bool
	term         bool // 当前基本块已由 br/ret/unreachable 封闭
	assigned     map[string]bool

	emitted        map[string]bool
	hasList        bool
	hasEmpty       bool
	hasIface       bool
	ifaces         map[string]bool
	vtables        []*vtableDef
	needIntToStr   bool
	needFloatToStr bool
	needStrCmp     bool
	needPanic      bool
}

func newEmitter(lp *lowered) *emitter {
	e := &emitter{
		fnMeta:  programMeta(lp.funcs, lp.mainStmts),
		vars:    map[string]varSlot{},
		structs: map[string][]structField{},
		sigs:    map[string]*funcSig{},
		strs:    map[string]strConst{},
	}
	for _, sd := range lp.structs {
		fs := make([]structField, 0, len(sd.fields))
		for i, n := range sd.fields {
			t := "int"
			if i < len(sd.fieldTypes) {
				t = sd.fieldTypes[i]
			}
			fs = append(fs, structField{name: n, typ: t})
		}
		e.structs[sd.name] = fs
	}
	for _, fd := range lp.funcs {
		e.sigs[fd.name] = &funcSig{name: fd.name, params: fd.params, ret: fd.ret}
	}
	e.ifaces = map[string]bool{}
	for _, n := range lp.ifaces {
		e.ifaces[n] = true
	}
	e.vtables = lp.vtables
	for _, ed := range lp.externs {
		e.sigs[ed.name] = &funcSig{name: ed.name, params: ed.params, ret: ed.ret}
	}
	// taskm 运行时（qthreads.c）：pid = 小整数句柄
	e.sigs["ql_spawn"] = &funcSig{name: "ql_spawn", ret: "int"}
	e.sigs["ql_channel_new"] = &funcSig{name: "ql_channel_new", params: []funcParam{{typ: "int"}}, ret: "Channel"}
	e.sigs["ql_block"] = &funcSig{name: "ql_block", params: []funcParam{{typ: "int"}}, ret: "void"}
	e.sigs["ql_done"] = &funcSig{name: "ql_done", params: []funcParam{{typ: "int"}}, ret: "cbool"}
	e.sigs["ql_merge"] = &funcSig{name: "ql_merge", params: []funcParam{{typ: "int"}, {typ: "runner"}, {typ: "int"}}, ret: "void"}
	e.sigs["ql_send"] = &funcSig{name: "ql_send", params: []funcParam{{typ: "Channel"}, {typ: "int"}}, ret: "int"}
	e.sigs["ql_recv"] = &funcSig{name: "ql_recv", params: []funcParam{{typ: "Channel"}}, ret: "int"}
	return e
}

// builtinDecls 是发射器固定声明的符号（FFI 重复声明会报 invalid redefinition）。
var builtinDecls = map[string]bool{
	"printf": true, "malloc": true, "calloc": true, "free": true, "realloc": true,
	"gettimeofday": true, "ql_strcat": true, "snprintf": true, "strcmp": true,
	"strtod": true, "write": true, "exit": true,
}

// linkerFlag 把 library 名映射为链接参数（裸名 → -lX；路径/带点 → 原样）。
func linkerFlag(lib string) string {
	if strings.ContainsAny(lib, "/.") {
		return lib
	}
	switch lib {
	case "c", "m", "dl", "pthread", "rt", "stdc++":
		return "-l" + lib
	}
	return "-l" + lib
}

// ensureIface 发射接口值类型：{ i8* data, i8** vt }。
func (e *emitter) ensureIface() {
	if e.hasIface {
		return
	}
	e.hasIface = true
	e.types.WriteString("%Iface = type { i8*, i8** }\n")
}

func (e *emitter) emitInstr(f string, args ...interface{}) {
	s := fmt.Sprintf(f, args...)
	e.body.WriteString("  " + s + "\n")
	// 终结指令（br/ret/unreachable）后当前基本块已封闭：后续语句不可达，
	// 且同一块内不得再出现第二条终结指令（否则 LLVM 会插入匿名块，SSA 编号错乱）。
	if strings.HasPrefix(s, "br ") || strings.HasPrefix(s, "ret ") || s == "unreachable" {
		e.term = true
	}
}

func (e *emitter) newBlock() string {
	e.blockCount++
	b := strconv.AppendInt(nil, int64(e.blockCount), 10)
	return "b" + string(b)
}

// setBlock 切换到命名基本块（首次切换写出块标签）。
func (e *emitter) setBlock(name string) {
	if e.cur != name {
		e.body.WriteString("\n" + name + ":\n")
		e.cur = name
	}
	e.term = false
}

func (e *emitter) newReg() string {
	e.regCount++
	b := strconv.AppendInt(nil, int64(e.regCount), 10)
	return "%" + string(b)
}

func (e *emitter) strConst(s string) strConst {
	if e.strs == nil {
		e.strs = map[string]strConst{}
	}
	if c, ok := e.strs[s]; ok {
		return c
	}
	e.strCount++
	c := strConst{name: fmt.Sprintf("@.str%d", e.strCount), size: len(s) + 1}
	e.strs[s] = c
	fmt.Fprintf(&e.globals, "%s = private unnamed_addr constant [%d x i8] c\"%s\", align 1\n",
		c.name, c.size, llvmEscape([]byte(s)))
	return c
}

func (e *emitter) i8Ptr(c strConst) string {
	r := e.newReg()
	e.emitInstr("%s = getelementptr inbounds [%d x i8], [%d x i8]* %s, i64 0, i64 0",
		r, c.size, c.size, c.name)
	return r
}

// llvmEscape 把字节序列转义为 LLVM IR 字符串体（可打印字符原样，其余 \XX，含结尾 NUL）。
func llvmEscape(b []byte) string {
	var sb strings.Builder
	for _, x := range append(b, 0) {
		switch {
		case x >= 0x20 && x <= 0x7e && x != '"' && x != '\\':
			sb.WriteByte(x)
		default:
			fmt.Fprintf(&sb, "\\%02X", x)
		}
	}
	return sb.String()
}

// ---------- 类型映射 ----------

// tyName 把语言类型名转成 LLVM 标识符安全的类型名（泛型实例：Box<int> → Box_int_）。
func tyName(t string) string {
	var sb strings.Builder
	for _, r := range t {
		switch r {
		case '<', '>', ',', ' ', '[', ']', '&':
			sb.WriteByte('_')
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ensureStruct 保证 struct 类型的 LLVM 定义已发射（按需、幂等）。
func (e *emitter) ensureStruct(name string) {
	fields, ok := e.structs[name]
	if !ok || e.emitted[name] {
		return
	}
	e.emitted[name] = true
	var sb strings.Builder
	fmt.Fprintf(&sb, "%%%s = type { ", tyName(name))
	for i, f := range fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(e.ir(f.typ))
	}
	sb.WriteString(" }\n")
	e.types.WriteString(sb.String())
}

// ensureList 保证 List 类型的 LLVM 定义已发射。
func (e *emitter) ensureList() {
	if e.hasList {
		return
	}
	e.hasList = true
	e.types.WriteString("%List = type { i32*, i32, i32 }\n") // buf, head, tail
}

// irElem 返回聚合类型的元素类型（GEP 的源类型）：%Point* → %Point；标量原样。
func (e *emitter) irElem(t string) string {
	p := e.ir(t)
	if strings.HasPrefix(p, "%") {
		return strings.TrimSuffix(p, "*")
	}
	return p
}

// ir 返回语言类型的 LLVM 表示。
func (e *emitter) ir(t string) string {
	switch t {
	case "int":
		return "i32"
	case "bool":
		return "i1"
	case "float":
		return "double"
	case "String":
		return "i8*"
	case "void", "":
		return "void"
	}
	if e.ifaces[t] {
		e.ensureIface()
		return "%Iface" // 接口是值类型（data + vtable 两个指针）
	}
	// library FFI 的 C ABI 类型（f32 = 单精度 float；double = 双精度；cbool = C int；long = i64）
	// 以及 taskm 运行时类型（thread = pid 句柄 i32；Channel = i8*；runner = 函数指针 i8*）
	switch t {
	case "f32":
		return "float"
	case "double":
		return "double"
	case "cbool":
		return "i32"
	case "long":
		return "i64"
	case "pointer":
		return "i8*"
	case "thread":
		return "i32"
	case "Channel", "channel":
		return "i8*"
	case "runner":
		return "i8*"
	}
	if t == "List<int>" || t == "List<String>" || t == "List<float>" || t == "List<bool>" {
		e.ensureList()
		return "%List*"
	}
	if _, ok := e.structs[t]; ok {
		e.ensureStruct(t)
		return "%" + tyName(t) + "*"
	}
	return "i32"
}

// alignOf 返回类型的对齐字节数。
func (e *emitter) alignOf(t string) int {
	switch t {
	case "bool":
		return 1
	case "thread":
		return 4
	case "float", "String", "List<int>", "List<String>", "List<float>", "List<bool>":
		return 8
	}
	if e.ifaces[t] {
		return 8
	}
	return 4
}

// emptyString 返回空串指针（String 的零值：解释器零值 String 打印为 ""，
// 不能是 null —— printf("%s", NULL) 会打印 "(null)"，strcmp 会崩溃）。
func (e *emitter) emptyString() string {
	if !e.hasEmpty {
		e.hasEmpty = true
		e.globals.WriteString("@.empty = private unnamed_addr constant [1 x i8] zeroinitializer, align 1\n")
	}
	r := e.newReg()
	e.emitInstr("%s = getelementptr inbounds [1 x i8], [1 x i8]* @.empty, i64 0, i64 0", r)
	return r
}

// zeroOf 返回语言类型的零值（按 LLVM 类型）。
func (e *emitter) zeroOf(t string) string {
	switch t {
	case "int", "thread":
		return "0"
	case "bool":
		return "false"
	case "float":
		return "0.0"
	case "String":
		return e.emptyString()
	}
	if e.ifaces[t] {
		return "zeroinitializer"
	}
	return "null" // List / struct：引用零值 = null
}

// zeroStructFields 把刚分配（calloc 清零）的 struct 的引用型字段初始化为零值对象：
// String → 空串；嵌套 struct → 递归分配零值实例（解释器零值 struct 的字段是有效对象）。
func (e *emitter) zeroStructFields(ptr, typ string, depth int) {
	if depth > 16 {
		return
	}
	fields := e.structs[typ]
	for i, f := range fields {
		if f.typ != "String" && !e.isStruct(f.typ) {
			continue
		}
		g := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %s, %s* %s, i32 0, i32 %d", g, e.irElem(typ), e.irElem(typ), ptr, i)
		if f.typ == "String" {
			v := e.emptyString()
			e.emitInstr("store i8* %s, i8** %s", v, g)
			continue
		}
		obj := e.structAlloc(f.typ)
		v := e.newReg()
		e.emitInstr("%s = bitcast i8* %s to %s", v, obj, e.ir(f.typ))
		e.zeroStructFields(v, f.typ, depth+1)
		e.emitInstr("store %s %s, %s* %s", e.ir(f.typ), v, e.ir(f.typ), g)
	}
}

// ---------- 程序组装 ----------

func (e *emitter) emitProgram(lp *lowered) string {
	e.emitted = map[string]bool{}
	e.decls.WriteString("declare i32 @printf(i8* noundef, ...)\n")
	e.decls.WriteString("declare i8* @malloc(i64)\n")
	e.decls.WriteString("declare i8* @calloc(i64, i64)\n")
	e.decls.WriteString("declare void @free(i8*)\n")
	e.decls.WriteString("declare i8* @realloc(i8*, i64)\n")
	e.decls.WriteString("declare i32 @gettimeofday({ i64, i64 }*, i8*)\n")
	e.decls.WriteString("declare i8* @ql_strcat(i8*, i8*)\n")
	e.decls.WriteString("declare i32 @snprintf(i8*, i64, i8*, ...)\n")
	e.decls.WriteString("declare i32 @strcmp(i8*, i8*)\n")
	e.decls.WriteString("declare double @strtod(i8*, i8**)\n")
	e.decls.WriteString("declare i64 @write(i32, i8*, i64)\n")
	e.decls.WriteString("declare void @exit(i32)\n")
	e.decls.WriteString("declare i32 @ql_spawn()\n")
	e.decls.WriteString("declare void @ql_merge(i32, i8*, i32)\n")
	e.decls.WriteString("declare void @ql_block(i32)\n")
	e.decls.WriteString("declare i32 @ql_done(i32)\n")
	e.decls.WriteString("declare i8* @ql_channel_new(i32)\n")
	e.decls.WriteString("declare i32 @ql_send(i8*, i32)\n")
	e.decls.WriteString("declare i32 @ql_recv(i8*)\n\n")

	// library FFI：外部符号声明 + 链接库标记（main.go 解析 ; qkc-link:）
	seenExt := map[string]bool{}
	for _, ed := range lp.externs {
		if seenExt[ed.name] || builtinDecls[ed.name] {
			continue
		}
		seenExt[ed.name] = true
		fmt.Fprintf(&e.decls, "declare %s @%s(", e.ir(ed.ret), ed.name)
		for i, p := range ed.params {
			if i > 0 {
				e.decls.WriteString(", ")
			}
			e.decls.WriteString(e.ir(p.typ))
		}
		e.decls.WriteString(")\n")
	}
	var link strings.Builder
	libs := map[string]bool{}
	for _, ed := range lp.externs {
		if ed.lib == "" || libs[ed.lib] {
			continue
		}
		libs[ed.lib] = true
		link.WriteString("; qkc-link: " + linkerFlag(ed.lib) + "\n")
	}

	e.vars = map[string]varSlot{}
	// 按依赖顺序预发射全部 struct 类型定义（lower 已保证被依赖者在前）
	for _, sd := range lp.structs {
		e.ensureStruct(sd.name)
	}
	// 预注册全部字符串常量
	e.strConst("true")
	e.strConst("false")
	for _, fd := range lp.funcs {
		for _, s := range fd.body {
			e.preRegisterStmt(s)
		}
	}
	for _, s := range lp.mainStmts {
		e.preRegisterStmt(s)
	}
	// 预扫描被赋值变量（决定 SSA 直通）
	e.assigned = map[string]bool{}
	for _, fd := range lp.funcs {
		scanAssigned(fd.body, e.assigned)
	}
	scanAssigned(lp.mainStmts, e.assigned)

	// 先生成全部函数体（期间动态追加字符串常量/类型定义），最后统一组装
	for _, fd := range lp.funcs {
		if fd.name != "main" {
			e.bodies.WriteString(e.emitFunc(fd))
		}
	}
	e.cur = "entry"
	e.curRet = "int" // main 的 LLVM 返回类型是 i32（log/末尾统一 ret i32 0）
	e.body.Reset()
	e.funcReturned = false
	e.term = false
	e.breaks = nil
	e.curTry = ""
	e.body.WriteString("entry:\n")
	e.emitBlock(lp.mainStmts)
	if !e.term {
		e.emitInstr("ret i32 0")
	}
	e.bodies.WriteString("define i32 @main()" + fnAttrs("main", e.fnMeta) + " {\n" + e.body.String() + "}\n")

	helpers := ""
	if e.needIntToStr {
		helpers += intToStrHelper
	}
	if e.needFloatToStr {
		helpers += floatToStrHelper
	}
	if e.needPanic {
		helpers += panicHelper
	}
	return link.String() + e.types.String() + e.globals.String() + e.decls.String() +
		helpers + e.bodies.String() + e.emitVtables(e.vtables) + emitRunners(lp.runners)
}

// emitFunc 生成单个非 main 函数的定义。
func (e *emitter) emitFunc(fd *funcDef) string {
	// 注意：regCount 不重置——LLVM 寄存器编号是模块全局递增的
	e.blockCount = 0
	e.body.Reset()
	e.vars = map[string]varSlot{}
	e.curRet = fd.ret
	e.curTry = ""
	e.breaks = nil
	retTy := e.ir(fd.ret)
	var sig strings.Builder
	sig.WriteString("define " + retTy + " @" + fd.name + "(")
	first := true
	addParam := func(typ, reg string) {
		if !first {
			sig.WriteString(", ")
		}
		first = false
		sig.WriteString(typ + " noundef " + reg)
	}
	if fd.selfTyp != "" {
		addParam(e.ir(fd.selfTyp), "%self")
		e.vars[fd.selfParam] = varSlot{reg: "%self", typ: fd.selfTyp, param: true}
	}
	paramRegs := make([]string, 0, len(fd.params))
	for i, p := range fd.params {
		reg := fmt.Sprintf("%%p%d", i)
		addParam(e.ir(p.typ), reg)
		paramRegs = append(paramRegs, reg)
		e.vars[p.name] = varSlot{reg: reg, typ: p.typ, param: true}
	}
	sig.WriteString(")" + fnAttrs(fd.name, e.fnMeta) + "\n")
	e.cur = "entry"
	e.body.WriteString("entry:\n")
	// 被赋值的参数提升为 alloca 槽：LLVM 的 SSA 参数寄存器不可写
	promote := func(name, typ, val string) {
		slot := e.newReg()
		llt := e.ir(typ)
		e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(typ))
		e.emitInstr("store %s %s, %s* %s", llt, val, llt, slot)
		e.vars[name] = varSlot{reg: slot, typ: typ}
	}
	if fd.selfTyp != "" && e.assigned[fd.selfParam] {
		promote(fd.selfParam, fd.selfTyp, "%self")
	}
	for i, p := range fd.params {
		if e.assigned[p.name] {
			promote(p.name, p.typ, paramRegs[i])
		}
	}
	e.funcReturned = false
	e.term = false
	e.emitBlock(fd.body)
	if !e.funcReturned && !e.term {
		switch fd.ret {
		case "int":
			e.emitInstr("ret i32 0")
		case "bool":
			e.emitInstr("ret i1 false")
		case "float":
			e.emitInstr("ret double 0.0")
		case "String":
			e.emitInstr("ret i8* %s", e.emptyString())
		default:
			e.emitInstr("ret void")
		}
	}
	return sig.String() + "{\n" + e.body.String() + "}\n"
}

// ---------- 语句 ----------

func (e *emitter) preRegisterStmt(s stmt) {
	switch st := s.(type) {
	case *tryStmt:
		for _, x := range st.then {
			e.preRegisterStmt(x)
		}
		for _, x := range st.catch {
			e.preRegisterStmt(x)
		}
	case *printlnStmt:
		for _, a := range st.args {
			e.preRegister(a)
		}
	case *printStmt:
		for _, a := range st.args {
			e.preRegister(a)
		}
	case *exprStmt:
		e.preRegister(st.x)
	case *declStmt:
		e.preRegister(st.init)
	case *assignStmt:
		e.preRegister(st.x)
	case *returnStmt:
		e.preRegister(st.x)
	case *indexAssignStmt:
		e.preRegister(st.idx)
		e.preRegister(st.x)
	case *fieldAssignStmt:
		e.preRegister(st.recv)
		e.preRegister(st.x)
	case *ifStmt:
		e.preRegister(st.cond)
		for _, s2 := range st.then {
			e.preRegisterStmt(s2)
		}
		for _, s2 := range st.els {
			e.preRegisterStmt(s2)
		}
	case *whileStmt:
		e.preRegister(st.cond)
		for _, s2 := range st.body {
			e.preRegisterStmt(s2)
		}
	case *forStmt:
		if st.init != nil {
			e.preRegisterStmt(st.init)
		}
		e.preRegister(st.cond)
		if st.step != nil {
			e.preRegisterStmt(st.step)
		}
		for _, s2 := range st.body {
			e.preRegisterStmt(s2)
		}
	case *forInStmt:
		for _, s2 := range st.body {
			e.preRegisterStmt(s2)
		}
	case *logStmt:
		e.preRegister(st.x)
	}
}

func (e *emitter) preRegister(x *expr) {
	if x == nil {
		return
	}
	if x.kind == kString {
		x.sc = e.strConst(x.s)
	}
	e.preRegister(x.l)
	e.preRegister(x.r)
	if x.call != nil {
		for _, a := range x.call.args {
			e.preRegister(a)
		}
	}
	if x.method != nil {
		e.preRegister(x.method.recv)
		for _, a := range x.method.args {
			e.preRegister(a)
		}
	}
	if x.field != nil {
		e.preRegister(x.field.recv)
	}
	if x.lst != nil {
		for _, it := range x.lst.items {
			e.preRegister(it)
		}
	}
	if x.sl != nil {
		for _, v := range x.sl.values {
			e.preRegister(v)
		}
	}
	if x.idx != nil {
		e.preRegister(x.idx.i)
	}
}

func (e *emitter) emitBlock(stmts []stmt) {
	for _, s := range stmts {
		if e.funcReturned || e.term {
			return // 终结指令之后的语句不可达，跳过
		}
		e.emitStmt(s)
	}
}

func (e *emitter) emitStmt(s stmt) {
	switch st := s.(type) {
	case *exprStmt:
		e.compileExpr(st.x) // 副作用求值，结果丢弃
	case *returnStmt:
		e.emitReturn(st)
	case *printlnStmt:
		e.emitPrint(st.args, true)
	case *printStmt:
		e.emitPrint(st.args, false)
	case *declStmt:
		e.emitDecl(st)
	case *assignStmt:
		e.emitAssign(st)
	case *indexAssignStmt:
		e.emitIndexAssign(st)
	case *fieldAssignStmt:
		e.emitFieldAssign(st)
	case *tryStmt:
		e.emitTry(st)
	case *deleteStmt:
		e.emitDelete(st)
	case *ifStmt:
		e.emitIf(st)
	case *whileStmt:
		e.emitWhile(st)
	case *forStmt:
		e.emitFor(st)
	case *forInStmt:
		e.emitForIn(st)
	case *breakStmt:
		if len(e.breaks) > 0 {
			e.emitInstr("br label %%%s", e.breaks[len(e.breaks)-1])
		}
	case *logStmt:
		e.compileExpr(st.x) // 求值（副作用），结果只进日志
		e.emitRetZero()     // log 记录后立即结束函数（解释器返回 nil）
	}
}

// emitRetZero 发射当前函数返回类型的零值返回（log 用）。
func (e *emitter) emitRetZero() {
	switch e.curRet {
	case "int":
		e.emitInstr("ret i32 0")
	case "bool":
		e.emitInstr("ret i1 false")
	case "float":
		e.emitInstr("ret double 0.0")
	case "String":
		e.emitInstr("ret i8* null")
	default:
		e.emitInstr("ret void")
	}
	e.funcReturned = true
}

func (e *emitter) emitReturn(st *returnStmt) {
	if st.x == nil {
		e.emitRetZero()
		return
	}
	// 尾调用优化：return f(args) → tail call（尾递归栈 O(1)）
	if st.x.kind == kCall && st.x.call != nil && st.x.call.name != "sum" && st.x.call.name != "clock" {
		c := st.x.call
		argRegs := e.compileArgs(c)
		retTy := e.ir(e.sigRet(c.name))
		r := e.newReg()
		e.body.WriteString(r)
		e.body.WriteString(" = tail call " + retTy + " @" + c.name + "(" + strings.Join(argRegs, ", ") + ")\n")
		e.emitInstr("ret %s %s", retTy, r)
		e.funcReturned = true
		return
	}
	v, vt := e.compileExpr(st.x)
	v = e.coerce(v, vt, e.curRet)
	e.emitInstr("ret %s %s", e.ir(e.curRet), v)
	e.funcReturned = true
}

// compileArgs 编译调用实参，并按被调函数签名做隐式转换（int → float）。
// 返回 "类型 寄存器" 形式的列表。
func (e *emitter) compileArgs(c *callExpr) []string {
	sig := e.sigs[c.name]
	out := make([]string, 0, len(c.args))
	for i, a := range c.args {
		v, vt := e.compileExpr(a)
		want := vt
		if sig != nil && i < len(sig.params) {
			want = sig.params[i].typ
		}
		v = e.coerce(v, vt, want)
		// 变参（printf 风格）不在此列，本 IR 全部为定参
		out = append(out, e.ir(want)+" "+v)
	}
	return out
}

// sigRet 返回被调函数的返回类型（无签名时按 i32 假定）。
func (e *emitter) sigRet(name string) string {
	if s, ok := e.sigs[name]; ok {
		return s.ret
	}
	return "int"
}

// coerce 在类型间做隐式转换：语言层 int → float（double），以及 FFI 的
// f32 ↔ double / bool ↔ C int 边界转换。
func (e *emitter) coerce(reg, from, to string) string {
	if from == to || to == "" || to == "?" {
		return reg
	}
	switch {
	case from == "int" && (to == "float" || to == "double"):
		r := e.newReg()
		e.emitInstr("%s = sitofp i32 %s to double", r, reg)
		return r
	case from == "int" && to == "f32":
		r := e.newReg()
		e.emitInstr("%s = sitofp i32 %s to float", r, reg)
		return r
	case from == "f32" && (to == "float" || to == "double"):
		r := e.newReg()
		e.emitInstr("%s = fpext float %s to double", r, reg)
		return r
	case (from == "float" || from == "double") && to == "f32":
		r := e.newReg()
		e.emitInstr("%s = fptrunc double %s to float", r, reg)
		return r
	case from == "bool" && to == "cbool":
		r := e.newReg()
		e.emitInstr("%s = zext i1 %s to i32", r, reg)
		return r
	case from == "cbool" && to == "bool":
		r := e.newReg()
		e.emitInstr("%s = icmp ne i32 %s, 0", r, reg)
		return r
	case from == "int" && to == "cbool":
		r := e.newReg()
		e.emitInstr("%s = icmp ne i32 %s, 0", r, reg)
		return r
	case from == "cbool" && to == "int":
		return reg
	}
	return reg
}

// ---------- 声明 / 赋值 ----------

func (e *emitter) emitDecl(st *declStmt) {
	switch st.typ {
	case "int", "bool", "float", "String":
		// SSA 直通：单赋值标量直接用寄存器（免 alloca/load/store）
		if !e.assigned[st.name] && st.init != nil && !e.funcReturned {
			v, vt := e.compileExpr(st.init)
			e.vars[st.name] = varSlot{reg: e.coerce(v, vt, st.typ), typ: st.typ, direct: true}
			return
		}
		reg := e.newReg()
		llt := e.ir(st.typ)
		e.emitInstr("%s = alloca %s, align %d", reg, llt, e.alignOf(st.typ))
		v := e.zeroOf(st.typ)
		if st.init != nil {
			x, xt := e.compileExpr(st.init)
			v = e.coerce(x, xt, st.typ)
		}
		e.emitInstr("store %s %s, %s* %s", llt, v, llt, reg)
		e.vars[st.name] = varSlot{reg: reg, typ: st.typ}
	case "List<int>":
		e.ensureList()
		if st.init.kind != kList {
			// List 变量 / 调用结果：引用语义，直接存指针
			slot := e.newReg()
			e.emitInstr("%s = alloca %%List*, align 8", slot)
			v, _ := e.compileExpr(st.init)
			e.emitInstr("store %%List* %s, %%List** %s", v, slot)
			e.vars[st.name] = varSlot{reg: slot, typ: st.typ}
			return
		}
		lit := st.init.lst
		n := len(lit.items)
		obj := e.newReg()
		e.emitInstr("%s = call i8* @calloc(i64 1, i64 16)", obj)
		lo := e.newReg()
		e.emitInstr("%s = bitcast i8* %s to %%List*", lo, obj)
		buf := e.newReg()
		e.emitInstr("%s = call i8* @malloc(i64 %d)", buf, max(1, n)*4)
		p := e.newReg()
		e.emitInstr("%s = bitcast i8* %s to i32*", p, buf)
		for i, it := range lit.items {
			v, vt := e.compileExpr(it)
			v = e.coerce(v, vt, "int")
			if i == 0 {
				e.emitInstr("store i32 %s, i32* %s", v, p)
			} else {
				g2 := e.newReg()
				e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %d", g2, p, i)
				e.emitInstr("store i32 %s, i32* %s", v, g2)
			}
		}
		bf := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 0", bf, lo)
		e.emitInstr("store i32* %s, i32** %s", p, bf)
		tf := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, lo)
		e.emitInstr("store i32 %d, i32* %s", n, tf)
		e.vars[st.name] = varSlot{reg: e.listSlot(lo), typ: st.typ}
	default:
		if e.ifaces[st.typ] {
			e.ensureIface()
			reg := e.newReg()
			e.emitInstr("%s = alloca %%Iface, align 8", reg)
			v := "zeroinitializer"
			if st.init != nil {
				x, _ := e.compileExpr(st.init)
				v = x
			}
			e.emitInstr("store %%Iface %s, %%Iface* %s", v, reg)
			e.vars[st.name] = varSlot{reg: reg, typ: st.typ}
			return
		}
		// struct 类型：引用语义（calloc 零值 + 可选字面量字段写入）
		reg := e.newReg()
		ptrTy := e.ir(st.typ)
		e.emitInstr("%s = alloca %s, align 8", reg, ptrTy)
		var v string
		if st.init != nil {
			v, _ = e.compileExpr(st.init)
		} else {
			obj := e.structAlloc(st.typ)
			v = e.newReg()
			e.emitInstr("%s = bitcast i8* %s to %s", v, obj, ptrTy)
			e.zeroStructFields(v, st.typ, 0)
		}
		e.emitInstr("store %s %s, %s* %s", ptrTy, v, ptrTy, reg)
		e.vars[st.name] = varSlot{reg: reg, typ: st.typ}
	}
}

// listSlot 把 List 对象指针存进 alloca 槽（变量引用语义：槽里放指针）。
func (e *emitter) listSlot(objPtr string) string {
	slot := e.newReg()
	e.emitInstr("%s = alloca %%List*, align 8", slot)
	e.emitInstr("store %%List* %s, %%List** %s", objPtr, slot)
	return slot
}

// structAlloc 分配一个清零的 struct 实例（LLVM 布局：GEP null,1 → 真实大小，
// 避免手算字段对齐导致越界写）。
func (e *emitter) structAlloc(typ string) string {
	elem := e.irElem(typ)
	g := e.newReg()
	e.emitInstr("%s = getelementptr %s, %s* null, i32 1", g, elem, elem)
	sz := e.newReg()
	e.emitInstr("%s = ptrtoint %s* %s to i64", sz, elem, g)
	obj := e.newReg()
	e.emitInstr("%s = call i8* @calloc(i64 1, i64 %s)", obj, sz)
	return obj
}

func (e *emitter) emitAssign(st *assignStmt) {
	info, ok := e.vars[st.name]
	if !ok {
		e.emitInstr("; undeclared variable %s", st.name)
		return
	}
	v, vt := e.compileExpr(st.x)
	v = e.coerce(v, vt, info.typ)
	if info.param || info.direct {
		// 参数/直通变量本不该被赋值（lowering 的 assigned 分析保证）
		e.emitInstr("; assignment to register variable %s", st.name)
		return
	}
	llt := e.ir(info.typ)
	e.emitInstr("store %s %s, %s* %s", llt, v, llt, info.reg)
}

func (e *emitter) emitIndexAssign(st *indexAssignStmt) {
	// l[i] = v
	lo := e.listObj(st.name)
	head, size := e.listHeadSize(lo)
	i, it := e.compileExpr(st.idx)
	i = e.coerce(i, it, "int")
	e.emitBoundsCheck(i, size, st.idx.line)
	idx := e.newReg()
	e.emitInstr("%s = add i32 %s, %s", idx, head, i)
	lp := e.listBuf(lo)
	i64 := e.toI64(idx)
	g2 := e.newReg()
	e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", g2, lp, i64)
	v, vt := e.compileExpr(st.x)
	v = e.coerce(v, vt, "int")
	e.emitInstr("store i32 %s, i32* %s", v, g2)
}

func (e *emitter) emitFieldAssign(st *fieldAssignStmt) {
	obj, otyp := e.compileExpr(st.recv)
	info := e.structs[otyp]
	fidx := -1
	var ftyp string
	for i, f := range info {
		if f.name == st.field {
			fidx, ftyp = i, f.typ
			break
		}
	}
	if fidx < 0 {
		return
	}
	g := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %s, %s* %s, i32 0, i32 %d", g, e.irElem(otyp), e.irElem(otyp), obj, fidx)
	v, vt := e.compileExpr(st.x)
	v = e.coerce(v, vt, ftyp)
	e.emitInstr("store %s %s, %s* %s", e.ir(ftyp), v, e.ir(ftyp), g)
}

// listObj 取 List 变量的对象指针。
func (e *emitter) listObj(name string) string {
	e.ensureList()
	info, ok := e.vars[name]
	if !ok {
		return "null"
	}
	if info.param || info.direct {
		return info.reg
	}
	r := e.newReg()
	e.emitInstr("%s = load %%List*, %%List** %s", r, info.reg)
	return r
}

// listBuf 取 List 的缓冲区指针。
func (e *emitter) listBuf(obj string) string {
	bf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 0", bf, obj)
	p := e.newReg()
	e.emitInstr("%s = load i32*, i32** %s", p, bf)
	return p
}

// listHeadSize 取 List 的 head 游标与可见元素个数（tail-head）。
func (e *emitter) listHeadSize(lo string) (string, string) {
	hf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 1", hf, lo)
	tf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, lo)
	h := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", h, hf)
	t := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", t, tf)
	sz := e.newReg()
	e.emitInstr("%s = sub i32 %s, %s", sz, t, h)
	return h, sz
}

// emitBoundsCheck 对 List 下标（已求值的可见下标 int 寄存器）做运行期检查：
// 0 <= i < size；越界 → try 内跳 catch，否则运行期错误（与解释器同一文案）。
func (e *emitter) emitBoundsCheck(i, size string, line int) {
	lo := e.newReg()
	e.emitInstr("%s = icmp slt i32 %s, 0", lo, i)
	hi := e.newReg()
	e.emitInstr("%s = icmp sge i32 %s, %s", hi, i, size)
	bad := e.newReg()
	e.emitInstr("%s = or i1 %s, %s", bad, lo, hi)
	okB := e.newBlock()
	badB := e.newBlock()
	e.emitInstr("br i1 %s, label %%%s, label %%%s", bad, badB, okB)
	e.setBlock(badB)
	e.emitIndexPanic(i, size, line)
	e.setBlock(okB)
}

// emitIndexPanic 发射 List 越界运行期错误。
func (e *emitter) emitIndexPanic(i, size string, line int) {
	if e.curTry != "" {
		e.emitInstr("br label %%%s", e.curTry)
		e.emitInstr("unreachable")
		return
	}
	e.needPanic = true
	e.emitInstr("call void @ql_panic_index(i32 %s, i32 %s, i32 %d)", i, size, line)
	e.emitInstr("unreachable")
}

func (e *emitter) emitTry(st *tryStmt) {
	oldTry := e.curTry
	catchL := e.newBlock()
	afterL := e.newBlock()
	e.curTry = catchL
	for _, s := range st.then {
		e.emitStmt(s)
	}
	if !e.funcReturned && !e.term {
		e.emitInstr("br label %%%s", afterL) // 正常路径跳过 catch
	}
	e.curTry = oldTry
	e.setBlock(catchL)
	for _, s := range st.catch {
		e.emitStmt(s)
	}
	if !e.funcReturned && !e.term {
		e.emitInstr("br label %%%s", afterL)
	}
	e.setBlock(afterL)
}

func (e *emitter) emitDelete(st *deleteStmt) {
	info, ok := e.vars[st.name]
	if !ok {
		return
	}
	if info.typ == "List<int>" {
		lo := e.listObj(st.name)
		p := e.listBuf(lo)
		c := e.newReg()
		e.emitInstr("%s = bitcast i32* %s to i8*", c, p)
		e.emitInstr("call void @free(i8* %s)", c)
		c2 := e.newReg()
		e.emitInstr("%s = bitcast %%List* %s to i8*", c2, lo)
		e.emitInstr("call void @free(i8* %s)", c2)
		return
	}
	if info.param || info.direct {
		return
	}
	c := e.newReg()
	e.emitInstr("%s = bitcast %s* %s to i8*", c, e.ir(info.typ), info.reg)
	e.emitInstr("call void @free(i8* %s)", c)
}

func (e *emitter) emitIf(st *ifStmt) {
	thenB, endB := e.newBlock(), e.newBlock()
	var elseB string
	if len(st.els) > 0 {
		elseB = e.newBlock()
	}
	c, _ := e.compileExpr(st.cond)
	if elseB != "" {
		e.emitInstr("br i1 %s, label %%%s, label %%%s", c, thenB, elseB)
	} else {
		e.emitInstr("br i1 %s, label %%%s, label %%%s", c, thenB, endB)
	}
	saved := e.funcReturned
	e.setBlock(thenB)
	e.emitBlock(st.then)
	thenRet := e.funcReturned || e.term
	e.funcReturned = saved
	if !thenRet {
		e.emitInstr("br label %%%s", endB)
	}
	if elseB != "" {
		e.setBlock(elseB)
		e.emitBlock(st.els)
		elseRet := e.funcReturned || e.term
		e.funcReturned = saved
		if !elseRet {
			e.emitInstr("br label %%%s", endB)
		}
	}
	e.funcReturned = saved
	e.setBlock(endB)
}

func (e *emitter) emitWhile(st *whileStmt) {
	condB, bodyB, endB := e.newBlock(), e.newBlock(), e.newBlock()
	e.emitInstr("br label %%%s", condB)
	e.setBlock(condB)
	c, _ := e.compileExpr(st.cond)
	e.emitInstr("br i1 %s, label %%%s, label %%%s", c, bodyB, endB)
	saved := e.funcReturned
	e.breaks = append(e.breaks, endB)
	e.setBlock(bodyB)
	e.emitBlock(st.body)
	bodyRet := e.funcReturned || e.term
	e.funcReturned = saved
	e.breaks = e.breaks[:len(e.breaks)-1]
	if !bodyRet {
		e.emitInstr("br label %%%s", condB)
	}
	e.funcReturned = saved
	e.setBlock(endB)
}

// emitFor 发射 C 风格 for：init; cond; step。
func (e *emitter) emitFor(st *forStmt) {
	if st.init != nil {
		e.emitStmt(st.init)
	}
	condB, bodyB, stepB, endB := e.newBlock(), e.newBlock(), e.newBlock(), e.newBlock()
	e.emitInstr("br label %%%s", condB)
	e.setBlock(condB)
	c, _ := e.compileExpr(st.cond)
	e.emitInstr("br i1 %s, label %%%s, label %%%s", c, bodyB, endB)
	saved := e.funcReturned
	e.breaks = append(e.breaks, endB)
	e.setBlock(bodyB)
	e.emitBlock(st.body)
	bodyRet := e.funcReturned || e.term
	e.funcReturned = saved
	if !bodyRet {
		e.emitInstr("br label %%%s", stepB)
	}
	e.setBlock(stepB)
	if st.step != nil {
		e.emitStmt(st.step)
	}
	if !e.funcReturned {
		e.emitInstr("br label %%%s", condB)
	}
	e.funcReturned = saved
	e.breaks = e.breaks[:len(e.breaks)-1]
	e.setBlock(endB)
}

// emitForIn 发射迭代 for：for (T x : list) —— 与解释器一致，按滚动游标消费
// （head 前进；循环结束后元素已消耗，size() 归零）。
func (e *emitter) emitForIn(st *forInStmt) {
	slot := e.newReg()
	llt := e.ir(st.typ)
	e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(st.typ))
	e.vars[st.name] = varSlot{reg: slot, typ: st.typ}
	lo := e.listObj(st.list)
	condB, bodyB, endB := e.newBlock(), e.newBlock(), e.newBlock()
	e.emitInstr("br label %%%s", condB)
	e.setBlock(condB)
	hf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 1", hf, lo)
	tf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, lo)
	h := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", h, hf)
	t := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", t, tf)
	c := e.newReg()
	e.emitInstr("%s = icmp slt i32 %s, %s", c, h, t)
	e.emitInstr("br i1 %s, label %%%s, label %%%s", c, bodyB, endB)
	saved := e.funcReturned
	e.breaks = append(e.breaks, endB)
	e.setBlock(bodyB)
	h2 := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", h2, hf)
	p := e.listBuf(lo)
	h64 := e.toI64(h2)
	ep := e.newReg()
	e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", ep, p, h64)
	ev := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", ev, ep)
	e.emitInstr("store i32 %s, i32* %s", ev, slot)
	nx := e.newReg()
	e.emitInstr("%s = add i32 %s, 1", nx, h2)
	e.emitInstr("store i32 %s, i32* %s", nx, hf)
	e.emitBlock(st.body)
	bodyRet := e.funcReturned || e.term
	e.funcReturned = saved
	if !bodyRet {
		e.emitInstr("br label %%%s", condB)
	}
	e.funcReturned = saved
	e.breaks = e.breaks[:len(e.breaks)-1]
	e.setBlock(endB)
}

// emitPrint 发射 io.println / io.print（newline 决定是否换行）。
func (e *emitter) emitPrint(args []*expr, newline bool) {
	type av struct {
		val string
		typ string
	}
	vals := make([]av, 0, len(args))
	fmts := make([]string, 0, len(args))
	for _, a := range args {
		v, t := e.compileExpr(a)
		vals = append(vals, av{v, t})
		switch t {
		case "int":
			fmts = append(fmts, "%d")
		case "float":
			e.needFloatToStr = true
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_float_to_str(double %s)", r, v)
			vals[len(vals)-1].val = r
			vals[len(vals)-1].typ = "String"
			fmts = append(fmts, "%s")
		case "bool":
			tp := e.i8Ptr(e.strs["true"])
			fp := e.i8Ptr(e.strs["false"])
			r := e.newReg()
			e.emitInstr("%s = select i1 %s, i8* %s, i8* %s", r, e.toI1(v), tp, fp)
			vals[len(vals)-1].val = r
			vals[len(vals)-1].typ = "String"
			fmts = append(fmts, "%s")
		default:
			fmts = append(fmts, "%s")
		}
	}
	line := strings.Join(fmts, " ")
	if newline {
		line += "\n"
	}
	if len(args) == 0 {
		if !newline {
			return
		}
		line = "\n"
	}
	fp := e.i8Ptr(e.strConst(line))
	var sb strings.Builder
	sb.WriteString("  " + e.newReg() + " = call i32 (i8*, ...) @printf(i8* " + fp)
	for _, a := range vals {
		sb.WriteString(", " + e.ir(a.typ) + " " + a.val)
	}
	sb.WriteString(")\n")
	e.body.WriteString(sb.String())
}

// llvmFloat 把 float64 格式化为 LLVM 浮点字面量（必须含小数点/指数，
// 否则 "2" 会被当成整数常量：fadd double 1.5, 2 非法）。
func llvmFloat(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eEnN") {
		s += ".0"
	}
	return s
}

// toI1 把值转成 i1（bool 直接用；int 判零）。
func (e *emitter) toI1(reg string) string {
	if reg == "true" || reg == "false" {
		return reg
	}
	return reg
}

// toI64 把 i32 寄存器扩展为 i64（字面量直接可用）。
func (e *emitter) toI64(reg string) string {
	if strings.HasPrefix(reg, "%") {
		r := e.newReg()
		e.emitInstr("%s = sext i32 %s to i64", r, reg)
		return r
	}
	return reg
}

// ---------- 表达式 ----------

// compileExpr 编译表达式，返回 (寄存器, 语言类型)（含接口装箱）。
func (e *emitter) compileExpr(x *expr) (string, string) {
	v, t := e.compileExprRaw(x)
	if x.ifaceBox != "" && t != x.typ {
		v = e.boxIface(v, t, x.ifaceBox)
		t = x.typ
	}
	return v, t
}

// boxIface 把具体 struct 指针装箱为接口值 { data, vtable }。
func (e *emitter) boxIface(reg, from, sym string) string {
	e.ensureIface()
	ptr := e.newReg()
	e.emitInstr("%s = bitcast %s %s to i8*", ptr, e.ir(from), reg)
	i0 := e.newReg()
	e.emitInstr("%s = insertvalue %%Iface undef, i8* %s, 0", i0, ptr)
	i1 := e.newReg()
	e.emitInstr("%s = insertvalue %%Iface %s, i8** %s, 1", i1, i0, sym)
	return i1
}

// compileExprRaw 编译表达式（不含装箱包装）。
func (e *emitter) compileExprRaw(x *expr) (string, string) {
	switch x.kind {
	case kInt:
		return fmt.Sprintf("%d", int32(x.i)), "int"
	case kFloat:
		return llvmFloat(x.f), "float"
	case kString:
		if x.sc.name == "" {
			x.sc = e.strConst(x.s)
		}
		return e.i8Ptr(x.sc), "String"
	case kRunner:
		r := e.newReg()
		e.emitInstr("%s = bitcast void (i32)* @%s to i8*", r, x.s)
		return r, "runner"
	case kBool:
		if x.b {
			return "true", "bool"
		}
		return "false", "bool"
	case kIdent:
		return e.loadVar(x.s)
	case kField:
		return e.compileField(x)
	case kCmp, kBin:
		return e.compileBin(x)
	case kAndOr:
		return e.compileLogic(x)
	case kMethod:
		return e.compileMethod(x)
	case kStructLit:
		return e.compileStructLit(x)
	case kIndex:
		return e.compileIndex(x)
	case kCall:
		return e.compileCall(x)
	case kToString:
		v, t := e.compileExpr(x.l)
		switch t {
		case "String":
			return v, "String"
		case "bool":
			tp := e.i8Ptr(e.strs["true"])
			fp := e.i8Ptr(e.strs["false"])
			r := e.newReg()
			e.emitInstr("%s = select i1 %s, i8* %s, i8* %s", r, v, tp, fp)
			return r, "String"
		case "float":
			e.needFloatToStr = true
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_float_to_str(double %s)", r, v)
			return r, "String"
		default:
			e.needIntToStr = true
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_int_to_str(i32 %s)", r, v)
			return r, "String"
		}
	case kList:
		return e.compileListLit(x)
	}
	return "0", "int"
}

// loadVar 读取变量（槽 → load；参数/直通 → 直接用）。
func (e *emitter) loadVar(name string) (string, string) {
	info, ok := e.vars[name]
	if !ok {
		return name, "int" // 未声明变量：让生成的 IR 报错（llvm-as 校验会拦截）
	}
	if info.param || info.direct {
		return info.reg, info.typ
	}
	r := e.newReg()
	llt := e.ir(info.typ)
	e.emitInstr("%s = load %s, %s* %s", r, llt, llt, info.reg)
	return r, info.typ
}

func (e *emitter) compileField(x *expr) (string, string) {
	obj, otyp := e.compileExpr(x.field.recv)
	fields := e.structs[otyp]
	for i, f := range fields {
		if f.name == x.field.name {
			g := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %s, %s* %s, i32 0, i32 %d", g, e.irElem(otyp), e.irElem(otyp), obj, i)
			v := e.newReg()
			e.emitInstr("%s = load %s, %s* %s", v, e.ir(f.typ), e.ir(f.typ), g)
			return v, f.typ
		}
	}
	return "0", "int"
}

func (e *emitter) compileStructLit(x *expr) (string, string) {
	typ := x.sl.typ
	ptrTy := e.ir(typ)
	obj := e.structAlloc(typ)
	v := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to %s", v, obj, ptrTy)
	e.zeroStructFields(v, typ, 0)
	fields := e.structs[typ]
	for i, val := range x.sl.values {
		if i >= len(fields) {
			break
		}
		fv, ft := e.compileExpr(val)
		fv = e.coerce(fv, ft, fields[i].typ)
		g := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %s, %s %s, i32 0, i32 %d", g, e.irElem(typ), ptrTy, v, i)
		e.emitInstr("store %s %s, %s* %s", e.ir(fields[i].typ), fv, e.ir(fields[i].typ), g)
	}
	return v, typ
}

func (e *emitter) compileIndex(x *expr) (string, string) {
	lo := e.listObj(x.idx.name)
	head, size := e.listHeadSize(lo)
	i, it := e.compileExpr(x.idx.i)
	i = e.coerce(i, it, "int")
	e.emitBoundsCheck(i, size, x.line)
	idx := e.newReg()
	e.emitInstr("%s = add i32 %s, %s", idx, head, i)
	p := e.listBuf(lo)
	i64 := e.toI64(idx)
	g2 := e.newReg()
	e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", g2, p, i64)
	v := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", v, g2)
	return v, "int"
}

func (e *emitter) compileListLit(x *expr) (string, string) {
	e.ensureList()
	n := len(x.lst.items)
	obj := e.newReg()
	e.emitInstr("%s = call i8* @calloc(i64 1, i64 16)", obj)
	lo := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to %%List*", lo, obj)
	buf := e.newReg()
	e.emitInstr("%s = call i8* @malloc(i64 %d)", buf, max(1, n)*4)
	p := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to i32*", p, buf)
	for i, it := range x.lst.items {
		v, vt := e.compileExpr(it)
		v = e.coerce(v, vt, "int")
		g2 := e.newReg()
		e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %d", g2, p, i)
		e.emitInstr("store i32 %s, i32* %s", v, g2)
	}
	bf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 0", bf, lo)
	e.emitInstr("store i32* %s, i32** %s", p, bf)
	tf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, lo)
	e.emitInstr("store i32 %d, i32* %s", n, tf)
	return lo, "List<int>"
}

// compileCall 编译普通函数调用与内建（sum/clock）。
func (e *emitter) compileCall(x *expr) (string, string) {
	c := x.call
	if c.name == "sum" && len(c.args) >= 3 && c.args[0].kind == kIdent {
		return e.emitSum(c), "int"
	}
	if c.name == "clock" {
		return e.emitClock(), "int"
	}
	ret := e.sigRet(c.name)
	args := e.compileArgs(c)
	if ret == "void" {
		e.body.WriteString("  call void @" + c.name + "(" + strings.Join(args, ", ") + ")\n")
		return "0", "void"
	}
	r := e.newReg()
	e.body.WriteString(r + " = call " + e.ir(ret) + " @" + c.name + "(" + strings.Join(args, ", ") + ")\n")
	switch ret {
	case "cbool":
		b := e.newReg()
		e.emitInstr("%s = icmp ne i32 %s, 0", b, r)
		return b, "bool"
	case "f32":
		d := e.newReg()
		e.emitInstr("%s = fpext float %s to double", d, r)
		return d, "float"
	case "double":
		return r, "float" // FFI double → 语言 float
	}
	return r, ret
}

// compileMethod 编译实例方法调用与内建方法。
func (e *emitter) compileMethod(x *expr) (string, string) {
	m := x.method
	if m.iface != "" {
		return e.compileIfaceCall(x)
	}
	recv, rtyp := e.compileExpr(m.recv)
	// List 内建方法
	if rtyp == "List<int>" {
		switch m.name {
		case "size":
			hf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 1", hf, recv)
			tf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, recv)
			h := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", h, hf)
			t := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", t, tf)
			r := e.newReg()
			e.emitInstr("%s = sub i32 %s, %s", r, t, h)
			return r, "int"
		case "get":
			head, size := e.listHeadSize(recv)
			i, it := e.compileExpr(m.args[0])
			i = e.coerce(i, it, "int")
			e.emitBoundsCheck(i, size, x.line)
			idx := e.newReg()
			e.emitInstr("%s = add i32 %s, %s", idx, head, i)
			p := e.listBuf(recv)
			i64 := e.toI64(idx)
			g2 := e.newReg()
			e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", g2, p, i64)
			v := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", v, g2)
			return v, "int"
		case "append":
			tf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, recv)
			t1 := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", t1, tf)
			// realloc(buf, max(tail*2,1)*4)
			n1 := e.newReg()
			e.emitInstr("%s = shl i32 %s, 1", n1, t1)
			n1b := e.newReg()
			e.emitInstr("%s = icmp slt i32 %s, 1", n1b, n1)
			n1c := e.newReg()
			e.emitInstr("%s = select i1 %s, i32 1, i32 %s", n1c, n1b, n1)
			n64 := e.newReg()
			e.emitInstr("%s = sext i32 %s to i64", n64, n1c)
			sz := e.newReg()
			e.emitInstr("%s = mul i64 %s, 4", sz, n64)
			bf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 0", bf, recv)
			lp := e.newReg()
			e.emitInstr("%s = load i32*, i32** %s", lp, bf)
			bc := e.newReg()
			e.emitInstr("%s = bitcast i32* %s to i8*", bc, lp)
			rc := e.newReg()
			e.emitInstr("%s = call i8* @realloc(i8* %s, i64 %s)", rc, bc, sz)
			np := e.newReg()
			e.emitInstr("%s = bitcast i8* %s to i32*", np, rc)
			e.emitInstr("store i32* %s, i32** %s", np, bf)
			t1i := e.newReg()
			e.emitInstr("%s = sext i32 %s to i64", t1i, t1)
			g2 := e.newReg()
			e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", g2, np, t1i)
			v, vt := e.compileExpr(m.args[0])
			v = e.coerce(v, vt, "int")
			e.emitInstr("store i32 %s, i32* %s", v, g2)
			t2 := e.newReg()
			e.emitInstr("%s = add i32 %s, 1", t2, t1)
			e.emitInstr("store i32 %s, i32* %s", t2, tf)
			return "0", "void"
		}
	}
	// struct 实例方法：call @Type_method(recv, args...)
	if m.sig != "" {
		args := []string{e.ir(rtyp) + " " + recv}
		sig := e.sigs[m.sig]
		for i, a := range m.args {
			v, vt := e.compileExpr(a)
			want := vt
			if sig != nil && i < len(sig.params) {
				want = sig.params[i].typ
			}
			v = e.coerce(v, vt, want)
			args = append(args, e.ir(want)+" "+v)
		}
		ret := e.sigRet(m.sig)
		if ret == "void" {
			e.body.WriteString("  call void @" + m.sig + "(" + strings.Join(args, ", ") + ")\n")
			return "0", "void"
		}
		r := e.newReg()
		e.body.WriteString(r + " = call " + e.ir(ret) + " @" + m.sig + "(" + strings.Join(args, ", ") + ")\n")
		return r, ret
	}
	return "0", "int"
}

// compileIfaceCall 编译接口方法调用：从 vtable 取函数指针，经 thunk 调具体方法。
func (e *emitter) compileIfaceCall(x *expr) (string, string) {
	m := x.method
	e.ensureIface()
	recv, _ := e.compileExpr(m.recv)
	data := e.newReg()
	e.emitInstr("%s = extractvalue %%Iface %s, 0", data, recv)
	vt := e.newReg()
	e.emitInstr("%s = extractvalue %%Iface %s, 1", vt, recv)
	slot := e.newReg()
	e.emitInstr("%s = getelementptr inbounds i8*, i8** %s, i32 %d", slot, vt, m.idx)
	fp := e.newReg()
	e.emitInstr("%s = load i8*, i8** %s", fp, slot)
	pty := []string{"i8*"}
	callArgs := []string{"i8* " + data}
	for i, a := range m.args {
		av, at := e.compileExpr(a)
		want := "?"
		if m.ifaceSig != nil && i < len(m.ifaceSig.params) {
			want = m.ifaceSig.params[i].typ
		}
		if want == "Self" {
			// Self 形参：接口值 → 取 data（具体实例指针）
			d := e.newReg()
			e.emitInstr("%s = extractvalue %%Iface %s, 0", d, av)
			pty = append(pty, "i8*")
			callArgs = append(callArgs, "i8* "+d)
			continue
		}
		av = e.coerce(av, at, want)
		pty = append(pty, e.ir(want))
		callArgs = append(callArgs, e.ir(want)+" "+av)
	}
	retIR := "void"
	if m.ifaceSig != nil {
		retIR = e.ir(m.ifaceSig.ret)
	}
	fnty := retIR + " (" + strings.Join(pty, ", ") + ")*"
	fn := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to %s", fn, fp, fnty)
	if retIR == "void" {
		e.body.WriteString("  call void " + fn + "(" + strings.Join(callArgs, ", ") + ")\n")
		return "0", "void"
	}
	r := e.newReg()
	e.body.WriteString(r + " = call " + retIR + " " + fn + "(" + strings.Join(callArgs, ", ") + ")\n")
	ret := "int"
	if m.ifaceSig != nil {
		ret = m.ifaceSig.ret
	}
	return r, ret
}

// thunkName 是 (具体类型, 接口, 方法) 的 thunk 名。
func thunkName(th *ifaceThunk) string {
	return "thunk$" + tyName(th.typ) + "$" + tyName(th.iface) + "$" + th.method
}

// thunkSig 返回 thunk 的 LLVM 函数签名 "ret (params)"（self data 恒为 i8*）。
func (e *emitter) thunkSig(th *ifaceThunk) string {
	ps := []string{"i8*"}
	for i, p := range th.params {
		if isSelfIdx(th.selfIdx, i) {
			ps = append(ps, "i8*")
			continue
		}
		ps = append(ps, e.ir(p))
	}
	retIR := e.ir(th.ret)
	if th.boxRet {
		e.ensureIface()
		retIR = "%Iface" // Self 返回：装箱后以接口值返回
	}
	return retIR + " (" + strings.Join(ps, ", ") + ")"
}

func isSelfIdx(idx []int, i int) bool {
	for _, v := range idx {
		if v == i {
			return true
		}
	}
	return false
}

// emitVtables 发射全部 vtable 常量与 thunk 函数（模块尾部；前向引用合法）。
func (e *emitter) emitVtables(vts []*vtableDef) string {
	if len(vts) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, v := range vts {
		fmt.Fprintf(&sb, "%s = private constant [%d x i8*] [", v.sym, len(v.thunks))
		for i, th := range v.thunks {
			if i > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "i8* bitcast (%s* @%s to i8*)", e.thunkSig(th), thunkName(th))
		}
		sb.WriteString("]\n")
	}
	for _, v := range vts {
		for _, th := range v.thunks {
			sb.WriteString(e.emitThunk(th))
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// emitThunk 生成一个 thunk：i8* data → 具体类型指针 → 调具体方法（Self 返回再装箱）。
func (e *emitter) emitThunk(th *ifaceThunk) string {
	savedBody := e.body
	e.body = strings.Builder{}
	e.ensureStruct(th.typ)
	ptrTy := e.ir(th.typ)
	retIR0 := e.ir(th.ret)
	if th.boxRet {
		e.ensureIface()
		retIR0 = "%Iface"
	}
	var sig strings.Builder
	sig.WriteString("define " + retIR0 + " @" + thunkName(th) + "(i8* noundef %d")
	regs := make([]string, len(th.params))
	for i, p := range th.params {
		regs[i] = fmt.Sprintf("%%p%d", i)
		if isSelfIdx(th.selfIdx, i) {
			sig.WriteString(", i8* noundef " + regs[i])
		} else {
			sig.WriteString(", " + e.ir(p) + " noundef " + regs[i])
		}
	}
	sig.WriteString(")\n")
	e.body.WriteString("entry:\n")
	self := e.newReg()
	e.emitInstr("%s = bitcast i8* %%d to %s", self, ptrTy)
	callArgs := []string{ptrTy + " " + self}
	for i, p := range th.params {
		if isSelfIdx(th.selfIdx, i) {
			c := e.newReg()
			e.emitInstr("%s = bitcast i8* %s to %s", c, regs[i], e.ir(p))
			callArgs = append(callArgs, e.ir(p)+" "+c)
			continue
		}
		callArgs = append(callArgs, e.ir(p)+" "+regs[i])
	}
	callRet := e.ir(th.ret) // 具体方法的真实返回类型（装箱前）
	if callRet == "void" {
		e.body.WriteString("  call void @" + th.irName + "(" + strings.Join(callArgs, ", ") + ")\n")
		e.body.WriteString("  ret void\n")
	} else {
		r := e.newReg()
		e.body.WriteString(r + " = call " + callRet + " @" + th.irName + "(" + strings.Join(callArgs, ", ") + ")\n")
		if th.boxRet {
			boxed := e.boxIface(r, th.typ, th.vtSym)
			e.emitInstr("ret %%Iface %s", boxed)
		} else {
			e.emitInstr("ret %s %s", callRet, r)
		}
	}
	out := sig.String() + "{\n" + e.body.String() + "}\n"
	e.body = savedBody
	return out
}

// emitClock 发射 clock()（gettimeofday 微秒）。
func (e *emitter) emitClock() string {
	tv := e.newReg()
	e.emitInstr("%s = alloca { i64, i64 }, align 8", tv)
	rc := e.newReg()
	e.emitInstr("%s = call i32 @gettimeofday({ i64, i64 }* %s, i8* null)", rc, tv)
	secf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds { i64, i64 }, { i64, i64 }* %s, i32 0, i32 0", secf, tv)
	secl := e.newReg()
	e.emitInstr("%s = load i64, i64* %s", secl, secf)
	usf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds { i64, i64 }, { i64, i64 }* %s, i32 0, i32 1", usf, tv)
	usl := e.newReg()
	e.emitInstr("%s = load i64, i64* %s", usl, usf)
	us := e.newReg()
	e.emitInstr("%s = mul i64 %s, 1000000", us, secl)
	u2 := e.newReg()
	e.emitInstr("%s = add i64 %s, %s", u2, us, usl)
	v := e.newReg()
	e.emitInstr("%s = trunc i64 %s to i32", v, u2)
	return v
}

// emitSum 把 sum(g, begin, stop[, step]) 内联展开为循环。
func (e *emitter) emitSum(c *callExpr) string {
	b, _ := e.compileExpr(c.args[1])
	s, _ := e.compileExpr(c.args[2])
	st := "1"
	if len(c.args) == 4 {
		st, _ = e.compileExpr(c.args[3])
	}
	gname := c.args[0].s
	total := e.newReg()
	e.emitInstr("%s = alloca i32, align 4", total)
	e.emitInstr("store i32 0, i32* %s", total)
	iv := e.newReg()
	e.emitInstr("%s = alloca i32, align 4", iv)
	e.emitInstr("store i32 %s, i32* %s", b, iv)
	condB, bodyB, endB := e.newBlock(), e.newBlock(), e.newBlock()
	e.emitInstr("br label %%%s", condB)
	e.setBlock(condB)
	li := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", li, iv)
	c1 := e.newReg()
	e.emitInstr("%s = icmp slt i32 %s, %s", c1, li, s)
	e.emitInstr("br i1 %s, label %%%s, label %%%s", c1, bodyB, endB)
	e.setBlock(bodyB)
	li2 := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", li2, iv)
	gv := e.newReg()
	e.emitInstr("%s = call i32 @%s(i32 %s)", gv, gname, li2)
	tl := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", tl, total)
	t2 := e.newReg()
	e.emitInstr("%s = add i32 %s, %s", t2, tl, gv)
	e.emitInstr("store i32 %s, i32* %s", t2, total)
	li3 := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", li3, iv)
	ni := e.newReg()
	e.emitInstr("%s = add i32 %s, %s", ni, li3, st)
	e.emitInstr("store i32 %s, i32* %s", ni, iv)
	e.emitInstr("br label %%%s", condB)
	e.setBlock(endB)
	tf := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", tf, total)
	return tf
}

// compileBin 编译二元运算（算术/比较/逻辑/String 拼接/运算符重载）。
func (e *emitter) compileBin(x *expr) (string, string) {
	if x.op == "&&" || x.op == "||" {
		return e.compileLogic(x)
	}
	if x.kind == kCmp {
		return e.compileCmp(x)
	}
	lv, lt := e.compileExpr(x.l)
	rv, rt := e.compileExpr(x.r)
	if x.strcat {
		rc := e.newReg()
		e.emitInstr("%s = call i8* @ql_strcat(i8* %s, i8* %s)", rc, lv, rv)
		return rc, "String"
	}
	// 运算符重载：struct 操作数 → 调用 __add__ 等方法
	if sig, ok := e.sigs["__op__"+x.op+"|"+lt]; ok && lt == rt {
		r := e.newReg()
		e.body.WriteString(r + " = call " + e.ir(sig.ret) + " @" + sig.name + "(" + e.ir(lt) + " " + lv + ", " + e.ir(rt) + " " + rv + ")\n")
		return r, sig.ret
	}
	if lt == "float" || rt == "float" {
		lv = e.coerce(lv, lt, "float")
		rv = e.coerce(rv, rt, "float")
		if (x.op == "/" || x.op == "%") && x.op == "/" {
			e.emitZeroCheckF(rv, "DivisionByZeroError: float division by zero", x.line)
		}
		op := map[string]string{"+": "fadd", "-": "fsub", "*": "fmul", "/": "fdiv"}[x.op]
		if op == "" {
			return "0.0", "float" // float % 不可达（lowering 已拦截）
		}
		reg := e.newReg()
		e.emitInstr("%s = %s double %s, %s", reg, op, lv, rv)
		return reg, "float"
	}
	if v, ok := constEval(x); ok {
		return fmt.Sprintf("%d", int32(v)), "int"
	}
	if x.op == "/" || x.op == "%" {
		e.emitZeroCheckI(rv, x.op, x.line)
	}
	op := map[string]string{"+": "add", "-": "sub", "*": "mul", "/": "sdiv", "%": "srem", "<<": "shl", ">>": "ashr"}[x.op]
	reg := e.newReg()
	e.emitInstr("%s = %s i32 %s, %s", reg, op, lv, rv)
	return reg, "int"
}

// emitZeroCheckI 整数除零检查（try 内跳 catch，否则运行期错误）。
func (e *emitter) emitZeroCheckI(rv, op string, line int) {
	cz := e.newReg()
	e.emitInstr("%s = icmp eq i32 %s, 0", cz, rv)
	msg := "DivisionByZeroError: integer division by zero"
	if op == "%" {
		msg = "DivisionByZeroError: modulo by zero"
	}
	okB := e.newBlock()
	var tgt string
	if e.curTry != "" {
		tgt = e.curTry
	} else {
		e.needPanic = true
		tgt = e.newBlock()
		e.emitInstr("br i1 %s, label %%%s, label %%%s", cz, tgt, okB)
		e.setBlock(tgt)
		c := e.i8Ptr(e.strConst(msg))
		e.emitInstr("call void @ql_panic(i8* %s, i32 %d)", c, line)
		e.emitInstr("unreachable")
		e.setBlock(okB)
		return
	}
	e.emitInstr("br i1 %s, label %%%s, label %%%s", cz, tgt, okB)
	e.setBlock(okB)
}

func (e *emitter) emitZeroCheckF(rv string, msg string, line int) {
	cz := e.newReg()
	e.emitInstr("%s = fcmp oeq double %s, 0.0", cz, rv)
	okB := e.newBlock()
	var tgt string
	if e.curTry != "" {
		tgt = e.curTry
	} else {
		e.needPanic = true
		tgt = e.newBlock()
		e.emitInstr("br i1 %s, label %%%s, label %%%s", cz, tgt, okB)
		e.setBlock(tgt)
		c := e.i8Ptr(e.strConst(msg))
		e.emitInstr("call void @ql_panic(i8* %s, i32 %d)", c, line)
		e.emitInstr("unreachable")
		e.setBlock(okB)
		return
	}
	e.emitInstr("br i1 %s, label %%%s, label %%%s", cz, tgt, okB)
	e.setBlock(okB)
}

// compileLogic 短路求值 && / ||（解释器语义：右侧仅在必要时求值）。
func (e *emitter) compileLogic(x *expr) (string, string) {
	if x.op == "!" {
		v, _ := e.compileExpr(x.l)
		r := e.newReg()
		e.emitInstr("%s = xor i1 %s, true", r, v)
		return r, "bool"
	}
	tmp := e.newReg()
	e.emitInstr("%s = alloca i1, align 1", tmp)
	lv, _ := e.compileExpr(x.l)
	lv = e.toI1(lv)
	e.emitInstr("store i1 %s, i1* %s", lv, tmp)
	rhsB := e.newBlock()
	endB := e.newBlock()
	if x.op == "&&" {
		e.emitInstr("br i1 %s, label %%%s, label %%%s", lv, rhsB, endB)
	} else {
		e.emitInstr("br i1 %s, label %%%s, label %%%s", lv, endB, rhsB)
	}
	e.setBlock(rhsB)
	rv, _ := e.compileExpr(x.r)
	rv = e.toI1(rv)
	e.emitInstr("store i1 %s, i1* %s", rv, tmp)
	e.emitInstr("br label %%%s", endB)
	e.setBlock(endB)
	r := e.newReg()
	e.emitInstr("%s = load i1, i1* %s", r, tmp)
	return r, "bool"
}

// compileCmp 编译比较运算（int/float/bool/String/struct）。
func (e *emitter) compileCmp(x *expr) (string, string) {
	lv, lt := e.compileExpr(x.l)
	rv, rt := e.compileExpr(x.r)
	// struct：__eq__/__ne__ 方法或引用比较
	if lt == rt && e.isStruct(lt) {
		if sig, ok := e.sigs["__op__"+x.op+"|"+lt]; ok {
			r := e.newReg()
			e.body.WriteString(r + " = call " + e.ir(sig.ret) + " @" + sig.name + "(" + e.ir(lt) + " " + lv + ", " + e.ir(lt) + " " + rv + ")\n")
			return r, sig.ret
		}
		c := e.newReg()
		e.emitInstr("%s = icmp %s %s %s, %s", c, cmpOps[x.op], e.ir(lt), lv, rv)
		return c, "bool"
	}
	if lt == "bool" && rt == "bool" {
		r := e.newReg()
		e.emitInstr("%s = icmp %s i1 %s, %s", r, cmpOps[x.op], lv, rv)
		return r, "bool"
	}
	if lt == "String" && rt == "String" {
		e.needStrCmp = true
		c := e.newReg()
		e.emitInstr("%s = call i32 @strcmp(i8* %s, i8* %s)", c, lv, rv)
		r := e.newReg()
		e.emitInstr("%s = icmp %s i32 %s, 0", r, cmpOps[x.op], c)
		return r, "bool"
	}
	if lt == "float" || rt == "float" {
		lv = e.coerce(lv, lt, "float")
		rv = e.coerce(rv, rt, "float")
		op := map[string]string{"==": "oeq", "!=": "une", "<": "olt", "<=": "ole", ">": "ogt", ">=": "oge"}[x.op]
		r := e.newReg()
		e.emitInstr("%s = fcmp %s double %s, %s", r, op, lv, rv)
		return r, "bool"
	}
	op := map[string]string{"==": "eq", "!=": "ne", "<": "slt", "<=": "sle", ">": "sgt", ">=": "sge"}[x.op]
	r := e.newReg()
	e.emitInstr("%s = icmp %s i32 %s, %s", r, op, lv, rv)
	return r, "bool"
}

func (e *emitter) isStruct(t string) bool {
	_, ok := e.structs[t]
	return ok
}

var cmpOps = map[string]string{"==": "eq", "!=": "ne", "<": "slt", "<=": "sle", ">": "sgt", ">=": "sge"}

// ---------- 函数属性分析（LLVM norecurse/mustprogress 安全判定） ----------

// fnMeta 收集单个函数的调用关系与循环标志。
type fnMeta struct {
	callees []string
	hasLoop bool
	hasMeth bool // 方法调用/函数引用（无法静态解析，保守不标 norecurse）
	impure  bool // 调用/方法/列表/结构体/下标等可能有外界副作用（禁止 memory(none)）
}

func analyzeExpr(e *expr, m *fnMeta) {
	if e == nil {
		return
	}
	switch e.kind {
	case kCall:
		if e.call != nil {
			m.callees = append(m.callees, e.call.name)
		}
		m.impure = true
	case kMethod:
		m.hasMeth = true
		for _, a := range e.method.args {
			analyzeExpr(a, m)
		}
	case kToString:
		m.impure = true // ql_int_to_str：malloc/snprintf 调用
		analyzeExpr(e.l, m)
	case kBin:
		if e.strcat {
			m.impure = true // ql_strcat 调用：有副作用，禁止 memory(none)
		}
		analyzeExpr(e.l, m)
		analyzeExpr(e.r, m)
	case kIndex:
		m.impure = true
		analyzeExpr(e.idx.i, m)
	case kList:
		m.impure = true
		for _, it := range e.lst.items {
			analyzeExpr(it, m)
		}
	case kStructLit:
		m.impure = true
		for _, v := range e.sl.values {
			analyzeExpr(v, m)
		}
	case kField:
		m.impure = true // 通过指针读 struct 字段：不是纯函数（禁止 memory(none)）
		analyzeExpr(e.field.recv, m)
	case kAndOr:
		analyzeExpr(e.l, m)
		analyzeExpr(e.r, m)
	case kCmp:
		analyzeExpr(e.l, m)
		analyzeExpr(e.r, m)
	}
}

func analyzeStmt(s stmt, m *fnMeta) {
	switch st := s.(type) {
	case *declStmt:
		analyzeExpr(st.init, m)
	case *assignStmt:
		analyzeExpr(st.x, m)
	case *printlnStmt:
		m.impure = true // printf：IO 副作用，禁止 memory(none)
		for _, a := range st.args {
			analyzeExpr(a, m)
		}
	case *printStmt:
		m.impure = true
		for _, a := range st.args {
			analyzeExpr(a, m)
		}
	case *ifStmt:
		analyzeExpr(st.cond, m)
		for _, x := range st.then {
			analyzeStmt(x, m)
		}
		for _, x := range st.els {
			analyzeStmt(x, m)
		}
	case *whileStmt:
		m.hasLoop = true
		analyzeExpr(st.cond, m)
		for _, x := range st.body {
			analyzeStmt(x, m)
		}
	case *forStmt:
		m.hasLoop = true
		if st.init != nil {
			analyzeStmt(st.init, m)
		}
		analyzeExpr(st.cond, m)
		if st.step != nil {
			analyzeStmt(st.step, m)
		}
		for _, x := range st.body {
			analyzeStmt(x, m)
		}
	case *forInStmt:
		m.hasLoop = true
		for _, x := range st.body {
			analyzeStmt(x, m)
		}
	case *returnStmt:
		analyzeExpr(st.x, m)
	case *exprStmt:
		analyzeExpr(st.x, m)
	case *logStmt:
		analyzeExpr(st.x, m)
	case *deleteStmt:
		m.impure = true // free
	case *indexAssignStmt:
		m.impure = true // 堆数组写入
		analyzeExpr(st.idx, m)
		analyzeExpr(st.x, m)
	case *fieldAssignStmt:
		m.impure = true
		analyzeExpr(st.recv, m)
		analyzeExpr(st.x, m)
	case *tryStmt:
		for _, x := range st.then {
			analyzeStmt(x, m)
		}
		for _, x := range st.catch {
			analyzeStmt(x, m)
		}
	}
}

// programMeta 建立函数元信息表（含 main）。
func programMeta(funcs []*funcDef, mainStmts []stmt) map[string]*fnMeta {
	meta := make(map[string]*fnMeta, len(funcs)+1)
	for _, fd := range funcs {
		m := &fnMeta{}
		for _, s := range fd.body {
			analyzeStmt(s, m)
		}
		meta[fd.name] = m
	}
	mm := &fnMeta{}
	for _, s := range mainStmts {
		analyzeStmt(s, mm)
	}
	meta["main"] = mm
	return meta
}

// reachesSelf 判断调用图是否形成含 name 的环（DFS，仅在已定义函数间走）。
func reachesSelf(name string, meta map[string]*fnMeta) bool {
	for _, c := range meta[name].callees {
		if _, isFn := meta[c]; !isFn {
			continue
		}
		seen := map[string]bool{}
		var walk func(n string) bool
		walk = func(n string) bool {
			if n == name {
				return true
			}
			if seen[n] {
				return false
			}
			seen[n] = true
			for _, cc := range meta[n].callees {
				if _, ok := meta[cc]; ok && walk(cc) {
					return true
				}
			}
			return false
		}
		if walk(c) {
			return true
		}
	}
	return false
}

// fnAttrs 返回函数定义属性串（mustprogress/norecurse/memory(none)）。
func fnAttrs(name string, meta map[string]*fnMeta) string {
	m := meta[name]
	if m == nil {
		return ""
	}
	attrs := ""
	if m.hasLoop {
		attrs += " mustprogress"
	}
	if !m.hasMeth && !reachesSelf(name, meta) {
		attrs += " norecurse"
	}
	if len(m.callees) == 0 && !m.impure {
		// 纯算术函数（无调用、无 IO、无堆访问）：仅操作局部与参数——LLVM 可跨调用消除/内联
		attrs += " memory(none)"
	}
	return attrs
}

// scanAssigned 收集语句中所有被赋值（assignStmt）的变量名 —— 决定哪些 decl 可 SSA 直通。
func scanAssigned(stmts []stmt, out map[string]bool) {
	for _, s := range stmts {
		switch st := s.(type) {
		case *assignStmt:
			out[st.name] = true
		case *ifStmt:
			scanAssigned(st.then, out)
			scanAssigned(st.els, out)
		case *whileStmt:
			scanAssigned(st.body, out)
		case *forStmt:
			if st.init != nil {
				scanAssigned([]stmt{st.init}, out)
			}
			if st.step != nil {
				scanAssigned([]stmt{st.step}, out)
			}
			scanAssigned(st.body, out)
		case *forInStmt:
			out[st.name] = true // 循环变量每次迭代被写入
			scanAssigned(st.body, out)
		case *tryStmt:
			scanAssigned(st.then, out)
			scanAssigned(st.catch, out)
		}
	}
}

// constEval 递归求值常量表达式（字面量算术折叠）。
func constEval(x *expr) (int64, bool) {
	if x == nil {
		return 0, false
	}
	if x.kind == kInt {
		return x.i, true
	}
	if x.kind == kBin {
		l, ok1 := constEval(x.l)
		r, ok2 := constEval(x.r)
		if !ok1 || !ok2 {
			return 0, false
		}
		switch x.op {
		case "+":
			return l + r, true
		case "-":
			return l - r, true
		case "*":
			return l * r, true
		case "/":
			if r != 0 {
				return l / r, true
			}
		case "%":
			if r != 0 {
				return l % r, true
			}
		case "<<":
			return int64(int32(l) << uint(r&31)), true
		case ">>":
			return int64(int32(l) >> uint(r&31)), true
		}
	}
	return 0, false
}

// ---------- 运行期助手（按需追加到模块尾部） ----------

// intToStrHelper 是 int.toString() 的运行时助手：malloc(12) + snprintf("%d")；
// 12 字节足够 int32 极值（-2147483648 + NUL）。
const intToStrHelper = `@.ql.fmt.d = private unnamed_addr constant [3 x i8] c"%d\00", align 1
define i8* @ql_int_to_str(i32 %v) {
entry:
  %b = call i8* @malloc(i64 12)
  %f = getelementptr inbounds [3 x i8], [3 x i8]* @.ql.fmt.d, i64 0, i64 0
  %n = call i32 (i8*, i64, i8*, ...) @snprintf(i8* %b, i64 12, i8* %f, i32 %v)
  ret i8* %b
}
`

// floatToStrHelper 是 float.toString()/打印的运行时助手：与解释器
// strconv.FormatFloat(v,'f',-1,64) 一致 —— 取最短可回读（round-trip）的定点小数。
// 实现：%.0f..%.17f 逐个 snprintf，用 strtod 回读校验，取首个能精确还原的精度。
const floatToStrHelper = `@.ql.fmt.f = private unnamed_addr constant [5 x i8] c"%.*f\00", align 1
define i8* @ql_float_to_str(double %v) {
entry:
  %buf = call i8* @malloc(i64 40)
  %fmt = getelementptr inbounds [5 x i8], [5 x i8]* @.ql.fmt.f, i64 0, i64 0
  br label %loop
loop:
  %p = phi i32 [ 0, %entry ], [ %pn, %next ]
  %n = call i32 (i8*, i64, i8*, ...) @snprintf(i8* %buf, i64 40, i8* %fmt, i32 %p, double %v)
  %back = call double @strtod(i8* %buf, i8** null)
  %same = fcmp oeq double %back, %v
  br i1 %same, label %done, label %next
next:
  %pn = add i32 %p, 1
  %over = icmp sgt i32 %pn, 17
  br i1 %over, label %done, label %loop
done:
  ret i8* %buf
}
`

// panicHelper 是运行期错误的打印与退出（与解释器 ReportError 的 "error: ..." 一致）。
const panicHelper = `@.ql.panic.fmt = private unnamed_addr constant [22 x i8] c"error: %s at line %d\0A\00", align 1
@.ql.panic.idx = private unnamed_addr constant [71 x i8] c"error: IndexOutOfBoundsError: index %d out of range [0,%d) at line %d\0A\00", align 1
define void @ql_panic(i8* %msg, i32 %line) {
entry:
  %buf = alloca [512 x i8], align 16
  %p = getelementptr inbounds [512 x i8], [512 x i8]* %buf, i64 0, i64 0
  %fmt = getelementptr inbounds [22 x i8], [22 x i8]* @.ql.panic.fmt, i64 0, i64 0
  %n = call i32 (i8*, i64, i8*, ...) @snprintf(i8* %p, i64 512, i8* %fmt, i8* %msg, i32 %line)
  %n64 = sext i32 %n to i64
  %w = call i64 @write(i32 2, i8* %p, i64 %n64)
  call void @exit(i32 1)
  unreachable
}
define void @ql_panic_index(i32 %idx, i32 %size, i32 %line) {
entry:
  %buf = alloca [512 x i8], align 16
  %p = getelementptr inbounds [512 x i8], [512 x i8]* %buf, i64 0, i64 0
  %fmt = getelementptr inbounds [71 x i8], [71 x i8]* @.ql.panic.idx, i64 0, i64 0
  %n = call i32 (i8*, i64, i8*, ...) @snprintf(i8* %p, i64 512, i8* %fmt, i32 %idx, i32 %size, i32 %line)
  %n64 = sext i32 %n to i64
  %w = call i64 @write(i32 2, i8* %p, i64 %n64)
  call void @exit(i32 1)
  unreachable
}
`
