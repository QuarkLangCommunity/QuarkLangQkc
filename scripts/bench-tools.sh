#!/usr/bin/env bash
# 工具链性能基准（进程级真实耗时，取中位数）。
#
#   scripts/bench-tools.sh [轮数]        # 默认每项 7 轮
#
# 覆盖：quark（启动/计算）、qkc（IR 生成）、qkcheck、qkdoc、qkrepl、qkfmt、
#       qktest、qkm build；qklsp 的分析延迟用 go test -bench 单独测（见 README）。
set -euo pipefail
cd "$(dirname "$0")/.."

N="${1:-7}"
BIN="$(mktemp -d)"
trap 'rm -rf "$BIN"' EXIT
FIX="bench/tools"

echo "→ 构建工具到 $BIN（默认优化参数，与用户实际构建一致）" >&2
go build -o "$BIN/quark" . >&2
go build -o "$BIN/qkcheck" ./cmd/qkcheck >&2
go build -o "$BIN/qkdoc" ./cmd/qkdoc >&2
go build -o "$BIN/qkrepl" ./cmd/qkrepl >&2
go build -o "$BIN/qklsp" ./cmd/qklsp >&2
(cd compiler && go build -o "$BIN/qkc" .) >&2

# 外部工具（qkfmt/qktest/qkm）在同级仓库里；缺失则跳过对应项
QKFMT=""; QKTEST=""; QKM=""
[ -d ../QuarkLangQkfmt ] && (cd ../QuarkLangQkfmt && go build -o "$BIN/qkfmt" .) >&2 && QKFMT="$BIN/qkfmt"
[ -d ../QuarkLangQktest ] && (cd ../QuarkLangQktest && go build -o "$BIN/qktest" .) >&2 && QKTEST="$BIN/qktest"
[ -d ../QuarkLangQkm ] && (cd ../QuarkLangQkm && go build -o "$BIN/qkm" .) >&2 && QKM="$BIN/qkm"

median_ms() { # 用法: median_ms <命令...>
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
echo "| 场景 | 中位耗时 ms |"
echo "|---|---|"
row "quark 启动（空程序）"        "$(median_ms "$BIN/quark" "$FIX/empty.qk")"
row "quark fib(25)"               "$(median_ms "$BIN/quark" "$FIX/fib.qk")"
row "quark 100 万次循环"          "$(median_ms "$BIN/quark" "$FIX/loop.qk")"
row "qkc IR 生成（fib.qk）"       "$(median_ms "$BIN/qkc" "$FIX/fib.qk")"
row "qkcheck（style.qk 1567 行）" "$(median_ms "$BIN/qkcheck" -exit0 "$FIX/style.qk")"
row "qkcheck --no-typecheck"      "$(median_ms "$BIN/qkcheck" -exit0 -no-typecheck "$FIX/style.qk")"
row "qkdoc（style.qk → /dev/null）" "$(median_ms "$BIN/qkdoc" -o /dev/null "$FIX/style.qk")"
row "qkrepl -e（单条）"           "$(median_ms "$BIN/qkrepl" -e "1 + 1")"
row "qkrepl 批处理 200 条语句"     "$(median_ms bash -c "cat '$FIX/repl_stmts.txt' | '$BIN/qkrepl'")"

if [ -n "$QKFMT" ]; then
  row "qkfmt -l（style.qk）"      "$(median_ms "$QKFMT" -l "$FIX/style.qk")"
fi
if [ -n "$QKTEST" ]; then
  row "qktest（qktest 仓 examples）" "$(median_ms env "QUARK=$BIN/quark" "$QKTEST" -L ../QuarkLangQktest ../QuarkLangQktest/examples)"
fi
if [ -n "$QKM" ]; then
  row "qkm build（小工程）"       "$(median_ms bash -c "cd '$FIX/qkmproj' 2>/dev/null && QKM_QUARK='$BIN/quark' '$QKM' build")"
fi
echo
echo "（每项取 $N 轮中位数；构建产物在临时目录，退出即清理）"
