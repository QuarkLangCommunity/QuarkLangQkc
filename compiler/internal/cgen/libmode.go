package cgen

// libmode.go — **lib mode**: compiling a library artifact (--emit-lib) exempts it from the main entry requirement.
//
// Why this lives in the language frontend: a library is "a normal product of the language proper", and the check should not be bypassed by stuffing in a fake main
// (a fake main would clash with the consumer's main and also pollute the exported symbol table).

var libMode bool

// SetLibMode toggles lib mode (set by qkc --emit-lib)
func SetLibMode(v bool) { libMode = v }

// InLibMode reports whether lib mode is currently on
func InLibMode() bool { return libMode }

// forceHelpers: when linking against a library the host program **must provide every runtime helper**.
// Reason: once a shared library is linked, LTO can no longer drop unused runtime code (such as ql_any_str_int),
// and their references to ql_int_to_str/ql_float_to_str must be satisfied by host-program definitions.
var forceHelpers bool

// SetForceRuntimeHelpers toggles "force-emit all runtime helpers"
func SetForceRuntimeHelpers(v bool) { forceHelpers = v }
