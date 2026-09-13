package lang

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// pendingDebug 包级预置调试器（execute 前安装；--debug 模式单程序运行）。
var pendingDebug func(in *interp)

type dbgState struct {
	bps      map[string]bool
	file     string
	stepping bool
	sc       *scope
	in       *interp
	r        *bufio.Reader
	w        io.Writer
}

// RunDebug 调试模式运行（编译产物/源码带 --bp "file:line,..."）。
func RunDebug(prog *Program, filename string, args []string, stdin io.Reader, stdout io.Writer, breakpoints []string) error {
	if len(breakpoints) == 0 {
		_, err := runWithInterp(prog, filename, args, stdin, stdout)
		return err
	}
	bps := map[string]bool{}
	for _, bp := range breakpoints {
		bp = strings.TrimSpace(bp)
		if bp != "" {
			bps[bp] = true
		}
	}
	pendingDebug = func(in *interp) {
		in.dbg = &dbgState{bps: bps, file: filename, in: in, r: bufio.NewReader(stdin), w: stdout}
	}
	_, err := runWithInterp(prog, filename, args, stdin, stdout)
	return err
}

// stPos 语句位置（断点行提取）。
func stPos(st Stmt) Pos {
	switch t := st.(type) {
	case *ExprStmt:
		return posOfExpr(t.X)
	case *LogStmt:
		return posOfExpr(t.X)
	case *ReturnStmt:
		return t.Pos
	case *DeclStmt:
		return t.Pos
	case *IfStmt:
		return posOfExpr(t.Cond)
	case *WhileStmt:
		return posOfExpr(t.Cond)
	case *ForStmt:
		return posOfExpr(t.Iter)
	case *ForCStmt:
		return t.Pos
	case *BreakStmt:
		return t.Pos
	case *DeleteStmt:
		return t.Pos
	case *TryStmt:
		return t.Pos
	case *AssignStmt:
		return t.Pos
	}
	return Pos{}
}

func posOfExpr(e Expr) Pos {
	switch t := e.(type) {
	case *IntLit:
		return t.Pos
	case *FloatLit:
		return t.Pos
	case *StrLit:
		return t.Pos
	case *BoolLit:
		return t.Pos
	case *Ident:
		return t.Pos
	case *BinOp:
		return t.Pos
	case *UnOp:
		return t.Pos
	case *CallExpr:
		return t.Pos
	case *MemberExpr:
		return t.Pos
	case *IndexExpr:
		return t.Pos
	case *ScopeCall:
		return t.Pos
	}
	return Pos{}
}

// hitBreak 断点检查（每语句一次判定；无调试器零开销）。
func (in *interp) hitBreak(pos Pos, sc *scope) bool {
	d := in.dbg
	if d == nil {
		return false
	}
	key := d.file + ":" + itoa(pos.Line)
	if !(d.bps[key] || d.stepping) {
		return false
	}
	d.stepping = false
	d.interact(sc, pos.Line)
	return true
}

func (d *dbgState) interact(sc *scope, line int) {
	d.sc = sc
	fmt.Fprintf(d.w, "break %s:%d — [c]ontinue [n]ext [p var] [q]uit> ", d.file, line)
	for {
		cmd, err := d.r.ReadString('\n')
		if err != nil {
			return
		}
		cmd = strings.TrimSpace(cmd)
		switch {
		case cmd == "c" || cmd == "":
			return
		case cmd == "n":
			d.stepping = true
			return
		case cmd == "q":
			panic(&dbgQuit{})
		case strings.HasPrefix(cmd, "p ") || strings.HasPrefix(cmd, "print "):
			name := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(cmd, "p "), "print "))
			fmt.Fprintln(d.w, d.debugVar(name))
			fmt.Fprintf(d.w, "> ")
		default:
			fmt.Fprintf(d.w, "unknown %q (c/n/p var/q)\n> ", cmd)
		}
	}
}

type dbgQuit struct{}

func (*dbgQuit) Error() string { return "debugger quit" }

func (d *dbgState) debugVar(name string) string {
	if v, err := d.sc.get(name, Pos{}); err == nil {
		return name + " = " + v.String()
	}
	return name + " = <undefined>"
}
