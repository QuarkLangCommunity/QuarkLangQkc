# tree-sitter-quarklang

QuarkLang 的 [tree-sitter](https://tree-sitter.github.io/tree-sitter/) 语法（工具链成员）。
正典语法见主仓 `SYNTAX.md`。

## 用法

```sh
# 生成解析器（需要 tree-sitter CLI，C 编译器由 CLI 自动调用）
tree-sitter generate

# 跑语法测试（test/corpus/*.txt）
tree-sitter test -p .

# 解析 + 高亮
tree-sitter parse path/to/file.qk
tree-sitter highlight -p . --scope source.quarklang path/to/file.qk
```

编辑器接入：

| 编辑器 | 方式 |
|---|---|
| Neovim | `nvim-treesitter.parsers.quarklang` 指向本目录；`queries/highlights.scm`、`queries/locals.scm` 会被自动加载 |
| Helix | `languages.toml` 里 `[[language]] name = "quarklang" grammar = "quarklang"` + 本地 grammar 目录 |
| Emacs | `treesit-language-source-alist` 指向本目录（`tree-sitter.json` 已含绑定信息） |
| VS Code | 装 `editors/vscode` 扩展（TextMate 高亮 + qklsp），tree-sitter 语法供其它工具复用 |

## 覆盖与实测

- 语法覆盖正典语法的全部声明形态：`fn`（含泛型/重载）、`type struct`/`type interface`（含 `dynamic` 方法、
  `expand interface`）、`impl`/`space`/`library`/`type` 别名、`program`/`import`/`pub`、`#macro` 宏体。
- 语句：`if/else`、`while`、C 风格 `for`、迭代 `for`、`try/catch`、`return`、`log`、`delete`、`break`、表达式语句。
- 表达式：赋值、二元/一元运算（按 C 优先级）、调用（含 `@签名` 调用）、成员/下标、`space::fn`、
  `.{...}` 字面量、`[..]` 列表、`new T` 与 `new T[size]`、宏调用 `name{...}`。
- 类型：内置类型、`T&`、裸 `pointer` 与 `pointer <内置类型>`、`List<T>`/`HashTable<K,V>`、匿名 `struct {…}`/`interface{…}`、`interface{}`。

**实测（本机 tree-sitter 0.26.11）**：主仓 + 6 个官方库仓共 **60 个 `.qk/.kq` 文件、0 个 ERROR/MISSING 节点**
（含 `style.qk` 1567 行、`regex.qk`、`cleg.qk`）；`tree-sitter test -p .` 5/5 通过。

```
$ tree-sitter parse $(主仓与库仓全部 .qk)   # 逐个检查 ERROR/MISSING → 0
$ tree-sitter test -p .
Total parses: 5; successful parses: 5; ... success percentage: 100.00%
```

## 设计说明（为什么这么写）

- **后缀链形态**：`a.b(c)[d]` 由 `primary` 递归承载，避免「先归约成 expression 再决定是否吃 `[`」的静态裁决问题。
- **赋值是表达式**：`x = 1` / `l[i] = v` / `*p = v` 统一走 `assignment_expression`，避免与声明语句静态冲突。
- **`new T[size]` 用 `token.immediate('[')`**：把「带长度的堆申请」与下标语法在词法层分开。
- **`pointer` 的两种写法**：裸 `pointer`（FFI 不透明句柄，语料常见）与 `pointer <内置类型>`；
  指向自定义类型的可空引用按语料惯例写 `T&`（这样不必在 `pointer p = ...` 与 `pointer T p` 之间做静态裁决）。
- **宏调用 `name[a, b]`**：与下标同形，按下标语法解析（`index_expression` 允许多个下标），高亮无碍。
- `conflicts` 只保留必要项（`tree-sitter generate` 不再给出 unnecessary 告警）。

## 文件

| 路径 | 说明 |
|---|---|
| `grammar.js` | 语法定义 |
| `tree-sitter.json` | 语言元数据（scope `source.quarklang`、文件后缀 `qk`/`kq`、查询路径） |
| `queries/highlights.scm` | 语法高亮查询 |
| `queries/locals.scm` | 作用域/定义/引用查询（重命名、变量高亮用） |
| `test/corpus/*.txt` | 语法测试用例（声明/语句/表达式/宏/最小程序） |
| `src/` | 生成的解析器（由 `tree-sitter generate` 产出） |
