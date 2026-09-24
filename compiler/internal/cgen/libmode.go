package cgen

// libmode.go —— **库模式**：编译库制品（--emit-lib）时豁免 main 入口要求。
//
// 为什么放在语言前端：库是"语言本体的正常产物"，不该靠塞一个假 main 来绕过检查
// （假 main 会与使用方的 main 冲突，也会污染导出符号表）。

var libMode bool

// SetLibMode 开关库模式（由 qkc --emit-lib 设置）
func SetLibMode(v bool) { libMode = v }

// InLibMode 当前是否库模式
func InLibMode() bool { return libMode }

// forceHelpers：与库链接时，宿主程序**必须提供全部运行时辅助函数**。
// 原因：链接共享库后 LTO 无法再丢弃未用的运行时代码（如 ql_any_str_int），
// 它们对 ql_int_to_str/ql_float_to_str 的引用需要由宿主程序的定义来满足。
var forceHelpers bool

// SetForceRuntimeHelpers 开关"强制发射全部运行时辅助函数"
func SetForceRuntimeHelpers(v bool) { forceHelpers = v }
