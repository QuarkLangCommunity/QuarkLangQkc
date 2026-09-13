# QuarkLang · compiler

**跨系统编译器分支**。后端：**LLVM IR**（不经 C 转译）——`qkc` 生成文本 LLVM IR，`clang` 产出任意 LLVM 支持平台的原生代码。

## 架构：单一语法源

编译器不再自带词法/语法分析，前端唯一入口是语言实现（`internal/lang`，与解释器同源）：

```
lang.CompileWithImports(src, filename)   // 词法/语法/类型检查 + import 递归合并
  → internal/cgen/lower.go               // 正典 AST → cgen IR（不支持即报错）
  → internal/cgen/cgen.go                // cgen IR → LLVM IR（发射器）
```

- `cgen.Transpile(src, filename)`：filename 决定 `import` 的同目录搜索范围，并用于诊断定位。
- 宏展开仍走解释器的 token 级宏系统（`compiler/macros.go`，compile 模式），再交给语言前端。
- 曾经的 cgen 自研 parser 已删除：语法只有正典语法（类型在前）一套。

## 用法

~~~sh
go run . testdata/hello.qk          # 输出 LLVM IR 到 stdout
go run . -run testdata/hello.qk     # clang 编译 IR 为原生二进制并执行
go run . -run testdata/demo.qk      # import + 递归 + List + try/catch 示例
go test ./...                       # 单测（llvm-as 语法校验 + lli 端到端）
testdata/compare.sh                 # 解释器 vs 编译器逐字节对比（全部语料）
~~~

## 支持的子集（语义基准 = 解释器）

**同一份源码在 `quark`（解释器）与 `qkc -run`（编译器）下输出逐字节一致**；
`testdata/compare.sh` 批量对比，`testdata/cases/*.kq`（+ `cases_run/`，需 clang 链接）
是回归语料，基准输出由解释器生成。

- 入口：`fn main(IOStream io) { ... }`（必须，首个参数为 IOStream）；`import` 递归合并
- 函数：任意标量/struct/接口返回与形参（int/bool/float/String/struct/接口）；调用/递归/尾调用
- 变量与语句：int/bool/float/String/List<int>/struct/接口/thread/Channel；
  赋值、`l[i] = v`、struct 字段赋值、`if/else`、`while`、C 风格 `for`、迭代 `for (T x : l)`、
  `break`、`log`、`try/catch (void e)`、`return`、`delete l`、`io.println/io.print`
- 表达式：int/float/bool/String 字面量、`+ - * / % << >>`、比较（int/float/String/bool）、
  短路 `&& || !`、String 拼接、`int/float/bool.toString()`、`l.size()/get(i)/append(v)/l[i]`、
  内置 `sum(...)` / `clock()`
- 对象模型：struct（引用语义，与解释器 *Struct 别名一致）、struct 字面量（命名+位置）、
  `impl` 实例/静态方法、`space`（`space::f`）、Operation 运算符重载（`__add__` 等）
- 泛型：`fn<T>` 与 `type struct<T>` / `impl<T>` 按调用点单态化
- 接口：结构化满足（由 internal/lang Typecheck 校验）+ vtable/thunk dynamic 分发
- library FFI：LLVM `declare` + C ABI 直调，IR 里 `; qkc-link: -lm` 标记由 qkc 转成链接参数
- taskm：`spawn/merge/block/done/channel` 对接 `qthreads.c` 运行时（pid = 小整数句柄）

## 暂不支持（必须报明确错误，绝不静默错编）

后端未 lower 的构造统一返回 `compiler: 暂未支持 …` 并带源码位置：

- String/List 内建方法（`substring`/`size`/`split`/`trim`/`indexOf`/`List.toString()` 等，
  仅 `int/float/bool.toString()` 已 lower）
- `List<T>`（T ≠ int）变量/形参/返回、HashTable、指针/`new`/`copyd`、匿名 struct
- `interface{}`（tAny）、签名调用 `f(...) @sign`、函数重载（同名函数）
- `long`、library FFI 的 `pointer` 参数/返回、float 取模（解释器运行期报错）
- 打印 struct/接口值（解释器字段序来自 Go map，本身不确定）
- 含 `log` 的函数的返回值被表达式使用（解释器返回 nil，静态类型无法表达）
- `merge` 的多参数/非 int 参数（qthreads 运行时只携带 1 个 int）、`thread.talk`

## 布局

- `main.go` —— `qkc` CLI（读 .qk，输出 LLVM IR；`-run` 编译执行；解析 `; qkc-link` 链接标记）
- `macros.go` —— 复用解释器 token 级宏系统（compile 模式）
- `qthreads.c.txt` —— taskm 运行时（嵌入；线程池 + 通道，pid 为小整数句柄）
- `internal/cgen/lower.go` —— 正典 AST → cgen IR 的 lowering 层与全部支持性诊断
- `internal/cgen/lower_generic.go` —— 泛型单态化（fn<T> / struct<T> / impl<T>）
- `internal/cgen/lower_iface.go` —— 接口 vtable + thunk dynamic 分发
- `internal/cgen/lower_taskm.go` —— taskm 线程/通道 lowering
- `internal/cgen/cgen.go` —— LLVM IR 发射器（类型/字符串常量/printf/SSA 值/函数属性/vtable）
- `testdata/cases/` `testdata/cases_run/` —— 解释器/编译器输出对比语料（`.out` 为解释器基准）
- `testdata/compare.sh` —— 批量逐字节对比（解释器 vs `qkc -run`）

语言设计见 `SYNTAX.md`（正典语法清单，与实现同步维护）。
