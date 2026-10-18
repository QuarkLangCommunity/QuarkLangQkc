#!/usr/bin/env bash
# Tide benchmark harness: one allocate/use/release shape in QuarkLang, C, Rust and Go, with an
# opaque sink so the allocation is observable. It also measures the sink-free dead-code floor, the
# historical no-use shape the old 1 ms claim measured, and the interpreter block-policy counters.
#
#   bench/tide/run.sh --quick              # 1e6 rounds, 3 runs: CI smoke
#   bench/tide/run.sh                      # 1e8 rounds, 5 runs: the recorded measurement
#   bench/tide/run.sh --record             # write spec/bench/tide.tsv and regenerate the report
#   bench/tide/run.sh --attribution        # add the phase rows that decompose the observable row
#
# The output is a TSV record; the first lines are metadata and the rest are the measured rows. A
# missing Rust or Go toolchain is skipped explicitly and recorded as a skip, never invented.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

N=100000000
RUNS=5
RECORD=0
ATTR=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --quick) N=1000000; RUNS=3 ;;
    --n) N="${2:?--n needs a value}"; shift ;;
    --runs) RUNS="${2:?--runs needs a value}"; shift ;;
    --record) RECORD=1 ;;
    --attribution) ATTR=1 ;;
    --help|-h)
      sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "tide: unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

[[ "$N" =~ ^[1-9][0-9]*$ ]] || { echo "tide: --n must be a positive integer" >&2; exit 2; }
[[ "$RUNS" =~ ^[1-9][0-9]*$ ]] || { echo "tide: --runs must be a positive integer" >&2; exit 2; }
if [ "$RECORD" -eq 1 ] && [ "$RUNS" -lt 3 ]; then
  echo "tide: --record needs at least 3 runs so the recorded spread is meaningful" >&2
  exit 2
fi
if [ "$RECORD" -eq 1 ] && [ "$ATTR" -eq 1 ]; then
  echo "tide: --attribution is a diagnostic and cannot be recorded as the canonical measurement" >&2
  exit 2
fi
command -v python3 >/dev/null 2>&1 || { echo "tide: python3 is required for the statistics" >&2; exit 2; }
command -v clang >/dev/null 2>&1 || { echo "tide: clang is required (qkc lowers to LLVM IR)" >&2; exit 2; }
command -v go >/dev/null 2>&1 || { echo "tide: go is required (qkc and the policy workload)" >&2; exit 2; }

CACHE="${XDG_CACHE_HOME:-$HOME/.cache}"
mkdir -p "$CACHE"
BUILD="$(mktemp -d "$CACHE/tide-bench.XXXXXX")"
case "$BUILD" in
  "$CACHE"/tide-bench.*) ;;
  *) echo "tide: refusing to use build directory $BUILD" >&2; exit 2 ;;
esac
cleanup() { rm -rf "$BUILD"; }
trap cleanup EXIT

echo "→ building qkc and the harness binaries in $BUILD" >&2
(cd compiler && go build -o "$BUILD/qkc" .)
export QUARK_CACHE="$BUILD/qkc-cache"

clang -O2 -c bench/tide/sink/sink.c -o "$BUILD/sink.o"
ar rcs "$BUILD/libtide.a" "$BUILD/sink.o"

sed -E "s/^    int rounds = [0-9]+;$/    int rounds = $N;/" bench/tide/qk/tide.qk > "$BUILD/tide.qk"
sed -E "s/^    int rounds = [0-9]+;$/    int rounds = $N;/" bench/tide/qk/tide_floor.qk > "$BUILD/tide_floor.qk"
sed -E "s/^    int rounds = [0-9]+;$/    int rounds = $N;/" bench/tide/qk/tide_legacy_floor.qk > "$BUILD/tide_legacy_floor.qk"
for src in tide tide_floor tide_legacy_floor; do
  grep -q "int rounds = $N;" "$BUILD/$src.qk" || {
    echo "tide: the round count in $src.qk was not replaced (expected $N)" >&2
    exit 1
  }
done
if [ "$ATTR" -eq 1 ]; then
  for src in tide_empty tide_sink; do
    sed -E "s/^    int rounds = [0-9]+;$/    int rounds = $N;/" "bench/tide/qk/$src.qk" > "$BUILD/$src.qk"
    grep -q "int rounds = $N;" "$BUILD/$src.qk" || {
      echo "tide: the round count in $src.qk was not replaced (expected $N)" >&2
      exit 1
    }
  done
fi

(cd "$BUILD" && ./qkc -c -o "$BUILD/tide-qk" "$BUILD/tide.qk") >/dev/null
(cd "$BUILD" && ./qkc -c -o "$BUILD/tide-qk-floor" "$BUILD/tide_floor.qk") >/dev/null
(cd "$BUILD" && ./qkc -c -o "$BUILD/tide-qk-legacy" "$BUILD/tide_legacy_floor.qk") >/dev/null
if [ "$ATTR" -eq 1 ]; then
  (cd "$BUILD" && ./qkc -c -o "$BUILD/tide-qk-empty" "$BUILD/tide_empty.qk") >/dev/null
  (cd "$BUILD" && ./qkc -c -o "$BUILD/tide-qk-sink" "$BUILD/tide_sink.qk") >/dev/null
fi

EVIDENCE="not checked on this platform"
if command -v nm >/dev/null 2>&1; then
  if ! nm -u "$BUILD/tide-qk" 2>/dev/null | grep -Eq 'malloc|calloc|free'; then
    echo "tide: the observable QuarkLang binary references no host allocator symbol" >&2
    exit 1
  fi
  if nm "$BUILD/tide-qk" 2>/dev/null | grep -Eq 'ql_alloc|ql_free'; then
    EVIDENCE="the observable QuarkLang binary defines ql_alloc/ql_free and routes the List allocation and the toString string through them; it imports malloc/free only for the allocator's host blocks"
  else
    EVIDENCE="nm -u on the observable QuarkLang binary lists malloc/calloc/free (the List allocation is not routed through ql_alloc on this platform)"
  fi
fi

CFLAGS_C="-O3 -flto=thin"
if ! clang $CFLAGS_C bench/tide/c/tide.c -o "$BUILD/tide-c" 2>"$BUILD/c.err"; then
  CFLAGS_C="-O3"
  clang $CFLAGS_C bench/tide/c/tide.c -o "$BUILD/tide-c"
fi
clang $CFLAGS_C bench/tide/c/tide_floor.c -o "$BUILD/tide-c-floor"

SKIPS=()
if command -v rustc >/dev/null 2>&1; then
  rustc -O bench/tide/rust/tide.rs -o "$BUILD/tide-rs"
  rustc -O bench/tide/rust/tide_floor.rs -o "$BUILD/tide-rs-floor"
  RUST_VERSION="$(rustc --version)"
else
  RUST_VERSION="skipped"
  SKIPS+=("rust: rustc not found")
fi

go build -o "$BUILD/tide-go" ./bench/tide/go/tide
go build -o "$BUILD/tide-go-floor" ./bench/tide/go/tide_floor

CLASS="$(uname -s) $(uname -m), $(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc) logical CPUs"
TOOLCHAIN="$(clang --version | head -1); $(go version); $RUST_VERSION"
STRICT="qkc default -O3 -flto=thin; C clang $CFLAGS_C; Rust rustc -O; Go go build (default)"
COUNTED="in-process loop wall time: clock()/clock_gettime(CLOCK_MONOTONIC)/Instant::now()/time.Now(); process startup and printing excluded; observable rows call an opaque sink so the allocation cannot be removed"
if [ "${#SKIPS[@]}" -eq 0 ]; then
  SKIPPED="none"
else
  SKIPPED="$(IFS=';'; printf '%s' "${SKIPS[*]}")"
fi

ROWS="$BUILD/rows.tsv"
: > "$ROWS"

# checksum_sink accumulates the checksums seen so far, so every control must agree on the work done.
CHECKSUMS=()

run_timing() { # usage: run_timing key label language checksum-mode note command...
  local key="$1" label="$2" language="$3" checksum_mode="$4" note="$5"
  shift 5
  local values=() out="$BUILD/out.txt" line
  local r
  # One untimed warm-up keeps the first timed run from carrying the cold-start cost.
  if ! "$@" "$N" > /dev/null 2>"$BUILD/err.txt"; then
    echo "tide: $key warm-up failed: $(cat "$BUILD/err.txt")" >&2
    exit 1
  fi
  for ((r = 1; r <= RUNS; r++)); do
    if ! "$@" "$N" > "$out" 2>"$BUILD/err.txt"; then
      echo "tide: $key failed: $(cat "$BUILD/err.txt")" >&2
      exit 1
    fi
    local first=""
    local second=""
    {
      IFS= read -r first || true
      IFS= read -r second || true
    } < "$out"
    [[ "$first" =~ ^[0-9]+$ ]] || { echo "tide: $key printed no microsecond duration: [$first]" >&2; exit 1; }
    values+=("$first")
    if [ "$checksum_mode" = "required" ]; then
      [[ "$second" =~ ^[0-9]+$ ]] || { echo "tide: $key printed no checksum: [$second]" >&2; exit 1; }
      if [ "${#CHECKSUMS[@]}" -eq 0 ]; then CHECKSUMS+=("$second"); fi
      if [ "$second" != "${CHECKSUMS[0]}" ]; then
        echo "tide: $key checksum $second does not match ${CHECKSUMS[0]}" >&2
        exit 1
      fi
    elif [ -n "$second" ]; then
      echo "tide: $key printed an unexpected second line: [$second]" >&2
      exit 1
    fi
  done
  local checksum="-"
  if [ "$checksum_mode" = "required" ]; then checksum="${CHECKSUMS[0]}"; fi
  printf '%s\n' "${values[@]}" | python3 bench/tide/stats.py \
    --key "$key" --label "$label" --language "$language" --rounds "$N" --runs "$RUNS" \
    --unit us --checksum "$checksum" --note "$note" >> "$ROWS"
}

NOTE_QK='`List<int>` literal + `toString` string (both counted slabs since D1) + opaque C sink + release + delete'
NOTE_C="malloc(12) + 3 writes + volatile sink + free"
NOTE_RUST="Vec::with_capacity(3) + 3 pushes + black_box(ptr) + drop"
NOTE_GO="make([]int32,3) + global sink + GC"

run_timing compiled_observable "QuarkLang compiled, observable" QuarkLang required "$NOTE_QK" "$BUILD/tide-qk"
run_timing c_observable "C control, volatile sink" C required "$NOTE_C" "$BUILD/tide-c"
if command -v rustc >/dev/null 2>&1; then
  run_timing rust_observable "Rust control, black_box" Rust required "$NOTE_RUST" "$BUILD/tide-rs"
fi
run_timing go_observable "Go control, global sink" Go required "$NOTE_GO" "$BUILD/tide-go"

run_timing compiled_floor "QuarkLang compiled, no sink" QuarkLang required "same loop with the sink removed; the checksum keeps the loop alive; the opaque ql_alloc/ql_free calls are not elided" "$BUILD/tide-qk-floor"
run_timing c_floor "C control, no sink (optimizer floor)" C required "same loop with the volatile sink removed" "$BUILD/tide-c-floor"
if command -v rustc >/dev/null 2>&1; then
  run_timing rust_floor "Rust control, no black_box (optimizer floor)" Rust required "same loop with black_box removed" "$BUILD/tide-rs-floor"
fi
run_timing go_floor "Go control, local slice (optimizer floor)" Go required "same loop with the global sink removed" "$BUILD/tide-go-floor"
run_timing legacy_floor "QuarkLang compiled, legacy shape (no read/sink)" QuarkLang optional "allocate/delete with no read, sink or checksum: the shape the old 1 ms claim measured on the pre-stage-1 libc path; stages 1-2 run it through counted slabs" "$BUILD/tide-qk-legacy"

# Attribution rows are diagnostic: they decompose the observable row and are not written to the
# canonical record, whose schema and claim block must stay stable.
ATTR_ROWS=""
if [ "$ATTR" -eq 1 ]; then
  ATTR_ROWS="$BUILD/attribution.tsv"
  : > "$ATTR_ROWS"
  ROWS_MAIN="$ROWS"
  ROWS="$ATTR_ROWS"
  run_timing attribution_empty "Attribution base: loop arith and checksum only" QuarkLang required "the tide.qk loop and checksum with no allocation, string or sink; every phase below is measured against this row" "$BUILD/tide-qk-empty"
  run_timing attribution_sink "Attribution base + opaque C sink" QuarkLang required "the base loop plus the opaque sink call on a constant string; no allocation and no release" "$BUILD/tide-qk-sink"

  # C controls that split the phases the language surface cannot isolate: the real runtime helper is
  # extracted verbatim, and the List allocations go through the same counted-slab runtime.
  # Extract the real helper together with the formatter it uses, verbatim.
  awk '
    /^static int ql_itoa\(int v, char\* buf\) \{/ { copy = 1 }
    copy { print }
    /^char\* ql_list_int_str\(const int\* buf, int head, int tail\) \{/ { in_list = 1 }
    in_list && /^\}/ { exit }
  ' compiler/qthreads.c.txt > "$BUILD/ql_list_int_str.inc"
  awk '
    /^char\* ql_list_int_str\(const int\* buf, int head, int tail\) \{/ { skip = 1 }
    skip == 0 { print }
    skip == 1 && /^\}/ { skip = 0 }
  ' compiler/qthreads.c.txt > "$BUILD/attrib_runtime_body.c"
  printf 'char* ql_list_int_str(const int* buf, int head, int tail);\n' > "$BUILD/attrib_runtime.c"
  cat compiler/internal/rtcat/msg_c.h "$BUILD/attrib_runtime_body.c" >> "$BUILD/attrib_runtime.c"
  clang -O2 -pthread -I"$BUILD" bench/tide/attrib/attrib.c "$BUILD/attrib_runtime.c" -o "$BUILD/tide-attrib"
  for phase in base allocpair header string_alloc int_fmt fast_fmt bookkeeping render; do
    run_timing "attr_$phase" "Attribution control, $phase" C optional "C control for the $phase phase of the observable path; the render control calls the real ql_list_int_str extracted from compiler/qthreads.c.txt" "$BUILD/tide-attrib" "$phase"
  done
  ROWS="$ROWS_MAIN"
fi

# Interpreter block-policy unit workload: counters, not wall time.
POLICY="$(go run ./bench/tide/policy -waves 10000 -frag-waves 1000)"
while IFS=$'\t' read -r kind waves allocs rate newblocks blocks frag; do
  case "$kind" in
    reuse)
      printf 'counter\tpolicy_reuse\tInterpreter block policy, reuse workload\tinterpreter\t%s\t%s\t%s\tratio\t-\t-\t-\t-\t-\tReusedCount/AllocCalls; the block count converges to %s blocks\n' \
        "$waves" "$allocs" "$rate" "$blocks" >> "$ROWS"
      ;;
    fragmentation)
      printf 'counter\tpolicy_fragmentation\tInterpreter block policy, mixed survival\tinterpreter\t%s\t%s\t%s\tratio\t-\t-\t-\t-\t-\tFragmentation() after the mixed-size waves; %s blocks stay live by design\n' \
        "$waves" "$allocs" "$frag" "$blocks" >> "$ROWS"
      ;;
    *)
      echo "tide: unexpected policy row: $kind" >&2
      exit 1
      ;;
  esac
done <<< "$POLICY"

print_record() {
  printf '# record\ttide-v1\n'
  printf '# class\t%s\n' "$CLASS"
  printf '# toolchain\t%s\n' "$TOOLCHAIN"
  printf '# flags\t%s\n' "$STRICT"
  printf '# counted\t%s\n' "$COUNTED"
  printf '# skipped\t%s\n' "$SKIPPED"
  printf '# evidence\t%s\n' "$EVIDENCE"
  printf '# columns\tkind\tkey\tlabel\tlanguage\trounds\truns\tmedian\tunit\tmin\tmax\tstdev\tspread_pct\tchecksum\tnote\n'
  cat "$ROWS"
}

if [ "$RECORD" -eq 1 ]; then
  TMP="$BUILD/tide.tsv"
  print_record > "$TMP"
  mv "$TMP" spec/bench/tide.tsv
  go generate ./internal/benchgen
  go run ./cmd/qkbench -check
  echo "→ recorded spec/bench/tide.tsv and regenerated docs/tide-benchmark.md" >&2
else
  print_record
  if [ "$ATTR" -eq 1 ]; then
    echo "→ attribution rows (diagnostic; not part of the record)" >&2
    cat "$ROWS" "$ATTR_ROWS"
  fi
fi
