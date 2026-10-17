# Compiled memory, stage 1: counted slabs

**Status.** Implementation note for [`compiled-memory-design.md`](compiled-memory-design.md), not a change to
`spec/STANDARD.md`. Stage 1 of the design's three-stage plan is the allocator and the code-generator
routing; it does not change the language surface.

The design document remains the review that names the open decisions. This note records which of those
decisions stage 1 had to take, which it deliberately left open, and what is therefore true of the built
artifact after the change.

## 1. Boundary

| In stage 1 | Not in stage 1 |
|---|---|
| A counted-slab allocator in `compiler/qthreads.c.txt` with a size class per object, a free list, a per-object release, an occupancy counter, large-object blocks, `compact` and a fragmentation accessor | Lowering `memory.setBlock` / `memory.compact` / `memory.clear` / `memory.Fragmentation` |
| Routing the allocations the code generator emits through the allocator (`ql_alloc` / `ql_calloc` / `ql_realloc` / `ql_free`) | Routing the runtime's own C allocations (`ql_strcat`, hash tables, split, memo, tasks) through it |
| Releasing an owned `List<int>` (a local, or a `copyd` parameter's callee-owned copy) through `ql_free` | Widening `delete` to other types, or `__delete__` dispatch in the compiled path |
| Compiler IR-shape tests, an allocator assertion/stress test, a parity churn case, and a checked-in benchmark | `memory.*` language surface, cross-engine `Fragmentation` equality, task ownership/reclaim |
| A build-time Windows fallback to libc (the runtime is still not linked there) | Enabling the compiled runtime on Windows |

The lowerer keeps exactly the `delete` domain 0.8.10 pinned: an owned local `List<int>` variable, or a
`copyd List<int>` parameter's callee-owned copy. A by-reference parameter, a struct, a scalar, a
`List<String>` and a `List<float>` keep their positioned hard errors. No accepted form emits nothing.

## 2. Decisions stage 1 had to take

The design leaves eleven decisions to the owner. Stage 1 cannot be written without taking a reading of
some of them; this table says which reading the code implements and which decisions stay open.

| # | Reading implemented in stage 1 | Why this reading | Still open |
|---|---|---|---|
| D1 | **Staged (c), language allocations first.** The code generator's 20 initial allocation sites and 4 growth sites go through the allocator. The runtime's own C allocations stay on the host allocator, except `ql_any_copy_list`, which can return a `List<int>` that `delete` releases and is therefore routed too. | The design's stage 1 is "allocator in the runtime + cgen routing"; routing every C helper is stage 3. `ql_any_copy_list` is the one runtime allocation that can cross into a deletable language value. | Runtime-wide routing (D1 b/c) and the scope sentence in §10. |
| D2 | **Option C, class-partitioned slabs.** A block is one slab of `setBlock(n)` bytes dedicated to one size class. | This is the design's recommendation and the only option that makes all four §10 rows meaningful. | The owner's blessing of the class-partitioned reading of "block" (D2 a vs b). |
| D3 | **(a) per-object release, payload preserved, no zero on `ql_alloc`.** `ql_calloc` zeroes because callers asked for `calloc`; `ql_free` does not wipe. A later same-class allocation may therefore see the previous payload. | The standard's `delete` row says the payload is **not** wiped, and `memory.clear()` is named as the wiping operation. Stage 1 must not silently turn `delete` into a wiping operation. | The contradiction in §10's "so a released block never hands another value's bytes to the next user" (F5) is still a normative decision. If the owner chooses "never hand over old bytes", the allocator needs a zero-on-allocate or a generation/clear rule; that is a stage 2 change. |
| D5 | **One global allocator lock, engaged when the task pool starts.** Before the first task pool exists the process cannot allocate from two threads, so the fast path is lock-free (the design's R2 mitigation). Once `qpool_ensure` runs, every allocation takes the global lock. Per-object `owner` is recorded, not enforced. | Correctness before ownership, without paying for an uncontended atomic on every single-threaded allocation. The pool store happens before any worker exists or any task is queued. | Per-thread arenas, cross-thread `delete`, task-end reclaim (D5). |
| D8 | **`ql_mem_fragmentation()` implements the design's §3.3 formula** (unused bytes inside live slabs and live large blocks, over those blocks' bytes) and is **not exposed to the language**. `ql_mem_occupancy()` is its complement. | The design's formula is the only definition that matches §10's "unused space inside live blocks". Exposing it or claiming cross-engine equality is a decision stage 1 does not take. | Exposure and cross-engine equality (D8 a/b/c). |
| D10 | **(b) every routed allocation is observable.** The compiler emits an opaque `ql_alloc`/`ql_free` call for every routed allocation and never elides one. D10(a) is rejected for the routed surface. | The standard makes `delete`, `compact` and `Fragmentation` observable; an allocation that the optimizer removes is not part of the memory policy the standard describes. The benchmark must be able to see the allocator (T5). | Whether the owner wants (a) for some shapes, or (c) with an escape analysis. Stage 1 takes the conservative reading and accepts the measured cost (R1). |
| D11 | **OOM and misuse are undefined at language level; assertions exist only under `QL_MEM_DEBUG`.** Release builds do not promise double-free/use-after-free detection. | §10 says releasing referenced storage is a programming error the runtime does not detect; adding a language-visible panic or generation check is a new guarantee. | The failure policy (D11 a/b/c). |

The allocation API keeps the design's owner parameter even though stage 1 does not act on it:
`ql_alloc(size, owner)`, `ql_calloc(count, size, owner)`, `ql_realloc(ptr, size)`, `ql_free(ptr)`.

## 3. Allocator shape

```
ql_slab  (a block: one size class)
  header: magic | class | slot_count | used | used_bytes | size | free_head
          next_partial | all_prev | all_next | raw
  slots : [object header 32 B][payload class_size] ...
  free list: per-slot `next_free` in the object header (payload is never overwritten)

object header (32 B)
  magic | kind (slab/large) | flags | requested | block | next_free
  payload starts at header + 32, so every payload is 16-byte aligned

large object
  [large header][object header][rounded payload]   (one object per block, tail waste counted)

state
  partial[class] : slabs with at least one free slot, scanned for the least-occupied slab
  all slabs      : for compact/check
  large live list: for check/fragmentation
  counters       : alloc_calls, free_calls, reuse, new_blocks, live_bytes, used_bytes
```

Size classes are 16, 32, 64, 128, 256, 512 and 1024 payload bytes. A request above 1024 bytes, or one
that cannot fit a slab of the current block size, goes to the large path. The default block size is 4096
bytes, matching the interpreter's `globalMemory.BlockSize`; `ql_mem_set_block(n)` accepts `1 … 2^20` and
affects slabs created afterwards. When `n` is smaller than the minimum a class needs, the created slab is
enlarged to hold one slot; that limit is recorded here because D7 (what a too-small `n` means) is still
open.

`ql_realloc` keeps the requested size in the header but may keep a larger capacity: on the large path it doubles the capacity when it moves, which is ordinary realloc slack and stops the list append loop from copying the whole buffer on every element once the tail passes half the capacity (design R6). `ql_free` reads the 32-byte object header immediately before the payload. Payloads from the host
allocator are not accepted: every code-generator allocation is routed, and `ql_any_copy_list` is the only
runtime C allocation routed. A release build treats a foreign pointer as undefined (D11); the debug build
asserts.

## 4. Entry points

| Symbol | Contract |
|---|---|
| `void* ql_alloc(size_t size, int owner)` | One object; payload preserved from previous use, 16-byte aligned; `size == 0` behaves like `size == 1`; returns NULL on host OOM. |
| `void* ql_calloc(size_t count, size_t size, int owner)` | Zeroes `count * size` bytes; overflow returns NULL. |
| `void* ql_realloc(void* ptr, size_t size)` | Same class (or same large capacity) keeps the pointer; otherwise allocate/copy/min/free. `ptr == NULL` allocates; `size == 0` releases and returns NULL. |
| `void ql_free(void* ptr)` | Per-object release back to the class slab; `NULL` is a no-op; does not wipe the payload. |
| `int ql_mem_set_block(int n)` | 1 for an accepted future slab size (`1 … 2^20`), 0 otherwise; existing slabs unchanged. (`memory.setBlock` still refuses; the language lowering is stage 2.) |
| `void ql_mem_compact(void)` | Releases every slab with `used == 0` to the host; never moves a live object. |
| `double ql_mem_fragmentation(void)` | `(live_bytes - used_bytes) / live_bytes`, or 0 when nothing is live. |
| `double ql_mem_occupancy(void)` | `used_bytes / live_bytes`, or 0 when nothing is live. |
| `long long ql_mem_block_count(void)` | Cached slabs plus live large blocks. |
| `long long ql_mem_live_bytes(void)` / `ql_mem_used_bytes(void)` | Sum of live block bytes / sum of requested bytes. |
| `long long ql_mem_alloc_calls(void)` / `ql_mem_free_calls(void)` | Allocation and release calls (a realloc that moves counts one of each). |
| `void ql_mem_stats(long long* reuse, long long* new_blocks, long long* block_count)` | Counter accessor named by the design; NULL outputs are ignored. |
| `int ql_mem_check(void)` | Walks every slab/large block and the free lists; 0 when the invariants hold, non-zero otherwise. |

`QL_MEM_DEBUG=1` adds assertions on the object/slab magic, occupancy bounds, free-list membership,
alignment, double release and counter consistency. The ordinary compiled runtime is built without it.

## 5. Windows

`compiler/main.go` still returns no runtime source on Windows: the compiled runtime is not linked there.
The allocator lives in that runtime, so stage 1's allocator is absent on Windows. The code generator keeps
its libc path on Windows (`malloc` / `calloc` / `realloc` / `free`), which is the behaviour Windows had
before this change: compile and link keep working, `delete` on an owned `List<int>` still releases host
storage, and none of the occupancy/fragmentation accounting exists. The allocator assertion test skips on
Windows for the same reason.

This is not a claim that stage 1 works on Windows. Making it work means either enabling the runtime there
(the design's R8/T6) or moving the allocator to a platform-neutral translation unit compiled on all three;
both are stage 3 work. The Windows libc path is deliberately not a silent no-op: `delete` still releases,
and every unwired `memory.*` surface still hard-errors.

## 6. Optimisation contract

Stage 1 implements D10(b) for the routed surface: every allocation emitted by the code generator is an
opaque call to the runtime allocator, and the compiler does not elide it. LLVM can no longer delete a
`malloc`/`free` pair that the language cannot observe, because there is no longer a malloc/free pair to
delete. This is the deliberate trade in R1: a memory policy that the optimizer erases is not observable,
and the design's own tide measurement was a measurement of dead-code elimination.

The cost is measured, not assumed: see §7 and the checked-in benchmark under `bench/memory/`. The DCE
floor is reported separately from the real allocator path.

## 7. Measurements

Measured on 2026-10-17 on Linux 7.2.10 (x86-64), clang 22.1.8, with the checked-in workload
`bench/memory/alloc_bench.c` (a `volatile` sink stores every returned pointer and the payload is
touched, so the optimizer cannot delete the allocation). Reproduce with:

```sh
scripts/bench-mem.sh            # n=15 runs x 2,000,000 iterations per cell, ns/op
```

Before = the pre-stage-1 libc path; after = `ql_alloc`/`ql_calloc` + `ql_free`. The values are medians
with the sample standard deviation across the 15 runs (full min/max in the script output).

| op | bytes | libc median (sd) | counted-slab median (sd) | ratio |
|---|---:|---:|---:|---:|
| malloc | 16 | 4.15 (0.90) | 7.41 (0.95) | 1.79x |
| malloc | 64 | 4.26 (0.59) | 5.89 (1.48) | 1.38x |
| malloc | 512 | 4.00 (0.80) | 7.03 (1.38) | 1.76x |
| malloc | 4096 | 12.82 (0.72) | 18.41 (1.29) | 1.44x |
| malloc | 8192 | 12.41 (1.42) | 18.00 (1.07) | 1.45x |
| calloc | 16 | 4.49 (0.68) | 9.36 (0.73) | 2.08x |
| calloc | 64 | 4.64 (0.74) | 8.78 (1.00) | 1.89x |
| calloc | 512 | 6.40 (0.52) | 10.70 (0.97) | 1.67x |
| calloc | 4096 | 32.21 (3.37) | 33.64 (2.46) | 1.04x |
| calloc | 8192 | 45.88 (1.52) | 48.63 (2.20) | 1.06x |

**R2 is real.** For ordinary small allocations the counted slabs are **1.4–2.1× slower than glibc's
tcache**, the range the design predicted (1.5–3×). The overhead is the per-object header, the free-list
walk through the object header (the payload must not be overwritten, so the link cannot live in it), the
class lookup, and the atomic fast path when the task pool is enabled. Large blocks and zeroed 4–8 KiB
allocations are close to libc (1.04–1.45×). This is the price of an allocator the program can observe;
stage 2/3 can inline the fast path or add per-thread arenas if the owner wants it back.

The end-to-end shapes use two checked-in programs:

| program | before (origin/main `0fc0808`) | after | ratio |
|---|---:|---:|---:|
| `bench/memory/tide_dce.kq` (allocate/delete, nothing observed) | 0.55 ms (sd 0.07, n=15) | 24.36 ms (sd 1.30) | 44x |
| `bench/memory/tide_loop.kq` (allocate, append, read, toString every 1000th) | 31.66 ms (sd 8.34, n=15; min 15.57, max 31.95) | 63.85 ms (sd 0.15) | 2.02x (median) |
| `bench/qk/list.qk` (1,000,000 appends) | 7.56 ms (sd 4.08) | 7.63 ms (sd 3.94) | 1.01x |

The first row is the design's R1 in one line: the old binary was fast because LLVM erased the allocation
(the DCE floor), not because the allocator was good. The second row is the same program with an
observable use; the after path is not just slower, it is **stable**, which is what a real allocator looks
like. The append shape does not regress: `ql_realloc`'s large-path slack avoids the O(n²) copy an
exactly-sized realloc would have caused once the list tail passed half its capacity.

The parity churn case `compiler/testdata/cases_run/z_delete_churn.kq` exercises the same delete/reallocate
path under `compare.sh`; its output is byte-identical in both engines.

## 8. What still hard-errors

| Surface | Status after stage 1 |
|---|---|
| `memory.setBlock(n)` | Hard error in lowering, naming the stage-1 runtime entry point and the one supported memory operation (`delete` on an owned `List<int>`), not the generic "available in the interpreter" fallback. |
| `memory.compact()` | Same. |
| `memory.clear()` | Same. |
| `memory.Fragmentation()` | Refused by the shared type checker; the compiled runtime has the accessor, no engine exposes it yet. |
| `delete` on a by-reference parameter, a struct, a scalar, `List<String>`, `List<float>` | Unchanged positioned hard errors; the supported domain is named in the message. |
| `memory.mode(...)` | Refused; not part of §10. |

## 9. What stage 2 needs

1. The owner's ruling on D2 (class-partitioned block), D3 (payload/reuse safety), D4 (delete domain),
   D8 (Fragmentation exposure/definition) and D10 (optimisation contract).
2. Lowering `memory.*` to the entry points above (including the D6 return reading and D7 `setBlock`
   semantics), adding `Fragmentation` to the shared type checker and to both engines, or deciding not to
   expose it.
3. `__delete__` dispatch before release in the compiled path and a delete matrix (List/struct/new[]/String)
   that both engines can run byte-for-byte.
4. A task-ownership answer (D5) before routing task allocations, and a runtime-wide routing decision (D1)
   before claiming §10 covers the whole artifact.
5. A Windows decision: enable the runtime or split the allocator into a platform-neutral unit (R8).
