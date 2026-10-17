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
| D5 | **One global allocator lock in stage 1.** Allocations are thread-safe, but the eight-worker task pool serializes on the allocator fast path. Per-object `owner` is recorded, not enforced. | Correctness before ownership. Per-thread arenas and task-end reclaim are stage 3. | Per-thread arenas, cross-thread `delete`, task-end reclaim (D5). |
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

`ql_free` reads the 32-byte object header immediately before the payload. Payloads from the host
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
| `int ql_mem_set_block(int n)` | 1 for an accepted future slab size (`1 … 2^20`), 0 otherwise; existing slabs unchanged. |
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

To be filled by the stage-1 run: the checked-in workload is `bench/memory/alloc_bench.c` (host vs counted
slabs, with a `volatile` sink), run through `scripts/bench-mem.sh`; the end-to-end program shape is
`bench/qk/list.qk`, and the churn parity case is `compiler/testdata/cases_run/z_delete_churn.kq`.

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
