package main

// lib.go —— 库制品（.qklib）：**可移植 IR 多平台变体**，编译期合并，不使用 .so / dlopen
//
// 设计（用户要求：不依赖 .so，用预处理命令跨系统）：
//   ① 语言本体与库分离：这里只做"库制品"的打包/读取/合并，不含任何游戏知识；
//   ② 库以**可移植 LLVM IR**分发，按目标平台保存**多个变体**（内容由预处理器 #if os()/arch() 决定）；
//   ③ 使用方 `qkc -L lib.qklib`：按自身目标平台挑变体，**把 IR 合并进自己的模块**，
//      一次编译出原生二进制 —— 无共享库、无 rpath、无平台二进制依赖，天然跨系统；
//   ④ 混淆：内部符号改名、去源码路径与目标行；导出名保留（供引用），制品里没有源码。
//
// 容器：`QKLB` + JSON{manifest, variants{ "<os>-<arch>": "<IR 文本>" }}

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const qklibMagic = "QKLB"

// QKExport 一个导出符号：IR 签名 + QL 级签名（供引用方生成声明）
type QKExport struct {
	Name   string `json:"name"`
	Sig    string `json:"sig"`
	QLSig  string `json:"ql_sig"`
	Params string `json:"params"`
	Ret    string `json:"ret"`
}

// QKLibManifest 库清单（对外只暴露元数据与导出签名，不含源码）
type QKLibManifest struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	ABI        string            `json:"abi"`
	Payload    string            `json:"payload"`
	Targets    []string          `json:"targets"`
	Obfuscated bool              `json:"obfuscated"`
	Exports    []QKExport        `json:"exports"`
	SHA256     map[string]string `json:"sha256"`
	BuiltAt    int64             `json:"built_at"`
	QKC        string            `json:"qkc"`
}

var (
	reDefine  = regexp.MustCompile(`(?m)^define\s+([^@]*?)@([A-Za-z_][A-Za-z0-9_.]*)\s*\(([^)]*)\)`)
	reSymRef  = regexp.MustCompile(`@(\.?[A-Za-z_][A-Za-z0-9_.]*)`) // 允许 .str1 这类私有全局
	reSrcLine = regexp.MustCompile(`(?m)^source_filename\s*=.*$`)
	reTarget  = regexp.MustCompile(`(?m)^target\s+(datalayout|triple)\s*=.*$`)
	reDbgTail = regexp.MustCompile(`(?m)^![0-9]+\s*=.*$`)
)

// isRuntimeSym 运行时/内建符号：混淆时**绝不动**
func isRuntimeSym(name string) bool {
	for _, p := range []string{"llvm.", "qk_", "ql_", "__", "main", "printf", "malloc", "free", "memcpy",
		"memmove", "memset", "strlen", "pthread_", "exit", "abort", "puts", "putchar", "calloc",
		"realloc", "fwrite", "fputs", "stdout", "stderr", "fopen", "fclose", "fflush", "snprintf"} {
		if name == p || strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// collectExports 从 IR 解析导出（顶层 define，排除 main 与内部 `_` 前缀）
func collectExports(ir string) []QKExport {
	var out []QKExport
	seen := map[string]bool{}
	for _, m := range reDefine.FindAllStringSubmatch(ir, -1) {
		name, params := m[2], strings.TrimSpace(m[3])
		if name == "main" || strings.HasPrefix(name, "_") || seen[name] {
			continue
		}
		seen[name] = true
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

var qlSigRe = regexp.MustCompile(`(?m)^\s*fn\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*([A-Za-z_][A-Za-z0-9_]*(?:<[^>]*>)?)?`)

// qlSigs 从源码抓函数的 QL 签名
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

// reParamPtr 匹配"指针形参"（例如 ` i32* noundef %p0`）
var reParamPtr = regexp.MustCompile(`^\s*([A-Za-z0-9_\.]+\*)\s+((?:noundef|nonnull|nocapture|readonly|signext|zeroext|align \d+)\s+)*%([A-Za-z0-9_\.]+)\s*$`)

// normalizeExportedABI —— **导出面 ABI 规范化**（跨系统的值类型边界）
//
// 语言内部用"按指针传参"，但跨模块调用必须与调用方一致（调用方按值传）。
// 这里只改**导出函数**：指针形参 → 值形参，并在入口处 alloca+store 复原同名指针，
// 使函数体原有用法完全不变；返回本就按值，无需处理。
func normalizeExportedABI(ir string, exports []QKExport) (string, map[string]string) {
	keep := map[string]bool{}
	for _, e := range exports {
		keep[e.Name] = true
	}
	lines := strings.Split(ir, "\n")
	out := make([]string, 0, len(lines)+16)
	sigs := map[string]string{}
	defRe := regexp.MustCompile(`^define\s+([^@]+)@([A-Za-z_][A-Za-z0-9_\.]*)\s*\((.*)\)(.*)$`)
	i := 0
	for i < len(lines) {
		m := defRe.FindStringSubmatch(lines[i])
		if m == nil || !keep[m[2]] {
			out = append(out, lines[i])
			i++
			continue
		}
		ret, name, params, tail := m[1], m[2], m[3], m[4]
		type conv struct{ slot, typ, val string }
		var convs []conv
		var newParts, types []string
		for _, p := range strings.Split(params, ",") {
			pm := reParamPtr.FindStringSubmatch(p)
			if pm == nil {
				newParts = append(newParts, strings.TrimSpace(p))
				types = append(types, strings.TrimSpace(p))
				continue
			}
			ptrType, id := pm[1], pm[3]
			valType := strings.TrimSuffix(ptrType, "*")
			val := id + "_val" // 注意：LLVM 参数名**不能带点**（%p0.val 会被误解析）
			newParts = append(newParts, valType+" %"+val)
			types = append(types, valType)
			convs = append(convs, conv{slot: id, typ: valType, val: val})
		}
		out = append(out, "define "+ret+"@"+name+"("+strings.Join(newParts, ", ")+")"+tail)
		sigs[name] = "(" + strings.Join(types, ", ") + ")"
		i++
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

// obfuscateIR 混淆 IR：内部符号改名 + 去源码路径/目标行
func obfuscateIR(ir string, exports []QKExport) string {
	keep := map[string]bool{"main": true}
	for _, e := range exports {
		keep[e.Name] = true
	}
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
	ir = reSrcLine.ReplaceAllString(ir, `source_filename = "<lib>"`)
	ir = reTarget.ReplaceAllString(ir, "")
	ir = reDbgTail.ReplaceAllStringFunc(ir, func(l string) string {
		if strings.Contains(l, "llvm.dbg") || strings.Contains(l, "DI") {
			return ""
		}
		return l
	})
	return ir
}

// stripModuleHeader 合并前清掉库 IR 的模块级头部（目标由使用方模块决定）
func stripModuleHeader(ir string) string {
	ir = reSrcLine.ReplaceAllString(ir, "")
	ir = reTarget.ReplaceAllString(ir, "")
	return strings.TrimSpace(ir)
}

// mergeLibIR 把库 IR 合并进使用方 IR。
//
// LLVM 模块级对象（命名类型 %T、declare 声明、attributes #N、元数据 !N）在两个模块里会冲突，
// 因此这里做**去重与清理**，只并入库自己的 define（函数实现）：
//
//	· 消费方已定义的同名类型 → 丢弃库里的重复定义
//	· 消费方已声明/定义的同名符号 → 丢弃库里的 declare
//	· 丢弃库的 attributes/metadata 行与 define 上的 #N 引用（避免组号错位）
func mergeLibIR(consumerIR, libIR string) string {
	// 收集消费方已有的类型与符号
	typeRe := regexp.MustCompile(`(?m)^%([A-Za-z0-9_.]+)\s*=\s*type\b`)
	symRe := regexp.MustCompile(`(?m)^(?:declare|define)[^@]*@([A-Za-z_][A-Za-z0-9_.]*)\s*\(`)
	haveType := map[string]bool{}
	for _, m := range typeRe.FindAllStringSubmatch(consumerIR, -1) {
		haveType[m[1]] = true
	}
	haveSym := map[string]bool{}
	for _, m := range symRe.FindAllStringSubmatch(consumerIR, -1) {
		haveSym[m[1]] = true
	}

	// 库将要**定义**的符号：使用方里的同名 declare 必须删掉
	// （clang 22 起，"declare 后 define 同名函数"会被判为 invalid redefinition，最小复现已验证）
	libDefs := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^define[^@]*@([A-Za-z_][A-Za-z0-9_.]*)\s*\(`).FindAllStringSubmatch(libIR, -1) {
		libDefs[m[1]] = true
	}
	var consumerKept []string
	for _, ln := range strings.Split(consumerIR, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "declare ") {
			if m := regexp.MustCompile(`@(\.?[A-Za-z_][A-Za-z0-9_.]*)\s*\(`).FindStringSubmatch(t); m != nil && libDefs[m[1]] {
				continue
			}
		}
		consumerKept = append(consumerKept, ln)
	}
	consumerIR = strings.Join(consumerKept, "\n")

	keep := []string{}
	for _, ln := range strings.Split(stripModuleHeader(libIR), "\n") {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		if m := regexp.MustCompile(`^%([A-Za-z0-9_.]+)\s*=\s*type\b`).FindStringSubmatch(t); m != nil {
			if haveType[m[1]] {
				continue // 重复类型：丢弃
			}
			haveType[m[1]] = true
			keep = append(keep, ln)
			continue
		}
		if strings.HasPrefix(t, "declare ") {
			if m := regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_.]*)\s*\(`).FindStringSubmatch(t); m != nil && haveSym[m[1]] {
				continue // 已有同名声明/定义：丢弃
			}
			keep = append(keep, ln)
			continue
		}
		if strings.HasPrefix(t, "attributes ") || strings.HasPrefix(t, "!") {
			continue // 属性组/元数据：丢弃（避免组号错位）
		}
		// define 行去掉尾部属性组引用 #N
		if strings.HasPrefix(t, "define ") {
			ln = regexp.MustCompile(`\s+#\d+`).ReplaceAllString(ln, "")
		}
		keep = append(keep, ln)
	}
	return strings.TrimRight(consumerIR, "\n") + "\n\n; ===== merged library IR =====\n" +
		strings.Join(keep, "\n") + "\n"
}

// DeclBlockFor 由清单生成 QL 声明块：library <name> { fn ...; }
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

// writeQKLib 打包：清单 + 多平台 IR 变体
func writeQKLib(out string, man QKLibManifest, variants map[string]string) error {
	man.SHA256 = map[string]string{}
	targets := make([]string, 0, len(variants))
	for t, ir := range variants {
		sum := sha256.Sum256([]byte(ir))
		man.SHA256[t] = hex.EncodeToString(sum[:])
		targets = append(targets, t)
	}
	sort.Strings(targets)
	man.Targets = targets
	if man.ABI == "" {
		man.ABI = "qk-ir-1"
	}
	man.Payload = "ir"
	if man.QKC == "" {
		man.QKC = version
	}
	if man.BuiltAt == 0 {
		man.BuiltAt = time.Now().Unix()
	}
	body := struct {
		Manifest QKLibManifest     `json:"manifest"`
		Variants map[string]string `json:"variants"`
	}{man, variants}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(out, append([]byte(qklibMagic), b...), 0o644)
}

// readQKLib 读取并校验库制品
func readQKLib(path string) (QKLibManifest, map[string]string, error) {
	var man QKLibManifest
	b, err := os.ReadFile(path)
	if err != nil {
		return man, nil, err
	}
	if len(b) < 4 || string(b[:4]) != qklibMagic {
		return man, nil, fmt.Errorf("不是 .qklib 制品（magic 不匹配）：%s", path)
	}
	var body struct {
		Manifest QKLibManifest     `json:"manifest"`
		Variants map[string]string `json:"variants"`
	}
	if err := json.Unmarshal(b[4:], &body); err != nil {
		return man, nil, fmt.Errorf("库制品解析失败：%w", err)
	}
	for t, ir := range body.Variants {
		if want, ok := body.Manifest.SHA256[t]; ok {
			sum := sha256.Sum256([]byte(ir))
			if hex.EncodeToString(sum[:]) != want {
				return body.Manifest, nil, fmt.Errorf("变体 %s 校验失败（内容被改动）", t)
			}
		}
	}
	return body.Manifest, body.Variants, nil
}

// pickVariant 按目标平台挑变体（target 形如 linux-x86_64）
func pickVariant(variants map[string]string, target string) (string, string, bool) {
	if ir, ok := variants[target]; ok {
		return target, ir, true
	}
	if ir, ok := variants["any"]; ok {
		return "any", ir, true
	}
	keys := make([]string, 0, len(variants))
	for k := range variants {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "", "", false
	}
	if osName, _, ok := strings.Cut(target, "-"); ok {
		for _, k := range keys {
			if strings.HasPrefix(k, osName+"-") {
				return k, variants[k], false
			}
		}
	}
	return keys[0], variants[keys[0]], false
}

func sanitizeName(s string) string {
	if s == "" {
		return "lib"
	}
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, s)
}
