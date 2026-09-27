//go:build cgo

package lang

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"quarklang/internal/i18n"
	"unsafe"
)

// callLibMethod calls an exported function of a system library (FFI: dlopen/dlsym + libffi, cross-platform POSIX/Windows).
func (in *interp) callLibMethod(lib *libObj, name string, args []Value, pos Pos, ctx *execCtx) (Value, error) {
	fn := lib.methods[name]
	if fn == nil {
		return NilV(), &RunError{Msg: fmt.Sprintf(i18n.T("LibraryError: 库 %s 没有声明函数 %q"), lib.name, name), Pos: pos, Ctx: ctx}
	}
	if lib.handle == nil {
		h, err := dlopenLib(lib.lib)
		if err != nil {
			return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
		}
		lib.handle = h
	}
	sym, err := lib.handle.resolve(name)
	if err != nil {
		return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
	}
	// This file only does the generic "Value <-> C argument" marshalling (C.CString/C.GoString are portable cgo);
	// the actual ABI call is provided by each platform's ffiCall (POSIX: libffi; Windows: a hand-written wrapper).
	//
	// Argument marshalling: int/bool/float -> numeric channel; String -> char* (freed after the call)
	types := make([]int, 0, len(fn.Params))
	nums := make([]float64, 0, len(fn.Params))
	ptrs := make([]unsafe.Pointer, 0, len(fn.Params))
	var cstrs []*C.char
	for i, p := range fn.Params {
		if i >= len(args) {
			return NilV(), &RunError{Msg: fmt.Sprintf(i18n.T("LibraryError: %s 参数不足"), name), Pos: pos, Ctx: ctx}
		}
		a := args[i]
		if p.Type == "pointer" { // opaque handle: accepts a pointer value or null (void*), round-trippable
			if a.IsNil() {
				types = append(types, ffiPtr)
				nums = append(nums, 0)
				ptrs = append(ptrs, nil)
				continue
			}
			if !a.IsPtr() {
				return NilV(), &RunError{Msg: fmt.Sprintf(i18n.T("TypeError: %s 参数 %s 需要 pointer 或 null"), name, p.Name), Pos: pos, Ctx: ctx}
			}
			types = append(types, ffiPtr)
			nums = append(nums, 0)
			ptrs = append(ptrs, a.Ptr())
			continue
		}
		switch ffiTypeOf(p.Type) {
		case ffiInt, ffiBool, ffiLong:
			if !a.IsInt() && !a.IsBool() {
				return NilV(), &RunError{Msg: fmt.Sprintf(i18n.T("TypeError: %s 参数 %s 需要 int/bool"), name, p.Name), Pos: pos, Ctx: ctx}
			}
			if a.IsBool() {
				if a.Bool() {
					nums = append(nums, 1)
				} else {
					nums = append(nums, 0)
				}
			} else {
				nums = append(nums, float64(a.Int()))
			}
			pt := ffiTypeOf(p.Type)
			if pt == ffiBool {
				pt = ffiInt
			}
			types = append(types, pt)
			ptrs = append(ptrs, nil)
		case ffiF32, ffiF64:
			if !a.IsFloat() && !a.IsInt() {
				return NilV(), &RunError{Msg: fmt.Sprintf(i18n.T("TypeError: %s 参数 %s 需要 float"), name, p.Name), Pos: pos, Ctx: ctx}
			}
			if a.IsInt() {
				nums = append(nums, float64(a.Int()))
			} else {
				nums = append(nums, a.Float())
			}
			types = append(types, ffiTypeOf(p.Type))
			ptrs = append(ptrs, nil)
		default: // String / pointer
			if !a.IsStr() {
				return NilV(), &RunError{Msg: fmt.Sprintf(i18n.T("TypeError: %s 参数 %s 目前仅支持 String/int/float"), name, p.Name), Pos: pos, Ctx: ctx}
			}
			cs := C.CString(a.Str())
			cstrs = append(cstrs, cs)
			types = append(types, ffiPtr)
			nums = append(nums, 0)
			ptrs = append(ptrs, unsafe.Pointer(cs))
		}
	}
	retType := ffiVoid
	switch fn.Ret {
	case "int", "char", "bool":
		retType = ffiInt
	case "long":
		retType = ffiLong
	case "f32":
		retType = ffiF32
	case "float", "double":
		retType = ffiF64
	case "String", "void", "":
		// void / String
	default:
		retType = ffiPtr
	}
	ri, rf, rp, err := ffiCall(sym, types, nums, ptrs, retType)
	for _, cs := range cstrs {
		C.free(unsafe.Pointer(cs))
	}
	if err != nil {
		return NilV(), &RunError{Msg: err.Error(), Pos: pos, Ctx: ctx}
	}
	switch fn.Ret {
	case "void", "":
		return NilV(), nil
	case "int", "char":
		return IntV(int64(int32(ri))), nil
	case "long":
		return IntV(ri), nil
	case "bool":
		return BoolV(ri != 0), nil
	case "f32", "float", "double":
		return FloatV(rf), nil
	case "pointer":
		return PtrV(rp), nil // opaque handle: returns the native pointer value (nil -> null)
	case "String":
		if rp == nil {
			return NilV(), nil
		}
		return StrV(C.GoString((*C.char)(rp))), nil
	default:
		// other pointer returns are still rendered as a hex string (only a `pointer` declaration yields a native pointer value)
		if rp == nil {
			return NilV(), nil
		}
		return StrV(fmt.Sprintf("0x%x", rp)), nil
	}
}
