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
const engineVersion = "7"

// engineFingerprint 缓存键前缀：引擎代次 + 线程运行时指纹（运行时任何改动自动失效）。
func engineFingerprint() string {
	h := sha256.Sum256([]byte(qthreadsC))
	return engineVersion + "-" + hex.EncodeToString(h[:8])
}

// linkFlags 提取 IR 里的链接标记（"; qkc-link: -lm"），供 clang 追加外部库。
func linkFlags(ir string) []string {
	var out []string
	for _, line := range strings.Split(ir, "\n") {
		if !strings.HasPrefix(line, "; qkc-link:") {
			continue
		}
		out = append(out, strings.Fields(strings.TrimPrefix(line, "; qkc-link:"))...)
	}
	return out
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
	clangArgs := []string{tmp.Name(), runtimeSrc(), "-o", binPath, "-Wno-override-module"}
	// POSIX 载体：线程运行时需 -pthread（Windows 分支用 CreateThread 版运行时不加）
	if runtime.GOOS != "windows" {
		clangArgs = append(clangArgs, "-pthread")
	}
	// library FFI：IR 里的 "; qkc-link: -lm" 标记 → 追加链接参数
	clangArgs = append(clangArgs, linkFlags(ir)...)
	// LTO 降级：默认含 -flto=thin 失败则去掉重试（仅当用户未显式指定 QUARK_CFLAGS）
	attempts := [][]string{strings.Fields(cflags)}
	if os.Getenv("QUARK_CFLAGS") == "" {
		attempts = append(attempts, strings.Fields(strings.ReplaceAll(cflags, " -flto=thin", "")))
	}
	var lastOut string
	ok := false
	for _, cf := range attempts {
		args := []string{tmp.Name(), runtimeSrc(), "-o", binPath, "-Wno-override-module"}
		if runtime.GOOS != "windows" {
			args = append(args, "-pthread")
		}
		args = append(args, linkFlags(ir)...)
		args = append(args, cf...)
		cmd := exec.Command("clang", args...)
		if out, err := cmd.CombinedOutput(); err == nil {
			ok = true
			break
		} else {
			lastOut = string(out)
		}
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "clang:", lastOut)
		os.Exit(1)
	}
	out, err := exec.Command(binPath).CombinedOutput()
	fmt.Print(string(out))
	if err != nil {
		os.Exit(1)
	}
}
