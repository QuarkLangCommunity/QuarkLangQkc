// qklsp：QuarkLang 语言服务器（工具链成员）
//
// 用法: qklsp [-L dir]...  （stdio 上跑 LSP；由编辑器启动）
//
// 能力：诊断（解析/类型/静态检查）、跳转定义、补全、悬停、文档符号。
// 零第三方依赖：JSON-RPC 帧与协议子集自实现；语言分析复用 internal/lang
// （与 qkcheck / qkdoc 同一套前端，保证编辑器里的结果与命令行一致）。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"quarklang/internal/lang"
)

// version 发布版本：构建时用 -ldflags "-X main.version=vX.Y.Z" 注入（见 scripts/build-release.sh）
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
				return nil, fmt.Errorf("-L 需要一个目录参数")
			}
			i++
			libDirs = append(libDirs, args[i])
		case "--version", "-V":
			fmt.Println("qklsp", version)
			os.Exit(0)
		case "-h", "--help":
			fmt.Println("usage: qklsp [-L dir]...  （stdio 上提供 LSP 服务）")
			os.Exit(0)
		default:
			return nil, fmt.Errorf("未知参数 %s", args[i])
		}
	}
	return libDirs, nil
}

// serve 主循环：读消息 → 分发 → 回包/通知；`exit` 或 EOF 结束。
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
					"change":    1, // 全量同步
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
			d.setText(p.ContentChanges[len(p.ContentChanges)-1].Text) // 全量同步
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
	// 未实现的方法：请求回 null（错误码 -32601），通知忽略
	if msg.isRequest() {
		return s.conn.replyErr(msg.ID, -32601, "qklsp: 未实现的方法 %s", msg.Method)
	}
	return nil
}

// textDocParams 解析 {textDocument:{uri}, position:{line,character}}。
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
		return nil, p.Position, fmt.Errorf("未打开的文档 %s", p.TextDocument.URI)
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

// publish 发送 textDocument/publishDiagnostics（本文件 + 受影响的其它文件）。
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

// ---------- 诊断 ----------

type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"` // 1=错误 2=警告 3=信息
	Code     string   `json:"code,omitempty"`
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

var lineSuffix = regexp.MustCompile(` at line \d+$`)

// diagnostics 返回「文件 → 诊断」。位置已映射回真实文件（import 合并坐标 → 源文件）。
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
	// 类型检查（含 import 合并）：错误位置经 SrcMap 映射回真实文件
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
		out[d.Path] = []lspDiagnostic{} // 空数组 = 清空旧诊断
	}
	return out
}

// lineText 取 0 基行号的整行文本。
func (d *Document) lineText(line0 int) string {
	if line0 < 0 || line0 >= len(d.Lines) {
		return ""
	}
	return d.Lines[line0]
}
