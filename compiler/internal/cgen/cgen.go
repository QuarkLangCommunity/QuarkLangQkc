// Package cgen compiles QuarkLang to LLVM IR (cross-platform compiler, LLVM backend).
//
// The frontend has exactly one syntax source: the canonical AST of internal/lang (shared with the
// interpreter). This package carries no lexer/parser of its own; the pipeline is:
//
//	lang.CompileWithImports(src, filename)  // lexing/parsing/typecheck + recursive import merge
//	  → lowerProgram (lower.go)            // canonical AST → cgen IR (error if unsupported)
//	  → emitter (this file)                // cgen IR → LLVM IR
//
// The list of supported and unsupported constructs is in the header of lower.go; a construct the
// backend has not lowered returns an explicit error with source location — never a silent miscompile.
//
// Value model (aligned with interpreter semantics):
//   - int → i32, bool → i1, float → double, String → i8*
//   - List<T> / struct are both **references** (heap object pointers), matching the interpreter's
//     *List / *Struct aliasing: assignment and argument passing copy the reference, so all aliases see field/element writes
//   - struct fields are laid out in declaration order (LLVM literal/named structs), zero value = calloc zeroing
package cgen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"quarklang/internal/lang"
)

// Transpile compiles QuarkLang source to LLVM IR.
//
// src must already have gone through compiler-side macro expansion (see expandMacros in compiler/main.go);
// filename is used for import resolution (same-directory .qk/.qlib, the same recursive merge semantics as
// the interpreter) and for diagnostics; it may be "".
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
	kRunner   // runner function reference for taskm.merge (an i8* function pointer)
	kNull     // null literal (zero value of a pointer)
	kMemoCall // signature call: f(args) @memorize (memoization wrapper)
	kMerge    // taskm.merge / t.merge: an i64 slot carries 0..4 arguments
	kTable    // HashTable method call (put/get/contains/remove/size/keys)
	kNewRef   // new T: allocate a single-element cell and return a T& pointer
	kListS    // List<String> literal: a heap %ListS whose buffer holds i8* elements
	kConv     // conversion call T(x) (spec §3.4): s names the destination type
)

// expr is a cgen IR expression. typ is filled in by lowering with the language type ("?" = undecided).
type expr struct {
	kind exprKind
	i    int64
	f    float64
	b    bool
	s    string
	op   string
	typ  string
	bits int // raw-bits width of the operation (kBin/kIndex), 0 = an ordinary view operation
	l, r *expr

	call   *callExpr   // kCall / kMemoCall
	memo   *memoExpr   // kMemoCall
	tbl    *tableExpr  // kTable
	lst    *listLit    // kList
	idx    *indexExpr  // kIndex
	method *methodExpr // kMethod
	sl     *structLit  // kStructLit
	field  *fieldExpr  // kField

	sc     strConst // pre-registered string constant (kString)
	strcat bool     // kBin "+" and it is String concatenation (has a ql_strcat side effect)
	line   int      // runtime error location (division by zero etc.)

	ifaceBox string // non-empty: box a concrete struct value into an interface value (the value is the vtable symbol)
	anyBox   string // non-empty: box a value of this concrete type into interface{} (the RTTI descriptor is emitted per type)
	noDeref  bool   // kIdent points at a T& variable: take the pointer itself instead of the dereferenced value (pointer comparison/rebinding)
}

type indexExpr struct {
	recv *expr // List expression (variable/struct field/…)
	i    *expr
}

type methodExpr struct {
	recv   *expr
	name   string // method name
	args   []*expr
	sig    string // IR function name (filled in after lowering resolves it)
	isSelf bool   // instance method (needs a receiver argument)
	byVal  bool   // argument passed by value (operator overload: the interpreter passes operands by value)

	iface    string   // non-empty: interface method call (vtable dispatch)
	idx      int      // method's slot in the interface dispatch table
	ifaceSig *funcSig // call signature (the Self parameter is marked "Self")
}

type fieldExpr struct {
	recv *expr
	name string
	typ  string // receiver's struct type name
}

type structLit struct {
	typ    string // target struct type name
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
	thru bool // target is a T& variable and the RHS is a T value: write through to the cell the pointer points at
}

type indexAssignStmt struct {
	recv *expr // List expression (variable/struct field/…)
	idx  *expr
	x    *expr
}

// bitAssignStmt writes one bit of raw bits — b[i] = v (spec §3.3) — by reading the raw value,
// replacing that bit and storing it back.
type bitAssignStmt struct {
	recv *expr
	idx  *expr
	x    *expr
	bits int
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
	name      string // IR function name
	args      []*expr
	byValArgs bool // arguments passed by value (signature memorize wrapper: the interpreter calls after derefArgs)
}

// tableExpr is the context of a HashTable method call (kTable).
type tableExpr struct {
	recv *expr   // table handle (i8*)
	name string  // put/get/contains/remove/size/keys
	args []*expr // put: [key, value]; others: [key] (size/keys take no argument)
	keyT string  // key static type
	valT string  // value static type
}

// isTableT reports whether a language type is HashTable<...>.
func isTableT(t string) bool { return strings.HasPrefix(strings.TrimSpace(t), "HashTable<") }

// memoExpr is the context of a signature memorize call (kMemoCall).
type memoExpr struct {
	handle *expr   // memorize instance (i8* handle)
	skips  []*expr // explicit @ prefix arguments: evaluated and discarded (the interpreter puts them into the prefix record)
}

type funcParam struct {
	name  string
	typ   string
	copyd bool // copyd parameter: deep-copied into a local callee cell on binding (no write back to the caller)
}

type funcDef struct {
	name      string // IR function name (already mangled: Point_sum / math_max / …)
	params    []funcParam
	ret       string
	body      []stmt
	selfTyp   string // struct type name of the impl method receiver
	selfParam string // receiver parameter name (self)
}

type structDef struct {
	name       string
	fields     []string
	fieldTypes []string
}

// lowered is the program after the canonical AST was lowered to cgen IR.
type lowered struct {
	funcs     []*funcDef // non-main functions
	mainStmts []stmt     // body of fn main
	structs   []structDef
	vtables   []*vtableDef
	ifaces    []string
	externs   []externDef
	runners   []runnerDef
}

// sortVTables keeps vtable emission order stable (for regression comparison).
func (lp *lowered) sortVTables() {
	sort.Slice(lp.vtables, func(i, j int) bool { return lp.vtables[i].sym < lp.vtables[j].sym })
}

// runnerDef is the executor for taskm.merge (qthreads.c calls it as void (*)(long long ×4)).
type runnerDef struct {
	fn     string   // target function IR name
	params []string // target function parameter types (≤4; carried at runtime in i64 slots)
}

// emitRunners generates the taskm.merge executors: restore the runtime i64 slots into the target
// function's parameter cells, then call it by reference (return value discarded). Supports 0..4 parameters of any lowerable type.
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

// externDef is a library FFI external symbol declaration (LLVM declare + C ABI call).
type externDef struct {
	name   string // C symbol name
	params []funcParam
	ret    string // cgen type (cbool/f32/double/…)
	lib    string // link library name (qkc-link marker)
}

// vtableDef is the dispatch table for (concrete type, interface).
type vtableDef struct {
	sym    string // @vt$A$Iface
	iface  string
	typ    string
	thunks []*ifaceThunk
}

// ifaceThunk lowers an interface call back to a concrete-type method call.
type ifaceThunk struct {
	iface   string
	typ     string
	method  string
	irName  string   // concrete method IR name
	ret     string   // concrete return type ("void" = no return)
	params  []string // concrete parameter types (excluding self)
	selfIdx []int    // indices of parameters the interface declares as Self (the thunk receives i8*)
	boxRet  bool     // interface returns Self → the thunk boxes the result and returns %Iface
	vtSym   string   // vtable symbol used for boxing
}

// ---------- LLVM IR emitter ----------

type strConst struct {
	name string
	size int
}

// varSlot is a variable's storage: an alloca slot (reg is the slot pointer) or an SSA passthrough/parameter (reg is the value).
type varSlot struct {
	reg    string
	typ    string // language type
	param  bool   // function parameter: value register (legacy form, no longer produced after by-ref)
	direct bool   // SSA passthrough (single-assignment scalar)
	ref    bool   // reg points at the caller's storage (by-reference parameter): writes go through to the caller
}

type structField struct {
	name string
	typ  string
}

type funcSig struct {
	name   string
	params []funcParam
	ret    string
	byRef  bool // user function: parameters passed by reference (the LLVM-level parameter type is "value type*")
}

type emitter struct {
	fnMeta map[string]*fnMeta

	types   strings.Builder // %Point = type { ... } / %List = type { ... }
	globals strings.Builder // string constants
	decls   strings.Builder // declare (external symbols)
	helpers strings.Builder // runtime helpers (on demand)
	bodies  strings.Builder // function definitions

	body strings.Builder // current function body
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
	term         bool // the current basic block is already closed by br/ret/unreachable
	assigned     map[string]bool

	emitted        map[string]bool
	deepCopied     map[string]bool // deep-copy helpers already emitted (on demand)
	rtti           map[string]bool // interface{} type descriptors already emitted (on demand)
	rttiFns        map[string]bool // struct printing helpers already emitted (on demand)
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
	needPanicBits  bool
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
		// User functions: parameters are always passed by reference (language semantics), LLVM-level parameter type = value type*
		e.sigs[fd.name] = &funcSig{name: fd.name, params: fd.params, ret: fd.ret, byRef: true}
	}
	e.ifaces = map[string]bool{"interface{}": true} // tAny: %Iface value + RTTI descriptor
	for _, n := range lp.ifaces {
		e.ifaces[n] = true
	}
	e.vtables = lp.vtables
	for _, ed := range lp.externs {
		e.sigs[ed.name] = &funcSig{name: ed.name, params: ed.params, ret: ed.ret}
	}
	// taskm runtime (qthreads.c): pid = small integer handle
	e.sigs["ql_spawn"] = &funcSig{name: "ql_spawn", ret: "int"}
	e.sigs["ql_channel_new"] = &funcSig{name: "ql_channel_new", params: []funcParam{{typ: "int"}}, ret: "Channel"}
	e.sigs["ql_block"] = &funcSig{name: "ql_block", params: []funcParam{{typ: "int"}}, ret: "void"}
	e.sigs["ql_done"] = &funcSig{name: "ql_done", params: []funcParam{{typ: "int"}}, ret: "cbool"}
	e.sigs["ql_merge"] = &funcSig{name: "ql_merge", params: []funcParam{
		{typ: "int"}, {typ: "runner"}, {typ: "long"}, {typ: "long"}, {typ: "long"}, {typ: "long"},
		{typ: "long"}, {typ: "long"}, {typ: "long"}, {typ: "long"}}, ret: "void"}
	e.sigs["ql_send"] = &funcSig{name: "ql_send", params: []funcParam{{typ: "Channel"}, {typ: "int"}}, ret: "int"}
	e.sigs["ql_recv"] = &funcSig{name: "ql_recv", params: []funcParam{{typ: "Channel"}}, ret: "int"}
	// signature memorize (builtin, not a user function: called by value)
	e.sigs["ql_memo_new"] = &funcSig{name: "ql_memo_new", ret: "memorize"}
	// HashTable construction/key list (the other methods are emitted directly by compileTable)
	e.sigs["ql_table_new"] = &funcSig{name: "ql_table_new", ret: "HashTable"}
	e.sigs["ql_table_keys"] = &funcSig{name: "ql_table_keys", params: []funcParam{{typ: "HashTable"}}, ret: "List<String>"}
	// String builtin methods
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

// builtinDecls are the symbols the emitter always declares (a duplicate FFI declaration reports invalid redefinition).
var builtinDecls = map[string]bool{
	"printf": true, "malloc": true, "calloc": true, "free": true, "realloc": true,
	"gettimeofday": true, "ql_strcat": true, "snprintf": true, "strcmp": true,
	"strtod": true, "write": true, "exit": true,
}

// linkCandidates maps a library name to **candidate link-argument groups** (in priority order).
// qkc tries each group in turn and, if all fail, diagnoses with "library name + attempted arguments" (instead of just passing raw clang output through).
func linkCandidates(lib string) []string {
	name := strings.TrimSpace(lib)
	// Libraries provided by -L object files: no -l argument is generated (symbols come directly from the object)
	if isObjectProvidedLib(name) {
		return nil
	}
	// Path / with extension: link by file
	if strings.ContainsAny(name, "/.") {
		return []string{name}
	}
	base := strings.TrimPrefix(name, "lib")
	switch strings.ToLower(base) {
	case "c":
		// libc is linked by default; an explicit -lc is harmless (and libc.so always exists)
		return []string{"-lc"}
	case "m":
		return []string{"-lm"}
	case "dl", "rt", "pthread", "stdc++", "gcc_s", "z", "curl", "sqlite3", "png", "jpeg":
		return []string{"-l" + base}
	case "gl":
		// The interpreter tries libGL.so.1 / libgl.so.1; the link side likewise falls back on case and soname
		return []string{"-lGL", "-l:libGL.so.1", "-lgl"}
	case "vulkan":
		return []string{"-lvulkan", "-l:libvulkan.so.1"}
	case "clegrt":
		// Project-local runtime first (artifact tree/assets), then fall back to the system path
		return []string{"-L. -l" + base, "-l" + base}
	}
	return []string{"-l" + base, "-L. -l" + base}
}

// ensureIface emits the interface value type: { i8* data, i8** vt }.
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
	// After a terminator (br/ret/unreachable) the current basic block is closed: the following statements
	// are unreachable, and a second terminator must not appear in the same block (LLVM would insert an anonymous block and scramble SSA numbering).
	if strings.HasPrefix(s, "br ") || strings.HasPrefix(s, "ret ") || s == "unreachable" {
		e.term = true
	}
}

func (e *emitter) newBlock() string {
	e.blockCount++
	b := strconv.AppendInt(nil, int64(e.blockCount), 10)
	return "b" + string(b)
}

// setBlock switches to a named basic block (the block label is written on the first switch).
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

// llvmEscape escapes a byte sequence into an LLVM IR string body (printable characters as-is, everything else \XX, including the trailing NUL).
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

// ---------- Type mapping ----------

// tyName turns a language type name into an LLVM-identifier-safe type name (generic instance: Box<int> → Box_int_).
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

// ensureStruct makes sure the LLVM definition of a struct type is emitted (on demand, idempotent).
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

// ensureList makes sure the LLVM definition of a List type is emitted.
func (e *emitter) ensureList() {
	if e.hasList {
		return
	}
	e.hasList = true
	e.types.WriteString("%List = type { i32*, i32, i32 }\n") // buf, head, tail
}

// ensureListS makes sure the LLVM definition of List<String> is emitted (elements are i8* pointers).
func (e *emitter) ensureListS() {
	if e.hasListS {
		return
	}
	e.hasListS = true
	e.types.WriteString("%ListS = type { i8**, i32, i32 }\n") // buf, head, tail
}

// irElem returns the element type of an aggregate type (the GEP source type): %Point* → %Point; scalars as-is.
func (e *emitter) irElem(t string) string {
	p := e.ir(t)
	if strings.HasPrefix(p, "%") {
		return strings.TrimSuffix(p, "*")
	}
	return p
}

// ir returns the LLVM representation of a language type.
func (e *emitter) ir(t string) string {
	if w := bitsNameWidth(t); w > 0 {
		return bitsLlvmType(w) // raw bits use the smallest LLVM integer that holds the width
	}
	if t == "uchar" {
		return "i8" // the unsigned 8-bit view over bits<8>
	}
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
		return "%Iface" // an interface is a value type (two pointers: data + vtable)
	}
	// C ABI types for library FFI (f32 = single-precision float; double = double precision; cbool = C int; long = i64)
	// plus taskm runtime types (thread = pid handle i32; Channel = i8*; runner = function pointer i8*)
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
		return "i8*" // signature instance handle (ql_memo_new)
	case "HashTable":
		return "i8*" // HashTable<K,V> handle (HasPrefix was already checked above)
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

// alignOf returns the alignment of a type in bytes.
func (e *emitter) alignOf(t string) int {
	if w := storeNameWidth(t); w > 0 {
		return bitsLlvmSize(w) // the storage type's natural alignment
	}
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

// emptyString returns the empty-string pointer (the zero value of String: the interpreter prints a zero String as "",
// it cannot be null — printf("%s", NULL) prints "(null)" and strcmp would crash).
func (e *emitter) emptyString() string {
	if !e.hasEmpty {
		e.hasEmpty = true
		e.globals.WriteString("@.empty = private unnamed_addr constant [1 x i8] zeroinitializer, align 1\n")
	}
	r := e.newReg()
	e.emitInstr("%s = getelementptr inbounds [1 x i8], [1 x i8]* @.empty, i64 0, i64 0", r)
	return r
}

// zeroOf returns the zero value of a language type (as an LLVM type).
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
	return "null" // List / struct / pointer: reference zero value = null
}

// zeroStructFields initializes the reference-typed fields of a freshly allocated (calloc-zeroed) struct to zero-value objects:
// String → empty string; nested struct → recursively allocate a zero-value instance (the fields of an interpreter zero-value struct are valid objects).
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

// ---------- Program assembly ----------

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
	// String builtin method runtime (qthreads.c: UTF-8 aware, same semantics as the interpreter)
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
	// interface{} (tAny) runtime: RTTI descriptor dispatch / unboxing and equality of boxed values
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
	// HashTable<K,V> runtime
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
	e.ensureListS() // declare references %ListS, so the type definition must be emitted before the declaration
	e.decls.WriteString("declare %ListS* @ql_str_split(i8*, i8*)\n")
	// signature memorize runtime (@mb() memoization: int key → int value)
	e.decls.WriteString("declare i8* @ql_memo_new()\n")
	e.decls.WriteString("declare i32 @ql_memo_get(i8*, i32, i32*, i32*)\n")
	e.decls.WriteString("declare void @ql_memo_put(i8*, i32, i32*, i32)\n")
	// List builtin method runtime (int elements; visible range head..tail)
	e.decls.WriteString("declare void @ql_list_int_sort(i32*, i32, i32)\n")
	e.decls.WriteString("declare i8* @ql_list_int_str(i32*, i32, i32)\n\n")

	// library FFI: external symbol declarations + link-library marker (main.go parses ; qkc-link:)
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
		// Libraries provided by qkc -L (the .so is already in the output directory, dlopen at runtime):
		// **no qkc-link marker is generated**, to avoid looking for a nonexistent system library at link time.
		if isObjectProvidedLib(ed.lib) {
			continue
		}
		// Format: ; qkc-link: <library name> => <candidate argument> => <candidate argument>
		link.WriteString("; qkc-link: " + ed.lib)
		for _, cand := range linkCandidates(ed.lib) {
			link.WriteString(" => " + cand)
		}
		link.WriteString("\n")
	}

	e.vars = map[string]varSlot{}
	// Pre-emit all struct type definitions in dependency order (lower guarantees dependencies come first)
	for _, sd := range lp.structs {
		e.ensureStruct(sd.name)
	}
	// Pre-register all string constants
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
	// Pre-scan assigned variables (decides SSA passthrough); an argument passed by reference must have its own storage
	// (otherwise the callee's writes cannot be written back), so it is treated the same as "assigned".
	e.assigned = map[string]bool{}
	for _, fd := range lp.funcs {
		scanAssigned(fd.body, e.assigned)
		scanByRefArgs(fd.body, e.sigs, e.assigned)
	}
	scanAssigned(lp.mainStmts, e.assigned)
	scanByRefArgs(lp.mainStmts, e.sigs, e.assigned)

	// First generate all function bodies (string constants/type definitions are appended dynamically), then assemble at the end
	for _, fd := range lp.funcs {
		if fd.name != "main" {
			e.bodies.WriteString(e.emitFunc(fd))
		}
	}
	e.cur = "entry"
	e.curRet = "int" // main's LLVM return type is i32 (log/uniform ret i32 0 at the end)
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
	// Lib mode: **main is not emitted** (a library with main would displace the host entry point at dynamic link time and break symbol resolution)
	if !libMode {
		e.bodies.WriteString("define i32 @main()" + fnAttrs("main", e.fnMeta) + " {\n" + e.body.String() + "}\n")
	}

	helpers := ""
	if libMode {
		// Lib mode (**link-time dynamic linking**, no dlopen, consistent across systems):
		// runtime helper functions are **declared but not defined**, provided by the host program — the dependency direction is "program (incl. runtime) ← library".
		if e.needIntToStr {
			helpers += "declare i8* @ql_int_to_str(i32)\n"
		}
		if e.needFloatToStr {
			helpers += "declare i8* @ql_float_to_str(double)\n"
		}
		if e.needPtrToStr {
			helpers += "declare i8* @ql_ptr_to_str(i8*)\n"
		}
		if e.needPanic {
			helpers += "declare void @ql_panic(i8*, i32)\n"
		}
		if e.needPanicBits {
			helpers += "declare void @ql_panic_bits(i32, i32, i32)\n"
		}
	} else {
		if e.needIntToStr || forceHelpers {
			helpers += intToStrHelper
		}
		if e.needFloatToStr || forceHelpers {
			helpers += floatToStrHelper
		}
		if e.needPtrToStr || forceHelpers {
			helpers += ptrToStrHelper
		}
		if e.needPanic || forceHelpers {
			helpers += panicHelper
		}
		if e.needPanicBits || forceHelpers {
			helpers += panicBitsHelper
		}
	}
	// The runner's parameter types may emit struct definitions on demand, so this must happen before reading e.types
	runners := e.emitRunners(lp.runners)
	return link.String() + e.types.String() + e.globals.String() + e.decls.String() +
		helpers + e.helpers.String() + e.bodies.String() + e.emitVtables(e.vtables) + runners
}

// emitFunc generates the definition of a single non-main function.
func (e *emitter) emitFunc(fd *funcDef) string {
	// Note: regCount is not reset — LLVM register numbering increases globally across the module
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
	// Parameters are always passed by reference: an LLVM parameter is a pointer to "the caller's argument cell", and the callee's loads/stores
	// go straight through (writing back to the caller). copyd parameters are deep-copied into a local cell at entry.
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

// ---------- By-reference parameters (call-by-reference) ----------

// isListType reports whether a language type is List<...> (the current backend only lowers List<int>).
func isListType(t string) bool { return strings.HasPrefix(t, "List<") }

// isIfaceTypeE reports whether a type is an interface (the emitter-side interface name table).
func (e *emitter) isIfaceTypeE(t string) bool { return e.ifaces[t] }

// bindCopydParam binds a copyd parameter: deep-copy the caller cell's value into a local callee cell,
// so writes inside the callee only affect the copy (interpreter: the declaration form binds the bare value, the type form binds a Copyd wrapper).
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

// emitDeepCopyStruct emits a struct deep-copy helper and returns its symbol name (on demand, idempotent).
// Semantics match the interpreter's copydCopy: fields are copied recursively (struct recursion, deep-copied List,
// String/scalar/interface by value); null pointer fields stay null.
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
			// Scalar / String / interface: copy by value (String is immutable; interface fields were already rejected in the lower stage)
			e.emitInstr("store %s %s, %s* %s", llt, sv, llt, dg)
		}
	}
	e.emitInstr("ret %s %s", ptrTy, dst)
	body := e.body.String()
	e.body, e.vars, e.cur = savedBody, savedVars, savedCur
	e.helpers.WriteString("define " + ptrTy + " @" + sym + "(" + ptrTy + " noundef %src) {\n" + body + "}\n")
	return sym
}

// emitDeepCopyStructCall calls the struct deep-copy helper.
func (e *emitter) emitDeepCopyStructCall(src, typ string) string {
	sym := e.emitDeepCopyStruct(typ)
	r := e.newReg()
	e.emitInstr("%s = call %s @%s(%s %s)", r, e.ir(typ), sym, e.ir(typ), src)
	return r
}

// emitDeepCopyList emits a List deep-copy helper (completed inside the helper function) and returns the new List pointer.
// The visible range head..tail is copied into a new buffer (head=0, tail=size), the interpreter's copyDeep semantics.
func (e *emitter) emitDeepCopyList(src string) string {
	e.ensureDeepCopyList()
	r := e.newReg()
	e.emitInstr("%s = call %%List* @ql_copyd_list(%%List* %s)", r, src)
	return r
}

// ensureDeepCopyList emits @ql_copyd_list on demand.
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
	// New buffer: at least 1 element (consistent with the capacity assumption of List literals/append)
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
	// Copy the visible elements in a loop
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

// ---------- Statements ----------

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
	case *bitAssignStmt:
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
			return // statements after a terminator are unreachable, skip
		}
		e.emitStmt(s)
	}
}

func (e *emitter) emitStmt(s stmt) {
	switch st := s.(type) {
	case *exprStmt:
		e.compileExpr(st.x) // evaluate for side effects, discard the result
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
	case *bitAssignStmt:
		e.emitBitAssign(st)
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
		e.compileExpr(st.x) // evaluate (side effects), the result only goes to the log
		e.emitRetZero()     // end the function right after the log record (the interpreter returns nil)
	}
}

// emitRetZero emits a zero-value return of the current function's return type (used by log).
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
	// Tail call optimization: return f(args) → tail call (tail-recursive stack O(1)).
	// With by-reference passing a tail call is only possible when "the argument address is stable for the duration of the call": if an argument is a
	// temporary alloca in the current frame, the tail call would invalidate the pointer (LLVM tail call rules), so we fall back to a normal call.
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
	v = e.coerceTo(v, vt, e.curRet)
	e.emitInstr("ret %s %s", e.ir(e.curRet), v)
	e.funcReturned = true
}

// compileArgs compiles call arguments and applies implicit conversions per the callee signature (int → float).
// Returns a list of "type register" strings; the second return value reports whether all arguments are tail-call safe
// (with by-reference passing and an argument cell inside the caller's frame, a tail call would invalidate the pointer, so TCO must be disabled).
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
		v = e.coerceTo(v, vt, want)
		// Variadic (printf style) is not covered here; every IR function in this backend has fixed arity
		out = append(out, e.ir(want)+" "+v)
	}
	return out, tailSafe
}

// compileArgAddr prepares storage for an argument passed by reference; returns (address, language type, tail safety).
//   - lvalue argument (variable/field/List index) with an exactly matching type → use its own storage (writes back to the caller);
//   - everything else (literal, arithmetic result, needs an int→float conversion, needs boxing) → a caller temporary cell,
//     where the callee's writes are not written back (consistent with the interpreter's "non-lvalue arguments are temporary cells").
//
// tailSafe reports whether that address is stable for the whole call (heap object/caller storage = safe;
// an alloca temporary cell in the current frame = invalidated by a tail call).
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
	v = e.coerceTo(v, vt, want)
	llt := e.ir(want)
	slot := e.newReg()
	e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(want))
	e.emitInstr("store %s %s, %s* %s", llt, v, llt, slot)
	return slot, want, false
}

// lvalueAddr tries to take the lvalue address of an expression (returns address, language type, tail-call safety).
// Only the three lvalue kinds the backend already has are supported: variable, struct field, List index (the same set as the interpreter's evalArg).
func (e *emitter) lvalueAddr(x *expr) (string, string, bool, bool) {
	switch x.kind {
	case kIdent:
		info, ok := e.vars[x.s]
		if !ok || info.param || info.direct {
			return "", "", false, false // SSA passthrough/register variables have no mutable storage
		}
		// ref=true: the storage belongs to the caller (or the heap), tail-call safe; a local alloca is not
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
				// A struct instance is either on the heap or belongs to the caller → the address is stable
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
		return g2, "int", true, true // the buffer is on the heap
	}
	return "", "", false, false
}

// sigRet returns the return type of the callee (assumed i32 when there is no signature).
func (e *emitter) sigRet(name string) string {
	if s, ok := e.sigs[name]; ok {
		return s.ret
	}
	return "int"
}

// coerce performs implicit conversions between types: language-level int → float (double), plus FFI's
// f32 ↔ double / bool ↔ C int boundary conversions.
func (e *emitter) coerce(reg, from, to string) string {
	if from == to || to == "" || to == "?" {
		return reg
	}
	if from == "uchar" || isBitsT(from) || to == "uchar" || isBitsT(to) {
		return e.convToType(reg, from, to) // raw bits and uchar re-read at the target's width
	}
	// interface{} → concrete scalar/String: unbox at runtime after checking the RTTI kind (an explicit runtime error on mismatch)
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

// ---------- Declaration / assignment ----------

func (e *emitter) emitDecl(st *declStmt) {
	// T& / pointer T: the slot holds a pointer; new T allocates a cell, no initial value → null
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
	// HashTable<K,V>: an i8* handle (construction is already emitted by HashTable::new())
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
	if isScalarStore(st.typ) {
		// SSA passthrough: a single-assignment scalar uses a register directly (no alloca/load/store)
		if !e.assigned[st.name] && st.init != nil && !e.funcReturned {
			v, vt := e.compileExpr(st.init)
			e.vars[st.name] = varSlot{reg: e.coerceTo(v, vt, st.typ), typ: st.typ, direct: true}
			return
		}
		reg := e.newReg()
		llt := e.ir(st.typ)
		e.emitInstr("%s = alloca %s, align %d", reg, llt, e.alignOf(st.typ))
		v := e.zeroOf(st.typ)
		if st.init != nil {
			x, xt := e.compileExpr(st.init)
			v = e.coerceTo(x, xt, st.typ)
		}
		e.emitInstr("store %s %s, %s* %s", llt, v, llt, reg)
		e.vars[st.name] = varSlot{reg: reg, typ: st.typ}
		return
	}
	switch st.typ {
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
			// List variable / call result: reference semantics, store the pointer directly
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
		// struct type: reference semantics (calloc zero value + optional literal field writes)
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

// listSObj returns the object pointer of a List<String> variable.
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

// listSRange returns the visible range and buffer of a List<String>.
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

// listSSlot stores a List<String> object pointer into an alloca slot (variable reference semantics).
func (e *emitter) listSSlot(objPtr string) string {
	e.ensureListS()
	slot := e.newReg()
	e.emitInstr("%s = alloca %%ListS*, align 8", slot)
	e.emitInstr("store %%ListS* %s, %%ListS** %s", objPtr, slot)
	return slot
}

// listSlot stores a List object pointer into an alloca slot (variable reference semantics: the slot holds a pointer).
func (e *emitter) listSlot(objPtr string) string {
	slot := e.newReg()
	e.emitInstr("%s = alloca %%List*, align 8", slot)
	e.emitInstr("store %%List* %s, %%List** %s", objPtr, slot)
	return slot
}

// structAlloc allocates a calloc-zeroed struct instance (LLVM layout: GEP null,1 → real size,
// avoiding out-of-bounds writes from hand-computed field alignment).
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
		// T& variable: write through the cell the pointer points at; when the pointer is null (uninitialized), allocate the cell lazily
		// (interpreter semantics: int& q; q = 7 → q itself holds that value)
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
		v = e.coerceTo(v, vt, base)
		e.emitInstr("store %s %s, %s* %s", blt, v, blt, cell)
		return
	}
	v, vt := e.compileExpr(st.x)
	v = e.coerceTo(v, vt, info.typ)
	if info.param || info.direct {
		// Parameters/passthrough variables should never be assigned (guaranteed by lowering's assigned analysis)
		e.emitInstr("; assignment to register variable %s", st.name)
		return
	}
	llt := e.ir(info.typ)
	e.emitInstr("store %s %s, %s* %s", llt, v, llt, info.reg)
}

func (e *emitter) emitIndexAssign(st *indexAssignStmt) {
	// l[i] = v (the receiver may be any List<int>/List<String> expression)
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
	v = e.coerceTo(v, vt, ftyp)
	e.emitInstr("store %s %s, %s* %s", e.ir(ftyp), v, e.ir(ftyp), g)
}

// listObj returns the object pointer of a List variable.
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

// listBuf returns the buffer pointer of a List.
func (e *emitter) listBuf(obj string) string {
	bf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%List, %%List* %s, i32 0, i32 0", bf, obj)
	p := e.newReg()
	e.emitInstr("%s = load i32*, i32** %s", p, bf)
	return p
}

// listHeadSize returns a List's head cursor and the number of visible elements (tail-head).
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

// emitBoundsCheck performs a runtime check on a List index (an already-evaluated visible-index int register):
// 0 <= i < size; out of bounds → jump to catch inside try, otherwise a runtime error (the same message as the interpreter).
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

// emitIndexPanic emits the List out-of-bounds runtime error.
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
		e.emitInstr("br label %%%s", afterL) // the normal path skips catch
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
		return // by-reference parameter: the storage belongs to the caller and must not be freed here
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

// emitFor emits a C-style for: init; cond; step.
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

// emitForIn emits an iteration for: for (T x : list) — consistent with the interpreter, consuming via a rolling cursor
// (head advances; after the loop the elements are consumed and size() is zero).
func (e *emitter) emitForIn(st *forInStmt) {
	slot := e.newReg()
	llt := e.ir(st.typ)
	e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(st.typ))
	e.vars[st.name] = varSlot{reg: slot, typ: st.typ}
	condB, bodyB, endB := e.newBlock(), e.newBlock(), e.newBlock()
	if st.typ == "String" {
		// List<String>: elements are i8* (the sliding head cursor semantics match List<int>)
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

// emitPrint emits io.println / io.print (newline decides whether a newline is printed).
func (e *emitter) emitPrint(args []*expr, newline bool) {
	type av struct {
		val string
		typ string
	}
	vals := make([]av, 0, len(args))
	fmts := make([]string, 0, len(args))
	for _, a := range args {
		if a.typ == "uchar" || isBitsT(a.typ) {
			// Raw bits and uchar print their unsigned decimal value (spec §3.1)
			v, _ := e.compileExpr(a)
			vals = append(vals, av{e.zextToI64(v, a.typ), "long"})
			fmts = append(fmts, "%llu")
			continue
		}
		v, t := e.compileExpr(a)
		// T& argument: null → "nil" (the interpreter prints the nil value as nil); convert to String uniformly, then select
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
			// Runtime dispatch by RTTI kind to the concrete type's stringification (null → "nil")
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

// llvmFloat formats a float64 as an LLVM floating-point literal (it must contain a decimal point/exponent,
// otherwise "2" is taken as an integer constant: fadd double 1.5, 2 is illegal).
func llvmFloat(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eEnN") {
		s += ".0"
	}
	return s
}

// toI1 converts a value to i1 (bool used directly; int tested against zero).
func (e *emitter) toI1(reg string) string {
	if reg == "true" || reg == "false" {
		return reg
	}
	return reg
}

// toI64 extends an i32 register to i64 (literals can be used directly).
func (e *emitter) toI64(reg string) string {
	if strings.HasPrefix(reg, "%") {
		r := e.newReg()
		e.emitInstr("%s = sext i32 %s to i64", r, reg)
		return r
	}
	return reg
}

// ---------- Expressions ----------

// compileExpr compiles an expression and returns (register, language type) (including interface boxing).
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

// ---------- interface{} (tAny): boxing + runtime type descriptor (RTTI) ----------
//
// The value representation keeps %Iface { i8* data, i8** rt }: rt points to the type descriptor
//
//	%RT = type { i8* (i8*)* str, i8* name, i32 kind }   kind: 0=int 1=float 2=bool
//	                                                           3=String 4=List<int> 5=struct 6=null
//
// data convention: a scalar/String points to a heap cell allocated for the call (value semantics, boxing is a snapshot); List/struct are directly
// the object pointer (reference semantics, consistent with the interpreter's *List/*Struct); null is the zero value.
// Printing/equality/unboxing are all dispatched at runtime by kind (qthreads.c's ql_any_*; a struct's str is generated by the compiler).

// anyKindOf returns the RTTI kind of a type.
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

// anyTypeName returns the type name consistent with the interpreter's Value.TypeName() (the HashTable key rule and
// interface{} unboxing diagnostics both depend on it).
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
	return t // struct: SType
}

// anyCopyFn returns the RTTI deep-copy function of the type (semantics aligned with the interpreter's deepCopy:
// List copies the buffer, struct shares the pointer, scalar/String copies the heap cell).
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

// ensureRT emits the LLVM struct definition of a type descriptor.
func (e *emitter) ensureRT() {
	if e.hasRT {
		return
	}
	e.hasRT = true
	e.types.WriteString("%RT = type { i8* (i8*)*, i8* (i8*)*, i8*, i32 }\n")
}

// rttiSym returns (emitting on demand) a type descriptor symbol.
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
		e.needIntToStr = true // reuse the same int→String formatting
		strFn = "@ql_any_str_int"
	case 1:
		e.needFloatToStr = true // reuse the same float→String formatting
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

// boxAny boxes a concrete value into an interface{} value.
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
		// struct / List<int>: the value itself is an object pointer (reference semantics)
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

// emitStructAnyStr generates the interface{} printing function of a struct (the interpreter's StructValue.String format:
// <T {field=value, ...}>, fields in declaration order).
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

// anyValueStr returns the string form of an arbitrary field value (used by struct printing; aligned with the interpreter's Value.String).
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

// strcat2 concatenates two String registers (ql_strcat).
func (e *emitter) strcat2(a, b string) string {
	r := e.newReg()
	e.emitInstr("%s = call i8* @ql_strcat(i8* %s, i8* %s)", r, a, b)
	return r
}

// boxIface boxes a concrete struct pointer into an interface value { data, vtable }.
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

// compileExprRaw compiles an expression (without the boxing wrapper).
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
	case kListS:
		return e.compileListSLit(x)
	case kConv:
		return e.compileConv(x)
	}
	return "0", "int"
}

// loadVar reads a variable (slot → load; parameter/passthrough → used directly).
// T& / pointer T variables are dereferenced automatically by default (interpreter semantics); with noDeref the pointer itself is taken.
func (e *emitter) loadVar(name string, noDeref bool) (string, string) {
	info, ok := e.vars[name]
	if !ok {
		return name, "int" // undeclared variable: let the generated IR fail (llvm-as verification catches it)
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
		// Nullable reference: when null, read a temporary cell (no crash; printing/comparison are handled with nil semantics at the call site)
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
		fv = e.coerceTo(fv, ft, fields[i].typ)
		g := e.newReg()
		e.emitInstr("%s = getelementptr inbounds %s, %s %s, i32 0, i32 %d", g, e.irElem(typ), ptrTy, v, i)
		e.emitInstr("store %s %s, %s* %s", e.ir(fields[i].typ), fv, e.ir(fields[i].typ), g)
	}
	return v, typ
}

func (e *emitter) compileIndex(x *expr) (string, string) {
	if x.bits > 0 {
		return e.compileBitIndex(x)
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

// compileBitIndex reads the i-th bit of raw bits as a bit (spec §3.3); the index is bounds-checked
// with the interpreter's error message.
func (e *emitter) compileBitIndex(x *expr) (string, string) {
	recv, rt := e.compileExpr(x.idx.recv)
	llty := e.ir(rt)
	v := e.convToType(recv, rt, rt)
	i, ity := e.compileExpr(x.idx.i)
	cnt := e.toIntWidth(i, ity, llty)
	e.emitBitBoundsCheck(e.toIntWidth(i, ity, "i32"), x.bits, x.line)
	sh := e.newReg()
	e.emitInstr("%s = lshr %s %s, %s", sh, llty, v, cnt)
	r := e.newReg()
	e.emitInstr("%s = and %s %s, 1", r, llty, sh)
	return r, "bit"
}

// emitBitBoundsCheck reports an i32 bit index outside 0..width-1 as a runtime error (spec §3.3).
func (e *emitter) emitBitBoundsCheck(i32idx string, width, line int) {
	bad := e.newReg()
	e.emitInstr("%s = icmp uge i32 %s, %d", bad, i32idx, width)
	okB := e.newBlock()
	if e.curTry != "" {
		e.emitInstr("br i1 %s, label %%%s, label %%%s", bad, e.curTry, okB)
		e.setBlock(okB)
		return
	}
	e.needPanicBits = true
	tgt := e.newBlock()
	e.emitInstr("br i1 %s, label %%%s, label %%%s", bad, tgt, okB)
	e.setBlock(tgt)
	e.emitInstr("call void @ql_panic_bits(i32 %s, i32 %d, i32 %d)", i32idx, width, line)
	e.emitInstr("unreachable")
	e.setBlock(okB)
}

// emitBitAssign writes one bit of raw bits: the lvalue is read, the bit is replaced and stored back
// (spec §3.3).
func (e *emitter) emitBitAssign(st *bitAssignStmt) {
	addr, typ, _, ok := e.lvalueAddr(st.recv)
	if !ok {
		e.emitInstr("; b[i] = v needs an addressable raw-bits target")
		return
	}
	llty := e.ir(typ)
	cur := e.newReg()
	e.emitInstr("%s = load %s, %s* %s", cur, llty, llty, addr)
	i, ity := e.compileExpr(st.idx)
	cnt := e.toIntWidth(i, ity, llty)
	e.emitBitBoundsCheck(e.toIntWidth(i, ity, "i32"), st.bits, st.idx.line)
	bit := e.newReg()
	e.emitInstr("%s = shl %s 1, %s", bit, llty, cnt)
	cleared := e.newReg()
	e.emitInstr("%s = xor %s %s, -1", cleared, llty, bit)
	keep := e.newReg()
	e.emitInstr("%s = and %s %s, %s", keep, llty, cur, cleared)
	v, vt := e.compileExpr(st.x)
	cv := e.convToType(v, vt, typ)
	set := e.newReg()
	e.emitInstr("%s = and %s %s, 1", set, llty, cv)
	next := e.newReg()
	e.emitInstr("%s = shl %s %s, %s", next, llty, set, cnt)
	res := e.newReg()
	e.emitInstr("%s = or %s %s, %s", res, llty, keep, next)
	e.emitInstr("store %s %s, %s* %s", llty, e.maskToWidth(res, llty, st.bits), llty, addr)
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

// compileListSLit builds a List<String> literal: a heap %ListS object whose buffer holds i8* elements.
// The element buffer is calloc-ed (not malloc) so that an untouched tail slot is a null pointer, never
// garbage; the interpreter prints a zero String as "", and calloc keeps the two backends identical.
func (e *emitter) compileListSLit(x *expr) (string, string) {
	e.ensureListS()
	n := len(x.lst.items)
	obj := e.newReg()
	e.emitInstr("%s = call i8* @calloc(i64 1, i64 16)", obj)
	lo := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to %%ListS*", lo, obj)
	buf := e.newReg()
	e.emitInstr("%s = call i8* @calloc(i64 %d, i64 8)", buf, max(1, n))
	p := e.newReg()
	e.emitInstr("%s = bitcast i8* %s to i8**", p, buf)
	for i, it := range x.lst.items {
		v, vt := e.compileExpr(it)
		v = e.coerce(v, vt, "String")
		g2 := e.newReg()
		e.emitInstr("%s = getelementptr inbounds i8*, i8** %s, i64 %d", g2, p, i)
		e.emitInstr("store i8* %s, i8** %s", v, g2)
	}
	bf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 0", bf, lo)
	e.emitInstr("store i8** %s, i8*** %s", p, bf)
	tf := e.newReg()
	e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 2", tf, lo)
	e.emitInstr("store i32 %d, i32* %s", n, tf)
	return lo, "List<String>"
}

// compileCall compiles ordinary function calls and builtins (sum/clock).
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
		return r, "float" // FFI double → language float
	case "long", "pointer":
		return r, ret // FFI long → i64; pointer → i8* opaque handle
	}
	return r, ret
}

// compileMemoCall compiles a signature memorize call f(args) @mb():
// look up the table by key (the int argument array); on a miss, call the wrapped function and record the result.
func (e *emitter) compileMemoCall(x *expr) string {
	c := x.call
	// explicit @ prefix arguments: evaluated and discarded (the interpreter likewise only puts them into the prefix record)
	if x.memo != nil {
		for _, s := range x.memo.skips {
			e.compileExpr(s)
		}
	}
	handle, _ := e.compileExpr(x.memo.handle)
	n := len(c.args)
	// argument cells (by value: the interpreter calls after derefArgs)
	cells := make([]string, 0, n)
	for _, a := range c.args {
		v, vt := e.compileExpr(a)
		v = e.coerce(v, vt, "int")
		cells = append(cells, e.valueOfCell(v, "int"))
	}
	// key array [n x i32]
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

// compileMerge compiles taskm.merge(pid, runner, args...): arguments are packed into i64 slots handed to the runtime,
// and the runner restores them into the target function's parameter cells (by-reference passing).
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

// packI64 packs an argument value into an i64 slot (the runtime's ql_task.args is 4 long longs).
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
		// String / pointer / struct / List / Channel / memorize: pointer value
		e.emitInstr("%s = ptrtoint %s %s to i64", r, e.ir(typ), v)
	}
	return r
}

// compileNewRef compiles new T: allocate a zeroed T cell and return a T& pointer.
func (e *emitter) compileNewRef(x *expr) (string, string) {
	base := x.s
	ptlt := e.ir(x.typ) // pointer type of T& (i32* / %P* / i8** …)
	// size: scalars use the LLVM type size, struct uses GEP null,1
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

// nilSelect chooses "nil" or the given string based on null (the nil semantics of dereferencing a nullable reference).
func (e *emitter) nilSelect(isnull, s string) string {
	nilp := e.i8Ptr(e.strConst("nil"))
	r := e.newReg()
	e.emitInstr("%s = select i1 %s, i8* %s, i8* %s", r, isnull, nilp, s)
	return r
}

// sizeOf returns the byte size of a scalar type (struct goes through GEP computation).
func (e *emitter) sizeOf(t string) int {
	if w := storeNameWidth(t); w > 0 {
		return bitsLlvmSize(w) // raw bits take ceil(N/8) bytes, uchar one byte
	}
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

// ============ Raw bits and the uchar view (spec §3) ============

// bitsNameWidth returns the width named by a raw-bits type ("bit" → 1, "bits<N>" → N), or 0 when the
// name is not a raw-bits type.
func bitsNameWidth(t string) int {
	if t == "bit" {
		return 1
	}
	if !strings.HasPrefix(t, "bits<") || !strings.HasSuffix(t, ">") {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(t, "bits<"), ">"))
	if err != nil || n < 1 {
		return 0
	}
	return n
}

// isBitsT reports whether a type name is raw bits (bit or bits<N>).
func isBitsT(t string) bool { return bitsNameWidth(t) > 0 }

// rawNameOf returns the canonical name of raw bits of a width (1 is spelled bit, because bits<1> IS bit).
func rawNameOf(n int) string {
	if n <= 1 {
		return "bit"
	}
	return "bits<" + strconv.Itoa(n) + ">"
}

// storeNameWidth returns the width a value stored in this type is masked to (raw bits and the uchar
// view); 0 means the type stores its own bits.
func storeNameWidth(t string) int {
	if w := bitsNameWidth(t); w > 0 {
		return w
	}
	if t == "uchar" {
		return 8
	}
	return 0
}

// bitsLlvmType returns the smallest LLVM integer that holds a raw width.
func bitsLlvmType(width int) string {
	switch {
	case width <= 8:
		return "i8"
	case width <= 16:
		return "i16"
	}
	return "i32"
}

// bitsLlvmSize returns the byte size of the LLVM integer that holds a raw width.
func bitsLlvmSize(width int) int {
	switch bitsLlvmType(width) {
	case "i8":
		return 1
	case "i16":
		return 2
	}
	return 4
}

// intLlvmWidth returns the bit width of an LLVM integer type name; 0 = not an integer type.
func intLlvmWidth(ty string) int {
	switch ty {
	case "i1":
		return 1
	case "i8":
		return 8
	case "i16":
		return 16
	case "i32":
		return 32
	case "i64":
		return 64
	}
	return 0
}

// isSignedName reports whether a language type reads its bits as two's complement.
func isSignedName(t string) bool { return t == "int" || t == "long" || t == "char" }

// isScalarStore reports whether a type is stored in a single scalar slot.
func isScalarStore(t string) bool {
	switch t {
	case "int", "bool", "float", "long", "String", "pointer", "thread", "Channel", "channel", "memorize", "uchar":
		return true
	}
	return isBitsT(t)
}

// toIntWidth re-reads an integer register as another LLVM integer type, sign-extending a signed view
// and zero-extending everything else.
func (e *emitter) toIntWidth(v, from, to string) string {
	fty := e.ir(from)
	if fty == to {
		return v
	}
	fw, tw := intLlvmWidth(fty), intLlvmWidth(to)
	if fw == 0 || tw == 0 {
		return v // not an integer pair: the caller handles it
	}
	r := e.newReg()
	switch {
	case fw < tw && isSignedName(from):
		e.emitInstr("%s = sext %s %s to %s", r, fty, v, to)
	case fw < tw:
		e.emitInstr("%s = zext %s %s to %s", r, fty, v, to)
	default:
		e.emitInstr("%s = trunc %s %s to %s", r, fty, v, to)
	}
	return r
}

// maskToWidth clears every bit at or above width, so bits past N are zero (spec §3.1).
func (e *emitter) maskToWidth(v, llty string, width int) string {
	full := intLlvmWidth(llty)
	if width <= 0 || full == 0 || width >= full {
		return v
	}
	r := e.newReg()
	e.emitInstr("%s = and %s %s, %d", r, llty, v, (int64(1)<<uint(width))-1)
	return r
}

// zextToI64 reads a raw-bits or uchar value as an unsigned i64, so that %llu prints its unsigned
// decimal value.
func (e *emitter) zextToI64(v, from string) string {
	return e.toIntWidth(v, from, "i64")
}

// convToType re-reads an integer value at another type's width, keeping the low bits: the conversion
// rule of spec §3.4 and the store rule of raw bits and uchar.
func (e *emitter) convToType(v, from, to string) string {
	llty := e.ir(to)
	return e.maskToWidth(e.toIntWidth(v, from, llty), llty, storeNameWidth(to))
}

// coerceTo converts a value to a target type: raw bits and uchar are masked to their width, every
// other target keeps the ordinary coercion rules.
func (e *emitter) coerceTo(v, from, to string) string {
	if to == "uchar" || isBitsT(to) {
		return e.convToType(v, from, to)
	}
	return e.coerce(v, from, to)
}

// compileTable compiles a HashTable method call: keys are structured into strings by the interpreter's rule
// (TypeName:String()), and values are boxed into interface{} (deep-copied per RTTI on Put).
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
			// missing key → (null, null) = nil (consistent with the interpreter's get returning NilV)
			i0 := e.newReg()
			e.emitInstr("%s = insertvalue %%Iface undef, i8* %s, 0", i0, d)
			i1 := e.newReg()
			e.emitInstr("%s = insertvalue %%Iface %s, i8** %s, 1", i1, i0, rt)
			return i1, "interface{}"
		}
		// concrete V: unbox per RTTI; a missing key/type mismatch → an explicit runtime error (nil cannot be carried by a static type)
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
			// struct / List: what is stored is the object pointer (struct shared and List deep-copied on Put)
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

// tableKey boxes the key expression and converts it into the interpreter's key string (TypeName:String()).
func (e *emitter) tableKey(k *expr) string {
	d, rt := e.anyPair(k)
	r := e.newReg()
	e.emitInstr("%s = call i8* @ql_table_key(i8* %s, i8** %s)", r, d, rt)
	return r
}

// anyPair takes the packed interface{} value (data, rt) of an expression.
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

// isAnyType reports whether the type is the empty interface interface{}.
func isAnyType(t string) bool { return strings.TrimSpace(t) == "interface{}" }

// ptrRefBase reports whether a language type is T& / pointer T (a nullable reference):
// returns (full type, pointee type, whether).
func ptrRefBase(t string) (string, string, bool) {
	t = strings.TrimSpace(t)
	if t == "pointer" || t == "" {
		return "", "", false // bare pointer = FFI opaque handle (i8*), not T&
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

// compileMethod compiles instance method calls and builtin methods.
func (e *emitter) compileMethod(x *expr) (string, string) {
	m := x.method
	if m.iface != "" {
		return e.compileIfaceCall(x)
	}
	recv, rtyp := e.compileExpr(m.recv)
	// List<String> builtin methods (keys() results etc.; only the read-only subset is lowered)
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
		case "append":
			// geometric growth with 8-byte elements (mirrors the List<int> path; elements are i8*)
			tf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 2", tf, recv)
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
			e.emitInstr("%s = mul i64 %s, 8", sz, n64)
			bf := e.newReg()
			e.emitInstr("%s = getelementptr inbounds %%ListS, %%ListS* %s, i32 0, i32 0", bf, recv)
			lp := e.newReg()
			e.emitInstr("%s = load i8**, i8*** %s", lp, bf)
			bc := e.newReg()
			e.emitInstr("%s = bitcast i8** %s to i8*", bc, lp)
			rc := e.newReg()
			e.emitInstr("%s = call i8* @realloc(i8* %s, i64 %s)", rc, bc, sz)
			np := e.newReg()
			e.emitInstr("%s = bitcast i8* %s to i8**", np, rc)
			e.emitInstr("store i8** %s, i8*** %s", np, bf)
			t1i := e.newReg()
			e.emitInstr("%s = sext i32 %s to i64", t1i, t1)
			g2 := e.newReg()
			e.emitInstr("%s = getelementptr inbounds i8*, i8** %s, i64 %s", g2, np, t1i)
			v, vt := e.compileExpr(m.args[0])
			v = e.coerce(v, vt, "String")
			e.emitInstr("store i8* %s, i8** %s", v, g2)
			t2 := e.newReg()
			e.emitInstr("%s = add i32 %s, 1", t2, t1)
			e.emitInstr("store i32 %s, i32* %s", t2, tf)
			return "0", "void"
		}
	}
	// List builtin methods
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
			// append to self's tail (realloc growth, same as append)
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
	// struct instance method: call @Type_method(self*, args...)
	// the receiver is bound by value (interpreter: self = ... does not write back to the caller) → temporary cell;
	// the remaining arguments are passed by reference (interpreter evalArgs: lvalue arguments write back to the caller).
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
				v = e.coerceTo(v, vt, want)
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

// compileIfaceCall compiles an interface method call: fetch the function pointer from the vtable and call the concrete method through a thunk.
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
			// Self parameter: interface value → take data (the concrete instance pointer) and put it into a temporary cell passed by reference
			av, _ := e.compileExpr(a)
			d := e.newReg()
			e.emitInstr("%s = extractvalue %%Iface %s, 0", d, av)
			pty = append(pty, "i8*")
			callArgs = append(callArgs, "i8* "+d)
			continue
		}
		// remaining arguments: the language level passes by reference → pass the value cell address (the thunk forwards it unchanged to the concrete method)
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

// thunkName is the thunk name for (concrete type, interface, method).
func thunkName(th *ifaceThunk) string {
	return "thunk$" + tyName(th.typ) + "$" + tyName(th.iface) + "$" + th.method
}

// thunkSig returns the thunk's LLVM function signature "ret (params)" (self data is always i8*).
func (e *emitter) thunkSig(th *ifaceThunk) string {
	ps := []string{"i8*"}
	for i, p := range th.params {
		if isSelfIdx(th.selfIdx, i) {
			ps = append(ps, "i8*")
			continue
		}
		ps = append(ps, e.ir(p)+"*") // pass by reference
	}
	retIR := e.ir(th.ret)
	if th.boxRet {
		e.ensureIface()
		retIR = "%Iface" // Self return: boxed and returned as an interface value
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

// emitVtables emits all vtable constants and thunk functions (at the end of the module; forward references are legal).
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

// emitThunk generates one thunk: i8* data → concrete type pointer → call the concrete method (a Self return is boxed again).
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
			// Self parameter: interface data (i8*) → concrete type pointer → temporary cell (passed by reference)
			c := e.newReg()
			e.emitInstr("%s = bitcast i8* %s to %s", c, regs[i], e.ir(p))
			callArgs = append(callArgs, e.ir(p)+"* "+e.valueOfCell(c, p))
			continue
		}
		// remaining parameters: the thunk forwards the caller's cell address directly
		callArgs = append(callArgs, e.ir(p)+"* "+regs[i])
	}
	callRet := e.ir(th.ret) // the concrete method's real return type (before boxing)
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

// emitClock emits clock() (gettimeofday microseconds).
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

// valueOfCell puts a value into a temporary cell of the current frame and returns the cell address (a value argument passed by reference).
func (e *emitter) valueOfCell(v, typ string) string {
	llt := e.ir(typ)
	slot := e.newReg()
	e.emitInstr("%s = alloca %s, align %d", slot, llt, e.alignOf(typ))
	e.emitInstr("store %s %s, %s* %s", llt, v, llt, slot)
	return slot
}

// emitSum expands sum(g, begin, stop[, step]) inline into a loop.
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

// compileBin compiles binary operations (arithmetic/comparison/logic/String concatenation/operator overload).
func (e *emitter) compileBin(x *expr) (string, string) {
	if x.op == "&&" || x.op == "||" {
		return e.compileLogic(x)
	}
	if x.kind == kCmp {
		return e.compileCmp(x)
	}
	if x.bits > 0 {
		return e.compileBitsBin(x) // raw bits keep their width and never wrap like an int (spec §3.3)
	}
	lv, lt := e.compileExpr(x.l)
	rv, rt := e.compileExpr(x.r)
	if x.strcat {
		rc := e.newReg()
		e.emitInstr("%s = call i8* @ql_strcat(i8* %s, i8* %s)", rc, lv, rv)
		return rc, "String"
	}
	// operator overload: struct operand → call __add__ and friends
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
			// interpreter semantics: float modulo reports a TypeError at runtime
			e.emitAbortF("TypeError: '%' requires int operands", x.line)
			return "0.0", "float" // dead code after this
		}
		op := map[string]string{"+": "fadd", "-": "fsub", "*": "fmul", "/": "fdiv"}[x.op]
		reg := e.newReg()
		e.emitInstr("%s = %s double %s, %s", reg, op, lv, rv)
		return reg, "float"
	}
	if v, ok := constEval(x); ok {
		return fmt.Sprintf("%d", int32(v)), "int"
	}
	// long in arithmetic: the interpreter truncates to 32 bits (wrapI32), so we truncate first here as well
	lv = e.coerce(lv, lt, "int")
	rv = e.coerce(rv, rt, "int")
	if x.op == "/" || x.op == "%" {
		e.emitZeroCheckI(rv, x.op, x.line)
	}
	if x.op == "<<" || x.op == ">>" {
		return e.emitIntShift(x.op, lv, rv), "int"
	}
	op := map[string]string{"+": "add", "-": "sub", "*": "mul", "/": "sdiv", "%": "srem",
		"&": "and", "|": "or", "^": "xor"}[x.op]
	reg := e.newReg()
	e.emitInstr("%s = %s i32 %s, %s", reg, op, lv, rv)
	return reg, "int"
}

// emitIntShift emits a shift of the int view; the count is masked to 5 bits because the interpreter
// shifts by sh&31 (a count of 32 or more would be poison in LLVM).
func (e *emitter) emitIntShift(op, lv, rv string) string {
	cnt := e.newReg()
	e.emitInstr("%s = and i32 %s, 31", cnt, rv)
	instr := "shl"
	if op == ">>" {
		instr = "ashr" // arithmetic: the interpreter sign-extends
	}
	reg := e.newReg()
	e.emitInstr("%s = %s i32 %s, %s", reg, instr, lv, cnt)
	return reg
}

// compileBitsBin emits the bitwise family on raw bits, masking the result to the operand width (spec §3.3).
func (e *emitter) compileBitsBin(x *expr) (string, string) {
	llty := e.ir(x.typ)
	lv, lt := e.compileExpr(x.l)
	lv = e.convToType(lv, lt, x.typ)
	rv, rt := e.compileExpr(x.r)
	if x.op == "<<" || x.op == ">>" {
		return e.emitBitsShift(x.op, lv, rv, rt, llty, x.bits), x.typ
	}
	rv = e.convToType(rv, rt, x.typ) // a constant adapts to the raw width
	op := map[string]string{"&": "and", "|": "or", "^": "xor"}[x.op]
	reg := e.newReg()
	e.emitInstr("%s = %s %s %s, %s", reg, op, llty, lv, rv)
	return e.maskToWidth(reg, llty, x.bits), x.typ
}

// emitBitsShift emits a shift of raw bits: the count is compared in 64 bits and clamped so the shift
// is never poison, and a count at or above the width shifts every bit out (spec §3.3).
func (e *emitter) emitBitsShift(op, lv, rv, rt, llty string, width int) string {
	cnt := e.toIntWidth(rv, rt, "i64")
	over := e.newReg()
	e.emitInstr("%s = icmp uge i64 %s, %d", over, cnt, width)
	clamped := e.newReg()
	e.emitInstr("%s = select i1 %s, i64 %d, i64 %s", clamped, over, width-1, cnt)
	safe := e.newReg()
	e.emitInstr("%s = trunc i64 %s to %s", safe, clamped, llty)
	instr := "shl"
	if op == ">>" {
		instr = "lshr" // vacated bits are zero (raw bits are unsigned)
	}
	sh := e.newReg()
	e.emitInstr("%s = %s %s %s, %s", sh, instr, llty, lv, safe)
	zero := e.newReg()
	e.emitInstr("%s = select i1 %s, %s 0, %s %s", zero, over, llty, llty, sh)
	return e.maskToWidth(zero, llty, width)
}

// compileConv lowers a conversion call T(x): the value is re-read at the destination's width, keeping
// the low bits (spec §3.4).
func (e *emitter) compileConv(x *expr) (string, string) {
	v, vt := e.compileExpr(x.l)
	return e.convToType(v, vt, x.s), x.s
}

// emitZeroCheckI integer division-by-zero check (jump to catch inside try, otherwise a runtime error).
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

// emitAbortF emits a floating-point path that "always fails" (the interpreter reports a runtime error here):
// jump to catch inside try, otherwise print the same message and exit.
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

// compileLogic short-circuit evaluation of && / || (interpreter semantics: the right side is evaluated only when needed).
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

// compileCmp compiles comparison operations (int/float/bool/String/struct).
func (e *emitter) compileCmp(x *expr) (string, string) {
	lv, lt := e.compileExpr(x.l)
	rv, rt := e.compileExpr(x.r)
	// interface{} equality: compared at runtime per RTTI (numbers across int/float, String by content, struct/List by reference)
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
	// struct: __eq__/__ne__ method or reference comparison
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
	// pointer / null: reference comparison (interpreter Value semantics: the same pointer is equal)
	if isPtrLike(lt) || isPtrLike(rt) {
		op := cmpOps[x.op]
		if x.op != "==" && x.op != "!=" {
			op = "eq" // pointers do not support ordered comparison (typecheck already rejects it)
		}
		r := e.newReg()
		e.emitInstr("%s = icmp %s i8* %s, %s", r, op, lv, rv)
		return r, "bool"
	}
	// long in comparisons: the interpreter compares after truncating to 32 bits
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

// isPtrLike reports whether the type is an FFI opaque pointer / null (i8* reference comparison).
func isPtrLike(t string) bool {
	if t == "pointer" || t == "null" {
		return true
	}
	_, _, ok := ptrRefBase(t)
	return ok
}

var cmpOps = map[string]string{"==": "eq", "!=": "ne", "<": "slt", "<=": "sle", ">": "sgt", ">=": "sge"}

// ---------- Function attribute analysis (safe determination of LLVM norecurse/mustprogress) ----------

// fnMeta collects a single function's call relations and loop flag.
type fnMeta struct {
	callees []string
	hasLoop bool
	hasMeth bool // method call/function reference (cannot be resolved statically, conservatively no norecurse)
	impure  bool // calls/methods/lists/structs/indexing may have external side effects (memory(none) forbidden)
}

func analyzeExpr(e *expr, m *fnMeta) {
	if e == nil {
		return
	}
	if e.anyBox != "" {
		m.impure = true // boxing calls malloc (scalar heap cell), not a pure function
	}
	switch e.kind {
	case kTable:
		m.impure = true // ql_table_*: heap table reads/writes
		if e.tbl != nil {
			analyzeExpr(e.tbl.recv, m)
			for _, a := range e.tbl.args {
				analyzeExpr(a, m)
			}
		}
	case kMerge:
		m.impure = true // ql_merge: starts a thread
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
		m.impure = true // ql_int_to_str: malloc/snprintf calls
		analyzeExpr(e.l, m)
	case kBin:
		if e.strcat {
			m.impure = true // ql_strcat call: has side effects, memory(none) forbidden
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
	case kListS:
		m.impure = true
		for _, it := range e.lst.items {
			analyzeExpr(it, m)
		}
	case kConv:
		analyzeExpr(e.l, m)
	case kStructLit:
		m.impure = true
		for _, v := range e.sl.values {
			analyzeExpr(v, m)
		}
	case kField:
		m.impure = true // reads a struct field through a pointer: not a pure function (memory(none) forbidden)
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
		m.impure = true // printf: IO side effect, memory(none) forbidden
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
		m.impure = true // heap array write
		analyzeExpr(st.idx, m)
		analyzeExpr(st.x, m)
	case *bitAssignStmt:
		m.impure = true // load/modify/store of the raw-bits lvalue
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

// programMeta builds the function metadata table (including main).
func programMeta(funcs []*funcDef, mainStmts []stmt) map[string]*fnMeta {
	meta := make(map[string]*fnMeta, len(funcs)+1)
	for _, fd := range funcs {
		m := &fnMeta{}
		for _, s := range fd.body {
			analyzeStmt(s, m)
		}
		// Parameters are passed by reference: a parameter is a pointer to the caller's storage, and the callee's loads/stores are
		// visible to the caller → a function with parameters cannot be marked memory(none) (otherwise LLVM would eliminate parameter writes).
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

// reachesSelf reports whether the call graph forms a cycle containing name (DFS, only across already-defined functions).
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

// fnAttrs returns the attribute string for a function definition (mustprogress/norecurse/memory(none)).
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
		// pure arithmetic function (no calls, no IO, no heap access): only touches locals and parameters — LLVM may eliminate/inline across calls
		attrs += " memory(none)"
	}
	return attrs
}

// scanAssigned collects the names of all variables assigned (assignStmt) in the statements — it decides which decls can use SSA passthrough.
func scanAssigned(stmts []stmt, out map[string]bool) {
	for _, s := range stmts {
		switch st := s.(type) {
		case *assignStmt:
			out[st.name] = true
		case *bitAssignStmt:
			// b[i] = v writes through the receiver's storage, so the receiver needs an address
			if st.recv != nil && st.recv.kind == kIdent {
				out[st.recv.s] = true
			}
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
			out[st.name] = true // the loop variable is written on every iteration
			scanAssigned(st.body, out)
		case *tryStmt:
			scanAssigned(st.then, out)
			scanAssigned(st.catch, out)
		}
	}
}

// scanByRefArgs collects "variable names that appear as by-reference arguments": these variables must have alloca storage,
// otherwise compileArgAddr can only degrade to a temporary cell and the callee's writes cannot be written back to the caller.
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
				// user method: arguments by reference; interface method (vtable dispatch): likewise by reference (interpreter evalArgs)
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

// constEval recursively evaluates a constant expression (literal arithmetic folding).
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

// ---------- Runtime helpers (appended to the end of the module on demand) ----------

// intToStrHelper is the runtime helper for int.toString(): malloc(12) + snprintf("%d");
// 12 bytes is enough for the int32 extremes (-2147483648 + NUL).
const intToStrHelper = `@.ql.fmt.d = private unnamed_addr constant [3 x i8] c"%d\00", align 1
define i8* @ql_int_to_str(i32 %v) {
entry:
  %b = call i8* @malloc(i64 12)
  %f = getelementptr inbounds [3 x i8], [3 x i8]* @.ql.fmt.d, i64 0, i64 0
  %n = call i32 (i8*, i64, i8*, ...) @snprintf(i8* %b, i64 12, i8* %f, i32 %v)
  ret i8* %b
}
`

// floatToStrHelper is the runtime helper for float.toString()/printing: consistent with the interpreter's
// strconv.FormatFloat(v,'f',-1,64) — take the shortest fixed-point decimal that round-trips.
// implementation: snprintf %.0f..%.17f one by one, verify by reading back with strtod, and take the first precision that restores the value exactly.
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

// ptrToStrHelper is the pointer printing helper: null → "nil", otherwise "0x%llx"
// (consistent with the interpreter's Value.String() 0x+hex rendering).
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

// panicBitsHelper reports an out-of-range bit index b[i] with the interpreter's exact message.
const panicBitsHelper = `@.ql.panic.bits = private unnamed_addr constant [91 x i8] c"error: IndexOutOfRangeError: bit index %d is out of range for bits<%d> (0..%d) at line %d\0A\00", align 1
define void @ql_panic_bits(i32 %idx, i32 %width, i32 %line) {
entry:
  %buf = alloca [512 x i8], align 16
  %p = getelementptr inbounds [512 x i8], [512 x i8]* %buf, i64 0, i64 0
  %fmt = getelementptr inbounds [91 x i8], [91 x i8]* @.ql.panic.bits, i64 0, i64 0
  %last = sub i32 %width, 1
  %n = call i32 (i8*, i64, i8*, ...) @snprintf(i8* %p, i64 512, i8* %fmt, i32 %idx, i32 %width, i32 %last, i32 %line)
  %n64 = sext i32 %n to i64
  %w = call i64 @write(i32 2, i8* %p, i64 %n64)
  call void @exit(i32 1)
  unreachable
}
`

// panicHelper prints and exits on a runtime error (consistent with the interpreter's ReportError "error: ...").
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
