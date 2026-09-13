# QuarkLang 内建速查表（源码核对版）

> 依据 internal/lang/eval.go（builtins 注册表 + 系统对象）、internal/lang/typecheck.go 与 examples/ 实测。
> 语法为正典语法（类型在前，唯一清单见 `SYNTAX.md`）。核对日期：2026-09-13。

## 1. 内建函数（in.builtins）

| 函数 | 签名 | 说明 |
|---|---|---|
| `clock` | `fn clock() int` | Unix 微秒 |
| `rand` | `fn rand() int` | LCG 确定性伪随机 [0, 2^31-1] |
| `sum` | `fn sum(fn, int from, int to) int` | 闭式 O(1) 求和（线性/周期/均匀随机期望） |
| `ifstream` | `fn ifstream(String path) InputStream` | 打开文件读流 |
| `ofstream` | `fn ofstream(String path) OutputStream` | 打开文件写流 |
| `iofstream` | `fn iofstream(String path) IOStream` | 读写流 |
| `FileInputStream` / `FileOutputStream` | 同上别名 | |
| `ConsoleInputStream` / `ConsoleOutputStream` | （全局控制台流） | |
| `qkexec` | `fn qkexec(String cmd) int` | shell 执行，返回码 |
| `qkexecv` | `fn qkexecv(String prog, List<String> args)` | argv 直传无注入面 |
| `qkpopen` | `fn qkpopen(String cmd) InputStream` | 捕获 stdout（≤8MiB） |
| `qkhttp_get` / `qkhttp_post` | 网络原语（10s 超时、8MiB、重试） | |
| `qkjson_dumps` / `qkjson_loads` | JSON 原语 | |
| `qkcleg_*` | `qkcleg_create/frame/rect/text/clear` | Cleg GUI |
| `qkscreen_*` | `qkscreen_open/present/close` | 屏幕 |

## 2. 类型 / 对象

| 类型 | 说明 |
|---|---|
| `InputStream` / `OutputStream` / `IOStream` | `.readln()` `.close()` |
| `String` 方法集（UTF-8 安全） | `size contains startsWith endsWith indexOf(-1=无) substring(start,end?) split(sep) trim trimLeft trimRight toLower toUpper replace old new charAt(i) toInt toFloat` |
| `List<T>` | 字面量 `[1,2]`；下标 `l[i]`；`size head tail next reset append appendAll toString`；排序钩子 `__sort__` |
| `HashTable<K,V>` | `HashTable::new()`；`put/get/contains/keys/remove/size` |
| `thread` / `Channel` | taskm 并发系统对象；`thread` 有 `pid()`、`merge(fn, args...)` |
| `pointer T` / `T&` | 可空引用（零值 `null`，自动解引用；堆分配由 block 管理） |

## 3. taskm 并发

```qk
thread t = taskm.spawn();         // 创建用户态任务（返回 thread 实例）
t.merge(work, 1);                 // 绑定函数+实参（实例形式）
taskm.merge(t.pid(), work, 1);    // 等价 pid 形式
taskm.block(t.pid());             // 等待完成（返回 void）
bool idle = taskm.done(t.pid());  // 线程是否空闲（没有函数占用）
Channel c = taskm.channel();      // 通道
c.send(v);  v = c.recv();
```

## 4. FFI（library 声明，dlopen+libffi）

```qk
library c { fn strlen(String s) long; fn rand() int; }                   // libc 预绑定
library m { fn sqrt(float x) float; }                                    // libm 预绑定
library gl { fn ClearColor(float r, float g, float b, float a) void; }   // 自定义库（跨系统解析）
```

## 5. 写 qk 常见坑（踩坑记录）

- 声明类型在前：`int x = 1;`、`String s = "a";`、`List<int> l = [1,2];`；函数参数/返回同理：`fn f(int a) bool`；
- `main` 写 `fn main(IOStream io)`（返回类型可省，视为 void；可选 env/args）；
- 实名结构体/接口必须带 `type`：`type struct { int x; } Point;`、`type interface { fn area(Self self) int; } Shape;`；实现写 `impl { ... } Point;`（不写接口名，结构化满足）；
- 空间：`space { fn ... } name;`，调用 `name::fn(...)`；
- List 字面量是 `[1,2]` 不是 `{}`；`HashTable::new()` 不是 `new HashTable()`；
- `break;` 可跳出 `while`/`for`；`for` 两种：`for (int i = 0; i < n; i = i + 1)` 与 `for (int x : l)`；
- String 无 `.get()`（下标 `s[i]`）；
- 库函数直接调用：`import "actions"` 后 `system::exec(...)`（space 名 `::` 函数），包装类 `Command::new(...).exec()`；
- popen 输出 ≤ 8MiB；网络非 2xx 抛错（可用 try/catch）；`String` 越界抛 `StringIndexOutOfBoundsError`；
- 捕获异常类型在前：`catch (void e) { ... }`。
