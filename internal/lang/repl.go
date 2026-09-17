package lang

// ============ qkrepl：交互式求值会话（多行块） ============
//
// 必须在包内实现：解释器的求值设施（interp / execStmt / evalExpr / scope）都是未导出的。
//
// 设计要点：
//   - 会话持有一个 *interp（由最小 bootstrap 程序经 runWithInterp 得到），
//     全局作用域跨输入存活 → 变量、函数、类型在后续输入中可见；
//   - 每段输入若解析为**顶层声明**（fn/struct/interface/impl/space/library/type），
//     走 registerProgram 登记（与整程序同一条路径，行为一致）；
//   - 否则包成 `fn __repl_N() void { ... }` 逐条语句求值：表达式回显其值，
//     `log` 的记录回显；只走 Lex/Parse（不 Compile）→ CallExpr.FnIdx 恒为 -1，
//     运行期按**名字**派发，避免跨段函数索引失效（详见 README）。

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// REPLSession 是一次交互式求值会话。
type REPLSession struct {
	in    *interp
	out   io.Writer
	stdin io.Reader
	chunk int
}

// NewREPLSession 创建会话：bootstrap 一个空 main 取得解释器，再声明 REPL 可用的 io。
func NewREPLSession(stdin io.Reader, stdout io.Writer) (*REPLSession, error) {
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	if stdout == nil {
		stdout = io.Discard
	}
	boot, err := Compile("fn main(IOStream io) void {\n}\n")
	if err != nil {
		return nil, fmt.Errorf("REPL bootstrap: %w", err)
	}
	in, err := runWithInterp(boot, "<repl>", nil, stdin, stdout)
	if err != nil {
		return nil, fmt.Errorf("REPL bootstrap: %w", err)
	}
	s := &REPLSession{in: in, out: stdout, stdin: stdin}
	// 会话自己的 IOStream（包内可设 rd，readln 不会 nil 解引用）
	stream := &IOStream{In: stdin, Out: stdout, rd: bufio.NewReader(stdin)}
	_ = in.globalScope.declare("io", IOV(stream), Pos{})
	return s, nil
}

// Eval 求值一段输入，返回应打印的文本（求值输出 + 表达式回显）。
func (s *REPLSession) Eval(src string) (out string, err error) {
	defer func() { // 解释器不保证无 panic（FFI / 外部对象），REPL 必须活下来
		if r := recover(); r != nil {
			out, err = "", fmt.Errorf("REPL: 内部错误（已恢复）: %v", r)
		}
	}()
	if strings.TrimSpace(src) == "" {
		return "", nil
	}
	// 1) 顶层声明：登记进会话
	if prog, perr := ParseSource(src); perr == nil && hasTopDecls(prog) {
		if rerr := s.in.registerProgramReplacing(prog); rerr != nil {
			return "", rerr
		}
		return declSummary(prog), nil
	}
	// 2) 语句 / 表达式：包一层函数后逐条求值
	s.chunk++
	prog, perr := ParseSource(wrapChunk(fmt.Sprintf("__repl_%d", s.chunk), src))
	if perr != nil {
		return "", shiftErrLine(perr, -1)
	}
	fd := prog.Funcs[0]
	fn := &Func{Name: fd.Name, Params: fd.Params, Ret: fd.Ret, Body: fd.Body, Pos: fd.Pos}
	return s.execBody(fd.Body, fn)
}

// mayEndStatement 末字符是否可能结束一条语句/表达式（据此决定是否自动补 `;`）。
func mayEndStatement(c byte) bool {
	switch {
	case c >= '0' && c <= '9', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= 0x80:
		return true
	case c == '_' || c == '"' || c == '\x60' || c == ')' || c == ']':
		return true
	}
	return false
}

// wrapChunk 把 REPL 输入包成一个函数体。语言要求语句以 `;` 结束：
// 末字符是表达式结尾（标识符/字面量/`)`/`]`）时自动补一个 `;`（REPL 便利）。
func wrapChunk(name, src string) string {
	body := src
	t := strings.TrimSpace(body)
	if t != "" && mayEndStatement(t[len(t)-1]) {
		body += ";"
	}
	return fmt.Sprintf("fn %s() void {\n%s\n}\n", name, body)
}

// execBody 在会话全局作用域中逐条执行语句，返回需打印的文本。
func (s *REPLSession) execBody(body *Block, fn *Func) (string, error) {
	var out strings.Builder
	for _, st := range body.Stmts {
		s.resolveStructLit(st)
		ctx := s.in.newCtx(fn, nil, posOfStmt(st))
		ctx.depth = 0 // 池化 ctx 可能残留深度
		var err error
		if es, ok := st.(*ExprStmt); ok {
			var v Value
			v, err = s.in.evalExpr(es.X, s.in.globalScope, ctx)
			if err == nil && !v.IsNil() {
				out.WriteString(v.String())
				out.WriteByte('\n')
			}
		} else {
			err = s.in.execStmt(st, s.in.globalScope, ctx)
		}
		ctx.ensureLog()
		for i := 0; i < ctx.Log.Size(); i++ {
			if lv, e := ctx.Log.Get(i); e == nil {
				out.WriteString(lv.String())
				out.WriteByte('\n')
			}
		}
		s.in.putCtx(ctx)
		if err == errReturn || err == errLoopBreak {
			break // return / log 结束本段
		}
		if err != nil {
			return out.String(), shiftErrLine(err, -1)
		}
	}
	return out.String(), nil
}

// resolveStructLit 为 `P p = .{...};` 这类声明补上字面量的目标类型名。
// 整程序路径由 typecheck 填充（typecheck.go: StructLit.Name = 目标结构体），
// REPL 跳过类型检查，因此在这里按声明的类型标注补齐——否则得到匿名结构体，方法调用会失败。
func (s *REPLSession) resolveStructLit(st Stmt) {
	ds, ok := st.(*DeclStmt)
	if !ok {
		return
	}
	sl, ok := ds.Init.(*StructLit)
	if !ok || sl.Name != "" {
		return
	}
	base := recvBaseName(ds.Type)
	if _, isStruct := s.in.structs[base]; isStruct {
		sl.Name = base
	}
}

// Incomplete 判断输入是否「还没打完」：块未闭合、字符串未结束等。
// CLI 据此继续读下一行（多行块）。
func (s *REPLSession) Incomplete(src string) bool {
	if strings.TrimSpace(src) == "" {
		return false
	}
	// 词法层未闭合（字符串 / 原始字符串 / 块注释）
	if _, _, err := LexWithComments(src); err != nil {
		return strings.Contains(err.Error(), "unterminated")
	}
	// 作为完整程序或语句块能解析 → 完整
	if _, err := ParseSource(src); err == nil {
		return false
	}
	if _, err := ParseSource(wrapChunk("__repl_probe", src)); err == nil {
		return false
	}
	// 两种解析都失败：错误为「块未闭合」即需要继续输入
	incomplete := func(err error) bool {
		return err != nil && strings.Contains(err.Error(), "unterminated block")
	}
	if _, err := ParseSource(src); incomplete(err) {
		return true
	}
	_, err := ParseSource(wrapChunk("__repl_probe", src))
	return incomplete(err)
}

// hasTopDecls 判断程序里是否有可登记的顶层声明。
func hasTopDecls(prog *Program) bool {
	return len(prog.Funcs) > 0 || len(prog.Structs) > 0 || len(prog.Interfaces) > 0 ||
		len(prog.Impls) > 0 || len(prog.Libraries) > 0 || len(prog.TypeAliases) > 0
}

// declSummary 汇总本次登记的定义（回显用）。
func declSummary(prog *Program) string {
	var b strings.Builder
	for _, f := range prog.Funcs {
		fmt.Fprintf(&b, "已定义 %s\n", funcSignature(f))
	}
	for _, sd := range prog.Structs {
		if sd.Name != "" && !strings.HasPrefix(sd.Name, "__anon_") {
			fmt.Fprintf(&b, "已定义 type struct%s %s;\n", typeParams(sd.TypeParams), sd.Name)
		}
	}
	for _, id := range prog.Interfaces {
		if id.Name != "" && !strings.HasPrefix(id.Name, "__anon_") {
			fmt.Fprintf(&b, "已定义 type interface%s %s;\n", typeParams(id.TypeParams), id.Name)
		}
	}
	for _, im := range prog.Impls {
		fmt.Fprintf(&b, "已定义 impl%s %s;（%d 个方法）\n", typeParams(im.TypeParams), im.Type, len(im.Methods))
	}
	for _, lb := range prog.Libraries {
		fmt.Fprintf(&b, "已定义 library %s;（%d 个符号）\n", lb.Name, len(lb.Methods))
	}
	for _, ta := range prog.TypeAliases {
		fmt.Fprintf(&b, "已定义 type %s %s;\n", ta.Type, ta.Name)
	}
	return b.String()
}

// shiftErrLine 把错误位置按 delta 平移（包装输入带来的行号偏移），最小为 1。
func shiftErrLine(err error, delta int) error {
	if err == nil || delta == 0 {
		return err
	}
	shift := func(p Pos) Pos {
		if p.Line > 0 {
			p.Line += delta
			if p.Line < 1 {
				p.Line = 1
			}
		}
		return p
	}
	switch e := err.(type) {
	case *ParseError:
		p := shift(Pos{Line: e.Line, Col: e.Col})
		return &ParseError{Msg: e.Msg, Line: p.Line, Col: p.Col}
	case *LexError:
		p := shift(Pos{Line: e.Line, Col: e.Col})
		return &LexError{Msg: e.Msg, Line: p.Line, Col: p.Col}
	case *CheckError:
		return &CheckError{Msg: e.Msg, Pos: shift(e.Pos)}
	case *RunError:
		return &RunError{Msg: e.Msg, Pos: shift(e.Pos), Ctx: e.Ctx}
	}
	return err
}
