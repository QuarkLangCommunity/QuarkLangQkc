package lang

// qkcheck 误报率 / 漏报率的**统计度量**：随机化用例 × 多轮采样 → P50/P90/P99/最差。
//
// 为什么需要它：单轮基准（lintbench_test.go）是确定性的，跑一万次结果一样，P99 没有意义。
// 这里每轮用不同随机种子生成一批「真值已知」的程序（干净骨架 + 注入已知缺陷），
// 因此轮与轮之间的误报/漏报计数是一个分布，尾部（P99）能暴露「罕见代码形态才触发」的问题。
//
// 口径：
//   - 每条用例的期望是 (诊断码, 行号) 集合；干净用例期望空集。
//   - FP = 报出但不在期望里；FN = 期望但没报出；TP = 命中。
//   - 单轮率 = FP/(TP+FP)、FN/(TP+FN)；多轮取分位数。
//   - 门禁：P99 的 FP、FN 计数必须为 0（尾部也不许出错），最差值一并打印供排查。
//
// 跨系统：纯 Go 实现（无 shell、无固定路径，临时目录用 t.TempDir），
// Windows/macOS/Linux 一致可跑。
//
// 环境变量：
//
//	LINT_ROUNDS=200   轮数（默认 200；CI 可调小）
//	LINT_SEED=1       随机种子基（默认 1，固定 → 可复现）
//	LINT_NOFAIL=1     只测量不失败（调参用）

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type statWant struct {
	code string
	line int
}

type statCase struct {
	name    string
	src     string
	libName string // 非空：额外写入同目录依赖文件 <libName>.qk
	libSrc  string
	params  bool
	want    []statWant
}

// ---------- 源码行构造器（注入器据此知道自己的行号 → 期望行号天然正确） ----------

type srcBuilder struct {
	lines []string
	rng   *rand.Rand
}

func (b *srcBuilder) add(format string, args ...interface{}) int {
	b.lines = append(b.lines, fmt.Sprintf(format, args...))
	return len(b.lines)
}

func (b *srcBuilder) blank() { b.lines = append(b.lines, "") }

func (b *srcBuilder) String() string { return strings.Join(b.lines, "\n") + "\n" }

// ---------- 干净积木（保证本身零诊断） ----------

// cleanFunc 生成一个 pub 函数：参数都用到、局部都读取、各分支都返回。
// 形态随机（1–3 个参数、分支/循环混合、返回表达式不同），以扩大随机覆盖面。
func cleanFunc(b *srcBuilder, i int) {
	switch b.rng.Intn(3) {
	case 0:
		b.add("pub fn fn%d(int a%d, String s%d) int {", i, i, i)
		b.add("    int t%d = a%d + 1;", i, i)
		b.add("    if (t%d > 0) {", i)
		b.add("        return t%d + s%d.size();", i, i)
		b.add("    } else {")
		b.add("        return s%d.size();", i)
		b.add("    }")
		b.add("}")
	case 1:
		b.add("pub fn calc%d(int a%d, int c%d) int {", i, i, i)
		b.add("    int acc%d = 0;", i)
		b.add("    int j%d = 0;", i)
		b.add("    while (j%d < a%d) {", i, i)
		b.add("        acc%d = acc%d + j%d * c%d;", i, i, i, i)
		b.add("        j%d = j%d + 1;", i, i)
		b.add("    }")
		b.add("    return acc%d;", i)
		b.add("}")
	case 2:
		b.add("pub fn join%d(String s%d, List<String> l%d) String {", i, i, i)
		b.add("    String out%d = s%d.trim();", i, i)
		b.add("    for (String w%d : l%d) {", i, i)
		b.add("        out%d = out%d + w%d;", i, i, i)
		b.add("    }")
		b.add("    return out%d;", i)
		b.add("}")
	}
	b.blank()
}

// cleanExtra 随机追加一段干净结构（while+break / for-in / try-catch / space 调用）。
func cleanExtra(b *srcBuilder, i int) {
	switch b.rng.Intn(4) {
	case 0:
		b.add("pub fn loop%d(List<int> l%d) int {", i, i)
		b.add("    int sum%d = 0;", i)
		b.add("    for (int v%d : l%d) {", i, i)
		b.add("        sum%d = sum%d + v%d;", i, i, i)
		b.add("    }")
		b.add("    return sum%d;", i)
		b.add("}")
		b.blank()
	case 1:
		b.add("pub fn guard%d(int n%d) int {", i, i)
		b.add("    int k%d = 0;", i)
		b.add("    while (k%d < n%d) {", i, i)
		b.add("        k%d = k%d + 1;", i, i)
		b.add("        if (k%d > 3) {", i)
		b.add("            break;")
		b.add("        }")
		b.add("    }")
		b.add("    return k%d;", i)
		b.add("}")
		b.blank()
	case 2:
		b.add("pub fn safe%d(int a%d, int c%d) int {", i, i, i)
		b.add("    try {")
		b.add("        return a%d / (c%d + 1);", i, i)
		b.add("    } catch (String e%d) {", i)
		b.add("        return a%d;", i)
		b.add("    }")
		b.add("}")
		b.blank()
	case 3:
		b.add("space {")
		b.add("    fn pick%d(int a%d, int c%d) int { return a%d; }", i, i, i, i)
		b.add("} sp%d;", i)
		b.blank()
		b.add("pub fn use%d(int n%d) int {", i, i)
		b.add("    return sp%d::pick%d(n%d, 1);", i, i, i)
		b.add("}")
		b.blank()
	case 4:
		b.add("pub type struct { int x%d; int y%d; } Pt%d;", i, i, i)
		b.blank()
		b.add("impl {")
		b.add("    fn sum%d(Pt%d self) int { return self.x%d + self.y%d; }", i, i, i, i)
		b.add("    fn scale%d(Pt%d self, int k%d) int { return (self.x%d + self.y%d) * k%d; }", i, i, i, i, i, i)
		b.add("} Pt%d;", i)
		b.blank()
		b.add("pub fn usePt%d(int a%d) int {", i, i)
		b.add("    Pt%d p%d = .{x: a%d, y: 1};", i, i, i)
		b.add("    return p%d.sum%d() + p%d.scale%d(2);", i, i, i, i)
		b.add("}")
		b.blank()
	case 5:
		b.add("type interface { fn val%d(Self self) int; } Val%d;", i, i)
		b.add("type struct { int v%d; } Holder%d;", i, i)
		b.add("impl { fn val%d(Holder%d self) int { return self.v%d; } } Holder%d;", i, i, i, i)
		b.blank()
		b.add("pub fn useIface%d(int a%d) int {", i, i)
		b.add("    Holder%d h%d = .{v: a%d};", i, i, i)
		b.add("    Val%d vv%d = h%d;", i, i, i)
		b.add("    return vv%d.val%d();", i, i)
		b.add("}")
		b.blank()
	case 6:
		b.add("pub fn table%d(int a%d) int {", i, i)
		b.add("    HashTable<String, int> h%d = HashTable::new();", i)
		b.add("    h%d.put(\"k\", a%d);", i, i)
		b.add("    List<int> l%d = [1, 2, 3];", i)
		b.add("    l%d.append(a%d);", i, i)
		b.add("    int total%d = 0;", i)
		b.add("    for (int v%d : l%d) {", i, i)
		b.add("        if (v%d > 1) {", i)
		b.add("            total%d = total%d + v%d;", i, i, i)
		b.add("        } else {")
		b.add("            total%d = total%d + 1;", i, i)
		b.add("        }")
		b.add("    }")
		b.add("    if (h%d.contains(\"k\")) {", i)
		b.add("        return total%d + h%d.get(\"k\");", i, i)
		b.add("    }")
		b.add("    return total%d;", i)
		b.add("}")
		b.blank()
	}
}

// shuffle 打乱声明顺序（缺陷不再总在同一位置）。
func shuffle[T any](rng *rand.Rand, xs []T) {
	for i := len(xs) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		xs[i], xs[j] = xs[j], xs[i]
	}
}

// ---------- 缺陷注入器 ----------

func caseClean(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("// 干净用例 %d", seq)
	b.blank()
	n := 1 + rng.Intn(3)
	for i := 0; i < n; i++ {
		cleanFunc(b, seq*10+i)
	}
	cleanExtra(b, seq*10+9)
	return statCase{name: "clean", src: b.String()}
}

func caseUnusedVar(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	cleanFunc(b, seq*10)
	b.add("pub fn holder%d(int a%d) int {", seq, seq)
	ln := b.add("    int unused%d = a%d + 1;", seq, seq)
	b.add("    return a%d;", seq)
	b.add("}")
	return statCase{name: "QK101", src: b.String(), want: []statWant{{CodeUnusedVar, ln}}}
}

func caseUnusedParam(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	// 未使用形参在**函数声明行**上（Param.Pos = 名字 token）
	ln := b.add("pub fn withParam%d(int a%d, int spare%d) int {", seq, seq, seq)
	b.add("    return a%d;", seq)
	b.add("}")
	return statCase{name: "QK102", src: b.String(), params: true, want: []statWant{{CodeUnusedParam, ln}}}
}

func caseUnreachable(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn early%d(int a%d) int {", seq, seq)
	b.add("    if (a%d > 0) {", seq)
	b.add("        return 1;")
	b.add("    } else {")
	b.add("        return 2;")
	b.add("    }")
	ln := b.add("    int dead%d = 3;", seq)
	b.add("    return dead%d;", seq)
	b.add("}")
	return statCase{name: "QK103", src: b.String(), want: []statWant{{CodeUnreachable, ln}}}
}

func caseShadow(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn shadow%d(List<int> l%d) int {", seq, seq)
	b.add("    int item%d = 9;", seq)
	ln := b.add("    for (int item%d : l%d) {", seq, seq)
	b.add("        item%d = item%d + 1;", seq, seq)
	b.add("    }")
	b.add("    return item%d;", seq)
	b.add("}")
	return statCase{name: "QK104", src: b.String(), want: []statWant{{CodeShadow, ln}}}
}

func caseMissingReturn(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	ln := b.add("pub fn maybe%d(int a%d) int {", seq, seq)
	b.add("    if (a%d > 0) {", seq)
	b.add("        return 1;")
	b.add("    }")
	b.add("}")
	return statCase{name: "QK105", src: b.String(), want: []statWant{{CodeMissingRet, ln}}}
}

func caseIfaceNearMiss(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("type interface { fn get%d(Self self) int; fn put%d(Self self, int v) void; } Store%d;", seq, seq, seq)
	b.add("type struct { int v%d; } Box%d;", seq, seq)
	ln := b.add("impl { fn get%d(Box%d self) int { return self.v%d; } } Box%d;", seq, seq, seq, seq)
	return statCase{name: "QK106", src: b.String(), want: []statWant{{CodeIfaceMissing, ln}}}
}

func caseVoidAsValue(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn noRet%d(int a%d) void {", seq, seq)
	b.add("    if (a%d > 0) {", seq)
	b.add("        return;")
	b.add("    }")
	b.add("}")
	b.blank()
	b.add("pub fn user%d(int a%d) int {", seq, seq)
	ln := b.add("    int y%d = noRet%d(a%d);", seq, seq, seq)
	b.add("    return y%d;", seq)
	b.add("}")
	return statCase{name: "QK107", src: b.String(), want: []statWant{{CodeVoidAsValue, ln}}}
}

func caseSelfAssign(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn selfA%d(int a%d) int {", seq, seq)
	ln := b.add("    a%d = a%d;", seq, seq)
	b.add("    return a%d;", seq)
	b.add("}")
	return statCase{name: "QK108", src: b.String(), want: []statWant{{CodeSelfAssign, ln}}}
}

func caseConstCond(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn constC%d(int a%d) int {", seq, seq)
	ln := b.add("    if (false) {")
	b.add("        return 99;")
	b.add("    }")
	b.add("    return a%d;", seq)
	b.add("}")
	return statCase{name: "QK109", src: b.String(), want: []statWant{{CodeConstCond, ln}}}
}

func caseDivZero(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn div%d(int a%d) int {", seq, seq)
	ln := b.add("    int z%d = 10 / 0;", seq)
	b.add("    return a%d + z%d;", seq, seq)
	b.add("}")
	return statCase{name: "QK110", src: b.String(), want: []statWant{{CodeDivZero, ln}}}
}

func caseUnusedImport(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	ln := b.add("import \"helper%d\";", seq)
	b.blank()
	b.add("pub fn local%d(int a%d) int {", seq, seq)
	b.add("    return a%d + 1;", seq)
	b.add("}")
	lib := fmt.Sprintf("program library;\n\npub fn helperFn%d(int n) int {\n    return n;\n}\n", seq)
	return statCase{name: "QK111", src: b.String(), libName: fmt.Sprintf("helper%d", seq), libSrc: lib,
		want: []statWant{{CodeUnusedImport, ln}}}
}

func caseShadowGlobal(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn double%d(int n) int {", seq)
	b.add("    return n * 2;")
	b.add("}")
	b.blank()
	b.add("pub fn user%d(int a%d) int {", seq, seq)
	ln := b.add("    int double%d = 3;", seq)
	b.add("    return a%d + double%d;", seq, seq)
	b.add("}")
	return statCase{name: "QK112", src: b.String(), want: []statWant{{CodeShadowGlobal, ln}}}
}

func caseDeadStore(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn store%d(int a%d) int {", seq, seq)
	ln := b.add("    int c%d = 1;", seq)
	b.add("    c%d = 2;", seq)
	b.add("    return a%d + c%d;", seq, seq)
	b.add("}")
	return statCase{name: "QK113", src: b.String(), want: []statWant{{CodeDeadStore, ln}}}
}

func caseSelfCompare(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program library;")
	b.blank()
	b.add("pub fn same%d(int a%d) int {", seq, seq)
	ln := b.add("    if (a%d == a%d) {", seq, seq)
	b.add("        return 1;")
	b.add("    }")
	b.add("    return 0;")
	b.add("}")
	return statCase{name: "QK114", src: b.String(), want: []statWant{{CodeSelfCompare, ln}}}
}

func caseUnusedFunc(rng *rand.Rand, seq int) statCase {
	b := &srcBuilder{rng: rng}
	b.add("program main;")
	b.blank()
	ln := b.add("fn never%d(int n) int {", seq)
	b.add("    return n * 3;")
	b.add("}")
	b.blank()
	b.add("fn main(IOStream io) void {")
	b.add("    io.println(%d);", seq)
	b.add("}")
	return statCase{name: "QK115", src: b.String(), want: []statWant{{CodeUnusedFunc, ln}}}
}

// 缺陷注入器表（每个都返回「真值已知」的用例）。
var statInjectors = []func(*rand.Rand, int) statCase{
	caseUnusedVar, caseUnusedParam, caseUnreachable, caseShadow, caseMissingReturn,
	caseIfaceNearMiss, caseVoidAsValue, caseSelfAssign, caseConstCond, caseDivZero,
	caseUnusedImport, caseShadowGlobal, caseDeadStore, caseSelfCompare, caseUnusedFunc,
}

// ---------- 度量与分位数 ----------

func quantile(sorted []int, q float64) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func quantileF(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func envIntStat(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// TestLintStatsP99 多轮随机化度量：误报率/漏报率的 P50/P90/P99 与最差样本。
func TestLintStatsP99(t *testing.T) {
	rounds := envIntStat("LINT_ROUNDS", 2000)
	seed := int64(envIntStat("LINT_SEED", 1))

	tmpRoot := t.TempDir() // 跨系统：临时目录由 testing 提供（仅 import 用例需要落盘）
	var fpCounts, fnCounts, tpCounts []int
	var fpRates, fnRates []float64
	worstFP, worstFN := 0, 0
	var samples []string

	for r := 0; r < rounds; r++ {
		rng := rand.New(rand.NewSource(seed + int64(r)))
		seq := r*1000 + 1
		// 每轮：干净用例 + 两个随机缺陷用例（缺陷种类轮转 → 覆盖全部 15 种），
		// 缺陷用例再随机穿插干净函数与注释，模拟真实文件的上下文。
		var cases []statCase
		cases = append(cases, caseClean(rng, seq), caseClean(rng, seq+1))
		for k := 0; k < 2; k++ {
			inj := statInjectors[(r*2+k)%len(statInjectors)]
			c := inj(rng, seq+2+k)
			c = decorate(rng, c)
			cases = append(cases, c)
		}
		roundDir := filepath.Join(tmpRoot, fmt.Sprintf("r%d", r))
		tp, fp, fn := 0, 0, 0
		for _, c := range cases {
			g, f, m, note := runStatCase(t, c, roundDir)
			tp += g
			fp += f
			fn += m
			if note != "" && len(samples) < 8 {
				samples = append(samples, note)
			}
		}
		fpCounts = append(fpCounts, fp)
		fnCounts = append(fnCounts, fn)
		tpCounts = append(tpCounts, tp)
		denFP := tp + fp
		denFN := tp + fn
		if denFP == 0 {
			denFP = 1
		}
		if denFN == 0 {
			denFN = 1
		}
		fpRates = append(fpRates, float64(fp)/float64(denFP)*100)
		fnRates = append(fnRates, float64(fn)/float64(denFN)*100)
		if fp > worstFP {
			worstFP = fp
		}
		if fn > worstFN {
			worstFN = fn
		}
	}

	sort.Ints(fpCounts)
	sort.Ints(fnCounts)
	sort.Ints(tpCounts)
	sort.Float64s(fpRates)
	sort.Float64s(fnRates)

	sum := func(xs []int) int {
		s := 0
		for _, x := range xs {
			s += x
		}
		return s
	}

	t.Logf("==== qkcheck 误报/漏报统计（%d 轮 × 4 用例/轮，seed=%d）====", rounds, seed)
	t.Logf("每轮命中 TP：P50=%d P99=%d（期望用例总数 %d）",
		quantile(tpCounts, 0.50), quantile(tpCounts, 0.99), sum(tpCounts))
	t.Logf("每轮误报 FP：P50=%d P90=%d P99=%d max=%d（合计 %d）",
		quantile(fpCounts, 0.50), quantile(fpCounts, 0.90), quantile(fpCounts, 0.99), worstFP, sum(fpCounts))
	t.Logf("每轮漏报 FN：P50=%d P90=%d P99=%d max=%d（合计 %d）",
		quantile(fnCounts, 0.50), quantile(fnCounts, 0.90), quantile(fnCounts, 0.99), worstFN, sum(fnCounts))
	t.Logf("误报率：P50=%.1f%% P90=%.1f%% P99=%.1f%% max=%.1f%%",
		quantileF(fpRates, 0.50), quantileF(fpRates, 0.90), quantileF(fpRates, 0.99), fpRates[len(fpRates)-1])
	t.Logf("漏报率：P50=%.1f%% P90=%.1f%% P99=%.1f%% max=%.1f%%",
		quantileF(fnRates, 0.50), quantileF(fnRates, 0.90), quantileF(fnRates, 0.99), fnRates[len(fnRates)-1])
	for _, s := range samples {
		t.Logf("样本：%s", s)
	}

	if os.Getenv("LINT_NOFAIL") != "" {
		return
	}
	if p99 := quantile(fpCounts, 0.99); p99 != 0 {
		t.Errorf("误报 P99 = %d（>0）：尾部仍有误报", p99)
	}
	if p99 := quantile(fnCounts, 0.99); p99 != 0 {
		t.Errorf("漏报 P99 = %d（>0）：尾部仍有漏报", p99)
	}
	if worstFP != 0 {
		t.Errorf("存在误报轮次（max=%d）", worstFP)
	}
	if worstFN != 0 {
		t.Errorf("存在漏报轮次（max=%d）", worstFN)
	}
}

// decorate 在缺陷用例前后插入随机数量的干净函数与注释（保持期望行号正确）。
func decorate(rng *rand.Rand, c statCase) statCase {
	lines := strings.Split(strings.TrimRight(c.src, "\n"), "\n")
	// 只有**库**用例可以插入辅助函数：main 程序里未被调用的函数本身就是死代码
	// （QK115 会如实报出，那是真问题，不该混进"误报"里）。
	isLib := strings.HasPrefix(c.src, "program library;")
	var pre []string
	if n := rng.Intn(3); n > 0 {
		for i := 0; i < n; i++ {
			pre = append(pre, "// 上下文注释 "+strconv.Itoa(rng.Intn(1000)))
			if isLib {
				pre = append(pre, "pub fn ctx"+strconv.Itoa(i)+strconv.Itoa(rng.Intn(100))+"(int a) int {")
				pre = append(pre, "    return a;")
				pre = append(pre, "}")
			}
			pre = append(pre, "")
		}
	}
	shift := len(pre) // 行号平移 = 实际插入的行数（别手算，容易错）
	lines = append(pre, lines...)
	src := strings.Join(lines, "\n") + "\n"
	for i := range c.want {
		c.want[i].line += shift
	}
	c.src = src
	return c
}

// runStatCase 跑一条统计用例，返回 (命中, 误报, 漏报, 样本说明)。
// 只有「未使用导入」用例需要真正落盘（其余直接用 AST，避免 8000 次文件写入）。
func runStatCase(t *testing.T, c statCase, dir string) (int, int, int, string) {
	t.Helper()
	file := filepath.Join(dir, "case.qk")
	libDirs := []string{dir}
	if c.libName != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, c.libName+".qk"), []byte(c.libSrc), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(c.src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	prog, err := ParseSource(c.src)
	if err != nil {
		// 生成器造出不可解析的代码属于**生成器缺陷**，直接暴露（不是被测工具的锅）
		return 0, 0, 0, fmt.Sprintf("生成器产出不可解析（%s）：%v", c.name, err)
	}
	diags := Lint(prog, LintOptions{Params: c.params, File: file, LibDirs: libDirs})
	got := map[string]bool{}
	for _, d := range diags {
		got[fmt.Sprintf("%s@%d", d.Code, d.Pos.Line)] = true
	}
	want := map[string]bool{}
	for _, w := range c.want {
		want[fmt.Sprintf("%s@%d", w.code, w.line)] = true
	}
	tp, fp, fn := 0, 0, 0
	var extra, missed []string
	for k := range got {
		if want[k] {
			tp++
		} else {
			fp++
			extra = append(extra, k)
		}
	}
	for k := range want {
		if !got[k] {
			fn++
			missed = append(missed, k)
		}
	}
	note := ""
	if fp > 0 || fn > 0 {
		sort.Strings(extra)
		sort.Strings(missed)
		note = fmt.Sprintf("%s 误报=%v 漏报=%v 源码首行=%q", c.name, extra, missed, firstLine(c.src))
	}
	return tp, fp, fn, note
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
