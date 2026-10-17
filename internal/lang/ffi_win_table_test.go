//go:build !windows

package lang

// Local semantics regression for the Windows FFI call shim (ffi_win.inc).
//
// The shim is compiled here on an x86-64 POSIX host: the file selects its System V twin when _WIN32 is
// not defined, so the same argument classifier (register file, stack slots, return decoding) runs
// natively. That makes the logic which used to reject a fifth argument testable without Windows, and
// the Windows CI leg runs the same dispatcher through the real Microsoft x64 thunk.
//
// The cases are the ones the old generated table could not express: a six-argument call (the shape
// Cleg's cleg_image has), a seventeen-argument call, a mixed int/float call whose arguments cross both
// register files, the 32-argument bound, and every return family.

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const ffiShimHarness = `
#include <stdio.h>

static int h6(int a, int b, int c, int d, int e, int f) {
    return a + b * 10 + c * 100 + d * 1000 + e * 10000 + f * 100000;
}
static long h17(int a1, int a2, int a3, int a4, int a5, int a6, int a7, int a8, int a9,
                int a10, int a11, int a12, int a13, int a14, int a15, int a16, int a17) {
    return a1 + 2*a2 + 3*a3 + 4*a4 + 5*a5 + 6*a6 + 7*a7 + 8*a8 + 9*a9 + 10*a10 +
           11*a11 + 12*a12 + 13*a13 + 14*a14 + 15*a15 + 16*a16 + 17*a17;
}
static double hmix(double a, int b, float c, double d, int e, float f, double g, int h,
                   double i, int j, float k, double l, int m, int n, int o, int p, int q, double r) {
    return a + b + c + d + e + f + g + h + i + j + k + l + m + n + o + p + q + r;
}
static long h32(int a1, int a2, int a3, int a4, int a5, int a6, int a7, int a8, int a9,
                int a10, int a11, int a12, int a13, int a14, int a15, int a16, int a17, int a18,
                int a19, int a20, int a21, int a22, int a23, int a24, int a25, int a26, int a27,
                int a28, int a29, int a30, int a31, int a32) {
    return a1 + 2*a2 + 3*a3 + 4*a4 + 5*a5 + 6*a6 + 7*a7 + 8*a8 + 9*a9 + 10*a10 +
           11*a11 + 12*a12 + 13*a13 + 14*a14 + 15*a15 + 16*a16 + 17*a17 + 18*a18 +
           19*a19 + 20*a20 + 21*a21 + 22*a22 + 23*a23 + 24*a24 + 25*a25 + 26*a26 +
           27*a27 + 28*a28 + 29*a29 + 30*a30 + 31*a31 + 32*a32;
}
static int64_t h_i64(void) { return 0x123456789LL; }
static float h_f32(void) { return 3.5f; }
static double h_f64(void) { return 2.25; }
static void* h_ptr(void) { return (void*)0x1234; }
static int h_i32(void) { return -7; }

static int fails = 0;
static void check_i(const char* name, long long got, long long want) {
    if (got != want) { printf("FAIL %s got=%lld want=%lld\n", name, got, want); fails++; }
    else            { printf("ok   %s\n", name); }
}
static void check_d(const char* name, double got, double want) {
    double d = got - want; if (d < 0) d = -d;
    if (d > 1e-9) { printf("FAIL %s got=%g want=%g\n", name, got, want); fails++; }
    else          { printf("ok   %s\n", name); }
}

int main(void) {
    int64_t ri = 0; double rf = 0; void* rp = 0; int rc;

    { int ty[6] = {1,1,1,1,1,1}; double nu[6] = {1,2,3,4,5,6}; void* pt[6] = {0};
      rc = qk_ffi((void*)h6, ty, nu, pt, 6, 1, &ri, &rf, &rp);
      check_i("six int arguments", rc ? -999 : ri, h6(1,2,3,4,5,6)); }

    /* The Microsoft x64 layout, checked off-Windows: position, not per-class order, chooses the register. */
    { int ty[18] = {3,1,2,3,1,2,3,1,3,1,2,3,1,1,1,1,1,3};
      static const int want_where[18] = {1,0,1,1,2,2,2,2,2,2,2,2,2,2,2,2,2,2};
      static const int want_slot[18]  = {0,1,2,3,0,1,2,3,4,5,6,7,8,9,10,11,12,13};
      int where[18], slot[18], ni = 0, nf = 0, ns = 0, ok = 1, k;
      for (k = 0; k < 18; k++) {
          where[k] = qk_lay_out(1, k, qk_is_float(ty[k]), &ni, &nf, &ns, &slot[k]);
          if (where[k] != want_where[k] || slot[k] != want_slot[k]) ok = 0;
      }
      check_i("Microsoft x64 positional layout", ok, 1); }

    { int ty[17]; double nu[17]; void* pt[17] = {0};
      for (int i = 0; i < 17; i++) { ty[i] = 1; nu[i] = i + 1; }
      rc = qk_ffi((void*)h17, ty, nu, pt, 17, 6, &ri, &rf, &rp);
      check_i("seventeen int arguments", rc ? -999 : ri, h17(1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17)); }

    { int ty[18] = {3,1,2,3,1,2,3,1,3,1,2,3,1,1,1,1,1,3};
      double nu[18] = {1.0,2,3.0f,4.0,5,6.0f,7.0,8,9.0,10,11.0f,12.0,13,14,15,16,17,18.0};
      void* pt[18] = {0};
      rc = qk_ffi((void*)hmix, ty, nu, pt, 18, 3, &ri, &rf, &rp);
      check_d("mixed registers and stack", rc ? -999 : rf,
              hmix(1.0,2,3.0f,4.0,5,6.0f,7.0,8,9.0,10,11.0f,12.0,13,14,15,16,17,18.0)); }

    { int ty[32]; double nu[32]; void* pt[32] = {0}; int a[32];
      for (int i = 0; i < 32; i++) { ty[i] = 1; nu[i] = a[i] = i + 1; }
      rc = qk_ffi((void*)h32, ty, nu, pt, 32, 6, &ri, &rf, &rp);
      check_i("thirty-two arguments (bound)", rc ? -999 : ri,
              h32(a[0],a[1],a[2],a[3],a[4],a[5],a[6],a[7],a[8],a[9],a[10],a[11],a[12],a[13],a[14],
                  a[15],a[16],a[17],a[18],a[19],a[20],a[21],a[22],a[23],a[24],a[25],a[26],a[27],
                  a[28],a[29],a[30],a[31])); }

    { int ty[33] = {0}; double nu[33] = {0}; void* pt[33] = {0};
      rc = qk_ffi((void*)h6, ty, nu, pt, 33, 1, &ri, &rf, &rp);
      check_i("over the bound is rejected", rc, -2); }

    { int ty[0]; double nu[0]; void* pt[0];
      rc = qk_ffi((void*)h_i64, ty, nu, pt, 0, 6, &ri, &rf, &rp);
      check_i("i64 return", rc ? -999 : ri, 0x123456789LL); }
    { int ty[0]; double nu[0]; void* pt[0];
      rc = qk_ffi((void*)h_f32, ty, nu, pt, 0, 2, &ri, &rf, &rp);
      check_d("f32 return", rc ? -999 : rf, 3.5); }
    { int ty[0]; double nu[0]; void* pt[0];
      rc = qk_ffi((void*)h_f64, ty, nu, pt, 0, 3, &ri, &rf, &rp);
      check_d("f64 return", rc ? -999 : rf, 2.25); }
    { int ty[0]; double nu[0]; void* pt[0];
      rc = qk_ffi((void*)h_ptr, ty, nu, pt, 0, 4, &ri, &rf, &rp);
      check_i("pointer return", rc ? -999 : (long long)(intptr_t)rp, 0x1234); }
    { int ty[0]; double nu[0]; void* pt[0];
      rc = qk_ffi((void*)h_i32, ty, nu, pt, 0, 1, &ri, &rf, &rp);
      check_i("i32 return sign", rc ? 999 : ri, -7); }

    if (fails) { printf("%d cases failed\n", fails); return 1; }
    printf("all passed\n");
    return 0;
}
`

func TestWindowsFFIShimDispatch(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("the shim's off-Windows twin is the x86-64 System V convention")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		if cc, err = exec.LookPath("gcc"); err != nil {
			t.Skip("no C compiler (cc/gcc), skipping the Windows FFI shim check")
		}
	}
	raw, err := os.ReadFile("ffi_win.inc")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cf := filepath.Join(dir, "shim_test.c")
	bin := filepath.Join(dir, "shim_test")
	src := string(raw) + ffiShimHarness
	if err := os.WriteFile(cf, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(cc, "-O1", "-w", "-o", bin, cf).CombinedOutput()
	if err != nil {
		t.Fatalf("compiling the shim failed: %v\n%s", err, out)
	}
	run, err := exec.Command(bin).CombinedOutput()
	got := strings.TrimSpace(string(run))
	t.Logf("shim case output:\n%s", got)
	if err != nil {
		t.Fatalf("a shim case failed: %v", err)
	}
	if !strings.Contains(got, "all passed") {
		t.Fatalf("every case should pass, but the results are: %s", got)
	}
}
