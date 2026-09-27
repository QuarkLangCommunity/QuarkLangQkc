#!/bin/bash
# Interpreter vs compiler output comparison (same source, byte for byte)
#
# Usage: compiler/testdata/compare.sh [case.qk ...]
#   By default it runs compiler/testdata/cases/*.kq; build both binaries first:
#     /tmp/quark  <- repository root (interpreter)
#     /tmp/qkc    <- compiler/ (native compiler)
# Any case whose stdout+stderr or exit code differs -> non-zero exit.
#
# Note: QUARK_CACHE is cleared every run so IR/binary caches cannot mask new codegen behaviour.
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CACHE="$(mktemp -d)"
trap 'rm -rf "$CACHE"' EXIT

echo "building interpreter /tmp/quark and compiler /tmp/qkc ..."
(cd "$ROOT" && go build -o /tmp/quark .) || exit 1
(cd "$ROOT/compiler" && go build -o /tmp/qkc .) || exit 1

cases=("$@")
if [ ${#cases[@]} -eq 0 ]; then
  # cases/: runnable with llvm-as + lli (go test covers them too); cases_run/: needs clang linking (FFI/taskm)
  cases=("$ROOT"/compiler/testdata/cases/*.kq "$ROOT"/compiler/testdata/cases_run/*.kq)
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
if [ $fail -eq 0 ]; then echo "all identical (interpreter == compiler)"; else echo "DIVERGENCE FOUND"; fi
exit $fail
