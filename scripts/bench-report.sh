#!/usr/bin/env bash
# Interpreter vs native (qkc) benchmark report over bench/qk/*.kq.
#
#   scripts/bench-report.sh [rounds]        # default: 3 rounds per side, median reported
#
# Both backends run the same .kq source and must print the identical line: parity is asserted here,
# not assumed (compiler/testdata/compare.sh covers semantics; this covers the programs people quote).
# A program whose two backends disagree is reported as a failure, not as a result.
set -euo pipefail
cd "$(dirname "$0")/.."

N="${1:-3}"
BIN="$(mktemp -d)"
trap 'rm -rf "$BIN"' EXIT

command -v clang >/dev/null 2>&1 || { echo 'error: clang is required (qkc lowers to LLVM IR)' >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo 'error: python3 is required (median timing)' >&2; exit 1; }

echo '→ Building interpreter and qkc (default flags)' >&2
go build -o "$BIN/quark" .
(cd compiler && go build -o "$BIN/qkc" .)

median_ms() { # usage: median_ms <command...>
  python3 - "$N" "$@" <<'PY'
import subprocess, sys, time, statistics
n = int(sys.argv[1]); cmd = sys.argv[2:]
times = []
for _ in range(n):
    t0 = time.perf_counter()
    subprocess.run(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    times.append((time.perf_counter() - t0) * 1000)
print(f"{statistics.median(times):.2f}")
PY
}

echo
echo '| Program | What it does | Interpreter (ms) | Native qkc (ms) | Speedup |'
echo '|---|---|---|---|---|'
failed=0
total=0
for src in bench/qk/*.qk; do
  total=$((total + 1))
  name="$(basename "$src" .qk)"
  bin="$BIN/$name"
  if ! "$BIN/qkc" -c -o "$bin" "$src" >/dev/null 2>"$BIN/$name.err"; then
    echo "| $name | **compile failed** | — | — | — |"
    sed 's/^/    /' "$BIN/$name.err" >&2
    failed=1
    continue
  fi
  qk_out="$(timeout 600 "$BIN/quark" "$src" 2>&1)" || true
  nat_out="$(timeout 600 "$bin" 2>&1)" || true
  if [ "$qk_out" != "$nat_out" ]; then
    echo "| $name | **PARITY FAILURE** interpreter=[$qk_out] native=[$nat_out] | — | — | — |"
    failed=1
    continue
  fi
  desc="$(head -1 "$src" | sed 's|^// *||')"
  t_qk="$(median_ms "$BIN/quark" "$src")"
  t_nat="$(median_ms "$bin")"
  speed="$(python3 -c "print(f'{$t_qk/max($t_nat,0.001):.0f}x')")"
  printf '| %s | %s | %s | %s | %s |\n' "$name" "$desc" "$t_qk" "$t_nat" "$speed"
done
echo
if [ "$failed" -eq 0 ]; then
  echo "output parity: interpreter == native on all $total programs" >&2
else
  echo 'output parity FAILED' >&2
  exit 1
fi
