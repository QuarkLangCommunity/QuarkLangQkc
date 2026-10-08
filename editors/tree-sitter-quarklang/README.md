# tree-sitter-quarklang

**English** · [简体中文](README.zh-CN.md)

The [tree-sitter](https://tree-sitter.github.io/tree-sitter/) grammar for QuarkLang (a toolchain member).
Canonical syntax lives in the main repo's `SYNTAX.md`.

## Usage

```sh
# Generate the parser (requires the tree-sitter CLI; the CLI invokes the C compiler itself)
tree-sitter generate

# Run the grammar tests (test/corpus/*.txt)
tree-sitter test -p .

# Parse + highlight
tree-sitter parse path/to/file.qk
tree-sitter highlight -p . --scope source.quarklang path/to/file.qk
```

Editor integration:

| Editor | How |
|---|---|
| Neovim | point `nvim-treesitter.parsers.quarklang` at this directory; `queries/highlights.scm` and `queries/locals.scm` are loaded automatically |
| Helix | `[[language]] name = "quarklang" grammar = "quarklang"` in `languages.toml` + a local grammar directory |
| Emacs | point `treesit-language-source-alist` at this directory (`tree-sitter.json` already carries the binding information) |
| VS Code | install the `editors/vscode` extension (TextMate highlighting + qklsp); the tree-sitter grammar is there for other tools to reuse |

## Coverage and measurements

- The grammar covers every declaration form of the canonical syntax: `fn` (including generics/overloads), `type struct`/`type interface` (including `dynamic` methods and
  `expand interface`), `impl`/`space`/`library`/`type` aliases, `program`/`import`/`pub`, and `macro name($a $b)` definitions.
- Statements: `if/else`, `while`, C-style `for`, iteration `for`, `try/catch`, `return`, `log`, `delete`, `break`, expression statements.
- Expressions: assignment, binary/unary operators (C precedence), calls (including `@signature` calls), member/index access, `space::fn`,
  `.{...}` literals, `[..]` lists, `new T` and `new T[size]`, macro calls `name{...}`.
- Types: built-in types, `T&`, bare `pointer` and `pointer <built-in type>`, `List<T>`/`HashTable<K,V>`, anonymous `struct {…}`/`interface{…}`, `interface{}`.

**Measured (tree-sitter 0.26.11 on this machine)**: the main repo plus 6 official library repos total **60 `.qk/.kq` files with 0 ERROR/MISSING nodes**
(including the 1,567-line `style.qk`, `regex.qk` and `cleg.qk`); `tree-sitter test -p .` passes 5/5.

```
$ tree-sitter parse $(every .qk file in the main repo and the library repos)   # check ERROR/MISSING one by one → 0
$ tree-sitter test -p .
Total parses: 5; successful parses: 5; ... success percentage: 100.00%
```

## Design notes (why it is written this way)

- **Suffix-chain shape**: `a.b(c)[d]` is carried recursively by `primary`, avoiding the static resolution problem of "reduce to expression first, then decide whether to consume `[`".
- **Assignment is an expression**: `x = 1` / `l[i] = v` / `*p = v` all go through `assignment_expression`, avoiding static conflicts with declaration statements.
- **`new T[size]` uses `token.immediate('[')`**: separates "sized heap allocation" from index syntax at the lexer level.
- **Two spellings of `pointer`**: bare `pointer` (an opaque FFI handle, common in the corpus) and `pointer <built-in type>`;
  a nullable reference to a custom type is written `T&` by corpus convention (so there is no static resolution between `pointer p = ...` and `pointer T p`).
- **Macro call `name[a, b]`**: shaped like an index, it parses as index syntax (`index_expression` allows several indices), which is fine for highlighting.
- `conflicts` keeps only the necessary entries (`tree-sitter generate` no longer emits unnecessary warnings).

## Files

| Path | Notes |
|---|---|
| `grammar.js` | grammar definition |
| `tree-sitter.json` | language metadata (scope `source.quarklang`, file extensions `qk`/`kq`, query paths) |
| `queries/highlights.scm` | syntax highlighting queries |
| `queries/locals.scm` | scope/definition/reference queries (for renaming and variable highlighting) |
| `test/corpus/*.txt` | grammar test cases (declarations/statements/expressions/macros/minimal program) |
| `src/` | the generated parser (produced by `tree-sitter generate`) |
