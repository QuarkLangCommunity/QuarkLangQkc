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
const engineVersion = "13"

// version 发布版本：构建时用 -ldflags "-X main.version=vX.Y.Z" 注入。
var version = "dev" // 13：T&/pointer T 可空引用（new T + 自动解引用 + nil 语义）

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
	compileOnly := false
	outPath := ""
	emitLib := false
	obfuscate := false
	libName := ""
	libVer := ""
	var libPaths []string
	for len(args) > 0 {
		a := args[0]
		switch a {
		case "-run":
			run = true
			args = args[1:]
		case "-c":
			compileOnly = true
			args = args[1:]
		case "--emit-lib":
			emitLib = true
			args = args[1:]
		case "--obfuscate":
			obfuscate = true
			args = args[1:]
		case "-L", "--use-lib":
			if len(args) < 2 {
				qkcUsage()
				os.Exit(2)
			}
			libPaths = append(libPaths, args[1])
			args = args[2:]
		case "--lib-name":
			if len(args) < 2 {
				qkcUsage()
				os.Exit(2)
			}
			libName = args[1]
			args = args[2:]
		case "--lib-version":
			if len(args) < 2 {
				qkcUsage()
				os.Exit(2)
			}
			libVer = args[1]
			args = args[2:]
		case "--emit-ir":
			args = args[1:] // 默认行为，显式写法
		case "--version", "-V":
			fmt.Println("qkc " + version + " (engine " + engineVersion + ")")
			return
		case "-h", "--help":
			qkcUsage()
			return
		case "-o":
			if len(args) < 2 {
				qkcUsage()
				os.Exit(2)
			}
			outPath = args[1]
			args = args[2:]
		default:
			goto parsed
		}
	}
parsed:
	if len(args) < 1 {
		qkcUsage()
		os.Exit(2)
	}
	hash, err := srcHash(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	irSuffix := ".ll"
	if emitLib {
		irSuffix = "-lib.ll" // 库模式单独缓存：模式变化必须重编译
	}
	irPath := filepath.Join(cacheDir(), hash+irSuffix)
	// 编译旗标：默认 -O3 跨系统便携（IR 与平台无关，目标平台 clang 生成原生二进制）；
	// 本机极限可 QUARK_CFLAGS="-O3 -march=native"（产物仅当前 CPU 可运行）。
	cflags := os.Getenv("QUARK_CFLAGS")
	if cflags == "" {
		// 默认 -O3 + thinLTO（跨翻译单元可见，fib35 实测 -4%）；LTO 不可用时降级纯 -O3
		cflags = "-O3 -flto=thin"
	}
	libFP := ""
	for _, lp := range libPaths {
		if man, obj, err := readQKLib(lp); err == nil {
			sum := sha256.Sum256(obj)
			libFP += "|" + man.Name + ":" + man.Version + ":" + hex.EncodeToString(sum[:8])
		} else {
			libFP += "|bad:" + lp
		}
	}
	binKey := hash + "|" + cflags + libFP
	binPath := filepath.Join(cacheDir(), binKey+".bin")

	// -run：二进制缓存命中 → 直接执行（跳过全编译 + clang）
	if run || compileOnly {
		if _, err := os.Stat(binPath); err == nil {
			if len(libPaths) > 0 { // 缓存命中：库文件仍需装到输出目录
				od := filepath.Dir(outPath)
				if outPath == "" {
					od = "."
				}
				if _, ierr := installLibs(libPaths, od); ierr != nil {
					fmt.Fprintln(os.Stderr, "error:", ierr)
					os.Exit(1)
				}
			}
			execPath := binPath
			if outPath != "" {
				if b, rerr := os.ReadFile(binPath); rerr == nil {
					if werr := os.WriteFile(outPath, b, 0o755); werr != nil {
						fmt.Fprintln(os.Stderr, "error:", werr)
						os.Exit(1)
					}
				}
				execPath = outPath
			}
			if compileOnly {
				fmt.Fprintln(os.Stderr, "qkc: 已构建 "+execPath)
				return
			}
			out, err := exec.Command(execPath).CombinedOutput()
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
		if emitLib {
			cgen.SetLibMode(true) // 库模式：豁免 main 入口要求
		}
		if len(libPaths) > 0 {
			// 引用库：把清单里的导出签名注入为 `library` 声明（零实现泄漏）
			var decls []string
			var provided []string
			for _, lp := range libPaths {
				if man, _, err := readQKLib(lp); err == nil {
					nm, blk := DeclBlockFor(man)
					decls = append(decls, blk)
					provided = append(provided, nm)
				}
			}
			cgen.SetObjectProvidedLibs(provided)
			cgen.SetForceRuntimeHelpers(true) // 宿主程序提供运行时辅助函数（供库的外部引用） // 这些库由对象文件提供
			src = []byte(strings.Join(decls, "\n") + string(src))
		}
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

	if emitLib {
		out := outPath
		if out == "" {
			out = strings.TrimSuffix(filepath.Base(args[0]), ".qk") + ".qklib"
		}
		if libName == "" {
			libName = strings.TrimSuffix(filepath.Base(out), ".qklib")
		}
		if libVer == "" {
			libVer = "0.1.0"
		}
		libSrc, _ := os.ReadFile(args[0])
		if err := emitLibFromIR(ir, string(libSrc), out, libName, libVer, obfuscate, strings.Fields(cflags)); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		lb := ""
		if obfuscate {
			lb = "（已混淆）"
		}
		fmt.Fprintf(os.Stderr, "qkc: 已生成库制品 %s%s\n", out, lb)
		return
	}

	if !run && !compileOnly {
		if outPath != "" {
			if err := os.WriteFile(outPath, []byte(ir), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		}
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
	// 用户引用的 .qklib 库制品：把共享库装到输出目录（运行时 dlopen，无需链接参数）
	outDir := filepath.Dir(outPath)
	if outPath == "" {
		outDir = "."
	}
	installedLibs, ierr := installLibs(libPaths, outDir)
	if ierr != nil {
		fmt.Fprintln(os.Stderr, "error:", ierr)
		os.Exit(1)
	}
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
			args := []string{tmp.Name(), runtimeSrc(), "-o", binPath, "-Wno-override-module",
				"-Wl,-rpath,$ORIGIN"} // 同目录下的库（如 libvfc_ext.so）可被找到
			// 引用的库：链接期解析符号（同时已装到输出目录，运行时也能找到）
			for _, lf := range installedLibs {
				args = append(args, lf)
			}
			if len(installedLibs) > 0 {
				args = append(args, "-rdynamic", "-Wl,--export-dynamic") // 库可反向解析宿主里的运行时符号
			}
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
	execPath := binPath
	if outPath != "" {
		if b, rerr := os.ReadFile(binPath); rerr == nil {
			if werr := os.WriteFile(outPath, b, 0o755); werr != nil {
				fmt.Fprintln(os.Stderr, "error:", werr)
				os.Exit(1)
			}
			execPath = outPath
		}
	}
	if compileOnly {
		// 新鲜构建后也要拷到 -o 指定路径（此前只在缓存命中时拷贝）
		if outPath != "" && execPath != outPath {
			if b, rerr := os.ReadFile(execPath); rerr == nil {
				if werr := os.WriteFile(outPath, b, 0o755); werr != nil {
					fmt.Fprintln(os.Stderr, "error:", werr)
					os.Exit(1)
				}
				execPath = outPath
			}
		}
		fmt.Fprintln(os.Stderr, "qkc: 已构建 "+execPath)
		return
	}
	out, err := exec.Command(execPath).CombinedOutput()
	fmt.Print(string(out))
	if err != nil {
		os.Exit(1)
	}
}

// qkcUsage 打印命令行用法。
func qkcUsage() {
	fmt.Fprintln(os.Stderr, `usage: qkc [options] <file.qk>

  （默认）            输出 LLVM IR 到 stdout
  --emit-ir           同上（显式）
  -run                编译为原生二进制并执行
  -c                  只编译为原生二进制（不执行）
  -o <path>           输出路径：配合 -c/-run 为二进制；--emit-lib 为 .qklib；否则为 IR 文件
  --emit-lib          编译为**库制品** .qklib（内含 lib<名>.so + 导出清单；不发源码）
  --obfuscate         混淆：内部符号改名 + 去调试/去标识（配合 --emit-lib 或 -c）
  -L <lib.qklib>      引用库：注入 library 声明 + 把 lib<名>.so 装到输出目录（可重复）
  --lib-name <name>   库名（默认取输出文件名）
  --lib-version <ver> 库版本（默认 0.1.0）
  --version, -V       打印版本与引擎代次
  -h, --help          本帮助

环境：QUARK_CACHE=缓存目录  QUARK_CFLAGS=clang 旗标（默认 "-O3 -flto=thin"）`)
}
