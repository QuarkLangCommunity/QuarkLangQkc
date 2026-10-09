package main

// Document model and language analysis: wraps the parse/check capabilities of internal/lang into the position information LSP needs.

import (
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// Document is an open document together with its analysis result.
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

	syms        []symbol // symbol table cache (invalidated when the text changes; shared by completion/definition/hover)
	symsValid   bool
	globalCands []symbol // the "independent of the cursor line" part of the completion candidates (keywords + builtins + global symbols)
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

// analyze re-parses and rebuilds the document model (the symbol table is rebuilt lazily on demand).
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

// uriToPath converts a file:// URI into a local path (non-file URIs are returned unchanged).
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

// pathFromURISlash strips the leading slash before a Windows drive letter (/C:/x -> C:/x); other platforms are returned unchanged.
// Extracted separately so it can be unit-tested on any platform (cross-system consistency).
func pathFromURISlash(p string) string {
	if runtime.GOOS == "windows" && len(p) >= 3 && p[0] == '/' &&
		((p[1] >= 'a' && p[1] <= 'z') || (p[1] >= 'A' && p[1] <= 'Z')) && p[2] == ':' {
		return p[1:]
	}
	return p
}

// pathToURI converts a local path into a file:// URI (LSP requires three slashes: file:///C:/x, file:///home/x).
func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return "file://" + uriSlashPath(filepath.ToSlash(abs))
}

// uriSlashPath ensures a slash path starts with / (Windows drive C:/x -> /C:/x), which becomes file:///C:/x once file:// is prefixed.
func uriSlashPath(p string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}

// ---------- Position conversion (LSP 0-based / UTF-16 columns <-> language 1-based / byte columns) ----------

// byteColToUTF16 converts an in-line byte column (1-based) to a UTF-16 code unit offset (0-based).
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

// utf16ToByteCol converts a UTF-16 code unit offset (0-based) to an in-line byte column (1-based).
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

// lineByteRange: the range of a whole line (used for diagnostics).
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

// ---------- Symbol table ----------

type symbol struct {
	Name  string
	Kind  string // func/struct/interface/alias/impl/space/library/field/method/local/param
	Sig   string
	Doc   string
	Line  int
	Col   int
	Local bool   // local variable/parameter (visible only inside the function scope)
	Scope [2]int // scope of a local symbol (function start and end lines, 1-based; 0 for non-local)
}

// symbols collects every symbol in the document (global + local); the result is cached per document version.
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

// collectLocals collects local declarations inside function bodies (variables / loop variables / catch variables).
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

// lookup finds a symbol at the given position: first a local symbol whose scope covers that position (the nearest preceding declaration),
// then falling back to global symbols. Returns nil when nothing is found.
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

// wordAt returns the identifier at the given position (1-based line, byte column).
func (d *Document) wordAt(line, col int) string {
	if line < 1 || line > len(d.Lines) {
		return ""
	}
	text := []rune(d.Lines[line-1])
	// byte column -> rune index
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
		runeIdx-- // the cursor sits after the end of the word
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

// completionCandidates collects completion candidates (keywords + types + symbols).
// The part independent of the cursor line (keywords/builtins/global symbols) is cached per document version;
// only the local symbols visible on that line are picked additionally -> near-zero allocation per keystroke.
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
