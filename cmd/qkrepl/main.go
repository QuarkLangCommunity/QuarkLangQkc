// qkrepl: QuarkLang interactive evaluator (toolchain member)
//
// Usage: qkrepl [flags]
//
//	-e code    Evaluate a snippet and exit (repeatable; exits 1 on failure)
//	-q         Quiet mode (no banner or prompt)
//	--lang       Output language zh|en (default Chinese; QK_LANG also works)
//	--version  Print version
//
// Interactive mode (stdin is a terminal): `qk> ` prompt; an unfinished block continues automatically (`..> `).
// Batch mode (stdin is piped/redirected): no prompt, evaluate line by line, keep going after errors, exit 1 if any error occurred.
//
// In-session commands (leading `:`):
//
//	:help            Show help
//	:quit / :exit    Quit
//	:load <file.qk>  Load a file (registers its functions/types/implementations; does not run main automatically)
//
// Exit codes: 0 = normal; 1 = evaluation error; 2 = usage error.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"quarklang/internal/i18n"
	"strings"

	"quarklang/internal/lang"
)

// version is the release version, injected at build time with -ldflags "-X main.version=vX.Y.Z" (see scripts/build-release.sh)
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: qkrepl [-e code] [-q] [--version]")
	fmt.Fprintln(w, i18n.T("  QuarkLang 交互式求值：多行块、跨输入持久环境、:help/:load/:quit"))
}

const helpText = `命令：
  :help            显示本帮助
  :quit / :exit    退出
  :load <file.qk>  加载文件：登记其中的函数/类型/实现（不自动执行 main；可随后调用 main(io)）
说明：
  - 表达式直接回显其值；语句以 ; 结尾（省略时自动补）
  - 块未闭合会自动续行；变量/函数/类型在后续输入中持续可见
  - log 的记录、io.println 的输出会即时显示`

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	quiet := false
	var exprs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-e":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, i18n.T("qkrepl: -e 需要一段代码"))
				return 2
			}
			i++
			exprs = append(exprs, args[i])
		case "-q":
			quiet = true
		case "--lang", "-lang":
			if i+1 < len(args) {
				i++
				i18n.SetLocale(args[i]) // zh | en; unknown values keep the current locale
			}
		case "--version", "-V":
			fmt.Fprintln(stdout, "qkrepl", version)
			return 0
		case "-h", "--help":
			usage(stdout)
			return 0
		default:
			fmt.Fprintf(stderr, i18n.T("qkrepl: 未知参数 %s\n"), a)
			usage(stderr)
			return 2
		}
	}

	sess, err := lang.NewREPLSession(stdin, stdout)
	if err != nil {
		fmt.Fprintln(stderr, "qkrepl:", err)
		return 1
	}

	// -e: evaluate once and exit
	if len(exprs) > 0 {
		failed := false
		for _, code := range exprs {
			out, err := sess.Eval(code)
			fmt.Fprint(stdout, out)
			if err != nil {
				fmt.Fprintln(stderr, "error:", err)
				failed = true
			}
		}
		if failed {
			return 1
		}
		return 0
	}

	interactive := isTerminal(stdin)
	if interactive && !quiet {
		fmt.Fprintf(stdout, i18n.T("qkrepl %s —— QuarkLang 交互式求值（:help 帮助，:quit 退出）\n"), version)
	}

	reader := bufio.NewReader(stdin)
	var buf strings.Builder
	hadErr := false
	for {
		if interactive {
			if buf.Len() == 0 {
				fmt.Fprint(stdout, "qk> ")
			} else {
				fmt.Fprint(stdout, "..> ")
			}
		}
		line, readErr := reader.ReadString('\n')
		done := readErr != nil
		if line != "" {
			trimmed := strings.TrimSpace(line)
			if buf.Len() == 0 && strings.HasPrefix(trimmed, ":") {
				quit, err := command(trimmed, sess, stdout, stderr)
				if err != nil {
					hadErr = true
					fmt.Fprintln(stderr, "error:", err)
				}
				if quit || done {
					break
				}
				continue
			}
			buf.WriteString(line)
			if !sess.Incomplete(buf.String()) {
				out, err := sess.Eval(buf.String())
				fmt.Fprint(stdout, out)
				if err != nil {
					hadErr = true
					fmt.Fprintln(stderr, "error:", err)
				}
				buf.Reset()
			}
		}
		if done {
			break
		}
	}
	if buf.Len() > 0 { // wrap-up: leftover unclosed input is still evaluated so the error is visible
		if out, err := sess.Eval(buf.String()); err != nil {
			hadErr = true
			fmt.Fprintln(stderr, "error:", err)
		} else {
			fmt.Fprint(stdout, out)
		}
	}
	if hadErr {
		return 1
	}
	return 0
}

// command handles a leading `:` command; reports whether to quit.
func command(line string, sess *lang.REPLSession, stdout, stderr io.Writer) (bool, error) {
	fields := strings.Fields(line)
	switch fields[0] {
	case ":help", ":h", ":?":
		fmt.Fprintln(stdout, i18n.T(helpText))
		return false, nil
	case ":quit", ":q", ":exit":
		return true, nil
	case ":load":
		if len(fields) < 2 {
			return false, errors.New(i18n.T(":load 需要一个文件参数"))
		}
		path := fields[1]
		data, err := os.ReadFile(path)
		if err != nil {
			return false, err
		}
		out, err := sess.Eval(string(data))
		fmt.Fprint(stdout, out)
		if err != nil {
			return false, err
		}
		return false, nil
	}
	return false, errors.New(i18n.T("未知命令 %s（:help 查看帮助）", fields[0]))
}

// isTerminal reports whether the input comes from a terminal (no third-party dependencies: character device check).
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
