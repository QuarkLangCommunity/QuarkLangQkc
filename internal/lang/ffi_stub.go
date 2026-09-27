//go:build !cgo

package lang

import (
	"fmt"
	"quarklang/internal/i18n"
)

// no-cgo build (e.g. macOS cross output): FFI is unavailable, return an explicit error.
func (in *interp) callLibMethod(lib *libObj, name string, args []Value, pos Pos, ctx *execCtx) (Value, error) {
	return NilV(), &RunError{Msg: fmt.Sprintf(i18n.T("FFIError: 本构建未启用 cgo（FFI/系统库不可用）")), Pos: pos, Ctx: ctx}
}
