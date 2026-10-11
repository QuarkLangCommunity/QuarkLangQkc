//go:build !cgo

package lang

import (
	"fmt"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n"
)

// no-cgo build (e.g. macOS cross output): FFI is unavailable, return an explicit error.
func (in *interp) callLibMethod(lib *libObj, name string, args []Value, pos Pos, ctx *execCtx) (Value, error) {
	return NilV(), &RunError{Msg: fmt.Sprintf(msgID(i18n.MsgRuntimeFfierrorBuildCgoDisabledFfiSystem)), Pos: pos, Ctx: ctx}
}
