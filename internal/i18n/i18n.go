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

// detect decides the initial language:
//
//	QK_LANG (explicit) > LC_ALL / LC_MESSAGES / LANG prefix > English.
//
// Locale-based detection means an English machine reports English and a Chinese machine Chinese,
// with no configuration; QK_LANG always wins for scripts and tests.
func detect() string {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("QK_LANG"))); v != "" {
		if strings.HasPrefix(v, ZH) {
			return ZH
		}
		if strings.HasPrefix(v, EN) {
			return EN
		}
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
		if v == "" {
			continue
		}
		if strings.HasPrefix(v, ZH) {
			return ZH
		}
		if strings.HasPrefix(v, EN) {
			return EN
		}
	}
	return EN // global default: the docs are English-primary
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
