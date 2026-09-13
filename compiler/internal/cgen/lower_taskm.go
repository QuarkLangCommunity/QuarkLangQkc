package cgen

// taskm 线程 / 通道 lowering（阶段 D）：对接 qthreads.c 运行时。
//
// 与解释器语义对齐：
//   - taskm.spawn() → 线程句柄（编译路径 = 运行时小整数 pid，与语言 pid 同为 int）
//   - t.merge(fn[, arg]) / taskm.merge(pid, fn[, arg]) → 在线程上执行 fn（返回值丢弃）
//   - taskm.block(pid) → 等待线程空闲；taskm.done(pid) → 线程是否空闲
//   - taskm.channel([n]) / c.send(v) / c.recv() → 有界通道（容量默认 1024）
//
// 限制（明确报错，不静默错编）：merge 的目标函数最多 1 个 int 形参（运行时 arg 为 int）；
// thread.talk 未 lower。

import (
	"quarklang/internal/lang"
)

// taskPid 取线程句柄（thread 变量或 int pid，编译路径同为 i32）。
func (fc *funcCtx) taskPid(x lang.Expr) (*expr, error) {
	t := fc.typeOf(x)
	if t != "int" && t != "thread" && t != "?" {
		return nil, fc.l.errf(exprPos(x, lang.Pos{Line: 1, Col: 1}), "taskm 需要 thread 或 pid，got %s", t)
	}
	return fc.expr(x)
}

// runnerFor 登记 merge 的 runner（目标函数最多 1 个 int 形参）。
func (fc *funcCtx) runnerFor(fnExpr lang.Expr, extra []lang.Expr, pos lang.Pos) (*expr, error) {
	l := fc.l
	id, ok := fnExpr.(*lang.Ident)
	if !ok {
		return nil, l.errf(pos, "暂未支持该 merge 目标（编译器要求具名函数）")
	}
	if _, isGen := l.generics[id.Name]; isGen {
		return nil, l.errf(id.Pos, "暂未支持 merge 泛型函数 %s（解释器可用）", id.Name)
	}
	fn, ok := l.fns[id.Name]
	if !ok {
		return nil, l.errf(id.Pos, "未知函数 %q（编译器只支持同一程序内定义的函数）", id.Name)
	}
	if len(fn.Params) != len(extra) {
		return nil, l.errf(pos, "函数 %s 需要 %d 个参数，got %d", id.Name, len(fn.Params), len(extra))
	}
	if len(fn.Params) > 1 {
		return nil, l.errf(pos, "暂未支持 merge 传 %d 个参数（运行时线程只携带 1 个 int）", len(extra))
	}
	if len(fn.Params) == 1 && fn.Params[0].Type != "int" {
		return nil, l.errf(fn.Params[0].Pos, "暂未支持 merge 的 %s 参数（运行时线程只携带 int）", fn.Params[0].Type)
	}
	if _, isNil := l.nilFns[id.Name]; isNil {
		return nil, l.errf(id.Pos, "暂未支持 merge 含 log 的函数 %s（返回值可能为 nil）", id.Name)
	}
	r := l.addRunner(id.Name, len(fn.Params) == 1)
	return &expr{kind: kRunner, typ: "runner", s: r}, nil
}

// addRunner 登记 runner（去重），返回 IR 名。
func (l *lowerer) addRunner(fn string, hasArg bool) string {
	key := fn
	if hasArg {
		key += "|1"
	}
	if n, ok := l.runnerIdx[key]; ok {
		return n
	}
	n := "runner_" + itoa(len(l.out.runners))
	l.runnerIdx[key] = n
	l.out.runners = append(l.out.runners, runnerDef{fn: fn, hasArg: hasArg})
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

// taskmCall lower taskm.<method>(...)。
func (fc *funcCtx) taskmCall(c *lang.CallExpr, me *lang.MemberExpr) (*expr, error) {
	l := fc.l
	switch me.Name {
	case "spawn":
		if len(c.Args) != 0 {
			return nil, l.errf(me.Pos, "taskm.spawn() 不接受参数，got %d", len(c.Args))
		}
		return &expr{kind: kCall, typ: "thread", call: &callExpr{name: "ql_spawn"}}, nil
	case "channel":
		if len(c.Args) > 1 {
			return nil, l.errf(me.Pos, "taskm.channel() 需要 0 或 1 个参数，got %d", len(c.Args))
		}
		arg := &expr{kind: kInt, typ: "int", i: 1024}
		if len(c.Args) == 1 {
			if t := fc.typeOf(c.Args[0]); t != "int" && t != "?" {
				return nil, l.errf(exprPos(c.Args[0], me.Pos), "taskm.channel(n) 需要 int 容量，got %s", t)
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
			return nil, l.errf(me.Pos, "taskm.block(pid) 需要 1 个参数，got %d", len(c.Args))
		}
		pid, err := fc.taskPid(c.Args[0])
		if err != nil {
			return nil, err
		}
		return &expr{kind: kCall, typ: "void", call: &callExpr{name: "ql_block", args: []*expr{pid}}}, nil
	case "done":
		if len(c.Args) != 1 {
			return nil, l.errf(me.Pos, "taskm.done(pid) 需要 1 个参数，got %d", len(c.Args))
		}
		pid, err := fc.taskPid(c.Args[0])
		if err != nil {
			return nil, err
		}
		return &expr{kind: kCall, typ: "bool", call: &callExpr{name: "ql_done", args: []*expr{pid}}}, nil
	case "merge":
		// taskm.merge(pid, fn[, arg])
		if len(c.Args) < 2 || len(c.Args) > 3 {
			return nil, l.errf(me.Pos, "taskm.merge(pid, fn[, arg]) 需要 2 或 3 个参数，got %d", len(c.Args))
		}
		pid, err := fc.taskPid(c.Args[0])
		if err != nil {
			return nil, err
		}
		return fc.mergeCall(me, pid, c.Args[1], c.Args[2:])
	case "talk":
		return nil, l.errf(me.Pos, "暂未支持 thread.talk（解释器可用）")
	}
	return nil, l.errf(me.Pos, "暂未支持 taskm.%s（编译器支持 spawn/merge/block/done/channel）", me.Name)
}

// threadCall lower t.merge(fn[, arg]) / t.pid()。
func (fc *funcCtx) threadCall(c *lang.CallExpr, me *lang.MemberExpr, recv *expr) (*expr, error) {
	l := fc.l
	switch me.Name {
	case "pid":
		if len(c.Args) != 0 {
			return nil, l.errf(me.Pos, "t.pid() 不接受参数")
		}
		return recv, nil // 编译路径 thread 变量本身就是 pid（i32）
	case "merge":
		if len(c.Args) < 1 || len(c.Args) > 2 {
			return nil, l.errf(me.Pos, "t.merge(fn[, arg]) 需要 1 或 2 个参数，got %d", len(c.Args))
		}
		return fc.mergeCall(me, recv, c.Args[0], c.Args[1:])
	case "talk":
		return nil, l.errf(me.Pos, "暂未支持 thread.talk（解释器可用）")
	}
	return nil, l.errf(me.Pos, "暂未支持 thread 方法 %q（编译器支持 merge/pid）", me.Name)
}

// mergeCall 构造 ql_merge(pid, runner, arg)。
func (fc *funcCtx) mergeCall(me *lang.MemberExpr, pid *expr, fnExpr lang.Expr, extra []lang.Expr) (*expr, error) {
	runner, err := fc.runnerFor(fnExpr, extra, me.Pos)
	if err != nil {
		return nil, err
	}
	arg := &expr{kind: kInt, typ: "int"}
	if len(extra) == 1 {
		if t := fc.typeOf(extra[0]); t != "int" && t != "?" {
			return nil, fc.l.errf(exprPos(extra[0], me.Pos), "暂未支持 merge 的 %s 参数（运行时线程只携带 int）", t)
		}
		x, err := fc.expr(extra[0])
		if err != nil {
			return nil, err
		}
		arg = x
	}
	return &expr{kind: kCall, typ: "void", call: &callExpr{name: "ql_merge", args: []*expr{pid, runner, arg}}}, nil
}

// channelCall lower c.send(v) / c.recv()。
func (fc *funcCtx) channelCall(c *lang.CallExpr, me *lang.MemberExpr, recv *expr) (*expr, error) {
	l := fc.l
	switch me.Name {
	case "send":
		if len(c.Args) != 1 {
			return nil, l.errf(me.Pos, "c.send(v) 需要 1 个参数")
		}
		if t := fc.typeOf(c.Args[0]); t != "int" && t != "?" {
			return nil, l.errf(exprPos(c.Args[0], me.Pos), "暂未支持 channel 发送 %s（运行时通道只承载 int）", t)
		}
		x, err := fc.expr(c.Args[0])
		if err != nil {
			return nil, err
		}
		return &expr{kind: kCall, typ: "void", call: &callExpr{name: "ql_send", args: []*expr{recv, x}}}, nil
	case "recv":
		if len(c.Args) != 0 {
			return nil, l.errf(me.Pos, "c.recv() 不接受参数")
		}
		return &expr{kind: kCall, typ: "int", call: &callExpr{name: "ql_recv", args: []*expr{recv}}}, nil
	}
	return nil, l.errf(me.Pos, "暂未支持 channel 方法 %q（编译器支持 send/recv）", me.Name)
}
