# tree-sitter-quarklang

The [tree-sitter](https://tree-sitter.github.io/tree-sitter/) grammar for QuarkLang (a toolchain member).
Canonical syntax lives in the main repo's `SYNTAX.md`; this directory is editor tooling, not the
language specification.

## Source and generated files

`grammar.js` is the source; `src/parser.c`, `src/grammar.json`, `src/node-types.json` and
`src/tree_sitter/*.h` are generated from it by the tree-sitter CLI. The generated files are committed so
editor integrations and other tree-sitter consumers can build this directory without Node or the CLI.
**Do not edit anything under `src/` by hand.**

The checked-in parser was generated with **tree-sitter 0.26.11**. Regenerate with exactly that version:

```sh
cd editors/tree-sitter-quarklang
tree-sitter --version              # must print "tree-sitter 0.26.11"
tree-sitter generate               # writes src/; warning-free
tree-sitter test -p .              # test/corpus/*.txt: 5/5
git diff --exit-code -- src        # generation is deterministic: no diff
```

If you change `grammar.js` or `test/corpus/*.txt`, regenerate `src/` and commit it in the same change.
CI runs the same generate + test step when the CLI is installed (and fails when `src/` is stale), so a
grammar change without its regenerated parser cannot land silently.

Why it lives in `editors/`: the grammar ships with the language's editor support, next to the VS Code
extension, and is exercised by the same CI job. Splitting it into its own repository would separate
grammar changes from the syntax reference they follow and add a release/versioning surface for no gain.

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
  `expand interface`), `impl`/`space`/`library`/`type` aliases, `program`/`import`/`pub`, and `#macro` bodies.
- Statements: `if/else`, `while`, C-style `for`, iteration `for`, `try/catch`, `return`, `log`, `delete`, `break`, expression statements.
- Expressions: assignment, binary/unary operators (C precedence), calls (including `@signature` calls), member/index access, `space::fn`,
  `.{...}` literals, `[..]` lists, `new T` and `new T[size]`, macro calls `name{...}`.
- Types: built-in types, `T&`, bare `pointer` and `pointer <built-in type>`, `List<T>`/`HashTable<K,V>`, anonymous `struct {…}`/`interface{…}`, `interface{}`.

**Historical measurement (tree-sitter 0.26.11, at the time of writing)**: the main repo plus 6 official
library repos total **60 `.qk/.kq` files with 0 ERROR/MISSING nodes** (including the 1,567-line
`style.qk`, `regex.qk` and `cleg.qk`); `tree-sitter test -p .` passes 5/5. The number is a record of
that corpus, not a gate for every file added since; the reproducible gate is `tree-sitter test -p .`.

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
- `conflicts` keeps only the one necessary entry; `tree-sitter generate` is warning-free.

## Files

| Path | Notes |
|---|---|
| `grammar.js` | **source** — grammar definition; edit this |
| `test/corpus/*.txt` | **source** — grammar tests (declarations/statements/expressions/macros/minimal program) |
| `src/` | **generated** by `tree-sitter generate` (parser.c, grammar.json, node-types.json, tree_sitter headers); never edit by hand |
| `tree-sitter.json` | language metadata (scope `source.quarklang`, file extensions `qk`/`kq`, query paths) |
| `queries/highlights.scm` | syntax highlighting queries |
| `queries/locals.scm` | scope/definition/reference queries (for renaming and variable highlighting) |
