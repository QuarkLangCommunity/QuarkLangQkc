# QuarkLang 语法清单（与实现同步维护 · 正典语法）

> **唯一正典形态：类型在前。** 旧「名字在前」写法已删除，不保留兼容。
> 本文件随实现同步更新；变更需求 → 先改本文件与实现。
> 编号供逐项核对：M=宏，K=语句，E=表达式，S=结构/泛型/接口，O=对象模型，T=类型，P=编译工具。

## K 声明（类型在前）

| 编号 | 语法 | 说明 |
|---|---|---|
| K1 | `<修饰> <类型> <名字>;` | 声明；修饰见 K2 |
| K2 | `const int N = 3;` `copyd List<int> l = [1,2];` | 修饰：`const`（常量）、`copyd`（传时复制）；修饰在**最前** |
| K3 | `int x = 1;` `Point p;` `List<int> l = [1,2];` | 初始化可省略（零值） |
| K4 | `x = 2;` `p.x = 3;` `l[i] = 1;` | 赋值（变量/成员/下标） |
| K5 | `if (cond) { } else if (cond) { } else { }` | 条件必须是 bool |
| K6 | `while (cond) { }` | |
| K7 | `for (int i = 0; i < n; i = i + 1) { }` | C 风格 for（初始化是类型在前声明） |
| K8 | `for (int x : l) { }` | 迭代 for（循环变量必须带类型） |
| K9 | `break;` | 跳出 while/for |
| K10 | `try { } catch (void e) { }` | catch 变量类型在前（`void` = 自由类型） |
| K11 | `return e;` `log e;` `delete x;` | return 返回并结束；log 记录并结束；delete 回收 |
| K12 | `program main;` `program library;` `import "path";` `pub fn ...` | 预制宏；library 不可运行 |

## S 函数 / 结构体 / 接口 / 实现 / 空间

| 编号 | 语法 | 说明 |
|---|---|---|
| S1 | `fn name(int a, String b) bool { ... }` | 函数；**参数类型在前**；返回类型必填（main 可省略＝void） |
| S2 | `fn<T, U> name(T v) T { ... }` | 泛型函数；调用自动推断：`name(5)` |
| S3 | `type struct { int x; int y; } Point;` | 实名结构体（必须写 `type`）；字段类型在前 |
| S4 | `struct { int x; };` | 匿名结构体类型 |
| S5 | `type struct<T> { T v; } Box;` | 泛型结构体；`impl<T>` 必须引入同样参数 |
| S6 | `type interface { fn sum(Self self) int; expand interface Other; } Iface;` | 接口；`Self` = 自身类型；`expand interface` 组合接口 |
| S7 | `interface { };` | 匿名接口（空接口＝`void`） |
| S8 | `impl { fn new(int x) Point { ... } fn sum(Point self) int { ... } } Point;` | 实现：**唯一形态**（名字在块后）；静态方法无 self，实例方法首参 `self` |
| S9 | `impl<T> { ... } Box;` | 泛型实现 |
| S10 | `space { fn max(int a, int b) int { ... } } math;` | 空间（自我实现）：无括号；**内外调用一律写 `math::max(1, 2)`**（空间内互调同样要带空间名） |
| S11 | `Point::new(3, 4)` `p.sum()` | 静态调用 `::`；实例方法 `.` |
| S12 | `Point p = .{x: 1, y: 2};` | 结构体字面量 |
| S13 | `fn __add__(Point self, Point other) Point { ... }` | 运算符方法名：`__add__ __sub__ __mul__ __div__ __mod__ __neg__ __eq__ __ne__ __lt__ __le__ __gt__ __ge__`（`a + b` 自动分发） |
| S14 | 接口**结构化满足**：类型方法齐全即满足接口，`impl` 上不再写接口名 | 赋值/传参处检查 |

## T 类型

| 编号 | 类型 | 说明 |
|---|---|---|
| T1 | `int`（32 位 wrap） `long` `char` `float` `bool` `String` | 标量 |
| T2 | `List<T>` `HashTable<K,V>` | 容器 |
| T3 | `T&` / `pointer T` | 可空引用，零值 `null` |
| T4 | `Copyd<T>` / `int[Copyd]` / `int[]` | 传时复制语义 |
| T5 | `void` `interface{}` `IOStream` `thread` `memorize` `memory` | 内建 |

## E 表达式

| 编号 | 语法 | 说明 |
|---|---|---|
| E1 | `123` `-7` `1.5` `"str"` `true` `.{x: 1}` | 字面量；`\``...\``` 原始字符串（Go 语义，不转义） |
| E2 | `f(x)` `obj.m(x)` `T::m(x)` `space::f(x)` `p.x` `l[i]` `*l` | 调用/成员/下标/取头 |
| E3 | `+ - * / %` `<< >>` `== != < <= > >=` `&& || !` | 运算符；`+` 支持 String 拼接 |
| E4 | `s.size() contains startsWith endsWith indexOf substring split trim toLower toUpper replace charAt toInt toFloat` | String 内建方法（rune 语义；`indexOf` 返回字节下标，-1=无） |
| E5 | `l.size()` `l.head()` `l.tail()` `l.get(i)` `l.next()` `l.reset()` `l.append(v)` `l.appendAll(l2)` `l.toString()` `l.__sort__()` `l[i]` `*l` | List（滚动双游标；`get(i)` 与 `l[i]` 同义；取头也可用 `*l`） |
| E6 | `h.put(k, v)` `h.get(k)` `h.contains(k)` `h.keys()` `h.remove(k)` `h.size()` | HashTable（缺键 `get` 返回 nil） |
| E7 | `f(args) @mb()` | 签名调用（Sign 实例） |

## M 宏与预处理

| 编号 | 语法 | 说明 |
|---|---|---|
| M1 | `#macro name (a, b) { 主体 }` | 命名参数宏；`()`/`[]`/`{}` 分隔符可互换；单趟展开，不递归 |
| M2 | `#when (run) { ... }` / `#when (compile) { ... }` | 态选择（解释器走 run） |
| M3 | `#error("msg")` `#return <expr>` `#insert(...)` `#execute(名字)` | 预处理命令 |
| M4 | `#` 开头的自定义命令 | 见内部宏表 |
| M5 | `library name { fn sym(int a, String b) int; }` | FFI 库绑定（参数同函数，类型在前）；方法名即符号名 |

## P 工具链

| 编号 | 行为 |
|---|---|
| P1 | `quark file.qk` 解释执行（`qkd` 断点调试） |
| P2 | `qkc file.qk` 输出 LLVM IR；`qkc -run file.qk` 编译执行（原生路径为正典） |
| P3 | `qkm init/build/debug/install/update` 工程与依赖管理 |
| P4 | 两条路径**语法与语义一致**（同一份源码到处可用） |
| P5 | `qkcheck [-json] [-params] [-L dir] files...` 静态检查：未使用变量/形参、不可达代码、遮蔽、缺返回、接口未实现、void 值误用（码 QK101–QK107） |
| P6 | `qkdoc [-o file] [-html] [-all] files...` API 文档：声明上方 `/* */`/`//` 注释 + `pub` 导出 → Markdown/HTML |
| P7 | `qkrepl [-e code] [-q]` 交互式求值：多行块续行、跨输入持久环境（变量/函数/类型）、`:help/:load/:quit` |
| P9 | `scripts/build-release.sh <版本>` 发布产物（三平台二进制 + 版本注入 + sha256 清单）；`scripts/changelog.sh` 变更日志；`v*` tag 触发 release 工作流 |
| P8 | `qklsp [-L dir]` 语言服务器（stdio LSP）：诊断/跳转定义/补全/悬停/文档符号；编辑器接入见 `editors/`（VS Code 扩展 + tree-sitter 语法） |
