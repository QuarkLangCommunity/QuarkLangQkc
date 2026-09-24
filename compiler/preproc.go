package main

// preproc.go —— QuarkLang **预处理器**（跨系统的核心工具）
//
// 目的（用户要求）：**不依赖 .so / 平台二进制**，用预处理指令表达平台/架构/特性差异，
// 在**使用方编译期**解析。这样一份库制品（可移植 IR）可以在任何系统上被合并编译。
//
// 支持的指令：
//   #define NAME                定义特性开关
//   #undef NAME
//   #ifdef NAME / #ifndef NAME
//   #if <cond> / #elif <cond> / #else / #endif
//   #include "path.qk"          相对当前文件
//   #error <msg>               条件不满足时报错
//
// 条件表达式：
//   os("linux"|"darwin"|"windows")   arch("x86_64"|"arm64"|"any")
//   defined(NAME)  true  false  !  &&  ||  括号
//
// 目标平台默认取宿主，可用 --target-os / --target-arch 指定（交叉预处理）。

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type preprocCtx struct {
	os      string
	arch    string
	defines map[string]bool
	depth   int
}

func newPreprocCtx() *preprocCtx {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "arm64"
	}
	return &preprocCtx{os: runtime.GOOS, arch: arch, defines: map[string]bool{}}
}

var (
	reIfLine   = regexp.MustCompile(`^\s*#\s*(if|elif)\s+(.*)$`)
	reDirLine  = regexp.MustCompile(`^\s*#\s*([a-z_]+)\b\s*(.*)$`)
	reInclude  = regexp.MustCompile(`^\s*#\s*include\s+"([^"]+)"\s*$`)
	reDefineNm = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// Process 预处理源码（返回展开后的源码）
func (p *preprocCtx) Process(src, filename string) (string, error) {
	if p.depth > 16 {
		return "", fmt.Errorf("预处理 #include 层数过深（>16）：%s", filename)
	}
	lines := strings.Split(src, "\n")
	var out []string
	// active: 当前分支是否输出；taken: 本组是否已有分支命中
	type frame struct{ active, taken, parentActive bool }
	stack := []frame{}
	active := true
	for i, ln := range lines {
		if m := reIfLine.FindStringSubmatch(ln); m != nil {
			kind, expr := m[1], m[2]
			cond := false
			if active || len(stack) > 0 {
				c, err := p.eval(expr)
				if err != nil {
					return "", fmt.Errorf("%s:%d: %v", filename, i+1, err)
				}
				cond = c
			}
			if kind == "if" {
				stack = append(stack, frame{active: false, taken: false, parentActive: active})
				top := &stack[len(stack)-1]
				top.active = top.parentActive && cond
				top.taken = cond
				active = top.active
			} else { // elif
				if len(stack) == 0 {
					return "", fmt.Errorf("%s:%d: #elif 没有对应的 #if", filename, i+1)
				}
				top := &stack[len(stack)-1]
				top.active = top.parentActive && !top.taken && cond
				top.taken = top.taken || cond
				active = top.active
			}
			continue
		}
		if m := reDirLine.FindStringSubmatch(strings.TrimSpace(ln)); m != nil {
			switch m[1] {
			case "else":
				if len(stack) == 0 {
					return "", fmt.Errorf("%s:%d: #else 没有对应的 #if", filename, i+1)
				}
				top := &stack[len(stack)-1]
				top.active = top.parentActive && !top.taken
				top.taken = true
				active = top.active
				continue
			case "endif":
				if len(stack) == 0 {
					return "", fmt.Errorf("%s:%d: #endif 没有对应的 #if", filename, i+1)
				}
				f := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				active = f.parentActive
				continue
			case "define":
				if active {
					if nm := strings.TrimSpace(m[2]); reDefineNm.MatchString(nm) {
						p.defines[nm] = true
					}
				}
				continue
			case "undef":
				if active {
					delete(p.defines, strings.TrimSpace(m[2]))
				}
				continue
			case "ifdef", "ifndef":
				nm := strings.TrimSpace(m[2])
				cond := p.defines[nm]
				if m[1] == "ifndef" {
					cond = !cond
				}
				stack = append(stack, frame{active: active && cond, taken: cond, parentActive: active})
				active = stack[len(stack)-1].active
				continue
			case "error":
				if active {
					return "", fmt.Errorf("%s:%d: #error %s", filename, i+1, strings.TrimSpace(m[2]))
				}
				continue
			case "macro":
				// 交给既有宏展开（保持兼容）
				if active {
					out = append(out, ln)
				}
				continue
			}
		}
		if m := reInclude.FindStringSubmatch(ln); m != nil {
			if !active {
				continue
			}
			inc := m[1]
			if !filepath.IsAbs(inc) {
				inc = filepath.Join(filepath.Dir(filename), inc)
			}
			b, err := os.ReadFile(inc)
			if err != nil {
				return "", fmt.Errorf("%s:%d: #include 失败：%v", filename, i+1, err)
			}
			p.depth++
			sub, err := p.Process(string(b), inc)
			p.depth--
			if err != nil {
				return "", err
			}
			out = append(out, sub)
			continue
		}
		if active {
			out = append(out, ln)
		}
	}
	if len(stack) != 0 {
		return "", fmt.Errorf("%s: 有 %d 个 #if 未闭合（缺 #endif）", filename, len(stack))
	}
	return strings.Join(out, "\n"), nil
}

// eval 求值条件表达式
func (p *preprocCtx) eval(expr string) (bool, error) {
	v, err := p.evalOr(strings.TrimSpace(expr))
	return v, err
}

func (p *preprocCtx) evalOr(e string) (bool, error) {
	parts := splitTop(e, "||")
	if len(parts) > 1 {
		for _, p0 := range parts {
			if v, err := p.evalAnd(strings.TrimSpace(p0)); err != nil {
				return false, err
			} else if v {
				return true, nil
			}
		}
		return false, nil
	}
	return p.evalAnd(e)
}

func (p *preprocCtx) evalAnd(e string) (bool, error) {
	parts := splitTop(e, "&&")
	if len(parts) > 1 {
		for _, p0 := range parts {
			if v, err := p.evalUnary(strings.TrimSpace(p0)); err != nil {
				return false, err
			} else if !v {
				return false, nil
			}
		}
		return true, nil
	}
	return p.evalUnary(e)
}

func (p *preprocCtx) evalUnary(e string) (bool, error) {
	e = strings.TrimSpace(e)
	if strings.HasPrefix(e, "!") {
		v, err := p.evalUnary(strings.TrimSpace(e[1:]))
		return !v, err
	}
	if strings.HasPrefix(e, "(") && strings.HasSuffix(e, ")") {
		return p.evalOr(strings.TrimSpace(e[1 : len(e)-1]))
	}
	switch e {
	case "true", "1":
		return true, nil
	case "false", "0", "":
		return false, nil
	}
	if m := regexp.MustCompile(`^os\s*\(\s*"([^"]*)"\s*\)$`).FindStringSubmatch(e); m != nil {
		want := m[1]
		return want == "any" || want == p.os, nil
	}
	if m := regexp.MustCompile(`^arch\s*\(\s*"([^"]*)"\s*\)$`).FindStringSubmatch(e); m != nil {
		want := m[1]
		return want == "any" || want == p.arch, nil
	}
	if m := regexp.MustCompile(`^defined\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)$`).FindStringSubmatch(e); m != nil {
		return p.defines[m[1]], nil
	}
	if reDefineNm.MatchString(e) {
		return p.defines[e], nil
	}
	return false, fmt.Errorf("无法识别的条件表达式：%q", e)
}

// splitTop 按顶层运算符切分（忽略括号内的）
func splitTop(s, op string) []string {
	var out []string
	depth, last := 0, 0
	for i := 0; i+len(op) <= len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && strings.HasPrefix(s[i:], op) {
			out = append(out, s[last:i])
			last = i + len(op)
			i += len(op) - 1
		}
	}
	out = append(out, s[last:])
	return out
}
