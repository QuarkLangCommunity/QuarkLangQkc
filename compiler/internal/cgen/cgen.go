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
	kRunner   // taskm.merge 的 runner 函数引用（i8* 函数指针）
	kNull     // null 字面量（指针零值）
	kMemoCall // 签名调用：f(args) @memorize（记忆化包装）
	kMerge    // taskm.merge / t.merge：i64 槽携带 0..4 个实参
	kTable    // HashTable 方法调用（put/get/contains/remove/size/keys）
	kNewRef   // new T：分配单元素存储，返回 T& 指针
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

	call   *callExpr   // kCall / kMemoCall
	memo   *memoExpr   // kMemoCall
	tbl    *tableExpr  // kTable
	lst    *listLit    // kList
	idx    *indexExpr  // kIndex
	method *methodExpr // kMethod
	sl     *structLit  // kStructLit
	field  *fieldExpr  // kField

	sc     strConst // 预注册的字符串常量（kString）
	strcat bool     // kBin "+" 且为 String 拼接（有 ql_strcat 副作用）
	line   int      // 运行期错误定位（除零等）

	ifaceBox string // 非空：把具体 struct 值装箱成接口值（值为 vtable 符号）
	anyBox   string // 非空：把该具体类型的值装箱成 interface{}（RTTI 描述符按类型发射）
	noDeref  bool   // kIdent 指向 T& 变量：取指针本身而非解引用值（指针比较/重绑定）
}

type indexExpr struct {
	recv *expr // List 表达式（变量/struct 字段/…）
	i    *expr
}

type methodExpr struct {
	recv   *expr
	name   string // 方法名
	args   []*expr
	sig    string // IR 函数名（lowering 解析后填入）
	isSelf bool   // 实例方法（需要 receiver 实参）
	byVal  bool   // 实参按值传递（运算符重载：解释器按值传入操作数）

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
	thru bool // 目标是 T& 变量且 RHS 是 T 值：写穿指针指向的单元
}

type indexAssignStmt struct {
	recv *expr // List 表达式（变量/struct 字段/…）
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
	name      string // IR 函数名
	args      []*expr
	byValArgs bool // 实参按值传递（签名 memorize 包装：解释器 derefArgs 后调用）
}

// tableExpr 是 HashTable 方法调用（kTable）的上下文。
type tableExpr struct {
	recv *expr   // 表句柄（i8*）
	name string  // put/get/contains/remove/size/keys
	args []*expr // put: [key, value]；其余: [key]（size/keys 无参）
	keyT string  // 键静态类型
	valT string  // 值静态类型
}

// isTableT 判断语言类型是否 HashTable<...>。
func isTableT(t string) bool { return strings.HasPrefix(strings.TrimSpace(t), "HashTable<") }

// memoExpr 是签名 memorize 调用（kMemoCall）的上下文。
type memoExpr struct {
	handle *expr   // memorize 实例（i8* 句柄）
	skips  []*expr // @ 显式 prefix 实参：求值丢弃（解释器放入 prefix 记录）
}

type funcParam struct {
	name  string
	typ   string
	copyd bool // copyd 形参：绑定时深拷贝到 callee 本地单元（不回写调用方）
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

// runnerDef 是 taskm.merge 的执行器（qthreads.c 以 void (*)(long long ×4) 调用）。
type runnerDef struct {
	fn     string   // 目标函数 IR 名
	params []string // 目标函数形参类型（≤4；运行时以 i64 槽携带）
}

// emitRunners 生成 taskm.merge 的执行器：把运行时 i64 槽还原成目标函数形参单元，
// 再按引用调用目标函数（返回值丢弃）。支持 0..4 个任意可 lower 类型的形参。
func (e *emitter) emitRunners(rs []runnerDef) string {
	if len(rs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, r := range rs {
		var sig, pre strings.Builder
		callArgs := make([]string, 0, len(r.params))
		for k, p := range r.params {
			fmt.Fprintf(&sig, ", i64 %%a%d", k)
			elem := e.ir(p)
			slot := fmt.Sprintf("%%a%d.slot", k)
			fmt.Fprintf(&pre, "  %s = alloca %s, align 8\n", slot, elem)
			switch p {
			case "int":
				fmt.Fprintf(&pre, "  %%a%d.v = trunc i64 %%a%d to i32\n", k, k)
				fmt.Fprintf(&pre, "  store i32 %%a%d.v, i32* %s\n", k, slot)
			case "bool":
				fmt.Fprintf(&pre, "  %%a%d.v = trunc i64 %%a%d to i1\n", k, k)
				fmt.Fprintf(&pre, "  store i1 %%a%d.v, i1* %s\n", k, slot)
			case "float":
				fmt.Fprintf(&pre, "  %%a%d.v = bitcast i64 %%a%d to double\n", k, k)
				fmt.Fprintf(&pre, "  store double %%a%d.v, double* %s\n", k, slot)
			case "String", "pointer", "Channel", "channel", "runner", "memorize":
				fmt.Fprintf(&pre, "  %%a%d.v = inttoptr i64 %%a%d to i8*\n", k, k)
				fmt.Fprintf(&pre, "  store i8* %%a%d.v, i8** %s\n", k, slot)
			case "long":
				fmt.Fprintf(&pre, "  store i64 %%a%d, i64* %s\n", k, slot)
			case "thread":
				fmt.Fprintf(&pre, "  %%a%d.v = trunc i64 %%a%d to i32\n", k, k)
				fmt.Fprintf(&pre, "  store i32 %%a%d.v, i32* %s\n", k, slot)
			default:
				fmt.Fprintf(&pre, "  %%a%d.v = inttoptr i64 %%a%d to %s\n", k, k, elem)
				fmt.Fprintf(&pre, "  store %s %%a%d.v, %s* %s\n", elem, k, elem, slot)
			}
			callArgs = append(callArgs, elem+"* "+slot)
		}
		fmt.Fprintf(&sb, "define void @runner_%d(%s) {\nentry:\n%s  call void @%s(%s)\n  ret void\n}\n",
			i, strings.TrimPrefix(sig.String(), ", "), pre.String(), r.fn, strings.Join(callArgs, ", "))
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
	param  bool   // 函数参数：值寄存器（历史形态，by-ref 后不再产生）
	direct bool   // SSA 直通（单赋值标量）
	ref    bool   // reg 指向调用方的存储（按引用形参）：写入穿透到调用方
}

type structField struct {
	name string
	typ  string
}

type funcSig struct {
	name   string
	params []funcParam
	ret    string
	byRef  bool // 用户函数：形参按引用传递（LLVM 层形参类型是「值类型*」）
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
	deepCopied     map[string]bool // 已发射的深拷贝助手（按需）
	rtti           map[string]bool // 已发射的 interface{} 类型描述符（按需）
	rttiFns        map[string]bool // 已发射的 struct 打印助手（按需）
	hasList        bool
	hasListS       bool
	hasEmpty       bool
	hasIface       bool
	hasRT          bool
	ifaces         map[string]bool
	vtables        []*vtableDef
	needIntToStr   bool
	needFloatToStr bool
	needLLongToStr bool
	needPtrToStr   bool
	needStrCmp     bool
	needPanic      bool
}

func newEmitter(lp *lowered) *emitter {
	e := &emitter{
		fnMeta:     programMeta(lp.funcs, lp.mainStmts),
		vars:       map[string]varSlot{},
		structs:    map[string][]structField{},
		sigs:       map[string]*funcSig{},
		strs:       map[string]strConst{},
		deepCopied: map[string]bool{},
		rtti:       map[string]bool{},
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
		// 用户函数：形参一律按引用传递（语言语义），LLVM 层形参类型 = 值类型*
		e.sigs[fd.name] = &funcSig{name: fd.name, params: fd.params, ret: fd.ret, byRef: true}
	}
	e.ifaces = map[string]bool{"interface{}": true} // tAny：%Iface 值 + RTTI 描述符
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
	e.sigs["ql_merge"] = &funcSig{name: "ql_merge", params: []funcParam{
		{typ: "int"}, {typ: "runner"}, {typ: "long"}, {typ: "long"}, {typ: "long"}, {typ: "long"},
		{typ: "long"}, {typ: "long"}, {typ: "long"}, {typ: "long"}}, ret: "void"}
	e.sigs["ql_send"] = &funcSig{name: "ql_send", params: []funcParam{{typ: "Channel"}, {typ: "int"}}, ret: "int"}
	e.sigs["ql_recv"] = &funcSig{name: "ql_recv", params: []funcParam{{typ: "Channel"}}, ret: "int"}
	// 签名 memorize（builtin，非用户函数：按值调用）
	e.sigs["ql_memo_new"] = &funcSig{name: "ql_memo_new", ret: "memorize"}
	// HashTable 构造/键列表（其余方法由 compileTable 直接发射）
	e.sigs["ql_table_new"] = &funcSig{name: "ql_table_new", ret: "HashTable"}
	e.sigs["ql_table_keys"] = &funcSig{name: "ql_table_keys", params: []funcParam{{typ: "HashTable"}}, ret: "List<String>"}
	// String 内建方法
	str := func(name, ret string, params ...string) {
		ps := make([]funcParam, 0, len(params))
		for _, p := range params {
			ps = append(ps, funcParam{typ: p})
		}
		e.sigs[name] = &funcSig{name: name, params: ps, ret: ret}
	}
	str("ql_str_size", "int", "String")
	str("ql_str_contains", "cbool", "String", "String")
	str("ql_str_startswith", "cbool", "String", "String")
	str("ql_str_endswith", "cbool", "String", "String")
	str("ql_str_indexof", "int", "String", "String")
	str("ql_str_sub", "String", "String", "int", "int", "int")
	str("ql_str_charat", "String", "String", "int", "int")
	str("ql_str_trim", "String", "String", "int")
	str("ql_str_lower", "String", "String")
	str("ql_str_upper", "String", "String")
	str("ql_str_replace", "String", "String", "String", "String")
	str("ql_str_toint", "int", "String", "int")
	str("ql_str_tofloat", "double", "String", "int")
	str("ql_str_split", "List<String>", "String", "String")
	str("ql_list_int_sort", "void", "intptr", "int", "int")
	str("ql_list_int_str", "String", "intptr", "int", "int")
	return e
}

// builtinDecls 是发射器固定声明的符号（FFI 重复声明会报 invalid redefinition）。
var builtinDecls = map[string]bool{
	"printf": true, "malloc": true, "calloc": true, "free": true, "realloc": true,
	"gettimeofday": true, "ql_strcat": true, "snprintf": true, "strcmp": true,
	"strtod": true, "write": true, "exit": true,
}

// linkCandidates 把 library 名映射为**候选链接参数组**（按优先级）。
// qkc 逐组尝试，全部失败时给出「库名 + 尝试过的参数」诊断（不透传 clang 原文了事）。
func linkCandidates(lib string) []string {
	name := strings.TrimSpace(lib)
	// 路径 / 带扩展名：按文件链接
	if strings.ContainsAny(name, "/.") {
		return []string{name}
	}
	base := strings.TrimPrefix(name, "lib")
	switch strings.ToLower(base) {
	case "c":
		// libc 默认已链接；显式 -lc 无害（且 libc.so 一定存在）
		return []string{"-lc"}
	case "m":
		return []string{"-lm"}
	case "dl", "rt", "pthread", "stdc++", "gcc_s", "z", "curl", "sqlite3", "png", "jpeg":
		return []string{"-l" + base}
	case "gl":
		// 解释器会试 libGL.so.1 / libgl.so.1；链接侧同样给大小写与 soname 兜底
		return []string{"-lGL", "-l:libGL.so.1", "-lgl"}
	case "vulkan":
		return []string{"-lvulkan", "-l:libvulkan.so.1"}
	case "clegrt":
		// 项目本地运行时优先（产物树/assets），再退回系统路径
		return []string{"-L. -l" + base, "-l" + base}
	}
	return []string{"-l" + base, "-L. -l" + base}
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

// ensureListS 保证 List<String> 的 LLVM 定义已发射（元素是 i8* 指针）。
func (e *emitter) ensureListS() {
	if e.hasListS {
		return
	}
	e.hasListS = true
	e.types.WriteString("%ListS = type { i8**, i32, i32 }\n") // buf, head, tail
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
	case "pointer", "null":
		return "i8*"
	case "thread":
		return "i32"
	case "Channel", "channel":
		return "i8*"
	case "runner":
		return "i8*"
	case "memorize":
		return "i8*" // 签名实例句柄（ql_memo_new）
	case "HashTable":
		return "i8*" // HashTable<K,V> 句柄（上面已判过 HasPrefix）
	case "intptr":
		return "i32*"
	}
	if t == "List<int>" {
		e.ensureList()
		return "%List*"
	}
	if t == "List<String>" {
		e.ensureListS()
		return "%ListS*"
	}
	if _, ok := e.structs[t]; ok {
		e.ensureStruct(t)
		return "%" + tyName(t) + "*"
	}
	if isTableT(t) {
		return "i8*"
	}
	if _, base, ok := ptrRefBase(t); ok {
		return e.ir(base) + "*"
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
	case "float", "double", "long", "pointer", "null", "String",
		"List<int>", "List<String>", "List<float>", "List<bool>", "memorize":
		return 8
	case "HashTable":
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
	case "int", "thread", "long":
		return "0"
	case "bool":
		return "false"
	case "float":
		return "0.0"
	case "String":
		return e.emptyString()
	case "memorize":
		return "null"
	case "HashTable":
		return "null"
	}
	if isTableT(t) {
		return "null"
	}
	if _, _, ok := ptrRefBase(t); ok {
		return "null"
	}
	if e.ifaces[t] {
		return "zeroinitializer"
	}
	return "null" // List / struct / pointer：引用零值 = null
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
	e.decls.WriteString("declare void @ql_merge(i32, i8*, i64, i64, i64, i64, i64, i64, i64, i64)\n")
	e.decls.WriteString("declare void @ql_block(i32)\n")
	e.decls.WriteString("declare i32 @ql_done(i32)\n")
	e.decls.WriteString("declare i8* @ql_channel_new(i32)\n")
	e.decls.WriteString("declare i32 @ql_send(i8*, i32)\n")
	e.decls.WriteString("declare i32 @ql_recv(i8*)\n")
	// String 内建方法运行时（qthreads.c：UTF-8 感知，与解释器同语义）
	e.decls.WriteString("declare i32 @ql_str_size(i8*)\n")
	e.decls.WriteString("declare i32 @ql_str_contains(i8*, i8*)\n")
	e.decls.WriteString("declare i32 @ql_str_startswith(i8*, i8*)\n")
	e.decls.WriteString("declare i32 @ql_str_endswith(i8*, i8*)\n")
	e.decls.WriteString("declare i32 @ql_str_indexof(i8*, i8*)\n")
	e.decls.WriteString("declare i8* @ql_str_sub(i8*, i32, i32, i32)\n")
	e.decls.WriteString("declare i8* @ql_str_charat(i8*, i32, i32)\n")
	e.decls.WriteString("declare i8* @ql_str_trim(i8*, i32)\n")
	e.decls.WriteString("declare i8* @ql_str_lower(i8*)\n")
	e.decls.WriteString("declare i8* @ql_str_upper(i8*)\n")
	e.decls.WriteString("declare i8* @ql_str_replace(i8*, i8*, i8*)\n")
	e.decls.WriteString("declare i32 @ql_str_toint(i8*, i32)\n")
	e.decls.WriteString("declare double @ql_str_tofloat(i8*, i32)\n")
	// interface{}（tAny）运行期：RTTI 描述符分派 / 装箱值拆箱与相等
	e.decls.WriteString("declare i8* @ql_any_str(i8*, i8**)\n")
	e.decls.WriteString("declare i32 @ql_any_eq(i8*, i8**, i8*, i8**)\n")
	e.decls.WriteString("declare i32 @ql_any_int(i8*, i8**)\n")
	e.decls.WriteString("declare double @ql_any_float(i8*, i8**)\n")
	e.decls.WriteString("declare i32 @ql_any_bool(i8*, i8**)\n")
	e.decls.WriteString("declare i8* @ql_any_string(i8*, i8**)\n")
	e.decls.WriteString("declare i8* @ql_any_str_int(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_str_float(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_str_bool(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_str_string(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_str_list(i8*)\n")
	e.decls.WriteString("declare i8* @ql_long_to_str(i64)\n")
	// HashTable<K,V> 运行期
	e.decls.WriteString("declare i8* @ql_table_new()\n")
	e.decls.WriteString("declare i8* @ql_table_key(i8*, i8**)\n")
	e.decls.WriteString("declare void @ql_table_put(i8*, i8*, i8*, i8**)\n")
	e.decls.WriteString("declare i8* @ql_table_get(i8*, i8*, i8**)\n")
	e.decls.WriteString("declare i32 @ql_table_contains(i8*, i8*)\n")
	e.decls.WriteString("declare void @ql_table_remove(i8*, i8*)\n")
	e.decls.WriteString("declare i32 @ql_table_size(i8*)\n")
	e.decls.WriteString("declare i8* @ql_table_keys(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_copy_int(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_copy_float(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_copy_bool(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_copy_string(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_copy_list(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_copy_identity(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_copy_nil(i8*)\n")
	e.decls.WriteString("declare i8* @ql_any_ptr(i8*, i8**)\n")
	e.decls.WriteString("declare i8* @ql_list_str_str(i8**, i32, i32)\n")
	e.ensureListS() // declare 引用了 %ListS，类型定义必须先于声明发射
	e.decls.WriteString("declare %ListS* @ql_str_split(i8*, i8*)\n")
	// 签名 memorize 运行时（@mb() 记忆化：int 键 → int 值）
	e.decls.WriteString("declare i8* @ql_memo_new()\n")
	e.decls.WriteString("declare i32 @ql_memo_get(i8*, i32, i32*, i32*)\n")
	e.decls.WriteString("declare void @ql_memo_put(i8*, i32, i32*, i32)\n")
	// List 内建方法运行时（int 元素；可见区间 head..tail）
	e.decls.WriteString("declare void @ql_list_int_sort(i32*, i32, i32)\n")
	e.decls.WriteString("declare i8* @ql_list_int_str(i32*, i32, i32)\n\n")

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
		// 格式：; qkc-link: <库名> => <候选参数> => <候选参数>
		link.WriteString("; qkc-link: " + ed.lib)
		for _, cand := range linkCandidates(ed.lib) {
			link.WriteString(" => " + cand)
		}
		link.WriteString("\n")
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
	// 预扫描被赋值变量（决定 SSA 直通）；按引用传递的实参必须有自己的存储
	// （否则 callee 的写入无法回写），因此与「被赋值」同等对待。
	e.assigned = map[string]bool{}
	for _, fd := range lp.funcs {
		scanAssigned(fd.body, e.assigned)
		scanByRefArgs(fd.body, e.sigs, e.assigned)
	}
	scanAssigned(lp.mainStmts, e.assigned)
	scanByRefArgs(lp.mainStmts, e.sigs, e.assigned)

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
	if e.needPtrToStr {
		helpers += ptrToStrHelper
	}
	if e.needPanic {
		helpers += panicHelper
	}
	// runner 的形参类型可能按需发射 struct 定义，必须在读 e.types 之前生成
	runners := e.emitRunners(lp.runners)
	return link.String() + e.types.String() + e.globals.String() + e.decls.String() +
		helpers + e.helpers.String() + e.bodies.String() + e.emitVtables(e.vtables) + runners
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
	// 形参一律按引用传递：LLVM 形参是「指向调用方实参单元」的指针，callee 读写
	// 直接 load/store 穿透（写回调用方）。copyd 形参在入口深拷贝到本地单元。
	if fd.selfTyp != "" {
		addParam(e.ir(fd.selfTyp)+"*", "%self")
		e.vars[fd.selfParam] = varSlot{reg: "%self", typ: fd.selfTyp, ref: true}
	}
	paramRegs := make([]string, 0, len(fd.params))
	for i, p := range fd.params {
		reg := fmt.Sprintf("%%p%d", i)
		addParam(e.ir(p.typ)+"*", reg)
		paramRegs = append(paramRegs, reg)
		e.vars[p.name] = varSlot{reg: reg, typ: p.typ, ref: true}
	}
	sig.WriteString(")" + fnAttrs(fd.name, e.fnMeta) + "\n")
	e.cur = "entry"
	e.body.WriteString("entry:\n")
	for i, p := range fd.params {
		if p.copyd {
			e.bindCopydParam(p, paramRegs[i])
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

// ---------- 按引用形参（call-by-reference） ----------

// isListType 判断语言类型是否 List<...>（当前后端只 lower List<int>）。
func isListType(t string) bool { return strings.HasPrefix(t, "List<") }

// isIfaceTypeE 判断类型是否接口（发射器侧的接口名表）。
func (e *emitter) isIfaceTypeE(t string) bool { return e.ifaces[t] }

// bindCopydParam 绑定 copyd 形参：把调用方单元的值深拷贝到 callee 本地单元，
// callee 内的写入只影响副本（解释器：声明写法绑裸值、类型写法绑 Copyd 包装值）。
func (e *emitter) bindCopydParam(p funcParam, src string) {
	typ := p.typ
	llt := e.ir(typ)
	slot := e.newReg()
	e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(typ))
	v := e.newReg()
	e.emitInstr("%s = load %s, %s* %s", v, llt, llt, src)
	switch {
	case e.isStruct(typ):
		sym := e.emitDeepCopyStruct(typ)
		cpB, endB := e.newBlock(), e.newBlock()
		nn := e.newReg()
		e.emitInstr("%s = icmp ne %s %s, null", nn, llt, v)
		e.emitInstr("br i1 %s, label %%%s, label %%%s", nn, cpB, endB)
		e.setBlock(cpB)
		cp := e.newReg()
		e.emitInstr("%s = call %s @%s(%s %s)", cp, llt, sym, llt, v)
		e.emitInstr("store %s %s, %s* %s", llt, cp, llt, slot)
		e.emitInstr("br label %%%s", endB)
		e.setBlock(endB)
	case isListType(typ):
		cp := e.emitDeepCopyList(v)
		e.emitInstr("store %s %s, %s* %s", llt, cp, llt, slot)
	default:
		e.emitInstr("store %s %s, %s* %s", llt, v, llt, slot)
	}
	e.vars[p.name] = varSlot{reg: slot, typ: typ}
}

// emitDeepCopyStruct 发射 struct 深拷贝助手并返回符号名（按需、幂等）。
// 语义与解释器 copydCopy 一致：字段递归复制（struct 递归、List 深拷贝、
// String/标量/接口按值）；空指针字段保持 null。
func (e *emitter) emitDeepCopyStruct(typ string) string {
	sym := "ql_copyd_struct_" + tyName(typ)
	if e.deepCopied == nil {
		e.deepCopied = map[string]bool{}
	}
	if e.deepCopied[sym] {
		return sym
	}
	e.deepCopied[sym] = true
	e.ensureStruct(typ)
	ptrTy := e.ir(typ)
	elem := e.irElem(typ)
	savedBody, savedVars, savedCur := e.body, e.vars, e.cur
	e.body = strings.Builder{}
	e.vars = map[string]varSlot{}
	e.blockCount = 0
	e.cur = "entry"
	e.body.WriteString("entry:\n")
	obj := e.structAlloc(typ)
	dst := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to %s", dst, obj, ptrTy)
	for i, f := range e.structs[typ] {
		ft := f.typ
		sg := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %s, %s* %%src, i32 0, i32 %d", sg, elem, elem, i)
		dg := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %s, %s* %s, i32 0, i32 %d", dg, elem, elem, dst, i)
		llt := e.ir(ft)
		sv := e.newReg()
		e.emitInstr("%s = load %s, %s* %s", sv, llt, llt, sg)
		switch {
		case e.isStruct(ft):
			cpB, endB := e.newBlock(), e.newBlock()
			nn := e.newReg()
			e.emitInstr("%s = icmp ne %s %s, null", nn, llt, sv)
			e.emitInstr("br i1 %s, label %%%s, label %%%s", nn, cpB, endB)
			e.setBlock(cpB)
			inner := e.emitDeepCopyStructCall(sv, ft)
			e.emitInstr("store %s %s, %s* %s", llt, inner, llt, dg)
			e.emitInstr("br label %%%s", endB)
			e.setBlock(endB)
		case isListType(ft):
			cpB, endB := e.newBlock(), e.newBlock()
			nn := e.newReg()
			e.emitInstr("%s = icmp ne %s %s, null", nn, llt, sv)
			e.emitInstr("br i1 %s, label %%%s, label %%%s", nn, cpB, endB)
			e.setBlock(cpB)
			inner := e.emitDeepCopyList(sv)
			e.emitInstr("store %s %s, %s* %s", llt, inner, llt, dg)
			e.emitInstr("br label %%%s", endB)
			e.setBlock(endB)
		default:
			// 标量 / String / 接口：按值复制（String 不可变；接口字段在 lower 阶段已拦截）
			e.emitInstr("store %s %s, %s* %s", llt, sv, llt, dg)
		}
	}
	e.emitInstr("ret %s %s", ptrTy, dst)
	body := e.body.String()
	e.body, e.vars, e.cur = savedBody, savedVars, savedCur
	e.helpers.WriteString("define " + ptrTy + " @" + sym + "(" + ptrTy + " noundef %src) {\n" + body + "}\n")
	return sym
}

// emitDeepCopyStructCall 调用 struct 深拷贝助手。
func (e *emitter) emitDeepCopyStructCall(src, typ string) string {
	sym := e.emitDeepCopyStruct(typ)
	r := e.newReg()
	e.emitInstr("%s = call %s @%s(%s %s)", r, e.ir(typ), sym, e.ir(typ), src)
	return r
}

// emitDeepCopyList 发射 List 深拷贝助手（在 helper 函数里完成），返回新 List 指针。
// 可见区间 head..tail 复制到新缓冲区（head=0, tail=size），解释器 copyDeep 语义。
func (e *emitter) emitDeepCopyList(src string) string {
	e.ensureDeepCopyList()
	r := e.newReg()
	e.emitInstr("%s = call %%List* @ql_copyd_list(%%List* %s)", r, src)
	return r
}

// ensureDeepCopyList 按需发射 @ql_copyd_list。
func (e *emitter) ensureDeepCopyList() {
	if e.deepCopied["ql_copyd_list"] {
		return
	}
	e.ensureList()
	e.deepCopied["ql_copyd_list"] = true
	savedBody, savedVars := e.body, e.vars
	e.body = strings.Builder{}
	e.vars = map[string]varSlot{}
	e.blockCount = 0
	e.body.WriteString("entry:\n")
	nn := e.newReg()
	e.emitInstr("%s = icmp ne %%List* %%src, null", nn)
	nB, yB := e.newBlock(), e.newBlock()
	e.emitInstr("br i1 %s, label %%%s, label %%%s", nn, yB, nB)
	e.setBlock(nB)
	e.emitInstr("ret %%List* null")
	e.setBlock(yB)
	obj := e.newReg()
	e.emitInstr("%s = call i8* @calloc(i64 1, i64 16)", obj)
	lo := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to %%List*", lo, obj)
	head, size := e.listHeadSize("%src")
	srcBuf := e.listBuf("%src")
	// 新缓冲区：至少 1 个元素（与 List 字面量/append 的容量假设一致）
	cap0 := e.newReg()
	e.emitInstr("%s = icmp slt i32 %s, 1", cap0, size)
	capv := e.newReg()
	e.emitInstr("%s = select i1 %s, i32 1, i32 %s", capv, cap0, size)
	c64 := e.newReg()
	e.emitInstr("%s = sext i32 %s to i64", c64, capv)
	sz := e.newReg()
	e.emitInstr("%s = mul i64 %s, 4", sz, c64)
	buf := e.newReg()
	e.emitInstr("%s = call i8* @malloc(i64 %s)", buf, sz)
	np := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to i32*", np, buf)
	// 循环复制可见元素
	iv := e.newReg()
	e.emitInstr("%s = alloca i32, align 4", iv)
	e.emitInstr("store i32 0, i32* %s", iv)
	condB, bodyB, endB := e.newBlock(), e.newBlock(), e.newBlock()
	e.emitInstr("br label %%%s", condB)
	e.setBlock(condB)
	li := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", li, iv)
	cnd := e.newReg()
	e.emitInstr("%s = icmp slt i32 %s, %s", cnd, li, size)
	e.emitInstr("br i1 %s, label %%%s, label %%%s", cnd, bodyB, endB)
	e.setBlock(bodyB)
	li2 := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", li2, iv)
	si := e.newReg()
	e.emitInstr("%s = add i32 %s, %s", si, head, li2)
	si64 := e.toI64(si)
	li64 := e.toI64(li2)
	sp := e.newReg()
	e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", sp, srcBuf, si64)
	dp := e.newReg()
	e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", dp, np, li64)
	v := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", v, sp)
	e.emitInstr("store i32 %s, i32* %s", v, dp)
	ni := e.newReg()
	e.emitInstr("%s = add i32 %s, 1", ni, li2)
	e.emitInstr("store i32 %s, i32* %s", ni, iv)
	e.emitInstr("br label %%%s", condB)
	e.setBlock(endB)
	bf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 0", bf, lo)
	e.emitInstr("store i32* %s, i32** %s", np, bf)
	hf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 1", hf, lo)
	e.emitInstr("store i32 0, i32* %s", hf)
	tf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, lo)
	e.emitInstr("store i32 %s, i32* %s", size, tf)
	e.emitInstr("ret %%List* %s", lo)
	body := e.body.String()
	e.body, e.vars = savedBody, savedVars
	e.helpers.WriteString("define %List* @ql_copyd_list(%List* noundef %src) {\n" + body + "}\n")
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
	if x.memo != nil {
		e.preRegister(x.memo.handle)
		for _, a := range x.memo.skips {
			e.preRegister(a)
		}
	}
	if x.tbl != nil {
		e.preRegister(x.tbl.recv)
		for _, a := range x.tbl.args {
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
		e.preRegister(x.idx.recv)
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
	// 尾调用优化：return f(args) → tail call（尾递归栈 O(1)）。
	// 按引用传递后只有「实参地址在调用期间稳定」时才能尾调用：若某个实参是当前帧的
	// 临时 alloca，尾调用会让指针失效（LLVM tail call 规则），此时退回普通调用。
	if st.x.kind == kCall && st.x.call != nil && st.x.call.name != "sum" && st.x.call.name != "clock" {
		c := st.x.call
		argRegs, tailSafe := e.compileArgs(c)
		if tailSafe {
			retTy := e.ir(e.sigRet(c.name))
			r := e.newReg()
			e.body.WriteString(r)
			e.body.WriteString(" = tail call " + retTy + " @" + c.name + "(" + strings.Join(argRegs, ", ") + ")\n")
			e.emitInstr("ret %s %s", retTy, r)
			e.funcReturned = true
			return
		}
	}
	v, vt := e.compileExpr(st.x)
	v = e.coerce(v, vt, e.curRet)
	e.emitInstr("ret %s %s", e.ir(e.curRet), v)
	e.funcReturned = true
}

// compileArgs 编译调用实参，并按被调函数签名做隐式转换（int → float）。
// 返回 "类型 寄存器" 形式的列表；第二返回值表示全部实参是否 tail-call 安全
// （按引用传递且实参单元在调用方帧内时，尾调用会让指针失效，必须禁用 TCO）。
func (e *emitter) compileArgs(c *callExpr) ([]string, bool) {
	sig := e.sigs[c.name]
	out := make([]string, 0, len(c.args))
	tailSafe := true
	for i, a := range c.args {
		want := "?"
		if sig != nil && i < len(sig.params) {
			want = sig.params[i].typ
		}
		if sig != nil && sig.byRef && !c.byValArgs {
			p, pt, safe := e.compileArgAddr(a, want)
			if !safe {
				tailSafe = false
			}
			out = append(out, e.ir(pt)+"* "+p)
			continue
		}
		v, vt := e.compileExpr(a)
		if want == "?" {
			want = vt
		}
		v = e.coerce(v, vt, want)
		// 变参（printf 风格）不在此列，本 IR 全部为定参
		out = append(out, e.ir(want)+" "+v)
	}
	return out, tailSafe
}

// compileArgAddr 为按引用传递的实参准备存储，返回（地址、语言类型、tail 安全性）。
//   - 左值实参（变量/字段/List 下标）且类型完全匹配 → 直接用其自身存储（写回调用方）；
//   - 其余（字面量、算术结果、需要 int→float 转换、需要装箱）→ 调用方临时单元，
//     callee 的写入不回写（与解释器「非左值实参是临时单元」一致）。
//
// tailSafe 表示该地址是否在整个调用期间稳定（堆对象/调用方存储 = 安全；
// 当前帧的 alloca 临时单元 = 尾调用会失效）。
func (e *emitter) compileArgAddr(x *expr, want string) (string, string, bool) {
	if x != nil && x.ifaceBox == "" {
		if p, t, safe, ok := e.lvalueAddr(x); ok && (want == "" || want == "?" || t == want) {
			return p, t, safe
		}
	}
	v, vt := e.compileExpr(x)
	if want == "" || want == "?" {
		want = vt
	}
	v = e.coerce(v, vt, want)
	llt := e.ir(want)
	slot := e.newReg()
	e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(want))
	e.emitInstr("store %s %s, %s* %s", llt, v, llt, slot)
	return slot, want, false
}

// lvalueAddr 尝试取表达式的左值地址（返回地址、语言类型、是否 tail-call 安全）。
// 只支持后端已有的三种左值：变量、struct 字段、List 下标（与解释器 evalArg 同集）。
func (e *emitter) lvalueAddr(x *expr) (string, string, bool, bool) {
	switch x.kind {
	case kIdent:
		info, ok := e.vars[x.s]
		if !ok || info.param || info.direct {
			return "", "", false, false // SSA 直通/寄存器变量没有可变存储
		}
		// ref=true：存储属于调用方（或堆），尾调用安全；本地 alloca 不安全
		return info.reg, info.typ, info.ref, true
	case kField:
		if x.field == nil || x.field.recv == nil {
			return "", "", false, false
		}
		obj, otyp := e.compileExpr(x.field.recv)
		if !e.isStruct(otyp) {
			return "", "", false, false
		}
		for i, f := range e.structs[otyp] {
			if f.name == x.field.name {
				g := e.newReg()
				e.emitInstr("%s = getelementptr inbounds %s, %s* %s, i32 0, i32 %d", g, e.irElem(otyp), e.irElem(otyp), obj, i)
				// struct 实例要么在堆上，要么是调用方的对象 → 地址稳定
				return g, f.typ, true, true
			}
		}
		return "", "", false, false
	case kIndex:
		if x.idx == nil || x.idx.recv == nil {
			return "", "", false, false
		}
		lo, lt := e.compileExpr(x.idx.recv)
		if lt == "List<String>" {
			p, head, size := e.listSRange(lo)
			i, it := e.compileExpr(x.idx.i)
			i = e.coerce(i, it, "int")
			e.emitBoundsCheck(i, size, x.line)
			idx := e.newReg()
			e.emitInstr("%s = add i32 %s, %s", idx, head, i)
			i64 := e.toI64(idx)
			g := e.newReg()
			e.emitInstr("%s = getelementptr inbounds i8*, i8** %s, i64 %s", g, p, i64)
			return g, "String", true, true
		}
		if !isListType(lt) {
			return "", "", false, false
		}
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
		return g2, "int", true, true // 缓冲区在堆上
	}
	return "", "", false, false
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
	// interface{} → 具体标量/String：运行期按 RTTI kind 校验后拆箱（不匹配则明确运行期错误）
	if from == "interface{}" {
		e.ensureIface()
		d := e.newReg()
		e.emitInstr("%s = extractvalue %%Iface %s, 0", d, reg)
		rt := e.newReg()
		e.emitInstr("%s = extractvalue %%Iface %s, 1", rt, reg)
		to2 := to
		if to2 == "double" {
			to2 = "float"
		}
		switch to2 {
		case "int":
			r := e.newReg()
			e.emitInstr("%s = call i32 @ql_any_int(i8* %s, i8** %s)", r, d, rt)
			return r
		case "float":
			r := e.newReg()
			e.emitInstr("%s = call double @ql_any_float(i8* %s, i8** %s)", r, d, rt)
			return r
		case "bool":
			r := e.newReg()
			e.emitInstr("%s = call i32 @ql_any_bool(i8* %s, i8** %s)", r, d, rt)
			b := e.newReg()
			e.emitInstr("%s = icmp ne i32 %s, 0", b, r)
			return b
		case "String":
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_any_string(i8* %s, i8** %s)", r, d, rt)
			return r
		}
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
	case from == "long" && to == "int":
		r := e.newReg()
		e.emitInstr("%s = trunc i64 %s to i32", r, reg)
		return r
	case from == "int" && to == "long":
		r := e.newReg()
		e.emitInstr("%s = sext i32 %s to i64", r, reg)
		return r
	case from == "null" && (to == "pointer" || to == "String" || to == "runner"):
		return reg
	}
	return reg
}

// ---------- 声明 / 赋值 ----------

func (e *emitter) emitDecl(st *declStmt) {
	// T& / pointer T：槽里放指针；new T 分配单元，无初值 → null
	if _, _, isRef := ptrRefBase(st.typ); isRef {
		slot := e.newReg()
		llt := e.ir(st.typ)
		e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(st.typ))
		v := "null"
		if st.init != nil {
			v, _ = e.compileExpr(st.init)
		}
		e.emitInstr("store %s %s, %s* %s", llt, v, llt, slot)
		e.vars[st.name] = varSlot{reg: slot, typ: st.typ}
		return
	}
	// HashTable<K,V>：i8* 句柄（构造已由 HashTable::new() 发射）
	if isTableT(st.typ) {
		slot := e.newReg()
		e.emitInstr("%s = alloca i8*, align 8", slot)
		v := "null"
		if st.init != nil {
			v, _ = e.compileExpr(st.init)
		}
		e.emitInstr("store i8* %s, i8** %s", v, slot)
		e.vars[st.name] = varSlot{reg: slot, typ: st.typ}
		return
	}
	switch st.typ {
	case "int", "bool", "float", "long", "String", "pointer", "thread", "Channel", "channel", "memorize":
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
	case "List<String>":
		e.ensureListS()
		slot := e.newReg()
		e.emitInstr("%s = alloca %%ListS*, align 8", slot)
		v := "null"
		if st.init != nil {
			v, _ = e.compileExpr(st.init)
		}
		e.emitInstr("store %%ListS* %s, %%ListS** %s", v, slot)
		e.vars[st.name] = varSlot{reg: slot, typ: st.typ}
	case "List<int>":
		e.ensureList()
		if st.init != nil && st.init.kind != kList {
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

// listSObj 取 List<String> 变量的对象指针。
func (e *emitter) listSObj(name string) string {
	e.ensureListS()
	info, ok := e.vars[name]
	if !ok {
		return "null"
	}
	if info.param || info.direct {
		return info.reg
	}
	r := e.newReg()
	e.emitInstr("%s = load %%ListS*, %%ListS** %s", r, info.reg)
	return r
}

// listSRange 取 List<String> 的可见区间与缓冲区。
func (e *emitter) listSRange(obj string) (buf, head, size string) {
	e.ensureListS()
	hf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 1", hf, obj)
	tf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 2", tf, obj)
	h := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", h, hf)
	t := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", t, tf)
	sz := e.newReg()
	e.emitInstr("%s = sub i32 %s, %s", sz, t, h)
	bf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 0", bf, obj)
	p := e.newReg()
	e.emitInstr("%s = load i8**, i8*** %s", p, bf)
	return p, h, sz
}

// listSSlot 把 List<String> 对象指针存进 alloca 槽（变量引用语义）。
func (e *emitter) listSSlot(objPtr string) string {
	e.ensureListS()
	slot := e.newReg()
	e.emitInstr("%s = alloca %%ListS*, align 8", slot)
	e.emitInstr("store %%ListS* %s, %%ListS** %s", objPtr, slot)
	return slot
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
	if st.thru {
		// T& 变量：写穿指针指向的单元；指针为 null（未初始化）时惰性分配单元
		// （解释器语义：int& q; q = 7 → q 自己持有该值）
		_, base, _ := ptrRefBase(info.typ)
		ptr := e.newReg()
		pllt := e.ir(info.typ)
		e.emitInstr("%s = load %s, %s* %s", ptr, pllt, pllt, info.reg)
		isnull := e.newReg()
		e.emitInstr("%s = icmp eq %s %s, null", isnull, pllt, ptr)
		raw := e.newReg()
		e.emitInstr("%s = call i8* @calloc(i64 1, i64 %d)", raw, e.sizeOf(base))
		blt := e.ir(base)
		fresh := e.newReg()
		e.emitInstr("%s = bitcast i8* %s to %s", fresh, raw, pllt)
		cell := e.newReg()
		e.emitInstr("%s = select i1 %s, %s %s, %s %s", cell, isnull, pllt, fresh, pllt, ptr)
		e.emitInstr("store %s %s, %s* %s", pllt, cell, pllt, info.reg)
		v, vt := e.compileExpr(st.x)
		v = e.coerce(v, vt, base)
		e.emitInstr("store %s %s, %s* %s", blt, v, blt, cell)
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
	// l[i] = v（接收者可以是任意 List<int>/List<String> 表达式）
	lo, lt := e.compileExpr(st.recv)
	if lt == "List<String>" {
		p, head, size := e.listSRange(lo)
		i, it := e.compileExpr(st.idx)
		i = e.coerce(i, it, "int")
		e.emitBoundsCheck(i, size, st.idx.line)
		idx := e.newReg()
		e.emitInstr("%s = add i32 %s, %s", idx, head, i)
		i64 := e.toI64(idx)
		g := e.newReg()
		e.emitInstr("%s = getelementptr inbounds i8*, i8** %s, i64 %s", g, p, i64)
		v, vt := e.compileExpr(st.x)
		v = e.coerce(v, vt, "String")
		e.emitInstr("store i8* %s, i8** %s", v, g)
		return
	}
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
	if info.ref {
		return // 按引用形参：存储属于调用方，不能在这里释放
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
	condB, bodyB, endB := e.newBlock(), e.newBlock(), e.newBlock()
	if st.typ == "String" {
		// List<String>：元素是 i8*（滑动的 head 游标语义与 List<int> 一致）
		lo := e.listSObj(st.list)
		_, _, _ = lo, condB, bodyB
		p, _, _ := e.listSRange(lo)
		bf := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 0", bf, lo)
		hf := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 1", hf, lo)
		tf := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 2", tf, lo)
		e.emitInstr("br label %%%s", condB)
		e.setBlock(condB)
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
		h64 := e.toI64(h2)
		ep := e.newReg()
		e.emitInstr("%s = getelementptr inbounds i8*, i8** %s, i64 %s", ep, p, h64)
		ev := e.newReg()
		e.emitInstr("%s = load i8*, i8** %s", ev, ep)
		e.emitInstr("store i8* %s, i8** %s", ev, slot)
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
		_ = bf
		return
	}
	lo := e.listObj(st.list)
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
		// T& 实参：null → "nil"（解释器 nil 值打印为 nil）；统一转 String 后 select
		ptrNil := ""
		if _, _, isRef := ptrRefBase(a.typ); isRef && a.kind == kIdent {
			ptr, pt := e.loadVar(a.s, true)
			ptrNil = e.newReg()
			e.emitInstr("%s = icmp eq %s %s, null", ptrNil, e.ir(pt), ptr)
		}
		vals = append(vals, av{v, t})
		switch t {
		case "int":
			if ptrNil != "" {
				e.needIntToStr = true
				s := e.newReg()
				e.emitInstr("%s = call i8* @ql_int_to_str(i32 %s)", s, v)
				vals[len(vals)-1] = av{e.nilSelect(ptrNil, s), "String"}
				fmts = append(fmts, "%s")
				break
			}
			fmts = append(fmts, "%d")
		case "long":
			fmts = append(fmts, "%lld")
		case "pointer", "null":
			e.needPtrToStr = true
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_ptr_to_str(i8* %s)", r, v)
			vals[len(vals)-1].val = r
			vals[len(vals)-1].typ = "String"
			fmts = append(fmts, "%s")
		case "List<String>":
			p, head, size := e.listSRange(v)
			tail := e.newReg()
			e.emitInstr("%s = add i32 %s, %s", tail, head, size)
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_list_str_str(i8** %s, i32 %s, i32 %s)", r, p, head, tail)
			vals[len(vals)-1].val = r
			vals[len(vals)-1].typ = "String"
			fmts = append(fmts, "%s")
		case "interface{}":
			// 运行期按 RTTI kind 分派到具体类型的字符串化（null → "nil"）
			d := e.newReg()
			e.emitInstr("%s = extractvalue %%Iface %s, 0", d, v)
			rt := e.newReg()
			e.emitInstr("%s = extractvalue %%Iface %s, 1", rt, v)
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_any_str(i8* %s, i8** %s)", r, d, rt)
			vals[len(vals)-1].val = r
			vals[len(vals)-1].typ = "String"
			fmts = append(fmts, "%s")
		case "float":
			e.needFloatToStr = true
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_float_to_str(double %s)", r, v)
			if ptrNil != "" {
				r = e.nilSelect(ptrNil, r)
			}
			vals[len(vals)-1].val = r
			vals[len(vals)-1].typ = "String"
			fmts = append(fmts, "%s")
		case "bool":
			tp := e.i8Ptr(e.strs["true"])
			fp := e.i8Ptr(e.strs["false"])
			r := e.newReg()
			e.emitInstr("%s = select i1 %s, i8* %s, i8* %s", r, e.toI1(v), tp, fp)
			if ptrNil != "" {
				r = e.nilSelect(ptrNil, r)
			}
			vals[len(vals)-1].val = r
			vals[len(vals)-1].typ = "String"
			fmts = append(fmts, "%s")
		default:
			if ptrNil != "" {
				vals[len(vals)-1].val = e.nilSelect(ptrNil, v)
				vals[len(vals)-1].typ = "String"
			}
			fmts = append(fmts, "%s")
		}
		if ptrNil != "" && t == "long" {
			e.needLLongToStr = true
			s := e.newReg()
			e.emitInstr("%s = call i8* @ql_long_to_str(i64 %s)", s, v)
			vals[len(vals)-1] = av{e.nilSelect(ptrNil, s), "String"}
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
	if x.anyBox != "" && t != x.typ {
		v = e.boxAny(v, t)
		t = x.typ
	}
	return v, t
}

// ---------- interface{}（tAny）：装箱 + 运行期类型描述符（RTTI） ----------
//
// 值表示沿用 %Iface { i8* data, i8** rt }：rt 指向类型描述符
//
//	%RT = type { i8* (i8*)* str, i8* name, i32 kind }   kind: 0=int 1=float 2=bool
//	                                                           3=String 4=List<int> 5=struct 6=null
//
// data 约定：标量/String 指向调用期分配的堆单元（按值语义，装箱即快照）；List/struct 直接是
// 对象指针（引用语义，与解释器 *List/*Struct 一致）；null 为零值。
// 打印/相等/拆箱都由运行期按 kind 分派（qthreads.c 的 ql_any_*；struct 的 str 由编译器生成）。

// anyKindOf 返回类型的 RTTI kind。
func anyKindOf(t string) int {
	switch t {
	case "int", "thread", "cbool":
		return 0
	case "float", "double", "f32":
		return 1
	case "bool":
		return 2
	case "String":
		return 3
	case "interface{}", "null", "void":
		return 6
	}
	if isListType(t) {
		return 4
	}
	return 5 // struct
}

// anyTypeName 返回与解释器 Value.TypeName() 一致的类型名（HashTable 键规则、
// interface{} 拆箱诊断都依赖它）。
func anyTypeName(t string) string {
	switch t {
	case "int", "float", "bool", "String":
		return t
	case "null", "void", "interface{}":
		return "nil"
	}
	if isListType(t) {
		return "List"
	}
	return t // struct：SType
}

// anyCopyFn 返回该类型的 RTTI 深拷贝函数（语义对齐解释器 deepCopy：
// List 复制缓冲区、struct 共享指针、标量/String 堆单元复制）。
func anyCopyFn(t string) string {
	switch anyKindOf(t) {
	case 0:
		return "@ql_any_copy_int"
	case 1:
		return "@ql_any_copy_float"
	case 2:
		return "@ql_any_copy_bool"
	case 3:
		return "@ql_any_copy_string"
	case 4:
		return "@ql_any_copy_list"
	case 6:
		return "@ql_any_copy_nil"
	}
	return "@ql_any_copy_identity"
}

// ensureRT 发射类型描述符的 LLVM 结构定义。
func (e *emitter) ensureRT() {
	if e.hasRT {
		return
	}
	e.hasRT = true
	e.types.WriteString("%RT = type { i8* (i8*)*, i8* (i8*)*, i8*, i32 }\n")
}

// rttiSym 返回（按需发射）类型描述符符号。
func (e *emitter) rttiSym(typ string) string {
	sym := "@rt$" + tyName(typ)
	if e.rtti[sym] {
		return sym
	}
	e.rtti[sym] = true
	e.ensureRT()
	kind := anyKindOf(typ)
	strFn := ""
	switch kind {
	case 0:
		e.needIntToStr = true // 复用同一 int→String 格式化
		strFn = "@ql_any_str_int"
	case 1:
		e.needFloatToStr = true // 复用同一 float→String 格式化
		strFn = "@ql_any_str_float"
	case 2:
		strFn = "@ql_any_str_bool"
	case 3:
		strFn = "@ql_any_str_string"
	case 4:
		e.ensureList()
		strFn = "@ql_any_str_list"
	default:
		strFn = "@" + e.emitStructAnyStr(typ)
	}
	nameC := e.strConst(anyTypeName(typ))
	fmt.Fprintf(&e.globals, "%s = private constant %%RT { i8* (i8*)* %s, i8* (i8*)* %s, i8* getelementptr inbounds ([%d x i8], [%d x i8]* %s, i64 0, i64 0), i32 %d }\n",
		sym, strFn, anyCopyFn(typ), nameC.size, nameC.size, nameC.name, kind)
	return sym
}

// boxAny 把具体值装箱成 interface{} 值。
func (e *emitter) boxAny(v, from string) string {
	e.ensureIface()
	sym := e.rttiSym(from)
	var data string
	switch from {
	case "int", "thread", "cbool":
		p := e.newReg()
		e.emitInstr("%s = call i8* @malloc(i64 4)", p)
		cp := e.newReg()
		e.emitInstr("%s = bitcast i8* %s to i32*", cp, p)
		e.emitInstr("store i32 %s, i32* %s", v, cp)
		data = p
	case "bool":
		p := e.newReg()
		e.emitInstr("%s = call i8* @malloc(i64 4)", p)
		cp := e.newReg()
		e.emitInstr("%s = bitcast i8* %s to i32*", cp, p)
		z := e.newReg()
		e.emitInstr("%s = zext i1 %s to i32", z, v)
		e.emitInstr("store i32 %s, i32* %s", z, cp)
		data = p
	case "float", "double":
		p := e.newReg()
		e.emitInstr("%s = call i8* @malloc(i64 8)", p)
		cp := e.newReg()
		e.emitInstr("%s = bitcast i8* %s to double*", cp, p)
		e.emitInstr("store double %s, double* %s", v, cp)
		data = p
	case "String":
		p := e.newReg()
		e.emitInstr("%s = call i8* @malloc(i64 8)", p)
		cp := e.newReg()
		e.emitInstr("%s = bitcast i8* %s to i8**", cp, p)
		e.emitInstr("store i8* %s, i8** %s", v, cp)
		data = p
	case "null":
		data = "null"
	default:
		// struct / List<int>：值本身就是对象指针（引用语义）
		data = e.newReg()
		e.emitInstr("%s = bitcast %s %s to i8*", data, e.ir(from), v)
	}
	i0 := e.newReg()
	e.emitInstr("%s = insertvalue %%Iface undef, i8* %s, 0", i0, data)
	rts := e.newReg()
	e.emitInstr("%s = bitcast %%RT* %s to i8**", rts, sym)
	i1 := e.newReg()
	e.emitInstr("%s = insertvalue %%Iface %s, i8** %s, 1", i1, i0, rts)
	return i1
}

// emitStructAnyStr 生成 struct 的 interface{} 打印函数（解释器 StructValue.String 格式：
// <T {字段=值, ...}>，字段按声明顺序）。
func (e *emitter) emitStructAnyStr(typ string) string {
	sym := "ql_any_str_struct_" + tyName(typ)
	if e.rttiFns == nil {
		e.rttiFns = map[string]bool{}
	}
	if e.rttiFns[sym] {
		return sym
	}
	e.rttiFns[sym] = true
	e.ensureStruct(typ)
	savedBody, savedVars := e.body, e.vars
	savedTerm, savedRet, savedCur := e.term, e.funcReturned, e.cur
	e.body = strings.Builder{}
	e.vars = map[string]varSlot{}
	e.blockCount = 0
	e.term = false
	e.funcReturned = false
	e.body.WriteString("entry:\n")
	p := e.newReg()
	e.emitInstr("%s = bitcast i8* %%d to %s", p, e.ir(typ))
	acc := e.i8Ptr(e.strConst("<" + typ + " {"))
	for i, f := range e.structs[typ] {
		if i > 0 {
			acc = e.strcat2(acc, e.i8Ptr(e.strConst(", ")))
		}
		acc = e.strcat2(acc, e.i8Ptr(e.strConst(f.name+"=")))
		g := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %s, %s* %s, i32 0, i32 %d", g, e.irElem(typ), e.irElem(typ), p, i)
		fv := e.newReg()
		e.emitInstr("%s = load %s, %s* %s", fv, e.ir(f.typ), e.ir(f.typ), g)
		acc = e.strcat2(acc, e.anyValueStr(fv, f.typ))
	}
	acc = e.strcat2(acc, e.i8Ptr(e.strConst("}>")))
	e.emitInstr("ret i8* %s", acc)
	body := e.body.String()
	e.body, e.vars = savedBody, savedVars
	e.term, e.funcReturned, e.cur = savedTerm, savedRet, savedCur
	e.helpers.WriteString("define i8* @" + sym + "(i8* %d) {\n" + body + "}\n")
	return sym
}

// anyValueStr 取任意字段值的字符串形式（struct 打印用；与解释器 Value.String 对齐）。
func (e *emitter) anyValueStr(v, typ string) string {
	switch {
	case typ == "int" || typ == "thread" || typ == "cbool":
		e.needIntToStr = true
		r := e.newReg()
		e.emitInstr("%s = call i8* @ql_int_to_str(i32 %s)", r, v)
		return r
	case typ == "long":
		e.needLLongToStr = true
		r := e.newReg()
		e.emitInstr("%s = call i8* @ql_long_to_str(i64 %s)", r, v)
		return r
	case typ == "float" || typ == "double" || typ == "f32":
		e.needFloatToStr = true
		fv := e.coerce(v, typ, "float")
		r := e.newReg()
		e.emitInstr("%s = call i8* @ql_float_to_str(double %s)", r, fv)
		return r
	case typ == "bool":
		r := e.newReg()
		e.emitInstr("%s = select i1 %s, i8* %s, i8* %s", r, e.toI1(v), e.i8Ptr(e.strConst("true")), e.i8Ptr(e.strConst("false")))
		return r
	case typ == "String":
		return v
	case typ == "pointer":
		e.needPtrToStr = true
		r := e.newReg()
		e.emitInstr("%s = call i8* @ql_ptr_to_str(i8* %s)", r, v)
		return r
	case typ == "interface{}":
		d := e.newReg()
		e.emitInstr("%s = extractvalue %%Iface %s, 0", d, v)
		rt := e.newReg()
		e.emitInstr("%s = extractvalue %%Iface %s, 1", rt, v)
		r := e.newReg()
		e.emitInstr("%s = call i8* @ql_any_str(i8* %s, i8** %s)", r, d, rt)
		return r
	case e.isStruct(typ):
		nn := e.newReg()
		e.emitInstr("%s = icmp ne %s %s, null", nn, e.ir(typ), v)
		okB, nilB, endB := e.newBlock(), e.newBlock(), e.newBlock()
		slot := e.newReg()
		e.emitInstr("%s = alloca i8*, align 8", slot)
		e.emitInstr("br i1 %s, label %%%s, label %%%s", nn, okB, nilB)
		e.setBlock(okB)
		fn := e.emitStructAnyStr(typ)
		c := e.newReg()
		e.emitInstr("%s = bitcast %s %s to i8*", c, e.ir(typ), v)
		r := e.newReg()
		e.emitInstr("%s = call i8* @%s(i8* %s)", r, fn, c)
		e.emitInstr("store i8* %s, i8** %s", r, slot)
		e.emitInstr("br label %%%s", endB)
		e.setBlock(nilB)
		e.emitInstr("store i8* %s, i8** %s", e.i8Ptr(e.strConst("nil")), slot)
		e.emitInstr("br label %%%s", endB)
		e.setBlock(endB)
		out := e.newReg()
		e.emitInstr("%s = load i8*, i8** %s", out, slot)
		return out
	case isListType(typ):
		e.ensureList()
		nn := e.newReg()
		e.emitInstr("%s = icmp ne %%List* %s, null", nn, v)
		okB, nilB, endB := e.newBlock(), e.newBlock(), e.newBlock()
		slot := e.newReg()
		e.emitInstr("%s = alloca i8*, align 8", slot)
		e.emitInstr("br i1 %s, label %%%s, label %%%s", nn, okB, nilB)
		e.setBlock(okB)
		head, size := e.listHeadSize(v)
		tail := e.newReg()
		e.emitInstr("%s = add i32 %s, %s", tail, head, size)
		p := e.listBuf(v)
		r := e.newReg()
		e.emitInstr("%s = call i8* @ql_list_int_str(i32* %s, i32 %s, i32 %s)", r, p, head, tail)
		e.emitInstr("store i8* %s, i8** %s", r, slot)
		e.emitInstr("br label %%%s", endB)
		e.setBlock(nilB)
		e.emitInstr("store i8* %s, i8** %s", e.i8Ptr(e.strConst("nil")), slot)
		e.emitInstr("br label %%%s", endB)
		e.setBlock(endB)
		out := e.newReg()
		e.emitInstr("%s = load i8*, i8** %s", out, slot)
		return out
	}
	return e.i8Ptr(e.strConst("nil"))
}

// strcat2 拼接两个 String 寄存器（ql_strcat）。
func (e *emitter) strcat2(a, b string) string {
	r := e.newReg()
	e.emitInstr("%s = call i8* @ql_strcat(i8* %s, i8* %s)", r, a, b)
	return r
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
		fnTy := "void ()"
		if x.i > 0 {
			slots := make([]string, x.i)
			for i := range slots {
				slots[i] = "i64"
			}
			fnTy = "void (" + strings.Join(slots, ", ") + ")"
		}
		e.emitInstr("%s = bitcast %s* @%s to i8*", r, fnTy, x.s)
		return r, "runner"
	case kNull:
		return "null", "null"
	case kBool:
		if x.b {
			return "true", "bool"
		}
		return "false", "bool"
	case kIdent:
		return e.loadVar(x.s, x.noDeref)
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
	case kMemoCall:
		return e.compileMemoCall(x), "int"
	case kTable:
		return e.compileTable(x)
	case kNewRef:
		return e.compileNewRef(x)
	case kMerge:
		return e.compileMerge(x), "void"
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
// T& / pointer T 变量默认自动解引用（解释器语义）；noDeref 时取指针本身。
func (e *emitter) loadVar(name string, noDeref bool) (string, string) {
	info, ok := e.vars[name]
	if !ok {
		return name, "int" // 未声明变量：让生成的 IR 报错（llvm-as 校验会拦截）
	}
	pt := info.typ
	if _, base, isRef := ptrRefBase(pt); isRef {
		var ptr string
		if info.param || info.direct {
			ptr = info.reg
		} else {
			llt := e.ir(pt)
			ptr = e.newReg()
			e.emitInstr("%s = load %s, %s* %s", ptr, llt, llt, info.reg)
		}
		if noDeref {
			return ptr, pt
		}
		// 可空引用：null 时读临时单元（不崩；打印/比较由调用点按 nil 语义处理）
		blt := e.ir(base)
		isnull := e.newReg()
		e.emitInstr("%s = icmp eq %s %s, null", isnull, e.ir(pt), ptr)
		dummy := e.newReg()
		e.emitInstr("%s = alloca %s, align %d", dummy, blt, e.alignOf(base))
		safe := e.newReg()
		e.emitInstr("%s = select i1 %s, %s %s, %s %s", safe, isnull, e.ir(pt), dummy, e.ir(pt), ptr)
		v := e.newReg()
		e.emitInstr("%s = load %s, %s* %s", v, blt, blt, safe)
		return v, base
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
	lo, lt := e.compileExpr(x.idx.recv)
	if lt == "List<String>" {
		p, head, size := e.listSRange(lo)
		i, it := e.compileExpr(x.idx.i)
		i = e.coerce(i, it, "int")
		e.emitBoundsCheck(i, size, x.line)
		idx := e.newReg()
		e.emitInstr("%s = add i32 %s, %s", idx, head, i)
		i64 := e.toI64(idx)
		g := e.newReg()
		e.emitInstr("%s = getelementptr inbounds i8*, i8** %s, i64 %s", g, p, i64)
		v := e.newReg()
		e.emitInstr("%s = load i8*, i8** %s", v, g)
		return v, "String"
	}
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
	args, _ := e.compileArgs(c)
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
	case "long", "pointer":
		return r, ret // FFI long → i64；pointer → i8* 不透明句柄
	}
	return r, ret
}

// compileMemoCall 编译签名 memorize 调用 f(args) @mb()：
// 按键（int 实参数组）查表，未命中则调用被包装函数并记录。
func (e *emitter) compileMemoCall(x *expr) string {
	c := x.call
	// @ 显式 prefix 实参：求值丢弃（解释器同样只把它们放进 prefix 记录）
	if x.memo != nil {
		for _, s := range x.memo.skips {
			e.compileExpr(s)
		}
	}
	handle, _ := e.compileExpr(x.memo.handle)
	n := len(c.args)
	// 实参单元（按值：解释器 derefArgs 后调用）
	cells := make([]string, 0, n)
	for _, a := range c.args {
		v, vt := e.compileExpr(a)
		v = e.coerce(v, vt, "int")
		cells = append(cells, e.valueOfCell(v, "int"))
	}
	// 键数组 [n x i32]
	keys := e.newReg()
	e.emitInstr("%s = alloca [%d x i32], align 4", keys, n)
	for i, cell := range cells {
		g := e.newReg()
		e.emitInstr("%s = getelementptr inbounds [%d x i32], [%d x i32]* %s, i32 0, i32 %d", g, n, n, keys, i)
		v := e.newReg()
		e.emitInstr("%s = load i32, i32* %s", v, cell)
		e.emitInstr("store i32 %s, i32* %s", v, g)
	}
	kp := e.newReg()
	e.emitInstr("%s = bitcast [%d x i32]* %s to i32*", kp, n, keys)
	found := e.newReg()
	e.emitInstr("%s = alloca i32, align 4", found)
	cached := e.newReg()
	e.emitInstr("%s = call i32 @ql_memo_get(i8* %s, i32 %d, i32* %s, i32* %s)", cached, handle, n, kp, found)
	fv := e.newReg()
	e.emitInstr("%s = load i32, i32* %s", fv, found)
	hit := e.newReg()
	e.emitInstr("%s = icmp ne i32 %s, 0", hit, fv)
	predB := e.cur
	missB, doneB := e.newBlock(), e.newBlock()
	e.emitInstr("br i1 %s, label %%%s, label %%%s", hit, doneB, missB)
	e.setBlock(missB)
	callArgs := make([]string, 0, n)
	for _, cell := range cells {
		callArgs = append(callArgs, "i32* "+cell)
	}
	r := e.newReg()
	e.body.WriteString(r + " = call i32 @" + c.name + "(" + strings.Join(callArgs, ", ") + ")\n")
	e.emitInstr("call void @ql_memo_put(i8* %s, i32 %d, i32* %s, i32 %s)", handle, n, kp, r)
	e.emitInstr("br label %%%s", doneB)
	e.setBlock(doneB)
	phi := e.newReg()
	e.emitInstr("%s = phi i32 [ %s, %%%s ], [ %s, %%%s ]", phi, cached, predB, r, missB)
	return phi
}

// compileMerge 编译 taskm.merge(pid, runner, args...)：实参打进 i64 槽交给运行时，
// 由 runner 还原成目标函数的形参单元（引用传递）。
func (e *emitter) compileMerge(x *expr) string {
	c := x.call
	pid, _ := e.compileExpr(c.args[0])
	runners, _ := e.compileExpr(c.args[1])
	slots := make([]string, 0, 8)
	for _, a := range c.args[2:] {
		v, vt := e.compileExpr(a)
		slots = append(slots, e.packI64(v, vt))
	}
	for len(slots) < 8 {
		slots = append(slots, "0")
	}
	e.body.WriteString("  call void @ql_merge(i32 " + pid + ", i8* " + runners + ", i64 " +
		strings.Join(slots, ", i64 ") + ")\n")
	return "0"
}

// packI64 把实参值打包进 i64 槽（运行时 ql_task.args 是 4 个 long long）。
func (e *emitter) packI64(v, typ string) string {
	r := e.newReg()
	switch typ {
	case "int":
		e.emitInstr("%s = sext i32 %s to i64", r, v)
	case "bool":
		e.emitInstr("%s = zext i1 %s to i64", r, v)
	case "float":
		e.emitInstr("%s = bitcast double %s to i64", r, v)
	case "long":
		return v
	case "thread":
		e.emitInstr("%s = sext i32 %s to i64", r, v)
	default:
		// String / pointer / struct / List / Channel / memorize：指针值
		e.emitInstr("%s = ptrtoint %s %s to i64", r, e.ir(typ), v)
	}
	return r
}

// compileNewRef 编译 new T：分配一个清零的 T 单元，返回 T& 指针。
func (e *emitter) compileNewRef(x *expr) (string, string) {
	base := x.s
	ptlt := e.ir(x.typ) // T& 的指针类型（i32* / %P* / i8** …）
	// 大小：标量按 LLVM 类型大小，struct 用 GEP null,1
	var sz string
	if e.isStruct(base) {
		elem := e.irElem(base)
		g := e.newReg()
		e.emitInstr("%s = getelementptr %s, %s* null, i32 1", g, elem, elem)
		sz = e.newReg()
		e.emitInstr("%s = ptrtoint %s* %s to i64", sz, elem, g)
	} else {
		sz = strconv.Itoa(e.sizeOf(base))
		if sz == "0" {
			sz = "8"
		}
	}
	obj := e.newReg()
	e.emitInstr("%s = call i8* @calloc(i64 1, i64 %s)", obj, sz)
	p := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to %s", p, obj, ptlt)
	return p, base + "&"
}

// nilSelect 按 null 判定选择 "nil" 或给定字符串（可空引用解引用的 nil 语义）。
func (e *emitter) nilSelect(isnull, s string) string {
	nilp := e.i8Ptr(e.strConst("nil"))
	r := e.newReg()
	e.emitInstr("%s = select i1 %s, i8* %s, i8* %s", r, isnull, nilp, s)
	return r
}

// sizeOf 返回标量类型的字节大小（struct 走 GEP 计算）。
func (e *emitter) sizeOf(t string) int {
	switch t {
	case "bool", "char":
		return 1
	case "int", "thread", "cbool", "f32":
		return 4
	case "float", "double", "long", "pointer", "null", "String", "Channel", "channel", "runner", "memorize":
		return 8
	}
	if isTableT(t) || isListType(t) || e.ifaces[t] {
		return 8
	}
	return 8
}

// compileTable 编译 HashTable 方法调用：键按解释器规则结构化成字符串
// （TypeName:String()），值装箱成 interface{}（Put 时按 RTTI 深拷贝）。
func (e *emitter) compileTable(x *expr) (string, string) {
	t := x.tbl
	h, _ := e.compileExpr(t.recv)
	switch t.name {
	case "size":
		r := e.newReg()
		e.emitInstr("%s = call i32 @ql_table_size(i8* %s)", r, h)
		return r, "int"
	case "contains":
		k := e.tableKey(t.args[0])
		r := e.newReg()
		e.emitInstr("%s = call i32 @ql_table_contains(i8* %s, i8* %s)", r, h, k)
		b := e.newReg()
		e.emitInstr("%s = icmp ne i32 %s, 0", b, r)
		return b, "bool"
	case "remove":
		k := e.tableKey(t.args[0])
		e.emitInstr("call void @ql_table_remove(i8* %s, i8* %s)", h, k)
		return "0", "void"
	case "put":
		k := e.tableKey(t.args[0])
		d, rt := e.anyPair(t.args[1])
		e.emitInstr("call void @ql_table_put(i8* %s, i8* %s, i8* %s, i8** %s)", h, k, d, rt)
		return "0", "void"
	case "get":
		k := e.tableKey(t.args[0])
		slot := e.newReg()
		e.emitInstr("%s = alloca i8**, align 8", slot)
		d := e.newReg()
		e.emitInstr("%s = call i8* @ql_table_get(i8* %s, i8* %s, i8** %s)", d, h, k, slot)
		rt := e.newReg()
		e.emitInstr("%s = load i8**, i8** %s", rt, slot)
		if isAnyType(t.valT) {
			// 缺键 → (null, null) = nil（与解释器 get 返回 NilV 一致）
			i0 := e.newReg()
			e.emitInstr("%s = insertvalue %%Iface undef, i8* %s, 0", i0, d)
			i1 := e.newReg()
			e.emitInstr("%s = insertvalue %%Iface %s, i8** %s, 1", i1, i0, rt)
			return i1, "interface{}"
		}
		// 具体 V：按 RTTI 拆箱；缺键/类型不符 → 明确运行期错误（nil 无法用静态类型承载）
		switch anyKindOf(t.valT) {
		case 0:
			r := e.newReg()
			e.emitInstr("%s = call i32 @ql_any_int(i8* %s, i8** %s)", r, d, rt)
			return r, "int"
		case 1:
			r := e.newReg()
			e.emitInstr("%s = call double @ql_any_float(i8* %s, i8** %s)", r, d, rt)
			return r, "float"
		case 2:
			r := e.newReg()
			e.emitInstr("%s = call i32 @ql_any_bool(i8* %s, i8** %s)", r, d, rt)
			b := e.newReg()
			e.emitInstr("%s = icmp ne i32 %s, 0", b, r)
			return b, "bool"
		case 3:
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_any_string(i8* %s, i8** %s)", r, d, rt)
			return r, "String"
		default:
			// struct / List：存的就是对象指针（Put 时 struct 共享、List 深拷贝）
			p := e.newReg()
			e.emitInstr("%s = call i8* @ql_any_ptr(i8* %s, i8** %s)", p, d, rt)
			ir := e.newReg()
			e.emitInstr("%s = bitcast i8* %s to %s", ir, p, e.ir(t.valT))
			return ir, t.valT
		}
	case "keys":
		l := e.newReg()
		e.emitInstr("%s = call i8* @ql_table_keys(i8* %s)", l, h)
		return l, "List<String>"
	}
	return "0", "void"
}

// tableKey 把键表达式装箱后转成解释器同款键串（TypeName:String()）。
func (e *emitter) tableKey(k *expr) string {
	d, rt := e.anyPair(k)
	r := e.newReg()
	e.emitInstr("%s = call i8* @ql_table_key(i8* %s, i8** %s)", r, d, rt)
	return r
}

// anyPair 取表达式的 packed interface{} 值 (data, rt)。
func (e *emitter) anyPair(x *expr) (string, string) {
	v, typ := e.compileExpr(x)
	if !isAnyType(typ) {
		v = e.boxAny(v, typ)
	}
	d := e.newReg()
	e.emitInstr("%s = extractvalue %%Iface %s, 0", d, v)
	rt := e.newReg()
	e.emitInstr("%s = extractvalue %%Iface %s, 1", rt, v)
	return d, rt
}

// isAnyType 判断是否是空接口 interface{}。
func isAnyType(t string) bool { return strings.TrimSpace(t) == "interface{}" }

// ptrRefBase 判断语言类型是否 T& / pointer T（可空引用）：
// 返回 (完整类型, 指向的类型, 是否)。
func ptrRefBase(t string) (string, string, bool) {
	t = strings.TrimSpace(t)
	if t == "pointer" || t == "" {
		return "", "", false // 裸 pointer = FFI 不透明句柄（i8*），不是 T&
	}
	if strings.HasPrefix(t, "pointer ") {
		inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(t[len("pointer "):]), "&"))
		if inner == "" {
			return "", "", false
		}
		return inner + "&", inner, true
	}
	if strings.HasSuffix(t, "&") {
		inner := strings.TrimSpace(strings.TrimSuffix(t, "&"))
		if inner == "" {
			return "", "", false
		}
		return inner + "&", inner, true
	}
	return "", "", false
}

// compileMethod 编译实例方法调用与内建方法。
func (e *emitter) compileMethod(x *expr) (string, string) {
	m := x.method
	if m.iface != "" {
		return e.compileIfaceCall(x)
	}
	recv, rtyp := e.compileExpr(m.recv)
	// List<String> 内建方法（keys() 结果等；只 lower 只读子集）
	if rtyp == "List<String>" {
		switch m.name {
		case "size":
			_, _, size := e.listSRange(recv)
			return size, "int"
		case "get":
			p, head, size := e.listSRange(recv)
			i, it := e.compileExpr(m.args[0])
			i = e.coerce(i, it, "int")
			e.emitBoundsCheck(i, size, x.line)
			idx := e.newReg()
			e.emitInstr("%s = add i32 %s, %s", idx, head, i)
			i64 := e.toI64(idx)
			g := e.newReg()
			e.emitInstr("%s = getelementptr inbounds i8*, i8** %s, i64 %s", g, p, i64)
			v := e.newReg()
			e.emitInstr("%s = load i8*, i8** %s", v, g)
			return v, "String"
		case "toString":
			p, head, size := e.listSRange(recv)
			tail := e.newReg()
			e.emitInstr("%s = add i32 %s, %s", tail, head, size)
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_list_str_str(i8** %s, i32 %s, i32 %s)", r, p, head, tail)
			return r, "String"
		}
	}
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
		case "head":
			hf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 1", hf, recv)
			h := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", h, hf)
			return h, "int"
		case "tail":
			tf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, recv)
			t := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", t, tf)
			return t, "int"
		case "next", "peek":
			hf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 1", hf, recv)
			tf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, recv)
			h := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", h, hf)
			t := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", t, tf)
			empty := e.newReg()
			e.emitInstr("%s = icmp eq i32 %s, %s", empty, h, t)
			okB := e.newBlock()
			badB := e.newBlock()
			e.emitInstr("br i1 %s, label %%%s, label %%%s", empty, badB, okB)
			e.setBlock(badB)
			msg := "ListExhaustedError: list is exhausted (head()==tail()); next() stops and errors"
			if m.name == "peek" {
				msg = "ListExhaustedError: list is exhausted (head()==tail()); '*' stops and errors"
			}
			if e.curTry != "" {
				e.emitInstr("br label %%%s", e.curTry)
				e.emitInstr("unreachable")
			} else {
				e.needPanic = true
				c := e.i8Ptr(e.strConst(msg))
				e.emitInstr("call void @ql_panic(i8* %s, i32 %d)", c, x.line)
				e.emitInstr("unreachable")
			}
			e.setBlock(okB)
			p := e.listBuf(recv)
			h64 := e.toI64(h)
			ep := e.newReg()
			e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", ep, p, h64)
			v := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", v, ep)
			if m.name == "next" {
				nx := e.newReg()
				e.emitInstr("%s = add i32 %s, 1", nx, h)
				e.emitInstr("store i32 %s, i32* %s", nx, hf)
			}
			return v, "int"
		case "reset":
			hf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 1", hf, recv)
			e.emitInstr("store i32 0, i32* %s", hf)
			return recv, "List<int>"
		case "appendAll":
			other, _ := e.compileExpr(m.args[0])
			head, size := e.listHeadSize(other)
			tf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 2", tf, recv)
			idx := e.newReg()
			e.emitInstr("%s = add i32 %s, %s", idx, head, size)
			selfBuf := e.listBuf(recv)
			otherBuf := e.listBuf(other)
			condB := e.newBlock()
			bodyB := e.newBlock()
			endB := e.newBlock()
			iv := e.newReg()
			e.emitInstr("%s = alloca i32, align 4", iv)
			e.emitInstr("store i32 %s, i32* %s", head, iv)
			e.emitInstr("br label %%%s", condB)
			e.setBlock(condB)
			li := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", li, iv)
			cnd := e.newReg()
			e.emitInstr("%s = icmp slt i32 %s, %s", cnd, li, idx)
			e.emitInstr("br i1 %s, label %%%s, label %%%s", cnd, bodyB, endB)
			e.setBlock(bodyB)
			li2 := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", li2, iv)
			li64 := e.toI64(li2)
			sp := e.newReg()
			e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", sp, otherBuf, li64)
			val := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", val, sp)
			// 追加到 self 尾部（realloc 扩容，同 append）
			t1 := e.newReg()
			e.emitInstr("%s = load i32, i32* %s", t1, tf)
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
			cur := e.newReg()
			e.emitInstr("%s = load i32*, i32** %s", cur, bf)
			bc := e.newReg()
			e.emitInstr("%s = bitcast i32* %s to i8*", bc, cur)
			rc := e.newReg()
			e.emitInstr("%s = call i8* @realloc(i8* %s, i64 %s)", rc, bc, sz)
			np := e.newReg()
			e.emitInstr("%s = bitcast i8* %s to i32*", np, rc)
			e.emitInstr("store i32* %s, i32** %s", np, bf)
			t1i := e.newReg()
			e.emitInstr("%s = sext i32 %s to i64", t1i, t1)
			dp := e.newReg()
			e.emitInstr("%s = getelementptr inbounds i32, i32* %s, i64 %s", dp, np, t1i)
			e.emitInstr("store i32 %s, i32* %s", val, dp)
			t2 := e.newReg()
			e.emitInstr("%s = add i32 %s, 1", t2, t1)
			e.emitInstr("store i32 %s, i32* %s", t2, tf)
			ni := e.newReg()
			e.emitInstr("%s = add i32 %s, 1", ni, li2)
			e.emitInstr("store i32 %s, i32* %s", ni, iv)
			e.emitInstr("br label %%%s", condB)
			e.setBlock(endB)
			_ = selfBuf
			return "0", "void"
		case "toString":
			p := e.listBuf(recv)
			head, size := e.listHeadSize(recv)
			tail := e.newReg()
			e.emitInstr("%s = add i32 %s, %s", tail, head, size)
			r := e.newReg()
			e.emitInstr("%s = call i8* @ql_list_int_str(i32* %s, i32 %s, i32 %s)", r, p, head, tail)
			return r, "String"
		case "__sort__":
			p := e.listBuf(recv)
			head, size := e.listHeadSize(recv)
			tail := e.newReg()
			e.emitInstr("%s = add i32 %s, %s", tail, head, size)
			e.emitInstr("call void @ql_list_int_sort(i32* %s, i32 %s, i32 %s)", p, head, tail)
			return "0", "void"
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
	// struct 实例方法：call @Type_method(self*, args...)
	// 接收者按值绑定（解释器：self = ... 不回写调用方）→ 临时单元；
	// 其余实参按引用传递（解释器 evalArgs：左值实参写回调用方）。
	if m.sig != "" {
		args := []string{e.ir(rtyp) + "* " + e.valueOfCell(recv, rtyp)}
		sig := e.sigs[m.sig]
		for i, a := range m.args {
			want := "?"
			if sig != nil && i < len(sig.params) {
				want = sig.params[i].typ
			}
			if m.byVal {
				v, vt := e.compileExpr(a)
				if want == "?" {
					want = vt
				}
				v = e.coerce(v, vt, want)
				args = append(args, e.ir(want)+" "+e.valueOfCell(v, want))
				continue
			}
			p, pt, _ := e.compileArgAddr(a, want)
			args = append(args, e.ir(pt)+"* "+p)
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
		want := "?"
		if m.ifaceSig != nil && i < len(m.ifaceSig.params) {
			want = m.ifaceSig.params[i].typ
		}
		if want == "Self" {
			// Self 形参：接口值 → 取 data（具体实例指针），放进临时单元按引用传递
			av, _ := e.compileExpr(a)
			d := e.newReg()
			e.emitInstr("%s = extractvalue %%Iface %s, 0", d, av)
			pty = append(pty, "i8*")
			callArgs = append(callArgs, "i8* "+d)
			continue
		}
		// 其余实参：语言层按引用传递 → 传值单元地址（thunk 原样转发给具体方法）
		p, pt, _ := e.compileArgAddr(a, want)
		pty = append(pty, e.ir(pt)+"*")
		callArgs = append(callArgs, e.ir(pt)+"* "+p)
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
		ps = append(ps, e.ir(p)+"*") // 按引用传递
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
			sig.WriteString(", " + e.ir(p) + "* noundef " + regs[i])
		}
	}
	sig.WriteString(")\n")
	e.body.WriteString("entry:\n")
	self := e.newReg()
	e.emitInstr("%s = bitcast i8* %%d to %s", self, ptrTy)
	callArgs := []string{ptrTy + "* " + e.valueOfCell(self, th.typ)}
	for i, p := range th.params {
		if isSelfIdx(th.selfIdx, i) {
			// Self 形参：接口 data（i8*）→ 具体类型指针 → 临时单元（按引用传递）
			c := e.newReg()
			e.emitInstr("%s = bitcast i8* %s to %s", c, regs[i], e.ir(p))
			callArgs = append(callArgs, e.ir(p)+"* "+e.valueOfCell(c, p))
			continue
		}
		// 其余形参：thunk 直接转发调用方的单元地址
		callArgs = append(callArgs, e.ir(p)+"* "+regs[i])
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

// valueOfCell 把值放进当前帧的临时单元，返回单元地址（按引用传递的值实参）。
func (e *emitter) valueOfCell(v, typ string) string {
	llt := e.ir(typ)
	slot := e.newReg()
	e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(typ))
	e.emitInstr("store %s %s, %s* %s", llt, v, llt, slot)
	return slot
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
	garg := e.valueOfCell(li2, "int")
	gv := e.newReg()
	e.emitInstr("%s = call i32 @%s(i32* %s)", gv, gname, garg)
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
		if x.op == "/" {
			e.emitZeroCheckF(rv, "DivisionByZeroError: float division by zero", x.line)
		}
		if x.op == "%" {
			// 解释器语义：float 取模在运行期报 TypeError
			e.emitAbortF("TypeError: '%' requires int operands", x.line)
			return "0.0", "float" // 之后的死代码
		}
		op := map[string]string{"+": "fadd", "-": "fsub", "*": "fmul", "/": "fdiv"}[x.op]
		reg := e.newReg()
		e.emitInstr("%s = %s double %s, %s", reg, op, lv, rv)
		return reg, "float"
	}
	if v, ok := constEval(x); ok {
		return fmt.Sprintf("%d", int32(v)), "int"
	}
	// long 参与算术：解释器按 32 位截断（wrapI32），这里同样先截断
	lv = e.coerce(lv, lt, "int")
	rv = e.coerce(rv, rt, "int")
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

// emitAbortF 发射「一定失败」的浮点运算路径（解释器在此报运行期错误）：
// try 内跳 catch，否则打印同文案错误并退出。
func (e *emitter) emitAbortF(msg string, line int) {
	okB := e.newBlock()
	tgt := e.curTry
	if tgt == "" {
		e.needPanic = true
		tgt = e.newBlock()
		e.emitInstr("br label %%%s", tgt)
		e.setBlock(tgt)
		c := e.i8Ptr(e.strConst(msg))
		e.emitInstr("call void @ql_panic(i8* %s, i32 %d)", c, line)
		e.emitInstr("unreachable")
		e.setBlock(okB)
		return
	}
	e.emitInstr("br label %%%s", tgt)
	e.emitInstr("unreachable")
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
	// interface{} 相等：运行期按 RTTI 比较（数值跨 int/float、String 按内容、struct/List 按引用）
	if lt == "interface{}" || rt == "interface{}" {
		if lt != "interface{}" {
			lv = e.boxAny(lv, lt)
			lt = "interface{}"
		}
		if rt != "interface{}" {
			rv = e.boxAny(rv, rt)
			rt = "interface{}"
		}
		e.ensureIface()
		ld := e.newReg()
		e.emitInstr("%s = extractvalue %%Iface %s, 0", ld, lv)
		lrt := e.newReg()
		e.emitInstr("%s = extractvalue %%Iface %s, 1", lrt, lv)
		rd := e.newReg()
		e.emitInstr("%s = extractvalue %%Iface %s, 0", rd, rv)
		rrt := e.newReg()
		e.emitInstr("%s = extractvalue %%Iface %s, 1", rrt, rv)
		cv := e.newReg()
		e.emitInstr("%s = call i32 @ql_any_eq(i8* %s, i8** %s, i8* %s, i8** %s)", cv, ld, lrt, rd, rrt)
		r := e.newReg()
		op := "ne"
		if x.op == "!=" {
			op = "eq"
		}
		e.emitInstr("%s = icmp %s i32 %s, 0", r, op, cv)
		return r, "bool"
	}
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
	// 指针 / null：引用比较（解释器 Value 语义：同一指针相等）
	if isPtrLike(lt) || isPtrLike(rt) {
		op := cmpOps[x.op]
		if x.op != "==" && x.op != "!=" {
			op = "eq" // 指针不支持顺序比较（typecheck 已拦截）
		}
		r := e.newReg()
		e.emitInstr("%s = icmp %s i8* %s, %s", r, op, lv, rv)
		return r, "bool"
	}
	// long 参与比较：解释器按 32 位截断比较
	lv = e.coerce(lv, lt, "int")
	rv = e.coerce(rv, rt, "int")
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

// isPtrLike 判断是否是 FFI 不透明指针 / null（i8* 引用比较）。
func isPtrLike(t string) bool {
	if t == "pointer" || t == "null" {
		return true
	}
	_, _, ok := ptrRefBase(t)
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
	if e.anyBox != "" {
		m.impure = true // 装箱调用 malloc（标量堆单元），不是纯函数
	}
	switch e.kind {
	case kTable:
		m.impure = true // ql_table_*：堆表读写
		if e.tbl != nil {
			analyzeExpr(e.tbl.recv, m)
			for _, a := range e.tbl.args {
				analyzeExpr(a, m)
			}
		}
	case kMerge:
		m.impure = true // ql_merge：启动线程
		if e.call != nil {
			for _, a := range e.call.args {
				analyzeExpr(a, m)
			}
		}
	case kMemoCall:
		m.impure = true
		if e.call != nil {
			m.callees = append(m.callees, e.call.name)
		}
		if e.memo != nil {
			analyzeExpr(e.memo.handle, m)
			for _, s := range e.memo.skips {
				analyzeExpr(s, m)
			}
		}
		if e.call != nil {
			for _, a := range e.call.args {
				analyzeExpr(a, m)
			}
		}
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
		analyzeExpr(e.idx.recv, m)
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
		// 形参按引用传递：形参是指向调用方存储的指针，callee 的 load/store 对调用方
		// 可见 → 有形参的函数不能标 memory(none)（否则 LLVM 会消除参数写入）。
		if len(fd.params) > 0 || fd.selfTyp != "" {
			m.impure = true
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

// scanByRefArgs 收集「作为按引用实参出现的变量名」：这些变量必须有 alloca 存储，
// 否则 compileArgAddr 只能退化成临时单元，callee 的写入无法回写调用方。
func scanByRefArgs(stmts []stmt, sigs map[string]*funcSig, out map[string]bool) {
	mark := func(x *expr) {
		if x != nil && x.kind == kIdent {
			out[x.s] = true
		}
	}
	var walkExpr func(x *expr)
	walkExpr = func(x *expr) {
		if x == nil {
			return
		}
		switch x.kind {
		case kTable:
			if x.tbl != nil {
				walkExpr(x.tbl.recv)
				for _, a := range x.tbl.args {
					walkExpr(a)
				}
			}
		case kMerge:
			if x.call != nil {
				for _, a := range x.call.args {
					walkExpr(a)
				}
			}
		case kMemoCall:
			if x.call != nil {
				for _, a := range x.call.args {
					walkExpr(a)
				}
			}
			if x.memo != nil {
				walkExpr(x.memo.handle)
				for _, s := range x.memo.skips {
					walkExpr(s)
				}
			}
		case kCall:
			if x.call != nil {
				if sig := sigs[x.call.name]; sig != nil && sig.byRef {
					for _, a := range x.call.args {
						mark(a)
					}
				}
				for _, a := range x.call.args {
					walkExpr(a)
				}
			}
		case kMethod:
			if x.method != nil {
				// 用户方法：实参按引用；接口方法（vtable 分发）：同样按引用（解释器 evalArgs）
				byRefCall := x.method.iface != ""
				if s := sigs[x.method.sig]; s != nil && s.byRef {
					byRefCall = true
				}
				if byRefCall && !x.method.byVal {
					for _, a := range x.method.args {
						mark(a)
					}
				}
				walkExpr(x.method.recv)
				for _, a := range x.method.args {
					walkExpr(a)
				}
			}
		}
		walkExpr(x.l)
		walkExpr(x.r)
		if x.field != nil {
			walkExpr(x.field.recv)
		}
		if x.lst != nil {
			for _, it := range x.lst.items {
				walkExpr(it)
			}
		}
		if x.sl != nil {
			for _, v := range x.sl.values {
				walkExpr(v)
			}
		}
		if x.idx != nil {
			walkExpr(x.idx.recv)
			walkExpr(x.idx.i)
		}
	}
	var walk func(ss []stmt)
	walk = func(ss []stmt) {
		for _, s := range ss {
			switch st := s.(type) {
			case *exprStmt:
				walkExpr(st.x)
			case *returnStmt:
				walkExpr(st.x)
			case *printlnStmt:
				for _, a := range st.args {
					walkExpr(a)
				}
			case *printStmt:
				for _, a := range st.args {
					walkExpr(a)
				}
			case *declStmt:
				walkExpr(st.init)
			case *assignStmt:
				walkExpr(st.x)
			case *indexAssignStmt:
				walkExpr(st.idx)
				walkExpr(st.x)
			case *fieldAssignStmt:
				walkExpr(st.recv)
				walkExpr(st.x)
			case *ifStmt:
				walkExpr(st.cond)
				walk(st.then)
				walk(st.els)
			case *whileStmt:
				walkExpr(st.cond)
				walk(st.body)
			case *forStmt:
				if st.init != nil {
					walk([]stmt{st.init})
				}
				walkExpr(st.cond)
				if st.step != nil {
					walk([]stmt{st.step})
				}
				walk(st.body)
			case *forInStmt:
				walk(st.body)
			case *tryStmt:
				walk(st.then)
				walk(st.catch)
			case *logStmt:
				walkExpr(st.x)
			}
		}
	}
	walk(stmts)
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

// ptrToStrHelper 是 pointer 打印助手：null → "nil"，否则 "0x%llx"
// （与解释器 Value.String() 的 0x+十六进制 呈现一致）。
const ptrToStrHelper = `@.ql.nil = private unnamed_addr constant [4 x i8] c"nil\00", align 1
@.ql.fmt.p = private unnamed_addr constant [7 x i8] c"0x%llx\00", align 1
define i8* @ql_ptr_to_str(i8* %p) {
entry:
  %isnull = icmp eq i8* %p, null
  br i1 %isnull, label %nil, label %hex
nil:
  %np = getelementptr inbounds [4 x i8], [4 x i8]* @.ql.nil, i64 0, i64 0
  ret i8* %np
hex:
  %b = call i8* @malloc(i64 20)
  %f = getelementptr inbounds [7 x i8], [7 x i8]* @.ql.fmt.p, i64 0, i64 0
  %v = ptrtoint i8* %p to i64
  %n = call i32 (i8*, i64, i8*, ...) @snprintf(i8* %b, i64 20, i8* %f, i64 %v)
  ret i8* %b
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
