//go:build (linux || darwin) && cgo

package lang

/*
#cgo linux LDFLAGS: -ldl -lffi
#cgo darwin LDFLAGS: -ldl -lffi
#include <stdlib.h>
#include <stdint.h>
void* qk_dlopen(const char* n);
void* qk_dlsym(void* h, const char* s);
const char* qk_dlerror(void);
int qk_ffi(void* fn, const int* types, const double* nums, void** ptrs,
           int n, int rettype, int64_t* ret_i, double* ret_f, void* ret_p_slot);
*/
import "C"

import (
	"errors"
	"strings"
	"unsafe"
)

// FFI type codes (matching ffi_shim.c).
const (
	ffiVoid = 0
	ffiInt  = 1 // int32
	ffiF32  = 2 // float (single precision)
	ffiF64  = 3 // double
	ffiPtr  = 4 // pointer / string
	ffiBool = 5
	ffiLong = 6 // int64
)

// resolve looks up an exported symbol.
func (lh *libHandle) resolve(sym string) (unsafe.Pointer, error) {
	// serialize symbol lookup (dlsym has its own lock, but keep it consistent)
	cname := C.CString(sym)
	defer C.free(unsafe.Pointer(cname))
	fn := C.qk_dlsym(lh.h, cname)
	if fn == nil {
		return nil, errors.New("FFISymbolError: symbol not found: " + sym)
	}
	return fn, nil
}

// ffiCall calls an imported function following the declared signature (cross-platform: POSIX/Windows both go through libffi with FFI_DEFAULT_ABI).
func ffiCall(fn unsafe.Pointer, paramTypes []int, nums []float64, ptrs []unsafe.Pointer, retType int) (int64, float64, unsafe.Pointer, error) {
	n := len(paramTypes)
	if n > 32 {
		return 0, 0, nil, errors.New("FFIError: too many args (>32)")
	}
	var (
		retI     int64
		retF     float64
		retPSlot [1]uintptr // pointer return slot (8 bytes, written by the C side)
	)
	ctypes := make([]C.int, n)
	cnums := make([]C.double, n)
	cptrs := make([]unsafe.Pointer, n)
	for i := 0; i < n; i++ {
		ctypes[i] = C.int(paramTypes[i])
		cnums[i] = C.double(nums[i])
		cptrs[i] = ptrs[i]
	}
	var t0 [1]C.int
	var d0 [1]C.double
	var pp0 [1]unsafe.Pointer
	if len(ctypes) == 0 {
		ctypes = t0[:]
		cnums = d0[:]
		cptrs = pp0[:]
	}
	rc := C.qk_ffi(fn, &ctypes[0], &cnums[0], &cptrs[0], C.int(n), C.int(retType),
		(*C.int64_t)(unsafe.Pointer(&retI)), (*C.double)(unsafe.Pointer(&retF)),
		unsafe.Pointer(&retPSlot[0]))
	if rc != 0 {
		return 0, 0, nil, errors.New("FFIError: ffi call failed rc=" + itoa(int(rc)))
	}
	return retI, retF, unsafe.Pointer(retPSlot[0]), nil
}

// dlopenLib loads a system library (cross-platform name resolution: original name/local -> libX.so.N/.so/.dylib/.dll, including case variants).
func dlopenLib(name string) (*libHandle, error) {
	cands := []string{name, "./" + name}
	if !strings.ContainsAny(name, "/.") &&
		!strings.HasSuffix(name, ".dll") && !strings.HasSuffix(name, ".dylib") {
		// bare short name: try the platform fill-ins; case variants (e.g. gl -> libGL.so.1)
		variants := []string{name}
		if up := strings.ToUpper(name[:1]) + name[1:]; up != name {
			variants = append(variants, up)
		}
		if all := strings.ToUpper(name); all != name {
			variants = append(variants, all)
		}
		for _, v := range variants {
			bases := []string{"lib" + v}
			if strings.HasPrefix(v, "lib") {
				bases = append(bases, v) // already carries the lib prefix: also try name.so.N directly (e.g. libc -> libc.so.6)
			}
			for _, bn := range bases {
				cands = append(cands, "./"+bn+".so") // project-local (build tree/assets) first
				for _, ver := range []string{"", ".0", ".1", ".2", ".3", ".4", ".5", ".6"} {
					cands = append(cands, bn+".so"+ver)
				}
				cands = append(cands, bn+".dylib", bn+".dll")
			}
		}
	}
	var lastErr string
	for _, c := range cands {
		cc := C.CString(c)
		h := C.qk_dlopen(cc)
		C.free(unsafe.Pointer(cc))
		if h != nil {
			return &libHandle{h: h}, nil
		}
		if msg := C.GoString(C.qk_dlerror()); msg != "" && msg != lastErr {
			lastErr = msg
		}
	}
	if lastErr == "" {
		lastErr = "library not found"
	}
	return nil, errors.New("FFILibraryError: cannot load " + name + " (" + lastErr + ")")
}

// ffiTypeOf maps a language type string to an FFI type code (int/long/char/bool -> i32 or i64 depending on the declaration; f32 is dedicated single precision).
func ffiTypeOf(typ string) int {
	switch {
	case typ == "f32":
		return ffiF32
	case typ == "float" || typ == "double":
		return ffiF64
	case typ == "int" || typ == "char" || typ == "bool":
		return ffiInt
	case typ == "long":
		return ffiLong
	default:
		return ffiPtr // String / pointer / everything else
	}
}
