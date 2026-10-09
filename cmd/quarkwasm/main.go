//go:build js && wasm

// Command quarkwasm is the WebAssembly build of the QuarkLang interpreter: the engine behind the
// web playground (QuarkLangCommunity/QuarkLangPlayground). Same front end as the interpreter and
// the LLVM backend — a third backend, the browser.
//
// The JavaScript surface, installed on globalThis by main():
//
//	quarkVersion()                     -> string        engine version (ldflags -X main.version)
//	quarkLang()                        -> "en" | "zh"   diagnostics language in effect
//	quarkSetLang(code)                 -> bool          switch the diagnostics language
//	quarkCheck(source, filename)       -> JSON {ok,error}            front end only (fast, for live diagnostics)
//	quarkRun(source, stdin, filename)  -> JSON {ok,output,error,ms}  compile + run, stdout captured
//
// Run it inside a Web Worker: the interpreter has no instruction budget, so the worker is the
// watchdog for a runaway program (worker.terminate()).
//
// Build:
//
//	GOOS=js GOARCH=wasm go build -o quark.wasm ./cmd/quarkwasm
//	cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// version is injected at build time with -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	js.Global().Set("quarkVersion", js.FuncOf(func(js.Value, []js.Value) any { return version }))
	js.Global().Set("quarkLang", js.FuncOf(func(js.Value, []js.Value) any { return lang.Localizer().Lang().String() }))
	js.Global().Set("quarkSetLang", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return false
		}
		l, ok := i18n.ParseLang(args[0].String())
		if !ok {
			return false
		}
		lang.SetLocalizer(i18n.New(i18n.Std(), l))
		return true
	}))
	js.Global().Set("quarkCheck", js.FuncOf(check))
	js.Global().Set("quarkRun", js.FuncOf(run))
	select {} // stay alive: the page keeps calling into the exports above
}

// check runs the front end only (lex -> macros -> parse -> type check): the playground uses it for
// live diagnostics while typing, where a compile error is not an error yet.
func check(_ js.Value, args []js.Value) any {
	src, name := arg(args, 0), arg(args, 1)
	if name == "" {
		name = "main.qk"
	}
	if _, err := lang.CompileWithImports(src, name); err != nil {
		return result(map[string]any{"ok": false, "error": err.Error()})
	}
	return result(map[string]any{"ok": true})
}

// run compiles and executes, capturing stdout and the error report into strings.
func run(_ js.Value, args []js.Value) any {
	src, stdin, name := arg(args, 0), arg(args, 1), arg(args, 2)
	if name == "" {
		name = "main.qk"
	}
	start := time.Now()
	var out bytes.Buffer
	ok, errText := true, ""
	func() {
		defer func() {
			if r := recover(); r != nil { // a runtime panic is reported to the page, never fatal to it
				ok, errText = false, "internal error: "+fmt.Sprint(r)
			}
		}()
		prog, err := lang.CompileWithImports(src, name)
		if err != nil {
			ok, errText = false, err.Error()
			return
		}
		if err := lang.Run(prog, name, nil, strings.NewReader(stdin), &out); err != nil {
			ok = false
			var eb bytes.Buffer
			lang.ReportError(err, &eb)
			errText = strings.TrimRight(eb.String(), "\n")
		}
	}()
	m := map[string]any{"ok": ok, "output": out.String(), "ms": float64(time.Since(start).Microseconds()) / 1000}
	if errText != "" {
		m["error"] = errText
	}
	return result(m)
}

// arg reads an optional string argument passed from JavaScript.
func arg(args []js.Value, i int) string {
	if i >= len(args) || args[i].Type() != js.TypeString {
		return ""
	}
	return args[i].String()
}

// result encodes a reply as a JSON string (no struct tags: the wire format stays explicit).
func result(m map[string]any) string {
	b, err := json.Marshal(m)
	if err != nil {
		return "{\"ok\":false,\"error\":\"internal: could not encode the result\"}"
	}
	return string(b)
}
