package cgen

// 接口 / dynamic 分发（Phase C）：vtable + thunk 方案。
//
// 值表示：%Iface = type { i8* data, i8** vt }
//   - data：具体 struct 实例指针（与解释器的 *StructValue 引用语义一致）
//   - vt：具体类型对该接口的 vtable（每个 (类型, 接口) 对生成一份常量表）
//
// 装箱发生在「具体 struct 值 → 接口槽位」的赋值/传参/返回处（此时具体类型已知）；
// 调用 iface.m(args) 时从 vtable 第 idx 槽取函数指针，经 thunk 还原具体类型后调用。
// 结构化满足（S14）由 internal/lang 的 Typecheck 在赋值/传参处校验，编译器只按
// 已通过的检查生成派发表；缺方法时给出明确诊断（绝不静默错编）。

import (
	"strings"

	"quarklang/internal/lang"
)

// ifaceM 是接口方法在派发表中的槽位。
type ifaceM struct {
	name   string
	idx    int
	params []string // 不含 self；"Self" 保留原样
	ret    string
}

// ifaceMethodList 返回接口方法表（含 expand interface 组合，按声明顺序，缓存）。
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
					continue // 接收者：按类型判定（Self / 接口类型基名），与形参名无关
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

// vtableFor 构造/复用 (typ, iface) 的 vtable（含各方法 thunk）。
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
		// 接口声明为 Self 的参数：thunk 接收 i8*（接口 data），调用前还原具体类型
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

// boxTo 把具体值装箱成接口值：
//   - want == interface{}（tAny）→ 装箱任意可 lower 值（标量/String 堆单元 + RTTI 描述符）；
//   - want 是具名接口 → vtable 方案（需要 struct 实现）。
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

// ifaceCall lower 接口方法调用（vtable 分发）。
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
			// Self 形参：装箱成接口值，调用点取 data（i8*）
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
