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
	"sort"
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
	targetOS := ""
	targetArch := ""
	libTargets := ""
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
		case "--lib-targets":
			if len(args) < 2 {
				qkcUsage()
				os.Exit(2)
			}
			libTargets = args[1]
			args = args[2:]
		case "--target-os":
			if len(args) < 2 {
				qkcUsage()
				os.Exit(2)
			}
			targetOS = args[1]
			args = args[2:]
		case "--target-arch":
			if len(args) < 2 {
				qkcUsage()
				os.Exit(2)
			}
			targetArch = args[1]
			args = args[2:]
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
	// 预处理器上下文（跨系统）：目标平台影响预处理结果 → **必须进缓存键**
	pp := newPreprocCtx()
	if targetOS != "" {
		pp.os = targetOS
	}
	if targetArch != "" {
		pp.arch = targetArch
	}
	ppTag := pp.os + "-" + pp.arch
	if os.Getenv("QKC_DEBUG_PP") != "" {
		fmt.Fprintln(os.Stderr, "qkc[debug]: 预处理目标 =", ppTag)
	}

	hash, err := srcHash(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	irSuffix := ".ll"
	if emitLib {
		irSuffix = "-lib.ll"
	}
	irPath := filepath.Join(cacheDir(), hash+"-"+ppTag+irSuffix) // 目标平台进键

	// ── 库模式：逐目标平台生成 IR 变体并打包（不进入程序流程）──
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
		targets := []string{ppTag}
		if strings.TrimSpace(libTargets) != "" {
			targets = nil
			for _, t := range strings.Split(libTargets, ",") {
				if t = strings.TrimSpace(t); t != "" {
					targets = append(targets, t)
				}
			}
		}
		variants := map[string]string{}
		var exports []QKExport
		rawSrc, _ := os.ReadFile(args[0])
		for _, tgt := range targets {
			osName, archName, _ := strings.Cut(tgt, "-")
			pctx := newPreprocCtx()
			pctx.os = osName
			if archName != "" {
				pctx.arch = archName
			}
			pre, perr := pctx.Process(string(rawSrc), args[0])
			if perr != nil {
				fmt.Fprintln(os.Stderr, "error:", perr)
				os.Exit(1)
			}
			ex, xerr := expandMacros(pre, "compile")
			if xerr != nil {
				fmt.Fprintln(os.Stderr, "error:", xerr)
				os.Exit(1)
			}
			cgen.SetLibMode(true) // 库：豁免 main，且不发射 main
			libIR, terr := cgen.Transpile(ex, args[0])
			cgen.SetLibMode(false)
			if terr != nil {
				fmt.Fprintln(os.Stderr, "error:", terr)
				os.Exit(1)
			}
			if len(exports) == 0 {
				exports = collectExports(libIR)
				sigs := qlSigs(string(rawSrc))
				for i := range exports {
					if q, ok := sigs[exports[i].Name]; ok {
						exports[i].QLSig, exports[i].Params, exports[i].Ret = q.QLSig, q.Params, q.Ret
					}
				}
			}
			// 导出面 ABI 规范化（值类型边界）：必须在混淆前做
			libIR, normSigs := normalizeExportedABI(libIR, exports)
			for k := range exports {
				if sig, ok := normSigs[exports[k].Name]; ok {
					exports[k].Sig = sig
				}
			}
			if obfuscate {
				libIR = obfuscateIR(libIR, exports)
			}
			variants[tgt] = libIR
		}
		man := QKLibManifest{Name: libName, Version: libVer, Obfuscated: obfuscate, Exports: exports}
		if err := writeQKLib(out, man, variants); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		note := ""
		if obfuscate {
			note = "（已混淆）"
		}
		keys := make([]string, 0, len(variants))
		for k := range variants {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(os.Stderr, "qkc: 已生成库制品 %s%s ｜ 平台变体：%s ｜ 导出 %d 个\n",
			out, note, strings.Join(keys, ","), len(exports))
		return
	}

	// ── 引用库：按目标平台挑变体（编译期合并进使用方模块；无 .so / 无 dlopen）──
	var libIRs []string
	libFP := ""
	decls := []string{}
	provided := []string{}
	for _, lp := range libPaths {
		man, variants, lerr := readQKLib(lp)
		if lerr != nil {
			fmt.Fprintln(os.Stderr, "error:", lerr)
			os.Exit(1)
		}
		name, lir, exact := pickVariant(variants, ppTag)
		if !exact {
			fmt.Fprintf(os.Stderr, "qkc: 提示：库 %s 无 %s 变体，退回 %s（跨系统请用 --lib-targets 生成对应变体）\n",
				man.Name, ppTag, name)
		}
		note := ""
		if man.Obfuscated {
			note = "，已混淆"
		}
		fmt.Fprintf(os.Stderr, "qkc: 引用库 %s v%s（变体 %s，导出 %d 个符号%s）\n",
			man.Name, man.Version, name, len(man.Exports), note)
		libIRs = append(libIRs, lir)
		sum := sha256.Sum256([]byte(lir))
		libFP += "|" + man.Name + ":" + man.Version + ":" + name + ":" + hex.EncodeToString(sum[:6])
		dn, blk := DeclBlockFor(man)
		decls = append(decls, blk)
		provided = append(provided, dn)
	}
	if len(provided) > 0 {
		cgen.SetObjectProvidedLibs(provided) // 这些库由合并进来的 IR 提供，不生成 -l
		cgen.SetForceRuntimeHelpers(true)    // 使用方提供运行时辅助函数
	}

	// 缓存键补入库指纹：命中的一定是"同一组库、已合并"的 IR，避免重复合并
	if libFP != "" {
		sum := sha256.Sum256([]byte(libFP))
		irPath = filepath.Join(cacheDir(), hash+"-"+ppTag+"-lib"+hex.EncodeToString(sum[:6])+irSuffix)
	}
	// ── 读缓存 IR 或编译 ──
	ir := ""
	if b, rerr := os.ReadFile(irPath); rerr == nil {
		ir = string(b) // 已合并过：不再合并
	}
	if ir == "" {
		src, rerr := os.ReadFile(args[0])
		if rerr != nil {
			fmt.Fprintln(os.Stderr, "error:", rerr)
			os.Exit(1)
		}
		pre, perr := pp.Process(string(src), args[0])
		if perr != nil {
			fmt.Fprintln(os.Stderr, "error:", perr)
			os.Exit(1)
		}
		if len(decls) > 0 { // 注入 library 声明（引用库的写法）
			pre = strings.Join(decls, "\n") + pre
		}
		expanded, xerr := expandMacros(pre, "compile")
		if xerr != nil {
			fmt.Fprintln(os.Stderr, "error:", xerr)
			os.Exit(1)
		}
		ir, err = cgen.Transpile(expanded, args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		for _, lIR := range libIRs {
			ir = mergeLibIR(ir, lIR) // 库 IR 合并（编译期，跨系统一致）
		}
		_ = os.WriteFile(irPath, []byte(ir), 0o644)
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

	// ── 构建原生二进制（-c / -run）──
	cflags := os.Getenv("QUARK_CFLAGS")
	if cflags == "" {
		cflags = "-O3 -flto=thin"
	}
	binKey := hash + "|" + ppTag + "|" + cflags
	binPath := filepath.Join(cacheDir(), binKey+".bin")
	if _, err := os.Stat(binPath); err != nil {
		tmp, terr := os.CreateTemp("", "quark-*.ll")
		if terr != nil {
			fmt.Fprintln(os.Stderr, "error:", terr)
			os.Exit(1)
		}
		defer os.Remove(tmp.Name())
		if _, werr := tmp.WriteString(ir); werr != nil {
			fmt.Fprintln(os.Stderr, "error:", werr)
			os.Exit(1)
		}
		tmp.Close()
		libs := parseLinkLibs(ir)
		combos := linkCombos(libs)
		attempts := [][]string{strings.Fields(cflags)}
		if os.Getenv("QUARK_CFLAGS") == "" {
			attempts = append(attempts, strings.Fields(strings.ReplaceAll(cflags, " -flto=thin", "")))
		}
		var lastOut string
		ok := false
		for _, cf := range attempts {
			for _, lf := range combos {
				cargs := []string{tmp.Name(), runtimeSrc(), "-o", binPath, "-Wno-override-module"}
				if runtime.GOOS != "windows" {
					cargs = append(cargs, "-pthread")
				}
				cargs = append(cargs, lf...)
				cargs = append(cargs, cf...)
				if outb, cerr := exec.Command("clang", cargs...).CombinedOutput(); cerr == nil {
					ok = true
					break
				} else {
					lastOut = string(outb)
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
	}
	if compileOnly {
		if outPath != "" && outPath != binPath {
			if b, rerr := os.ReadFile(binPath); rerr == nil {
				if werr := os.WriteFile(outPath, b, 0o755); werr != nil {
					fmt.Fprintln(os.Stderr, "error:", werr)
					os.Exit(1)
				}
			}
		}
		fmt.Fprintln(os.Stderr, "qkc: 已构建 "+outPath)
		return
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
  --target-os <os>    预处理目标系统（linux/darwin/windows；默认宿主）
  --target-arch <a>   预处理目标架构（x86_64/arm64；默认宿主）
  --emit-lib          编译为**库制品** .qklib（内含 lib<名>.so + 导出清单；不发源码）
  --obfuscate         混淆：内部符号改名 + 去调试/去标识（配合 --emit-lib 或 -c）
  -L <lib.qklib>      引用库：注入 library 声明 + 把 lib<名>.so 装到输出目录（可重复）
  --lib-name <name>   库名（默认取输出文件名）
  --lib-version <ver> 库版本（默认 0.1.0）
  --lib-targets <列表> 出库时生成的平台变体，逗号分隔（如 linux-x86_64,windows-x86_64,darwin-arm64）
  --version, -V       打印版本与引擎代次
  -h, --help          本帮助

环境：QUARK_CACHE=缓存目录  QUARK_CFLAGS=clang 旗标（默认 "-O3 -flto=thin"）`)
}
