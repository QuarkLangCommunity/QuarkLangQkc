package lang

import (
	"sync"
	"unsafe"
)

// libObj is the runtime library binding object (shared across platforms): library name + handle (lazy-loaded) + exported symbol signature table.
type libObj struct {
	name    string
	lib     string // system library name (dlopen: "libGL.so.1" / "opengl32")
	handle  *libHandle
	methods map[string]*Func
}

// libHandle is a handle to a runtime-loaded system library (cross-platform: dlopen/LoadLibrary).
type libHandle struct {
	h   unsafe.Pointer
	mu  sync.Mutex
	err string
}
