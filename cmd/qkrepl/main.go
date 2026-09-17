// qkrepl：QuarkLang 交互式求值（工具链成员）
//
// 用法: qkrepl [flags]
//
//	-e code    求值一段代码后退出（可重复；失败退出 1）
//	-q         安静模式（不打印横幅与提示符）
//	--version  打印版本
//
// 交互模式（stdin 是终端）：`qk> ` 提示符；块未闭合时自动续行（`..> `）。
// 批处理模式（stdin 被管道/重定向）：不打印提示符，逐行求值，出错继续，结束时若有错退出 1。
//
// 会话内命令（行首 `:`）：
//
//	:help            显示帮助
//	:quit / :exit    退出
//	:load <file.qk>  加载文件（登记其中的函数/类型/实现，不自动执行 main）
//
// 退出码：0 = 正常；1 = 求值出错；2 = 用法错误。
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"quarklang/internal/lang"
)

// version 发布版本：构建时用 -ldflags "-X main.version=vX.Y.Z" 注入（见 scripts/build-release.sh）
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: qkrepl [-e code] [-q] [--version]")
	fmt.Fprintln(w, "  QuarkLang 交互式求值：多行块、跨输入持久环境、:help/:load/:quit")
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
				fmt.Fprintln(stderr, "qkrepl: -e 需要一段代码")
				return 2
			}
			i++
			exprs = append(exprs, args[i])
		case "-q":
			quiet = true
		case "--version", "-V":
			fmt.Fprintln(stdout, "qkrepl", version)
			return 0
		case "-h", "--help":
			usage(stdout)
			return 0
		default:
			fmt.Fprintf(stderr, "qkrepl: 未知参数 %s\n", a)
			usage(stderr)
			return 2
		}
	}

	sess, err := lang.NewREPLSession(stdin, stdout)
	if err != nil {
		fmt.Fprintln(stderr, "qkrepl:", err)
		return 1
	}

	// -e：一次性求值
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
		fmt.Fprintf(stdout, "qkrepl %s —— QuarkLang 交互式求值（:help 帮助，:quit 退出）\n", version)
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
	if buf.Len() > 0 { // 收尾：未闭合的残留输入仍尝试求值，报错可见
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

// command 处理行首 `:` 命令；返回是否退出。
func command(line string, sess *lang.REPLSession, stdout, stderr io.Writer) (bool, error) {
	fields := strings.Fields(line)
	switch fields[0] {
	case ":help", ":h", ":?":
		fmt.Fprintln(stdout, helpText)
		return false, nil
	case ":quit", ":q", ":exit":
		return true, nil
	case ":load":
		if len(fields) < 2 {
			return false, fmt.Errorf(":load 需要一个文件参数")
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
	return false, fmt.Errorf("未知命令 %s（:help 查看帮助）", fields[0])
}

// isTerminal 判断输入是否来自终端（无第三方依赖：字符设备判定）。
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
