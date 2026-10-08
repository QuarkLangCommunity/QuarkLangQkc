# QuarkLang Standard — frozen snapshot

**Version:** 0.6.1 — **`byte` is the mother of every basic type**: `byte`/`bytes<N>` is the only
storage there is, every other basic type is a view over it, and raw bytes admit bitwise
operations only
**Frozen:** 2026-10-08 · **Review after:** 2026-11-08
**Status:** the surface described here is intended to stay unchanged for one month. Corrections to
factual errors are allowed; new features are not. Anything not described here is *not* part of the
standard — it is either a standard-library facility or a work in progress.

This document is the single source of truth for "what the language is". The implementation must follow
it, and `compiler/testdata/compare.sh` must keep printing `all identical (interpreter == compiler)`.

---

## 1. Core principles

1. **Minimal core.** The language standard is *bytes and pointers*: one storage atom (`byte`, and its
   fixed-width aggregates `bytes<N>`), the basic types that are views over it, and pointers. There are
   no built-in containers. Every collection, smart pointer and higher-level facility is an
   **extension**, shipped in the standard library and imported explicitly.
2. **Minimal dependencies.** The toolchain depends on nothing outside its own toolchain (no third-party
   libraries). A new capability is written in QuarkLang or in the toolchain's own language.
3. **No exceptions.** Failure and absence are *values*: `Result<T, E>` and `Option<T>` from the standard
   library. There is no `try`/`catch`, no `throw`. A core `panic(msg)` primitive aborts the program when a
   library must refuse.
4. **One semantic reference.** The tree-walking interpreter defines behaviour; the native compiler must
   match it byte for byte. A construct the compiler cannot lower is a **hard error**, never a silent
   difference.
5. **The core may shrink, never grow.** Legacy built-ins are migrated onto library types; the count of
   built-in containers is frozen by a test. Simplification counts as shrinkage: where there were six
   unrelated storage types, 0.6 has one storage atom and a table of views.

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
- **Integer literals:** decimal (`42`) or hexadecimal (`0x2A`); the hex form is the notation for raw
  byte data (§3.5).

## 3. Types — `byte` is the mother of every basic type

### 3.1 One storage atom

All storage in QuarkLang is a sequence of bytes. There is exactly one elementary type and one family of
fixed-width aggregates:

| Type | Meaning |
|---|---|
| `byte` | one byte: 8 bits, values `0 … 255` |
| `bytes<N>` | `N` consecutive bytes, `N ≥ 1` a constant expression; **`bytes<1>` *is* `byte`** |

Nothing else is a storage type. Every other basic type is a **view**: an interpretation laid over a byte
sequence of a fixed width. A view and its bytes occupy the *same* storage — a view is not a wrapper,
never adds a field, and never moves.

### 3.2 The views

| View | Mother | Width | Interpretation |
|---|---|---|---|
| `bool` | `byte` | 1 | `0` = false, `1` = true; no other value |
| `char` | `bytes<4>` | 4 | one Unicode scalar value |
| `int32` `int64` `int128` `int256` | `bytes<4>` `bytes<8>` `bytes<16>` `bytes<32>` | 4 / 8 / 16 / 32 | two's-complement integer |
| `float32` `float64` | `bytes<4>` `bytes<8>` | 4 / 8 | IEEE-754 binary32 / binary64 |
| `&T` `*T` | address | implementation-defined (`bytes<8>` in the reference implementation) | reference / pointer |

`int` means `int32`, `float` means `float32` — the smallest is the default. `void` and `any` are the
empty interface type, registered by the type system itself: `void` is *nothing*, `any` is *anything*.
`String` is **standard library**, not standard — a library `String` is bytes plus a length. There is no
`null`, and no `pointer T` spelling: a pointer is `*T`, a reference is `&T`.

`byte` and `bytes<N>` are value types: assigning or passing one copies its bytes. A `bytes<N>` has
alignment 1 and no padding; a view has the alignment of its width; multi-byte views are **little-endian**.

### 3.3 What raw bytes admit

`byte` and `bytes<N>` carry **no arithmetic and no ordering**. The operations they admit are exactly:

| Admitted | `&` `|` `^` `~` `<<` `>>` — bitwise, keeping the left operand's width |
|---|---|
| | `==` `!=` — equality, compared bit by bit |
| | `b[i]` — the i-th byte (0-based) as a `byte`; `b[i] = v` writes it |
| | `===` — storage identity, which every type has |
| **Not admitted** | `+ - * / %`, `< <= > >=`, `&& || !`, and every operation method |

A shift keeps the left operand's width: bits shifted out are lost, vacated bits are zero. `b[i]` outside
`0 … N-1` is a compile-time error when `i` is constant and an `IndexOutOfRangeError` otherwise.

Arithmetic and ordering require a **view** — that is what views are for: write `int32(b) + 1`, not `b + 1`.

### 3.4 Reinterpretation is free, conversion is explicit

The conversion call keeps its spelling `T(x)`; its meaning is decided by what the operands *are*:

| Form | Meaning | Cost |
|---|---|---|
| `bytes<N>(v)` / `byte(v)`, where `v` is a view of width `N` | **drop** the interpretation: the same bits, seen as raw bytes | free |
| `int32(b)`, `char(b)`, `bool(b)`, `float64(b)` …, where `b` is `byte`/`bytes<N>` of that width | **gain** a view: the same bits, reinterpreted | free |
| `int32(f)`, `float64(n)` …, where the operand is itself a view | a **value** conversion: the bits do change (`int32(3.9)` is `3`) | real work |

- The first two rows never collide with the third: the third's operand is always a view, the first two
  always take raw bytes.
- A width mismatch is a compile-time error. There is no implicit widening or truncation of raw bytes.
- Between two views, `T(x)` **always** means the value conversion. To reinterpret bits instead, pass
  through the mother: `int32(bytes<4>(f))`.

### 3.5 Fixed-size data needs no container

`bytes<N>` is the core's fixed-size aggregate: a hash, an initialisation vector, a protocol field or a
cipher block is a `bytes<N>` and needs no new type. A hex literal is an `int32` view, so raw data is
written `bytes<4> magic = bytes<4>(0x7F454C46);`.

**Why this is smaller.** One storage atom, one layout rule, one reinterpretation rule. Arithmetic exists
only in views, so the core implements bitwise operators and nothing else; every width relationship in the
language is the single table of §3.2; and a change of meaning that keeps the width is free rather than a
conversion.

## 4. References and permissions

```
&perm scope Base name = &lvalue;
```

- **perm** is a stack of `r` (read), `w` (write), `m` (move): `r`, `w`, `m`, `rw`, `rm`, `wm`, `rwm`.
- **scope** is `u` (program), `f` (function), `a` (area), `t` (thread) — 7 × 4 = **28 reference types**.
- A bare `T&` keeps full permission and program scope.
- `&lvalue` takes the address of a variable, a struct field or a list element; writes through it reach
  the original storage.

**References and pointers are different things, not two spellings of one.** A reference (`&`) is an
alias of storage and never moves. A pointer (`*`) is an address value and is the only thing that moves:
`p++` advances it one element and rebinds the variable holding it (requires `m`), and moving past the
end is an error rather than a dangling pointer.

Enforced statically, in the type checker, for declarations and assignments alike: writing needs `w`,
reading needs `r`, moving needs `m`, a reference may not be stored in a wider-scoped variable, and a
more restrictive reference may not be widened.

## 5. Operators

Arithmetic `+ - * / %`, the bitwise family `& | ^ ~`, shifts `<< >>`, comparison `< <= > >=`, equality
`== !=`, logic `&& || !`, unary `- *`, address `&`.

Arithmetic and bitwise are defined for **int views** — and, once they land, for `byte`/`bytes<N>`, which
take the bitwise family and nothing else (§3.3). The type checker refuses them elsewhere.

Precedence, loosest first: `||`, `&&`, `== != < <= > >=`, `|`, `^`, `&`, `<< >>`, `+ -`, `* / %`, then
the unary operators. The bitwise family therefore binds tighter than comparison, so `a & b == c` is
`(a & b) == c`. And:

- **`===` is storage identity** — it asks whether the two operands *occupy the same storage space*,
  never what they hold. `x === x` is true; `x === y` is false even when the values are equal; two
  pointers to one target are `==` true and `===` false. It accepts any pair of types and is **not
  overloadable**.
- **`==` compares values** and *is* overloadable through the operation method `__eq__`.

**Operation methods** (`__add__`, `__sub__`, `__mul__`, `__div__`, `__mod__`, `__eq__`, `__neg__`, …)
are how a struct type overloads an operator; `byte` and `bytes<N>` take none of them.

## 6. Overloadable notations (extensions the core resolves by name)

A container outside the core is a first-class citizen because these are resolved by **method name**, so
the core never learns the container's name:

| Notation | Protocol |
|---|---|
| `[a, b, c]` | type declares static `__literal__()` (empty container) and `__element__(Self, T)` |
| `for x : c` | receiver provides `size() int` and `get(int) T` |
| `*h` on a handle | defined as calling `get()` on it |

These are library protocols; the only types with built-in notations are `byte`/`bytes<N>`, whose `b[i]`
is core (§3.3).

## 6.5 `program` and `main` are not mandatory

| Declaration | Requires `main()` |
|---|---|
| `program main;` | yes — that is what the declaration asks for |
| `program lib;` | no |
| none at all | no |

The `program` statement is itself only a *concretization of runtime capability*; it is never mandatory.

## 6.6 Runtime = the preprocessor directives

| Directive | Meaning |
|---|---|
| `#expand <node> in <node>` | **expand** a node into a node: `#expand main in program` puts the `main` function node into the program node |
| `#run <node>` | **run** a node directly: `#run main` — the interpretation-time entry |
| `#ifdef` · `#ifndef` · `#if` · `#ifn` · `#endif` | the conditional family |
| `macro x($a $b) { … }` | **macro nodes**: the macro facility, at node level |

The difference between the two, spelled out:

```quark
#ifdef COMPILE
#expand main in program     // shape the program: main goes into the program node, nothing runs
#endif

#ifdef EXPLAIN
#run main                   // run it now: the interpreter executes the main node
#endif
```

`#expand` places a node inside the structure; `#run` executes it.

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
| `byte`, `bytes<N>` — the storage atom | `String` (bytes + length) |
| the views `bool` `char` `int32/64/128/256` `float32/64`, and `void` `any` | `Vec<T>` (`stdlib/vec.qk`) |
| `&T` references, `*T` pointers, `&lvalue`, `p++` | `Option<T>`, `Result<T, E>` |
| `struct`, `interface`, `impl`, generics, `type` aliases | smart pointers (permission references as a library layer — planned) |
| `taskm`/threads/channels, `IOStream`, `panic` | every other collection |

- **Legacy spellings** (`long`, `double`, and `char` used as a narrow integer) are implementation
  compatibility, not standard; they are being migrated onto the views of §3.2.
- **Legacy built-ins still present** (`List`, `HashTable`): scheduled for migration onto library types.
  Their presence is frozen by a test — the count may only go down.
- **Not yet implemented:** `byte`/`bytes<N>` and the width-split views are the 0.6 target. Until they
  land in both engines, `int`/`float` remain the working spellings, and `spec` is ahead of the code by
  design.

## 9. Change policy and revision log

- Corrections of **factual errors** in this document, and fixes that make the implementation match it,
  are allowed at any time.
- **New surface** (syntax, types, operators, protocols) is not added during the freeze window.
- Anything that must change the standard waits for the review date and a version bump of this file.

| Version | Date | Change |
|---|---|---|
| 0.6.1 | 2026-10-08 | Corrections: integer views admit the bitwise family `& \| ^ ~` (0.6 had stated it for raw bytes only), operator precedence written down, hex integer literals recorded |
| 0.6 | 2026-10-08 | `byte` made the mother of every basic type; `bytes<N>` added; basic types restated as views over it; raw bytes restricted to bitwise operations; the three meanings of the conversion call `T(x)` fixed; layout rules written down; the split type table and the duplicated §8 row of 0.5 repaired. Freeze restarts: review after 2026-11-08 |
| 0.5 | 2026-10-07 | integers and floats split by width, pointers separated from references, `char` added, `String` moved to the library, `null` and `pointer T` removed, `program` and the runtime directives recorded |
