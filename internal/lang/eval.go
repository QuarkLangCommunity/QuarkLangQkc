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
	slots      []Value // parameter slots (the first len(paramNames) are parameters; linear access avoids hashing)
	paramNames []string
	nParams    int                  // parameter count: the first nParams entries of paramNames are parameters (declare does not overwrite them), the rest are local variables
	refCache   map[string]*refValue // refIdent handle cache: a handle is stateless (it records only scope + name) → reusable
	outer      *scope
}

// newScope lazily allocates vars (a nil map until the first declare — saves an empty-map allocation in hot paths).
func newScope(outer *scope) *scope {
	return &scope{outer: outer}
}

// setParams binds function parameters into linear slots (parameters are usually few; a linear scan beats map hashing).
func (s *scope) setParams(names []string, args []Value) {
	s.paramNames = names
	s.nParams = len(names)
	s.slots = args
}

// paramIndex linearly finds a parameter's slot index (-1 means it is not a parameter).
// refHandle returns a reference handle to `name` in this scope (interned by name so calls do not allocate).
// refIdent handles are stateless (load/store resolve by scope+name each time), so they are safe to reuse.
func (s *scope) refHandle(name string) *refValue {
	if r, ok := s.refCache[name]; ok {
		return r
	}
	r := &refValue{kind: refIdent, sc: s, name: name}
	if s.refCache == nil {
		s.refCache = map[string]*refValue{}
	}
	s.refCache[name] = r
	return r
}

func (s *scope) paramIndex(name string) int {
	for i, n := range s.paramNames {
		if n == name {
			return i
		}
	}
	return -1
}

// findScope returns the scope that defines `name` (used to build reference cells for by-reference arguments; nil = undeclared).
func (s *scope) findScope(name string) *scope {
	for sc := s; sc != nil; sc = sc.outer {
		if sc.paramIndex(name) >= 0 {
			return sc
		}
		if sc.vars != nil {
			if _, ok := sc.vars[name]; ok {
				return sc
			}
		}
	}
	return nil
}

// rawGet reads the raw slot value of `name` (no dereference; used by reference-cell load/store).
func (s *scope) rawGet(name string) Value {
	if i := s.paramIndex(name); i >= 0 {
		return s.slots[i]
	}
	if s.vars != nil {
		if v, ok := s.vars[name]; ok {
			return v
		}
	}
	return NilV()
}

// rawSet writes the raw slot of `name` (does not write through references; false = this scope has no such name).
func (s *scope) rawSet(name string, v Value) bool {
	if i := s.paramIndex(name); i >= 0 {
		s.slots[i] = v
		return true
	}
	if s.vars != nil {
		if _, ok := s.vars[name]; ok {
			s.vars[name] = v
			return true
		}
	}
	return false
}

func (s *scope) declare(name string, v Value, pos Pos) error {
	// The parameter slot already exists (bound by execute) → keep the bound value; a repeated local declaration (re-declared each loop round)
	// → update the slot (otherwise the old value lingers forever, e.g. `int p = html.indexOf(...)` inside a loop would keep the first round's index)
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
		// Reference passing: when a parameter slot holds a reference cell, write through to the caller's argument (by-reference semantics) instead of overwriting the reference itself
		if i := sc.paramIndex(name); i >= 0 {
			if cur := sc.slots[i]; cur.IsRef() {
				if cur.Ref().store(v) {
					return nil
				}
			}
			sc.slots[i] = v
			return nil
		}
		if sc.vars == nil {
			continue
		}
		if cur, ok := sc.vars[name]; ok {
			if cur.IsRef() {
				if cur.Ref().store(v) {
					return nil
				}
			}
			sc.vars[name] = v
			return nil
		}
	}
	return &RunError{Msg: fmt.Sprintf("CompileError: assignment to undeclared identifier %q", name), Pos: pos}
}

func (s *scope) get(name string, pos Pos) (Value, error) {
	for sc := s; sc != nil; sc = sc.outer {
		if pn := sc.paramNames; len(pn) > 0 && pn[0] == name {
			return sc.slots[0].deref(), nil // pass by reference: the read path dereferences uniformly
		}
		if i := sc.paramIndex(name); i >= 0 {
			return sc.slots[i].deref(), nil
		}
		if sc.vars == nil {
			continue
		}
		if v, ok := sc.vars[name]; ok {
			return v.deref(), nil
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

// MemBlock is an allocation block of the global memory manager (spec §14.1).
type MemBlock struct {
	ID          int
	Size        int
	Used        int  // used space (occupancy = Used/Size; <1 means internal free space)
	Dirty       bool // records that the block was modified (dirty flag)
	OwnerPID    int  // owning thread (0 = global)
	Reclaimable bool // nobody occupies it, reclaimable
}

// memHeap keeps blocks in a min-heap by occupancy (ascending Used): the least-occupied block comes first.
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

// MemoryManager manages block allocation, dirty marking and reclamation.
// Linear allocation: prefer the least-occupied block (occupancy < 1) and use its internal free space;
// otherwise request a new block. This is optimal for concurrent, compute-heavy workloads (xmind §memory).
type MemoryManager struct {
	mu          sync.Mutex
	nextID      int
	blocks      map[int]*MemBlock
	minHeap     memHeap // min-heap ordered ascending by occupancy (Used/Size)
	AllocCalls  int64   // total allocation calls
	ReusedCount int64   // reuse count (hits free space inside a block)
	NewBlocks   int64   // number of newly requested blocks
}

func NewMemoryManager() *MemoryManager {
	return &MemoryManager{blocks: map[int]*MemBlock{}}
}

// Alloc: linear allocation: prefer the least-occupied, not-yet-full block; otherwise request a new block.
func (m *MemoryManager) Alloc(size, owner int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AllocCalls++
	// Prefers reuse: the least-occupied block with internal room (Used+size <= BlockSize)
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
		// full, pop it (cannot satisfy the allocation)
		heap.Pop(&m.minHeap)
	}
	// New block: fixed BlockSize, subdivided internally (occupancy = internal used / BlockSize)
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

// ReclaimTask clears a task's block after it finishes and pushes it back into the occupancy heap (linear reuse).
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

// Compact cleans blocks nobody occupies (Used==0) and returns the reclaimed count (not returned at language level).
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

// Fragmentation returns the fragmentation ratio: the unused internal space of live blocks (Used>0).
// When block granularity is fully used or fully empty there is no internal split, so tide/reuse workloads stay at 0 — better than malloc-style fragmentation.
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

// BlockCount returns the current number of blocks (observable from tests).
func (m *MemoryManager) BlockCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.blocks)
}

// Delete pushes a block onto the free queue (occupancy zeroed, data kept) for linear reuse; only clear actually wipes it.
func (m *MemoryManager) Delete(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.blocks[id]; ok {
		b.Used = 0
		b.OwnerPID = 0
		heap.Push(&m.minHeap, b)
	}
}

// Clear really wipes free blocks (Used==0) for data safety; in-use blocks are kept.
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
	Types      map[string]string // member name → type annotation
	TypesOrder []string          // field declaration order (used for positional .{} binding)
}

// InterfaceDef is a registered interface declaration.
type InterfaceDef struct {
	Name    string
	Methods []MethodSig
	Expands []string // expand interface composes interfaces
	Partial bool     // optionally implemented interface (event family): a missing method is not an error (emit ignores it at runtime)
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
	libObjs     map[string]*libObj // library: system library binding object (lazily loaded handle)
	dbg         *dbgState          // --debug breakpoint state (nil = zero overhead)
	sigs        map[string]*signDef
	builtins    map[string]builtinFn
	structs     map[string]*StructDef
	interfaces  map[string]*InterfaceDef
	impls       map[string]*ImplDef
	tasks       map[int]*Task
	taskMu      sync.Mutex
	nextPid     int
	mem         *MemoryManager
	randState   uint64  // rand() LCG state (deterministic pseudo-random)
	globalScope *scope  // global scope (constants pre-declared once)
	fnList      []*Func // function table (fetched directly via CallExpr.FnIdx, no map lookup)
}

// Run executes prog. args are command-line arguments for main(); stdin/stdout
// are the default console streams for io.
// Run executes prog; see runWithInterp.
func Run(prog *Program, filename string, args []string, stdin io.Reader, stdout io.Writer) error {
	_, err := runWithInterp(prog, filename, args, stdin, stdout)
	return err
}

// runWithInterp runs prog and returns the interpreter (tests can observe the memory manager and other internals).
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
	if err := in.registerProgram(prog); err != nil {
		return nil, err
	}

	if prog.Kind == "library" {
		return nil, errors.New(msg("RunError: #error (\"cannot run a library\"): program library; 编译为库，不可运行"))
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

// registerProgram registers a program's declarations into the interpreter: functions/overloads, structs, built-in and user interfaces,
// impls / spaces (instance and static methods separately), IO builtins and FFI library objects.
//
// runWithInterp and the REPL (qkrepl) share this path: registering new declarations per input behaves like a whole program.
func (in *interp) registerProgram(prog *Program) error {
	return in.registerProgramMode(prog, false)
}

// registerProgramReplacing is registerProgram but allows **redefinition to override** (REPL semantics:
// redefining a function/struct with the same name and signature takes effect instead of reporting a duplicate).
func (in *interp) registerProgramReplacing(prog *Program) error {
	return in.registerProgramMode(prog, true)
}

func (in *interp) registerProgramMode(prog *Program, replace bool) error {
	for _, f := range prog.Funcs {
		nfn := &Func{Name: f.Name, Params: f.Params, Ret: f.Ret, Body: f.Body, Pos: f.Pos}
		if cur, dup := in.fns[f.Name]; dup {
			if !sameSig(cur, nfn) {
				if in.overloads == nil {
					in.overloads = map[string][]*Func{}
				}
				replaced := false
				if replace {
					for i, o := range in.overloads[f.Name] {
						if sameSig(o, nfn) {
							in.overloads[f.Name][i] = nfn
							replaced = true
							break
						}
					}
				}
				if !replaced {
					in.overloads[f.Name] = append(in.overloads[f.Name], nfn)
				}
			} else if replace {
				in.fns[f.Name] = nfn
			} else {
				return fmt.Errorf("CompileError: duplicate overload %q", f.Name)
			}
		} else {
			in.fns[f.Name] = nfn
		}
	}
	// Function table (indexed by FnIdx, same order as FnList — filled after fns)
	for _, fd := range prog.FnList {
		in.fnList = append(in.fnList, in.fns[fd.Name])
	}
	for _, s := range prog.Structs {
		if _, dup := in.structs[s.Name]; dup && !replace {
			return fmt.Errorf("CompileError: duplicate struct %q", s.Name)
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
		if _, dup := in.interfaces[i.Name]; dup && !replace {
			return fmt.Errorf("CompileError: duplicate interface %q", i.Name)
		}
		in.interfaces[i.Name] = &InterfaceDef{Name: i.Name, Methods: i.Methods, Expands: i.Expands}
	}
	for _, im := range prog.Impls {
		// A type may have several impl blocks: methods are aggregated (xmind §classes: impl<T> {...} name;)
		key := implKeyOf(im.Type, im.Iface)
		def, exists := in.impls[key]
		if !exists {
			def = &ImplDef{Type: im.Type, Iface: im.Iface, TypeParams: im.TypeParams, Methods: map[string]*Func{}, SelfMethods: map[string]*Func{}}
			in.impls[key] = def
		}
		for _, m := range im.Methods {
			recv := false
			if len(m.Params) > 0 {
				recv = isRecvParam(im.Type, &m.Params[0]) // receiver determined by type (independent of the parameter name)
			}
			fn := &Func{Name: m.Name, Params: m.Params, Ret: m.Ret, Body: m.Body, Pos: m.Pos}
			if recv {
				if _, dup := def.SelfMethods[fn.Name]; dup && !replace {
					return fmt.Errorf("CompileError: duplicate method %q on %s", fn.Name, im.Type)
				}
				def.SelfMethods[fn.Name] = fn
			} else {
				if _, dup := def.Methods[fn.Name]; dup && !replace {
					return fmt.Errorf("CompileError: duplicate method %q on %s", fn.Name, im.Type)
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
	return nil
}

// zeroInstance builds a struct zero-value instance (members take their type's zero value).
func (in *interp) zeroInstance(def *StructDef) *StructValue {
	sv := &StructValue{SType: def.Name, Fields: map[string]Value{}}
	for name, typ := range def.Types {
		sv.Fields[name] = in.zeroValue(typ)
	}
	return sv
}

// baseTypeName strips the generic instantiation from a type annotation (node<int> → node).
func baseTypeName(typ string) string {
	if i := strings.Index(typ, "<"); i >= 0 {
		return typ[:i]
	}
	return typ
}

// zeroValue produces the zero value for a type annotation.
func (in *interp) zeroValue(typ string) Value {
	if strings.HasSuffix(typ, "&") {
		return NilV() // pointer zero value = null
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

	sc := &ctx.sc             // reuse the scope embedded in ctx (avoids a heap allocation per call)
	sc.outer = in.globalScope // global scope (DynamicStackAndHeap and the like pre-declared once, visible through the outer chain)
	fn := ctx.Fn
	// Parameters are bound into linear slots: parameter names are shared/cached (zero allocation) and values reuse ctx.Args directly
	args := ctx.Args
	if flags := fn.CopydFlags(); flags != nil {
		for i, f := range flags {
			if !f || i >= len(args) {
				continue
			}
			v := args[i]
			if v.IsRef() { // callers that build the ctx directly (taskm threads etc.): copyd does not reference
				v = v.deref()
				args[i] = v
			}
			if v.IsCopyd() { // callFunc has already wrapped and deep-copied it
				continue
			}
			args[i] = copydCopy(v)
		}
	}
	sc.setParams(fn.ParamNames(), args)
	// Bytecode path: compiled lazily on first call, only for functions whose whole body fits the
	// supported subset (vm.go). Placed after parameter binding so the slots the VM reads are bound.
	if in.vmFor(fn) != nil {
		err := in.runVM(ctx)
		if err != errVMDeopt {
			return err
		}
		// Blacklist: this function keeps tree-walking from now on. The guard fires before any side
		// effect, so nothing has been printed or recorded yet.
		fn.vm = nil
		sc = &ctx.sc
	}
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

// qkstyleLookup: QSS key lookup with legacy-alias fallback (background-color←bg, font-size←size, font-family←font) — normalizes style-table entries.
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

// errLoopBreak sentinel: break leaves a loop (caught by while/for; bubbling to the top reports break outside loop).
var errLoopBreak = errors.New("loop break")

func (in *interp) execStmt(st Stmt, sc *scope, ctx *execCtx) error {
	// Debug mode: breakpoint check (a single test when dbg == nil, so zero overhead)
	if in.dbg != nil {
		in.hitBreak(stPos(st), sc)
	}
	switch s := st.(type) {
	case *ExprStmt:
		_, err := in.evalExpr(s.X, sc, ctx)
		return err
	case *DeleteStmt:
		// delete: run __delete__() first (if any), then push onto the free queue (data kept; only clear wipes it)
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
			// List joins the free queue (allocate first when there is no block, so it can be recorded)
			id := l.blockID
			if id == 0 {
				id = in.mem.Alloc(globalMemory.BlockSize, 0)
			}
			in.mem.Delete(id)
		}
		return nil
	case *LogStmt:
		// log records a log entry and ends the function (any value, nil by default)
		v, err := in.evalExpr(s.X, sc, ctx)
		if err != nil {
			return err
		}
		ctx.ensureLog().Append(StrV(v.String()))
		ctx.result = NilV()
		return errReturn
	case *TryStmt:
		// try/catch: when the try block errors (not a return), bind the error into the catch variable (interface{}) and run the catch block
		err := in.execBlock(s.Try, sc, ctx)
		if err != nil {
			if err == errReturn {
				return err
			}
			inner := newScope(sc)
			_ = inner.declare(s.CatchVar, StrV(err.Error()), s.Pos) // the error is bound into the declared type (interface{} etc.), free-form system
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
		// C style: for (<init>; <cond>; <step>) { ... }
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
		// copyd modifier: the declaration is a Copyd value right away (same as the [Copyd] annotation: copy on pass, .ptr() unwraps)
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
			if t.Slot > 0 {
				if i := int(t.Slot) - 1; i < len(sc.slots) && i < len(sc.paramNames) && sc.paramNames[i] == t.Name {
					if cur := sc.slots[i]; cur.IsRef() { // pass by reference: write through to the target (consistent with scope.set)
						if cur.Ref().store(v) {
							return nil
						}
					}
					sc.slots[i] = v
					return nil
				}
			}
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
					return &RunError{Msg: msg("TypeError: 列表索引必须是 int"), Pos: s.Pos, Ctx: ctx}
				}
				return obj.List().setIndex(int(key.Int()), v)
			}
			return &RunError{Msg: fmt.Sprintf(msg("TypeError: 不支持对 %s 索引赋值"), obj.TypeName()), Pos: s.Pos, Ctx: ctx}
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
		// new <type>[size]: allocate on the heap (block allocation); invalid size → badAlloc
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
		// allocate on the heap (managed by blocks)
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
		// Positional form (empty Name): bind by the target type's field order (same as the compiler)
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
		// Slot fast path: resolved at compile time and name-verified at runtime → direct index (no linear name scan)
		if x.Slot > 0 {
			if i := int(x.Slot) - 1; i < len(sc.slots) && i < len(sc.paramNames) && sc.paramNames[i] == x.Name {
				return sc.slots[i].deref(), nil
			}
		}
		v, err := sc.get(x.Name, x.Pos)
		if err == nil {
			return v, nil
		}
		if fn, ok := in.fns[x.Name]; ok {
			return FuncV(&FuncValue{fn: fn}), nil
		}
		// Builtin functions used as function references (sum(rand, ...))
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
		// === is the storage-identity primitive: it compares where the operands live, never their
		// values, and it is deliberately not overloadable (no operation method, no user hook).
		if x.Op == "===" {
			return in.storageIdentical(x.L, x.R, sc), nil
		}
		l, err := in.evalExpr(x.L, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		// Operation overloading: both sides are the same struct and an aggregate method matches → call the protocol method
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
		// Fast path: int-only arithmetic/shift goes straight through (skips binOp dispatch; the hottest path for fib/loops)
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
		return evalMember(c.V, name, pos, ctx) // Copyd pass-through
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
	// Builtin signature @styleConfigure(file): read the JSON file → merge into the first argument node's style, then run the call
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
			return NilV(), &RunError{Msg: msg("TypeError: @styleConfigure(file) 需要 file 或 String"), Pos: c.Pos, Ctx: ctx}
		}
		// @styleConfigure is meant for setStyle: read file → file contents as string → hand to setStyle(string)
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
			// Find the style field of the first argument (receiver for member calls, first argument for plain calls) and merge
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
		mark := len(ctx.argArena)
		defer func() { ctx.argArena = ctx.argArena[:mark] }()
		argVals, err := in.evalCallArgs(c, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		argVals = derefArgs(argVals) // the signature mechanism records the in list by value
		fn := in.bestMatchV(in.allDefs(id.Name), argVals)
		if fn == nil {
			return NilV(), &RunError{Msg: overloadErr(in.allDefs(id.Name), id.Name, len(argVals)), Pos: id.Pos, Ctx: ctx}
		}
		if len(argVals) != len(fn.Params) {
			return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(argVals)), Pos: id.Pos, Ctx: ctx}
		}
		// v2 signature: f(args) @instance(prefix) ≡ instance.call(prefix)(.{in, out})
		// 1) instance is a variable (a Sign instance, any name); 2) the prefix carries the wrapped function fn; 3) record .{in: List(args), out: nil}
		mbv, err := in.evalExpr(&Ident{Name: c.Sign.Name, Pos: c.Pos}, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		// Build the prefix: the original function lives in the prefix (plus any explicit arguments at the @ site)
		prefix := &StructValue{SType: ".", Fields: map[string]Value{"fn": FuncV(&FuncValue{fn: fn})}}
		for _, a := range c.Sign.Args {
			av, err := in.evalExpr(a, sc, ctx)
			if err != nil {
				return NilV(), err
			}
			prefix.Fields["prefix"] = av
		}
		// Record .{in, out}
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
		mark := len(ctx.argArena)
		args, err := in.evalCallArgs(c, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		res, err := in.callMethod(obj, m.Name, args, ctx, m.Pos)
		ctx.argArena = ctx.argArena[:mark]
		return res, err
	}

	id, ok := c.Fn.(*Ident)
	if !ok {
		return NilV(), &RunError{Msg: "TypeError: this expression is not callable", Pos: c.Pos, Ctx: ctx}
	}
	mark := len(ctx.argArena)
	argVals, err := in.evalCallArgs(c, sc, ctx)
	if err != nil {
		return NilV(), err
	}
	res, err := in.dispatchPlainCall(c, id, argVals, sc, ctx)
	ctx.argArena = ctx.argArena[:mark] // return the argument reuse area (the values are already bound into the callee's slots)
	return res, err
}

// dispatchPlainCall resolves f(args) in four ways: compile-time index / overloads / function-reference variable / builtin.
func (in *interp) dispatchPlainCall(c *CallExpr, id *Ident, argVals []Value, sc *scope, ctx *execCtx) (Value, error) {
	// FnIdx was resolved at compile time: with no overloads take fnList directly (skips the map-hash hot path)
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
	// Function-reference variable: f is a variable holding a FuncValue → call it
	if v, err := sc.get(id.Name, id.Pos); err == nil {
		if v.IsFunc() {
			fv := v.Func()
			return in.callFunc(fv.fn, argVals, id.Pos, ctx.depth)
		}
	}
	if b, ok := in.builtins[id.Name]; ok {
		return b(derefArgs(argVals), id.Pos, ctx) // builtins accept values only
	}
	return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: undeclared function %q", id.Name), Pos: id.Pos, Ctx: ctx}
}

// callFunc (v2): a call returns what return produced (a log-terminated function returns nil).
func (in *interp) callFunc(fn *Func, args []Value, pos Pos, parentDepth int) (Value, error) {
	// Builtin function references (such as sum's generator rand): only for pseudo functions (no Body) — a user method with the same name is not hijacked
	if fn.Body == nil {
		if b, ok := in.builtins[fn.Name]; ok {
			return b(derefArgs(args), pos, nil) // builtins accept values only
		}
	}
	if len(args) != len(fn.Params) {
		return NilV(), &RunError{Msg: fmt.Sprintf("CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(args)), Pos: pos}
	}
	if parentDepth >= 8192 {
		return NilV(), &RunError{Msg: "StackOverflowError: recursion depth exceeded 8192", Pos: pos}
	}
	// Parameter binding (the only canonical form):
	//   ① plain parameter → an lvalue argument binds by reference (the reference lives in the slot; reads/writes go through to the caller);
	//      a non-lvalue argument is a temporary cell, so callee writes have no side effect;
	//   ② copyd parameter (type int[Copyd]/Copyd<T> or the copyd modifier) → deep-copied before binding,
	//      never written back to the caller: the type form binds the Copyd wrapper (readable via .ptr()), the modifier form binds the bare value;
	//   ③ the argument itself is a copyd-declared Copyd value while the parameter is not copyd → unwrap and deep copy (copy on pass).
	flags := fn.CopydFlags()
	for i := range args {
		flagged := flags != nil && i < len(flags) && flags[i]
		if args[i].IsCopyd() { // reference transparency: a reference to a Copyd value is handled as a Copyd value
			cv := args[i]
			if cv.IsRef() {
				cv = cv.deref()
			}
			if flagged {
				args[i] = cv // already a Copyd value: bind as is (no further copy)
			} else {
				args[i] = deepCopy(cv.Copyd().V)
			}
			continue
		}
		if flagged {
			v := args[i]
			if v.IsRef() {
				v = v.deref() // a copyd parameter is not dereferenced/referenced: take the argument's current value
			}
			if fn.paramWrapCopyd(i) {
				args[i] = CopydV(&CopydValue{V: copydCopy(v)})
			} else {
				args[i] = copydCopy(v)
			}
		}
	}
	ctx := in.newCtx(fn, args, pos)
	defer in.putCtx(ctx)
	ctx.depth = parentDepth + 1
	if err := in.execute(ctx); err != nil {
		return NilV(), err
	}
	if ctx.result.IsNil() {
		return NilV(), nil // a path without return (log ends etc.) yields nil
	}
	return ctx.result, nil
}

// evalArgs evaluates an argument list: lvalue arguments (variable / struct field / List index) produce reference values,
// everything else is by value — parameters always bind by reference (except copyd parameters, see callFunc).
// evalArgs evaluates the argument list. The resulting slice comes from ctx.argArena (no per-call allocation):
// the caller must restore `ctx.argArena = ctx.argArena[:mark]` **after the call ends** (take mark = len before the call).
// Nested evaluation is naturally LIFO: an inner call truncates at its own mark and never overwrites the outer arguments.
// storageIdentical implements ===: it reports whether two operand expressions occupy the same
// storage space. Both sides must be lvalues that resolve to the same location; anything else (a
// literal, a temporary, two distinct variables — even ones holding equal values or pointers to the
// same target) is false.
//
// This is why a pointer compared with === is false while == is true: == compares values (for
// pointers, the target), === compares the storage the operands themselves occupy.
func (in *interp) storageIdentical(l, r Expr, sc *scope) Value {
	li, lok := in.storageID(l, sc)
	if !lok {
		return BoolV(false)
	}
	ri, rok := in.storageID(r, sc)
	if !rok {
		return BoolV(false)
	}
	return BoolV(li == ri)
}

// storageID returns a stable identity string for an lvalue expression's storage. It only reads
// through already-resolved variables and pure member/index paths, so no user code runs and no side
// effect can be observed by evaluating ===.
func (in *interp) storageID(e Expr, sc *scope) (string, bool) {
	switch x := e.(type) {
	case *Ident:
		if x.Slot > 0 {
			if i := int(x.Slot) - 1; i < len(sc.slots) && i < len(sc.paramNames) && sc.paramNames[i] == x.Name {
				return fmt.Sprintf("slot:%p:%d", sc, i), true
			}
		}
		if owner := sc.findScope(x.Name); owner != nil {
			if i := owner.paramIndex(x.Name); i >= 0 {
				return fmt.Sprintf("slot:%p:%d", owner, i), true
			}
			if owner.vars != nil {
				if _, ok := owner.vars[x.Name]; ok {
					return fmt.Sprintf("var:%p:%s", owner, x.Name), true
				}
			}
		}
		return "", false
	case *MemberExpr:
		obj, err := in.evalExpr(x.X, sc, &execCtx{Log: NewList()})
		if err != nil {
			return "", false
		}
		if obj.IsCopyd() {
			obj = obj.Copyd().V
		}
		if !obj.IsStruct() {
			return "", false
		}
		if _, ok := obj.Struct().Fields[x.Name]; !ok {
			return "", false
		}
		return fmt.Sprintf("field:%p:%s", obj.Struct(), x.Name), true
	case *IndexExpr:
		obj, err := in.evalExpr(x.X, sc, &execCtx{Log: NewList()})
		if err != nil {
			return "", false
		}
		idx, err := in.evalExpr(x.Idx, sc, &execCtx{Log: NewList()})
		if err != nil || !idx.IsInt() {
			return "", false
		}
		switch {
		case obj.IsList():
			return fmt.Sprintf("elem:%p:%d", obj.List(), idx.Int()), true
		case obj.IsStr():
			return "", false // strings are immutable: no element storage to identify
		}
		return "", false
	}
	return "", false
}

// isLValueExpr reports whether an expression can produce a reference cell in evalArg (declared
// variable, struct member, list index). Anything it lists is treated conservatively: those argument
// lists keep going through evalArg.
func isLValueExpr(e Expr) bool {
	switch e.(type) {
	case *Ident, *MemberExpr, *IndexExpr:
		return true
	}
	return false
}

// evalCallArgs evaluates a call's arguments into the ctx arena. For argument lists that cannot form
// lvalues (the common case: literals, arithmetic, nested calls) it skips evalArg's lvalue probe and
// evaluates each argument directly — same values, one dispatch less per argument.
func (in *interp) evalCallArgs(c *CallExpr, sc *scope, ctx *execCtx) ([]Value, error) {
	if !c.argsClassified {
		c.plainArgs = true
		for _, a := range c.Args {
			if isLValueExpr(a) {
				c.plainArgs = false
				break
			}
		}
		c.argsClassified = true
	}
	if !c.plainArgs {
		return in.evalArgs(c.Args, sc, ctx) // fallback: the general path with lvalue/reference handling
	}
	start := len(ctx.argArena)
	for _, a := range c.Args {
		v, err := in.evalExpr(a, sc, ctx)
		if err != nil {
			ctx.argArena = ctx.argArena[:start]
			return nil, err
		}
		ctx.argArena = append(ctx.argArena, v)
	}
	return ctx.argArena[start:], nil
}

func (in *interp) evalArgs(args []Expr, sc *scope, ctx *execCtx) ([]Value, error) {
	start := len(ctx.argArena)
	for _, a := range args {
		v, err := in.evalArg(a, sc, ctx)
		if err != nil {
			ctx.argArena = ctx.argArena[:start]
			return nil, err
		}
		ctx.argArena = append(ctx.argArena, v)
	}
	return ctx.argArena[start:], nil
}

// evalArg evaluates one argument: anything that can form an lvalue cell returns a reference value (written back to the caller), otherwise by value.
func (in *interp) evalArg(a Expr, sc *scope, ctx *execCtx) (Value, error) {
	switch x := a.(type) {
	case *Ident:
		// Declared variable → reference cell (an undeclared name may be a function reference; fall back to by-value)
		if x.Slot > 0 { // slot already resolved: the owning scope is the current scope, so no chained findScope scan is needed
			if i := int(x.Slot) - 1; i < len(sc.slots) && i < len(sc.paramNames) && sc.paramNames[i] == x.Name {
				return RefV(sc.refHandle(x.Name)), nil
			}
		}
		if owner := sc.findScope(x.Name); owner != nil {
			return RefV(owner.refHandle(x.Name)), nil
		}
	case *MemberExpr:
		obj, err := in.evalExpr(x.X, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		if obj.IsCopyd() {
			obj = obj.Copyd().V // Copyd pass-through (consistent with evalMember)
		}
		if obj.IsStruct() {
			if _, ok := obj.Struct().Fields[x.Name]; ok {
				return RefV(&refValue{kind: refMember, obj: obj, name: x.Name}), nil
			}
		}
		return evalMember(obj, x.Name, x.Pos, ctx)
	case *IndexExpr:
		obj, err := in.evalExpr(x.X, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		idx, err := in.evalExpr(x.Idx, sc, ctx)
		if err != nil {
			return NilV(), err
		}
		if !obj.IsList() {
			return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: indexing requires a List, got %s", obj.TypeName()), Pos: x.Pos, Ctx: ctx}
		}
		if !idx.IsInt() {
			return NilV(), &RunError{Msg: "TypeError: index must be int", Pos: x.Pos, Ctx: ctx}
		}
		// Take the value with the original semantics first (including bounds checks), then build the element reference
		if _, gerr := obj.List().Get(int(idx.Int())); gerr != nil {
			return NilV(), &RunError{Msg: gerr.Error(), Pos: x.Pos, Ctx: ctx}
		}
		return RefV(&refValue{kind: refIndex, obj: obj, key: idx}), nil
	}
	return in.evalExpr(a, sc, ctx)
}

func (in *interp) callMethod(obj Value, name string, args []Value, ctx *execCtx, pos Pos) (Value, error) {
	// Copyd pass-through (moved ahead of type dispatch; mutually exclusive with the List/Table decisions)
	if obj.IsCopyd() {
		if name == "ptr" {
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return obj.Copyd().V, nil // .ptr() extracts the address wrapped by Copyd
		}
		return in.callMethod(obj.Copyd().V, name, args, ctx, pos)
	}
	// Non-struct receivers (List/Table/String/IO/FFI/…): builtin methods take values only, so arguments are dereferenced.
	// Struct receivers go through user impl methods: reference arguments are kept (parameters bind by reference).
	if !obj.IsStruct() {
		args = derefArgs(args)
	}
	if obj.IsLib() {
		return in.callLibMethod(obj.Lib(), name, args, pos, ctx)
	}
	if obj.IsPtr() { // FFI opaque handle: only toString is supported (hexadecimal)
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
				return NilV(), &RunError{Msg: msg("TypeError: get(i) 需要 int 下标"), Pos: pos, Ctx: ctx}
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
	} else if obj.IsTaskm() {
		_ = obj.Taskm() // taskm is a global singleton; its methods go through in
		switch name {
		case "spawn":
			// v2: taskm.spawn() takes no arguments; it creates a thread and returns its pid
			if len(args) != 0 {
				return NilV(), wantArity("taskm.spawn", 0, len(args), pos, ctx)
			}
			pid := in.newThread()
			t, _ := in.lookupTask(pid)
			return ThreadV(&ThreadValue{Pid: pid, t: t}), nil
		case "merge":
			// v2: taskm.merge(pid, fn, args...) runs a function on thread pid
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
			// v2: taskm.block(pid) returns void (it only waits until the thread is idle)
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
			return NilV(), nil // block returns void
		case "done":
			// done(pid): is that task's thread idle (no function occupying it)? v0.1 = has the task finished?
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
			return BoolV(idle), nil // done = whether the thread is idle (no function occupying it)
		case "channel":
			cap := 1024 // default capacity (spec §14.2)
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
				return NilV(), &RunError{Msg: msg("TypeError: thread.talk 需要 channel 类实例"), Pos: pos, Ctx: ctx}
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
				return NilV(), &RunError{Msg: msg("TypeError: contains 需要 String 参数"), Pos: pos, Ctx: ctx}
			}
			return BoolV(strings.Contains(o, args[0].Str())), nil
		case "startsWith":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() {
				return NilV(), &RunError{Msg: msg("TypeError: startsWith 需要 String 参数"), Pos: pos, Ctx: ctx}
			}
			return BoolV(strings.HasPrefix(o, args[0].Str())), nil
		case "endsWith":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() {
				return NilV(), &RunError{Msg: msg("TypeError: endsWith 需要 String 参数"), Pos: pos, Ctx: ctx}
			}
			return BoolV(strings.HasSuffix(o, args[0].Str())), nil
		case "indexOf":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsStr() {
				return NilV(), &RunError{Msg: msg("TypeError: indexOf 需要 String 参数"), Pos: pos, Ctx: ctx}
			}
			return IntV(int64(strings.Index(o, args[0].Str()))), nil // -1 = not found
		case "substring":
			if len(args) < 1 || len(args) > 2 || !args[0].IsInt() {
				return NilV(), &RunError{Msg: msg("TypeError: substring(start, end?) 需要 int 参数"), Pos: pos, Ctx: ctx}
			}
			// Extreme optimization: no full []rune conversion — decode the prefix to a byte offset and slice there (zero copy)
			n := int64(utf8.RuneCountInString(o))
			start := args[0].Int()
			end := n
			if len(args) == 2 {
				if !args[1].IsInt() {
					return NilV(), &RunError{Msg: msg("TypeError: substring 的 end 必须是 int"), Pos: pos, Ctx: ctx}
				}
				end = args[1].Int()
			}
			if start < 0 || end < start || end > n {
				return NilV(), &RunError{Msg: fmt.Sprintf(msg("StringIndexOutOfBoundsError: substring(%d, %d) 越界 [0,%d]"), start, end, n), Pos: pos, Ctx: ctx}
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
				return NilV(), &RunError{Msg: msg("TypeError: split 需要 String 分隔符"), Pos: pos, Ctx: ctx}
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
				return NilV(), &RunError{Msg: msg("TypeError: replace(old, new) 需要 String 参数"), Pos: pos, Ctx: ctx}
			}
			return StrV(strings.ReplaceAll(o, args[0].Str(), args[1].Str())), nil
		case "charAt":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsInt() {
				return NilV(), &RunError{Msg: msg("TypeError: charAt 需要 int 索引"), Pos: pos, Ctx: ctx}
			}
			// Extreme optimization: decode the prefix only up to the target rune, i.e. exactly i characters
			i := args[0].Int()
			b := 0
			for k := int64(0); k < i; k++ {
				if b >= len(o) {
					return NilV(), &RunError{Msg: fmt.Sprintf(msg("StringIndexOutOfBoundsError: charAt(%d) 越界 [0,%d)"), i, utf8.RuneCountInString(o)), Pos: pos, Ctx: ctx}
				}
				_, sz := utf8.DecodeRuneInString(o[b:])
				b += sz
			}
			if b >= len(o) {
				return NilV(), &RunError{Msg: fmt.Sprintf(msg("StringIndexOutOfBoundsError: charAt(%d) 越界 [0,%d)"), i, utf8.RuneCountInString(o)), Pos: pos, Ctx: ctx}
			}
			// align to a character boundary
			_, sz := utf8.DecodeRuneInString(o[b:])
			return StrV(o[b : b+sz]), nil
		case "toInt":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			v, err := strconv.ParseInt(strings.TrimSpace(o), 10, 32)
			if err != nil {
				return NilV(), &RunError{Msg: fmt.Sprintf(msg("ParseError: %q 不是合法整数"), o), Pos: pos, Ctx: ctx}
			}
			return IntV(v), nil
		case "toFloat":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(o), 64)
			if err != nil {
				return NilV(), &RunError{Msg: fmt.Sprintf(msg("ParseError: %q 不是合法浮点数"), o), Pos: pos, Ctx: ctx}
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
				return NilV(), &RunError{Msg: msg("TypeError: call 第一参数必须是 prefix 记录"), Pos: pos, Ctx: ctx}
			}
			pref := args[0].Struct()
			if !args[1].IsStruct() {
				return NilV(), &RunError{Msg: msg("TypeError: call 第二参数必须是 .{in,out} 记录"), Pos: pos, Ctx: ctx}
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
			in.mem.Clear() // globalMemory.clear(): cleans directly from the modification log
			return NilV(), nil
		case "mode":
			// Experimental: GlobalMemory.mode(DynamicStackAndHeap) allocates the stack and heap dynamically
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			return NilV(), nil
		case "compact":
			if err := wantArity(name, 0, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			// compact() never returns at all (spec §14.1); it actually cleans blocks nobody occupies
			in.mem.Compact()
			return NilV(), nil
		case "setBlock":
			if err := wantArity(name, 1, len(args), pos, ctx); err != nil {
				return NilV(), err
			}
			if !args[0].IsInt() || args[0].Int() < 1 || args[0].Int() > 1<<20 {
				return NilV(), &RunError{Msg: "TypeError: setBlock(n) requires a positive int block size", Pos: pos, Ctx: ctx}
			}
			o.BlockSize = int(args[0].Int()) // dynamically adjust the block dirty-flag granularity
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
	mark := len(ctx.argArena)
	defer func() { ctx.argArena = ctx.argArena[:mark] }()
	args, err := in.evalArgs(x.Args, sc, ctx)
	if err != nil {
		return NilV(), err
	}
	// Builtin space methods take values only (reference arguments are dereferenced); user space static methods keep reference passing.
	switch x.Scope {
	case "memorize", "HashTable", "List", "taskm", "IO", "file":
		args = derefArgs(args)
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
		// taskm is a global variable: the correct syntax is taskm.spawn(...) / taskm.block(...) etc.
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

// allDefs returns every definition of a function (primary + overloads).
func (in *interp) allDefs(name string) []*Func {
	if fn, ok := in.fns[name]; ok {
		return append([]*Func{fn}, in.overloads[name]...)
	}
	return in.overloads[name]
}

// bestMatchV picks the best overload by argument values (matching arity first; identical type kinds score highest; a unique best wins).
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

// overloadErr builds a compatible error when no overload matches (an arity mismatch keeps the expects wording).
func overloadErr(defs []*Func, name string, n int) string {
	for _, d := range defs {
		if len(d.Params) != n {
			return fmt.Sprintf("CompileError: %s expects %d args, got %d", name, len(d.Params), n)
		}
	}
	return fmt.Sprintf(msg("CompileError: 未找到匹配重载 %q（参数类型不匹配）"), name)
}

// selfMethodOf looks up an instance method across aggregated impls (self first parameter).
func (in *interp) selfMethodOf(typ, name string) *Func {
	for _, d := range in.implDefsFor(typ) {
		if fn := d.SelfMethods[name]; fn != nil {
			return fn
		}
	}
	return nil
}

// staticMethodOf looks up a static method across aggregated impls (no self parameter).
func (in *interp) staticMethodOf(typ, name string) *Func {
	for _, d := range in.implDefsFor(typ) {
		if fn := d.Methods[name]; fn != nil {
			return fn
		}
	}
	return nil
}

// fileInputStreamBuiltin opens a file input stream (shared by ifstream/FileInputStream).
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

// fileOutputStreamBuiltin creates a file output stream (shared by ofstream/FileOutputStream).
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

// sumBuiltin: sum(generate, begin, stop, step?) — the summation optimization.
// A linear generator (constant second difference) uses the closed form n*(first+last)/2 (multiply-add, O(1));
// anything non-linear degrades to a loop. The bit-level permutation the user proposed (rearranging per-bit 1-counts into a multiply-add)
// cannot see the generator's internal bit pattern at runtime, and the linear closed form already covers the common sum(index)/sum(a*i+b) cases.
func (in *interp) sumBuiltin(args []Value, pos Pos, ctx *execCtx) (Value, error) {
	if len(args) != 3 && len(args) != 4 {
		return NilV(), wantArity("sum", 3, len(args), pos, ctx)
	}
	if !args[0].IsFunc() {
		return NilV(), &RunError{Msg: msg("TypeError: sum 第一个参数必须是函数引用 generate"), Pos: pos, Ctx: ctx}
	}
	// Bit-level permutation: a uniform random generator (rand) has n/2 ones per column in expectation → multiply-add closed form O(1)
	// each position is 1 with probability 1/2, so after any permutation every column is uniform: Σ = Σ_k (n/2)·2^k = n·2^30
	gen := args[0].Func()
	if gen.fn.Name == "rand" {
		if args[1].IsInt() && args[2].IsInt() {
			begin, stop := args[1].Int(), args[2].Int()
			{
				n := stop - begin
				if n > 0 {
					return wrapI32(n << 30), nil // n × 2^30 (the uniform expectation over [0,2^31-1])
				}
				return IntV(0), nil
			}
		}
	}
	if !args[1].IsInt() {
		return NilV(), &RunError{Msg: msg("TypeError: sum 的 begin 必须是 int"), Pos: pos, Ctx: ctx}
	}
	if !args[2].IsInt() {
		return NilV(), &RunError{Msg: msg("TypeError: sum 的 stop 必须是 int"), Pos: pos, Ctx: ctx}
	}
	begin, stop := args[1].Int(), args[2].Int()
	step := int64(1)
	if len(args) == 4 {
		if args[3].IsInt() && args[3].Int() != 0 {
			step = args[3].Int()
		} else {
			return NilV(), &RunError{Msg: msg("TypeError: sum 的 step 必须是非零 int"), Pos: pos, Ctx: ctx}
		}
	}
	g := func(i int64) (int64, error) {
		v, err := in.callFunc(gen.fn, []Value{IntV(i)}, pos, ctx.depth)
		if err != nil {
			return 0, err
		}
		if !v.IsInt() {
			return 0, &RunError{Msg: msg("TypeError: generate 必须返回 int"), Pos: pos, Ctx: ctx}
		}
		return v.Int(), nil
	}
	// Linear probing: constant second difference → closed form O(1) (the 3-argument form with default step=1 probes the same way)
	if true {
		g0, _ := g(begin)
		g1, _ := g(begin + step)
		g2, _ := g(begin + 2*step)
		if d1, d2 := g1-g0, g2-g1; d1 == d2 {
			n := (stop - begin + step - 1) / step
			if n <= 0 {
				return IntV(0), nil
			}
			// Verify the last term: a periodic function (such as n%3) can coincidentally match at three points; if the last term disagrees it is non-linear
			last := g0 + d1*(n-1)
			if actual, err := g(begin + (n-1)*step); err == nil && actual == last {
				return wrapI32(n * (g0 + last) / 2), nil // multiply-add closed form (int = 32-bit, wrap)
			}
		}
	}
	// Bit-level permutation: when the generator sequence has period P → precompute per-bit counts over that period, multiply-add in O(P+bits) (far faster than looping when P << n)
	if n := (stop - begin + step - 1) / step; n > 0 {
		if p, ok := detectPeriod(g, begin, step, n); ok && p < n {
			// Per-bit counts over one period: counts[k] = how many times bit k is 1 within the period
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
			// Add the remainder term by term
			for j := int64(0); j < r; j++ {
				v, err := g(begin + j*step)
				if err != nil {
					return NilV(), err
				}
				sum += v
			}
			return wrapI32(sum), nil // periodic bit-level permutation closed form: int = 32-bit wrap
		}
	}
	// No short period (e.g. a full-period 32-bit random number): add term by term (faster than per-bit counting)
	var total int64
	for i := int64(begin); i < int64(stop); i += int64(step) {
		v, err := g(i)
		if err != nil {
			return NilV(), err
		}
		total += v
	}
	return wrapI32(total), nil // degenerate loop / period fallback: int = 32-bit wrap
}

// detectPeriod detects the generator sequence's period (probes at most 65536 terms with the given step).
func detectPeriod(g func(int64) (int64, error), begin, step, n int64) (int64, bool) {
	// Probe only short periods (≤256): a true random has none, so it fails fast and goes straight to addition (O(n) is already the lower bound)
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
			// Verify one more bit to confirm the period
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
		return IntV(int64(int32(time.Now().UnixMicro()))), nil // microseconds
	}
	in.builtins["rand"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 0 {
			return NilV(), wantArity("rand", 0, len(args), pos, ctx)
		}
		// LCG: deterministic pseudo-random in [0, 2^31-1]
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
		// Two-way file stream: read + write
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
	// [official library system primitive] process execution: qkexec(cmd) -> exit code (shell -c)
	in.builtins["qkexec"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) < 1 || len(args) > 2 || !args[0].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qkexec(cmd String[, retries int])", Pos: pos, Ctx: ctx}
		}
		retries := retriesOf(args, pos, ctx)
		var code int64
		for attempt := 0; attempt <= retries; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(200*(1<<uint(attempt-1))) * time.Millisecond) // exponential backoff
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
				return NilV(), &RunError{Msg: fmt.Sprintf(msg("IOError: 无法执行命令：%v"), err), Pos: pos, Ctx: ctx}
			}
			code = 0
			break
		}
		return IntV(code), nil
	}
	// qkexecv(prog, args List<String>) -> exit code (no shell; argv passed directly)
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
				return NilV(), &RunError{Msg: msg("TypeError: qkexecv 参数必须是 List<String>"), Pos: pos, Ctx: ctx}
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
				return NilV(), &RunError{Msg: fmt.Sprintf(msg("IOError: 无法启动程序：%v"), err), Pos: pos, Ctx: ctx}
			}
			code = 0
			break
		}
		return IntV(code), nil
	}
	// qkpopen(cmd) -> InputStream (captures stdout; 8 MB cap)
	in.builtins["qkpopen"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 1 || !args[0].IsStr() {
			return NilV(), &RunError{Msg: msg("TypeError: qkpopen(cmd String) 需要一个字符串命令"), Pos: pos, Ctx: ctx}
		}
		cmd := exec.Command("sh", "-c", args[0].Str())
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf(msg("IOError: 命令执行失败：%v"), err), Pos: pos, Ctx: ctx}
		}
		if len(out) > 8<<20 {
			return NilV(), &RunError{Msg: msg("IOError: qkpopen 输出超过 8MB 上限"), Pos: pos, Ctx: ctx}
		}
		return InV(&InputStream{R: bytes.NewReader(out)}), nil
	} // [actions library primitive] network layer: qkhttp_get(url) -> String (10s timeout, 8MiB response cap)
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
			return NilV(), &RunError{Msg: fmt.Sprintf(msg("IOError: 请求失败：%v"), err), Pos: pos, Ctx: ctx}
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf(msg("IOError: 读取响应失败：%v"), err), Pos: pos, Ctx: ctx}
		}
		if len(body) > 8<<20 {
			return NilV(), &RunError{Msg: msg("IOError: 响应超过 8MiB 上限"), Pos: pos, Ctx: ctx}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return NilV(), &RunError{Msg: fmt.Sprintf(msg("HTTPError: 状态码 %d"), resp.StatusCode), Pos: pos, Ctx: ctx}
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
			return NilV(), &RunError{Msg: fmt.Sprintf(msg("IOError: 请求失败：%v"), err), Pos: pos, Ctx: ctx}
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
		if err != nil {
			return NilV(), &RunError{Msg: fmt.Sprintf(msg("IOError: 读取响应失败：%v"), err), Pos: pos, Ctx: ctx}
		}
		if len(body) > 8<<20 {
			return NilV(), &RunError{Msg: msg("IOError: 响应超过 8MiB 上限"), Pos: pos, Ctx: ctx}
		}
		return StrV(string(body)), nil
	}

	// [json library primitives] json serialization/deserialization
	in.builtins["qkjson_dumps"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) != 1 {
			return NilV(), &RunError{Msg: msg("TypeError: qkjson_dumps(v) 需要一个参数"), Pos: pos, Ctx: ctx}
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
	// [cleg screen host] cross-platform window presentation (X11/GDI)
	// [file primitives] read/write whole files
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
	// [cleg style parsing] qkcleg_style_parse(styleTable, jsonText): JSON string → HashTable → merged into style
	// [cleg style loading] qkcleg_style_load(styleTable, path): read a JSON file → HashTable → merged into style
	// [cleg rendering primitives] CPU raster framebuffer (at the performance limit: linear u32, preallocated, zero-allocation hot path)

	// [cleg auto repaint] qkcleg_auto(node) registers; qkcleg_tick() renders them one by one (frame-loop model)
	// [cleg signals] qksignal_emit(node, name): calls node methods such as onClicked when implemented
	in.builtins["qksignal_emit"] = func(args []Value, pos Pos, ctx *execCtx) (Value, error) {
		if len(args) < 2 || !args[0].IsStruct() || !args[1].IsStr() {
			return NilV(), &RunError{Msg: "TypeError: qksignal_emit(node, name String [, args...])", Pos: pos, Ctx: ctx}
		}
		node := args[0]
		name := args[1].Str()
		// Normalize short names: clicked → onClicked (try the full name first, otherwise on+capitalized)
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
	// [cleg qss reading] primitive version: get/num/cr
}

// qkjsonFromGo converts a json.Unmarshal result into a Value (HashTable / List / scalar; integral values become int).
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
			h.m[hashKey(StrV(k))] = v // keys normalized to the hashKey format (consistent with Put, so get/contains can find them)
		}
		return TableV(h), nil
	}
	return StrV(fmt.Sprintf("%v", raw)), nil
}

// qkjsonV converts a Value into a JSON-serializable Go object (round-trip safe).
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
		// Deep-copy values by walking the internal map directly; keys are restored to native keys (stripping hashKey's "TypeName:" prefix)
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
		return v.String(), nil // other types are turned into a string for JSON
	}
}

// retriesOf reads the optional retries argument (default retries=1, i.e. one more attempt after a failure; 0 = no retry).
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

// registerTask registers a task and returns its pid.
func (in *interp) registerTask(t *Task) int {
	in.taskMu.Lock()
	defer in.taskMu.Unlock()
	in.nextPid++
	in.tasks[in.nextPid] = t
	return in.nextPid
}

// lookupTask finds a task by pid.
func (in *interp) lookupTask(pid int) (*Task, bool) {
	in.taskMu.Lock()
	defer in.taskMu.Unlock()
	t, ok := in.tasks[pid]
	return t, ok
}

// newThread creates a thread (taskm.spawn()) and returns its pid.
func (in *interp) newThread() int {
	t := &Task{doneCh: make(chan struct{}), BlockID: in.mem.Alloc(globalMemory.BlockSize, 0)}
	pid := in.registerTask(t)
	t.Pid = pid
	return pid
}

// runOnThread runs a function on thread pid (taskm.merge).
func (in *interp) runOnThread(t *Task, fn *Func, args []Value, pos Pos) error {
	// The argument slice may come from the caller's reuse area (argArena), while this function hands execution to a goroutine asynchronously
	// → it must copy, otherwise the arguments would be overwritten once the caller returns the arena.
	args = append([]Value(nil), args...)
	in.taskMu.Lock()
	if t.Busy {
		in.taskMu.Unlock()
		return &RunError{Msg: "RuntimeError: thread busy — merge only when idle (taskm.done)", Pos: pos}
	}
	t.Busy = true
	t.doneCh = make(chan struct{})
	in.taskMu.Unlock()

	// The memory block belongs to that thread; it is marked reclaimable when the task ends
	blockID := in.mem.Alloc(globalMemory.BlockSize, t.Pid)
	t.BlockID = blockID
	ctx := in.newCtx(fn, args, pos) // the direct-allocation version shares nothing, so it is safe for taskm threads
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

// taskArg accepts a Task or a pid and returns the corresponding task.
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

// memorizeBufferCall implements call(prefix, rec) for the built-in memorize:
// memoize by rec.in, run prefix.fn, write the result back to rec.out.
func (in *interp) memorizeBufferCall(mb *MemorizeBuffer, prefix *StructValue, rec *StructValue, pos Pos, ctx *execCtx) (Value, error) {
	nv, ok := prefix.Fields["fn"]
	if !ok {
		return NilV(), &RunError{Msg: msg("TypeError: memorize prefix 缺少被包装函数 fn"), Pos: pos, Ctx: ctx}
	}
	if !nv.IsFunc() {
		return NilV(), &RunError{Msg: msg("TypeError: prefix.fn 不是函数"), Pos: pos, Ctx: ctx}
	}
	f := nv.Func()
	if !rec.Fields["in"].IsList() {
		return NilV(), &RunError{Msg: msg("TypeError: rec.in 必须是 List"), Pos: pos, Ctx: ctx}
	}
	// Memoization key: the in argument list
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

// pow2 returns n such that v == 2^n (v > 0 and a power of two).
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
			return NilV(), &RunError{Msg: msg("TypeError: 位移运算需要 int 操作数"), Pos: pos, Ctx: ctx}
		}
		li, sh := l.Int(), r.Int()
		if op == "<<" {
			return IntV(int64(int32(li) << uint(sh&31))), nil
		}
		return IntV(int64(int32(li) >> uint(sh&31))), nil // arithmetic right shift (sign extension)
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
			// Bit-op optimization: multiply by 2^n ≡ shift left by n (identical under int32 wraparound, exactly equivalent)
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
