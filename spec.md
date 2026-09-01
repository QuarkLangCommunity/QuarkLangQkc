# QuarkLang 语言规范（v2）

> 本文档与代码同步维护。v2（2026-08-30）相比 v1 的修订：**FuncBuffer 类型彻底删除**；函数必须声明返回类型，`return` 返回并结束、`log` 记录并结束；`out` 与多返回值移除（多输出用 `List<T>`）；`void` 成为空接口 `interface{}` 的默认名字；错误处理改为 `try/catch(名字 类型)`；结构体字面量 `.{...}`；签名改为实例形式 `f(args) @mb(prefix)`；新增**宏系统**。

## 0. 术语表

- **宏（macro）**：`macro {模式} {主体}` 定义的编译期/运行期双向预处理单元。
- **预处理命令**：`#` 开头、在宏主体中动态执行（`#when/#insert/#execute/#error`）。
- **预制宏**：`program`/`import`/`pub` 等语言内置宏。
- **签名（Sign）**：`@mb(prefix)` 形式的调用包装；mb 是 Sign 接口的实例。
- **线程（taskm）**：`taskm.spawn()` 创建；`merge` 并入函数执行。

## 1. 设计目标

面向 CLI 工具的语言：语法无新概念，编译期严格，运行期错误可定位（log + try/catch），一套设计两套实现（Go 解释器 + LLVM 编译器），带宏系统对接编译期与运行期。

## 2. 词法与程序结构

- 注释 `//`、`/* */`；标识符；整数/浮点/字符串字面量（`\n \t \r \" \\` 转义）。
- 关键字：`fn struct impl interface out(已废弃) return if else while for in true false log try catch macro null`。
- `#` 开头为预处理命令。
- 程序 = 若干顶层声明 + 可选预制宏声明（`program`/`import`/`pub`） + `macro` 定义。

## 3. 类型系统

- `void` = 空接口 `interface{}` 的默认名字（预声明 `interface {} void;`）——"不是空，是没有接口"。
- `interface { ... }` 匿名接口；`interface { ... } 名字;` 有名接口。
- `struct { ... }` 匿名结构体类型；`struct { ... } 名字;` 有名结构体。
- 结构体字面量：`.{field: value, ...}`（字段名允许 `in`/`out` 等关键字）。
- 泛型：`struct<T>`/`impl<T>`；struct 有泛型参数时 impl 必须引入同样参数；实例化替换检查。
- 指针：`T&` 可空引用，零值 `null`，成员访问自动解引用，解引用 null 抛 `NullPointerError`。
- `Copyd<T>`：参数传递时深拷贝；`.ptr()` 取出包装值。
- `int` 为 32 位（wrapI32）；越界字面量是编译错误。
- 内建类型：`IOStream`、`HashTable<K,V>`、`Channel`、`Task`（内部）、`MemorizeBuffer`、`Sign`。

## 4. 滚动 List<T>（★ 核心）

双指针缓冲（head 游标/tail 游标）：

- `*l`：取开头（只读，List 专用）。
- `l.next()`：取下一个并滚动；`head()==tail()` 时耗尽，报 `ListExhaustedError`（next 停止并报错）。
- `l.reset()`：head 回到 0。
- `for (x : l)`：语法糖，从头滚到尾。
- `l.append(v)`/`l.size()`/`l.get(i)`/`l.sort()`/`l.contains(v)` 等内建方法。
- `List<int>` 等类型标注；`__sort__` 排序钩子。

## 5. 函数（★ 核心）

- **必须声明返回类型**：`fn f(a int, b String) bool { ... }`（main 可省略，视为 `void`）。
- `return expr;`：返回结果并**结束函数**。
- `log expr;`：记录一条日志并**结束函数**（返回任意值，默认 nil）。
- **没有 `out`、没有多返回值**；多输出返回 `List<T>` 单值。
- 函数调用直接得到返回值：`x int = f(1, "a");`。
- 参数可声明 `Copyd` 修饰（`a int[Copyd]`）触发深拷贝。

## 6. 错误处理

```quark
try {
    y int = 1 / 0;
} catch (e void) {        // catch 必须写名字和类型
    io.println("caught: " + e);
}
```

- catch 变量类型可声明为 `void`（自由）或具体类型；出错时装入错误信息。

## 7. 签名与 Sign 接口（★ 核心）

- `f(args) @mb(prefix)` ≡ `mb.call(prefix)(.{in, out})`：
  - mb 是 Sign 接口的**实例**（`mb memorize = memorize::new();`）；
  - `mb.call(prefix)` 输出一个**函数**，该函数接收 `.{in, out}` 两字段记录，返回结果；
  - **原函数放在 prefix 里**（`prefix.fn`）；按 `in` 记忆化，命中直接填 `out`。
- Sign 接口：`interface { fn call(prefix void, rec void) void; } Sign;`——只要求 `call`；任何类型 `impl Sign {...}` 即成为签名类型。
- 内置：`memorize`（类，实现 Sign，`memorize::new()` 建实例，按 in 记忆化）。
- memorize 的 call 输入为空 → 写作 `@mb()`；带参写作 `@mb(prefix)`。

## 8. struct / impl / interface

```quark
struct {
    x int;
    y int;
} Point;

impl {
    fn translate(self, dx int, dy int) void {
        self.x = self.x + dx;
    }
    fn sum(self) int {
        return self.x + self.y;
    }
    fn new() Point {
        p Point;
        p.x = 3;
        p.y = 4;
        return p;
    }
} Point;
```

- `impl { ... } T;` 匿名实现；`impl Iface { ... } T;` 有名实现（实现接口，缺方法报编译错误）。
- 实例方法首参 `self`；静态方法（如 `new`）用 `T::name()` 调用。
- 泛型 `struct<T>`/`impl<T>` 同理；`T` 在方法参数/返回/成员注解中可引用。

## 9. main 与启动

- `fn main(io IOStream)`（可省略返回类型 = void）；可带 `env HashTable<String,String>`、`args List<String>`。
- io 注入：`io.println(expr, ...)`、`io.print`、`io.readln()`、`io.setOut(FileOutputStream(path))`、`io.setIn`。
- IO 执行表：按到达时间 FIFO，读优先于写；并发 IO 由语言层串行化。

## 10. taskm 线程与内存

```quark
pid int = taskm.spawn();          // 创建线程（无参），直接返回 pid
taskm.merge(pid, add, 3, 4);      // 把函数并入线程执行
taskm.block(pid);                 // 等待线程空闲（返回 void）
taskm.done(pid);                  // 线程是否空闲（没有函数占用）
ch Channel = taskm.channel();     // 默认容量 1024；taskm.channel(n) 指定容量
```

- 内存：block 分配（默认 4096，`memory.setBlock(n)`/`GlobalMemory::setBlock(n)` 动态调粒度）；写时脏标记；协程结束自动标记可回收；`GlobalMemory::compact()` 实际清理（不返回）。

## 11. 宏系统（★ 新）

### 定义与调用

```quark
macro {模式} {主体}
```

- 模式支持任意形式（如 `macro {program ...} {}`），但**模式内不能再嵌套大括号**；`...` 是通配捕获（v1：须在模式末尾，捕获到本语句结束）。
- 调用：按模式匹配源码 token 形态，匹配处替换为展开结果。

### 动态预处理命令（主体内，`#` 开头）

- `#when (compile) { ... }` / `#when (run) { ... }`：按编译期/运行期条件选择块（解释器为 run 态）。
- `#insert(#ast(名字))`：把捕获内容（或符号）插入当前位置。
- `#execute(名字)`：运行期执行（解释器中即普通调用）。
- `#error("消息")`：预处理报错。

### 预制宏

- `program main;`：包装为可运行程序（否则编译/运行结果为空）；`program library;`：编译为库，不可运行（`#when (run) { #error("cannot run a library") }`）。
- `pub`：前缀于函数/结构体，在库中公开（`prog.Pub`）。
- `import 路径;`：按编译/运行选项寻找 imports（同目录默认在搜索范围内）。

## 12. 严格检查与错误诊断

- 编译期：未声明标识符/成员、使用前未初始化、类型不匹配、参数数量/类型、条件非布尔、算术类型、签名注册、泛型替换、接口一致性、重复声明、返回类型缺失——全部静态报错（带行号）。
- 运行期：除零（DivisionByZeroError）、越界、空指针（NullPointerError）、列表耗尽（ListExhaustedError）——可用 `try/catch` 捕获，`log` 记录定位。

## 13. 示例

```quark
fn sq(n int) int {
    return n * n;
}

fn main(io IOStream) {
    mb memorize = memorize::new();
    io.println(sq(41) @mb());          // 1681（记忆化）
    pid int = taskm.spawn();
    taskm.merge(pid, sq, 7);
    taskm.block(pid);
    try {
        io.println(1 / 0);
    } catch (e void) {
        io.println("caught");
    }
}
```

## 14. 已确认决议（2026-08-30）

- FuncBuffer 类型彻底删除；函数返回真实值；out/多返回值移除；void=空接口默认名。
- try/catch(名字 类型)；log 记录并结束函数；return 返回并结束函数。
- `.{...}` 结构体字面量；字段/成员名可为 `in`/`out` 等关键字。
- 签名 `@mb(prefix)` ≡ `mb.call(prefix)(.{in,out})`，原函数在 prefix。
- taskm.spawn() 无参返回 pid；merge(pid,fn,args) 入线程；block 返回 void；done=线程空闲。
- 宏系统：macro{模式}{主体}、动态预处理、program/pub/import 预制宏。
- 编译器后端为 LLVM（不经 C 转译）。

## 15. 实现路线

- ✅ v0.2 解释器：词法/语法/类型检查/求值/运行时；滚动 List、严格检查、struct/impl/interface、泛型/指针/Copyd、内存系统、taskm、签名（@mb()）、宏系统（30 项测试全绿）。
- ✅ v0.2 编译器：LLVM IR 后端（qkc，-run 编译执行；变量/控制流/算术/比较/布尔，7 项测试全绿）。
- ⏳ 宏系统编译期对接（#ast 符号级插入）、import 解析落地、标准库与包管理。
