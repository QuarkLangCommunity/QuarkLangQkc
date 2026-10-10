package lang

import "fmt"

// The two depth limits of the implementation. Both are **explicit counters**, never a probe of the Go
// stack: a process stack is a platform-sized resource, so an implementation that lets it decide would
// answer differently on linux, darwin and windows. They live here, next to each other, because the
// interpreter and the compiler must agree on the number: the interpreter enforces them directly and the
// compiler bakes the same numbers into the code it generates.

// MaxExprDepth is the nesting depth the shared front end accepts in one expression. The type checker
// recurses once per expression node, so without this counter a long chain (a single source line of a
// few hundred thousand additions) exhausts the Go stack and takes the whole process down instead of
// producing a diagnostic. The limit is deliberately far above anything a human writes — the largest
// expression in the repository's stress corpus is ten thousand chained calls — and far below the depth
// at which the checker's own recursion (measured: ~320 000 nodes) runs the stack out, so refusing is
// always possible.
const MaxExprDepth = 65536

// MaxCallDepth is the number of nested function calls one program may make: a call made from a frame
// that is already this deep is refused. The interpreter counts it per call context (so it stays correct
// across taskm threads) and the compiler emits the same counter into the generated code, which is what
// makes the two engines report the same error at the same depth — including for a self tail call, which
// the compiler must count even though it needs no new native frame.
const MaxCallDepth = 8192

// MaxCallDepthMessage is the diagnostic both engines print when a call chain passes MaxCallDepth. The
// interpreter formats it at the refused call site and the compiler bakes it into the generated binary,
// so a single source for the wording is what keeps the two outputs byte-identical.
var MaxCallDepthMessage = fmt.Sprintf("StackOverflowError: recursion depth exceeded %d", MaxCallDepth)
