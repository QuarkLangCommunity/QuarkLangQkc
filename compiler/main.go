// qkc: QuarkLang → LLVM IR compiler (compiler branch, v0.2).
// Usage: qkc file.qk           write LLVM IR to stdout (incremental: an IR cache hit skips the full compile)
//
//	qkc -run file.qk      compile the IR to a native binary with clang and run it (incremental: a binary cache hit skips clang)
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"quarklang/compiler/internal/cgen"
	"quarklang/internal/i18n"
	"runtime"
	"sort"
	"strings"
)

// Incremental compilation: cache the IR and native binary keyed by source file content hash.
// The cache dir can be overridden with QUARK_CACHE; default <tmp>/quarklang-cache.
// runtimeSrc writes the embedded thread runtime to a temp file and returns its path (parallelism carrier: POSIX pthread; Windows branch still to be added).
func runtimeSrc() string {
	if runtime.GOOS == "windows" {
		return "" // Windows runtime still to be added (CreateThread version)
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

// engineVersion is the compiler/runtime generation: it must be bumped on any cgen/macro-expansion behavior change,
// to keep the IR/binary cache from returning artifacts of an old engine (a past trap: the macro-expansion mode and the done-bool fix were swallowed by the cache).
const engineVersion = "13"

// version is the release version: injected at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev" // 13: T&/pointer T nullable references (new T + auto-dereference + nil semantics)

// engineFingerprint is the cache-key prefix: engine generation + thread-runtime fingerprint (any runtime change invalidates it automatically).
func engineFingerprint() string {
	h := sha256.Sum256([]byte(qthreadsC))
	return engineVersion + "-" + hex.EncodeToString(h[:8])
}

// linkLib is one library's link candidate (in priority order; each group is a list of clang arguments).
type linkLib struct {
	name   string
	groups [][]string
}

// parseLinkLibs parses the link markers in the IR:
//
//	; qkc-link: <lib name> => <candidate args> => <candidate args>
//
// Compatible with the old format ("; qkc-link: -lm").
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

// linkCombos expands candidate combinations (one group per library, capped at 8 to avoid a combinatorial explosion).
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

// linkDiag builds an explicit diagnostic for a link failure (library name + the link arguments tried).
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
		parts = append(parts, fmt.Sprintf(msg("%s（尝试过：%s）"), l.name, strings.Join(gs, " / ")))
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
	srcFile := ""
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
			args = args[1:] // default behavior, explicit form
		case "--lang", "-lang":
			// This loop advances by chopping off the first element; there is no index variable
			if len(args) >= 2 {
				if l, ok := i18n.ParseLang(args[1]); ok {
					i18n.SetLocale(l)
					cgen.SetLocalizer(i18n.New(nil, l)) // lowering diagnostics
				} // zh | en; an unknown value keeps the current language
				args = args[2:]
			} else {
				args = args[1:]
			}
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
			// A positional argument is the source file. Flags that follow it are still parsed:
			// `qkc -c f.qk -o out` used to drop `-o` silently and build nothing.
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(os.Stderr, msg("qkc: 未知参数 %s\n"), a)
				qkcUsage()
				os.Exit(2)
			}
			if srcFile != "" {
				fmt.Fprintf(os.Stderr, msg("qkc: 只能指定一个源文件（已给出 %s）\n"), srcFile)
				os.Exit(2)
			}
			srcFile = a
			args = args[1:]
		}
	}
	if srcFile == "" {
		qkcUsage()
		os.Exit(2)
	}
	// `-c` without `-o`: default to the source basename, like clang, instead of building nothing.
	if compileOnly && outPath == "" {
		outPath = strings.TrimSuffix(filepath.Base(srcFile), filepath.Ext(srcFile))
	}
	// Preprocessor context (cross-platform): the target platform affects preprocessing results → it **must go into the cache key**
	pp := newPreprocCtx()
	if targetOS != "" {
		pp.os = targetOS
	}
	if targetArch != "" {
		pp.arch = targetArch
	}
	ppTag := pp.os + "-" + pp.arch
	if os.Getenv("QKC_DEBUG_PP") != "" {
		fmt.Fprintln(os.Stderr, msg("qkc[debug]: 预处理目标 ="), ppTag)
	}

	hash, err := srcHash(srcFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	irSuffix := ".ll"
	if emitLib {
		irSuffix = "-lib.ll"
	}
	irPath := filepath.Join(cacheDir(), hash+"-"+ppTag+irSuffix) // the target platform goes into the key

	// ── Library mode: generate an IR variant per target platform and package it (does not enter the program flow) ──
	if emitLib {
		out := outPath
		if out == "" {
			out = strings.TrimSuffix(filepath.Base(srcFile), ".qk") + ".qklib"
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
		rawSrc, _ := os.ReadFile(srcFile)
		for _, tgt := range targets {
			osName, archName, _ := strings.Cut(tgt, "-")
			pctx := newPreprocCtx()
			pctx.os = osName
			if archName != "" {
				pctx.arch = archName
			}
			pre, perr := pctx.Process(string(rawSrc), srcFile)
			if perr != nil {
				fmt.Fprintln(os.Stderr, "error:", perr)
				os.Exit(1)
			}
			ex, xerr := expandMacros(pre, "compile")
			if xerr != nil {
				fmt.Fprintln(os.Stderr, "error:", xerr)
				os.Exit(1)
			}
			cgen.SetLibMode(true) // library: main is exempt and main is not emitted
			libIR, terr := cgen.Transpile(ex, srcFile)
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
			// Exported-surface ABI normalization (value-type boundaries): must happen before obfuscation
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
		fmt.Fprintf(os.Stderr, msg("qkc: 已生成库制品 %s%s ｜ 平台变体：%s ｜ 导出 %d 个\n"), out, note, strings.Join(keys, ","), len(exports))
		return
	}

	// ── Referenced libraries: pick a variant per target platform (merged into the consumer module at compile time; no .so / no dlopen) ──
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
			fmt.Fprintf(os.Stderr, msg("qkc: 提示：库 %s 无 %s 变体，退回 %s（跨系统请用 --lib-targets 生成对应变体）\n"), man.Name, ppTag, name)
		}
		note := ""
		if man.Obfuscated {
			note = "，已混淆"
		}
		fmt.Fprintf(os.Stderr, msg("qkc: 引用库 %s v%s（变体 %s，导出 %d 个符号%s）\n"), man.Name, man.Version, name, len(man.Exports), note)
		libIRs = append(libIRs, lir)
		sum := sha256.Sum256([]byte(lir))
		libFP += "|" + man.Name + ":" + man.Version + ":" + name + ":" + hex.EncodeToString(sum[:6])
		dn, blk := DeclBlockFor(man)
		decls = append(decls, blk)
		provided = append(provided, dn)
	}
	if len(provided) > 0 {
		cgen.SetObjectProvidedLibs(provided) // these libraries are provided by the merged-in IR; no -l is generated
		cgen.SetForceRuntimeHelpers(true)    // the consumer provides the runtime helper functions
	}

	// Add the library fingerprint to the cache key: a hit is always the "same set of libraries, already merged" IR, avoiding repeated merging
	if libFP != "" {
		sum := sha256.Sum256([]byte(libFP))
		irPath = filepath.Join(cacheDir(), hash+"-"+ppTag+"-lib"+hex.EncodeToString(sum[:6])+irSuffix)
	}
	// ── Read the cached IR or compile ──
	ir := ""
	if b, rerr := os.ReadFile(irPath); rerr == nil {
		ir = string(b) // already merged: do not merge again
	}
	if ir == "" {
		src, rerr := os.ReadFile(srcFile)
		if rerr != nil {
			fmt.Fprintln(os.Stderr, "error:", rerr)
			os.Exit(1)
		}
		pre, perr := pp.Process(string(src), srcFile)
		if perr != nil {
			fmt.Fprintln(os.Stderr, "error:", perr)
			os.Exit(1)
		}
		if len(decls) > 0 { // inject library declarations (the way a referenced library is written)
			pre = strings.Join(decls, "\n") + pre
		}
		expanded, xerr := expandMacros(pre, "compile")
		if xerr != nil {
			fmt.Fprintln(os.Stderr, "error:", xerr)
			os.Exit(1)
		}
		ir, err = cgen.Transpile(expanded, srcFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		for _, lIR := range libIRs {
			ir = mergeLibIR(ir, lIR) // library IR merge (compile time, consistent across platforms)
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

	// ── Build the native binary (-c / -run) ──
	cflags := os.Getenv("QUARK_CFLAGS")
	if cflags == "" {
		cflags = "-O3 -flto=thin"
	}
	// The cache file name must be **legal across platforms**: Windows names cannot contain `|` or spaces
	// (measured in CI: clang LNK1104 cannot open file '…|windows-x86_64|-O3 -flto=thin.bin')
	rawKey := hash + "|" + ppTag + "|" + cflags + libFP
	ksum := sha256.Sum256([]byte(rawKey))
	binKey := hash[:12] + "-" + ppTag + "-" + hex.EncodeToString(ksum[:8])
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

// qkcUsage prints the command-line usage.
func qkcUsage() {
	fmt.Fprintln(os.Stderr, msg(`usage: qkc [options] <file.qk>

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

环境：QUARK_CACHE=缓存目录  QUARK_CFLAGS=clang 旗标（默认 "-O3 -flto=thin"）`))
}
