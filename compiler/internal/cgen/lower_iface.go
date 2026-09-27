package cgen

// Interface / dynamic dispatch (Phase C): vtable + thunk scheme.
//
// Value representation: %Iface = type { i8* data, i8** vt }
//   - data: pointer to a concrete struct instance (same reference semantics as the interpreter's *StructValue)
//   - vt: the concrete type's vtable for this interface (one constant table per (type, interface) pair)
//
// Boxing happens at the assignment/argument passing/return of "concrete struct value → interface slot" (the concrete type is known there);
// calling iface.m(args) fetches the function pointer from vtable slot idx and calls it after the thunk restores the concrete type.
// Structural satisfaction (S14) is checked by internal/lang's Typecheck at assignment/argument passing; the compiler only
// generates dispatch tables for checks already passed; a missing method yields an explicit diagnostic (never a silent miscompile).

import (
	"strings"

	"quarklang/internal/lang"
)

// ifaceM is an interface method's slot in the dispatch table.
type ifaceM struct {
	name   string
	idx    int
	params []string // excludes self; "Self" kept as-is
	ret    string
}

// ifaceMethodList returns the interface method table (including expand interface composition, in declaration order, cached).
func (l *lowerer) ifaceMethodList(name string) []ifaceM {
	if t, ok := l.ifaceTbl[name]; ok {
		return t
	}
	var out []ifaceM
	seen := map[string]bool{}
	var walk func(n string)
	walk = func(n string) {
		id, ok := l.ifaces[n]
		if !ok || seen[n] {
			return
		}
		seen[n] = true
		for _, ex := range id.Expands {
			walk(ex)
		}
		for _, m := range id.Methods {
			var params []string
			for i := range m.Params {
				if i == 0 && isRecvParam(n, &m.Params[0]) {
					continue // receiver: determined by type (Self / interface type base name), independent of the parameter name
				}
				params = append(params, m.Params[i].Type)
			}
			out = append(out, ifaceM{name: m.Name, idx: len(out), params: params, ret: m.Ret})
		}
	}
	walk(name)
	if l.ifaceTbl == nil {
		l.ifaceTbl = map[string][]ifaceM{}
	}
	l.ifaceTbl[name] = out
	return out
}

// vtableFor builds/reuses the vtable for (typ, iface) (with a thunk per method).
func (l *lowerer) vtableFor(typ, iface string, fc *funcCtx, pos lang.Pos) (*vtableDef, error) {
	sym := "@vt$" + tyName(typ) + "$" + tyName(iface)
	if v, ok := l.vtables[sym]; ok {
		return v, nil
	}
	v := &vtableDef{sym: sym, iface: iface, typ: typ}
	l.vtables[sym] = v
	for _, m := range l.ifaceMethodList(iface) {
		mi := l.lookupSelf(typ, m.name)
		if mi == nil {
			return nil, l.errf(pos, "接口 %s 的方法 %s 在 %s 上未实现（类型检查本应拦截）", iface, m.name, typ)
		}
		imi, err := l.instantiateFor(mi, typ, nil, fc, pos)
		if err != nil {
			return nil, err
		}
		th := &ifaceThunk{iface: iface, typ: typ, method: m.name, irName: imi.irName, vtSym: sym}
		th.ret = substType(imi.fn.Ret, imi.subst)
		if th.ret == "Self" || th.ret == "" {
			if th.ret == "" {
				th.ret = "void"
			} else {
				th.ret = typ
			}
		}
		for _, p := range imi.fn.Params[1:] {
			th.params = append(th.params, substType(p.Type, imi.subst))
		}
		if len(th.params) != len(m.params) {
			return nil, l.errf(pos, "接口 %s 的方法 %s 与 %s 的实现参数个数不一致（类型检查本应拦截）", iface, m.name, typ)
		}
		// Interface parameters declared as Self: the thunk takes an i8* (interface data) and restores the concrete type before the call
		for i, pt := range m.params {
			if strings.TrimSpace(pt) == "Self" || pt == iface {
				th.selfIdx = append(th.selfIdx, i)
			}
		}
		th.boxRet = strings.TrimSpace(m.ret) == "Self"
		v.thunks = append(v.thunks, th)
	}
	l.ensureStructTy(typ)
	return v, nil
}

// boxTo boxes a concrete value into an interface value:
//   - want == interface{} (tAny) → box any lowerable value (scalar/String heap cell + RTTI descriptor);
//   - want is a named interface → vtable scheme (requires a struct impl).
func (fc *funcCtx) boxTo(e *expr, want string, pos lang.Pos) (*expr, error) {
	if e == nil || e.typ == want {
		return e, nil
	}
	if isAnyT(want) {
		switch e.typ {
		case "int", "float", "bool", "String", "interface{}", "null":
		default:
			if !fc.l.isStructType(e.typ) && e.typ != "List<int>" {
				return nil, fc.l.errf(pos, "暂未支持把 %s 值装箱成 interface{}（编译器支持 int/float/bool/String/List<int>/struct）", e.typ)
			}
		}
		e.anyBox = e.typ
		e.typ = want
		return e, nil
	}
	if !fc.l.isIfaceType(want) {
		return e, nil
	}
	if !fc.l.isStructType(e.typ) {
		return nil, fc.l.errf(pos, "暂未支持把 %s 值装入接口 %s（编译器需要 struct 实现）", e.typ, want)
	}
	vt, err := fc.l.vtableFor(e.typ, want, fc, pos)
	if err != nil {
		return nil, err
	}
	e.ifaceBox = vt.sym
	e.typ = want
	return e, nil
}

// ifaceCall lowers an interface method call (vtable dispatch).
func (fc *funcCtx) ifaceCall(c *lang.CallExpr, me *lang.MemberExpr, rt string) (*expr, error) {
	l := fc.l
	tbl := l.ifaceMethodList(rt)
	var m *ifaceM
	for i := range tbl {
		if tbl[i].name == me.Name {
			m = &tbl[i]
			break
		}
	}
	if m == nil {
		return nil, l.errf(me.Pos, "接口 %s 没有方法 %q", rt, me.Name)
	}
	if len(c.Args) != len(m.params) {
		return nil, l.errf(me.Pos, "接口方法 %s.%s 需要 %d 个参数，got %d", rt, me.Name, len(m.params), len(c.Args))
	}
	recv, err := fc.expr(me.X)
	if err != nil {
		return nil, err
	}
	sig := &funcSig{name: rt + "." + me.Name, ret: m.ret}
	if sig.ret == "Self" {
		sig.ret = rt
	}
	if sig.ret == "" {
		sig.ret = "void"
	}
	args := make([]*expr, 0, len(c.Args))
	for i, a := range c.Args {
		want := m.params[i]
		at := fc.typeOf(a)
		if strings.TrimSpace(want) == "Self" || want == rt {
			// Self parameter: boxed into an interface value; the call site takes data (i8*)
			x, err := fc.exprAs(a, rt)
			if err != nil {
				return nil, err
			}
			sig.params = append(sig.params, funcParam{typ: "Self"})
			args = append(args, x)
			continue
		}
		if !fc.assignable(at, want) {
			return nil, l.errf(exprPos(a, me.Pos), "暂未支持向接口方法 %s.%s 传递 %s 参数（需要 %s）", rt, me.Name, at, want)
		}
		x, err := fc.exprAs(a, want)
		if err != nil {
			return nil, err
		}
		sig.params = append(sig.params, funcParam{typ: want})
		args = append(args, x)
	}
	return &expr{kind: kMethod, typ: sig.ret, line: me.Pos.Line, method: &methodExpr{
		recv: recv, name: me.Name, args: args, iface: rt, idx: m.idx, ifaceSig: sig,
	}}, nil
}
