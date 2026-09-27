#!/usr/bin/env bash
# Toolchain performance benchmark (real process-level timings, median).
#
#   scripts/bench-tools.sh [rounds]      # default: 7 rounds per item
#
# Covers: quark (startup/compute), qkc (IR generation), qkcheck, qkdoc, qkrepl, qkfmt,
#         qktest, qkm build; qklsp analysis latency is measured separately with go test -bench (see README).
set -euo pipefail
cd "$(dirname "$0")/.."

N="${1:-7}"
BIN="$(mktemp -d)"
trap 'rm -rf "$BIN"' EXIT
FIX="bench/tools"

echo "→ Building tools into $BIN (default optimization flags, matching a real user build)" >&2
go build -o "$BIN/quark" . >&2
go build -o "$BIN/qkcheck" ./cmd/qkcheck >&2
go build -o "$BIN/qkdoc" ./cmd/qkdoc >&2
go build -o "$BIN/qkrepl" ./cmd/qkrepl >&2
go build -o "$BIN/qklsp" ./cmd/qklsp >&2
(cd compiler && go build -o "$BIN/qkc" .) >&2

# External tools (qkfmt/qktest/qkm) live in sibling repositories; skip any that are missing
QKFMT=""; QKTEST=""; QKM=""
[ -d ../QuarkLangQkfmt ] && (cd ../QuarkLangQkfmt && go build -o "$BIN/qkfmt" .) >&2 && QKFMT="$BIN/qkfmt"
[ -d ../QuarkLangQktest ] && (cd ../QuarkLangQktest && go build -o "$BIN/qktest" .) >&2 && QKTEST="$BIN/qktest"
[ -d ../QuarkLangQkm ] && (cd ../QuarkLangQkm && go build -o "$BIN/qkm" .) >&2 && QKM="$BIN/qkm"

median_ms() { # usage: median_ms <command...>
  python3 - "$N" "$@" <<'PY'
import subprocess, sys, time, statistics, os
n = int(sys.argv[1]); cmd = sys.argv[2:]
times = []
env = dict(os.environ)
devnull = subprocess.DEVNULL
for _ in range(n):
    t0 = time.perf_counter()
    subprocess.run(cmd, stdout=devnull, stderr=devnull, env=env)
    times.append((time.perf_counter() - t0) * 1000)
print(f"{statistics.median(times):8.2f}")
PY
}

row() { printf '| %-26s | %s |\n' "$1" "$2"; }
echo
echo "| Scenario | Median ms |"
echo "|---|---|"
row "quark startup (empty program)"         "$(median_ms "$BIN/quark" "$FIX/empty.qk")"
row "quark fib(25)"                         "$(median_ms "$BIN/quark" "$FIX/fib.qk")"
row "quark 1M-iteration loop"               "$(median_ms "$BIN/quark" "$FIX/loop.qk")"
row "qkc IR generation (fib.qk)"            "$(median_ms "$BIN/qkc" "$FIX/fib.qk")"
row "qkcheck (style.qk, 1567 lines)"        "$(median_ms "$BIN/qkcheck" -exit0 "$FIX/style.qk")"
row "qkcheck --no-typecheck"                "$(median_ms "$BIN/qkcheck" -exit0 -no-typecheck "$FIX/style.qk")"
row "qkdoc (style.qk -> /dev/null)"         "$(median_ms "$BIN/qkdoc" -o /dev/null "$FIX/style.qk")"
row "qkrepl -e (one statement)"             "$(median_ms "$BIN/qkrepl" -e "1 + 1")"
row "qkrepl batch, 200 statements"          "$(median_ms bash -c "cat '$FIX/repl_stmts.txt' | '$BIN/qkrepl'")"

if [ -n "$QKFMT" ]; then
  row "qkfmt -l (style.qk)"                 "$(median_ms "$QKFMT" -l "$FIX/style.qk")"
fi
if [ -n "$QKTEST" ]; then
  row "qktest (examples in qktest repo)"    "$(median_ms env "QUARK=$BIN/quark" "$QKTEST" -L ../QuarkLangQktest ../QuarkLangQktest/examples)"
fi
if [ -n "$QKM" ]; then
  row "qkm build (small project)"           "$(median_ms bash -c "cd '$FIX/qkmproj' 2>/dev/null && QKM_QUARK='$BIN/quark' '$QKM' build")"
fi
echo
echo "(median of $N rounds per item; build artifacts live in a temporary directory and are removed on exit)"
