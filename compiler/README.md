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
go test ./...
~~~

## 支持的子集（本轮 lowering 范围）

- 入口：`fn main(IOStream io) { ... }`（必须，首个参数为 IOStream）
- 函数：`fn name(int a, ...) int|void|String { ... }`，调用/递归；参数可被赋值；String 返回编译为 `i8*`（拼接/打印/赋值/尾调用都支持）
- import：`import "同目录库";`（.qk 源码或 .qlib，递归合并，与解释器同一语义）
- 变量：`int x = 1;` `String s = "hi";` `List<int> l = [1, 2];`（int/String 可省略初值）
- 语句：赋值、`l[i] = v`、`if/else if/else`、`while`、`try/catch (void e)`、`return`、`delete l`、`io.println(...)`
- 表达式：int/String/bool 字面量、`+ - * / % << >>`、int 比较 `== != < <= > >=`（String/bool 比较暂未支持）、`&& || !`、
  String 拼接、`int/bool.toString()`（返回 String，可参与拼接/打印/返回）、`l.size()` / `l.append(v)` / `l[i]`、
  内置 `sum(g, begin, stop[, step])` / `clock()`

## 暂不支持（必须报明确错误，绝不静默错编）

后端未 lower 的构造统一返回 `compiler: 暂未支持 …` 并带源码位置，例如：

```
compiler: 暂未支持 impl 方法调用 p.sum（解释器可用）
  --> sample.qk:6:19
   |
 6|     io.println(p.sum());
   |                  ^
```

覆盖范围：struct 实例化/字段访问/字段赋值/方法调用、`impl`/`space` 方法块、`space::f(...)` 调用、
接口与 `dynamic` 分发、泛型实例化、Operation 运算符重载、`library` FFI、taskm 线程/通道、
`for`、`break`、`log`、`io.print`、float/bool 变量、String 形参、非标量返回类型（如 `List<String>`）、
指针/`new`/copyd、函数重载、非正典（名字在前）写法。

语言内建方法只 lower 了 `List.size()/get(i)/append(v)` 与 `int/bool.toString()`；
`String.substring/trim/split/...`、`List.toString()` 等会报「暂未支持 … 内建方法」——
类型推断仍按语言前端的方法表正确推断（不会再把 `"x" + x.toString()` 误判为拼接类型错误）。

## 布局

- `main.go` —— `qkc` CLI（读 .qk，输出 LLVM IR；`-run` 编译执行）
- `macros.go` —— 复用解释器 token 级宏系统（compile 模式）
- `internal/cgen/lower.go` —— 正典 AST → cgen IR 的 lowering 层与全部支持性诊断
- `internal/cgen/cgen.go` —— LLVM IR 发射器（字符串常量/printf/SSA 值/函数属性）
- `testdata/` —— 正典语法示例（`hello.qk`、`demo.qk` + 库 `mathlib.qk`）

语言设计见 `SYNTAX.md`（正典语法清单，与实现同步维护）。
