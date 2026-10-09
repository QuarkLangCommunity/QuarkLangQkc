#!/bin/sh
# Minimal reproduction of the front-end stack overflow on a long expression chain.
#
# The smallest input found is a left-deep chain of additions in one expression; it is ~300 000 terms, so
# the reproduction is materialised here instead of being committed as a ~600 KB source file. Both engines
# share the front end and both die the same way, because the type checker recurses once per binary node
# (internal/lang/typecheck.go: (*checker).inferBin -> infer):
#
#   quark    <generated file>   -> exit 2, "runtime: goroutine stack exceeds 1000000000-byte limit"
#   qkc -run <generated file>   -> exit 2, the same crash while generating IR
#
# Usage: stress/repro/expression_chain_stack_overflow.sh [terms] > chain.kq
# Measured on the machine in stress/REPORT.md: 320 000 terms crash in ~0.7 s, 160 000 terms do not crash
# but did not finish the front end within 30 s, 80 000 terms finish in well under a second.
set -eu
terms=${1:-320000}
awk -v n="$terms" 'BEGIN { printf "fn main(IOStream io) { io.println(0"; for (i = 0; i < n; i++) printf "+1"; print "); }" }'
