// Package i18n 提供用户可见文案的多语言支持（默认中文；QK_LANG / LC_ALL / LANG 决定语言）。
//
// 用法：把中文模板交给 T 即可——表里有英文就用英文渲染，没有则原样用中文（**绝不丢信息**）：
//
//	&RunError{Msg: i18n.T("类型 %s 不能赋给 %s", a, b)}
//
// 覆盖率由 internal/i18n 的 TestCoverage 守住：源码里出现的中文模板必须在表中登记。
package i18n

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// 语言标识：目前支持 "zh"（默认）与 "en"。
const (
	ZH = "zh"
	EN = "en"
)

var locale = detect()

// detect 决定初始语言：**只看 QK_LANG**，默认中文。
// 刻意不读 LC_ALL/LANG：CI（macOS/Windows runner）常设英文 locale，若据此切换会让
// 「默认中文」的承诺在开发者机器与 CI 之间漂移，也会让断言中文输出的测试无故失败。
// 需要英文请显式设置 QK_LANG=en 或使用 --lang en。
func detect() string {
	if strings.ToLower(strings.TrimSpace(os.Getenv("QK_LANG"))) == EN {
		return EN
	}
	return ZH
}

// SetLocale 显式设置语言（工具 CLI 的 --lang 用）。
func SetLocale(l string) {
	l = strings.ToLower(strings.TrimSpace(l))
	if l == EN || l == ZH {
		locale = l
	}
}

// Locale 返回当前语言。
func Locale() string { return locale }

// T 渲染文案：英文模式下查表，未登记则原样返回中文（保证信息不丢）。
func T(format string, args ...any) string {
	if locale == EN {
		if en, ok := table[format]; ok {
			format = en
		}
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// Has 报告某模板是否已有英文译文（覆盖统计用）。
func Has(format string) bool {
	_, ok := table[format]
	return ok
}

// Untranslated 返回给定模板集合中尚未翻译的条目（按字典序）。
func Untranslated(formats []string) []string {
	var out []string
	for _, f := range formats {
		if !Has(f) {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}
