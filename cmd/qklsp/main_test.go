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

// frame 把消息编成 LSP 帧。
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

// runSession 跑一次完整会话，返回按顺序的输出消息。
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
			t.Fatalf("解析输出帧失败: %v", err)
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

// resultOf 按请求 id 取响应的 result 字段并解码到 v（id 是唯一可靠的匹配键）。
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
			t.Fatalf("解码响应失败: %v", err)
		}
		if raw.Result == nil {
			return false
		}
		if err := json.Unmarshal(raw.Result, v); err != nil {
			t.Fatalf("解码 result 失败: %v", err)
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
		t.Fatalf("未收到 initialize 响应: %+v", msgs)
	}
	for _, cap := range []string{"definitionProvider", "completionProvider", "hoverProvider", "documentSymbolProvider"} {
		if res.Capabilities[cap] == nil {
			t.Errorf("缺少能力 %s", cap)
		}
	}
	if res.ServerInfo["name"] != "qklsp" {
		t.Errorf("serverInfo 不对: %+v", res.ServerInfo)
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
				t.Errorf("诊断缺少消息: %+v", d)
			}
		}
		if !codes["QK101"] {
			t.Errorf("应有未使用变量诊断 QK101，got %+v", p.Diagnostics)
		}
		var hasTypeErr bool
		for _, d := range p.Diagnostics {
			if d.Severity == 1 {
				hasTypeErr = true
			}
		}
		if !hasTypeErr {
			t.Errorf("应有类型错误（severity=1），got %+v", p.Diagnostics)
		}
	}
	if !found {
		t.Fatalf("未收到本文件的诊断通知: %+v", msgs)
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
				t.Errorf("干净文件不应有诊断，got %+v", p.Diagnostics)
			}
			return
		}
	}
	t.Fatal("未收到诊断通知")
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
		t.Errorf("修改后应清空诊断，最后一次收到 %d 条", last)
	}
}

func TestLSPDefinitionAndHover(t *testing.T) {
	uri := docURI("def.qk")
	src := "// 翻倍。\nfn double(int n) int {\n    return n * 2;\n}\nfn main(IOStream io) {\n    io.println(double(21));\n}\n"
	msgs := runSession(t,
		request(1, "initialize", map[string]interface{}{"capabilities": map[string]interface{}{}}),
		didOpen(uri, src),
		request(2, "textDocument/definition", posParams(uri, 5, 17)), // 第 6 行 `double(21)` 的 double
		request(3, "textDocument/hover", posParams(uri, 5, 17)),
		request(4, "shutdown", nil),
		notification("exit", nil),
	)
	var loc lspLocation
	if !resultOf(t, msgs, 2, &loc) {
		t.Fatalf("未收到 definition 响应: %+v", msgs)
	}
	if loc.URI != uri {
		t.Errorf("定义应在同一文件，got %s", loc.URI)
	}
	if loc.Range.Start.Line != 1 { // double 声明在第 2 行（0 基 = 1）
		t.Errorf("定义应指向第 2 行，got %+v", loc.Range)
	}
	var hov struct {
		Contents struct {
			Kind  string `json:"kind"`
			Value string `json:"value"`
		} `json:"contents"`
	}
	if !resultOf(t, msgs, 3, &hov) {
		t.Fatal("未收到 hover 响应")
	}
	if !strings.Contains(hov.Contents.Value, "fn double(int n) int") {
		t.Errorf("hover 应含签名，got %q", hov.Contents.Value)
	}
	if !strings.Contains(hov.Contents.Value, "翻倍。") {
		t.Errorf("hover 应含文档注释，got %q", hov.Contents.Value)
	}
}

// UTF-16 列换算：标识符前有中文（多字节）时，跳转位置必须仍然正确。
func TestLSPUTF16PositionWithCJK(t *testing.T) {
	uri := docURI("cjk.qk")
	src := "fn helper(int n) int {\n    return n;\n}\nfn main(IOStream io) {\n    io.println(\"中文前缀\" , helper(1));\n}\n"
	line := "    io.println(\"中文前缀\" , helper(1));"
	char := len([]rune(line[:strings.Index(line, "helper")])) // rune 数 ≠ UTF-16 单位数（中文 1:1，emoji 才 1:2）
	msgs := runSession(t,
		didOpen(uri, src),
		request(2, "textDocument/definition", posParams(uri, 4, char)),
		notification("exit", nil),
	)
	var loc lspLocation
	if !resultOf(t, msgs, 2, &loc) {
		t.Fatalf("未收到 definition 响应: %+v", msgs)
	}
	if loc.Range.Start.Line != 0 {
		t.Errorf("应跳到 helper 声明（第 1 行，0 基 0），got %+v", loc.Range)
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
		t.Fatalf("未收到 completion 响应: %+v", msgs)
	}
	labels := map[string]bool{}
	for _, it := range res.Items {
		labels[it.Label] = true
		if it.Label == "double" && it.Detail == "" {
			t.Errorf("函数候选应带签名 detail")
		}
	}
	for _, want := range []string{"double", "Point", "local", "io", "fn", "while", "List"} {
		if !labels[want] {
			t.Errorf("补全缺少 %q（共 %d 项）", want, len(res.Items))
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
		t.Fatalf("未收到 documentSymbol 响应: %+v", msgs)
	}
	names := map[string]bool{}
	for _, s := range syms {
		names[s.Name] = true
	}
	for _, want := range []string{"alpha", "Beta", "main"} {
		if !names[want] {
			t.Errorf("符号表缺少 %q：%+v", want, syms)
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
				t.Errorf("错误码应为 -32601，got %d", m.Error.Code)
			}
			return
		}
	}
	t.Fatalf("应返回未实现错误: %+v", msgs)
}

func TestLSPUnknownDocument(t *testing.T) {
	msgs := runSession(t,
		request(1, "textDocument/definition", posParams(docURI("missing.qk"), 0, 0)),
		notification("exit", nil),
	)
	if len(msgs) == 0 {
		t.Fatal("应返回 null 结果而不是崩溃")
	}
}

// 诊断位置换算：真实的 import 合并坐标 → 本文件行号。
func TestLSPDiagnosticsMapImports(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "lib.qk")
	uri := "file://" + filepath.ToSlash(filepath.Join(dir, "app.qk"))
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
			if p.Diagnostics[0].Range.Start.Line != 2 { // lib.qk 第 3 行
				t.Errorf("跨文件诊断应定位到 lib.qk 第 3 行，got %+v", p.Diagnostics[0].Range)
			}
		}
	}
	if !foundLibURI {
		t.Error("import 文件中的错误应发布到该文件的 URI")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
