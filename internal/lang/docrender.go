package lang

// ============ qkdoc: Markdown / HTML rendering ============

import (
	"fmt"
	"html"
	"strings"
)

// DocOptions controls rendering.
type DocOptions struct {
	Title string // title (defaults to Path)
	All   bool   // true = include non-pub symbols; false = render pub only (when the file has no pub, everything is rendered)
	Path  string // source file path (for display)
}

// visible decides whether an item should be rendered:
//   - with -all, or when the file has no pub → everything is rendered;
//   - impl / space / library do not take part in pub filtering (the language does not allow pub on them;
//     they are namespaces and implementation bodies, which is exactly what a library's public API consists of).
func (d *Doc) visible(it *DocItem, opts DocOptions) bool {
	if opts.All || !d.HasPub() {
		return true
	}
	switch it.Kind {
	case DocImpl, DocSpace, DocLibrary, DocMacro:
		return true // the language does not allow pub on them; macros/spaces/impls are exactly what a library exposes as its API
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

// Markdown renders a Markdown document.
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
	b.WriteString(msg("> 形态：`program "))
	b.WriteString(d.Kind)
	b.WriteString(";`")
	if len(d.Imports) > 0 {
		var q []string
		for _, im := range d.Imports {
			q = append(q, "`"+im+"`")
		}
		b.WriteString(msg("　·　导入："))
		b.WriteString(strings.Join(q, ", "))
	}
	b.WriteString("\n\n")

	// Overview
	var rows []*DocItem
	for _, it := range d.All() {
		if d.visible(it, opts) {
			rows = append(rows, it)
		}
	}
	if len(rows) == 0 {
		b.WriteString(msg("_（没有可导出的符号）_\n"))
		return b.String()
	}
	b.WriteString(msg("## 概览\n\n| 类别 | 名称 | 签名 | 摘要 |\n|---|---|---|---|\n"))
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

	// Details
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
		b.WriteString(msg("| 字段 | 类型 | 说明 |\n|---|---|---|\n"))
		for _, f := range it.Fields {
			writeRow(f.Name, "`"+f.Type+"`", cell(Summary(f.Doc)))
		}
		return
	}
	b.WriteString(msg("| 成员 | 签名 | 说明 |\n|---|---|---|\n"))
	for _, f := range it.Fields {
		name, sig := f.Name, f.Signature
		if sig == "" {
			sig = f.Name // expand interface X; and the like
		}
		writeRow(name, cell(sig), cell(Summary(f.Doc)))
	}
}

// markdownSizeHint estimates the output size (about 200 bytes per item plus the file comment) for a single preallocation.
func markdownSizeHint(d *Doc) int {
	n := len(d.All())
	size := 256 + n*220 + len(d.FileDoc)
	return size
}

// cell makes text safe to place inside a Markdown table.
func cell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// HTML renders a self-contained HTML page (no dependencies, no external assets).
func (d *Doc) HTML(opts DocOptions) string {
	return MarkdownToHTML(d.Markdown(opts), d.title(opts))
}

// MarkdownToHTML converts qkdoc-generated Markdown into a self-contained HTML page
// (covering only the syntax subset this package emits: headings/tables/code blocks/quotes/paragraphs; all text is escaped).
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

// renderMDBlock converts one Markdown block into HTML (covering the syntax subset qkdoc itself emits).
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
			if i == 1 && strings.Contains(ln, "---") { // separator row
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

// inlineMD handles inline `code` and **bold**, escaping everything else.
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

// splitImpls separates impl / space (both are ImplDecl, distinguished by Kind).
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
