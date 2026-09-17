package lang

// qkcheck 基准：误报率 / 漏报率可复现测量。
//
// 基准集（testdata/lintbench/）：
//   - defects/*.qk：人工植入的已知缺陷，每处期望被某个诊断码在指定行报出
//   - clean/*.qk：人工复核的干净代码（含刻意不报的反例：try 内除零、while(true)+break、
//     重载里的 void、完整接口实现、被使用的导入）
//   - expect.txt：期望清单（`<相对路径> [params] <CODE@行,... | ->`）
//
// 口径：
//   TP = 报出且标注过的诊断；FP = 报出但未标注（含干净文件上的任何报出）；
//   FN = 标注了但没报出。
//   误报率 = FP/(TP+FP)；漏报率 = FN/(TP+FN)。
//
// 默认任何 FP 或 FN 都让测试失败（回归门禁）；测量模式：
//
//	LINTBENCH_NOFAIL=1 go test ./internal/lang/ -run TestLintBenchmark -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type benchWant struct {
	code string
	line int
}

func (w benchWant) String() string { return fmt.Sprintf("%s@%d", w.code, w.line) }

func loadBenchExpect(t *testing.T) (map[string][]benchWant, map[string]bool) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "lintbench", "expect.txt"))
	if err != nil {
		t.Fatalf("读取期望清单: %v", err)
	}
	want := map[string][]benchWant{}
	params := map[string]bool{}
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			t.Fatalf("期望清单格式错误: %q", ln)
		}
		path := fields[0]
		rest := fields[1:]
		for i, f := range rest {
			if f == "params" {
				params[path] = true
				rest = append(rest[:i], rest[i+1:]...)
				break
			}
		}
		if len(rest) != 1 {
			t.Fatalf("期望清单格式错误（应为 <路径> [params] <期望>）: %q", ln)
		}
		spec := rest[0]
		if spec == "-" {
			want[path] = nil
			continue
		}
		for _, item := range strings.Split(spec, ",") {
			code, lineStr, ok := strings.Cut(item, "@")
			if !ok {
				t.Fatalf("期望项格式错误（应为 CODE@行）: %q", item)
			}
			line, err := strconv.Atoi(lineStr)
			if err != nil {
				t.Fatalf("期望项行号错误 %q: %v", item, err)
			}
			want[path] = append(want[path], benchWant{code: code, line: line})
		}
	}
	return want, params
}

// runBenchCase 对单个基准文件跑一次检查。
func runBenchCase(t *testing.T, path string, withParams bool) []Diag {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	prog, err := ParseSource(string(src))
	if err != nil {
		t.Fatalf("%s 解析失败: %v", path, err)
	}
	dir := filepath.Dir(path)
	return Lint(prog, LintOptions{
		Params:  withParams,
		File:    path,
		LibDirs: []string{dir},
	})
}

func TestLintBenchmark(t *testing.T) {
	want, params := loadBenchExpect(t)
	paths := make([]string, 0, len(want))
	for p := range want {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	tp, fp, fn := 0, 0, 0
	type row struct {
		path              string
		got, want, missed []string
		extra             []string
	}
	var rows []row

	for _, rel := range paths {
		full := filepath.Join("testdata", "lintbench", rel)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("基准文件缺失: %s", rel)
			continue
		}
		diags := runBenchCase(t, full, params[rel])
		gotSet := map[string]bool{}
		for _, d := range diags {
			gotSet[fmt.Sprintf("%s@%d", d.Code, d.Pos.Line)] = true
		}
		wantSet := map[string]bool{}
		for _, w := range want[rel] {
			wantSet[w.String()] = true
		}
		r := row{path: rel}
		for k := range gotSet {
			r.got = append(r.got, k)
			if wantSet[k] {
				tp++
			} else {
				fp++
				r.extra = append(r.extra, k)
			}
		}
		for k := range wantSet {
			if !gotSet[k] {
				fn++
				r.missed = append(r.missed, k)
			}
		}
		sort.Strings(r.got)
		sort.Strings(r.extra)
		sort.Strings(r.missed)
		rows = append(rows, r)
	}

	// 报告
	t.Log("==== qkcheck 基准测量（testdata/lintbench）====")
	for _, r := range rows {
		status := "✔"
		switch {
		case len(r.extra) > 0 && len(r.missed) > 0:
			status = "✘ 误报+漏报"
		case len(r.extra) > 0:
			status = "✘ 误报"
		case len(r.missed) > 0:
			status = "✘ 漏报"
		}
		fmt.Printf("%s %-34s 报出=%-28s 误报=%-14s 漏报=%s\n",
			status, r.path, strings.Join(r.got, ","), strings.Join(r.extra, ","), strings.Join(r.missed, ","))
	}
	total := tp + fp + fn
	var fpRate, fnRate float64
	if tp+fp > 0 {
		fpRate = float64(fp) / float64(tp+fp) * 100
	}
	if tp+fn > 0 {
		fnRate = float64(fn) / float64(tp+fn) * 100
	}
	fmt.Printf("\n样本 %d 条期望 / 报出 %d 条：TP=%d FP=%d FN=%d ｜ 误报率=%.1f%% 漏报率=%.1f%% ｜ 用例 %d 个\n",
		total, tp+fp, tp, fp, fn, fpRate, fnRate, len(rows))

	if os.Getenv("LINTBENCH_NOFAIL") != "" {
		return
	}
	if fp > 0 {
		t.Errorf("存在误报 %d 条（误报率 %.1f%%）", fp, fpRate)
	}
	if fn > 0 {
		t.Errorf("存在漏报 %d 条（漏报率 %.1f%%）", fn, fnRate)
	}
}
