# Performance claims

This report is the checked-in measurement behind the performance claims in `README.md`. It is
generated from `spec/bench/claims.tsv`, the record `bench/claims/run.sh` writes, and the prose below;
the marked block is embedded verbatim in the README, so a hand-edited number fails
`TestPerformanceClaimsMatchMeasurement` instead of drifting away from the record.

The inventory of every claim, including the ones this report replaces or withdraws, is
[`performance-claims-inventory.md`](performance-claims-inventory.md).

{{summary}}

## What each workload measures

- **fib** — recursive `fib(n)`, one call per timed run. The QuarkLang and Erlang sources carry the
  argument; the C, Rust and Go controls read it from argv (C through a `volatile`) so the optimizer
  cannot fold the recursion at compile time. Every run prints the result, which is the checksum.
- **sum** — one `sum(ident, 0, 1e9)` call in QuarkLang, which the interpreter and compiler resolve to
  the arithmetic-series closed form, against a plain Go loop over the same terms. The elapsed time is
  measured in process; the QuarkLang row is below `clock()`'s one-microsecond resolution on this
  build, and the record says so instead of inventing a finer number.
- **fib(20) latency** — the P50 and P99 thresholds of 1000 `fib(20)` calls. A threshold is found by
  counting how many calls finish at or below it and binary-searching that count, so the histogram
  itself is not in the timed loop. The row's value is the threshold in microseconds, not the total
  runtime.
- **interpreter versus native** — the same six checked-in programs run once by `quark` and once as a
  `qkc` binary, with the timed region inside the program on both sides. Output parity is required:
  the harness fails when the two checksums differ.

Two measured results contradict the table this report replaced, and they stay visible rather than
being tuned away: the Erlang `fib(30)` control is far faster than the withdrawn four-digit figure, and
the `str` row's native binary is slower than the interpreter on this build, so the earlier
"quadratic on both paths" note did not survive. The block reports the medians and the spread for both
rows; no workload was reshaped to restore the earlier ordering.

## Method

`bench/claims/run.sh` builds every program once, runs it once untimed, then runs it five times and
reports the median, minimum, maximum, standard deviation and `(max-min)/median` spread. No run is
presented as "the best". `--record` writes `spec/bench/claims.tsv` and refuses to do so unless the
host's 1-minute load average is below 0.25 per logical CPU, using the same probe as the tide harness;
the measured load is part of the record and is stated above.

The record's authority depends on its shape, so the harness has no `--n` override: every workload's
round count and the five-run count are fixed in the checked-in sources and in
`spec/bench/claims-shapes.tsv`. `--record` refuses any other shape, and the generator rejects a
record whose rows do not match that table. `--record --dry-run` validates the shape without measuring
or writing, and `--quick` takes one run per row for CI smoke and can never be recorded.

A missing Rust or Erlang toolchain is skipped explicitly and named under `skipped` in the record; no
number is invented for it. The record is generated once on the reference class named in the block and
checked in; CI runs the quick harness to keep the workloads working, and the drift gate compares the
generated text rather than rerunning the timing on a different machine.

## Reproducing and checking

```sh
./bench/claims/run.sh                    # print the record (5 timed runs per row)
./bench/claims/run.sh --record           # on a quiet host: write spec/bench/claims.tsv and regenerate this report
./bench/claims/run.sh --record --dry-run # validate the canonical shape, print the plan, write nothing
./bench/claims/run.sh --quick            # one run per row: CI smoke
./bench/claims/run.sh --compile-only     # build the C controls with -Werror, measure nothing
go generate ./internal/benchgen          # render this report and docs/tide-benchmark.md
go run ./cmd/qkbench -check -suite claims
```

After `--record`, copy the marked block into `README.md`. `TestGeneratedPerformanceClaimsIsCurrent`
runs the check in the ordinary `go test ./...` gate, and `TestPerformanceClaimsMatchMeasurement`
compares the README's copy with this report and with the record, so a quoted number cannot survive a
record change on its own.

## What this report does not claim

The README used to carry performance numbers with no checked-in producer. They were removed rather
than restated, and the inventory keeps the earlier figure and why it could not stay:

- **The engine table** (`BenchmarkEvalLoop1M` 73.95 → 51.09 ms and its two neighbours) compared a
  tree-walking engine with the bytecode VM, but the pre-VM engine is no longer selectable in this
  tree; the figures are history, not a current comparison.
- **The toolchain before/after table** compared binaries that are not in the tree. The "before"
  column cannot be rebuilt from this checkout, so it is not a claim about the current build.
- **The incremental-compilation multiple, the cgo-call cost and the lexing experiment** were one-off
  observations; no checked-in workload produces them.
- **The regex and Cleg numbers** belong to sibling repositories, not to this one.
- **The 8-way concurrency row** has no checked-in workload on either side.
- **The CPython comparison** has no checked-in CPython control.
- **The qkdoc line count and time** are reproducible only as a manual `scripts/bench-tools.sh` run,
  which writes no record.

A claim that can be measured again belongs in the record above. A claim that cannot is either
withdrawn or named here as history, and it does not keep a number where a reader could take it as a
fact about the current build.
