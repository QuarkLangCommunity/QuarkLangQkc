# QuarkLang for VS Code

**English** · [简体中文](README.zh-CN.md)

QuarkLang language support extension (a toolchain member): **syntax highlighting** (TextMate) + **language service** (diagnostics / go-to-definition / completion / hover / outline).

## Install (local development)

```sh
# 1) Build the language server (repo root)
go build -o qklsp ./cmd/qklsp

# 2) Make the extension find it
export PATH="$PWD:$PATH"      # or install qklsp into PATH; or set it in the settings:
#   quarklang.serverPath = /absolute/path/qklsp

# 3) Load the extension in the development host
code --extensionDevelopmentPath=editors/vscode <some .qk project>
```

Package and install (requires `npx @vscode/vsce`, optional):

```sh
cd editors/vscode && npx @vscode/vsce package     # produces quarklang-0.1.0.vsix
code --install-extension quarklang-0.1.0.vsix
```

## Capabilities

| Capability | Provider | Notes |
|---|---|---|
| Syntax highlighting | `syntaxes/quarklang.tmLanguage.json` | comments/strings (including backtick raw strings)/numbers/keywords/built-in types/function names/macros `#…`/signature calls `@mb`/operators |
| Snippets | `snippets/quarklang.json` | `fnmain` `fn` `struct` `interface` `impl` `space` `library` `for` `forin` `try` `macro` |
| Diagnostics | `qklsp` | parse/type errors (`qkc`) + static analysis (`qkcheck`: unused variables, unreachable code, shadowing, missing return, unimplemented interface, void misuse) |
| Go to definition | `qklsp` | symbols within the same file (including locals/parameters/fields) |
| Completion | `qklsp` | keywords + built-in types + built-in methods + functions/types/spaces/locals declared in the current file |
| Hover | `qklsp` | signature + the doc comment above the declaration |
| Outline | `qklsp` | `documentSymbol`: functions/structs/interfaces/impls/spaces/system libraries |

## Settings

| Setting | Default | Notes |
|---|---|---|
| `quarklang.serverPath` | `qklsp` | language server executable (on PATH or an absolute path) |
| `quarklang.libDirs` | `[]` | extra import search directories (passed to `qklsp -L`; repeatable) |
| `quarklang.trace.server` | `false` | log request/response traffic to the "Output → QuarkLang" channel, for troubleshooting |

## Implementation notes (zero npm dependencies)

- `src/protocol.js`: JSON-RPC over stdio framing plus the request/notification lifecycle; it **does not import vscode**,
  so it can be exercised directly with Node (see below).
- `src/extension.js`: the VS Code glue layer (DiagnosticCollection, Definition/Completion/Hover/DocumentSymbol providers).
- No dependency on `vscode-languageclient`: the extension only needs the vscode API and Node built-in modules, so it works right after cloning.

## Test

```sh
go build -o /tmp/qklsp ./cmd/qklsp
QKLSP=/tmp/qklsp node editors/vscode/test/protocol.test.js
```

Measured output (all eight checks pass):

```
✓ initialize returns capabilities          ✓ didOpen receives diagnostics (including QK101 unused variable)
✓ diagnostics carry a range and a message  ✓ definition jumps to line 2
✓ completion includes double / io / while  ✓ hover shows signature and doc
✓ diagnostics cleared after didChange      ✓ server shuts down gracefully
```

> The tree-sitter grammar lives in `editors/tree-sitter-quarklang/` (usable directly from Neovim/Helix/Emacs).
