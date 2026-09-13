// qkc：QuarkLang → LLVM IR 编译器（compiler 分支，v0.2）。
// 用法：qkc file.qk           输出 LLVM IR 到 stdout（增量：IR 缓存命中跳过全编译）
//
//	qkc -run file.qk      用 clang 编译 IR 为原生二进制并执行（增量：二进制缓存命中跳过 clang）
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"quarklang/compiler/internal/cgen"
)

// 增量编译：按源文件内容哈希缓存 IR 与原生二进制。
// 缓存目录可用 QUARK_CACHE 覆盖，默认 <tmp>/quarklang-cache。
// runtimeSrc 把嵌入的线程运行时写入临时文件并返回路径（并行载体：POSIX pthread；Windows 分支待补）。
func runtimeSrc() string {
	if runtime.GOOS == "windows" {
		return "" // Windows 运行时待补（CreateThread 版）
	}
	f, err := os.CreateTemp("", "qthreads-*.c")
	if err != nil {
		fmt.Fprintln(os.Stderr, "runtimeSrc: create temp:", err)
		return ""
	}
	if _, err := f.WriteString(qthreadsC); err != nil {
		fmt.Fprintln(os.Stderr, "runtimeSrc: write:", err)
	}
	f.Close()
	return f.Name()
}

func cacheDir() string {
	dir := os.Getenv("QUARK_CACHE")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "quarklang-cache")
	}
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// engineVersion 编译器/运行时代次：任何 cgen/宏展开行为变化都必须递增，
// 避免 IR/二进制缓存返回旧引擎产物（本次踩坑：宏展开模式与 done-bool 修复被缓存吞掉）。
const engineVersion = "11" // 11：interface{}（tAny）装箱 + RTTI 打印/相等/拆箱

// engineFingerprint 缓存键前缀：引擎代次 + 线程运行时指纹（运行时任何改动自动失效）。
func engineFingerprint() string {
	h := sha256.Sum256([]byte(qthreadsC))
	return engineVersion + "-" + hex.EncodeToString(h[:8])
}

// linkLib 是一个 library 的链接候选（按优先级，每组是一串 clang 参数）。
type linkLib struct {
	name   string
	groups [][]string
}

// parseLinkLibs 解析 IR 里的链接标记：
//
//	; qkc-link: <库名> => <候选参数> => <候选参数>
//
// 兼容旧格式（"; qkc-link: -lm"）。
func parseLinkLibs(ir string) []linkLib {
	var out []linkLib
	for _, line := range strings.Split(ir, "\n") {
		if !strings.HasPrefix(line, "; qkc-link:") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "; qkc-link:"))
		if rest == "" {
			continue
		}
		parts := strings.Split(rest, "=>")
		if len(parts) == 1 {
			out = append(out, linkLib{name: rest, groups: [][]string{strings.Fields(rest)}})
			continue
		}
		name := strings.TrimSpace(parts[0])
		var groups [][]string
		for _, g := range parts[1:] {
			if f := strings.Fields(strings.TrimSpace(g)); len(f) > 0 {
				groups = append(groups, f)
			}
		}
		if len(groups) == 0 {
			continue
		}
		out = append(out, linkLib{name: name, groups: groups})
	}
	return out
}

// linkCombos 展开候选组合（每个库取一组，上限 8 组避免组合爆炸）。
func linkCombos(libs []linkLib) [][]string {
	combos := [][]string{{}}
	for _, l := range libs {
		var next [][]string
		for _, c := range combos {
			for _, g := range l.groups {
				nc := append(append([]string{}, c...), g...)
				next = append(next, nc)
				if len(next) >= 8 {
					break
				}
			}
			if len(next) >= 8 {
				break
			}
		}
		combos = next
	}
	return combos
}

// linkDiag 生成链接失败的明确诊断（库名 + 尝试过的链接参数）。
func linkDiag(libs []linkLib) string {
	if len(libs) == 0 {
		return ""
	}
	var parts []string
	for _, l := range libs {
		var gs []string
		for _, g := range l.groups {
			gs = append(gs, strings.Join(g, " "))
		}
		parts = append(parts, fmt.Sprintf("%s（尝试过：%s）", l.name, strings.Join(gs, " / ")))
	}
	return "library FFI 链接失败：" + strings.Join(parts, "；")
}

func srcHash(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	c := sha256.Sum256([]byte(engineFingerprint()))
	key := append([]byte(engineFingerprint()), b...)
	h := sha256.Sum256(key)
	_ = c
	return hex.EncodeToString(h[:16]), nil
}

func main() {
	args := os.Args[1:]
	run := false
	if len(args) > 0 && args[0] == "-run" {
		run = true
		args = args[1:]
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: qkc [-run] <file.qk>   # 输出 LLVM IR；-run 编译并执行")
		os.Exit(2)
	}
	hash, err := srcHash(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	irPath := filepath.Join(cacheDir(), hash+".ll")
	// 编译旗标：默认 -O3 跨系统便携（IR 与平台无关，目标平台 clang 生成原生二进制）；
	// 本机极限可 QUARK_CFLAGS="-O3 -march=native"（产物仅当前 CPU 可运行）。
	cflags := os.Getenv("QUARK_CFLAGS")
	if cflags == "" {
		// 默认 -O3 + thinLTO（跨翻译单元可见，fib35 实测 -4%）；LTO 不可用时降级纯 -O3
		cflags = "-O3 -flto=thin"
	}
	binKey := hash + "|" + cflags
	binPath := filepath.Join(cacheDir(), binKey+".bin")

	// -run：二进制缓存命中 → 直接执行（跳过全编译 + clang）
	if run {
		if _, err := os.Stat(binPath); err == nil {
			out, err := exec.Command(binPath).CombinedOutput()
			fmt.Print(string(out))
			if err != nil {
				os.Exit(1)
			}
			return
		}
	}

	// IR 缓存命中 → 跳过 lex/parse/typecheck/emit
	var ir string
	if b, err := os.ReadFile(irPath); err == nil {
		ir = string(b)
	} else {
		src, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		// 宏展开（工程化：与解释器共用 token 级宏系统，compile 模式）
		expanded, err := expandMacros(string(src), "compile")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		ir, err = cgen.Transpile(expanded, args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(irPath, []byte(ir), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "warn: 无法写 IR 缓存:", err)
		}
	}

	if !run {
		fmt.Print(ir)
		return
	}

	// -run：编译 IR → 原生二进制并缓存
	tmp, err := os.CreateTemp("", "quark-*.ll")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(ir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	tmp.Close()
	// library FFI：IR 里的 "; qkc-link: <库名> => <候选参数>" 标记 → 逐组尝试链接参数
	libs := parseLinkLibs(ir)
	combos := linkCombos(libs)
	// LTO 降级：默认含 -flto=thin 失败则去掉重试（仅当用户未显式指定 QUARK_CFLAGS）
	attempts := [][]string{strings.Fields(cflags)}
	if os.Getenv("QUARK_CFLAGS") == "" {
		attempts = append(attempts, strings.Fields(strings.ReplaceAll(cflags, " -flto=thin", "")))
	}
	var lastOut string
	ok := false
	for _, cf := range attempts {
		for _, lf := range combos {
			args := []string{tmp.Name(), runtimeSrc(), "-o", binPath, "-Wno-override-module"}
			// POSIX 载体：线程运行时需 -pthread（Windows 分支用 CreateThread 版运行时不加）
			if runtime.GOOS != "windows" {
				args = append(args, "-pthread")
			}
			args = append(args, lf...)
			args = append(args, cf...)
			cmd := exec.Command("clang", args...)
			if out, err := cmd.CombinedOutput(); err == nil {
				ok = true
				break
			} else {
				lastOut = string(out)
			}
		}
		if ok {
			break
		}
	}
	if !ok {
		if diag := linkDiag(libs); diag != "" {
			fmt.Fprintln(os.Stderr, "error:", diag)
		}
		fmt.Fprintln(os.Stderr, "clang:", lastOut)
		os.Exit(1)
	}
	out, err := exec.Command(binPath).CombinedOutput()
	fmt.Print(string(out))
	if err != nil {
		os.Exit(1)
	}
}
