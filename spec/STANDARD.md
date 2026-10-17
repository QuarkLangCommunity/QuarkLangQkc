# QuarkLang Standard — frozen snapshot

**Version:** 0.8.3 — **`bit` is the mother of every basic type**: every basic type is *implemented from*
`bit`/`bits<N>` — they are not separate storage — and raw bits admit bitwise operations only
**Frozen:** 2026-10-10 · **Review after:** 2026-11-10
**Status:** the surface described here is intended to stay unchanged for one month. Corrections to
factual errors are allowed; new features are not. Anything not described here is *not* part of the
standard — it is either a standard-library facility or a work in progress.

This document is the single source of truth for "what the language is". The implementation must follow
it, and `compiler/testdata/compare.sh` must keep printing `all identical (interpreter == compiler)`.

---

## 1. Core principles

1. **Minimal core.** The language standard is *bits and pointers*: one storage atom (`bit`, and its
   fixed-width aggregates `bits<N>`), the basic types that are views over it, and pointers. There are
   no built-in containers. Every collection, smart pointer and higher-level facility is an
   **extension**, shipped in the standard library and imported explicitly.
2. **Minimal dependencies.** The toolchain depends on nothing outside its own toolchain (no third-party
   libraries). A new capability is written in QuarkLang or in the toolchain's own language.
3. **No exceptions.** Failure and absence are *values*: `Result<T, E>` and `Option<T>` from the standard
   library. There is no `try`/`catch`, no `throw`. A core `panic(msg)` primitive aborts the program when a
   library must refuse.
4. **One semantic reference.** The tree-walking interpreter defines behaviour; the native compiler must
   match it bit for bit. A construct the compiler cannot lower is a **hard error**, never a silent
   difference. The parity gate proves *interpreter == compiler*, not *code == this document*: a rule both
   engines implement the other way round, or a feature both engines omit, leaves it green, so §11.3 lists
   what it cannot see.
5. **The core may shrink, never grow.** Legacy built-ins are migrated onto library types; the count of
   built-in containers is frozen by a test. Simplification counts as shrinkage: where there were six
   unrelated storage types, 0.7 has one storage family and a table of views.

Diagnostics are not part of the language surface. A conforming program is one the *engines* accept; the
linter's codes (`QK101`–`QK115`), the formatter, the language server and the doc renderer are tools around
the language, and this document does not define them (`docs/primitives.md` §7.2).

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
  bit data (§3.5).

## 3. Types — `bit` is the mother of every basic type

### 3.1 One storage atom

All storage in QuarkLang is a sequence of bits. There is exactly one elementary type and one family of
fixed-width aggregates:

| Type | Meaning |
|---|---|
| `bit` | one bit: values `0` and `1` |
| `bits<N>` | `N` consecutive bits, `N ≥ 1` a constant expression; **`bits<1>` *is* `bit`** |

Nothing else is a storage type. Every other basic type is a **view**: an interpretation laid over a bit
sequence of a fixed width. A view and its bits occupy the *same* storage — a view is not a wrapper,
never adds a field, and never moves.

Every basic type is therefore *implemented from* `bit`/`bits<N>`: `bool` is a one-bit view, `uchar` an
eight-bit one, `char` a 32-bit one, `float64` a 64-bit one. The core has exactly one storage
representation, one copy rule and one reinterpretation rule for all of them; views differ only in the
arithmetic the type checker then lets you apply.

Memory is addressed in bytes: a `bits<N>` occupies `⌈N/8⌉` bytes, byte-aligned, and every bit past `N` is
zero. So `bits<5>` is a five-bit value in one byte, and `bits<40>` is five bytes.

### 3.2 The views

| View | Mother | Width (bits) | Interpretation |
|---|---|---|---|
| `bool` | `bit` | 1 | `0` = false, `1` = true; no other value |
| `char` | `bits<32>` | 32 | one Unicode scalar value |
| `uchar` | `bits<8>` | 8 | unsigned integer `0 … 255` — the octet view |
| `int32` `int64` `int128` `int256` | `bits<32>` `bits<64>` `bits<128>` `bits<256>` | 32 / 64 / 128 / 256 | two's-complement integer |
| `float32` `float64` | `bits<32>` `bits<64>` | 32 / 64 | IEEE-754 binary32 / binary64 |
| `&T` `*T` | address | implementation-defined (`bits<64>` in the reference implementation) | reference / pointer |

A view's name states its own width — `int32` is `bits<32>`, `float64` is `bits<64>` — which is the point
of measuring storage in bits rather than in bytes.

`uchar` is the octet type, keeping the C tradition's name for it (`unsigned char`): the element type for
raw data, and the only 8-bit view — there is no signed one. `char` stays the text scalar (`bits<32>`, one
Unicode code point) and is a different thing, not a wider `uchar`.

`int` means `int32`, `float` means `float32` — the smallest is the default. `void` and `any` are the
empty interface type, registered by the type system itself: `void` is *nothing*, `any` is *anything*.
`String` is **standard library**, not standard — a library `String` is bits plus a length. There is no
`null`. A pointer is `*T` and a reference is `&T`; `pointer T` is the implementation's own spelling for the
nullable raw block a `new T[n]` yields, which is compatibility surface (§9) and not a view of this table.

`bit` and `bits<N>` are value types: assigning or passing one copies its bits. A `bits<N>` has alignment
1 and no padding beyond the trailing zero bits; a view has the alignment of its whole width; multi-byte
views are **little-endian**.

### 3.3 What raw bits admit

`bit` and `bits<N>` carry **no arithmetic and no ordering**. The operations they admit are exactly:

| Admitted | `&` `|` `^` `~` `<<` `>>` — bitwise, keeping the left operand's width |
|---|---|
| | `==` `!=` — equality, compared bit by bit |
| | `b[i]` — the i-th bit (0-based) as a `bit`; `b[i] = v` writes it |
| | `===` — storage identity, which every type has |
| **Not admitted** | `+ - * / %`, `< <= > >=`, `&& || !`, and every operation method |

A shift keeps the left operand's width: bits shifted out are lost, vacated bits are zero. `b[i]` outside
`0 … N-1` is a compile-time error when `i` is constant and an `IndexOutOfRangeError` otherwise.

Arithmetic and ordering require a **view** — that is what views are for: write `int32(b) + 1`, not `b + 1`.

### 3.4 Reinterpretation is free, conversion is explicit

The conversion call keeps its spelling `T(x)`; its meaning is decided by what the operands *are*:

| Form | Meaning | Cost |
|---|---|---|
| `bits<N>(v)` / `bit(v)`, where `v` is a view of width `N` | **drop** the interpretation: the same bits, seen as raw bits | free |
| `int32(b)`, `char(b)`, `bool(b)`, `float64(b)` …, where `b` is `bit`/`bits<N>` of that width | **gain** a view: the same bits, reinterpreted | free |
| `int32(f)`, `float64(n)` …, where the operand is itself a view | a **value** conversion: the bits do change (`int32(3.9)` is `3`) | real work |

- The first two rows never collide with the third: the third's operand is always a view, the first two
  always take raw bits.
- A width mismatch is a compile-time error. There is no implicit widening or truncation of raw bits.
- Between two views, `T(x)` **always** means the value conversion. To reinterpret bits instead, pass
  through the mother: `int32(bits<32>(f))`.
- A value conversion **narrows by keeping the low bits**: `uchar(300)` is `44`, `uchar(-1)` is `255`.
  Reinterpretation never truncates — it requires equal widths — so this rule applies only to the third row.

### 3.5 Fixed-size data needs no container

`bits<N>` is the core's fixed-size aggregate: a hash, an initialisation vector, a protocol field or a
cipher block is a `bits<N>` and needs no new type. A hex literal is an `int32` view, so raw data is
written `bits<32> magic = bits<32>(0x7F454C46);`.

**Why this is smaller.** One storage family, one layout rule, one reinterpretation rule — and the width
unit is the one the type names already use. Arithmetic exists only in views, so the core implements
bitwise operators and nothing else; every width relationship in the language is the single table of §3.2;
and a change of meaning that keeps the width is free rather than a conversion.

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

Arithmetic and bitwise are defined for **int views** — and, once they land, for `bit`/`bits<N>`, which
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
are how a struct type overloads an operator; `bit` and `bits<N>` take none of them.

## 6. Overloadable notations (extensions the core resolves by name)

A container outside the core is a first-class citizen because these are resolved by **method name**, so
the core never learns the container's name:

| Notation | Protocol |
|---|---|
| `[a, b, c]` | type declares static `__literal__()` (empty container) and `__element__(Self, T)` |
| `for x : c` | receiver provides `size() int` and `get(int) T` |
| `*h` on a handle | defined as calling `get()` on it |

These are library protocols; the only types with built-in notations are `bit`/`bits<N>`, whose `b[i]`
is core (§3.3).

## 7. Program shape and runtime

### 7.1 `program` and `main` are not mandatory

| Declaration | Requires `main()` |
|---|---|
| `program main;` | yes — that is what the declaration asks for |
| `program lib;` | no |
| none at all | no |

The `program` statement is itself only a *concretization of runtime capability*; it is never mandatory.

### 7.2 Runtime = the preprocessor directives

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

## 8. Error model

- No `try`/`catch`, no `throw`.
- **`Option<T>`** (library): `some(v)`, `none(sample)`, `isSome`, `isNone`, `unwrap`, `unwrapOr`.
- **`Result<T, E>`** (library): `ok(v, sample)`, `err(e, sample)`, `isOk`, `isErr`, `unwrap`,
  `unwrapOr`, `unwrapErr`.
- `unwrap` on `none`/`err` **terminates** the program with a `PanicError` rather than inventing a zero
  value. `panic(msg)` is the core primitive behind it.
- Runtime errors that remain (`DivisionByZeroError`, `IndexOutOfRangeError`, `NullPointerError`,
  `PanicError`) are reported with position and an execution log.

## 9. Core vs standard library

| In the core | In the standard library (imported) |
|---|---|
| `bit`, `bits<N>` — the storage atom | `String` (bits + length) |
| the views `bool` `char` `uchar` `int32/64/128/256` `float32/64`, and `void` `any` | `Vec<T>` (`stdlib/vec.qk`) |
| `&T` references, `*T` pointers, `&lvalue`, `p++` | `Option<T>`, `Result<T, E>` |
| `struct`, `interface`, `impl`, generics, `type` aliases | smart pointers (permission references as a library layer — planned) |
| `taskm`/threads/channels, `IOStream`, `panic` | every other collection |

- **Legacy spellings** (`long`, `double`, `pointer T`, and `char` used as a narrow integer) are
  implementation compatibility, not standard; they are being migrated onto the views of §3.2 — a `char`
  used as a narrow integer becomes `uchar`. §11.1 records what each one means in the implementation today.
- **Legacy built-ins still present** (`List`, `HashTable`): scheduled for migration onto library types.
  Their presence is frozen by a test — the count may only go down. See §11.3.

## 10. Memory and lifetime

QuarkLang has **no garbage collector**. Storage is released explicitly and the release is
deterministic: nothing is collected behind your back, and no allocation ever waits for a collection.

**Scope.** This section describes the contract of the compiled artifact; the reference interpreter
implements that policy as a layer over its host runtime's allocator.

| Primitive | Meaning |
|---|---|
| `delete x;` | release `x`'s storage: run `__delete__()` if the type declares it, then return the storage to the free pool. The payload is **not** wiped — wiping is `clear`'s job, so a released block never hands another value's bytes to the next user |
| `memory.setBlock(n)` | set the block size, the unit the runtime allocates and reuses |
| `memory.compact()` | free the blocks nobody occupies; it does not return |
| `memory.Fragmentation()` | the unused space inside live blocks, as a ratio (inspection) |

**Reuse is the point.** Allocation prefers the least-occupied block that has room, so a workload whose
temporary data arrives in *tides* — allocate, use, delete, repeat — reuses a small, stable set of blocks
instead of growing the heap. The reference implementation measures 99.96% reuse, 0.195% fragmentation and
a block count that converges to a constant on the tide benchmark; those numbers are measurements, not
promises, but the policy above is normative. When a task ends, the blocks it owns become reclaimable.

Releasing storage that something still refers to is a programming error: the runtime does not detect it,
and the permission system of §4 is what keeps it visible at compile time.

## 11. Implementation status

This section is **not normative**: it records how far the code has caught up with §1–§10, so a reader can
tell the specification from the implementation. The standard is the source of truth — where the two
differ, the code is wrong.

### 11.1 Implemented, in both engines

- `bit`, `bits<N>` for `1 ≤ N ≤ 32`, and `uchar`: the bitwise family, `b[i]` read and write, the three
  conversion meanings of §3.4, and unsigned decimal printing. `compare.sh` keeps the interpreter and the
  compiler identical on every corpus case.
- the bitwise family on int views, hex literals, `===` storage identity, permission references, `&lvalue`
  and the reference move `p++`.
- the legacy spellings of §9 that §3.2 does not define, each folded onto the view this implementation
  lowers: `char` is the int view (`internal/lang/typecheck.go` maps it to `tInt`), `double` is the 64-bit
  float view this implementation spells `float` (see §11.2 on the width), `long` computes as the int view in
  both engines (measured: `long x = 1000000000; x * 4` is `-294967296` in each), and `pointer T` is the
  nullable handle `new T[n]` yields. Both engines accept them in a declaration, a parameter, a return type, a
  struct field, a conversion (`char(66)`) and a `List` element (`List<char>` is `List<int>`, as the
  interpreter's `char[]` is). `compiler/testdata/cases/z_legacy_views.kq` is the case that keeps the two
  engines on one behaviour; before it, the interpreter accepted these spellings and the compiler refused them
  with a hard error, which the parity gate could not see.
- §5's precedence table at the shift level: `<<`/`>>` is *looser* than `+`/`-`, so `1 + 2 << 3` reads
  `(1 + 2) << 3` = 24. Measured in both engines under `LANG=C.UTF-8` and `LANG=zh_CN.UTF-8`:

  | Expression | Both engines | §5's order |
  |---|---|---|
  | `1 + 2 << 3` | 24 — `(1 + 2) << 3` | 24 — agrees |
  | `1 << 2 + 3` | 32 — `1 << (2 + 3)` | 32 — agrees |
  | `10 - 2 >> 1` | 4 — `(10 - 2) >> 1` | 4 — agrees |
  | `1 + (2 << 3)` | 17 | 17 — agrees |
  | `(1 + 2) << 3` | 24 | 24 — agrees |

  The grouping is decided in the syntax layer, the pinned module
  `github.com/QuarkLangCommunity/QuarkLangQkparser`, which now threads
  `parseBitAnd → parseShift → parseAdd → parseMul` — that chain, and nothing in this checkout, is what
  makes the three rows above print §5's values. It is also the only place they *can* be decided: that
  parser erases the parenthesis node (its `TLParen` case returns the inner expression), so an engine
  receives a tree whose operands are already grouped, and the two readings must therefore be two trees
  when they leave the parser. `internal/lang/precedence_test.go` pins both facts — that
  `1 + 2 << 3` and `(1 + 2) << 3` are one shape, and that `1 + (2 << 3)` is another — and
  `compiler/testdata/cases/z_precedence.kq` pins one line per level of §5 against the standard rather
  than against the other engine. §11.3 keeps the record of the divergence this closes; the change was
  semantic (§5's order differs from C's and Go's), so it landed as its own reviewed change — three lines
  and their comment in the syntax layer, pinned by `parser_precedence_test.go` there.
- `List<float>`, and its §9 spelling `List<double>`, in both engines: the literal, the element read and
  write, `append`, `size`/`get`/`toString`, for-in iteration and print. The compiler carries each element in
  an i64 slot as a bit-cast double, so a float list stores the bits the interpreter's list stores, and both
  engines render an element through one float spelling. §11.2 states what a float list still lacks; before
  this, the interpreter accepted `List<double>` and the compiler refused it with a hard error — an engine
  difference the parity gate cannot see, because no case exercised the construct.

### 11.2 Limits of the current implementation

- `bits<N>` with `N > 32` is refused by the shared frontend, with one identical message in both engines.
- a width takes a decimal literal only, not an arbitrary constant expression.
- reinterpreting `float`/`double` is refused: this implementation's `float` is 64-bit while §3.2 says
  `float32`, so the widths cannot be trusted to agree yet.
- a `List` element is `int`, `float` or `String` in the compiler, so `List<double>` is lowered as the
  interpreter's `float` list, on every path the `List<int>` lowering has: literal, element read and write,
  `append`, `size`/`get`/`toString`, for-in iteration and print. Both spellings work (`float`, and the §9
  compatibility spelling `double`), and `List<char>` is *not* in this paragraph at all because it folds to
  `List<int>`. Elements live in i64 slots each holding the element as a bit-cast double — the bit-cast the
  other float paths use — and the runtime renders them through the same float spelling a bare float gets, so
  a list and its elements cannot disagree. Measured in both engines: `compiler/testdata/cases/z_list_double.kq`
  (no list printing, so it also runs under `lli` in `go test`) and `cases_run/z_list_double_print.kq` (the
  print and `toString` paths, which need the linked runtime). What a `List<float>` still does not have is what
  only the `int` list has: `head`/`tail`/`next`/`reset`/`appendAll`/`__sort__` are refused with `List<float>.%s
  is not supported yet (size/get/toString/append and for-in iteration are lowered)`; `delete` on a float list is
  refused because the compiler releases a `List<int>` variable only (§11.3's §10 table); boxing one into
  `interface{}` is not lowered; and printing a `List<int>` is still refused although a `List<String>` and a
  `List<float>` print. A declaration without an initializer builds the interpreter's empty list where
  `List<int>` still refuses one, because a null handle would fault the read paths — the lowerer synthesizes the
  empty literal. Closing the rest needs an element type threaded through the method lowering, `delete`, the
  boxing path and the zero-value path: one small change per path, and no new standard surface. `copyd
  List<float>` is refused the way `copyd List<String>` is — deep-copying a float list is not lowered, and the
  helper copies 4-byte elements of a `%List`, so copying a float list through it would be a silent miscompile
  — but a float list *inside* a struct that is itself passed `copyd` is not covered by that refusal: it
  reaches the struct deep-copy path, which emits invalid IR for a `List<int>` member too (measured on 0.8.4,
  before this lowering, so it is a defect of that path and not of the element type).
- layout: a `bits<N>` whose width is not 8, 16 or 32 occupies the next storage size in compiled structs
  and is aligned to that type rather than to 1 as §3.1 requires. Values are exact; only layout rounds up.
- call depth is capped at 8192 (`lang.MaxCallDepth`). A call made from a frame that deep is refused with
  `StackOverflowError: recursion depth exceeded 8192` and the line of the refused call. The interpreter
  counts the depth per call context and the compiler bakes the same counter into the generated code, so
  both engines stop at the same depth — including a self tail call, which the compiler counts although it
  needs no new native frame.
- expression nesting is capped at 65536 levels (`lang.MaxExprDepth`, the parser module's
  `qkparser.MaxExprDepth`). The depth is measured differently on the two paths that enforce it: the
  parser counts recursion levels and refuses a deep parenthesis nest as a positioned `ParseError`,
  `nesting is deeper than 65536 levels, the limit of this implementation: split the expression into
  smaller statements`, while the type checker counts expression nodes and refuses a long operator chain
  as `CompileError: expression nesting is deeper than 65536 levels, the limit of this implementation:
  split the expression into smaller statements`. Both limits are explicit counters, never a probe of the
  Go stack, so the three platforms answer alike.

### 11.3 The code does not follow the standard here

- **§7.2 macro spelling.** The standard says `macro x($a $b) { … }`. The implementation still accepts
  `#macro name (params) { body }` and rejects `$`. A change that made the code conform was proposed and
  then rolled back; it has to come back as its own reviewed change.
- **§8 no exceptions.** `try`/`catch` still exist in the parser, the AST, the evaluator, the linter and
  the compiler, although the standard has no exceptions. They are scheduled for removal, after which
  `Option<T>` and `Result<T, E>` are the only way to carry failure.
- **§9 core minus containers.** `List` and `HashTable` are still built in; their count is frozen by a test.
- **§5 shift precedence — closed in 0.8.4.** Kept here as the record of what the divergence was. §5 says:
  "Precedence, loosest first: `||`, `&&`, `== != < <= > >=`, `|`, `^`, `&`, `<< >>`, `+ -`, `* / %`, then
  the unary operators." Read as written, `<<`/`>>` is *looser* than `+`/`-` (C's grouping), so `1 + 2 << 3`
  is `(1 + 2) << 3` = 24; both engines grouped it the other way (Go's), measured under `LANG=C.UTF-8` and
  `LANG=zh_CN.UTF-8`:

  | Expression | Both engines, before | §5's order |
  |---|---|---|
  | `1 + 2 << 3` | 17 — `1 + (2 << 3)` | 24 — `(1 + 2) << 3` |
  | `1 << 2 + 3` | 7 — `(1 << 2) + 3` | 32 — `1 << (2 + 3)` |
  | `10 - 2 >> 1` | 9 — `10 - (2 >> 1)` | 4 — `(10 - 2) >> 1` |
  | `1 + (2 << 3)` | 17 | 17 — agrees |
  | `(1 + 2) << 3` | 24 | 24 — agrees |

  The cause was never in this repository: the syntax layer is the pinned module
  `github.com/QuarkLangCommunity/QuarkLangQkparser`, whose parser called `parseShift` from `parseAdd`
  (`parseAdd → parseShift → parseMul`), i.e. additive looser than shift. The fix is that chain's exchange
  (`parseBitAnd` takes `parseShift`, `parseShift` takes `parseAdd`, `parseAdd` takes `parseMul`, plus the
  comment above them), and it cannot be done in this checkout afterwards either: that parser erases
  parentheses (its `TLParen` case returns the inner expression), so `1 + 2 << 3` and `1 + (2 << 3)` used to
  reach both engines as **one and the same tree**, and no later pass could have given them 24 and 17. The
  syntax layer carries the change (its PR QuarkLangQkparser#5, pinned here as
  `v0.0.0-20261015231811-3de63a5ff52d`), the two readings are two trees again, and §11.1 carries the
  measurement of the closed state. Of the 40 corpus cases the pin then covered — 23 under `cases/` and 17
  under `cases_run/` — exactly one changed value, `z_precedence.kq`, and only its three shift rows (17, 7 and
  9 become 24, 32 and 4); every other case is byte-identical in both engines, which was measured by running
  each case through an engine built from the commit before the pin and one built after it. No existing
  program in this repository changed.
- **§10 memory on the compiled path.** §10 states its own scope: "This section describes the contract of
  the compiled artifact; the reference interpreter implements that policy as a layer over its host
  runtime's allocator." Measured per engine and per primitive:

  | §10 row | Interpreter (`internal/lang`) | Compiled artifact (`compiler/`) |
  |---|---|---|
  | `delete x;` | any variable: runs `__delete__()` for a struct, then returns a `List`'s block to the free pool (`eval.go` `DeleteStmt`); a scalar is a silent no-op | **only a `List<int>` the callee owns — a local variable, or a `copyd` parameter whose callee-owned copy is released**; every other operand is a hard error. A by-reference parameter is refused with `delete on a by-reference parameter %q is not supported (the compiler releases a local List<int> variable only; the parameter's storage belongs to the caller)`: the interpreter accepts it and requeues the caller's block, so the refusal is this table's record of a gap, not a release |
  | `memory.setBlock(n)` | implemented, `1 ≤ n ≤ 2^20` (`eval.go` `MemoryManager`, `typecheck.go` `setBlock`) | none: `calling method ? on setBlock is not supported by the compiler (available in the interpreter)` |
  | `memory.compact()` | implemented (`MemoryManager.Compact`) | none: same hard error |
  | `memory.Fragmentation()` | **not exposed**: the member does not exist, although `MemoryManager.Fragmentation()` does (`eval.go`) | not exposed: the shared type checker refuses it first, `TypeError: no method "Fragmentation" on memory` |
  | (`memory.clear()`, not a row of §10) | implemented | none: same hard error as `setBlock` |

  So the compiled artifact implements one of §10's four rows, in a narrowed form, and the interpreter
  implements three of them (one of those — `clear` — is not in §10's table at all). The narrowed `delete`
  row is explicit about ownership: 0.8.5 accepted `delete` on a by-reference `List<int>` parameter and
  emitted nothing, which was neither a release nor an error; the compiled path now refuses that form, and
  the interpreter still releases the caller's block — an engine difference recorded here rather than
  hidden behind an accepted no-op. Closing the rest on the compiled path means writing the block policy
  into `compiler/qthreads.c.txt`, which is `malloc`/`calloc`-based today (24 allocation sites, no pools,
  no block sizes, nothing to hook), and then lowering four primitives plus `__delete__` dispatch into
  `compiler/internal/cgen`: a new allocator with a free list and an occupancy rule, three runtime entry
  points, `memory.*` call lowering, and `delete` for every variable kind. Size: large — one design review
  of its own, not a correction. Until then this table is the honest statement of the divergence §10's
  scope sentence assumes away.

### 11.4 What `docs/primitives.md` §7.6 asks for, and what this revision does not do

§7.6 lists twelve changes, smallest first, and calls the first nine pure restatement. This revision adopts
the ones that are corrections or one-line facts and records the rest with their size, because a rewrite of
a normative section is not a correction and §12 puts it behind the owner's ruling:

| §7.6 | Item | State |
|---|---|---|
| 1 | generate §11 outside the standard | **not done**: it needs a new generator plus a drift gate of the `cmd/qki18n -check` kind, and it would remove the section this revision has just made accurate. Size: one tool and its gate |
| 2 | add the two missing corpus kinds | **done**: `z_legacy_views.kq` (one case per §3.2 view row the standard claims is core) and `z_precedence.kq` (one line per precedence level, against §5 rather than against the other engine) |
| 3 | record the tool/lint boundary in one sentence | **done**: the paragraph closing §1 |
| 4 | state what `compare.sh` proves | **done**: the last sentence of §1.4 |
| 5 | `stdlib/vec.qk`'s `pointer T`, or §3.2 corrected | **done as the correction**: `pointer T` is measured to exist in both engines and is what `new T[n]` yields, which `stdlib/vec.qk` is built on, so §3.2's claim that the spelling does not exist was the factual error |
| 6 | delete `#ifdef`/`#ifndef` in favour of `#if defined(…)`, and `#ifn` | **not done**: deleting a spelling is surface removal, and the spellings live in the syntax layer's preprocessor (a sibling repository). §8's Q-4 also records that `#ifn` is defined nowhere, so a ruling is needed before either list shrinks |
| 7 | collapse `IndexOutOfBoundsError`/`IndexOutOfRangeError` to one id | **not done**: §7.6 itself notes it "changes a user-visible message". The two name different conditions (a `List` subscript and a bit subscript), both are in the message catalogues of both engines and in the compiled runtime's strings, so it is a surface change awaiting the ruling, not a restatement |
| 8 | reduce §4's 28 reference types to a `(set, rank)` algebra | **not done**: §4 is normative and §8's Q-1/Q-2 record that `m`'s meaning for a reference and `a`'s definition are still open — a rewrite would answer questions this document says are unresolved |
| 9 | reduce §10 to release + block granularity | **not done**: §10 is normative and §7.6's own Q-6 ("does release wipe?") is open; rewriting the table would also erase the row-by-row status §11.3 has just recorded |

§7.6's items 10–12 are the semantic ones: item 10 (`char`/`double`) was closed the way §7.6 offers — by
lowering them in the compiler (§11.1) — and no program changes meaning; item 11 (shift precedence) is
**closed** — the ruling was given, the syntax layer carries the three-line exchange, this checkout pins it,
and §11.3 keeps the record of the divergence (§11.1 measures the closed state); item 12
(`#expand`/`#run`/`macro x($a $b)`) is untouched, needs a definition of "node" (§8's Q-3, Q-5, and §11.3's
macro item), and stays open.

## 12. Change policy and revision log

- Corrections of **factual errors** in this document, and fixes that make the implementation match it,
  are allowed at any time.
- **New surface** (syntax, types, operators, protocols) is not added during the freeze window.
- Anything that must change the standard waits for the review date and a version bump of this file.

| Version | Date | Change |
|---|---|---|
| 0.8.6 | 2026-10-17 | §11.3's `delete` row is corrected, and the silent acceptance behind it is closed: 0.8.5 accepted `delete` on a by-reference `List<int>` parameter and emitted no code — neither a release nor an error. Lowering now tracks by-reference parameters and refuses the form with a positioned message (`delete on a by-reference parameter %q is not supported (the compiler releases a local List<int> variable only; the parameter's storage belongs to the caller)`), while the supported shapes — a local `List<int>` variable and a `copyd` parameter's callee-owned copy — keep releasing their buffer and header. The interpreter still accepts the parameter form, so §11.3 records it as an engine difference; the new `compiler/testdata/cases/z_delete_local.kq` keeps the supported path byte-identical between the engines, and `TestCompiledMemorySurfaceRefusesBothLocales` pins the refusals of the whole audited memory surface in both languages through the catalog. Two diagnostics were appended at the append point, so no shipped id was renumbered. Status sections only; no normative text changed, no language surface added |
| 0.8.5 | 2026-10-16 | §11.2's last `List` limit is closed: the compiler lowers `List<float>`, and `List<double>` with it. Elements live in i64 slots each carrying the element as a bit-cast double, the runtime renders them through the one float spelling a bare float gets, and the surface covers the literal, the element read and write, `append`, `size`/`get`/`toString`, for-in iteration and print. Pinned by `compiler/testdata/cases/z_list_double.kq` (which avoids list printing, so it also runs under `lli` in `go test`), `cases_run/z_list_double_print.kq` (the print and `toString` paths, which need the linked runtime) and `TestListDoubleIsLoweredLikeListInt`, which replaces the refusal test this section used to point at. What remains is what only `List<int>` has — `head`/`tail`/`next`/`reset`/`appendAll`/`__sort__`, `delete`, `interface{}` boxing — a declaration without an initializer builds the interpreter's empty list where `List<int>` still refuses one, and `copyd List<float>` is refused like `copyd List<String>` because the deep-copy helper copies 4-byte elements. The four diagnostics the change needed were filed at the append point of the message tables, so no shipped id was renumbered, and the texts that named only `List<int>`/`List<String>` now name `List<float>` too. Status sections and one lowering; no normative text changed, no language surface added |
| 0.8.4 | 2026-10-16 | **§11.3's shift divergence is closed.** The pinned syntax layer now threads `parseBitAnd → parseShift → parseAdd → parseMul`, so §5's order holds: `1 + 2 << 3` is 24, `1 << 2 + 3` is 32 and `10 - 2 >> 1` is 4 in both engines under `LANG=C.UTF-8` and `LANG=zh_CN.UTF-8`, while `1 + (2 << 3)` still reads 17 — the parser erases the parenthesis node but decides the grouping, so the two readings are two trees and neither is traded for the other. The pin moves to `github.com/QuarkLangCommunity/QuarkLangQkparser v0.0.0-20261015231811-3de63a5ff52d` in both `go.mod` files, the closed state is measured in §11.1, the history of the divergence stays in §11.3, and item 11 of §11.4 is now closed. Of the 40 corpus cases the pin then covered exactly one changed value, `z_precedence.kq`, the case that pins this level against the standard rather than against the other engine, and only its three shift rows; no existing program changed. Status sections only; no normative text changed, no language surface added |
| 0.8.3 | 2026-10-15 | Status and corrections, no normative text changed: §11.3 records the two divergences it could not see before — §5's shift precedence (a two-line exchange in the pinned syntax layer, measured in both engines under both locales, with the reason it cannot be repaired in this repository: that parser erases parentheses) and §10's four primitives per engine and per path, with the size of closing the compiled path; §11.2 adds the one `List` element type the compiler still refuses (`List<double>`), and §11.1 the legacy spellings both engines now accept (`char`, `double`, `long`, `pointer T`, `List<char>`), which the compiler refused until this revision. §1.4 now says what `compare.sh` proves and §1 records that diagnostics are not language surface; §3.2's claim that `pointer T` does not exist is corrected (it is what `new T[n]` yields, which `stdlib/vec.qk` is built on); §9 lists `pointer T` among the compatibility spellings. §11.4 records which of `docs/primitives.md` §7.6's twelve items this revision adopts and why the others wait for the owner's ruling |
| 0.8.2 | 2026-10-11 | §10 gains one scope sentence: the section describes the contract of the compiled artifact, and the reference interpreter implements that policy as a layer over its host runtime's allocator. The corresponding doc comment in `internal/lang/value.go` said memory "is backed by the host Go GC" and cited a `§14` that does not exist; it now describes what the reference interpreter does (its `MemoryManager` implements the block policy as a logical layer above the host allocator) and cites §10. Scope wording and a stale citation only; no normative content changed and no language surface added |
| 0.8.1 | 2026-10-10 | §11.2 records the two depth limits a user can hit, with the numbers and what they see: call depth 8192 (`lang.MaxCallDepth`, `StackOverflowError: recursion depth exceeded 8192`, counted identically in the interpreter and in generated code) and expression nesting 65536 (`lang.MaxExprDepth`/`qkparser.MaxExprDepth`, refused by the parser as `nesting is deeper than …` and by the type checker as `expression nesting is deeper than …`). Status-section facts only; no normative content changed and no language surface added |
| 0.8 | 2026-10-10 | **Memory and lifetime became part of the standard** (§10): no garbage collector, `delete` is a deterministic release that runs `__delete__()` and returns the storage to a free pool without wiping it, allocation reuses the least-occupied block so a tide of temporary data keeps a stable block set, `memory.setBlock`/`compact`/`Fragmentation` are the knobs and the inspection point, and a task's blocks become reclaimable when it ends. The section after it moved to §11 and the change policy to §12. Freeze restarts: review after 2026-11-10 |
| 0.7.2 | 2026-10-09 | Reorganization and status correction: §6.5/§6.6 became §7 "Program shape and runtime" with subsections, the following sections were renumbered, and a new §10 "Implementation status" was added — the old note still claimed `bit`/`bits<N>`/`uchar` were unimplemented, while both engines have had them since PR #60. §10 now lists what is implemented, the implementation's limits, and the three places where the code does not yet follow the standard (§7.2 macro spelling, §8 no exceptions, §9 built-in containers). No normative content changed |
| 0.7.1 | 2026-10-08 | `uchar` added: the unsigned 8-bit view over `bits<8>` (`0 … 255`), the octet type and the element type for raw data — there is no signed 8-bit view. §3.1 now states explicitly that every basic type is *implemented from* `bit`/`bits<N>`. Value conversions narrow by keeping the low bits |
| 0.7 | 2026-10-08 | The storage family is measured in **bits**: `byte`/`bytes<N>` became `bit`/`bits<N>`. A view's name now states its own width (`int32` is `bits<32>`, `float64` is `bits<64>`), layout is `⌈N/8⌉` bytes with the bits past `N` zero, and `b[i]` reads or writes the i-th bit. Freeze restarts: review after 2026-11-08 |
| 0.6.1 | 2026-10-08 | Corrections: integer views admit the bitwise family `& \| ^ ~` (0.6 had stated it for raw bytes only), operator precedence written down, hex integer literals recorded |
| 0.6 | 2026-10-08 | `byte` made the mother of every basic type; `bytes<N>` added; basic types restated as views over it; raw bytes restricted to bitwise operations; the three meanings of the conversion call `T(x)` fixed; layout rules written down; the split type table and the duplicated §8 row of 0.5 repaired. Freeze restarts: review after 2026-11-08 |
| 0.5 | 2026-10-07 | integers and floats split by width, pointers separated from references, `char` added, `String` moved to the library, `null` and `pointer T` removed, `program` and the runtime directives recorded |
