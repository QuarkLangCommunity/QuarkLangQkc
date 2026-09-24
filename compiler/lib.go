package main

// lib.go —— **编译出库（--emit-lib）** 与 **混淆（--obfuscate）**、**引用库（-L）**
//
// 设计目标（用户需求）：
//   ① 库制品可直接被引用，**不泄漏核心代码**（只发二进制对象 + 导出清单，不发源码）；
//   ② 混淆：内部符号改名、去调试/去标识、路径清理（提升逆向成本）；
//   ③ 库可以带版本与 ABI 指纹，引用时校验，避免"二进制不匹配"的隐性故障。
//
// .qklib 容器布局（大端）：
//   "QKLB" | ver(1) | manifestLen(uint32) | manifest(JSON) | object(ELF .o)
//   manifest: {name, version, abi, obfuscated, exports:[{name, sig}], sha256_obj, built_at, qkc}
//
// 引用方式：qkc -c -L path/to/lib.qklib prog.qk -o prog
//   → 解包对象文件到临时目录，作为 clang 的额外输入参与链接（导出名保持不变，故可跨混淆链接）。

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const qklibMagic = "QKLB"
const qklibVersion = 1

// qlSigRe 从源码里抓导出函数的 QL 签名（库作者自己的源码，编译期可见）
var qlSigRe = regexp.MustCompile(`(?m)^\s*fn\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*([A-Za-z_][A-Za-z0-9_]*(?:<[^>]*>)?)?`)

// qlSigs 解析源码里所有顶层函数的 QL 签名
func qlSigs(src string) map[string]QKExport {
	out := map[string]QKExport{}
	for _, m := range qlSigRe.FindAllStringSubmatch(src, -1) {
		name, params, ret := m[1], strings.TrimSpace(m[2]), strings.TrimSpace(m[3])
		if ret == "" {
			ret = "void"
		}
		out[name] = QKExport{Name: name, Params: params, Ret: ret,
			QLSig: fmt.Sprintf("fn %s(%s) %s", name, params, ret)}
	}
	return out
}

// QKLibManifest 库清单（对外可见的只有这些元数据与导出签名）
type QKLibManifest struct {
	Name       string     `json:"name"`
	Version    string     `json:"version"`
	ABI        string     `json:"abi"`
	SOName     string     `json:"soname"`
	Platform   string     `json:"platform"` // linux / darwin / windows
	Obfuscated bool       `json:"obfuscated"`
	Exports    []QKExport `json:"exports"`
	SHA256Obj  string     `json:"sha256_obj"`
	BuiltAt    int64      `json:"built_at"`
	QKC        string     `json:"qkc"`
}

// QKExport 一个导出符号：IR 签名 + **QL 级签名**（供引用方自动生成声明）
type QKExport struct {
	Name   string `json:"name"`
	Sig    string `json:"sig"`    // IR 级（诊断用）
	QLSig  string `json:"ql_sig"` // 形如 "fn vfc_room_cap(int base) int"
	Params string `json:"params"` // 形如 "int base"
	Ret    string `json:"ret"`    // 形如 "int"（空=void）
}

var (
	reDefine   = regexp.MustCompile(`(?m)^define\s+([^@]*?)@([A-Za-z_][A-Za-z0-9_.]*)\s*\(([^)]*)\)`)
	reDeclare  = regexp.MustCompile(`(?m)^declare\s+([^@]*?)@([A-Za-z_][A-Za-z0-9_.]*)\s*\(([^)]*)\)`)
	reSymRef   = regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_.]*)`)
	reSourceLn = regexp.MustCompile(`(?m)^source_filename\s*=.*$`)
	reDbgRef   = regexp.MustCompile(`!llvm\.dbg[^\n]*\n?`)
	reDbgTail  = regexp.MustCompile(`(?m)^!\d+\s*=.*$`)
	reStrConst = regexp.MustCompile(`(?m)^(@[A-Za-z0-9_.]+)\s*=\s*(private|internal)?\s*(unnamed_addr\s*)?constant\s*\[(\d+)\s*x\s*i8\]\s*c"((?:[^"\\]|\\.)*)"`)
)

// isRuntimeSym 运行时/内建符号：混淆时**绝不改名**（否则链接不上运行时）
func isRuntimeSym(name string) bool {
	for _, p := range []string{"llvm.", "qk_", "__", "main", "printf", "malloc", "free", "memcpy",
		"memmove", "memset", "strlen", "pthread_", "exit", "abort", "puts", "putchar", "calloc",
		"realloc", "fwrite", "fputs", "stdout", "stderr", "fopen", "fclose", "fflush"} {
		if strings.HasPrefix(name, p) || name == p {
			return true
		}
	}
	return false
}

// collectExports 从 IR 里解析导出（顶层 define，排除 main 与内部 _ 前缀）
func collectExports(ir string) []QKExport {
	var out []QKExport
	seen := map[string]bool{}
	for _, m := range reDefine.FindAllStringSubmatch(ir, -1) {
		name, params := m[2], strings.TrimSpace(m[3])
		if name == "main" || strings.HasPrefix(name, "_") || seen[name] {
			continue
		}
		seen[name] = true
		// 参数只保留类型列表（不泄漏参数名/局部信息）
		var types []string
		for _, p := range strings.Split(params, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if i := strings.LastIndex(p, " "); i > 0 {
				p = p[:i]
			}
			types = append(types, p)
		}
		out = append(out, QKExport{Name: name, Sig: "(" + strings.Join(types, ", ") + ")"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// obfuscateIR 混淆 IR：
//
//	· 内部符号（非运行时、非导出、非 main）改名为 @o_<hash>
//	· 删除 source_filename 与调试元数据（不泄漏源码路径/变量名）
//	· 字符串常量按需保留（v1 只改名与清理；加密留待 v2，见文档）
func obfuscateIR(ir string, exports []QKExport) string {
	keep := map[string]bool{}
	for _, e := range exports {
		keep[e.Name] = true
	}
	keep["main"] = true
	rename := map[string]string{}
	ir = reSymRef.ReplaceAllStringFunc(ir, func(m string) string {
		name := m[1:]
		if keep[name] || isRuntimeSym(name) {
			return m
		}
		if nn, ok := rename[name]; ok {
			return "@" + nn
		}
		sum := sha256.Sum256([]byte(name + "|qk-obf"))
		nn := "o_" + hex.EncodeToString(sum[:])[:12]
		rename[name] = nn
		return "@" + nn
	})
	ir = reSourceLn.ReplaceAllString(ir, "source_filename = \"<obfuscated>\"")
	ir = reDbgRef.ReplaceAllString(ir, "")
	ir = reDbgTail.ReplaceAllStringFunc(ir, func(s string) string {
		if strings.Contains(s, "llvm.dbg") || strings.Contains(s, "DILocation") ||
			strings.Contains(s, "DISubprogram") || strings.Contains(s, "DIExpression") ||
			strings.Contains(s, "DIFile") || strings.Contains(s, "DICompileUnit") {
			return ""
		}
		return s
	})
	return ir
}

// normalizeExportedABI —— **导出面 ABI 规范化**（跨系统的值类型边界）
//
// 问题：库模式发射的导出函数，参数按**指针**传递（%p0 是 i32*），而调用方按**值**传递
// （`call i32 @f(i32 21)`）→ 运行时解引用 0x15 直接段错误（gdb 实测）。
//
// 做法：只改**导出函数**：把指针形参改成值形参，并在入口处 alloca+store 复原出同名指针，
// 使函数体（原本就通过 %p0 读写）**无需任何改动**。返回值本就是按值，无需处理。
// 这样库与程序的调用约定一致，且不依赖各平台/各编译单元的逃逸分析差异。
var reParamPtr = regexp.MustCompile(`^\s*([A-Za-z0-9_\.]+\*)\s+((?:noundef|nonnull|nocapture|readonly|signext|zeroext|align \d+)\s+)*%([A-Za-z0-9_\.]+)\s*$`)

func normalizeExportedABI(ir string, exports []QKExport) (string, map[string]string) {
	keep := map[string]bool{}
	for _, e := range exports {
		keep[e.Name] = true
	}
	lines := strings.Split(ir, "\n")
	out := make([]string, 0, len(lines)+16)
	sigs := map[string]string{}
	i := 0
	for i < len(lines) {
		ln := lines[i]
		// 找到导出函数定义起始
		m := regexp.MustCompile(`^define\s+([^@]+)@([A-Za-z_][A-Za-z0-9_\.]*)\s*\((.*)\)(.*)$`).FindStringSubmatch(ln)
		if m == nil || !keep[m[2]] {
			out = append(out, ln)
			i++
			continue
		}
		ret, name, params, tail := m[1], m[2], m[3], m[4]
		parts := strings.Split(params, ",")
		type conv struct{ slot, typ, val string }
		var convs []conv
		var newParts []string
		var types []string
		for _, p := range parts {
			pm := reParamPtr.FindStringSubmatch(p)
			if pm == nil {
				newParts = append(newParts, p)
				types = append(types, strings.TrimSpace(strings.TrimSuffix(p, "%"+lastIdent(p))))
				continue
			}
			ptrType, id := pm[1], pm[3] // 例如 i32* 与 p0
			valType := strings.TrimSuffix(ptrType, "*")
			val := id + ".val"
			newParts = append(newParts, " "+valType+" %"+val)
			types = append(types, valType)
			convs = append(convs, conv{slot: id, typ: valType, val: val})
		}
		for k := range newParts {
			newParts[k] = strings.TrimSpace(newParts[k])
		}
		out = append(out, "define "+ret+"@"+name+"("+strings.Join(newParts, ", ")+")"+tail)
		sigs[name] = "(" + strings.Join(types, ", ") + ")"
		i++
		// 复制函数体，并在首个标签行之后插入 alloca/store
		inserted := false
		for i < len(lines) {
			body := lines[i]
			out = append(out, body)
			if !inserted && strings.HasSuffix(strings.TrimSpace(body), ":") {
				for _, c := range convs {
					out = append(out, "  %"+c.slot+" = alloca "+c.typ)
					out = append(out, "  store "+c.typ+" %"+c.val+", "+c.typ+"* %"+c.slot)
				}
				inserted = true
			}
			i++
			if strings.TrimSpace(body) == "}" {
				break
			}
		}
	}
	return strings.Join(out, "\n"), sigs
}

// lastIdent 取 "%x" 里的 x（无则返回空串）
func lastIdent(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "%"); idx >= 0 {
		return strings.TrimSpace(s[idx+1:])
	}
	return ""
}

// writeQKLib 打包库制品：manifest + 对象文件
func writeQKLib(out string, man QKLibManifest, obj []byte) error {
	sum := sha256.Sum256(obj)
	man.SHA256Obj = hex.EncodeToString(sum[:])
	if man.ABI == "" {
		man.ABI = "qk-c-abi-1"
	}
	if man.QKC == "" {
		man.QKC = version
	}
	if man.BuiltAt == 0 {
		man.BuiltAt = time.Now().Unix()
	}
	mb, err := json.Marshal(man)
	if err != nil {
		return err
	}
	var buf []byte
	buf = append(buf, []byte(qklibMagic)...)
	buf = append(buf, byte(qklibVersion))
	var lb [4]byte
	binary.BigEndian.PutUint32(lb[:], uint32(len(mb)))
	buf = append(buf, lb[:]...)
	buf = append(buf, mb...)
	buf = append(buf, obj...)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil && filepath.Dir(out) != "." {
		return err
	}
	return os.WriteFile(out, buf, 0o644)
}

// readQKLib 读取库制品（供引用时校验）
func readQKLib(path string) (QKLibManifest, []byte, error) {
	var man QKLibManifest
	b, err := os.ReadFile(path)
	if err != nil {
		return man, nil, err
	}
	if len(b) < 6 || string(b[:4]) != qklibMagic {
		return man, nil, fmt.Errorf("不是 .qklib 制品（magic 不匹配）：%s", path)
	}
	n := binary.BigEndian.Uint32(b[5:9])
	if len(b) < 9+int(n) {
		return man, nil, fmt.Errorf("库制品损坏（manifest 越界）：%s", path)
	}
	if err := json.Unmarshal(b[9:9+int(n)], &man); err != nil {
		return man, nil, fmt.Errorf("库清单解析失败：%w", err)
	}
	obj := b[9+int(n):]
	if man.SHA256Obj != "" {
		sum := sha256.Sum256(obj)
		if hex.EncodeToString(sum[:]) != man.SHA256Obj {
			return man, nil, fmt.Errorf("库制品校验失败（对象文件被改动）：%s", path)
		}
	}
	return man, obj, nil
}

// emitLibFromIR 由 IR 生成库制品：clang -c 出对象 → 可选混淆 → 打包 .qklib
func emitLibFromIR(ir, src, out, name, ver string, obfuscate bool, cflags []string) error {
	exports := collectExports(ir)
	sigs := qlSigs(src)
	for i := range exports {
		if q, ok := sigs[exports[i].Name]; ok {
			exports[i].QLSig, exports[i].Params, exports[i].Ret = q.QLSig, q.Params, q.Ret
		}
	}
	// 导出面 ABI 规范化（值类型边界）：必须在混淆之前做（改名后难以定位参数）
	ir, normSigs := normalizeExportedABI(ir, exports)
	for i := range exports {
		if sig, ok := normSigs[exports[i].Name]; ok {
			exports[i].Sig = sig
		}
	}
	if obfuscate {
		ir = obfuscateIR(ir, exports)
	}
	dir, err := os.MkdirTemp("", "qklib-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	irFile := filepath.Join(dir, "lib.ll")
	objFile := filepath.Join(dir, "lib.o")
	if err := os.WriteFile(irFile, []byte(ir), 0o644); err != nil {
		return err
	}
	// 跨系统：按目标平台给出库文件名（不依赖 dlopen，链接器直接解析）
	base := sanitizeName(name)
	soname := "lib" + base + ".so"
	switch {
	case runtime.GOOS == "darwin":
		soname = "lib" + base + ".dylib"
	case runtime.GOOS == "windows":
		soname = base + ".dll"
	}
	// 版本脚本：**只导出清单里的符号**（内部符号一律 local），比 strip 更精确可靠
	var vs strings.Builder
	vs.WriteString("{\n  global:\n")
	if len(exports) == 0 {
		vs.WriteString("    __qk_no_exports;\n")
	}
	for _, e := range exports {
		fmt.Fprintf(&vs, "    %s;\n", e.Name)
	}
	vs.WriteString("  local: *;\n};\n")
	vsFile := filepath.Join(dir, "exports.map")
	if err := os.WriteFile(vsFile, []byte(vs.String()), 0o644); err != nil {
		return err
	}
	// 注意：库**不打进 C 运行时**（运行时符号留给使用方程序提供），避免重复定义
	args := []string{"-shared", "-fPIC", irFile, "-o", objFile,
		"-Wl,-soname," + soname, "-Wl,--version-script=" + vsFile,
		"-Wno-override-module", "-fno-ident", "-pthread"}
	args = append(args, cflags...)
	if out2, err := runCmd("clang", args...); err != nil {
		return fmt.Errorf("clang -shared 失败：%v\n%s", err, out2)
	}
	// 库不得导出 main：否则动态链接时会**顶替宿主程序的 main**（实测段错误）
	if out2, err := runCmd("objcopy", "--localize-symbol=main", objFile); err != nil {
		fmt.Fprintln(os.Stderr, "提示：objcopy 不可用，未本地化 main：", strings.TrimSpace(out2))
	}
	if obfuscate {
		// 只剥调试信息：**不可用 --strip-unneeded**（会把共享库的导出符号一起剥掉，实测链接失败）。
		// 内部符号本就由版本脚本 local:* 隐藏，这才是可靠的"不泄漏"手段。
		if _, err := runCmd("strip", "--strip-debug", objFile); err != nil {
			fmt.Fprintln(os.Stderr, "提示：strip 不可用，未剥离调试信息（不影响使用）")
		}
	}
	obj, err := os.ReadFile(objFile)
	if err != nil {
		return err
	}
	man := QKLibManifest{Name: name, Version: ver, Obfuscated: obfuscate, Exports: exports,
		SOName: soname, Platform: runtime.GOOS}
	return writeQKLib(out, man, obj)
}

// DeclBlockFor 由库清单生成 QL 声明块：library <name> { fn ...; }
// 引用方据此"直接引用库"，**看不到任何实现代码**。
func DeclBlockFor(man QKLibManifest) (string, string) {
	libName := sanitizeName(man.Name)
	var b strings.Builder
	fmt.Fprintf(&b, "library %s {\n", libName)
	for _, e := range man.Exports {
		if e.QLSig != "" {
			fmt.Fprintf(&b, "    %s;\n", e.QLSig)
		} else {
			fmt.Fprintf(&b, "    fn %s(%s) %s;\n", e.Name, e.Params, e.Ret)
		}
	}
	b.WriteString("}\n")
	return libName, b.String()
}

// installLibs 把 -L 指定的库制品里的共享库安装到 outDir（运行时优先加载 ./lib<名>.so）。
// 调用方无需链接参数：语言运行时按库名 dlopen。
func installLibs(paths []string, outDir string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	var installed []string
	for _, p := range paths {
		man, payload, err := readQKLib(p)
		if err != nil {
			return installed, err
		}
		soname := man.SOName
		if soname == "" {
			soname = "lib" + sanitizeName(man.Name) + ".so"
		}
		dst := filepath.Join(outDir, soname)
		if err := os.WriteFile(dst, payload, 0o755); err != nil {
			return installed, err
		}
		note := ""
		if man.Obfuscated {
			note = "，已混淆"
		}
		fmt.Fprintf(os.Stderr, "qkc: 引用库 %s v%s（导出 %d 个符号%s）→ %s\n",
			man.Name, man.Version, len(man.Exports), note, dst)
		installed = append(installed, dst)
	}
	return installed, nil
}

func sanitizeName(s string) string {
	if s == "" {
		return "lib"
	}
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}

// runCmd 运行外部命令并返回合并输出（失败时返回 error）
func runCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}
