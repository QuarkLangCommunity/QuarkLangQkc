// qkdoc：QuarkLang API 文档生成器（工具链成员）
//
// 用法: qkdoc [flags] files...
//
//	-o file    写入文件（默认 stdout）
//	-html      输出自包含 HTML（默认 Markdown）
//	-all       包含未 pub 的符号（默认只导出 pub；文件无 pub 时导出全部）
//	-title T   标题（默认取文件名）
//	--version  打印版本
//
// 文档来源：声明上方紧邻的 `//` / `/* */` 注释块；没有上方注释时取同行行尾注释。
// 退出码：0 = 成功；1 = 解析失败；2 = 用法错误。
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"quarklang/internal/lang"
)

// version 发布版本：构建时用 -ldflags "-X main.version=vX.Y.Z" 注入（见 scripts/build-release.sh）
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: qkdoc [-o file] [-html] [-all] [-title T] [--version] files...")
	fmt.Fprintln(w, "  QuarkLang API 文档生成：/* */ 与 // 注释 + pub 导出 → Markdown / HTML")
}

func run(args []string, stdout, stderr io.Writer) int {
	var files []string
	outFile, title := "", ""
	htmlOut, all := false, false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-o":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "qkdoc: -o 需要一个文件参数")
				return 2
			}
			i++
			outFile = args[i]
		case "-title":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "qkdoc: -title 需要一个参数")
				return 2
			}
			i++
			title = args[i]
		case "-html":
			htmlOut = true
		case "-all":
			all = true
		case "--version", "-V":
			fmt.Fprintln(stdout, "qkdoc", version)
			return 0
		case "-h", "--help":
			usage(stdout)
			return 0
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				fmt.Fprintf(stderr, "qkdoc: 未知参数 %s\n", a)
				usage(stderr)
				return 2
			}
			files = append(files, a)
		}
	}
	if len(files) == 0 {
		usage(stderr)
		return 2
	}

	var sections []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintln(stderr, "qkdoc:", err)
			return 2
		}
		prog, comments, err := lang.ParseSourceWithComments(string(data))
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", f, err)
			return 1
		}
		doc := lang.BuildDoc(prog, comments)
		doc.Path = f
		t := title
		if t == "" || len(files) > 1 {
			t = f
		}
		sections = append(sections, doc.Markdown(lang.DocOptions{Title: t, All: all, Path: f}))
	}

	md := strings.Join(sections, "\n---\n\n")
	var out string
	if htmlOut {
		t := title
		if t == "" {
			t = "QuarkLang API"
			if len(files) == 1 {
				t = files[0]
			}
		}
		out = lang.MarkdownToHTML(md, t)
	} else {
		out = md
	}

	if outFile == "" {
		fmt.Fprint(stdout, out)
		return 0
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(outFile); err == nil {
		mode = info.Mode()
	}
	if err := os.WriteFile(outFile, []byte(out), mode); err != nil {
		fmt.Fprintln(stderr, "qkdoc:", err)
		return 2
	}
	fmt.Fprintf(stdout, "qkdoc: 已写入 %s（%d 字节）\n", outFile, len(out))
	return 0
}
