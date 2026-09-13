package cgen

// 泛型单态化（Phase C）：按调用点的类型实参实例化泛型函数 / 泛型 struct。

import (
	"quarklang/internal/lang"
)

// callGeneric 实例化并调用泛型函数 fn<T,...>（按实参推断类型参数）。
func (fc *funcCtx) callGeneric(name string, c *lang.CallExpr, pos lang.Pos) (*expr, error) {
	return nil, fc.l.errf(pos, "暂未支持泛型实例化 %s（解释器可用）", name)
}
