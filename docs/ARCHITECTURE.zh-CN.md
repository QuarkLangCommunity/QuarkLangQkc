# QuarkLang —— 接口清单与组合地图

[English](ARCHITECTURE.md) | 中文

本文基于 `67ce88c`（撰写时 `main` 的顶端）的代码分析。每一条断言都带 `file:line`
引用。既无法从源码核实、也无法通过运行门禁核实的断言，一律标注 **UNVERIFIED**——
文中没有"推测但未核对"的陈述。

本文把 QuarkLang 描述为一组**接口**以及消费这些接口的**组合**。全文不写"某项目红了"，
也不写"项目做了 X"。它写的是：

> 接口 `I` 承诺 `P`；在组合 `C` 下该承诺不成立。

仓库中两处说法互相矛盾时，两处都引用原文，并**指出矛盾而不擅自裁决**（§6），
由读者判断哪一边是错的。

---

## 0. 为什么用这个形状

一个语言实现不是一个程序，而是一叠接口：每个接口在某些组合下被验证、在另一些组合下无人验证。
真正有价值的工程信息恰好就在这种不对称里：深度上限在所有操作系统上都成立，因为它是显式计数器
而不是探测栈（`internal/lang/limits.go:9-13`）；而逐字节的引擎一致性只对已入库的语料成立，
并且只是**一个平台**才会跑的一个 CI 步骤（`.github/workflows/ci.yml:118-121`）。

全文两个约定：

| 记号 | 含义 |
|---|---|
| `I-k` | 清单（§1）里的一个接口 |
| `C-k` | 地图（§2）里的一个组合 |
| ✔ / ✘ / ⚠ | 承诺成立 / 不成立 / 仅在所述限制下成立 |

涉及的三个 Go 模块：

```
┌─────────────────────────────────────────────────────────────────────────┐
│ module github.com/QuarkLangCommunity/QuarkLangQkc            （根）     │
│   go.mod:5-10   require qkparser @ v0.0.0-20261010115438-20ae746ffdee   │
│   internal/lang        解释器、类型检查、linter、VM、文档、REPL          │
│   internal/i18n        目录 + 本地化器                                  │
│   cmd/{qkcheck,qkdoc,qkrepl,qklsp,qkstress,quarkwasm}                    │
├─────────────────────────────────────────────────────────────────────────┤
│ module .../QuarkLangQkc/compiler                     （子模块）          │
│   go.mod:7      replace .../QuarkLangQkc => ../                         │
│   go.mod:12     再次钉住 qkparser（indirect）                            │
│   internal/cgen        LLVM IR 后端                                     │
├─────────────────────────────────────────────────────────────────────────┤
│ module github.com/QuarkLangCommunity/QuarkLangQkparser    （共享）       │
│   go.mod        没有任何 require —— 只用标准库                          │
│   词法器、宏/预处理器、语法器、AST、位置、本地化器接口                    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 1. 接口清单

### 1.0 汇总表

| `I-k` | 接口 | 位置 | 验证者 | 失效方式 |
|---|---|---|---|---|
| I-1 | 词法器 | `qkparser/token.go` | `lintcorpus_test.go:148` | `\r` 视为空白；位置与行尾无关 |
| I-2 | 宏 / 预处理器 | `qkparser/macro.go` | `eval_test.go:574`、`macroinsert_test.go` | **两个引擎的模式不一致**（§6.6） |
| I-3 | 语法器 | `qkparser/parser.go` | `parser_test.go`、`parser_limits_test.go` | 嵌套超过 `MaxExprDepth` 即拒绝 |
| I-4 | AST + 位置 | `qkparser/ast.go:4-8` | 全部下游测试 | `Stmt`/`Expr` 由未导出方法封口 |
| I-5 | 类型检查器 | `internal/lang/typecheck.go` | `internal/lang` 内 typecheck 测试 | 显式深度计数器，不探测栈 |
| I-6 | 树遍历解释器 | `internal/lang/eval.go` | `eval_test.go`、`edges_test.go` | `try` 的错误模型有分歧（§6.1） |
| I-7 | 字节码 VM 与其 deopt | `internal/lang/vm.go` | `vm_diff_test.go` | 遇非 int 值 deopt，绝不报错 |
| I-8 | cgen → LLVM IR | `compiler/internal/cgen` | `parity_test.go`、`compare.sh` | 不支持即拒绝；绝不静默错误编译 |
| I-9 | 内存管理器 | `internal/lang/eval.go:199-354` | `eval_test.go:372,708` | 仅解释器一侧的引擎面 |
| I-10 | i18n 目录 + 本地化器 | `internal/i18n` | `coverage_test.go`（8 道门） | 棘轮约束，不是穷尽覆盖 |
| I-11 | 模块接线 | 根 `go.mod`、`compiler/go.mod` | `go build ./...` × 3 OS × 2 模块 | 本地 `replace` 只在检出内生效 |
| I-12 | 工具前端 | `cmd/*` | 各 `cmd` 的 `main_test.go` | CLI 帮助写着"默认中文"（§6.5） |
| I-13 | CI 门禁 | `.github/workflows/*.yml` | — | 见 §2.3 的 作业 × OS 矩阵 |

### I-1 · 词法器

- **位置**：`qkparser/token.go:170`（`Lex`）、`:176`（`LexWithComments`）、`:180`（`lex`）。
- **承诺**：`Lex(src string) ([]Token, error)`；每个 token 带 `Kind`、`Text`、`Line`、`Col`
  （`token.go:94-102`）。行号与列号**与行尾无关**：把每个 `\n` 换成 `\r\n` 后，
  token 数量相同，且逐个下标的 `Kind`/`Line`/`Col` 完全相同。
- **谁在组合它**：`internal/lang/compile.go:113`（解释器前端）；`compiler/macros.go:51`
  （编译器的宏阶段）；`internal/lang.LexWithComments` 在 `internal/lang/syntax.go` 再导出；
  `cmd/qkcheck` 经 `internal/lang.LintSource`。
- **如何验证**：`internal/lang/lintcorpus_test.go:148` 的 `TestLintCRLFPositions` 逐 token 比对
  CRLF 下的整条 token 流，`:176` 再断言诊断集合完全相同。CI 把它作为具名步骤运行
  （`.github/workflows/ci.yml:41-42`）。
- **失效方式**：字符串或原始字符串未闭合时返回 `LexError`（`token.go:104-112`）。
  位宽超出支持范围是语法器的检查而不是词法器的（`parser.go:99` `isBuiltinTypeName`）。

### I-2 · 宏 / 预处理器 —— 行为有分歧，见 §6.6

- **位置**：`qkparser/macro.go`；入口 `SplitMacroDefs`（`:22`）、`ExpandMacros`（`:149`）、
  `expandBody`（`:211`）。
- **承诺（按实现）**：`ExpandMacros(toks, macros, mode)` 重写宏调用。`mode` 决定 `#when` 的
  发生时机：`expandBody:268` 在 `args[0].Text == mode || (mode == "explain" && args[0].Text == "run")`
  时执行分支。拒绝信息把可接受集合写成 `compile|run`（`macro.go:255`，目录条目
  `internal/i18n/table_macro.go:18`）。
- **谁在组合它**：解释器传 `"explain"`（`internal/lang/compile.go:123`）；编译器传 `"compile"`
  （`compiler/main.go:334`、`:441`），经它自己的包装（`compiler/macros.go:44`），
  而该包装转发给同一个 `lang.SplitMacroDefs` / `lang.ExpandMacros`（`macros.go:53,60`）。
  也就是说宏**逻辑**是共享的，仅**模式**不同。
- **如何验证**：`internal/lang/eval_test.go:574` 的 `TestMacroWhenCompileDropped` 把解释器钉在
  `run-line`（宏同时带两个分支时）；`macroinsert_test.go` 覆盖 `#insert`/`#ast`；
  `TestMacroErrorDirective`（`eval_test.go:594`）覆盖 `#error`。`compiler/macros_test.go`
  只存在于未合并分支 `chore/install-sh`，**不在** `67ce88c` 里。
- **失效方式**：`#error` 中止展开（`macro.go:279`）；未知的 `#` 命令被拒绝（`macro.go:308`）；
  而 `#when` 参数落在可接受集合之外时**不执行任何分支且静默丢弃**——已实测，见 §6.6。

### I-3 · 语法器

- **位置**：`qkparser/parser.go:30`（`Parse`）、`:113`（`parseProgram`）；错误类型
  `ParseError{Msg, Line, Col}`（`:9-17`）。
- **承诺**：token 流变成 `*Program`。递归下降自己数嵌套层数，超过 `MaxExprDepth`
  （`qkparser/limits.go:17`，值 `65536`）的第一层就带位置拒绝；**绝不让 Go 栈来决定**
  （`limits.go:8-13` 给了理由：进程栈是与平台相关的大小，让它决定的实现会在不同 OS 上给出不同答案）。
- **谁在组合它**：`internal/lang/compile.go:128`；`qkparser.ParseSource*`
  （`frontend.go:11,17,24`），在 `internal/lang/syntax.go:195,199` 再导出。
- **如何验证**：`qkparser/parser_test.go`、`qkparser/parser_limits_test.go`；根模块通过压力用例
  `scale_line_1mb` 端到端复核该上限（`stress/NOTES.md:20-33`）。
- **失效方式**：合法但过深的构造是被拒绝而不是崩掉——上限比实测崩溃深度低 4.5 倍
  （`stress/NOTES.md:31`）。

### I-4 · AST 与位置

- **位置**：`qkparser/ast.go`；`Pos{Line, Col}` 在 `:4-8`；`Program` 在 `:16`。
- **承诺**：`Stmt` 与 `Expr` 是**封口接口**——`isStmt()`/`isExpr()` 未导出（`ast.go:130-133`，
  实现见 `:222-232`），所以 `qkparser` 之外的任何包都无法新增节点种类。下游消费者在一个封闭集合上分支。
- **谁在组合它**：`internal/lang` 以类型别名再导出节点类型（`internal/lang/syntax.go`，
  例如 `type Block = qkparser.Block`），因此解释器与 linter 是**同一套**节点的客户端；
  `compiler/internal/cgen/lower.go:927` 遍历同一个 `*lang.Block`。
- **如何验证**：由每个消费者测试隐式验证；封口本身是语言级保证。
- **失效方式**：封口使得新增节点种类成为"只改 `qkparser`"的变更，而每个引擎都必须同步跟进——
  这正是设计意图中的强制机制，也是解释器文档把它称作"正典 AST"的原因（`cgen.go:4-5`）。

### I-5 · 类型检查器

- **位置**：`internal/lang/typecheck.go`；入口 `Typecheck(prog *Program) error` 在 `:675`，
  `checkBlock` 在 `:1154`。
- **承诺**：在任何引擎运行之前拒绝程序；返回带位置的 `CheckError`。它每个表达式节点递归一次，
  因此按 `internal/lang.MaxExprDepth` 自己计数——与语法器用的是**同一个**常量，且以别名声明而非
  第二个数字（`internal/lang/limits.go:30`；`:15-29` 记录了"过去曾有一份同值同措辞的副本，而没有任何东西
  让它保持一致"）。
- **谁在组合它**：`internal/lang/compile.go:133`，位于 `compileSlow` 内，因此**两个**引擎都会经过它——
  cgen 调用 `lang.CompileWithImports`（`compiler/internal/cgen/cgen.go:35-37`）。
  `cmd/qkcheck -no-typecheck` 可退出该检查（`cmd/qkcheck/main.go:7`）。
- **如何验证**：`internal/lang` 内的 typecheck 测试；深度上限由 `scale_line_1mb` 压力用例检验，
  其结论是 `REJECT_AGREE`——两个引擎都带位置拒绝（`stress/NOTES.md:35-37`）。
- **失效方式**：两个深度计数器**量的不是同一件事**——语法器数递归层数，检查器数表达式节点数——
  所以深 `+` 链由检查器拒绝、深括号嵌套由语法器拒绝（`internal/lang/limits.go:27-29`）。
  这是有记录的不对称，不是缺陷。

### I-6 · 树遍历解释器

- **位置**：`internal/lang/eval.go`；入口 `Run` / `RunDebug`（在 `internal/lang/syntax.go` 再导出）；
  解释器结构体在 `eval.go:397`，内存管理器在 `:424` 初始化。
- **承诺**：**语义基准**。`compiler/internal/cgen/consistency_test.go:8-16` 写下了整个系统依赖的规则：
  当编译器无法用与解释器完全相同的语义实现某构造时，它必须拒绝该程序，
  绝不接受它并给出不同答案。
- **谁在组合它**：所有 `cmd/*` 经 `internal/lang.Run`；`cmd/qkrepl` 经 REPL 会话
  （`internal/lang/repl.go:106`）；`cmd/quarkwasm`——第三个后端，浏览器
  （`cmd/quarkwasm/main.go:1-5`）。
- **如何验证**：`internal/lang/eval_test.go`、`edges_test.go`、`refmove_test.go`、
  `strict_eq_test.go`，以及 §2.1 的差分语料。
- **失效方式**：在**被调者内部**抛出的运行时错误，解释器会被调用者的 `try` 捕获，
  而编译出的二进制会直接终止——`stress/NOTES.md:102-105` 把它记为真实存在的既存分歧，
  报告出来而不是藏起来。

### I-7 · 字节码 VM 与其 deopt 路径

- **位置**：`internal/lang/vm.go`（829 行）。deopt 哨兵 `errVMDeopt` 在 `:33`；入口 `runVM` 在 `:595`；
  安全模型见 `:12-18`。
- **承诺**：**是优化，绝不是第二套语义**（`vm.go:5-9`）。只有整个函数体都落在支持子集内才编译；
  其它一律回落到树遍历器（`vm.go:12-15`）：`try`/`catch`、for-in、`io.print`/`io.println` 之外的方法调用、
  space 调用、struct/list/string 操作、嵌套块内声明、遮蔽。VM 复用解释器自己的原语
  （`wrapI32`、`callFunc`、`ioPrintln`），因此错误值完全相同。deopt 是**信号**而不是错误：
  运行时值与静态推断类型矛盾时，函数被拉黑并由树遍历器重跑（`vm.go:29-34`）。
- **组合开关**：`QUARK_NO_VM=1` 完全关闭 VM（`vm.go:16-17`）——差分测试翻转的正是它。
- **如何验证**：`internal/lang/vm_diff_test.go`——`TestVMEqualsTreeWalker`（`:114`）断言 VM 开与关时
  stdout **以及**一行结论都相同；`TestVMEqualsTreeWalkerOnCorpus`（`:158`）在
  `../../examples` 与 `../../compiler/testdata/cases` 上重复该断言。关键是
  `TestVMCompilesHotShapes`（`:132`）断言 `vmExecutions`（`vm.go:25-27`）为三个具名形态增长、
  而**不为** `string_ops_fall_back` 增长——这是防"静默回退"的棘轮：
  没有它，一个永久 deopt 的 VM 会让所有语义测试保持绿色而什么优化都没做。
- **失效方式**：字节码只实现了整数；本语言允许把 float 传进声明为 `int` 的形参，
  此时必须 deopt 而不是报错（`vm.go:601-602`）。"VM 是否仍在被使用"的检查是一张**4 条白名单**
  （`vm_diff_test.go:137-141`），因此未列出的形态不受验证（§6.2 G-9）。

### I-8 · cgen → LLVM IR

- **位置**：`compiler/internal/cgen/`；入口 `Transpile(src, filename) (string, error)`（`cgen.go:34`）。
  流水线见 `cgen.go:3-9`：`lang.CompileWithImports → lowerProgram (lower.go) → emitter (cgen.go)`。
- **承诺**：输入源码，输出 **LLVM IR 文本**。它自己不携带词法器或语法器（`cgen.go:5-6`）——
  语法只有一个来源：`internal/lang`。值模型见 `cgen.go:15-22`：`int→i32`、`bool→i1`、
  `float→double`、`String→i8*`；`List<T>` 与 struct 都是堆引用，以匹配解释器的别名语义，
  从而任何别名上的写入都可见。后端未降级的构造返回带位置的显式错误——
  **绝不静默错误编译**（`cgen.go:11-12`）。
- **谁在组合它**：`compiler/main.go`（`qkc` CLI）以及编译器模块的测试。
- **如何验证**：`parity_test.go`——`TestParityCaseIRSyntax`（`:35`）把每个用例喂给 LLVM 自己的
  `llvm-as`，并在它缺失时**跳过**（`:38-40`）；`TestParityCaseOutput`（`:69`）把 stdout 与
  由解释器生成的已入库 `.out` 基线**逐字节**比对，并在 `lli` 缺失时**跳过**（`:71-73`）。
  端到端由 `compiler/testdata/compare.sh:31` 要求 stdout+stderr **以及**退出码相等。
- **失效方式**：两条 `t.Skip` 路径意味着：凡缺少 LLVM 工具之处，IR 与输出这两条承诺
  **根本没有被检查**（§6.2 G-10）。`.out` 基线之所以入库，只因一条定向的 `.gitignore` 反向规则
  （`.gitignore:11-13`）；没有它，一致性测试会因为基线缺失而失败，而不是因为内容不一致。

### I-9 · 内存管理器

- **位置**：`internal/lang/eval.go:199-354`——`MemBlock`（`:200`）、`memHeap`（`:210`）、
  `MemoryManager`（`:229`）、`Alloc`（`:242`）、`ReclaimTask`（`:275`）、`Compact`（`:292`）、
  `Fragmentation`（`:307`）、`BlockCount`（`:324`）、`Delete`（`:331`）、`Clear`（`:342`）。
  内建 `memory` 值是 `globalMemory = &Memory{BlockSize: 4096}`（`value.go:754-760`）。
- **承诺**：没有垃圾回收；`delete` 是确定性释放，把存储还给空闲池且**不清零**
  （`spec/STANDARD.md:264-269`）。分配优先选**内部仍有空位、且占用最少**的块，否则申请新块
  （`eval.go:242-271`）——这正是"潮汐式"负载能复用一组稳定块的原因；占用率用按 `Used`
  排序的最小堆跟踪（`eval.go:210-215`）。
- **谁在组合它**：只有解释器——`interp.mem`（`eval.go:397`，`:424` 初始化），
  用于 `memory.*` 内建（`:1128`、`:2388-2400`）、`new`/指针（`:741`）与 `taskm`（`:3353`、`:3374`）。
- **如何验证**：`internal/lang/eval_test.go:372`（`compact` 之后恰好剩下线程的常驻块）与
  `:708`（`delete`+`compact` 之后块数为 0）；基准在 `bench_test.go:85,130`。
- **失效方式**：**只有解释器拥有这个管理器。** cgen 没有 `MemoryManager`；编译出的二进制的存储
  由 `calloc` 加它自己发出的运行时承担。这是边界而不是缺陷，但它意味着
  `memory.Fragmentation()` 之类是**引擎专属面**，一致性用例必须避开它们。
  释放仍被引用的存储按设计是未定义且不检测的（`spec/STANDARD.md:280-282`）。
  两处文档注释漂移记录在 §6.4。

### I-10 · i18n 目录与本地化器

- **位置**：`internal/i18n/`；`Detect`（`i18n.go:56`）、`Catalog`（`:69`）、`Load`（`:82`）、
  `Localizer`（`:167`）、`T`（`:201`）、`SetLocale`（`:268`）、未命中计数器（`miss.go:13-40`）。
- **承诺**：`Load` 从十个按领域拆分的表构建不可变目录（`i18n.go:148-161`）。
  **条目的方向是它 key 的性质**：key 含汉字即为中文源、其值是英文译文；否则为英文源、
  其值是中文译文（`i18n.go:96-102`、`:220-237`）。`Localizer.T` 查的是**本地化器自己的语言**
  （`:201-213`）——即 `43bbff6` 的修复；在那之前查表被限定为 `EN`，
  于是 105 条英文源条目在 `zh` 下永远取不到（§5.2）。
- **谁在组合它**：`internal/lang/localizer.go:18-23` 的 `SetLocalizer` 把本地化器同时注入
  **本包**与 `qkparser`；`:34` 的 `init()` 完成 parser 侧接线。各 `cmd/*` 通过 `--lang` / `QK_LANG`
  设定语言。
- **如何验证**：`internal/i18n/coverage_test.go` 里的八道门——`TestTemplatesRegistered`（`:124`）、
  `TestCatalogIntegrity`（`:170`）、`TestVerbSetsMatch`（`:190`）、`TestNoSilentFallback`（`:212`）、
  `TestDecouplingRatchet`（`:240`）、`TestEnglishSourceRatchet`（`:267`）、
  `TestTRendersTheCatalogForTheRequestedLanguage`（`:294`）、
  `TestCatalogDirectionFollowsTheSourceLanguage`（`:319`）——外加 `verborder_test.go:19`。
- **失效方式**：两种语言的 `fmt` 动词多重集必须一致（`coverage_test.go:190`）。
  目录中没有的模板会以它自己的源语言渲染，并且**被计数而不是被拒绝**（`i18n.go:206-207`），
  所以静默回退是可观测的，但不阻断。接线扫描只走本仓库（`coverage_test.go:44`），
  且 `skipNestedRepo`（`:355-361`）会在任何嵌套 `.git` 处停下，因此
  **被钉住的 parser 模块自身的模板落在所有门禁之外**——探针与结果见 §5.2。

### I-11 · 模块接线

- **位置**：根 `go.mod:1-10`、`compiler/go.mod:1-12`。
- **承诺**：`go.mod:5-10` 把语法层钉在一个精确的伪版本
  （`v0.0.0-20261010115438-20ae746ffdee`）上，经模块代理解析，因此全新克隆无需同级目录的检出。
  `go.mod:5-9` 的注释记录了这次修复所针对的失败：相对 `replace` 只在检出内有效、会被导入方忽略，
  "移动语法层正是这样让三个平台上的 CI 全崩的"。
- **子模块**：`compiler/go.mod:7` 把根模块 replace 到 `../`；`:12` 重复 parser 钉版，
  因为**当编译器是主模块时，根模块声明的 replace 不生效**（`compiler/go.mod:9-11`）。
- **如何验证**：每个模块在各平台上 `go build ./...` + `go test ./...`，三个 OS
  （`.github/workflows/ci.yml:28-36`）。本机 2026-10-10（linux、Go 1.26.9、clang 22.1.8）：
  根与编译器模块均构建、测试通过（`ok internal/lang 10.6s`、`ok compiler/internal/cgen 1.1s`）。
- **失效方式**：共享前端**只用标准库**（`qkparser/go.mod` 没有 require 块）——这正是它能脱离本仓库
  被消费的原因。代价是 parser 不能导入 `internal/i18n`，于是它自定义了一个单方法 `Localizer`
  接口并**等待被注入**（`qkparser/localizer.go:8-11`）：在 `SetLocalizer` 运行之前，
  每条 parser 消息都经 `verbatim{}` 原样渲染（`:15-26`）。

### I-12 · 工具前端

| 工具 | 位置 | 承诺 |
|---|---|---|
| `qkcheck`（linter） | `cmd/qkcheck/main.go:1-13` | 15 个稳定编号 `QK101`–`QK115`（`internal/lang/lint.go:20-35`）；`-json`、`-params`、`-no-typecheck`、`-L`、`-exit0`、`--lang` |
| `qkdoc` | `cmd/qkdoc/main.go:1-12` | `Doc`/`DocItem` 来自 `internal/lang/docrender.go`；默认 Markdown，`-html` 自包含 |
| `qkrepl` | `cmd/qkrepl/main.go:1-11`；`internal/lang/repl.go:106` | `-e` 片段；交互式 `qk> ` 提示符；stdin 非终端时为批处理，出错后继续 |
| `qklsp` | `cmd/qklsp/main.go:1-8` | 诊断、跳转定义、补全、悬停、符号；自实现 JSON-RPC，**零第三方依赖** |
| `quarkwasm` | `cmd/quarkwasm/main.go:1-25` | `//go:build js && wasm`；`globalThis` 上的 `quarkCheck`/`quarkRun`；浏览器后端 |
| `qkfmt`（格式化器） | **不在本仓库** | `QuarkLangQkfmt`，模块 `quarklang/qkfmt`，`go 1.21`，自带扫描器——见 §5.1 |

linter 的设计承诺写在 `internal/lang/lint.go:5-9`：作用域规则与类型检查器**逐条对齐**，
且**只**报告编译器不报的东西——硬语法/类型错误属于 `typecheck`，由 CLI 另行呈现。
这正是"P99 零误报"这道门可以达到而不是空想的原因。

四个以 `internal` 为后端的工具都写着"默认中文"（`cmd/qkcheck/main.go:10`、
`cmd/qkdoc/main.go:9`、`cmd/qkrepl/main.go:7`），而 `i18n.Detect` 的进程默认返回 `EN`
（`internal/i18n/i18n.go:65`）。二者只靠 `Detect` 的环境优先次序勉强自洽；
矛盾记录在 §6.5。

### I-13 · CI 门禁

`.github/workflows/ci.yml`——4 个作业、263 行、`permissions: contents: read`（`:6-7`）、
同一 ref 上取消进行中运行的作业级并发（`:10-12`），每个 action 都用 SHA 固定并附版本注释
（`:24-25`、`:69-70`、`:99`、`:111-115`）。

| 作业 | 矩阵 | 它定义的组合 |
|---|---|---|
| `test`（`:16`） | ubuntu、macos、windows | 根模块再 `compiler/` 的 `go build`+`go test`；P99 linter 门（`:37-40`）；CRLF/扰动门（`:41-42`） |
| `stress`（`:58`） | ubuntu、macos、windows | 极端语料对**两个**引擎运行，`-timeout 90s`，跳过两个用例，带允许清单文件 |
| `linux-extras`（`:107`） | 仅 ubuntu | 双路径 `compare.sh`、工具链构建、编辑器/LSP、wasm、覆盖率、基准、发布冒烟 |
| `libraries`（`:182`） | ubuntu、macos、windows | emit-lib → 链接 → **运行**，断言平台分支值 58/59/61 |

刻意的平台差异写在工作流里而不是靠暗示：RSS 采样在非 Linux 上退化为 `-1`（`:55-57`）；
两个压力用例被跳过，原因是"代价"而非语言性质在支配它们（`:49-53`）。

---

## 2. 组合地图

一个组合是（接口集合、平台、环境、语料）的具体元组。本节要说明的是：
**不变量属于组合，从不单独属于接口。**

### 2.1 C-1 · 双引擎一致性

```
        ┌──────────────────────────────┐
        │ qkparser: 词法→宏→语法        │   I-1, I-2, I-3, I-4
        └───────────────┬──────────────┘
                        │ 同一棵 AST（I-4 已封口）
        ┌───────────────┴───────────────┐
        ▼                               ▼
┌──────────────────┐          ┌──────────────────────┐
│ I-6 解释器        │          │ I-8 cgen → LLVM IR   │
│ （+ I-7 VM 可选）  │          │ → clang → 二进制      │
└──────────────────┘          └──────────────────────┘
        │                               │
        └──────── 是否逐字节相同？ ──────┘
                        │
         compare.sh（32 个用例）+ parity_test.go
```

**构成**：I-1…I-5、I-6、I-7（可选）、I-8，外加一套位于仓库之外的 LLVM 工具链。

**在该组合下成立的不变量**

| 不变量 | 适用范围 | 证据 |
|---|---|---|
| stdout+stderr 逐字节相同、退出码相同 | 已入库语料：15 个 `cases/*.kq` + 17 个 `cases_run/*.kq` = 32 | `compare.sh:23,31`；**2026-10-10 实测为绿**（`all identical`） |
| IR 通过 LLVM 自己的语法检查 | 每个用例，*当 `llvm-as` 存在时* | `parity_test.go:35-40` |
| 运行结果等于已入库的 `.out` | 15 个用例，*当 `lli` 存在时* | `parity_test.go:69-73` |
| 递归上限一致，含尾调用 | 4 个 `p_recursion_*` 用例 | `stress/NOTES.md:68-78` |
| 未实现构造 ⇒ 拒绝而非错误编译 | `===`、权限引用、`&lvalue`、`p++` | `consistency_test.go:1-16` |

**该组合不承诺什么**

- ✘ 对**每个**程序都一致：语料只有 32 个文件，§6.6 给出了一个语料之外、两引擎必然分歧的程序。
- ✘ 在 CI 里除 linux 之外的平台一致：`compare.sh` 只跑在 `linux-extras`
  （`.github/workflows/ci.yml:118-121`）的 `ubuntu-latest` 上，而且它硬编码 `/tmp/quark`、`/tmp/qkc`
  （`compare.sh:17-18`）——Windows 上不存在这些路径。
- ✘ 捕获"被调者内部的运行时错误"——解释器捕获，编译出的二进制终止
  （`stress/NOTES.md:102-105`）。
- ⚠ Go 层的这两条一致性测试**会自我禁用**：缺少 `llvm-as`/`lli` 会把两条承诺变成 skip，
  于是一台没有 LLVM 的机器报绿，而实际上什么都没检查（`parity_test.go:38-40,71-73`）。

**与平台、环境无关的部分**：前端上限（显式计数器，`limits.go:9-13` 与
`qkparser/limits.go:8-13`）、AST 契约（I-4）、类型检查器的拒绝语义、宏**逻辑**，
都是作用在字符串与整数计数器上的纯函数——代码自己这么说，设计也依赖这一点。
**与环境相关的是**：编译器这条路径是否被行使（取决于 LLVM 是否存在）、
峰值 RSS 是否能测量（仅 Linux `/proc`）、以及 `compare.sh` 的绝对路径。

### 2.2 C-2 · 语言查找（按 locale）

```
   --lang / QK_LANG ─┐
                     ├─► i18n.Detect() ─► Lang ─► Localizer.T(template, args)
   LC_ALL/LC_MESSAGES/LANG ─┘                         │
                                            ┌─────────┴──────────┐
                                            ▼                    ▼
                                  Lookup(template, lang)   sourceLang(tpl) != lang
                                  → 目录值                  → countMiss(tpl)
```

**各 locale 下的不变量**

| 组合 | 承诺 | 状态 |
|---|---|---|
| `zh` 查找 | `T` 返回"所请求语言"的目录值 | ✔ 自 `43bbff6` 起成立；由 `coverage_test.go:294` 对全部 918 对钉住 |
| `en` 查找 | 镜像承诺 | ✔ 同一道门 |
| 方向 | 条目方向读自 key，绝不从"表"假定 | ✔ `coverage_test.go:319`、`i18n.go:96-102` |
| 动词元数 | 译文的 `fmt` 动词与源文完全一致 | ✔ `coverage_test.go:190` |
| 覆盖率 | 每条接线模板都有中文值 | ⚠ **棘轮而非硬门**：允许缺 ≤124 条（`coverage_test.go:132`） |
| 解耦 | 新代码携带 `Localizer` | ⚠ 棘轮：直接 `i18n.T(` ≤29 处（`coverage_test.go:241`）；当前实测 **25** |
| 源语言方向 | 新消息为英文源 | ⚠ 棘轮：中文源 ≤318 条（`coverage_test.go:268`） |

**"`zh` 这个组合"其实是三件不同的事**，而只有第一件是密不透风的：

1. 注入 `zh` 本地化器后的 `Localizer.T`——已在全部 918 对（模板, 语言）上穷尽验证
   （`coverage_test.go:294-317`）；
2. 环境说是 `zh` 的机器上的**进程默认语言**——只由 `Detect` 自身行为验证；
3. 开发者或 CI runner 的机器，其默认可能是 `en`（GitHub runner 是 en-US）。

那 124 条待办存在于第 1 层的数据里，但只以一个计数体现，
于是"加一条英文源消息、忘了它的中文值"这件事被允许发生 124 次。

### 2.3 C-3 · CI 作业 × OS × shell × 检出

| 作业 | OS | shell | 检出换行 | 实际强制的内容 |
|---|---|---|---|---|
| `test` | ubuntu/macos/windows | 各 OS 默认 | 原生（windows 为 CRLF） | 构建、测试、P99 lint = 0、CRLF/扰动不变性 |
| `stress` | ubuntu/macos/windows | 显式 **bash**（`ci.yml:65-67`） | 原生 | 两个引擎在墙钟上限下运行；RSS 上限仅 Linux |
| `linux-extras` | ubuntu | bash | LF | 双路径一致性、wasm 构建、编辑器、发布冒烟 |
| `libraries` | ubuntu/macos/windows | 显式 **bash**（`:189-191`） | 原生 | emit→link→run，断言平台分支 58/59/61 |

两个事实决定了一个步骤在给定组合里**能否存在**：

- `.gitattributes` **没有任何 `text`/`eol` 规则**——只有给 `internal/lang/ffi_win.inc` 的 linguist
  标记——因此 Windows 检出的工作树文件是**原生 CRLF**。
- 含 POSIX shell 语法（`$(...)`、`[ ... ]`、`command -v`、`case`）的两个多 OS 作业都显式声明
  `shell: bash`。`test` 没有声明，它能工作是因为它的步骤是纯 `go` 调用、不使用任何 shell 特性。

**§5.1 的 Windows 格式化失败，全部内容就是这些**：一个 CI **步骤**接口
（`qkfmt -l $(git ls-files '*.qk')`）组合了 `windows × pwsh × CRLF 检出`，
而这三个因素中有两个各自独立地破坏它。

### 2.4 C-4 · Linter 语料扰动

**构成**：I-1（词法位置）、I-3/I-4、`internal/lang/lint.go`，以及对本仓库自身 `.qk`/`.kq`
文件的一次遍历（`lintcorpus_test.go:56-77`），跳过 `.git`、`dist`、`dist-ci`、`node_modules`、
`lintbench`、`bench` 与每个嵌套仓库。

**不变量**：诊断集合在（1）CRLF 转换、（2）每行加行尾注释、（3）行位移下保持不变
（`lintcorpus_test.go:88-146`）；外加统计门：**P99** 误报数与漏报数在 2000 轮 × 4 用例下**恰好为 0**
（`lintstat_test.go:456,544,547`）。

**不承诺**：重排序或语义改写下的不变性；也不承诺本仓库之外的文件——同级检出里的库按构造就在遍历之外。

### 2.5 C-5 · 格式化器（树外）

**构成**：`QuarkLangQkfmt` 于 `2396137`——独立模块（`quarklang/qkfmt`、`go 1.21`），
带**自己的扫描器**（`main.go:10-45`），不使用 `qkparser`。它的 README 声称两件事：
只改空白且第二遍字节一致（幂等），以及 CI 模式 `-l` 对任何未格式化文件退出 1。

**各组合下的不变量**：见 §5.1。决定性事实是**本仓库没有任何东西调用它**：
`qkfmt` 只出现在 `scripts/bench-tools.sh:26,60`（一行基准、一个同级仓库构建辅助），
`ci.yml`、`release.yml`、`bench.yml` 里都没有；格式化器自己的 CI 是 `ubuntu-latest` 上的
`go build ./... && go vet ./...`。因此这条格式化承诺，今天在**任何** CI 组合下都不成立。

---

## 3. 边界与传播

### 3.1 共享 vs 私有

```
 共享（已发布模块，消费方是我们看不见的导入者）
 ┌──────────────────────────────────────────────────────────────┐
 │ QuarkLangQkparser：词法器、宏、语法器、AST+Pos、limits、       │
 │ ValidRefPerm/ValidRefScope、Localizer 接口、verbatim          │
 │ 约束：只用标准库 —— 它不能导入 internal/i18n                   │
 └──────────────────────────────────────────────────────────────┘
        ▲ 运行时注入                              ▲ 伪版本钉住
        │ (SetLocalizer)                          │ (go.mod:5-10, compiler/go.mod:12)
 私有（模块内部；导入者无法触及）
 ┌──────────────────────────────┐   ┌──────────────────────────────────┐
 │ internal/lang                │   │ internal/i18n                    │
 │ internal/lang/stressgen      │   │ （表 + Localizer 实现）           │
 │ cmd/*                        │   └──────────────────────────────────┘
 └──────────────────────────────┘   ┌──────────────────────────────────┐
                                    │ compiler/internal/cgen           │
                                    │ 独立模块；自带                    │
                                    │ preproc.go + macros.go           │
                                    └──────────────────────────────────┘
```

`internal/` 按构造让解释器、目录与 cgen 私有：外部消费者无法导入它们。唯一的共享面是 parser 模块——
这正是**注入方向**重要的原因。parser 不能调用目录，所以是根模块去调用 parser
（`internal/lang/localizer.go:18-23`），而 parser 只看到一个单方法接口
（`qkparser/localizer.go:8-11`）。

### 3.2 一处改动会传播到哪里

| 改动 | 传播到 | 由什么强制 |
|---|---|---|
| `qkparser/ast.go` 里的节点种类 | 解释器分支、linter 分支、cgen `lower.go` | 没有强制——Go 的 switch 不穷尽，封口只阻止在模块**外部**新增种类 |
| parser 模块里的 `MaxExprDepth` | 无需改动：根模块以别名读取（`internal/lang/limits.go:30`） | 漂移在构造上已不可能 |
| `MaxCallDepth`（`internal/lang/limits.go:37`） | 解释器的计数器**以及**写进生成 IR 的值——连措辞也是（`:39-42`） | `p_recursion_*` 一致性用例 |
| 一条消息模板 | 同一提交里的目录条目 | `TestTemplatesRegistered`——**但只覆盖写在本仓库里的模板**（§5.2） |
| parser 的钉版提交 | 全部；钉版是一行（`go.mod:5-10`），在 `compiler/go.mod:12` 重复 | 3 OS × 2 模块的 `go build` |
| 新增工具前端 | `cmd/`、`scripts/build-release.sh`、`install.sh`、`ci.yml` 的构建循环 | 手工清单——没有生成器 |

最后一行是真实的边界隐患：`ci.yml:124` 用一段手写循环构建 `qkcheck qkdoc qkrepl qklsp`，
`ci.yml:160-162` 校验发布产物 `quark qkc qkcheck qkdoc qkrepl qklsp`，
而 `install.sh:32` 又加上同级工具 `QuarkLangQkfmt:qkfmt QuarkLangQktest:qktest QuarkLangQkm:qkm
QuarkLangQkd:qkd`。四份各自独立的"工具链清单"，靠手工保持一致。

### 3.3 文档承诺 vs 实际强制

| 边界 | 文档位置 | 由什么强制 |
|---|---|---|
| 编译器无法降级就必须拒绝 | `consistency_test.go:8-16`、`cgen.go:11-12` | `TestInterpreterOnlyConstructsAreRefusedNotMiscompiled` |
| 前端上限与平台无关 | `internal/lang/limits.go:9-13`、`qkparser/limits.go:8-13` | 压力用例 `scale_line_1mb`、`p_recursion_*` |
| 两个引擎只有一个语法来源 | `cgen.go:4-6` | cgen 没有词法器；它调用 `lang.CompileWithImports` |
| linter 与类型检查器共享作用域规则 | `internal/lang/lint.go:5-9` | 语料不变性 + P99 = 0 |
| 解释器是语义基准 | `consistency_test.go:8-16`、`vm.go:5-9` | VM 差分测试、`compare.sh` |

---

## 4. 不变量与棘轮清单

**棘轮（ratchet）**＝只能朝一个方向移动的界，使待办被迫收缩、无法增长。
每条给出：钉住什么、由哪个测试强制、只允许往哪个方向动。

| # | 名称 | 钉住什么 | 测试 | 方向 |
|---|---|---|---|---|
| R-1 | 未译模板 | 已接线但没有目录条目的模板 | `coverage_test.go:132`（`≤124`） | **只降** |
| R-2 | 英文源迁移 | 中文源已接线模板 | `coverage_test.go:268`（`≤318`） | **只降** |
| R-3 | 本地化器解耦 | 直接 `i18n.T(` 调用点 | `coverage_test.go:241`（`≤29`；实测 25） | **只降** |
| R-4 | VM 确实在被使用 | 三个具名形态走了字节码，且 `string_ops_fall_back` 没走 | `vm_diff_test.go:132` | 双向布尔 |
| R-5 | 语料上的引擎一致性 | stdout+stderr+退出码，32 个用例 | `compare.sh:31`、`parity_test.go:69` | 必须精确不变 |
| R-6 | 同一个递归上限 | 深度、措辞、两个引擎、含尾调用 | `cases_run/p_recursion_*`、`limits.go:37-42` | 必须精确不变 |
| R-7 | 前端深度上限 | 语法器 65536 / 检查器 65536，只有一个数 | `qkparser/limits.go:17`、`internal/lang/limits.go:30` | 必须精确不变 |
| R-8 | linter 精度 | P99 误报 = 0 **且** P99 漏报 = 0 | `lintstat_test.go:544,547` | 必须保持 0 |
| R-9 | CRLF 不变性 | CRLF 下 token 与诊断完全相同 | `lintcorpus_test.go:148`、`:88` | 必须精确不变 |
| R-10 | 拒绝优先于错误编译 | 未实现构造 ⇒ 带位置的错误 | `consistency_test.go:24+`、`feature_test.go:647+` | 必须精确不变 |
| R-11 | 已知失败允许清单 | 列入的用例必须有复现；修好后必须移除 | `stress/known-failures.txt:10-18` | **只缩**（当前为空） |
| R-12 | 编译器侧本地化措辞 | cgen 不支持构造的消息钉在目录中文值上 | `locale_test.go:17-23`、`feature_test.go:647+` | 必须精确不变 |

R-1…R-3 是仅有的**数值**棘轮，其余都是精确值契约。R-11 是**故意为空**的，
文件里两处都这么说（`stress/known-failures.txt:16-18`）——这正是它算棘轮而不是 TODO 列表的原因。

---

## 5. 实例拆解

两个真实失败，拆成"接口 + 组合"，而不是"某个项目红了"。

### 5.1 Windows 格式化门 —— `windows × pwsh × CRLF` 下的 `qkfmt -l`

**涉及的接口**

| # | 接口 | 承诺 |
|---|---|---|
| A | `qkfmt` 的扫描/格式化 | 只改空白；第二遍字节一致（幂等）；`-l` 列出与格式化结果字节不同的文件并退出 1（`QuarkLangQkfmt/main.go:431-432`） |
| B | **步骤** `qkfmt -l $(git ls-files '*.qk')` | 一段 POSIX shell 命令替换，由 CI runner 的 shell 求值 |
| C | Windows runner 上的 `git checkout` | 产出另外两者所读的工作树字节 |

**组合**：`C = A × windows × pwsh × CRLF 检出`。

**每条承诺在哪里断掉。** 实测于 2026-10-10、`QuarkLangQkfmt@2396137`；
两者都是 shell/换行性质，因此凡这些因素出现之处都会复现：

| 因素 | 观察 | 断掉的承诺 |
|---|---|---|
| CRLF 下的 A | 对 `fn main(IOStream io) {\r\n    io.println("hi");\r\n}\r\n` 运行 `qkfmt -l`：打印路径并**退出 1**；同样的文字用 LF 则退出 0。`qkfmt` 的输出把 CRLF **归一化为 LF**（`od -c`：两种输入都是 47 字节、只有 `\n`） | A 的"列出未格式化的文件"——在 CRLF 工作树里每个文件都被报出来，因为该工具的规范输出是 LF |
| pwsh 下的 B | `$(...)` 在 PowerShell 里是子表达式，pwsh 不会把 `git ls-files` 当命令替换执行 | B 的"产生文件清单"——该步骤必须声明 `shell: bash` 才能工作 |
| C | `.gitattributes` **没有任何 `text`/`eol` 规则** | C 的"稳定的检出字节"——没有任何东西让 Windows 检出与 Linux 检出相同 |

**比第一层更要紧的二阶效应**：对该 CRLF 文件运行 `qkfmt -w` 会把它重写成 LF
（已验证——重跑 `-l` 随即退出 0）。因此在 Windows 上"顺手"跑格式化器的写模式，
会把每个 `.qk` 文件转成 LF，把整个工作树朝相反方向弄脏。
该工具没有 `--eol`/`--crlf` 开关：它的标志就是 `-w`、`-l`、`-d`（`QuarkLangQkfmt/main.go:362-366`）。

**今天这条承诺的状态**：CI 里**没有被违反，因为该步骤并不存在**。
本仓库没有任何工作流调用 `qkfmt`（只有 `scripts/bench-tools.sh:26,60` 构建并基准它），
而格式化器自己的 CI 在 `ubuntu-latest` 上跑 `go build ./... && go vet ./...`。
格式化器 README 里的那段步骤是一个**建议**，没有任何组合去实现它。精确表述如下：

> 接口 A 承诺对 LF 输入做幂等的、只改空白的格式化。在组合 `windows × pwsh × CRLF` 下，
> A 的 `-l` 模式把每个文件都报成未格式化，A 的 `-w` 模式则静默把整棵树转成 LF。
> 本仓库当前没有任何组合在运行 A，因此该缺陷是**潜伏的，而非正在发生**。

### 5.2 中文 locale 查找缺口 —— `zh` 组合下的 `Localizer.T`

**接口**：`Localizer.T(template, args...)` 承诺渲染**本地化器自己语言**的目录条目，
没有条目时回落到模板本身并计一次未命中（`internal/i18n/i18n.go:194-213`）。

**组合**：`C = Localizer.T × zh × {英文源目录条目}`。

**当时出了什么**：在 `43bbff6` 之前，`T` 把查表限定为 `L.Lang() == EN`。
于是每一条迁移到英文源模板的条目，在 `QK_LANG=zh` 下都原样返回调用点的模板。
目录有 459 条，其中 105 条的中文渲染与模板不同；这 105 条**全部无法经 `T(zh)` 取到**，
而 `Catalog.Lookup(key, ZH)` 却答得正确。译文存在，只是永远走不到。
两条刚加的深度诊断就在其中，因此 `LANG=zh_CN.UTF-8 ./quark deep_parens.kq` 输出英文
（`43bbff6` 的提交信息）。

**这个组合为什么没有被覆盖**：**前提本身被写进了测试**。
`TestChineseLocalizerRendersTemplates` 断言中文渲染"无需翻译、无未命中"——
也就是说，那条测试把缺陷钉成了预期行为。条目方向当时也是从**表**而不是从 **key** 读的
（`6a37193`），这恰好把英文值放到了中文值前面，只对已迁移的条目成立。
修复后的组合现在由 `TestTRendersTheCatalogForTheRequestedLanguage`
（`coverage_test.go:294-317`）钉住：它把**全部 918 对**（模板, 语言）经 `T()` 渲染并与目录比对——
对那 105 条翻转会失败——并由 `TestCatalogDirectionFollowsTheSourceLanguage`（`:319`）钉住
"方向随 key"。

**残留缺口，仍以接口 + 组合表述**：未命中计数器现在是对称的（`i18n.go:206-207`），
所以英文源模板在 `zh` 下渲染"是**可观测**的——但没有任何东西让它失败，
因为 R-1 允许 124 条未登记的接线模板。因此"在 `zh` 下，接线模板渲染中文"这条承诺，
**对每条有目录条目的模板成立**，而**不是** `zh` 这个组合整体的性质。

**同一组合里第二个更尖锐的缺口——实测，而非推断。** 目录校验扫描器只走本仓库，
并在任何嵌套 `.git` 处停下（`internal/i18n/coverage_test.go:44`、`:355-361`）。
parser 模块根本不在本仓库里——它从模块缓存解析——因此它的消息模板落在所有门禁之外。
我在 2026-10-10 用 `go/ast` 解析被钉住模块的 Go 源码，并就每条模板向目录提问，得到：

| 探针 | 结果 |
|---|---|
| `qkparser` 中传给 `msg(` 的字面模板（`macro.go:48-326`、`parser.go:72`、`token.go:197`） | **19 条不同**；19 条**全部在目录中** |
| 它们的方向 | 19 条中文源、0 条英文源 |
| 若其中一条**缺失**会怎样 | 把中文模板**原样渲染给英文读者**并让未命中计数 +1——已验证：`EN` 本地化器下 `T` 返回了中文，且 `MissTotal()==1` |

所以目前状态是"正确但**没有任何门禁在验证**"：parser 模块的目录覆盖之所以成立，
是因为有人手工誊抄了 19 条模板；而一次新增第 20 条模板的 parser 版本升级，
会让英文机器上出现中文输出，且没有任何东西失败。这才是精确的表述，
它与"i18n 坏了"是不同的表述。

---

## 6. 已知缺口

### 6.0 与三份一手来源的交叉核对

| 来源 | `67ce88c` 时的状态 |
|---|---|
| `spec/STANDARD.md` §11.1 | `bit`、`bits<N>`（1≤N≤32）、`uchar`、位运算族、`b[i]` 读写、三种转换含义、`===`、权限引用、`&lvalue` 与 `p++` 在两个引擎里都已实现；`compare.sh` 让语料保持一致 |
| `spec/STANDARD.md` §11.2 | N>32 的 `bits<N>` 由共享前端拒绝，两个引擎同一条消息；位宽只接受十进制字面量；`float`/`double` 的重解释被拒绝（本实现的 `float` 是 64 位，而 §3.2 写的是 `float32`）；宽度不是 8/16/32 的 `bits<N>` 在编译结构体里布局上取整 |
| `spec/STANDARD.md` §11.3 | **代码在此不遵循标准**：§7.2 宏拼写（仍接受 `#macro`、拒绝 `$`；一次合规改造被回滚）；§8 `try`/`catch` 仍在，尽管标准里没有异常；§9 `List`/`HashTable` 仍是内建，其数量被一条测试冻结 |
| `stress/NOTES.md` | 首轮全量运行的三项发现**全部关闭**；三个已修缺陷是前端崩溃、VM 编译的二次复杂度、以及两引擎在递归上限上的分歧 |
| `stress/known-failures.txt` | 为空，并写明"如何新增条目"的规则，以及为什么过期条目本身会被报成警告 |

### 6.1 来源自己声明的缺口

| # | 缺口 | 出处 |
|---|---|---|
| G-1 | 调用者的 `try` 捕获不到被调者内部抛出的运行时错误（编译端） | `stress/NOTES.md:102-105` |
| G-2 | cgen 的 IR 生成在深链上是超线性的：2 万加法 6.0 s、4 万 29.2 s、8 万 175.5 s，而共享前端 8 万只要 0.15 s | `stress/NOTES.md:107-112` |
| G-3 | `path_fn_10k_params` 恰好卡在 10 s 默认上限之内（9.18 s、峰值 1660 MiB）——被**故意**不列入允许清单，CI 跳过它 | `stress/NOTES.md:116-121`、`ci.yml:49-53` |
| G-4 | 宏拼写、异常、内建容器仍与标准分歧 | `spec/STANDARD.md` §11.3 |
| G-5 | `bits<N>` 布局按类型取整，而不是按 §3.1 要求的对齐到 1；值本身精确 | `spec/STANDARD.md` §11.2 |

### 6.2 本轮读代码发现的缺口

| # | 缺口 | 证据 |
|---|---|---|
| G-6 | **两个引擎今天在 `#when` 上就不一致**：只在解释器出现的宏分支会执行 `#when(compile)` 体，于是 `#when(compile){#return 111}` / `#when(run){#return 222}` 解释执行为 `222`、编译执行为 `111` | 已实测；机制是 `compile.go:123`（`"explain"`）对 `main.go:334,441`（`"compile"`）与 `macro.go:268` 的相互作用——详见 §6.6 |
| G-7 | **格式化器不在本仓库，且没有任何门禁组合它** | `qkfmt` 仅被 `scripts/bench-tools.sh:26,60` 引用；`QuarkLangQkfmt` 的 CI 是 ubuntu 上的 `go build && go vet` |
| G-8 | parser 模块的模板落在所有 i18n 门禁之外（当前正确，但无人强制） | §5.2 的探针；`coverage_test.go:44` + `:355-361` 的范围 |
| G-9 | `TestVMCompilesHotShapes` 是一张 4 条白名单，"VM 是否仍被使用"只对 3 个形态检查 | `vm_diff_test.go:137-141` |
| G-10 | 缺少 `llvm-as`/`lli` 时，Go 层一致性测试自我跳过，于是没有工具链的机器报绿 | `parity_test.go:38-40,71-73` |
| G-11 | 双路径一致性是仅 linux 的 CI 步骤，且脚本硬编码 `/tmp` 路径 | `ci.yml:118-121`、`compare.sh:17-18` |
| G-12 | 四份手工维护的"工具链清单" | `ci.yml:124`、`ci.yml:160-162`、`install.sh:32`、`scripts/build-release.sh` |
| G-13 | 已知失败条目必须带已入库的复现，或（输入太大不能入库时）一条能生成输入的命令——`stress/repro/` 里恰好有一个这样的脚本 | `stress/known-failures.txt:10-18`、`stress/repro/expression_chain_stack_overflow.sh` |

### 6.3 本轮未核实的内容

| 项 | 未核实原因 |
|---|---|
| **windows** 或 **macos** 上的任何行为 | 本次分析在 linux（Fedora、Go 1.26.9、clang 22.1.8）上进行；只有 CI 矩阵才会执行那些组合 |
| `67ce88c` 在三平台上的 CI 矩阵确实为绿 | 本文没有触发任何 CI 运行；拉取请求上的 `gh pr checks` 是第一份真实证据 |
| 压力语料（27 个用例、两个引擎）的结果 | 该套件受墙钟与 RSS 限制，本次未执行；`stress/REPORT.md` 是已入库的测量 |
| `bits<N>` 的编译端布局取整 | 由 §11.2 陈述，未重新测量 |
| 主检出里的 `dist/`、`quark`、`lang.test`、`qkcheck.test` | 被 gitignore 的构建产物，不属于本次分析的修订 |

### 6.4 矛盾 · 内存管理器注释 vs 标准

`internal/lang/value.go:750-753` 这样描述 `memory` 类型：

> "block-managed; spec §14. v0.1: **memory is backed by the host Go GC**；compact() 不返回任何值，
> 而 BlockSize 是动态的块粒度设置（memory.setBlock(n)）。"

`spec/STANDARD.md:263` 说：

> "QuarkLang **没有垃圾回收器**。存储是显式释放的，释放是确定性的：没有东西在你背后回收……"

两者不可能同时描述同一个机制。代码里有真实的块管理器
（`MemoryManager`，`eval.go:229-354`），由 `memory.setBlock`/`compact`/`Fragmentation`
驱动（`eval.go:1128`、`:2388-2400`），所以标准里的模型才是被实现的模型，
而"backed by the host Go GC"是一句过时的 v0.1 注释。相关漂移：同样的注释引用
**`spec §14.1` 与 `§14.2`**（`eval.go:199,2130,2400`、`internal/lang/value.go:720,751,764`），
而 `spec/STANDARD.md` 只有 **12 节**——它的变更记录写明 0.7.2 版时 §11 变成 §12、§10 变成 §11。
**指出矛盾，不擅自裁决。**

### 6.5 矛盾 · CLI 帮助文本 vs 进程默认

`cmd/qkcheck/main.go:10`、`cmd/qkdoc/main.go:9`、`cmd/qkrepl/main.go:7` 都写着
"`--lang` Output language zh|en（**默认中文**；QK_LANG 亦生效）"。
而 `internal/i18n/i18n.go:65` 把 `Detect` 的最后一行实现为 `return EN`，
注释是 "global default: the documentation is English-first"。

两种说法都是活的，它们之所以能勉强自洽，只是因为 `Detect` 在该默认之前先看 `QK_LANG`
再看 `LC_ALL`/`LC_MESSAGES`/`LANG`（`i18n.go:56-66`）——于是实际默认是
"环境说什么就是什么，否则英文"，而两句话都没有这么说。历史显示这是一次刻意的制度变更，
帮助文本没有跟上：`aba7e20`（2026-09-27）把默认改成中文且**只**读 `QK_LANG`；
`981abe1`/`6234e82`（同一天）改为跟随系统 locale、默认英文；`65495e4`（同一天）重写了消息层。
**指出矛盾，不擅自裁决。**

### 6.6 矛盾 · `#when` 未被规范，且两个引擎对它不一致

`spec/STANDARD.md:211-233`（§7.2）把运行时指令定义为 `#expand`、`#run`，
以及条件族 `#ifdef`/`#ifndef`/`#if`/`#ifn`/`#endif`，示例用的是 `#ifdef COMPILE` / `#ifdef EXPLAIN`。
**`#when` 在标准里根本没有出现。** 它只存在于实现中，语法由拒绝信息
`#when (compile|run)` 固定（`macro.go:255`，目录条目 `internal/i18n/table_macro.go:18`）。

行为上，对一个同时带两个分支、各自返回不同常量的宏——2026-10-10 用本修订构建的两个引擎复现：

```
$ cat when3.qk
#macro which () {
    #when (compile) { #return 111 }
    #when (run) { #return 222 }
}
fn main(IOStream io) { io.println(which()); }

$ ./quark when3.qk        → 222
$ ./qkc -run when3.qk     → 111
```

机制毫无歧义：`expandBody` 在 `args[0].Text == mode` 时执行分支，
而 `mode == "explain"` 还会额外触发 `"run"`（`macro.go:268`）；
解释器传 `"explain"`（`internal/lang/compile.go:123`），编译器传 `"compile"`
（`compiler/main.go:334`、`:441`）。也就是说 `#when(compile)` 选择的是"不是编译器"而不是"编译期"，
而解释器——系统中其他部分都称它为语义基准——恰恰是那个背离它自己所打印消息的引擎。

按接口表述其后果：

- `compare.sh` 自己的承诺（"same source, byte for byte"，`compare.sh:2`）对已入库语料成立
  （实测 32/32 相同），对这个程序**不成立**。
- 语料看不见它：`grep` `#when` 只能在 `examples/macro.qk:2` 与 `eval_test.go`、
  `macroinsert_test.go`、`stressgen.go` 里找到 `#when (run)`——**除 `eval_test.go:573-590` 外，
  没有任何 `#when(compile)`**。那唯一一条解释器测试断言的是**规则**
  （"`#when(compile)` 块在运行时被丢弃"，`eval_test.go:573`），
  但它用的宏体第一个分支是 `#return` 形状的，于是运行时走的是 `run` 分支、打印出预期的 `run-line`。
  这条测试在一个并未实现它所宣称规则的模式下**通过**了。
- 也没有任何引擎校验模式集合：集合之外的参数不执行任何分支并被**静默**丢弃——
  已实测，`#when(bogus){#return 999}` 与 `#when(run){#return 222}` 并存时输出 `222`，没有任何诊断。

**UNVERIFIED**：正确的修法究竟是从解释器传 `"compile"`、把模式集合改称 `explain|run`、
还是按标准的 `#ifdef` 家族彻底去掉 `#when`。三种都能与树里的**某个**产物自洽，
所以这里报告的是矛盾，而不是给出确定修法的缺陷。

---

## 7. 与其它文档的关系

本文增加的是对接口边界的描述，不复述规范性内容：语言的事实来源仍是 `spec/STANDARD.md`，
鲁棒性实测证据的事实来源仍是 `stress/NOTES.md` / `stress/REPORT.md`。
本文与它们冲突时，以它们为准，且 §6 记录该冲突。
