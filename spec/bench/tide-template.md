# Tide benchmark

The tide workload allocates a three-element list, uses it and releases it in a loop. The earlier
README row `Tide, 1e8 rounds = 1 ms` had no checked-in source, and a design review showed that the
1 ms was LLVM dead-code elimination: at `-O3` the allocation was removed, while the same IR at `-O0`
still referenced `malloc`/`calloc`/`free`. This report and the harness behind it replace the claim
with a measurement the optimizer cannot erase. Since stage 1 of the compiled memory design
([`compiled-memory-stage1.md`](compiled-memory-stage1.md)) the compiled `List` allocation goes through
the counted-slab runtime allocator on POSIX, so the QuarkLang rows below measure that allocator, and
the historical no-use shape is no longer erased by the optimizer.

{{summary}}

## Method

The harness is one shape in four languages, plus the dead-code floor and the interpreter's logical
block policy:

- `bench/tide/qk/tide.qk` — the language's tide: a `List<int>` literal, a read of both ends, and
  `delete`, repeated. The compiler has no address-of, so the opaque sink is the string returned by
  `toString()`: `tide.qtide_sink` stores it in a volatile global and `tide.qtide_release` releases it
  through the compiled runtime's `ql_free`, because since stage 2 (D1) the string builders are
  counted-slab allocations too.
  `bench/tide/qk/tide_floor.qk` is the same loop with the sink removed, and
  `bench/tide/qk/tide_legacy_floor.qk` is the historical shape with no read, sink or checksum.
- `bench/tide/c/tide.c` — the C control: `malloc(12)`, three writes, a volatile sink, and `free`.
  Its floor variant removes the sink.
- `bench/tide/rust/tide.rs` — the Rust control: `Vec::with_capacity(3)`, three pushes,
  `black_box(v.as_ptr())` and drop. Its floor variant removes `black_box`.
- `bench/tide/go/tide/main.go` — the Go control: `make([]int32, 3)`, three writes, a global sink and
  the garbage collector. Its floor variant uses a local slice and no sink.
- `bench/tide/policy` — the interpreter's `MemoryManager`, driven through the same
  allocate/use/release shape so its reuse, fragmentation and block-count counters are measured rather
  than quoted.

`bench/tide/run.sh` builds every program once, runs it once untimed, then runs it `--runs` times and
reports the median, minimum, maximum, standard deviation and `(max-min)/median` spread. No run is
presented as "the best"; a single best run is not the result. `--record` writes
`spec/bench/tide.tsv`, the authored input to this generated report. `--quick` uses 1e6 rounds for CI
smoke; the checked-in record uses 1e8 rounds and five runs. `--attribution` adds diagnostic controls:
the same loop with no allocation, string or sink; the same loop with only the opaque sink; and C
controls that time the counted-slab List allocations, the List header stores, the counted-slab
string allocation, the integer formatter, the buffer framing and the real `ql_list_int_str` helper. Those
rows are printed after the record and are never written to `spec/bench/tide.tsv`, so the recorded
schema and the embedded claim block stay stable.

The QuarkLang observable row necessarily includes the `List` header, the buffer and `toString()`,
because that runtime call is the only opaque sink the compiler exposes. The C control is the minimal
`malloc`/`free` shape the old claim compared against, not a rendering-equivalent workload. On the
pre-stage-1 libc path the `QuarkLang compiled, no sink` row and the legacy row measured LLVM removing
the allocation and then the whole loop. Stage 1 emits opaque `ql_alloc`/`ql_free` calls, so those two
rows now measure the counted slabs as well; the historical no-use shape is kept precisely to show
that the optimizer can no longer erase it. The C/Rust/Go floor rows still measure their own optimizer
floors.

`List<int>.toString()` formats each element in place with a local decimal writer, so it allocates no
per-element string and calls no libc formatter; the one allocation it still performs is the returned
string, which has been a counted-slab allocation since stage 2 (D1) and is released through
`ql_free`. The attribution controls retain the `snprintf` path as
a comparison row, which is why the rejected direction is still measured rather than assumed.

The interpreter implements the block policy as a logical layer over the host allocator; its policy
rows describe that accounting and not the compiled tide. The compiled artifact's `List` allocations
go through the counted-slab allocator on POSIX, and since stage 2 (D1) so do the runtime's own
helper allocations (strings, split lists, hash tables, task and channel objects); the only host
allocations left are the allocator's own block storage. Windows keeps the libc path because the
runtime is not linked there. The compiled rows therefore measure the counted-slab path, not the
interpreter's logical blocks.

## Reproducing and checking

```sh
./bench/tide/run.sh                 # 1e8 rounds, 5 runs, prints the TSV record
./bench/tide/run.sh --attribution   # adds the phase rows that decompose the observable row
./bench/tide/run.sh --record        # writes spec/bench/tide.tsv, then regenerates this report
./bench/tide/run.sh --quick         # 1e6 rounds, 3 runs: CI smoke
go generate ./internal/benchgen     # renders docs/tide-benchmark.md from the record and template
go run ./cmd/qkbench -check         # fails when the committed report is stale
```

After `--record`, copy the marked measurement block into `README.md` and `spec/STANDARD.md` §10 as well;
the report and those two claims are checked against the record, and the test fails until all three agree.

`TestGeneratedTideBenchmarkIsCurrent` runs the check in the ordinary `go test ./...` gate.
`TestTideClaimsMatchMeasurement` extracts the marked block below from this report, `README.md` and
`spec/STANDARD.md` §10 and fails when the three copies differ, so a hand-edited number cannot survive.
A toolchain that is missing is skipped explicitly and listed under `skipped` in the record; no number
is invented for it. The record is generated once on the reference class named in the block and
checked in; CI runs the quick harness to keep it working, and the drift gate compares the generated
text rather than rerunning the timing on a different machine.
