// Lowering layer of package cgen: canonical AST (internal/lang) → cgen IR.
//
// This file is the compiler's only frontend entry: cgen.go keeps just the LLVM IR emitter and the IR data structures,
// while lexing/parsing/typecheck all come from internal/lang (a single syntax source, shared with the interpreter).
//
// Semantic baseline: the **interpreter**. The same source must produce byte-identical output under
// /tmp/quark (interpreter) and qkc -run (compiler); every lowering rule follows the Typecheck and
// eval semantics of internal/lang (no rules invented here).
//
// Value semantics (aligned with the interpreter):
//   - List / struct are **references** (in the interpreter they are *List / *Struct): assignment and argument passing copy the reference,
//     so field/element writes are visible through every alias (q = p; q.a = 1 also changes p.a).
//   - int → float is the only implicit conversion (consistent with typecheck.assignable).
//
// Constructs already lowered (phases A/B/C):
//
//   - statements: declaration (int/bool/float/String/List<int>/struct/interface), assignment (variable/List index/
//     struct field), if/else, while, for (C style + iteration for-in), break, try/catch(void),
//     return, log, delete List, io.println/io.print, expression statement
//
//   - expressions: int/float/bool/String literals, arithmetic/comparison/short-circuit logic, String concatenation,
//     int/float/bool.toString(), function call/recursion, List size/get/append/[i],
//     struct literal (named + positional), field read/write, instance method/static method, space call,
//     Operation operator overload (__add__ etc.), generic function/generic struct/generic impl monomorphization,
//     interface boxing and vtable dynamic dispatch, built-in sum/clock
//
//   - top level: fn, main(IOStream io), import (lang.CompileWithImports recursive merge),
//     type struct, impl, space
//
//   - top level: library (FFI: LLVM declare + direct C ABI call + ; qkc-link marker),
//     taskm (spawn/merge/block/done/channel, hooked up to the qthreads runtime)
//
// Parameter semantics (canonical): parameters are always passed by reference (LLVM parameter = value type*), copyd parameters are deep-copied at entry;
// lvalue arguments use their own storage, non-lvalue arguments become caller temporary cells; the self receiver is bound by value.
//
// interface{} (tAny): box any scalar/String/List<int>/struct + runtime type descriptor (RTTI),
// println dispatches by kind, ==/!= go through ql_any_eq, and unboxing to a scalar/String checks the kind at runtime.
//
// Still not lowered (always returns an explicit error with location, never a silent miscompile):
// pointer types/new/Copyd<T> type annotations, HashTable, custom Sign instances other than memorize,
// copyd interface parameters/interface fields, List<T≠int>, interface{} unboxing to struct/List,
// printing a multi-field struct (the interpreter's field order comes from a Go map and is not deterministic), using the return value of a function containing log,
// merge with more than 4 arguments.
package cgen

import (
	"fmt"
	"strings"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// ---------- Diagnostics with location ----------

// diagError is a compile-time diagnostic carrying a source location and source line.
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

// expandTabs expands tabs in a source line to 4 spaces and converts the column number to the expanded position.
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

// exprPos returns an expression's position; falls back to fallback when missing.
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

// ---------- Declaration tables ----------

// fnItem is an ordinary function waiting to be lowered (declaration + IR name).
type fnItem struct {
	decl *lang.FuncDecl
	ir   string
}

// retOf returns the declared return type (void by default).
func retOf(f *lang.FuncDecl) string {
	if f.Ret == "" {
		return "void"
	}
	return f.Ret
}

// mangleOverload generates the IR name of an overload: add$int$String (argument types joined with LLVM-safe names).
func mangleOverload(name string, params []lang.Param) string {
	if len(params) == 0 {
		return name + "$void"
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		parts = append(parts, tyName(strings.TrimSpace(p.Type)))
	}
	return name + "$" + strings.Join(parts, "$")
}

// kindOf classifies a language type into a "type kind", used for overload scoring
// (the same granularity as internal/lang bestMatchT's Kind comparison: structs share a kind, interfaces share a kind).
func (l *lowerer) kindOf(t string) string {
	switch t {
	case "int", "long":
		return "int"
	case "float":
		return "float"
	case "bool":
		return "bool"
	case "String":
		return "string"
	case "void":
		return "void"
	case "null":
		return "null"
	case "pointer":
		return "ptr"
	case "?":
		return "?"
	}
	if strings.HasPrefix(t, "List<") {
		return "list"
	}
	if strings.HasPrefix(t, "HashTable<") {
		return "table"
	}
	if l.isStructType(t) {
		return "struct"
	}
	if l.isIfaceType(t) {
		return "interface"
	}
	return "?"
}

// methodInfo is one method in an impl/space.
type methodInfo struct {
	fn      *lang.FuncDecl
	impl    *lang.ImplDecl
	selfTyp string            // receiver type ("Self" already replaced by the concrete type)
	isSelf  bool              // has a self first parameter (instance method)
	irName  string            // IR function name (Point_sum / math_max)
	subst   map[string]string // impl type parameter → concrete type
}

// lowerer holds the symbol information and diagnostic context of the whole program.
type lowerer struct {
	prog      *lang.Program
	file      string
	lines     []string // prog.Src (source after import merging) split into lines
	mainLines int      // main file line count (src lines before import merging)
	imported  bool

	fns      map[string]*lang.FuncDecl // ordinary non-generic, non-main functions (unique names)
	fnOrder  []string
	fnRet    map[string]string           // IR function name → return type
	ovl      map[string][]*lang.FuncDecl // overloaded functions: lang name → candidate declarations
	fnIR     map[*lang.FuncDecl]string   // declaration → IR name (overloads mangled by argument types)
	fnList   []fnItem                    // ordinary functions waiting to be lowered (declaration + IR name)
	generics map[string]*lang.FuncDecl
	structs  map[string]*lang.StructDecl // named structs (including generic templates)
	ifaces   map[string]*lang.InterfaceDecl
	libs     map[string]*lang.LibraryDecl
	methods  map[string]map[string]*methodInfo // type/space name → method name → info
	spaces   map[string]bool                   // space names (impl.Type with no matching struct)

	tys       []structDef // struct types that must be emitted (including generic instances)
	tyEmit    map[string]bool
	insts     map[string]string // generic instance → already-instantiated marker
	instMi    map[string]*methodInfo
	ifaceTbl  map[string][]ifaceM
	vtables   map[string]*vtableDef
	extSeen   map[string]bool
	runnerIdx map[string]string
	sigs      map[string]*funcDef
	nilFns    map[string]bool // functions containing log (their return value may be nil, interpreter-only)
	lowering  map[string]bool // functions currently being lowered (guards against recursive-instantiation loops)

	out *lowered
}

// lowerProgram lowers the canonical AST to cgen IR. Any construct the backend has not lowered returns
// an explicit error with location — never silently wrong code.
// src is the main file's original source (used to decide diagnostic line ranges).
func lowerProgram(prog *lang.Program, file, src string) (*lowered, error) {
	l := &lowerer{
		prog:      prog,
		file:      file,
		lines:     strings.Split(prog.Src, "\n"),
		mainLines: len(strings.Split(src, "\n")),
		imported:  len(prog.Imports) > 0,
		fns:       map[string]*lang.FuncDecl{},
		fnRet:     map[string]string{},
		ovl:       map[string][]*lang.FuncDecl{},
		fnIR:      map[*lang.FuncDecl]string{},
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

// errf builds a diagnostic with location.
func (l *lowerer) errf(pos lang.Pos, format string, args ...interface{}) error {
	d := &diagError{msg: msg(format, args...), file: l.file, line: pos.Line, col: pos.Col}
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

// errAt builds a diagnostic at a fixed hint position (when no AST position is available, e.g. a missing main).
func (l *lowerer) errAt(format string, args ...interface{}) error {
	return l.errf(lang.Pos{Line: 1, Col: 1}, format, args...)
}

// ---------- Top-level collection ----------

func (l *lowerer) collect() error {
	if l.prog.Kind == "library" {
		return l.errAt("compiling a program library is not supported (library form is handled by the interpreter/export flow)")
	}
	for _, sd := range l.prog.Structs {
		if sd.Name == "" {
			continue // unnamed top-level declaration: the interpreter registers it as an unreferenceable dead type, the compiler just ignores it
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

	// Function name collection: a unique name uses the original name; overloads are mangled by "argument types"
	// (add$int$int / add$String$String), and call sites are resolved with bestMatch semantics.
	groups := map[string][]*lang.FuncDecl{}
	var order []string
	var mainFn *lang.FuncDecl
	for _, f := range l.prog.Funcs {
		if f.Name == "sum" || f.Name == "clock" {
			return l.errf(f.Pos, "redefining builtin function %q is not supported (the compiler treats %s as builtin)", f.Name, f.Name)
		}
		if len(f.TypeParams) > 0 {
			if _, dup := l.generics[f.Name]; dup {
				return l.errf(f.Pos, "overloading a generic function %q is not supported (the compiler requires unique generic function names)", f.Name)
			}
			l.generics[f.Name] = f
			continue
		}
		if f.Name == "main" {
			mainFn = f
			continue
		}
		if _, seen := groups[f.Name]; !seen {
			order = append(order, f.Name)
		}
		groups[f.Name] = append(groups[f.Name], f)
	}
	for _, name := range order {
		defs := groups[name]
		if _, isGen := l.generics[name]; isGen {
			return l.errf(defs[0].Pos, "a generic and a plain function sharing the name %q is not supported (the compiler requires one or the other)", name)
		}
		if len(defs) == 1 {
			f := defs[0]
			l.fns[name] = f
			l.fnIR[f] = name
			l.fnRet[name] = retOf(f)
			l.fnList = append(l.fnList, fnItem{decl: f, ir: name})
			continue
		}
		sigs := map[string]bool{}
		for _, f := range defs {
			ir := mangleOverload(name, f.Params)
			if sigs[ir] {
				return l.errf(f.Pos, "duplicate function definition %s (same parameter type signature)", ir)
			}
			sigs[ir] = true
			l.ovl[name] = append(l.ovl[name], f)
			l.fnIR[f] = ir
			l.fnRet[ir] = retOf(f)
			l.fnList = append(l.fnList, fnItem{decl: f, ir: ir})
		}
	}
	if mainFn == nil && !libMode {
		return l.errAt("未找到 main 函数（正典入口：fn main(IOStream io) { ... }）")
	}

	// Lower ordinary functions first (the signature/statement/expression support checks all happen here)
	for _, item := range l.fnList {
		fd, err := l.lowerFunc(item.decl, item.ir, "", "", nil)
		if err != nil {
			return err
		}
		l.out.funcs = append(l.out.funcs, fd)
	}
	// Then lower impl/space methods (non-generic impls; generic impls are monomorphized at the call site)
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
	if mainFn != nil { // no main in lib mode: skip entry-point lowering
		stmts, err := l.lowerMain(mainFn)
		if err != nil {
			return err
		}
		l.out.mainStmts = stmts
	}
	return nil
}

// registerImpl registers the method table of an impl/space (including duplicate-method checks).
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
			return l.errf(m.Pos, "overloading/redefining method %s.%s is not supported (the compiler emits name-mangled functions and cannot distinguish overloads)", typ, m.Name)
		}
		mi := &methodInfo{fn: m, impl: im, irName: typ + "_" + m.Name, subst: map[string]string{}}
		for i, tp := range im.TypeParams {
			_ = i
			mi.subst[tp] = tp
		}
		if len(m.Params) > 0 && isRecvParam(typ, &m.Params[0]) {
			mi.isSelf = true
			mi.selfTyp = typ
		}
		l.methods[typ][m.Name] = mi
	}
	return nil
}

// isRecvParam decides whether the first parameter of a method/interface method is the **receiver**: by type (Self or the impl/interface
// type's base name), independently of the parameter name; when the first parameter has no type annotation it is treated as the receiver and the type is filled in.
// The rule is exactly the same as internal/lang's isRecvParam (the interpreter is the semantic baseline).
func isRecvParam(recvType string, p *lang.Param) bool {
	if p == nil {
		return false
	}
	t := strings.TrimSpace(p.Type)
	if t == "" {
		p.Type = recvType
		return true
	}
	return t == "Self" || recvBaseName(t) == recvBaseName(recvType)
}

// recvBaseName takes the base name of a type: strips a trailing & and generic arguments (node<T>& → node).
func recvBaseName(t string) string {
	t = strings.TrimSpace(t)
	t = strings.TrimSuffix(t, "&")
	if i := strings.IndexByte(t, '<'); i >= 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

// ---------- Type utilities ----------

// splitGeneric splits "Box<int>" → ("Box", ["int"]); a non-generic type returns (t, nil).
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

// substType expands a type string with a type-parameter substitution table (T → int, Box<T> → Box<int>).
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

// structSubst returns the type-parameter substitution table of a struct instance (Point → {}; Box<int> → {T:int}).
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

// fieldType returns a struct field type (including generic substitution); "" if it does not exist.
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

// ensureStructTy registers a struct type definition that must be emitted (including generic instances).
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
	// Dependencies are emitted first: an LLVM named type must be defined before it is used
	for _, ft := range def.fieldTypes {
		if l.isStructType(ft) {
			l.ensureStructTy(ft)
		}
	}
	l.tys = append(l.tys, def)
}

// listElem resolves the element type of List<T>.
func listElem(t string) (string, bool) {
	base, args := splitGeneric(t)
	if base != "List" || len(args) != 1 {
		return "", false
	}
	return args[0], true
}

// isStructType reports whether a type is a struct declared by this program (including generic instances).
func (l *lowerer) isStructType(t string) bool {
	base, _ := splitGeneric(t)
	_, ok := l.structs[base]
	return ok
}

// isIfaceType reports whether a type is an interface (including the empty interface interface{}).
func (l *lowerer) isIfaceType(t string) bool {
	if isAnyT(t) {
		return true
	}
	_, ok := l.ifaces[t]
	return ok
}

// isPtrRefT reports whether a type is T& / pointer T (a nullable reference).
func isPtrRefT(t string) bool {
	_, _, ok := ptrRefBase(t)
	return ok
}

// isAnyT reports whether the type is the empty interface interface{} (boxing + RTTI path).
func isAnyT(t string) bool { return strings.TrimSpace(t) == "interface{}" }

// checkType validates that a type can be used (a non-lowerable type gets an explicit diagnostic).
func (l *lowerer) checkType(t string, pos lang.Pos, what string) error {
	t = strings.TrimSpace(t)
	switch t {
	case "int", "bool", "float", "String", "List<int>", "uchar":
		return nil
	case "void":
		return l.errf(pos, "暂未支持 void 类型 %s", what)
	}
	if isBitsT(t) {
		return nil // raw bits: bit / bits<N> for N <= 32 (a wider N is refused by the shared frontend)
	}
	if elem, ok := listElem(t); ok {
		if elem == "String" {
			return nil // List<String>: read-only subset (keys() results / size/get/toString/for-in/indexing)
		}
		return l.errf(pos, "%s type %q is not supported (the compiler handles List<int> and List<String>; element %s)", what, t, elem)
	}
	if l.isStructType(t) {
		return nil
	}
	if l.isIfaceType(t) {
		return nil // interface type: vtable dispatch (Phase C)
	}
	switch t {
	case "long":
		return nil // FFI 64-bit channel (i64); usable for language-level variables/parameters/returns
	case "pointer":
		return nil // FFI opaque handle (i8*, nullable)
	case "char":
		return l.errf(pos, "%s type %q is not supported (not lowered by the compiler yet)", what, t)
	case "thread", "Task", "Channel", "channel":
		return nil // taskm: thread = pid(i32), Channel = i8* handle
	case "memorize":
		return nil // built-in memorize signature instance (i8* handle; @mb() memoization)
	case "IOStream":
		return l.errf(pos, "IOStream %s is not supported (the compiler binds io only at the main entry)", what)
	case "interface{}":
		return nil // boxes any value + runtime type descriptor (RTTI); printing/equality/scalar unboxing already lowered
	}
	if strings.HasPrefix(t, "List") {
		return l.errf(pos, "%s type %q is not supported (the compiler handles List<int> only)", what, t)
	}
	if isTableT(t) {
		return nil // HashTable<K,V>: i8* handle + runtime structured keys (the interpreter's key rule)
	}
	if strings.HasPrefix(t, "HashTable") {
		return l.errf(pos, "暂未支持 %s 类型 %q（HashTable 写法：HashTable<K, V>）", what, t)
	}
	if _, base, ok := ptrRefBase(t); ok {
		// T& / pointer T: nullable reference (zero value null); only a single-element scalar/String/struct
		if err := l.checkType(base, pos, what); err != nil {
			return err
		}
		switch {
		case base == "int", base == "float", base == "bool", base == "String", base == "long":
			return nil
		}
		if l.isStructType(base) {
			return nil
		}
		return l.errf(pos, "%s type %q is not supported (the compiler lowers T& for scalars/String/struct only)", what, t)
	}
	if strings.Contains(t, "[Copyd]") || strings.HasPrefix(t, "Copyd<") || strings.HasSuffix(t, "[]") {
		return l.errf(pos, "pointer/copy-on-pass type %q is not supported (copyd as a parameter modifier is supported; there is no Copyd<T> type)", t)
	}
	return l.errf(pos, "未知类型 %q（%s 声明）", t, what)
}

// retOK reports whether a return type can be lowered.
func (l *lowerer) retOK(t string) bool {
	if isBitsT(t) || t == "uchar" {
		return true
	}
	switch t {
	case "int", "bool", "float", "String", "long", "pointer", "void", "interface{}":
		return true
	}
	if _, _, isRef := ptrRefBase(t); isRef {
		return true
	}
	return l.isStructType(t) || l.isIfaceType(t) || isTableT(t)
}

// ---------- Scopes ----------

// scope is a block scope (used for variable type lookup and duplicate/shadowing checks).
type scope struct {
	vars   map[string]string
	parent *scope
}

// funcCtx is the context of lowering a single function.
type funcCtx struct {
	l      *lowerer
	fn     string // IR function name
	ret    string
	ioName string // main's IOStream parameter name
	top    *scope
	logged bool // log appears in the body (the return value may be nil)

	// discarded is the current "value is discarded" call (a top-level call in an expression statement):
	// the return value of a function containing log may be nil, so it may only be called in this position.
	discarded *lang.CallExpr

	// expectT is the expected type of the current expression (used to fill in the type arguments of a generic struct literal:
	// typecheck only fills the name of .{...} with the base name "Box", leaving the arguments to the declared type).
	expectT string

	// subst is the type-parameter substitution table of the current generic instance (T → int).
	subst map[string]string
}

// resolveT expands the type parameters of the current instance (T → int; Box<T> → Box<int>).
func (fc *funcCtx) resolveT(t string) string {
	if len(fc.subst) == 0 {
		return t
	}
	return substType(t, fc.subst)
}

// typeOfAs infers the type of an expression in the context of the expected type want.
func (fc *funcCtx) typeOfAs(x lang.Expr, want string) string {
	old := fc.expectT
	fc.expectT = want
	t := fc.typeOf(x)
	fc.expectT = old
	return t
}

// exprAs lowers an expression in the context of the expected type want (including interface boxing).
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

// lookup finds a variable's type (walking the scope chain).
func (fc *funcCtx) lookup(name string) (string, bool) {
	for s := fc.top; s != nil; s = s.parent {
		if t, ok := s.vars[name]; ok {
			return t, true
		}
	}
	return "", false
}

// declare registers a variable; a duplicate name or shadowing of an outer variable is an explicit error (the emitter's variable table is flat,
// so shadowing would be a silent miscompile and must be rejected here).
func (fc *funcCtx) declare(name, typ string, pos lang.Pos) error {
	if _, ok := fc.lookup(name); ok {
		return fc.l.errf(pos, "duplicate/shadowed variable %q is not supported (the compiler's variable table has no same-name variables; rename it)", name)
	}
	fc.top.vars[name] = typ
	return nil
}

// ---------- Function lowering ----------

// lowerFunc lowers an ordinary function/method. subst is the impl generic-parameter substitution table.
func (l *lowerer) lowerFunc(f *lang.FuncDecl, irName, selfTyp, selfParam string, subst map[string]string) (*funcDef, error) {
	ret := f.Ret
	if ret == "" {
		ret = "void"
	}
	ret = substType(ret, subst)
	if canon, _, ok := ptrRefBase(ret); ok {
		ret = canon // pointer T → T&
	}
	if !l.retOK(ret) {
		return nil, l.errf(f.Pos, "return type %q is not supported (the compiler handles int/bool/float/String/void/struct)", f.Ret)
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
		if canon, _, ok := ptrRefBase(pt); ok {
			pt = canon // pointer T → T&
		}
		if err := l.checkType(pt, p.Pos, "参数"); err != nil {
			return nil, err
		}
		if err := fc.declare(p.Name, pt, p.Pos); err != nil {
			return nil, err
		}
		cp := paramCopyd(p)
		if cp {
			if err := l.checkCopydType(pt, p.Pos, 0); err != nil {
				return nil, err
			}
		}
		params = append(params, funcParam{name: p.Name, typ: pt, copyd: cp})
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

// paramCopyd reports whether a parameter is copyd (two canonical spellings: the copyd modifier, the type Copyd<T>/T[Copyd];
// the same rule as internal/lang's Func.CopydFlags).
func paramCopyd(p lang.Param) bool {
	return p.Decor == "copyd" || strings.Contains(p.Type, "Copyd")
}

// checkCopydType validates that a copyd parameter's type can be deep-copied (consistent with the interpreter's copydCopy semantics):
// scalar/String/List<int> are copyable; a struct has its fields validated recursively; interface fields/interface types have an unknown runtime type,
// so the backend cannot deep-copy them → explicit error (never a silent shallow-copy miscompile).
func (l *lowerer) checkCopydType(t string, pos lang.Pos, depth int) error {
	if depth > 16 {
		return nil
	}
	if l.isIfaceType(t) {
		return l.errf(pos, "暂未支持 copyd 接口形参 %s（运行期类型未知，无法深拷贝）", t)
	}
	if isTableT(t) {
		return l.errf(pos, "暂未支持 copyd HashTable 形参 %s（表值深拷贝未 lower）", t)
	}
	if t == "List<String>" {
		return l.errf(pos, "暂未支持 copyd List<String> 形参（字符串列表深拷贝未 lower）")
	}
	sd, sub := l.structSubst(t)
	if sd == nil {
		return nil // scalar / String / List<int> / unknown types are handled by checkType
	}
	for _, m := range sd.Members {
		ft := substType(m.Type, sub)
		if l.isIfaceType(ft) {
			return l.errf(pos, "暂未支持 copyd struct %s 的接口字段 %s（运行期类型未知，无法深拷贝）", t, m.Name)
		}
		if nsd, _ := l.structSubst(ft); nsd != nil {
			if err := l.checkCopydType(ft, pos, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// lowerMain lowers the program entry fn main(IOStream io).
func (l *lowerer) lowerMain(f *lang.FuncDecl) ([]stmt, error) {
	if len(f.TypeParams) > 0 {
		return nil, l.errf(f.Pos, "暂未支持泛型 main 函数")
	}
	if f.Ret != "" && f.Ret != "void" {
		return nil, l.errf(f.Pos, "main cannot return a value (got %q)", f.Ret)
	}
	if len(f.Params) == 0 || f.Params[0].Type != "IOStream" {
		return nil, l.errf(f.Pos, "main's first parameter must be IOStream (canonical: fn main(IOStream io) { ... })")
	}
	if len(f.Params) > 1 {
		return nil, l.errf(f.Params[1].Pos, "main's %q parameter is not supported (the compiler binds IOStream io only)", f.Params[1].Name)
	}
	fc := l.newCtx("main", "void")
	fc.ioName = f.Params[0].Name
	if err := fc.declare(fc.ioName, "IOStream", f.Params[0].Pos); err != nil {
		return nil, err
	}
	return fc.block(f.Body)
}

// ---------- Statement lowering ----------

func (fc *funcCtx) block(b *lang.Block) ([]stmt, error) {
	fc.push()
	defer fc.pop()
	out := make([]stmt, 0, len(b.Stmts))
	for _, s := range b.Stmts {
		st, err := fc.stmt(s)
		if err != nil {
			return nil, err
		}
		if st != nil { // statements that are only registered and produce no code
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
			return nil, l.errf(st.Pos, "the copyd modifier is not supported (copy-on-pass is interpreter-only)")
		}
		return fc.declStmt(st)

	case *lang.AssignStmt:
		return fc.assignStmt(st)

	case *lang.ExprStmt:
		// io.println(...) / io.print(...) statement
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
						return nil, l.errf(call.Pos, "IOStream method %q is not supported (the compiler handles io.println/io.print only)", me.Name)
					}
				}
			}
		}
		// top-level call in an expression statement: the value is discarded, and functions containing log may also be called
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
			return nil, l.errf(st.Pos, "a void function cannot return a value (the interpreter ignores the value; the compiler refuses to ignore it silently)")
		}
		t := fc.typeOfAs(st.X, fc.ret)
		if !fc.assignable(t, fc.ret) {
			return nil, l.errf(exprPos(st.X, st.Pos), "暂未支持从 %s 函数返回 %s 值", fc.ret, t)
		}
		x, err := fc.exprAs(st.X, fc.ret)
		if err != nil {
			return nil, err
		}
		if isPtrRefT(fc.ret) {
			x.noDeref = true // return the T& itself (the pointer value) without dereferencing
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
			return nil, l.errf(st.Pos, "catch (%s %s) is not supported (the compiler handles catch (void e) only)", st.CatchVarType, st.CatchVar)
		}
		if st.CatchVar != "" && blockUsesIdent(st.Catch, st.CatchVar) {
			return nil, l.errf(st.Pos, "using %q inside a catch body is not supported (error value passing is interpreter-only)", st.CatchVar)
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
			return nil, l.errf(st.Pos, "delete on a non-variable expression is not supported (the compiler handles delete <List variable> only)")
		}
		if t, _ := fc.lookup(id.Name); t != "List<int>" {
			return nil, l.errf(st.Pos, "delete %q is not supported (the compiler handles delete on a List variable only)", id.Name)
		}
		return &deleteStmt{name: id.Name}, nil
	}
	return nil, l.errf(lang.Pos{Line: 1, Col: 1}, "compiler 内部错误：未知语句类型 %T", s)
}

// forIn lowers the iteration for: for (<T> x : list).
func (fc *funcCtx) forIn(st *lang.ForStmt) (stmt, error) {
	l := fc.l
	lt := fc.typeOf(st.Iter)
	base, args := splitGeneric(lt)
	if base != "List" || len(args) != 1 {
		return nil, l.errf(st.Pos, "for-iteration over %s is not supported (the compiler handles List<int>)", lt)
	}
	elem := args[0]
	if elem != "int" && elem != "String" {
		return nil, l.errf(st.Pos, "iterating List<%s> is not supported (the compiler handles List<int> and List<String>)", elem)
	}
	id, ok := st.Iter.(*lang.Ident)
	if !ok {
		return nil, l.errf(st.Pos, "for-iteration over a non-variable list is not supported (the compiler requires a List variable)")
	}
	vt, _ := fc.lookup(id.Name)
	if vt != "List<int>" && vt != "List<String>" {
		return nil, l.errf(st.Pos, "for-iteration over %s is not supported (the compiler handles List<int> and List<String>)", vt)
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

// ---------- Variable declarations ----------

func (fc *funcCtx) declStmt(st *lang.DeclStmt) (stmt, error) {
	if st.Perm != "" {
		// Permission-carrying references (&rw u int p) are interpreter-only for now.
		return nil, fc.l.errf(st.Pos, "compiling a permission reference &%s %s is not supported yet (implemented in the interpreter only)", st.Perm, st.Scope)
	}
	l := fc.l
	t := fc.resolveT(st.Type)
	if err := l.checkType(t, st.Pos, "variable"); err != nil {
		return nil, err
	}
	switch t {
	case "int", "bool", "float", "long", "String", "pointer", "thread", "Channel", "channel", "memorize":
		if st.Init == nil {
			if err := fc.declare(st.Name, t, st.Pos); err != nil {
				return nil, err
			}
			return &declStmt{name: st.Name, typ: t}, nil
		}
		it := fc.typeOf(st.Init)
		if !fc.assignable(it, t) && !(isBitsT(t) && isIntConstLit(st.Init)) {
			return nil, l.errf(exprPos(st.Init, st.Pos), "using %s to initialize %s variable %q is not supported", it, t, st.Name)
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
	if isBitsT(t) || t == "uchar" {
		// Raw bits and the uchar view are scalar slots: the value is re-read at the declared width
		// on the way in (declarations and assignments mask, spec §3.1).
		if st.Init == nil {
			if err := fc.declare(st.Name, t, st.Pos); err != nil {
				return nil, err
			}
			return &declStmt{name: st.Name, typ: t}, nil
		}
		it := fc.typeOf(st.Init)
		if !fc.assignable(it, t) && !(isBitsT(t) && isIntConstLit(st.Init)) {
			return nil, l.errf(exprPos(st.Init, st.Pos), "using %s to initialize %s variable %q is not supported (raw bits take raw bits of the same width, or a constant)", it, t, st.Name)
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

	switch t {
	case "List<String>":
		if err := fc.declare(st.Name, t, st.Pos); err != nil {
			return nil, err
		}
		if st.Init == nil {
			return nil, l.errf(st.Pos, "List<String> %q without an initializer is not supported (the compiler only lowers existing lists such as keys())", st.Name)
		}
		if _, isLit := st.Init.(*lang.ListLit); isLit {
			// ["a", "b", ...]: the dedicated List<String> literal path (elements are i8* pointers)
			x, err := fc.exprAs(st.Init, t)
			if err != nil {
				return nil, err
			}
			return &declStmt{name: st.Name, typ: t, init: x}, nil
		}
		it := fc.typeOfAs(st.Init, t)
		if it != "List<String>" {
			return nil, l.errf(exprPos(st.Init, st.Pos), "initializing List<String> with %s is not supported (the compiler lowers literals, HashTable.keys() results and List<String> variables)", it)
		}
		x, err := fc.exprAs(st.Init, t)
		if err != nil {
			return nil, err
		}
		return &declStmt{name: st.Name, typ: t, init: x}, nil

	case "List<int>":
		if st.Init == nil {
			return nil, l.errf(st.Pos, "List<int> %q without an initializer is not supported (the interpreter's zero value is an empty list; the compiler backend does not lower it)", st.Name)
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
				return nil, l.errf(exprPos(it, st.Pos), "initializing List<int> with %s elements is not supported", et)
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

	// T& / pointer T: nullable reference (new T allocates a cell; no initial value = null); normalized to T&
	if canon, _, isRef := ptrRefBase(t); isRef {
		t = canon
		if err := fc.declare(st.Name, t, st.Pos); err != nil {
			return nil, err
		}
		if st.Init == nil {
			return &declStmt{name: st.Name, typ: t}, nil
		}
		it := fc.typeOfAs(st.Init, t)
		if !fc.assignable(it, t) {
			return nil, l.errf(exprPos(st.Init, st.Pos), "using %s to initialize %s variable %q is not supported", it, t, st.Name)
		}
		x, err := fc.expr(st.Init)
		if err != nil {
			return nil, err
		}
		return &declStmt{name: st.Name, typ: t, init: x}, nil
	}

	// HashTable<K,V>: an i8* handle (HashTable::new() or another table)
	// (the T& branch was already handled above under the canonical name t)
	if isTableT(t) {
		if err := fc.declare(st.Name, t, st.Pos); err != nil {
			return nil, err
		}
		if st.Init == nil {
			return &declStmt{name: st.Name, typ: t}, nil
		}
		it := fc.typeOfAs(st.Init, t)
		if !fc.assignable(it, t) {
			return nil, l.errf(exprPos(st.Init, st.Pos), "using %s to initialize %s variable %q is not supported", it, t, st.Name)
		}
		x, err := fc.exprAs(st.Init, t)
		if err != nil {
			return nil, err
		}
		return &declStmt{name: st.Name, typ: t, init: x}, nil
	}

	// interface type (vtable boxing)
	if l.isIfaceType(t) {
		if st.Init == nil {
			if err := fc.declare(st.Name, t, st.Pos); err != nil {
				return nil, err
			}
			return &declStmt{name: st.Name, typ: t}, nil
		}
		it := fc.typeOfAs(st.Init, t)
		if !fc.assignable(it, t) {
			return nil, l.errf(exprPos(st.Init, st.Pos), "using %s to initialize interface %s variable %q is not supported", it, t, st.Name)
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
	// struct type
	l.ensureStructTy(t)
	if st.Init == nil {
		if err := fc.declare(st.Name, t, st.Pos); err != nil {
			return nil, err
		}
		return &declStmt{name: st.Name, typ: t}, nil
	}
	it := fc.typeOfAs(st.Init, t)
	if it != t {
		return nil, l.errf(exprPos(st.Init, st.Pos), "using %s to initialize %s variable %q is not supported", it, t, st.Name)
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

// printArgs lowers the arguments of io.println/io.print.
func (fc *funcCtx) printArgs(call *lang.CallExpr) ([]*expr, error) {
	args := make([]*expr, 0, len(call.Args))
	for _, a := range call.Args {
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		switch t := fc.typeOf(a); t {
		case "int", "String", "bool", "float", "long", "pointer", "null", "?", "interface{}", "List<String>", "uchar":
		case "int&", "float&", "bool&", "String&", "long&": // T&: print the dereferenced value (same as the interpreter)
		default:
			if !isBitsT(t) {
				return nil, fc.l.errf(exprPos(a, call.Pos), "printing a value of type %s is not supported (the compiler handles int/float/bool/String/interface{})", t)
			}
		}
		args = append(args, x)
	}
	return args, nil
}

// cond lowers a condition expression (typecheck already guarantees bool).
func (fc *funcCtx) cond(x lang.Expr) (*expr, error) {
	switch t := fc.typeOf(x); t {
	case "bool", "?":
	default:
		return nil, fc.l.errf(exprPos(x, lang.Pos{Line: 1, Col: 1}), "暂未支持 %s 类型的条件（需要 bool）", t)
	}
	return fc.expr(x)
}

// ---------- Assignment ----------

func (fc *funcCtx) assignStmt(st *lang.AssignStmt) (stmt, error) {
	l := fc.l
	switch tgt := st.Target.(type) {
	case *lang.Ident:
		vt, ok := fc.lookup(tgt.Name)
		if !ok {
			return nil, l.errf(tgt.Pos, "赋值目标 %q 未声明", tgt.Name)
		}
		xt := fc.typeOfAs(st.X, vt)
		if _, base, isRef := ptrRefBase(vt); isRef {
			// p = v (v is a T) → write through; p = q (q is a T&) → rebind the pointer
			if !fc.assignable(xt, vt) && !fc.assignable(xt, base) {
				return nil, l.errf(exprPos(st.X, st.Pos), "assigning %s to %s variable %q is not supported", xt, vt, tgt.Name)
			}
			x, err := fc.expr(st.X)
			if err != nil {
				return nil, err
			}
			// interpreter semantics: p = q (q is a T&) → write the value q points at into the cell p points at (no rebinding)
			return &assignStmt{name: tgt.Name, x: x, thru: true}, nil
		}
		if !fc.assignable(xt, vt) && !(isBitsT(vt) && isIntConstLit(st.X)) {
			return nil, l.errf(exprPos(st.X, st.Pos), "assigning %s to %s variable %q is not supported", xt, vt, tgt.Name)
		}
		x, err := fc.exprAs(st.X, vt)
		if err != nil {
			return nil, err
		}
		return &assignStmt{name: tgt.Name, x: x}, nil

	case *lang.IndexExpr:
		rt := fc.typeOf(tgt.X)
		if isBitsT(rt) {
			return fc.bitAssign(st, tgt)
		}
		if _, ok := listElem(rt); !ok {
			return nil, l.errf(st.Pos, "this subscript assignment is not supported (the compiler handles List<int> only)")
		}
		if t := fc.typeOf(tgt.Idx); t != "int" && t != "?" {
			return nil, l.errf(exprPos(tgt.Idx, st.Pos), "暂未支持下标为 %s（需要 int）", t)
		}
		if t := fc.typeOf(st.X); !fc.assignable(t, "int") {
			return nil, l.errf(exprPos(st.X, st.Pos), "assigning %s to a List<int> element is not supported", t)
		}
		recv, err := fc.expr(tgt.X)
		if err != nil {
			return nil, err
		}
		idx, err := fc.expr(tgt.Idx)
		if err != nil {
			return nil, err
		}
		x, err := fc.expr(st.X)
		if err != nil {
			return nil, err
		}
		return &indexAssignStmt{recv: recv, idx: idx, x: x}, nil

	case *lang.MemberExpr:
		rt := fc.typeOf(tgt.X)
		if l.isStructType(rt) {
			ft, ok := l.fieldType(rt, tgt.Name)
			if !ok {
				return nil, l.errf(tgt.Pos, "struct %s 没有字段 %q", rt, tgt.Name)
			}
			if !fc.assignable(fc.typeOf(st.X), ft) && !(isBitsT(ft) && isIntConstLit(st.X)) {
				return nil, l.errf(exprPos(st.X, st.Pos), "assigning %s to %s.%s (%s) is not supported", fc.typeOf(st.X), rt, tgt.Name, ft)
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
		return nil, l.errf(tgt.Pos, "member assignment is not supported by the compiler (interface dispatch is interpreter-only)")
	}
	return nil, l.errf(st.Pos, "this assignment target is not supported (the compiler handles variables, List subscripts and struct fields only)")
}

// ---------- Expressions ----------

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
		return &expr{kind: kNull, typ: "null"}, nil

	case *lang.Ident:
		t, ok := fc.lookup(e.Name)
		if !ok {
			if _, isLib := l.libs[e.Name]; isLib {
				return nil, l.errf(e.Pos, "暂未支持 library 库对象 %q 作为值（FFI 仅解释器可用）", e.Name)
			}
			return nil, l.errf(e.Pos, "undeclared identifier %q (the compiler does not support function references or global names)", e.Name)
		}
		if t == "IOStream" {
			return nil, l.errf(e.Pos, "暂未支持 IOStream 值 %q（编译器仅在 main 入口绑定 io）", e.Name)
		}
		return &expr{kind: kIdent, typ: t, s: e.Name}, nil

	case *lang.IncExpr:
		// Moving a reference (p++) belongs to the reference machinery the compiler does not lower yet.
		return nil, l.errf(e.Pos, "compiling ++ (reference move) is not supported yet (implemented in the interpreter only)")
	case *lang.BinOp:
		return fc.binOp(e)
	case *lang.UnOp:
		return fc.unOp(e)
	case *lang.CallExpr:
		return fc.call(e)
	case *lang.IndexExpr:
		rt := fc.typeOf(e.X)
		if e.Bits > 0 {
			return fc.bitIndex(e, rt)
		}
		elem, ok := listElem(rt)
		if !ok {
			return nil, l.errf(e.Pos, "this subscript read is not supported (the compiler handles List<int> only)")
		}
		if t := fc.typeOf(e.Idx); t != "int" && t != "?" {
			return nil, l.errf(exprPos(e.Idx, e.Pos), "暂未支持下标为 %s（需要 int）", t)
		}
		recv, err := fc.expr(e.X)
		if err != nil {
			return nil, err
		}
		idx, err := fc.expr(e.Idx)
		if err != nil {
			return nil, err
		}
		return &expr{kind: kIndex, typ: elem, line: e.Pos.Line, idx: &indexExpr{recv: recv, i: idx}}, nil

	case *lang.StructLit:
		return fc.structLit(e, fc.expectT)

	case *lang.NewExpr:
		if e.Size != nil {
			return nil, l.errf(e.Pos, "暂未支持 new %s[n]（解释器产出 List<T>&，无法赋给 List<T>；请用 [..] 字面量）", e.Typ)
		}
		pt := strings.TrimSpace(e.Typ)
		if err := l.checkType(pt, e.Pos, "new"); err != nil {
			return nil, err
		}
		if _, _, ok := ptrRefBase(pt); ok {
			return nil, l.errf(e.Pos, "暂未支持 new %s（new 的元素类型不能是指针）", e.Typ)
		}
		return &expr{kind: kNewRef, typ: pt + "&", s: pt}, nil
	case *lang.ListLit:
		// list literal: valid in any expression position (the compiler lowers List<int> and List<String>)
		t := "List<int>"
		if _, ok := listElem(fc.expectT); ok {
			t = fc.expectT
		}
		elem, ok := listElem(t)
		if !ok || (elem != "int" && elem != "String") {
			return nil, l.errf(lang.Pos{Line: 1, Col: 1}, "list literal %s is not supported (the compiler lowers List<int> and List<String>)", t)
		}
		items := make([]*expr, 0, len(e.Items))
		for _, it := range e.Items {
			et := fc.typeOf(it)
			if !fc.assignable(et, elem) {
				return nil, l.errf(lang.Pos{Line: 1, Col: 1}, "initializing %s with %s elements is not supported", et, t)
			}
			x, err := fc.expr(it)
			if err != nil {
				return nil, err
			}
			items = append(items, x)
		}
		kind := kList
		if elem == "String" {
			kind = kListS
		}
		return &expr{kind: kind, typ: t, lst: &listLit{items: items}}, nil
	case *lang.ScopeCall:
		return fc.scopeCall(e)
	case *lang.MemberExpr:
		return fc.member(e)
	}
	return nil, l.errf(lang.Pos{Line: 1, Col: 1}, "compiler 内部错误：未知表达式类型 %T", x)
}

// binOp lowers a binary operation (arithmetic/comparison/logic/String concatenation/operator overload).
func (fc *funcCtx) binOp(e *lang.BinOp) (*expr, error) {
	l := fc.l
	lt, rt := fc.typeOf(e.L), fc.typeOf(e.R)
	pos := e.Pos
	// operator overload: same struct type and the impl defines __add__ etc.
	if lt == rt && l.isStructType(lt) {
		if op := opMethodFor(e.Op); op != "" {
			if mi := l.lookupSelf(lt, op); mi != nil && len(mi.fn.Params) == 2 {
				imi, err := l.instantiateFor(mi, lt, []lang.Expr{e.L, e.R}, fc, e.Pos)
				if err != nil {
					return nil, err
				}
				x, err := fc.methodCall(imi, lt, e.L, []lang.Expr{e.R}, e.Pos)
				if err != nil {
					return nil, err
				}
				// operator overload: the interpreter passes operands by value (BinOp evaluation → not a reference), so arguments are not written back
				x.method.byVal = true
				return x, nil
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
			return nil, l.errf(pos, "using %q on %s / %s is not supported (Operation overloading works for same-type structs only)", lt, rt, e.Op)
		}
		x, err := fc.binary(e, "+")
		if err != nil {
			return nil, err
		}
		x.typ = numResult(lt, rt)
		return x, nil
	case "-", "*", "/":
		if !numLike(lt) || !numLike(rt) {
			return nil, l.errf(pos, "using %q on %s / %s is not supported (Operation overloading works for same-type structs only)", lt, rt, e.Op)
		}
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.typ = numResult(lt, rt)
		x.line = pos.Line
		return x, nil
	case "%":
		// float modulo: the interpreter reports a TypeError at runtime (the compiled path reports the same error at runtime, see the float % branch in cgen)
		if lt == "float" || rt == "float" {
			x, err := fc.binary(e, e.Op)
			if err != nil {
				return nil, err
			}
			x.typ = "float"
			x.line = pos.Line
			return x, nil
		}
		if !numLike(lt) || !numLike(rt) {
			return nil, l.errf(pos, "using %q on %s / %s is not supported (Operation overloading works for same-type structs only)", lt, rt, e.Op)
		}
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.typ = "int"
		x.line = pos.Line
		return x, nil
	case "&", "|", "^":
		// The bitwise family: raw bits keep their width, an int view is the 32-bit view (spec §3.3).
		if isBitsT(lt) || isBitsT(rt) {
			return fc.rawBitsBin(e, lt, rt)
		}
		if !isIntViewT(lt) || !isIntViewT(rt) {
			return nil, l.errf(pos, "using %q on %s / %s is not supported (bitwise operators need int views or raw bits)", e.Op, lt, rt)
		}
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.typ = "int"
		return x, nil
	case "<<", ">>":
		// Raw bits shift within their own width; an int view keeps the existing 32-bit behaviour.
		if isBitsT(lt) || isBitsT(rt) {
			return fc.rawBitsBin(e, lt, rt)
		}
		if !isIntViewT(lt) || !isIntViewT(rt) {
			return nil, l.errf(pos, "using %q on %s / %s is not supported (shifts need int operands)", lt, rt, e.Op)
		}
		x, err := fc.binary(e, e.Op)
		if err != nil {
			return nil, err
		}
		x.typ = "int"
		return x, nil
	case "===":
		// Storage identity is implemented in the interpreter. The compiler would need address
		// computation for both operands, which arrives with the pointer work (R2); until then it is a
		// hard error instead of a silently different answer (project policy for unsupported constructs).
		return nil, l.errf(e.Pos, "compiling %s is not supported yet (=== storage identity is implemented in the interpreter only)", e.Op)
	case "==", "!=", "<", "<=", ">", ">=":
		// T& / pointer T comparison: against null it compares pointer identity (the interpreter's equalValues(ref, nil));
		// against a value it compares the dereferenced value (the interpreter's p == 0 → compares the pointed-to cell).
		if (isPtrRefT(lt) && rt == "null") || (lt == "null" && isPtrRefT(rt)) || (isPtrRefT(lt) && isPtrRefT(rt)) {
			if e.Op != "==" && e.Op != "!=" {
				return nil, l.errf(pos, "using %q on %s / %s is not supported (pointers allow == / != only)", lt, rt, e.Op)
			}
			x, err := fc.expr(e.L)
			if err != nil {
				return nil, err
			}
			y, err := fc.expr(e.R)
			if err != nil {
				return nil, err
			}
			if isPtrRefT(lt) {
				x.noDeref = true
			}
			if isPtrRefT(rt) {
				y.noDeref = true
			}
			return &expr{kind: kCmp, op: e.Op, typ: "bool", l: x, r: y, line: pos.Line}, nil
		}
		// T& vs value comparison: the interpreter does equalValues(dereferenced value, value), but when the nullable reference is nil
		// equalValues(nil, value) = false → expand into (ptr != null) && (deref OP value) (for ==;
		// for != use (ptr == null) || (deref != value)). The pointer side must be a variable (to avoid double evaluation).
		if isPtrRefT(lt) != isPtrRefT(rt) && lt != "null" && rt != "null" {
			if e.Op != "==" && e.Op != "!=" {
				return nil, l.errf(pos, "using %q on %s / %s is not supported (nullable references allow == / != only)", lt, rt, e.Op)
			}
			refSide, valSide := e.L, e.R
			if isPtrRefT(rt) {
				refSide, valSide = e.R, e.L
			}
			if _, ok := refSide.(*lang.Ident); !ok {
				return nil, l.errf(exprPos(refSide, pos), "comparing a non-variable nullable reference is not supported (the compiler requires a T& variable)")
			}
			rv, err := fc.expr(refSide)
			if err != nil {
				return nil, err
			}
			rd, err := fc.expr(refSide)
			if err != nil {
				return nil, err
			}
			vv, err := fc.expr(valSide)
			if err != nil {
				return nil, err
			}
			rv.noDeref = true
			nullCmp := &expr{kind: kCmp, op: "!=", typ: "bool", l: rv, r: &expr{kind: kNull, typ: "null"}, line: pos.Line}
			valCmp := &expr{kind: kCmp, op: e.Op, typ: "bool", l: rd, r: vv, line: pos.Line}
			if e.Op == "!=" {
				nullCmp.op = "=="
				return &expr{kind: kAndOr, op: "||", typ: "bool", l: nullCmp, r: valCmp, line: pos.Line}, nil
			}
			return &expr{kind: kAndOr, op: "&&", typ: "bool", l: nullCmp, r: valCmp, line: pos.Line}, nil
		}
		// T& vs value comparison: by the dereferenced value type (the interpreter's p == 0 / p == "x" both compare values)
		if _, b, ok := ptrRefBase(lt); ok {
			lt = b
		}
		if _, b, ok := ptrRefBase(rt); ok {
			rt = b
		}
		// interface{} (tAny) equality: if either side is any, compare as any (dispatched at runtime per RTTI;
		// consistent with the interpreter's equalValues: int/float compared numerically across types, String by content, struct/List by reference)
		if isAnyT(lt) || isAnyT(rt) {
			if e.Op != "==" && e.Op != "!=" {
				return nil, l.errf(pos, "using %q on interface{} is not supported (only == / !=)", e.Op)
			}
			x, err := fc.expr(e.L)
			if err != nil {
				return nil, err
			}
			y, err := fc.expr(e.R)
			if err != nil {
				return nil, err
			}
			if !isAnyT(lt) {
				if x, err = fc.boxTo(x, "interface{}", exprPos(e.L, pos)); err != nil {
					return nil, err
				}
			}
			if !isAnyT(rt) {
				if y, err = fc.boxTo(y, "interface{}", exprPos(e.R, pos)); err != nil {
					return nil, err
				}
			}
			return &expr{kind: kCmp, op: e.Op, typ: "bool", l: x, r: y, line: pos.Line}, nil
		}
		if l.isStructType(lt) && lt == rt {
			// __eq__ etc. were already handled by the overload branch above; everything else = reference comparison (interpreter Value semantics)
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
		// pointer / null: reference comparison (interpreter Value semantics: the same pointer is equal; null is the zero value)
		if isPtrType(lt) || isPtrType(rt) {
			if e.Op != "==" && e.Op != "!=" {
				return nil, l.errf(pos, "using %q on %s / %s is not supported (pointers allow == / != only)", lt, rt, e.Op)
			}
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
				return nil, l.errf(pos, "using %q on bool is not supported (the interpreter allows == / != only)", e.Op)
			}
			x, err := fc.binary(e, e.Op)
			if err != nil {
				return nil, err
			}
			x.kind = kCmp
			x.typ = "bool"
			return x, nil
		}
		// raw bits: equality compares bit by bit, ordering is not admitted (spec §3.3)
		if isBitsT(lt) || isBitsT(rt) {
			if lt != rt && !rawVsConst(e, lt, rt) {
				return nil, l.errf(pos, "using %q on %s / %s is not supported: the widths differ", e.Op, lt, rt)
			}
			if e.Op != "==" && e.Op != "!=" {
				return nil, l.errf(pos, "using %q on %s is not supported (raw bits admit == and != only)", e.Op, lt)
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
			return nil, l.errf(pos, "using %q on %s / %s is not supported (String/Operation comparison needs the same type)", lt, rt, e.Op)
		}
		// ordered comparison: both int and float are compared as float64 (the interpreter's ordCmp)
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
				return nil, l.errf(pos, "%s with %q is not supported (bool required)", t, e.Op)
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
	return nil, l.errf(pos, "operator %q is not supported by the compiler (available in the interpreter)", e.Op)
}

// opMethodFor maps an operator to an Operation protocol method name (consistent with internal/lang/typecheck).
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

// rawVsConst reports whether an operation pairs raw bits with an integer constant, which adapts to
// the raw width (spec §3.3).
func rawVsConst(e *lang.BinOp, lt, rt string) bool {
	return (isBitsT(lt) && isIntConstLit(e.R)) || (isBitsT(rt) && isIntConstLit(e.L))
}

// rawBitsBin lowers a bitwise operation with at least one raw-bits operand: both sides end up raw bits
// of the operation's width, which the type checker stamped on the node (spec §3.3).
func (fc *funcCtx) rawBitsBin(e *lang.BinOp, lt, rt string) (*expr, error) {
	l := fc.l
	if e.Bits == 0 {
		return nil, l.errf(e.Pos, "using %q on %s / %s is not supported (raw bits need one width)", e.Op, lt, rt)
	}
	typ := rawNameOf(e.Bits)
	for _, side := range []string{lt, rt} {
		if isBitsT(side) && side != typ {
			return nil, l.errf(e.Pos, "using %q on %s / %s is not supported: the widths differ", e.Op, lt, rt)
		}
	}
	x, err := fc.binary(e, e.Op)
	if err != nil {
		return nil, err
	}
	x.typ = typ
	x.bits = e.Bits
	return x, nil
}

// binary lowers the left and right operands and builds the operation expression.
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

// complement lowers the bitwise complement ~x: raw bits keep their width, an int view is complemented
// in 32 bits (spec §3.3).
func (fc *funcCtx) complement(e *lang.UnOp) (*expr, error) {
	t := fc.typeOf(e.X)
	if !isBitsT(t) && !isIntViewT(t) {
		return nil, fc.l.errf(e.Pos, "the bitwise complement '~' requires an int operand, got %s", t)
	}
	inner, err := fc.expr(e.X)
	if err != nil {
		return nil, err
	}
	x := &expr{kind: kBin, op: "^", typ: "int", l: &expr{kind: kInt, typ: "int", i: -1}, r: inner, line: e.Pos.Line}
	if isBitsT(t) {
		x.typ = rawNameOf(e.Bits)
		x.bits = e.Bits
	}
	return x, nil
}

// bitIndex lowers b[i] on raw bits: the i-th bit comes out as a bit (spec §3.3).
func (fc *funcCtx) bitIndex(e *lang.IndexExpr, rt string) (*expr, error) {
	if !isBitsT(rt) {
		return nil, fc.l.errf(e.Pos, "subscripting a bit requires raw bits, got %s", rt)
	}
	if t := fc.typeOf(e.Idx); !isIntViewT(t) && t != "?" {
		return nil, fc.l.errf(exprPos(e.Idx, e.Pos), "a bit index must be int, got %s", t)
	}
	recv, err := fc.expr(e.X)
	if err != nil {
		return nil, err
	}
	idx, err := fc.expr(e.Idx)
	if err != nil {
		return nil, err
	}
	return &expr{kind: kIndex, typ: "bit", bits: e.Bits, line: e.Pos.Line, idx: &indexExpr{recv: recv, i: idx}}, nil
}

// conversion lowers a conversion call T(x) — the type checker stamped the destination (spec §3.4).
func (fc *funcCtx) conversion(c *lang.CallExpr, arg lang.Expr) (*expr, error) {
	x, err := fc.expr(arg)
	if err != nil {
		return nil, err
	}
	return &expr{kind: kConv, typ: c.ConvDst, s: c.ConvDst, l: x}, nil
}

// bitAssign lowers b[i] = v: the raw value is read, its i-th bit replaced and stored back (spec §3.3).
func (fc *funcCtx) bitAssign(st *lang.AssignStmt, tgt *lang.IndexExpr) (stmt, error) {
	rt := fc.typeOf(tgt.X)
	if !isBitsT(rt) {
		return nil, fc.l.errf(st.Pos, "b[i] = v requires raw bits, got %s", rt)
	}
	if t := fc.typeOf(st.X); !fc.assignable(t, "bit") && !isIntConstLit(st.X) {
		return nil, fc.l.errf(exprPos(st.X, st.Pos), "assigning %s to a bit is not supported", t)
	}
	recv, err := fc.expr(tgt.X)
	if err != nil {
		return nil, err
	}
	idx, err := fc.expr(tgt.Idx)
	if err != nil {
		return nil, err
	}
	val, err := fc.expr(st.X)
	if err != nil {
		return nil, err
	}
	return &bitAssignStmt{recv: recv, idx: idx, x: val, bits: tgt.Bits}, nil
}

// unOp lowers unary operations: - expands to 0 - x (fneg semantics for float), ! goes through logical negation.
func (fc *funcCtx) unOp(e *lang.UnOp) (*expr, error) {
	l := fc.l
	if e.Op == "&" {
		// Address-of feeds the reference machinery, which the compiler does not lower yet. Hard error
		// rather than a silently different answer (project policy for unsupported constructs).
		return nil, l.errf(e.Pos, "compiling & (address-of) is not supported yet (references are implemented in the interpreter only)")
	}
	if e.Op == "~" {
		return fc.complement(e)
	}
	t := fc.typeOf(e.X)
	switch e.Op {
	case "-":
		if l.isStructType(t) {
			if mi := l.lookupSelf(t, "__neg__"); mi != nil {
				imi, err := l.instantiateFor(mi, t, []lang.Expr{e.X}, fc, e.Pos)
				if err != nil {
					return nil, err
				}
				x, err := fc.methodCall(imi, t, e.X, nil, e.Pos)
				if err != nil {
					return nil, err
				}
				x.method.byVal = true // unary operator overload: operand by value
				return x, nil
			}
		}
		if !numLike(t) {
			return nil, l.errf(e.Pos, "negating %s is not supported (Operation overloading works for same-type structs only)", t)
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
		if elem, ok := listElem(t); ok && elem == "int" {
			recv, err := fc.expr(e.X)
			if err != nil {
				return nil, err
			}
			return &expr{kind: kMethod, typ: "int", line: e.Pos.Line, method: &methodExpr{recv: recv, name: "peek"}}, nil
		}
		return nil, l.errf(e.Pos, "dereference * is not supported (pointers are interpreter-only)")
	}
	return nil, l.errf(e.Pos, "unary operator %q is not supported", e.Op)
}

// ---------- Calls ----------

func (fc *funcCtx) call(c *lang.CallExpr) (*expr, error) {
	l := fc.l
	if c.Sign != nil {
		return fc.signCall(c)
	}
	// Conversion call T(x): the shared frontend stamped the destination type (spec §3.4).
	if c.ConvDst != "" && len(c.Args) == 1 {
		return fc.conversion(c, c.Args[0])
	}
	switch fn := c.Fn.(type) {
	case *lang.Ident:
		return fc.callNamed(fn.Name, fn.Pos, c)
	case *lang.MemberExpr:
		return fc.callMethod(c, fn)
	case *lang.ScopeCall:
		return fc.scopeCallCall(c, fn)
	}
	return nil, l.errf(c.Pos, "this call form is not supported (the compiler handles f(...), obj.m(...) and T::m(...) only)")
}

// signCall lowers a signature call f(args) @sign(prefix) (canonical §6: sign.call(prefix) receives
// .{in: List(args), out: nil} and returns rec.out). The compiler only lowers the built-in memorize signature instance
// (@mb() memoization: caches the wrapped function's return value by argument list); other Sign instances are an explicit error.
func (fc *funcCtx) signCall(c *lang.CallExpr) (*expr, error) {
	l := fc.l
	id, ok := c.Fn.(*lang.Ident)
	if !ok {
		return nil, l.errf(c.Pos, "this signature call is not supported (the compiler requires a direct function name: f(args) @sign(...))")
	}
	st, ok := fc.lookup(c.Sign.Name)
	if !ok {
		return nil, l.errf(c.Pos, "未声明的签名实例 %q（@%s）", c.Sign.Name, c.Sign.Name)
	}
	if st != "memorize" {
		return nil, l.errf(c.Pos, "暂未支持 %s 类型的签名实例 %q（编译器只 lower 内置 memorize）", st, c.Sign.Name)
	}
	irName, fd, err := fc.resolveFunc(id.Name, c.Args, id.Pos)
	if err != nil {
		return nil, err
	}
	if len(c.Args) != len(fd.Params) {
		return nil, l.errf(id.Pos, "函数 %s 需要 %d 个参数，got %d", id.Name, len(fd.Params), len(c.Args))
	}
	if l.fnRet[irName] != "int" {
		return nil, l.errf(id.Pos, "暂未支持 memorize 包装返回 %s 的函数 %s（运行时缓存只承载 int）", l.fnRet[irName], id.Name)
	}
	for _, p := range fd.Params {
		if pt := strings.TrimSpace(p.Type); pt != "int" {
			return nil, l.errf(p.Pos, "暂未支持 memorize 包装 %s 形参（运行时缓存键只承载 int）", pt)
		}
	}
	// arguments of the wrapped call: the interpreter records the in list by value after derefArgs (non-lvalue semantics)
	args := make([]*expr, 0, len(c.Args))
	for _, a := range c.Args {
		t := fc.typeOf(a)
		if !fc.assignable(t, "int") {
			return nil, l.errf(exprPos(a, id.Pos), "暂未支持 memorize 的 %s 实参（运行时缓存键只承载 int）", t)
		}
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		args = append(args, x)
	}
	// explicit prefix arguments at @: the interpreter evaluates them into the prefix record (unused by memorize) → evaluate and discard the side effects
	skips := make([]*expr, 0, len(c.Sign.Args))
	for _, a := range c.Sign.Args {
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		skips = append(skips, x)
	}
	return &expr{kind: kMemoCall, typ: "int", line: c.Pos.Line,
		call: &callExpr{name: irName, args: args, byValArgs: true},
		memo: &memoExpr{handle: &expr{kind: kIdent, typ: "memorize", s: c.Sign.Name}, skips: skips}}, nil
}

// callNamed lowers a named function call (including the built-ins sum/clock and generic instantiation).
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
	irName, fd, err := fc.resolveFunc(name, c.Args, pos)
	if err != nil {
		return nil, err
	}
	if len(c.Args) != len(fd.Params) {
		return nil, l.errf(pos, "函数 %s 需要 %d 个参数，got %d", name, len(fd.Params), len(c.Args))
	}
	if l.nilFns[irName] && fc.discarded != c {
		return nil, l.errf(pos, "using the return value of the log-containing function %s in an expression is not supported (the interpreter returns nil)", name)
	}
	args, err := fc.callArgs(c, fd.Params)
	if err != nil {
		return nil, err
	}
	return &expr{kind: kCall, typ: l.fnRet[irName], line: pos.Line, call: &callExpr{name: irName, args: args}}, nil
}

// resolveGenerator resolves sum's generator (picks the only int(int) candidate among overloads).
func (l *lowerer) resolveGenerator(name string, pos lang.Pos) (*lang.FuncDecl, string, error) {
	if cands, ok := l.ovl[name]; ok {
		var hit *lang.FuncDecl
		for _, f := range cands {
			if len(f.Params) == 1 && f.Params[0].Type == "int" && retOf(f) == "int" {
				if hit != nil {
					return nil, "", l.errf(pos, "sum 生成器 %q 有多个 int(%s) 重载，无法确定", name, name)
				}
				hit = f
			}
		}
		if hit == nil {
			return nil, "", l.errf(pos, "sum 生成器 %q 没有 int(%s) 重载", name, name)
		}
		return hit, l.fnIR[hit], nil
	}
	gfn, ok := l.fns[name]
	if !ok {
		return nil, "", l.errf(pos, "unknown generator function %q (the compiler only supports non-generic functions defined in the same program)", name)
	}
	return gfn, name, nil
}

// resolveRunnerTarget resolves merge's target function (picks the only candidate whose argument count matches).
func (l *lowerer) resolveRunnerTarget(name string, nargs int, pos lang.Pos) (*lang.FuncDecl, string, error) {
	cands, ok := l.ovl[name]
	if !ok {
		fn, ok := l.fns[name]
		if !ok {
			return nil, "", l.errf(pos, "unknown function %q (the compiler only supports functions defined in the same program)", name)
		}
		return fn, name, nil
	}
	var hit *lang.FuncDecl
	for _, f := range cands {
		if len(f.Params) == nargs {
			if hit != nil {
				return nil, "", l.errf(pos, "merge 目标 %q 有多个同参数个数的重载，无法确定", name)
			}
			hit = f
		}
	}
	if hit == nil {
		return nil, "", l.errf(pos, "merge 目标 %q 没有 %d 个参数的重载", name, nargs)
	}
	return hit, l.fnIR[hit], nil
}

// resolveFunc resolves a function call (including overloads): returns the IR name and the selected declaration.
// Scoring matches internal/lang bestMatchT: same kind +10 / assignable +5 / unknown +1,
// the argument count must match, and the highest score wins (ties go to the first declared).
func (fc *funcCtx) resolveFunc(name string, args []lang.Expr, pos lang.Pos) (string, *lang.FuncDecl, error) {
	l := fc.l
	cands, isOvl := l.ovl[name]
	if !isOvl {
		fd, ok := l.fns[name]
		if !ok {
			return "", nil, l.errf(pos, "unknown function %q (the compiler only supports functions defined in the same program)", name)
		}
		return name, fd, nil
	}
	var best *lang.FuncDecl
	bestScore := -1
	for _, fn := range cands {
		if len(fn.Params) != len(args) {
			continue
		}
		score := 0
		ok := true
		for i, p := range fn.Params {
			pt := strings.TrimSpace(p.Type)
			at := fc.typeOf(args[i])
			switch {
			case l.kindOf(at) == l.kindOf(pt):
				score += 10
			case fc.assignable(at, pt):
				score += 5
			case at == "?" || pt == "?":
				score += 1
			default:
				ok = false
			}
			if !ok {
				break
			}
		}
		if ok && score > bestScore {
			best, bestScore = fn, score
		}
	}
	if best == nil {
		return "", nil, l.errf(pos, "未找到匹配重载 %q（参数个数/类型不匹配）", name)
	}
	return l.fnIR[best], best, nil
}

// callArgs lowers the argument list and checks assignability (int → float allowed).
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

// callSum lowers the built-in sum(generator, begin, stop[, step]).
func (fc *funcCtx) callSum(c *lang.CallExpr, pos lang.Pos) (*expr, error) {
	l := fc.l
	if len(c.Args) != 3 && len(c.Args) != 4 {
		return nil, l.errf(pos, "sum(generate, begin, stop[, step]) 需要 3 或 4 个参数，got %d", len(c.Args))
	}
	gid, ok := c.Args[0].(*lang.Ident)
	if !ok {
		return nil, l.errf(exprPos(c.Args[0], pos), "this sum generator is not supported (the compiler requires a named function)")
	}
	if gfn, isGen := l.generics[gid.Name]; isGen {
		// generic generator: sum requires int(int), monomorphized with T=int
		if len(gfn.TypeParams) != 1 || len(gfn.Params) != 1 ||
			strings.TrimSpace(gfn.Params[0].Type) != gfn.TypeParams[0] ||
			strings.TrimSpace(gfn.Ret) != gfn.TypeParams[0] {
			return nil, l.errf(gid.Pos, "this generic generator %s is not supported (the compiler requires fn<T> %s(T) T)", gid.Name, gid.Name)
		}
		irName, err := l.instantiateFunc(gid.Name, gfn, map[string]string{gfn.TypeParams[0]: "int"})
		if err != nil {
			return nil, err
		}
		args := []*expr{{kind: kIdent, typ: "function", s: irName}}
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
	gfn, gIR, err := l.resolveGenerator(gid.Name, gid.Pos)
	if err != nil {
		return nil, err
	}
	if len(gfn.Params) != 1 || gfn.Params[0].Type != "int" || l.fnRet[gIR] != "int" {
		return nil, l.errf(gid.Pos, "generator %q is not supported (the compiler requires int %s(int))", gid.Name, gid.Name)
	}
	args := []*expr{{kind: kIdent, typ: "function", s: gIR}}
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

// callMethod lowers obj.m(...).
func (fc *funcCtx) callMethod(c *lang.CallExpr, me *lang.MemberExpr) (*expr, error) {
	l := fc.l
	rt := fc.typeOf(me.X)
	// HashTable<K,V> builtin methods
	if isTableT(rt) {
		return fc.tableCall(c, me, rt)
	}
	// built-in toString()
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
			if len(c.Args) != 0 {
				return nil, l.errf(me.Pos, "toString() 不接受参数")
			}
			recv, err := fc.expr(me.X)
			if err != nil {
				return nil, err
			}
			return &expr{kind: kMethod, typ: "String", method: &methodExpr{recv: recv, name: "toString"}}, nil
		}
	}
	// List<String> builtin methods (read-only subset: size/get/toString)
	if elem, ok := listElem(rt); ok && elem == "String" {
		rx, err := fc.expr(me.X)
		if err != nil {
			return nil, err
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
			return &expr{kind: kMethod, typ: "int", method: &methodExpr{recv: rx, name: "size"}}, nil
		case "get":
			if len(c.Args) != 1 {
				return nil, l.errf(me.Pos, "List.get(i) 需要 1 个参数")
			}
			if t := fc.typeOf(c.Args[0]); t != "int" && t != "?" {
				return nil, l.errf(exprPos(c.Args[0], me.Pos), "暂未支持 List.get 的 %s 下标（需要 int）", t)
			}
			return &expr{kind: kMethod, typ: "String", line: me.Pos.Line, method: &methodExpr{recv: rx, name: "get", args: args}}, nil
		case "toString":
			if len(c.Args) != 0 {
				return nil, l.errf(me.Pos, "toString() 不接受参数")
			}
			return &expr{kind: kMethod, typ: "String", method: &methodExpr{recv: rx, name: "toString"}}, nil
		case "append":
			if len(c.Args) != 1 {
				return nil, l.errf(me.Pos, "List.append(v) 需要 1 个参数")
			}
			if t := fc.typeOf(c.Args[0]); t != "String" && t != "?" {
				return nil, l.errf(exprPos(c.Args[0], me.Pos), "暂未支持 List<String>.append 的 %s 参数（需要 String）", t)
			}
			return &expr{kind: kMethod, typ: "void", line: me.Pos.Line, method: &methodExpr{recv: rx, name: "append", args: args}}, nil
		}
		return nil, l.errf(me.Pos, "暂未支持 List<String>.%s（编译器 lower size/get/toString/append 与 for-in 迭代）", me.Name)
	}
	// List builtin methods (the receiver may be any List<int> expression: a variable / struct field etc.)
	if elem, ok := listElem(rt); ok && elem == "int" {
		rx, err := fc.expr(me.X)
		if err != nil {
			return nil, err
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
			return &expr{kind: kMethod, typ: "int", method: &methodExpr{recv: rx, name: "size"}}, nil
		case "get":
			if len(c.Args) != 1 {
				return nil, l.errf(me.Pos, "List.get(i) 需要 1 个参数")
			}
			if t := fc.typeOf(c.Args[0]); t != "int" && t != "?" {
				return nil, l.errf(exprPos(c.Args[0], me.Pos), "暂未支持 List.get 的 %s 下标（需要 int）", t)
			}
			return &expr{kind: kMethod, typ: "int", line: me.Pos.Line, method: &methodExpr{recv: rx, name: "get", args: args}}, nil
		case "append":
			if len(c.Args) != 1 {
				return nil, l.errf(me.Pos, "List.append(v) 需要 1 个参数")
			}
			if t := fc.typeOf(c.Args[0]); !fc.assignable(t, "int") {
				return nil, l.errf(exprPos(c.Args[0], me.Pos), "暂未支持 List<int>.append 的 %s 参数（需要 int）", t)
			}
			return &expr{kind: kMethod, typ: "void", method: &methodExpr{recv: rx, name: "append", args: args}}, nil
		case "head", "tail", "next":
			if len(c.Args) != 0 {
				return nil, l.errf(me.Pos, "List.%s() 不接受参数", me.Name)
			}
			return &expr{kind: kMethod, typ: "int", line: me.Pos.Line, method: &methodExpr{recv: rx, name: me.Name}}, nil
		case "reset":
			if len(c.Args) != 0 {
				return nil, l.errf(me.Pos, "List.reset() 不接受参数")
			}
			return &expr{kind: kMethod, typ: "List<int>", method: &methodExpr{recv: rx, name: "reset"}}, nil
		case "appendAll":
			if len(c.Args) != 1 {
				return nil, l.errf(me.Pos, "List.appendAll(l2) 需要 1 个参数")
			}
			if t := fc.typeOf(c.Args[0]); t != "List<int>" && t != "?" {
				return nil, l.errf(exprPos(c.Args[0], me.Pos), "List.appendAll 需要 List<int>，got %s", t)
			}
			return &expr{kind: kMethod, typ: "void", method: &methodExpr{recv: rx, name: "appendAll", args: args}}, nil
		case "__sort__":
			if len(c.Args) != 0 {
				return nil, l.errf(me.Pos, "List.__sort__() 不接受参数")
			}
			return &expr{kind: kMethod, typ: "void", method: &methodExpr{recv: rx, name: "__sort__"}}, nil
		}
		return nil, l.errf(me.Pos, "暂未支持 List 方法 %q（编译器支持 size/get/append/head/tail/next/reset/appendAll/toString/__sort__）", me.Name)
	}
	// String builtin methods (rune/byte semantics match the interpreter; the runtime lives in qthreads.c)
	if rt == "String" {
		if x, handled, err := fc.strCall(c, me); handled || err != nil {
			return x, err
		}
	}
	// taskm global / thread variable / Channel variable
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
	// library FFI (external symbols: call the C symbol directly, converting arguments per the C ABI)
	if id, ok := me.X.(*lang.Ident); ok {
		if _, isLib := l.libs[id.Name]; isLib {
			return fc.libCall(c, me, id.Name)
		}
	}
	// interface method (vtable dispatch)
	if l.isIfaceType(rt) {
		return fc.ifaceCall(c, me, rt)
	}
	// struct instance method
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
			return nil, l.errf(me.Pos, "calling static method %s.%s on an instance is not supported (canonical: %s::%s(...))", rt, me.Name, rt, me.Name)
		}
		return nil, l.errf(me.Pos, "struct %s 没有方法 %q（解释器在类型检查期报错）", rt, me.Name)
	}
	// built-in objects: taskm / library / io
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
	return nil, l.errf(me.Pos, "calling method %s on %s is not supported by the compiler (available in the interpreter)", rt, me.Name)
}

// methodCall lowers a single instance method call (including operator overload).
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

// tableKinds splits the key/value types of HashTable<K, V>.
func tableKinds(t string) (string, string) {
	base, args := splitGeneric(t)
	if base != "HashTable" || len(args) != 2 {
		return "?", "?"
	}
	return args[0], args[1]
}

// tableCall lowers the HashTable builtin methods: put/get/contains/remove/size/keys.
// Keys are structured into strings at runtime by the interpreter's rule (TypeName:String()); values are boxed into interface{},
// and deep-copied per RTTI on Put (aligned with the interpreter's HashTable.Put deepCopy).
func (fc *funcCtx) tableCall(c *lang.CallExpr, me *lang.MemberExpr, rt string) (*expr, error) {
	l := fc.l
	keyT, valT := tableKinds(rt)
	recv, err := fc.expr(me.X)
	if err != nil {
		return nil, err
	}
	boxed := func(a lang.Expr, what string) (*expr, error) {
		at := fc.typeOf(a)
		if !fc.assignable(at, "interface{}") {
			return nil, l.errf(exprPos(a, me.Pos), "暂未支持 HashTable.%s 的 %s 参数", me.Name, at)
		}
		_ = what
		return fc.exprAs(a, "interface{}")
	}
	switch me.Name {
	case "size":
		if len(c.Args) != 0 {
			return nil, l.errf(me.Pos, "HashTable.size() 不接受参数")
		}
		return &expr{kind: kTable, typ: "int", tbl: &tableExpr{recv: recv, name: "size", keyT: keyT, valT: valT}}, nil
	case "keys":
		if len(c.Args) != 0 {
			return nil, l.errf(me.Pos, "HashTable.keys() 不接受参数")
		}
		if keyT != "String" {
			// the interpreter's Keys() returns the original key values (int key → int value), which List<String> cannot carry
			return nil, l.errf(me.Pos, "暂未支持 HashTable<%s, %s>.keys()（解释器返回键的原值；编译器只 lower String 键 → List<String>）", keyT, valT)
		}
		return &expr{kind: kTable, typ: "List<String>", tbl: &tableExpr{recv: recv, name: "keys", keyT: keyT, valT: valT}}, nil
	case "put":
		if len(c.Args) != 2 {
			return nil, l.errf(me.Pos, "HashTable.put(k, v) 需要 2 个参数")
		}
		if t := fc.typeOf(c.Args[0]); !fc.assignable(t, keyT) {
			return nil, l.errf(exprPos(c.Args[0], me.Pos), "HashTable<%s, %s>.put 键需要 %s，got %s", keyT, valT, keyT, t)
		}
		if t := fc.typeOf(c.Args[1]); !fc.assignable(t, valT) {
			return nil, l.errf(exprPos(c.Args[1], me.Pos), "HashTable<%s, %s>.put 值需要 %s，got %s", keyT, valT, valT, t)
		}
		k, err := boxed(c.Args[0], "键")
		if err != nil {
			return nil, err
		}
		v, err := boxed(c.Args[1], "值")
		if err != nil {
			return nil, err
		}
		return &expr{kind: kTable, typ: "void", tbl: &tableExpr{recv: recv, name: "put", args: []*expr{k, v}, keyT: keyT, valT: valT}}, nil
	case "get":
		if len(c.Args) != 1 {
			return nil, l.errf(me.Pos, "HashTable.get(k) 需要 1 个参数")
		}
		if t := fc.typeOf(c.Args[0]); !fc.assignable(t, keyT) {
			return nil, l.errf(exprPos(c.Args[0], me.Pos), "HashTable<%s, %s>.get 键需要 %s，got %s", keyT, valT, keyT, t)
		}
		k, err := boxed(c.Args[0], "键")
		if err != nil {
			return nil, err
		}
		t := valT
		if t == "void" || t == "" {
			t = "interface{}"
		}
		return &expr{kind: kTable, typ: t, tbl: &tableExpr{recv: recv, name: "get", args: []*expr{k}, keyT: keyT, valT: valT}}, nil
	case "contains", "remove":
		if len(c.Args) != 1 {
			return nil, l.errf(me.Pos, "HashTable.%s(k) 需要 1 个参数", me.Name)
		}
		if t := fc.typeOf(c.Args[0]); !fc.assignable(t, keyT) {
			return nil, l.errf(exprPos(c.Args[0], me.Pos), "HashTable<%s, %s>.%s 键需要 %s，got %s", keyT, valT, me.Name, keyT, t)
		}
		k, err := boxed(c.Args[0], "键")
		if err != nil {
			return nil, err
		}
		ret := "void"
		if me.Name == "contains" {
			ret = "bool"
		}
		return &expr{kind: kTable, typ: ret, tbl: &tableExpr{recv: recv, name: me.Name, args: []*expr{k}, keyT: keyT, valT: valT}}, nil
	}
	return nil, l.errf(me.Pos, "HashTable method %q is not supported (the compiler handles put/get/contains/remove/size/keys)", me.Name)
}

// scopeCallCall lowers the T::m(...) / space::f(...) call form.
func (fc *funcCtx) scopeCallCall(c *lang.CallExpr, sc *lang.ScopeCall) (*expr, error) {
	l := fc.l
	// built-in HashTable::new(): an i8* handle (key/value types come from the expected type at the declaration)
	if sc.Scope == "HashTable" {
		if sc.Name != "new" || len(sc.Args) != 0 {
			return nil, l.errf(sc.Pos, "暂未支持 HashTable::%s（编译器只 lower HashTable::new()）", sc.Name)
		}
		t := "HashTable<interface{}, interface{}>"
		if isTableT(fc.expectT) {
			t = fc.expectT
		}
		return &expr{kind: kCall, typ: t, call: &callExpr{name: "ql_table_new"}}, nil
	}
	// built-in memorize::new(): a signature instance (@mb() memoization)
	if sc.Scope == "memorize" {
		if sc.Name != "new" || len(sc.Args) != 0 {
			return nil, l.errf(sc.Pos, "暂未支持 memorize::%s（编译器只 lower memorize::new()）", sc.Name)
		}
		return &expr{kind: kCall, typ: "memorize", call: &callExpr{name: "ql_memo_new"}}, nil
	}
	mi, err := l.staticMethodFor(sc.Scope, sc.Name, sc.Args, fc, sc.Pos)
	if err != nil {
		return nil, err
	}
	ret := substType(mi.fn.Ret, mi.subst)
	if ret == "" {
		ret = "void"
	}
	if l.nilFns[mi.irName] {
		return nil, l.errf(sc.Pos, "using the return value of the log-containing function %s in an expression is not supported (the interpreter returns nil)", mi.irName)
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

// scopeCall lowers a space/T:: call in expression position.
func (fc *funcCtx) scopeCall(sc *lang.ScopeCall) (*expr, error) {
	return fc.scopeCallCall(&lang.CallExpr{Fn: sc, Pos: sc.Pos}, sc)
}

// member lowers member access p.x / obj.m (not a call).
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
		return nil, l.errf(e.Pos, "interface member access %s.%s is not supported (dynamic dispatch is interpreter-only)", rt, e.Name)
	}
	if id, ok := e.X.(*lang.Ident); ok {
		if _, isLib := l.libs[id.Name]; isLib {
			return nil, l.errf(e.Pos, "暂未支持 library 成员访问 %s.%s（FFI 仅解释器可用）", id.Name, e.Name)
		}
	}
	return nil, l.errf(e.Pos, "member access %s.%s is not supported by the compiler (available in the interpreter)", rt, e.Name)
}

// structLit lowers a .{...} literal (when target is empty, take the Name filled in by typecheck).
func (fc *funcCtx) structLit(e *lang.StructLit, target string) (*expr, error) {
	l := fc.l
	typ := e.Name
	if typ == "" {
		typ = target
	}
	base, targs := splitGeneric(typ)
	if sd0, ok := l.structs[base]; ok && len(sd0.TypeParams) > 0 && len(targs) == 0 {
		// generic struct literal: typecheck only fills the base name, the instance type is completed from the expected type
		tb, ta := splitGeneric(target)
		if tb == base && len(ta) > 0 {
			typ = target
		} else {
			return nil, l.errf(e.Pos, "a generic struct literal .{...} whose type arguments cannot be inferred is not supported (%s; available in the interpreter)", base)
		}
	}
	sd, sub := l.structSubst(typ)
	if sd == nil {
		return nil, l.errf(e.Pos, "anonymous struct literal .{...} is not supported (the compiler requires a named struct)")
	}
	l.ensureStructTy(typ)
	// collect field values in declaration order (both named and positional forms)
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
		if !fc.assignable(vt, ft) && !(isBitsT(ft) && isIntConstLit(f.X)) {
			return nil, l.errf(exprPos(f.X, e.Pos), "initializing field %s.%s with %s is not supported (%s)", vt, typ, sd.Members[idx].Name, ft)
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
			values[i] = &expr{kind: kInt, typ: "int"} // fields not given = zero value (same as the interpreter)
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

// strCall lowers String builtin methods (SYNTAX.md E4). Returns handled=false for ones not lowered.
func (fc *funcCtx) strCall(c *lang.CallExpr, me *lang.MemberExpr) (*expr, bool, error) {
	l := fc.l
	recv := func() lang.Expr { return me.X }
	// argument lowering helper
	strArg := func(i int) (*expr, error) {
		a := c.Args[i]
		if t := fc.typeOf(a); t != "String" {
			return nil, l.errf(exprPos(a, me.Pos), "%s 需要 String 参数，got %s", me.Name, t)
		}
		return fc.expr(a)
	}
	intArg := func(i int) (*expr, error) {
		a := c.Args[i]
		if t := fc.typeOf(a); t != "int" && t != "?" {
			return nil, l.errf(exprPos(a, me.Pos), "%s 需要 int 参数，got %s", me.Name, t)
		}
		return fc.expr(a)
	}
	line := &expr{kind: kInt, typ: "int", i: int64(me.Pos.Line)}
	call := func(name, typ string, args ...*expr) (*expr, bool, error) {
		rx, err := fc.expr(recv())
		if err != nil {
			return nil, true, err
		}
		return &expr{kind: kCall, typ: typ, line: me.Pos.Line,
			call: &callExpr{name: name, args: append([]*expr{rx}, args...)}}, true, nil
	}
	switch me.Name {
	case "split":
		if len(c.Args) != 1 {
			return nil, true, l.errf(me.Pos, "split(sep) 需要 1 个参数")
		}
		a, err := strArg(0)
		if err != nil {
			return nil, true, err
		}
		return call("ql_str_split", "List<String>", a)
	case "size":
		return call("ql_str_size", "int")
	case "contains", "startsWith", "endsWith":
		a, err := strArg(0)
		if err != nil {
			return nil, true, err
		}
		name := map[string]string{"contains": "ql_str_contains", "startsWith": "ql_str_startswith", "endsWith": "ql_str_endswith"}[me.Name]
		return call(name, "bool", a)
	case "indexOf":
		a, err := strArg(0)
		if err != nil {
			return nil, true, err
		}
		return call("ql_str_indexof", "int", a)
	case "substring":
		start, err := intArg(0)
		if err != nil {
			return nil, true, err
		}
		if len(c.Args) == 2 {
			end, err := intArg(1)
			if err != nil {
				return nil, true, err
			}
			return call("ql_str_sub", "String", start, end, line)
		}
		// single argument: end = rune count (interpreter semantics), take size first
		rx, err := fc.expr(recv())
		if err != nil {
			return nil, true, err
		}
		sz := &expr{kind: kCall, typ: "int", call: &callExpr{name: "ql_str_size", args: []*expr{rx}}}
		return call("ql_str_sub", "String", start, sz, line)
	case "trim", "trimLeft", "trimRight":
		mode := map[string]int64{"trim": 0, "trimLeft": 1, "trimRight": 2}[me.Name]
		return call("ql_str_trim", "String", &expr{kind: kInt, typ: "int", i: mode})
	case "toLower":
		return call("ql_str_lower", "String")
	case "toUpper":
		return call("ql_str_upper", "String")
	case "replace":
		a, err := strArg(0)
		if err != nil {
			return nil, true, err
		}
		b, err := strArg(1)
		if err != nil {
			return nil, true, err
		}
		return call("ql_str_replace", "String", a, b)
	case "charAt":
		i, err := intArg(0)
		if err != nil {
			return nil, true, err
		}
		return call("ql_str_charat", "String", i, line)
	case "toInt":
		return call("ql_str_toint", "int", line)
	case "toFloat":
		return call("ql_str_tofloat", "float", line)
	}
	return nil, false, nil
}

// ---------- Method table lookup ----------

// hasMethod reports whether a type (base type) declares that method name.
func (l *lowerer) hasMethod(typ, name string) bool {
	base, _ := splitGeneric(typ)
	m, ok := l.methods[base]
	if !ok {
		return false
	}
	_, ok = m[name]
	return ok
}

// lookupSelf looks up an instance method (with a self first parameter) on the **base type** declaration, un-monomorphized.
func (l *lowerer) lookupSelf(typ, name string) *methodInfo {
	base, _ := splitGeneric(typ)
	if m, ok := l.methods[base]; ok {
		if mi, ok := m[name]; ok && mi.isSelf {
			return mi
		}
	}
	return nil
}

// lookupStatic looks up a static method's base type declaration, un-monomorphized.
func (l *lowerer) lookupStatic(scope, name string) *methodInfo {
	if m, ok := l.methods[scope]; ok {
		if mi, ok := m[name]; ok && !mi.isSelf {
			return mi
		}
	}
	return nil
}

// inferStaticSubst infers impl type parameters from a static method call's arguments.
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

// staticMethodFor looks up a static method and completes monomorphization (T::m / space::f).
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

// methodRetType is the method's return type under the receiver type (without instantiating the body; used for type inference).
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

// callGeneric / lowerGeneric are implemented in lower_generic.go (generic monomorphization).

// ---------- Static type inference ----------

// assignable reports whether from is assignable to to (aligned with internal/lang/typecheck.assignable).
func (fc *funcCtx) assignable(from, to string) bool {
	if from == to || from == "?" || to == "?" || to == "" {
		return true
	}
	if from == "int" && to == "float" {
		return true
	}
	if from == "int" && to == "long" {
		return true
	}
	// Raw bits and uchar (spec §3.4): only raw bits of the very same width are assignable, and a view
	// reaches them through the conversion call T(x). uchar takes any integer view by value conversion.
	if isBitsT(to) {
		return from == to
	}
	if isBitsT(from) {
		return false
	}
	if to == "uchar" {
		return isIntViewT(from)
	}
	if from == "uchar" {
		return isIntViewT(to)
	}
	// T& / pointer T: nullable reference (can only be assigned null or a reference of the same base type; not interchangeable with value types)
	if _, base, isRef := ptrRefBase(to); isRef {
		if from == "null" {
			return true
		}
		fb, fbase, isFromRef := ptrRefBase(from)
		_ = fb
		return isFromRef && fbase == base
	}
	if _, _, isFromRef := ptrRefBase(from); isFromRef {
		return false // interpreter: assigning int& to int / using it in arithmetic is a runtime error
	}
	// interface{} (tAny): any value can be boxed; unboxing only supports scalar/String (the kind is checked at runtime)
	if to == "interface{}" {
		return from != "void"
	}
	if from == "interface{}" {
		switch to {
		case "int", "float", "bool", "String", "interface{}":
			return true
		}
		return false
	}
	if from == "null" && (to == "pointer" || to == "String" || fc.l.isIfaceType(to)) {
		return true
	}
	if from == "pointer" && to == "pointer" {
		return true
	}
	// long → int: an interpreter int variable keeps 64 bits (only arithmetic/comparison truncate to 32 bits),
	// so the compiler does not truncate silently and requires an explicit long variable.
	if from == "long" && to == "int" {
		return false
	}
	// interface: concrete struct / interface → interface (structural satisfaction is checked by typecheck at assignment/argument passing)
	if fc.l.isIfaceType(to) && (fc.l.isStructType(from) || fc.l.isIfaceType(from)) {
		return true
	}
	// HashTable: key/value assignable (assignment between tables of different concrete types is guarded by typecheck)
	if isTableT(from) && isTableT(to) {
		return true
	}
	// null is legal only in pointer/reference positions (the compiler does not support the null literal yet)
	return false
}

func numLike(t string) bool {
	return t == "int" || t == "float" || t == "long" || t == "uchar" || t == "?"
}

// isIntViewT reports whether a type name is an integer view stored as an integer (int, long, char, uchar).
func isIntViewT(t string) bool {
	switch t {
	case "int", "long", "char", "uchar":
		return true
	}
	return false
}

// isIntConstLit reports whether an expression is an integer constant (a literal, possibly negated).
func isIntConstLit(e lang.Expr) bool {
	switch v := e.(type) {
	case *lang.IntLit:
		return true
	case *lang.UnOp:
		return v.Op == "-" && isIntConstLit(v.X)
	}
	return false
}

// isPtrType reports whether the type is an FFI opaque pointer / null (i8* reference comparison).
func isPtrType(t string) bool { return t == "pointer" || t == "null" }

// numResult returns the result type of a numeric operation (float if either side is float).
func numResult(a, b string) string {
	if a == "float" || b == "float" {
		return "float"
	}
	if a == "?" || b == "?" {
		return "?"
	}
	// long in arithmetic: the interpreter's wrapI32 (32-bit wraparound), the result is treated as int
	return "int"
}

// typeOf infers the language type of an expression as well as it can; returns "?" when undecidable.
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
	case *lang.ListLit:
		// a list literal takes its expected type when there is one (List<int> vs List<String>),
		// otherwise it is typed by its first element
		if _, ok := listElem(fc.expectT); ok {
			return fc.expectT
		}
		for _, it := range e.Items {
			if fc.typeOf(it) == "String" {
				return "List<String>"
			}
		}
		return "List<int>"
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
	case *lang.NewExpr:
		return e.Typ + "&"
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
		if e.Bits > 0 {
			return "bit" // b[i] on raw bits reads one bit (spec §3.3)
		}
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
			if e.Bits > 0 {
				return rawNameOf(e.Bits) // raw bits keep the left operand's width
			}
			return "int"
		case "&", "|", "^":
			if e.Bits > 0 {
				return rawNameOf(e.Bits)
			}
			if numLike(lt) && numLike(rt) {
				return numResult(lt, rt)
			}
			return "?"
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||":
			return "bool"
		}
		return "?"
	case *lang.UnOp:
		switch e.Op {
		case "*":
			if elem, ok := listElem(fc.typeOf(e.X)); ok {
				return elem
			}
			return "?"
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
		case "~":
			if e.Bits > 0 {
				return rawNameOf(e.Bits) // raw bits keep their width, an int view is complemented in 32 bits
			}
			return "int"
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

// callType infers the return type of a call expression.
func (fc *funcCtx) callType(e *lang.CallExpr) string {
	l := fc.l
	if e.ConvDst != "" && len(e.Args) == 1 {
		return e.ConvDst // conversion call T(x): the type checker stamped the destination
	}
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
		if isTableT(rt) {
			_, valT := tableKinds(rt)
			switch me.Name {
			case "size":
				return "int"
			case "contains":
				return "bool"
			case "keys":
				return "List<String>"
			case "get":
				if valT == "void" || valT == "" {
					return "interface{}"
				}
				return valT
			case "put", "remove":
				return "void"
			}
		}
		if elem, ok := listElem(rt); ok && elem == "String" {
			switch me.Name {
			case "size":
				return "int"
			case "get":
				return "String"
			case "toString":
				return "String"
			}
		}
		if elem, ok := listElem(rt); ok && elem == "int" {
			switch me.Name {
			case "size", "get", "head", "tail", "next":
				return "int"
			case "append", "appendAll", "__sort__":
				return "void"
			case "reset":
				return rt
			case "toString":
				return "String"
			}
		}
		if rt == "String" {
			switch me.Name {
			case "size", "indexOf", "toInt":
				return "int"
			case "contains", "startsWith", "endsWith":
				return "bool"
			case "substring", "trim", "trimLeft", "trimRight", "toLower", "toUpper", "replace", "charAt":
				return "String"
			case "toFloat":
				return "float"
			case "split":
				return "List<String>"
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
	if _, isOvl := l.ovl[id.Name]; isOvl {
		_, fd, err := fc.resolveFunc(id.Name, e.Args, e.Pos)
		if err != nil {
			return "?"
		}
		return retOf(fd)
	}
	if ret, ok := l.fnRet[id.Name]; ok {
		return ret
	}
	return "?"
}

// inferGenSubst infers a generic function's type parameters from the arguments (the same rule as typecheck.inferGenSubst).
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

// identifierName takes the identifier name of a call target (for diagnostics).
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

// ffiLangType maps a C type in a library declaration to a cgen-internal type:
// int/char → int; bool → cbool (C int); float/double → double; f32 → f32; String → String.
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
		return "pointer", nil // opaque handle (void*): i8*, nullable and round-trippable
	}
	if t == "long" {
		return "long", nil // 64-bit integer channel (i64)
	}
	return "", l.errf(pos, "暂未支持 library FFI 的类型 %q", t)
}

// libRetType returns the type a library call exposes at the language level.
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
		case "f32", "double":
			return "float"
		case "void":
			return "void"
		case "long":
			return "long"
		case "pointer":
			return "pointer"
		}
		return t
	}
	return "?"
}

// libCall lowers a library FFI call m.f(args) (LLVM declare + direct C ABI call).
func (fc *funcCtx) libCall(c *lang.CallExpr, me *lang.MemberExpr, libName string) (*expr, error) {
	l := fc.l
	ld := l.libs[libName]
	var fn *lang.FuncDecl
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
		// argument assignability: int → double/f32, bool → cbool, float → f32 are allowed
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
	case "long":
		vis = "long"
	case "pointer":
		vis = "pointer"
	}
	if vis == "void" {
		return &expr{kind: kCall, typ: "void", call: &callExpr{name: fn.Name, args: args}}, nil
	}
	return &expr{kind: kCall, typ: vis, line: me.Pos.Line, call: &callExpr{name: fn.Name, args: args}}, nil
}

// ffiArgOK reports whether an FFI parameter type can be lowered.
func (fc *funcCtx) ffiArgOK(t string) bool {
	switch t {
	case "int", "cbool", "f32", "double", "long", "pointer", "String":
		return true
	}
	return false
}

// ffiAssignable reports whether a language-typed argument can be passed to an FFI parameter.
func (fc *funcCtx) ffiAssignable(from, to string) bool {
	if from == "?" {
		return true
	}
	switch to {
	case "int":
		// long → int parameter: the positional argument is passed as a C int (libffi also truncates to 32 bits)
		return from == "int" || from == "long"
	case "long":
		return from == "int" || from == "long"
	case "pointer":
		return from == "pointer" || from == "null" || from == "?"
	case "cbool":
		return from == "bool" || from == "int"
	case "f32", "double":
		return from == "int" || from == "float" || from == "double" || from == "f32"
	case "String":
		return from == "String" || from == "null"
	}
	return false
}

// addExtern registers an external symbol (deduplicated).
func (l *lowerer) addExtern(name, lib string, params []funcParam, ret string) {
	if _, ok := l.extSeen[name]; ok {
		return
	}
	l.extSeen[name] = true
	l.out.externs = append(l.out.externs, externDef{name: name, params: params, ret: ret, lib: lib})
}

// ---------- catch variable usage detection ----------

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
