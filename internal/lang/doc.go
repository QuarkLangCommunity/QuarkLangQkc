package lang

// ============ qkdoc：API 文档模型 ============
//
// 从单文件 AST + 注释构建文档模型：`/* */` 与 `//` 均可作文档注释，
// 取「紧邻声明上方、无空行」的连续注释块（godoc 规则）。`pub` 决定是否导出。

import (
	"sort"
	"strings"
)

// DocItemKind 文档条目类别。
const (
	DocFunc      = "func"
	DocStruct    = "struct"
	DocInterface = "interface"
	DocAlias     = "alias"
	DocImpl      = "impl"
	DocSpace     = "space"
	DocLibrary   = "library"
)

// DocField 是结构体字段 / 接口方法 / 实现方法 / 库符号。
type DocField struct {
	Name      string
	Signature string // 方法/符号的完整签名（字段为空）
	Type      string // 字段类型（方法为空）
	Doc       string
	Pos       Pos
}

// DocItem 是一个文档条目。
type DocItem struct {
	Kind      string
	Name      string
	Signature string
	Doc       string
	Pos       Pos
	Pub       bool
	Fields    []DocField
}

// Doc 是一个文件的文档模型。
type Doc struct {
	Path      string
	Kind      string // main | library
	FileDoc   string
	Imports   []string
	Funcs     []*DocItem
	Types     []*DocItem // struct / interface / alias
	Impls     []*DocItem // impl / space
	Libraries []*DocItem
}

// All 按源码顺序返回全部条目（用于概览表）。
func (d *Doc) All() []*DocItem {
	var out []*DocItem
	out = append(out, d.Types...)
	out = append(out, d.Impls...)
	out = append(out, d.Libraries...)
	out = append(out, d.Funcs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pos.Line != out[j].Pos.Line {
			return out[i].Pos.Line < out[j].Pos.Line
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// HasPub 判断文件是否使用了 pub 导出。
func (d *Doc) HasPub() bool {
	for _, it := range d.All() {
		if it.Pub {
			return true
		}
	}
	return false
}

// BuildDoc 由单文件 AST 与注释构建文档模型。prog.Src 用于判定「行尾注释」。
func BuildDoc(prog *Program, comments []Comment) *Doc {
	d := &Doc{Kind: prog.Kind, Imports: append([]string{}, prog.Imports...)}
	if d.Kind == "" {
		d.Kind = "main"
	}
	docs := newDocIndex(comments, prog.Src)
	pub := map[string]bool{}
	for _, p := range prog.Pub {
		pub[p] = true
	}

	firstLine := 0
	noteDecl := func(pos Pos) {
		if firstLine == 0 || (pos.Line > 0 && pos.Line < firstLine) {
			firstLine = pos.Line
		}
	}

	// 别名
	for _, ta := range prog.TypeAliases {
		noteDecl(ta.Pos)
		d.Types = append(d.Types, &DocItem{
			Kind: DocAlias, Name: ta.Name, Pos: ta.Pos, Pub: pub[ta.Name],
			Signature: "type " + ta.Type + " " + ta.Name + ";",
			Doc:       docs.docFor(ta.Pos.Line),
		})
	}
	// 结构体
	structNames := map[string]bool{}
	for _, sd := range prog.Structs {
		if sd.Name == "" || strings.HasPrefix(sd.Name, "__anon_") {
			continue
		}
		structNames[sd.Name] = true
		noteDecl(sd.Pos)
		it := &DocItem{
			Kind: DocStruct, Name: sd.Name, Pos: sd.Pos, Pub: pub[sd.Name],
			Signature: "type struct" + typeParams(sd.TypeParams) + " " + sd.Name + ";",
			Doc:       docs.docFor(sd.Pos.Line),
		}
		for _, m := range sd.Members {
			it.Fields = append(it.Fields, DocField{Name: m.Name, Type: m.Type, Pos: m.Pos, Doc: docs.docFor(m.Pos.Line)})
		}
		d.Types = append(d.Types, it)
	}
	// 接口
	for _, id := range prog.Interfaces {
		if id.Name == "" || strings.HasPrefix(id.Name, "__anon_") {
			continue
		}
		noteDecl(id.Pos)
		it := &DocItem{
			Kind: DocInterface, Name: id.Name, Pos: id.Pos, Pub: pub[id.Name],
			Signature: "type interface" + typeParams(id.TypeParams) + " " + id.Name + ";",
			Doc:       docs.docFor(id.Pos.Line),
		}
		for _, m := range id.Methods {
			it.Fields = append(it.Fields, DocField{
				Name: m.Name, Pos: m.Pos, Doc: docs.docFor(m.Pos.Line),
				Signature: "fn " + m.Name + "(" + paramsText(m.Params) + ") " + retText(m.Ret) + ";",
			})
		}
		for _, ex := range id.Expands {
			it.Fields = append(it.Fields, DocField{Name: "expand interface " + ex})
		}
		d.Types = append(d.Types, it)
	}
	// impl / space（无接口名的自我实现；space 与 impl 同为 ImplDecl，按是否实名 struct 区分）
	for _, im := range prog.Impls {
		noteDecl(im.Pos)
		kind := DocImpl
		sig := "impl" + typeParams(im.TypeParams) + " " + im.Type + ";"
		if !structNames[im.Type] {
			kind = DocSpace
			sig = "space " + im.Type + ";"
		}
		it := &DocItem{
			Kind: kind, Name: im.Type, Pos: im.Pos, Pub: pub[im.Type],
			Signature: sig, Doc: docs.docFor(im.Pos.Line),
		}
		for _, m := range im.Methods {
			it.Fields = append(it.Fields, DocField{
				Name: m.Name, Pos: m.Pos, Doc: docs.docFor(m.Pos.Line),
				Signature: funcSignature(m),
			})
		}
		d.Impls = append(d.Impls, it)
	}
	// library（FFI 绑定）
	for _, lb := range prog.Libraries {
		noteDecl(lb.Pos)
		it := &DocItem{
			Kind: DocLibrary, Name: lb.Name, Pos: lb.Pos, Pub: pub[lb.Name],
			Signature: "library " + lb.Name + ";", Doc: docs.docFor(lb.Pos.Line),
		}
		for _, m := range lb.Methods {
			it.Fields = append(it.Fields, DocField{Name: m.Name, Signature: libFuncSignature(m), Pos: m.Pos, Doc: docs.docFor(m.Pos.Line)})
		}
		d.Libraries = append(d.Libraries, it)
	}
	// 顶层函数
	for _, fn := range prog.Funcs {
		noteDecl(fn.Pos)
		d.Funcs = append(d.Funcs, &DocItem{
			Kind: DocFunc, Name: fn.Name, Pos: fn.Pos, Pub: pub[fn.Name],
			Signature: funcSignature(fn), Doc: docs.docFor(fn.Pos.Line),
		})
	}
	d.FileDoc = docs.fileDoc(firstLine)
	return d
}

func typeParams(tp []string) string {
	if len(tp) == 0 {
		return ""
	}
	return "<" + strings.Join(tp, ", ") + ">"
}

func retText(ret string) string {
	if strings.TrimSpace(ret) == "" {
		return "void"
	}
	return ret
}

func paramsText(params []Param) string {
	var parts []string
	for _, p := range params {
		s := p.Type + " " + p.Name
		if p.Decor != "" {
			s = p.Decor + " " + s
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

func funcSignature(f *FuncDecl) string {
	return "fn" + typeParams(f.TypeParams) + " " + f.Name + "(" + paramsText(f.Params) + ") " + retText(f.Ret)
}

// libFuncSignature：library 绑定符号是已编译的 Func（无函数体）。
func libFuncSignature(f *Func) string {
	return "fn" + typeParams(f.TypeParams) + " " + f.Name + "(" + paramsText(f.Params) + ") " + retText(f.Ret)
}

// ---------- 注释索引 ----------

// docComment 是注释 + 位置上下文：trailing = 同行前面已有代码（`int x; // 说明`）。
type docComment struct {
	Comment
	trailing bool
}

type docIndex struct {
	byEnd    map[int][]docComment
	byStart  map[int]docComment
	earliest int
}

// newDocIndex 建立注释索引。src 用于判定行尾注释（同一行前面是否有代码）。
func newDocIndex(comments []Comment, src string) *docIndex {
	idx := &docIndex{byEnd: map[int][]docComment{}, byStart: map[int]docComment{}, earliest: 1 << 30}
	lines := strings.Split(src, "\n")
	for _, c := range comments {
		dc := docComment{Comment: c}
		if c.Pos.Line >= 1 && c.Pos.Line <= len(lines) {
			line := lines[c.Pos.Line-1]
			col := c.Pos.Col - 1
			if col > len(line) {
				col = len(line)
			}
			if col > 0 && strings.TrimSpace(line[:col]) != "" {
				dc.trailing = true
			}
		}
		idx.byEnd[c.EndLine] = append(idx.byEnd[c.EndLine], dc)
		if _, ok := idx.byStart[c.Pos.Line]; !ok {
			idx.byStart[c.Pos.Line] = dc
		}
		if c.Pos.Line < idx.earliest {
			idx.earliest = c.Pos.Line
		}
	}
	for k := range idx.byEnd {
		cs := idx.byEnd[k]
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].Pos.Col < cs[j].Pos.Col })
		idx.byEnd[k] = cs
	}
	return idx
}

// docFor 取声明上方紧邻（无空行）的连续注释块；没有上方注释时退回同行行尾注释。
func (idx *docIndex) docFor(declLine int) string {
	if declLine <= 0 {
		return ""
	}
	if s := idx.leadingDoc(declLine); s != "" {
		return s
	}
	return idx.trailingDoc(declLine)
}

// leadingDoc 声明上方紧邻的连续注释块（只取独立成行的注释）。
func (idx *docIndex) leadingDoc(declLine int) string {
	cur := declLine - 1
	var blocks []string
	for {
		cs, ok := idx.byEnd[cur]
		if !ok || len(cs) == 0 {
			break
		}
		var c docComment
		found := false
		for _, x := range cs { // 同一行多个注释时取最左的非行尾注释
			if !x.trailing {
				c, found = x, true
				break
			}
		}
		if !found {
			break // 上一行只有行尾注释（`... ; // 说明`）→ 不是本声明的文档
		}
		blocks = append([]string{cleanComment(c.Comment)}, blocks...)
		cur = c.Pos.Line - 1
	}
	return strings.TrimSpace(strings.Join(blocks, ""))
}

// trailingDoc 与声明同行的行尾注释（`int x; // 横坐标`）。
func (idx *docIndex) trailingDoc(declLine int) string {
	for _, c := range idx.byEnd[declLine] {
		if c.trailing {
			return strings.TrimSpace(cleanComment(c.Comment))
		}
	}
	return ""
}

// fileDoc 取文件头注释：从最早注释起、彼此紧邻的连续注释块；
// 若该块正好是首个声明的文档注释（与本行紧邻），则不重复算作文件注释。
func (idx *docIndex) fileDoc(firstDeclLine int) string {
	if idx.earliest >= 1<<30 {
		return ""
	}
	first, ok := idx.byStart[idx.earliest]
	if !ok || first.trailing {
		return ""
	}
	if firstDeclLine > 0 && first.Pos.Line >= firstDeclLine {
		return ""
	}
	var blocks []string
	cur := first.Pos.Line
	lastEnd := 0
	for {
		c, ok := idx.byStart[cur]
		if !ok || c.trailing {
			break
		}
		blocks = append(blocks, cleanComment(c.Comment))
		lastEnd = c.EndLine
		cur = c.EndLine + 1
	}
	if firstDeclLine > 0 && lastEnd == firstDeclLine-1 {
		return "" // 这段就是首个声明的文档注释
	}
	return strings.TrimSpace(strings.Join(blocks, ""))
}

// cleanComment 去掉注释定界符并对齐缩进。
func cleanComment(c Comment) string {
	text := c.Text
	if c.Block {
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		var lines []string
		for _, ln := range strings.Split(text, "\n") {
			ln = strings.TrimRight(ln, " \t")
			ln = strings.TrimPrefix(ln, " ")
			ln = strings.TrimPrefix(ln, "*")
			ln = strings.TrimPrefix(ln, " ")
			lines = append(lines, ln)
		}
		text = strings.Join(lines, "\n")
	} else {
		var lines []string
		for _, ln := range strings.Split(text, "\n") {
			ln = strings.TrimPrefix(strings.TrimSpace(ln), "//")
			ln = strings.TrimPrefix(ln, " ")
			lines = append(lines, ln)
		}
		text = strings.Join(lines, "\n")
	}
	// 去公共缩进
	lines := strings.Split(text, "\n")
	indent := -1
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		n := len(ln) - len(strings.TrimLeft(ln, " \t"))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	if indent > 0 {
		for i, ln := range lines {
			if len(ln) >= indent {
				lines[i] = ln[indent:]
			}
		}
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n") + "\n"
}

// Summary 取文档首行摘要（表格用）。
func Summary(doc string) string {
	for _, ln := range strings.Split(doc, "\n") {
		s := strings.TrimSpace(ln)
		if s != "" {
			return s
		}
	}
	return ""
}
