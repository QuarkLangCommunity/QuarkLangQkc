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

// Value is any QuarkLang runtime value：24 字节标签联合体。
// charley 设计: 标量（int/float/bool）内联在 i 字段零堆分配（消灭 convT64/mallocgc）；
// 对象与字符串引用走 unsafe.Pointer（Go GC 可见，保活无野指针）。
type Value struct {
	tag uint8
	i   int64          // 标量内联：int 值 / float 位模式 / bool 0|1
	ptr unsafe.Pointer // 对象与字符串引用（*strRef / 堆对象）
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
	vPtr // FFI 原生指针值（不透明句柄；新增在末尾，保持既有枚举序号不变）
)

func FileV(f *FileValue) Value { return Value{tag: byte(vFile), ptr: unsafe.Pointer(f)} }

// ---- file 值（路径对象） ----
type FileValue struct {
	Path string
}

func (f *FileValue) IsFile() bool     { return true }
func (f *FileValue) FilePath() string { return f.Path }
func (f *FileValue) String() string   { return "<file " + f.Path + ">" }
func (f *FileValue) TypeName() string { return "file" }

// ---- 标量构造器（保持旧名，调用点无需改） ----

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

// ---- FFI 原生指针值（不透明句柄：可空、可往返 FFI，不做指针算术） ----

// PtrV 包装来自 FFI 的裸指针（void* 句柄）。指针由系统库分配/返回，
// 不归 Go 堆管，直接存 ptr 字段（GC 只扫描 Go 堆指针，非 Go 堆地址自动忽略）。
// nil 归一化为 null（NilV）：空指针的语言表示就是 null。
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

// ---- 类型判定 ----

func (v Value) IsNil() bool         { return v.tag == byte(vNil) }
func (v Value) IsInt() bool         { return v.tag == byte(vInt) }
func (v Value) IsFloat() bool       { return v.tag == byte(vFloat) }
func (v Value) IsBool() bool        { return v.tag == byte(vBool) }
func (v Value) IsStr() bool         { return v.tag == byte(vStr) }
func (v Value) IsList() bool        { return v.tag == byte(vList) }
func (v Value) IsTable() bool       { return v.tag == byte(vTable) }
func (v Value) IsIO() bool          { return v.tag == byte(vIO) }
func (v Value) IsIn() bool          { return v.tag == byte(vIn) }
func (v Value) IsOut() bool         { return v.tag == byte(vOut) }
func (v Value) IsMemorize() bool    { return v.tag == byte(vMemorize) }
func (v Value) IsMemory() bool      { return v.tag == byte(vMemory) }
func (v Value) IsTaskm() bool       { return v.tag == byte(vTaskm) }
func (v Value) IsTask() bool        { return v.tag == byte(vTask) }
func (v Value) IsThread() bool      { return v.tag == byte(vThread) }
func (v Value) IsFunc() bool        { return v.tag == byte(vFunc) }
func (v Value) IsCopyd() bool       { return v.tag == byte(vCopyd) }
func (v Value) IsChan() bool        { return v.tag == byte(vChan) }
func (v Value) IsStruct() bool      { return v.tag == byte(vStruct) }
func (v Value) IsLib() bool         { return v.tag == byte(vLib) }
func (v Value) IsFile() bool        { return v.tag == byte(vFile) }
func (v Value) IsPtr() bool         { return v.tag == byte(vPtr) }
func (v Value) Ptr() unsafe.Pointer { return v.ptr }
func (v Value) File() *FileValue {
	ptr := (*FileValue)(v.ptr)
	if ptr == nil {
		return &FileValue{}
	}
	return ptr
}

// ---- 取值（调用方保证类型匹配；不匹配返回零值/空，语义由测试兜底） ----

func (v Value) Int() int64                { return v.i }
func (v Value) Float() float64            { return math.Float64frombits(uint64(v.i)) }
func (v Value) Bool() bool                { return v.i == 1 }
func (v Value) Str() string               { return (*strRef)(v.ptr).s }
func (v Value) List() *List               { return (*List)(v.ptr) }
func (v Value) Table() *HashTable         { return (*HashTable)(v.ptr) }
func (v Value) IO() *IOStream             { return (*IOStream)(v.ptr) }
func (v Value) In() *InputStream          { return (*InputStream)(v.ptr) }
func (v Value) Out() *OutputStream        { return (*OutputStream)(v.ptr) }
func (v Value) Memorize() *MemorizeBuffer { return (*MemorizeBuffer)(v.ptr) }
func (v Value) Memory() *Memory           { return (*Memory)(v.ptr) }
func (v Value) Taskm() *TaskManager       { return (*TaskManager)(v.ptr) }
func (v Value) Task() *Task               { return (*Task)(v.ptr) }
func (v Value) Thread() *ThreadValue      { return (*ThreadValue)(v.ptr) }
func (v Value) Func() *FuncValue          { return (*FuncValue)(v.ptr) }
func (v Value) Copyd() *CopydValue        { return (*CopydValue)(v.ptr) }
func (v Value) Chan() *Channel            { return (*Channel)(v.ptr) }
func (v Value) Struct() *StructValue      { return (*StructValue)(v.ptr) }
func (v Value) Lib() *libObj              { return (*libObj)(v.ptr) }

// TypeName 返回值的运行时类型名。
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
	}
	return "<unknown>"
}

// String 返回值的显示字符串。
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
	}
	return "<unknown>"
}

// ---- rolling List<T> (spec §4) ----

// List is a rolling two-pointer buffer: visible elements live in [head, tail).
// mem/blockID 挂接全局内存管理器（写入时标记所属 block 为脏）。
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

// reset 复用 List 的底层切片（对象池化用）。
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

// setIndex 写 i-th 可见元素（0-based，相对 head）；越界报错。
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

// Keys 返回键列表（String 键重建；其他类型按 "类型名:值" 前缀重建）。
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
	TypeParams []string // 泛型函数 func<T,...>（xmind §函数）
	Params     []Param
	Ret        string
	Body       *Block
	Pos        Pos
	paramNames []string // 参数名缓存（槽位绑定用，共享零分配）
	paramCopyd []bool   // Copyd 参数标志（惰性缓存，调用热路径免字符串扫描）
}

// ParamNames 返回参数名数组（惰性缓存，所有调用共享）。
func (f *Func) ParamNames() []string {
	if f.paramNames == nil {
		f.paramNames = make([]string, len(f.Params))
		for i, p := range f.Params {
			f.paramNames[i] = p.Name
		}
	}
	return f.paramNames
}

// CopydFlags 返回参数 Copyd 标志（惰性缓存，与 ParamNames 同一思路）。
func (f *Func) CopydFlags() []bool {
	if f.paramCopyd == nil {
		f.paramCopyd = make([]bool, len(f.Params))
		for i, p := range f.Params {
			f.paramCopyd[i] = isCopydType(p.Type)
		}
	}
	return f.paramCopyd
}

// execCtx 是函数执行的内部上下文（v2：语言面不再有 FuncBuffer）。
// 函数执行记录日志（log），结果由 return 直接产生。
type execCtx struct {
	Fn       *Func
	Args     []Value
	Log      *List
	result   Value
	executed bool
	pos      Pos
	sc       scope    // 函数执行作用域（复用，免每次调用堆分配）
	depth    int      // 调用深度（栈溢出防护，按 ctx 传播，线程安全）
	link     *execCtx // 空闲链（无锁 LIFO 栈）
}

// newCtx 接管调用参数切片所有权（evalArgs 每次新建，免复制）。
// 无锁 LIFO 空闲链（atomic CAS）：实测优于纯分配（malloc+GC 扫描 > 2 次原子操作），
// 且对 taskm 并发线程安全（CAS 无共享状态破坏）。
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

// ensureLog 确保日志列表存在（惰性兼容：池版 Log 恒非空，此调用几乎免费）。
func (ctx *execCtx) ensureLog() *List {
	if ctx.Log == nil {
		ctx.Log = NewList()
	}
	return ctx.Log
}

// putCtx 归还原子链（无锁 CAS 推入）。
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
// mu implements the 执行表 (spec §14.2): FIFO by arrival time, reads (RLock)
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

// Task 是线程（taskm）的执行上下文：done = 线程是否空闲。
type Task struct {
	ctx     *execCtx
	doneCh  chan struct{}
	err     error
	Pid     int
	BlockID int
	Busy    bool // 是否有函数占用（done 即 !Busy）
}

// ThreadValue 是 thread 类实例（xmind：taskm.spawn() 返回 thread 类，内含 pid）。
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

// CopydValue 包装 Copyd<T> 参数值；.ptr() 取出包装的地址。
type CopydValue struct{ V Value }

func (c *CopydValue) TypeName() string { return "Copyd" }
func (c *CopydValue) String() string   { return c.V.String() } // Copyd 透明

// Channel is the coroutine communication primitive (block-buffered).
type Channel struct{ ch chan Value }

// NewChannel 创建容量为 cap 的缓冲 channel。
func NewChannel(cap int) *Channel { return &Channel{ch: make(chan Value, cap)} }

func (c *Channel) TypeName() string { return "Channel" }
func (c *Channel) String() string   { return "<Channel>" }

// ---- deep copy (Copyd semantics; HashTable stores deep copies) ----

func deepCopy(v Value) Value {
	if v.IsList() {
		t := v.List()
		items := make([]Value, len(t.items))
		for i, it := range t.items {
			items[i] = deepCopy(it)
		}
		cp := &List{items: items, head: t.head, tail: t.tail}
		return ListV(cp)
	}
	if v.IsTable() {
		t := v.Table()
		h := NewHashTable()
		for k, it := range t.m {
			h.m[k] = deepCopy(it)
		}
		return TableV(h)
	}
	return v
}

// implDefsFor 聚合某类型所有 impl（无接口 + 各接口实现）。
func (in *interp) implDefsFor(typ string) []*ImplDef {
	var out []*ImplDef
	for k, d := range in.impls {
		if strings.HasPrefix(k, typ) && (len(k) == len(typ) || (len(k) > len(typ) && k[len(typ)] == '\x00')) {
			out = append(out, d)
		}
	}
	return out
}
