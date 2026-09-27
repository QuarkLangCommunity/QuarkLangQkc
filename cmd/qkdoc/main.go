// qkdoc: QuarkLang API documentation generator (toolchain member)
//
// Usage: qkdoc [flags] files...
//
//	-o file    Write to a file (default stdout)
//	-html      Emit self-contained HTML (default Markdown)
//	-all       Include non-pub symbols (default exports pub only; exports everything when the file has no pub)
//	-title T   Title (defaults to the file name)
//	--lang       Output language zh|en (default Chinese; QK_LANG also works)
//	--version  Print version
//
// Documentation source: the `//` / `/* */` comment block immediately above a declaration; when there is none, the trailing comment on the same line is used.
// Exit codes: 0 = success; 1 = parse failure; 2 = usage error.
package main

import (
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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: qkdoc [-o file] [-html] [-all] [-title T] [--version] files...")
	fmt.Fprintln(w, i18n.T("  QuarkLang API 文档生成：/* */ 与 // 注释 + pub 导出 → Markdown / HTML"))
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
				fmt.Fprintln(stderr, i18n.T("qkdoc: -o 需要一个文件参数"))
				return 2
			}
			i++
			outFile = args[i]
		case "-title":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, i18n.T("qkdoc: -title 需要一个参数"))
				return 2
			}
			i++
			title = args[i]
		case "-html":
			htmlOut = true
		case "-all":
			all = true
		case "--lang", "-lang":
			if i+1 < len(args) {
				i++
				if l, ok := i18n.ParseLang(args[i]); ok {
					i18n.SetLocale(l)
					lang.SetLocalizer(i18n.New(nil, l)) // interpreter diagnostics follow the same language
				} // zh | en; unknown values keep the current locale
			}
		case "--version", "-V":
			fmt.Fprintln(stdout, "qkdoc", version)
			return 0
		case "-h", "--help":
			usage(stdout)
			return 0
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				fmt.Fprintf(stderr, i18n.T("qkdoc: 未知参数 %s\n"), a)
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
		prog, comments, macros, err := lang.ParseSourceAll(string(data))
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", f, err)
			return 1
		}
		doc := lang.BuildDocWithMacros(prog, comments, macros)
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
	fmt.Fprintf(stdout, i18n.T("qkdoc: 已写入 %s（%d 字节）\n"), outFile, len(out))
	return 0
}
