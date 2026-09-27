package lang

// ============ qkrepl: interactive evaluation session (multi-line blocks) ============
//
// It must live inside the package: the interpreter's evaluation facilities (interp / execStmt / evalExpr / scope) are unexported.
//
// Design notes:
//   - the session holds an *interp (obtained from a minimal bootstrap program via runWithInterp), and the
//     global scope survives across inputs → variables, functions and types stay visible in later inputs;
//   - when an input parses as a **top-level declaration** (fn/struct/interface/impl/space/library/type),
//     it goes through registerProgram (the same path as a whole program, so behaviour matches);
//   - otherwise it is wrapped in `fn __repl_N() void { ... }` and evaluated statement by statement: expressions echo their value,
//     `log` entries are echoed; only Lex/Parse run (not Compile) → CallExpr.FnIdx stays -1 and
//     dispatch happens **by name** at runtime, so function indices from earlier chunks cannot go stale (see README).

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"quarklang/internal/i18n"
	"strings"
)

// REPLSession is one interactive evaluation session.
type REPLSession struct {
	in    *interp
	out   io.Writer
	stdin io.Reader
	chunk int
}

// NewREPLSession creates a session: bootstrap an empty main to obtain the interpreter, then declare the io the REPL can use.
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
	// The session's own IOStream (rd is settable inside the package, so readln cannot nil-dereference)
	stream := &IOStream{In: stdin, Out: stdout, rd: bufio.NewReader(stdin)}
	_ = in.globalScope.declare("io", IOV(stream), Pos{})
	return s, nil
}

// Eval evaluates one input and returns the text to print (evaluation output + expression echo).
func (s *REPLSession) Eval(src string) (out string, err error) {
	defer func() { // 解释器不保证无 panic（FFI / 外部对象），REPL 必须活下来
		if r := recover(); r != nil {
			out, err = "", errors.New(i18n.T("REPL: 内部错误（已恢复）: %v", r))
		}
	}()
	if strings.TrimSpace(src) == "" {
		return "", nil
	}
	// 1) top-level declaration: register it into the session
	if prog, perr := ParseSource(src); perr == nil && hasTopDecls(prog) {
		if rerr := s.in.registerProgramReplacing(prog); rerr != nil {
			return "", rerr
		}
		return declSummary(prog), nil
	}
	// 2) statement / expression: wrap in a function and evaluate statement by statement
	s.chunk++
	prog, perr := ParseSource(wrapChunk(fmt.Sprintf("__repl_%d", s.chunk), src))
	if perr != nil {
		return "", shiftErrLine(perr, -1)
	}
	fd := prog.Funcs[0]
	fn := &Func{Name: fd.Name, Params: fd.Params, Ret: fd.Ret, Body: fd.Body, Pos: fd.Pos}
	return s.execBody(fd.Body, fn)
}

// mayEndStatement reports whether the last character can end a statement/expression (deciding whether to append an automatic `;`).
func mayEndStatement(c byte) bool {
	switch {
	case c >= '0' && c <= '9', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= 0x80:
		return true
	case c == '_' || c == '"' || c == '\x60' || c == ')' || c == ']':
		return true
	}
	return false
}

// wrapChunk wraps REPL input as a function body. The language requires statements to end with `;`:
// when the last character ends an expression (identifier/literal/`)`/`]`) a `;` is appended automatically (REPL convenience).
func wrapChunk(name, src string) string {
	body := src
	t := strings.TrimSpace(body)
	if t != "" && mayEndStatement(t[len(t)-1]) {
		body += ";"
	}
	return fmt.Sprintf("fn %s() void {\n%s\n}\n", name, body)
}

// execBody executes statements one by one in the session's global scope and returns the text to print.
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

// resolveStructLit fills in the target type name for declarations such as `P p = .{...};`.
// The whole-program path fills this in during typecheck (typecheck.go: StructLit.Name = the target struct),
// but the REPL skips type checking, so it is completed here from the declared type annotation — otherwise the value would become an anonymous struct and method calls would fail.
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

// Incomplete reports whether the input is "not finished yet": an unclosed block, an unterminated string, etc.
// The CLI keeps reading the next line accordingly (multi-line blocks).
func (s *REPLSession) Incomplete(src string) bool {
	if strings.TrimSpace(src) == "" {
		return false
	}
	// Unterminated at the lexical level (string / raw string / block comment)
	if _, _, err := LexWithComments(src); err != nil {
		return strings.Contains(err.Error(), "unterminated")
	}
	// Parses as a complete program or statement block → complete
	if _, err := ParseSource(src); err == nil {
		return false
	}
	if _, err := ParseSource(wrapChunk("__repl_probe", src)); err == nil {
		return false
	}
	// Both parses failed: if the error is "unclosed block" more input is needed
	incomplete := func(err error) bool {
		return err != nil && strings.Contains(err.Error(), "unterminated block")
	}
	if _, err := ParseSource(src); incomplete(err) {
		return true
	}
	_, err := ParseSource(wrapChunk("__repl_probe", src))
	return incomplete(err)
}

// hasTopDecls reports whether the program has any registrable top-level declaration.
func hasTopDecls(prog *Program) bool {
	return len(prog.Funcs) > 0 || len(prog.Structs) > 0 || len(prog.Interfaces) > 0 ||
		len(prog.Impls) > 0 || len(prog.Libraries) > 0 || len(prog.TypeAliases) > 0
}

// declSummary summarises the definitions registered this round (for echoing).
func declSummary(prog *Program) string {
	var b strings.Builder
	for _, f := range prog.Funcs {
		fmt.Fprintf(&b, i18n.T("已定义 %s\n"), funcSignature(f))
	}
	for _, sd := range prog.Structs {
		if sd.Name != "" && !strings.HasPrefix(sd.Name, "__anon_") {
			fmt.Fprintf(&b, i18n.T("已定义 type struct%s %s;\n"), typeParams(sd.TypeParams), sd.Name)
		}
	}
	for _, id := range prog.Interfaces {
		if id.Name != "" && !strings.HasPrefix(id.Name, "__anon_") {
			fmt.Fprintf(&b, i18n.T("已定义 type interface%s %s;\n"), typeParams(id.TypeParams), id.Name)
		}
	}
	for _, im := range prog.Impls {
		fmt.Fprintf(&b, i18n.T("已定义 impl%s %s;（%d 个方法）\n"), typeParams(im.TypeParams), im.Type, len(im.Methods))
	}
	for _, lb := range prog.Libraries {
		fmt.Fprintf(&b, i18n.T("已定义 library %s;（%d 个符号）\n"), lb.Name, len(lb.Methods))
	}
	for _, ta := range prog.TypeAliases {
		fmt.Fprintf(&b, i18n.T("已定义 type %s %s;\n"), ta.Type, ta.Name)
	}
	return b.String()
}

// shiftErrLine shifts an error position by delta (the line offset introduced by wrapping the input), with a minimum of 1.
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
