#!/bin/bash
# Interpreter benchmark runner: best-of-N per benchmark, so one noisy sample cannot mislead.
#
# Usage: scripts/bench-interp.sh [runs] [benchtime]
#   runs       samples per benchmark (default 10)
#   benchtime  Go benchmark duration per sample (default 1s)
#   BENCHES    regex of benchmarks to run (default: the four interpreter suites)
#
# Reports min, median and dispersion per benchmark. Why min: background load only ever adds time,
# so the minimum is the closest estimate of the true cost, and the dispersion shows how noisy the
# box is. On a desktop with a GUI running, dispersion of 10-20% is normal and micro-optimizations
# cannot be resolved there; on a CI runner (no GUI load) it is typically <1%, which is why the
# benchmark workflow exists.
set -eu
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RUNS="${1:-10}"
BENCHTIME="${2:-1s}"
BENCHES="${BENCHES:-BenchmarkEvalLoop1M|BenchmarkFib24|BenchmarkFuncCalls100K|BenchmarkStringProcessing1M}"

cd "$ROOT"
echo "# interpreter benchmarks: ${RUNS} runs x ${BENCHTIME}"
go test ./internal/lang/ -run '^$' -bench "$BENCHES" -benchtime="$BENCHTIME" -count="$RUNS" 2>/dev/null \
  | grep '^Benchmark' | awk '{ print $1, $3 }' \
  | awk '
    {
      name = $1; ns = $2 + 0
      if (!(name in seen)) { seen[name] = 1; order[++n] = name }
      cnt[name]++
      val[name, cnt[name]] = ns
      if (!(name in min) || ns < min[name]) min[name] = ns
      if (ns > max[name]) max[name] = ns
    }
    END {
      printf "%-46s %10s %12s %12s\n", "benchmark", "min ms", "median ms", "dispersion"
      for (i = 1; i <= n; i++) {
        name = order[i]
        # insertion sort the few samples to get the median
        for (a = 1; a <= cnt[name]; a++) sorted[a] = val[name, a]
        for (a = 2; a <= cnt[name]; a++) {
          key = sorted[a]; b = a - 1
          while (b >= 1 && sorted[b] > key) { sorted[b+1] = sorted[b]; b-- }
          sorted[b+1] = key
        }
        mid = sorted[int((cnt[name] + 1) / 2)]
        printf "%-46s %10.2f %12.2f %11.1f%%\n", name, min[name]/1e6, mid/1e6, (max[name]-min[name])/min[name]*100
      }
    }'
