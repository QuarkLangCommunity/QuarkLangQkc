//go:build !windows

package lang

// **Local semantics regression** for the Windows FFI dispatch table (ffi_win.inc).
//
// Background: ffi_win.inc is the in-house, libffi-free ABI wrapper table (4 types × 0-4 parameters × 5 return types),
// only Windows + cgo compiles it, so if it breaks the problem cannot be hit locally at all. Here the same table is compiled on the **current platform**
// (with the _WIN32 guard and windows.h removed and the loader replaced by stubs), calling qk_ffi directly to verify the dispatch logic:
//   - return type coverage: void / i32 / f32 / f64 / ptr / **i64(long)**
//   - parameter type coverage: int32 / float / double / pointer
//   - key regression points: zero-argument non-void return, and code reaching 3e9~4e9 with 4 parameters (the C side must be 64-bit)
//
// A real defect fixed earlier: the table was generated with a 64-bit code but the C side used `int code` (unreachable with ≥4 parameters);
// and the i64 return family was missing (any C function returning long necessarily reported "ffi call failed" on Windows).
//
// Skipped automatically when there is no C compiler (so tests on other platforms are not blocked).

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const ffiTableHarness = `
#include <stdio.h>
#include <string.h>

static int      t_i32_0(void) { return 42; }
static double   t_f64_0(void) { return 2.5; }
static int32_t  t_i32_3(int32_t a, int32_t b, int32_t c) { return a * 100 + b * 10 + c; }
static double   t_f64_33(double x, double y) { return x * y; }
static int64_t  t_i64_1(int32_t a) { return (int64_t)a * 1000000000LL; }
static int64_t  t_i64_4(int32_t a, float b, double c, void* p) { return a + (int64_t)b + (int64_t)c + (p ? 1 : 0); }
static double   t_f64_4444(void* a, void* b, void* c, void* d) { return (a!=0)+(b!=0)+(c!=0)+(d!=0); }
static void*    t_ptr_4444(void* a, void* b, void* c, void* d) { return (void*)((char*)a + 3); }
static int32_t  t_i32_1111(int32_t a, int32_t b, int32_t c, int32_t d) { return a + b + c + d; }

static int fails = 0;
static void check_i(const char* name, int64_t got, int64_t want) {
    if (got != want) { printf("FAIL %s got=%lld want=%lld\n", name, (long long)got, (long long)want); fails++; }
    else             { printf("ok   %s %lld\n", name, (long long)got); }
}
static void check_f(const char* name, double got, double want) {
    double d = got - want; if (d < 0) d = -d;
    if (d > 1e-9) { printf("FAIL %s got=%g want=%g\n", name, got, want); fails++; }
    else          { printf("ok   %s %g\n", name, got); }
}

int main(void) {
    int64_t ri; double rf; void* rs; int rc;
    { int ty[1]={1}; double nu[1]={7}; void* pt[1]={0};
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_i64_1, ty, nu, pt, 1, 6, &ri, &rf, &rs);
      check_i("i64 + int 参数", rc==0 ? ri : -1, 7000000000LL); }
    { int ty[1]={0}; double nu[1]={0}; void* pt[1]={0};
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_i32_0, ty, nu, pt, 0, 1, &ri, &rf, &rs);
      check_i("i32 零参数", rc==0 ? ri : -999, 42);
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_f64_0, ty, nu, pt, 0, 3, &ri, &rf, &rs);
      check_f("f64 零参数", rc==0 ? rf : -999, 2.5); }
    { int ty[3]={1,1,1}; double nu[3]={1,2,3}; void* pt[3]={0,0,0};
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_i32_3, ty, nu, pt, 3, 1, &ri, &rf, &rs);
      check_i("i32 + 3×int", rc==0 ? ri : -999, 123); }
    { int ty[2]={3,3}; double nu[2]={6.0,7.0}; void* pt[2]={0,0};
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_f64_33, ty, nu, pt, 2, 3, &ri, &rf, &rs);
      check_f("f64 + 2×double", rc==0 ? rf : -999, 42.0); }
    { int ty[4]={1,2,3,4}; double nu[4]={1,2.0,3.0,0}; void* pt[4]={0,0,0,(void*)1};
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_i64_4, ty, nu, pt, 4, 6, &ri, &rf, &rs);
      check_i("i64 + 4 参数混合", rc==0 ? ri : -999, 7); }
    { int ty[4]={4,4,4,4}; double nu[4]={0,0,0,0}; void* pt[4]={(void*)1,(void*)1,0,0};
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_f64_4444, ty, nu, pt, 4, 3, &ri, &rf, &rs);
      check_f("f64 + 4×ptr（code≈3e9）", rc==0 ? rf : -999, 2.0);
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_ptr_4444, ty, nu, pt, 4, 4, &ri, &rf, &rs);
      check_i("ptr + 4×ptr（code≈4e9）", rc==0 ? (int64_t)(intptr_t)rs : -999, 4); }
    { int ty[4]={1,1,1,1}; double nu[4]={1,2,3,4}; void* pt[4]={0,0,0,0};
      ri=0; rf=0; rs=0; rc=qk_ffi((void*)t_i32_1111, ty, nu, pt, 4, 1, &ri, &rf, &rs);
      check_i("i32 + 4×int", rc==0 ? ri : -999, 10); }
    if (fails) { printf("%d 项失败\n", fails); return 1; }
    printf("全部通过\n");
    return 0;
}
`

func TestWindowsFFITableDispatch(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		if cc, err = exec.LookPath("gcc"); err != nil {
			t.Skip("无 C 编译器（cc/gcc），跳过 Windows FFI 派发表验证")
		}
	}
	raw, err := os.ReadFile("ffi_win.inc")
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	// strip the Windows-specific parts so that the same table compiles on the current platform
	src = strings.Replace(src, "#ifdef _WIN32\n", "", 1)
	src = strings.Replace(src, "#include <windows.h>\n", "", 1)
	src = regexp.MustCompile(`(?s)void\* qk_dlopen\(const char\* n\) \{.*?\n`).ReplaceAllString(src, "void* qk_dlopen(const char* n) { (void)n; return 0; }\n")
	src = regexp.MustCompile(`(?s)void\* qk_dlsym\(void\* h, const char\* s\) \{.*?\n`).ReplaceAllString(src, "void* qk_dlsym(void* h, const char* s) { (void)h; (void)s; return 0; }\n")
	src = regexp.MustCompile(`(?s)const char\* qk_dlerror\(void\) \{.*?\n`).ReplaceAllString(src, "const char* qk_dlerror(void) { return \"stub\"; }\n")
	if i := strings.LastIndex(src, "#endif"); i >= 0 {
		src = src[:i]
	}
	dir := t.TempDir()
	cf := filepath.Join(dir, "table_test.c")
	bin := filepath.Join(dir, "table_test")
	if err := os.WriteFile(cf, []byte(src+ffiTableHarness), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(cc, "-O1", "-w", "-o", bin, cf).CombinedOutput()
	if err != nil {
		t.Fatalf("编译派发表失败: %v\n%s", err, out)
	}
	run, err := exec.Command(bin).CombinedOutput()
	got := strings.TrimSpace(string(run))
	t.Logf("派发表用例输出：\n%s", got)
	if err != nil {
		t.Fatalf("派发表用例失败: %v", err)
	}
	if !strings.Contains(got, "全部通过") {
		t.Fatalf("期望全部通过，实际：%s", got)
	}
}
