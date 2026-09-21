package lang

import (
	"sync"
	"unsafe"
)

// libObj 是运行时库绑定对象（跨系统共用）：库名 + 句柄（懒加载）+ 导出符号签名表。
type libObj struct {
	name    string
	lib     string // 系统库名（dlopen："libGL.so.1" / "opengl32"）
	handle  *libHandle
	methods map[string]*Func
}

// libHandle 是运行时加载的系统库句柄（跨系统：dlopen/LoadLibrary）。
type libHandle struct {
	h   unsafe.Pointer
	mu  sync.Mutex
	err string
}
