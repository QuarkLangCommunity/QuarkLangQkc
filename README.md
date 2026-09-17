# QuarkLangQkc（QuarkLang 主仓——语言 + qkc 编译器）

一门**编译型编程语言**：为高计算、高并发、海量临时数据场景设计。同一种语法，双后端：**Go 解释器**（开发/调试）+ **LLVM IR 编译器**（`qkc`，性能 = C 级）。

## 亮点

- **编译路径性能 = C**：LLVM `-O3` 同后端（fib35 21ms vs C 22ms；P99 延迟分布逐分位同级）；
- **并发模型 = Erlang 式**：`spawn/merge/block/done/channel` 用户态任务 + 线程池——并发模型是 Erlang 式，性能是 C 级；
- **零 GC 内存**：block 线性分配 + 占用度最小堆，`delete` 入空闲队列（数据保留）、`clear` 才清空——无 GC 停顿、碎片率 0.195%、复用率 99.96%；
- **sum 数学优化**：线性闭式 / 周期位级置换 / 均匀随机期望——10 亿项求和 O(1)（2ms，Go 循环 223ms）；
- **增量编译**：IR+二进制两级缓存，二次编译 16 倍提速；
- **零第三方依赖**：词法/解析/类型检查/求值/LLVM IR 发射全部手写。

## 特性（v2 语法面）

- **函数**：显式返回类型，`return expr` 结束并返回，`log expr;` 记录并结束；
- **try/catch**：`try { } catch (e void) { }`（除零等错误可捕获）；
- **void = 空接口**：任意值可赋；
- **struct / impl / interface**：`type struct { a int; } Point;`、`impl Point { fn sum(self) int {...} }`、`.{3, 5}` 字面量、`self.a` 字段访问；
- **泛型**：`fn f<T>(...)` / `f<int>(x)`（类型擦除，编译可用）；
- **函数引用**：`type function<int, int> F;`、函数作为值传递与调用；
- **指针 / 堆申请**：`pointer <T>` 修饰、`new <type>[size]` 堆上申请（非法大小 `badAlloc`）、空指针解引用 `NullPointerError`；
- **签名**：`f(args) @instance(prefix)` ≡ `instance.call(prefix)(.{in, out})`（instance 为任意 Sign 实例变量名）——记忆化/包装；
- **taskm 并发**：`t thread = taskm.spawn(); t.merge(fn, args); taskm.block(t.pid()); taskm.done(pid); c channel = taskm.channel(); c.send(v); c.recv();`——用户态任务 + 线程池（编译路径 pthread 载体，跨系统）；
- **宏系统**：`#macro name (参数) { 主体 }` 命名参数宏——参数不限、`()/[]/{}` 分隔符任意，调用 `name(args)`/`name[args]`/`name{args}`，参数按名替换；主体支持 `#when(compile/run)` 与 `#error`——**解释器与编译器共享同一 token 级宏展开**；
- **delete/clear 语义**：`delete` 入空闲队列（数据保留，可复用），`clear` 真正清空空闲数据（使用中保留，数据安全）；
- **List<int>**：字面量/下标/`size()`/`get(i)`/`append(v)`（几何增长 O(n)）；
- **program/library**：`program main;`/`library;`、`import`、`pub`——可发布为 `.qlib` 库；
- **解释器与编译器语法完全一致**（同前端，双后端）。

## 快速开始

### 解释器（仓库根，Go 模块 `quarklang`）

```sh
go build -o quark .
./quark examples/hello.qk
go test ./internal/lang/     # 全量测试（-race 可跑）
```

### 编译器（`compiler/`，LLVM 后端）

```sh
cd compiler && go build -o qkc .
./qkc -run hello.qk          # LLVM IR → clang 原生 → 执行
./qkc hello.qk               # 仅输出 IR
```

**依赖**：Go ≥ 1.26（零第三方 Go 依赖）；LLVM 工具链（`clang`/`lli`/`llvm-as`，系统包）。可选 `rustc`/`gcc` 仅用于跨语言对比基准。

### 静态检查 qkcheck（`cmd/qkcheck`）

```sh
go build -o qkcheck ./cmd/qkcheck
./qkcheck examples/                                  # 目录或文件；有诊断退出 1
./qkcheck -json -L ../QuarkLangLibs-Style lib.qk     # CI/编辑器消费；-L 追加 import 搜索目录
./qkcheck -params src.qk                             # 附带检查未使用形参
```

| 码 | 检查 | 说明 |
|---|---|---|
| QK101 | 未使用变量 | 声明后从未读取（含「只被赋值」）；`_` 前缀忽略 |
| QK102 | 未使用形参 | 需 `-params`（接口实现常有忽略形参） |
| QK103 | 不可达代码 | `return`/`log`/`break` 之后的语句；两分支皆返回之后的语句 |
| QK104 | 遮蔽 | `for`（两种）/`catch` 变量遮蔽外层同名变量（编译器未拦截的三处） |
| QK105 | 缺返回 | 声明非 void 返回类型却存在不返回值即结束的路径（解释器得 nil，`qkc` 补零值——两端不一致） |
| QK106 | 接口未实现 | 近失配：实现了接口的部分方法、缺其余（列出缺失方法名） |
| QK107 | void 值误用 | void 函数调用结果被当作值使用（运行期为 nil） |

**实测**：主仓全部 `.qk/.kq` 语料 → 0 错误、5 条警告且逐条复核为真问题（`log` 结束函数后仍写
`return`、迭代变量未使用等）；4 个官方库仓约 4000 行（含 `style.qk` 1500 行、`cleg.qk`）→ 仅
1 条真问题（局部变量从不读取）、0 误报。

### API 文档 qkdoc（`cmd/qkdoc`）

```sh
go build -o qkdoc ./cmd/qkdoc
./qkdoc lib.qk                       # Markdown 到 stdout
./qkdoc -all -o API.md style.qk      # 含未 pub 符号，写入文件
./qkdoc -html -o API.html style.qk   # 自包含 HTML（零外部资源）
```

- 文档注释：声明**上方紧邻**的 `//` 或 `/* */` 注释块（godoc 规则）；无上方注释时取**同行行尾**注释
  （`int x; // 横坐标`）；文件头注释作为文件说明。
- 默认只导出 `pub` 符号；文件没有 `pub`（如 `program main`）时导出全部；`impl`/`space`/`library`
  不受 `pub` 过滤（语言中 `pub` 不能前缀它们，而它们正是库对外 API 的载体）。
- 实测：`style.qk`（1567 行）→ 8ms 生成 335 行 Markdown（概览表 + 函数/类型/实现/空间分节）。

### 交互式求值 qkrepl（`cmd/qkrepl`）

```sh
go build -o qkrepl ./cmd/qkrepl
./qkrepl                      # 交互：qk> 提示符，块未闭合自动续行 ..>
./qkrepl -e "int x = 6;" -e "x * 7"   # 一次性求值（多段共享环境）
printf 'fib(20)\n' | ./qkrepl # 批处理（stdin 非终端）
```

```
qk> fn fib(int n) int {      # 多行块：{ 未闭合 → 续行
..>     if (n <= 1) { return n; }
..>     return fib(n - 1) + fib(n - 2);
..> }
已定义 fn fib(int n) int
qk> fib(20)
6765
```

- **持久环境**：变量/函数/结构体/接口/impl 跨输入存活；同名函数可重定义（覆盖生效）。
- 表达式直接回显值；`log` 的记录与 `io.println` 输出即时显示；语句可省略 `;`（自动补）。
- 命令：`:help`、`:quit`、`:load <file.qk>`（登记文件内的定义，随后可 `main(io)`）。
- 实现要点：会话持有一个解释器实例，每段输入按需走「顶层声明登记」或「包成函数体逐条求值」；
  与 `quark file.qk` 共用注册路径（`registerProgram`），因此语义一致。
- 实测：`fib(20)` → 6765；`:load examples/struct.qk` 后 `Point p = .{a:20, b:22}; p.sum()` → 42。

### 编辑器支持：qklsp + VS Code / tree-sitter（`cmd/qklsp`、`editors/`）

```sh
go build -o qklsp ./cmd/qklsp        # 语言服务器（stdio LSP）
./qklsp -L ../QuarkLangLibs-Style    # 额外 import 搜索目录
```

| 能力 | 说明 |
|---|---|
| 诊断 | 解析/类型错误（`qkc`）+ 静态检查（`qkcheck` 的 QK101–QK107），跨 import 文件也定位正确 |
| 跳转定义 | 函数/结构体/接口/空间/字段/局部变量（UTF-16 列换算，中文源码下同样准确） |
| 补全 | 关键字 + 内置类型/方法 + 当前文件的函数/类型/空间/局部变量 |
| 悬停 / 大纲 | 签名 + 文档注释；`documentSymbol` 列出全部顶层符号 |

编辑器产物：`editors/vscode`（VS Code 扩展：TextMate 语法高亮 + 片段 + 零依赖 LSP 客户端）、
`editors/tree-sitter-quarklang`（tree-sitter 语法：Neovim/Helix/Emacs 可直接用）。

**实测**：`editors/tree-sitter-quarklang` 对主仓 + 6 个库仓共 **60 个 `.qk/.kq` 文件解析 0 错误**
（含 `style.qk` 1567 行、`cleg.qk`），`tree-sitter test` 5/5 通过；
`editors/vscode/test/protocol.test.js` 用 Node 直连 `qklsp` 八项全过（初始化/诊断/跳转/补全/悬停/改动静默/退出）。

**跨系统**：qkc 产出与平台无关的 LLVM IR，目标平台 `clang`/`llc` 生成原生二进制；`.qlib` 库（gob）跨系统；线程运行时（`qthreads.c` 内嵌）POSIX/Windows 双载体。

**优化旗标**：默认 `-O3`（便携）；`QUARK_CFLAGS="-O3 -march=native"` 本机极限（产物仅当前 CPU）；PGO 可用 `-fprofile-generate/-fprofile-use`（fib35 -29%）。

**缓存**：增量编译缓存默认 `/tmp/quarklang-cache`（`QUARK_CACHE` 覆盖）；`QUARK_CFLAGS` 参与缓存键。

## 性能（实测，可复现）

| 基准 | QuarkLang(编译) | C | Rust | Go | Erlang |
|---|---|---|---|---|---|
| fib(30) | **3ms** | 3ms | 3ms | 6ms | 1106ms |
| fib(35) | **21ms** | 22ms | **17ms** | 36ms | — |
| 8 路并发 ×1e7 | **1ms** | — | — | 6ms | 61s(1e5) |
| P99（fib20×1000） | **14/27µs** | 16/25µs | — | — | — |
| 潮汐 1 亿轮 | **1ms** | 1ms | 29ms | 92ms | — |
| sum 10 亿项（闭式） | **2ms** | — | — | 223ms | — |

复现：`bench/Makefile`（跨语言）+ `docs/benchmarks.md`（方法/公正性声明）。

## QuarkLangQkc（QuarkLang 主仓——语言 + qkc 编译器） 官方项目

| 项目 | 说明 | 仓库 |
|---|---|---|
| QuarkLangLibs-Cleg | 官方认证的 `cleg` GUI 框架：`ClegNode` 接口（dynamic 派发）+ `ClegWindow` 默认窗口/`ClegLabel`/`ClegButton`，全节点 Style(HashTable) 驱动渲染（`label.style["text"]=...`）；运行时光栅内核（fill4K 2.5ms，零分配热路径） | https://github.com/QuarkLangCommunity/QuarkLangLibs-Cleg |
| QuarkLangLibs-GL | 官方认证的 `gl` 库：**OpenGL 声明集**（`library gl { ... }` 直接调用系统 GL 导出符号，`glc::` 常量空间；运行时跨系统 dlopen/LoadLibrary + libffi） | https://github.com/QuarkLangCommunity/QuarkLangLibs-GL |
| QuarkLangLibs-Vulkan | 官方认证的 `vulkan` 库：**Vulkan 声明集**（`library vulkan { ... }`，实例/设备/交换链/内存/缓冲常用面 + `vk::` 判定常量） | https://github.com/QuarkLangCommunity/QuarkLangLibs-Vulkan |
| QuarkLangLibs-Json | 官方认证的 `json` 库：Python 风格 `json::dumps` / `json::loads`（值↔JSON，对象→HashTable/数组→List/整数→int，非法输入报 JSONError） | https://github.com/QuarkLangCommunity/QuarkLangLibs-Json |
| QuarkLangLibs-Actions | 官方认证的 `actions` 库（两级）：`space` 系统级函数（`system`/`network` 空间，`exec`/`execv`/`popen`/`get`/`post`）+ `Command`/`Network` 类实现 `Executor` 接口（`self Self`，`.exec()`）；含 shell 注入说明与 8 MiB/10s 上限 | https://github.com/QuarkLangCommunity/QuarkLangLibs-Actions |

使用：把库的 `.qk` 文件放在与源码同目录，`import "actions";`（进程/网络）、`import "json";`（JSON）、`import "gl";` / `import "vulkan";`（图形）、`import "cleg";`（GUI 框架）后即可调用。

## 布局

- `main.go` + `internal/lang/` —— 解释器（lexer/parser/typecheck/eval/runtime/宏）
- `compiler/` —— LLVM 编译器（`qkc` + `internal/cgen` IR 发射器 + 内嵌线程运行时）
- `cmd/` —— 工具链（复用同一前端）：`qkcheck` 静态检查、`qkdoc` API 文档、`qkrepl` 交互求值、`qklsp` 语言服务器
- `editors/` —— 编辑器支持：VS Code 扩展（`vscode/`）+ tree-sitter 语法（`tree-sitter-quarklang/`）
- `bench/` —— 跨语言对比源（C/Rust/Go/Erlang + Makefile）
- `examples/` —— 示例

## 分支

| 分支 | 内容 |
|---|---|
| `main` | 集成（解释器 + 编译器 + bench） |
| `interpreter` | 解释器历史 |
| `examples` | 示例 |
| `docs` | 设计文档（独立维护，不并入 main；含 benchmarks.md） |
| `design` | XMind 设计蓝图（只读保护） |

设计文档：[docs 分支 spec.md](https://github.com/QuarkLangCommunity/QuarkLang/blob/docs/spec.md)。

## 状态

v2 语法面完整（解释器 + 编译器一致），性能 = C 级（编译路径）。见 `docs/benchmarks.md` 与宣传视频（`/home/jack/quarklang-promo/`）。