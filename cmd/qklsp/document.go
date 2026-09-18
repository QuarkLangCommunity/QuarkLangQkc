package main

// 文档模型与语言分析：把 internal/lang 的解析/检查能力包装成 LSP 需要的位置信息。

import (
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"quarklang/internal/lang"
)

// Document 是一个打开的文档及其分析结果。
type Document struct {
	URI   string
	Path  string
	Text  string
	Lines []string

	prog     *lang.Program
	comments []lang.Comment
	doc      *lang.Doc

	parseErr   error
	parseErrLn int
	parseErrCl int

	syms        []symbol // 符号表缓存（文本变化时失效；补全/跳转/悬停共用）
	symsValid   bool
	globalCands []symbol // 补全候选的「与光标行无关」部分（关键字 + 内置 + 全局符号）
	globalValid bool
}

func newDocument(uri, text string) *Document {
	d := &Document{URI: uri, Path: uriToPath(uri), Text: text}
	d.analyze()
	return d
}

func (d *Document) setText(text string) {
	d.Text = text
	d.analyze()
}

// analyze 重新解析并构建文档模型（符号表按需惰性重建）。
func (d *Document) analyze() {
	d.syms, d.symsValid = nil, false
	d.globalCands, d.globalValid = nil, false
	d.Lines = strings.Split(d.Text, "\n")
	d.prog, d.comments, d.parseErr = nil, nil, nil
	d.doc = nil
	d.parseErrLn, d.parseErrCl = 0, 0
	prog, comments, err := lang.ParseSourceWithComments(d.Text)
	if err != nil {
		d.parseErr = err
		d.parseErrLn, d.parseErrCl = errLineCol(err)
		return
	}
	d.prog, d.comments = prog, comments
	d.doc = lang.BuildDoc(prog, comments)
	d.doc.Path = d.Path
}

// uriToPath 把 file:// URI 转成本地路径（非 file URI 原样返回）。
func uriToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return uri
	}
	u, err := url.Parse(uri)
	if err != nil {
		return strings.TrimPrefix(uri, "file://")
	}
	p := u.Path
	if p == "" {
		p = u.Opaque
	}
	if decoded, err := url.PathUnescape(p); err == nil {
		p = decoded
	}
	return filepath.FromSlash(pathFromURISlash(p))
}

// pathFromURISlash 去掉 Windows 盘符前的引导斜杠（/C:/x → C:/x），其余平台原样返回。
// 单独抽出便于在任何平台做单元测试（跨系统一致性）。
func pathFromURISlash(p string) string {
	if runtime.GOOS == "windows" && len(p) >= 3 && p[0] == '/' &&
		((p[1] >= 'a' && p[1] <= 'z') || (p[1] >= 'A' && p[1] <= 'Z')) && p[2] == ':' {
		return p[1:]
	}
	return p
}

// pathToURI 把本地路径转成 file:// URI（LSP 要求三段斜杠：file:///C:/x、file:///home/x）。
func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return "file://" + uriSlashPath(filepath.ToSlash(abs))
}

// uriSlashPath 保证斜杠路径以 / 开头（Windows 盘符 C:/x → /C:/x），拼上 file:// 后为 file:///C:/x。
func uriSlashPath(p string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}

// ---------- 位置换算（LSP 0 基 / UTF-16 列 ↔ 语言 1 基 / 字节列） ----------

// byteColToUTF16 行内字节列（1 基）→ UTF-16 码元偏移（0 基）。
func byteColToUTF16(line string, byteCol int) int {
	if byteCol <= 1 {
		return 0
	}
	idx := byteCol - 1
	if idx > len(line) {
		idx = len(line)
	}
	prefix := line[:idx]
	n := 0
	for _, r := range prefix {
		n += len(utf16.Encode([]rune{r}))
	}
	return n
}

// utf16ToByteCol UTF-16 码元偏移（0 基）→ 行内字节列（1 基）。
func utf16ToByteCol(line string, char int) int {
	if char <= 0 {
		return 1
	}
	units := 0
	byteIdx := 0
	for byteIdx < len(line) {
		r, size := utf8.DecodeRuneInString(line[byteIdx:])
		w := len(utf16.Encode([]rune{r}))
		if units+w > char {
			break
		}
		units += w
		byteIdx += size
	}
	return byteIdx + 1
}

type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspLocation struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

func rangeAt(line, col, endCol int) lspRange {
	if endCol <= col {
		endCol = col + 1
	}
	return lspRange{
		Start: lspPosition{Line: line - 1, Character: col - 1},
		End:   lspPosition{Line: line - 1, Character: endCol - 1},
	}
}

// lineByteRange：整行范围（诊断用）。
func (d *Document) lineRange(line int) lspRange {
	if line < 1 {
		line = 1
	}
	text := ""
	if line <= len(d.Lines) {
		text = d.Lines[line-1]
	}
	return lspRange{
		Start: lspPosition{Line: line - 1, Character: 0},
		End:   lspPosition{Line: line - 1, Character: len(utf16.Encode([]rune(text)))},
	}
}

func errLineCol(err error) (int, int) {
	switch e := err.(type) {
	case *lang.LexError:
		return e.Line, e.Col
	case *lang.ParseError:
		return e.Line, e.Col
	case *lang.CheckError:
		return e.Pos.Line, e.Pos.Col
	case *lang.RunError:
		return e.Pos.Line, e.Pos.Col
	}
	return 0, 0
}

// ---------- 符号表 ----------

type symbol struct {
	Name  string
	Kind  string // func/struct/interface/alias/impl/space/library/field/method/local/param
	Sig   string
	Doc   string
	Line  int
	Col   int
	Local bool   // 局部变量/形参（只在函数范围内可见）
	Scope [2]int // 局部符号的作用范围（函数起止行，1 基；非局部为 0）
}

// symbols 汇总文档中的全部符号（全局 + 局部）；结果按文档版本缓存。
func (d *Document) symbols() []symbol {
	if d.symsValid {
		return d.syms
	}
	out := d.buildSymbols()
	d.syms, d.symsValid = out, true
	return out
}

func (d *Document) buildSymbols() []symbol {
	var out []symbol
	if d.doc != nil {
		for _, it := range d.doc.All() {
			kind := it.Kind
			out = append(out, symbol{Name: it.Name, Kind: kind, Sig: it.Signature, Doc: it.Doc, Line: it.Pos.Line, Col: it.Pos.Col})
			for _, f := range it.Fields {
				if f.Name == "" {
					continue
				}
				out = append(out, symbol{Name: f.Name, Kind: "member", Sig: f.Signature, Doc: f.Doc, Line: f.Pos.Line, Col: f.Pos.Col})
			}
		}
	}
	if d.prog != nil {
		for _, fn := range d.prog.Funcs {
			start, end := fn.Pos.Line, fn.BodyEnd.Line
			for _, p := range fn.Params {
				out = append(out, symbol{Name: p.Name, Kind: "param", Sig: p.Type, Line: p.Pos.Line, Col: p.Pos.Col, Local: true, Scope: [2]int{start, end}})
			}
			collectLocals(fn.Body, start, end, &out)
		}
		for _, im := range d.prog.Impls {
			for _, m := range im.Methods {
				start, end := m.Pos.Line, m.BodyEnd.Line
				for _, p := range m.Params {
					out = append(out, symbol{Name: p.Name, Kind: "param", Sig: p.Type, Line: p.Pos.Line, Col: p.Pos.Col, Local: true, Scope: [2]int{start, end}})
				}
				collectLocals(m.Body, start, end, &out)
			}
		}
	}
	return out
}

// collectLocals 收集函数体内的局部声明（变量 / 循环变量 / catch 变量）。
func collectLocals(b *lang.Block, start, end int, out *[]symbol) {
	if b == nil {
		return
	}
	add := func(name, typ string, pos lang.Pos, kind string) {
		if name == "" || strings.HasPrefix(name, "_") {
			return
		}
		*out = append(*out, symbol{Name: name, Kind: kind, Sig: typ, Line: pos.Line, Col: pos.Col, Local: true, Scope: [2]int{start, end}})
	}
	for _, st := range b.Stmts {
		switch s := st.(type) {
		case *lang.DeclStmt:
			add(s.Name, s.Type, s.Pos, "local")
		case *lang.ForStmt:
			add(s.Var, s.Type, s.Pos, "local")
			collectLocals(s.Body, start, end, out)
		case *lang.ForCStmt:
			if s.Init != nil {
				collectLocals(&lang.Block{Stmts: []lang.Stmt{s.Init}}, start, end, out)
			}
			collectLocals(s.Body, start, end, out)
		case *lang.TryStmt:
			add(s.CatchVar, s.CatchVarType, s.Pos, "local")
			collectLocals(s.Try, start, end, out)
			collectLocals(s.Catch, start, end, out)
		case *lang.IfStmt:
			collectLocals(s.Then, start, end, out)
			collectLocals(s.Else, start, end, out)
		case *lang.WhileStmt:
			collectLocals(s.Body, start, end, out)
		}
	}
}

// lookup 在给定位置查找符号：先找作用域覆盖该位置的局部符号（取最近的前向声明），
// 再退回全局符号。返回 nil 表示未找到。
func (d *Document) lookup(name string, line int) *symbol {
	syms := d.symbols()
	var best *symbol
	for i := range syms {
		s := &syms[i]
		if s.Name != name {
			continue
		}
		if s.Local {
			if line < s.Scope[0] || line > s.Scope[1] || s.Line > line {
				continue
			}
			if best == nil || (best.Local && s.Line > best.Line) {
				best = s
			}
			continue
		}
		if best == nil || !best.Local {
			if best == nil || strings.Compare(kindRank(s.Kind), kindRank(best.Kind)) < 0 {
				best = s
			}
		}
	}
	return best
}

func kindRank(kind string) string {
	switch kind {
	case "func":
		return "0"
	case "struct", "interface", "alias":
		return "1"
	case "impl", "space":
		return "2"
	case "library":
		return "3"
	}
	return "4"
}

// wordAt 取指定位置（1 基行、字节列）上的标识符。
func (d *Document) wordAt(line, col int) string {
	if line < 1 || line > len(d.Lines) {
		return ""
	}
	text := []rune(d.Lines[line-1])
	// 字节列 → rune 下标
	byteIdx := col - 1
	if byteIdx < 0 {
		byteIdx = 0
	}
	line0 := d.Lines[line-1]
	if byteIdx > len(line0) {
		byteIdx = len(line0)
	}
	runeIdx := utf8.RuneCountInString(line0[:byteIdx])
	isWord := func(r rune) bool {
		return r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') || r > 0x7f
	}
	if runeIdx < len(text) && !isWord(text[runeIdx]) && runeIdx > 0 && isWord(text[runeIdx-1]) {
		runeIdx-- // 光标落在词尾之后
	}
	start, end := runeIdx, runeIdx
	for start > 0 && isWord(text[start-1]) {
		start--
	}
	for end < len(text) && isWord(text[end]) {
		end++
	}
	if start == end {
		return ""
	}
	return string(text[start:end])
}

// completionCandidates 汇总补全候选（关键字 + 类型 + 符号）。
// 「与光标行无关」的部分（关键字/内置/全局符号）按文档版本缓存；
// 每次只额外挑选该行可见的局部符号 → 逐键补全近乎零分配。
func (d *Document) completionCandidates(line int) []symbol {
	if !d.globalValid {
		seen := map[string]bool{}
		var globals []symbol
		push := func(s symbol) {
			if s.Name == "" || seen[s.Name] {
				return
			}
			seen[s.Name] = true
			globals = append(globals, s)
		}
		for _, kw := range keywords {
			push(symbol{Name: kw, Kind: "keyword"})
		}
		for _, s := range d.symbols() {
			if !s.Local {
				push(s)
			}
		}
		for _, b := range builtins {
			push(symbol{Name: b, Kind: "builtin"})
		}
		sort.SliceStable(globals, func(i, j int) bool {
			if globals[i].Kind != globals[j].Kind {
				return kindRank(globals[i].Kind) < kindRank(globals[j].Kind)
			}
			return globals[i].Name < globals[j].Name
		})
		d.globalCands, d.globalValid = globals, true
	}
	out := make([]symbol, 0, len(d.globalCands)+8)
	out = append(out, d.globalCands...)
	seenGlobal := func(name string) bool {
		for i := range d.globalCands {
			if d.globalCands[i].Name == name {
				return true
			}
		}
		return false
	}
	for _, s := range d.symbols() {
		if !s.Local || line < s.Scope[0] || line > s.Scope[1] || s.Line > line {
			continue
		}
		if seenGlobal(s.Name) {
			continue
		}
		out = append(out, s)
	}
	return out
}

var keywords = []string{
	"fn", "type", "struct", "interface", "impl", "space", "library", "program", "import", "pub",
	"const", "copyd", "if", "else", "while", "for", "break", "return", "log", "delete", "try", "catch",
	"new", "null", "true", "false", "self", "void", "Self", "expand", "dynamic", "memorize",
	"int", "long", "char", "float", "bool", "String", "List", "HashTable", "pointer", "IOStream", "thread",
}

var builtins = []string{
	"size", "get", "append", "appendAll", "contains", "startsWith", "endsWith", "indexOf", "substring",
	"split", "trim", "toLower", "toUpper", "replace", "charAt", "toInt", "toFloat", "toString",
	"head", "tail", "next", "reset", "put", "keys", "remove", "sort", "__sort__",
	"println", "print", "readln", "setIn", "setOut",
}
