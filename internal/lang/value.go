package lang

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// Value is any QuarkLang runtime value: a 24-byte tagged union.
// charley's design: scalars (int/float/bool) are inlined in the i field with zero heap allocation (removes convT64/mallocgc);
// objects and string references go through unsafe.Pointer (visible to the Go GC, kept alive, no dangling pointers).
type Value struct {
	tag uint8
	i   int64          // scalar inline: int value / float bit pattern / bool 0|1
	ptr unsafe.Pointer // object and string reference (*strRef / heap object)
}

type ValueKind uint8

const (
	vNil ValueKind = iota
	vInt
	vFloat
	vBool
	vStr
	vList
	vTable
	vIO
	vIn
	vOut
	vMemorize
	vMemory
	vTaskm
	vTask
	vThread
	vFunc
	vCopyd
	vChan
	vStruct
	vLib
	vFile
	vPtr // FFI native pointer value (opaque handle; appended at the end to keep existing enum ordinals unchanged)
	vRef // argument reference cell (pass by reference; appended at the end to keep existing enum ordinals unchanged)
)

func FileV(f *FileValue) Value { return Value{tag: byte(vFile), ptr: unsafe.Pointer(f)} }

// ---- file value (path object) ----
type FileValue struct {
	Path string
}

func (f *FileValue) IsFile() bool     { return true }
func (f *FileValue) FilePath() string { return f.Path }
func (f *FileValue) String() string   { return "<file " + f.Path + ">" }
func (f *FileValue) TypeName() string { return "file" }

// ---- scalar constructors (old names kept so call sites need no change) ----

func IntV(n int64) Value   { return Value{tag: byte(vInt), i: n} }
func IntV32(n int32) Value { return IntV(int64(n)) }
func FloatV(f float64) Value {
	return Value{tag: byte(vFloat), i: int64(math.Float64bits(f))}
}
func BoolV(b bool) Value {
	if b {
		return Value{tag: byte(vBool), i: 1}
	}
	return Value{tag: byte(vBool)}
}

type strRef struct{ s string }

func StrV(str string) Value {
	return Value{tag: byte(vStr), ptr: unsafe.Pointer(&strRef{s: str})}
}
func NilV() Value { return Value{} }

// ---- FFI native pointer value (opaque handle: nullable, round-trippable through FFI, no pointer arithmetic) ----

// PtrV wraps a raw pointer coming from FFI (a void* handle). The memory is allocated/returned by a system library and is not
// managed by the Go heap, so it is stored directly in the ptr field (the GC scans only Go heap pointers and ignores non-Go addresses).
// nil is normalized to null (NilV): the language representation of a null pointer is null.
func PtrV(p unsafe.Pointer) Value {
	if p == nil {
		return NilV()
	}
	return Value{tag: byte(vPtr), ptr: p}
}

func ListV(l *List) Value               { return Value{tag: byte(vList), ptr: unsafe.Pointer(l)} }
func TableV(h *HashTable) Value         { return Value{tag: byte(vTable), ptr: unsafe.Pointer(h)} }
func IOV(s *IOStream) Value             { return Value{tag: byte(vIO), ptr: unsafe.Pointer(s)} }
func InV(s *InputStream) Value          { return Value{tag: byte(vIn), ptr: unsafe.Pointer(s)} }
func OutV(s *OutputStream) Value        { return Value{tag: byte(vOut), ptr: unsafe.Pointer(s)} }
func MemorizeV(m *MemorizeBuffer) Value { return Value{tag: byte(vMemorize), ptr: unsafe.Pointer(m)} }
func MemoryV(m *Memory) Value           { return Value{tag: byte(vMemory), ptr: unsafe.Pointer(m)} }
func TaskmV(t *TaskManager) Value       { return Value{tag: byte(vTaskm), ptr: unsafe.Pointer(t)} }
func TaskV(t *Task) Value               { return Value{tag: byte(vTask), ptr: unsafe.Pointer(t)} }
func ThreadV(t *ThreadValue) Value      { return Value{tag: byte(vThread), ptr: unsafe.Pointer(t)} }
func FuncV(f *FuncValue) Value          { return Value{tag: byte(vFunc), ptr: unsafe.Pointer(f)} }
func CopydV(c *CopydValue) Value        { return Value{tag: byte(vCopyd), ptr: unsafe.Pointer(c)} }
func ChanV(c *Channel) Value            { return Value{tag: byte(vChan), ptr: unsafe.Pointer(c)} }
func StructV(st *StructValue) Value     { return Value{tag: byte(vStruct), ptr: unsafe.Pointer(st)} }
func LibraryV(o *libObj) Value          { return Value{tag: byte(vLib), ptr: unsafe.Pointer(o)} }
func RefV(r *refValue) Value            { return Value{tag: byte(vRef), ptr: unsafe.Pointer(r)} }

// ---- reference value (by-reference argument passing) ----

// refKind is the kind of reference cell.
type refKind uint8

const (
	refIdent  refKind = iota // name in a scope (variable/parameter slot)
	refMember                // struct field (obj is the struct value, name is the field name)
	refIndex                 // List element (obj is the List value, key is the index)
)

// refValue is an argument's reference cell: when a parameter binds by reference, reads/writes go straight through to the caller's lvalue cell.
// Non-lvalue arguments do not build a reference (they are passed by value and treated as a temporary cell inside the callee).
type refValue struct {
	kind refKind
	sc   *scope // refIdent: scope holding the name
	name string // refIdent/refMember: name
	obj  Value  // refMember/refIndex: container
	key  Value  // refIndex: index
}

// load reads the reference cell's current value (it does not follow a dereference chain; deref/store handles chains).
func (r *refValue) load() Value {
	switch r.kind {
	case refIdent:
		if r.sc == nil {
			return NilV()
		}
		return r.sc.rawGet(r.name)
	case refMember:
		if r.obj.IsStruct() {
			if v, ok := r.obj.Struct().Fields[r.name]; ok {
				return v
			}
		}
	case refIndex:
		if r.obj.IsList() {
			v, err := r.obj.List().Get(int(r.key.Int()))
			if err == nil {
				return v
			}
		}
	}
	return NilV()
}

// store writes the reference cell (writing through reference chains; false = the target is not writable).
func (r *refValue) store(v Value) bool {
	switch r.kind {
	case refIdent:
		if r.sc == nil {
			return false
		}
		// The target slot is itself a reference (an f→g chain): keep writing through instead of overwriting the reference cell
		if cur := r.sc.rawGet(r.name); cur.IsRef() && cur.Ref() != r {
			return cur.Ref().store(v)
		}
		return r.sc.rawSet(r.name, v)
	case refMember:
		if r.obj.IsStruct() {
			if _, ok := r.obj.Struct().Fields[r.name]; ok {
				r.obj.Struct().Fields[r.name] = v
				return true
			}
		}
	case refIndex:
		if r.obj.IsList() {
			return r.obj.List().setIndex(int(r.key.Int()), v) == nil
		}
	}
	return false
}

// deref follows a reference chain: reference values are completely transparent to the language (every read path dereferences).
// Chains are built from call arguments (calling g(x) inside f(x) reuses the same reference cell, so the chain length is ≤ 1).
// deref dereferences (the unified read path for reference passing). Hot path: the vast majority of values are not references,
// so this keeps a minimal, inlinable sentinel check and moves the real dereference loop into derefSlow.
func (v Value) deref() Value {
	if v.tag != byte(vRef) {
		return v
	}
	return v.derefSlow()
}

func (v Value) derefSlow() Value {
	for n := 0; v.tag == byte(vRef); n++ {
		if n > 1<<20 { // cycle guard (unreachable by construction, defensive fallback)
			return NilV()
		}
		r := (*refValue)(v.ptr)
		if r == nil {
			return NilV()
		}
		v = r.load()
	}
	return v
}

// derefArgs dereferences an argument list (FFI/builtins/thread boundaries accept values only).
func derefArgs(args []Value) []Value {
	for i, a := range args {
		if a.IsRef() {
			args[i] = a.deref()
		}
	}
	return args
}

// IsRef is the only test that can see the reference cell itself (all other tests and reads are transparent to references).
func (v Value) IsRef() bool    { return v.tag == byte(vRef) }
func (v Value) Ref() *refValue { return (*refValue)(v.ptr) }

// ---- type tests ----

func (v Value) IsNil() bool         { return v.deref().tag == byte(vNil) }
func (v Value) IsInt() bool         { return v.deref().tag == byte(vInt) }
func (v Value) IsFloat() bool       { return v.deref().tag == byte(vFloat) }
func (v Value) IsBool() bool        { return v.deref().tag == byte(vBool) }
func (v Value) IsStr() bool         { return v.deref().tag == byte(vStr) }
func (v Value) IsList() bool        { return v.deref().tag == byte(vList) }
func (v Value) IsTable() bool       { return v.deref().tag == byte(vTable) }
func (v Value) IsIO() bool          { return v.deref().tag == byte(vIO) }
func (v Value) IsIn() bool          { return v.deref().tag == byte(vIn) }
func (v Value) IsOut() bool         { return v.deref().tag == byte(vOut) }
func (v Value) IsMemorize() bool    { return v.deref().tag == byte(vMemorize) }
func (v Value) IsMemory() bool      { return v.deref().tag == byte(vMemory) }
func (v Value) IsTaskm() bool       { return v.deref().tag == byte(vTaskm) }
func (v Value) IsTask() bool        { return v.deref().tag == byte(vTask) }
func (v Value) IsThread() bool      { return v.deref().tag == byte(vThread) }
func (v Value) IsFunc() bool        { return v.deref().tag == byte(vFunc) }
func (v Value) IsCopyd() bool       { return v.deref().tag == byte(vCopyd) }
func (v Value) IsChan() bool        { return v.deref().tag == byte(vChan) }
func (v Value) IsStruct() bool      { return v.deref().tag == byte(vStruct) }
func (v Value) IsLib() bool         { return v.deref().tag == byte(vLib) }
func (v Value) IsFile() bool        { return v.deref().tag == byte(vFile) }
func (v Value) IsPtr() bool         { return v.deref().tag == byte(vPtr) }
func (v Value) Ptr() unsafe.Pointer { return v.deref().ptr }
func (v Value) File() *FileValue {
	v = v.deref()
	ptr := (*FileValue)(v.ptr)
	if ptr == nil {
		return &FileValue{}
	}
	return ptr
}

// ---- value extraction (the caller guarantees the type; a mismatch yields zero/empty and the tests pin the semantics) ----

func (v Value) Int() int64                { v = v.deref(); return v.i }
func (v Value) Float() float64            { v = v.deref(); return math.Float64frombits(uint64(v.i)) }
func (v Value) Bool() bool                { v = v.deref(); return v.i == 1 }
func (v Value) Str() string               { v = v.deref(); return (*strRef)(v.ptr).s }
func (v Value) List() *List               { v = v.deref(); return (*List)(v.ptr) }
func (v Value) Table() *HashTable         { v = v.deref(); return (*HashTable)(v.ptr) }
func (v Value) IO() *IOStream             { v = v.deref(); return (*IOStream)(v.ptr) }
func (v Value) In() *InputStream          { v = v.deref(); return (*InputStream)(v.ptr) }
func (v Value) Out() *OutputStream        { v = v.deref(); return (*OutputStream)(v.ptr) }
func (v Value) Memorize() *MemorizeBuffer { v = v.deref(); return (*MemorizeBuffer)(v.ptr) }
func (v Value) Memory() *Memory           { v = v.deref(); return (*Memory)(v.ptr) }
func (v Value) Taskm() *TaskManager       { v = v.deref(); return (*TaskManager)(v.ptr) }
func (v Value) Task() *Task               { v = v.deref(); return (*Task)(v.ptr) }
func (v Value) Thread() *ThreadValue      { v = v.deref(); return (*ThreadValue)(v.ptr) }
func (v Value) Func() *FuncValue          { v = v.deref(); return (*FuncValue)(v.ptr) }
func (v Value) Copyd() *CopydValue        { v = v.deref(); return (*CopydValue)(v.ptr) }
func (v Value) Chan() *Channel            { v = v.deref(); return (*Channel)(v.ptr) }
func (v Value) Struct() *StructValue      { v = v.deref(); return (*StructValue)(v.ptr) }
func (v Value) Lib() *libObj              { v = v.deref(); return (*libObj)(v.ptr) }

// TypeName returns the value's runtime type name.
func (v Value) TypeName() string {
	switch ValueKind(v.tag) {
	case vNil:
		return "nil"
	case vInt:
		return "int"
	case vFloat:
		return "float"
	case vBool:
		return "bool"
	case vStr:
		return "String"
	case vList:
		return "List"
	case vTable:
		return "HashTable"
	case vIO:
		return "IOStream"
	case vIn:
		return "InputStream"
	case vOut:
		return "OutputStream"
	case vMemorize:
		return "memorize"
	case vMemory:
		return "memory"
	case vTaskm:
		return "taskm"
	case vTask:
		return "Task"
	case vThread:
		return "thread"
	case vFunc:
		return "fn"
	case vCopyd:
		return "Copyd"
	case vChan:
		return "Channel"
	case vStruct:
		return v.Struct().SType
	case vPtr:
		return "pointer"
	case vRef:
		return v.deref().TypeName() // reference transparency (defensive: the read path should already have dereferenced)
	}
	return "<unknown>"
}

// String returns the value's display string.
func (v Value) String() string {
	switch ValueKind(v.tag) {
	case vNil:
		return "nil"
	case vInt:
		return strconv.FormatInt(v.i, 10)
	case vFloat:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case vBool:
		if v.i == 1 {
			return "true"
		}
		return "false"
	case vStr:
		return (*strRef)(v.ptr).s
	case vList:
		return v.List().String()
	case vTable:
		return v.Table().String()
	case vIO:
		return "<IOStream>"
	case vIn:
		return "<InputStream>"
	case vOut:
		return "<OutputStream>"
	case vMemorize:
		return "<memorize buffer>"
	case vMemory:
		return "<memory>"
	case vTaskm:
		return "<taskm>"
	case vTask:
		return v.Task().String()
	case vThread:
		return v.Thread().String()
	case vFunc:
		return v.Func().String()
	case vCopyd:
		return v.Copyd().V.String()
	case vChan:
		return "<Channel>"
	case vStruct:
		return v.Struct().String()
	case vLib:
		return "<library " + v.Lib().name + ">"
	case vPtr:
		return "0x" + strconv.FormatUint(uint64(uintptr(v.ptr)), 16)
	case vRef:
		return v.deref().String() // reference transparency (defensive: the read path should already have dereferenced)
	}
	return "<unknown>"
}

// ---- rolling List<T> (spec §4) ----

// List is a rolling two-pointer buffer: visible elements live in [head, tail).
// mem/blockID hook into the global memory manager (a write marks the owning block dirty).
type List struct {
	items   []Value
	head    int
	tail    int
	mem     *MemoryManager
	blockID int
}

func NewList(items ...Value) *List {
	return &List{items: items, tail: len(items)}
}

// reset reuses a List's backing slice (for object pooling).
func (l *List) reset() {
	l.items = l.items[:0]
	l.head = 0
	l.tail = 0
	l.mem = nil
	l.blockID = 0
}

func (l *List) TypeName() string { return "List" }

func (l *List) String() string {
	parts := make([]string, 0, l.tail-l.head)
	for i := l.head; i < l.tail; i++ {
		parts = append(parts, l.items[i].String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// Head returns the head pointer position.
func (l *List) Head() int { return l.head }

// Tail returns the tail pointer position.
func (l *List) Tail() int { return l.tail }

// Size returns tail-head, the number of visible elements.
func (l *List) Size() int { return l.tail - l.head }

// Peek returns the element at the head without moving the pointer ('*list').
func (l *List) Peek() (Value, error) {
	if l.head == l.tail {
		return NilV(), fmt.Errorf("ListExhaustedError: list is exhausted (head()==tail()); '*' stops and errors")
	}
	return l.items[l.head], nil
}

// Next returns the head element and rolls the head pointer forward one step.
func (l *List) Next() (Value, error) {
	if l.head == l.tail {
		return NilV(), fmt.Errorf("ListExhaustedError: list is exhausted (head()==tail()); next() stops and errors")
	}
	v := l.items[l.head]
	l.head++
	return v, nil
}

// Reset moves the head pointer back to 0, making the full history visible again.
func (l *List) Reset() { l.head = 0 }

// Append writes v at the tail and moves the tail forward.
func (l *List) Append(v Value) {
	l.items = append(l.items, v)
	l.tail++
	if l.mem != nil {
		l.mem.MarkDirty(l.blockID)
	}
}

// AppendAll copies every visible element of o onto the tail (o is not consumed).
func (l *List) AppendAll(o *List) {
	for i := o.head; i < o.tail; i++ {
		l.Append(o.items[i])
	}
}

// setIndex writes the i-th visible element (0-based, relative to head); out of range is an error.
func (l *List) setIndex(i int, v Value) error {
	idx := l.head + i
	if i < 0 || idx >= l.tail {
		return &RunError{Msg: fmt.Sprintf("IndexOutOfBoundsError: index %d out of range [0,%d)", i, l.Size())}
	}
	l.items[idx] = v
	if l.mem != nil {
		l.mem.MarkDirty(l.blockID)
	}
	return nil
}

// Get returns the i-th visible element (0-based, relative to head).
func (l *List) Get(i int) (Value, error) {
	idx := l.head + i
	if i < 0 || idx >= l.tail {
		return NilV(), fmt.Errorf("IndexOutOfBoundsError: index %d out of range [0,%d)", i, l.Size())
	}
	return l.items[idx], nil
}

// copyVisible returns a new List with a copy of the visible range.
func (l *List) copyVisible() *List {
	items := make([]Value, 0, l.Size())
	for i := l.head; i < l.tail; i++ {
		items = append(items, l.items[i])
	}
	return NewList(items...)
}

// sortInPlace sorts the visible range (int/float/String elements only).
func (l *List) sortInPlace() error {
	items := l.items[l.head:l.tail]
	var sortErr error
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		switch {
		case a.IsInt():
			if b.IsInt() {
				return a.Int() < b.Int()
			}
			if b.IsFloat() {
				return float64(a.Int()) < b.Float()
			}
		case a.IsFloat():
			if b.IsFloat() {
				return a.Float() < b.Float()
			}
			if b.IsInt() {
				return a.Float() < float64(b.Int())
			}
		case a.IsStr():
			if b.IsStr() {
				return a.Str() < b.Str()
			}
		}
		sortErr = fmt.Errorf("TypeError: __sort__ supports int/float/String elements only, got %s and %s", a.TypeName(), b.TypeName())
		return false
	})
	return sortErr
}

// ---- HashTable<K,V> (spec §3.6) ----

type HashTable struct {
	m map[string]Value
}

func NewHashTable() *HashTable { return &HashTable{m: map[string]Value{}} }

func (h *HashTable) TypeName() string { return "HashTable" }

func (h *HashTable) String() string {
	parts := make([]string, 0, len(h.m))
	for k, v := range h.m {
		parts = append(parts, k+" -> "+v.String())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// Put stores a deep copy of v under key k.
func (h *HashTable) Put(k, v Value) { h.m[hashKey(k)] = deepCopy(v) }

// Get returns the value stored under k.
func (h *HashTable) Get(k Value) (Value, bool) {
	v, ok := h.m[hashKey(k)]
	return v, ok
}

// Contains reports whether k is present.
func (h *HashTable) Contains(k Value) bool {
	_, ok := h.m[hashKey(k)]
	return ok
}

// Remove deletes the entry for k.
func (h *HashTable) Remove(k Value) { delete(h.m, hashKey(k)) }

// Size returns the number of entries.
func (h *HashTable) Size() int { return len(h.m) }

// Keys returns the key list (String keys are rebuilt; other types are rebuilt with a "typeName:value" prefix).
func (h *HashTable) Keys() []Value {
	out := make([]Value, 0, len(h.m))
	for k := range h.m {
		if i := len("String:"); len(k) > i && k[:i] == "String:" {
			out = append(out, StrV(k[i:]))
			continue
		}
		if i := len("int:"); len(k) > i && k[:i] == "int:" {
			if n, err := strconv.Atoi(k[i:]); err == nil {
				out = append(out, IntV(int64(n)))
			}
			continue
		}
		out = append(out, StrV(k))
	}
	return out
}

// hashKey builds a stable structural key for any Value.
func hashKey(v Value) string { return v.TypeName() + ":" + v.String() }

// ---- FuncBuffer (spec §5) ----

// Func is a compiled QuarkLang function. Ret != "" means the call yields the
// value of `return expr;`; otherwise the call yields the whole FuncBuffer.
type Func struct {
	Name       string
	TypeParams []string // generic function func<T,...> (xmind §functions)
	Params     []Param
	Ret        string
	Body       *Block
	Pos        Pos
	paramNames []string // parameter-name cache (for slot binding, shared, zero allocation)
	paramCopyd []bool   // Copyd parameter flags (lazy cache, avoids string scanning on the call hot path)
}

// ParamNames returns the parameter-name array (lazily cached and shared by every call).
func (f *Func) ParamNames() []string {
	if f.paramNames == nil {
		f.paramNames = make([]string, len(f.Params))
		for i, p := range f.Params {
			f.paramNames[i] = p.Name
		}
	}
	return f.paramNames
}

// CopydFlags returns the parameter Copyd flags (lazily cached, same idea as ParamNames).
// Both spellings are recognised: the type annotation int[Copyd]/Copyd<T>, and the parameter modifier copyd (fn f(copyd int a)).
func (f *Func) CopydFlags() []bool {
	if f.paramCopyd == nil {
		f.paramCopyd = make([]bool, len(f.Params))
		for i, p := range f.Params {
			f.paramCopyd[i] = isCopydType(p.Type) || p.Decor == "copyd"
		}
	}
	return f.paramCopyd
}

// paramWrapCopyd reports whether parameter i binds as a "Copyd type" (it binds the Copyd value, readable via .ptr());
// a copyd-modified parameter binds the deep-copied bare value (scalars can be used directly; .ptr() does not apply).
func (f *Func) paramWrapCopyd(i int) bool {
	return i >= 0 && i < len(f.Params) && isCopydType(f.Params[i].Type)
}

// execCtx is the internal context of a function execution (v2: there is no FuncBuffer at the language level any more).
// A function execution records log entries; the result comes directly from return.
type execCtx struct {
	Fn       *Func
	Args     []Value
	Log      *List
	result   Value
	executed bool
	pos      Pos
	sc       scope    // function execution scope (reused, avoids a heap allocation per call)
	depth    int      // call depth (stack overflow protection, propagated per ctx, thread-safe)
	link     *execCtx // free chain (lock-free LIFO stack)

	// argArena is the reuse area for argument slices (owned by this ctx → no sharing across goroutines).
	// evalArgs appends at its tail and the call site truncates it back after the call; the capacity is kept for reuse.
	argArena []Value
}

// newCtx takes ownership of the call's argument slice (evalArgs creates a fresh one per call, so no copy is needed).
// A lock-free LIFO free chain (atomic CAS): measured better than plain allocation (malloc+GC scanning beats two atomics),
// and it is safe for taskm threads (CAS breaks no shared state).
func (in *interp) newCtx(fn *Func, args []Value, pos Pos) *execCtx {
	var ctx *execCtx
	for {
		h := in.ctxHead.Load()
		if h == nil {
			break
		}
		if in.ctxHead.CompareAndSwap(h, h.link) {
			ctx = h
			break
		}
	}
	if ctx == nil {
		ctx = &execCtx{Log: NewList()}
	}
	ctx.link = nil
	ctx.argArena = ctx.argArena[:0]
	ctx.Fn = fn
	ctx.Args = args
	ctx.pos = pos
	ctx.result = NilV()
	ctx.executed = false
	ctx.Log.reset()
	ctx.sc.outer = nil
	ctx.sc.vars = nil
	ctx.sc.slots = nil
	ctx.sc.paramNames = nil
	return ctx
}

// ensureLog makes sure the log list exists (lazy compatibility: a pooled Log is never nil, so this call is nearly free).
func (ctx *execCtx) ensureLog() *List {
	if ctx.Log == nil {
		ctx.Log = NewList()
	}
	return ctx.Log
}

// putCtx returns the ctx to the atomic chain (lock-free CAS push).
func (in *interp) putCtx(ctx *execCtx) {
	ctx.Fn = nil
	ctx.Args = nil
	ctx.sc.outer = nil
	ctx.sc.vars = nil
	ctx.sc.slots = nil
	ctx.sc.paramNames = nil
	for {
		h := in.ctxHead.Load()
		ctx.link = h
		if in.ctxHead.CompareAndSwap(h, ctx) {
			return
		}
	}
}

// ---- IO objects (spec §10) ----

// IOStream is the default main() parameter: input + output + redirectable.
// mu implements the execution table (spec §14.2): FIFO by arrival time, reads (RLock)
// have priority over writes (Lock).
type IOStream struct {
	In  io.Reader
	Out io.Writer
	rd  *bufio.Reader
	mu  sync.RWMutex
}

func (s *IOStream) TypeName() string { return "IOStream" }
func (s *IOStream) String() string   { return "<IOStream>" }

// InputStream is the input base type (File* / Console* subclasses).
type InputStream struct{ R io.Reader }

func (s *InputStream) TypeName() string { return "InputStream" }
func (s *InputStream) String() string   { return "<InputStream>" }

// OutputStream is the output base type.
type OutputStream struct{ W io.Writer }

func (s *OutputStream) TypeName() string { return "OutputStream" }
func (s *OutputStream) String() string   { return "<OutputStream>" }

// MemorizeBuffer is the state of the built-in memorize signature.
type MemorizeBuffer struct{ Table *HashTable }

func (m *MemorizeBuffer) TypeName() string { return "memorize" }
func (m *MemorizeBuffer) String() string   { return "<memorize buffer>" }

// Memory is the default concrete instance of the global memory struct
// (block-managed; spec §14). v0.1: memory is backed by the host Go GC;
// compact() returns nothing (no value) and BlockSize is the dynamic
// block-granularity setting (memory.setBlock(n)).
type Memory struct{ BlockSize int }

func (m *Memory) TypeName() string { return "memory" }
func (m *Memory) String() string   { return "<memory>" }

// globalMemory is the built-in `memory` identifier.
var globalMemory = &Memory{BlockSize: 4096}

// TaskManager is the coroutine manager; taskm is a GLOBAL VARIABLE, so the
// correct call syntax is taskm.spawn(...) / taskm.block(pid) / taskm.done(pid)
// / taskm.merge(pid) / taskm.channel([n]) (spec §14.2).
type TaskManager struct{}

func (t *TaskManager) TypeName() string { return "taskm" }
func (t *TaskManager) String() string   { return "<taskm>" }

// globalTaskm is the built-in `taskm` identifier.
var globalTaskm = &TaskManager{}

// FuncValue is a first-class function reference (for taskm::spawn etc.).
type FuncValue struct{ fn *Func }

func (f *FuncValue) TypeName() string { return "fn" }
func (f *FuncValue) String() string   { return "<fn " + f.fn.Name + ">" }

// Task is a thread's (taskm) execution context: done = is the thread idle.
type Task struct {
	ctx     *execCtx
	doneCh  chan struct{}
	err     error
	Pid     int
	BlockID int
	Busy    bool // whether a function occupies it (done means !Busy)
}

// ThreadValue is a thread class instance (xmind: taskm.spawn() returns a thread instance carrying a pid).
type ThreadValue struct {
	Pid int
	t   *Task
}

func (tv *ThreadValue) TypeName() string { return "thread" }
func (tv *ThreadValue) String() string   { return "<thread " + itoa(tv.Pid) + ">" }

func (t *Task) TypeName() string { return "Task" }
func (t *Task) String() string   { return "<Task " + itoa(t.Pid) + ">" }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// StructValue is an instance of a user-defined struct.
type StructValue struct {
	SType  string
	Fields map[string]Value
}

func (s *StructValue) TypeName() string { return s.SType }

func (s *StructValue) String() string {
	parts := make([]string, 0, len(s.Fields))
	for k, v := range s.Fields {
		parts = append(parts, k+"="+v.String())
	}
	return "<" + s.SType + " {" + strings.Join(parts, ", ") + "}>"
}

// CopydValue wraps a Copyd<T> parameter value; .ptr() returns the wrapped address.
type CopydValue struct{ V Value }

func (c *CopydValue) TypeName() string { return "Copyd" }
func (c *CopydValue) String() string   { return c.V.String() } // Copyd is transparent

// Channel is the coroutine communication primitive (block-buffered).
type Channel struct{ ch chan Value }

// NewChannel creates a buffered channel with capacity cap.
func NewChannel(cap int) *Channel { return &Channel{ch: make(chan Value, cap)} }

func (c *Channel) TypeName() string { return "Channel" }
func (c *Channel) String() string   { return "<Channel>" }

// ---- deep copy (Copyd semantics; HashTable stores deep copies) ----

// deepCopy deep-copies List/HashTable (HashTable.Put's existing semantics: store a snapshot of the table value);
// struct is not included (existing behaviour is kept, so HashTable/memory semantics are unaffected) — Copyd arguments use copyDeep.
func deepCopy(v Value) Value { return copyDeep(v, false) }

// copydCopy is the deep copy used for Copyd argument passing/binding: on top of deepCopy it also copies structs recursively
// (so callee writes to a copyd parameter's fields do not affect the caller).
func copydCopy(v Value) Value { return copyDeep(v, true) }

// copyDeep copies recursively; with deepStruct=true structs are copied too (List/Table elements recurse as well).
func copyDeep(v Value, deepStruct bool) Value {
	v = v.deref()
	if v.IsList() {
		t := v.List()
		items := make([]Value, len(t.items))
		for i, it := range t.items {
			items[i] = copyDeep(it, deepStruct)
		}
		cp := &List{items: items, head: t.head, tail: t.tail}
		return ListV(cp)
	}
	if v.IsTable() {
		t := v.Table()
		h := NewHashTable()
		for k, it := range t.m {
			h.m[k] = copyDeep(it, deepStruct)
		}
		return TableV(h)
	}
	if deepStruct && v.IsStruct() {
		sv := v.Struct()
		cp := &StructValue{SType: sv.SType, Fields: make(map[string]Value, len(sv.Fields))}
		for k, fv := range sv.Fields {
			cp.Fields[k] = copyDeep(fv, true)
		}
		return StructV(cp)
	}
	return v
}

// implDefsFor aggregates all impls of a type (no interface + each interface implementation).
func (in *interp) implDefsFor(typ string) []*ImplDef {
	var out []*ImplDef
	for k, d := range in.impls {
		if strings.HasPrefix(k, typ) && (len(k) == len(typ) || (len(k) > len(typ) && k[len(typ)] == '\x00')) {
			out = append(out, d)
		}
	}
	return out
}
