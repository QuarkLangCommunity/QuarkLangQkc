package lang

import (
	"errors"
	"fmt"
	"strings"
)

// ============ Static types (v0.1 structural subset, spec §11.1) ============

type tKind int

const (
	tInt tKind = iota
	tFloat
	tString
	tBool
	tNil
	tAny // interface{}
	tList
	tHashTable
	tFuncBuffer
	tIOStream
	tInputStream
	tOutputStream
	tChannel
	tTask
	tMemorize
	tMemory
	tFunc
	tStruct
	tTaskm
	tPtr
	tCopyd
	tNull
	tTypeVar
	tLib
	tInterface
	tFile
)

type Type struct {
	Kind   tKind
	Elem   *Type
	Key    *Type
	Val    *Type
	FName  string           // tFunc: function name ("" = unknown); tStruct: struct name
	Args   []*Type          // tStruct: generic instance arguments
	Fields map[string]*Type // tStruct anonymous (FName="."): field type table
}

func mk(k tKind) *Type         { return &Type{Kind: k} }
func mkList(e *Type) *Type     { return &Type{Kind: tList, Elem: e} }
func mkTable(k, v *Type) *Type { return &Type{Kind: tHashTable, Key: k, Val: v} }
func mkFunc(name string) *Type { return &Type{Kind: tFunc, FName: name} }

var (
	tIntV          = mk(tInt)
	tFloatV        = mk(tFloat)
	tStringV       = mk(tString)
	tBoolV         = mk(tBool)
	tNilV          = mk(tNil)
	tAnyV          = mk(tAny)
	tFuncBufferV   = mk(tFuncBuffer)
	tIOStreamV     = mk(tIOStream)
	tInputStreamV  = mk(tInputStream)
	tOutputStreamV = mk(tOutputStream)
	tChannelV      = mk(tChannel)
	tTaskV         = mk(tTask)
	tMemorizeV     = mk(tMemorize)
	tMemoryV       = mk(tMemory)
	tTaskmV        = mk(tTaskm)
)

var kindName = map[tKind]string{
	tInt: "int", tFloat: "float", tString: "String", tBool: "bool",
	tNil: "nil", tAny: "interface{}", tFuncBuffer: "FuncBuffer",
	tIOStream: "IOStream", tInputStream: "InputStream", tOutputStream: "OutputStream",
	tChannel: "Channel", tTask: "Task", tMemorize: "memorize", tMemory: "memory",
	tFunc: "fn", tStruct: "struct", tTaskm: "taskm", tPtr: "ptr", tCopyd: "Copyd", tNull: "null", tTypeVar: "typevar", tLib: "library", tInterface: "interface", tFile: "file",
}

func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}
	switch t.Kind {
	case tList:
		return "List<" + t.Elem.String() + ">"
	case tHashTable:
		return "HashTable<" + t.Key.String() + ", " + t.Val.String() + ">"
	case tFunc:
		if t.FName != "" {
			return "fn " + t.FName
		}
		return "fn"
	case tStruct:
		if len(t.Args) > 0 {
			parts := make([]string, len(t.Args))
			for i, a := range t.Args {
				parts[i] = a.String()
			}
			return t.FName + "<" + strings.Join(parts, ", ") + ">"
		}
		return t.FName
	case tPtr:
		if t.Elem == nil {
			return "pointer"
		}
		return t.Elem.String() + "&"
	case tCopyd:
		return "Copyd<" + t.Elem.String() + ">"
	case tNull:
		return "null"
	}
	if n, ok := kindName[t.Kind]; ok {
		return n
	}
	return "unknown"
}

// assignable reports whether from can be assigned to to (strict; int→float numeric widening is allowed).
func assignable(from, to *Type) bool {
	if from == nil || to == nil {
		return false
	}
	if to.Kind == tAny {
		return true
	}
	if from.Kind == tAny {
		return true // any is assignable to any concrete type (runtime value compatible; dynamic parse cases such as json.loads)
	}
	if from.Kind == tNil {
		return false
	}
	if from.Kind == tInt && to.Kind == tFloat {
		return true
	}
	// Pointer: a value may be assigned to a pointer; null may be assigned to a pointer
	if to.Kind == tPtr {
		if from.Kind == tNull {
			return true
		}
		if to.Elem == nil { // opaque handle: accepts only pointer / null
			return from.Kind == tPtr
		}
		if from.Kind == tPtr {
			if from.Elem == nil {
				return true
			}
			return assignable(from.Elem, to.Elem) || to.Elem.Kind == tAny || from.Elem.Kind == tAny
		}
		return assignable(from, to.Elem) || to.Elem.Kind == tAny
	}
	// Type variable (generic function): mutually assignable with any type
	if from.Kind == tTypeVar || to.Kind == tTypeVar {
		return true
	}
	// any (interface{}): assignable to any concrete type (runtime value compatibility; dynamic cases such as json.loads)
	if from.Kind == tAny || to.Kind == tAny {
		return true
	}
	// Named interface: struct/interface → interface is allowed leniently (strict implementation checks happen where a checker context exists, e.g. checkCallArgs)
	// Interface → interface satisfies **structurally**: from's method set only has to cover to's (names need not match)
	if to.Kind == tInterface {
		if from.Kind == tInterface {
			return true
		}
		if from.Kind == tStruct {
			return true
		}
		return false
	}
	// Copyd: mutually assignable with the inner type
	if from.Kind == tCopyd {
		return assignable(from.Elem, to)
	}
	if to.Kind == tCopyd {
		return assignable(from, to.Elem)
	}
	if from.Kind != to.Kind {
		return false
	}
	switch to.Kind {
	case tList:
		return from.Elem.Kind == tAny || to.Elem.Kind == tAny || assignable(from.Elem, to.Elem)
	case tHashTable:
		keyOK := from.Key.Kind == tAny || to.Key.Kind == tAny || assignable(from.Key, to.Key)
		valOK := from.Val.Kind == tAny || to.Val.Kind == tAny || assignable(from.Val, to.Val)
		return keyOK && valOK
	case tFunc:
		// Function name → function<...> signature compatibility (the concrete signature is checked at the call site at runtime)
		return to.FName == "" || strings.HasPrefix(to.FName, "function<") || strings.HasPrefix(from.FName, "function<") || from.FName == to.FName
	case tStruct:
		if from.FName != to.FName || len(from.Args) != len(to.Args) {
			return false
		}
		for i := range from.Args {
			a, b := from.Args[i], to.Args[i]
			if a.Kind != tAny && b.Kind != tAny && !assignable(a, b) {
				return false
			}
		}
		return true
	case tPtr:
		if from.Elem == nil || to.Elem == nil {
			return from.Elem == nil && to.Elem == nil
		}
		return assignable(from.Elem, to.Elem)
	case tCopyd:
		return assignable(from.Elem, to.Elem)
	}
	return true
}

// parseTypeStr parses a type annotation. Array<T>/T[]/T[Copyd] are all normalized to List<T> in v0.1
// (at runtime there is only List; Copyd copy semantics are handled by the runtime from the parameter annotation).
func parseTypeStr(s string) (*Type, error) {
	s = strings.TrimSpace(s)
	if s == "interface{}" || s == "any" {
		return tAnyV, nil
	}
	if s == "Self" {
		return &Type{Kind: tTypeVar, FName: "Self"}, nil
	}
	base, inner, suffix := splitType(s)
	elem := func() (*Type, error) {
		if inner == "" {
			return tAnyV, nil
		}
		return parseTypeStr(inner)
	}
	switch base {
	case "int", "long", "char": // long/char are treated as int in v0.1 (width: see §3.1 status)
		if suffix != "" || strings.HasSuffix(s, "[]") {
			e, err := elem()
			if err != nil {
				return nil, err
			}
			return mkList(e), nil
		}
		return tIntV, nil
	case "float", "double":
		return tFloatV, nil
	case "bool":
		return tBoolV, nil
	case "String":
		return tStringV, nil
	case "void":
		return tAnyV, nil // v2: void = the default name of the empty interface interface{}
	case "FuncBuffer":
		return tFuncBufferV, nil
	case "IOStream":
		return tIOStreamV, nil
	case "InputStream", "istream", "ifstream":
		return tInputStreamV, nil
	case "OutputStream", "ostream", "ofstream":
		return tOutputStreamV, nil
	case "iofstream":
		return tInputStreamV, nil // v1: a bidirectional file stream has the input-stream type (the object both reads and writes)
	case "Channel":
		return tChannelV, nil
	case "Task", "thread":
		return tTaskV, nil
	case "memorize":
		return tMemorizeV, nil
	case "memory":
		return tMemoryV, nil
	case "taskm":
		return tTaskmV, nil
	case "List", "Array":
		e, err := elem()
		if err != nil {
			return nil, err
		}
		return mkList(e), nil
	case "Copyd":
		e, err := elem()
		if err != nil {
			return nil, err
		}
		return e, nil // Copyd<T> is semantically just T (the copy happens on passing)
	case "HashTable":
		k, v, err := splitTopComma(inner)
		if err != nil {
			return nil, err
		}
		kt, err := parseTypeStr(k)
		if err != nil {
			return nil, err
		}
		vt, err := parseTypeStr(v)
		if err != nil {
			return nil, err
		}
		return mkTable(kt, vt), nil
	}
	return nil, fmt.Errorf("unknown type %q", s)
}

func splitType(s string) (base, inner, suffix string) {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "["); i >= 0 && strings.HasSuffix(s, "]") {
		suffix = strings.TrimSpace(s[i+1 : len(s)-1])
		if suffix == "" {
			suffix = "[]" // int[] = Copyd<Array<int>>
		}
		s = s[:i]
	}
	if i := strings.Index(s, "<"); i >= 0 && strings.HasSuffix(s, ">") {
		base = s[:i]
		inner = s[i+1 : len(s)-1]
	} else {
		base = s
	}
	return base, strings.TrimSpace(inner), suffix
}

// splitTopComma splits on top-level commas (ignoring commas inside angle brackets).
func splitTopComma(s string) (string, string, error) {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), nil
			}
		}
	}
	return "", "", fmt.Errorf("type %q needs two arguments (K, V)", s)
}

// ============ Checker ============

// CheckError is a compile-time (static check) error.
type CheckError struct {
	Msg string
	Pos Pos
}

func (e *CheckError) Error() string {
	return fmt.Sprintf("%s at line %d", e.Msg, e.Pos.Line)
}

type cVar struct {
	typ     *Type
	init    bool
	isConst bool
	// Permission-carrying references (R2): the permission lives on the declaration, so it is recorded
	// here and every read/write check consults it. Bare T& keeps full permission and program scope.
	refPerm  string
	refScope string
}

type cScope struct {
	vars  map[string]*cVar
	outer *cScope
}

func newCScope(outer *cScope) *cScope {
	return &cScope{vars: map[string]*cVar{}, outer: outer}
}

func (s *cScope) declare(name string, v *cVar, pos Pos) error {
	if _, dup := s.vars[name]; dup {
		return &CheckError{Msg: fmt.Sprintf("CompileError: duplicate declaration of %q", name), Pos: pos}
	}
	s.vars[name] = v
	return nil
}

func (s *cScope) lookup(name string) *cVar {
	for sc := s; sc != nil; sc = sc.outer {
		if v, ok := sc.vars[name]; ok {
			return v
		}
	}
	return nil
}

type checker struct {
	fns        map[string]*Func
	overloads  map[string][]*Func
	sigs       map[string]int // signature name → <Prefix> parameter count (memorize:1, async:0)
	structs    map[string]*StructDef
	interfaces map[string]*InterfaceDef
	impls      map[string]*ImplDef
	aliases    map[string]string       // type <type> name; type alias
	libs       map[string]*LibraryDecl // library: system library binding
	curRet     *Type
	loopDepth  int
	curSubst   map[string]*Type // type-parameter substitution for a generic method body / call site
	typeVars   map[string]bool  // type parameters of the current scope in a generic function (fn<T,...>)
}

// Typecheck performs all of the compile-time strict checks of §11.1.
// file type: value = path; file::new(name); f.read() / f.write(s)
var tFileV = &Type{Kind: tFile, FName: "file"}

// implKeyOf: the key for multiple impls — no interface (self-impl/static) = Type; an interface implementation = Type + "\x00" + Iface.
func implKeyOf(typ, iface string) string {
	if iface == "" {
		return typ
	}
	return typ + "\x00" + iface
}

// staticMeth looks up a static method across aggregated impls (no self parameter).
func (c *checker) staticMeth(typ, name string) *Func {
	for _, d := range c.implDefsFor(typ) {
		if fn := d.Methods[name]; fn != nil {
			return fn
		}
	}
	return nil
}

// selfMeth looks up an instance method across aggregated impls (self first parameter).
func (c *checker) selfMeth(typ, name string) *Func {
	for _, d := range c.implDefsFor(typ) {
		if fn := d.SelfMethods[name]; fn != nil {
			return fn
		}
	}
	return nil
}

// implDefsFor aggregates all impls of a type (no interface + each interface implementation; methods across impls do not overlap).
func (c *checker) implDefsFor(typ string) []*ImplDef {
	var out []*ImplDef
	for k, d := range c.impls {
		if strings.HasPrefix(k, typ) && (len(k) == len(typ) || (len(k) > len(typ) && k[len(typ)] == '\x00')) {
			out = append(out, d)
		}
	}
	return out
}

// overloadErrT reports no matching overload on the type side (same wording as at runtime).
func overloadErrT(defs []*Func, name string, n int) string {
	for _, d := range defs {
		if len(d.Params) != n {
			return fmt.Sprintf("CompileError: %s expects %d args, got %d", name, len(d.Params), n)
		}
	}
	return fmt.Sprintf(msg("CompileError: 未找到匹配重载 %q（参数类型不匹配）"), name)
}

// opMethodFor maps an operator → the Operation protocol method name (xmind §interfaces: operator methods such as __add__; dynamic interface dispatch).
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

// opUnaryMethodFor maps a unary operator → its protocol method name.
func opUnaryMethodFor(op string) string {
	if op == "-" {
		return "__neg__"
	}
	return ""
}

// builtinOperationIfaces: the language's built-in Operation interface family (dynamic protocol; methods across impls do not overlap).
func builtinOperationIfaces() map[string]*InterfaceDef {
	self2 := []Param{{Name: "self", Type: "Self"}, {Name: "o", Type: "Self"}}
	self1 := []Param{{Name: "self", Type: "Self"}}
	self := self1
	fn := func(n string, p []Param, ret string) MethodSig {
		return MethodSig{Name: n, Params: p, Ret: ret, Dynamic: true}
	}
	return map[string]*InterfaceDef{
		"AddOperation": {Name: "AddOperation", Methods: []MethodSig{fn("__add__", self2, "Self")}},
		"SubOperation": {Name: "SubOperation", Methods: []MethodSig{fn("__sub__", self2, "Self")}},
		"MulOperation": {Name: "MulOperation", Methods: []MethodSig{fn("__mul__", self2, "Self")}},
		"DivOperation": {Name: "DivOperation", Methods: []MethodSig{fn("__div__", self2, "Self")}},
		"ModOperation": {Name: "ModOperation", Methods: []MethodSig{fn("__mod__", self2, "Self")}},
		"NegOperation": {Name: "NegOperation", Methods: []MethodSig{fn("__neg__", self1, "Self")}},
		"EqOperation":  {Name: "EqOperation", Methods: []MethodSig{fn("__eq__", self2, "bool")}},
		"NeOperation":  {Name: "NeOperation", Methods: []MethodSig{fn("__ne__", self2, "bool")}},
		"LtOperation":  {Name: "LtOperation", Methods: []MethodSig{fn("__lt__", self2, "bool")}},
		"LeOperation":  {Name: "LeOperation", Methods: []MethodSig{fn("__le__", self2, "bool")}},
		"GtOperation":  {Name: "GtOperation", Methods: []MethodSig{fn("__gt__", self2, "bool")}},
		"GeOperation":  {Name: "GeOperation", Methods: []MethodSig{fn("__ge__", self2, "bool")}},
		"Operation":    {Name: "Operation", Expands: []string{"AddOperation", "SubOperation", "MulOperation", "DivOperation", "ModOperation", "NegOperation", "EqOperation", "NeOperation", "LtOperation", "LeOperation", "GtOperation", "GeOperation"}},
		// ---- Event interface family (component protocol: users impl it directly; qksignal_emit dispatches) ----
		"ClegClickable":  {Name: "ClegClickable", Methods: []MethodSig{fn("onClicked", self, "void"), fn("onPressed", self, "void"), fn("onReleased", self, "void")}},
		"ClegCheckable":  {Name: "ClegCheckable", Methods: []MethodSig{fn("onToggled", []Param{{Name: "self", Type: "Self"}, {Name: "checked", Type: "bool"}}, "void")}},
		"ClegEditable":   {Name: "ClegEditable", Methods: []MethodSig{fn("onTextChanged", []Param{{Name: "self", Type: "Self"}, {Name: "newText", Type: "String"}}, "void"), fn("onReturnPressed", self, "void")}},
		"ClegValueable":  {Name: "ClegValueable", Methods: []MethodSig{fn("onValueChanged", []Param{{Name: "self", Type: "Self"}, {Name: "value", Type: "int"}}, "void")}},
		"ClegSelectable": {Name: "ClegSelectable", Methods: []MethodSig{fn("onCurrentIndexChanged", []Param{{Name: "self", Type: "Self"}, {Name: "index", Type: "int"}}, "void")}},
		"ClegItemable":   {Name: "ClegItemable", Methods: []MethodSig{fn("onItemClicked", []Param{{Name: "self", Type: "Self"}, {Name: "index", Type: "int"}}, "void")}},
		"ClegCellable":   {Name: "ClegCellable", Methods: []MethodSig{fn("onCellClicked", []Param{{Name: "self", Type: "Self"}, {Name: "row", Type: "int"}, {Name: "col", Type: "int"}}, "void")}},
		"ClegCloseable":  {Name: "ClegCloseable", Methods: []MethodSig{fn("onCloseRequested", self, "void")}},
		"ClegActionable": {Name: "ClegActionable", Methods: []MethodSig{fn("onTriggered", self, "void")}},
	}
}

// registerBuiltinIfaces pre-registers built-in interfaces (shared by typecheck and eval).
func registerBuiltinIfaces(intfs map[string]*InterfaceDef) {
	for name, def := range builtinOperationIfaces() {
		if _, ok := intfs[name]; !ok {
			if strings.HasPrefix(name, "Cleg") {
				def.Partial = true // event interface family: optional implementation
			}
			intfs[name] = def
		}
	}
}

func Typecheck(prog *Program) error {
	c := &checker{
		fns:        map[string]*Func{},
		sigs:       map[string]int{"memorize": 1, "async": 0},
		structs:    map[string]*StructDef{},
		interfaces: map[string]*InterfaceDef{},
		impls:      map[string]*ImplDef{},
		aliases:    map[string]string{},
		overloads:  map[string][]*Func{},
	}
	registerBuiltinIfaces(c.interfaces)
	for _, a := range prog.TypeAliases {
		if _, dup := c.aliases[a.Name]; dup {
			return &CheckError{Msg: fmt.Sprintf("CompileError: duplicate type alias %q", a.Name), Pos: a.Pos}
		}
		c.aliases[a.Name] = a.Type
	}
	for _, f := range prog.Funcs {
		nfn := &Func{Name: f.Name, TypeParams: f.TypeParams, Params: f.Params, Ret: f.Ret, Body: f.Body, Pos: f.Pos}
		if _, dup := c.fns[f.Name]; dup {
			// Function overloads: same name appends (different signatures are fine; an identical one is an error)
			for _, od := range c.overloads[f.Name] {
				if sameSig(od, nfn) {
					return &CheckError{Msg: fmt.Sprintf(msg("CompileError: duplicate overload %q (与已有签名相同)"), f.Name), Pos: f.Pos}
				}
			}
			c.overloads[f.Name] = append(c.overloads[f.Name], nfn)
		} else {
			c.fns[f.Name] = nfn
		}
	}
	for _, s := range prog.Structs {
		if _, dup := c.structs[s.Name]; dup {
			return &CheckError{Msg: fmt.Sprintf("CompileError: duplicate struct %q", s.Name), Pos: s.Pos}
		}
		def := &StructDef{Name: s.Name, TypeParams: s.TypeParams, Types: map[string]string{}}
		for _, m := range s.Members {
			def.TypesOrder = append(def.TypesOrder, m.Name)
		}
		for _, m := range s.Members {
			def.Types[m.Name] = m.Type
		}
		c.structs[s.Name] = def
	}
	for _, i := range prog.Interfaces {
		if _, dup := c.interfaces[i.Name]; dup {
			return &CheckError{Msg: fmt.Sprintf("CompileError: duplicate interface %q", i.Name), Pos: i.Pos}
		}
		c.interfaces[i.Name] = &InterfaceDef{Name: i.Name, TypeParams: i.TypeParams, Methods: i.Methods, Expands: i.Expands}
	}
	for _, lb := range prog.Libraries {
		if c.libs == nil {
			c.libs = map[string]*LibraryDecl{}
		}
		if _, dup := c.libs[lb.Name]; dup {
			return &CheckError{Msg: fmt.Sprintf("CompileError: duplicate library %q", lb.Name), Pos: lb.Pos}
		}
		c.libs[lb.Name] = lb
	}
	for _, im := range prog.Impls {
		// Generic rule: when a struct has generic parameters its impl must introduce the same ones; when it has none the impl must not
		if sd, ok := c.structs[im.Type]; ok {
			if len(sd.TypeParams) > 0 && len(im.TypeParams) != len(sd.TypeParams) {
				return &CheckError{Msg: fmt.Sprintf("CompileError: struct %s has %d type parameter(s) — impl must introduce the same parameters (impl<T> {...} %s)", im.Type, len(sd.TypeParams), im.Type), Pos: im.Pos}
			}
			if len(sd.TypeParams) == 0 && len(im.TypeParams) > 0 {
				return &CheckError{Msg: fmt.Sprintf("CompileError: struct %s has no type parameters, but impl declares %d", im.Type, len(im.TypeParams)), Pos: im.Pos}
			}
		}
		// A type may have several impl blocks: methods are aggregated (xmind §classes: impl<T> {...} name;); a duplicate method name is still an error
		key := implKeyOf(im.Type, im.Iface)
		def, exists := c.impls[key]
		if !exists {
			def = &ImplDef{Type: im.Type, Iface: im.Iface, TypeParams: im.TypeParams, Methods: map[string]*Func{}, SelfMethods: map[string]*Func{}}
			c.impls[key] = def
		}
		for _, m := range im.Methods {
			recv := false
			if len(m.Params) > 0 {
				recv = isRecvParam(im.Type, &m.Params[0]) // receiver determined by type (independent of the parameter name)
			}
			fn := &Func{Name: m.Name, TypeParams: m.TypeParams, Params: m.Params, Ret: m.Ret, Body: m.Body, Pos: m.Pos}
			if recv {
				if _, dup := def.SelfMethods[fn.Name]; dup {
					return &CheckError{Msg: fmt.Sprintf("CompileError: duplicate method %q on %s", fn.Name, im.Type), Pos: m.Pos}
				}
				def.SelfMethods[fn.Name] = fn
			} else {
				if _, dup := def.Methods[fn.Name]; dup {
					return &CheckError{Msg: fmt.Sprintf("CompileError: duplicate method %q on %s", fn.Name, im.Type), Pos: m.Pos}
				}
				def.Methods[fn.Name] = fn
			}
		}
	}
	// impl interface consistency (§11.1.3): every interface method must be implemented (name + parameter count)
	for _, im := range prog.Impls {
		if im.Iface == "" {
			continue
		}
		iface, ok := c.interfaces[im.Iface]
		if !ok {
			return &CheckError{Msg: fmt.Sprintf("CompileError: unknown interface %q", im.Iface), Pos: im.Pos}
		}
		if iface.Partial {
			// Optional interfaces: implementing some methods is enough (emit at runtime only fires the implemented ones)
			continue
		}
		// Composed interfaces: recursively collect methods expanded by `expand` (including Expands of expanded interfaces)
		methods := append([]MethodSig{}, iface.Methods...)
		collectExpands := func(names []string) {}
		var collect func([]string)
		collect = func(names []string) {
			for _, ex := range names {
				ei, ok := c.interfaces[ex]
				if !ok {
					return
				}
				methods = append(methods, ei.Methods...)
				collect(ei.Expands)
			}
		}
		_ = collectExpands
		collect(iface.Expands)
		for _, sig := range methods {
			fn := c.selfMeth(im.Type, sig.Name)
			if fn == nil {
				fn = c.staticMeth(im.Type, sig.Name)
			}
			if fn == nil {
				return &CheckError{Msg: fmt.Sprintf("CompileError: impl %s for %s: missing method %q", im.Type, im.Iface, sig.Name), Pos: im.Pos}
			}
			want := len(sig.Params) // the interface signature includes self
			got := len(fn.Params)
			if got != want {
				return &CheckError{Msg: fmt.Sprintf("CompileError: impl %s for %s: method %q takes %d params, interface requires %d", im.Type, im.Iface, sig.Name, got, want), Pos: im.Pos}
			}
		}
	}
	for _, f := range prog.Funcs {
		if f.Name == "main" && (len(f.Params) < 1 || len(f.Params) > 3) {
			return &CheckError{Msg: fmt.Sprintf("CompileError: main must take 1-3 params in order (io, env, args), got %d", len(f.Params)), Pos: f.Pos}
		}
		if err := c.checkFunc(c.fns[f.Name]); err != nil {
			return err
		}
	}
	for _, im := range prog.Impls {
		subst := map[string]*Type{}
		for _, tp := range im.TypeParams {
			subst[tp] = tAnyV // generic method body: parameters checked loosely as interface{}
		}
		subst["Self"] = &Type{Kind: tStruct, FName: im.Type} // within impl, Self = this type (consistent with the interface's Self)
		prev := c.curSubst
		c.curSubst = subst
		for _, m := range im.Methods {
			if err := c.checkFunc(&Func{Name: m.Name, TypeParams: m.TypeParams, Params: m.Params, Ret: m.Ret, Body: m.Body, Pos: m.Pos}); err != nil {
				c.curSubst = prev
				return err
			}
		}
		c.curSubst = prev
	}
	return nil
}

// resolveType resolves a type annotation (no substitution context).
func (c *checker) resolveType(s string, pos Pos) (*Type, error) {
	return c.substType(s, nil, pos)
}

// substType resolves a type annotation and substitutes generic type parameters (subst: parameter name → concrete type).
// Supported: T (a parameter name), T& (pointer), node<T> (generic instantiation), List<T>/HashTable<K,V>,
// Copyd<T>, int[Copyd]/int[] (≈Copyd<Array>/Array), null.
func (c *checker) substType(s string, subst map[string]*Type, pos Pos) (*Type, error) {
	s = strings.TrimSpace(s)
	if s == "null" {
		return &Type{Kind: tNull}, nil
	}
	if s == "interface{}" || s == "any" {
		return tAnyV, nil // both any and void are implemented as the empty interface (xmind §interfaces)
	}
	if s == "Self" {
		if subst != nil {
			if t, ok := subst["Self"]; ok {
				return t, nil // impl case: Self = the impl's type
			}
		}
		return &Type{Kind: tTypeVar, FName: "Self"}, nil // interface case: Self = the referenced type (xmind §interfaces)
	}
	if s == "" {
		return nil, &CheckError{Msg: "CompileError: empty type annotation", Pos: pos}
	}
	// Pointer suffix: T&
	if strings.HasSuffix(s, "&") {
		e, err := c.substType(strings.TrimSuffix(s, "&"), subst, pos)
		if err != nil {
			return nil, err
		}
		return &Type{Kind: tPtr, Elem: e}, nil
	}
	// Bare pointer: an opaque handle (FFI void*, nullable, round-trippable); Elem == nil means opaque
	if s == "pointer" {
		return &Type{Kind: tPtr}, nil
	}
	// pointer modifier: `pointer T` is equivalent to T& (xmind/user request: the pointer modifier)
	if strings.HasPrefix(s, "pointer ") {
		e, err := c.substType(strings.TrimPrefix(s, "pointer "), subst, pos)
		if err != nil {
			return nil, err
		}
		return &Type{Kind: tPtr, Elem: e}, nil
	}
	base, inner, suffix := splitType(s)
	// Bare type-parameter substitution (no inner type, no suffix)
	if subst != nil && inner == "" && suffix == "" {
		if t, ok := subst[base]; ok {
			return t, nil
		}
	}
	// Numeric primitives + array/Copyd suffixes
	switch base {
	case "int", "long", "char":
		if suffix == "[]" {
			return mkList(tIntV), nil
		}
		if suffix == "Copyd" {
			return &Type{Kind: tCopyd, Elem: mkList(tIntV)}, nil // int[Copyd] = Copyd<Array<int>>
		}
		return tIntV, nil
	case "float", "double":
		if suffix == "[]" {
			return mkList(tFloatV), nil
		}
		if suffix == "Copyd" {
			return &Type{Kind: tCopyd, Elem: mkList(tFloatV)}, nil
		}
		return tFloatV, nil
	case "String":
		return tStringV, nil
	case "bool":
		return tBoolV, nil
	case "void":
		return tAnyV, nil // v2: void = the default name of the empty interface interface{}
	case "FuncBuffer":
		return tFuncBufferV, nil
	case "file":
		return tFileV, nil
	case "IOStream":
		return tIOStreamV, nil
	case "InputStream", "istream", "ifstream":
		return tInputStreamV, nil
	case "OutputStream", "ostream", "ofstream":
		return tOutputStreamV, nil
	case "iofstream":
		return tInputStreamV, nil // v1: a bidirectional file stream has the input-stream type (the object both reads and writes)
	case "Channel":
		return tChannelV, nil
	case "Task", "thread":
		return tTaskV, nil
	case "memorize":
		return tMemorizeV, nil
	case "memory":
		return tMemoryV, nil
	case "taskm":
		return tTaskmV, nil
	case "List", "Array":
		e := tAnyV
		if inner != "" {
			var err error
			e, err = c.substType(inner, subst, pos)
			if err != nil {
				return nil, err
			}
		}
		lst := mkList(e)
		if suffix == "Copyd" || suffix == "[]" {
			return &Type{Kind: tCopyd, Elem: lst}, nil
		}
		return lst, nil
	case "Copyd":
		if inner == "" {
			return &Type{Kind: tCopyd, Elem: tAnyV}, nil
		}
		e, err := c.substType(inner, subst, pos)
		if err != nil {
			return nil, err
		}
		return &Type{Kind: tCopyd, Elem: e}, nil
	case "HashTable":
		k, v, err := splitTopComma(inner)
		if err != nil {
			return nil, &CheckError{Msg: "CompileError: " + err.Error(), Pos: pos}
		}
		kt, err := c.substType(k, subst, pos)
		if err != nil {
			return nil, err
		}
		vt, err := c.substType(v, subst, pos)
		if err != nil {
			return nil, err
		}
		tbl := mkTable(kt, vt)
		if suffix == "Copyd" || suffix == "[]" {
			return &Type{Kind: tCopyd, Elem: tbl}, nil
		}
		return tbl, nil
	}
	// Generic struct instantiation: node / node<T> / node<T, U>
	if def, ok := c.structs[base]; ok {
		var args []*Type
		if inner != "" {
			for _, a := range splitTopCommas(inner) {
				at, err := c.substType(a, subst, pos)
				if err != nil {
					return nil, err
				}
				args = append(args, at)
			}
			if len(def.TypeParams) > 0 && len(args) != len(def.TypeParams) {
				return nil, &CheckError{Msg: fmt.Sprintf("CompileError: %s takes %d type argument(s), got %d", base, len(def.TypeParams), len(args)), Pos: pos}
			}
		}
		return &Type{Kind: tStruct, FName: base, Args: args}, nil
	}
	if def, ok := c.interfaces[base]; ok {
		// Generic interface instantiation: Index<Vec<int>> carries its type arguments so the interface's
		// method signatures can be substituted (T -> Vec<int>) wherever they are consulted.
		var args []*Type
		if inner != "" {
			for _, a := range splitTopCommas(inner) {
				at, err := c.substType(a, subst, pos)
				if err != nil {
					return nil, err
				}
				args = append(args, at)
			}
		}
		if len(def.TypeParams) > 0 && len(args) != len(def.TypeParams) {
			return nil, &CheckError{Msg: fmt.Sprintf("CompileError: %s takes %d type argument(s), got %d", base, len(def.TypeParams), len(args)), Pos: pos}
		}
		if len(def.TypeParams) == 0 && len(args) > 0 {
			return nil, &CheckError{Msg: fmt.Sprintf("CompileError: interface %s is not generic, but %d type argument(s) were given", base, len(args)), Pos: pos}
		}
		return &Type{Kind: tInterface, FName: base, Args: args}, nil
	}
	// Function type function<ret, p1, ...>: tFunc (signature string)
	if base == "function" {
		return mkFunc(s), nil
	}
	// Type alias expansion
	if alias, ok := c.aliases[base]; ok {
		return c.substType(alias, subst, pos)
	}
	// Type variable of a generic function (fn<T,...>): T is a type variable while in scope
	if c.typeVars[base] {
		return &Type{Kind: tTypeVar, FName: base}, nil
	}
	return nil, &CheckError{Msg: fmt.Sprintf("CompileError: unknown type %q", s), Pos: pos}
}

// isBuiltinFuncName reports whether a name is a builtin function (usable as a function reference).
func isBuiltinFuncName(s string) bool {
	switch s {
	case "qkexec", "qkexecv", "qkpopen", "qkhttp_get", "qkhttp_post", "qkjson_dumps", "qkjson_loads", "qkfile_read", "qkfile_write", "qksignal_emit":
		return true
	}
	return false
}

// funcTypeRet extracts the return type from function<ret, p1, ...>.
func funcTypeRet(fn string) (string, bool) {
	const p = "function<"
	if !strings.HasPrefix(fn, p) || !strings.HasSuffix(fn, ">") {
		return "", false
	}
	inner := fn[len(p) : len(fn)-1]
	if i := strings.IndexByte(inner, ','); i >= 0 {
		return strings.TrimSpace(inner[:i]), true
	}
	return strings.TrimSpace(inner), true
}

// splitTopCommas splits on top-level commas (ignoring commas inside angle brackets).
func splitTopCommas(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if rest := strings.TrimSpace(s[start:]); rest != "" {
		parts = append(parts, rest)
	}
	return parts
}

func (c *checker) errf(pos Pos, format string, args ...interface{}) error {
	return &CheckError{Msg: msg(format, args...), Pos: pos}
}

func (c *checker) paramType(p Param, pos Pos) (*Type, error) {
	return c.substType(p.Type, c.curSubst, p.Pos)
}

func (c *checker) checkFunc(f *Func) error {
	// Generic function: type parameters enter scope first (both the return type and parameters may use T)
	prevVars := c.typeVars
	if len(f.TypeParams) > 0 {
		c.typeVars = map[string]bool{}
		for _, tp := range f.TypeParams {
			c.typeVars[tp] = true
		}
	}
	defer func() { c.typeVars = prevVars }()

	var retType *Type
	if f.Ret != "" {
		t, err := c.substType(f.Ret, c.curSubst, f.Pos)
		if err != nil {
			return err
		}
		retType = t
	}
	prev := c.curRet
	c.curRet = retType
	defer func() { c.curRet = prev }()

	sc := newCScope(nil)
	for _, p := range f.Params {
		if p.Type == "" {
			return &CheckError{Msg: fmt.Sprintf("CompileError: parameter %q of %s is missing a type annotation", p.Name, f.Name), Pos: p.Pos}
		}
		t, err := c.paramType(p, f.Pos)
		if err != nil {
			return err
		}
		// Parameter modifiers (same as the declaration form): const = not assignable inside the callee; copyd only affects binding-time copying (eval side)
		perm, scope := p.Perm, p.Scope
		if perm == "" {
			perm, scope = refDefaultPerm, refDefaultScope
		}
		if err := sc.declare(p.Name, &cVar{typ: t, init: true, isConst: p.Decor == "const", refPerm: perm, refScope: scope}, p.Pos); err != nil {
			return err
		}
	}
	return c.checkBlock(f.Body, sc)
}

func (c *checker) checkBlock(b *Block, sc *cScope) error {
	for _, st := range b.Stmts {
		if err := c.checkStmt(st, sc); err != nil {
			return err
		}
	}
	return nil
}

func (c *checker) checkStmt(st Stmt, sc *cScope) error {
	switch s := st.(type) {
	case *ExprStmt:
		_, err := c.infer(s.X, sc)
		return err
	case *DeleteStmt:
		_, err := c.infer(s.X, sc)
		return err
	case *LogStmt:
		_, err := c.infer(s.X, sc)
		return err
	case *TryStmt:
		if err := c.checkBlock(s.Try, sc); err != nil {
			return err
		}
		ct, err := c.resolveType(s.CatchVarType, s.Pos)
		if err != nil {
			return err
		}
		inner := newCScope(sc)
		if err := inner.declare(s.CatchVar, &cVar{typ: ct, init: true}, s.Pos); err != nil {
			return err
		}
		return c.checkBlock(s.Catch, inner)
	case *ReturnStmt:
		if s.X != nil {
			t, err := c.infer(s.X, sc)
			if err != nil {
				return err
			}
			if c.curRet != nil && !assignable(t, c.curRet) {
				return c.errf(s.Pos, "TypeError: return type is %s, got %s", c.curRet, t)
			}
			if c.curRet != nil {
				if err := c.checkIfaceStrict(t, c.curRet, s.Pos, "return"); err != nil {
					return err
				}
			}
		}
		return nil
	case *BreakStmt:
		if c.loopDepth == 0 {
			return c.errf(s.Pos, "CompileError: break 只能在 while/for 循环体内")
		}
		return nil
	case *IfStmt:
		if err := c.requireBool(s.Cond, sc); err != nil {
			return err
		}
		if err := c.checkBlock(s.Then, sc); err != nil {
			return err
		}
		if s.Else != nil {
			return c.checkBlock(s.Else, sc)
		}
		return nil
	case *WhileStmt:
		if err := c.requireBool(s.Cond, sc); err != nil {
			return err
		}
		c.loopDepth++
		err := c.checkBlock(s.Body, sc)
		c.loopDepth--
		return err
	case *ForStmt:
		it, err := c.infer(s.Iter, sc)
		if err != nil {
			return err
		}
		if it.Kind != tList {
			return c.errf(s.Pos, "TypeError: for-in requires a List, got %s", it)
		}
		inner := newCScope(sc)
		if err := inner.declare(s.Var, &cVar{typ: it.Elem, init: true}, s.Pos); err != nil {
			return err
		}
		if s.Type != "" { // loop variable type declared first: for (<type> <name> : <list>)
			dt, err := c.substType(s.Type, c.curSubst, s.Pos)
			if err != nil {
				return err
			}
			if !assignable(it.Elem, dt) {
				return c.errf(s.Pos, "TypeError: for 迭代变量类型 %s 与元素类型 %s 不匹配", dt, it.Elem)
			}
			if err := c.checkIfaceStrict(it.Elem, dt, s.Pos, "for 迭代变量"); err != nil {
				return err
			}
		}
		c.loopDepth++
		err2 := c.checkBlock(s.Body, inner)
		c.loopDepth--
		return err2
	case *ForCStmt:
		// C style: for (<init>; <cond>; <step>) { ... }
		inner := newCScope(sc)
		if s.Init != nil {
			if err := c.checkStmt(s.Init, inner); err != nil {
				return err
			}
		}
		if err := c.requireBool(s.Cond, inner); err != nil {
			return err
		}
		if s.Step != nil {
			if err := c.checkStmt(s.Step, inner); err != nil {
				return err
			}
		}
		c.loopDepth++
		err := c.checkBlock(s.Body, inner)
		c.loopDepth--
		return err
	case *DeclStmt:
		// Variable modifiers: copyd = copy on pass (the type annotation gets [Copyd] appended); const = constant
		typStr := s.Type
		if s.Decor == "copyd" {
			typStr = typStr + "[Copyd]"
		}
		typ, err := c.substType(typStr, c.curSubst, s.Pos)
		if err != nil {
			return err
		}
		perm, scope := s.Perm, s.Scope
		if perm == "" {
			perm, scope = refDefaultPerm, refDefaultScope
		}
		v := &cVar{typ: typ, init: false, isConst: s.Decor == "const", refPerm: perm, refScope: scope}
		if s.Init != nil {
			// .{...} literal → named struct: fields are matched and the type name is bound (xmind §struct literals)
			if sl, ok := s.Init.(*StructLit); ok && typ.Kind == tStruct && typ.FName != "." {
				def, has := c.structs[typ.FName]
				if !has {
					return c.errf(s.Pos, "CompileError: unknown struct %q", typ.FName)
				}
				if len(sl.Fields) != len(def.Types) {
					return c.errf(s.Pos, "TypeError: %s 需要 %d 个字段，字面量给了 %d", typ.FName, len(def.Types), len(sl.Fields))
				}
				for i, f := range sl.Fields {
					fname := f.Name
					if fname == "" && i < len(def.TypesOrder) {
						fname = def.TypesOrder[i]
					}
					ft, has := def.Types[fname]
					if !has {
						return c.errf(s.Pos, "TypeError: %s 没有字段 %q", typ.FName, fname)
					}
					fv, err := c.infer(f.X, sc)
					if err != nil {
						return err
					}
					subst := c.instanceSubst(def, typ)
					want, err := c.substType(ft, subst, s.Pos)
					if err != nil {
						return err
					}
					if err := c.checkIfaceStrict(fv, want, s.Pos, "struct literal "+s.Name+"."+f.Name); err != nil {
						return err
					}
					if !assignable(fv, want) {
						return c.errf(s.Pos, "TypeError: 字段 %q: cannot assign %s to %s", f.Name, fv, want)
					}
				}
				sl.Name = typ.FName
				v.init = true
			} else {
				it, err := c.infer(s.Init, sc)
				if err != nil {
					return err
				}
				if !assignable(it, typ) {
					return c.errf(s.Pos, "TypeError: cannot assign %s to %s", it, typ)
				}
				// R2 reference rules apply to declarations too, not only to later assignments: a
				// reference may not be parked in a wider-scoped variable, and a more restrictive
				// permission may not be widened.
				if isRefKind(typ) {
					if src, ok := s.Init.(*Ident); ok {
						if sv := sc.lookup(src.Name); sv != nil && isRefKind(sv.typ) {
							if !refPermCovers(sv.refPerm, perm) {
								return c.errf(s.Pos, msg("TypeError: cannot assign a %q reference to a %q reference (the source is more restrictive)", sv.refPerm, perm))
							}
							if refScopeRank(sv.refScope) < refScopeRank(scope) {
								return c.errf(s.Pos, msg("TypeError: cannot store a %s-scoped reference in a %s-scoped variable (the variable would outlive the reference)", refScopeName(sv.refScope), refScopeName(scope)))
							}
						}
					}
				}
				if err := c.checkIfaceStrict(it, typ, s.Pos, "decl "+s.Name); err != nil {
					return err
				}
				v.init = true
			}
		} else if typ.Kind == tStruct || typ.Kind == tPtr {
			v.init = true // struct zero-value instance / pointer zero value (null) is usable
		}
		return sc.declare(s.Name, v, s.Pos)
	case *AssignStmt:
		t, err := c.infer(s.X, sc)
		if err != nil {
			return err
		}
		switch target := s.Target.(type) {
		case *Ident:
			v := sc.lookup(target.Name)
			if v == nil {
				return c.errf(target.Pos, "CompileError: assignment to undeclared identifier %q", target.Name)
			}
			if v.isConst {
				return c.errf(target.Pos, "CompileError: const 变量 %q 不可重新赋值", target.Name)
			}
			// R2 permission enforcement (write side): writing through a reference requires the w
			// permission. This is the same rule the compiler front end applies, and it is what makes
			// "&r int p" read-only rather than merely documented as read-only.
			if isRefKind(v.typ) && !refAllows(v.refPerm, "w") {
				return c.errf(target.Pos, msg("TypeError: reference %q has permission %q and cannot be written through (w is missing)", target.Name, v.refPerm))
			}
			if !assignable(t, v.typ) {
				return c.errf(s.Pos, "TypeError: cannot assign %s to %s", t, v.typ)
			}
			// R2 permission enforcement (reference assignment): the source must grant everything the
			// destination requires, and its scope must not be narrower than the destination's.
			if isRefKind(v.typ) {
				if src, isSrcIdent := s.X.(*Ident); isSrcIdent {
					if sv := sc.lookup(src.Name); sv != nil && isRefKind(sv.typ) {
						if !refPermCovers(sv.refPerm, v.refPerm) {
							return c.errf(s.Pos, msg("TypeError: cannot assign a %q reference to a %q reference (the source is more restrictive)", sv.refPerm, v.refPerm))
						}
						if refScopeRank(sv.refScope) < refScopeRank(v.refScope) {
							return c.errf(s.Pos, msg("TypeError: cannot store a %s-scoped reference in a %s-scoped variable (the variable would outlive the reference)", refScopeName(sv.refScope), refScopeName(v.refScope)))
						}
					}
				}
			}
			if err := c.checkIfaceStrict(t, v.typ, s.Pos, "assign "+target.Name); err != nil {
				return err
			}
			v.init = true
			return nil
		case *IndexExpr:
			recvT, err := c.infer(target.X, sc)
			if err != nil {
				return err
			}
			kidx, err := c.infer(target.Idx, sc)
			if err != nil {
				return err
			}
			switch recvT.Kind {
			case tHashTable:
				if recvT.Key != nil && !assignable(kidx, recvT.Key) {
					return c.errf(target.Pos, "TypeError: 索引键需要 %s，给了 %s", recvT.Key, kidx)
				}
				if recvT.Elem != nil && !assignable(t, recvT.Elem) {
					return c.errf(target.Pos, "TypeError: 值需要 %s，给了 %s", recvT.Elem, t)
				}
				if err := c.checkIfaceStrict(t, recvT.Elem, target.Pos, "table value"); err != nil {
					return err
				}
			case tList:
				if kidx.Kind != tInt {
					return c.errf(target.Pos, "TypeError: 列表索引必须是 int")
				}
				if recvT.Elem != nil && !assignable(t, recvT.Elem) {
					return c.errf(target.Pos, "TypeError: 值需要 %s，给了 %s", recvT.Elem, t)
				}
				if err := c.checkIfaceStrict(t, recvT.Elem, target.Pos, "list element"); err != nil {
					return err
				}
			default:
				return c.errf(target.Pos, "TypeError: 不支持对 %s 索引赋值", recvT)
			}
			return nil
		case *MemberExpr:
			objT, err := c.infer(target.X, sc)
			if err != nil {
				return err
			}
			if objT.Kind != tStruct {
				return c.errf(s.Pos, "TypeError: cannot assign member of %s", objT)
			}
			def, ok := c.structs[objT.FName]
			if !ok {
				return c.errf(s.Pos, "TypeError: unknown struct %s", objT.FName)
			}
			fieldT, ok := def.Types[target.Name]
			if !ok {
				return c.errf(target.Pos, "TypeError: no member %q on %s", target.Name, objT.FName)
			}
			ft, err := c.substType(fieldT, c.instanceSubst(def, objT), target.Pos)
			if err != nil {
				return err
			}
			if !assignable(t, ft) {
				return c.errf(s.Pos, "TypeError: cannot assign %s to %s.%s (%s)", t, objT.FName, target.Name, ft)
			}
			if err := c.checkIfaceStrict(t, ft, s.Pos, objT.FName+"."+target.Name); err != nil {
				return err
			}
			return nil
		}
		return c.errf(s.Pos, "TypeError: unsupported assignment target")
	}
	return nil
}

func (c *checker) requireBool(e Expr, sc *cScope) error {
	t, err := c.infer(e, sc)
	if err != nil {
		return err
	}
	if t.Kind != tBool {
		return c.errf(posOf(e), "TypeError: condition must be bool, got %s", t)
	}
	return nil
}

// posOf returns an approximate position for an expression.
// exprKindName names an expression form for error messages ("a call", "a literal", …).
func exprKindName(e Expr) string {
	switch e.(type) {
	case *Ident:
		return "an identifier"
	case *IntLit, *FloatLit, *StrLit, *BoolLit, *NullLit:
		return "a literal"
	case *CallExpr:
		return "a call"
	case *BinOp:
		return "an expression"
	case *IndexExpr:
		return "an element access"
	case *MemberExpr:
		return "a field access"
	}
	return "this expression"
}

func posOf(e Expr) Pos {
	switch x := e.(type) {
	case *IntLit:
		return x.Pos
	case *FloatLit:
		return x.Pos
	case *StrLit:
		return x.Pos
	case *BoolLit:
		return x.Pos
	case *NullLit:
		return x.Pos
	case *StructLit:
		return x.Pos
	case *Ident:
		return x.Pos
	case *BinOp:
		return x.Pos
	case *UnOp:
		return x.Pos
	case *IncExpr:
		return x.Pos
	case *CallExpr:
		return x.Pos
	case *MemberExpr:
		return x.Pos
	case *ScopeCall:
		return x.Pos
	case *IndexExpr:
		return x.Pos
	}
	return Pos{}
}

// join computes the common type of a list literal's elements.
func join(a, b *Type) *Type {
	if a == nil {
		return b
	}
	if a.Kind == tAny || b.Kind == tAny {
		return tAnyV
	}
	if a.Kind == b.Kind {
		if a.Kind == tList && a.Elem.Kind == b.Elem.Kind {
			return mkList(join(a.Elem, b.Elem))
		}
		return a
	}
	if (a.Kind == tInt && b.Kind == tFloat) || (a.Kind == tFloat && b.Kind == tInt) {
		return tFloatV
	}
	return tAnyV
}

func (c *checker) infer(e Expr, sc *cScope) (*Type, error) {
	switch x := e.(type) {
	case *IntLit:
		return tIntV, nil
	case *FloatLit:
		return tFloatV, nil
	case *StrLit:
		return tStringV, nil
	case *BoolLit:
		return tBoolV, nil
	case *NullLit:
		return &Type{Kind: tNull}, nil
	case *NewExpr:
		// new <type>[size]: returns a pointer (to List<T>; a single element points at T)
		elem, err := c.substType(x.Typ, c.curSubst, x.Pos)
		if err != nil {
			return nil, err
		}
		if x.Size != nil {
			if _, err := c.infer(x.Size, sc); err != nil {
				return nil, err
			}
			return &Type{Kind: tPtr, Elem: mkList(elem)}, nil
		}
		return &Type{Kind: tPtr, Elem: elem}, nil
	case *StructLit:
		// Anonymous struct literal (such as .{in,out}): an anonymous struct with a field-type table
		fields := map[string]*Type{}
		for _, f := range x.Fields {
			ft, err := c.infer(f.X, sc)
			if err != nil {
				return nil, err
			}
			fields[f.Name] = ft
		}
		return &Type{Kind: tStruct, FName: ".", Fields: fields}, nil
	case *IncExpr:
		// R2 move permission: advancing a reference is what m exists for.
		id, ok := x.X.(*Ident)
		if !ok {
			return nil, c.errf(x.Pos, msg("TypeError: ++ moves a reference variable, got %s", exprKindName(x.X)))
		}
		v := sc.lookup(id.Name)
		if v == nil {
			return nil, c.errf(x.Pos, msg("CompileError: undeclared identifier %q", id.Name))
		}
		if !isRefKind(v.typ) {
			return nil, c.errf(x.Pos, msg("TypeError: ++ needs a reference (permission m), but %q is %s", id.Name, v.typ))
		}
		if !refAllows(v.refPerm, "m") {
			return nil, c.errf(x.Pos, msg("TypeError: reference %q has permission %q and cannot move (m is missing)", id.Name, v.refPerm))
		}
		return v.typ, nil
	case *Ident:
		// R2 permission enforcement (read side): using a reference as a value requires the r
		// permission. Assignment targets never reach this path (the assignment case resolves the target
		// through the scope directly), so a write-only reference stays assignable but unreadable.
		if v := sc.lookup(x.Name); v != nil {
			if isRefKind(v.typ) && !refAllows(v.refPerm, "r") {
				return nil, c.errf(x.Pos, msg("TypeError: reference %q has permission %q and cannot be read (r is missing)", x.Name, v.refPerm))
			}
		}
		if c.libs != nil {
			if lb, ok := c.libs[x.Name]; ok {
				return &Type{Kind: tLib, FName: lb.Name}, nil // system library object (library X binding)
			}
		}
		if x.Name == "true" || x.Name == "false" {
			return tBoolV, nil
		}
		if x.Name == "memory" || x.Name == "GlobalMemory" {
			return tMemoryV, nil
		}
		if x.Name == "taskm" {
			return tTaskmV, nil
		}
		if x.Name == "DynamicStackAndHeap" {
			return tStringV, nil // experimental memory-mode constant (xmind §memory)
		}
		if v := sc.lookup(x.Name); v != nil {
			if !v.init {
				return nil, c.errf(x.Pos, "CompileError: variable %q used before initialization", x.Name)
			}
			return v.typ, nil
		}
		if _, ok := c.fns[x.Name]; ok {
			return mkFunc(x.Name), nil
		}
		if isBuiltinFuncName(x.Name) {
			return mkFunc(x.Name), nil // builtin function as a function reference (sum(rand, ...))
		}
		return nil, c.errf(x.Pos, "CompileError: undeclared identifier %q", x.Name)
	case *ListLit:
		var elem *Type
		for _, it := range x.Items {
			t, err := c.infer(it, sc)
			if err != nil {
				return nil, err
			}
			elem = join(elem, t)
		}
		if elem == nil {
			elem = tAnyV
		}
		return mkList(elem), nil
	case *UnOp:
		t, err := c.infer(x.X, sc)
		if err != nil {
			return nil, err
		}
		switch x.Op {
		case "&":
			// Address-of: only lvalues have storage to point at, and the result is a bare reference
			// (full permission, program scope) unless a declaration narrows it.
			switch x.X.(type) {
			case *Ident, *MemberExpr, *IndexExpr:
			default:
				return nil, c.errf(x.Pos, msg("TypeError: cannot take the address of this expression (needs a variable, field or list element)"))
			}
			t, err := c.infer(x.X, sc)
			if err != nil {
				return nil, err
			}
			return &Type{Kind: tPtr, Elem: t}, nil
		case "*":
			if t.Kind != tList {
				return nil, c.errf(x.Pos, "TypeError: '*' requires a List, got %s", t)
			}
			return t.Elem, nil
		case "-":
			if t.Kind == tStruct {
				for _, def := range c.implDefsFor(t.FName) {
					if fn := def.SelfMethods["__neg__"]; fn != nil {
						return c.resolveType(fn.Ret, x.Pos)
					}
				}
			}
			if t.Kind != tInt && t.Kind != tFloat {
				return nil, c.errf(x.Pos, "TypeError: unary '-' requires a number, got %s", t)
			}
			return t, nil
		case "!":
			if t.Kind != tBool {
				return nil, c.errf(x.Pos, "TypeError: '!' requires bool, got %s", t)
			}
			return tBoolV, nil
		}
		return nil, c.errf(x.Pos, "internal: unknown unary operator %s", x.Op)
	case *BinOp:
		return c.inferBin(x, sc)
	case *MemberExpr:
		recv, err := c.infer(x.X, sc)
		if err != nil {
			return nil, err
		}
		return c.memberType(recv, x.Name, x.Pos)
	case *CallExpr:
		return c.inferCall(x, sc)
	case *ScopeCall:
		return c.inferScope(x, sc)
	case *IndexExpr:
		t, err := c.infer(x.X, sc)
		if err != nil {
			return nil, err
		}
		if t.Kind != tList {
			return nil, c.errf(x.Pos, "TypeError: indexing requires a List, got %s", t)
		}
		it, err := c.infer(x.Idx, sc)
		if err != nil {
			return nil, err
		}
		if it.Kind != tInt {
			return nil, c.errf(x.Pos, "TypeError: index must be int, got %s", it)
		}
		return t.Elem, nil
	}
	return nil, c.errf(Pos{}, "internal: unknown expression node")
}

func isNumeric(t *Type) bool { return t.Kind == tInt || t.Kind == tFloat }

// sameSig reports whether two function signatures are identical (same parameter count and type annotations).
func sameSig(a, b *Func) bool {
	if len(a.Params) != len(b.Params) {
		return false
	}
	for i := range a.Params {
		if a.Params[i].Type != b.Params[i].Type {
			return false
		}
	}
	return true
}

// allDefs returns every definition of a function (primary + overloads).
func (c *checker) allDefs(name string) []*Func {
	if fn, ok := c.fns[name]; ok {
		return append([]*Func{fn}, c.overloads[name]...)
	}
	return c.overloads[name]
}

// bestMatchT picks the best overload by argument types.
func (c *checker) bestMatchT(defs []*Func, tys []*Type) *Func {
	var best *Func
	bestScore := -1
	for _, fn := range defs {
		if len(fn.Params) != len(tys) {
			continue
		}
		score := 0
		ok := true
		for i, p := range fn.Params {
			pt, err := c.substType(p.Type, c.curSubst, fn.Pos)
			if err != nil {
				pt = tAnyV
			}
			if tys[i].Kind == pt.Kind {
				score += 10
			} else if assignable(tys[i], pt) {
				score += 5
			} else if pt.Kind == tAny {
				score += 1
			} else {
				ok = false
				break
			}
		}
		if ok && score > bestScore {
			best = fn
			bestScore = score
		}
	}
	return best
}

func (c *checker) inferBin(x *BinOp, sc *cScope) (*Type, error) {
	l, err := c.infer(x.L, sc)
	if err != nil {
		return nil, err
	}
	switch x.Op {
	case "&&", "||":
		if l.Kind != tBool {
			return nil, c.errf(x.Pos, "TypeError: %s requires bool operands, got %s", x.Op, l)
		}
		r, err := c.infer(x.R, sc)
		if err != nil {
			return nil, err
		}
		if r.Kind != tBool {
			return nil, c.errf(x.Pos, "TypeError: %s requires bool operands, got %s", x.Op, r)
		}
		return tBoolV, nil
	}
	r, err := c.infer(x.R, sc)
	if err != nil {
		return nil, err
	}
	// Operation overloading: same struct and an aggregate method matches → return the method's type
	if l.Kind == tStruct && r.Kind == tStruct && l.FName == r.FName {
		if m := opMethodFor(x.Op); m != "" {
			for _, def := range c.implDefsFor(l.FName) {
				if fn := def.SelfMethods[m]; fn != nil && len(fn.Params) == 2 {
					return c.resolveType(fn.Ret, x.Pos)
				}
			}
		}
	}
	switch x.Op {
	case "+":
		if l.Kind == tString || r.Kind == tString {
			return tStringV, nil
		}
		if !isNumeric(l) || !isNumeric(r) {
			return nil, c.errf(x.Pos, "TypeError: '+' requires numbers or a String, got %s and %s", l, r)
		}
		if l.Kind == tFloat || r.Kind == tFloat {
			return tFloatV, nil
		}
		return tIntV, nil
	case "-", "*", "/", "%":
		if !isNumeric(l) || !isNumeric(r) {
			return nil, c.errf(x.Pos, "TypeError: arithmetic requires numbers, got %s and %s", l, r)
		}
		if l.Kind == tFloat || r.Kind == tFloat {
			return tFloatV, nil
		}
		return tIntV, nil
	case "<<", ">>":
		if l.Kind != tInt || r.Kind != tInt {
			return nil, c.errf(x.Pos, "TypeError: 位移运算需要 int 操作数，got %s and %s", l, r)
		}
		return tIntV, nil
	case "===":
		// Storage identity: the operands' types do not matter, only where they live, so any pair of
		// types is accepted. It is a language primitive and cannot be overloaded (opMethodFor has no
		// entry for it, so the user-method hook above never fires for ===).
		return tBoolV, nil
	case "==", "!=":
		ok := (isNumeric(l) && isNumeric(r)) || (l.Kind == tString && r.Kind == tString) ||
			(l.Kind == tBool && r.Kind == tBool) || (l.Kind == tAny || r.Kind == tAny) ||
			(l.Kind == tNull || r.Kind == tNull) || (l.Kind == tPtr || r.Kind == tPtr) ||
			(l.Kind == tStruct && r.Kind == tStruct) || (l.Kind == tCopyd || r.Kind == tCopyd)
		if !ok {
			return nil, c.errf(x.Pos, "TypeError: cannot compare %s and %s", l, r)
		}
		return tBoolV, nil
	case "<", "<=", ">", ">=":
		if (isNumeric(l) && isNumeric(r)) || (l.Kind == tString && r.Kind == tString) {
			return tBoolV, nil
		}
		return nil, c.errf(x.Pos, "TypeError: cannot order-compare %s and %s", l, r)
	}
	return nil, c.errf(x.Pos, "internal: unknown operator %s", x.Op)
}

// memberType checks member access (an undeclared member is a compile error).
func (c *checker) memberType(recv *Type, name string, pos Pos) (*Type, error) {
	switch recv.Kind {
	case tFuncBuffer, tTask:
		switch name {
		case "head":
			return mkList(tAnyV), nil
		case "tail":
			return mkList(tAnyV), nil
		case "log":
			return mkList(tStringV), nil
		}
	case tMemorize:
		// v0.1: memorize is a built-in signature state object with no public members
	case tStruct:
		if recv.Fields != nil {
			if ft, ok := recv.Fields[name]; ok {
				return ft, nil
			}
			return nil, c.errf(pos, "TypeError: no member %q on 匿名结构体", name)
		}
		if def, ok := c.structs[recv.FName]; ok {
			if typ, exists := def.Types[name]; exists {
				subst := c.instanceSubst(def, recv)
				return c.substType(typ, subst, pos)
			}
		}
	case tPtr:
		if recv.Elem == nil { // opaque handle (FFI void*): no members
			return nil, c.errf(pos, "TypeError: no member %q on pointer", name)
		}
		// Pointer member access dereferences automatically
		return c.memberType(recv.Elem, name, pos)
	case tCopyd:
		return c.memberType(recv.Elem, name, pos)
	}
	return nil, c.errf(pos, "TypeError: no member %q on %s", name, recv)
}

// instanceSubst builds the type-parameter substitution table for an instance (curSubst inside a generic method takes precedence).
func (c *checker) instanceSubst(def *StructDef, recv *Type) map[string]*Type {
	subst := map[string]*Type{}
	for i, tp := range def.TypeParams {
		if i < len(recv.Args) {
			subst[tp] = recv.Args[i]
		} else {
			subst[tp] = tAnyV
		}
	}
	for k, v := range c.curSubst {
		subst[k] = v
	}
	return subst
}

func (c *checker) checkArity(name string, want, got int, pos Pos) error {
	if want != got {
		return c.errf(pos, "CompileError: %s() expects %d args, got %d", name, want, got)
	}
	return nil
}

// methodType checks a method call (receiver type + argument count + argument types).
func (c *checker) methodType(recv *Type, name string, args []*Type, pos Pos) (*Type, error) {
	switch recv.Kind {
	case tInterface:
		iface, ok := c.interfaces[recv.FName]
		if !ok {
			return nil, c.errf(pos, "TypeError: 未知接口 %q", recv.FName)
		}
		methods := append([]MethodSig{}, iface.Methods...)
		for _, ex := range iface.Expands {
			if ei, ok := c.interfaces[ex]; ok {
				methods = append(methods, ei.Methods...)
			}
		}
		for _, sig := range methods {
			if sig.Name == name {
				want := len(sig.Params)
				if want > 0 {
					p0 := sig.Params[0]
					if isRecvParam(recv.FName, &p0) {
						want-- // the interface signature's receiver is identified by **type** (Self or the interface's own type), independent of the parameter name
					}
				}
				if len(args) != want {
					return nil, c.errf(pos, "TypeError: %s.%s 需要 %d 个参数，给了 %d", recv.FName, name, want, len(args))
				}
				ret := sig.Ret
				if ret == "" || ret == "void" {
					return tNilV, nil
				}
				if ret == "Self" {
					return tAnyV, nil // Self return: the concrete runtime value
				}
				// Generic interface: substitute its type parameters with the receiver's arguments, so
				// Index<T> { fn get(Self self) T; } used as Index<Box> returns Box rather than the
				// unresolved parameter T.
				if len(iface.TypeParams) > 0 {
					subst := map[string]*Type{}
					for i, tp := range iface.TypeParams {
						if i < len(recv.Args) {
							subst[tp] = recv.Args[i]
						}
					}
					if _, isParam := subst[ret]; isParam {
						return subst[ret], nil
					}
					return c.substType(ret, subst, pos)
				}
				return c.resolveType(ret, pos)
			}
		}
		return nil, c.errf(pos, "TypeError: 接口 %s 没有方法 %q", recv.FName, name)
	case tLib:
		if c.libs == nil {
			return nil, c.errf(pos, "TypeError: 未知系统库 %q", recv.FName)
		}
		lb := c.libs[recv.FName]
		for _, fn := range lb.Methods {
			if fn.Name == name {
				if len(args) != len(fn.Params) {
					return nil, c.errf(pos, "LibraryError: %s.%s 需要 %d 个参数，给了 %d", recv.FName, name, len(fn.Params), len(args))
				}
				for i, p := range fn.Params {
					want := tFloatV
					if p.Type != "f32" {
						rt, err := c.resolveType(p.Type, pos)
						if err != nil {
							return nil, err
						}
						want = rt
					}
					if !assignable(args[i], want) {
						return nil, c.errf(pos, "TypeError: %s.%s 参数 %s 需要 %s，给了 %s", recv.FName, name, p.Name, want, args[i])
					}
				}
				if fn.Ret == "" || fn.Ret == "void" {
					return tNilV, nil
				}
				ret := fn.Ret
				if ret == "f32" {
					return tFloatV, nil
				}
				return c.resolveType(ret, pos)
			}
		}
		return nil, c.errf(pos, "LibraryError: 库 %s 没有导出函数 %q", recv.FName, name)
	case tInt, tFloat, tBool:
		if name == "toString" {
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tStringV, nil
		}
		return nil, c.errf(pos, "TypeError: no method %q", name)
	case tFile:
		switch name {
		case "read", "name":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tStringV, nil
		case "write":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		}
		return nil, c.errf(pos, "TypeError: file 没有方法 %q", name)
	case tString:
		switch name {
		case "size", "indexOf", "toInt":
			if err := c.checkArity(name, 0, len(args), pos); err != nil && name != "indexOf" {
				return nil, err
			}
			if name == "indexOf" {
				if err := c.checkArity(name, 1, len(args), pos); err != nil {
					return nil, err
				}
				if args[0].Kind != tString {
					return nil, c.errf(pos, "TypeError: indexOf 需要 String 参数")
				}
			}
			if name == "toInt" {
				return tIntV, nil
			}
			return tIntV, nil
		case "contains", "startsWith", "endsWith":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tString {
				return nil, c.errf(pos, "TypeError: %s 需要 String 参数", name)
			}
			return tBoolV, nil
		case "substring":
			if len(args) < 1 || len(args) > 2 {
				return nil, c.errf(pos, "TypeError: substring(start, end?) 需要 1-2 个 int 参数")
			}
			for _, a := range args {
				if a.Kind != tInt {
					return nil, c.errf(pos, "TypeError: substring 参数必须是 int")
				}
			}
			return tStringV, nil
		case "split":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tString {
				return nil, c.errf(pos, "TypeError: split 需要 String 分隔符")
			}
			return c.resolveType("List<String>", pos)
		case "trim", "trimLeft", "trimRight", "toLower", "toUpper", "charAt", "toFloat":
			if name == "charAt" {
				if err := c.checkArity(name, 1, len(args), pos); err != nil {
					return nil, err
				}
				if args[0].Kind != tInt {
					return nil, c.errf(pos, "TypeError: charAt 需要 int 索引")
				}
			} else if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tStringV, nil
		case "replace":
			if err := c.checkArity(name, 2, len(args), pos); err != nil {
				return nil, err
			}
			for _, a := range args {
				if a.Kind != tString {
					return nil, c.errf(pos, "TypeError: replace(old, new) 需要 String 参数")
				}
			}
			return tStringV, nil
		}
	case tList:
		switch name {
		case "head", "tail", "size":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tIntV, nil
		case "get":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tInt {
				return nil, c.errf(pos, "TypeError: get(i) 需要 int 下标，得到 %s", args[0])
			}
			return recv.Elem, nil
		case "next":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return recv.Elem, nil
		case "reset":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return recv, nil
		case "append":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if !assignable(args[0], recv.Elem) {
				return nil, c.errf(pos, "TypeError: append expects %s, got %s", recv.Elem, args[0])
			}
			if err := c.checkIfaceStrict(args[0], recv.Elem, pos, "append"); err != nil {
				return nil, err
			}
			return tNilV, nil
		case "appendAll":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tList {
				return nil, c.errf(pos, "TypeError: appendAll requires a List, got %s", args[0])
			}
			return tNilV, nil
		case "toString":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tStringV, nil
		case "__sort__":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		}
	case tHashTable:
		switch name {
		case "put":
			if err := c.checkArity(name, 2, len(args), pos); err != nil {
				return nil, err
			}
			if !assignable(args[0], recv.Key) {
				return nil, c.errf(pos, "TypeError: put key expects %s, got %s", recv.Key, args[0])
			}
			if !assignable(args[1], recv.Val) {
				return nil, c.errf(pos, "TypeError: put value expects %s, got %s", recv.Val, args[1])
			}
			if err := c.checkIfaceStrict(args[1], recv.Val, pos, "put value"); err != nil {
				return nil, err
			}
			return tNilV, nil
		case "get":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			return recv.Val, nil
		case "contains":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			return tBoolV, nil
		case "keys":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return mkList(tStringV), nil

		case "remove":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		case "size":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tIntV, nil
		}
	case tFuncBuffer:
		if name == "execute" {
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tFuncBufferV, nil
		}
	case tInputStream:
		switch name {
		case "readln":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tStringV, nil
		case "close":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		}
	case tOutputStream:
		switch name {
		case "println", "print":
			return tNilV, nil
		case "write":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		case "close":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		}
	case tIOStream:
		switch name {
		case "println", "print":
			return tNilV, nil
		case "setIn":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tString && args[0].Kind != tInputStream {
				return nil, c.errf(pos, "TypeError: setIn requires a path String or an InputStream, got %s", args[0])
			}
			return tNilV, nil
		case "setOut":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tString && args[0].Kind != tOutputStream {
				return nil, c.errf(pos, "TypeError: setOut requires a path String or an OutputStream, got %s", args[0])
			}
			return tNilV, nil
		case "readln":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tStringV, nil
		}
	case tChannel:
		switch name {
		case "send":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		case "recv":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tAnyV, nil
		}
	case tTask:
		// thread class methods (xmind: merge / pid / talk)
		switch name {
		case "merge":
			if len(args) < 1 {
				return nil, c.errf(pos, "CompileError: thread.merge(fn, args...) requires at least 1 arg")
			}
			if args[0].Kind != tFunc {
				return nil, c.errf(pos, "TypeError: thread.merge second arg must be a function reference, got %s", args[0])
			}
			if args[0].FName != "" {
				if fn, ok := c.fns[args[0].FName]; ok && len(fn.Params) != len(args)-1 {
					return nil, c.errf(pos, "CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(args)-1)
				}
			}
			return tNilV, nil
		case "pid":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tIntV, nil
		case "talk":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tChannel {
				return nil, c.errf(pos, "TypeError: thread.talk requires a channel, got %s", args[0])
			}
			return tNilV, nil
		}
	case tMemory:
		switch name {
		case "clear":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		case "mode":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil
		case "compact":
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return tNilV, nil // compact() does not return at all
		case "setBlock":
			if err := c.checkArity(name, 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tInt {
				return nil, c.errf(pos, "TypeError: setBlock(n) requires an int block size, got %s", args[0])
			}
			return tNilV, nil
		}
	case tTaskm:
		switch name {
		case "spawn":
			// xmind: taskm.spawn() takes no arguments and returns a thread instance
			if len(args) != 0 {
				return nil, c.errf(pos, "CompileError: taskm.spawn() takes no args, got %d", len(args))
			}
			return tTaskV, nil
		case "merge":
			// v2: taskm.merge(pid, fn, args...)
			if len(args) < 2 {
				return nil, c.errf(pos, "CompileError: taskm.merge(pid, fn, args...) requires at least 2 args, got %d", len(args))
			}
			if args[0].Kind != tInt {
				return nil, c.errf(pos, "TypeError: taskm.merge first arg must be a pid (int), got %s", args[0])
			}
			if args[1].Kind != tFunc {
				return nil, c.errf(pos, "TypeError: taskm.merge second arg must be a function reference, got %s", args[1])
			}
			if args[1].FName != "" {
				if fn, ok := c.fns[args[1].FName]; ok {
					if len(fn.Params) != len(args)-2 {
						return nil, c.errf(pos, "CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(args)-2)
					}
					for i, p := range fn.Params {
						pt, err := c.paramType(p, pos)
						if err != nil {
							return nil, err
						}
						if !assignable(args[i+2], pt) {
							return nil, c.errf(pos, "TypeError: argument %d of %s: cannot assign %s to %s", i+1, fn.Name, args[i+2], pt)
						}
					}
				}
			}
			return tNilV, nil
		case "block":
			// v2: taskm.block(pid) returns void
			if err := c.checkArity("taskm.block", 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tTask && args[0].Kind != tInt {
				return nil, c.errf(pos, "TypeError: taskm.block requires a pid, got %s", args[0])
			}
			return tNilV, nil
		case "done":
			if err := c.checkArity("taskm.done", 1, len(args), pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tInt {
				return nil, c.errf(pos, "TypeError: taskm.done requires a pid (int), got %s", args[0])
			}
			return tBoolV, nil
		case "channel":
			if len(args) == 1 {
				if args[0].Kind != tInt {
					return nil, c.errf(pos, "TypeError: taskm.channel(n) requires an int capacity, got %s", args[0])
				}
			} else if len(args) != 0 {
				return nil, c.errf(pos, "CompileError: taskm.channel() expects 0 or 1 args, got %d", len(args))
			}
			return tChannelV, nil
		}
	case tStruct:
		defs := c.implDefsFor(recv.FName)
		if len(defs) == 0 {
			return nil, c.errf(pos, "TypeError: type %s has no impl", recv.FName)
		}
		if sd, ok := c.structs[recv.FName]; ok {
			prev := c.curSubst
			c.curSubst = c.instanceSubst(sd, recv)
			defer func() { c.curSubst = prev }()
		}
		if fn := c.selfMeth(recv.FName, name); fn != nil {
			if len(fn.Params)-1 != len(args) {
				return nil, c.errf(pos, "CompileError: %s() expects %d args, got %d", name, len(fn.Params)-1, len(args))
			}
			for i, p := range fn.Params[1:] {
				pt, err := c.paramType(p, pos)
				if err != nil {
					return nil, err
				}
				if !assignable(args[i], pt) {
					return nil, c.errf(pos, "TypeError: argument %d of %s: cannot assign %s to %s", i+1, name, args[i], pt)
				}
			}
			if fn.Ret != "" {
				return c.substType(fn.Ret, c.curSubst, pos)
			}
			return tFuncBufferV, nil
		}
		if c.staticMeth(recv.FName, name) != nil {
			return nil, c.errf(pos, "TypeError: %s is a static method; call it via %s::%s(...)", name, recv.FName, name)
		}
	case tPtr:
		if recv.Elem == nil { // opaque handle (FFI void*): supports toString (hex rendering), no other methods
			if name == "toString" {
				if err := c.checkArity(name, 0, len(args), pos); err != nil {
					return nil, err
				}
				return tStringV, nil
			}
			return nil, c.errf(pos, "TypeError: no method %q on pointer", name)
		}
		// Pointer method calls dereference automatically
		return c.methodType(recv.Elem, name, args, pos)
	case tCopyd:
		if name == "ptr" {
			if err := c.checkArity(name, 0, len(args), pos); err != nil {
				return nil, err
			}
			return recv.Elem, nil // .ptr() extracts the address wrapped by Copyd
		}
		return c.methodType(recv.Elem, name, args, pos)
	}
	return nil, c.errf(pos, "TypeError: no method %q on %s", name, recv)
}

func (c *checker) inferCall(x *CallExpr, sc *cScope) (*Type, error) {
	// Signature wrapper (§6): fn(args) @sign(prefix)
	if x.Sign != nil {
		// Builtin signature @styleConfigure(file): JSON configuration → the first argument node's style; follows the wrapped call's (method/function) type
		if x.Sign.Name == "styleConfigure" {
			if m, ok := x.Fn.(*MemberExpr); ok {
				recv, err := c.infer(m.X, sc)
				if err != nil {
					return nil, err
				}
				argTys, err := c.inferArgs(x.Args, sc)
				if err != nil {
					return nil, err
				}
				return c.methodType(recv, m.Name, argTys, x.Pos)
			}
			id, ok := x.Fn.(*Ident)
			if !ok {
				return nil, c.errf(x.Pos, "TypeError: @styleConfigure 只能包装函数/方法调用")
			}
			fn, ok := c.fns[id.Name]
			if !ok {
				return nil, c.errf(id.Pos, "CompileError: undeclared function %q", id.Name)
			}
			if err := c.checkCallArgs(fn, x.Args, sc, x.Pos); err != nil {
				return nil, err
			}
			if fn.Ret != "" {
				return c.substType(fn.Ret, c.curSubst, x.Pos)
			}
			return tNilV, nil
		}
		id, ok := x.Fn.(*Ident)
		if !ok {
			return nil, c.errf(x.Pos, "TypeError: a signature can only wrap a direct function call")
		}
		fn, ok := c.fns[id.Name]
		if !ok {
			return nil, c.errf(id.Pos, "CompileError: undeclared function %q", id.Name)
		}
		// v2 signature: @instance(prefix) — instance is a variable (a Sign instance, any name); the result type is the wrapped function's return type
		if err := c.checkCallArgs(fn, x.Args, sc, x.Pos); err != nil {
			return nil, err
		}
		// instance must be in scope (a Sign instance variable, any name)
		if v := sc.lookup(x.Sign.Name); v == nil {
			return nil, c.errf(x.Pos, "CompileError: @%s —— 签名名必须是作用域中的 Sign 实例变量（名字任意）", x.Sign.Name)
		}
		// Result type: the return type of the wrapped function fn
		if fn.Ret != "" {
			return c.substType(fn.Ret, c.curSubst, x.Pos)
		}
		return tNilV, nil
	}
	// Method call
	if m, ok := x.Fn.(*MemberExpr); ok {
		recv, err := c.infer(m.X, sc)
		if err != nil {
			return nil, err
		}
		args, err := c.inferArgs(x.Args, sc)
		if err != nil {
			return nil, err
		}
		return c.methodType(recv, m.Name, args, m.Pos)
	}
	id, ok := x.Fn.(*Ident)
	if !ok {
		return nil, c.errf(x.Pos, "TypeError: this expression is not callable")
	}
	args, err := c.inferArgs(x.Args, sc)
	if err != nil {
		return nil, err
	}
	// Function-reference variable: identifier call → resolve the return type from function<ret, p1, ...>
	if v := sc.lookup(id.Name); v != nil && v.typ.Kind == tFunc {
		if _, err := c.inferArgs(x.Args, sc); err != nil {
			return nil, err
		}
		if ret, ok := funcTypeRet(v.typ.FName); ok {
			return c.substType(ret, c.curSubst, x.Pos)
		}
		return tAnyV, nil
	}
	if defs := c.allDefs(id.Name); len(defs) > 0 {
		argTys, aerr := c.inferArgs(x.Args, sc)
		if aerr != nil {
			return nil, aerr
		}
		fn := c.bestMatchT(defs, argTys)
		if fn == nil {
			defs0 := c.allDefs(id.Name)
			if len(defs0) == 1 && len(defs0[0].Params) == len(x.Args) {
				if err := c.checkCallArgs(defs0[0], x.Args, sc, x.Pos); err != nil {
					return nil, err
				}
			}
			return nil, c.errf(id.Pos, "%s", overloadErrT(c.allDefs(id.Name), id.Name, len(x.Args)))
		}
		if err := c.checkCallArgs(fn, x.Args, sc, x.Pos); err != nil {
			return nil, err
		}
		if fn.Ret != "" {
			if len(fn.TypeParams) > 0 {
				gs, err := c.inferGenSubst(fn, x.Args, sc)
				if err != nil {
					return nil, err
				}
				return c.substType(fn.Ret, gs, x.Pos)
			}
			return c.resolveType(fn.Ret, x.Pos)
		}
		return tFuncBufferV, nil
	}
	switch id.Name {
	case "qkexec", "qkexecv":
		want := 1
		if id.Name == "qkexecv" {
			want = 2
		}
		if len(args) != want && len(args) != want+1 {
			return nil, c.errf(id.Pos, "CompileError: %s expects %d args (可加 retries), got %d", id.Name, want, len(args))
		}
		if id.Name == "qkexec" && args[0].Kind != tString {
			return nil, c.errf(id.Pos, "TypeError: qkexec requires a command String, got %s", args[0])
		}
		if id.Name == "qkexecv" {
			if args[0].Kind != tString || args[1].Kind != tList {
				return nil, c.errf(id.Pos, "TypeError: qkexecv(prog String, args List<String>)")
			}
		}
		return tIntV, nil
	case "qkpopen":
		if err := c.checkArity("qkpopen", 1, len(args), id.Pos); err != nil {
			return nil, err
		}
		if args[0].Kind != tString {
			return nil, c.errf(id.Pos, "TypeError: qkpopen requires a path String, got %s", args[0])
		}
		return tInputStreamV, nil
	case "qkhttp_get":
		if len(args) != 1 && len(args) != 2 {
			return nil, c.errf(id.Pos, "CompileError: qkhttp_get expects 1-2 args (url[, retries]), got %d", len(args))
		}
		if args[0].Kind != tString {
			return nil, c.errf(id.Pos, "TypeError: qkhttp_get requires url String, got %s", args[0])
		}
		return tStringV, nil
	case "qkfile_read":
		if err := c.checkArity(id.Name, 1, len(args), id.Pos); err != nil {
			return nil, err
		}
		return tStringV, nil
	case "qkfile_write":
		if err := c.checkArity(id.Name, 2, len(args), id.Pos); err != nil {
			return nil, err
		}
		return tNilV, nil
	case "qksignal_emit":
		if len(args) < 2 {
			return nil, c.errf(id.Pos, "CompileError: qksignal_emit(node, name String [, args...])")
		}
		return tNilV, nil
	case "qkjson_dumps":
		if err := c.checkArity("qkjson_dumps", 1, len(args), id.Pos); err != nil {
			return nil, err
		}
		return tStringV, nil
	case "qkjson_loads":
		if err := c.checkArity("qkjson_loads", 1, len(args), id.Pos); err != nil {
			return nil, err
		}
		if args[0].Kind != tString {
			return nil, c.errf(id.Pos, "TypeError: qkjson_loads requires String, got %s", args[0])
		}
		return tAnyV, nil
	case "qkhttp_post":
		if len(args) != 3 && len(args) != 4 {
			return nil, c.errf(id.Pos, "CompileError: qkhttp_post expects 3-4 args (url, body, ct[, retries]), got %d", len(args))
		}
		for _, a := range args {
			if a.Kind != tString {
				return nil, c.errf(id.Pos, "TypeError: qkhttp_post(url, body, contentType) 全部为 String")
			}
		}
		return tStringV, nil
	case "FileInputStream", "ifstream", "iofstream":
		if err := c.checkArity("FileInputStream", 1, len(args), id.Pos); err != nil {
			return nil, err
		}
		if args[0].Kind != tString {
			return nil, c.errf(id.Pos, "TypeError: FileInputStream requires a path String, got %s", args[0])
		}
		return tInputStreamV, nil
	case "FileOutputStream", "ofstream":
		if err := c.checkArity("FileOutputStream", 1, len(args), id.Pos); err != nil {
			return nil, err
		}
		if args[0].Kind != tString {
			return nil, c.errf(id.Pos, "TypeError: FileOutputStream requires a path String, got %s", args[0])
		}
		return tOutputStreamV, nil
	case "ConsoleInputStream":
		if err := c.checkArity("ConsoleInputStream", 0, len(args), id.Pos); err != nil {
			return nil, err
		}
		return tInputStreamV, nil
	case "ConsoleOutputStream":
		if err := c.checkArity("ConsoleOutputStream", 0, len(args), id.Pos); err != nil {
			return nil, err
		}
		return tOutputStreamV, nil
	case "rand", "clock":
		if err := c.checkArity(id.Name, 0, len(args), id.Pos); err != nil {
			return nil, err
		}
		return tIntV, nil
	case "sum":
		if len(args) != 3 && len(args) != 4 {
			return nil, c.errf(id.Pos, "CompileError: sum(generate, begin, stop[, step]) 需要 3 或 4 个参数，got %d", len(args))
		}
		if args[0].Kind != tFunc {
			return nil, c.errf(id.Pos, "TypeError: sum 第一个参数必须是函数引用，got %s", args[0])
		}
		for _, a := range args[1:] {
			if a.Kind != tInt {
				return nil, c.errf(id.Pos, "TypeError: sum 的边界参数必须是 int，got %s", a)
			}
		}
		return tIntV, nil
	}
	return nil, c.errf(id.Pos, "CompileError: undeclared function %q", id.Name)
}

func (c *checker) inferArgs(args []Expr, sc *cScope) ([]*Type, error) {
	out := make([]*Type, 0, len(args))
	for _, a := range args {
		t, err := c.infer(a, sc)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// inferGenSubst infers a generic function's type parameters from the arguments (fn<T,...>).
func (c *checker) inferGenSubst(fn *Func, args []Expr, sc *cScope) (map[string]*Type, error) {
	argTys, err := c.inferArgs(args, sc)
	if err != nil {
		return nil, err
	}
	genSubst := map[string]*Type{}
	for i, p := range fn.Params {
		for _, tp := range fn.TypeParams {
			if genSubst[tp] == nil && strings.Contains(p.Type, tp) && i < len(argTys) {
				genSubst[tp] = argTys[i]
			}
		}
	}
	return genSubst, nil
}

// checkCallArgs checks argument count and types against the parameters.
func (c *checker) checkCallArgs(fn *Func, args []Expr, sc *cScope, pos Pos) error {
	argTys, err := c.inferArgs(args, sc)
	if err != nil {
		return err
	}
	if len(argTys) != len(fn.Params) {
		return c.errf(pos, "CompileError: %s expects %d args, got %d", fn.Name, len(fn.Params), len(argTys))
	}
	genSubst := map[string]*Type{}
	if len(fn.TypeParams) > 0 {
		genSubst, err = c.inferGenSubst(fn, args, sc)
		if err != nil {
			return err
		}
	}
	for i, p := range fn.Params {
		pt, err := c.substType(p.Type, genSubst, pos)
		if err != nil {
			return err
		}
		if pt.Kind == tInterface {
			// Strict interface-implementation check for interface parameters: a struct/interface argument must cover every interface method (dynamic dispatch guarantee)
			if argTys[i].Kind == tStruct {
				if err := c.checkImplements(argTys[i].FName, pt.FName); err != nil {
					return c.errf(pos, "TypeError: argument %d of %s: %v", i+1, fn.Name, err)
				}
			} else if argTys[i].Kind == tInterface {
				if err := c.checkIfaceCovers(argTys[i].FName, pt.FName); err != nil {
					return c.errf(pos, "TypeError: argument %d of %s: %v", i+1, fn.Name, err)
				}
			} else if !assignable(argTys[i], pt) {
				return c.errf(pos, "TypeError: argument %d of %s: cannot assign %s to %s", i+1, fn.Name, argTys[i], pt)
			}
			continue
		}
		if !assignable(argTys[i], pt) {
			return c.errf(pos, "TypeError: argument %d of %s: cannot assign %s to %s", i+1, fn.Name, argTys[i], pt)
		}
		if err := c.checkIfaceStrict(argTys[i], pt, pos, fmt.Sprintf("argument %d of %s", i+1, fn.Name)); err != nil {
			return err
		}
	}
	return nil
}

// ifaceMethodNames collects an interface's method names (including recursive `expand`).
func (c *checker) ifaceMethodNames(name string, seen map[string]bool) []string {
	if seen[name] {
		return nil
	}
	seen[name] = true
	def, ok := c.interfaces[name]
	if !ok {
		return nil
	}
	names := []string{}
	for _, sig := range def.Methods {
		names = append(names, sig.Name)
	}
	for _, ex := range def.Expands {
		names = append(names, c.ifaceMethodNames(ex, seen)...)
	}
	return names
}

// isRecvParam decides whether an impl method's first parameter is a **receiver**: by type (Self or the impl's type),
// regardless of the parameter name; an untyped first parameter counts as the receiver and is completed with the impl's type.
func isRecvParam(implType string, p *Param) bool {
	if p == nil {
		return false
	}
	t := strings.TrimSpace(p.Type)
	if t == "" {
		p.Type = implType
		return true
	}
	return t == "Self" || recvBaseName(t) == recvBaseName(implType)
}

// recvBaseName takes the base type name: strips a trailing & and generic arguments (node<T>& → node).
func recvBaseName(t string) string {
	t = strings.TrimSpace(t)
	t = strings.TrimSuffix(t, "&")
	if i := strings.IndexByte(t, '<'); i >= 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

// isSelfType reports whether this is the Self placeholder (Self = the implementing type, xmind §interfaces).
func isSelfType(s string) bool { return strings.TrimSpace(s) == "Self" }

// sigCompatible reports whether from's method signature satisfies to: **strict by type** (same name + parameter count + identical parameter/return types).
// Self is the implementing-type placeholder: Self ↔ Self is equivalent, and Self ↔ a concrete type also holds when the other side is bound.
func sigCompatible(fromSig, toSig MethodSig) bool {
	if len(fromSig.Params) != len(toSig.Params) {
		return false
	}
	retA, retB := strings.TrimSpace(fromSig.Ret), strings.TrimSpace(toSig.Ret)
	if retA != retB && !isSelfType(retA) && !isSelfType(retB) {
		return false
	}
	for i := range toSig.Params {
		a := strings.TrimSpace(fromSig.Params[i].Type)
		b := strings.TrimSpace(toSig.Params[i].Type)
		if a == b || isSelfType(a) || isSelfType(b) {
			continue
		}
		return false
	}
	return true
}

// methodSigOf looks up a method signature in an interface (including `expand`); ok=false means the interface has no such method.
func (c *checker) methodSigOf(iface, name string) (MethodSig, bool) {
	def, ok := c.interfaces[iface]
	if !ok {
		return MethodSig{}, false
	}
	for _, sig := range def.Methods {
		if sig.Name == name {
			return sig, true
		}
	}
	for _, ex := range def.Expands {
		if s, ok := c.methodSigOf(ex, name); ok {
			return s, true
		}
	}
	return MethodSig{}, false
}

// checkIfaceCovers verifies that fromIface satisfies toIface: **strict by type** structural satisfaction
// (method set coverage + identical signatures; interface names are not compared).
func (c *checker) checkIfaceCovers(fromIface, toIface string) error {
	if fromIface == toIface {
		return nil
	}
	for _, n := range c.ifaceMethodNames(toIface, map[string]bool{}) {
		fsig, ok := c.methodSigOf(fromIface, n)
		if !ok {
			return errors.New(msg("接口 %s 缺少 %s 的方法 %q", fromIface, toIface, n))
		}
		tsig, _ := c.methodSigOf(toIface, n)
		if !sigCompatible(fsig, tsig) {
			return errors.New(msg("接口 %s 的方法 %q 与 %s 的签名不兼容（按类型严格）", fromIface, n, toIface))
		}
	}
	return nil
}

// checkIfaceStrict: **strict implementation** checking for interfaces (interface names do not participate):
//
//	struct → interface: the type must implement every method the interface requires (aggregated over impls);
//	interface → interface: the source interface's method set must cover the target's with identical signatures.
func (c *checker) checkIfaceStrict(from, to *Type, pos Pos, what string) error {
	if from == nil || to == nil || to.Kind != tInterface {
		return nil
	}
	switch from.Kind {
	case tInterface:
		if err := c.checkIfaceCovers(from.FName, to.FName); err != nil {
			return c.errf(pos, "TypeError: %s: %v", what, err)
		}
	case tStruct:
		if err := c.checkImplements(from.FName, to.FName); err != nil {
			return c.errf(pos, "TypeError: %s: %v", what, err)
		}
	}
	return nil
}

// checkImplements verifies that typ implements interface iface (method-name set coverage, aggregated over impls).
func (c *checker) checkImplements(typ, iface string) error {
	def, ok := c.interfaces[iface]
	if !ok {
		return errors.New(msg("未知接口 %s", iface))
	}
	if def.Partial {
		return nil // optional implementation: no full coverage required
	}
	methods := append([]MethodSig{}, def.Methods...)
	for _, ex := range def.Expands {
		if ei, ok := c.interfaces[ex]; ok {
			methods = append(methods, ei.Methods...)
		}
	}
	for _, sig := range methods {
		if c.selfMeth(typ, sig.Name) == nil {
			return errors.New(msg("类型 %s 未实现接口方法 %q", typ, sig.Name))
		}
	}
	return nil
}

func (c *checker) inferScope(x *ScopeCall, sc *cScope) (*Type, error) {
	args, err := c.inferArgs(x.Args, sc)
	if err != nil {
		return nil, err
	}
	switch x.Scope {
	case "memorize":
		if x.Name == "new" && len(args) == 0 {
			return tMemorizeV, nil
		}
		return nil, c.errf(x.Pos, "TypeError: memorize has no static method %q", x.Name)
	case "HashTable":
		if x.Name == "new" && len(args) == 0 {
			return mkTable(tAnyV, tAnyV), nil
		}
		return nil, c.errf(x.Pos, "TypeError: HashTable has no static method %q", x.Name)
	case "List":
		if x.Name == "new" && len(args) == 0 {
			return mkList(tAnyV), nil
		}
		return nil, c.errf(x.Pos, "TypeError: List has no static method %q", x.Name)
	case "IO":
		switch x.Name {
		case "setIn":
			if err := c.checkArity("IO::setIn", 2, len(args), x.Pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tIOStream {
				return nil, c.errf(x.Pos, "TypeError: IO::setIn's first arg must be an IOStream, got %s", args[0])
			}
			if args[1].Kind != tString && args[1].Kind != tInputStream {
				return nil, c.errf(x.Pos, "TypeError: IO::setIn requires a path String or an InputStream, got %s", args[1])
			}
			return tNilV, nil
		case "setOut":
			if err := c.checkArity("IO::setOut", 2, len(args), x.Pos); err != nil {
				return nil, err
			}
			if args[0].Kind != tIOStream {
				return nil, c.errf(x.Pos, "TypeError: IO::setOut's first arg must be an IOStream, got %s", args[0])
			}
			if args[1].Kind != tString && args[1].Kind != tOutputStream {
				return nil, c.errf(x.Pos, "TypeError: IO::setOut requires a path String or an OutputStream, got %s", args[1])
			}
			return tNilV, nil
		}
		return nil, c.errf(x.Pos, "TypeError: IO has no static method %q", x.Name)
	case "taskm":
		// taskm is a global variable: the correct syntax is taskm.spawn(...) etc.
		return nil, c.errf(x.Pos, "TypeError: taskm is a global variable — use taskm.spawn(...) / taskm.block(pid) / taskm.done(pid) / taskm.merge(pid) / taskm.channel([n])")
	}
	if x.Scope == "file" && x.Name == "new" {
		if err := c.checkArity("file::new", 1, len(x.Args), x.Pos); err != nil {
			return nil, err
		}
		return tFileV, nil
	}
	// Generic static method: type parameters are leniently substituted with interface{} (aggregating impls whose methods do not overlap)
	if defs := c.implDefsFor(x.Scope); len(defs) > 0 {
		prev := c.curSubst
		subst := map[string]*Type{}
		for _, def := range defs {
			for _, tp := range def.TypeParams {
				subst[tp] = tAnyV
			}
		}
		c.curSubst = subst
		defer func() { c.curSubst = prev }()
		for _, def := range defs {
			if fn, ok := def.Methods[x.Name]; ok {
				if len(fn.Params) != len(args) {
					return nil, c.errf(x.Pos, "CompileError: %s::%s expects %d args, got %d", x.Scope, x.Name, len(fn.Params), len(args))
				}
				for i, p := range fn.Params {
					pt, err := c.paramType(p, x.Pos)
					if err != nil {
						return nil, err
					}
					if !assignable(args[i], pt) {
						return nil, c.errf(x.Pos, "TypeError: argument %d of %s::%s: cannot assign %s to %s", i+1, x.Scope, x.Name, args[i], pt)
					}
				}
				if fn.Ret != "" {
					return c.substType(fn.Ret, c.curSubst, x.Pos)
				}
				return tFuncBufferV, nil
			}
		}
		return nil, c.errf(x.Pos, "TypeError: %s has no static method %q", x.Scope, x.Name)
	}
	return nil, c.errf(x.Pos, "CompileError: unknown scope %q", x.Scope)
}
