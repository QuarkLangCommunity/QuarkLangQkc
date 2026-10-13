package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// frame encodes a message into an LSP frame.
func frame(t *testing.T, v interface{}) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
}

func request(id int, method string, params interface{}) map[string]interface{} {
	m := map[string]interface{}{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		m["params"] = params
	}
	return m
}

func notification(method string, params interface{}) map[string]interface{} {
	m := map[string]interface{}{"jsonrpc": "2.0", "method": method}
	if params != nil {
		m["params"] = params
	}
	return m
}

// runSession runs one full session and returns the output messages in order.
func runSession(t *testing.T, msgs ...map[string]interface{}) []rpcMessage {
	t.Helper()
	var in strings.Builder
	for _, m := range msgs {
		in.WriteString(frame(t, m))
	}
	var out bytes.Buffer
	s := &server{conn: newRPCConn(strings.NewReader(in.String()), &out), docs: map[string]*Document{}}
	if err := s.serve(); err != nil {
		t.Fatalf("serve: %v", err)
	}
	rd := newRPCConn(bytes.NewReader(out.Bytes()), io.Discard)
	var got []rpcMessage
	for {
		m, err := rd.read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parsing the output frame failed: %v", err)
		}
		got = append(got, m)
	}
	return got
}

func docURI(name string) string { return "file:///tmp/qklsp-test/" + name }

func didOpen(uri, text string) map[string]interface{} {
	return notification("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri, "text": text, "languageId": "quarklang", "version": 1},
	})
}

func posParams(uri string, line, char int) map[string]interface{} {
	return map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
		"position":     map[string]interface{}{"line": line, "character": char},
	}
}

// resultOf looks up the result field of the response with the given request id and decodes it into v (the id is the only reliable matching key).
func resultOf(t *testing.T, msgs []rpcMessage, id int, v interface{}) bool {
	t.Helper()
	want := fmt.Sprintf("%d", id)
	for _, m := range msgs {
		if m.ID == nil || strings.TrimSpace(string(*m.ID)) != want {
			continue
		}
		b, _ := json.Marshal(m)
		var raw struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatalf("decoding the response failed: %v", err)
		}
		if raw.Result == nil {
			return false
		}
		if err := json.Unmarshal(raw.Result, v); err != nil {
			t.Fatalf("decoding the result failed: %v", err)
		}
		return true
	}
	return false
}

func TestLSPInitializeCapabilities(t *testing.T) {
	msgs := runSession(t,
		request(1, "initialize", map[string]interface{}{"processId": nil, "rootUri": nil, "capabilities": map[string]interface{}{}}),
		notification("initialized", map[string]interface{}{}),
		request(2, "shutdown", nil),
		notification("exit", nil),
	)
	var res struct {
		Capabilities map[string]interface{} `json:"capabilities"`
		ServerInfo   map[string]interface{} `json:"serverInfo"`
	}
	if !resultOf(t, msgs, 1, &res) {
		t.Fatalf("no initialize response arrived: %+v", msgs)
	}
	for _, cap := range []string{"definitionProvider", "completionProvider", "hoverProvider", "documentSymbolProvider"} {
		if res.Capabilities[cap] == nil {
			t.Errorf("missing capability %s", cap)
		}
	}
	if res.ServerInfo["name"] != "qklsp" {
		t.Errorf("wrong serverInfo: %+v", res.ServerInfo)
	}
}

func TestLSPDiagnosticsOnOpen(t *testing.T) {
	uri := docURI("a.qk")
	src := "fn main(IOStream io) {\n    int unused = 1;\n    int bad = \"s\";\n    io.println(bad);\n}\n"
	msgs := runSession(t,
		request(1, "initialize", map[string]interface{}{"capabilities": map[string]interface{}{}}),
		didOpen(uri, src),
		request(2, "shutdown", nil),
		notification("exit", nil),
	)
	var found bool
	for _, m := range msgs {
		if m.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p struct {
			URI         string          `json:"uri"`
			Diagnostics []lspDiagnostic `json:"diagnostics"`
		}
		if err := json.Unmarshal(m.Params, &p); err != nil {
			t.Fatal(err)
		}
		if p.URI != uri {
			continue
		}
		found = true
		codes := map[string]bool{}
		for _, d := range p.Diagnostics {
			codes[d.Code] = true
			if d.Message == "" {
				t.Errorf("the diagnostic carries no message: %+v", d)
			}
		}
		if !codes["QK101"] {
			t.Errorf("an unused-variable diagnostic QK101 was expected, got %+v", p.Diagnostics)
		}
		var hasTypeErr bool
		for _, d := range p.Diagnostics {
			if d.Severity == 1 {
				hasTypeErr = true
			}
		}
		if !hasTypeErr {
			t.Errorf("a type error (severity=1) was expected, got %+v", p.Diagnostics)
		}
	}
	if !found {
		t.Fatalf("no diagnostics notification for this file arrived: %+v", msgs)
	}
}

func TestLSPDiagnosticsClean(t *testing.T) {
	uri := docURI("ok.qk")
	src := "fn main(IOStream io) {\n    int x = 1;\n    io.println(x);\n}\n"
	msgs := runSession(t, didOpen(uri, src), notification("exit", nil))
	for _, m := range msgs {
		if m.Method == "textDocument/publishDiagnostics" {
			var p struct {
				Diagnostics []lspDiagnostic `json:"diagnostics"`
			}
			if err := json.Unmarshal(m.Params, &p); err != nil {
				t.Fatal(err)
			}
			if len(p.Diagnostics) != 0 {
				t.Errorf("a clean file should have no diagnostics, got %+v", p.Diagnostics)
			}
			return
		}
	}
	t.Fatal("no diagnostics notification arrived")
}

func TestLSPDidChangeUpdatesDiagnostics(t *testing.T) {
	uri := docURI("b.qk")
	bad := "fn main(IOStream io) {\n    int unused = 1;\n    io.println(\"x\");\n}\n"
	good := "fn main(IOStream io) {\n    io.println(\"x\");\n}\n"
	msgs := runSession(t,
		didOpen(uri, bad),
		notification("textDocument/didChange", map[string]interface{}{
			"textDocument":   map[string]interface{}{"uri": uri, "version": 2},
			"contentChanges": []map[string]interface{}{{"text": good}},
		}),
		notification("exit", nil),
	)
	var last int
	for _, m := range msgs {
		if m.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p struct {
			Diagnostics []lspDiagnostic `json:"diagnostics"`
		}
		if err := json.Unmarshal(m.Params, &p); err != nil {
			t.Fatal(err)
		}
		last = len(p.Diagnostics)
	}
	if last != 0 {
		t.Errorf("the diagnostics should be cleared after the edit; the last notification carried %d", last)
	}
}

func TestLSPDefinitionAndHover(t *testing.T) {
	uri := docURI("def.qk")
	src := "// 翻倍。\nfn double(int n) int {\n    return n * 2;\n}\nfn main(IOStream io) {\n    io.println(double(21));\n}\n"
	msgs := runSession(t,
		request(1, "initialize", map[string]interface{}{"capabilities": map[string]interface{}{}}),
		didOpen(uri, src),
		request(2, "textDocument/definition", posParams(uri, 5, 17)), // the `double` of `double(21)` on line 6
		request(3, "textDocument/hover", posParams(uri, 5, 17)),
		request(4, "shutdown", nil),
		notification("exit", nil),
	)
	var loc lspLocation
	if !resultOf(t, msgs, 2, &loc) {
		t.Fatalf("no definition response arrived: %+v", msgs)
	}
	if loc.URI != uri {
		t.Errorf("the definition should be in the same file, got %s", loc.URI)
	}
	if loc.Range.Start.Line != 1 { // double is declared on line 2 (0-based = 1)
		t.Errorf("the definition should point at line 2, got %+v", loc.Range)
	}
	var hov struct {
		Contents struct {
			Kind  string `json:"kind"`
			Value string `json:"value"`
		} `json:"contents"`
	}
	if !resultOf(t, msgs, 3, &hov) {
		t.Fatal("no hover response arrived")
	}
	if !strings.Contains(hov.Contents.Value, "fn double(int n) int") {
		t.Errorf("hover should carry the signature, got %q", hov.Contents.Value)
	}
	if !strings.Contains(hov.Contents.Value, "翻倍。") {
		t.Errorf("hover should carry the doc comment, got %q", hov.Contents.Value)
	}
}

// UTF-16 column conversion: when multi-byte Chinese text precedes the identifier, the definition position must still be correct.
func TestLSPUTF16PositionWithCJK(t *testing.T) {
	uri := docURI("cjk.qk")
	src := "fn helper(int n) int {\n    return n;\n}\nfn main(IOStream io) {\n    io.println(\"中文前缀\" , helper(1));\n}\n"
	line := "    io.println(\"中文前缀\" , helper(1));"
	char := len([]rune(line[:strings.Index(line, "helper")])) // rune count != UTF-16 code unit count (Chinese is 1:1, only emoji is 1:2)
	msgs := runSession(t,
		didOpen(uri, src),
		request(2, "textDocument/definition", posParams(uri, 4, char)),
		notification("exit", nil),
	)
	var loc lspLocation
	if !resultOf(t, msgs, 2, &loc) {
		t.Fatalf("no definition response arrived: %+v", msgs)
	}
	if loc.Range.Start.Line != 0 {
		t.Errorf("the jump should land on the helper declaration (line 1, 0-based 0), got %+v", loc.Range)
	}
}

func TestLSPCompletion(t *testing.T) {
	uri := docURI("comp.qk")
	src := "fn double(int n) int {\n    return n * 2;\n}\ntype struct {\n    int v;\n} Point;\nfn main(IOStream io) {\n    int local = 1;\n    io.println(local);\n}\n"
	msgs := runSession(t,
		didOpen(uri, src),
		request(1, "textDocument/completion", posParams(uri, 7, 4)),
		notification("exit", nil),
	)
	var res struct {
		IsIncomplete bool `json:"isIncomplete"`
		Items        []struct {
			Label  string `json:"label"`
			Kind   int    `json:"kind"`
			Detail string `json:"detail"`
		} `json:"items"`
	}
	if !resultOf(t, msgs, 1, &res) {
		t.Fatalf("no completion response arrived: %+v", msgs)
	}
	labels := map[string]bool{}
	for _, it := range res.Items {
		labels[it.Label] = true
		if it.Label == "double" && it.Detail == "" {
			t.Errorf("function candidates should carry the signature as detail")
		}
	}
	for _, want := range []string{"double", "Point", "local", "io", "fn", "while", "List"} {
		if !labels[want] {
			t.Errorf("the completion list is missing %q (%d items in total)", want, len(res.Items))
		}
	}
}

func TestLSPDocumentSymbol(t *testing.T) {
	uri := docURI("sym.qk")
	src := "fn alpha(int n) int {\n    return n;\n}\ntype struct {\n    int v;\n} Beta;\nfn main(IOStream io) {\n    io.println(alpha(1));\n}\n"
	msgs := runSession(t,
		didOpen(uri, src),
		request(1, "textDocument/documentSymbol", map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
		}),
		notification("exit", nil),
	)
	var syms []struct {
		Name string `json:"name"`
		Kind int    `json:"kind"`
	}
	if !resultOf(t, msgs, 1, &syms) {
		t.Fatalf("no documentSymbol response arrived: %+v", msgs)
	}
	names := map[string]bool{}
	for _, s := range syms {
		names[s.Name] = true
	}
	for _, want := range []string{"alpha", "Beta", "main"} {
		if !names[want] {
			t.Errorf("the symbol table is missing %q: %+v", want, syms)
		}
	}
}

func TestLSPUnimplementedMethod(t *testing.T) {
	msgs := runSession(t,
		request(1, "textDocument/references", map[string]interface{}{}),
		notification("exit", nil),
	)
	for _, m := range msgs {
		if m.Error != nil {
			if m.Error.Code != -32601 {
				t.Errorf("the error code should be -32601, got %d", m.Error.Code)
			}
			return
		}
	}
	t.Fatalf("an unimplemented error was expected: %+v", msgs)
}

func TestLSPUnknownDocument(t *testing.T) {
	msgs := runSession(t,
		request(1, "textDocument/definition", posParams(docURI("missing.qk"), 0, 0)),
		notification("exit", nil),
	)
	if len(msgs) == 0 {
		t.Fatal("it should return a null result instead of crashing")
	}
}

// Diagnostic position conversion: real import merge coordinates -> this file's line numbers.
func TestLSPDiagnosticsMapImports(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "lib.qk")
	// use the production pathToURI: a Windows drive needs file:///C:/... (three slashes); a hand-made URI in the test was once hard-coded for POSIX
	uri := pathToURI(filepath.Join(dir, "app.qk"))
	src := "import \"lib\";\n\nfn main(IOStream io) {\n    io.println(lib());\n}\n"
	if err := writeFile(libPath, "program library;\npub fn lib() int {\n    return missingVar;\n}\n"); err != nil {
		t.Fatal(err)
	}
	msgs := runSession(t,
		didOpen(uri, src),
		notification("exit", nil),
	)
	foundLibURI := false
	for _, m := range msgs {
		if m.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p struct {
			URI         string          `json:"uri"`
			Diagnostics []lspDiagnostic `json:"diagnostics"`
		}
		if err := json.Unmarshal(m.Params, &p); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(p.URI, "lib.qk") && len(p.Diagnostics) > 0 {
			foundLibURI = true
			if p.Diagnostics[0].Range.Start.Line != 2 { // line 3 of lib.qk
				t.Errorf("the cross-file diagnostic should point at lib.qk line 3, got %+v", p.Diagnostics[0].Range)
			}
		}
	}
	if !foundLibURI {
		t.Error("an error in an imported file should be published under that file's URI")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
