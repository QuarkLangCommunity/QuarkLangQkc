//go:build (linux || darwin) && cgo

package lang

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// callLibMethod 调用系统库导出函数（FFI：dlopen/dlsym + libffi，跨系统 POSIX/Windows）。
func (in *interp) callLibMethod(lib *libObj, name string, args []Value, pos Pos, ctx *execCtx) (Value, error) {
	fn := lib.methods[name]
	if fn == nil {
		return NilV(), &RunError{Msg: fmt.Sprintf("LibraryError: 库 %s 没有声明函数 %q", lib.name, name), Pos: pos, Ctx: ctx}
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
	// 参数打包：int/bool/float→数值通道；String→char*（调用后释放）
	types := make([]int, 0, len(fn.Params))
	nums := make([]float64, 0, len(fn.Params))
	ptrs := make([]unsafe.Pointer, 0, len(fn.Params))
	var cstrs []*C.char
	for i, p := range fn.Params {
		if i >= len(args) {
			return NilV(), &RunError{Msg: fmt.Sprintf("LibraryError: %s 参数不足", name), Pos: pos, Ctx: ctx}
		}
		a := args[i]
		if p.Type == "pointer" { // 不透明句柄：接受指针值或 null（void*），可往返
			if a.IsNil() {
				types = append(types, ffiPtr)
				nums = append(nums, 0)
				ptrs = append(ptrs, nil)
				continue
			}
			if !a.IsPtr() {
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: %s 参数 %s 需要 pointer 或 null", name, p.Name), Pos: pos, Ctx: ctx}
			}
			types = append(types, ffiPtr)
			nums = append(nums, 0)
			ptrs = append(ptrs, a.Ptr())
			continue
		}
		switch ffiTypeOf(p.Type) {
		case ffiInt, ffiBool, ffiLong:
			if !a.IsInt() && !a.IsBool() {
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: %s 参数 %s 需要 int/bool", name, p.Name), Pos: pos, Ctx: ctx}
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
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: %s 参数 %s 需要 float", name, p.Name), Pos: pos, Ctx: ctx}
			}
			if a.IsInt() {
				nums = append(nums, float64(a.Int()))
			} else {
				nums = append(nums, a.Float())
			}
			types = append(types, ffiTypeOf(p.Type))
			ptrs = append(ptrs, nil)
		default: // String / 指针
			if !a.IsStr() {
				return NilV(), &RunError{Msg: fmt.Sprintf("TypeError: %s 参数 %s 目前仅支持 String/int/float", name, p.Name), Pos: pos, Ctx: ctx}
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
		return PtrV(rp), nil // 不透明句柄：返回原生指针值（nil → null）
	case "String":
		if rp == nil {
			return NilV(), nil
		}
		return StrV(C.GoString((*C.char)(rp))), nil
	default:
		// 其它指针返回仍以十六进制字符串呈现（声明为 pointer 才得到原生指针值）
		if rp == nil {
			return NilV(), nil
		}
		return StrV(fmt.Sprintf("0x%x", rp)), nil
	}
}
