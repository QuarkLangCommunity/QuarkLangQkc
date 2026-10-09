// qklsp: QuarkLang language server (toolchain member)
//
// Usage: qklsp [-L dir]...  (runs LSP over stdio; started by the editor)
//
// Capabilities: diagnostics (parse/type/static checks), go-to-definition, completion, hover, document symbols.
// Zero third-party dependencies: the JSON-RPC framing and protocol subset are self-implemented; language analysis reuses internal/lang
// (the same front end as qkcheck / qkdoc, so results in the editor match the command line).
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// version is the release version, injected at build time with -ldflags "-X main.version=vX.Y.Z" (see scripts/build-release.sh)
var version = "dev"

type server struct {
	conn    *rpcConn
	docs    map[string]*Document
	libDirs []string
	quit    bool
}

func main() {
	libDirs, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "qklsp:", err)
		os.Exit(2)
	}
	s := &server{conn: newRPCConn(os.Stdin, os.Stdout), docs: map[string]*Document{}, libDirs: libDirs}
	if err := s.serve(); err != nil {
		fmt.Fprintln(os.Stderr, "qklsp:", err)
		os.Exit(1)
	}
}

func parseArgs(args []string) ([]string, error) {
	var libDirs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-L":
			if i+1 >= len(args) {
				return nil, errors.New(i18n.T("-L 需要一个目录参数"))
			}
			i++
			libDirs = append(libDirs, args[i])
		case "--lang", "-lang":
			if i+1 < len(args) {
				i++
				if l, ok := i18n.ParseLang(args[i]); ok {
					i18n.SetLocale(l)
					lang.SetLocalizer(i18n.New(nil, l)) // interpreter diagnostics follow the same language
				} // zh | en; unknown values keep the current locale
			}
		case "--version", "-V":
			fmt.Println("qklsp", version)
			os.Exit(0)
		case "-h", "--help":
			fmt.Println(i18n.T("usage: qklsp [-L dir]...  （stdio 上提供 LSP 服务）"))
			os.Exit(0)
		default:
			return nil, errors.New(i18n.T("未知参数 %s", args[i]))
		}
	}
	return libDirs, nil
}

// serve is the main loop: read a message -> dispatch -> reply/notify; ends on `exit` or EOF.
func (s *server) serve() error {
	for !s.quit {
		msg, err := s.conn.read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.handle(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) handle(msg rpcMessage) error {
	switch msg.Method {
	case "initialize":
		return s.conn.reply(msg.ID, map[string]interface{}{
			"capabilities": map[string]interface{}{
				"textDocumentSync": map[string]interface{}{
					"openClose": true,
					"change":    1, // full sync
				},
				"definitionProvider":         true,
				"completionProvider":         map[string]interface{}{"triggerCharacters": []string{".", ":", "<"}},
				"hoverProvider":              true,
				"documentSymbolProvider":     true,
				"diagnosticProvider":         nil,
				"workspaceSymbolProvider":    false,
				"referencesProvider":         false,
				"renameProvider":             false,
				"documentFormattingProvider": false,
			},
			"serverInfo": map[string]interface{}{"name": "qklsp", "version": version},
		})
	case "initialized":
		return nil
	case "shutdown":
		return s.conn.reply(msg.ID, nil)
	case "exit":
		s.quit = true
		return nil
	case "textDocument/didOpen":
		var p struct {
			TextDocument struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return nil
		}
		s.docs[p.TextDocument.URI] = newDocument(p.TextDocument.URI, p.TextDocument.Text)
		return s.publish(p.TextDocument.URI)
	case "textDocument/didChange":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return nil
		}
		d, ok := s.docs[p.TextDocument.URI]
		if !ok {
			return nil
		}
		if len(p.ContentChanges) > 0 {
			d.setText(p.ContentChanges[len(p.ContentChanges)-1].Text) // full sync
		}
		return s.publish(p.TextDocument.URI)
	case "textDocument/didClose":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return nil
		}
		delete(s.docs, p.TextDocument.URI)
		return s.conn.notify("textDocument/publishDiagnostics",
			map[string]interface{}{"uri": p.TextDocument.URI, "diagnostics": []interface{}{}})
	case "textDocument/definition":
		return s.definition(msg)
	case "textDocument/completion":
		return s.completion(msg)
	case "textDocument/hover":
		return s.hover(msg)
	case "textDocument/documentSymbol":
		return s.documentSymbol(msg)
	}
	// Unimplemented methods: requests reply null (error code -32601), notifications are ignored
	if msg.isRequest() {
		return s.conn.replyErr(msg.ID, -32601, "qklsp: 未实现的方法 %s", msg.Method)
	}
	return nil
}

// textDocParams parses {textDocument:{uri}, position:{line,character}}.
type positionParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position lspPosition `json:"position"`
}

func (s *server) docAt(msg rpcMessage) (*Document, lspPosition, error) {
	var p positionParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return nil, lspPosition{}, err
	}
	d, ok := s.docs[p.TextDocument.URI]
	if !ok {
		return nil, p.Position, errors.New(i18n.T("未打开的文档 %s", p.TextDocument.URI))
	}
	return d, p.Position, nil
}

func (s *server) definition(msg rpcMessage) error {
	d, pos, err := s.docAt(msg)
	if err != nil {
		return s.conn.reply(msg.ID, nil)
	}
	line := pos.Line + 1
	col := utf16ToByteCol(d.lineText(pos.Line), pos.Character)
	name := d.wordAt(line, col)
	if name == "" {
		return s.conn.reply(msg.ID, nil)
	}
	sym := d.lookup(name, line)
	if sym == nil {
		return s.conn.reply(msg.ID, nil)
	}
	return s.conn.reply(msg.ID, lspLocation{URI: d.URI, Range: rangeAt(sym.Line, sym.Col, sym.Col+len(name))})
}

func (s *server) completion(msg rpcMessage) error {
	d, pos, err := s.docAt(msg)
	if err != nil {
		return s.conn.reply(msg.ID, map[string]interface{}{"isIncomplete": false, "items": []interface{}{}})
	}
	line := pos.Line + 1
	items := []map[string]interface{}{}
	for _, sym := range d.completionCandidates(line) {
		items = append(items, map[string]interface{}{
			"label":  sym.Name,
			"kind":   completionKind(sym.Kind),
			"detail": completionDetail(sym),
		})
	}
	return s.conn.reply(msg.ID, map[string]interface{}{"isIncomplete": false, "items": items})
}

func completionKind(kind string) int {
	switch kind {
	case "func":
		return 3 // Function
	case "local", "param":
		return 6 // Variable
	case "struct":
		return 22 // Struct
	case "interface":
		return 8 // Interface
	case "alias":
		return 25 // TypeParameter
	case "impl", "space":
		return 9 // Module
	case "library":
		return 9
	case "member":
		return 2 // Method
	case "keyword":
		return 14 // Keyword
	case "builtin":
		return 1 // Text
	}
	return 1
}

func completionDetail(sym symbol) string {
	switch {
	case sym.Sig != "" && sym.Kind == "func":
		return sym.Sig
	case sym.Sig != "":
		return sym.Sig
	}
	return sym.Kind
}

func (s *server) hover(msg rpcMessage) error {
	d, pos, err := s.docAt(msg)
	if err != nil {
		return s.conn.reply(msg.ID, nil)
	}
	line := pos.Line + 1
	col := utf16ToByteCol(d.lineText(pos.Line), pos.Character)
	name := d.wordAt(line, col)
	if name == "" {
		return s.conn.reply(msg.ID, nil)
	}
	sym := d.lookup(name, line)
	if sym == nil {
		return s.conn.reply(msg.ID, nil)
	}
	var b strings.Builder
	b.WriteString("```qk\n")
	if sym.Sig != "" {
		b.WriteString(sym.Sig)
	} else {
		b.WriteString(sym.Name)
	}
	b.WriteString("\n```")
	if sym.Doc != "" {
		b.WriteString("\n\n")
		b.WriteString(sym.Doc)
	}
	return s.conn.reply(msg.ID, map[string]interface{}{
		"contents": map[string]interface{}{"kind": "markdown", "value": b.String()},
		"range":    rangeAt(sym.Line, sym.Col, sym.Col+len(name)),
	})
}

func (s *server) documentSymbol(msg rpcMessage) error {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return s.conn.reply(msg.ID, []interface{}{})
	}
	d, ok := s.docs[p.TextDocument.URI]
	if !ok || d.doc == nil {
		return s.conn.reply(msg.ID, []interface{}{})
	}
	items := []map[string]interface{}{}
	for _, it := range d.doc.All() {
		rng := rangeAt(it.Pos.Line, it.Pos.Col, it.Pos.Col+len(it.Name))
		items = append(items, map[string]interface{}{
			"name":           it.Name,
			"kind":           completionKind(it.Kind),
			"range":          rng,
			"selectionRange": rng,
		})
	}
	return s.conn.reply(msg.ID, items)
}

// publish sends textDocument/publishDiagnostics (for this file plus the other affected files).
func (s *server) publish(uri string) error {
	d, ok := s.docs[uri]
	if !ok {
		return nil
	}
	for file, diags := range d.diagnostics(s.libDirs) {
		target := uri
		if file != "" && file != d.Path {
			target = pathToURI(file)
		}
		if err := s.conn.notify("textDocument/publishDiagnostics", map[string]interface{}{
			"uri": target, "diagnostics": diags,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ---------- Diagnostics ----------

type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"` // 1=error 2=warning 3=info
	Code     string   `json:"code,omitempty"`
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

var lineSuffix = regexp.MustCompile(` at line \d+$`)

// diagnostics returns "file -> diagnostics". Positions are already mapped back to the real file (import merge coordinates -> source file).
func (d *Document) diagnostics(libDirs []string) map[string][]lspDiagnostic {
	out := map[string][]lspDiagnostic{}
	if d.parseErr != nil {
		line := d.parseErrLn
		if line < 1 {
			line = 1
		}
		out[d.Path] = append(out[d.Path], lspDiagnostic{
			Range: d.lineRange(line), Severity: 1, Source: "qklsp",
			Message: strings.TrimSpace(lineSuffix.ReplaceAllString(d.parseErr.Error(), "")),
		})
		return out
	}
	for _, diag := range lang.Lint(d.prog, lang.LintOptions{}) {
		line := diag.Pos.Line
		if line < 1 {
			line = 1
		}
		out[d.Path] = append(out[d.Path], lspDiagnostic{
			Range: d.lineRange(line), Severity: 2, Code: diag.Code, Source: "qkcheck", Message: diag.Msg,
		})
	}
	// Type checking
	// No imports (the common case): check the same already-parsed AST directly, saving one full lex+parse;
	// With imports: the merged source must be compiled (library symbols come from the merged source); error positions are mapped back to the real file via SrcMap.
	if d.prog != nil && len(d.prog.Imports) == 0 {
		if terr := lang.Typecheck(d.prog); terr != nil {
			line, _ := errLineCol(terr)
			if line < 1 {
				line = 1
			}
			out[d.Path] = append(out[d.Path], lspDiagnostic{
				Range: d.lineRange(line), Severity: 1, Source: "qkc",
				Message: strings.TrimSpace(lineSuffix.ReplaceAllString(terr.Error(), "")),
			})
		}
		if len(out) == 0 {
			out[d.Path] = []lspDiagnostic{}
		}
		return out
	}
	if _, sm, err := lang.CompileWithImportPaths(d.Text, d.Path, libDirs); err != nil {
		line, col := errLineCol(err)
		file, fline := d.Path, line
		if sm != nil {
			if f, l := sm.Map(line); f != "" {
				file, fline = f, l
			}
		}
		if fline < 1 {
			fline = 1
		}
		msg := strings.TrimSpace(lineSuffix.ReplaceAllString(err.Error(), ""))
		diag := lspDiagnostic{Range: d.lineRange(fline), Severity: 1, Source: "qkc", Message: msg}
		if file != d.Path && col > 0 {
			diag.Range = lspRange{
				Start: lspPosition{Line: fline - 1, Character: col - 1},
				End:   lspPosition{Line: fline - 1, Character: col},
			}
		}
		out[file] = append(out[file], diag)
	}
	if len(out) == 0 {
		out[d.Path] = []lspDiagnostic{} // empty array = clear the old diagnostics
	}
	return out
}

// lineText returns the whole line text for a 0-based line number.
func (d *Document) lineText(line0 int) string {
	if line0 < 0 || line0 >= len(d.Lines) {
		return ""
	}
	return d.Lines[line0]
}
