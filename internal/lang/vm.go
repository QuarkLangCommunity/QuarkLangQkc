package lang

// Bytecode VM for the hot subset of the language.
//
// Why it exists: a tree-walking interpreter spends most of its time in the node dispatch chain
// (pprof on fib(24): evalExpr 29% flat, plus execStmt/evalCall/callFunc around it). Fusing the AST
// into a compact instruction stream with pre-resolved slots and call targets removes most of that
// dispatch. It is not a second semantics: the VM reuses the interpreter's own primitives
// (wrapI32, callFunc, ioPrintln) and produces the identical error values, so the dual-path parity
// gate keeps holding.
//
// Safety model — a function is compiled only when its whole body fits the supported subset;
// anything else (try/catch, for-in, method calls other than io.print/io.println, space calls,
// struct/list/string operations, nested-block declarations, shadowing) makes the whole function
// fall back to the tree-walker. QUARK_NO_VM=1 disables the VM entirely, which is what the
// differential test uses to compare both engines on the same programs.

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
)

// vmExecutions counts bytecode invocations. It exists so tests can prove the VM is actually used
// (a silent permanent fallback would keep every semantic test green while the optimization did nothing).
var vmExecutions atomic.Int64

// errVMDeopt means "this function must not run as bytecode" — the VM saw a value whose runtime type
// does not match the statically proven type (the language allows implicit int->float promotion, so a
// parameter declared int can legitimately receive a float). The caller blacklists the function and
// re-runs it with the tree-walker, which is the semantic reference.
var errVMDeopt = errors.New("vm: deoptimized")

type vmOp uint8

const (
	opConstInt  vmOp = iota // a = int64 constant
	opConstBool             // a = 0/1
	opLoadSlot              // a = slot
	opLoadRef               // a = slot -> reference cell (arguments bind by reference)
	opStoreSlot             // a = slot (pops)
	opAddI
	opSubI
	opMulI
	opDivI // a = source position index
	opModI // a = source position index
	opLtI
	opLeI
	opGtI
	opGeI
	opEqI
	opNeI
	opNegI
	opJump      // a = target
	opJumpFalse // a = target (pops)
	opJumpTrue  // a = target (pops) — used by && / ||
	opCall      // a = function index, b = argc
	opPrintln   // a = argc, b = 1 println / 0 print; reads the stream from slot b2
	opLog       // records the top of stack and ends the function
	opReturn    // returns the top of the stack
	opReturnNil // returns nil
	opPop
)

type vmInstr struct {
	op   vmOp
	a, b int32
	pos  Pos
}

// vmProg is one function's compiled body.
type vmProg struct {
	code      []vmInstr
	nSlots    int      // parameters followed by locals
	slotNames []string // slot -> declared name (needed to build reference cells for arguments)
	ioSlot    int      // slot holding the IOStream (io), -1 when the function never prints
}

// ---- compilation ----

type vmCompiler struct {
	in        *interp
	fn        *Func
	code      []vmInstr
	slots     map[string]int
	types     map[string]string // slot name -> declared type: the VM only emits int/bool operations
	typeCache map[Expr]string   // expression node -> typeOf answer (memoised: see typeOf)
	slotNames []string          // slot -> name
	nSlots    int
	ioSlot    int
	failed    bool
	loops     []vmLoop
}

type vmLoop struct {
	breaks    []int
	continues []int
	contPos   int // patch target for continue (the loop's step/cond), -1 until known
}

func (in *interp) vmFor(fn *Func) *vmProg {
	if os.Getenv("QUARK_NO_VM") != "" {
		return nil
	}
	if fn.vmTried {
		return fn.vm
	}
	fn.vmTried = true
	if len(fn.TypeParams) > 0 || fn.Body == nil {
		return nil
	}
	c := &vmCompiler{in: in, fn: fn, slots: map[string]int{}, types: map[string]string{}, typeCache: map[Expr]string{}, ioSlot: -1}
	c.slotNames = make([]string, 0, len(fn.Params)+4)
	for i, p := range fn.Params {
		c.slots[p.Name] = i
		c.types[p.Name] = p.Type
		c.slotNames = append(c.slotNames, p.Name)
	}
	c.nSlots = len(fn.Params)
	if !c.block(fn.Body) {
		return nil
	}
	c.emit(vmInstr{op: opReturnNil})
	fn.vm = &vmProg{code: c.code, nSlots: c.nSlots, slotNames: c.slotNames, ioSlot: c.ioSlot}
	if os.Getenv("QUARK_VM_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "vm: compiled %s(%d params, %d instrs)\n", fn.Name, len(fn.Params), len(c.code))
	}
	return fn.vm
}

func (c *vmCompiler) emit(ins vmInstr) int {
	c.code = append(c.code, ins)
	return len(c.code) - 1
}

func (c *vmCompiler) fail() bool {
	c.failed = true
	return false
}

func (c *vmCompiler) block(b *Block) bool {
	if b == nil {
		return true
	}
	for _, st := range b.Stmts {
		if !c.stmt(st, false) {
			return false
		}
	}
	return true
}

// blockScoped compiles a nested block; declarations inside it would need scoped slots, so they are
// rejected and the function falls back to the tree-walker.
func (c *vmCompiler) blockScoped(b *Block) bool {
	if b == nil {
		return true
	}
	for _, st := range b.Stmts {
		if _, isDecl := st.(*DeclStmt); isDecl {
			return c.fail()
		}
		if !c.stmt(st, true) {
			return false
		}
	}
	return true
}

func (c *vmCompiler) stmt(s Stmt, nested bool) bool {
	_ = nested
	switch x := s.(type) {
	case *ExprStmt:
		// Only calls with no observable result are allowed here; the value is discarded.
		call, ok := x.X.(*CallExpr)
		if !ok {
			return c.fail()
		}
		if !c.callDiscard(call) {
			return false
		}
		return true
	case *DeclStmt:
		if x.Decor == "copyd" || x.Decor == "const" {
			return c.fail()
		}
		if x.Type != "int" && x.Type != "bool" {
			return c.fail()
		}
		if _, dup := c.slots[x.Name]; dup {
			return c.fail() // shadowing/redeclaration needs scoped slots
		}
		slot := c.nSlots
		c.nSlots++
		c.slots[x.Name] = slot
		c.types[x.Name] = x.Type
		c.slotNames = append(c.slotNames, x.Name)
		if x.Init != nil {
			if !c.expr(x.Init) {
				return false
			}
			c.emit(vmInstr{op: opStoreSlot, a: int32(slot), pos: x.Pos})
		} else {
			c.emit(vmInstr{op: opConstInt, a: 0, pos: x.Pos})
			c.emit(vmInstr{op: opStoreSlot, a: int32(slot), pos: x.Pos})
		}
		return true
	case *AssignStmt:
		id, ok := x.Target.(*Ident)
		if !ok {
			return c.fail() // member/index targets stay on the tree-walker
		}
		slot, ok := c.slots[id.Name]
		if !ok {
			return c.fail()
		}
		if !c.expr(x.X) {
			return false
		}
		c.emit(vmInstr{op: opStoreSlot, a: int32(slot), pos: x.Pos})
		return true
	case *IfStmt:
		if c.typeOf(x.Cond) != "bool" || !c.expr(x.Cond) {
			return false
		}
		jf := c.emit(vmInstr{op: opJumpFalse, pos: posOf(x.Cond)})
		if !c.blockScoped(x.Then) {
			return false
		}
		if x.Else != nil {
			jend := c.emit(vmInstr{op: opJump})
			c.code[jf].a = int32(len(c.code))
			if !c.blockScoped(x.Else) {
				return false
			}
			c.code[jend].a = int32(len(c.code))
		} else {
			c.code[jf].a = int32(len(c.code))
		}
		return true
	case *WhileStmt:
		return c.loop(x.Cond, nil, nil, x.Body, posOf(x.Cond))
	case *ForCStmt:
		// for (<decl>; <cond>; <step>) — the declaration is compiled here so the loop variable gets
		// its own slot; the step runs after the body on every iteration including continues.
		start := len(c.code)
		if x.Init != nil {
			if !c.stmt(x.Init, false) {
				return false
			}
		}
		condAt := len(c.code)
		var jf int
		if x.Cond != nil {
			if !c.expr(x.Cond) {
				return false
			}
			jf = c.emit(vmInstr{op: opJumpFalse, pos: posOf(x.Cond)})
		}
		lp := vmLoop{contPos: -1}
		c.loops = append(c.loops, lp)
		if !c.blockScoped(x.Body) {
			return false
		}
		// continue targets the step
		lp = c.loops[len(c.loops)-1]
		for _, at := range lp.continues {
			c.code[at].a = int32(len(c.code))
		}
		if x.Step != nil {
			if !c.stmt(x.Step, true) {
				return false
			}
		}
		c.emit(vmInstr{op: opJump, a: int32(condAt)})
		end := len(c.code)
		if jf != 0 || x.Cond != nil {
			c.code[jf].a = int32(end)
		}
		for _, at := range lp.breaks {
			c.code[at].a = int32(end)
		}
		c.loops = c.loops[:len(c.loops)-1]
		_ = start
		return true
	case *ReturnStmt:
		if x.X == nil {
			c.emit(vmInstr{op: opReturnNil, pos: x.Pos})
			return true
		}
		if !c.expr(x.X) {
			return false
		}
		c.emit(vmInstr{op: opReturn, pos: x.Pos})
		return true
	case *LogStmt:
		if !c.expr(x.X) {
			return false
		}
		c.emit(vmInstr{op: opLog, pos: posOf(x.X)})
		return true
	case *BreakStmt:
		if len(c.loops) == 0 {
			return c.fail()
		}
		at := c.emit(vmInstr{op: opJump, pos: x.Pos})
		i := len(c.loops) - 1
		c.loops[i].breaks = append(c.loops[i].breaks, at)
		return true
	}
	return c.fail()
}

// loop compiles while-style loops (for-in never reaches here).
func (c *vmCompiler) loop(cond Expr, init, step Stmt, body *Block, pos Pos) bool {
	start := len(c.code)
	if init != nil && !c.stmt(init, false) {
		return false
	}
	if !c.expr(cond) {
		return false
	}
	jf := c.emit(vmInstr{op: opJumpFalse, pos: posOf(cond)})
	lp := vmLoop{}
	c.loops = append(c.loops, lp)
	if !c.blockScoped(body) {
		return false
	}
	lp = c.loops[len(c.loops)-1]
	for _, at := range lp.continues {
		c.code[at].a = int32(len(c.code))
	}
	if step != nil && !c.stmt(step, true) {
		return false
	}
	c.emit(vmInstr{op: opJump, a: int32(start)})
	end := len(c.code)
	c.code[jf].a = int32(end)
	for _, at := range lp.breaks {
		c.code[at].a = int32(end)
	}
	c.loops = c.loops[:len(c.loops)-1]
	return true
}

// callDiscard compiles a call whose result is unused.
func (c *vmCompiler) callDiscard(call *CallExpr) bool {
	v, ok := c.callValue(call)
	if !ok || !v {
		return false
	}
	c.emit(vmInstr{op: opPop, pos: call.Pos}) // the call's value is unused
	return true
}

// callValue compiles a call expression, leaving its result on the stack.
func (c *vmCompiler) callValue(call *CallExpr) (bool, bool) {
	// io.println / io.print on the main entry's stream
	if mem, ok := call.Fn.(*MemberExpr); ok {
		if id, ok := mem.X.(*Ident); ok && (mem.Name == "println" || mem.Name == "print") {
			slot, known := c.slots[id.Name]
			if !known {
				return false, false // io not in a slot: tree-walk
			}
			for _, a := range call.Args {
				if !c.expr(a) {
					return false, false
				}
			}
			newline := int32(0)
			if mem.Name == "println" {
				newline = 1
			}
			c.ioSlot = slot
			c.emit(vmInstr{op: opPrintln, a: int32(len(call.Args)), b: newline, pos: call.Pos})
			c.emit(vmInstr{op: opConstInt, a: 0, pos: call.Pos}) // println yields nothing usable
			return true, true
		}
	}
	// direct function call resolved at compile time
	id, ok := call.Fn.(*Ident)
	if !ok || call.FnIdx < 0 {
		return false, false
	}
	if _, isVar := c.slots[id.Name]; isVar {
		return false, false // a variable shadows the function name: not a static call
	}
	if len(c.in.overloads[id.Name]) > 0 {
		return false, false // overload selection is decided by argument values at run time
	}
	for _, a := range call.Args {
		// A declared variable is an lvalue: the tree-walker hands the callee a reference cell so writes
		// go through to the caller. The VM must pass the same thing or by-reference semantics break.
		if id, isID := a.(*Ident); isID {
			if slot, known := c.slots[id.Name]; known {
				c.emit(vmInstr{op: opLoadRef, a: int32(slot), pos: id.Pos})
				continue
			}
		}
		if !c.expr(a) {
			return false, false
		}
	}
	c.emit(vmInstr{op: opCall, a: int32(call.FnIdx), b: int32(len(call.Args)), pos: call.Pos})
	return true, true
}

// typeOf is the VM's static type view: it only needs to prove that an expression is int or bool,
// because those are the only operations the bytecode implements. Anything it cannot prove makes the
// whole function fall back to the tree-walker. The answer for a node never changes while one function
// is compiled, so it is memoised per node: without the cache a left-deep chain of n nodes re-walks the
// whole subtree at every node and VM compilation costs O(n²) visits instead of O(n).
func (c *vmCompiler) typeOf(e Expr) string {
	if t, ok := c.typeCache[e]; ok {
		return t
	}
	t := c.typeOfNode(e)
	c.typeCache[e] = t
	return t
}

// typeOfNode computes the type of one node, recursing through the memoised typeOf for its children.
func (c *vmCompiler) typeOfNode(e Expr) string {
	switch x := e.(type) {
	case *IntLit:
		return "int"
	case *BoolLit:
		return "bool"
	case *Ident:
		return c.types[x.Name]
	case *UnOp:
		if x.Op == "-" && c.typeOf(x.X) == "int" {
			return "int"
		}
	case *BinOp:
		switch x.Op {
		case "+", "-", "*", "/", "%":
			if c.typeOf(x.L) == "int" && c.typeOf(x.R) == "int" {
				return "int"
			}
		case "<", "<=", ">", ">=":
			if c.typeOf(x.L) == "int" && c.typeOf(x.R) == "int" {
				return "bool"
			}
		case "==", "!=":
			if t := c.typeOf(x.L); t != "" && t == c.typeOf(x.R) && (t == "int" || t == "bool") {
				return "bool"
			}
		case "&&", "||":
			if c.typeOf(x.L) == "bool" && c.typeOf(x.R) == "bool" {
				return "bool"
			}
		}
	case *CallExpr:
		if id, isID := x.Fn.(*Ident); isID && x.FnIdx >= 0 && x.FnIdx < len(c.in.fnList) {
			if _, isVar := c.slots[id.Name]; !isVar && len(c.in.overloads[id.Name]) == 0 {
				return c.in.fnList[x.FnIdx].Ret
			}
		}
	}
	return ""
}

func (c *vmCompiler) expr(e Expr) bool {
	switch x := e.(type) {
	case *IntLit:
		c.emit(vmInstr{op: opConstInt, a: int32(x.V), pos: x.Pos})
		return true
	case *BoolLit:
		v := int32(0)
		if x.V {
			v = 1
		}
		c.emit(vmInstr{op: opConstBool, a: v, pos: x.Pos})
		return true
	case *Ident:
		slot, ok := c.slots[x.Name]
		if !ok {
			// Constants and globals would need a load path; fall back rather than guess.
			return c.fail()
		}
		c.emit(vmInstr{op: opLoadSlot, a: int32(slot), pos: x.Pos})
		return true
	case *UnOp:
		if x.Op != "-" || c.typeOf(x.X) != "int" {
			return c.fail()
		}
		if !c.expr(x.X) {
			return false
		}
		c.emit(vmInstr{op: opNegI, pos: x.Pos})
		return true
	case *BinOp:
		return c.binary(x)
	case *CallExpr:
		ok, left := c.callValue(x)
		if !ok {
			return false
		}
		_ = left
		return true
	}
	return c.fail()
}

func (c *vmCompiler) binary(x *BinOp) bool {
	// && and || short-circuit, exactly like the tree-walker
	switch x.Op {
	case "&&":
		if !c.expr(x.L) {
			return false
		}
		jf := c.emit(vmInstr{op: opJumpFalse, pos: x.Pos})
		if !c.expr(x.R) {
			return false
		}
		jf2 := c.emit(vmInstr{op: opJumpFalse, pos: x.Pos})
		c.emit(vmInstr{op: opConstBool, a: 1, pos: x.Pos})
		jend := c.emit(vmInstr{op: opJump})
		c.code[jf].a = int32(len(c.code))
		c.code[jf2].a = int32(len(c.code))
		c.emit(vmInstr{op: opConstBool, a: 0, pos: x.Pos})
		c.code[jend].a = int32(len(c.code))
		return true
	case "||":
		if !c.expr(x.L) {
			return false
		}
		jt := c.emit(vmInstr{op: opJumpTrue, pos: x.Pos})
		if !c.expr(x.R) {
			return false
		}
		jt2 := c.emit(vmInstr{op: opJumpTrue, pos: x.Pos})
		c.emit(vmInstr{op: opConstBool, a: 0, pos: x.Pos})
		jend := c.emit(vmInstr{op: opJump})
		c.code[jt].a = int32(len(c.code))
		c.code[jt2].a = int32(len(c.code))
		c.emit(vmInstr{op: opConstBool, a: 1, pos: x.Pos})
		c.code[jend].a = int32(len(c.code))
		return true
	}
	lt, rt := c.typeOf(x.L), c.typeOf(x.R)
	switch x.Op {
	case "<", "<=", ">", ">=", "-", "*", "/", "%":
		if lt != "int" || rt != "int" {
			return c.fail() // only integers are compiled; everything else keeps tree-walking
		}
	case "+":
		// int addition only: string concatenation and float arithmetic must not reach the VM
		if lt != "int" || rt != "int" {
			return c.fail()
		}
	case "==", "!=":
		if lt != rt || (lt != "int" && lt != "bool") {
			return c.fail()
		}
	}
	if !c.expr(x.L) || !c.expr(x.R) {
		return false
	}
	var op vmOp
	switch x.Op {
	case "+":
		op = opAddI
	case "-":
		op = opSubI
	case "*":
		op = opMulI
	case "/":
		op = opDivI
	case "%":
		op = opModI
	case "<":
		op = opLtI
	case "<=":
		op = opLeI
	case ">":
		op = opGtI
	case ">=":
		op = opGeI
	case "==":
		op = opEqI
	case "!=":
		op = opNeI
	default:
		return c.fail() // shifts, bit ops, pointer/reference ops stay on the tree-walker
	}
	c.emit(vmInstr{op: op, pos: x.Pos})
	return true
}

// ---- execution ----

func (in *interp) runVM(ctx *execCtx) error {
	vmExecutions.Add(1)
	fn := ctx.Fn
	p := fn.vm
	// Entry guard (before any side effect): every parameter must hold a value of its declared type.
	// Mixing int/float is legal in the language, and the tree-walker's generic arithmetic handles it;
	// the bytecode only implements integers, so such a call deoptimizes instead of erroring.
	for i, prm := range fn.Params {
		if i >= len(ctx.Args) {
			break
		}
		v := ctx.Args[i]
		for v.IsRef() {
			v = v.deref()
		}
		switch prm.Type {
		case "int":
			if !v.IsInt() {
				return errVMDeopt
			}
		case "bool":
			if !v.IsBool() {
				return errVMDeopt
			}
		}
	}
	sc := &ctx.sc
	if len(sc.slots) < p.nSlots {
		sc.slots = append(sc.slots, make([]Value, p.nSlots-len(sc.slots))...)
	}
	// Register local slot names the way the tree-walker's declare does: reference cells and scope
	// lookups resolve variables by name, so a VM-local that is passed by reference must be findable.
	if len(sc.paramNames) < p.nSlots && len(p.slotNames) >= p.nSlots {
		sc.paramNames = append(sc.paramNames, p.slotNames[len(sc.paramNames):p.nSlots]...)
	}
	st := ctx.vmStack[:0]
	pc := 0
	for {
		ins := p.code[pc]
		switch ins.op {
		case opConstInt:
			st = append(st, IntV(int64(ins.a)))
		case opConstBool:
			st = append(st, BoolV(ins.a != 0))
		case opLoadRef:
			name := ""
			if int(ins.a) < len(p.slotNames) {
				name = p.slotNames[ins.a]
			}
			st = append(st, RefV(sc.refHandle(name)))
		case opLoadSlot:
			v := sc.slots[ins.a]
			if v.IsRef() {
				v = v.deref() // reads dereference uniformly, exactly like the tree-walker
			}
			st = append(st, v)
		case opStoreSlot:
			v := st[len(st)-1]
			st = st[:len(st)-1]
			if cur := sc.slots[ins.a]; cur.IsRef() {
				// The slot holds a reference cell (by-reference parameter binding): assignment writes
				// through to the caller's lvalue, exactly like scope.set does for the tree-walker.
				if cur.Ref().store(v) {
					break
				}
			}
			sc.slots[ins.a] = v
		case opPop:
			st = st[:len(st)-1]
		case opAddI, opSubI, opMulI, opDivI, opModI, opLtI, opLeI, opGtI, opGeI, opEqI, opNeI:
			r := st[len(st)-1]
			l := st[len(st)-2]
			st = st[:len(st)-2]
			v, err := intBinOpVM(ins.op, l, r, ins.pos, ctx)
			if err != nil {
				ctx.vmStack = st
				return err
			}
			st = append(st, v)
		case opNegI:
			v := st[len(st)-1]
			if !v.IsInt() {
				ctx.vmStack = st
				return errVMDeopt
			}
			st[len(st)-1] = wrapI32(-v.Int())
		case opJump:
			pc = int(ins.a)
			continue
		case opJumpFalse, opJumpTrue:
			v := st[len(st)-1]
			st = st[:len(st)-1]
			b, cerr := truthy(v)
			if cerr != nil {
				ctx.vmStack = st
				return &RunError{Msg: cerr.Error(), Pos: ins.pos, Ctx: ctx}
			}
			if b == (ins.op == opJumpTrue) {
				pc = int(ins.a)
				continue
			}
		case opCall:
			argc := int(ins.b)
			start := len(ctx.argArena)
			ctx.argArena = append(ctx.argArena, st[len(st)-argc:]...)
			args := ctx.argArena[start:]
			st = st[:len(st)-argc]
			fnIdx := int(ins.a)
			var callee *Func
			if fnIdx >= 0 && fnIdx < len(in.fnList) {
				callee = in.fnList[fnIdx]
			}
			if callee == nil {
				ctx.vmStack = st
				return &RunError{Msg: "RuntimeError: unresolved call target", Pos: ins.pos, Ctx: ctx}
			}
			v, err := in.callFunc(callee, args, ins.pos, ctx.depth)
			ctx.argArena = ctx.argArena[:start] // the arena slice must not escape into the callee's result
			if err != nil {
				ctx.vmStack = st
				return err
			}
			st = append(st, v)
		case opPrintln:
			argc := int(ins.a)
			stream := sc.slots[p.ioSlot]
			if stream.IsRef() {
				stream = stream.deref()
			}
			if !stream.IsIO() {
				ctx.vmStack = st
				return &RunError{Msg: msg("TypeError: io.println needs an IOStream, got %s", stream.TypeName()), Pos: ins.pos, Ctx: ctx}
			}
			start := len(ctx.argArena)
			ctx.argArena = append(ctx.argArena, st[len(st)-argc:]...)
			args := derefArgs(ctx.argArena[start:])
			st = st[:len(st)-argc]
			if _, err := ioPrintln(stream.IO(), args, ins.b != 0, ins.pos, ctx); err != nil {
				ctx.argArena = ctx.argArena[:start]
				ctx.vmStack = st
				return err
			}
			ctx.argArena = ctx.argArena[:start]
		case opLog:
			v := st[len(st)-1]
			st = st[:len(st)-1]
			ctx.ensureLog().Append(StrV(v.String()))
			ctx.result = NilV()
			ctx.vmStack = st
			return nil
		case opReturn:
			v := st[len(st)-1]
			ctx.vmStack = st
			ctx.result = v
			return nil
		case opReturnNil:
			ctx.vmStack = st
			ctx.result = NilV()
			return nil
		}
		pc++
	}
}

// intBinOpVM applies the same integer semantics as the tree-walker's fast path.
func intBinOpVM(op vmOp, l, r Value, pos Pos, ctx *execCtx) (Value, error) {
	if !l.IsInt() || !r.IsInt() {
		// Runtime type surprise: hand the function back to the tree-walker rather than inventing an
		// error the interpreter would not raise.
		return NilV(), errVMDeopt
	}
	li, ri := l.Int(), r.Int()
	switch op {
	case opAddI:
		return wrapI32(li + ri), nil
	case opSubI:
		return wrapI32(int64(li) - int64(ri)), nil
	case opMulI:
		if n, ok := pow2(int32(ri)); ok {
			return wrapI32(int64(int32(li) << uint(n))), nil
		}
		return wrapI32(int64(li) * int64(ri)), nil
	case opDivI:
		if ri == 0 {
			return NilV(), &RunError{Msg: "DivisionByZeroError: integer division by zero", Pos: pos, Ctx: ctx}
		}
		return wrapI32(int64(li) / int64(ri)), nil
	case opModI:
		if ri == 0 {
			return NilV(), &RunError{Msg: "DivisionByZeroError: modulo by zero", Pos: pos, Ctx: ctx}
		}
		return wrapI32(int64(li) % int64(ri)), nil
	case opLtI:
		return BoolV(int32(li) < int32(ri)), nil
	case opLeI:
		return BoolV(int32(li) <= int32(ri)), nil
	case opGtI:
		return BoolV(int32(li) > int32(ri)), nil
	case opGeI:
		return BoolV(int32(li) >= int32(ri)), nil
	case opEqI:
		return BoolV(li == ri), nil
	case opNeI:
		return BoolV(li != ri), nil
	}
	return NilV(), &RunError{Msg: "TypeError: unsupported integer operation", Pos: pos, Ctx: ctx}
}

func vmOpName(op vmOp) string {
	switch op {
	case opAddI:
		return "+"
	case opSubI:
		return "-"
	case opMulI:
		return "*"
	case opDivI:
		return "/"
	case opModI:
		return "%"
	case opLtI:
		return "<"
	case opLeI:
		return "<="
	case opGtI:
		return ">"
	case opGeI:
		return ">="
	case opEqI:
		return "=="
	case opNeI:
		return "!="
	}
	return "operation"
}
