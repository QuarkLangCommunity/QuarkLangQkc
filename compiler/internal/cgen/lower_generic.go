package cgen

// Generic monomorphization (Phase C): collect instances per call site/type annotation, generate one
// monomorphic function (fn<T> → id$int) and monomorphic struct (Box<int> → %Box_int_) per concrete type combination.
//
// Interpreter consistency: the interpreter runs generic bodies with dynamic values at runtime; the compiler expands
// by type arguments at compile time with identical semantics (typecheck is done by internal/lang; here we only instantiate and mangle names).

import (
	"strings"

	"quarklang/internal/lang"
)

// typeArgsKey builds the type-argument key ("int" / "Box<int>" → "int" / "Box_int_").
func typeArgsKey(tps []string, sub map[string]string) string {
	parts := make([]string, 0, len(tps))
	for _, tp := range tps {
		parts = append(parts, tyName(sub[tp]))
	}
	return strings.Join(parts, "$")
}

// instName builds the IR name of an instance: id + $int → id$int; Box + $int + _get → Box$int_get.
func instName(base string, tps []string, sub map[string]string) string {
	if len(tps) == 0 {
		return base
	}
	return base + "$" + typeArgsKey(tps, sub)
}

// instantiateFunc instantiates a generic function (idempotent; registers before lowering to avoid infinite recursion).
func (l *lowerer) instantiateFunc(name string, fn *lang.FuncDecl, sub map[string]string) (string, error) {
	key := "fn:" + name + "$" + typeArgsKey(fn.TypeParams, sub)
	if ir, ok := l.insts[key]; ok {
		return ir, nil
	}
	ir := instName(name, fn.TypeParams, sub)
	l.insts[key] = ir
	fd, err := l.lowerFunc(fn, ir, "", "", sub)
	if err != nil {
		return "", err
	}
	l.out.funcs = append(l.out.funcs, fd)
	return ir, nil
}

// callGeneric instantiates and calls a generic function fn<T,...> (infers type parameters from arguments).
func (fc *funcCtx) callGeneric(name string, c *lang.CallExpr, pos lang.Pos) (*expr, error) {
	l := fc.l
	fn := l.generics[name]
	if len(c.Args) != len(fn.Params) {
		return nil, l.errf(pos, "函数 %s 需要 %d 个参数，got %d", name, len(fn.Params), len(c.Args))
	}
	sub := inferGenSubst(fn, c.Args, fc)
	for _, tp := range fn.TypeParams {
		if sub[tp] == "" {
			return nil, l.errf(pos, "暂未支持无法从实参推断类型参数的泛型调用 %s（解释器可用）", name)
		}
	}
	ir, err := l.instantiateFunc(name, fn, sub)
	if err != nil {
		return nil, err
	}
	if l.nilFns[ir] && fc.discarded != c {
		return nil, l.errf(pos, "暂未支持在表达式中使用含 log 的函数 %s 的返回值（解释器返回 nil）", name)
	}
	params := make([]lang.Param, len(fn.Params))
	for i, p := range fn.Params {
		params[i] = lang.Param{Name: p.Name, Type: substType(p.Type, sub), Decor: p.Decor, Pos: p.Pos}
	}
	args, err := fc.callArgs(c, params)
	if err != nil {
		return nil, err
	}
	ret := substType(fn.Ret, sub)
	if ret == "" {
		ret = "void"
	}
	return &expr{kind: kCall, typ: ret, line: pos.Line, call: &callExpr{name: ir, args: args}}, nil
}

// recvSubst derives the impl type-parameter substitution from the receiver type (Box<int> + impl<T> → {T:int}).
func (l *lowerer) recvSubst(mi *methodInfo, recvType string) map[string]string {
	sub := map[string]string{}
	if len(mi.impl.TypeParams) == 0 {
		return sub
	}
	_, targs := splitGeneric(recvType)
	for i, tp := range mi.impl.TypeParams {
		if i < len(targs) {
			sub[tp] = targs[i]
		}
	}
	return sub
}

// instSelfType builds the receiver type from the impl base name + type arguments (Box + {T:int} → Box<int>).
func (l *lowerer) instSelfType(base string, sub map[string]string) string {
	sd, ok := l.structs[base]
	if !ok || len(sd.TypeParams) == 0 {
		return base
	}
	args := make([]string, 0, len(sd.TypeParams))
	for _, tp := range sd.TypeParams {
		args = append(args, sub[tp])
	}
	return base + "<" + strings.Join(args, ",") + ">"
}

// instantiateFor monomorphizes impl methods by type arguments. Instance methods take arguments from the receiver type,
// static methods infer them from call arguments; idempotent (results cached, registers before lowering to prevent recursion).
func (l *lowerer) instantiateFor(mi *methodInfo, recvType string, args []lang.Expr, fc *funcCtx, pos lang.Pos) (*methodInfo, error) {
	if len(mi.impl.TypeParams) == 0 {
		return mi, nil
	}
	var sub map[string]string
	if mi.isSelf {
		sub = l.recvSubst(mi, recvType)
	} else {
		sub = map[string]string{}
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
	}
	for _, tp := range mi.impl.TypeParams {
		if sub[tp] == "" {
			return nil, l.errf(pos, "暂未支持无法推断类型参数的泛型方法 %s.%s（解释器可用）", mi.impl.Type, mi.fn.Name)
		}
	}
	key := "mi:" + mi.impl.Type + "_" + mi.fn.Name + "$" + typeArgsKey(mi.impl.TypeParams, sub)
	if c, ok := l.instMi[key]; ok {
		return c, nil
	}
	ir := instName(mi.impl.Type, mi.impl.TypeParams, sub) + "_" + mi.fn.Name
	selfTyp, selfName := "", ""
	if mi.isSelf {
		selfTyp = l.instSelfType(mi.impl.Type, sub)
		l.ensureStructTy(selfTyp)
		if len(mi.fn.Params) > 0 {
			selfName = mi.fn.Params[0].Name
		}
	}
	out := &methodInfo{fn: mi.fn, impl: mi.impl, selfTyp: selfTyp, isSelf: mi.isSelf, irName: ir, subst: sub}
	l.instMi[key] = out
	fd, err := l.lowerFunc(mi.fn, ir, selfTyp, selfName, sub)
	if err != nil {
		return nil, err
	}
	l.out.funcs = append(l.out.funcs, fd)
	return out, nil
}
