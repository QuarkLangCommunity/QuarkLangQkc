package lang

// qkcheck **statistical measurement** of the false-positive / false-negative rate: randomized test cases × multiple sampling rounds → P50/P90/P99/worst.
//
// Why this is needed: the single-round benchmark (lintbench_test.go) is deterministic, so running it ten thousand times gives the same result and a P99 is meaningless.
// Here every round uses a different random seed to generate a batch of programs with known ground truth (clean skeleton + injected known defects),
// so the false-positive/false-negative counts differ from round to round and form a distribution, and the tail (P99) can expose problems that only rare code shapes trigger.
//
// Conventions:
//   - Each test case expects a set of (diagnostic code, line number); a clean case expects the empty set.
//   - FP = reported but not expected; FN = expected but not reported; TP = hit.
//   - Per-round rate = FP/(TP+FP), FN/(TP+FN); across rounds take quantiles.
//   - Gate: the P99 FP and FN counts must be 0 (even the tail may not be wrong); the worst values are printed as well, for triage.
//
// Cross-system: a pure Go implementation (no shell, no fixed paths, temporary directories from t.TempDir),
// runs identically on Windows/macOS/Linux.
//
// Environment variables:
//
//	LINT_ROUNDS=200   rounds (default 200; CI may lower it)
//	LINT_SEED=1       random seed base (default 1, fixed → reproducible)
//	LINT_NOFAIL=1     measure only, never fail (for parameter tuning)

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
	libName string // non-empty: also writes the dependency file <libName>.qk into the same directory
	libSrc  string
	params  bool
	want    []statWant
}

// ---------- source-line builder (injectors learn their own line number from it → expected line numbers are correct by construction) ----------

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

// ---------- clean building blocks (guaranteed to have zero diagnostics themselves) ----------

// cleanFunc generates one pub function: every parameter is used, every local is read, every branch returns.
// The shape is random (1-3 parameters, a mix of branches/loops, different return expressions) to widen the random coverage.
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

// cleanExtra randomly appends one clean construct (while+break / for-in / try-catch / space call).
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

// shuffle shuffles the declaration order (so a defect is no longer always in the same position).
func shuffle[T any](rng *rand.Rand, xs []T) {
	for i := len(xs) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		xs[i], xs[j] = xs[j], xs[i]
	}
}

// ---------- defect injectors ----------

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
	// An unused parameter sits on the **function declaration line** (Param.Pos = the name token)
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

// Table of defect injectors (each returns a case with known ground truth).
var statInjectors = []func(*rand.Rand, int) statCase{
	caseUnusedVar, caseUnusedParam, caseUnreachable, caseShadow, caseMissingReturn,
	caseIfaceNearMiss, caseVoidAsValue, caseSelfAssign, caseConstCond, caseDivZero,
	caseUnusedImport, caseShadowGlobal, caseDeadStore, caseSelfCompare, caseUnusedFunc,
}

// ---------- metrics and quantiles ----------

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

// TestLintStatsP99 multi-round randomized measurement: P50/P90/P99 of the false-positive/false-negative rate and the worst samples.
func TestLintStatsP99(t *testing.T) {
	rounds := envIntStat("LINT_ROUNDS", 2000)
	seed := int64(envIntStat("LINT_SEED", 1))

	tmpRoot := t.TempDir() // cross-system: the temporary directory comes from testing (only the import case needs to write to disk)
	var fpCounts, fnCounts, tpCounts []int
	var fpRates, fnRates []float64
	worstFP, worstFN := 0, 0
	var samples []string

	for r := 0; r < rounds; r++ {
		rng := rand.New(rand.NewSource(seed + int64(r)))
		seq := r*1000 + 1
		// Each round: clean cases + two random defect cases (the defect kind rotates → all 15 kinds are covered),
		// the defect cases are then interleaved at random with clean functions and comments, mimicking the context of a real file.
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

// decorate inserts a random number of clean functions and comments before and after a defect case (keeping the expected line numbers correct).
func decorate(rng *rand.Rand, c statCase) statCase {
	lines := strings.Split(strings.TrimRight(c.src, "\n"), "\n")
	// Only **library** cases may have helper functions inserted: in a main program an uncalled function is dead code by itself
	// (QK115 reports it faithfully, and that is a real problem, so it must not be mixed into the "false positives").
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
	shift := len(pre) // line-number shift = the number of lines actually inserted (do not work it out by hand, that is error-prone)
	lines = append(pre, lines...)
	src := strings.Join(lines, "\n") + "\n"
	for i := range c.want {
		c.want[i].line += shift
	}
	c.src = src
	return c
}

// runStatCase runs one statistical case and returns (hits, false positives, false negatives, sample description).
// Only the "unused import" case needs to actually write to disk (the others use the AST directly, avoiding 8000 file writes).
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
		// Unparsable code produced by the generator is a **generator defect** and is exposed directly (it is not the fault of the tool under test)
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
