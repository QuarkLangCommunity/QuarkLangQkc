# Compiled memory, stage 2: the §10 surface on the counted slabs

**Status.** Implementation note for [`compiled-memory-design.md`](compiled-memory-design.md) and the
continuation of [`compiled-memory-stage1.md`](compiled-memory-stage1.md). It records which of the
design's open decisions stage 2 had to take, what each surface now does in the compiled path, what
the accounting covers and does not, and where the two engines still differ. Stage 2 changes no
`spec/STANDARD.md` primitive: it wires rows the standard already names (`memory.setBlock`,
`memory.compact`, `memory.clear` — the derived wipe — and the `delete` matrix), routes the runtime's
own allocations (D1), and gives a task's blocks the reclaim-on-end behaviour §10 states (D5). The
interspersed measurement-wording revision is `spec/STANDARD.md` 0.8.13.

The governing rule for every surface was: open it either end to end or as a positioned hard error
that names what is supported. Where the standard and the compiled path disagree about *semantics*
rather than support, the reference interpreter's behaviour wins; where the interpreter has no answer,
the gap is recorded with its size instead of invented.

## 1. Boundary

| In stage 2 | Not in stage 2 |
|---|---|
| `memory.setBlock(n)` lowered to `ql_mem_set_block` with the interpreter's `1 … 2^20` range and the interpreter's error | Exposing `memory.Fragmentation()` in either engine (D8 stays internal) |
| `memory.compact()` lowered to `ql_mem_compact`, with the interpreter's answer to D6 (it returns) | A per-thread arena allocator or a lock-free fast path beyond stage 1's pool-started lock |
| `memory.clear()` lowered to a new `ql_mem_clear` (release + zero-fill) | `memory.mode`, which §10 does not name and both engines refuse |
| The delete matrix: every lowered list element type, an owned struct with `__delete__` dispatch, a `copyd` copy | Releasing a by-reference parameter's storage, a scalar/String (no heap storage), or a non-variable expression |
| D1: the runtime's own C allocations routed through `ql_alloc`/`ql_calloc`/`ql_realloc`/`ql_free` | Routing what is not an allocation: static data, the C stack, the allocator's host block storage |
| D5: per-task block ownership and reclaim at task end, proven by C harness + a task parity case | Cross-thread `delete` rules, or detection of a pointer that escaped a finished task |
| A parity or refusal case per row of the matrix, plus allocator-harness tests | Windows: the runtime is still not linked there (unchanged) |

## 2. Decisions stage 2 had to take

Stage 1 recorded D1 as staged, D2 Option C, D3(a), D5's global lock, D8 internal, D10(b) and D11
undefined. Stage 2 takes the next reading of the ones it touches; the others stand.

| # | Reading implemented in stage 2 | Why this reading |
|---|---|---|
| D1 | **(c) completed: every allocation in the compiled artifact is routed.** Channel and task objects, string builders, split lists, hash tables and their bucket arrays, memo entries and interface boxes all call the counted-slab API. The only libc allocation left is the allocator's host block storage (`ql_mem_host_alloc`/`ql_mem_host_free`), which is the pair the metric measures the inside of. | §10 scopes itself to the compiled artifact; leaving the runtime on malloc would make `Fragmentation()` a partial metric. The audit test `TestRuntimeAllocationsStayRouted` fails if a helper reaches for libc again, and the C harness reads `ql_mem_alloc_calls`/`ql_mem_used_bytes` around a routed string. |
| D4 | **The matrix, not one type.** An owned variable releases: `List<int>`, `List<String>`, `List<float>` (header + buffer), and a struct (its cell, after `__delete__()`), whether locally declared or a `copyd` parameter's callee-owned copy. A by-reference parameter, a scalar, a `String`, and a non-variable expression keep positioned hard errors naming the widened domain. | The interpreter accepts any variable; the compiled path can honestly release exactly the storage it allocated. A scalar or String has no heap storage to release, so a no-op would be silent acceptance; a by-reference parameter's storage belongs to the caller, so releasing it would free someone else's cell. Each remaining row has a refusal test in both shipped languages. |
| D5 | **(c) block ownership, task-end release.** A block (slab, or large block) records the owner that created it. While a task runs, the worker sets a thread-local owner and `ql_alloc(size, 0)` inside it belongs to that task; main-thread storage stays owner 0 and is never shared with a task's slabs. When the task ends, the worker releases every block with its owner *before* publishing `done`, so `ql_block` never observes a finished task whose storage is still counted. | §10 says "when a task ends, the blocks it owns become reclaimable". The interpreter's `ReclaimTask` zeroes the block's occupancy at the same point; the compiled allocator's block is the slab, so the slab is the ownership unit. An object that escapes a finished task dangles, exactly as the interpreter's accounted-for value survives a block it no longer owns — §10 calls releasing referenced storage a programming error the runtime does not detect. |
| D6 | **(a) it returns.** Read from the interpreter, not chosen: `eval.go` calls `MemoryManager.Compact()` and returns normally; only its comment says "never returns at all". The compiled path emits `ql_mem_compact` and falls through, and the parity case prints before and after two calls. | A terminating lowering would be a control-flow primitive the standard does not describe. |
| D8 | **Stays internal.** `ql_mem_fragmentation()` and `ql_mem_occupancy()` exist for tests and the benchmark; no engine exposes `Fragmentation`, and the shared checker keeps refusing it. | Cross-engine equality is still an owner decision; printing two different numbers would be worse than the current honest gap. |
| D10 | **(b) unchanged.** Every routed allocation is an opaque call; the optimizer cannot elide the memory policy. D1 extends the same rule to the runtime's helpers. | The tide record is a measurement of the policy, not of dead-code elimination. |
| D11 | **Unchanged: undefined in release, asserted under `QL_MEM_DEBUG`.** A foreign pointer, a double release, use after release and a pointer escaping a finished task are undefined. | §10 says the runtime does not detect releasing referenced storage; adding detection is a new guarantee. |

## 3. Surface by surface

### 3.1 `memory.setBlock(n)`

The interpreter validates at run time and accepts `1 ≤ n ≤ 2^20`; the compiled lowering calls
`ql_mem_set_block`, branches to a positioned runtime error carrying the interpreter's message for a
value outside the range, and the error is catchable by `try` exactly like the interpreter's
`RunError`. The size applies to slabs created afterwards, matching the interpreter reading
`BlockSize` when a `MemBlock` is created (D7(a)). Windows keeps a positioned refusal that names the
missing runtime.

Parity: `compiler/testdata/cases/z_memory_setblock.kq` (accepted values, byte-identical), and
`cases_run/z_memory_setblock_range.kq` (the `0` case: message, position and exit code compared).
Tests: `TestMemorySetBlockLowersToRuntimeEntryPoint`, `TestMemorySetBlockErrorJoinsTry`,
`TestCompiledMemorySurfaceRefusesBothLocales` (the refusal surface in English and Chinese).

### 3.2 `memory.compact()`

`ql_mem_compact` releases every slab with no live object and never moves a live one. D6 is answered
above: control continues. Parity: `cases/z_memory_compact.kq`.

### 3.3 `memory.clear()`, the derived wipe

`ql_mem_clear` is the pair §10 names: every slab nobody occupies returns to the host, and every slot
that is still free in a retained slab has its payload zeroed. A live object's bytes are untouched —
clear is not delete. The interpreter's clear drops the emptied logical blocks; because its blocks are
Go values rather than storage, it has no released bytes to zero, a gap of one semantic half in a layer
that has no bytes (recorded, not invented).

Tests: the allocator harness's `clear_wipes_released_region` (release keeps the payload, clear zeroes
it, a live payload survives, the empty slab is released), the interpreter's
`TestMemoryManagerClearDropsEmptyBlocks`, and the parity case `cases/z_memory_clear.kq`.

### 3.4 The delete matrix

| Form | Interpreter | Compiled after stage 2 | Pinned by |
|---|---|---|---|
| owned local `List<int>` | releases the block | buffer + header through `ql_free` | `TestDeleteOnOwnedListIntStillLowers`, `z_delete_local.kq` |
| owned local `List<String>` / `List<float>` | releases the block | same release path (header + buffer) | `TestDeleteOnListStringAndFloatLowers`, `z_delete_matrix.kq` |
| `copyd List<int>` parameter | releases the callee's copy | same | `TestDeleteOnCopydListIntParameterStillLowers` |
| owned local struct, with or without `__delete__` | runs `__delete__()`, no storage accounting | runs `__delete__()` (ordinary self-method call), then releases the cell | `TestDeleteOnStructRunsDeleteMethodThenReleases`, `TestDeleteOnStructWithoutDeleteMethodReleases`, `z_delete_matrix.kq` |
| `copyd` struct parameter | same, on the copy | same | `TestDeleteOnCopydStructParameterLowers` |
| by-reference parameter | releases the caller's block | **positioned hard error** (storage belongs to the caller) | `TestCompiledMemorySurfaceRefusesBothLocales` (list and struct), `cases_run/z_delete_churn.kq` for the accepted churn |
| scalar / `String` local | silent no-op | **positioned hard error** (no heap storage to release) | `TestCompiledMemorySurfaceRefusesBothLocales` |
| field / call / `new[]` expression | accepts (releases a block or does nothing) | **positioned hard error** (the compiler releases owned variables only) | `TestCompiledMemorySurfaceRefusesBothLocales` |
| `pointer T` from `new T[n]` | accepts as a no-op | the construction is refused before `delete` is reached | existing `new` refusal tests |

`z_delete_matrix.kq` is the parity case. `__delete__`'s effect is observed through a field that
aliases another struct (not through released storage), and a `copyd` copy is released without touching
the caller's value. The interpreter test `TestDeleteRunsStructDeleteMethod` keeps pinning its half.

### 3.5 D1: the runtime's own allocations

All 37 historical runtime allocation sites and the 11 reallocation sites now go through the counted
slabs; so do the channel/task objects and the boxed `interface{}` copies. The audit test
`TestRuntimeAllocationsStayRouted` scans `compiler/qthreads.c.txt`: only the two host-block lines may
call libc. The C harness proves a routed string's allocation is counted and released.

**What the accounting covers now:** every byte the compiled artifact allocates from the heap, except
the allocator's own host block storage — the block header and the aligned host chunk are what the
metric measures the inside of, so counting them inside themselves would be circular. Static data and
the C stack are not allocations.

**ABI consequence, recorded:** a `String` returned by `toString`/`split`/`replace` and the other
runtime builders is a counted-slab pointer, not a libc one. A linked C library that receives such a
string must release it with `ql_free`; the tide sink did call libc `free` and aborted with
`invalid size`, and now calls `ql_free`. QL code is unaffected (it never hands a string to libc free).

### 3.6 D5: task ownership

The runtime worker marks the task, executes the compiled function, releases every block owned by the
task, and only then publishes `done`. The C harness has two proofs: a direct owner test (a task block
is released, the main thread's block survives, both slab and large paths) and an integration test
(the real pool runs a worker that allocates without releasing, and `ql_block` observes the allocator
back at its pre-merge used count). `cases_run/z_task_reclaim.kq` runs a task that allocates/deletes,
ends, a second task, and the main thread — byte-identical in both engines.

## 4. What still differs, and its size

| Gap | Size / consequence |
|---|---|
| `delete` on a by-reference parameter | One form; the compiled path cannot release caller storage. Positioned error, both languages. |
| `delete` scalar / `String` / non-variable expression | Three forms the interpreter accepts as no-ops or by dereferencing an expression; the compiled path has no owned storage to release. Positioned errors. |
| `memory.Fragmentation()` | Unexposed in both engines; the compiled accessor is internal (D8). One metric definition decision. |
| Windows | The compiled runtime is not linked, so no allocator, no `memory.*` lowering, no routing, no task reclaim; `delete` keeps the libc path. Unchanged statement. |
| Escaping values / aliasing | A pointer that escapes a finished task or an aliased `delete` is undefined, as §10 says. No detection. |
| A task's cross-owner free | `ql_free` of a main-thread object from a task is memory-safe but the object returns to the main slab; the ownership model describes blocks, and reclaim is per owner. Documented, not detected. |

## 5. Measurements

The D1 routing moved the tide observable row: its `toString` string is now a counted-slab allocation
instead of a host one. The record was re-recorded through `bench/tide/run.sh --record` (the generated
block is in `README.md`, §10 and `docs/tide-benchmark.md`) and the standard's measurement wording
moved to 0.8.13; the rows and their spreads are the record's, not a claim of a fast machine. The
allocator's own overhead was measured in stage 1 and is unchanged; routing the runtime's small
allocations adds the same per-object cost to strings and hash entries that stage 1 measured for
language allocations (1.4–2.1× libc for small sizes), which is the price of a metric that covers the
whole artifact.

## 6. Tests added or extended

| Gate | What it pins |
|---|---|
| `compiler/internal/cgen/memory_surface_test.go` | lowering of the wired rows, the setBlock range/error/try control flow, the clear call |
| `compiler/internal/cgen/delete_surface_test.go` | every enabled form and every remaining refusal, in English and Chinese |
| `compiler/runtime_routing_test.go` | D1: no runtime helper reaches libc, and the routed shapes stay |
| `compiler/testdata/memory/qthreads_mem_test.c` | clear wipe, routed runtime string, task ownership, task-end reclaim, plus the stage-1 stress |
| `internal/lang/memory_surface_test.go` | interpreter clear/compact/setBlock and the fragmentation refusal |
| `compiler/testdata/cases*/z_memory_*.kq`, `z_delete_matrix.kq`, `z_task_reclaim.kq` | parity/refusal corpus rows |
| `spec/status/claims.tsv` 11.1-10, 11.3-08..12 | the status report's claims and evidence |
