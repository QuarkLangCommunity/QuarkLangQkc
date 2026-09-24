package cgen

import "strings"

// linkedlibs.go —— 由 qkc -L 提供的库（对象文件已参与链接）：
// 这些 `library <name>` 声明**不应**再生成 -l<name> 链接参数，否则链接器会去找不存在的库。

var objectProvidedLibs = map[string]bool{}

// SetObjectProvidedLibs 登记"由对象文件提供"的库名（小写比较）
func SetObjectProvidedLibs(names []string) {
	for _, n := range names {
		objectProvidedLibs[strings.ToLower(n)] = true
	}
}

// isObjectProvidedLib 该库是否由对象文件提供
func isObjectProvidedLib(name string) bool { return objectProvidedLibs[strings.ToLower(name)] }
