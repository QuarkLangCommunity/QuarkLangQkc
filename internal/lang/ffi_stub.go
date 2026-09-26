//go:build !cgo

package lang

import (
	"fmt"
	"quarklang/internal/i18n"
)

// 无 cgo 构建（如 macOS 交叉帧输出）：FFI 不可用，返回明确错误。
func (in *interp) callLibMethod(lib *libObj, name string, args []Value, pos Pos, ctx *execCtx) (Value, error) {
	return NilV(), &RunError{Msg: fmt.Sprintf(i18n.T("FFIError: 本构建未启用 cgo（FFI/系统库不可用）")), Pos: pos, Ctx: ctx}
}
