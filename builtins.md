# QuarkLang built-ins quick reference (verified against the sources)

**English** · [简体中文](builtins.zh-CN.md)

> Based on `internal/lang/eval.go` (builtin registry + system objects), `internal/lang/typecheck.go` and
> verified against `examples/`. Syntax is canonical (type-first; the single checklist is `SYNTAX.md`).
> Verified on 2026-09-13.

## 1. Built-in functions (`in.builtins`)

| Function | Signature | Notes |
|---|---|---|
| `clock` | `fn clock() int` | Unix microseconds |
| `rand` | `fn rand() int` | deterministic LCG pseudo-random in [0, 2^31-1] |
| `sum` | `fn sum(fn, int from, int to) int` | closed-form O(1) summation (linear/periodic/uniform-random expectation) |
| `ifstream` | `fn ifstream(String path) InputStream` | open a file for reading |
| `ofstream` | `fn ofstream(String path) OutputStream` | open a file for writing |
| `iofstream` | `fn iofstream(String path) IOStream` | read/write stream |
| `FileInputStream` / `FileOutputStream` | aliases of the above | |
| `ConsoleInputStream` / `ConsoleOutputStream` | (global console streams) | |
| `qkexec` | `fn qkexec(String cmd) int` | run through the shell, returns the exit code |
| `qkexecv` | `fn qkexecv(String prog, List<String> args)` | argv passed directly — no injection surface |
| `qkpopen` | `fn qkpopen(String cmd) InputStream` | capture stdout (≤ 8 MiB) |
| `qkhttp_get` / `qkhttp_post` | network primitives (10 s timeout, 8 MiB, retries) | |
| `qkjson_dumps` / `qkjson_loads` | JSON primitives | |
| `qkcleg_*` | `qkcleg_create/frame/rect/text/clear` | Cleg GUI |
| `qkscreen_*` | `qkscreen_open/present/close` | screen |

## 2. Types / objects

| Type | Notes |
|---|---|
| `InputStream` / `OutputStream` / `IOStream` | `.readln()` `.close()` |
| `String` method set (UTF-8 safe) | `size contains startsWith endsWith indexOf(-1=absent) substring(start,end?) split(sep) trim trimLeft trimRight toLower toUpper replace old new charAt(i) toInt toFloat` |
| `List<T>` | literal `[1,2]`; indexing `l[i]`; `size head tail next reset append appendAll toString`; sorting hook `__sort__` |
| `HashTable<K,V>` | `HashTable::new()`; `put/get/contains/keys/remove/size` |
| `thread` / `Channel` | taskm concurrency objects; `thread` has `pid()` and `merge(fn, args...)` |
| `pointer T` / `T&` | nullable references (zero value `null`, automatic dereference; heap allocation managed by blocks) |

## 3. taskm concurrency

```qk
thread t = taskm.spawn();         // create a user-space task (returns a thread instance)
t.merge(work, 1);                 // bind a function + arguments (instance form)
taskm.merge(t.pid(), work, 1);    // equivalent pid form
taskm.block(t.pid());             // wait for completion (returns void)
bool idle = taskm.done(t.pid());  // is the thread idle (no function occupying it)?
Channel c = taskm.channel();      // channel
c.send(v);  v = c.recv();
```

## 4. FFI (`library` declarations, dlopen + libffi)

```qk
library c { fn strlen(String s) long; fn rand() int; }                   // libc pre-bound
library m { fn sqrt(float x) float; }                                    // libm pre-bound
library gl { fn ClearColor(float r, float g, float b, float a) void; }   // custom library (cross-platform resolution)
```

## 5. Common pitfalls when writing qk (field notes)

- Declarations are type-first: `int x = 1;`, `String s = "a";`, `List<int> l = [1,2];`; the same for parameters
  and return types: `fn f(int a) bool`;
- `main` is `fn main(IOStream io)` (the return type may be omitted, treated as void; `env`/`args` optional);
- Named structs/interfaces require `type`: `type struct { int x; } Point;`,
  `type interface { fn area(Self self) int; } Shape;`; the impl is `impl { ... } Point;` (no interface name —
  structural satisfaction);
- Spaces: `space { fn ... } name;`, called as `name::fn(...)`;
- List literals are `[1,2]`, not `{}`; hash tables are `HashTable::new()`, not `new HashTable()`;
- `break;` leaves `while`/`for`; there are two `for` forms: `for (int i = 0; i < n; i = i + 1)` and `for (int x : l)`;
- `String` has no `.get()` (use indexing `s[i]`);
- Library functions are called directly: after `import "actions"`, `system::exec(...)` (space name + `::`);
  wrapper classes are `Command::new(...).exec()`;
- `popen` output is capped at 8 MiB; non-2xx network responses raise (catch with try/catch);
  out-of-range `String` access raises `StringIndexOutOfBoundsError`;
- The catch type comes first: `catch (void e) { ... }`.
