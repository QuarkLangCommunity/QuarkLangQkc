package lang

// ============ qkdoc: API documentation model ============
//
// Builds a documentation model from a single-file AST plus comments: both `/* */` and `//` count as doc comments,
// taking the contiguous comment block immediately above a declaration with no blank line in between (the godoc rule). `pub` decides export.

import (
	"sort"
	"strings"
)

// DocItemKind is a documentation item kind.
const (
	DocFunc      = "func"
	DocStruct    = "struct"
	DocInterface = "interface"
	DocAlias     = "alias"
	DocImpl      = "impl"
	DocSpace     = "space"
	DocLibrary   = "library"
	DocMacro     = "macro"
)

// DocField is a struct field / interface method / impl method / library symbol.
type DocField struct {
	Name      string
	Signature string // full signature of the method/symbol (empty for a field)
	Type      string // field type (empty for a method)
	Doc       string
	Pos       Pos
}

// DocItem is one documentation item.
type DocItem struct {
	Kind      string
	Name      string
	Signature string
	Doc       string
	Pos       Pos
	Pub       bool
	Fields    []DocField
}

// Doc is a file's documentation model.
type Doc struct {
	Path      string
	Kind      string // main | library
	FileDoc   string
	Imports   []string
	Funcs     []*DocItem
	Types     []*DocItem // struct / interface / alias
	Impls     []*DocItem // impl / space
	Libraries []*DocItem
	Macros    []*DocItem // #macro named-parameter macros (expanded at token level, not in the AST)
}

// All returns every item in source order (used by the overview table).
func (d *Doc) All() []*DocItem {
	var out []*DocItem
	out = append(out, d.Types...)
	out = append(out, d.Impls...)
	out = append(out, d.Libraries...)
	out = append(out, d.Macros...)
	out = append(out, d.Funcs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pos.Line != out[j].Pos.Line {
			return out[i].Pos.Line < out[j].Pos.Line
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// HasPub reports whether the file uses pub exports.
func (d *Doc) HasPub() bool {
	for _, it := range d.All() {
		if it.Pub {
			return true
		}
	}
	return false
}

// BuildDoc builds the documentation model from a single-file AST and comments (macro definitions are excluded; see BuildDocWithMacros).
func BuildDoc(prog *Program, comments []Comment) *Doc {
	return BuildDocWithMacros(prog, comments, nil)
}

// BuildDocWithMacros is the same but also includes macro definitions (#macro is split out before parsing, so it must be passed in explicitly).
func BuildDocWithMacros(prog *Program, comments []Comment, macros []*MacroDef) *Doc {
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

	// Alias
	for _, ta := range prog.TypeAliases {
		noteDecl(ta.Pos)
		d.Types = append(d.Types, &DocItem{
			Kind: DocAlias, Name: ta.Name, Pos: ta.Pos, Pub: pub[ta.Name],
			Signature: "type " + ta.Type + " " + ta.Name + ";",
			Doc:       docs.docFor(ta.Pos.Line),
		})
	}
	// Struct
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
	// Interface
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
	// impl / space (self-implementations with no interface name; space and impl are both ImplDecl, distinguished by whether the struct is named)
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
	// library (FFI binding)
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
	// Macro (named-parameter macro)
	for _, m := range macros {
		noteDecl(m.Pos)
		sig := "#macro " + m.Name + " (" + strings.Join(m.Params, ", ") + ")"
		d.Macros = append(d.Macros, &DocItem{
			Kind: DocMacro, Name: m.Name, Pos: m.Pos,
			Signature: sig, Doc: docs.docFor(m.Pos.Line),
		})
	}
	// Top-level functions
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

// libFuncSignature: a library binding symbol is an already-compiled Func (with no body).
func libFuncSignature(f *Func) string {
	return "fn" + typeParams(f.TypeParams) + " " + f.Name + "(" + paramsText(f.Params) + ") " + retText(f.Ret)
}

// ---------- Comment index ----------

// docComment is a comment plus its positional context: trailing = code already appeared earlier on the same line (`int x; // note`).
type docComment struct {
	Comment
	trailing bool
}

type docIndex struct {
	byEnd    map[int][]docComment
	byStart  map[int]docComment
	earliest int
}

// newDocIndex builds the comment index. src is used to detect end-of-line comments (whether code precedes them on the same line).
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

// docFor takes the contiguous comment block immediately above a declaration (no blank line); with none, it falls back to a same-line trailing comment.
func (idx *docIndex) docFor(declLine int) string {
	if declLine <= 0 {
		return ""
	}
	if s := idx.leadingDoc(declLine); s != "" {
		return s
	}
	return idx.trailingDoc(declLine)
}

// leadingDoc: the contiguous comment block immediately above a declaration (only line-standing comments).
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
		for _, x := range cs { // with several comments on the same line, take the leftmost non-trailing one
			if !x.trailing {
				c, found = x, true
				break
			}
		}
		if !found {
			break // the previous line has only a trailing comment (`... ; // description`) → not documentation for this declaration
		}
		blocks = append([]string{cleanComment(c.Comment)}, blocks...)
		cur = c.Pos.Line - 1
	}
	return strings.TrimSpace(strings.Join(blocks, ""))
}

// trailingDoc: the end-of-line comment on the declaration's line (`int x; // x coordinate`).
func (idx *docIndex) trailingDoc(declLine int) string {
	for _, c := range idx.byEnd[declLine] {
		if c.trailing {
			return strings.TrimSpace(cleanComment(c.Comment))
		}
	}
	return ""
}

// fileDoc takes the file header comment: the contiguous comment block starting from the earliest comment;
// if that block is exactly the first declaration's doc comment (adjacent to this line), it is not counted again as a file comment.
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
		return "" // this block is the doc comment of the first declaration
	}
	return strings.TrimSpace(strings.Join(blocks, ""))
}

// cleanComment strips the comment delimiters and aligns the indentation.
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
	// Strip the common indentation
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

// Summary takes the first line of documentation (for tables).
func Summary(doc string) string {
	for _, ln := range strings.Split(doc, "\n") {
		s := strings.TrimSpace(ln)
		if s != "" {
			return s
		}
	}
	return ""
}
