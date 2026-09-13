# QuarkLang 语言规范（v2 · 正典语法）

> 本文档与代码同步维护。**语法唯一正典清单见主工作区 `SYNTAX.md`**：声明一律**类型在前**（`<修饰> <类型> <名字>`），旧「名字在前」写法已删除、**不保留兼容**（见 §16 迁移说明）。
>
> v2（2026-08-30）相比 v1 的修订：**FuncBuffer 类型彻底删除**；函数必须声明返回类型，`return` 返回并结束、`log` 记录并结束；`out` 与多返回值移除（多输出用 `List<T>`）；`void` 成为空接口 `interface{}` 的默认名字；错误处理改为 `try/catch(类型 名字)`；结构体字面量 `.{...}`；签名改为实例形式 `f(args) @instance(prefix)`；新增**宏系统**。
>
> 2026-09 正典语法落地：解释器已按类型在前重做词法/语法/类型检查/求值并实测通过（`examples/*.qk` 全绿）；编译器前端正在统一为与解释器**同一套 AST**，语义 lower 进行中。实名结构体/接口必须写 `type`；`impl` 唯一形态 `impl<T, ...> { ... } 名字;`（接口结构化满足，impl 上不写接口名）；空间 `space { ... } 名字;`；`for` 两种形态；`catch (void e)`；运算符方法名 `__add__` 等。

## 0. 术语表

- **宏（macro）**：`#macro name (p1, p2) { 主体 }` 定义的预处理单元；实参 token 按形参名替换，单趟展开、不递归。
- **预处理命令**：`#` 开头、在宏主体中动态执行（`#when`/`#return`/`#error` 等）。
- **预制宏**：`program`/`import`/`pub` 等语言内置宏。
- **签名（Sign）**：`@instance(prefix)` 形式的调用包装；instance 是 Sign 接口的实例。
- **线程（taskm）**：`taskm.spawn()` 创建（返回 `thread` 实例）；`merge` 并入函数执行。

## 1. 设计目标

面向 CLI 工具的语言：语法无新概念，编译期严格，运行期错误可定位（log + try/catch），一套设计两套实现（Go 解释器 + LLVM 编译器），带宏系统对接编译期与运行期。

## 2. 词法与程序结构

- 注释 `//`、`/* */`；标识符；整数/浮点/字符串字面量（`\n \t \r \" \\` 转义）；反引号原始字符串（Go 语义，不转义）。
- 关键字：`fn type struct interface impl space expand const copyd return log delete if else while for break try catch true false null void pointer Self program library import pub`。
- `#` 开头为预处理命令。
- 程序 = 若干顶层声明（函数 / 实名类型 / 实现 / 空间 / 预制宏） + `#macro` 定义。
- **声明正典形态**：`<修饰> <类型> <名字> [= 初值];`；`const`/`copyd` 修饰写在**最前**（如 `const int N = 3;`、`copyd List<int> l = [1,2];`）；初始化可省略（零值）。修饰声明当前在函数体内使用。

## 3. 类型系统

- 变量声明：`int x = 1;`、`Point p;`、`List<int> l = [1,2];`；初始化可省略（零值）；赋值支持变量/成员/下标（`x = 2;`、`p.x = 3;`、`l[i] = 1;`）。
- 修饰：`const`（常量）、`copyd`（传时复制），写在类型前面。
- `void` = 空接口 `interface{}` 的默认名字（语言预声明 `type interface { } void;`）——"不是空，是没有接口"。
- `interface { ... }` 匿名接口；**实名接口必须写 `type`**：`type interface { ... } 名字;`。
- `struct { ... }` 匿名结构体类型；**实名结构体必须写 `type`**：`type struct { ... } 名字;`。
- 接口内可 `expand interface 其他接口;` 组合接口；`Self` 表示自身类型。
- 结构体字面量：`.{field: value, ...}`（字段名允许 `in`/`out` 等关键字）。
- 泛型：`type struct<T> { ... } Box;` / `impl<T> { ... } Box;`；struct 有泛型参数时 impl 必须引入同样参数；实例化替换检查。泛型调用自动推断（`id(5)`）；**显式泛型实参暂不支持**。
- 指针：`T&` / `pointer T` 可空引用，零值 `null`，成员访问自动解引用，解引用 null 抛 `NullPointerError`。
- `Copyd<T>`：参数传递时深拷贝，`.ptr()` 取出包装值；参数写 `int[Copyd] a`，局部声明写 `copyd List<int> l = [1,2];`。
- `int` 为 32 位（wrapI32）；越界字面量是编译错误。
- 内建类型：`List<T>`、`HashTable<K,V>`、`IOStream`、`Channel`、`thread`（`Task` 内部）、`memorize`、`memory`、`Sign`。

## 4. 滚动 List<T>（★ 核心）

双指针缓冲（head 游标/tail 游标）：

- `*l`：取开头（只读，List 专用）。
- `l.next()`：取下一个并滚动；`head()==tail()` 时耗尽，报 `ListExhaustedError`（next 停止并报错）。
- `l.reset()`：head 回到 0。
- `for (int x : l) { ... }`：语法糖，从头滚到尾；**循环变量必须带类型**。
- `l[i]` 下标；`l.append(v)`/`l.appendAll(xs)`/`l.size()`/`l.head()`/`l.tail()`/`l.toString()` 等内建方法。
- `List<int>` 等类型标注；`__sort__` 排序钩子。

## 5. 函数（★ 核心）

```quark
fn f(int a, String b) bool { ... }      // 参数类型在前；返回类型必填
fn main(IOStream io) { ... }            // main 可省略返回类型，视为 void
fn<T> id(T v) T { return v; }           // 泛型函数；调用自动推断：int x = id(5);
```

- `return expr;`：返回结果并**结束函数**。
- `log expr;`：记录一条日志并**结束函数**（返回任意值，默认 nil）。
- **没有 `out`、没有多返回值**；多输出返回 `List<T>` 单值。
- 函数调用直接得到返回值：`int x = f(1, "a");`。
- 参数可声明 `Copyd`（`int[Copyd] a`）触发深拷贝；也可用 `copyd` 修饰局部声明。

## 6. 错误处理

```quark
try {
    int y = 1 / 0;
} catch (void e) {        // catch 变量类型在前
    io.println("caught: " + e);
}
```

- `catch (类型 名字)`：类型可写 `void`（自由类型）或具体类型；出错时装入错误信息。

## 7. 签名与 Sign 接口（★ 核心）

- `f(args) @instance(prefix)` ≡ `instance.call(prefix)(.{in, out})`：
  - instance 是 Sign 接口的**实例**，名字任意（如 `memorize memo = memorize::new();`）；
  - `instance.call(prefix)` 输出一个**函数**，该函数接收 `.{in, out}` 两字段记录，返回结果；
  - **原函数放在 prefix 里**（`prefix.fn`）；按 `in` 记忆化，命中直接填 `out`。
- Sign 接口（正典写法）：`type interface { fn call(void prefix, void rec) void; } Sign;`——只要求 `call`；接口**结构化满足**，任何类型方法齐全即为签名类型（`impl` 上不写接口名）。
- 内置：`memorize`（类，满足 Sign，`memorize::new()` 建实例，按 in 记忆化）。
- memorize 的 call 输入为空 → 写作 `@instance()`；带参写作 `@instance(prefix)`（`@` 后跟任意实例名）。

## 8. struct / impl / interface

```quark
type struct {
    int x;
    int y;
} Point;

impl {
    fn translate(Point self, int dx, int dy) void {
        self.x = self.x + dx;
    }
    fn sum(Point self) int {
        return self.x + self.y;
    }
    fn new(int x, int y) Point {
        Point p;
        p.x = x;
        p.y = y;
        return p;
    }
} Point;
```

- **实现唯一形态**：`impl<T, ...> { 方法 } 名字;`（名字写在块后）；`impl` 上**不写接口名**——接口结构化满足，类型方法齐全即满足接口，在赋值/传参处检查。
- 实例方法首参为 `Self`/具体类型（约定名 `self`），调用 `p.sum()`；静态方法无 self（如 `new`），调用 `Point::new(3, 4)`。
- 泛型：`type struct<T> { ... } Box;` + `impl<T> { ... } Box;`；`T` 在方法参数/返回/成员注解中可引用。

## 9. main 与启动

- `fn main(IOStream io)`（返回类型可省略 = void）；可带 `HashTable<String,String> env`、`List<String> args`。
- io 注入：`io.println(expr, ...)`、`io.print`、`io.readln()`、`io.setOut(FileOutputStream(path))`、`io.setIn`。
- IO 执行表：按到达时间 FIFO，读优先于写；并发 IO 由语言层串行化。

## 10. taskm 线程与内存

```quark
thread t = taskm.spawn();           // 创建线程（无参），返回 thread 实例
t.merge(work, 1);                   // 实例形式：把函数并入线程执行
taskm.merge(t.pid(), work, 1);      // 等价 pid 形式：merge(pid, fn, args...)
taskm.block(t.pid());               // 等待线程空闲（返回 void）
bool idle = taskm.done(t.pid());    // 线程是否空闲（没有函数占用）
Channel ch = taskm.channel();       // 默认容量 1024；taskm.channel(n) 指定容量
ch.send(v); v = ch.recv();          // 通道收发
```

- 内存：block 分配（默认 4096，`memory.setBlock(n)`/`GlobalMemory::setBlock(n)` 动态调粒度）；写时脏标记；协程结束自动标记可回收；`GlobalMemory::compact()` 实际清理（不返回）。

## 11. 宏系统（命名参数宏）

### 定义

```quark
#macro name (p1, p2, ...) { 主体 }
```

- 参数写在 `()` 内、逗号分隔、**个数不限**；参数列表分隔符 `()`/`[]`/`{}` 任意互换（`#macro name [a, b] { ... }`、`#macro name {a, b} { ... }` 均可）。
- 主体同样可用 `()`/`[]`/`{}` 之一包裹（`#macro name (x) [x]` 亦合法）。

### 调用

```quark
name(arg1, arg2, ...)      // 分隔符 () / [] / {} 任意
```

- 调用形式与定义同形、分隔符可互换：`name(...)`、`name[...]`、`name{...}`。
- 实参个数必须与形参个数一致（不限量指形参个数任意）；主体内形参名按名替换为实参 token。
- 展开先于解析：`name(...)` 形态只要 name 是宏即展开；声明位置（`fn`/`struct`/`impl`/`interface` 后）与 `.`/`::` 成员访问后不展开。
- 宏体不递归再展开（单趟展开）。

### 动态预处理命令（主体内，`#` 开头）

- `#when (compile) { ... }` / `#when (run) { ... }`：编译态/运行态选块（解释器为 run 态）。
- `#return 表达式`：从展开处返回值（如 `examples/macro.qk` 的 `#return a + b`）。
- `#error("消息")`：预处理报错。
- `#insert(#ast(...))` / `#execute(...)` / `#exec`：随旧 pattern 语法移除。

### 预制宏

- `program main;`：包装为可运行程序（否则可运行空）；`program library;`：编译为库，不可运行。
- `pub`：前缀于函数/结构体，在库中公开（如 `pub fn f(...) ...`）。
- `import 路径;`：按编译/运行选项寻找 imports（同目录默认在搜索范围内）。

## 12. 严格检查与错误诊断

- 编译期：未声明标识符/成员、使用前未初始化、类型不匹配、参数数量/类型、条件非布尔、算术类型、签名注册、泛型替换、接口一致性、重复声明、返回类型缺失、旧「名字在前」写法——全部静态报错（带行号）。
- 运行期：除零（DivisionByZeroError）、越界、空指针（NullPointerError）、列表耗尽（ListExhaustedError）——可用 `try/catch` 捕获，`log` 记录定位。

## 13. 示例

```quark
fn sq(int n) int {
    return n * n;
}

fn main(IOStream io) {
    memorize memo = memorize::new();
    io.println(sq(41) @memo());          // 1681（记忆化）
    thread t = taskm.spawn();
    t.merge(sq, 7);
    taskm.block(t.pid());
    try {
        io.println(1 / 0);
    } catch (void e) {
        io.println("caught");
    }
}
```

## 14. 已确认决议

### v2 决议（2026-08-30）

- FuncBuffer 类型彻底删除；函数返回真实值；out/多返回值移除；void=空接口默认名。
- try/catch(类型 名字)；log 记录并结束函数；return 返回并结束函数。
- `.{...}` 结构体字面量；字段/成员名可为 `in`/`out` 等关键字。
- 签名 `@instance(prefix)` ≡ `instance.call(prefix)(.{in,out})`，原函数在 prefix。
- taskm.spawn() 无参返回 thread；merge(fn,args) 入线程；block 返回 void；done=线程空闲。
- 宏系统：`#macro name (p1, p2) { 主体 }`、动态预处理、program/pub/import 预制宏。
- 编译器后端为 LLVM（不经 C 转译）。

### 正典语法决议（2026-09-13）

- 声明一律**类型在前**：`<修饰> <类型> <名字> [= 初值];`；旧「名字在前」写法全部删除，**不保留兼容**。
- 函数：`fn<T, ...> 名字(<类型> <名字>, ...) <返回> { ... }`；`main` 可省返回类型。
- 实名结构体/接口必须写 `type`；匿名形态不写名字。
- 实现唯一形态：`impl<T, ...> { ... } 名字;`；接口**结构化满足**，`impl` 上不写接口名。
- 空间：`space { fn ... } 名字;`（无括号）；调用 `名字::fn(...)`。
- 语句：`for (int i = 0; i < n; i = i + 1)` 与 `for (int x : l)`；`catch (void e)`。
- 运算符方法名：`__add__ __sub__ __mul__ __div__ __mod__ __neg__ __eq__ __ne__ __lt__ __le__ __gt__ __ge__`。
- 泛型调用自动推断（`id(5)`）；显式泛型实参暂不支持。
- FFI：`library name { fn sym(int a, String b) int; }`（参数类型在前）。

## 15. 实现路线

- ✅ 解释器：**正典语法已实现并实测通过**（词法/语法/类型检查/求值/运行时；滚动 List、严格检查、struct/impl/interface、泛型/指针/Copyd、内存系统、taskm、签名、宏系统；`examples/*.qk` 全绿）。
- 🚧 编译器前端：正在统一为与解释器**同一套 AST**（类型在前、`type` 实名类型、`impl ... 名字;`、`space` 等），语义 lower 进行中。
- ✅ 编译器后端：LLVM IR（qkc，`-run` 编译执行；变量/控制流/算术/比较/布尔）。
- ⏳ 宏系统编译期对接（`#ast` 符号级插入）、import 解析落地、标准库与包管理。

## 16. 迁移说明（旧写法已删除）

> 下表仅用于迁移对照；**旧写法已从语言中删除，不保留兼容**。

| 旧写法（已删除） | 正典写法 |
|---|---|
| `x int = 1;` | `int x = 1;` |
| `fn f(a int, b String) bool` | `fn f(int a, String b) bool` |
| `func f(...)` | `fn f(...)` |
| `struct { x int; } Point;` | `type struct { int x; } Point;` |
| `interface { ... } Name;` | `type interface { ... } Name;` |
| `impl Iface { ... } T;` | `impl { ... } T;`（接口结构化满足，impl 不写接口名） |
| `space Name { ... }` / `space (Name) { ... }` | `space { ... } Name;` |
| `space.fn(...)` / `Name.fn(...)` 调空间函数 | `Name::fn(...)` |
| `for (x : l)` | `for (int x : l)` |
| `catch (e void)` | `catch (void e)` |
| 运算符方法 `add`/`sub`/`mul`/`div`/`mod`/`neg`/`eq`/`ne`/`lt`/`le`/`gt`/`ge` | `__add__`/`__sub__`/`__mul__`/`__div__`/`__mod__`/`__neg__`/`__eq__`/`__ne__`/`__lt__`/`__le__`/`__gt__`/`__ge__` |
| `macro {模式} {主体}` | `#macro name (p1, p2) { 主体 }` |
| `impl Sign { ... } 类名;` / 显式接口实现 | 方法齐全即满足接口（结构化满足） |
