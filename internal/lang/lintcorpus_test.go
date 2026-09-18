package lang

// qkcheck 语料扰动不变性：对**真实源文件**做「不改变语义」的改写，诊断集合必须保持一致。
//
// 覆盖两类真实风险：
//  1. 位置稳健性：插入注释/空行后，诊断行号必须精确平移（编辑器与 CI 都靠这个定位）；
//  2. 跨系统：CRLF（Windows 换行）与行尾空白不应改变任何诊断——这是跨平台一致性的核心用例。
//
// 与 lintbench（人工标注）互补：那边验证「该报的报、不该报的不报」，
// 这边验证「同样的代码换个写法结论不变」。

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type diagKey struct {
	code string
	line int
}

func diagSet(t *testing.T, src string) []diagKey {
	t.Helper()
	prog, err := ParseSource(src)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	var out []diagKey
	for _, d := range Lint(prog, LintOptions{}) {
		out = append(out, diagKey{d.Code, d.Pos.Line})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].line != out[j].line {
			return out[i].line < out[j].line
		}
		return out[i].code < out[j].code
	})
	return out
}

func sameDiags(a, b []diagKey) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func corpusFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	root := filepath.Join("..", "..")
	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			switch fi.Name() {
			case ".git", "dist", "dist-ci", "node_modules", "lintbench", "bench":
				// lintbench 是人工标注基准（由 lintstat/lintbench 两个测试负责）；
				// bench/tools 是性能夹具（含第三方库副本）
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".qk") || strings.HasSuffix(p, ".kq") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func TestLintCorpusPerturbationInvariance(t *testing.T) {
	files := corpusFiles(t)
	if len(files) < 5 {
		t.Skipf("语料过少（%d）", len(files))
	}
	checked := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		if _, err := ParseSource(src); err != nil {
			t.Logf("跳过不可解析文件 %s：%v", f, err)
			continue
		}
		base := diagSet(t, src)

		// ① CRLF（Windows 换行）：诊断必须逐条相同
		if got := diagSet(t, strings.ReplaceAll(src, "\n", "\r\n")); !sameDiags(got, base) {
			t.Errorf("%s：CRLF 后诊断变化\n base=%v\n  crlf=%v", f, base, got)
		}
		// ② 行尾注释：不改变任何诊断（注释由词法层剥离，位置不动）
		var withComments []string
		for _, ln := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
			if strings.TrimSpace(ln) == "" || strings.Contains(ln, "//") || strings.Contains(ln, "/*") {
				withComments = append(withComments, ln)
				continue
			}
			withComments = append(withComments, ln+" // 扰动注释")
		}
		if got := diagSet(t, strings.Join(withComments, "\n")+"\n"); !sameDiags(got, base) {
			t.Errorf("%s：加行尾注释后诊断变化\n base=%v\n  pert=%v", f, base, got)
		}
		// ③ 头部插入 3 行注释：诊断行号整体 +3
		head := "// 扰动头部 1\n// 扰动头部 2\n// 扰动头部 3\n" + src
		want := make([]diagKey, len(base))
		for i, d := range base {
			want[i] = diagKey{d.code, d.line + 3}
		}
		if got := diagSet(t, head); !sameDiags(got, want) {
			t.Errorf("%s：头部插注释后行号未按预期平移\n want=%v\n  got=%v", f, want, got)
		}
		// ④ 行尾空白 + 文件末尾空行：诊断不变
		var padded []string
		for _, ln := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
			padded = append(padded, ln+"   \t")
		}
		if got := diagSet(t, strings.Join(padded, "\n")+"\n\n\n"); !sameDiags(got, base) {
			t.Errorf("%s：行尾空白/末尾空行后诊断变化\n base=%v\n  pert=%v", f, base, got)
		}
		checked++
	}
	t.Logf("语料扰动不变性通过：%d 个文件 × 4 种扰动（CRLF / 行尾注释 / 头部插行 / 行尾空白）", checked)
	if checked < 5 {
		t.Errorf("实际校验文件过少：%d", checked)
	}
}

// TestLintCRLFPositions 单点核对：CRLF 下 token 行列号与 LF 完全一致（跨系统一致性）。
func TestLintCRLFPositions(t *testing.T) {
	src := "program library;\n\npub fn f(int a) int {\n    int unused = 1;\n    return a;\n}\n"
	lf := diagSet(t, src)
	crlf := diagSet(t, strings.ReplaceAll(src, "\n", "\r\n"))
	if !sameDiags(lf, crlf) {
		t.Fatalf("CRLF 与 LF 诊断不一致：lf=%v crlf=%v", lf, crlf)
	}
	if len(lf) != 1 || lf[0].code != CodeUnusedVar || lf[0].line != 4 {
		t.Fatalf("期望 QK101 在第 4 行，got %v", lf)
	}
	// 顺带核对词法层：CRLF 不产生额外 token、行列号一致
	toksLF, err := Lex(src)
	if err != nil {
		t.Fatal(err)
	}
	toksCRLF, err := Lex(strings.ReplaceAll(src, "\n", "\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(toksLF) != len(toksCRLF) {
		t.Fatalf("token 数不同：LF=%d CRLF=%d", len(toksLF), len(toksCRLF))
	}
	for i := range toksLF {
		if toksLF[i].Kind != toksCRLF[i].Kind || toksLF[i].Line != toksCRLF[i].Line || toksLF[i].Col != toksCRLF[i].Col {
			t.Fatalf("第 %d 个 token 在 CRLF 下不一致：LF=%+v CRLF=%+v", i, toksLF[i], toksCRLF[i])
		}
	}
}
