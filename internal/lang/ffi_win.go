//go:build windows && cgo

package lang

/*
#cgo windows LDFLAGS: -luser32
#include "ffi_win.inc"
*/
import "C"

import (
	"errors"
	"strings"
	"unsafe"
)

// Windows FFI：LoadLibrary/GetProcAddress + 自研 ABI wrapper（4 型 x <=4 参数组合）。

const (
	ffiVoid = 0
	ffiInt  = 1
	ffiF32  = 2
	ffiF64  = 3
	ffiPtr  = 4
	ffiBool = 5
	ffiLong = 6
)

func dlopenLib(name string) (*libHandle, error) {
	cands := []string{name}
	if !strings.ContainsAny(name, "/.") && !strings.HasPrefix(name, "lib") {
		cands = append(cands, name+".dll", "lib"+name+".dll")
	}
	var lastErr string
	for _, c := range cands {
		cc := C.CString(c)
		h := C.qk_dlopen(cc)
		C.free(unsafe.Pointer(cc))
		if h != nil {
			return &libHandle{h: h}, nil
		}
		lastErr = "library not found"
	}
	return nil, errors.New("FFILibraryError: cannot load " + name + " (" + lastErr + ")")
}

func (lh *libHandle) resolve(sym string) (unsafe.Pointer, error) {
	cc := C.CString(sym)
	defer C.free(unsafe.Pointer(cc))
	fn := C.qk_dlsym(lh.h, cc)
	if fn == nil {
		return nil, errors.New("FFISymbolError: symbol not found: " + sym)
	}
	return fn, nil
}

func ffiCall(fn unsafe.Pointer, paramTypes []int, nums []float64, ptrs []unsafe.Pointer, retType int) (int64, float64, unsafe.Pointer, error) {
	n := len(paramTypes)
	if n > 4 {
		return 0, 0, nil, errors.New("FFIError: windows 自研 ABI 支持 <=4 参数")
	}
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
	var p0 [1]unsafe.Pointer
	if n == 0 {
		ctypes = t0[:]
		cnums = d0[:]
		cptrs = p0[:]
	}
	var (
		retI     int64
		retF     float64
		retPSlot [1]uintptr
	)
	rc := C.qk_ffi(fn, &ctypes[0], &cnums[0], &cptrs[0], C.int(n), C.int(retType),
		(*C.int64_t)(unsafe.Pointer(&retI)), (*C.double)(unsafe.Pointer(&retF)),
		unsafe.Pointer(&retPSlot[0]))
	if rc != 0 {
		return 0, 0, nil, errors.New("FFIError: ffi call failed rc=" + itoa(int(rc)))
	}
	return retI, retF, unsafe.Pointer(retPSlot[0]), nil
}

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
		return ffiPtr
	}
}
