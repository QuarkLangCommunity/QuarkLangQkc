#!/bin/bash
# 解释器 / 编译器输出对比（同一份源码，逐字节）
#
# 用法：compiler/testdata/compare.sh [case.qk ...]
#   默认跑 compiler/testdata/cases/*.kq；先构建两个二进制：
#     /tmp/quark  ← 仓库根（解释器）
#     /tmp/qkc    ← compiler/（原生编译器）
# 任一用例 stdout+stderr 或退出码不一致 → 非零退出。
#
# 注意：每次清空 QUARK_CACHE，避免 IR/二进制缓存掩盖新的 codegen 行为。
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CACHE="$(mktemp -d)"
trap 'rm -rf "$CACHE"' EXIT

echo "构建解释器 /tmp/quark 与编译器 /tmp/qkc ..."
(cd "$ROOT" && go build -o /tmp/quark .) || exit 1
(cd "$ROOT/compiler" && go build -o /tmp/qkc .) || exit 1

cases=("$@")
if [ ${#cases[@]} -eq 0 ]; then
  cases=("$ROOT"/compiler/testdata/cases/*.kq)
fi

fail=0
for f in "${cases[@]}"; do
  a=$(/tmp/quark "$f" 2>&1); ra=$?
  rm -rf "$CACHE"; mkdir -p "$CACHE"
  b=$(QUARK_CACHE="$CACHE" /tmp/qkc -run "$f" 2>&1); rb=$?
  if [ "$a" == "$b" ] && [ $ra -eq $rb ]; then
    echo "OK   $(basename "$f")"
  else
    echo "DIFF $(basename "$f") (interp=$ra, comp=$rb)"
    diff <(printf '%s' "$a") <(printf '%s' "$b") | head -40
    fail=1
  fi
done
if [ $fail -eq 0 ]; then echo "全部一致"; else echo "存在不一致"; fi
exit $fail
