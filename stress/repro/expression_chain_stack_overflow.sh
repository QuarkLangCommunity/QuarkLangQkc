#!/bin/sh
# The small input behind the front-end stack overflow on a long expression chain (fixed regression).
#
# A left-deep chain of additions in one expression is ~300 000 terms at the size that used to kill the
# process, so the input is materialised here instead of being committed as a ~600 KB source file. Both
# engines share the front end, and both used to die the same way, because the type checker recursed once
# per binary node (internal/lang/typecheck.go: (*checker).inferBin -> infer):
#
#   quark    <generated file>   -> exit 2, "runtime: goroutine stack exceeds 1000000000-byte limit"
#   qkc -run <generated file>   -> exit 2, the same crash while generating IR
#
# Since the shared depth limits landed, both engines refuse the chain with a source position instead:
#
#   error: CompileError: expression nesting is deeper than 65536 levels, the limit of this
#          implementation: split the expression into smaller statements at line 1
#
# with exit status 1 from both, and identical bytes on stdout and stderr. Measured: 290 000 terms used to
# survive the front end and 320 000 used to crash it; the limit is 65 536, so the refusal happens while
# there is still a 4.5x stack margin, and the whole run costs ~0.2 s.
#
# Usage: stress/repro/expression_chain_stack_overflow.sh [terms] > chain.kq
set -eu
terms=${1:-320000}
awk -v n="$terms" 'BEGIN { printf "fn main(IOStream io) { io.println(0"; for (i = 0; i < n; i++) printf "+1"; print "); }" }'
