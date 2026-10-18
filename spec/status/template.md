# Implementation status

This report is the status section §11 of `spec/STANDARD.md` cites. It is **not normative**: the standard
is the source of truth, and where the code differs, the code is wrong.

It is generated from `spec/status/claims.tsv`, where every claim names the test or parity case that pins
it, and from this template, which holds the prose that cannot be generated. `go generate
./internal/statusgen` rewrites the report; `go run ./cmd/qkstatus -check`, driven by
`TestGeneratedStatusIsCurrent`, fails when the committed report is stale. Evidence is resolved against
the tree before rendering, so a claim cannot cite a test that no longer exists and a corpus case cannot
be left out silently.

## 11.1 Implemented, in both engines

{{table:11.1}}

{{corpus}}

This table is the corrected form of the old hand-written list. That list grouped `===`, permission
references, `&lvalue` and `p++` with the constructs both engines implement; the compiler refuses all
four (§11.3), so they are recorded there now, and the generated table cannot repeat the mistake: each
row's compiler evidence is a compiler test or a parity case. The conditional row excludes `#ifn`, which
is unimplemented and recorded as a gap in §11.3.

## 11.2 Limits of the current implementation

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
  refused because the compiler releases an owned `List<int>` only (§11.3's §10 table); boxing one into
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

## 11.3 The code does not follow the standard here

{{table:11.3}}

The interpreter-only rows are the project's hard-error policy: when the compiler cannot reproduce the
interpreter's semantics exactly, it refuses the program rather than compiling something that only looks
the same. That is what makes "the interpreter is the semantic reference" true rather than aspirational.
The `#ifn` row is a missing definition, not a missing implementation: the standard names it, defines it
nowhere, and the safer reading recorded there is `#if !(condition)`, the complement of `#ifndef`.

### §5 shift precedence — closed in 0.8.4

Kept here as the record of what the divergence was. §5 says: "Precedence, loosest first: `||`, `&&`,
`== != < <= > >=`, `|`, `^`, `&`, `<< >>`, `+ -`, `* / %`, then the unary operators." Read as written,
`<<`/`>>` is *looser* than `+`/`-` (C's grouping), so `1 + 2 << 3` is `(1 + 2) << 3` = 24; both engines
grouped it the other way (Go's) until 0.8.4:

| Expression | Both engines, before | §5's order |
|---|---|---|
| `1 + 2 << 3` | 17 — `1 + (2 << 3)` | 24 — `(1 + 2) << 3` |
| `1 << 2 + 3` | 7 — `(1 << 2) + 3` | 32 — `1 << (2 + 3)` |
| `10 - 2 >> 1` | 9 — `10 - (2 >> 1)` | 4 — `(10 - 2) >> 1` |

The cause was never in this repository: the syntax layer is the pinned module
`github.com/QuarkLangCommunity/QuarkLangQkparser`, whose parser called `parseShift` from `parseAdd`
(`parseAdd → parseShift → parseMul`). The fix is that chain's exchange; the syntax layer carries it (its
PR QuarkLangQkparser#5, pinned here as `v0.0.0-20261015231811-3de63a5ff52d`), and §11.1 of the old
standard carried the measurement of the closed state. The old §11.3 recorded the history; this report
keeps it in one place so the fix is not mistaken for permission to relax the pin.

### §10 memory on the compiled path

The four primitive rows and the derived `clear` are the surface §10 defines. The compiled artifact
implements `delete` for an owned variable of every heap-backed type it lowers — a list of any element
type it supports (`List<int>`, `List<String>`, `List<float>`), released through the counted-slab
allocator as a buffer plus a header, or a struct, whose `__delete__()` runs before its cell is
returned — for a local variable or a `copyd` parameter's callee-owned copy. A by-reference parameter
(the storage belongs to the caller), a scalar or a `String` (no heap storage to release) and a
non-variable expression keep positioned hard errors. Stage 2 has also wired
`memory.setBlock(n)` to the runtime block allocator: the accepted range is the interpreter's `1 … 2^20`,
a value outside it is the interpreter's runtime error with the same message and position (catchable by
`try`), and the size applies to slabs created afterwards, as the interpreter reads `BlockSize` when a
`MemBlock` is created. `memory.compact()` is wired too, and with the interpreter's answer to D6: it
returns, so control continues after the call in both engines (only the interpreter's comment says
"never returns at all"). `memory.clear()`, the derived wipe, is wired to a runtime that releases the
empty slabs and zeroes every still-free slot of a retained slab, so the region it clears cannot hand
the previous value's bytes to the next user while a live payload is untouched; the interpreter's clear
drops the emptied logical blocks, which is the release half without bytes to zero (its blocks are Go
values, not storage). `memory.Fragmentation()` remains unexposed in both engines. Stage 1 of the design
([`compiled-memory-stage1.md`](compiled-memory-stage1.md)) put the block policy in the runtime: counted
slabs with size classes, a per-class free list, per-object release, occupancy and fragmentation
accessors, `compact`, large-object blocks, and the code generator's 20 initial allocation sites plus 4
growth sites routed through `ql_alloc`/`ql_calloc`/`ql_realloc`/`ql_free`. Stage 2 also routes the
runtime's own C allocations — string builders, split lists, hash tables and their bucket arrays, memo
entries, interface boxes, channels and task objects — through the same API (D1), so the occupancy and
fragmentation numbers describe the whole compiled artifact instead of only what the code generator
emits. What remains outside the metric is the allocator's own host block storage, which is the pair
(`ql_mem_host_alloc`/`ql_mem_host_free`) the metric measures the inside of; static data and the C
stack are not allocations. What remains is exposing `Fragmentation` in both engines or deciding not
to, task-end reclaim, and Windows (the runtime is still not linked there; the generator keeps the
libc path).
Stage 2 adds no normative text and no language surface beyond the rows the standard already names.
Size: large — one design review of its own, not a correction.

### §10 memory on Windows

`spec/status/claims.tsv` 11.3-15 records the platform gap. The compiled runtime is not linked on Windows,
so stage 1's counted-slab allocator is absent there: compile and link keep the libc path and `delete` on an
owned `List<int>` still releases host storage. What a Windows user loses is the policy itself — there is no
occupancy or fragmentation accounting and no block-reuse policy, so the allocator's stable block set
cannot be observed; the `memory.setBlock`, `memory.compact` and `memory.Fragmentation` accessors have no
compiled implementation on any platform yet, and on Windows there is not even the internal counter they
would read. This is the accepted position, not a promise. Closing it means linking the compiled runtime on
Windows (the design's R8/T6) or splitting the allocator into a platform-neutral translation unit compiled
on all three; either is stage 3 work. Size: large — one runtime port or one allocator extraction, not a
correction.

## 11.4 What `docs/primitives.md` §7.6 asks for, and what this revision does not do

§7.6 lists twelve changes, smallest first, and calls the first nine pure restatement. This revision adopts
the ones that are corrections or one-line facts and records the rest with their size, because a rewrite of
a normative section is not a correction and §12 puts it behind the owner's ruling:

| §7.6 | Item | State |
|---|---|---|
| 1 | generate §11 outside the standard | **done**: the standard's §11 is now a citation, the report is generated by `cmd/qkstatus` from `spec/status/claims.tsv` and `spec/status/template.md`, and `TestGeneratedStatusIsCurrent` fails when it is stale |
| 2 | add the two missing corpus kinds | **done**: `z_legacy_views.kq` (one case per §3.2 view row the standard claims is core) and `z_precedence.kq` (one line per precedence level, against §5 rather than against the other engine) |
| 3 | record the tool/lint boundary in one sentence | **done**: the paragraph closing §1 |
| 4 | state what `compare.sh` proves | **done**: the last sentence of §1.4 |
| 5 | `stdlib/vec.qk`'s `pointer T`, or §3.2 corrected | **done as the correction**: `pointer T` is measured to exist in both engines and is what `new T[n]` yields, which `stdlib/vec.qk` is built on, so §3.2's claim that the spelling does not exist was the factual error |
| 6 | delete `#ifdef`/`#ifndef` in favour of `#if defined(…)`, and `#ifn` | **not done as a deletion, done as the record**: deleting a spelling is surface removal, and the spellings live in the syntax layer's preprocessor (a sibling repository); the standard's status records that `#ifn` is defined nowhere, with the safer reading `#if !(condition)`, and `TestIfnHasNoDefinedMeaning` pins the refusal. Defining or deleting it still needs the owner's ruling |
| 7 | collapse `IndexOutOfBoundsError`/`IndexOutOfRangeError` to one id | **done as the opposite**: the two name different conditions — a container subscript and a bit subscript (or a reference moved past the end) — so §8 now defines each one and where it is raised, and the names stay distinct. The catalog rows and both runtime messages are unchanged; two tests pin the distinction |
| 8 | reduce §4's 28 reference types to a `(set, rank)` algebra | **done as the reconciliation, not the rewrite**: Q-1 and Q-2 are answered in §4 without new surface — the reference variable may be re-bound with `m` while its referent does not move, the four scopes are one lifetime order `u > t > a > f`, `a` has no behaviour beyond that rank, and the empty permission set is not spellable because it would permit nothing. The 7 × 4 enumeration is exactly what the checker accepts, so reducing it to an algebra is a later simplification, not a correction |
| 9 | reduce §10 to release + block granularity | **done as the reconciliation, not the rewrite**: Q-6 is answered in §10 without new surface — `delete` does not wipe, `memory.clear()` is the derived wipe (release plus zero-fill), and the four primitive rows plus that derived operation are the surface; the per-engine divergence for those rows stays in §11.3, so reducing the table further is a later simplification rather than a factual correction |

§7.6's items 10–12 are the semantic ones: item 10 (`char`/`double`) was closed the way §7.6 offers — by
lowering them in the compiler (§11.1) — and no program changes meaning; item 11 (shift precedence) is
**closed** — the syntax layer carries the exchange, this checkout pins it, and the history is above; item
12 (`#expand`/`#run`/`macro x($a $b)`) is untouched, needs a definition of "node" (§8's Q-3, Q-5), and
stays open.
