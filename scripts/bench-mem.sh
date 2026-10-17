#!/bin/bash
# Measure the stage-1 allocator against the libc path on the checked-in workload with the volatile
# sink. Reproduce: scripts/bench-mem.sh [output.md]
#
#   RUNS=15 ITERS=2000000 SIZES="16 64 512 4096 8192" scripts/bench-mem.sh
#
# The host binary is the compiled artifact's pre-stage-1 path (malloc/calloc/free); the ql binary is
# the counted-slab runtime. Both run the same bench/memory/alloc_bench.c, same sizes, same sink.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RUNS="${RUNS:-15}"
ITERS="${ITERS:-2000000}"
SIZES="${SIZES:-16 64 512 4096 8192}"
BUILD="${BUILD_DIR:-$HOME/.cache/qk-mem-bench}"
OUT="${1:-}"

command -v clang >/dev/null 2>&1 || { echo "clang is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 1; }

case "$BUILD" in
  ""|"/"|"$HOME") echo "refusing BUILD_DIR=$BUILD: give a scratch directory to rebuild" >&2; exit 2 ;;
esac
rm -rf "$BUILD"
mkdir -p "$BUILD"
cat "$ROOT/compiler/internal/rtcat/msg_c.h" "$ROOT/compiler/qthreads.c.txt" > "$BUILD/runtime.c"

clang -O3 -DUSE_QL_ALLOC=0 "$ROOT/bench/memory/alloc_bench.c" -o "$BUILD/host"
clang -O3 -DUSE_QL_ALLOC=1 "$BUILD/runtime.c" "$ROOT/bench/memory/alloc_bench.c" -pthread -o "$BUILD/ql"

raw="$BUILD/raw.csv"
: > "$raw"
for size in $SIZES; do
  for op in malloc calloc; do
    for allocator in host ql; do
      for ((r = 0; r < RUNS; r++)); do
        "$BUILD/$allocator" "$op" "$size" "$ITERS" >> "$raw"
      done
    done
  done
done

{
  echo "# Counted-slab allocator benchmark"
  echo
  echo "Measured on $(date -u '+%Y-%m-%dT%H:%M:%SZ') · $(uname -srm)"
  echo
  echo "$(clang --version | head -1)"
  echo
  echo "Workload: \`bench/memory/alloc_bench.c\` (volatile sink, payload touched, allocate + free per"
  echo "iteration). Host = the pre-stage-1 libc path; counted-slab = \`ql_alloc\`/\`ql_calloc\` + \`ql_free\`."
  echo "n = $RUNS runs of $ITERS iterations per cell; times are ns/op."
  echo
  python3 - "$raw" <<'PY'
import csv, statistics, sys

rows = []
with open(sys.argv[1], newline='') as f:
    for r in csv.reader(f):
        if len(r) != 6:
            continue
        allocator, op, size, iters, total_ns, ns = r
        rows.append((allocator, op, int(size), float(ns)))

groups = {}
for allocator, op, size, ns in rows:
    groups.setdefault((op, size, allocator), []).append(ns)

def stats(xs):
    mean = statistics.fmean(xs)
    sd = statistics.stdev(xs) if len(xs) > 1 else 0.0
    return statistics.median(xs), mean, sd, min(xs), max(xs)

print("| op | bytes | allocator | n | median ns/op | mean ns/op | sd | min | max |")
print("|---|---:|---|---:|---:|---:|---:|---:|---:|")
for (op, size, allocator) in sorted(groups, key=lambda k: (k[0], k[1], k[2])):
    med, mean, sd, lo, hi = stats(groups[(op, size, allocator)])
    print(f"| {op} | {size} | {allocator} | {len(groups[(op, size, allocator)])} | {med:.2f} | {mean:.2f} | {sd:.2f} | {lo:.2f} | {hi:.2f} |")

print()
print("| op | bytes | libc median | counted-slab median | ratio (slab / libc) |")
print("|---|---:|---:|---:|---:|")
for op in ("malloc", "calloc"):
    for size in sorted({s for (o, s, _a) in groups if o == op}):
        h = statistics.median(groups[(op, size, "libc")])
        q = statistics.median(groups[(op, size, "counted-slab")])
        print(f"| {op} | {size} | {h:.2f} | {q:.2f} | {q / h:.2f}x |")
PY
} | if [ -n "$OUT" ]; then tee "$OUT"; else cat; fi
