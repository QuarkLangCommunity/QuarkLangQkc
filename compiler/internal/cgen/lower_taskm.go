package cgen

// taskm thread / channel lowering (Phase D): hooks into the qthreads.c runtime.
//
// Aligned with interpreter semantics:
//   - taskm.spawn() → thread handle (compiled path = runtime small-integer pid, an int like the language pid)
//   - t.merge(fn[, arg]) / taskm.merge(pid, fn[, arg]) → run fn on the thread (return value discarded)
//   - taskm.block(pid) → wait for the thread to become idle; taskm.done(pid) → whether the thread is idle
//   - taskm.channel([n]) / c.send(v) / c.recv() → bounded channel (default capacity 1024)
//
// Limits (explicit errors, no silent miscompiles): a merge target function takes at most 1 int parameter (the runtime arg is an int);
// thread.talk is not lowered.

import (
	"strings"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// taskPid takes the thread handle (a thread variable or an int pid; both are i32 on the compiled path).
func (fc *funcCtx) taskPid(x lang.Expr) (*expr, error) {
	t := fc.typeOf(x)
	if t != "int" && t != "thread" && t != "?" {
		return nil, fc.l.errfID(exprPos(x, lang.Pos{Line: 1, Col: 1}), i18n.MsgCompilerTaskmRequiresThreadOrPidGot, t)
	}
	return fc.expr(x)
}

// runnerFor registers a merge runner (the target function takes at most 1 int parameter).
func (fc *funcCtx) runnerFor(fnExpr lang.Expr, extra []lang.Expr, pos lang.Pos) (*expr, error) {
	l := fc.l
	id, ok := fnExpr.(*lang.Ident)
	if !ok {
		return nil, l.errfID(pos, i18n.MsgCompilerThisMergeTargetNotSupportedYet)
	}
	if _, isGen := l.generics[id.Name]; isGen {
		return nil, l.errfID(id.Pos, i18n.MsgCompilerMergingGenericFunctionSNotSupported, id.Name)
	}
	fn, irName, err := l.resolveRunnerTarget(id.Name, len(extra), id.Pos)
	if err != nil {
		return nil, err
	}
	if len(fn.Params) != len(extra) {
		return nil, l.errfID(pos, i18n.MsgCompilerFunctionSRequiresDArgumentsGot, id.Name, len(fn.Params), len(extra))
	}
	if len(fn.Params) > 8 {
		return nil, l.errfID(pos, i18n.MsgCompilerMergingWithDArgumentsNotSupported, len(extra))
	}
	var ptypes []string
	for _, p := range fn.Params {
		pt := strings.TrimSpace(p.Type)
		switch pt {
		case "int", "bool", "float", "long", "String", "pointer", "thread", "Channel", "channel":
		default:
			if l.isStructType(pt) || l.isListTypeE(pt) {
				l.ensureStructTy(pt)
			} else {
				return nil, l.errfID(p.Pos, i18n.MsgCompilerASMergeParameterNotSupported, pt)
			}
		}
		ptypes = append(ptypes, pt)
	}
	if _, isNil := l.nilFns[irName]; isNil {
		return nil, l.errfID(id.Pos, i18n.MsgCompilerMergingFunctionSContainsLogNot, id.Name)
	}
	r := l.addRunner(irName, ptypes)
	return &expr{kind: kRunner, typ: "runner", s: r, i: int64(len(ptypes))}, nil
}

// isListTypeE reports whether a language type is List<...>.
func (l *lowerer) isListTypeE(t string) bool { return strings.HasPrefix(t, "List<") }

// addRunner registers a runner (deduplicated) and returns its IR name.
func (l *lowerer) addRunner(fn string, params []string) string {
	key := fn + "|" + strings.Join(params, ",")
	if n, ok := l.runnerIdx[key]; ok {
		return n
	}
	n := "runner_" + itoa(len(l.out.runners))
	l.runnerIdx[key] = n
	l.out.runners = append(l.out.runners, runnerDef{fn: fn, params: params})
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// taskmCall lowers taskm.<method>(...).
func (fc *funcCtx) taskmCall(c *lang.CallExpr, me *lang.MemberExpr) (*expr, error) {
	l := fc.l
	switch me.Name {
	case "spawn":
		if len(c.Args) != 0 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerTaskmSpawnTakesNoArgumentsGot, len(c.Args))
		}
		return &expr{kind: kCall, typ: "thread", call: &callExpr{name: "ql_spawn"}}, nil
	case "channel":
		if len(c.Args) > 1 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerTaskmChannelTakes0Or1, len(c.Args))
		}
		arg := &expr{kind: kInt, typ: "int", i: 1024}
		if len(c.Args) == 1 {
			if t := fc.typeOf(c.Args[0]); t != "int" && t != "?" {
				return nil, l.errfID(exprPos(c.Args[0], me.Pos), i18n.MsgCompilerTaskmChannelNRequiresIntCapacity, t)
			}
			x, err := fc.expr(c.Args[0])
			if err != nil {
				return nil, err
			}
			arg = x
		}
		return &expr{kind: kCall, typ: "Channel", call: &callExpr{name: "ql_channel_new", args: []*expr{arg}}}, nil
	case "block":
		if len(c.Args) != 1 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerTaskmBlockPidRequires1Argument, len(c.Args))
		}
		pid, err := fc.taskPid(c.Args[0])
		if err != nil {
			return nil, err
		}
		return &expr{kind: kCall, typ: "void", call: &callExpr{name: "ql_block", args: []*expr{pid}}}, nil
	case "done":
		if len(c.Args) != 1 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerTaskmDonePidRequires1Argument, len(c.Args))
		}
		pid, err := fc.taskPid(c.Args[0])
		if err != nil {
			return nil, err
		}
		return &expr{kind: kCall, typ: "bool", call: &callExpr{name: "ql_done", args: []*expr{pid}}}, nil
	case "merge":
		// taskm.merge(pid, fn[, args...])
		if len(c.Args) < 2 || len(c.Args) > 10 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerTaskmMergePidFnArgsRequires, len(c.Args))
		}
		pid, err := fc.taskPid(c.Args[0])
		if err != nil {
			return nil, err
		}
		return fc.mergeCall(me, pid, c.Args[1], c.Args[2:])
	case "talk":
		return nil, l.errfID(me.Pos, i18n.MsgCompilerThreadTalkNotSupportedYetAvailable)
	}
	return nil, l.errfID(me.Pos, i18n.MsgCompilerTaskmSNotSupportedYetCompiler, me.Name)
}

// threadCall lowers t.merge(fn[, arg]) / t.pid().
func (fc *funcCtx) threadCall(c *lang.CallExpr, me *lang.MemberExpr, recv *expr) (*expr, error) {
	l := fc.l
	switch me.Name {
	case "pid":
		if len(c.Args) != 0 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerTPidTakesNoArguments)
		}
		return recv, nil // on the compiled path a thread variable is itself the pid (i32)
	case "merge":
		if len(c.Args) < 1 || len(c.Args) > 9 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerTMergeFnArgsRequires1, len(c.Args))
		}
		return fc.mergeCall(me, recv, c.Args[0], c.Args[1:])
	case "talk":
		// Interpreter semantics: verify the argument is a channel, then do nothing
		if len(c.Args) != 1 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerTTalkCRequires1Argument)
		}
		if t := fc.typeOf(c.Args[0]); t != "Channel" && t != "channel" && t != "?" {
			return nil, l.errfID(exprPos(c.Args[0], me.Pos), i18n.MsgCompilerThreadTalkRequiresChannelInstanceGot, t)
		}
		if _, err := fc.expr(c.Args[0]); err != nil {
			return nil, err
		}
		return &expr{kind: kInt, typ: "void"}, nil
	}
	return nil, l.errfID(me.Pos, i18n.MsgCompilerThreadMethodQNotSupportedYet, me.Name)
}

// mergeCall builds taskm.merge(pid, runner, args...) (arguments are passed to the runtime through i64 slots).
func (fc *funcCtx) mergeCall(me *lang.MemberExpr, pid *expr, fnExpr lang.Expr, extra []lang.Expr) (*expr, error) {
	runner, err := fc.runnerFor(fnExpr, extra, me.Pos)
	if err != nil {
		return nil, err
	}
	args := []*expr{pid, runner}
	for _, a := range extra {
		t := fc.typeOf(a)
		if t != "?" && !fc.mergeArgOK(t) {
			return nil, fc.l.errfID(exprPos(a, me.Pos), i18n.MsgCompilerASMergeArgumentNotSupported, t)
		}
		x, err := fc.expr(a)
		if err != nil {
			return nil, err
		}
		args = append(args, x)
	}
	return &expr{kind: kMerge, typ: "void", call: &callExpr{name: "ql_merge", args: args}}, nil
}

// mergeArgOK reports whether an argument type can be carried in an i64 slot.
func (fc *funcCtx) mergeArgOK(t string) bool {
	switch t {
	case "int", "bool", "float", "long", "String", "pointer", "thread", "Channel", "channel":
		return true
	}
	return fc.l.isStructType(t) || fc.l.isListTypeE(t)
}

// channelCall lowers c.send(v) / c.recv().
func (fc *funcCtx) channelCall(c *lang.CallExpr, me *lang.MemberExpr, recv *expr) (*expr, error) {
	l := fc.l
	switch me.Name {
	case "send":
		if len(c.Args) != 1 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerCSendVRequires1Argument)
		}
		if t := fc.typeOf(c.Args[0]); t != "int" && t != "?" {
			return nil, l.errfID(exprPos(c.Args[0], me.Pos), i18n.MsgCompilerSendingSOverChannelNotSupported, t)
		}
		x, err := fc.expr(c.Args[0])
		if err != nil {
			return nil, err
		}
		return &expr{kind: kCall, typ: "void", call: &callExpr{name: "ql_send", args: []*expr{recv, x}}}, nil
	case "recv":
		if len(c.Args) != 0 {
			return nil, l.errfID(me.Pos, i18n.MsgCompilerCRecvTakesNoArguments)
		}
		return &expr{kind: kCall, typ: "int", call: &callExpr{name: "ql_recv", args: []*expr{recv}}}, nil
	}
	return nil, l.errfID(me.Pos, i18n.MsgCompilerChannelMethodQNotSupportedYet, me.Name)
}
