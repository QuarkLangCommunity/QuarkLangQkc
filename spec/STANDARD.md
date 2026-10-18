# QuarkLang Standard — frozen snapshot

**Version:** 0.8.13 — **`bit` is the mother of every basic type**: every basic type is *implemented from*
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
   engines implement the other way round, or a feature both engines omit, leaves it green, so the status report cited by §11 lists
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
  These seven are the complete permission set; the empty set is not a permission, because a reference
  carrying it could neither read, write, nor move.
- **scope** is `u` (program), `f` (function), `a` (area), `t` (thread) — 7 × 4 = **28 reference types**.
  Scopes order by lifetime, widest first: `u` > `t` > `a` > `f`; a reference may be stored only in a
  variable of equal or narrower scope. `a` is an area scope, and this standard gives it no behaviour
  beyond that rank.
- A bare `T&` keeps full permission and program scope.
- `&lvalue` takes the address of a variable, a struct field or a list element; writes through it reach
  the original storage.

**References and pointers are different things, not two spellings of one.** A reference (`&`) is an
alias of storage; the storage it names does not move. A pointer (`*`) is an address value. `p++` advances
a pointer, or a reference that names a container element, one element and rebinds the variable holding it
(requires `m`), and moving past the end is an error rather than a dangling pointer: a reference is an
address with a permission, and `m` is the bit that lets the address be re-bound.

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
- Runtime errors that remain (`DivisionByZeroError`, `IndexOutOfBoundsError`, `IndexOutOfRangeError`,
  `NullPointerError`, `PanicError`) are reported with position and an execution log.
- **`IndexOutOfBoundsError`** is a **container subscript** outside the container: a `List` read or write
  `l[i]` with `i` outside `0 … size-1`, reported as
  `IndexOutOfBoundsError: index <i> out of range [0,<size>)`. Strings have their own
  `StringIndexOutOfBoundsError`, which names the string operation and its range.
- **`IndexOutOfRangeError`** is a **bit subscript** outside the width of §3.3 (`b[i]` with `i` outside
  `0 … N-1` when `i` is not a constant), or a **reference moved past the end** of its container in §4
  (`p++`). These are different conditions from the container subscript above, so the names stay
  distinct; §11 records that the reference-move form is interpreter-only today.

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
  used as a narrow integer becomes `uchar`. The status report cited by §11 records what each one means in the implementation today.
- **Legacy built-ins still present** (`List`, `HashTable`): scheduled for migration onto library types.
  Their presence is frozen by a test — the count may only go down. See the status report cited by §11 (§11.3).

## 10. Memory and lifetime

QuarkLang has **no garbage collector**. Storage is released explicitly and the release is
deterministic: nothing is collected behind your back, and no allocation ever waits for a collection.

**Scope.** This section describes the contract of the compiled artifact; the reference interpreter
implements that policy as a layer over its host runtime's allocator.

| Primitive | Meaning |
|---|---|
| `delete x;` | release `x`'s storage: run `__delete__()` if the type declares it, then return the storage to the free pool. The payload is **not** wiped — wiping is `clear`'s job, so a released block never hands another value's bytes to the next user |
| `memory.setBlock(n)` | set the block size, the unit the runtime allocates and reuses |
| `memory.compact()` | free the blocks nobody occupies; the call returns no value and control continues |
| `memory.Fragmentation()` | the unused space inside live blocks, as a ratio (inspection) |

`memory.clear()` is the wiping operation the `delete` row refers to: it is derived — release plus
zero-fill, not a separate primitive — so `delete` keeps the bytes and `clear` removes them. The table's
four rows are the primitive surface, and `clear` is the one derived operation the `delete` row already
names.

**Reuse is the point.** Allocation prefers the least-occupied block that has room, so a workload whose
temporary data arrives in *tides* — allocate, use, delete, repeat — reuses a small, stable set of blocks
instead of growing the heap. The policy above is normative; the numbers are generated measurements.
The reference interpreter implements the policy as a logical layer over the host allocator. Stage 1 of
the compiled memory design (`docs/compiled-memory-stage1.md`) routes the code generator's allocations
through the counted-slab runtime allocator on POSIX; stage 2 (`docs/compiled-memory-stage2.md`) routes
the runtime's own C allocations through the same allocator, so the allocator's own host block storage
is the only allocation outside the metric, and Windows keeps the libc path because the runtime is not
linked there. The checked-in `bench/tide` harness makes the compiled `List` allocation observable with
an opaque sink, reports the no-sink and historical no-use rows (no longer optimizer floors on POSIX
after stage 1), and drives the interpreter's counters. A recording run is refused unless the host's
1-minute load average is below 0.25 per logical CPU, and the generated block states the load the table
was taken under. The earlier compiled `1 ms` figure measured the legacy dead-code floor on the
pre-stage-1 libc path, not allocation. The block below is generated from `spec/bench/tide.tsv`;
`TestTideClaimsMatchMeasurement` fails when this copy, the copy in `README.md`, or
`docs/tide-benchmark.md` disagrees. When a task ends, the blocks it owns become reclaimable.

<!-- tide-measurements -->
Measured on Linux x86_64, 16 logical CPUs. Toolchain: clang version 22.1.8 (Fedora 22.1.8-4.fc44); go version go1.26.9-X:nodwarf5 linux/amd64; rustc 1.99.0 (b940084d7 2026-09-28) (Fedora 1.99.0-1.fc44). Flags: qkc default -O3 -flto=thin; C clang -O3 -flto=thin; Rust rustc -O; Go go build (default). Skipped controls: none.

What is counted: in-process loop wall time: clock()/clock_gettime(CLOCK_MONOTONIC)/Instant::now()/time.Now(); process startup and printing excluded; observable rows call an opaque sink so the allocation cannot be removed. Evidence: the observable QuarkLang binary defines ql_alloc/ql_free and routes the List allocation and the toString string through them; it imports malloc/free only for the allocator's host blocks.

| Workload | Rounds | Runs | Median | Min–max | Spread | Checksum | What is counted |
|---|---:|---:|---:|---:|---:|---:|---|
| QuarkLang compiled, observable | 1e8 | 5 | 3.62 s | 3.53–3.74 s | 5.84% | 899,999,977 | `List<int>` literal + `toString` string (both counted slabs since D1) + opaque C sink + release + delete |
| C control, volatile sink | 1e8 | 5 | 466.00 ms | 455.55–481.66 ms | 5.60% | 899,999,977 | malloc(12) + 3 writes + volatile sink + free |
| Rust control, black_box | 1e8 | 5 | 678.43 ms | 648.99–754.48 ms | 15.55% | 899,999,977 | Vec::with_capacity(3) + 3 pushes + black_box(ptr) + drop |
| Go control, global sink | 1e8 | 5 | 1.28 s | 1.23–2.24 s | 78.84% | 899,999,977 | make([]int32,3) + global sink + GC |

The same loops without the sink. The checksum keeps the loop alive; on the pre-stage-1 libc path the legacy row removed the use entirely. Since stage 1 the QuarkLang rows call the opaque counted-slab allocator, so they measure allocation too; the C/Rust/Go floor rows still measure their own optimizer floors.

| Workload | Rounds | Runs | Median | Min–max | Spread | Checksum | What is counted |
|---|---:|---:|---:|---:|---:|---:|---|
| QuarkLang compiled, no sink | 1e8 | 5 | 1.15 s | 1.10–1.17 s | 5.66% | 899,999,977 | same loop with the sink removed; the checksum keeps the loop alive; the opaque ql_alloc/ql_free calls are not elided |
| C control, no sink (optimizer floor) | 1e8 | 5 | 92.68 ms | 92.20–98.08 ms | 6.34% | 899,999,977 | same loop with the volatile sink removed |
| Rust control, no black_box (optimizer floor) | 1e8 | 5 | 165.44 ms | 159.40–174.47 ms | 9.11% | 899,999,977 | same loop with black_box removed |
| Go control, local slice (optimizer floor) | 1e8 | 5 | 111.48 ms | 104.18–122.11 ms | 16.08% | 899,999,977 | same loop with the global sink removed |
| QuarkLang compiled, legacy shape (no read/sink) | 1e8 | 5 | 2.05 s | 1.55–2.21 s | 32.20% | - | allocate/delete with no read, sink or checksum: the shape the old 1 ms claim measured on the pre-stage-1 libc path; stages 1-2 run it through counted slabs |

The historical `1 ms` claim measured the legacy row on the pre-stage-1 libc path, where LLVM removed the loop; that is what `0 µs` meant. On the stage-1 tree the same row is no longer erased (the allocator call is opaque), so the recorded value is a real allocation measurement and the label no longer calls it a dead-code floor.

| Policy measurement | Waves | Allocations | Value | Note |
|---|---:|---:|---:|---|
| Interpreter block policy, reuse workload | 10,000 | 1,000,000 | 99.9998% | ReusedCount/AllocCalls; the block count converges to 2 blocks |
| Interpreter block policy, mixed survival | 1,000 | 1,000,000 | 0.1953% | Fragmentation() after the mixed-size waves; 3005 blocks stay live by design |
<!-- /tide-measurements -->

Releasing storage that something still refers to is a programming error: the runtime does not detect it,
and the permission system of §4 is what keeps it visible at compile time.

## 11. Implementation status

This section is **not normative**. The status report — what is implemented in both engines, what each
engine refuses, the limits and the remaining divergences — is generated from machine-checkable facts into
`docs/implementation-status.md`, and is not maintained by hand here. Its claims and their evidence live
in `spec/status/claims.tsv`, the prose that cannot be generated lives in `spec/status/template.md`, and
`go run ./cmd/qkstatus -check` (driven by `TestGeneratedStatusIsCurrent`) fails when the committed report
is stale. The standard is the source of truth: where the code differs, the code is wrong.

## 12. Change policy and revision log

- Corrections of **factual errors** in this document, and fixes that make the implementation match it,
  are allowed at any time.
- **New surface** (syntax, types, operators, protocols) is not added during the freeze window.
- Anything that must change the standard waits for the review date and a version bump of this file.

| Version | Date | Change |
|---|---|---|
| 0.8.13 | 2026-10-18 | **§10's measurement paragraph and generated block now describe stage 2 of the compiled memory design.** The runtime's own C allocations — string builders, split lists, hash tables, memo entries, interface boxes, channels and task objects — are routed through the same counted-slab allocator the code generator uses (D1), so the occupancy and fragmentation numbers describe the whole compiled artifact; the allocator's own host block storage is the only allocation outside the metric. The tide table was re-recorded on the stage-2 tree, and a sink that releases a language string now calls `ql_free` because the string is a counted-slab pointer. The §10 primitive rows and policy are unchanged; measurement wording only, no language surface changed |
| 0.8.12 | 2026-10-17 | **§10's measurement paragraph and generated block now describe stage 1 of the compiled memory design.** The compiled `List` allocation is routed through the counted-slab runtime allocator on POSIX; runtime helper strings remain on the host allocator, and Windows keeps the libc path. The tide table was re-recorded on the stage-1 tree, and the historical no-use row is no longer an optimizer floor because `ql_alloc`/`ql_free` are opaque. The §10 primitive rows and policy are unchanged; measurement wording only, no language surface changed |
| 0.8.11 | 2026-10-17 | **§10's tide numbers are generated, not hand-written, and the old `1 ms` is named for what it measured: LLVM dead-code elimination, not allocation.** The checked-in `bench/tide` harness makes the compiled allocation observable with an opaque sink and reports the observable path, the sink-free and legacy floors, the C/Rust/Go controls, and the interpreter's reuse/fragmentation/block counters; a missing toolchain is skipped explicitly. `spec/bench/tide.tsv` is the measured record, `docs/tide-benchmark.md` is generated from it, and the marked block is embedded here and in `README.md`; `TestTideClaimsMatchMeasurement` fails when the three copies or the recorded arithmetic disagree. The prior 99.96% reuse and 0.195% fragmentation figures are replaced by the measured 99.9998% and 0.1953% (the second was already right). The §10 primitives and policy are unchanged; measurement wording and the historical correction only, and no language surface changed |
 | 2026-10-17 | **§10's tide numbers are generated, not hand-written, and the old `1 ms` is named for what it measured: LLVM dead-code elimination, not allocation.** The checked-in `bench/tide` harness makes the compiled allocation observable with an opaque sink and reports the observable path, the sink-free and legacy floors, the C/Rust/Go controls, and the interpreter's reuse/fragmentation/block counters; a missing toolchain is skipped explicitly. `spec/bench/tide.tsv` is the measured record, `docs/tide-benchmark.md` is generated from it, and the marked block is embedded here and in `README.md`; `TestTideClaimsMatchMeasurement` fails when the three copies or the recorded arithmetic disagree. The prior 99.96% reuse and 0.195% fragmentation figures are replaced by the measured 99.9998% and 0.1953% (the second was already right). The §10 primitives and policy are unchanged; measurement wording and the historical correction only, and no language surface changed |
| 0.8.10 | 2026-10-17 | §11's generated status report records the correction of 0.8.5's silent `delete`: on a by-reference `List<int>` parameter the compiler accepted the form and emitted nothing, neither a release nor an error, while the interpreter releases the caller's block. Lowering now tracks by-reference parameters and refuses them with a positioned message; an owned `List<int>` — a local variable or a `copyd` parameter's callee-owned copy — still releases its buffer and header. `spec/status/claims.tsv` 11.3-08 and `spec/status/template.md` §10 record the narrowed ownership rule and the interpreter/compiler gap, pinned by `TestCompiledMemorySurfaceRefusesBothLocales` (eight forms, both languages), `TestDeleteOnOwnedListIntStillLowers`, `TestDeleteOnCopydListIntParameterStillLowers`, the kept `TestMemoryPrimitivesAreRefusedExceptListDelete`, and `compiler/testdata/cases/z_delete_local.kq` (byte-identical in both engines). Two diagnostics were appended at the append point, so no shipped message id was renumbered. Status sections only; no normative text changed, no language surface added |
| 0.8.9 | 2026-10-17 | **§11 is generated, not hand-written.** The section is now a citation to `docs/implementation-status.md`, rendered by `cmd/qkstatus` from `spec/status/claims.tsv` (each claim names the test or parity case that pins it) and `spec/status/template.md` (the prose that cannot be generated). Generation resolves every evidence reference, requires a test and both engine paths for each implemented claim, and checks that every parity case is covered; `TestGeneratedStatusIsCurrent` regenerates in memory and fails when the committed report is stale, in the same style as `cmd/qki18n -check`. The generated report corrects the old §11.1 list, which had grouped `===`, permission references, `&lvalue` and `p++` with the constructs both engines implement although the compiler refuses all four; they are §11.3 rows now. The report also carries §11.2's limits, the closed shift-precedence history, the §10 per-engine table and §11.4's item states. Status section only; no normative text changed, no language surface added |
| 0.8.8 | 2026-10-17 | §4 and §10 now say the measured truth their open questions left unresolved, without new surface. §4 distinguishes the referent from the reference variable: a reference's storage does not move, while `p++` re-binds the variable holding either a reference or a pointer when it has `m`; the four scopes are one lifetime order `u > t > a > f`, `a` has no behaviour beyond that rank, and the seven permission spellings are complete because the empty set would permit nothing. §10 names `memory.clear()` as the derived wiping operation (release plus zero-fill) the `delete` row already refers to, so release does not wipe and the four primitive rows stay the surface. These answer `docs/primitives.md` Q-1, Q-2 and Q-6; §11.4's items 8 and 9 record them as reconciliations rather than rewrites. No syntax, type, operator or protocol was added |
| 0.8.7 | 2026-10-17 | §11.3 records what §7.2 leaves open about `#ifn`: the standard names it and defines it nowhere, the shared preprocessor gives it no meaning (only `#if`/`#elif` open a group, so the following `#endif` is refused in both engines), and the safer reading — if it is ever defined — is `#if !(condition)`, the complement of `#ifndef`. The choice is to record the gap rather than invent the semantics: defining it needs surface and a syntax-layer change, while deleting it is surface removal, so both wait for the owner. `TestIfnHasNoDefinedMeaning` pins that the implementation refuses it instead of accepting an invented meaning. §11.4's item 6 is updated to the record. Status section only; no normative text changed, no language surface added |
| 0.8.6 | 2026-10-17 | §8 defines the two index errors it used to leave ambiguous: `IndexOutOfBoundsError` is a container subscript outside `0 … size-1` (a `List` read or write), while `IndexOutOfRangeError` is a bit subscript outside `0 … N-1` (§3.3) or a reference moved past the end of its container (§4, interpreter-only today). The names stay distinct because they name different conditions and a user sees them; `internal/lang/error_names_test.go` pins one case each and asserts the other name is absent. The header's version number was stale at 0.8.3 while the log already carried 0.8.4 and 0.8.5; it now matches the log. Correction only; no language surface changed |
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
