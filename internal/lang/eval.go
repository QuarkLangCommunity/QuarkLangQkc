package lang

import (
	"bufio"
	"bytes"
	"container/heap"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// RunError is a runtime (or strict-check) error with source position and,
// when available, the execCtx whose log explains what happened.
type RunError struct {
	Msg string
	Pos Pos
	Ctx *execCtx
}

func (e *RunError) Error() string {
	return fmt.Sprintf("%s at line %d", e.Msg, e.Pos.Line)
}

// errReturn is an internal control-flow signal, never shown to the user.
var errReturn = errors.New("__return__")

type scope struct {
	vars       map[string]Value
	slots      []Value // 参数槽位（前 len(paramNames) 个为参数，线性访问免哈希）
	paramNames []string
	nParams    int // 参数个数：paramNames 中前 nParams 个是参数（declare 不覆盖），其后为局部变量
	outer      *scope
}

// newScope 懒分配 vars（nil map 直到首个 declare —— 高计算场景省空 map 分配）。
func newScope(outer *scope) *scope {
	return &scope{outer: outer}
}

// setParams 绑定函数参数到线性槽位（参数通常少，线性扫描比 map 哈希快）。
func (s *scope) setParams(names []string, args []Value) {
	s.paramNames = names
	s.nParams = len(names)
	s.slots = args
}

// paramIndex 线性查找参数槽位索引（-1 表示非参数）。
func (s *scope) paramIndex(name string) int {
	for i, n := range s.paramNames {
		if n == name {
			return i
		}
	}
	return -1
}

func (s *scope) declare(name string, v Value, pos Pos) error {
	// 参数槽位已存在（execute 绑定）→ 保留绑定值；若是局部变量重复声明（循环内每轮重新声明）
	// → 更新槽位（否则旧值永久残留，如循环中 p int = html.indexOf(...) 拿到首轮旧下标）
	if i := s.paramIndex(name); i >= 0 {
		if i < s.nParams {
			return nil
		}
		s.slots[i] = v
		return nil
	}
	for _, n := range s.paramNames {
		if n == name {
			return &RunError{Msg: fmt.Sprintf("CompileError: duplicate declaration of %q", name), Pos: pos}
		}
	}
	s.paramNames = append(s.paramNames, name)
	s.slots = append(s.slots, v)
	return nil
}

func (s *scope) set(name string, v Value, pos Pos) error {
	for sc := s; sc != nil; sc = sc.outer {
		if pn := sc.paramNames; len(pn) > 0 && pn[0] == name {
			sc.slots[0] = v
			return nil
		}
		if i := sc.paramIndex(name); i >= 0 {
			sc.slots[i] = v
			return nil
		}
		if sc.vars == nil {
			continue
		}
		if _, ok := sc.vars[name]; ok {
			sc.vars[name] = v
			return nil
		}
	}
	return &RunError{Msg: fmt.Sprintf("CompileError: assignment to undeclared identifier %q", name), Pos: pos}
}

func (s *scope) get(name string, pos Pos) (Value, error) {
	for sc := s; sc != nil; sc = sc.outer {
		if pn := sc.paramNames; len(pn) > 0 && pn[0] == name {
			return sc.slots[0], nil
		}
		if i := sc.paramIndex(name); i >= 0 {
			return sc.slots[i], nil
		}
		if sc.vars == nil {
			continue
		}
		if v, ok := sc.vars[name]; ok {
			return v, nil
		}
	}
	return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: undeclared identifier %q", name), Pos: pos}
}

// signDef is a registered signature: name -> call implementation.
type signDef struct {
	name string
	call func(prefix Value, ctx *execCtx) (Value, error)
}

type builtinFn func(args []Value, pos Pos, ctx *execCtx) (Value, error)

// MemBlock 是全局内存管理器的分配区块（spec §14.1）。
type MemBlock struct {
	ID          int
	Size        int
	Used        int  // 已用空间（占用度 = Used/Size，<1 表示内部有空闲）
	Dirty       bool // 区块被更改会记录（脏标记）
	OwnerPID    int  // 所属协程（0 = 全局）
	Reclaimable bool // 无人占用，可回收
}

// memHeap 按占用度（Used 升序）维护 block 的最小堆：占用度最小者优先。
type memHeap []*MemBlock

func (h memHeap) Len() int            { return len(h) }
func (h memHeap) Less(i, j int) bool  { return h[i].Used < h[j].Used }
func (h memHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *memHeap) Push(x interface{}) { *h = append(*h, x.(*MemBlock)) }
func (h *memHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// MemoryManager 管理 block 分配/脏标记/回收。
// 线性分配：优先找占用程度最小（占用度 <1）的 block，进其内部空闲空间；
// 没有则申请新 block。高并发高计算场景效率最优（xmind §内存）。
type MemoryManager struct {
	mu          sync.Mutex
	nextID      int
	blocks      map[int]*MemBlock
	minHeap     memHeap // 按占用度（Used/Size）升序的最小堆
	AllocCalls  int64   // 分配调用总数
	ReusedCount int64   // 复用次数（命中空闲 block 内部空间）
	NewBlocks   int64   // 新申请 block 数
}

func NewMemoryManager() *MemoryManager {
	return &MemoryManager{blocks: map[int]*MemBlock{}}
}

// Alloc 线性分配：占用度最小且未满的 block 优先；否则申请新 block。
func (m *MemoryManager) Alloc(size, owner int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AllocCalls++
	// 优先复用：占用度最小的 block，内部有空闲（Used+size <= BlockSize）就进
	for len(m.minHeap) > 0 {
		b := m.minHeap[0]
		if b.Used+size <= b.Size {
			b.Used += size
			b.Dirty = true
			b.OwnerPID = owner
			b.Reclaimable = false
			heap.Fix(&m.minHeap, 0)
			m.ReusedCount++
			return b.ID
		}
		// 满了，pop 掉（不满足分配）
		heap.Pop(&m.minHeap)
	}
	// 新 block：固定 BlockSize，内部按分配细分（占用度 = 内部已用/BlockSize）
	m.nextID++
	b := &MemBlock{ID: m.nextID, Size: globalMemory.BlockSize, Used: size, Dirty: true, OwnerPID: owner}
	m.blocks[m.nextID] = b
	heap.Push(&m.minHeap, b)
	m.NewBlocks++
	return m.nextID
}

func (m *MemoryManager) MarkDirty(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.blocks[id]; ok {
		b.Dirty = true
	}
}

// ReclaimTask 协程结束后清空其 block 并重入最小占用堆（线性复用）。
func (m *MemoryManager) ReclaimTask(pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.blocks {
		if b.OwnerPID == pid && b.Used > 0 {
			b.Used = 0
			b.OwnerPID = 0
			heap.Push(&m.minHeap, b)
		}
	}
}

// Compact 清理无人占用的 block（Used==0），返回回收数（语言层面不返回）。
func (m *MemoryManager) Compact() (reclaimed int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, b := range m.blocks {
		if b.Used == 0 {
			delete(m.blocks, id)
			reclaimed++
		}
	}
	m.minHeap = nil
	return reclaimed
}

// Fragmentation 返回碎片率：只统计存活（Used>0）block 的内部未用空间占比。
// block 粒度满用或全空（无内部分裂）→ 潮汐/复用场景碎片率恒为 0，优于 malloc 式碎片。
func (m *MemoryManager) Fragmentation() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	var used, size int
	for _, b := range m.blocks {
		if b.Used > 0 {
			used += b.Used
			size += b.Size
		}
	}
	if size == 0 {
		return 0
	}
	return 1 - float64(used)/float64(size)
}

// BlockCount 返回当前 block 总数（测试可观测）。
func (m *MemoryManager) BlockCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.blocks)
}

// Delete 把 block 加入空闲队列（占用度归零，数据保留），供线性复用；clear 才真正清空。
func (m *MemoryManager) Delete(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.blocks[id]; ok {
		b.Used = 0
		b.OwnerPID = 0
		heap.Push(&m.minHeap, b)
	}
}

// Clear 真正清空空闲（Used==0）block 的数据，保障数据安全；使用中的 block 保留。
func (m *MemoryManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, b := range m.blocks {
		if b.Used == 0 {
			delete(m.blocks, id)
		}
	}
	m.minHeap = nil
	for _, b := range m.blocks {
		heap.Push(&m.minHeap, b)
	}
}

// StructDef is a registered struct declaration.
type StructDef struct {
	Name       string
	TypeParams []string
	Types      map[string]string // 成员名 → 类型注解
	TypesOrder []string          // 字段声明顺序（位置 .{} 绑定用）
}

// InterfaceDef is a registered interface declaration.
type InterfaceDef struct {
	Name    string
	Methods []MethodSig
	Expands []string // expand interface 组合接口
	Partial bool     // 可选实现接口（事件族）：缺少方法不报错（emit 运行时忽略）
}

// ImplDef is a registered impl block: Methods are static (no self), SelfMethods
// take self as first parameter.
type ImplDef struct {
	Type        string
	Iface       string
	TypeParams  []string
	Methods     map[string]*Func
	SelfMethods map[string]*Func
}

type interp struct {
	ctxHead     atomic.Pointer[execCtx]
	fns         map[string]*Func
	overloads   map[string][]*Func
	libObjs     map[string]*libObj // library 系统库绑定对象（懒加载句柄）
	dbg         *dbgState          // --debug 断点状态（nil = 零开销）
	sigs        map[string]*signDef
	builtins    map[string]builtinFn
	structs     map[string]*StructDef
	interfaces  map[string]*InterfaceDef
	impls       map[string]*ImplDef
	tasks       map[int]*Task
	taskMu      sync.Mutex
	nextPid     int
	mem         *MemoryManager
	randState   uint64  // rand() LCG 状态（确定性伪随机）
	globalScope *scope  // 全局作用域（常量预声明一次）
	fnList      []*Func // 函数表（CallExpr.FnIdx 直取，免 map）
}

// Run executes prog. args are command-line arguments for main(); stdin/stdout
// are the default console streams for io.
// Run executes prog; see runWithInterp.
func Run(prog *Program, filename string, args []string, stdin io.Reader, stdout io.Writer) error {
	_, err := runWithInterp(prog, filename, args, stdin, stdout)
	return err
}

// runWithInterp 执行 prog 并返回解释器（测试可观测内存管理器等内部状态）。
func runWithInterp(prog *Program, filename string, args []string, stdin io.Reader, stdout io.Writer) (*interp, error) {
	if err := Typecheck(prog); err != nil {
		return nil, err
	}
	in := &interp{
		fns:        map[string]*Func{},
		sigs:       map[string]*signDef{},
		builtins:   map[string]builtinFn{},
		structs:    map[string]*StructDef{},
		interfaces: map[string]*InterfaceDef{},
		impls:      map[string]*ImplDef{},
		tasks:      map[int]*Task{},
		mem:        NewMemoryManager(),
	}
	in.globalScope = newScope(nil)
	_ = in.globalScope.declare("DynamicStackAndHeap", StrV("DynamicStackAndHeap"), Pos{})
	for _, f := range prog.Funcs {
		nfn := &Func{Name: f.Name, Params: f.Params, Ret: f.Ret, Body: f.Body, Pos: f.Pos}
		if _, dup := in.fns[f.Name]; dup {
			if !sameSig(in.fns[f.Name], nfn) {
				if in.overloads == nil {
					in.overloads = map[string][]*Func{}
				}
				in.overloads[f.Name] = append(in.overloads[f.Name], nfn)
			} else {
				return nil, fmt.Errorf("CompileError: duplicate overload %q", f.Name)
			}
		} else {
			in.fns[f.Name] = nfn
		}
	}
	// 函数表（FnIdx 索引，顺序与 FnList 一致——fns 填充后）
	for _, fd := range prog.FnList {
		in.fnList = append(in.fnList, in.fns[fd.Name])
	}
	for _, s := range prog.Structs {
		if _, dup := in.structs[s.Name]; dup {
			return nil, fmt.Errorf("CompileError: duplicate struct %q", s.Name)
		}
		def := &StructDef{Name: s.Name, Types: map[string]string{}}
		for _, m := range s.Members {
			def.Types[m.Name] = m.Type
			def.TypesOrder = append(def.TypesOrder, m.Name)
		}
		in.structs[s.Name] = def
	}
	registerBuiltinIfaces(in.interfaces)
	for _, i := range prog.Interfaces {
		if _, dup := in.interfaces[i.Name]; dup {
			return nil, fmt.Errorf("CompileError: duplicate interface %q", i.Name)
		}
		in.interfaces[i.Name] = &InterfaceDef{Name: i.Name, Methods: i.Methods, Expands: i.Expands}
	}
	for _, im := range prog.Impls {
		// 同一类型允许多个 impl 块：方法聚合（xmind §类：impl<T> {...} name;）
		key := implKeyOf(im.Type, im.Iface)
		def, exists := in.impls[key]
		if !exists {
			def = &ImplDef{Type: im.Type, Iface: im.Iface, TypeParams: im.TypeParams, Methods: map[string]*Func{}, SelfMethods: map[string]*Func{}}
			in.impls[key] = def
		}
		for _, m := range im.Methods {
			if len(m.Params) > 0 && m.Params[0].Name == "self" && m.Params[0].Type == "" {
				m.Params[0].Type = im.Type
			}
			fn := &Func{Name: m.Name, Params: m.Params, Ret: m.Ret, Body: m.Body, Pos: m.Pos}
			if len(m.Params) > 0 && m.Params[0].Name == "self" {
				if _, dup := def.SelfMethods[fn.Name]; dup {
					return nil, fmt.Errorf("CompileError: duplicate method %q on %s", fn.Name, im.Type)
				}
				def.SelfMethods[fn.Name] = fn
			} else {
				if _, dup := def.Methods[fn.Name]; dup {
					return nil, fmt.Errorf("CompileError: duplicate method %q on %s", fn.Name, im.Type)
				}
				def.Methods[fn.Name] = fn
			}
		}
	}
	in.registerIOBuiltins()
	for _, lb := range prog.Libraries {
		if in.libObjs == nil {
			in.libObjs = map[string]*libObj{}
		}
		obj := &libObj{name: lb.Name, lib: lb.Lib, methods: map[string]*Func{}}
		for _, fn := range lb.Methods {
			obj.methods[fn.Name] = fn
		}
		in.libObjs[lb.Name] = obj
	}

	if prog.Kind == "library" {
		return nil, fmt.Errorf("RunError: #error (\"cannot run a library\"): program library; 编译为库，不可运行")
	}
	mainFn, ok := in.fns["main"]
	if !ok {
		return nil, fmt.Errorf("CompileError: no main function found (expected: fn main(io IOStream, ...))")
	}
	ioObj := &IOStream{In: stdin, Out: stdout, rd: bufio.NewReader(stdin)}
	env := envTable()
	argList := NewList()
	for _, a := range args {
		argList.Append(StrV(a))
	}
	// main's injected params, in fixed order: io, env, args (spec §8).
	var mainArgs []Value
	switch len(mainFn.Params) {
	case 1:
		mainArgs = []Value{IOV(ioObj)}
	case 2:
		mainArgs = []Value{IOV(ioObj), TableV(env)}
	case 3:
		mainArgs = []Value{IOV(ioObj), TableV(env), ListV(argList)}
	default:
		return nil, fmt.Errorf("CompileError: main must take 1-3 params in order (io IOStream, env HashTable<String,String>, args List<String>), got %d", len(mainFn.Params))
	}
	ctx := in.newCtx(mainFn, mainArgs, mainFn.Pos)
	if pendingDebug != nil {
		pendingDebug(in)
		pendingDebug = nil
	}
	return in, in.execute(ctx)
}

// zeroInstance 构造结构体零值实例（成员按类型注解取零值）。
func (in *interp) zeroInstance(def *StructDef) *StructValue {
	sv := &StructValue{SType: def.Name, Fields: map[string]Value{}}
	for name, typ := range def.Types {
		sv.Fields[name] = in.zeroValue(typ)
	}
	return sv
}

// baseTypeName 取泛型实例类型注解的基名（node<int> → node）。
func baseTypeName(typ string) string {
	if i := strings.Index(typ, "<"); i >= 0 {
		return typ[:i]
	}
	return typ
}

// zeroValue 按类型注解生成零值。
func (in *interp) zeroValue(typ string) Value {
	if strings.HasSuffix(typ, "&") {
		return NilV() // 指针零值 = null
	}
	if d, ok := in.structs[baseTypeName(typ)]; ok {
		return StructV(in.zeroInstance(d))
	}
	switch baseTypeName(typ) {
	case "int", "long", "char":
		return IntV(0)
	case "float", "double":
		return FloatV(0)
	case "String":
		return StrV("")
	case "bool":
		return BoolV(false)
	}
	if strings.Contains(typ, "List") || strings.Contains(typ, "Array") {
		return ListV(NewList())
	}
	if strings.Contains(typ, "HashTable") {
		return TableV(NewHashTable())
	}
	return NilV()
}

func envTable() *HashTable {
	h := NewHashTable()
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			h.m[hashKey(StrV(parts[0]))] = StrV(parts[1])
		}
	}
	return h
}

// execute runs a execCtx's function body, filling Tail and Log.
func (in *interp) execute(ctx *execCtx) error {
	if ctx.executed {
		return &RunError{Msg: fmt.Sprintf("RuntimeError: execCtx for %s already executed", ctx.Fn.Name), Pos: ctx.pos, Ctx: ctx}
	}
	ctx.executed = true

	sc := &ctx.sc             // 复用 ctx 内嵌作用域（免每次调用堆分配）
	sc.outer = in.globalScope // 全局作用域（DynamicStackAndHeap 等预声明一次，outer 链可见）
	fn := ctx.Fn
	// 参数绑定到线性槽位：参数名缓存共享（零分配），值直接复用 ctx.Args
	args := ctx.Args
	if flags := fn.CopydFlags(); flags != nil {
		for i, f := range flags {
			if f {
				args[i] = deepCopy(args[i])
			}
		}
	}
	sc.setParams(fn.ParamNames(), args)
	if err := in.execBlock(fn.Body, sc, ctx); err != nil {
		if err == errReturn {
			return nil
		}
		return err
	}
	return nil
}

func (in *interp) execBlock(b *Block, sc *scope, ctx *execCtx) error {
	for _, st := range b.Stmts {
		if err := in.execStmt(st, sc, ctx); err != nil {
			return err
		}
	}
	return nil
}

// qkstyleLookup：QSS 键读取 + 旧别名回退（background-color←bg、font-size←size、font-family←font）——style 表口归一。
func qkstyleLookup(t *HashTable, key Value) (Value, bool) {
	if v, ok := t.Get(key); ok {
		return v, true
	}
	if key.IsStr() {
		var alias string
		switch key.Str() {
		case "background-color":
			alias = "bg"
		case "font-size":
			alias = "size"
		case "font-family":
			alias = "font"
		}
		if alias != "" {
			return t.Get(StrV(alias))
		}
	}
	return NilV(), false
}

// errLoopBreak 哨兵：break 跳出循环（while/for 捕获；顶层冒泡报 break outside loop）。
var errLoopBreak = errors.New("loop break")

func (in *interp) execStmt(st Stmt, sc *scope, ctx *execCtx) error {
	// 调试模式：断点检查（无 dbg = nil 单判定，零开销）
	if in.dbg != nil {
		in.hitBreak(stPos(st), sc)
	}
	switch s := st.(type) {
	case *ExprStmt:
		_, err := in.evalExpr(s.X, sc, ctx)
		return err
	case *DeleteStmt:
		// delete：先执行 __delete__()（如果有），再加入空闲队列（数据保留，clear 才清空）
		v, err := in.evalExpr(s.X, sc, ctx)
		if err != nil {
			return err
		}
		if v.IsStruct() {
			sv := v.Struct()
			if def, has := in.impls[sv.SType]; has {
				if fn, ok := def.SelfMethods["__delete__"]; ok {
					if _, err := in.callFunc(fn, []Value{StructV(sv)}, s.Pos, ctx.depth); err != nil {
						return err
					}
				}
			}
		}
		if v.IsList() {
			l := v.List()
			// List 加入空闲队列（无 block 时先分配以记录）
			id := l.blockID
			if id == 0 {
				id = in.mem.Alloc(globalMemory.BlockSize, 0)
			}
			in.mem.Delete(id)
		}
		return nil
	case *LogStmt:
		// log 记录日志并结束函数（返回任意值，默认 nil）
		v, err := in.evalExpr(s.X, sc, ctx)
		if err != nil {
			return err
		}
		ctx.ensureLog().Append(StrV(v.String()))
		ctx.result = NilV()
		return errReturn
	case *TryStmt:
		// try/catch：try 块出错（非 return）时把错误装入 catch 变量（interface{}），执行 catch 块
		err := in.execBlock(s.Try, sc, ctx)
		if err != nil {
			if err == errReturn {
				return err
			}
			inner := newScope(sc)
			_ = inner.declare(s.CatchVar, StrV(err.Error()), s.Pos) // 错误装入声明类型（interface{} 等），自由系统
			if cerr := in.execBlock(s.Catch, inner, ctx); cerr != nil {
				return cerr
			}
		}
		return nil
	case *BreakStmt:
		return errLoopBreak

	case *ReturnStmt:
		if s.X != nil {
			v, err := in.evalExpr(s.X, sc, ctx)
			if err != nil {
				return err
			}
			ctx.result = v
		}
		return errReturn
	case *IfStmt:
		c, err := in.evalExpr(s.Cond, sc, ctx)
		if err != nil {
			return err
		}
		b, err := truthy(c)
		if err != nil {
			return err
		}
		if b {
			return in.execBlock(s.Then, sc, ctx)
		}
		if s.Else != nil {
			return in.execBlock(s.Else, sc, ctx)
		}
		return nil
	case *WhileStmt:
		for {
			c, err := in.evalExpr(s.Cond, sc, ctx)
			if err != nil {
				return err
			}
			b, err := truthy(c)
			if err != nil {
				return err
			}
			if !b {
				return nil
			}
			if err := in.execBlock(s.Body, sc, ctx); err != nil {
				if errors.Is(err, errLoopBreak) {
					return nil
				}
				return err
			}
		}
	case *ForStmt:
		v, err := in.evalExpr(s.Iter, sc, ctx)
		if err != nil {
			return err
		}
		if !v.IsList() {
			return &RunError{Msg: fmt.Sprintf("TypeError: for-in requires a List, got %s", v.TypeName()), Pos: s.Pos, Ctx: ctx}
		}
		l := v.List()
		inner := newScope(sc)
		for l.Head() != l.Tail() {
			item, err := l.Next()
			if err != nil {
				return &RunError{Msg: err.Error(), Pos: s.Pos, Ctx: ctx}
			}
			if inner.vars == nil {
				inner.vars = map[string]Value{}
			}
			inner.vars[s.Var] = item
			if err := in.execBlock(s.Body, inner, ctx); err != nil {
				if errors.Is(err, errLoopBreak) {
					return nil
				}
				return err
			}
		}
		return nil
	case *ForCStmt:
		// C 风格：for (<init>; <cond>; <step>) { ... }
		inner := newScope(sc)
		if s.Init != nil {
			if err := in.execStmt(s.Init, inner, ctx); err != nil {
				return err
			}
		}
		for {
			c, err := in.evalExpr(s.Cond, inner, ctx)
			if err != nil {
				return err
			}
			b, err := truthy(c)
			if err != nil {
				return err
			}
			if !b {
				return nil
			}
			if err := in.execBlock(s.Body, inner, ctx); err != nil {
				if errors.Is(err, errLoopBreak) {
					return nil
				}
				return err
			}
			if s.Step != nil {
				if err := in.execStmt(s.Step, inner, ctx); err != nil {
					return err
				}
			}
		}
	case *DeclStmt:
		var v Value = NilV()
		if s.Init != nil {
			var err error
			v, err = in.evalExpr(s.Init, sc, ctx)
			if err != nil {
				return err
			}
		} else if def, ok := in.structs[baseTypeName(s.Type)]; ok {
			v = StructV(in.zeroInstance(def))
		}
		// copyd 修饰：声明即为 Copyd 值（与类型标注 [Copyd] 一致：传时复制、.ptr() 取包装值）
		if s.Decor == "copyd" && !v.IsCopyd() {
			v = CopydV(&CopydValue{V: deepCopy(v)})
		}
		return sc.declare(s.Name, v, s.Pos)
	case *AssignStmt:
		v, err := in.evalExpr(s.X, sc, ctx)
		if err != nil {
			return err
		}
		switch t := s.Target.(type) {
		case *Ident:
			return sc.set(t.Name, v, t.Pos)
		case *IndexExpr:
			obj, err := in.evalExpr(t.X, sc, ctx)
			if err != nil {
				return err
			}
			if obj.IsTable() {
				key, err := in.evalExpr(t.Idx, sc, ctx)
				if err != nil {
					return err
				}
				obj.Table().Put(key, v)
				return nil
			}
			if obj.IsList() {
				key, err := in.evalExpr(t.Idx, sc, ctx)
				if err != nil {
					return err
				}
				if !key.IsInt() {
					return &RunError{Msg: "TypeError: 列表索引必须是 int", Pos: s.Pos, Ctx: ctx}
				}
				return obj.List().setIndex(int(key.Int()), v)
			}
			return &RunError{Msg: fmt.Sprintf("TypeError: 不支持对 %s 索引赋值", obj.TypeName()), Pos: s.Pos, Ctx: ctx}
		case *MemberExpr:
			obj, err := in.evalExpr(t.X, sc, ctx)
			if err != nil {
				return err
			}
			if obj.IsNil() {
				return &RunError{Msg: "NullPointerError: assignment through null pointer", Pos: s.Pos, Ctx: ctx}
			}
			if obj.IsCopyd() {
				c := obj.Copyd()
				obj = c.V
			}
			if !obj.IsStruct() {
				return &RunError{Msg: fmt.Sprintf("TypeError: cannot assign member of %s", obj.TypeName()), Pos: s.Pos, Ctx: ctx}
			}
			sv := obj.Struct()
			if _, exists := sv.Fields[t.Name]; !exists {
				return &RunError{Msg: fmt.Sprintf("TypeError: no member %q on %s", t.Name, sv.SType), Pos: t.Pos, Ctx: ctx}
			}
			sv.Fields[t.Name] = v
			return nil
		}
		return &RunError{Msg: "TypeError: unsupported assignment target", Pos: s.Pos, Ctx: ctx}
	}
	return nil
}

func (in *interp) evalExpr(e Expr, sc *scope, ctx *execCtx) (Value, error) {
	switch x := e.(type) {
	case *IntLit:
		return IntV(x.V), nil
	case *FloatLit:
		return FloatV(x.V), nil
	case *StrLit:
		return StrV(x.V), nil
	case *BoolLit:
		return BoolV(x.V), nil
	case *NullLit:
		return NilV(), nil
	case *NewExpr:
		// new <type>[size]：堆上申请（block 分配）；size 非法 → badAlloc
		size := 1
		if x.Size != nil {
			sv, err := in.evalExpr(x.Size, sc, ctx)
			if err != nil {
				return NilV(), err
			}
			if !sv.IsInt() || sv.Int() < 0 || sv.Int() > 1<<23 {
				ctx.ensureLog().Append(StrV("badAlloc: invalid size"))
				return NilV(), &RunError{Msg: "badAlloc: new " + x.Typ + " 申请大小非法", Pos: x.Pos, Ctx: ctx}
			}
			size = int(sv.Int())
		}
		// 在堆（block 管理）上申请
		id := in.mem.Alloc(size*8, 0)
		l := NewList()
		l.mem = in.mem
		l.blockID = id
		return ListV(l), nil
	case *StructLit:
		st := x.Name
		if st == "" {
			st = "."
		}
		sv := &StructValue{SType: st, Fields: map[string]Value{}}
		// 位置形式（Name 空）：按目标类型字段顺序绑定（与编译器一致）
		var fieldNames []string
		if sd, ok := in.structs[st]; ok {
			fieldNames = sd.TypesOrder
		}
		idx := 0
		for _, f := range x.Fields {
			v, err := in.evalExpr(f.X, sc, ctx)
			if err != nil {
				return NilV(), err
			}
			name := f.Name
			if name == "" && idx < len(fieldNames) {
				name = fieldNames[idx]
			}
			idx++
			sv.Fields[name] = v
		}
		return StructV(sv), nil
	case *Ident:
		if x.Name == "true" {
			return BoolV(true), nil
		}
		if x.Name == "false" {
			return BoolV(false), nil
		}
		if in.libObjs != nil {
			if lb, ok := in.libObjs[x.Name]; ok {
				return LibraryV(lb), nil
			}
		}
		if x.Name == "memory" || x.Name == "GlobalMemory" {
			return MemoryV(globalMemory), nil
		}
		if x.Name == "taskm" {
			return TaskmV(globalTaskm), nil
		}
		v, err := sc.get(x.Name, x.Pos)
		if err == nil {
			return v, nil
		}
		if fn, ok := in.fns[x.Name]; ok {
			return FuncV(&FuncValue{fn: fn}), nil
		}
		// 内置函数作为函数引用（sum(rand, ...)）
		if isBuiltinFuncName(x.Name) {
			return FuncV(&FuncValue{fn: &Func{Name: x.Name, Params: []Param{}, Ret: "int"}}), nil
		}
		return NilV(), err
	case *ListLit:
		l := NewList()
		for _, it := range x.Items {
			v, err := in.evalExpr(it, sc, ctx)
			if err != nil {
				return NilV(), err
			}
			l.Append(v)
		}
		return ListV(l), nil
	case *UnOp:
		v, err := in.evalExpr(x.X, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		switch x.Op {
		case "*":
			if !v.IsList() {
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: '*' requires a List, got %s", v.TypeName()), Pos: x.Pos, Ctx: ctx}
			}
			l := v.List()
			item, err := l.Peek()
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
			}
			return item, nil
		case "-":
			if v.IsInt() {
				return wrapI32(-v.Int()), nil
			}
			if v.IsFloat() {
				return FloatV(-v.Float()), nil
			}
			if v.IsStruct() {
				if fn := in.selfMethodOf(v.Struct().SType, "__neg__"); fn != nil {
					return in.callFunc(fn, []Value{v}, x.Pos, ctx.depth)
				}
			}
			return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: unary '-' requires a number, got %s", v.TypeName()), Pos: x.Pos, Ctx: ctx}
		case "!":
			b, err := truthy(v)
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
			}
			return BoolV(!b), nil
		}
		return NilV(), &RunError{Msg: "internal: unknown unary operator " + x.Op, Pos: x.Pos, Ctx: ctx}
	case *BinOp:
		l, err := in.evalExpr(x.L, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		// Operation 运算符重载：两侧同 struct 且聚合方法命中 → 调用协议方法
		if l.IsStruct() && x.Op != "&&" && x.Op != "||" {
			if rv, err := in.evalExpr(x.R, sc, ctx); err == nil {
				if rv.IsStruct() && l.Struct().SType == rv.Struct().SType {
					if m := opMethodFor(x.Op); m != "" {
						if fn := in.selfMethodOf(l.Struct().SType, m); fn != nil {
							return in.callFunc(fn, []Value{l, rv}, x.Pos, ctx.depth)
						}
					}
				}
			}
		}
		// 快路径：两侧都是 int 的算术/位移直通（免 binOp 分发，fib/循环类大热）
		if l.IsInt() && x.Op != "&&" && x.Op != "||" {
			li := l.Int()
			if rv, err := in.evalExpr(x.R, sc, ctx); err == nil {
				if rv.IsInt() {
					ri := rv.Int()
					switch x.Op {
					case "+":
						return wrapI32(li + ri), nil
					case "-":
						return wrapI32(int64(li) - int64(ri)), nil
					case "*":
						if n, ok := pow2(int32(ri)); ok {
							return wrapI32(int64(int32(li) << uint(n))), nil
						}
						return wrapI32(int64(li) * int64(ri)), nil
					case "/":
						if ri == 0 {
							return NilV(), &RunError{Msg: "DivisionByZeroError: integer division by zero", Pos: x.Pos, Ctx: ctx}
						}
						return wrapI32(int64(li) / int64(ri)), nil
					case "%":
						if ri == 0 {
							return NilV(), &RunError{Msg: "DivisionByZeroError: modulo by zero", Pos: x.Pos, Ctx: ctx}
						}
						return wrapI32(int64(li) % int64(ri)), nil
					case "<<":
						return IntV(int64(int32(li) << uint(ri&31))), nil
					case ">>":
						return IntV(int64(int32(li) >> uint(ri&31))), nil
					case "<":
						return BoolV(int32(li) < int32(ri)), nil
					case "<=":
						return BoolV(int32(li) <= int32(ri)), nil
					case ">":
						return BoolV(int32(li) > int32(ri)), nil
					case ">=":
						return BoolV(int32(li) >= int32(ri)), nil
					}
				}
				return binOp(x.Op, l, rv, x.Pos, ctx)
			}
		}
		if x.Op == "&&" {
			lb, err := truthy(l)
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
			}
			if !lb {
				return BoolV(false), nil
			}
			r, err := in.evalExpr(x.R, sc, ctx)
			if err != nil {
				return NilV(), err
			}
			rb, err := truthy(r)
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
			}
			return BoolV(rb), nil
		}
		if x.Op == "||" {
			lb, err := truthy(l)
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
			}
			if lb {
				return BoolV(true), nil
			}
			r, err := in.evalExpr(x.R, sc, ctx)
			if err != nil {
				return NilV(), err
			}
			rb, err := truthy(r)
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
			}
			return BoolV(rb), nil
		}
		r, err := in.evalExpr(x.R, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		return binOp(x.Op, l, r, x.Pos, ctx)
	case *CallExpr:
		return in.evalCall(x, sc, ctx)
	case *MemberExpr:
		obj, err := in.evalExpr(x.X, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		return evalMember(obj, x.Name, x.Pos, ctx)
	case *ScopeCall:
		return in.evalScopeCall(x, sc, ctx)
	case *IndexExpr:
		v, err := in.evalExpr(x.X, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		i, err := in.evalExpr(x.Idx, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		if !v.IsList() {
			return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: indexing requires a List, got %s", v.TypeName()), Pos: x.Pos, Ctx: ctx}
		}
		l := v.List()
		if !i.IsInt() {
			return NilV(), &RunError{Msg: "TypeError: index must be int", Pos: x.Pos, Ctx: ctx}
		}
		iv := i.Int()
		item, err := l.Get(int(iv))
		if err != nil {
			return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
		}
		return item, nil
	}
	return NilV(), &RunError{Msg: "internal: unknown expression node", Ctx: ctx}
}

// evalMember reads a plain member (no call) — e.g. execCtx.head/tail/log.
func evalMember(obj Value, name string, pos Pos, ctx *execCtx) (Value, error) {
	if obj.IsNil() {
		return NilV(), &RunError{Msg: "NullPointerError: dereference of null pointer", Pos: pos, Ctx: ctx}
	}
	if obj.IsCopyd() {
		c := obj.Copyd()
		return evalMember(c.V, name, pos, ctx) // Copyd 透传
	}
	if obj.IsStruct() {
		o := obj.Struct()
		if v, exists := o.Fields[name]; exists {
			return v, nil
		}
		return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: no member %q on %s", name, obj.TypeName()), Pos: pos, Ctx: ctx}
	}
	return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: no member %q on %s", name, obj.TypeName()), Pos: pos, Ctx: ctx}
}

func (in *interp) evalCall(c *CallExpr, sc *scope, ctx *execCtx) (Value, error) {
	// 内置签名 @styleConfigure(file)：先读 JSON 文件 → merge 进被调用首参节点的 style，再执行调用
	if c.Sign != nil && c.Sign.Name == "styleConfigure" {
		if len(c.Sign.Args) != 1 {
			return NilV(), &RunError{Msg: "TypeError: @styleConfigure(file String)", Pos: c.Pos, Ctx: ctx}
		}
		fv, err := in.evalExpr(c.Sign.Args[0], sc, ctx)
		if err != nil {
			return NilV(), err
		}
		path := ""
		if fv.IsFile() {
			path = fv.File().Path
		} else if fv.IsStr() {
			path = fv.Str()
		} else {
			return NilV(), &RunError{Msg: "TypeError: @styleConfigure(file) 需要 file 或 String", Pos: c.Pos, Ctx: ctx}
		}
		// @styleConfigure 是给 setStyle 用的：读 file → 文件内容字符串 → 投递给 setStyle(string)
		if mem, ok := c.Fn.(*MemberExpr); ok && mem.Name == "setStyle" && len(c.Args) >= 1 {
			content, rerr := os.ReadFile(path)
			if rerr == nil {
				c.Args[0] = &StrLit{V: string(content), Pos: c.Pos}
			}
			clean := *c
			clean.Sign = nil
			return in.evalExpr(&clean, sc, ctx)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return NilV(), &RunError{Msg: "IOError: " + err.Error(), Pos: c.Pos, Ctx: ctx}
		}
		var js interface{}
		if err := json.Unmarshal(raw, &js); err != nil {
			return NilV(), &RunError{Msg: "JSONError: " + err.Error(), Pos: c.Pos, Ctx: ctx}
		}
		loaded, err := qkjsonFromGo(js)
		if err != nil {
			return NilV(), &RunError{Msg: err.Error(), Pos: c.Pos, Ctx: ctx}
		}
		if loaded.IsTable() {
			// 找被调用首参（成员调用=接收者；普通调用=第一个实参）的 style 字段并 merge
			var nodes []Value
			if mem, ok := c.Fn.(*MemberExpr); ok {
				if rv, err := in.evalExpr(mem.X, sc, ctx); err == nil {
					nodes = append(nodes, rv)
				}
			} else {
				for i, a := range c.Args {
					if i == 0 {
						if av, err := in.evalExpr(a, sc, ctx); err == nil {
							nodes = append(nodes, av)
						}
						break
					}
				}
			}
			for _, n := range nodes {
				if n.IsStruct() {
					for k, v := range n.Struct().Fields {
						if k == "style" && v.IsTable() {
							for kk, vv := range loaded.Table().m {
								v.Table().m[kk] = vv
							}
						}
					}
				}
			}
		}
		clean := *c
		clean.Sign = nil
		return in.evalExpr(&clean, sc, ctx)
	}
	// Signature wrapper: f(args) @sign(prefix) -> sign::call(prefix)(ctx) (spec §6).
	if c.Sign != nil {
		id, ok := c.Fn.(*Ident)
		if !ok {
			return NilV(), &RunError{Msg: "TypeError: a signature can only wrap a direct function call", Pos: c.Pos, Ctx: ctx}
		}
		argVals, err := in.evalArgs(c.Args, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		fn := in.bestMatchV(in.allDefs(id.Name), argVals)
		if fn == nil {
			return NilV(), &RunError{Msg: overloadErr(in.allDefs(id.Name), id.Name, len(argVals)), Pos: id.Pos, Ctx: ctx}
		}
		if len(argVals) != len(fn.Params) {
			return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(argVals)), Pos: id.Pos, Ctx: ctx}
		}
		// v2 签名：f(args) @instance(prefix) ≡ instance.call(prefix)(.{in, out})
		// 1) instance 是变量（Sign 实例，名字任意）；2) prefix 中放被包装的原函数 fn；3) 记录 .{in: List(args), out: nil}
		mbv, err := in.evalExpr(&Ident{Name: c.Sign.Name, Pos: c.Pos}, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		// 构造 prefix：原函数在 prefix 中（外加 @ 处的显式参数）
		prefix := &StructValue{SType: ".", Fields: map[string]Value{"fn": FuncV(&FuncValue{fn: fn})}}
		for _, a := range c.Sign.Args {
			av, err := in.evalExpr(a, sc, ctx)
			if err != nil {
				return NilV(), err
			}
			prefix.Fields["prefix"] = av
		}
		// 记录 .{in, out}
		inList := NewList()
		for _, a := range argVals {
			inList.Append(a)
		}
		rec := &StructValue{SType: ".", Fields: map[string]Value{"in": ListV(inList), "out": NilV()}}
		if _, err := in.callMethod(mbv, "call", []Value{StructV(prefix), StructV(rec)}, ctx, c.Pos); err != nil {
			return NilV(), err
		}
		return rec.Fields["out"], nil
	}

	// Method call: obj.name(args).
	if m, ok := c.Fn.(*MemberExpr); ok {
		obj, err := in.evalExpr(m.X, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		args, err := in.evalArgs(c.Args, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		return in.callMethod(obj, m.Name, args, ctx, m.Pos)
	}

	id, ok := c.Fn.(*Ident)
	if !ok {
		return NilV(), &RunError{Msg: "TypeError: this expression is not callable", Pos: c.Pos, Ctx: ctx}
	}
	argVals, err := in.evalArgs(c.Args, sc, ctx)
	if err != nil {
		return NilV(), err
	}
	// FnIdx 编译期已解析：无重载时直取 fnList（免 map 哈希热路径）
	if c.FnIdx >= 0 && c.FnIdx < len(in.fnList) && len(in.overloads[id.Name]) == 0 {
		return in.callFunc(in.fnList[c.FnIdx], argVals, id.Pos, ctx.depth)
	}
	if defs := in.allDefs(id.Name); len(defs) > 0 {
		fn := in.bestMatchV(defs, argVals)
		if fn == nil {
			return NilV(), &RunError{Msg: overloadErr(in.allDefs(id.Name), id.Name, len(argVals)), Pos: id.Pos, Ctx: ctx}
		}
		return in.callFunc(fn, argVals, id.Pos, ctx.depth)
	}
	// 函数引用变量：f 是变量且值为 FuncValue → 调用
	if v, err := sc.get(id.Name, id.Pos); err == nil {
		if v.IsFunc() {
			fv := v.Func()
			return in.callFunc(fv.fn, argVals, id.Pos, ctx.depth)
		}
	}
	if b, ok := in.builtins[id.Name]; ok {
		return b(argVals, id.Pos, ctx)
	}
	return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: undeclared function %q", id.Name), Pos: id.Pos, Ctx: ctx}
}

// callFunc：v2 —— 函数调用返回 return 的结果（log 结束则返回 nil）。
func (in *interp) callFunc(fn *Func, args []Value, pos Pos, parentDepth int) (Value, error) {
	// 内置函数引用（如 sum 的生成器 rand）：仅当是伪函数（无 Body）时——用户同名方法不被劫持
	if fn.Body == nil {
		if b, ok := in.builtins[fn.Name]; ok {
			return b(args, pos, nil)
		}
	}
	if len(args) != len(fn.Params) {
		return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(args)), Pos: pos}
	}
	if parentDepth >= 8192 {
		return NilV(), &RunError{Msg: "StackOverflowError: recursion depth exceeded 8192", Pos: pos}
	}
	// 传时复制（copyd）：
	//   ① 形参标注 [Copyd] → 实参包装为 Copyd 值（深拷贝一次）
	//   ② 实参本身是 copyd 声明的 Copyd 值、而形参未标注 → 解包并深拷贝（传时复制）
	flags := fn.CopydFlags()
	for i := range args {
		flagged := flags != nil && i < len(flags) && flags[i]
		if args[i].IsCopyd() {
			if !flagged {
				args[i] = deepCopy(args[i].Copyd().V)
			}
			continue
		}
		if flagged {
			args[i] = CopydV(&CopydValue{V: deepCopy(args[i])})
		}
	}
	ctx := in.newCtx(fn, args, pos)
	defer in.putCtx(ctx)
	ctx.depth = parentDepth + 1
	if err := in.execute(ctx); err != nil {
		return NilV(), err
	}
	if ctx.result.IsNil() {
		return NilV(), nil // 未 return 的路径（log 结束等）返回 nil
	}
	return ctx.result, nil
}

func (in *interp) evalArgs(args []Expr, sc *scope, ctx *execCtx) ([]Value, error) {
	vals := make([]Value, 0, len(args))
	for _, a := range args {
		v, err := in.evalExpr(a, sc, ctx)
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, nil
}

func (in *interp) callMethod(obj Value, name string, args []Value, ctx *execCtx, pos Pos) (Value, error) {
	if obj.IsLib() {
		return in.callLibMethod(obj.Lib(), name, args, pos, ctx)
	}
	if obj.IsPtr() { // FFI 不透明句柄：仅支持 toString（十六进制）
		if name == "toString" {
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(obj.String()), nil
		}
		return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: no method %q on pointer", name), Pos: pos, Ctx: ctx}
	}
	if obj.IsList() {
		o := obj.List()
		switch name {
		case "head":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return IntV(int64(o.Head())), nil
		case "tail":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return IntV(int64(o.Tail())), nil
		case "size":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return IntV(int64(o.Size())), nil
		case "get":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsInt() {
				return NilV(), &RunError{Msg: "TypeError: get(i) 需要 int 下标", Pos: pos, Ctx: ctx}
			}
			gv, gerr := o.Get(int(args[0].Int()))
			if gerr != nil {
				return NilV(), &RunError{Msg: gerr.Error(), Pos: pos, Ctx: ctx}
			}
			return gv, nil
		case "next":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			v, err := o.Next()
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
			}
			return v, nil
		case "reset":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			o.Reset()
			return obj, nil
		case "append":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			o.Append(args[0])
			return NilV(), nil
		case "appendAll":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsList() {
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: appendAll requires a List, got %s", args[0].TypeName()), Pos: pos, Ctx: ctx}
			}
			l := args[0].List()
			o.AppendAll(l)
			return NilV(), nil
		case "toString":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(o.String()), nil
		case "__sort__":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if err := o.sortInPlace(); err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
			}
			return obj, nil
		}
	} else if obj.IsTable() {
		o := obj.Table()
		switch name {
		case "put":
			if err := wantArity(name, 2, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			o.Put(args[0], args[1])
			return NilV(), nil
		case "get":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			v, ok := o.Get(args[0])
			if !ok || v.IsNil() {
				return NilV(), nil
			}
			return v, nil
		case "contains":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return BoolV(o.Contains(args[0])), nil
		case "keys":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			lst := NewList()
			for _, kv := range o.Keys() {
				lst.Append(kv)
			}
			return ListV(lst), nil

		case "remove":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			o.Remove(args[0])
			return NilV(), nil
		case "size":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return IntV(int64(o.Size())), nil
		}
	} else if obj.IsCopyd() {
		o := obj.Copyd()
		if name == "ptr" {
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return o.V, nil // .ptr() 取出 Copyd 包装的地址
		}
		return in.callMethod(o.V, name, args, ctx, pos) // Copyd 透传
	} else if obj.IsTaskm() {
		_ = obj.Taskm() // taskm 是全局单例；方法走 in
		switch name {
		case "spawn":
			// v2：taskm.spawn() 参数为空，创建线程并直接返回 pid
			if len(args) != 0 {
				return NilV(), wantArity("taskm.spawn", 0, len(args), pos, ctx)
			}
			pid := in.newThread()
			t, _ := in.lookupTask(pid)
			return ThreadV(&ThreadValue{Pid: pid, t: t}), nil
		case "merge":
			// v2：taskm.merge(pid, fn, args...) 把函数并入线程 pid 执行
			if len(args) < 2 {
				return NilV(), wantArity("taskm.merge", 2, len(args), pos, ctx)
			}
			if !args[0].IsInt() {
				return NilV(), &RunError{Msg: "TypeError: taskm.merge first arg must be a pid (int)", Pos: pos, Ctx: ctx}
			}
			pid := args[0].Int()
			t, ok := in.lookupTask(int(pid))
			if !ok {
				return NilV(), &RunError{Msg: fmt.Sprintf("RuntimeError: unknown task pid %d", int(pid)), Pos: pos, Ctx: ctx}
			}
			fn, err := lookupFunc(args[1], in)
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
			}
			if len(args)-2 != len(fn.Params) {
				return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(args)-2), Pos: pos, Ctx: ctx}
			}
			if err := in.runOnThread(t, fn, args[2:], pos); err != nil {
				return NilV(), err
			}
			return NilV(), nil
		case "block":
			// v2：taskm.block(pid) 返回 void（只等待线程空闲）
			if len(args) != 1 {
				return NilV(), wantArity("taskm.block", 1, len(args), pos, ctx)
			}
			t, err := in.taskArg(args[0], pos, ctx)
			if err != nil {
				return NilV(), err
			}
			in.taskMu.Lock()
			busy := t.Busy
			ch := t.doneCh
			in.taskMu.Unlock()
			if busy {
				<-ch
			}
			if t.err != nil {
				return NilV(), t.err
			}
			return NilV(), nil // block 返回 void
		case "done":
			// done(pid)：该协程的线程是否空闲（没有函数占用）；v0.1 = 协程是否结束
			if len(args) != 1 {
				return NilV(), wantArity("taskm.done", 1, len(args), pos, ctx)
			}
			if !args[0].IsInt() {
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: taskm.done requires a pid (int), got %s", args[0].TypeName()), Pos: pos, Ctx: ctx}
			}
			pid := args[0].Int()
			t, ok := in.lookupTask(int(pid))
			if !ok {
				return NilV(), &RunError{Msg: fmt.Sprintf("RuntimeError: unknown task pid %d", int(pid)), Pos: pos, Ctx: ctx}
			}
			in.taskMu.Lock()
			idle := !t.Busy
			in.taskMu.Unlock()
			return BoolV(idle), nil // done = 线程是否空闲（没有函数占用）
		case "channel":
			cap := 1024 // 默认容量（spec §14.2）
			if len(args) == 1 {
				if !args[0].IsInt() || args[0].Int() < 1 || args[0].Int() > 1<<20 {
					return NilV(), &RunError{Msg: "TypeError: taskm.channel(n) requires 0 < n <= 1048576", Pos: pos, Ctx: ctx}
				}
				cap = int(args[0].Int())
			} else if len(args) != 0 {
				return NilV(), wantArity("taskm.channel", 0, len(args), pos, ctx)
			}
			return ChanV(NewChannel(cap)), nil
		}
	} else if obj.IsThread() {
		o := obj.Thread()
		switch name {
		case "merge":
			if len(args) < 1 {
				return NilV(), wantArity("thread.merge", 1, len(args), pos, ctx)
			}
			fn, err := lookupFunc(args[0], in)
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
			}
			if len(args)-1 != len(fn.Params) {
				return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(args)-1), Pos: pos, Ctx: ctx}
			}
			if err := in.runOnThread(o.t, fn, args[1:], pos); err != nil {
				return NilV(), err
			}
			return NilV(), nil
		case "pid":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return IntV(int64(o.Pid)), nil
		case "talk":
			if len(args) != 1 {
				return NilV(), wantArity(name, 1, len(args), pos, ctx)
			}
			if !args[0].IsChan() {
				return NilV(), &RunError{Msg: "TypeError: thread.talk 需要 channel 类实例", Pos: pos, Ctx: ctx}
			}
			return NilV(), nil
		}
	} else if obj.IsFile() {
		f := obj.File()
		switch name {
		case "read":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			b, err := os.ReadFile(f.Path)
			if err != nil {
				return NilV(), &RunError{Msg: "IOError: " + err.Error(), Pos: pos, Ctx: ctx}
			}
			return StrV(string(b)), nil
		case "write":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil || !args[0].IsStr() {
				if err != nil {
					return NilV(), err
				}
				return NilV(), &RunError{Msg: "TypeError: write(s String)", Pos: pos, Ctx: ctx}
			}
			if err := os.WriteFile(f.Path, []byte(args[0].Str()), 0o644); err != nil {
				return NilV(), &RunError{Msg: "IOError: " + err.Error(), Pos: pos, Ctx: ctx}
			}
			return NilV(), nil
		case "name":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(f.Path), nil
		}
	} else if obj.IsInt() || obj.IsFloat() || obj.IsBool() {
		if name == "toString" {
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(obj.String()), nil
		}
	} else if obj.IsStr() {
		o := obj.Str()
		switch name {
		case "size":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return IntV(int64(utf8.RuneCountInString(o))), nil
		case "contains":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() {
				return NilV(), &RunError{Msg: "TypeError: contains 需要 String 参数", Pos: pos, Ctx: ctx}
			}
			return BoolV(strings.Contains(o, args[0].Str())), nil
		case "startsWith":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() {
				return NilV(), &RunError{Msg: "TypeError: startsWith 需要 String 参数", Pos: pos, Ctx: ctx}
			}
			return BoolV(strings.HasPrefix(o, args[0].Str())), nil
		case "endsWith":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() {
				return NilV(), &RunError{Msg: "TypeError: endsWith 需要 String 参数", Pos: pos, Ctx: ctx}
			}
			return BoolV(strings.HasSuffix(o, args[0].Str())), nil
		case "indexOf":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() {
				return NilV(), &RunError{Msg: "TypeError: indexOf 需要 String 参数", Pos: pos, Ctx: ctx}
			}
			return IntV(int64(strings.Index(o, args[0].Str()))), nil // -1 = 不存在
		case "substring":
			if len(args) < 1 || len(args) > 2 || !args[0].IsInt() {
				return NilV(), &RunError{Msg: "TypeError: substring(start, end?) 需要 int 参数", Pos: pos, Ctx: ctx}
			}
			// 极限优化：不做全量 []rune 转换，前缀解码到字节偏移后再切片（零拷贝）
			n := int64(utf8.RuneCountInString(o))
			start := args[0].Int()
			end := n
			if len(args) == 2 {
				if !args[1].IsInt() {
					return NilV(), &RunError{Msg: "TypeError: substring 的 end 必须是 int", Pos: pos, Ctx: ctx}
				}
				end = args[1].Int()
			}
			if start < 0 || end < start || end > n {
				return NilV(), &RunError{Msg: fmt.Sprintf("StringIndexOutOfBoundsError: substring(%d, %d) 越界 [0,%d]", start, end, n), Pos: pos, Ctx: ctx}
			}
			b0 := 0
			for k := int64(0); k < start; k++ {
				_, sz := utf8.DecodeRuneInString(o[b0:])
				b0 += sz
			}
			b1 := b0
			for k := start; k < end; k++ {
				_, sz := utf8.DecodeRuneInString(o[b1:])
				b1 += sz
			}
			return StrV(o[b0:b1]), nil
		case "split":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() {
				return NilV(), &RunError{Msg: "TypeError: split 需要 String 分隔符", Pos: pos, Ctx: ctx}
			}
			parts := strings.Split(o, args[0].Str())
			lst := NewList()
			for _, p := range parts {
				lst.Append(StrV(p))
			}
			return ListV(lst), nil
		case "trim":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(strings.TrimSpace(o)), nil
		case "trimLeft":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(strings.TrimLeft(o, " \t\r\n")), nil
		case "trimRight":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(strings.TrimRight(o, " \t\r\n")), nil
		case "toLower":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(strings.ToLower(o)), nil
		case "toUpper":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return StrV(strings.ToUpper(o)), nil
		case "replace":
			if err := wantArity(name, 2, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() || !args[1].IsStr() {
				return NilV(), &RunError{Msg: "TypeError: replace(old, new) 需要 String 参数", Pos: pos, Ctx: ctx}
			}
			return StrV(strings.ReplaceAll(o, args[0].Str(), args[1].Str())), nil
		case "charAt":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsInt() {
				return NilV(), &RunError{Msg: "TypeError: charAt 需要 int 索引", Pos: pos, Ctx: ctx}
			}
			// 极限优化：前缀解码到目标 rune，只解码 i 个字符
			i := args[0].Int()
			b := 0
			for k := int64(0); k < i; k++ {
				if b >= len(o) {
					return NilV(), &RunError{Msg: fmt.Sprintf("StringIndexOutOfBoundsError: charAt(%d) 越界 [0,%d)", i, utf8.RuneCountInString(o)), Pos: pos, Ctx: ctx}
				}
				_, sz := utf8.DecodeRuneInString(o[b:])
				b += sz
			}
			if b >= len(o) {
				return NilV(), &RunError{Msg: fmt.Sprintf("StringIndexOutOfBoundsError: charAt(%d) 越界 [0,%d)", i, utf8.RuneCountInString(o)), Pos: pos, Ctx: ctx}
			}
			// 对齐到字符边界
			_, sz := utf8.DecodeRuneInString(o[b:])
			return StrV(o[b : b+sz]), nil
		case "toInt":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			v, err := strconv.ParseInt(strings.TrimSpace(o), 10, 32)
			if err != nil {
				return NilV(), &RunError{Msg: fmt.Sprintf("ParseError: %q 不是合法整数", o), Pos: pos, Ctx: ctx}
			}
			return IntV(v), nil
		case "toFloat":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(o), 64)
			if err != nil {
				return NilV(), &RunError{Msg: fmt.Sprintf("ParseError: %q 不是合法浮点数", o), Pos: pos, Ctx: ctx}
			}
			return FloatV(v), nil
		}
	} else if obj.IsMemorize() {
		o := obj.Memorize()
		if name == "call" {
			if len(args) != 2 {
				return NilV(), wantArity("call", 2, len(args), pos, ctx)
			}
			if !args[0].IsStruct() {
				return NilV(), &RunError{Msg: "TypeError: call 第一参数必须是 prefix 记录", Pos: pos, Ctx: ctx}
			}
			pref := args[0].Struct()
			if !args[1].IsStruct() {
				return NilV(), &RunError{Msg: "TypeError: call 第二参数必须是 .{in,out} 记录", Pos: pos, Ctx: ctx}
			}
			recv := args[1].Struct()
			return in.memorizeBufferCall(o, pref, recv, pos, ctx)
		}
	} else if obj.IsMemory() {
		o := obj.Memory()
		switch name {
		case "clear":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			in.mem.Clear() // globalMemory.clear()：按修改日志直接清理
			return NilV(), nil
		case "mode":
			// 实验性：GlobalMemory.mode(DynamicStackAndHeap) 将栈和堆动态分配
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return NilV(), nil
		case "compact":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			// compact() 根本不返回（spec §14.1）；实际清理无人占用的 block
			in.mem.Compact()
			return NilV(), nil
		case "setBlock":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsInt() || args[0].Int() < 1 || args[0].Int() > 1<<20 {
				return NilV(), &RunError{Msg: "TypeError: setBlock(n) requires a positive int block size", Pos: pos, Ctx: ctx}
			}
			o.BlockSize = int(args[0].Int()) // 动态调整 block 脏标记粒度
			return NilV(), nil
		}
	} else if obj.IsTask() {
		o := obj.Task()
		if name == "done" {
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			select {
			case <-o.doneCh:
				return BoolV(true), nil
			default:
				return BoolV(false), nil
			}
		}
	} else if obj.IsChan() {
		o := obj.Chan()
		switch name {
		case "send":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			o.ch <- args[0]
			return NilV(), nil
		case "recv":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return <-o.ch, nil
		}
	} else if obj.IsIO() {
		o := obj.IO()
		switch name {
		case "println":
			return ioPrintln(o, args, true, pos, ctx)
		case "print":
			return ioPrintln(o, args, false, pos, ctx)
		case "setIn":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if err := setInput(o, args[0]); err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
			}
			return NilV(), nil
		case "setOut":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if err := setOutput(o, args[0]); err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
			}
			return NilV(), nil
		case "readln":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			o.mu.RLock()
			line, err := o.rd.ReadString('\n')
			o.mu.RUnlock()
			if err != nil && line == "" {
				return NilV(), &RunError{Msg: "IOError: " + err.Error(), Pos: pos, Ctx: ctx}
			}
			return StrV(strings.TrimRight(line, "\r\n")), nil
		}
	} else if obj.IsIn() {
		o := obj.In()
		switch name {
		case "readln":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			line, err := bufio.NewReader(o.R).ReadString('\n')
			if err != nil && line == "" {
				return NilV(), &RunError{Msg: "IOError: " + err.Error(), Pos: pos, Ctx: ctx}
			}
			return StrV(strings.TrimRight(line, "\r\n")), nil
		case "close":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if c, ok := o.R.(io.Closer); ok {
				_ = c.Close()
			}
			return NilV(), nil
		}
	} else if obj.IsOut() {
		o := obj.Out()
		switch name {
		case "println", "print":
			line := ""
			for i, a := range args {
				if i > 0 {
					line += " "
				}
				line += a.String()
			}
			if name == "println" {
				line += "\n"
			}
			if _, err := fmt.Fprint(o.W, line); err != nil {
				return NilV(), &RunError{Msg: "IOError: " + err.Error(), Pos: pos, Ctx: ctx}
			}
			return NilV(), nil
		case "write":
			if len(args) != 1 {
				return NilV(), wantArity(name, 1, len(args), pos, ctx)
			}
			if _, err := fmt.Fprint(o.W, args[0].String()); err != nil {
				return NilV(), &RunError{Msg: "IOError: " + err.Error(), Pos: pos, Ctx: ctx}
			}
			return NilV(), nil
		case "close":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if c, ok := o.W.(io.Closer); ok {
				_ = c.Close()
			}
			return NilV(), nil
		}
	} else if obj.IsStruct() {
		o := obj.Struct()
		if fn := in.selfMethodOf(o.SType, name); fn != nil {
			callArgs := append([]Value{obj}, args...)
			return in.callFunc(fn, callArgs, pos, ctx.depth)
		}
		if in.staticMethodOf(o.SType, name) != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: %s.%s is a static method; call it via %s::%s(...)", o.SType, name, o.SType, name), Pos: pos, Ctx: ctx}
		}
	}
	return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: no method %q on %s", name, obj.TypeName()), Pos: pos, Ctx: ctx}
}

func wantArity(name string, want, got int, pos Pos, ctx *execCtx) error {
	if want != got {
		return &RunError{Msg: fmt.Sprintf("CompileError: %s() expects %d args, got %d", name, want, got), Pos: pos, Ctx: ctx}
	}
	return nil
}

func ioPrintln(s *IOStream, args []Value, newline bool, pos Pos, ctx *execCtx) (Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.String()
	}
	out := strings.Join(parts, " ")
	if newline {
		out += "\n"
	}
	if _, err := io.WriteString(s.Out, out); err != nil {
		return NilV(), &RunError{Msg: "IOError: " + err.Error(), Pos: pos, Ctx: ctx}
	}
	return NilV(), nil
}

func setInput(s *IOStream, v Value) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.IsStr() {
		f, err := os.Open(v.Str())
		if err != nil {
			return fmt.Errorf("IOError: cannot open %q: %v", v.Str(), err)
		}
		s.In = f
		s.rd = bufio.NewReader(f)
		return nil
	}
	if v.IsIn() {
		t := v.In()
		s.In = t.R
		s.rd = bufio.NewReader(t.R)
		return nil
	}
	return fmt.Errorf("TypeError: setIn requires a path string or an InputStream, got %s", v.TypeName())
}

func setOutput(s *IOStream, v Value) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.IsStr() {
		f, err := os.Create(v.Str())
		if err != nil {
			return fmt.Errorf("IOError: cannot create %q: %v", v.Str(), err)
		}
		s.Out = f
		return nil
	}
	if v.IsOut() {
		s.Out = v.Out().W
		return nil
	}
	return fmt.Errorf("TypeError: setOut requires a path string or an OutputStream, got %s", v.TypeName())
}

func (in *interp) evalScopeCall(x *ScopeCall, sc *scope, ctx *execCtx) (Value, error) {
	args, err := in.evalArgs(x.Args, sc, ctx)
	if err != nil {
		return NilV(), err
	}
	switch x.Scope {
	case "memorize":
		if x.Name == "new" && len(args) == 0 {
			return MemorizeV(&MemorizeBuffer{Table: NewHashTable()}), nil
		}
		return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: memorize has no static method %q", x.Name), Pos: x.Pos, Ctx: ctx}
	case "HashTable":
		if x.Name == "new" && len(args) == 0 {
			return TableV(NewHashTable()), nil
		}
		return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: HashTable has no static method %q", x.Name), Pos: x.Pos, Ctx: ctx}
	case "List":
		if x.Name == "new" && len(args) == 0 {
			return ListV(NewList()), nil
		}
		return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: List has no static method %q", x.Name), Pos: x.Pos, Ctx: ctx}
	case "taskm":
		// taskm 是全局变量：正确语法是 taskm.spawn(...) / taskm.block(...) 等
		return NilV(), &RunError{Msg: "TypeError: taskm is a global variable — use taskm.spawn(...) / taskm.block(pid) / taskm.done(pid) / taskm.merge(pid) / taskm.channel([n])", Pos: x.Pos, Ctx: ctx}
	case "IO":
		switch x.Name {
		case "setIn":
			if len(args) != 2 {
				return NilV(), wantArity("IO::setIn", 2, len(args), x.Pos, ctx)
			}
			if !args[0].IsIO() {
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: IO::setIn's first arg must be an IOStream, got %s", args[0].TypeName()), Pos: x.Pos, Ctx: ctx}
			}
			ioObj := args[0].IO()
			if err := setInput(ioObj, args[1]); err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
			}
			return NilV(), nil
		case "setOut":
			if len(args) != 2 {
				return NilV(), wantArity("IO::setOut", 2, len(args), x.Pos, ctx)
			}
			if !args[0].IsIO() {
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: IO::setOut's first arg must be an IOStream, got %s", args[0].TypeName()), Pos: x.Pos, Ctx: ctx}
			}
			ioObj := args[0].IO()
			if err := setOutput(ioObj, args[1]); err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: x.Pos, Ctx: ctx}
			}
			return NilV(), nil
		}
		return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: IO has no static method %q", x.Name), Pos: x.Pos, Ctx: ctx}
	}
	if x.Scope == "file" && x.Name == "new" {
		if len(args) != 1 || !args[0].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: file::new(name String)", Pos: x.Pos, Ctx: ctx}
		}
		return FileV(&FileValue{Path: args[0].Str()}), nil
	}
	if fn := in.staticMethodOf(x.Scope, x.Name); fn != nil {
		return in.callFunc(fn, args, x.Pos, ctx.depth)
	}
	return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: unknown scope %q", x.Scope), Pos: x.Pos, Ctx: ctx}
}

// allDefs 函数全定义（主 + 重载）。
func (in *interp) allDefs(name string) []*Func {
	if fn, ok := in.fns[name]; ok {
		return append([]*Func{fn}, in.overloads[name]...)
	}
	return in.overloads[name]
}

// bestMatchV 按实参值选最优重载（参数数匹配优先；类型 Kind 一致计分最高；唯一最优取胜）。
func (in *interp) bestMatchV(defs []*Func, args []Value) *Func {
	var best *Func
	bestScore := -1
	for _, fn := range defs {
		if len(fn.Params) != len(args) {
			continue
		}
		score := 0
		for i := range fn.Params {
			tp := fn.Params[i].Type
			switch {
			case (tp == "int" || tp == "long" || tp == "char") && args[i].IsInt():
				score += 10
			case (tp == "float" || tp == "double") && args[i].IsFloat():
				score += 10
			case (tp == "f32") && args[i].IsFloat():
				score += 10
			case tp == "String" && args[i].IsStr():
				score += 10
			case tp == "bool" && args[i].IsBool():
				score += 10
			case tp == "any" || tp == "interface{}" || tp == "":
				score += 2
			default:
				score += 3
			}
		}
		if score > bestScore {
			best = fn
			bestScore = score
		}
	}
	return best
}

// overloadErr 无匹配重载时生成兼容报错（参数数不匹配沿用 expects 文案）。
func overloadErr(defs []*Func, name string, n int) string {
	for _, d := range defs {
		if len(d.Params) != n {
			return fmt.Sprintf("CompileError: %s expects %d args, got %d", name, len(d.Params), n)
		}
	}
	return fmt.Sprintf("CompileError: 未找到匹配重载 %q（参数类型不匹配）", name)
}

// selfMethodOf 聚合多 impl 查找实例方法（self 首参）。
func (in *interp) selfMethodOf(typ, name string) *Func {
	for _, d := range in.implDefsFor(typ) {
		if fn := d.SelfMethods[name]; fn != nil {
			return fn
		}
	}
	return nil
}

// staticMethodOf 聚合多 impl 查找静态方法（无 self 首参）。
func (in *interp) staticMethodOf(typ, name string) *Func {
	for _, d := range in.implDefsFor(typ) {
		if fn := d.Methods[name]; fn != nil {
			return fn
		}
	}
	return nil
}

// fileInputStreamBuiltin 打开文件输入流（ifstream/FileInputStream 共用）。
func (in *interp) fileInputStreamBuiltin(args []Value, pos Pos, ctx *execCtx) (Value, error) {
	if len(args) != 1 {
		return NilV(), wantArity("ifstream", 1, len(args), pos, ctx)
	}
	if !args[0].IsStr() {
		return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: ifstream requires a path string, got %s", args[0].TypeName()), Pos: pos, Ctx: ctx}
	}
	p := args[0].Str()
	f, err := os.Open(p)
	if err != nil {
		return NilV(), &RunError{Msg: fmt.Sprintf("IOError: cannot open %q: %v", p, err), Pos: pos, Ctx: ctx}
	}
	return InV(&InputStream{R: f}), nil
}

// fileOutputStreamBuiltin 创建文件输出流（ofstream/FileOutputStream 共用）。
func (in *interp) fileOutputStreamBuiltin(args []Value, pos Pos, ctx *execCtx) (Value, error) {
	if len(args) != 1 {
		return NilV(), wantArity("ofstream", 1, len(args), pos, ctx)
	}
	if !args[0].IsStr() {
		return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: ofstream requires a path string, got %s", args[0].TypeName()), Pos: pos, Ctx: ctx}
	}
	f, err := os.Create(args[0].Str())
	if err != nil {
		return NilV(), &RunError{Msg: fmt.Sprintf("IOError: cannot create %q: %v", args[0].Str(), err), Pos: pos, Ctx: ctx}
	}
	return OutV(&OutputStream{W: f}), nil
}

// sumBuiltin：sum(generate, begin, stop, step?) —— 求和优化。
// 线性生成器（二阶差分恒定）直接算术闭式 n*(first+last)/2（乘加，O(1)）；
// 非线性退化为循环。用户提出的位级置换（每位 1 计数重排成乘加）在运行时无法获知
// 生成器内部位模式，线性闭式已覆盖最常见的 sum(index)/sum(a*i+b) 场景。
func (in *interp) sumBuiltin(args []Value, pos Pos, ctx *execCtx) (Value, error) {
	if len(args) != 3 && len(args) != 4 {
		return NilV(), wantArity("sum", 3, len(args), pos, ctx)
	}
	if !args[0].IsFunc() {
		return NilV(), &RunError{Msg: "TypeError: sum 第一个参数必须是函数引用 generate", Pos: pos, Ctx: ctx}
	}
	// 位级置换：均匀随机生成器（rand）每列 1 计数 = n/2（期望）→ 乘加闭式 O(1)
	// 随机数每位置 1 概率 1/2，交换/置换后每列均匀，Σ = Σ_k (n/2)·2^k = n·2^30
	gen := args[0].Func()
	if gen.fn.Name == "rand" {
		if args[1].IsInt() && args[2].IsInt() {
			begin, stop := args[1].Int(), args[2].Int()
			{
				n := stop - begin
				if n > 0 {
					return wrapI32(n << 30), nil // n × 2^30（[0,2^31-1] 均匀的期望）
				}
				return IntV(0), nil
			}
		}
	}
	if !args[1].IsInt() {
		return NilV(), &RunError{Msg: "TypeError: sum 的 begin 必须是 int", Pos: pos, Ctx: ctx}
	}
	if !args[2].IsInt() {
		return NilV(), &RunError{Msg: "TypeError: sum 的 stop 必须是 int", Pos: pos, Ctx: ctx}
	}
	begin, stop := args[1].Int(), args[2].Int()
	step := int64(1)
	if len(args) == 4 {
		if args[3].IsInt() && args[3].Int() != 0 {
			step = args[3].Int()
		} else {
			return NilV(), &RunError{Msg: "TypeError: sum 的 step 必须是非零 int", Pos: pos, Ctx: ctx}
		}
	}
	g := func(i int64) (int64, error) {
		v, err := in.callFunc(gen.fn, []Value{IntV(i)}, pos, ctx.depth)
		if err != nil {
			return 0, err
		}
		if !v.IsInt() {
			return 0, &RunError{Msg: "TypeError: generate 必须返回 int", Pos: pos, Ctx: ctx}
		}
		return v.Int(), nil
	}
	// 线性探测：二阶差分恒定 → 闭式 O(1)（3 参默认 step=1 同样探测）
	if true {
		g0, _ := g(begin)
		g1, _ := g(begin + step)
		g2, _ := g(begin + 2*step)
		if d1, d2 := g1-g0, g2-g1; d1 == d2 {
			n := (stop - begin + step - 1) / step
			if n <= 0 {
				return IntV(0), nil
			}
			// 校验末项：周期函数（如 n%3）三点差分可能巧合相等，末项不符则非线性
			last := g0 + d1*(n-1)
			if actual, err := g(begin + (n-1)*step); err == nil && actual == last {
				return wrapI32(n * (g0 + last) / 2), nil // 乘加闭式（int=32 位，wrap）
			}
		}
	}
	// 位级置换：生成器序列有周期 P → 周期位统计预计算，O(P+bits) 乘加（P << n 时远快于循环）
	if n := (stop - begin + step - 1) / step; n > 0 {
		if p, ok := detectPeriod(g, begin, step, n); ok && p < n {
			// 一个周期的位统计：counts[k] = 周期内第 k 位为 1 的次数
			var counts [32]int64
			for j := int64(0); j < p; j++ {
				v, err := g(begin + j*step)
				if err != nil {
					return NilV(), err
				}
				for k := 0; k < 32; k++ {
					if v&(1<<uint(k)) != 0 {
						counts[k]++
					}
				}
			}
			q := n / p
			r := n % p
			var sum int64
			for k := 0; k < 32; k++ {
				sum += (q * counts[k]) << uint(k)
			}
			// 余数部分逐项补
			for j := int64(0); j < r; j++ {
				v, err := g(begin + j*step)
				if err != nil {
					return NilV(), err
				}
				sum += v
			}
			return wrapI32(sum), nil // 周期位级置换闭式：int=32 位 wrap
		}
	}
	// 无短周期（如 32 位全周期随机数）：逐项加法（比逐项位统计更快）
	var total int64
	for i := int64(begin); i < int64(stop); i += int64(step) {
		v, err := g(i)
		if err != nil {
			return NilV(), err
		}
		total += v
	}
	return wrapI32(total), nil // 退化循环/周期兜底：int=32 位 wrap
}

// detectPeriod 检测生成器序列周期（最多探测 65536 项，步进 step）。
func detectPeriod(g func(int64) (int64, error), begin, step, n int64) (int64, bool) {
	// 只探测短周期（≤256）：真随机数无周期，快速失败直接走加法（O(n) 已是下界）
	limit := n
	if limit > 256 {
		limit = 256
	}
	v0, err := g(begin)
	if err != nil {
		return 0, false
	}
	for p := int64(1); p < limit; p++ {
		v, err := g(begin + p*step)
		if err != nil {
			return 0, false
		}
		if v == v0 {
			// 再验证一位确认周期
			v2, err := g(begin + (p+1)*step)
			if err == nil {
				v3, _ := g(begin + step)
				if v2 == v3 {
					return p, true
				}
			}
		}
	}
	return 0, false
}

func (in *interp) registerIOBuiltins() {
	in.builtins["clock"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 0 {
			return NilV(), wantArity("clock", 0, len(args), pos, ctx)
		}
		return IntV(int64(int32(time.Now().UnixMicro()))), nil // 微秒
	}
	in.builtins["rand"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 0 {
			return NilV(), wantArity("rand", 0, len(args), pos, ctx)
		}
		// LCG：确定性伪随机 [0, 2^31-1]
		in.randState = in.randState*6364136223846793005 + 1442695040888963407
		return IntV(int64(int32(in.randState >> 33))), nil
	}
	in.builtins["sum"] = in.sumBuiltin
	in.builtins["FileInputStream"] = in.fileInputStreamBuiltin
	in.builtins["ifstream"] = in.fileInputStreamBuiltin
	in.builtins["iofstream"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 1 {
			return NilV(), wantArity("iofstream", 1, len(args), pos, ctx)
		}
		if !args[0].IsStr() {
			return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: iofstream requires a path string, got %s", args[0].TypeName()), Pos: pos, Ctx: ctx}
		}
		// 双向文件流：读+写
		rf, err := os.OpenFile(args[0].Str(), os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("IOError: cannot open %q: %v", args[0].Str(), err), Pos: pos, Ctx: ctx}
		}
		return InV(&InputStream{R: rf}), nil
	}
	in.builtins["FileOutputStream"] = in.fileOutputStreamBuiltin
	in.builtins["ofstream"] = in.fileOutputStreamBuiltin
	in.builtins["ConsoleInputStream"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 0 {
			return NilV(), wantArity("ConsoleInputStream", 0, len(args), pos, ctx)
		}
		return InV(&InputStream{R: os.Stdin}), nil
	}
	in.builtins["ConsoleOutputStream"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 0 {
			return NilV(), wantArity("ConsoleOutputStream", 0, len(args), pos, ctx)
		}
		return OutV(&OutputStream{W: os.Stdout}), nil
	}
	// [官方库 system 原语] 进程执行：qkexec(cmd) -> 退出码（shell -c）
	in.builtins["qkexec"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) < 1 || len(args) > 2 || !args[0].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qkexec(cmd String[, retries int])", Pos: pos, Ctx: ctx}
		}
		retries := retriesOf(args, pos, ctx)
		var code int64
		for attempt := 0; attempt <= retries; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(200*(1<<uint(attempt-1))) * time.Millisecond) // 指数退避
			}
			cmd := exec.Command("sh", "-c", args[0].Str())
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					code = int64(ee.ExitCode())
					if code != 0 && attempt < retries {
						continue
					}
					return IntV(code), nil
				}
				if attempt < retries {
					continue
				}
				return NilV(), &RunError{Msg: fmt.Sprintf("IOError: 无法执行命令：%v", err), Pos: pos, Ctx: ctx}
			}
			code = 0
			break
		}
		return IntV(code), nil
	}
	// qkexecv(prog, args List<String>) -> 退出码（不经 shell，argv 直传）
	in.builtins["qkexecv"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) < 2 || len(args) > 3 || !args[0].IsStr() || !args[1].IsList() {
			return NilV(), &RunError{Msg: "TypeError: qkexecv(prog String, args List<String>[, retries])", Pos: pos, Ctx: ctx}
		}
		argList := args[1].List()
		argv := make([]string, 0, argList.Size())
		for argList.Head() != argList.Tail() {
			it, err := argList.Next()
			if err != nil {
				return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
			}
			if !it.IsStr() {
				return NilV(), &RunError{Msg: "TypeError: qkexecv 参数必须是 List<String>", Pos: pos, Ctx: ctx}
			}
			argv = append(argv, it.Str())
		}
		retries := retriesOf(args, pos, ctx)
		var code int64
		for attempt := 0; attempt <= retries; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(200*(1<<uint(attempt-1))) * time.Millisecond)
			}
			cmd := exec.Command(args[0].Str(), argv...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					code = int64(ee.ExitCode())
					if code != 0 && attempt < retries {
						continue
					}
					return IntV(code), nil
				}
				if attempt < retries {
					continue
				}
				return NilV(), &RunError{Msg: fmt.Sprintf("IOError: 无法启动程序：%v", err), Pos: pos, Ctx: ctx}
			}
			code = 0
			break
		}
		return IntV(code), nil
	}
	// qkpopen(cmd) -> InputStream（捕获 stdout；8MB 上限）
	in.builtins["qkpopen"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 1 || !args[0].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qkpopen(cmd String) 需要一个字符串命令", Pos: pos, Ctx: ctx}
		}
		cmd := exec.Command("sh", "-c", args[0].Str())
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("IOError: 命令执行失败：%v", err), Pos: pos, Ctx: ctx}
		}
		if len(out) > 8<<20 {
			return NilV(), &RunError{Msg: "IOError: qkpopen 输出超过 8MB 上限", Pos: pos, Ctx: ctx}
		}
		return InV(&InputStream{R: bytes.NewReader(out)}), nil
	} // [actions 库原语] 网络层：qkhttp_get(url) -> String（10s 超时，8MiB 响应上限）
	in.builtins["qkhttp_get"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) < 1 || len(args) > 2 || !args[0].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qkhttp_get(url String[, retries int])", Pos: pos, Ctx: ctx}
		}
		cli := &http.Client{Timeout: 10 * time.Second}
		retries := retriesOf(args, pos, ctx)
		var resp *http.Response
		var err error
		var lastStatus int
		for attempt := 0; ; attempt++ {
			resp, err = cli.Get(args[0].Str())
			if resp != nil {
				lastStatus = resp.StatusCode
			}
			if (err == nil && lastStatus < 500) || attempt >= retries {
				break
			}
			if resp != nil {
				resp.Body.Close()
				resp = nil
			}
			time.Sleep(time.Duration(300*(1<<uint(attempt))) * time.Millisecond)
		}
		if err != nil && (resp == nil || lastStatus == 0) {
			return NilV(), &RunError{Msg: fmt.Sprintf("IOError: 请求失败：%v", err), Pos: pos, Ctx: ctx}
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("IOError: 读取响应失败：%v", err), Pos: pos, Ctx: ctx}
		}
		if len(body) > 8<<20 {
			return NilV(), &RunError{Msg: "IOError: 响应超过 8MiB 上限", Pos: pos, Ctx: ctx}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return NilV(), &RunError{Msg: fmt.Sprintf("HTTPError: 状态码 %d", resp.StatusCode), Pos: pos, Ctx: ctx}
		}
		return StrV(string(body)), nil
	}
	// qkhttp_post(url, body String, contentType String) -> String
	in.builtins["qkhttp_post"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) < 3 || len(args) > 4 || !args[0].IsStr() || !args[1].IsStr() || !args[2].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qkhttp_post(url, body, contentType[, retries])", Pos: pos, Ctx: ctx}
		}
		cli := &http.Client{Timeout: 10 * time.Second}
		retries := retriesOf(args, pos, ctx)
		var resp *http.Response
		var err error
		var lastStatus int
		for attempt := 0; ; attempt++ {
			resp, err = cli.Post(args[0].Str(), args[2].Str(), strings.NewReader(args[1].Str()))
			if resp != nil {
				lastStatus = resp.StatusCode
			}
			if (err == nil && lastStatus < 500) || attempt >= retries {
				break
			}
			if resp != nil {
				resp.Body.Close()
				resp = nil
			}
			time.Sleep(time.Duration(300*(1<<uint(attempt))) * time.Millisecond)
		}
		if err != nil && (resp == nil || lastStatus == 0) {
			return NilV(), &RunError{Msg: fmt.Sprintf("IOError: 请求失败：%v", err), Pos: pos, Ctx: ctx}
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("IOError: 读取响应失败：%v", err), Pos: pos, Ctx: ctx}
		}
		if len(body) > 8<<20 {
			return NilV(), &RunError{Msg: "IOError: 响应超过 8MiB 上限", Pos: pos, Ctx: ctx}
		}
		return StrV(string(body)), nil
	}

	// [json 库原语] json 序列化/反序列化
	in.builtins["qkjson_dumps"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 1 {
			return NilV(), &RunError{Msg: "TypeError: qkjson_dumps(v) 需要一个参数", Pos: pos, Ctx: ctx}
		}
		obj, err := qkjsonV(args[0])
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("JSONError: %v", err), Pos: pos, Ctx: ctx}
		}
		b, err := json.Marshal(obj)
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("JSONError: %v", err), Pos: pos, Ctx: ctx}
		}
		return StrV(string(b)), nil
	}
	in.builtins["qkjson_loads"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 1 || !args[0].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qkjson_loads(s String)", Pos: pos, Ctx: ctx}
		}
		var raw interface{}
		if err := json.Unmarshal([]byte(args[0].Str()), &raw); err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("JSONError: %v", err), Pos: pos, Ctx: ctx}
		}
		return qkjsonFromGo(raw)
	}
	// [cleg 屏幕宿主] 跨系统窗口呈现（X11/GDI）
	// [file 原语] 读/写全文件
	in.builtins["qkfile_read"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 1 || !args[0].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qkfile_read(path String)", Pos: pos, Ctx: ctx}
		}
		b, err := os.ReadFile(args[0].Str())
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("IOError: %v", err), Pos: pos, Ctx: ctx}
		}
		return StrV(string(b)), nil
	}
	in.builtins["qkfile_write"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 2 || !args[0].IsStr() || !args[1].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qkfile_write(path String, content String)", Pos: pos, Ctx: ctx}
		}
		if err := os.WriteFile(args[0].Str(), []byte(args[1].Str()), 0o644); err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf("IOError: %v", err), Pos: pos, Ctx: ctx}
		}
		return NilV(), nil
	}
	// [cleg style 解析] qkcleg_style_parse(styleTable, jsonText)：JSON 字符串 → HashTable → merge 进 style
	// [cleg style 装载] qkcleg_style_load(styleTable, path)：读 JSON 文件 → HashTable → merge 进 style
	// [cleg 渲染原语] CPU 光栅帧缓冲（性能极限：线性 u32、预分配、零分配热路径）

	// [cleg 自动重绘] qkcleg_auto(node) 注册；qkcleg_tick() 逐个 render（帧循环模型）
	// [cleg 信号] qksignal_emit(node, name)：节点实现 onClicked 等方法则调用
	in.builtins["qksignal_emit"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) < 2 || !args[0].IsStruct() || !args[1].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qksignal_emit(node, name String [, args...])", Pos: pos, Ctx: ctx}
		}
		node := args[0]
		name := args[1].Str()
		// 短名归一化：clicked → onClicked（先直查全名，未命中则 on+首字母大写）
		if fn := in.selfMethodOf(node.Struct().SType, name); fn != nil {
			callArgs := append([]Value{node}, args[2:]...)
			return in.callFunc(fn, callArgs, pos, ctx.depth)
		}
		if name != "" && name[0] != 'o' {
			onName := "on" + strings.ToUpper(name[:1]) + name[1:]
			if fn := in.selfMethodOf(node.Struct().SType, onName); fn != nil {
				callArgs := append([]Value{node}, args[2:]...)
				return in.callFunc(fn, callArgs, pos, ctx.depth)
			}
		}
		return NilV(), nil
	}
	// [cleg qss 读取] 原语版：get/num/cr
}

// qkjsonFromGo 把 json.Unmarshal 结果转为 Value（HashTable / List / 标量；整数值视作 int）。
func qkjsonFromGo(raw interface{}) (Value, error) {
	switch t := raw.(type) {
	case nil:
		return NilV(), nil
	case bool:
		return BoolV(t), nil
	case float64:
		if t == float64(int64(t)) && t >= -1<<31 && t < 1<<31 {
			return IntV(int64(int32(t))), nil
		}
		return FloatV(t), nil
	case string:
		return StrV(t), nil
	case []interface{}:
		lst := NewList()
		for _, it := range t {
			v, err := qkjsonFromGo(it)
			if err != nil {
				return NilV(), err
			}
			lst.Append(v)
		}
		return ListV(lst), nil
	case map[string]interface{}:
		h := NewHashTable()
		for k, it := range t {
			v, err := qkjsonFromGo(it)
			if err != nil {
				return NilV(), err
			}
			h.m[hashKey(StrV(k))] = v // 键规整为 hashKey 格式（与 Put 一致，get/contains 可查）
		}
		return TableV(h), nil
	}
	return StrV(fmt.Sprintf("%v", raw)), nil
}

// qkjsonV 把 Value 转为 JSON 可序列化的 Go 对象（round-trip 安全）。
func qkjsonV(v Value) (interface{}, error) {
	switch {
	case v.IsNil():
		return nil, nil
	case v.IsInt():
		return v.Int(), nil
	case v.IsFloat():
		return v.Float(), nil
	case v.IsBool():
		return v.Bool(), nil
	case v.IsStr():
		return v.Str(), nil
	case v.IsList():
		out := make([]interface{}, 0, v.List().Size())
		l := v.List()
		for l.Head() != l.Tail() {
			it, err := l.Next()
			if err != nil {
				return nil, err
			}
			got, err := qkjsonV(it)
			if err != nil {
				return nil, err
			}
			out = append(out, got)
		}
		return out, nil
	case v.IsTable():
		out := map[string]interface{}{}
		// 直接遍历内部 map 深拷贝值；键还原为原生键（剥离 hashKey 的 "TypeName:" 前缀）
		for k, it := range v.Table().m {
			got, err := qkjsonV(it)
			if err != nil {
				return nil, err
			}
			key := k
			if i := strings.IndexByte(k, ':'); i >= 0 {
				key = k[i+1:]
			}
			out[key] = got
		}
		return out, nil
	default:
		return v.String(), nil // 其它类型 JSON 化字符串
	}
}

// retriesOf 读取可选 retries 参数（默认 retries=1，即失败后再尝试 1 次；0=不重试）。
func retriesOf(args []Value, pos Pos, ctx *execCtx) int {
	if len(args) == 0 {
		return 1
	}
	last := args[len(args)-1]
	if last.IsInt() {
		n := last.Int()
		if n < 0 || n > 10 {
			return 1
		}
		return int(n)
	}
	return 1
}

// registerTask 登记协程，返回其 pid。
func (in *interp) registerTask(t *Task) int {
	in.taskMu.Lock()
	defer in.taskMu.Unlock()
	in.nextPid++
	in.tasks[in.nextPid] = t
	return in.nextPid
}

// lookupTask 按 pid 查找协程。
func (in *interp) lookupTask(pid int) (*Task, bool) {
	in.taskMu.Lock()
	defer in.taskMu.Unlock()
	t, ok := in.tasks[pid]
	return t, ok
}

// newThread 创建线程（taskm.spawn()），返回 pid。
func (in *interp) newThread() int {
	t := &Task{doneCh: make(chan struct{}), BlockID: in.mem.Alloc(globalMemory.BlockSize, 0)}
	pid := in.registerTask(t)
	t.Pid = pid
	return pid
}

// runOnThread 把函数并入线程 pid 执行（taskm.merge）。
func (in *interp) runOnThread(t *Task, fn *Func, args []Value, pos Pos) error {
	in.taskMu.Lock()
	if t.Busy {
		in.taskMu.Unlock()
		return &RunError{Msg: "RuntimeError: thread busy — merge only when idle (taskm.done)", Pos: pos}
	}
	t.Busy = true
	t.doneCh = make(chan struct{})
	in.taskMu.Unlock()

	// 内存 block 归属该线程；结束后自动标记可回收
	blockID := in.mem.Alloc(globalMemory.BlockSize, t.Pid)
	t.BlockID = blockID
	ctx := in.newCtx(fn, args, pos) // 直接分配版本身无共享，taskm 线程安全
	lg := ctx.ensureLog()
	lg.mem = in.mem
	lg.blockID = blockID
	lg.Append(StrV(fmt.Sprintf("taskm: merged %s(%d) pid=%d", fn.Name, len(args), t.Pid)))
	go func() {
		t.err = in.execute(ctx)
		ctx.ensureLog().Append(StrV("taskm: done"))
		in.mem.ReclaimTask(t.Pid)
		in.taskMu.Lock()
		t.Busy = false
		in.taskMu.Unlock()
		close(t.doneCh)
	}()
	return nil
}

// taskArg 接受 Task 或 pid，返回对应协程。
func (in *interp) taskArg(v Value, pos Pos, ctx *execCtx) (*Task, error) {
	if v.IsTask() {
		return v.Task(), nil
	}
	if v.IsInt() {
		t, ok := in.lookupTask(int(v.Int()))
		if !ok {
			return nil, &RunError{Msg: fmt.Sprintf("RuntimeError: unknown task pid %d", v.Int()), Pos: pos, Ctx: ctx}
		}
		return t, nil
	}
	return nil, &RunError{Msg: fmt.Sprintf("TypeError: taskm requires a Task or pid, got %s", v.TypeName()), Pos: pos, Ctx: ctx}
}

// lookupFunc resolves a function reference (FuncValue or a name string).
func lookupFunc(v Value, in *interp) (*Func, error) {
	if v.IsFunc() {
		return v.Func().fn, nil
	}
	if v.IsStr() {
		if fn, ok := in.fns[v.Str()]; ok {
			return fn, nil
		}
		return nil, fmt.Errorf("CompileError: undeclared function %q", v.Str())
	}
	return nil, fmt.Errorf("TypeError: taskm::spawn requires a function reference, got %s", v.TypeName())
}

// memorizeBufferCall 实现内置 memorize 实例的 call(prefix, rec)：
// 按 rec.in 记忆化，执行 prefix.fn，结果写回 rec.out。
func (in *interp) memorizeBufferCall(mb *MemorizeBuffer, prefix *StructValue, rec *StructValue, pos Pos, ctx *execCtx) (Value, error) {
	nv, ok := prefix.Fields["fn"]
	if !ok {
		return NilV(), &RunError{Msg: "TypeError: memorize prefix 缺少被包装函数 fn", Pos: pos, Ctx: ctx}
	}
	if !nv.IsFunc() {
		return NilV(), &RunError{Msg: "TypeError: prefix.fn 不是函数", Pos: pos, Ctx: ctx}
	}
	f := nv.Func()
	if !rec.Fields["in"].IsList() {
		return NilV(), &RunError{Msg: "TypeError: rec.in 必须是 List", Pos: pos, Ctx: ctx}
	}
	// 记忆化键：in 参数列表
	inList := rec.Fields["in"].List()
	if cached, hit := mb.Table.Get(ListV(inList)); hit {
		rec.Fields["out"] = cached
		return cached, nil
	}
	argVals := make([]Value, 0, inList.Size())
	for i := 0; i < inList.Size(); i++ {
		v, _ := inList.Get(i)
		argVals = append(argVals, v)
	}
	res, err := in.callFunc(f.fn, argVals, pos, ctx.depth)
	if err != nil {
		return NilV(), err
	}
	rec.Fields["out"] = res
	mb.Table.Put(ListV(inList), res)
	return res, nil
}

// pow2 返回 n 使得 v == 2^n（v>0 且为 2 的幂）。
func pow2(v int32) (int, bool) {
	if v <= 0 {
		return 0, false
	}
	n := 0
	for v > 1 {
		if v&1 == 1 {
			return 0, false
		}
		v >>= 1
		n++
	}
	return n, true
}

func binOp(op string, l, r Value, pos Pos, ctx *execCtx) (Value, error) {
	if op == "+" {
		if l.IsStr() || r.IsStr() {
			return StrV(l.String() + r.String()), nil
		}
	}
	switch op {
	case "+", "-", "*", "/", "%":
		return arith(op, l, r, pos, ctx)
	case "<<", ">>":
		if !l.IsInt() || !r.IsInt() {
			return NilV(), &RunError{Msg: "TypeError: 位移运算需要 int 操作数", Pos: pos, Ctx: ctx}
		}
		li, sh := l.Int(), r.Int()
		if op == "<<" {
			return IntV(int64(int32(li) << uint(sh&31))), nil
		}
		return IntV(int64(int32(li) >> uint(sh&31))), nil // 算术右移（符号扩展）
	case "==", "!=", "<", "<=", ">", ">=":
		return cmp(op, l, r, pos, ctx)
	}
	return NilV(), &RunError{Msg: "internal: unknown operator " + op, Pos: pos, Ctx: ctx}
}

func arith(op string, l, r Value, pos Pos, ctx *execCtx) (Value, error) {
	li, lInt, ri, rInt := l.Int(), l.IsInt(), r.Int(), r.IsInt()
	if lInt && rInt {
		switch op {
		case "+":
			return wrapI32(li + ri), nil
		case "-":
			return wrapI32(int64(li) - int64(ri)), nil
		case "*":
			// 位运算优化：乘 2^n ≡ 左移 n（int32 环绕一致，精确等价）
			if n, ok := pow2(int32(ri)); ok {
				return wrapI32(int64(int32(li) << uint(n))), nil
			}
			return wrapI32(int64(li) * int64(ri)), nil
		case "/":
			if ri == 0 {
				return NilV(), &RunError{Msg: "DivisionByZeroError: integer division by zero", Pos: pos, Ctx: ctx}
			}
			return wrapI32(int64(li) / int64(ri)), nil
		case "%":
			if ri == 0 {
				return NilV(), &RunError{Msg: "DivisionByZeroError: modulo by zero", Pos: pos, Ctx: ctx}
			}
			return wrapI32(int64(li) % int64(ri)), nil
		}
	}
	lf, err := toFloat(l)
	if err != nil {
		return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
	}
	rf, err := toFloat(r)
	if err != nil {
		return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
	}
	switch op {
	case "+":
		return FloatV(lf + rf), nil
	case "-":
		return FloatV(lf - rf), nil
	case "*":
		return FloatV(lf * rf), nil
	case "/":
		if rf == 0 {
			return NilV(), &RunError{Msg: "DivisionByZeroError: float division by zero", Pos: pos, Ctx: ctx}
		}
		return FloatV(lf / rf), nil
	case "%":
		return NilV(), &RunError{Msg: "TypeError: '%' requires int operands", Pos: pos, Ctx: ctx}
	}
	return NilV(), &RunError{Msg: "internal: unknown arithmetic operator " + op, Pos: pos, Ctx: ctx}
}

// wrapI32 truncates to 32-bit two's complement, matching C int overflow.
func wrapI32(x int64) Value { return IntV(int64(int32(x))) }

func toFloat(v Value) (float64, error) {
	if v.IsInt() {
		return float64(v.Int()), nil
	}
	if v.IsFloat() {
		return v.Float(), nil
	}
	return 0, fmt.Errorf("TypeError: arithmetic requires numbers, got %s", v.TypeName())
}

func cmp(op string, l, r Value, pos Pos, ctx *execCtx) (Value, error) {
	if op == "==" || op == "!=" {
		eq, err := equalValues(l, r)
		if err != nil {
			return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
		}
		if op == "==" {
			return BoolV(eq), nil
		}
		return BoolV(!eq), nil
	}
	switch {
	case l.IsInt():
		if r.IsInt() {
			return BoolV(ordCmp(float64(l.Int()), float64(r.Int()), op)), nil
		}
		if r.IsFloat() {
			return BoolV(ordCmp(float64(l.Int()), r.Float(), op)), nil
		}
	case l.IsFloat():
		if r.IsFloat() {
			return BoolV(ordCmp(l.Float(), r.Float(), op)), nil
		}
		if r.IsInt() {
			return BoolV(ordCmp(l.Float(), float64(r.Int()), op)), nil
		}
	case l.IsStr():
		if r.IsStr() {
			return BoolV(ordCmpStr(l.Str(), r.Str(), op)), nil
		}
	}
	return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: cannot order-compare %s and %s", l.TypeName(), r.TypeName()), Pos: pos, Ctx: ctx}
}

func ordCmp(a, b float64, op string) bool {
	switch op {
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	}
	return false
}

func ordCmpStr(a, b, op string) bool {
	switch op {
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	}
	return false
}

// equalValues implements == / != with strict type discipline.
func equalValues(l, r Value) (bool, error) {
	switch {
	case l.IsInt():
		if r.IsInt() {
			return l.Int() == r.Int(), nil
		}
		if r.IsFloat() {
			return float64(l.Int()) == r.Float(), nil
		}
		return false, nil
	case l.IsFloat():
		if r.IsFloat() {
			return l.Float() == r.Float(), nil
		}
		if r.IsInt() {
			return l.Float() == float64(r.Int()), nil
		}
		return false, nil
	case l.IsStr():
		if r.IsStr() {
			return l.Str() == r.Str(), nil
		}
		return false, nil
	case l.IsBool():
		if r.IsBool() {
			return l.Bool() == r.Bool(), nil
		}
		return false, nil
	}
	if l.IsNil() || r.IsNil() {
		return l.IsNil() && r.IsNil(), nil
	}
	return l.tag == r.tag && l.i == r.i && l.ptr == r.ptr, nil
}

func truthy(v Value) (bool, error) {
	if v.IsBool() {
		return v.Bool(), nil
	}
	return false, fmt.Errorf("TypeError: condition must be bool, got %s", v.TypeName())
}

func isCopydType(t string) bool { return strings.Contains(t, "Copyd") }

// ReportError prints an error plus the replay of the execCtx log that
// explains what went wrong (spec §11.2).
func ReportError(err error, w io.Writer) {
	fmt.Fprintf(w, "error: %s\n", err.Error())
	if re, ok := err.(*RunError); ok && re.Ctx != nil && re.Ctx.ensureLog().Size() > 0 {
		fmt.Fprintln(w, "---- execution log ----")
		l := re.Ctx.Log.copyVisible()
		for l.Head() != l.Tail() {
			v, _ := l.Next()
			fmt.Fprintf(w, "  %s\n", v.String())
		}
	}
}
