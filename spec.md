# QuarkLang Language Specification (v2 · canonical syntax)

**English** · [简体中文](spec.zh-CN.md)

> This document is maintained alongside the code. **The single canonical syntax checklist lives in the main
> workspace as `SYNTAX.md`**: declarations are always **type-first** (`<modifier> <type> <name>`); the old
> "name-first" spelling has been removed with **no backwards compatibility** (see §16 migration notes).
>
> v2 (2026-08-30) revisions over v1: the **FuncBuffer type was removed entirely**; functions must declare a
> return type, `return` returns and ends, `log` records and ends; `out` and multiple return values were removed
> (use `List<T>` for multiple outputs); `void` became the default name of the empty interface `interface{}`;
> error handling became `try/catch(<type> <name>)`; struct literals are `.{...}`; signatures became the instance
> form `f(args) @instance(prefix)`; a **macro system** was added.
>
> 2026-09 canonical-syntax landing: the interpreter was rewritten (lexer/parser/type checker/evaluator) for
> type-first syntax and passes end-to-end tests (`examples/*.qk` all green); the compiler front end is being
> unified onto the **same AST** as the interpreter, with semantic lowering in progress. Named structs/interfaces
> require `type`; `impl` has exactly one form `impl<T, ...> { ... } Name;` (interfaces are satisfied structurally,
> so `impl` never names an interface); spaces are `space { ... } Name;`; two `for` forms; `catch (void e)`;
> operator method names such as `__add__`.

## 0. Glossary

- **Macro**: a preprocessing unit defined by `#macro name (p1, p2) { body }`; argument tokens are substituted by
  parameter name, expanded in a single pass and not recursively.
- **Preprocessor command**: starts with `#`; executed dynamically inside a macro body (`#when`/`#return`/`#error`, …).
- **Predefined macro**: language built-ins such as `program`/`import`/`pub`.
- **Signature (Sign)**: a call wrapper of the form `@instance(prefix)`; `instance` is an instance of the Sign interface.
- **Thread (taskm)**: created with `taskm.spawn()` (returns a `thread` instance); `merge` runs a function on it.

## 1. Design goals

A language for CLI tools: no new concepts in the syntax, strict at compile time, locatable runtime errors
(`log` + `try/catch`), one design with two implementations (Go interpreter + LLVM compiler), and a macro system
bridging compile time and run time.

## 2. Lexical structure and program shape

- Comments `//`, `/* */`; identifiers; integer/float/string literals (`\n \t \r \" \\` escapes); backtick raw
  strings (Go semantics, no escapes).
- Keywords: `fn type struct interface impl space expand const copyd return log delete if else while for break try catch true false null void pointer Self program library import pub`.
- `#` at the start of a line introduces a preprocessor command.
- A program = top-level declarations (functions / named types / impls / spaces / predefined macros) + `#macro` definitions.
- **Canonical declaration form**: `<modifier> <type> <name> [= initializer];`; the `const`/`copyd` modifiers are
  written **first** (e.g. `const int N = 3;`, `copyd List<int> l = [1,2];`); the initializer may be omitted (zero value).
  Modified declarations are currently used inside function bodies.
- **There is no `let`/`var` or `const` keyword declaration**: `const`/`copyd` are modifiers, not declaration
  keywords, and must be written together with the type before the name. `let x = 1;`, `var y = 2;` and
  `const N = 3;` (no type) are all invalid; the canonical spellings are `int x = 1;` and `const int N = 3;`.
  Specifications, examples, templates, completions and documentation must never use `let`/`var` forms.

## 3. Type system

- Variable declarations: `int x = 1;`, `Point p;`, `List<int> l = [1,2];`; the initializer may be omitted (zero
  value); assignment supports variables/members/indices (`x = 2;`, `p.x = 3;`, `l[i] = 1;`).
- Modifiers: `const` (constant), `copyd` (copy-on-pass), written before the type.
- `void` = the default name of the empty interface `interface{}` (the language predeclares
  `type interface { } void;`) — "not empty, but no interface".
- `interface { ... }` is an anonymous interface; **a named interface requires `type`**:
  `type interface { ... } Name;`.
- `struct { ... }` is an anonymous struct type; **a named struct requires `type`**: `type struct { ... } Name;`.
- An interface may compose others with `expand interface Other;`; `Self` denotes the type itself.
- Struct literals: `.{field: value, ...}` (field names may be keywords such as `in`/`out`).
- Generics: `type struct<T> { ... } Box;` / `impl<T> { ... } Box;`; when a struct has generic parameters, its
  `impl` must introduce the same ones; instantiation is checked by substitution. Generic call sites infer
  automatically (`id(5)`); **explicit generic arguments are not supported yet**.
- Pointers: `T&` / `pointer T` are nullable references with zero value `null`; member access dereferences
  automatically; dereferencing null raises `NullPointerError`.
- `Copyd<T>`: deep-copies on parameter passing; `.ptr()` unwraps; parameters are written `int[Copyd] a`, local
  declarations `copyd List<int> l = [1,2];`.
- `int` is 32-bit (wrapI32); out-of-range literals are compile errors.
- Built-in types: `List<T>`, `HashTable<K,V>`, `IOStream`, `Channel`, `thread` (internal `Task`), `memorize`,
  `memory`, `Sign`.

## 4. Rolling List<T> (★ core)

A two-cursor buffer (head cursor / tail cursor):

- `*l`: the head (read-only, List-only).
- `l.next()`: returns the next element and rolls forward; when `head()==tail()` the list is exhausted and
  `ListExhaustedError` is raised (`next` stops and reports).
- `l.reset()`: moves head back to 0.
- `for (int x : l) { ... }`: syntactic sugar that rolls from head to tail; **the loop variable must carry a type**.
- `l[i]` indexing; built-ins `l.append(v)` / `l.appendAll(xs)` / `l.size()` / `l.head()` / `l.tail()` / `l.toString()`.
- Type annotations such as `List<int>`; the `__sort__` sorting hook.

## 5. Functions (★ core)

```quark
fn f(int a, String b) bool { ... }      // parameter types first; the return type is required
fn main(IOStream io) { ... }            // main may omit it, treated as void
fn<T> id(T v) T { return v; }           // generic function; calls infer: int x = id(5);
```

- `return expr;`: returns the result and **ends the function**.
- `log expr;`: records a log entry and **ends the function** (returning whatever, nil by default).
- **There is no `out` and no multiple return values**; multiple outputs are returned as a single `List<T>`.
- A call evaluates directly to the returned value: `int x = f(1, "a");`.
- Parameters may be declared `Copyd` (`int[Copyd] a`) to trigger a deep copy; local declarations can use the
  `copyd` modifier instead.

## 6. Error handling

```quark
try {
    int y = 1 / 0;
} catch (void e) {        // the catch variable is type-first
    io.println("caught: " + e);
}
```

- `catch (<type> <name>)`: the type may be `void` (free type) or a concrete type; on error it receives the error
  information.

## 7. Signatures and the Sign interface (★ core)

- `f(args) @instance(prefix)` ≡ `instance.call(prefix)(.{in, out})`:
  - `instance` is an **instance** of the Sign interface with any name
    (e.g. `memorize memo = memorize::new();`);
  - `instance.call(prefix)` yields a **function** that takes a `. {in, out}` two-field record and returns a result;
  - the **original function lives in the prefix** (`prefix.fn`); results are memoized by `in`, and a hit fills `out`.
- The Sign interface (canonical): `type interface { fn call(void prefix, void rec) void; } Sign;` — only `call`
  is required; interfaces are satisfied **structurally**, so any type with the right methods is a signature type
  (`impl` never names an interface).
- Built-in: `memorize` (a class satisfying Sign; `memorize::new()` creates an instance; memoizes by `in`).
- With empty `call` input, write `@instance()`; with arguments, `@instance(prefix)` (any instance name after `@`).

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

- **The only impl form**: `impl<T, ...> { methods } Name;` (name after the block); an `impl` **never names an
  interface** — interfaces are satisfied structurally: a type with all the methods satisfies the interface,
  checked at assignment/call sites.
- Instance methods take `Self`/the concrete type as their first parameter (conventionally `self`) and are called
  as `p.sum()`; static methods take no `self` (such as `new`) and are called as `Point::new(3, 4)`.
- Generics: `type struct<T> { ... } Box;` + `impl<T> { ... } Box;`; `T` may appear in method parameters, return
  types and field annotations.

## 9. main and startup

- `fn main(IOStream io)` (the return type may be omitted = void); it may also take
  `HashTable<String,String> env` and `List<String> args`.
- io injection: `io.println(expr, ...)`, `io.print`, `io.readln()`, `io.setOut(FileOutputStream(path))`, `io.setIn`.
- IO execution: FIFO by arrival time, reads before writes; concurrent IO is serialized by the language layer.

## 10. taskm threads and memory

```quark
thread t = taskm.spawn();           // create a thread (no arguments), returns a thread instance
t.merge(work, 1);                   // instance form: run the function on that thread
taskm.merge(t.pid(), work, 1);      // equivalent pid form: merge(pid, fn, args...)
taskm.block(t.pid());               // wait until the thread is idle (returns void)
bool idle = taskm.done(t.pid());    // is the thread idle (no function occupying it)?
Channel ch = taskm.channel();       // default capacity 1024; taskm.channel(n) sets it
ch.send(v); v = ch.recv();          // channel send/receive
```

- Memory: block allocation (4096 by default; `memory.setBlock(n)` / `GlobalMemory::setBlock(n)` adjust the
  granularity); dirty marking on write; blocks are marked reclaimable when a task ends;
  `GlobalMemory::compact()` performs the actual cleanup (and does not return).

## 11. Macro system (named-parameter macros)

### Definition

```quark
#macro name (p1, p2, ...) { body }
```

- Parameters go inside `()`, comma-separated, with **no limit on their number**; the delimiter may be `()`, `[]`
  or `{}` interchangeably (`#macro name [a, b] { ... }` and `#macro name {a, b} { ... }` are both valid).
- The body may likewise be wrapped in any of `()`/`[]`/`{}` (`#macro name (x) [x]` is legal).

### Invocation

```quark
name(arg1, arg2, ...)      // any of () / [] / {}
```

- The call mirrors the definition and the delimiter is interchangeable: `name(...)`, `name[...]`, `name{...}`.
- The argument count must match the parameter count (unlimited means any number of parameters); parameter names
  inside the body are replaced by the argument tokens.
- Expansion happens before parsing: any `name(...)` whose `name` is a macro expands; declaration positions
  (after `fn`/`struct`/`impl`/`interface`) and member accesses after `.`/`::` do not expand.
- A macro body is not re-expanded (single pass).

### Dynamic preprocessor commands (inside a body, starting with `#`)

- `#when (compile) { ... }` / `#when (run) { ... }`: choose a block for compile state / run state (the
  interpreter is in run state).
- `#return expr`: returns from the expansion site (e.g. `#return a + b` in `examples/macro.qk`).
- `#error("message")`: raises a preprocessor error.
- `#insert(#ast(...))` / `#execute(...)` / `#exec`: removed together with the legacy pattern syntax.

### Predefined macros

- `program main;`: wraps the file as a runnable program (otherwise it may run empty); `program library;`:
  compiles as a library that cannot be run.
- `pub`: prefixes functions/structs to expose them from a library (e.g. `pub fn f(...) ...`).
- `import path;`: resolves imports according to compile/run options (the same directory is searched by default).

## 12. Strict checking and diagnostics

- Compile time: undeclared identifiers/members, use before initialization, type mismatches, argument count/type,
  non-boolean conditions, arithmetic types, signature registration, generic substitution, interface consistency,
  duplicate declarations, missing return types, legacy "name-first" spellings — all reported statically (with line numbers).
- Run time: division by zero (`DivisionByZeroError`), out-of-range access, null pointers (`NullPointerError`),
  exhausted lists (`ListExhaustedError`) — catchable with `try/catch`, locatable via `log`.

## 13. Example

```quark
fn sq(int n) int {
    return n * n;
}

fn main(IOStream io) {
    memorize memo = memorize::new();
    io.println(sq(41) @memo());          // 1681 (memoized)
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

## 14. Settled decisions

### v2 decisions (2026-08-30)

- The FuncBuffer type was removed entirely; functions return real values; out/multiple returns were removed;
  void is the default name of the empty interface.
- `try/catch(<type> <name>)`; `log` records and ends the function; `return` returns and ends the function.
- `.{...}` struct literals; field/member names may be keywords such as `in`/`out`.
- Signatures `@instance(prefix)` ≡ `instance.call(prefix)(.{in,out})`, with the original function in the prefix.
- `taskm.spawn()` takes no arguments and returns a thread; `merge(fn,args)` runs on that thread; `block` returns
  void; `done` reports whether the thread is idle.
- Macro system: `#macro name (p1, p2) { body }`, dynamic preprocessing, the `program`/`pub`/`import` predefined macros.
- The compiler backend is LLVM (no C transpilation).

### Canonical syntax decisions (2026-09-13)

- Declarations are always **type-first**: `<modifier> <type> <name> [= initializer];`; the old "name-first"
  spelling was removed entirely with **no compatibility layer**.
- Declarations have **no keyword form**: there is no `let`/`var` declaration, nor a type-less `const N = 3;`
  (`const`/`copyd` are merely modifiers written before the type).
- Functions: `fn<T, ...> name(<type> <name>, ...) <return> { ... }`; `main` may omit the return type.
- Named structs/interfaces require `type`; anonymous forms have no name.
- The only impl form: `impl<T, ...> { ... } Name;`; interfaces are satisfied **structurally**, so `impl` never
  names an interface.
- Spaces: `space { fn ... } Name;` (no parentheses); calls are `Name::fn(...)`.
- Statements: `for (int i = 0; i < n; i = i + 1)` and `for (int x : l)`; `catch (void e)`.
- Operator method names: `__add__ __sub__ __mul__ __div__ __mod__ __neg__ __eq__ __ne__ __lt__ __le__ __gt__ __ge__`.
- Generic calls infer automatically (`id(5)`); explicit generic arguments are not supported yet.
- FFI: `library name { fn sym(int a, String b) int; }` (parameter types first).

## 15. Implementation roadmap

- ✅ Interpreter: **canonical syntax implemented and verified** (lexer/parser/type checker/evaluator/runtime;
  rolling List, strict checking, struct/impl/interface, generics/pointers/Copyd, the memory system, taskm,
  signatures, the macro system; `examples/*.qk` all green).
- 🚧 Compiler front end: being unified onto the **same AST** as the interpreter (type-first, `type` named types,
  `impl ... Name;`, `space`, …), with semantic lowering in progress.
- ✅ Compiler backend: LLVM IR (`qkc`, `-run` compiles and executes; variables/control flow/arithmetic/comparison/boolean).
- ⏳ Macro system at compile time (`#ast` symbol-level insertion), import resolution, standard library and package management.

## 16. Migration notes (legacy spellings removed)

> The table below is for migration reference only; **the legacy spellings have been removed from the language
> and are not supported**.

| Legacy spelling (removed) | Canonical spelling |
|---|---|
| `x int = 1;` | `int x = 1;` |
| `fn f(a int, b String) bool` | `fn f(int a, String b) bool` |
| `func f(...)` | `fn f(...)` |
| `struct { x int; } Point;` | `type struct { int x; } Point;` |
| `interface { ... } Name;` | `type interface { ... } Name;` |
| `impl Iface { ... } T;` | `impl { ... } T;` (structural satisfaction; impl never names an interface) |
| `space Name { ... }` / `space (Name) { ... }` | `space { ... } Name;` |
| `space.fn(...)` / `Name.fn(...)` for space functions | `Name::fn(...)` |
| `for (x : l)` | `for (int x : l)` |
| `catch (e void)` | `catch (void e)` |
| operator methods `add`/`sub`/`mul`/`div`/`mod`/`neg`/`eq`/`ne`/`lt`/`le`/`gt`/`ge` | `__add__`/`__sub__`/`__mul__`/`__div__`/`__mod__`/`__neg__`/`__eq__`/`__ne__`/`__lt__`/`__le__`/`__gt__`/`__ge__` |
| `macro {pattern} {body}` | `#macro name (p1, p2) { body }` |
| `impl Sign { ... } ClassName;` / explicit interface implementation | a type with all the methods satisfies the interface (structural) |
