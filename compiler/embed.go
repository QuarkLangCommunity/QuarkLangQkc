package main

import _ "embed"

// The C runtime, written next to the generated IR and compiled by clang into every program qkc
// builds. The message catalog is a second embedded file rather than part of this one so the generator
// owns it end to end: cmd/qki18n writes compiler/internal/rtcat/msg_c.h from the authored tables, and
// this file is prepended verbatim at compile time. A template edited in the runtime source instead of
// in internal/i18n/source would therefore be reported by the staleness gate, not silently shipped.

//go:embed qthreads.c.txt
var qthreadsC string

//go:embed internal/rtcat/msg_c.h
var runtimeCatalogC string

// runtimeSource is the whole translation unit: the generated catalog first, because the runtime's error
// sites call into it, then the runtime itself.
func runtimeSource() string { return runtimeCatalogC + "\n" + qthreadsC }
