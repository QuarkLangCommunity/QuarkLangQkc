package main

// lib.go — library artifact (.qklib): **portable IR multi-platform variants**, merged at compile time, no .so / dlopen
//
// Design (user requirement: no .so dependency, cross-system via preprocessor directives):
//   ① Language core and libraries separated: here we only pack/read/merge "library artifacts", with no game knowledge;
//   ② Libraries ship as **portable LLVM IR**, keeping **multiple variants** per target platform (content decided by the preprocessor #if os()/arch());
//   ③ Consumer runs `qkc -L lib.qklib`: pick the variant for its own target platform, **merge the IR into its own module**,
//       and compile a native binary in one shot — no shared library, no rpath, no platform binary dependency, cross-system by nature;
//   ④ Obfuscation: rename internal symbols, strip source paths and target lines; exported names kept (for reference), no source in the artifact.
//
// Container: `QKLB` + JSON{manifest, variants{ "<os>-<arch>": "<IR text>" }}

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"quarklang/internal/i18n"
	"regexp"
	"sort"
	"strings"
	"time"
)

const qklibMagic = "QKLB"

// QKExport an exported symbol: IR signature + QL-level signature (for the referrer to generate declarations)
type QKExport struct {
	Name   string `json:"name"`
	Sig    string `json:"sig"`
	QLSig  string `json:"ql_sig"`
	Params string `json:"params"`
	Ret    string `json:"ret"`
}

// QKLibManifest library manifest (exposes only metadata and export signatures, no source)
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
	reSymRef  = regexp.MustCompile(`@(\.?[A-Za-z_][A-Za-z0-9_.]*)`) // allow private globals like .str1
	reSrcLine = regexp.MustCompile(`(?m)^source_filename\s*=.*$`)
	reTarget  = regexp.MustCompile(`(?m)^target\s+(datalayout|triple)\s*=.*$`)
	reDbgTail = regexp.MustCompile(`(?m)^![0-9]+\s*=.*$`)
)

// isRuntimeSym runtime/builtin symbols: **never touched** during obfuscation
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

// collectExports parse exports from IR (top-level defines, excluding main and the internal `_` prefix)
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

// qlSigs grab the QL signatures of functions from source
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

// reParamPtr matches "pointer parameters" (e.g. ` i32* noundef %p0`)
var reParamPtr = regexp.MustCompile(`^\s*([A-Za-z0-9_\.]+\*)\s+((?:noundef|nonnull|nocapture|readonly|signext|zeroext|align \d+)\s+)*%([A-Za-z0-9_\.]+)\s*$`)

// normalizeExportedABI — **exported-surface ABI normalization** (cross-system value-type boundary)
//
// The language passes arguments by pointer internally, but a cross-module call must agree with the caller (which passes by value).
// Here only **exported functions** are rewritten: pointer parameters → value parameters, with alloca+store at the entry to restore a pointer of the same name,
// so the body's existing usage stays completely unchanged; returns are already by value, nothing to handle.
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
			val := id + "_val" // Note: LLVM parameter names **must not contain a dot** (%p0.val would be misparsed)
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

// obfuscateIR obfuscate IR: rename internal symbols + strip source paths/target lines
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

// stripModuleHeader clear the library IR's module-level header before merging (the target is decided by the consumer module)
func stripModuleHeader(ir string) string {
	ir = reSrcLine.ReplaceAllString(ir, "")
	ir = reTarget.ReplaceAllString(ir, "")
	return strings.TrimSpace(ir)
}

// mergeLibIR merge library IR into the consumer IR.
//
// LLVM module-level objects (named types %T, declare declarations, attributes #N, metadata !N) collide between the two modules,
// so this does **dedup and cleanup**, merging only the library's own defines (function impls):
//
//	· Type with the same name already defined by the consumer → drop the library's duplicate definition
//	· Symbol with the same name already declared/defined by the consumer → drop the library's declare
//	· Drop the library's attributes/metadata lines and #N references on defines (avoid group-number mismatch)
func mergeLibIR(consumerIR, libIR string) string {
	// Collect the types and symbols the consumer already has
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

	// Symbols the library will **define**: same-named declares in the consumer must be deleted
	// (since clang 22, "declare then define a function of the same name" is rejected as invalid redefinition; minimal repro verified)
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
				continue // duplicate type: drop
			}
			haveType[m[1]] = true
			keep = append(keep, ln)
			continue
		}
		if strings.HasPrefix(t, "declare ") {
			if m := regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_.]*)\s*\(`).FindStringSubmatch(t); m != nil && haveSym[m[1]] {
				continue // already declared/defined under the same name: drop
			}
			keep = append(keep, ln)
			continue
		}
		if strings.HasPrefix(t, "attributes ") || strings.HasPrefix(t, "!") {
			continue // attribute group/metadata: drop (avoid group-number mismatch)
		}
		// Strip the trailing attribute-group reference #N from define lines
		if strings.HasPrefix(t, "define ") {
			ln = regexp.MustCompile(`\s+#\d+`).ReplaceAllString(ln, "")
		}
		keep = append(keep, ln)
	}
	return strings.TrimRight(consumerIR, "\n") + "\n\n; ===== merged library IR =====\n" +
		strings.Join(keep, "\n") + "\n"
}

// DeclBlockFor build the QL declaration block from the manifest: library <name> { fn ...; }
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

// writeQKLib pack: manifest + multi-platform IR variants
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

// readQKLib read and verify a library artifact
func readQKLib(path string) (QKLibManifest, map[string]string, error) {
	var man QKLibManifest
	b, err := os.ReadFile(path)
	if err != nil {
		return man, nil, err
	}
	if len(b) < 4 || string(b[:4]) != qklibMagic {
		return man, nil, errors.New(i18n.T("不是 .qklib 制品（magic 不匹配）：%s", path))
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
				return body.Manifest, nil, errors.New(i18n.T("变体 %s 校验失败（内容被改动）", t))
			}
		}
	}
	return body.Manifest, body.Variants, nil
}

// pickVariant pick the variant for the target platform (target looks like linux-x86_64)
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
