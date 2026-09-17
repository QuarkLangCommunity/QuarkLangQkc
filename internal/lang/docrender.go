package lang

// ============ qkdoc：Markdown / HTML 渲染 ============

import (
	"fmt"
	"html"
	"strings"
)

// DocOptions 控制渲染。
type DocOptions struct {
	Title string // 标题（默认取 Path）
	All   bool   // true = 含未 pub 的符号；false = 只渲染 pub（文件无 pub 时自动渲染全部）
	Path  string // 源文件路径（展示用）
}

// visible 判断条目是否应渲染：
//   - -all 或文件没有 pub → 全部渲染；
//   - impl / space / library 不参与 pub 过滤（语言里 pub 不能前缀它们；
//     它们是命名空间与实现主体，正是库对外 API 的载体）。
func (d *Doc) visible(it *DocItem, opts DocOptions) bool {
	if opts.All || !d.HasPub() {
		return true
	}
	switch it.Kind {
	case DocImpl, DocSpace, DocLibrary, DocMacro:
		return true // 语言里 pub 不能前缀它们；宏/空间/实现正是库对外 API 的载体
	}
	return it.Pub
}

func (d *Doc) title(opts DocOptions) string {
	if opts.Title != "" {
		return opts.Title
	}
	if opts.Path != "" {
		return opts.Path
	}
	if d.Path != "" {
		return d.Path
	}
	return "QuarkLang API"
}

func kindLabel(kind string) string {
	switch kind {
	case DocFunc:
		return "函数"
	case DocStruct:
		return "结构体"
	case DocInterface:
		return "接口"
	case DocAlias:
		return "类型别名"
	case DocImpl:
		return "实现"
	case DocSpace:
		return "空间"
	case DocLibrary:
		return "系统库"
	case DocMacro:
		return "宏"
	}
	return kind
}

// Markdown 渲染为 Markdown 文档。
func (d *Doc) Markdown(opts DocOptions) string {
	var b strings.Builder
	b.Grow(markdownSizeHint(d))
	b.WriteString("# ")
	b.WriteString(d.title(opts))
	b.WriteString("\n\n")
	if d.FileDoc != "" {
		b.WriteString(d.FileDoc)
		b.WriteString("\n")
	}
	b.WriteString("> 形态：`program ")
	b.WriteString(d.Kind)
	b.WriteString(";`")
	if len(d.Imports) > 0 {
		var q []string
		for _, im := range d.Imports {
			q = append(q, "`"+im+"`")
		}
		b.WriteString("　·　导入：")
		b.WriteString(strings.Join(q, ", "))
	}
	b.WriteString("\n\n")

	// 概览
	var rows []*DocItem
	for _, it := range d.All() {
		if d.visible(it, opts) {
			rows = append(rows, it)
		}
	}
	if len(rows) == 0 {
		b.WriteString("_（没有可导出的符号）_\n")
		return b.String()
	}
	b.WriteString("## 概览\n\n| 类别 | 名称 | 签名 | 摘要 |\n|---|---|---|---|\n")
	for _, it := range rows {
		b.WriteString("| ")
		b.WriteString(kindLabel(it.Kind))
		b.WriteString(" | `")
		b.WriteString(it.Name)
		b.WriteString("` | ")
		b.WriteString(cell(it.Signature))
		b.WriteString(" | ")
		b.WriteString(cell(Summary(it.Doc)))
		b.WriteString(" |\n")
	}
	b.WriteString("\n")

	// 详情
	impls, spaces := splitImpls(d.Impls)
	sections := []struct {
		title string
		items []*DocItem
	}{
		{"函数", d.Funcs},
		{"类型", d.Types},
		{"实现", impls},
		{"空间", spaces},
		{"宏", d.Macros},
		{"系统库", d.Libraries},
	}
	for _, sec := range sections {
		var vis []*DocItem
		for _, it := range sec.items {
			if d.visible(it, opts) {
				vis = append(vis, it)
			}
		}
		if len(vis) == 0 {
			continue
		}
		b.WriteString("## ")
		b.WriteString(sec.title)
		b.WriteString("\n\n")
		for _, it := range vis {
			b.WriteString("### ")
			b.WriteString(it.Name)
			b.WriteString("\n\n```qk\n")
			b.WriteString(it.Signature)
			b.WriteString("\n```\n\n")
			if it.Doc != "" {
				b.WriteString(it.Doc)
				b.WriteString("\n")
			}
			if len(it.Fields) > 0 {
				b.WriteString("\n")
				writeFieldsMD(&b, it)
			}
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func writeFieldsMD(b *strings.Builder, it *DocItem) {
	writeRow := func(a, c, d string) {
		b.WriteString("| `")
		b.WriteString(a)
		b.WriteString("` | ")
		b.WriteString(c)
		b.WriteString(" | ")
		b.WriteString(d)
		b.WriteString(" |\n")
	}
	if it.Kind == DocStruct {
		b.WriteString("| 字段 | 类型 | 说明 |\n|---|---|---|\n")
		for _, f := range it.Fields {
			writeRow(f.Name, "`"+f.Type+"`", cell(Summary(f.Doc)))
		}
		return
	}
	b.WriteString("| 成员 | 签名 | 说明 |\n|---|---|---|\n")
	for _, f := range it.Fields {
		name, sig := f.Name, f.Signature
		if sig == "" {
			sig = f.Name // expand interface X; 之类
		}
		writeRow(name, cell(sig), cell(Summary(f.Doc)))
	}
}

// markdownSizeHint 预估输出规模（每条目约 200 字节 + 文件注释），用于一次性预分配。
func markdownSizeHint(d *Doc) int {
	n := len(d.All())
	size := 256 + n*220 + len(d.FileDoc)
	return size
}

// cell 让文本可安全放进 Markdown 表格。
func cell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// HTML 渲染为自包含 HTML 页面（零依赖、无外部资源）。
func (d *Doc) HTML(opts DocOptions) string {
	return MarkdownToHTML(d.Markdown(opts), d.title(opts))
}

// MarkdownToHTML 把 qkdoc 生成的 Markdown 转为自包含 HTML 页面
// （只覆盖本包生成的语法子集：标题/表格/代码块/引用/段落；转义所有文本）。
func MarkdownToHTML(md, title string) string {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"zh\">\n<head>\n<meta charset=\"utf-8\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", html.EscapeString(title))
	b.WriteString(`<style>
:root { color-scheme: light dark; }
body { font-family: system-ui, "Noto Sans CJK SC", sans-serif; max-width: 60rem; margin: 2rem auto; padding: 0 1rem; line-height: 1.6; }
h1 { border-bottom: 2px solid #8884; padding-bottom: .3rem; }
h2 { margin-top: 2rem; border-bottom: 1px solid #8884; padding-bottom: .2rem; }
h3 { margin-bottom: .3rem; font-family: ui-monospace, monospace; }
pre { background: #8881; padding: .6rem .8rem; border-radius: 6px; overflow-x: auto; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #8884; padding: .3rem .5rem; text-align: left; vertical-align: top; }
code { font-family: ui-monospace, monospace; }
blockquote { color: #888; margin: .5rem 0; }
</style>
</head>
<body>
`)
	for _, block := range strings.Split(md, "\n\n") {
		b.WriteString(renderMDBlock(block))
	}
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

// renderMDBlock 把一小段 Markdown 块转成 HTML（覆盖 qkdoc 自己生成的语法子集）。
func renderMDBlock(block string) string {
	block = strings.Trim(block, "\n")
	if block == "" {
		return ""
	}
	lines := strings.Split(block, "\n")
	switch {
	case strings.HasPrefix(lines[0], "```"):
		var body []string
		for _, ln := range lines[1:] {
			if strings.HasPrefix(ln, "```") {
				break
			}
			body = append(body, ln)
		}
		return "<pre><code>" + html.EscapeString(strings.Join(body, "\n")) + "</code></pre>\n"
	case strings.HasPrefix(lines[0], "# "):
		return "<h1>" + inlineMD(strings.TrimPrefix(lines[0], "# ")) + "</h1>\n"
	case strings.HasPrefix(lines[0], "## "):
		return "<h2>" + inlineMD(strings.TrimPrefix(lines[0], "## ")) + "</h2>\n"
	case strings.HasPrefix(lines[0], "### "):
		return "<h3>" + inlineMD(strings.TrimPrefix(lines[0], "### ")) + "</h3>\n"
	case strings.HasPrefix(lines[0], "> "):
		return "<blockquote>" + inlineMD(strings.TrimPrefix(lines[0], "> ")) + "</blockquote>\n"
	case strings.HasPrefix(lines[0], "|"):
		var b strings.Builder
		b.WriteString("<table>\n")
		for i, ln := range lines {
			if strings.TrimSpace(ln) == "" {
				continue
			}
			if i == 1 && strings.Contains(ln, "---") { // 分隔行
				continue
			}
			cells := splitRow(ln)
			tag := "td"
			if i == 0 {
				tag = "th"
			}
			b.WriteString("<tr>")
			for _, c := range cells {
				fmt.Fprintf(&b, "<%s>%s</%s>", tag, inlineMD(c), tag)
			}
			b.WriteString("</tr>\n")
		}
		b.WriteString("</table>\n")
		return b.String()
	}
	return "<p>" + inlineMD(strings.Join(lines, " ")) + "</p>\n"
}

func splitRow(ln string) []string {
	ln = strings.TrimSpace(ln)
	ln = strings.TrimPrefix(ln, "|")
	ln = strings.TrimSuffix(ln, "|")
	if ln == "" {
		return nil
	}
	parts := strings.Split(ln, "|")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(strings.ReplaceAll(p, "\\|", "|"))
	}
	return parts
}

// inlineMD 处理行内 `code` 与 **粗体**，其余转义。
func inlineMD(s string) string {
	var b strings.Builder
	for {
		i := strings.IndexByte(s, '`')
		if i < 0 {
			b.WriteString(boldMD(s))
			break
		}
		j := strings.IndexByte(s[i+1:], '`')
		if j < 0 {
			b.WriteString(boldMD(s))
			break
		}
		b.WriteString(boldMD(s[:i]))
		b.WriteString("<code>" + html.EscapeString(s[i+1:i+1+j]) + "</code>")
		s = s[i+j+2:]
	}
	return b.String()
}

func boldMD(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "**")
		if i < 0 {
			b.WriteString(html.EscapeString(s))
			break
		}
		j := strings.Index(s[i+2:], "**")
		if j < 0 {
			b.WriteString(html.EscapeString(s))
			break
		}
		b.WriteString(html.EscapeString(s[:i]))
		b.WriteString("<strong>" + html.EscapeString(s[i+2:i+2+j]) + "</strong>")
		s = s[i+j+4:]
	}
	return b.String()
}

// splitImpls 把 impl / space 分开（两者同为 ImplDecl，按 Kind 区分）。
func splitImpls(items []*DocItem) (impls, spaces []*DocItem) {
	for _, it := range items {
		if it.Kind == DocSpace {
			spaces = append(spaces, it)
		} else {
			impls = append(impls, it)
		}
	}
	return impls, spaces
}
