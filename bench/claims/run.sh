#!/usr/bin/env bash
# Performance-claims harness: the fib, closed-form sum and fib(20) latency claims in README.md, plus
# the interpreter-versus-native table, measured as one record.
#
#   bench/claims/run.sh                    # print the record (5 timed runs per row)
#   bench/claims/run.sh --record           # write spec/bench/claims.tsv and regenerate the report
#   bench/claims/run.sh --record --dry-run # validate the canonical shape, print the plan, write nothing
#   bench/claims/run.sh --quick            # one timed run per row: CI smoke, never recorded
#   bench/claims/run.sh --compile-only     # build every C control with -Werror, measure nothing
#
# Every row prints the result of its work as a checksum, and the harness refuses a row whose checksum
# changes between runs. The fib sources read their argument from argv (C/Rust/Go) or carry it in the
# checked-in source (QuarkLang/Erlang), so the result cannot be constant-folded away.
#
# --record is a measurement, so it refuses a busy host through the same load probe as the tide
# harness, and it refuses every shape except the canonical one: the record's rows must carry the
# rounds declared in spec/bench/claims-shapes.tsv and five timed runs. The generator checks the same
# shape, so a probe record cannot pass as the authoritative one.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

CANONICAL_RUNS=5
RUNS=$CANONICAL_RUNS
RECORD=0
DRY_RUN=0
COMPILE_ONLY=0
QUICK=0
LOAD_PER_CPU_MAX=0.25
while [ "$#" -gt 0 ]; do
  case "$1" in
    --record) RECORD=1 ;;
    --dry-run) DRY_RUN=1 ;;
    --quick) QUICK=1; RUNS=1 ;;
    --runs) RUNS="${2:?--runs needs a value}"; shift ;;
    --compile-only) COMPILE_ONLY=1 ;;
    --help|-h)
      sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "claims: unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

[[ "$RUNS" =~ ^[1-9][0-9]*$ ]] || { echo "claims: --runs must be a positive integer" >&2; exit 2; }
if [ "$DRY_RUN" -eq 1 ] && [ "$RECORD" -eq 0 ]; then
  echo "claims: --dry-run previews a record and needs --record" >&2
  exit 2
fi
if [ "$RECORD" -eq 1 ] && { [ "$QUICK" -eq 1 ] || [ "$RUNS" -ne "$CANONICAL_RUNS" ]; }; then
  echo "claims: refusing --record for a non-canonical shape: got --runs $RUNS, but the record is the canonical $CANONICAL_RUNS-run measurement" >&2
  echo "claims: spec/bench/claims.tsv is authoritative only at that shape; a probe must run without --record" >&2
  exit 2
fi
if [ "$COMPILE_ONLY" -eq 1 ] && { [ "$RECORD" -eq 1 ] || [ "$DRY_RUN" -eq 1 ] || [ "$QUICK" -eq 1 ]; }; then
  echo "claims: --compile-only builds the C controls and cannot be combined with a measurement mode" >&2
  exit 2
fi
if [ "$DRY_RUN" -eq 1 ]; then
  echo "claims: dry run: canonical shape accepted ($RUNS timed runs; the workload rounds are in spec/bench/claims-shapes.tsv)"
  echo "claims: dry run: the record metadata would declare '# record<TAB>claims-v1' and '# runs<TAB>$RUNS'"
  echo "claims: dry run: would write spec/bench/claims.tsv after measuring the fib, sum, latency and backend rows"
  echo "claims: dry run: no measurement taken and no file written"
  exit 0
fi

CACHE="${XDG_CACHE_HOME:-$HOME/.cache}"
mkdir -p "$CACHE"
BUILD="$(mktemp -d "$CACHE/qk-claims.XXXXXX")"
case "$BUILD" in
  "$CACHE"/qk-claims.*) ;;
  *) echo "claims: refusing to use build directory $BUILD" >&2; exit 2 ;;
esac
cleanup() { rm -rf "$BUILD"; }
trap cleanup EXIT

# clang flags for the C controls; the fallback drops thin LTO when the local plugin is missing.
CFLAGS_C="-O3 -flto=thin"

# compile_controls builds every checked-in C control with warnings fatal; it is the CI gate that keeps
# a stale prototype from waiting for the next measurement.
compile_controls() {
  clang $CFLAGS_C -Wall -Wextra -Werror bench/claims/fib/fib.c -o "$BUILD/fib-c" 2>"$BUILD/c.err" || {
    CFLAGS_C="-O3"
    clang $CFLAGS_C -Wall -Wextra -Werror bench/claims/fib/fib.c -o "$BUILD/fib-c"
  }
  clang $CFLAGS_C -Wall -Wextra -Werror bench/claims/p99/p99.c -o "$BUILD/p99-c"
}

if [ "$COMPILE_ONLY" -eq 1 ]; then
  compile_controls
  echo "claims: compiled the checked-in C controls with -Werror; no measurement was run" >&2
  exit 0
fi

command -v go >/dev/null 2>&1 || { echo "claims: go is required (qkc and the controls)" >&2; exit 2; }
command -v clang >/dev/null 2>&1 || { echo "claims: clang is required" >&2; exit 2; }
command -v python3 >/dev/null 2>&1 || { echo "claims: python3 is required for the statistics" >&2; exit 2; }

# The load probe and the record precondition are shared with the tide harness.
source bench/lib/load.sh

check_record_load claims "$RECORD" "$LOAD_PER_CPU_MAX"

echo "→ building qkc, the interpreter and the controls in $BUILD" >&2
(cd compiler && go build -o "$BUILD/qkc" .)
go build -o "$BUILD/quark" .
export QUARK_CACHE="$BUILD/qkc-cache"
compile_controls

qk_build() { # usage: qk_build <source> <output> <sed-expression>
  sed -E "$3" "$1" > "$BUILD/tmp.qk"
  grep -qE "$4" "$BUILD/tmp.qk" || {
    echo "claims: the expected line was not replaced in $1" >&2
    exit 1
  }
  "$BUILD/qkc" -c -o "$2" "$BUILD/tmp.qk" >/dev/null 2>"$BUILD/qkc.err" || {
    echo "claims: qkc failed on $1: $(cat "$BUILD/qkc.err")" >&2
    exit 1
  }
}

qk_build bench/claims/fib/fib.qk "$BUILD/fib30-qk" 's/^    int n = [0-9]+;$/    int n = 30;/' 'int n = 30;'
qk_build bench/claims/fib/fib.qk "$BUILD/fib35-qk" 's/^    int n = [0-9]+;$/    int n = 35;/' 'int n = 35;'
qk_build bench/claims/p99/p99.qk "$BUILD/p99-qk-p50" 's/^    int target = [0-9]+;$/    int target = 500;/' 'int target = 500;'
qk_build bench/claims/p99/p99.qk "$BUILD/p99-qk-p99" 's/^    int target = [0-9]+;$/    int target = 990;/' 'int target = 990;'
"$BUILD/qkc" -c -o "$BUILD/sum-qk" bench/claims/sum/sum.qk >/dev/null 2>"$BUILD/qkc.err" || {
  echo "claims: qkc failed on sum.qk: $(cat "$BUILD/qkc.err")" >&2
  exit 1
}
for name in fib30 loop list struct hash str; do
  "$BUILD/qkc" -c -o "$BUILD/back-$name" "bench/claims/backends/$name.qk" >/dev/null 2>"$BUILD/qkc.err" || {
    echo "claims: qkc failed on backends/$name.qk: $(cat "$BUILD/qkc.err")" >&2
    exit 1
  }
done

SKIPS=()
if command -v rustc >/dev/null 2>&1; then
  rustc -O bench/claims/fib/fib.rs -o "$BUILD/fib-rs"
  RUST_VERSION="$(rustc --version)"
else
  RUST_VERSION="skipped"
  SKIPS+=("rust: rustc not found")
fi
if command -v erl >/dev/null 2>&1 && command -v erlc >/dev/null 2>&1; then
  erlc -o "$BUILD" bench/claims/fib/fib.erl
  ERL_VERSION="$(erl -noshell -eval 'io:format("Erlang/OTP ~s", [erlang:system_info(otp_release)]), halt().' 2>/dev/null)"
else
  ERL_VERSION="skipped"
  SKIPS+=("erlang: erl not found")
fi
go build -o "$BUILD/fib-go" ./bench/claims/fib/go
go build -o "$BUILD/sum-go" ./bench/claims/sum/go

CLASS="$(uname -s) $(uname -m), $(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc) logical CPUs"
TOOLCHAIN="$(clang --version | head -1); $(go version); $RUST_VERSION; $ERL_VERSION"
STRICT="qkc default -O3 -flto=thin; C clang $CFLAGS_C; Rust rustc -O; Go go build (default); Erlang erlc (default)"
COUNTED="in-process wall time: clock()/clock_gettime(CLOCK_MONOTONIC)/Instant::now()/time.Now()/timer:tc(); process startup and printing excluded; every row prints its result as a checksum"
EVIDENCE="every workload prints its result as the checksum; the harness fails when the checksum changes between runs; the C, Rust and Go fib controls read the argument from argv so the optimizer cannot constant-fold the recursion"
if [ "${#SKIPS[@]}" -eq 0 ]; then
  SKIPPED="none"
else
  SKIPPED="$(IFS=';'; printf '%s' "${SKIPS[*]}")"
fi

ROWS="$BUILD/rows.tsv"
: > "$ROWS"
OUT="$BUILD/out.txt"
ERR="$BUILD/run.err"

# CHECKSUM is the one checksum every row in the current family must print.
CHECKSUM=""

measure() { # usage: measure <key> <label> <language> <rounds> <note> <command...>
  local key="$1" label="$2" language="$3" rounds="$4" note="$5"
  shift 5
  local values=() first second r
  if ! "$@" > /dev/null 2>"$ERR"; then
    echo "claims: $key warm-up failed: $(cat "$ERR")" >&2
    exit 1
  fi
  for ((r = 1; r <= RUNS; r++)); do
    if ! "$@" > "$OUT" 2>"$ERR"; then
      echo "claims: $key failed: $(cat "$ERR")" >&2
      exit 1
    fi
    first=""
    second=""
    {
      IFS= read -r first || true
      IFS= read -r second || true
    } < "$OUT"
    [[ "$first" =~ ^[0-9]+$ ]] || { echo "claims: $key printed no microsecond value: [$first]" >&2; exit 1; }
    [[ "$second" =~ ^-?[0-9]+$ ]] || { echo "claims: $key printed no integer checksum: [$second]" >&2; exit 1; }
    if [ -z "$CHECKSUM" ]; then
      CHECKSUM="$second"
    elif [ "$second" != "$CHECKSUM" ]; then
      echo "claims: $key checksum $second does not match $CHECKSUM in its family" >&2
      exit 1
    fi
    values+=("$first")
  done
  printf '%s\n' "${values[@]}" | python3 bench/stats.py \
    --key "$key" --label "$label" --language "$language" --rounds "$rounds" --runs "$RUNS" \
    --unit us --checksum "$CHECKSUM" --note "$note" >> "$ROWS"
}

# fib: the cross-language control the README compared against.
CHECKSUM=""
measure fib30_qk "fib(30), QuarkLang compiled" QuarkLang 1 "recursive fib(30), result printed as the checksum" "$BUILD/fib30-qk"
measure fib30_c "fib(30), C control" C 1 "recursive fib(30) with a volatile argument" "$BUILD/fib-c" 30
if command -v rustc >/dev/null 2>&1; then
  measure fib30_rust "fib(30), Rust control" Rust 1 "recursive fib(30), argument from argv" "$BUILD/fib-rs" 30
fi
measure fib30_go "fib(30), Go control" Go 1 "recursive fib(30), argument from argv" "$BUILD/fib-go" 30
if command -v erl >/dev/null 2>&1; then
  measure fib30_erl "fib(30), Erlang control" Erlang 1 "recursive fib(30) timed with timer:tc" erl -noshell -pa "$BUILD" -run fib main -s init stop
fi

CHECKSUM=""
measure fib35_qk "fib(35), QuarkLang compiled" QuarkLang 1 "recursive fib(35), result printed as the checksum" "$BUILD/fib35-qk"
measure fib35_c "fib(35), C control" C 1 "recursive fib(35) with a volatile argument" "$BUILD/fib-c" 35
if command -v rustc >/dev/null 2>&1; then
  measure fib35_rust "fib(35), Rust control" Rust 1 "recursive fib(35), argument from argv" "$BUILD/fib-rs" 35
fi
measure fib35_go "fib(35), Go control" Go 1 "recursive fib(35), argument from argv" "$BUILD/fib-go" 35

# Closed-form sum: the QuarkLang row is one O(1) call over 1e9 terms; the Go control is the loop.
CHECKSUM=""
measure sum_qk "sum over 1e9 terms, QuarkLang closed form" QuarkLang 1000000000 "one sum(ident, 0, 1e9) call; the elapsed time is below clock()'s 1 us resolution on this build" "$BUILD/sum-qk"
measure sum_go "sum over 1e9 terms, Go loop" Go 1000000000 "the same terms added by a plain Go loop" "$BUILD/sum-go"

# fib(20) latency: the percentile threshold found by binary search, not a total runtime.
CHECKSUM=""
measure p99_p50_qk "fib(20) P50, QuarkLang compiled" QuarkLang 1000 "P50 of 1000 fib(20) calls, found by counting and binary search" "$BUILD/p99-qk-p50"
measure p99_p50_c "fib(20) P50, C control" C 1000 "P50 of 1000 fib(20) calls, found by counting and binary search" "$BUILD/p99-c" 500 1000

CHECKSUM=""
measure p99_p99_qk "fib(20) P99, QuarkLang compiled" QuarkLang 1000 "P99 of 1000 fib(20) calls, found by counting and binary search" "$BUILD/p99-qk-p99"
measure p99_p99_c "fib(20) P99, C control" C 1000 "P99 of 1000 fib(20) calls, found by counting and binary search" "$BUILD/p99-c" 990 1000

# Interpreter versus native qkc over the same checked-in programs.
BACK_ROUNDS=(
  "fib30 1"
  "loop 10000000"
  "list 1000000"
  "struct 1000000"
  "hash 1000000"
  "str 50000"
)
for entry in "${BACK_ROUNDS[@]}"; do
  name="${entry%% *}"
  rounds="${entry##* }"
  CHECKSUM=""
  measure "back_${name}_qk" "$name, interpreter" "QuarkLang interpreter" "$rounds" "the same source run by the interpreter, timed in-process" "$BUILD/quark" "bench/claims/backends/$name.qk"
  measure "back_${name}_native" "$name, native qkc" "QuarkLang compiled" "$rounds" "the same source compiled by qkc, timed in-process" "$BUILD/back-$name"
done

print_record() {
  printf '# record\tclaims-v1\n'
  printf '# runs\t%s\n' "$RUNS"
  printf '# class\t%s\n' "$CLASS"
  if [ -n "${LOAD_META:-}" ]; then
    printf '# load\t%s\n' "$LOAD_META"
  fi
  printf '# toolchain\t%s\n' "$TOOLCHAIN"
  printf '# flags\t%s\n' "$STRICT"
  printf '# counted\t%s\n' "$COUNTED"
  printf '# skipped\t%s\n' "$SKIPPED"
  printf '# evidence\t%s\n' "$EVIDENCE"
  printf '# columns\tkind\tkey\tlabel\tlanguage\trounds\truns\tmedian\tunit\tmin\tmax\tstdev\tspread_pct\tchecksum\tnote\n'
  cat "$ROWS"
}

if [ "$RECORD" -eq 1 ]; then
  TMP="$BUILD/claims.tsv"
  print_record > "$TMP"
  mv "$TMP" spec/bench/claims.tsv
  go generate ./internal/benchgen
  go run ./cmd/qkbench -check -suite claims
  echo "→ recorded spec/bench/claims.tsv and regenerated docs/performance-claims.md" >&2
else
  print_record
fi
