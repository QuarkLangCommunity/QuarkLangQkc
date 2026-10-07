# QuarkLang Standard — frozen snapshot

**Version:** 0.4 (snapshot of the implemented core)
**Frozen:** 2026-10-07 · **Review after:** 2026-11-07
**Status:** the surface described here is intended to stay unchanged for one month. Corrections to
factual errors are allowed; new features are not. Anything not described here is *not* part of the
standard — it is either a standard-library facility or a work in progress.

This document is the single source of truth for "what the language is". The implementation must follow
it, and `compiler/testdata/compare.sh` must keep printing `all identical (interpreter == compiler)`.

---

## 1. Core principles

1. **Minimal core.** The language standard is *pointers and basic types* only. There are no built-in
   containers. Every collection, smart pointer and higher-level facility is an **extension**, shipped
   in the standard library and imported explicitly.
2. **Minimal dependencies.** The toolchain depends on nothing outside its own toolchain (no third-party
   libraries). A new capability is written in QuarkLang or in the toolchain's own language.
3. **No exceptions.** Failure and absence are *values*: `Result<T, E>` and `Option<T>` from the standard
   library. There is no `try`/`catch`, no `throw`. A core `panic(msg)` primitive aborts the program when a
   library must refuse.
4. **One semantic reference.** The tree-walking interpreter defines behaviour; the native compiler must
   match it byte for byte. A construct the compiler cannot lower is a **hard error**, never a silent
   difference.
5. **The core may shrink, never grow.** Legacy built-ins are migrated onto library types; the count of
   built-in containers is frozen by a test.

## 2. Lexical and declarative form

- **Canonical declaration:** `<modifier> <type> <name> [= initializer];` — modifiers come first.
- **No `let`/`var`.** `const` and `copyd` are *modifiers*, not declarations: `const int N = 3;`,
  `copyd List<int> l = [1,2];`.
- **Type aliases:** `type <existing> <new>;` makes `<new>` another name for the type.
- **Named types:** `type struct<T> { … } Name;` · `type interface<T> { … } Name;` — type parameters are
  written **after the keyword**, never after the name.
- **Implements:** `impl<T> { … } Name;` — structural: a type satisfies an interface by having the
  methods. Method members cannot carry `pub` (top-level declarations only).
- **Comments:** `//` to end of line. Preprocessor: `#` at the start of a line.

## 3. Types

| Type | Notes |
|---|---|
| `int` | 32-bit, wrapping (`wrapI32`); out-of-range literals are compile errors |
| `float`, `bool`, `String` | scalars; `String` supports `+` concatenation |
| `void` | no value |
| `null` | the zero value of references |
| `T&`, `pointer T` | **nullable references** (zero value `null`); member access dereferences; dereferencing null raises `NullPointerError` |
| `struct` | value type; generic (`Vec<T>` shapes are library code) |
| `interface<T>` | **generic interfaces**; `Index<Vec<int>>` is a legal type |
| `thread` / `Task` | cooperative tasks (`taskm`), channels |
| `IOStream` | `io` binds at the `main` entry: `fn main(IOStream io)` |

**Modifier `copyd`:** a declaration modifier (`copyd int a`, `copyd List<int> a`) that deep-copies on
passing. There is **no** `Copyd<T>` type and no `T[Copyd]` spelling.

## 4. References and permissions

```
&perm scope Base name = &lvalue;
```

- **perm** is a stack of `r` (read), `w` (write), `m` (move): `r`, `w`, `m`, `rw`, `rm`, `wm`, `rwm`.
- **scope** is `u` (program), `f` (function), `a` (area), `t` (thread) — 7 × 4 = **28 reference types**.
- A bare `T&` keeps full permission and program scope.
- `&lvalue` takes the address of a variable, a struct field or a list element; writes through it reach
  the original storage.
- `p++` **moves** a reference one element and rebinds the variable holding it (requires `m`); moving past
  the end is an error, not a dangling reference.

Enforced statically, in the type checker, for declarations and assignments alike: writing needs `w`,
reading needs `r`, moving needs `m`, a reference may not be stored in a wider-scoped variable, and a
more restrictive reference may not be widened.

## 5. Operators

Arithmetic `+ - * / %`, shifts `<< >>`, comparison `< <= > >=`, equality `== !=`, logic `&& || !`,
unary `- *`, address `&`, and:

- **`===` is storage identity** — it asks whether the two operands *occupy the same storage space*,
  never what they hold. `x === x` is true; `x === y` is false even when the values are equal; two
  pointers to one target are `==` true and `===` false. It accepts any pair of types and is **not
  overloadable**.
- **`==` compares values** and *is* overloadable through the operation method `__eq__`.

**Operation methods** (`__add__`, `__sub__`, `__mul__`, `__div__`, `__mod__`, `__eq__`, `__neg__`, …)
are how a struct type overloads an operator.

## 6. Overloadable notations (extensions the core resolves by name)

A container outside the core is a first-class citizen because these are resolved by **method name**, so
the core never learns the container's name:

| Notation | Protocol |
|---|---|
| `[a, b, c]` | type declares static `__literal__()` (empty container) and `__element__(Self, T)` |
| `for x : c` | receiver provides `size() int` and `get(int) T` |
| `*h` on a handle | defined as calling `get()` on it |

## 7. Error model

- No `try`/`catch`, no `throw`.
- **`Option<T>`** (library): `some(v)`, `none(sample)`, `isSome`, `isNone`, `unwrap`, `unwrapOr`.
- **`Result<T, E>`** (library): `ok(v, sample)`, `err(e, sample)`, `isOk`, `isErr`, `unwrap`,
  `unwrapOr`, `unwrapErr`.
- `unwrap` on `none`/`err` **terminates** the program with a `PanicError` rather than inventing a zero
  value. `panic(msg)` is the core primitive behind it.
- Runtime errors that remain (`DivisionByZeroError`, `IndexOutOfRangeError`, `NullPointerError`,
  `PanicError`) are reported with position and an execution log.

## 8. Core vs standard library

| In the core | In the standard library (imported) |
|---|---|
| `int`, `float`, `bool`, `String`, `void`, `null` | `Vec<T>` (`stdlib/vec.qk`) |
| `T&`, `pointer T`, `&lvalue`, `p++` | `Option<T>`, `Result<T, E>` |
| `struct`, `interface`, `impl`, generics, `type` aliases | smart pointers (permission references as a library layer — planned) |
| `taskm`/threads/channels, `IOStream`, `panic` | every other collection |

**Legacy built-ins still present** (`List`, `HashTable`): scheduled for migration onto library types.
Their presence is frozen by a test — the count may only go down.

## 9. Change policy for the freeze window

- Corrections of **factual errors** in this document, and fixes that make the implementation match it,
  are allowed at any time.
- **New surface** (syntax, types, operators, protocols) is not added during the window.
- Anything that must change the standard waits for the review date and a version bump of this file.
