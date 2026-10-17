# Tide benchmark

The tide workload allocates a three-element list, uses it and releases it in a loop. The earlier
README row `Tide, 1e8 rounds = 1 ms` had no checked-in source, and a design review showed that the
1 ms was LLVM dead-code elimination: at `-O3` the allocation was removed, while the same IR at `-O0`
still referenced `malloc`/`calloc`/`free`. This report and the harness behind it replace the claim
with a measurement the optimizer cannot erase.

{{summary}}

## Method

The harness is one shape in four languages, plus the dead-code floor and the interpreter's logical
block policy:

- `bench/tide/qk/tide.qk` — the language's tide: a `List<int>` literal, a read of both ends, and
  `delete`, repeated. The compiler has no address-of, so the opaque sink is the string returned by
  `toString()`: `tide.qtide_sink` stores it in a volatile global and `tide.qtide_release` frees it.
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
smoke; the checked-in record uses 1e8 rounds and five runs.

The QuarkLang observable row necessarily includes the `List` header, the buffer and `toString()`,
because that runtime call is the only opaque sink the compiler exposes. The C control is the minimal
`malloc`/`free` shape the old claim compared against, not a rendering-equivalent workload. The
`QuarkLang compiled, no sink` row keeps the checksum and removes only the sink, so it separates
"the allocation was removed" from "the whole loop was removed"; the legacy row is the fully removed
loop. The last row is therefore the dead-code floor the old `1 ms` figure measured, not an
allocation result.

The interpreter and the compiled artifact are different allocators today: the interpreter implements
the block policy as a logical layer over the host allocator, while `qkc` lowers `List` allocation to
the host `malloc`/`calloc`/`free`. The policy rows describe the former and do not describe the
compiled tide.

## Reproducing and checking

```sh
./bench/tide/run.sh                 # 1e8 rounds, 5 runs, prints the TSV record
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
