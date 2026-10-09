# QuarkLang Syntax Reference (kept in sync with the implementation · canonical syntax)

**English** · [简体中文](SYNTAX.zh-CN.md)

> **The one canonical form: type-first.** The old "name-first" spelling has been removed; there is no
> backwards compatibility.
> **There is no `let`/`var` or `const` keyword declaration**: `const`/`copyd` are **modifiers** and must be
> written together with the type *before* the name
> (`const int N = 3;` ✔ / `let x = 1;`, `var y = 2;`, `const N = 3;` ✘).
> This file is updated with the implementation; for a behaviour change → change this file and the implementation first.
> Numbering for item-by-item review: M=macros, K=statements, E=expressions, S=structures/generics/interfaces,
> O=object model, T=types, P=toolchain.

## K Declarations (type-first)

| ID | Syntax | Notes |
|---|---|---|
| K1 | `<modifier> <type> <name>;` | declaration; modifiers in K2 |
| K2 | `const int N = 3;` `copyd List<int> l = [1,2];` | modifiers: `const` (constant), `copyd` (copy-on-pass); written **first** |
| K3 | `int x = 1;` `Point p;` `List<int> l = [1,2];` | the initializer may be omitted (zero value) |
| K4 | `x = 2;` `p.x = 3;` `l[i] = 1;` | assignment (variable / member / index) |
| K5 | `if (cond) { } else if (cond) { } else { }` | the condition must be bool |
| K6 | `while (cond) { }` | |
| K7 | `for (int i = 0; i < n; i = i + 1) { }` | C-style for (its initializer is a type-first declaration) |
| K8 | `for (int x : l) { }` | iteration for (the loop variable must carry a type) |
| K9 | `break;` | leaves while/for |
| K10 | `try { } catch (void e) { }` | the catch variable is type-first (`void` = free type) |
| K11 | `return e;` `log e;` `delete x;` | return ends the function with a value; log records and ends; delete reclaims |
| K12 | `program main;` `program library;` `import "path";` `pub fn ...` | preprocessor macros; a library cannot be run |

> **K-note (declaration red line)**: a declaration is *always* "modifier + type + name" — there is **no
> `let`/`var` keyword declaration**, and no type-less `const N = 3;` (`const` is just a modifier).
> Specs, examples, templates, completions and documentation must never use `let`/`var` declaration forms:
>
> ```qk
> int x = 1;            // ✘ let x = 1;   ✘ var x = 1;
> const int N = 3;      // ✘ const N = 3;
> copyd List<int> l = [1, 2];
> ```

## S Functions / Structs / Interfaces / Impls / Spaces

| ID | Syntax | Notes |
|---|---|---|
| S1 | `fn name(int a, String b) bool { ... }` | function; **parameter types come first**; the return type is required (`main` may omit it = void) |
| S2 | `fn<T, U> name(T v) T { ... }` | generic function; call sites infer: `name(5)` |
| S3 | `type struct { int x; int y; } Point;` | named struct (`type` is required); field types come first |
| S4 | `struct { int x; };` | anonymous struct type |
| S5 | `type struct<T> { T v; } Box;` | generic struct; `impl<T>` must introduce the same parameters |
| S6 | `type interface { fn sum(Self self) int; expand interface Other; } Iface;` | interface; `Self` = the type itself; `expand interface` composes interfaces |
| S7 | `interface { };` | anonymous interface (empty interface = `void`) |
| S8 | `impl { fn new(int x) Point { ... } fn sum(Point self) int { ... } } Point;` | impl: **the only form** (name after the block); static methods take no `self`, instance methods take `self` first |
| S9 | `impl<T> { ... } Box;` | generic impl |
| S10 | `space { fn max(int a, int b) int { ... } } math;` | space (self-impl): no parentheses; **always call it as `math::max(1, 2)`**, including from inside the space |
| S11 | `Point::new(3, 4)` `p.sum()` | static call with `::`; instance methods with `.` |
| S12 | `Point p = .{x: 1, y: 2};` | struct literal |
| S13 | `fn __add__(Point self, Point other) Point { ... }` | operator method names: `__add__ __sub__ __mul__ __div__ __mod__ __neg__ __eq__ __ne__ __lt__ __le__ __gt__ __ge__` (`a + b` dispatches automatically) |
| S14 | Interfaces are satisfied **structurally**: a type with all the methods satisfies the interface; the interface name is not written on `impl` | checked at assignment/call sites |

## T Types

| ID | Types | Notes |
|---|---|---|
| T1 | `int` (32-bit wrap) `long` `char` `float` `bool` `String` | scalars |
| T2 | `List<T>` `HashTable<K,V>` | containers |
| T3 | `T&` / `pointer T` | nullable reference; zero value `null` |
| T4 | `Copyd<T>` / `int[Copyd]` / `int[]` | copy-on-pass semantics |
| T5 | `void` `interface{}` `IOStream` `thread` `memorize` `memory` | built-ins |

## E Expressions

| ID | Syntax | Notes |
|---|---|---|
| E1 | `123` `-7` `1.5` `"str"` `true` `.{x: 1}` | literals; `` `...` `` raw strings (Go semantics, no escapes) |
| E2 | `f(x)` `obj.m(x)` `T::m(x)` `space::f(x)` `p.x` `l[i]` `*l` | calls / members / indexing / head |
| E3 | `+ - * / %` `<< >>` `== != < <= > >=` `&& \|\| !` | operators; `+` also concatenates String |
| E4 | `s.size() contains startsWith endsWith indexOf substring split trim toLower toUpper replace charAt toInt toFloat` | String built-ins (rune semantics; `indexOf` returns a **byte** index, -1 = absent) |
| E5 | `l.size()` `l.head()` `l.tail()` `l.get(i)` `l.next()` `l.reset()` `l.append(v)` `l.appendAll(l2)` `l.toString()` `l.__sort__()` `l[i]` `*l` | List (rolling dual cursor; `get(i)` is the same as `l[i]`; `*l` also returns the head) |
| E6 | `h.put(k, v)` `h.get(k)` `h.contains(k)` `h.keys()` `h.remove(k)` `h.size()` | HashTable (`get` on a missing key returns nil) |
| E7 | `f(args) @mb()` | signature call (a Sign instance) |

## M Macros and preprocessing

| ID | Syntax | Notes |
|---|---|---|
| M1 | `#macro name (a, b) { body }` | named-parameter macro; `()`/`[]`/`{}` delimiters are interchangeable; single-pass expansion, not recursive |
| M2 | `#when (run) { ... }` / `#when (compile) { ... }` | state selection (the interpreter takes `run`) |
| M3 | `#error("msg")` `#return <expr>` `#insert(...)` `#execute(name)` | preprocessor commands |
| M4 | custom commands starting with `#` | see the internal macro table |
| M5 | `library name { fn sym(int a, String b) int; }` | FFI library binding (parameters like functions, types first); the method name is the symbol name |

## P Toolchain

| ID | Behaviour |
|---|---|
| P1 | `quark file.qk` interprets (debug with `qkd`) |
| P2 | `qkc file.qk` emits LLVM IR; `qkc -run file.qk` compiles and runs (the native path is canonical) |
| P3 | `qkm init/build/debug/install/update` project and dependency management |
| P4 | Both paths share **identical syntax and semantics** (one source file works everywhere) |
| P5 | `qkcheck [-json] [-params] [-L dir] files...` static analysis: unused variable/parameter, unreachable code, shadowing, missing return, unimplemented interface, void value misuse (codes QK101–QK114: unused, unreachable, shadowing, missing return, interface near-miss, void misuse, self-assignment, constant condition, constant divide-by-zero, unused import, global-function shadowing, dead store, self-comparison) |
| P6 | `qkdoc [-o file] [-html] [-all] files...` API docs: the `/* */`/`//` comment above a declaration + `pub` exports → Markdown/HTML |
| P7 | `qkrepl [-e code] [-q]` interactive evaluation: multi-line continuation, a persistent environment (variables/functions/types), `:help/:load/:quit` |
| P8 | `qklsp [-L dir]` language server (stdio LSP): diagnostics / go-to-definition / completion / hover / document symbols; editor integration in `editors/` (VS Code extension + tree-sitter grammar) |
| P9 | `scripts/build-release.sh <version>` release artifacts (three-platform binaries + version injection + sha256 manifest); `scripts/changelog.sh` changelog; a `v*` tag triggers the release workflow |
