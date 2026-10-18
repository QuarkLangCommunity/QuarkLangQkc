# The compiled path's memory model: a design review

**Status.** Design review only. This document changes no code, does not edit `spec/STANDARD.md`, and
decides nothing: §5 lists what the owner must decide *before* code is written, marked as decisions
rather than resolutions. It is not the standard; where it reports a divergence, the standard is quoted
and the ruling is left to the owner.

**Reviewed at** `ad7a98c` (tip of `origin/main` when this review was written, 2026-10-16), against
`spec/STANDARD.md` **0.8.3** as its header says (`spec/STANDARD.md:3`) even though the revision log
already carries 0.8.4 and 0.8.5 entries (`spec/STANDARD.md:479-480`). Every `file:line` below is a claim
about that revision. The implementation moves; a later reader should re-measure rather than trust a
line number.

**Conventions.**

| Symbol | Meaning |
|---|---|
| ✔ | read or run at the cited site / command |
| ⚠ | verified for some cases, an estimate, or a number this review could not reproduce |
| ✘ | **UNVERIFIED** — stated as open rather than asserted |
| ESTIMATE | a size or cost this review did not measure; only the line ranges it is based on are measured |

**What this review found, in one table.**

| # | Finding | Evidence |
|---|---|---|
| F1 | The compiled path implements one §10 row in a narrowed form (`delete` on a `List<int>` variable) and has no lowering at all for `memory.*`; `Fragmentation` is exposed in neither engine. This is the recorded gap. | §1.2, §1.3; `compiler/internal/cgen/lower.go:1176-1184`, `:2726`; `internal/lang/typecheck.go:2474-2496` |
| F2 | §11.3's "24 allocation sites" is not reproducible at the recording revision: the plainest count is 36 initial allocation calls there and 37 now, plus `realloc` and one `free`. | §2.1; counts in Appendix A.1 |
| F3 | Almost nothing is ever freed: the embedded runtime has exactly **one** `free` (a hash-table bucket array), and generated code frees only a `List<int>` on an explicit `delete`. Everything else lives until process exit. | §2.2–§2.4; `compiler/qthreads.c.txt:801`; `compiler/internal/cgen/cgen.go:2402-2413` |
| F4 | The advertised compiled "Tide, 1e8 rounds = 1 ms" is **not a measurement of allocation**: the project's own changelog names "identical allocation elimination", and a probe of the same shape at `-O3` compiles to no malloc/calloc/free calls while the same IR at `-O0` still references them. The number measures dead-code elimination, not a memory policy. | §3.5; `README.md:380`, `CHANGELOG.md:175`; probe A.3 |
| F5 | §10's own `delete` row is internally contradictory: the payload is "not wiped", yet "a released block never hands another value's bytes to the next user". No allocator can satisfy both without zeroing or block-granular release. This is a normative decision, not an implementation detail. | §1.1; `spec/STANDARD.md:278` |
| F6 | `compare.sh` cannot see any of this: the corpus has no `delete` or `memory.*` case, and the architecture document already says these surfaces must be kept out of parity cases. | §1.4; `docs/ARCHITECTURE.md:257-260` |

---

## 1. What the standard promises, and what each path does today

### 1.1 The promise (§10)

§10 scopes itself to the compiled artifact and makes the interpreter an emulation layer over its host
allocator (`spec/STANDARD.md:268-273`):

> QuarkLang has **no garbage collector**. Storage is released explicitly and the release is
> deterministic: nothing is collected behind your back, and no allocation ever waits for a collection.
>
> **Scope.** This section describes the contract of the compiled artifact; the reference interpreter
> implements that policy as a layer over its host runtime's allocator.

The table that follows is the normative surface (`spec/STANDARD.md:276-281`). Quoting the rows, because
their exact wording is the subject of this review. The `memory.compact()` row was corrected in 0.8.14;
the quote records the revision that was reviewed:

| Primitive | Meaning (verbatim, `spec/STANDARD.md:276-281`) |
|---|---|
| `delete x;` | release `x`'s storage: run `__delete__()` if the type declares it, then return the storage to the free pool. The payload is **not** wiped — wiping is `clear`'s job, so a released block never hands another value's bytes to the next user |
| `memory.setBlock(n)` | set the block size, the unit the runtime allocates and reuses |
| `memory.compact()` | free the blocks nobody occupies; it does not return |
| `memory.Fragmentation()` | the unused space inside live blocks, as a ratio (inspection) |

The policy paragraph adds four obligations (`spec/STANDARD.md:283-290`): allocation prefers the
least-occupied block that has room; a tide of temporary data must reuse a small stable set of blocks;
the reference implementation measured 99.96% reuse, 0.195% fragmentation and a converging block count;
and "when a task ends, the blocks it owns become reclaimable". Releasing storage something still refers
to is a programming error the runtime does not detect.

Five problems are visible in the text itself, before any code is read:

1. **The `delete` row contradicts itself.** If the payload is not wiped and storage is returned to a
   free pool, a later allocation of the same block can read the old bytes. The clause "so a released
   block never hands another value's bytes to the next user" does not follow from "not wiped"; it
   follows only from zeroing, from block-granular ownership that discards the whole block, or from
   treating the data as unobservable after release. §5 D3.
2. **`clear` is used but never defined.** It appears only inside the `delete` row; there is no
   `memory.clear()` row. §11.3 records this too: "(one of those — `clear` — is not in §10's table at
   all)" (`spec/STANDARD.md:437-438`). §5 D9.
3. **"`compact()` … does not return" is ambiguous.** It could mean "returns no value" or "never returns
   to its caller". The interpreter's own comment reads the second way ("compact() never returns at
   all") and then returns normally (`internal/lang/eval.go:2381-2384`); a probe confirms control
   continues (`memory.compact(); memory.clear(); io.println(1)` prints `1` in the interpreter — A.3).
   §5 D6. Resolved in 0.8.14: the standard row and the interpreter and type-checker comments now say
   the call returns no value and control continues.
4. **`delete` is specified for "x", not for a type.** The interpreter accepts it for any expression
   syntactically and silently does nothing for scalars; the compiler accepts only one static type.
   §5 D4.
5. **`Fragmentation()` is normative surface but has no definition of "live" or of what is counted**
   (internal waste? rounding? metadata? external fragmentation?). §5 D8.

### 1.2 The recorded gap, and where the record is itself stale

§11.3 states the divergence this review is about (`spec/STANDARD.md:425-443`). Its measured table is
quoted here because it is the baseline this document verifies:

| §10 row | Interpreter | Compiled artifact |
|---|---|---|
| `delete x;` | "any variable: runs `__delete__()` for a struct, then returns a `List`'s block to the free pool" | "**only `List<int>` variables**; any other operand is a hard error" |
| `memory.setBlock(n)` | "implemented, `1 ≤ n ≤ 2^20`" | "none: `calling method ? on setBlock is not supported by the compiler (available in the interpreter)`" |
| `memory.compact()` | "implemented" | "none: same hard error" |
| `memory.Fragmentation()` | "**not exposed**" although `MemoryManager.Fragmentation()` exists | "not exposed: the shared type checker refuses it first" |
| `memory.clear()` | "implemented" | "none: same hard error as `setBlock`" |

Three corrections to that record, all measured in this review:

- **The "24 allocation sites" is not reproducible.** At `0895b53` — the commit that recorded 0.8.3 —
  `compiler/qthreads.c.txt` had 22 `malloc` + 14 `calloc` = **36** initial allocation calls, 8
  `realloc` and 1 `free`; at `ad7a98c` it has 23 + 14 = **37**, 11 `realloc` and 1 `free`. A
  malloc-only count is 22 then and 23 now, a calloc-only count is 14 both times. No rule this review
  tried produces 24 (Appendix A.1; ⚠ the number is stale or uses an undocumented rule). It should not
  be used to size the work.
- **"The interpreter implements three of §10's four rows" needs a qualifier.** It accepts
  `setBlock`/`compact`/`clear`, and `delete` for lists and structs, but `delete scalar` is a silent
  no-op (`internal/lang/eval.go:704-730`: only the `IsStruct` and `IsList` branches act), and
  `Fragmentation()` is not reachable from either engine (`internal/lang/typecheck.go:2474-2496` has no
  case; measured `TypeError: no method "Fragmentation" on memory`, A.3). So the interpreter implements
  three *rows* only if "delete" is read as "delete a list or a struct".
- **The compiled path also accepts `delete` on a `List<int>` *parameter*, but emits nothing for it.**
  `emitDelete` returns early for a by-reference parameter because the storage belongs to the caller
  (`compiler/internal/cgen/cgen.go:2395-2397`). That is a defensible choice, but it is not §10's
  "release x's storage"; it is a no-op, and it is another shape the standard does not describe.

### 1.3 Measured behaviour today

The table below is first-hand: each row was produced by running the interpreter and compiler binaries
built from `ad7a98c` (A.3). Errors are quoted in English (`LANG=C.UTF-8`).

| Primitive / shape | Interpreter (`internal/lang`) | Compiled (`compiler/`) | Evidence |
|---|---|---|---|
| `delete` a `List<int>` variable | accepts; block requeued | accepts; frees buffer and header | `internal/lang/eval.go:717-725`; `compiler/internal/cgen/lower.go:1176-1184`, `cgen.go:2389-2414`; probe prints `3` then `9` |
| `delete` a `List<int>` **parameter** | accepts; block requeued | accepts, emits **nothing** (by-reference parameter belongs to the caller); the list is still usable afterwards | probe prints `2` in both engines; `cgen.go:2395-2397` |
| `delete` an `int` | accepts, **no-op** (prints the value) | **hard error**: `delete "x" is not supported (the compiler handles delete on a List variable only)` | probe A.3; `eval.go:704-730`; `lower.go:1180` |
| `delete` a `List<String>` / `List<float>` | accepts | hard error, same message (the gate compares the type name to `List<int>` exactly) | probe A.3; `lower.go:1179-1183`; `cgen.go:815,824,837` |
| `delete` a `pointer T` from `new T[n]` | accepts | the **construction** is refused first (`new int[n] is not supported yet`), so `delete` is not reached; the delete gate would also refuse the static type | probe A.3; `eval.go:1058-1062`; `lower.go:1179` |
| `delete` a struct declaring `__delete__` | calls `__delete__` (probe prints `0` from the method's mutation, and a panicking method aborts with `PanicError: deleted`) | hard error before dispatch: `delete "p" is not supported (the compiler handles delete on a List variable only)` | probe A.3; `eval.go:706-716`; `lower.go:1179` |
| `memory.setBlock(n)` | accepts `1 … 2^20`, changes `BlockSize` for **future** blocks | hard error: `calling method ? on setBlock is not supported by the compiler (available in the interpreter)` | `eval.go:2386-2399`; probe A.3; `lower.go:2726` |
| `memory.compact()` | accepts; returns control | hard error, same message | `eval.go:2381-2384`; probe A.3 |
| `memory.clear()` | accepts; clears free blocks | hard error, same message | `eval.go:2369-2374`; probe A.3 |
| `memory.Fragmentation()` | **refused by the type checker** | **refused by the type checker** | `TypeError: no method "Fragmentation" on memory` in both (probe A.3); `typecheck.go:2474-2496` |

The compiler-side error for every `memory.*` call is the generic "method call not lowered" fallback,
not a memory-specific diagnostic: `lower.go:2726` formats `MsgCompilerCallingMethodSOnSNot` with the
receiver type and method name (`internal/i18n/source/compiler.tsv:136`). The printed `?` is the
unresolved receiver type, which is why the message reads "calling method ? on setBlock" rather than
naming `memory`. This is a diagnostic defect in its own right, but the design consequence is the
important one: **the front end accepts the call and the lowering layer refuses it**, so a feature that
the type checker has already admitted cannot be caught earlier.

### 1.4 Why the parity gate does not see this

`compiler/testdata/compare.sh` runs both engines over `compiler/testdata/cases/*.kq` and
`cases_run/*.kq` and requires byte-identical output (`compiler/testdata/compare.sh:1-40`). At this
revision:

- no corpus case contains `delete` or `memory.` — `grep -rn 'delete \|memory\.' compiler/testdata` finds
  nothing;
- no compiler test references `delete` at all — `grep -rn delete compiler --include=*.go --include=*.kq`
  finds only the implementation (`lower.go`, `cgen.go`), not a test;
- `docs/ARCHITECTURE.md:257-260` states the architectural position explicitly: "the interpreter is the
  only engine with this manager. cgen has no `MemoryManager` … `memory.Fragmentation()` and friends are
  engine-specific surfaces that a parity case must avoid."

The standard itself warns about this: the parity gate "proves *interpreter == compiler*, not *code ==
this document*: a rule both engines implement the other way round, or a feature both engines omit,
leaves it green, so §11.3 lists what it cannot see" (`spec/STANDARD.md:26-29`). So the §1.4 "one
semantic reference" rule (`spec/STANDARD.md:24-26`) and the parity gate both hold here only because the
construct is absent from the corpus, not because the engines agree. The
§11.3 table is the only place the divergence is recorded. This matters for any design here: **a
`compare.sh` that prints `all identical` after a memory change proves nothing unless the change adds
cases that use the changed surface**, and if `Fragmentation()` prints a number, the two engines must
produce the same number or the case must be excluded (which is itself a standards problem — §5 D8).

---

## 2. What a compiled program actually allocates today

### 2.1 Pipeline and counting rule

A compiled program is Go lowering to LLVM IR plus an embedded C runtime (`compiler/embed.go:11-19`):
`cgen` emits `malloc`/`calloc`/`realloc`/`free` calls directly into the IR, and `clang` links the IR
with `compiler/qthreads.c.txt` (the runtime) and the generated message catalog (`compiler/main.go:27-40`,
`:475-480`). There are therefore **two allocation domains**, and a design that only covers one of them
is only half a memory model.

Counts use an explicit rule: one initial-allocation call site per source line, both platform branches,
`malloc` and `calloc`, excluding `realloc` (growth, not initial allocation) and `free` (release). The
rule is stated because §11.3 does not state one.

| Corpus | `malloc` | `calloc` | initial total | `realloc` | `free` | lines |
|---|---:|---:|---:|---:|---:|---:|
| `compiler/qthreads.c.txt` at `0895b53` (0.8.3 record) | 22 | 14 | **36** | 8 | 1 | 844 |
| `compiler/qthreads.c.txt` at `ad7a98c` (this review) | 23 | 14 | **37** | 11 | 1 | 870 |
| `compiler/internal/cgen/cgen.go` at `ad7a98c` (emitted per program) | 10 | 10 | **20** | 4 | 3 | — |

Totals at this revision: **57 initial allocation call sites, 15 `realloc`, 4 `free`**. The cgen side is
not in `qthreads.c.txt` at all, so a "runtime allocation count" is not a count of what a compiled
program actually allocates. Appendix A.1 gives the commands; Appendix B lists every line.

### 2.2 The runtime's allocation sites

`compiler/qthreads.c.txt` is 870 lines. Its 37 initial allocations group as follows. "Never" in the
last column means the storage is still live at process exit; only explicit release would change that,
and the runtime has one `free` (below).

| # | Category | Sites (`qthreads.c.txt:`) | What / size | Released |
|---|---|---|---|---|
| 1 | Windows channel | 13, 15 | channel struct; `cap` ints (default 1024 → 4 KiB) | never |
| 2 | Windows task | 38 | `ql_task` object (8 argument slots + state) | never |
| 3 | POSIX channel | 140, 142 | channel struct; `cap` ints (default 1024 → 4 KiB) | never |
| 4 | POSIX task | 167 | `ql_task` object | never |
| 5 | String concat | 193 | `strlen(a)+strlen(b)+1` | never |
| 6 | String `substring` / `charAt` | 329, 343 | rune bytes + 1; at most 4 + 1 | never |
| 7 | String `trim` / `upper`/`lower` | 380, 417 | `n+1`; `4*len+4` | never |
| 8 | String `replace` | 435, 446 (+451 `realloc`) | worst-case expansion; starts `len+1` and grows | never |
| 9 | `List<int>.toString` | 523 (+529, 532, 535 `realloc`) | starts 16 bytes and grows | never |
| 10 | `memorize` buffer | 551, 553, 577, 579 | 16-byte table; 64-entry bucket array (512 B); 24-byte entry; `n` ints | never |
| 11 | `long.toString` | 593 | fixed 24 bytes | never |
| 12 | `interface{}` deep copy | 655, 656, 657, 658 | 4/8-byte boxed scalar | never |
| 13 | `String.split` | 667 (+666 `realloc`), 691, 693 | one copy per part (`len+1`); fallback 8 B; 16-byte list header | never |
| 14 | `List<String>/<float>.toString` | 702, 726 (+708, 711, 714, 734, 737, 740 `realloc`) | starts 16 bytes and grows | never |
| 15 | `HashTable` | 756, 758, 772, 783, 791 | 16-byte table; 512-byte buckets; key string; 24-byte entry; grown buckets | old bucket array freed in `ql_table_grow` (`:801`); everything else never |
| 16 | `HashTable.keys` | 829, 837 | `n` pointers + 16-byte list header | never |
| 17 | `List<int>` RTTI copy | 858, 860 | `4n` buffer + 16-byte list header | never |

The **single `free`** is `compiler/qthreads.c.txt:801`, freeing the old bucket array inside
`ql_table_grow`. `realloc` at 451/529/532/535/666/708/711/714/734/737/740 frees the old buffer
implicitly. Everything else is process-lifetime.

Two details matter to the design more than the raw count:

- **The runtime allocation sites are invisible to the AST and to cgen.** They are ordinary C library
  calls inside helper functions; a block allocator that only changes what cgen emits leaves them on
  the host allocator. Whether that is acceptable is §5 D1.
- **`realloc` is part of the allocation contract.** Four cgen sites and eleven runtime sites grow a
  buffer (`cgen.go:3990,4050,4200,4257`; `qthreads.c.txt:451,529,532,535,666,708,711,714,734,737,740`).
  A block allocator must either implement `realloc` semantics (copy/move and update occupancy) or
  forbid it and rewrite the growth paths to allocate-copy-free. This is not mentioned in §11.3's
  "new allocator with a free list and an occupancy rule", but it is a required piece (§4.4 R6).

### 2.3 The allocations cgen emits (per compiled program)

These are emitted into every program that uses the construct, regardless of the runtime. Sizes are the
LLVM-level constants at the cited line; struct sizes are computed (`e.sizeOf`).

| # | Construct | Sites (`cgen.go:`) | What / size | Released |
|---|---|---|---|---|
| 1 | `List<int>` literal or uninitialised declaration | 1995, 1999; 3393, 3397 | 16-byte `%List` header; `4n` element buffer | only by an explicit `delete` on the variable (`2402`, `2405`) |
| 2 | `List<String>` literal | 3423, 3427 | 16-byte header; `8n` zeroed buffer | never (`delete` refuses the type) |
| 3 | `List<float>` literal | 3454, 3458 | 16-byte header; `8n` zeroed buffer | never |
| 4 | `ql_copyd_list` (deep copy) | 1440, 1455 | 16-byte header; `4n` buffer | only if the copy reaches a `List<int>` variable and is deleted; copies inside structs/other values leak |
| 5 | List `append` growth | 3990, 4050, 4200, 4257 (`realloc`) | 4/8-byte elements, geometric growth | old buffer freed by `realloc`; new buffer only by `delete List<int>` |
| 6 | Struct allocation / struct literal | 2190, 3627 | `e.sizeOf(base)` zeroed bytes | never (`delete` on a struct is refused) |
| 7 | `new T` reference cell / lazily allocated `T&` cell | 2210, 3627 | one `sizeOf(base)`, zeroed | never (`delete pointer T` is refused, and `new T[n]` is refused earlier) |
| 8 | `interface{}` boxing | 2920, 2927, 2936, 2943 | 4/8-byte heap cell | never |
| 9 | IR string helpers (`int`/`float`/`long`.toString) | 5352, 5365, 5395 | 12/40/20 bytes | never |

`cgen` declares the four libc functions at `cgen.go:1017-1020` and emits `free` at only three sites:
`2402` and `2405` (a `List<int>`'s buffer and header, inside `emitDelete`) and `2413` (a generic
`free` of a non-list variable that the lowerer never reaches — see 2.4). List layouts are
`%List = { i32*, i32, i32 }`, `%ListS = { i8**, i32, i32 }`, `%ListF = { i64*, i32, i32 }`
(`cgen.go:815,824,837`), i.e. a pointer plus two cursors; there is no size or owner word in the
object, which is exactly what a block allocator would need at `free`.

### 2.4 `delete`'s lowering, in full

The lowerer's gate is three lines (`compiler/internal/cgen/lower.go:1176-1184`):

```go
case *lang.DeleteStmt:
    id, ok := st.X.(*lang.Ident)
    if !ok { /* non-variable expression: hard error */ }
    if t, _ := fc.lookup(id.Name); t != "List<int>" {
        return nil, l.errfID(st.Pos, i18n.MsgCompilerDeleteQNotSupportedCompilerHandles, id.Name)
    }
    return &deleteStmt{name: id.Name}, nil
```

`emitDelete` (`cgen.go:2389-2414`) then emits two `free`s for a `List<int>` (buffer, header), returns
early for a by-reference parameter, and has a trailing generic `free` branch that is **dead code for
every type the lowerer lets through**, because the gate has already restricted the type to
`List<int>`. The parameter case is a silent exception worth stating on its own: the probe
`fn f(List<int> l) void { delete l; }` compiles and prints `2` afterwards in both engines (A.3), i.e.
the compiled `delete` is a no-op on a parameter — not a hard error, and not a release. There is no
`__delete__` call anywhere in the compiled path: a struct cannot even reach `emitDelete`, and the
`impl` body is compiled only when user code calls it directly.

This is not a "narrowed block policy"; it is a hand-written pair of frees for one static type. It does
not know the allocation's size, owner, block or generation, so it cannot be extended to another type
without either (a) passing the type's size/layout down to `emitDelete`, or (b) adding an object header
that `free` can read. The current `List` layout has no room for a header inside the 16 bytes, and the
buffer pointer is handed to user code as `i32*`, so a header would have to sit *before* the pointer
(§4.4 R5).

### 2.5 Lifetimes, and what this means for §10

The compiled program's de facto lifetime model is "allocate, use, optionally delete a `List<int>`, exit".
There is no scope cleanup, no escaping analysis-based free, and no runtime reclamation other than the
one hash-table growth `free`. Consequently:

- §10's "nothing is collected behind your back" is true; §10's "storage is released explicitly" is true
  for one shape only. Every `String`, `List<String>`, `List<float>`, struct, `new[]`, boxed interface,
  hash entry, split list, memo entry and task/channel leaks until the process exits. Calling that a
  memory *model* would be generous; it is the host allocator's lifetime model with one explicit
  `free` pair.
- The interpreter's `MemoryManager` is **not** an oracle for the compiled runtime. It does not allocate
  bytes; it tracks logical block IDs over Go slices (`eval.go:226-360`). Its `delete` zeroes a block's
  occupancy, and a block can hold several allocations: `Alloc` adds `size` to an existing block's
  `Used` if it fits and returns that block's ID (`eval.go:244-270`), while `Delete(id)` sets the whole
  block's `Used = 0` (`eval.go:333-342`). In the interpreter, two small `new T[n]` allocations can therefore share a
  block; deleting one zeroes the occupancy of both. The interpreter does not corrupt the other list's
  Go slice, because the slice is independent of the accounting, but the *accounting* is unsound for
  shared blocks. A compiled allocator that copied this policy literally would free both allocations
  (§5 D2/D3).
- The interpreter's `delete` on a list with `blockID == 0` allocates a brand-new block just to delete it
  (`eval.go:717-724`). That inflates `AllocCalls`/`NewBlocks`, and it shows the model is accounting
  rather than storage management. §10's 99.96% reuse / 0.195% fragmentation come from that accounting
  layer (benchmarks at `internal/lang/bench_test.go:84-145`), not from the compiled runtime.

---

## 3. Design options for a block allocator behind the compiled runtime

### 3.0 Requirements the options are judged against

Derived from §10 and the constraints above; each is cited to its source.

| # | Requirement | Source |
|---|---|---|
| R1 | Per-object release: `delete x` returns `x`'s storage without invalidating other objects | `spec/STANDARD.md:278` |
| R2 | `__delete__` dispatch before release | `spec/STANDARD.md:278` |
| R3 | A block size settable at run time; a block is the allocation/reuse unit | `spec/STANDARD.md:279` |
| R4 | Least-occupied block with room is preferred | `spec/STANDARD.md:283-284` |
| R5 | `Fragmentation()` reports unused space inside live blocks | `spec/STANDARD.md:281` |
| R6 | `compact()` frees unoccupied blocks | `spec/STANDARD.md:280` |
| R7 | Payload survives release; the next user does not receive another value's bytes | `spec/STANDARD.md:278` (contradictory as written; §5 D3) |
| R8 | A task's blocks become reclaimable when it ends | `spec/STANDARD.md:290` |
| R9 | Both engines agree byte for byte on the corpus; a construct the compiler cannot lower is a hard error, and the gate proves engine equality rather than document equality | `spec/STANDARD.md:24-29` |
| R10 | No new standard surface during the freeze; changing §10 is an owner decision | `spec/STANDARD.md:471-484` |
| C1 | Runtime allocations (C) and generated allocations (IR) are two domains; covering one is not covering §10 | §2.1 |
| C2 | `realloc` is used on hot growth paths; the allocator must support it or the growth paths must change | §2.2 |
| C3 | Windows does not link `qthreads.c.txt` today (`runtimeSrc` returns `""` on Windows) | `compiler/main.go:27-30` |

### 3.1 Option A — flat free list per size class

**Mechanics.** Round each request to a size class (e.g. 16, 32, 64 …); keep one singly-linked free
list per class; `alloc` pops, `free` pushes; large requests go to the host. This is the simplest
allocator and the closest to what a libc `malloc` fast path does.

**Language-visible consequences.**

- `delete` is naturally per-object and cheap (R1 ✔). It can preserve the payload (no wiping) until the
  slot is reused (R7 partially), but a LIFO free list hands the same bytes to the next same-class
  allocation, which is exactly the case §10's "never hands another value's bytes" sentence forbids.
- `Fragmentation()` has **no block concept to report on**. If it is defined as rounding waste
  (`class_size - requested` over live allocations), it is a different quantity from the interpreter's
  block metric and from §10's "unused space inside live blocks". If the implementation invents arenas
  to have blocks, it has become Option C with a hidden block layer (R3/R5 ⚠).
- `setBlock(n)` has no natural meaning: the allocation unit is a slot, not a block. It could set the
  size of the host chunk backing the free lists, but no language-visible operation would reveal that
  except indirectly through `compact`/`Fragmentation`; the standard calls it "the unit the runtime
  allocates and reuses" (R3 ✘).
- `compact()` can return empty host chunks, but with no per-block occupancy there is nothing to decide
  "nobody occupies" per block (R6 ⚠).
- Reuse/tide: converges by class after warm-up; "block count" is not meaningful. The standard's
  "least-occupied block" is not honoured (R4 ✘).

**Verdict.** The best *allocator* and the worst *§10 implementation*: it cannot honestly expose
`setBlock`/`Fragmentation`/`compact` as specified without recreating blocks. It is a useful fast path
inside a counted block design, not a complete option.

### 3.2 Option B — one sorted free list (address-ordered extents)

**Mechanics.** Keep free extents sorted by address; allocation searches for a fit (first-fit or
best-fit), splits the extent, and `free` inserts and coalesces with neighbours. Blocks are extents
carved from a heap region; a header per extent records size and block.

**Language-visible consequences.**

- `delete` is per-object (R1 ✔); coalescing keeps the free pool usable; data survives (R7 partial, same
  reuse question).
- `Fragmentation()` can report free bytes inside live extents, but §10's definition is *internal*
  fragmentation (space inside live blocks), while the interesting failure of a sorted free list is
  *external* fragmentation (free space between live blocks). The standard's metric would look good
  while the heap grows — a metric that cannot see the main risk (R5 ⚠).
- `setBlock(n)` can be the arena/extent size; it is meaningful (R3 ✔), but "least-occupied block" needs
  a separate per-block occupancy index on top of the sorted list (R4 ⚠).
- `compact()` can free wholly empty extents; it cannot defragment live ones without moving (and moving
  would invalidate language pointers, which are raw addresses; §4 of the standard has no moving GC)
  (R6 ⚠).
- Tide: a same-size tide reuses extents after the first wave; a mixed-size tide fragments and the
  search cost grows with the number of extents. Convergence to a constant block count is not
  guaranteed without a compaction that moves, which the language forbids.
- `delete`'s promise is the easiest to satisfy exactly (data preserved, per-object), and its accounting
  can mirror the interpreter's exact sizes — the closest parity of the three (R9 ⚠).

**Verdict.** The most faithful to a "free pool" and exact-size accounting, and the most expensive:
O(number of extents) search/coalesce, pointer stability means no compaction, and the standard's metric
does not measure its main risk.

### 3.3 Option C — slabs (fixed-size slots per size class, block = slab)

**Mechanics.** A block is a slab of `setBlock(n)` bytes dedicated to one size class. It has a header
(size, class, used count, next) and a free list of slots. `alloc` takes the least-occupied slab of the
class that has a free slot, pops a slot; `free` pushes the slot back and returns a fully empty slab to
the class's empty list (not to the host until `compact`). Large objects (larger than the largest class,
or larger than a slab) get a dedicated block with one object and a tail.

**Language-visible consequences.**

- `delete` is per-object (R1 ✔); `__delete__` runs before the slot is returned (R2 ✔ if lowered).
- `setBlock(n)` is exactly "the unit the runtime allocates and reuses": it sets the slab size for
  blocks created afterwards; existing slabs keep their size (which matches the interpreter, where
  `BlockSize` is read when a new `MemBlock` is created, `eval.go:265`) (R3 ✔).
- `Fragmentation()` is exactly §10's wording: unused slots + tail inside slabs with `Used > 0`,
  divided by the bytes of those slabs. It includes size-class rounding, which the interpreter's
  exact-size accounting does not; the number is meaningful but **not equal to the interpreter's**
  unless both round sizes the same way (R5 ✔/R9 ⚠).
- `compact()` frees slabs nobody occupies (`Used == 0`) — literally the standard's sentence (R6 ✔).
- "Least-occupied block with room" is honoured *within a size class*; blocks are class-partitioned.
  That is a reading of §10, not the only reading: the interpreter lets any size use any block
  (`eval.go:244-270`). If the owner requires the literal any-size reading, this becomes Option B/C
  hybrid (§5 D2).
- Tide: same-class temporaries reuse the same slots; the slab count converges to the class high-water
  mark; fragmentation is bounded by one slot per live slab (R4 ✔, tide ✔).
- Loops: one free-list pop/push per allocation; no growth after the high-water mark; rounding can
  increase resident memory (R1/R9 ⚠). A loop that allocates a *new size* each iteration slowly adds
  classes/slabs; a loop that retains objects still grows (no GC).
- The payload question is unchanged: a freed slot keeps its bytes and the next same-class allocation
  sees them unless the allocator zeroes on alloc/free (R7 decision, §5 D3).

**Verdict.** The only option that makes `setBlock`, `compact`, `Fragmentation` and per-object `delete`
all mean what §10 says, while keeping the implementation small enough to review. Its price is a
class-partitioned interpretation of "block", rounding waste in the metric, and a large-object path.

### 3.4 Comparison

| Dimension | A: flat size-class free list | B: one sorted free list | C: counted slabs |
|---|---|---|---|
| Allocation cost | O(1) | O(free extents) / O(log n) | O(1) free-list pop; partial-slab lookup O(1)–O(log n) |
| Per-object `delete` (R1) | ✔ | ✔ | ✔ |
| `__delete__` dispatch (R2) | possible | possible | possible |
| `setBlock` is a real block (R3) | ✘ (slot, not block) | ✔ (extent) | ✔ (slab) |
| "Least-occupied block" (R4) | ✘ | ⚠ (needs a second index) | ✔ within class |
| `Fragmentation()` matches §10's wording (R5) | ✘ (rounding only) | ⚠ (internal vs external) | ✔ (live slabs) |
| `compact()` frees unoccupied blocks (R6) | ⚠ (host chunks) | ⚠ (only whole extents) | ✔ |
| Payload/reuse safety (R7) | ⚠ LIFO reuse | ⚠ reuse | ⚠ reuse; zeroing decision |
| Task ownership (R8) | ⚠ per-thread lists | ⚠ | ⚠ per-thread slab lists |
| Parity with interpreter (R9) | ✘ metric differs | ⚠ closest | ⚠ rounding differs |
| Growth paths (`realloc`, C2) | needs `realloc` emulation | needs split/coalesce | needs `realloc` (copy to new class) |
| Implementation size | small | large | moderate |
| Windows (C3) | platform-neutral possible | same | same |

### 3.5 The tide benchmark: what it measures now, and what it must measure next

The current situation is worse than "the compiled benchmark is missing". It is **measuring the wrong
thing**:

- `README.md:380` advertises "Tide, 1e8 rounds: QuarkLang (compiled) 1 ms, C 1 ms, Rust 29 ms, Go 92 ms".
  `CHANGELOG.md:175` records the same result and names the cause: "matching C (same LLVM optimizer,
  identical allocation elimination)". `CHANGELOG.md:232` records the interpreter's 99.96% reuse /
  0.195% fragmentation / 2 blocks, and `bench_test.go:84-145` contains the only checked-in benchmarks:
  they call `NewMemoryManager` directly from Go, not a compiled program and not the language.
- There is **no checked-in tide source**: `grep -rin tide` finds only README/CHANGELOG/spec prose and
  two comments; `bench/Makefile` has fib targets only. The 1 ms row cannot be reproduced from this
  checkout.
- A probe that approximates it — a compiled `while` loop doing `List<int> a = [1,2,3]; delete a;` for
  1e8 rounds — exits in **0.00–0.01 s** at the default `-O3`. The optimized object has **no undefined
  `malloc`/`calloc`/`free` symbols**; the same IR at `-O0` does (`nm -u`). The allocations were deleted
  by LLVM, not reused by a policy (A.3).
- A real allocate/free loop costs something. A C control with the same shape and a `volatile` sink
  (so it cannot be optimized away) takes **0.42–0.50 s** for 1e8 `malloc(12)+free` on the same machine
  (A.3). That is the order of magnitude any honest allocator starts from.

So the existing tide benchmark, to keep measuring the same *program*, must change what it *observes*:

1. Keep the tide program's shape and output (allocate, use, delete, repeat) and make it a checked-in
   corpus/benchmark case, so it cannot be an ad-hoc result.
2. Force the allocation to survive optimization: a sink the optimizer cannot see through (a language
   value printed through an opaque runtime call, a global/volatile-equivalent, or a separately
   compiled sink), and a control run without the sink to report the dead-code floor separately.
3. Record the optimizer flags and prove the allocation was not eliminated (no `malloc`/`calloc`/`free`
   undefined symbols for the allocator's entry points, or a counter that proves the calls happened).
4. Report allocator counters (reuse rate, new blocks, block count convergence, fragmentation) and RSS,
   not wall time alone. The interpreter's `AllocCalls`/`ReusedCount`/`NewBlocks` counters
   (`eval.go:234-236`) are the model; the compiled runtime has none today.
5. Compare against a C control that also forces allocation (the current C 1 ms is the same DCE floor,
   not a malloc baseline). Keep the old 1 ms as a historical "DCE floor" row, clearly labelled.
6. Do not compare `Fragmentation()` across engines unless the accounting is made identical (§5 D8).

The 1 ms number should not be used as evidence for or against any option here. The options differ in
what the allocator does when it is *not* optimized away; the current benchmark cannot see that
difference at all.

### 3.6 What breaks for programs that allocate in loops

- **Today, such loops can be free.** If the compiler can see that a loop's allocation never escapes and
  is deleted every iteration, LLVM removes the malloc/free pair (A.3). Any out-of-line allocator
  (`ql_alloc`/`ql_free`) is opaque to LLVM, so the pair stays. A loop that was 1 ms can become hundreds
  of milliseconds. That is the single biggest performance risk in this design (§4.4 R1).
- **Steady state is achievable.** For a loop that allocates and deletes the same shape, Options A and C
  reuse the same slot after the first iteration; Option B reuses a coalesced extent. None of them grows
  unboundedly, and none of them pauses. The cost is the per-iteration free-list operation, not growth.
- **Retaining allocations still grows.** No option adds a collector. A loop that appends without
  `delete` leaks in every option, exactly as it does today; `compact()` only helps after blocks are
  empty. A design that is presented as fixing leaks would be wrong.
- **Mixed sizes fragment differently.** A: per-class free lists never coalesce across classes, so a
  program alternating 16 B and 4 KiB allocations keeps both classes' high-water marks. B: coalescing
  keeps the pool whole but search cost grows. C: one slab set per class; the large-object path has tail
  waste. This is where `Fragmentation()` becomes visible and where the interpreter's metric can diverge.
- **`append` growth uses `realloc`.** In a loop that appends, the growth path is on the hot path
  (`cgen.go:3990,4050,4200,4257`). A block allocator that does not implement `realloc` must rewrite
  those to allocate-copy-free, which changes both performance and peak memory.
- **Threads multiply the problem.** The runtime has an 8-worker pool (`qthreads.c.txt:62-73`), and
  §10 says a task's blocks become reclaimable when it ends. A global free list needs a lock on every
  allocation (contention); per-thread lists need a registry and a cross-thread `delete` policy. The
  current compiled path does not even link the runtime on Windows (`compiler/main.go:27-30`), so a
  loop in a task on Windows is a separate open question (§5 D5).

---

## 4. Recommendation, cost, tests, risks

### 4.1 Recommendation: Option C, counted slabs, in two stages

**Recommend Option C.** It is the smallest design that makes all four §10 primitives mean what the
standard says, and the only one whose `Fragmentation()` is the quantity §10 actually names ("unused
space inside live blocks"). It also keeps `delete` per-object and cheap. Its class-partitioned reading
of "block" must be blessed by the owner (§5 D2); if the owner insists on any-size blocks, the fallback
is a hybrid of C's block accounting with B's per-block extents, not Option A.

**Shape** (all names are proposals, not existing API):

```
ql_block (slab) header: size | class | used | owner | next
  └─ slots: [free-list next] [payload ...]   (header per slot: size/class/block)
global: partial[class] (least-occupied first; min-heap or sorted list)
        empty blocks (returned by delete; released by compact)
        large blocks (one object, tail waste)
alloc(size, owner):
  class = class_of(size)
  if partial[class] has a block with a free slot: pop slot; used++
  else: new block(setBlock bytes, class); pop slot; push to partial
free(ptr):
  slot = header(ptr); block = slot.block
  push slot; used--
  if used == 0: move block from partial[class] to empty
compact():
  for every empty block: return to host; recompute stats
Fragmentation():
  sum over blocks with used > 0 of (block.size - used_payload_bytes) / sum(block.size)
```

**Entry points** (the "three runtime entry points" of `spec/STANDARD.md:442` plus the allocation API):

| Proposed symbol | Role |
|---|---|
| `ql_mem_set_block(int n)` | set the slab size for blocks created afterwards; validate `1 … 2^20` (match `eval.go:2390-2396`) |
| `ql_mem_compact(void)` | free empty slabs; return no value (D6 resolved in 0.8.14) |
| `ql_mem_fragmentation(void)` | return the ratio; pending D8 |
| `ql_alloc(size_t, owner)` / `ql_free(void*)` / `ql_realloc(void*, size_t)` | allocation API; the runtime's own C allocations can be macro-routed to these, and cgen emits calls to them instead of libc |
| `ql_mem_stats(&reuse, &new_blocks, &block_count)` | benchmark/test counters; optional, not language surface |

**Lowering changes** (all in `compiler/internal/cgen`):

- replace the 20 initial-allocation sites and the 4 `realloc` sites with calls to the runtime helpers;
  centralise them in one `emitAlloc`/`emitFree` helper so the audit does not have to be repeated;
- replace `emitDelete`'s two `free`s with `ql_free` and add `__delete__` dispatch before the release;
  the lowerer's type gate expands from `List<int>` to the decided domain (§5 D4);
- lower `memory.setBlock/compact/clear/Fragmentation` to the runtime entry points; the type checker
  must expose `Fragmentation` (`internal/lang/typecheck.go:2474-2496`) and the interpreter must expose
  it too (`internal/lang/eval.go:2364-2399`) or `compare.sh` will still see one engine refuse;
- route the runtime's 37 initial allocations (and 11 `realloc`s) through the same helpers **only if
  §5 D1 says the whole compiled artifact is in scope**; otherwise document the remaining host
  allocations in §10's scope sentence, which would itself be a standard change.

**Stage the work:**

| Stage | Content | Why it is separable |
|---|---|---|
| 1 | Allocator in the runtime + cgen routing + tests/benchmark; no language surface change | proves the allocator and the DCE/performance question without touching the standard |
| 2 | `delete` for the decided domain + `__delete__` dispatch + `memory.*` lowering + `Fragmentation` exposure | the language-visible part; needs the §5 decisions |
| 3 | Runtime-wide allocation routing and task ownership/reclamation | §5 D1/D5; can ship later without changing stage 1–2 semantics |

### 4.2 Cost

Line counts are **ESTIMATE**s based on the measured site counts; they are not code. The measured inputs
are the 57 initial allocation sites, 15 `realloc`s, 4 `free`s, 20 cgen emission sites, and the
`delete`/`memory` lowering sites cited in §2.

| File | Change | Estimated lines | Confidence |
|---|---|---|---|
| `compiler/qthreads.c.txt` | allocator (~250–350), headers/entry points (~60–100), stats (~40), C-site routing (~50 if D1 = all) | **+400–550** | ⚠ |
| `compiler/internal/cgen/cgen.go` | 20 alloc + 4 realloc + 3 free sites → helpers; declarations; `delete`/`__delete__` dispatch | **120–180 changed** | ⚠ |
| `compiler/internal/cgen/lower.go` | delete gate/domain; `memory.*` lowering | **80–130** | ⚠ |
| `internal/lang/typecheck.go` | `Fragmentation` case; delete type rules | **20–40** | ⚠ |
| `internal/lang/eval.go` | expose `Fragmentation`; align `delete` domain (if D4 changes it) | **15–40** | ⚠ |
| `internal/i18n/source/*.tsv` | 3–5 messages (generated files follow) | **~5 rows** | ⚠ |
| `compiler/internal/cgen/*_test.go` | IR-shape tests: no raw malloc/free, `delete` dispatch, memory lowering | **150–250** | ⚠ |
| `internal/lang/*_test.go` + allocator unit tests | allocator/accounting/compact/fragmentation | **200–350** | ⚠ |
| `compiler/testdata/cases(_run)/*` | 6–10 parity cases (delete matrix, memory calls, no Fragmentation print) | **80–150** | ⚠ |
| `cmd/qkstress` / stress corpus | the delete/realloc-under-load case | **50–100** | ⚠ |
| `bench/` + benchmark case | forced-allocation tide, C control, report | **60–120** | ⚠ |
| **Total** | | **≈ 1,200–1,800 lines, 9–11 files, 2–3 PRs** | ⚠ |

§11.3 called this "large — one design review of its own" (`spec/STANDARD.md:443`); the table above is
that review's size estimate. It is larger than the 24-site count suggests because §11.3 counted the
runtime sites and not the cgen emission sites, `realloc`, parity cases or the benchmark. It is not a
one-sitting change.

### 4.3 Tests that would prove it

| # | Test | What it proves | Shape |
|---|---|---|---|
| T1 | Allocator unit tests (Go test driving the C runtime, or a C test harness) | alloc/free/reuse per class; occupancy; `setBlock` only affects new slabs; `compact` frees empty slabs; `Fragmentation` formula; zero-size; oversized; alignment | table-driven; no compiler |
| T2 | cgen IR-shape tests | every allocation site emits `ql_alloc`/`ql_realloc`, not `@malloc`/`@calloc`; `delete` emits `__delete__` then `ql_free`; no raw `@free` remains except the runtime's own | extend `compiler/internal/cgen/feature_test.go` style |
| T3 | Parity corpus cases | both engines byte-identical on the decided `delete` domain and on `memory.setBlock/compact/clear`; `Fragmentation` either prints the same value in both or is excluded by an explicit decision | add to `compiler/testdata/cases(_run)`; run `compare.sh` |
| T4 | **Stress: delete and reallocate under load** | no crash, no double free, bounded RSS, output parity, and the allocator is actually exercised (not DCE'd) | see below |
| T5 | Tide benchmark with escape + counters | the allocator, not LLVM, is measured; reuse/new-blocks/block-count/fragmentation reported; the DCE floor is reported separately | checked-in benchmark |
| T6 | Platform gate | Linux/macOS/Windows CI; the allocator is compiled on all three (today `runtimeSrc` is empty on Windows, `compiler/main.go:27-30`) | CI matrix |

**T4 in more detail.** A compiled program that deletes and reallocates under load, with a sink that
prevents the optimizer from removing the work:

```quark
// design sketch, not committed code; n and the print cadence are test parameters
fn churn(IOStream io, int n) void {
    int i = 0;
    int acc = 0;
    while (i < n) {
        List<int> a = [i % 7, i % 11, i % 13];
        a.append(i % 17);
        // An opaque read: toString() must see the buffer, so the allocation cannot be
        // scalarized away for the iterations that reach this branch.
        if (i % 1000 == 0) { String s = a.toString(); io.println(s, acc); }
        acc = acc + a.get(0) + a.get(a.size() - 1);
        delete a;
        i = i + 1;
    }
    io.println(acc);
}
```

The test runs this for a few million iterations at several sizes, alternating delete/keep, and checks:
exit 0; stdout identical in both engines; peak RSS under a bound (Linux only; the existing stress
runner already treats RSS as unavailable on macOS/Windows, `.github/workflows/ci.yml:45-56`); no
ASan/UBSan report when built with a sanitizer; the binary/IR still references the allocator entry
points (to prove the work was not optimized out). A second variant with a global sink/volatile-style
escape is needed because the `toString` branch is only taken once per 1000 iterations and an optimizer
may still hoist or split the loop. The test must fail if the allocator is bypassed (e.g. if a raw
`@malloc` remains on the delete path). This is the case that would have caught every one of F3–F5.

### 4.4 Risks, especially "slower than the current malloc/calloc path"

| # | Risk | Why / evidence | Mitigation or decision |
|---|---|---|---|
| R1 | **Opaque allocator calls lose LLVM's malloc/free elimination** | Measured: the tide loop is 0.00–0.01 s at `-O3` and its object has no `malloc`/`calloc`/`free` undefined symbols; at `-O0` the same IR does (A.3). A C control with a `volatile` sink takes 0.42–0.50 s/1e8. Any out-of-line `ql_alloc`/`ql_free` is opaque to LLVM, so allocation-heavy loops can go from "optimized away" to "real work". | Accept and document; make the benchmark report both the DCE floor and the real path; consider inlining the fast path into the IR (header pop/push) if the measured regression is unacceptable. **This is a deliberate trade: §10 asks for a memory policy, and a policy that the optimizer erases is not observable.** |
| R2 | Simple allocator is slower than glibc for ordinary allocation | glibc's tcache is a size-class free list with years of tuning; a first-cut slab allocator with a global lock can be 1.5–3× slower per op. ⚠ ESTIMATE; measure per class. | Fast path: per-class free list, no lock when single-threaded; benchmark against glibc; do not add stats atomics on the fast path. |
| R3 | Global lock contention in taskm | 8 workers (`qthreads.c.txt:62-73`); a global free list serializes them. | Per-thread arenas + owner tag; or accept single-threaded stage 1 and hard-error the surface in tasks (a divergence) until stage 3. §5 D5. |
| R4 | Memory blow-up from retained slabs/free lists | `delete` returns storage to the pool, not to the host; `compact` is the only release; the interpreter's own model keeps blocks until `compact` (`eval.go:293-305`). | Document that RSS is a high-water mark; offer an explicit `compact`; bound per-class caches; stress test RSS. |
| R5 | Headers change layout/ABI | List buffers are exposed as `i32*` (`cgen.go:815`); structs can be passed to FFI/library calls; a header before the object changes the pointer the language sees. | Put headers before the payload and return the payload pointer; reserve alignment; test FFI/library layouts; never add headers inside language-visible struct fields. |
| R6 | `realloc` growth on hot paths | 4 cgen + 11 runtime sites (§2.2); a slot cannot grow in place if the class changes. | Implement `ql_realloc` as alloc-copy-free for class changes and in-place for same-class growth; or rewrite append to allocate-copy-free. Measure `list.append` (the repo's own 1M-append benchmark, `bench/qk/list.qk`). |
| R7 | `Fragmentation()` parity/two-engine divergence | The interpreter's accounting is exact-size logical blocks; C's slabs round to classes and have headers. Printing a float from both engines will differ. | §5 D8: either make the accounting identical in both engines (including rounding/header rules) or declare the metric implementation-defined and keep it out of parity cases — which changes the standard's "one semantic reference" claim. |
| R8 | Windows does not link the runtime | `runtimeSrc` returns `""` on Windows (`compiler/main.go:27-30`), and the Windows branch of `qthreads.c.txt` is therefore not compiled at all. A runtime-only allocator is absent on Windows. | Put the allocator in a platform-neutral unit compiled on all three (IR helpers or a second embedded C file), or enable the Windows runtime as its own change. CI must exercise T4/T5 on Windows. |
| R9 | `__delete__` double-free / ownership | User `__delete__` may delete fields; the outer release then frees the struct. | Define ownership in §5 D3/D4; test a struct with a list field and `__delete__`; document that deleting a member and the whole is an error (the runtime does not detect it, per §10). |
| R10 | OOM/double-free behavior is undefined | Current `malloc` results are unchecked; `emitDelete` can be called twice (`cgen.go:2389-2414` has no generation check). | §5 D11: panic/abort/undefined; add a header magic/generation only if the owner wants detection; do not promise detection §10 does not promise. |
| R11 | This is a standard change, not just a compiler fix | §10 is normative and under freeze; §11.3 calls this "large — one design review of its own"; §12 puts changes behind the owner's ruling (`spec/STANDARD.md:471-484`). | Stage 1 can land as implementation without changing surface; stages 2–3 need the owner's decisions. |

**The one-sentence version of R1:** the current compiled path is fast on allocation loops because it
does not allocate; a conforming block allocator will be slower there, and the project should say so
before the benchmark is rerun.

---

## 5. Decisions the owner must make before code is written

None of these is resolved by this document. Each changes the standard, the language surface, or the
acceptance tests; picking one silently is how the next §11.3 entry gets written. The "why" column points
to the section that establishes the conflict.

| # | Decision | Options | Why it cannot be settled here | Consequence if wrong |
|---|---|---|---|---|
| D1 | **Allocator scope**: which allocations are in §10's contract? | (a) language-visible allocations only (`new`, List, structs, boxing, delete); (b) all compiled allocations including runtime strings/hash/tasks; (c) staged (a) then (b) | §10 says "the contract of the compiled artifact"; the runtime's 37 sites are part of the artifact. (a) is cheaper but leaves most allocations on malloc and makes `Fragmentation` a partial metric. | (a) makes §10 true for a subset while §11.3 can still point at the rest; (b) doubles the surface and touches every runtime helper; (c) is honest only if the scope sentence says so. |
| D2 | **What is a block?** | (a) slab per size class (Option C); (b) any-size arena with per-object extents (B/C hybrid); (c) flat size-class free list (A) | The standard says "least-occupied block that has room" but never says whether blocks are class-partitioned; the interpreter lets any size use any block (`eval.go:244-270`). | (c) cannot implement `setBlock`/`Fragmentation` honestly; (a) is a normative reinterpretation; (b) is the literal reading at a higher cost. |
| D3 | **`delete` granularity and payload safety** | (a) per-object release, payload preserved, next user may see old bytes; (b) per-object release, zero on allocation (or on free); (c) block-granular release (interpreter's accounting) | §10's `delete` row is self-contradictory (§1.1 F5); (a) violates "never hands another value's bytes", (b) costs a wipe, (c) is unsound when a block is shared (`eval.go:244-270,333-342`). | The chosen answer determines whether `delete` is a security boundary and whether `Fragmentation`/`compact` can be block-based. |
| D4 | **`delete`'s domain** | (a) only heap-owned values (List/struct/`new[]`/String?); (b) any variable, scalars are no-ops (interpreter today); (c) any variable, scalars are hard errors (compiler-ish); (d) one rule per type | The standard says `delete x` for any `x`; the two engines disagree on scalars, `List<String>`, `List<float>`, `new[]`, and structs (`§1.3`). | A test matrix cannot be written, and `compare.sh` cannot cover the surface. |
| D5 | **Threads/tasks and ownership** | (a) single-threaded only in stage 1, hard-error in tasks; (b) global allocator with a lock; (c) per-thread arenas with owner tags and task-end reclaim; (d) main-thread only, task allocations unchanged | §10 says a task's blocks become reclaimable at task end, but compiled taskm is POSIX-only (`main.go:29-30`) and the interpreter's `ReclaimTask` is scheduling-sensitive (`internal/lang/eval_test.go:348-380`). | (a) is a divergence; (b) may serialize the 8-worker pool; (c) is the faithful design but needs cross-thread `delete` rules; (d) leaves §10's task clause unimplemented on the compiled path. |
| D6 | **`compact()` returns or not** | (a) returns no value and control continues (interpreter's actual behaviour, probe A.3); (b) never returns (terminates) — a literal reading of "it does not return" | The standard's sentence is ambiguous; the interpreter comment says "never returns at all" but the code and the type checker return. Resolved in 0.8.14: the row and both comments now say the call returns no value and control continues | A terminating `compact` is a language-visible control-flow primitive; a void-returning one is a call. The compiler lowering differs completely. |
| D7 | **`setBlock(n)` semantics** | (a) affects blocks created afterwards only (interpreter, `eval.go:265`); (b) resizes existing blocks; (c) only a hint | The standard says "set the block size" without saying whether existing blocks change; the interpreter's `MemBlock.Size` is fixed at creation; `1 ≤ n ≤ 2^20` (`eval.go:2390-2396`) is an implementation limit, not §10 | Resizing existing blocks can invalidate live pointers; (a) must be written down or users will expect (b). Also define `n` smaller than a slot/header. |
| D8 | **`Fragmentation()` exposure and definition** | (a) expose in both engines, exact same accounting; (b) expose, implementation-defined, keep out of parity cases; (c) keep it inspection-only, not language surface | The standard lists it as a primitive; neither engine exposes it today; the interpreter's accounting is exact-size logical blocks, a compiled allocator's is class/header-based. `docs/ARCHITECTURE.md:257-260` already says parity cases must avoid it. | (a) forces the compiled allocator to emulate the interpreter's accounting (including its block-sharing bug) and constrains optimization; (b) contradicts "one semantic reference"; (c) is a standard change. |
| D9 | **`clear()`** | (a) add it to §10 with a definition; (b) remove it (the interpreter keeps a non-standard method); (c) keep it as an interpreter-only extension | It is used in the `delete` row but has no row of its own; the interpreter wipes free blocks only (`eval.go:344-356`), the compiler refuses it | The compiled path cannot be "complete" against an undefined primitive; the interpreter will keep an engine-specific surface. |
| D10 | **Optimization contract** | (a) allocation/delete may be elided when unobservable; (b) every allocation is observable (so `Fragmentation`/counters constrain optimization); (c) observable only when the program calls the inspection methods | The current 1 ms tide is DCE (A.3); `Fragmentation()` is "inspection" but a program can print it; `compare.sh` is byte-exact | (a) lets the optimizer keep performance but makes the memory model unobservable and the benchmark meaningless; (b) is a large performance regression; (c) needs an escape analysis the compiler does not have today. |
| D11 | **Failure and misuse** | (a) OOM/double-free/use-after-free are undefined (as §10 says the runtime does not detect release-while-referenced); (b) panic/abort with a message; (c) debug-only magic/generation checks | §10 explicitly says the runtime does not detect releasing referenced storage; adding detection is a new guarantee; unchecked `malloc` can segfault today | (a) is the cheapest and matches §10's "programming error"; (b) is a language-visible change; (c) adds header space and cost; tests need one answer. |

---

## Appendix A — measurements and commands

### A.1 Counts

```sh
git show 0895b53:compiler/qthreads.c.txt > "$HOME/.cache/q-0895b53.c"   # the recording commit
grep -cE '\bmalloc\s*\('  "$HOME/.cache/q-0895b53.c"   # 22
grep -cE '\bcalloc\s*\('  "$HOME/.cache/q-0895b53.c"   # 14
grep -cE '\brealloc\s*\(' "$HOME/.cache/q-0895b53.c"   # 8
grep -cE '\bfree\s*\('    "$HOME/.cache/q-0895b53.c"   # 1

grep -cE '\bmalloc\s*\('  compiler/qthreads.c.txt   # 23
grep -cE '\bcalloc\s*\('  compiler/qthreads.c.txt   # 14
grep -cE '\brealloc\s*\(' compiler/qthreads.c.txt   # 11
grep -cE '\bfree\s*\('    compiler/qthreads.c.txt   # 1

grep -nE '@(malloc|calloc|realloc|free)' compiler/internal/cgen/cgen.go
# 20 initial (10 malloc + 10 calloc), 4 realloc, 3 free, 4 declarations
```

`realloc` lines in the runtime: 451, 529, 532, 535, 666, 708, 711, 714, 734, 737, 740. The single
`free` is line 801.

### A.2 Interpreter accounting numbers

```sh
go test ./internal/lang -run '^$' \
  -bench 'BenchmarkMemManagerChurn|BenchmarkFragmentationAfterChurn' \
  -benchtime=200000x -count=1
```

Measured on this machine (2026-10-17): `reuse-rate 1.000`, `new-blocks 2`, `fragmentation 0.001953`
(i.e. 0.1953%, matching §10's 0.195%). These are direct `MemoryManager` calls
(`internal/lang/bench_test.go:84-145`), not a compiled program; `ReportMetric` prints three significant
digits, so 99.96% appears as `1.000` here.

### A.3 Probes

All probes were built from `ad7a98c`; the interpreter was `go build -o quark .`, the compiler
`(cd compiler && go build -o qkc .)`. Errors below are English (`LANG=C.UTF-8`).

| Probe | Interpreter | Compiled |
|---|---|---|
| `delete int x` after `int x = 1;` | prints `1` (no-op) | `delete "x" is not supported (the compiler handles delete on a List variable only)` |
| `List<int> l = [1,2,3]; delete l;` | prints `3` then `9` | prints `3` then `9` |
| `List<int> l` passed to `fn f(List<int> l) void { delete l; }` | prints `2` (no-op on the parameter) | prints `2` (no-op; `emitDelete` returns for the by-reference parameter) |
| `List<String> s = ["a","b"]; delete s;` | prints `1` (accepted) | `delete "s" is not supported (the compiler handles delete on a List variable only)` |
| `List<float> f = [1.5,2.5]; delete f;` | prints `1` (accepted) | `delete "f" is not supported (the compiler handles delete on a List variable only)` |
| `pointer int p = new int[4]; delete p;` | prints `1` (accepted) | the construction is refused first: `new int[n] is not supported yet (the interpreter yields List<T>&, which cannot be assigned to List<T>; use a [..] literal)` |
| `memory.setBlock(64); memory.compact(); memory.clear(); io.println(1);` | prints `1` (all accepted; control continues after `compact`) | `calling method ? on setBlock is not supported by the compiler (available in the interpreter)` |
| `io.println(memory.Fragmentation());` | `TypeError: no method "Fragmentation" on memory` | `TypeError: no method "Fragmentation" on memory` (the shared type checker refuses first) |
| struct with `__delete__` that sets a field to 0, then `delete p` | prints `0` (the method ran and mutated the struct) | `delete "p" is not supported (the compiler handles delete on a List variable only)` |
| struct with `__delete__` that panics, then `delete p` | `PanicError: deleted` (the method ran) | `panic` alone reports `unknown function "panic"` before the delete is lowered |

Tide probe (1e8 rounds, `List<int> a = [1,2,3]; delete a;` in a `while`):

```sh
qkc tide_loop.kq > tide.ll
clang -O3 -c tide.ll -o tide-O3.o && nm -u tide-O3.o | grep -E 'malloc|calloc|free'   # none
clang -O0 -c tide.ll -o tide-O0.o && nm -u tide-O0.o | grep -E 'malloc|calloc|free'   # U calloc, U free, U malloc
qkc -run tide_loop.kq          # 0.00-0.01 s wall at the default -O3
```

The `-O0` binary itself does not link in this revision: `ql_int_to_str` and `ql_float_to_str` are
undefined when the whole runtime is compiled without optimization, while the optimized link succeeds.
That is an unrelated runtime defect; the symbol evidence above is from the object files, which is enough
to show the calls are present at `-O0` and absent at `-O3`. C control:

```c
void* volatile sink;
int main(void){ for (long i=0;i<100000000L;i++){ void* p=malloc(12); sink=p; free(p);} return 0; }
```

`clang -O2`, three runs: 0.42 s, 0.46 s, 0.50 s.

### A.4 Gate results

Run in a linked worktree on `ad7a98c` plus this document, 2026-10-17 (A.5 has the commands):

| Gate | Result |
|---|---|
| root `go build ./... && go test ./... -count=1`, `LANG=C.UTF-8` | pass |
| compiler `go build ./... && go test ./... -count=1`, `LANG=C.UTF-8` | pass |
| root + compiler, `LANG=zh_CN.UTF-8` | pass |
| `bash compiler/testdata/compare.sh` | `all identical (interpreter == compiler)` |
| `gofmt -l .` | empty |
| `go generate ./internal/i18n` + `go run ./cmd/qki18n -check` + `git diff --exit-code` | clean |
| stress subset (`qkstress run`, `-timeout 90s`, `-skip path_fn_10k_params\|res_few_mb`, `-known stress/known-failures.txt`) | exit 0 (28 cases: AGREE 14, REJECT_AGREE 11, REJECT_DIFF 3) |

### A.5 Gate commands

```sh
# root and compiler, both locales
LANG=C.UTF-8 go build ./... && LANG=C.UTF-8 go test ./... -count=1
(cd compiler && LANG=C.UTF-8 go build ./... && LANG=C.UTF-8 go test ./... -count=1)
LANG=zh_CN.UTF-8 go build ./... && LANG=zh_CN.UTF-8 go test ./... -count=1
(cd compiler && LANG=zh_CN.UTF-8 go build ./... && LANG=zh_CN.UTF-8 go test ./... -count=1)

# parity
bash compiler/testdata/compare.sh

# formatting and generated catalog
test -z "$(gofmt -l .)"
go generate ./internal/i18n
git diff --exit-code -- internal/i18n/msg_gen.go compiler/internal/rtcat/msg_c.h
go run ./cmd/qki18n -check

# stress subset (from .github/workflows/ci.yml:58-105)
mkdir -p stress-out/bin
go build -o stress-out/bin/quark .
(cd compiler && go build -o ../stress-out/bin/qkc .)
go run ./cmd/qkstress gen -out stress-out/cases
go run ./cmd/qkstress run -cases stress-out/cases \
  -interp stress-out/bin/quark -compiler stress-out/bin/qkc -work stress-out \
  -timeout 90s -skip 'path_fn_10k_params|res_few_mb' -known stress/known-failures.txt
```

### A.6 What remains unverified

- **✘ Runtime cost of a candidate allocator.** No allocator was implemented; the only cost data are
  the DCE measurement and the glibc control. The 1.5–3× estimate in §4.4 R2 is an estimate.
- **✘ `realloc` behavior of a block allocator.** The design says "implement or rewrite"; neither was
  built, and the growth paths were not measured under a block allocator.
- **✘ Windows.** The compiled runtime is not linked on Windows today (`compiler/main.go:27-30`), so a
  runtime-only allocator would not exist there; this was not exercised. The Windows branch of
  `qthreads.c.txt` is present but not compiled by `runtimeSrc`.
- **✘ Thread/task behavior under a block allocator.** `ReclaimTask` and per-task ownership exist only as
  interpreter accounting; the compiled taskm path was not combined with an allocator in this review.
- **✘ OOM, double free, use-after-free.** The current code does not check `malloc` results or detect
  misuse; the standard says release-while-referenced is undetected. A final design needs an answer
  (§5 D11), but this review did not test failure paths.
- **✘ RSS.** No long-running compiled allocation stress was run under a block allocator (there is no
  allocator yet). The RSS bound in T4 is a test design, not a measurement.
- **✘ `Fragmentation()` equality across engines.** Not possible to test before the surface exists; the
  analysis in §3.3/§5 D8 says the numbers will differ unless accounting is aligned.
