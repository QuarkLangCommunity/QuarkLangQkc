package cgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// parityCases 返回 compiler/testdata/cases/*.kq（测试工作目录是 internal/cgen）。
// 这些用例只用 libc + 模块内 ql_strcat，可用 lli 直接执行。
func parityCases(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "testdata", "cases", "*.kq"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no parity cases found: %v", err)
	}
	return files
}

// parityRunCases 返回 compiler/testdata/cases_run/*.kq：需要 clang 链接（library FFI
// 要 -lm/-lc，taskm 要 qthreads 运行时），因此只做 IR 语法校验；端到端由
// compiler/testdata/compare.sh（qkc -run）与解释器逐字节对比。
func parityRunCases(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "testdata", "cases_run", "*.kq"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no run parity cases found: %v", err)
	}
	return files
}

// TestParityCaseIRSyntax 校验每个对比用例生成的 IR 都通过 LLVM 官方语法检查。
func TestParityCaseIRSyntax(t *testing.T) {
	llvmAs, err := exec.LookPath("llvm-as")
	if err != nil {
		t.Skip("llvm-as not available")
	}
	all := append(parityCases(t), parityRunCases(t)...)
	for _, f := range all {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			ir, err := Transpile(string(src), f)
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			tmp, err := os.CreateTemp("", "quark-parity-*.ll")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmp.Name())
			if _, err := tmp.WriteString(ir); err != nil {
				t.Fatal(err)
			}
			tmp.Close()
			if out, err := exec.Command(llvmAs, tmp.Name(), "-o", os.DevNull).CombinedOutput(); err != nil {
				t.Fatalf("llvm-as rejected IR: %v\n%s\nIR:\n%s", err, out, ir)
			}
		})
	}
}

// TestParityCaseOutput 用 lli 执行每个用例，与解释器产出的 .out 逐字节对比。
// .out 由解释器生成（compiler/testdata/compare.sh 会重新核对两者一致）。
func TestParityCaseOutput(t *testing.T) {
	if _, err := exec.LookPath("lli"); err != nil {
		t.Skip("lli not available")
	}
	for _, f := range parityCases(t) {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(f, ".kq") + ".out")
			if err != nil {
				t.Fatalf("missing expected output fixture: %v", err)
			}
			ir, err := Transpile(string(src), f)
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			got := lliRun(t, withTestRuntime(ir))
			if got != string(want) {
				t.Fatalf("output mismatch\n got: %q\nwant: %q", got, string(want))
			}
		})
	}
}
