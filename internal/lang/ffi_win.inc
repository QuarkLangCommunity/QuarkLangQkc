#ifdef _WIN32
#include <windows.h>
#include <stdint.h>
void* qk_dlopen(const char* n) { return (void*)LoadLibraryA(n); }
void* qk_dlsym(void* h, const char* s) { return (void*)GetProcAddress((HMODULE)h, s); }
const char* qk_dlerror(void) { return "LoadLibrary/GetProcAddress failed (win32)"; }

static void qk_w_void_0(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)())f)();
}

static int32_t qk_w_i32_0(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)())f)();
}

static float qk_w_f32_0(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)())f)();
}

static double qk_w_f64_0(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)())f)();
}

static void* qk_w_ptr_0(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)())f)();
}

static void qk_w_void_1(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t))f)((int32_t)a1);
}

static int32_t qk_w_i32_1(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t))f)((int32_t)a1);
}

static float qk_w_f32_1(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t))f)((int32_t)a1);
}

static double qk_w_f64_1(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t))f)((int32_t)a1);
}

static void* qk_w_ptr_1(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t))f)((int32_t)a1);
}

static void qk_w_void_2(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float))f)((float)a1);
}

static int32_t qk_w_i32_2(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float))f)((float)a1);
}

static float qk_w_f32_2(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float))f)((float)a1);
}

static double qk_w_f64_2(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float))f)((float)a1);
}

static void* qk_w_ptr_2(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float))f)((float)a1);
}

static void qk_w_void_3(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double))f)(a1);
}

static int32_t qk_w_i32_3(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double))f)(a1);
}

static float qk_w_f32_3(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double))f)(a1);
}

static double qk_w_f64_3(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double))f)(a1);
}

static void* qk_w_ptr_3(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double))f)(a1);
}

static void qk_w_void_4(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*))f)(p1);
}

static int32_t qk_w_i32_4(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*))f)(p1);
}

static float qk_w_f32_4(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*))f)(p1);
}

static double qk_w_f64_4(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*))f)(p1);
}

static void* qk_w_ptr_4(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*))f)(p1);
}

static void qk_w_void_11(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t))f)((int32_t)a1, (int32_t)a2);
}

static int32_t qk_w_i32_11(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t))f)((int32_t)a1, (int32_t)a2);
}

static float qk_w_f32_11(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t))f)((int32_t)a1, (int32_t)a2);
}

static double qk_w_f64_11(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t))f)((int32_t)a1, (int32_t)a2);
}

static void* qk_w_ptr_11(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t))f)((int32_t)a1, (int32_t)a2);
}

static void qk_w_void_12(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float))f)((int32_t)a1, (float)a2);
}

static int32_t qk_w_i32_12(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float))f)((int32_t)a1, (float)a2);
}

static float qk_w_f32_12(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float))f)((int32_t)a1, (float)a2);
}

static double qk_w_f64_12(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float))f)((int32_t)a1, (float)a2);
}

static void* qk_w_ptr_12(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float))f)((int32_t)a1, (float)a2);
}

static void qk_w_void_13(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double))f)((int32_t)a1, a2);
}

static int32_t qk_w_i32_13(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double))f)((int32_t)a1, a2);
}

static float qk_w_f32_13(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double))f)((int32_t)a1, a2);
}

static double qk_w_f64_13(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double))f)((int32_t)a1, a2);
}

static void* qk_w_ptr_13(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double))f)((int32_t)a1, a2);
}

static void qk_w_void_14(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*))f)((int32_t)a1, p2);
}

static int32_t qk_w_i32_14(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*))f)((int32_t)a1, p2);
}

static float qk_w_f32_14(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*))f)((int32_t)a1, p2);
}

static double qk_w_f64_14(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*))f)((int32_t)a1, p2);
}

static void* qk_w_ptr_14(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*))f)((int32_t)a1, p2);
}

static void qk_w_void_21(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t))f)((float)a1, (int32_t)a2);
}

static int32_t qk_w_i32_21(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t))f)((float)a1, (int32_t)a2);
}

static float qk_w_f32_21(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t))f)((float)a1, (int32_t)a2);
}

static double qk_w_f64_21(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t))f)((float)a1, (int32_t)a2);
}

static void* qk_w_ptr_21(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t))f)((float)a1, (int32_t)a2);
}

static void qk_w_void_22(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float))f)((float)a1, (float)a2);
}

static int32_t qk_w_i32_22(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float))f)((float)a1, (float)a2);
}

static float qk_w_f32_22(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float))f)((float)a1, (float)a2);
}

static double qk_w_f64_22(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float))f)((float)a1, (float)a2);
}

static void* qk_w_ptr_22(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float))f)((float)a1, (float)a2);
}

static void qk_w_void_23(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double))f)((float)a1, a2);
}

static int32_t qk_w_i32_23(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double))f)((float)a1, a2);
}

static float qk_w_f32_23(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double))f)((float)a1, a2);
}

static double qk_w_f64_23(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double))f)((float)a1, a2);
}

static void* qk_w_ptr_23(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double))f)((float)a1, a2);
}

static void qk_w_void_24(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*))f)((float)a1, p2);
}

static int32_t qk_w_i32_24(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*))f)((float)a1, p2);
}

static float qk_w_f32_24(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*))f)((float)a1, p2);
}

static double qk_w_f64_24(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*))f)((float)a1, p2);
}

static void* qk_w_ptr_24(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*))f)((float)a1, p2);
}

static void qk_w_void_31(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t))f)(a1, (int32_t)a2);
}

static int32_t qk_w_i32_31(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t))f)(a1, (int32_t)a2);
}

static float qk_w_f32_31(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t))f)(a1, (int32_t)a2);
}

static double qk_w_f64_31(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t))f)(a1, (int32_t)a2);
}

static void* qk_w_ptr_31(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t))f)(a1, (int32_t)a2);
}

static void qk_w_void_32(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float))f)(a1, (float)a2);
}

static int32_t qk_w_i32_32(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float))f)(a1, (float)a2);
}

static float qk_w_f32_32(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float))f)(a1, (float)a2);
}

static double qk_w_f64_32(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float))f)(a1, (float)a2);
}

static void* qk_w_ptr_32(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float))f)(a1, (float)a2);
}

static void qk_w_void_33(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double))f)(a1, a2);
}

static int32_t qk_w_i32_33(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double))f)(a1, a2);
}

static float qk_w_f32_33(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double))f)(a1, a2);
}

static double qk_w_f64_33(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double))f)(a1, a2);
}

static void* qk_w_ptr_33(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double))f)(a1, a2);
}

static void qk_w_void_34(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*))f)(a1, p2);
}

static int32_t qk_w_i32_34(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*))f)(a1, p2);
}

static float qk_w_f32_34(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*))f)(a1, p2);
}

static double qk_w_f64_34(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*))f)(a1, p2);
}

static void* qk_w_ptr_34(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*))f)(a1, p2);
}

static void qk_w_void_41(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t))f)(p1, (int32_t)a2);
}

static int32_t qk_w_i32_41(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t))f)(p1, (int32_t)a2);
}

static float qk_w_f32_41(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t))f)(p1, (int32_t)a2);
}

static double qk_w_f64_41(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t))f)(p1, (int32_t)a2);
}

static void* qk_w_ptr_41(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t))f)(p1, (int32_t)a2);
}

static void qk_w_void_42(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float))f)(p1, (float)a2);
}

static int32_t qk_w_i32_42(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float))f)(p1, (float)a2);
}

static float qk_w_f32_42(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float))f)(p1, (float)a2);
}

static double qk_w_f64_42(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float))f)(p1, (float)a2);
}

static void* qk_w_ptr_42(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float))f)(p1, (float)a2);
}

static void qk_w_void_43(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double))f)(p1, a2);
}

static int32_t qk_w_i32_43(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double))f)(p1, a2);
}

static float qk_w_f32_43(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double))f)(p1, a2);
}

static double qk_w_f64_43(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double))f)(p1, a2);
}

static void* qk_w_ptr_43(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double))f)(p1, a2);
}

static void qk_w_void_44(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*))f)(p1, p2);
}

static int32_t qk_w_i32_44(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*))f)(p1, p2);
}

static float qk_w_f32_44(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*))f)(p1, p2);
}

static double qk_w_f64_44(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*))f)(p1, p2);
}

static void* qk_w_ptr_44(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*))f)(p1, p2);
}

static void qk_w_void_111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3);
}

static int32_t qk_w_i32_111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3);
}

static float qk_w_f32_111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3);
}

static double qk_w_f64_111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3);
}

static void* qk_w_ptr_111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3);
}

static void qk_w_void_112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (float)a3);
}

static int32_t qk_w_i32_112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (float)a3);
}

static float qk_w_f32_112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (float)a3);
}

static double qk_w_f64_112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (float)a3);
}

static void* qk_w_ptr_112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (float)a3);
}

static void qk_w_void_113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, a3);
}

static int32_t qk_w_i32_113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, a3);
}

static float qk_w_f32_113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, a3);
}

static double qk_w_f64_113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, a3);
}

static void* qk_w_ptr_113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, a3);
}

static void qk_w_void_114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, p3);
}

static int32_t qk_w_i32_114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, p3);
}

static float qk_w_f32_114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, p3);
}

static double qk_w_f64_114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, p3);
}

static void* qk_w_ptr_114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, p3);
}

static void qk_w_void_121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3);
}

static int32_t qk_w_i32_121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3);
}

static float qk_w_f32_121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3);
}

static double qk_w_f64_121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3);
}

static void* qk_w_ptr_121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3);
}

static void qk_w_void_122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, float))f)((int32_t)a1, (float)a2, (float)a3);
}

static int32_t qk_w_i32_122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, float))f)((int32_t)a1, (float)a2, (float)a3);
}

static float qk_w_f32_122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, float))f)((int32_t)a1, (float)a2, (float)a3);
}

static double qk_w_f64_122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, float))f)((int32_t)a1, (float)a2, (float)a3);
}

static void* qk_w_ptr_122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, float))f)((int32_t)a1, (float)a2, (float)a3);
}

static void qk_w_void_123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, double))f)((int32_t)a1, (float)a2, a3);
}

static int32_t qk_w_i32_123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, double))f)((int32_t)a1, (float)a2, a3);
}

static float qk_w_f32_123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, double))f)((int32_t)a1, (float)a2, a3);
}

static double qk_w_f64_123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, double))f)((int32_t)a1, (float)a2, a3);
}

static void* qk_w_ptr_123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, double))f)((int32_t)a1, (float)a2, a3);
}

static void qk_w_void_124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, void*))f)((int32_t)a1, (float)a2, p3);
}

static int32_t qk_w_i32_124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, void*))f)((int32_t)a1, (float)a2, p3);
}

static float qk_w_f32_124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, void*))f)((int32_t)a1, (float)a2, p3);
}

static double qk_w_f64_124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, void*))f)((int32_t)a1, (float)a2, p3);
}

static void* qk_w_ptr_124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, void*))f)((int32_t)a1, (float)a2, p3);
}

static void qk_w_void_131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, int32_t))f)((int32_t)a1, a2, (int32_t)a3);
}

static int32_t qk_w_i32_131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, int32_t))f)((int32_t)a1, a2, (int32_t)a3);
}

static float qk_w_f32_131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, int32_t))f)((int32_t)a1, a2, (int32_t)a3);
}

static double qk_w_f64_131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, int32_t))f)((int32_t)a1, a2, (int32_t)a3);
}

static void* qk_w_ptr_131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, int32_t))f)((int32_t)a1, a2, (int32_t)a3);
}

static void qk_w_void_132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, float))f)((int32_t)a1, a2, (float)a3);
}

static int32_t qk_w_i32_132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, float))f)((int32_t)a1, a2, (float)a3);
}

static float qk_w_f32_132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, float))f)((int32_t)a1, a2, (float)a3);
}

static double qk_w_f64_132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, float))f)((int32_t)a1, a2, (float)a3);
}

static void* qk_w_ptr_132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, float))f)((int32_t)a1, a2, (float)a3);
}

static void qk_w_void_133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, double))f)((int32_t)a1, a2, a3);
}

static int32_t qk_w_i32_133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, double))f)((int32_t)a1, a2, a3);
}

static float qk_w_f32_133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, double))f)((int32_t)a1, a2, a3);
}

static double qk_w_f64_133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, double))f)((int32_t)a1, a2, a3);
}

static void* qk_w_ptr_133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, double))f)((int32_t)a1, a2, a3);
}

static void qk_w_void_134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, void*))f)((int32_t)a1, a2, p3);
}

static int32_t qk_w_i32_134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, void*))f)((int32_t)a1, a2, p3);
}

static float qk_w_f32_134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, void*))f)((int32_t)a1, a2, p3);
}

static double qk_w_f64_134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, void*))f)((int32_t)a1, a2, p3);
}

static void* qk_w_ptr_134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, void*))f)((int32_t)a1, a2, p3);
}

static void qk_w_void_141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, int32_t))f)((int32_t)a1, p2, (int32_t)a3);
}

static int32_t qk_w_i32_141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, int32_t))f)((int32_t)a1, p2, (int32_t)a3);
}

static float qk_w_f32_141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, int32_t))f)((int32_t)a1, p2, (int32_t)a3);
}

static double qk_w_f64_141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, int32_t))f)((int32_t)a1, p2, (int32_t)a3);
}

static void* qk_w_ptr_141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, int32_t))f)((int32_t)a1, p2, (int32_t)a3);
}

static void qk_w_void_142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, float))f)((int32_t)a1, p2, (float)a3);
}

static int32_t qk_w_i32_142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, float))f)((int32_t)a1, p2, (float)a3);
}

static float qk_w_f32_142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, float))f)((int32_t)a1, p2, (float)a3);
}

static double qk_w_f64_142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, float))f)((int32_t)a1, p2, (float)a3);
}

static void* qk_w_ptr_142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, float))f)((int32_t)a1, p2, (float)a3);
}

static void qk_w_void_143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, double))f)((int32_t)a1, p2, a3);
}

static int32_t qk_w_i32_143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, double))f)((int32_t)a1, p2, a3);
}

static float qk_w_f32_143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, double))f)((int32_t)a1, p2, a3);
}

static double qk_w_f64_143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, double))f)((int32_t)a1, p2, a3);
}

static void* qk_w_ptr_143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, double))f)((int32_t)a1, p2, a3);
}

static void qk_w_void_144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, void*))f)((int32_t)a1, p2, p3);
}

static int32_t qk_w_i32_144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, void*))f)((int32_t)a1, p2, p3);
}

static float qk_w_f32_144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, void*))f)((int32_t)a1, p2, p3);
}

static double qk_w_f64_144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, void*))f)((int32_t)a1, p2, p3);
}

static void* qk_w_ptr_144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, void*))f)((int32_t)a1, p2, p3);
}

static void qk_w_void_211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3);
}

static int32_t qk_w_i32_211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3);
}

static float qk_w_f32_211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3);
}

static double qk_w_f64_211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3);
}

static void* qk_w_ptr_211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3);
}

static void qk_w_void_212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, float))f)((float)a1, (int32_t)a2, (float)a3);
}

static int32_t qk_w_i32_212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, float))f)((float)a1, (int32_t)a2, (float)a3);
}

static float qk_w_f32_212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, float))f)((float)a1, (int32_t)a2, (float)a3);
}

static double qk_w_f64_212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, float))f)((float)a1, (int32_t)a2, (float)a3);
}

static void* qk_w_ptr_212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, float))f)((float)a1, (int32_t)a2, (float)a3);
}

static void qk_w_void_213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, double))f)((float)a1, (int32_t)a2, a3);
}

static int32_t qk_w_i32_213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, double))f)((float)a1, (int32_t)a2, a3);
}

static float qk_w_f32_213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, double))f)((float)a1, (int32_t)a2, a3);
}

static double qk_w_f64_213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, double))f)((float)a1, (int32_t)a2, a3);
}

static void* qk_w_ptr_213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, double))f)((float)a1, (int32_t)a2, a3);
}

static void qk_w_void_214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, void*))f)((float)a1, (int32_t)a2, p3);
}

static int32_t qk_w_i32_214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, void*))f)((float)a1, (int32_t)a2, p3);
}

static float qk_w_f32_214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, void*))f)((float)a1, (int32_t)a2, p3);
}

static double qk_w_f64_214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, void*))f)((float)a1, (int32_t)a2, p3);
}

static void* qk_w_ptr_214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, void*))f)((float)a1, (int32_t)a2, p3);
}

static void qk_w_void_221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, int32_t))f)((float)a1, (float)a2, (int32_t)a3);
}

static int32_t qk_w_i32_221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, int32_t))f)((float)a1, (float)a2, (int32_t)a3);
}

static float qk_w_f32_221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, int32_t))f)((float)a1, (float)a2, (int32_t)a3);
}

static double qk_w_f64_221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, int32_t))f)((float)a1, (float)a2, (int32_t)a3);
}

static void* qk_w_ptr_221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, int32_t))f)((float)a1, (float)a2, (int32_t)a3);
}

static void qk_w_void_222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, float))f)((float)a1, (float)a2, (float)a3);
}

static int32_t qk_w_i32_222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, float))f)((float)a1, (float)a2, (float)a3);
}

static float qk_w_f32_222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, float))f)((float)a1, (float)a2, (float)a3);
}

static double qk_w_f64_222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, float))f)((float)a1, (float)a2, (float)a3);
}

static void* qk_w_ptr_222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, float))f)((float)a1, (float)a2, (float)a3);
}

static void qk_w_void_223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, double))f)((float)a1, (float)a2, a3);
}

static int32_t qk_w_i32_223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, double))f)((float)a1, (float)a2, a3);
}

static float qk_w_f32_223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, double))f)((float)a1, (float)a2, a3);
}

static double qk_w_f64_223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, double))f)((float)a1, (float)a2, a3);
}

static void* qk_w_ptr_223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, double))f)((float)a1, (float)a2, a3);
}

static void qk_w_void_224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, void*))f)((float)a1, (float)a2, p3);
}

static int32_t qk_w_i32_224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, void*))f)((float)a1, (float)a2, p3);
}

static float qk_w_f32_224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, void*))f)((float)a1, (float)a2, p3);
}

static double qk_w_f64_224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, void*))f)((float)a1, (float)a2, p3);
}

static void* qk_w_ptr_224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, void*))f)((float)a1, (float)a2, p3);
}

static void qk_w_void_231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, int32_t))f)((float)a1, a2, (int32_t)a3);
}

static int32_t qk_w_i32_231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, int32_t))f)((float)a1, a2, (int32_t)a3);
}

static float qk_w_f32_231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, int32_t))f)((float)a1, a2, (int32_t)a3);
}

static double qk_w_f64_231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, int32_t))f)((float)a1, a2, (int32_t)a3);
}

static void* qk_w_ptr_231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, int32_t))f)((float)a1, a2, (int32_t)a3);
}

static void qk_w_void_232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, float))f)((float)a1, a2, (float)a3);
}

static int32_t qk_w_i32_232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, float))f)((float)a1, a2, (float)a3);
}

static float qk_w_f32_232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, float))f)((float)a1, a2, (float)a3);
}

static double qk_w_f64_232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, float))f)((float)a1, a2, (float)a3);
}

static void* qk_w_ptr_232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, float))f)((float)a1, a2, (float)a3);
}

static void qk_w_void_233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, double))f)((float)a1, a2, a3);
}

static int32_t qk_w_i32_233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, double))f)((float)a1, a2, a3);
}

static float qk_w_f32_233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, double))f)((float)a1, a2, a3);
}

static double qk_w_f64_233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, double))f)((float)a1, a2, a3);
}

static void* qk_w_ptr_233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, double))f)((float)a1, a2, a3);
}

static void qk_w_void_234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, void*))f)((float)a1, a2, p3);
}

static int32_t qk_w_i32_234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, void*))f)((float)a1, a2, p3);
}

static float qk_w_f32_234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, void*))f)((float)a1, a2, p3);
}

static double qk_w_f64_234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, void*))f)((float)a1, a2, p3);
}

static void* qk_w_ptr_234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, void*))f)((float)a1, a2, p3);
}

static void qk_w_void_241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, int32_t))f)((float)a1, p2, (int32_t)a3);
}

static int32_t qk_w_i32_241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, int32_t))f)((float)a1, p2, (int32_t)a3);
}

static float qk_w_f32_241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, int32_t))f)((float)a1, p2, (int32_t)a3);
}

static double qk_w_f64_241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, int32_t))f)((float)a1, p2, (int32_t)a3);
}

static void* qk_w_ptr_241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, int32_t))f)((float)a1, p2, (int32_t)a3);
}

static void qk_w_void_242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, float))f)((float)a1, p2, (float)a3);
}

static int32_t qk_w_i32_242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, float))f)((float)a1, p2, (float)a3);
}

static float qk_w_f32_242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, float))f)((float)a1, p2, (float)a3);
}

static double qk_w_f64_242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, float))f)((float)a1, p2, (float)a3);
}

static void* qk_w_ptr_242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, float))f)((float)a1, p2, (float)a3);
}

static void qk_w_void_243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, double))f)((float)a1, p2, a3);
}

static int32_t qk_w_i32_243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, double))f)((float)a1, p2, a3);
}

static float qk_w_f32_243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, double))f)((float)a1, p2, a3);
}

static double qk_w_f64_243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, double))f)((float)a1, p2, a3);
}

static void* qk_w_ptr_243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, double))f)((float)a1, p2, a3);
}

static void qk_w_void_244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, void*))f)((float)a1, p2, p3);
}

static int32_t qk_w_i32_244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, void*))f)((float)a1, p2, p3);
}

static float qk_w_f32_244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, void*))f)((float)a1, p2, p3);
}

static double qk_w_f64_244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, void*))f)((float)a1, p2, p3);
}

static void* qk_w_ptr_244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, void*))f)((float)a1, p2, p3);
}

static void qk_w_void_311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3);
}

static int32_t qk_w_i32_311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3);
}

static float qk_w_f32_311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3);
}

static double qk_w_f64_311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3);
}

static void* qk_w_ptr_311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3);
}

static void qk_w_void_312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, float))f)(a1, (int32_t)a2, (float)a3);
}

static int32_t qk_w_i32_312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, float))f)(a1, (int32_t)a2, (float)a3);
}

static float qk_w_f32_312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, float))f)(a1, (int32_t)a2, (float)a3);
}

static double qk_w_f64_312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, float))f)(a1, (int32_t)a2, (float)a3);
}

static void* qk_w_ptr_312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, float))f)(a1, (int32_t)a2, (float)a3);
}

static void qk_w_void_313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, double))f)(a1, (int32_t)a2, a3);
}

static int32_t qk_w_i32_313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, double))f)(a1, (int32_t)a2, a3);
}

static float qk_w_f32_313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, double))f)(a1, (int32_t)a2, a3);
}

static double qk_w_f64_313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, double))f)(a1, (int32_t)a2, a3);
}

static void* qk_w_ptr_313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, double))f)(a1, (int32_t)a2, a3);
}

static void qk_w_void_314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, void*))f)(a1, (int32_t)a2, p3);
}

static int32_t qk_w_i32_314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, void*))f)(a1, (int32_t)a2, p3);
}

static float qk_w_f32_314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, void*))f)(a1, (int32_t)a2, p3);
}

static double qk_w_f64_314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, void*))f)(a1, (int32_t)a2, p3);
}

static void* qk_w_ptr_314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, void*))f)(a1, (int32_t)a2, p3);
}

static void qk_w_void_321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, int32_t))f)(a1, (float)a2, (int32_t)a3);
}

static int32_t qk_w_i32_321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, int32_t))f)(a1, (float)a2, (int32_t)a3);
}

static float qk_w_f32_321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, int32_t))f)(a1, (float)a2, (int32_t)a3);
}

static double qk_w_f64_321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, int32_t))f)(a1, (float)a2, (int32_t)a3);
}

static void* qk_w_ptr_321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, int32_t))f)(a1, (float)a2, (int32_t)a3);
}

static void qk_w_void_322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, float))f)(a1, (float)a2, (float)a3);
}

static int32_t qk_w_i32_322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, float))f)(a1, (float)a2, (float)a3);
}

static float qk_w_f32_322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, float))f)(a1, (float)a2, (float)a3);
}

static double qk_w_f64_322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, float))f)(a1, (float)a2, (float)a3);
}

static void* qk_w_ptr_322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, float))f)(a1, (float)a2, (float)a3);
}

static void qk_w_void_323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, double))f)(a1, (float)a2, a3);
}

static int32_t qk_w_i32_323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, double))f)(a1, (float)a2, a3);
}

static float qk_w_f32_323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, double))f)(a1, (float)a2, a3);
}

static double qk_w_f64_323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, double))f)(a1, (float)a2, a3);
}

static void* qk_w_ptr_323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, double))f)(a1, (float)a2, a3);
}

static void qk_w_void_324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, void*))f)(a1, (float)a2, p3);
}

static int32_t qk_w_i32_324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, void*))f)(a1, (float)a2, p3);
}

static float qk_w_f32_324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, void*))f)(a1, (float)a2, p3);
}

static double qk_w_f64_324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, void*))f)(a1, (float)a2, p3);
}

static void* qk_w_ptr_324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, void*))f)(a1, (float)a2, p3);
}

static void qk_w_void_331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, int32_t))f)(a1, a2, (int32_t)a3);
}

static int32_t qk_w_i32_331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, int32_t))f)(a1, a2, (int32_t)a3);
}

static float qk_w_f32_331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, int32_t))f)(a1, a2, (int32_t)a3);
}

static double qk_w_f64_331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, int32_t))f)(a1, a2, (int32_t)a3);
}

static void* qk_w_ptr_331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, int32_t))f)(a1, a2, (int32_t)a3);
}

static void qk_w_void_332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, float))f)(a1, a2, (float)a3);
}

static int32_t qk_w_i32_332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, float))f)(a1, a2, (float)a3);
}

static float qk_w_f32_332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, float))f)(a1, a2, (float)a3);
}

static double qk_w_f64_332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, float))f)(a1, a2, (float)a3);
}

static void* qk_w_ptr_332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, float))f)(a1, a2, (float)a3);
}

static void qk_w_void_333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, double))f)(a1, a2, a3);
}

static int32_t qk_w_i32_333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, double))f)(a1, a2, a3);
}

static float qk_w_f32_333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, double))f)(a1, a2, a3);
}

static double qk_w_f64_333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, double))f)(a1, a2, a3);
}

static void* qk_w_ptr_333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, double))f)(a1, a2, a3);
}

static void qk_w_void_334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, void*))f)(a1, a2, p3);
}

static int32_t qk_w_i32_334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, void*))f)(a1, a2, p3);
}

static float qk_w_f32_334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, void*))f)(a1, a2, p3);
}

static double qk_w_f64_334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, void*))f)(a1, a2, p3);
}

static void* qk_w_ptr_334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, void*))f)(a1, a2, p3);
}

static void qk_w_void_341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, int32_t))f)(a1, p2, (int32_t)a3);
}

static int32_t qk_w_i32_341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, int32_t))f)(a1, p2, (int32_t)a3);
}

static float qk_w_f32_341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, int32_t))f)(a1, p2, (int32_t)a3);
}

static double qk_w_f64_341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, int32_t))f)(a1, p2, (int32_t)a3);
}

static void* qk_w_ptr_341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, int32_t))f)(a1, p2, (int32_t)a3);
}

static void qk_w_void_342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, float))f)(a1, p2, (float)a3);
}

static int32_t qk_w_i32_342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, float))f)(a1, p2, (float)a3);
}

static float qk_w_f32_342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, float))f)(a1, p2, (float)a3);
}

static double qk_w_f64_342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, float))f)(a1, p2, (float)a3);
}

static void* qk_w_ptr_342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, float))f)(a1, p2, (float)a3);
}

static void qk_w_void_343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, double))f)(a1, p2, a3);
}

static int32_t qk_w_i32_343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, double))f)(a1, p2, a3);
}

static float qk_w_f32_343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, double))f)(a1, p2, a3);
}

static double qk_w_f64_343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, double))f)(a1, p2, a3);
}

static void* qk_w_ptr_343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, double))f)(a1, p2, a3);
}

static void qk_w_void_344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, void*))f)(a1, p2, p3);
}

static int32_t qk_w_i32_344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, void*))f)(a1, p2, p3);
}

static float qk_w_f32_344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, void*))f)(a1, p2, p3);
}

static double qk_w_f64_344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, void*))f)(a1, p2, p3);
}

static void* qk_w_ptr_344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, void*))f)(a1, p2, p3);
}

static void qk_w_void_411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3);
}

static int32_t qk_w_i32_411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3);
}

static float qk_w_f32_411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3);
}

static double qk_w_f64_411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3);
}

static void* qk_w_ptr_411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3);
}

static void qk_w_void_412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, float))f)(p1, (int32_t)a2, (float)a3);
}

static int32_t qk_w_i32_412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, float))f)(p1, (int32_t)a2, (float)a3);
}

static float qk_w_f32_412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, float))f)(p1, (int32_t)a2, (float)a3);
}

static double qk_w_f64_412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, float))f)(p1, (int32_t)a2, (float)a3);
}

static void* qk_w_ptr_412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, float))f)(p1, (int32_t)a2, (float)a3);
}

static void qk_w_void_413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, double))f)(p1, (int32_t)a2, a3);
}

static int32_t qk_w_i32_413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, double))f)(p1, (int32_t)a2, a3);
}

static float qk_w_f32_413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, double))f)(p1, (int32_t)a2, a3);
}

static double qk_w_f64_413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, double))f)(p1, (int32_t)a2, a3);
}

static void* qk_w_ptr_413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, double))f)(p1, (int32_t)a2, a3);
}

static void qk_w_void_414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, void*))f)(p1, (int32_t)a2, p3);
}

static int32_t qk_w_i32_414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, void*))f)(p1, (int32_t)a2, p3);
}

static float qk_w_f32_414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, void*))f)(p1, (int32_t)a2, p3);
}

static double qk_w_f64_414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, void*))f)(p1, (int32_t)a2, p3);
}

static void* qk_w_ptr_414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, void*))f)(p1, (int32_t)a2, p3);
}

static void qk_w_void_421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, int32_t))f)(p1, (float)a2, (int32_t)a3);
}

static int32_t qk_w_i32_421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, int32_t))f)(p1, (float)a2, (int32_t)a3);
}

static float qk_w_f32_421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, int32_t))f)(p1, (float)a2, (int32_t)a3);
}

static double qk_w_f64_421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, int32_t))f)(p1, (float)a2, (int32_t)a3);
}

static void* qk_w_ptr_421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, int32_t))f)(p1, (float)a2, (int32_t)a3);
}

static void qk_w_void_422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, float))f)(p1, (float)a2, (float)a3);
}

static int32_t qk_w_i32_422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, float))f)(p1, (float)a2, (float)a3);
}

static float qk_w_f32_422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, float))f)(p1, (float)a2, (float)a3);
}

static double qk_w_f64_422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, float))f)(p1, (float)a2, (float)a3);
}

static void* qk_w_ptr_422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, float))f)(p1, (float)a2, (float)a3);
}

static void qk_w_void_423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, double))f)(p1, (float)a2, a3);
}

static int32_t qk_w_i32_423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, double))f)(p1, (float)a2, a3);
}

static float qk_w_f32_423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, double))f)(p1, (float)a2, a3);
}

static double qk_w_f64_423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, double))f)(p1, (float)a2, a3);
}

static void* qk_w_ptr_423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, double))f)(p1, (float)a2, a3);
}

static void qk_w_void_424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, void*))f)(p1, (float)a2, p3);
}

static int32_t qk_w_i32_424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, void*))f)(p1, (float)a2, p3);
}

static float qk_w_f32_424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, void*))f)(p1, (float)a2, p3);
}

static double qk_w_f64_424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, void*))f)(p1, (float)a2, p3);
}

static void* qk_w_ptr_424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, void*))f)(p1, (float)a2, p3);
}

static void qk_w_void_431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, int32_t))f)(p1, a2, (int32_t)a3);
}

static int32_t qk_w_i32_431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, int32_t))f)(p1, a2, (int32_t)a3);
}

static float qk_w_f32_431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, int32_t))f)(p1, a2, (int32_t)a3);
}

static double qk_w_f64_431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, int32_t))f)(p1, a2, (int32_t)a3);
}

static void* qk_w_ptr_431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, int32_t))f)(p1, a2, (int32_t)a3);
}

static void qk_w_void_432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, float))f)(p1, a2, (float)a3);
}

static int32_t qk_w_i32_432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, float))f)(p1, a2, (float)a3);
}

static float qk_w_f32_432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, float))f)(p1, a2, (float)a3);
}

static double qk_w_f64_432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, float))f)(p1, a2, (float)a3);
}

static void* qk_w_ptr_432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, float))f)(p1, a2, (float)a3);
}

static void qk_w_void_433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, double))f)(p1, a2, a3);
}

static int32_t qk_w_i32_433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, double))f)(p1, a2, a3);
}

static float qk_w_f32_433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, double))f)(p1, a2, a3);
}

static double qk_w_f64_433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, double))f)(p1, a2, a3);
}

static void* qk_w_ptr_433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, double))f)(p1, a2, a3);
}

static void qk_w_void_434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, void*))f)(p1, a2, p3);
}

static int32_t qk_w_i32_434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, void*))f)(p1, a2, p3);
}

static float qk_w_f32_434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, void*))f)(p1, a2, p3);
}

static double qk_w_f64_434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, void*))f)(p1, a2, p3);
}

static void* qk_w_ptr_434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, void*))f)(p1, a2, p3);
}

static void qk_w_void_441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, int32_t))f)(p1, p2, (int32_t)a3);
}

static int32_t qk_w_i32_441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, int32_t))f)(p1, p2, (int32_t)a3);
}

static float qk_w_f32_441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, int32_t))f)(p1, p2, (int32_t)a3);
}

static double qk_w_f64_441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, int32_t))f)(p1, p2, (int32_t)a3);
}

static void* qk_w_ptr_441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, int32_t))f)(p1, p2, (int32_t)a3);
}

static void qk_w_void_442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, float))f)(p1, p2, (float)a3);
}

static int32_t qk_w_i32_442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, float))f)(p1, p2, (float)a3);
}

static float qk_w_f32_442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, float))f)(p1, p2, (float)a3);
}

static double qk_w_f64_442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, float))f)(p1, p2, (float)a3);
}

static void* qk_w_ptr_442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, float))f)(p1, p2, (float)a3);
}

static void qk_w_void_443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, double))f)(p1, p2, a3);
}

static int32_t qk_w_i32_443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, double))f)(p1, p2, a3);
}

static float qk_w_f32_443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, double))f)(p1, p2, a3);
}

static double qk_w_f64_443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, double))f)(p1, p2, a3);
}

static void* qk_w_ptr_443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, double))f)(p1, p2, a3);
}

static void qk_w_void_444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, void*))f)(p1, p2, p3);
}

static int32_t qk_w_i32_444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, void*))f)(p1, p2, p3);
}

static float qk_w_f32_444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, void*))f)(p1, p2, p3);
}

static double qk_w_f64_444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, void*))f)(p1, p2, p3);
}

static void* qk_w_ptr_444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, void*))f)(p1, p2, p3);
}

static void qk_w_void_1111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_1111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_1111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_1111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_1111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_1112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_1112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_1112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_1112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_1112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_1113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_1113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, a4);
}

static float qk_w_f32_1113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, a4);
}

static double qk_w_f64_1113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_1113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, a4);
}

static void qk_w_void_1114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_1114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, p4);
}

static float qk_w_f32_1114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, p4);
}

static double qk_w_f64_1114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_1114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, p4);
}

static void qk_w_void_1121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, float, int32_t))f)((int32_t)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_1121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, float, int32_t))f)((int32_t)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_1121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, float, int32_t))f)((int32_t)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_1121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, float, int32_t))f)((int32_t)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_1121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, float, int32_t))f)((int32_t)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_1122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, float, float))f)((int32_t)a1, (int32_t)a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_1122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, float, float))f)((int32_t)a1, (int32_t)a2, (float)a3, (float)a4);
}

static float qk_w_f32_1122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, float, float))f)((int32_t)a1, (int32_t)a2, (float)a3, (float)a4);
}

static double qk_w_f64_1122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, float, float))f)((int32_t)a1, (int32_t)a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_1122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, float, float))f)((int32_t)a1, (int32_t)a2, (float)a3, (float)a4);
}

static void qk_w_void_1123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, float, double))f)((int32_t)a1, (int32_t)a2, (float)a3, a4);
}

static int32_t qk_w_i32_1123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, float, double))f)((int32_t)a1, (int32_t)a2, (float)a3, a4);
}

static float qk_w_f32_1123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, float, double))f)((int32_t)a1, (int32_t)a2, (float)a3, a4);
}

static double qk_w_f64_1123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, float, double))f)((int32_t)a1, (int32_t)a2, (float)a3, a4);
}

static void* qk_w_ptr_1123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, float, double))f)((int32_t)a1, (int32_t)a2, (float)a3, a4);
}

static void qk_w_void_1124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, float, void*))f)((int32_t)a1, (int32_t)a2, (float)a3, p4);
}

static int32_t qk_w_i32_1124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, float, void*))f)((int32_t)a1, (int32_t)a2, (float)a3, p4);
}

static float qk_w_f32_1124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, float, void*))f)((int32_t)a1, (int32_t)a2, (float)a3, p4);
}

static double qk_w_f64_1124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, float, void*))f)((int32_t)a1, (int32_t)a2, (float)a3, p4);
}

static void* qk_w_ptr_1124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, float, void*))f)((int32_t)a1, (int32_t)a2, (float)a3, p4);
}

static void qk_w_void_1131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, double, int32_t))f)((int32_t)a1, (int32_t)a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_1131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, double, int32_t))f)((int32_t)a1, (int32_t)a2, a3, (int32_t)a4);
}

static float qk_w_f32_1131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, double, int32_t))f)((int32_t)a1, (int32_t)a2, a3, (int32_t)a4);
}

static double qk_w_f64_1131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, double, int32_t))f)((int32_t)a1, (int32_t)a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_1131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, double, int32_t))f)((int32_t)a1, (int32_t)a2, a3, (int32_t)a4);
}

static void qk_w_void_1132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, double, float))f)((int32_t)a1, (int32_t)a2, a3, (float)a4);
}

static int32_t qk_w_i32_1132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, double, float))f)((int32_t)a1, (int32_t)a2, a3, (float)a4);
}

static float qk_w_f32_1132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, double, float))f)((int32_t)a1, (int32_t)a2, a3, (float)a4);
}

static double qk_w_f64_1132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, double, float))f)((int32_t)a1, (int32_t)a2, a3, (float)a4);
}

static void* qk_w_ptr_1132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, double, float))f)((int32_t)a1, (int32_t)a2, a3, (float)a4);
}

static void qk_w_void_1133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, double, double))f)((int32_t)a1, (int32_t)a2, a3, a4);
}

static int32_t qk_w_i32_1133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, double, double))f)((int32_t)a1, (int32_t)a2, a3, a4);
}

static float qk_w_f32_1133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, double, double))f)((int32_t)a1, (int32_t)a2, a3, a4);
}

static double qk_w_f64_1133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, double, double))f)((int32_t)a1, (int32_t)a2, a3, a4);
}

static void* qk_w_ptr_1133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, double, double))f)((int32_t)a1, (int32_t)a2, a3, a4);
}

static void qk_w_void_1134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, double, void*))f)((int32_t)a1, (int32_t)a2, a3, p4);
}

static int32_t qk_w_i32_1134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, double, void*))f)((int32_t)a1, (int32_t)a2, a3, p4);
}

static float qk_w_f32_1134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, double, void*))f)((int32_t)a1, (int32_t)a2, a3, p4);
}

static double qk_w_f64_1134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, double, void*))f)((int32_t)a1, (int32_t)a2, a3, p4);
}

static void* qk_w_ptr_1134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, double, void*))f)((int32_t)a1, (int32_t)a2, a3, p4);
}

static void qk_w_void_1141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, void*, int32_t))f)((int32_t)a1, (int32_t)a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_1141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, void*, int32_t))f)((int32_t)a1, (int32_t)a2, p3, (int32_t)a4);
}

static float qk_w_f32_1141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, void*, int32_t))f)((int32_t)a1, (int32_t)a2, p3, (int32_t)a4);
}

static double qk_w_f64_1141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, void*, int32_t))f)((int32_t)a1, (int32_t)a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_1141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, void*, int32_t))f)((int32_t)a1, (int32_t)a2, p3, (int32_t)a4);
}

static void qk_w_void_1142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, void*, float))f)((int32_t)a1, (int32_t)a2, p3, (float)a4);
}

static int32_t qk_w_i32_1142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, void*, float))f)((int32_t)a1, (int32_t)a2, p3, (float)a4);
}

static float qk_w_f32_1142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, void*, float))f)((int32_t)a1, (int32_t)a2, p3, (float)a4);
}

static double qk_w_f64_1142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, void*, float))f)((int32_t)a1, (int32_t)a2, p3, (float)a4);
}

static void* qk_w_ptr_1142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, void*, float))f)((int32_t)a1, (int32_t)a2, p3, (float)a4);
}

static void qk_w_void_1143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, void*, double))f)((int32_t)a1, (int32_t)a2, p3, a4);
}

static int32_t qk_w_i32_1143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, void*, double))f)((int32_t)a1, (int32_t)a2, p3, a4);
}

static float qk_w_f32_1143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, void*, double))f)((int32_t)a1, (int32_t)a2, p3, a4);
}

static double qk_w_f64_1143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, void*, double))f)((int32_t)a1, (int32_t)a2, p3, a4);
}

static void* qk_w_ptr_1143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, void*, double))f)((int32_t)a1, (int32_t)a2, p3, a4);
}

static void qk_w_void_1144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, int32_t, void*, void*))f)((int32_t)a1, (int32_t)a2, p3, p4);
}

static int32_t qk_w_i32_1144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, int32_t, void*, void*))f)((int32_t)a1, (int32_t)a2, p3, p4);
}

static float qk_w_f32_1144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, int32_t, void*, void*))f)((int32_t)a1, (int32_t)a2, p3, p4);
}

static double qk_w_f64_1144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, int32_t, void*, void*))f)((int32_t)a1, (int32_t)a2, p3, p4);
}

static void* qk_w_ptr_1144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, int32_t, void*, void*))f)((int32_t)a1, (int32_t)a2, p3, p4);
}

static void qk_w_void_1211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, int32_t, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_1211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, int32_t, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_1211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, int32_t, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_1211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, int32_t, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_1211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, int32_t, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_1212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, int32_t, float))f)((int32_t)a1, (float)a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_1212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, int32_t, float))f)((int32_t)a1, (float)a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_1212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, int32_t, float))f)((int32_t)a1, (float)a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_1212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, int32_t, float))f)((int32_t)a1, (float)a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_1212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, int32_t, float))f)((int32_t)a1, (float)a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_1213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, int32_t, double))f)((int32_t)a1, (float)a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_1213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, int32_t, double))f)((int32_t)a1, (float)a2, (int32_t)a3, a4);
}

static float qk_w_f32_1213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, int32_t, double))f)((int32_t)a1, (float)a2, (int32_t)a3, a4);
}

static double qk_w_f64_1213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, int32_t, double))f)((int32_t)a1, (float)a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_1213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, int32_t, double))f)((int32_t)a1, (float)a2, (int32_t)a3, a4);
}

static void qk_w_void_1214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, int32_t, void*))f)((int32_t)a1, (float)a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_1214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, int32_t, void*))f)((int32_t)a1, (float)a2, (int32_t)a3, p4);
}

static float qk_w_f32_1214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, int32_t, void*))f)((int32_t)a1, (float)a2, (int32_t)a3, p4);
}

static double qk_w_f64_1214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, int32_t, void*))f)((int32_t)a1, (float)a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_1214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, int32_t, void*))f)((int32_t)a1, (float)a2, (int32_t)a3, p4);
}

static void qk_w_void_1221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, float, int32_t))f)((int32_t)a1, (float)a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_1221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, float, int32_t))f)((int32_t)a1, (float)a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_1221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, float, int32_t))f)((int32_t)a1, (float)a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_1221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, float, int32_t))f)((int32_t)a1, (float)a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_1221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, float, int32_t))f)((int32_t)a1, (float)a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_1222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, float, float))f)((int32_t)a1, (float)a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_1222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, float, float))f)((int32_t)a1, (float)a2, (float)a3, (float)a4);
}

static float qk_w_f32_1222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, float, float))f)((int32_t)a1, (float)a2, (float)a3, (float)a4);
}

static double qk_w_f64_1222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, float, float))f)((int32_t)a1, (float)a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_1222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, float, float))f)((int32_t)a1, (float)a2, (float)a3, (float)a4);
}

static void qk_w_void_1223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, float, double))f)((int32_t)a1, (float)a2, (float)a3, a4);
}

static int32_t qk_w_i32_1223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, float, double))f)((int32_t)a1, (float)a2, (float)a3, a4);
}

static float qk_w_f32_1223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, float, double))f)((int32_t)a1, (float)a2, (float)a3, a4);
}

static double qk_w_f64_1223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, float, double))f)((int32_t)a1, (float)a2, (float)a3, a4);
}

static void* qk_w_ptr_1223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, float, double))f)((int32_t)a1, (float)a2, (float)a3, a4);
}

static void qk_w_void_1224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, float, void*))f)((int32_t)a1, (float)a2, (float)a3, p4);
}

static int32_t qk_w_i32_1224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, float, void*))f)((int32_t)a1, (float)a2, (float)a3, p4);
}

static float qk_w_f32_1224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, float, void*))f)((int32_t)a1, (float)a2, (float)a3, p4);
}

static double qk_w_f64_1224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, float, void*))f)((int32_t)a1, (float)a2, (float)a3, p4);
}

static void* qk_w_ptr_1224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, float, void*))f)((int32_t)a1, (float)a2, (float)a3, p4);
}

static void qk_w_void_1231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, double, int32_t))f)((int32_t)a1, (float)a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_1231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, double, int32_t))f)((int32_t)a1, (float)a2, a3, (int32_t)a4);
}

static float qk_w_f32_1231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, double, int32_t))f)((int32_t)a1, (float)a2, a3, (int32_t)a4);
}

static double qk_w_f64_1231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, double, int32_t))f)((int32_t)a1, (float)a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_1231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, double, int32_t))f)((int32_t)a1, (float)a2, a3, (int32_t)a4);
}

static void qk_w_void_1232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, double, float))f)((int32_t)a1, (float)a2, a3, (float)a4);
}

static int32_t qk_w_i32_1232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, double, float))f)((int32_t)a1, (float)a2, a3, (float)a4);
}

static float qk_w_f32_1232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, double, float))f)((int32_t)a1, (float)a2, a3, (float)a4);
}

static double qk_w_f64_1232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, double, float))f)((int32_t)a1, (float)a2, a3, (float)a4);
}

static void* qk_w_ptr_1232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, double, float))f)((int32_t)a1, (float)a2, a3, (float)a4);
}

static void qk_w_void_1233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, double, double))f)((int32_t)a1, (float)a2, a3, a4);
}

static int32_t qk_w_i32_1233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, double, double))f)((int32_t)a1, (float)a2, a3, a4);
}

static float qk_w_f32_1233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, double, double))f)((int32_t)a1, (float)a2, a3, a4);
}

static double qk_w_f64_1233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, double, double))f)((int32_t)a1, (float)a2, a3, a4);
}

static void* qk_w_ptr_1233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, double, double))f)((int32_t)a1, (float)a2, a3, a4);
}

static void qk_w_void_1234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, double, void*))f)((int32_t)a1, (float)a2, a3, p4);
}

static int32_t qk_w_i32_1234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, double, void*))f)((int32_t)a1, (float)a2, a3, p4);
}

static float qk_w_f32_1234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, double, void*))f)((int32_t)a1, (float)a2, a3, p4);
}

static double qk_w_f64_1234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, double, void*))f)((int32_t)a1, (float)a2, a3, p4);
}

static void* qk_w_ptr_1234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, double, void*))f)((int32_t)a1, (float)a2, a3, p4);
}

static void qk_w_void_1241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, void*, int32_t))f)((int32_t)a1, (float)a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_1241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, void*, int32_t))f)((int32_t)a1, (float)a2, p3, (int32_t)a4);
}

static float qk_w_f32_1241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, void*, int32_t))f)((int32_t)a1, (float)a2, p3, (int32_t)a4);
}

static double qk_w_f64_1241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, void*, int32_t))f)((int32_t)a1, (float)a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_1241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, void*, int32_t))f)((int32_t)a1, (float)a2, p3, (int32_t)a4);
}

static void qk_w_void_1242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, void*, float))f)((int32_t)a1, (float)a2, p3, (float)a4);
}

static int32_t qk_w_i32_1242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, void*, float))f)((int32_t)a1, (float)a2, p3, (float)a4);
}

static float qk_w_f32_1242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, void*, float))f)((int32_t)a1, (float)a2, p3, (float)a4);
}

static double qk_w_f64_1242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, void*, float))f)((int32_t)a1, (float)a2, p3, (float)a4);
}

static void* qk_w_ptr_1242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, void*, float))f)((int32_t)a1, (float)a2, p3, (float)a4);
}

static void qk_w_void_1243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, void*, double))f)((int32_t)a1, (float)a2, p3, a4);
}

static int32_t qk_w_i32_1243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, void*, double))f)((int32_t)a1, (float)a2, p3, a4);
}

static float qk_w_f32_1243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, void*, double))f)((int32_t)a1, (float)a2, p3, a4);
}

static double qk_w_f64_1243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, void*, double))f)((int32_t)a1, (float)a2, p3, a4);
}

static void* qk_w_ptr_1243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, void*, double))f)((int32_t)a1, (float)a2, p3, a4);
}

static void qk_w_void_1244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, float, void*, void*))f)((int32_t)a1, (float)a2, p3, p4);
}

static int32_t qk_w_i32_1244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, float, void*, void*))f)((int32_t)a1, (float)a2, p3, p4);
}

static float qk_w_f32_1244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, float, void*, void*))f)((int32_t)a1, (float)a2, p3, p4);
}

static double qk_w_f64_1244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, float, void*, void*))f)((int32_t)a1, (float)a2, p3, p4);
}

static void* qk_w_ptr_1244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, float, void*, void*))f)((int32_t)a1, (float)a2, p3, p4);
}

static void qk_w_void_1311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, int32_t, int32_t))f)((int32_t)a1, a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_1311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, int32_t, int32_t))f)((int32_t)a1, a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_1311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, int32_t, int32_t))f)((int32_t)a1, a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_1311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, int32_t, int32_t))f)((int32_t)a1, a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_1311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, int32_t, int32_t))f)((int32_t)a1, a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_1312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, int32_t, float))f)((int32_t)a1, a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_1312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, int32_t, float))f)((int32_t)a1, a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_1312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, int32_t, float))f)((int32_t)a1, a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_1312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, int32_t, float))f)((int32_t)a1, a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_1312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, int32_t, float))f)((int32_t)a1, a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_1313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, int32_t, double))f)((int32_t)a1, a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_1313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, int32_t, double))f)((int32_t)a1, a2, (int32_t)a3, a4);
}

static float qk_w_f32_1313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, int32_t, double))f)((int32_t)a1, a2, (int32_t)a3, a4);
}

static double qk_w_f64_1313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, int32_t, double))f)((int32_t)a1, a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_1313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, int32_t, double))f)((int32_t)a1, a2, (int32_t)a3, a4);
}

static void qk_w_void_1314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, int32_t, void*))f)((int32_t)a1, a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_1314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, int32_t, void*))f)((int32_t)a1, a2, (int32_t)a3, p4);
}

static float qk_w_f32_1314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, int32_t, void*))f)((int32_t)a1, a2, (int32_t)a3, p4);
}

static double qk_w_f64_1314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, int32_t, void*))f)((int32_t)a1, a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_1314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, int32_t, void*))f)((int32_t)a1, a2, (int32_t)a3, p4);
}

static void qk_w_void_1321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, float, int32_t))f)((int32_t)a1, a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_1321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, float, int32_t))f)((int32_t)a1, a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_1321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, float, int32_t))f)((int32_t)a1, a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_1321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, float, int32_t))f)((int32_t)a1, a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_1321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, float, int32_t))f)((int32_t)a1, a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_1322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, float, float))f)((int32_t)a1, a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_1322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, float, float))f)((int32_t)a1, a2, (float)a3, (float)a4);
}

static float qk_w_f32_1322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, float, float))f)((int32_t)a1, a2, (float)a3, (float)a4);
}

static double qk_w_f64_1322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, float, float))f)((int32_t)a1, a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_1322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, float, float))f)((int32_t)a1, a2, (float)a3, (float)a4);
}

static void qk_w_void_1323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, float, double))f)((int32_t)a1, a2, (float)a3, a4);
}

static int32_t qk_w_i32_1323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, float, double))f)((int32_t)a1, a2, (float)a3, a4);
}

static float qk_w_f32_1323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, float, double))f)((int32_t)a1, a2, (float)a3, a4);
}

static double qk_w_f64_1323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, float, double))f)((int32_t)a1, a2, (float)a3, a4);
}

static void* qk_w_ptr_1323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, float, double))f)((int32_t)a1, a2, (float)a3, a4);
}

static void qk_w_void_1324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, float, void*))f)((int32_t)a1, a2, (float)a3, p4);
}

static int32_t qk_w_i32_1324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, float, void*))f)((int32_t)a1, a2, (float)a3, p4);
}

static float qk_w_f32_1324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, float, void*))f)((int32_t)a1, a2, (float)a3, p4);
}

static double qk_w_f64_1324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, float, void*))f)((int32_t)a1, a2, (float)a3, p4);
}

static void* qk_w_ptr_1324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, float, void*))f)((int32_t)a1, a2, (float)a3, p4);
}

static void qk_w_void_1331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, double, int32_t))f)((int32_t)a1, a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_1331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, double, int32_t))f)((int32_t)a1, a2, a3, (int32_t)a4);
}

static float qk_w_f32_1331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, double, int32_t))f)((int32_t)a1, a2, a3, (int32_t)a4);
}

static double qk_w_f64_1331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, double, int32_t))f)((int32_t)a1, a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_1331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, double, int32_t))f)((int32_t)a1, a2, a3, (int32_t)a4);
}

static void qk_w_void_1332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, double, float))f)((int32_t)a1, a2, a3, (float)a4);
}

static int32_t qk_w_i32_1332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, double, float))f)((int32_t)a1, a2, a3, (float)a4);
}

static float qk_w_f32_1332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, double, float))f)((int32_t)a1, a2, a3, (float)a4);
}

static double qk_w_f64_1332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, double, float))f)((int32_t)a1, a2, a3, (float)a4);
}

static void* qk_w_ptr_1332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, double, float))f)((int32_t)a1, a2, a3, (float)a4);
}

static void qk_w_void_1333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, double, double))f)((int32_t)a1, a2, a3, a4);
}

static int32_t qk_w_i32_1333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, double, double))f)((int32_t)a1, a2, a3, a4);
}

static float qk_w_f32_1333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, double, double))f)((int32_t)a1, a2, a3, a4);
}

static double qk_w_f64_1333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, double, double))f)((int32_t)a1, a2, a3, a4);
}

static void* qk_w_ptr_1333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, double, double))f)((int32_t)a1, a2, a3, a4);
}

static void qk_w_void_1334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, double, void*))f)((int32_t)a1, a2, a3, p4);
}

static int32_t qk_w_i32_1334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, double, void*))f)((int32_t)a1, a2, a3, p4);
}

static float qk_w_f32_1334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, double, void*))f)((int32_t)a1, a2, a3, p4);
}

static double qk_w_f64_1334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, double, void*))f)((int32_t)a1, a2, a3, p4);
}

static void* qk_w_ptr_1334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, double, void*))f)((int32_t)a1, a2, a3, p4);
}

static void qk_w_void_1341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, void*, int32_t))f)((int32_t)a1, a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_1341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, void*, int32_t))f)((int32_t)a1, a2, p3, (int32_t)a4);
}

static float qk_w_f32_1341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, void*, int32_t))f)((int32_t)a1, a2, p3, (int32_t)a4);
}

static double qk_w_f64_1341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, void*, int32_t))f)((int32_t)a1, a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_1341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, void*, int32_t))f)((int32_t)a1, a2, p3, (int32_t)a4);
}

static void qk_w_void_1342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, void*, float))f)((int32_t)a1, a2, p3, (float)a4);
}

static int32_t qk_w_i32_1342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, void*, float))f)((int32_t)a1, a2, p3, (float)a4);
}

static float qk_w_f32_1342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, void*, float))f)((int32_t)a1, a2, p3, (float)a4);
}

static double qk_w_f64_1342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, void*, float))f)((int32_t)a1, a2, p3, (float)a4);
}

static void* qk_w_ptr_1342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, void*, float))f)((int32_t)a1, a2, p3, (float)a4);
}

static void qk_w_void_1343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, void*, double))f)((int32_t)a1, a2, p3, a4);
}

static int32_t qk_w_i32_1343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, void*, double))f)((int32_t)a1, a2, p3, a4);
}

static float qk_w_f32_1343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, void*, double))f)((int32_t)a1, a2, p3, a4);
}

static double qk_w_f64_1343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, void*, double))f)((int32_t)a1, a2, p3, a4);
}

static void* qk_w_ptr_1343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, void*, double))f)((int32_t)a1, a2, p3, a4);
}

static void qk_w_void_1344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, double, void*, void*))f)((int32_t)a1, a2, p3, p4);
}

static int32_t qk_w_i32_1344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, double, void*, void*))f)((int32_t)a1, a2, p3, p4);
}

static float qk_w_f32_1344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, double, void*, void*))f)((int32_t)a1, a2, p3, p4);
}

static double qk_w_f64_1344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, double, void*, void*))f)((int32_t)a1, a2, p3, p4);
}

static void* qk_w_ptr_1344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, double, void*, void*))f)((int32_t)a1, a2, p3, p4);
}

static void qk_w_void_1411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, int32_t, int32_t))f)((int32_t)a1, p2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_1411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, int32_t, int32_t))f)((int32_t)a1, p2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_1411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, int32_t, int32_t))f)((int32_t)a1, p2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_1411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, int32_t, int32_t))f)((int32_t)a1, p2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_1411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, int32_t, int32_t))f)((int32_t)a1, p2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_1412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, int32_t, float))f)((int32_t)a1, p2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_1412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, int32_t, float))f)((int32_t)a1, p2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_1412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, int32_t, float))f)((int32_t)a1, p2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_1412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, int32_t, float))f)((int32_t)a1, p2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_1412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, int32_t, float))f)((int32_t)a1, p2, (int32_t)a3, (float)a4);
}

static void qk_w_void_1413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, int32_t, double))f)((int32_t)a1, p2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_1413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, int32_t, double))f)((int32_t)a1, p2, (int32_t)a3, a4);
}

static float qk_w_f32_1413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, int32_t, double))f)((int32_t)a1, p2, (int32_t)a3, a4);
}

static double qk_w_f64_1413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, int32_t, double))f)((int32_t)a1, p2, (int32_t)a3, a4);
}

static void* qk_w_ptr_1413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, int32_t, double))f)((int32_t)a1, p2, (int32_t)a3, a4);
}

static void qk_w_void_1414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, int32_t, void*))f)((int32_t)a1, p2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_1414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, int32_t, void*))f)((int32_t)a1, p2, (int32_t)a3, p4);
}

static float qk_w_f32_1414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, int32_t, void*))f)((int32_t)a1, p2, (int32_t)a3, p4);
}

static double qk_w_f64_1414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, int32_t, void*))f)((int32_t)a1, p2, (int32_t)a3, p4);
}

static void* qk_w_ptr_1414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, int32_t, void*))f)((int32_t)a1, p2, (int32_t)a3, p4);
}

static void qk_w_void_1421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, float, int32_t))f)((int32_t)a1, p2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_1421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, float, int32_t))f)((int32_t)a1, p2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_1421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, float, int32_t))f)((int32_t)a1, p2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_1421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, float, int32_t))f)((int32_t)a1, p2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_1421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, float, int32_t))f)((int32_t)a1, p2, (float)a3, (int32_t)a4);
}

static void qk_w_void_1422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, float, float))f)((int32_t)a1, p2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_1422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, float, float))f)((int32_t)a1, p2, (float)a3, (float)a4);
}

static float qk_w_f32_1422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, float, float))f)((int32_t)a1, p2, (float)a3, (float)a4);
}

static double qk_w_f64_1422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, float, float))f)((int32_t)a1, p2, (float)a3, (float)a4);
}

static void* qk_w_ptr_1422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, float, float))f)((int32_t)a1, p2, (float)a3, (float)a4);
}

static void qk_w_void_1423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, float, double))f)((int32_t)a1, p2, (float)a3, a4);
}

static int32_t qk_w_i32_1423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, float, double))f)((int32_t)a1, p2, (float)a3, a4);
}

static float qk_w_f32_1423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, float, double))f)((int32_t)a1, p2, (float)a3, a4);
}

static double qk_w_f64_1423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, float, double))f)((int32_t)a1, p2, (float)a3, a4);
}

static void* qk_w_ptr_1423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, float, double))f)((int32_t)a1, p2, (float)a3, a4);
}

static void qk_w_void_1424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, float, void*))f)((int32_t)a1, p2, (float)a3, p4);
}

static int32_t qk_w_i32_1424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, float, void*))f)((int32_t)a1, p2, (float)a3, p4);
}

static float qk_w_f32_1424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, float, void*))f)((int32_t)a1, p2, (float)a3, p4);
}

static double qk_w_f64_1424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, float, void*))f)((int32_t)a1, p2, (float)a3, p4);
}

static void* qk_w_ptr_1424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, float, void*))f)((int32_t)a1, p2, (float)a3, p4);
}

static void qk_w_void_1431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, double, int32_t))f)((int32_t)a1, p2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_1431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, double, int32_t))f)((int32_t)a1, p2, a3, (int32_t)a4);
}

static float qk_w_f32_1431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, double, int32_t))f)((int32_t)a1, p2, a3, (int32_t)a4);
}

static double qk_w_f64_1431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, double, int32_t))f)((int32_t)a1, p2, a3, (int32_t)a4);
}

static void* qk_w_ptr_1431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, double, int32_t))f)((int32_t)a1, p2, a3, (int32_t)a4);
}

static void qk_w_void_1432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, double, float))f)((int32_t)a1, p2, a3, (float)a4);
}

static int32_t qk_w_i32_1432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, double, float))f)((int32_t)a1, p2, a3, (float)a4);
}

static float qk_w_f32_1432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, double, float))f)((int32_t)a1, p2, a3, (float)a4);
}

static double qk_w_f64_1432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, double, float))f)((int32_t)a1, p2, a3, (float)a4);
}

static void* qk_w_ptr_1432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, double, float))f)((int32_t)a1, p2, a3, (float)a4);
}

static void qk_w_void_1433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, double, double))f)((int32_t)a1, p2, a3, a4);
}

static int32_t qk_w_i32_1433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, double, double))f)((int32_t)a1, p2, a3, a4);
}

static float qk_w_f32_1433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, double, double))f)((int32_t)a1, p2, a3, a4);
}

static double qk_w_f64_1433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, double, double))f)((int32_t)a1, p2, a3, a4);
}

static void* qk_w_ptr_1433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, double, double))f)((int32_t)a1, p2, a3, a4);
}

static void qk_w_void_1434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, double, void*))f)((int32_t)a1, p2, a3, p4);
}

static int32_t qk_w_i32_1434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, double, void*))f)((int32_t)a1, p2, a3, p4);
}

static float qk_w_f32_1434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, double, void*))f)((int32_t)a1, p2, a3, p4);
}

static double qk_w_f64_1434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, double, void*))f)((int32_t)a1, p2, a3, p4);
}

static void* qk_w_ptr_1434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, double, void*))f)((int32_t)a1, p2, a3, p4);
}

static void qk_w_void_1441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, void*, int32_t))f)((int32_t)a1, p2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_1441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, void*, int32_t))f)((int32_t)a1, p2, p3, (int32_t)a4);
}

static float qk_w_f32_1441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, void*, int32_t))f)((int32_t)a1, p2, p3, (int32_t)a4);
}

static double qk_w_f64_1441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, void*, int32_t))f)((int32_t)a1, p2, p3, (int32_t)a4);
}

static void* qk_w_ptr_1441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, void*, int32_t))f)((int32_t)a1, p2, p3, (int32_t)a4);
}

static void qk_w_void_1442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, void*, float))f)((int32_t)a1, p2, p3, (float)a4);
}

static int32_t qk_w_i32_1442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, void*, float))f)((int32_t)a1, p2, p3, (float)a4);
}

static float qk_w_f32_1442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, void*, float))f)((int32_t)a1, p2, p3, (float)a4);
}

static double qk_w_f64_1442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, void*, float))f)((int32_t)a1, p2, p3, (float)a4);
}

static void* qk_w_ptr_1442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, void*, float))f)((int32_t)a1, p2, p3, (float)a4);
}

static void qk_w_void_1443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, void*, double))f)((int32_t)a1, p2, p3, a4);
}

static int32_t qk_w_i32_1443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, void*, double))f)((int32_t)a1, p2, p3, a4);
}

static float qk_w_f32_1443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, void*, double))f)((int32_t)a1, p2, p3, a4);
}

static double qk_w_f64_1443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, void*, double))f)((int32_t)a1, p2, p3, a4);
}

static void* qk_w_ptr_1443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, void*, double))f)((int32_t)a1, p2, p3, a4);
}

static void qk_w_void_1444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(int32_t, void*, void*, void*))f)((int32_t)a1, p2, p3, p4);
}

static int32_t qk_w_i32_1444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(int32_t, void*, void*, void*))f)((int32_t)a1, p2, p3, p4);
}

static float qk_w_f32_1444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(int32_t, void*, void*, void*))f)((int32_t)a1, p2, p3, p4);
}

static double qk_w_f64_1444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(int32_t, void*, void*, void*))f)((int32_t)a1, p2, p3, p4);
}

static void* qk_w_ptr_1444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(int32_t, void*, void*, void*))f)((int32_t)a1, p2, p3, p4);
}

static void qk_w_void_2111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_2111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_2111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_2111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_2111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_2112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, int32_t, float))f)((float)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_2112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, int32_t, float))f)((float)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_2112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, int32_t, float))f)((float)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_2112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, int32_t, float))f)((float)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_2112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, int32_t, float))f)((float)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_2113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, int32_t, double))f)((float)a1, (int32_t)a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_2113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, int32_t, double))f)((float)a1, (int32_t)a2, (int32_t)a3, a4);
}

static float qk_w_f32_2113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, int32_t, double))f)((float)a1, (int32_t)a2, (int32_t)a3, a4);
}

static double qk_w_f64_2113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, int32_t, double))f)((float)a1, (int32_t)a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_2113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, int32_t, double))f)((float)a1, (int32_t)a2, (int32_t)a3, a4);
}

static void qk_w_void_2114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, int32_t, void*))f)((float)a1, (int32_t)a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_2114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, int32_t, void*))f)((float)a1, (int32_t)a2, (int32_t)a3, p4);
}

static float qk_w_f32_2114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, int32_t, void*))f)((float)a1, (int32_t)a2, (int32_t)a3, p4);
}

static double qk_w_f64_2114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, int32_t, void*))f)((float)a1, (int32_t)a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_2114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, int32_t, void*))f)((float)a1, (int32_t)a2, (int32_t)a3, p4);
}

static void qk_w_void_2121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, float, int32_t))f)((float)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_2121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, float, int32_t))f)((float)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_2121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, float, int32_t))f)((float)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_2121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, float, int32_t))f)((float)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_2121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, float, int32_t))f)((float)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_2122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, float, float))f)((float)a1, (int32_t)a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_2122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, float, float))f)((float)a1, (int32_t)a2, (float)a3, (float)a4);
}

static float qk_w_f32_2122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, float, float))f)((float)a1, (int32_t)a2, (float)a3, (float)a4);
}

static double qk_w_f64_2122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, float, float))f)((float)a1, (int32_t)a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_2122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, float, float))f)((float)a1, (int32_t)a2, (float)a3, (float)a4);
}

static void qk_w_void_2123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, float, double))f)((float)a1, (int32_t)a2, (float)a3, a4);
}

static int32_t qk_w_i32_2123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, float, double))f)((float)a1, (int32_t)a2, (float)a3, a4);
}

static float qk_w_f32_2123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, float, double))f)((float)a1, (int32_t)a2, (float)a3, a4);
}

static double qk_w_f64_2123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, float, double))f)((float)a1, (int32_t)a2, (float)a3, a4);
}

static void* qk_w_ptr_2123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, float, double))f)((float)a1, (int32_t)a2, (float)a3, a4);
}

static void qk_w_void_2124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, float, void*))f)((float)a1, (int32_t)a2, (float)a3, p4);
}

static int32_t qk_w_i32_2124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, float, void*))f)((float)a1, (int32_t)a2, (float)a3, p4);
}

static float qk_w_f32_2124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, float, void*))f)((float)a1, (int32_t)a2, (float)a3, p4);
}

static double qk_w_f64_2124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, float, void*))f)((float)a1, (int32_t)a2, (float)a3, p4);
}

static void* qk_w_ptr_2124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, float, void*))f)((float)a1, (int32_t)a2, (float)a3, p4);
}

static void qk_w_void_2131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, double, int32_t))f)((float)a1, (int32_t)a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_2131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, double, int32_t))f)((float)a1, (int32_t)a2, a3, (int32_t)a4);
}

static float qk_w_f32_2131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, double, int32_t))f)((float)a1, (int32_t)a2, a3, (int32_t)a4);
}

static double qk_w_f64_2131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, double, int32_t))f)((float)a1, (int32_t)a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_2131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, double, int32_t))f)((float)a1, (int32_t)a2, a3, (int32_t)a4);
}

static void qk_w_void_2132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, double, float))f)((float)a1, (int32_t)a2, a3, (float)a4);
}

static int32_t qk_w_i32_2132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, double, float))f)((float)a1, (int32_t)a2, a3, (float)a4);
}

static float qk_w_f32_2132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, double, float))f)((float)a1, (int32_t)a2, a3, (float)a4);
}

static double qk_w_f64_2132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, double, float))f)((float)a1, (int32_t)a2, a3, (float)a4);
}

static void* qk_w_ptr_2132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, double, float))f)((float)a1, (int32_t)a2, a3, (float)a4);
}

static void qk_w_void_2133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, double, double))f)((float)a1, (int32_t)a2, a3, a4);
}

static int32_t qk_w_i32_2133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, double, double))f)((float)a1, (int32_t)a2, a3, a4);
}

static float qk_w_f32_2133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, double, double))f)((float)a1, (int32_t)a2, a3, a4);
}

static double qk_w_f64_2133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, double, double))f)((float)a1, (int32_t)a2, a3, a4);
}

static void* qk_w_ptr_2133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, double, double))f)((float)a1, (int32_t)a2, a3, a4);
}

static void qk_w_void_2134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, double, void*))f)((float)a1, (int32_t)a2, a3, p4);
}

static int32_t qk_w_i32_2134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, double, void*))f)((float)a1, (int32_t)a2, a3, p4);
}

static float qk_w_f32_2134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, double, void*))f)((float)a1, (int32_t)a2, a3, p4);
}

static double qk_w_f64_2134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, double, void*))f)((float)a1, (int32_t)a2, a3, p4);
}

static void* qk_w_ptr_2134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, double, void*))f)((float)a1, (int32_t)a2, a3, p4);
}

static void qk_w_void_2141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, void*, int32_t))f)((float)a1, (int32_t)a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_2141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, void*, int32_t))f)((float)a1, (int32_t)a2, p3, (int32_t)a4);
}

static float qk_w_f32_2141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, void*, int32_t))f)((float)a1, (int32_t)a2, p3, (int32_t)a4);
}

static double qk_w_f64_2141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, void*, int32_t))f)((float)a1, (int32_t)a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_2141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, void*, int32_t))f)((float)a1, (int32_t)a2, p3, (int32_t)a4);
}

static void qk_w_void_2142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, void*, float))f)((float)a1, (int32_t)a2, p3, (float)a4);
}

static int32_t qk_w_i32_2142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, void*, float))f)((float)a1, (int32_t)a2, p3, (float)a4);
}

static float qk_w_f32_2142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, void*, float))f)((float)a1, (int32_t)a2, p3, (float)a4);
}

static double qk_w_f64_2142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, void*, float))f)((float)a1, (int32_t)a2, p3, (float)a4);
}

static void* qk_w_ptr_2142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, void*, float))f)((float)a1, (int32_t)a2, p3, (float)a4);
}

static void qk_w_void_2143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, void*, double))f)((float)a1, (int32_t)a2, p3, a4);
}

static int32_t qk_w_i32_2143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, void*, double))f)((float)a1, (int32_t)a2, p3, a4);
}

static float qk_w_f32_2143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, void*, double))f)((float)a1, (int32_t)a2, p3, a4);
}

static double qk_w_f64_2143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, void*, double))f)((float)a1, (int32_t)a2, p3, a4);
}

static void* qk_w_ptr_2143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, void*, double))f)((float)a1, (int32_t)a2, p3, a4);
}

static void qk_w_void_2144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, int32_t, void*, void*))f)((float)a1, (int32_t)a2, p3, p4);
}

static int32_t qk_w_i32_2144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, int32_t, void*, void*))f)((float)a1, (int32_t)a2, p3, p4);
}

static float qk_w_f32_2144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, int32_t, void*, void*))f)((float)a1, (int32_t)a2, p3, p4);
}

static double qk_w_f64_2144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, int32_t, void*, void*))f)((float)a1, (int32_t)a2, p3, p4);
}

static void* qk_w_ptr_2144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, int32_t, void*, void*))f)((float)a1, (int32_t)a2, p3, p4);
}

static void qk_w_void_2211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, int32_t, int32_t))f)((float)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_2211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, int32_t, int32_t))f)((float)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_2211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, int32_t, int32_t))f)((float)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_2211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, int32_t, int32_t))f)((float)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_2211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, int32_t, int32_t))f)((float)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_2212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, int32_t, float))f)((float)a1, (float)a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_2212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, int32_t, float))f)((float)a1, (float)a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_2212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, int32_t, float))f)((float)a1, (float)a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_2212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, int32_t, float))f)((float)a1, (float)a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_2212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, int32_t, float))f)((float)a1, (float)a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_2213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, int32_t, double))f)((float)a1, (float)a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_2213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, int32_t, double))f)((float)a1, (float)a2, (int32_t)a3, a4);
}

static float qk_w_f32_2213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, int32_t, double))f)((float)a1, (float)a2, (int32_t)a3, a4);
}

static double qk_w_f64_2213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, int32_t, double))f)((float)a1, (float)a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_2213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, int32_t, double))f)((float)a1, (float)a2, (int32_t)a3, a4);
}

static void qk_w_void_2214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, int32_t, void*))f)((float)a1, (float)a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_2214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, int32_t, void*))f)((float)a1, (float)a2, (int32_t)a3, p4);
}

static float qk_w_f32_2214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, int32_t, void*))f)((float)a1, (float)a2, (int32_t)a3, p4);
}

static double qk_w_f64_2214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, int32_t, void*))f)((float)a1, (float)a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_2214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, int32_t, void*))f)((float)a1, (float)a2, (int32_t)a3, p4);
}

static void qk_w_void_2221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, float, int32_t))f)((float)a1, (float)a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_2221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, float, int32_t))f)((float)a1, (float)a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_2221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, float, int32_t))f)((float)a1, (float)a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_2221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, float, int32_t))f)((float)a1, (float)a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_2221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, float, int32_t))f)((float)a1, (float)a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_2222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, float, float))f)((float)a1, (float)a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_2222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, float, float))f)((float)a1, (float)a2, (float)a3, (float)a4);
}

static float qk_w_f32_2222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, float, float))f)((float)a1, (float)a2, (float)a3, (float)a4);
}

static double qk_w_f64_2222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, float, float))f)((float)a1, (float)a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_2222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, float, float))f)((float)a1, (float)a2, (float)a3, (float)a4);
}

static void qk_w_void_2223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, float, double))f)((float)a1, (float)a2, (float)a3, a4);
}

static int32_t qk_w_i32_2223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, float, double))f)((float)a1, (float)a2, (float)a3, a4);
}

static float qk_w_f32_2223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, float, double))f)((float)a1, (float)a2, (float)a3, a4);
}

static double qk_w_f64_2223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, float, double))f)((float)a1, (float)a2, (float)a3, a4);
}

static void* qk_w_ptr_2223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, float, double))f)((float)a1, (float)a2, (float)a3, a4);
}

static void qk_w_void_2224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, float, void*))f)((float)a1, (float)a2, (float)a3, p4);
}

static int32_t qk_w_i32_2224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, float, void*))f)((float)a1, (float)a2, (float)a3, p4);
}

static float qk_w_f32_2224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, float, void*))f)((float)a1, (float)a2, (float)a3, p4);
}

static double qk_w_f64_2224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, float, void*))f)((float)a1, (float)a2, (float)a3, p4);
}

static void* qk_w_ptr_2224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, float, void*))f)((float)a1, (float)a2, (float)a3, p4);
}

static void qk_w_void_2231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, double, int32_t))f)((float)a1, (float)a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_2231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, double, int32_t))f)((float)a1, (float)a2, a3, (int32_t)a4);
}

static float qk_w_f32_2231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, double, int32_t))f)((float)a1, (float)a2, a3, (int32_t)a4);
}

static double qk_w_f64_2231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, double, int32_t))f)((float)a1, (float)a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_2231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, double, int32_t))f)((float)a1, (float)a2, a3, (int32_t)a4);
}

static void qk_w_void_2232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, double, float))f)((float)a1, (float)a2, a3, (float)a4);
}

static int32_t qk_w_i32_2232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, double, float))f)((float)a1, (float)a2, a3, (float)a4);
}

static float qk_w_f32_2232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, double, float))f)((float)a1, (float)a2, a3, (float)a4);
}

static double qk_w_f64_2232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, double, float))f)((float)a1, (float)a2, a3, (float)a4);
}

static void* qk_w_ptr_2232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, double, float))f)((float)a1, (float)a2, a3, (float)a4);
}

static void qk_w_void_2233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, double, double))f)((float)a1, (float)a2, a3, a4);
}

static int32_t qk_w_i32_2233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, double, double))f)((float)a1, (float)a2, a3, a4);
}

static float qk_w_f32_2233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, double, double))f)((float)a1, (float)a2, a3, a4);
}

static double qk_w_f64_2233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, double, double))f)((float)a1, (float)a2, a3, a4);
}

static void* qk_w_ptr_2233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, double, double))f)((float)a1, (float)a2, a3, a4);
}

static void qk_w_void_2234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, double, void*))f)((float)a1, (float)a2, a3, p4);
}

static int32_t qk_w_i32_2234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, double, void*))f)((float)a1, (float)a2, a3, p4);
}

static float qk_w_f32_2234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, double, void*))f)((float)a1, (float)a2, a3, p4);
}

static double qk_w_f64_2234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, double, void*))f)((float)a1, (float)a2, a3, p4);
}

static void* qk_w_ptr_2234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, double, void*))f)((float)a1, (float)a2, a3, p4);
}

static void qk_w_void_2241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, void*, int32_t))f)((float)a1, (float)a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_2241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, void*, int32_t))f)((float)a1, (float)a2, p3, (int32_t)a4);
}

static float qk_w_f32_2241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, void*, int32_t))f)((float)a1, (float)a2, p3, (int32_t)a4);
}

static double qk_w_f64_2241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, void*, int32_t))f)((float)a1, (float)a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_2241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, void*, int32_t))f)((float)a1, (float)a2, p3, (int32_t)a4);
}

static void qk_w_void_2242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, void*, float))f)((float)a1, (float)a2, p3, (float)a4);
}

static int32_t qk_w_i32_2242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, void*, float))f)((float)a1, (float)a2, p3, (float)a4);
}

static float qk_w_f32_2242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, void*, float))f)((float)a1, (float)a2, p3, (float)a4);
}

static double qk_w_f64_2242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, void*, float))f)((float)a1, (float)a2, p3, (float)a4);
}

static void* qk_w_ptr_2242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, void*, float))f)((float)a1, (float)a2, p3, (float)a4);
}

static void qk_w_void_2243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, void*, double))f)((float)a1, (float)a2, p3, a4);
}

static int32_t qk_w_i32_2243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, void*, double))f)((float)a1, (float)a2, p3, a4);
}

static float qk_w_f32_2243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, void*, double))f)((float)a1, (float)a2, p3, a4);
}

static double qk_w_f64_2243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, void*, double))f)((float)a1, (float)a2, p3, a4);
}

static void* qk_w_ptr_2243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, void*, double))f)((float)a1, (float)a2, p3, a4);
}

static void qk_w_void_2244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, float, void*, void*))f)((float)a1, (float)a2, p3, p4);
}

static int32_t qk_w_i32_2244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, float, void*, void*))f)((float)a1, (float)a2, p3, p4);
}

static float qk_w_f32_2244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, float, void*, void*))f)((float)a1, (float)a2, p3, p4);
}

static double qk_w_f64_2244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, float, void*, void*))f)((float)a1, (float)a2, p3, p4);
}

static void* qk_w_ptr_2244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, float, void*, void*))f)((float)a1, (float)a2, p3, p4);
}

static void qk_w_void_2311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, int32_t, int32_t))f)((float)a1, a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_2311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, int32_t, int32_t))f)((float)a1, a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_2311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, int32_t, int32_t))f)((float)a1, a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_2311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, int32_t, int32_t))f)((float)a1, a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_2311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, int32_t, int32_t))f)((float)a1, a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_2312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, int32_t, float))f)((float)a1, a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_2312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, int32_t, float))f)((float)a1, a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_2312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, int32_t, float))f)((float)a1, a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_2312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, int32_t, float))f)((float)a1, a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_2312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, int32_t, float))f)((float)a1, a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_2313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, int32_t, double))f)((float)a1, a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_2313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, int32_t, double))f)((float)a1, a2, (int32_t)a3, a4);
}

static float qk_w_f32_2313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, int32_t, double))f)((float)a1, a2, (int32_t)a3, a4);
}

static double qk_w_f64_2313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, int32_t, double))f)((float)a1, a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_2313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, int32_t, double))f)((float)a1, a2, (int32_t)a3, a4);
}

static void qk_w_void_2314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, int32_t, void*))f)((float)a1, a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_2314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, int32_t, void*))f)((float)a1, a2, (int32_t)a3, p4);
}

static float qk_w_f32_2314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, int32_t, void*))f)((float)a1, a2, (int32_t)a3, p4);
}

static double qk_w_f64_2314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, int32_t, void*))f)((float)a1, a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_2314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, int32_t, void*))f)((float)a1, a2, (int32_t)a3, p4);
}

static void qk_w_void_2321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, float, int32_t))f)((float)a1, a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_2321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, float, int32_t))f)((float)a1, a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_2321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, float, int32_t))f)((float)a1, a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_2321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, float, int32_t))f)((float)a1, a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_2321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, float, int32_t))f)((float)a1, a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_2322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, float, float))f)((float)a1, a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_2322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, float, float))f)((float)a1, a2, (float)a3, (float)a4);
}

static float qk_w_f32_2322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, float, float))f)((float)a1, a2, (float)a3, (float)a4);
}

static double qk_w_f64_2322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, float, float))f)((float)a1, a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_2322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, float, float))f)((float)a1, a2, (float)a3, (float)a4);
}

static void qk_w_void_2323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, float, double))f)((float)a1, a2, (float)a3, a4);
}

static int32_t qk_w_i32_2323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, float, double))f)((float)a1, a2, (float)a3, a4);
}

static float qk_w_f32_2323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, float, double))f)((float)a1, a2, (float)a3, a4);
}

static double qk_w_f64_2323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, float, double))f)((float)a1, a2, (float)a3, a4);
}

static void* qk_w_ptr_2323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, float, double))f)((float)a1, a2, (float)a3, a4);
}

static void qk_w_void_2324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, float, void*))f)((float)a1, a2, (float)a3, p4);
}

static int32_t qk_w_i32_2324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, float, void*))f)((float)a1, a2, (float)a3, p4);
}

static float qk_w_f32_2324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, float, void*))f)((float)a1, a2, (float)a3, p4);
}

static double qk_w_f64_2324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, float, void*))f)((float)a1, a2, (float)a3, p4);
}

static void* qk_w_ptr_2324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, float, void*))f)((float)a1, a2, (float)a3, p4);
}

static void qk_w_void_2331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, double, int32_t))f)((float)a1, a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_2331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, double, int32_t))f)((float)a1, a2, a3, (int32_t)a4);
}

static float qk_w_f32_2331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, double, int32_t))f)((float)a1, a2, a3, (int32_t)a4);
}

static double qk_w_f64_2331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, double, int32_t))f)((float)a1, a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_2331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, double, int32_t))f)((float)a1, a2, a3, (int32_t)a4);
}

static void qk_w_void_2332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, double, float))f)((float)a1, a2, a3, (float)a4);
}

static int32_t qk_w_i32_2332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, double, float))f)((float)a1, a2, a3, (float)a4);
}

static float qk_w_f32_2332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, double, float))f)((float)a1, a2, a3, (float)a4);
}

static double qk_w_f64_2332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, double, float))f)((float)a1, a2, a3, (float)a4);
}

static void* qk_w_ptr_2332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, double, float))f)((float)a1, a2, a3, (float)a4);
}

static void qk_w_void_2333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, double, double))f)((float)a1, a2, a3, a4);
}

static int32_t qk_w_i32_2333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, double, double))f)((float)a1, a2, a3, a4);
}

static float qk_w_f32_2333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, double, double))f)((float)a1, a2, a3, a4);
}

static double qk_w_f64_2333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, double, double))f)((float)a1, a2, a3, a4);
}

static void* qk_w_ptr_2333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, double, double))f)((float)a1, a2, a3, a4);
}

static void qk_w_void_2334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, double, void*))f)((float)a1, a2, a3, p4);
}

static int32_t qk_w_i32_2334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, double, void*))f)((float)a1, a2, a3, p4);
}

static float qk_w_f32_2334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, double, void*))f)((float)a1, a2, a3, p4);
}

static double qk_w_f64_2334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, double, void*))f)((float)a1, a2, a3, p4);
}

static void* qk_w_ptr_2334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, double, void*))f)((float)a1, a2, a3, p4);
}

static void qk_w_void_2341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, void*, int32_t))f)((float)a1, a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_2341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, void*, int32_t))f)((float)a1, a2, p3, (int32_t)a4);
}

static float qk_w_f32_2341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, void*, int32_t))f)((float)a1, a2, p3, (int32_t)a4);
}

static double qk_w_f64_2341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, void*, int32_t))f)((float)a1, a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_2341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, void*, int32_t))f)((float)a1, a2, p3, (int32_t)a4);
}

static void qk_w_void_2342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, void*, float))f)((float)a1, a2, p3, (float)a4);
}

static int32_t qk_w_i32_2342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, void*, float))f)((float)a1, a2, p3, (float)a4);
}

static float qk_w_f32_2342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, void*, float))f)((float)a1, a2, p3, (float)a4);
}

static double qk_w_f64_2342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, void*, float))f)((float)a1, a2, p3, (float)a4);
}

static void* qk_w_ptr_2342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, void*, float))f)((float)a1, a2, p3, (float)a4);
}

static void qk_w_void_2343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, void*, double))f)((float)a1, a2, p3, a4);
}

static int32_t qk_w_i32_2343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, void*, double))f)((float)a1, a2, p3, a4);
}

static float qk_w_f32_2343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, void*, double))f)((float)a1, a2, p3, a4);
}

static double qk_w_f64_2343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, void*, double))f)((float)a1, a2, p3, a4);
}

static void* qk_w_ptr_2343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, void*, double))f)((float)a1, a2, p3, a4);
}

static void qk_w_void_2344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, double, void*, void*))f)((float)a1, a2, p3, p4);
}

static int32_t qk_w_i32_2344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, double, void*, void*))f)((float)a1, a2, p3, p4);
}

static float qk_w_f32_2344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, double, void*, void*))f)((float)a1, a2, p3, p4);
}

static double qk_w_f64_2344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, double, void*, void*))f)((float)a1, a2, p3, p4);
}

static void* qk_w_ptr_2344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, double, void*, void*))f)((float)a1, a2, p3, p4);
}

static void qk_w_void_2411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, int32_t, int32_t))f)((float)a1, p2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_2411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, int32_t, int32_t))f)((float)a1, p2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_2411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, int32_t, int32_t))f)((float)a1, p2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_2411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, int32_t, int32_t))f)((float)a1, p2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_2411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, int32_t, int32_t))f)((float)a1, p2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_2412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, int32_t, float))f)((float)a1, p2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_2412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, int32_t, float))f)((float)a1, p2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_2412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, int32_t, float))f)((float)a1, p2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_2412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, int32_t, float))f)((float)a1, p2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_2412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, int32_t, float))f)((float)a1, p2, (int32_t)a3, (float)a4);
}

static void qk_w_void_2413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, int32_t, double))f)((float)a1, p2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_2413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, int32_t, double))f)((float)a1, p2, (int32_t)a3, a4);
}

static float qk_w_f32_2413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, int32_t, double))f)((float)a1, p2, (int32_t)a3, a4);
}

static double qk_w_f64_2413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, int32_t, double))f)((float)a1, p2, (int32_t)a3, a4);
}

static void* qk_w_ptr_2413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, int32_t, double))f)((float)a1, p2, (int32_t)a3, a4);
}

static void qk_w_void_2414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, int32_t, void*))f)((float)a1, p2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_2414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, int32_t, void*))f)((float)a1, p2, (int32_t)a3, p4);
}

static float qk_w_f32_2414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, int32_t, void*))f)((float)a1, p2, (int32_t)a3, p4);
}

static double qk_w_f64_2414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, int32_t, void*))f)((float)a1, p2, (int32_t)a3, p4);
}

static void* qk_w_ptr_2414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, int32_t, void*))f)((float)a1, p2, (int32_t)a3, p4);
}

static void qk_w_void_2421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, float, int32_t))f)((float)a1, p2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_2421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, float, int32_t))f)((float)a1, p2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_2421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, float, int32_t))f)((float)a1, p2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_2421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, float, int32_t))f)((float)a1, p2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_2421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, float, int32_t))f)((float)a1, p2, (float)a3, (int32_t)a4);
}

static void qk_w_void_2422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, float, float))f)((float)a1, p2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_2422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, float, float))f)((float)a1, p2, (float)a3, (float)a4);
}

static float qk_w_f32_2422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, float, float))f)((float)a1, p2, (float)a3, (float)a4);
}

static double qk_w_f64_2422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, float, float))f)((float)a1, p2, (float)a3, (float)a4);
}

static void* qk_w_ptr_2422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, float, float))f)((float)a1, p2, (float)a3, (float)a4);
}

static void qk_w_void_2423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, float, double))f)((float)a1, p2, (float)a3, a4);
}

static int32_t qk_w_i32_2423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, float, double))f)((float)a1, p2, (float)a3, a4);
}

static float qk_w_f32_2423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, float, double))f)((float)a1, p2, (float)a3, a4);
}

static double qk_w_f64_2423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, float, double))f)((float)a1, p2, (float)a3, a4);
}

static void* qk_w_ptr_2423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, float, double))f)((float)a1, p2, (float)a3, a4);
}

static void qk_w_void_2424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, float, void*))f)((float)a1, p2, (float)a3, p4);
}

static int32_t qk_w_i32_2424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, float, void*))f)((float)a1, p2, (float)a3, p4);
}

static float qk_w_f32_2424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, float, void*))f)((float)a1, p2, (float)a3, p4);
}

static double qk_w_f64_2424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, float, void*))f)((float)a1, p2, (float)a3, p4);
}

static void* qk_w_ptr_2424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, float, void*))f)((float)a1, p2, (float)a3, p4);
}

static void qk_w_void_2431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, double, int32_t))f)((float)a1, p2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_2431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, double, int32_t))f)((float)a1, p2, a3, (int32_t)a4);
}

static float qk_w_f32_2431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, double, int32_t))f)((float)a1, p2, a3, (int32_t)a4);
}

static double qk_w_f64_2431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, double, int32_t))f)((float)a1, p2, a3, (int32_t)a4);
}

static void* qk_w_ptr_2431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, double, int32_t))f)((float)a1, p2, a3, (int32_t)a4);
}

static void qk_w_void_2432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, double, float))f)((float)a1, p2, a3, (float)a4);
}

static int32_t qk_w_i32_2432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, double, float))f)((float)a1, p2, a3, (float)a4);
}

static float qk_w_f32_2432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, double, float))f)((float)a1, p2, a3, (float)a4);
}

static double qk_w_f64_2432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, double, float))f)((float)a1, p2, a3, (float)a4);
}

static void* qk_w_ptr_2432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, double, float))f)((float)a1, p2, a3, (float)a4);
}

static void qk_w_void_2433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, double, double))f)((float)a1, p2, a3, a4);
}

static int32_t qk_w_i32_2433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, double, double))f)((float)a1, p2, a3, a4);
}

static float qk_w_f32_2433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, double, double))f)((float)a1, p2, a3, a4);
}

static double qk_w_f64_2433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, double, double))f)((float)a1, p2, a3, a4);
}

static void* qk_w_ptr_2433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, double, double))f)((float)a1, p2, a3, a4);
}

static void qk_w_void_2434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, double, void*))f)((float)a1, p2, a3, p4);
}

static int32_t qk_w_i32_2434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, double, void*))f)((float)a1, p2, a3, p4);
}

static float qk_w_f32_2434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, double, void*))f)((float)a1, p2, a3, p4);
}

static double qk_w_f64_2434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, double, void*))f)((float)a1, p2, a3, p4);
}

static void* qk_w_ptr_2434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, double, void*))f)((float)a1, p2, a3, p4);
}

static void qk_w_void_2441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, void*, int32_t))f)((float)a1, p2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_2441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, void*, int32_t))f)((float)a1, p2, p3, (int32_t)a4);
}

static float qk_w_f32_2441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, void*, int32_t))f)((float)a1, p2, p3, (int32_t)a4);
}

static double qk_w_f64_2441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, void*, int32_t))f)((float)a1, p2, p3, (int32_t)a4);
}

static void* qk_w_ptr_2441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, void*, int32_t))f)((float)a1, p2, p3, (int32_t)a4);
}

static void qk_w_void_2442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, void*, float))f)((float)a1, p2, p3, (float)a4);
}

static int32_t qk_w_i32_2442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, void*, float))f)((float)a1, p2, p3, (float)a4);
}

static float qk_w_f32_2442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, void*, float))f)((float)a1, p2, p3, (float)a4);
}

static double qk_w_f64_2442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, void*, float))f)((float)a1, p2, p3, (float)a4);
}

static void* qk_w_ptr_2442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, void*, float))f)((float)a1, p2, p3, (float)a4);
}

static void qk_w_void_2443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, void*, double))f)((float)a1, p2, p3, a4);
}

static int32_t qk_w_i32_2443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, void*, double))f)((float)a1, p2, p3, a4);
}

static float qk_w_f32_2443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, void*, double))f)((float)a1, p2, p3, a4);
}

static double qk_w_f64_2443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, void*, double))f)((float)a1, p2, p3, a4);
}

static void* qk_w_ptr_2443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, void*, double))f)((float)a1, p2, p3, a4);
}

static void qk_w_void_2444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(float, void*, void*, void*))f)((float)a1, p2, p3, p4);
}

static int32_t qk_w_i32_2444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(float, void*, void*, void*))f)((float)a1, p2, p3, p4);
}

static float qk_w_f32_2444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(float, void*, void*, void*))f)((float)a1, p2, p3, p4);
}

static double qk_w_f64_2444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(float, void*, void*, void*))f)((float)a1, p2, p3, p4);
}

static void* qk_w_ptr_2444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(float, void*, void*, void*))f)((float)a1, p2, p3, p4);
}

static void qk_w_void_3111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_3111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_3111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_3111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_3111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_3112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, int32_t, float))f)(a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_3112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, int32_t, float))f)(a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_3112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, int32_t, float))f)(a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_3112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, int32_t, float))f)(a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_3112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, int32_t, float))f)(a1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_3113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, int32_t, double))f)(a1, (int32_t)a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_3113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, int32_t, double))f)(a1, (int32_t)a2, (int32_t)a3, a4);
}

static float qk_w_f32_3113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, int32_t, double))f)(a1, (int32_t)a2, (int32_t)a3, a4);
}

static double qk_w_f64_3113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, int32_t, double))f)(a1, (int32_t)a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_3113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, int32_t, double))f)(a1, (int32_t)a2, (int32_t)a3, a4);
}

static void qk_w_void_3114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, int32_t, void*))f)(a1, (int32_t)a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_3114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, int32_t, void*))f)(a1, (int32_t)a2, (int32_t)a3, p4);
}

static float qk_w_f32_3114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, int32_t, void*))f)(a1, (int32_t)a2, (int32_t)a3, p4);
}

static double qk_w_f64_3114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, int32_t, void*))f)(a1, (int32_t)a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_3114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, int32_t, void*))f)(a1, (int32_t)a2, (int32_t)a3, p4);
}

static void qk_w_void_3121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, float, int32_t))f)(a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_3121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, float, int32_t))f)(a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_3121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, float, int32_t))f)(a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_3121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, float, int32_t))f)(a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_3121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, float, int32_t))f)(a1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_3122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, float, float))f)(a1, (int32_t)a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_3122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, float, float))f)(a1, (int32_t)a2, (float)a3, (float)a4);
}

static float qk_w_f32_3122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, float, float))f)(a1, (int32_t)a2, (float)a3, (float)a4);
}

static double qk_w_f64_3122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, float, float))f)(a1, (int32_t)a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_3122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, float, float))f)(a1, (int32_t)a2, (float)a3, (float)a4);
}

static void qk_w_void_3123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, float, double))f)(a1, (int32_t)a2, (float)a3, a4);
}

static int32_t qk_w_i32_3123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, float, double))f)(a1, (int32_t)a2, (float)a3, a4);
}

static float qk_w_f32_3123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, float, double))f)(a1, (int32_t)a2, (float)a3, a4);
}

static double qk_w_f64_3123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, float, double))f)(a1, (int32_t)a2, (float)a3, a4);
}

static void* qk_w_ptr_3123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, float, double))f)(a1, (int32_t)a2, (float)a3, a4);
}

static void qk_w_void_3124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, float, void*))f)(a1, (int32_t)a2, (float)a3, p4);
}

static int32_t qk_w_i32_3124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, float, void*))f)(a1, (int32_t)a2, (float)a3, p4);
}

static float qk_w_f32_3124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, float, void*))f)(a1, (int32_t)a2, (float)a3, p4);
}

static double qk_w_f64_3124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, float, void*))f)(a1, (int32_t)a2, (float)a3, p4);
}

static void* qk_w_ptr_3124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, float, void*))f)(a1, (int32_t)a2, (float)a3, p4);
}

static void qk_w_void_3131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, double, int32_t))f)(a1, (int32_t)a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_3131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, double, int32_t))f)(a1, (int32_t)a2, a3, (int32_t)a4);
}

static float qk_w_f32_3131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, double, int32_t))f)(a1, (int32_t)a2, a3, (int32_t)a4);
}

static double qk_w_f64_3131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, double, int32_t))f)(a1, (int32_t)a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_3131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, double, int32_t))f)(a1, (int32_t)a2, a3, (int32_t)a4);
}

static void qk_w_void_3132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, double, float))f)(a1, (int32_t)a2, a3, (float)a4);
}

static int32_t qk_w_i32_3132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, double, float))f)(a1, (int32_t)a2, a3, (float)a4);
}

static float qk_w_f32_3132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, double, float))f)(a1, (int32_t)a2, a3, (float)a4);
}

static double qk_w_f64_3132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, double, float))f)(a1, (int32_t)a2, a3, (float)a4);
}

static void* qk_w_ptr_3132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, double, float))f)(a1, (int32_t)a2, a3, (float)a4);
}

static void qk_w_void_3133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, double, double))f)(a1, (int32_t)a2, a3, a4);
}

static int32_t qk_w_i32_3133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, double, double))f)(a1, (int32_t)a2, a3, a4);
}

static float qk_w_f32_3133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, double, double))f)(a1, (int32_t)a2, a3, a4);
}

static double qk_w_f64_3133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, double, double))f)(a1, (int32_t)a2, a3, a4);
}

static void* qk_w_ptr_3133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, double, double))f)(a1, (int32_t)a2, a3, a4);
}

static void qk_w_void_3134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, double, void*))f)(a1, (int32_t)a2, a3, p4);
}

static int32_t qk_w_i32_3134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, double, void*))f)(a1, (int32_t)a2, a3, p4);
}

static float qk_w_f32_3134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, double, void*))f)(a1, (int32_t)a2, a3, p4);
}

static double qk_w_f64_3134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, double, void*))f)(a1, (int32_t)a2, a3, p4);
}

static void* qk_w_ptr_3134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, double, void*))f)(a1, (int32_t)a2, a3, p4);
}

static void qk_w_void_3141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, void*, int32_t))f)(a1, (int32_t)a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_3141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, void*, int32_t))f)(a1, (int32_t)a2, p3, (int32_t)a4);
}

static float qk_w_f32_3141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, void*, int32_t))f)(a1, (int32_t)a2, p3, (int32_t)a4);
}

static double qk_w_f64_3141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, void*, int32_t))f)(a1, (int32_t)a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_3141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, void*, int32_t))f)(a1, (int32_t)a2, p3, (int32_t)a4);
}

static void qk_w_void_3142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, void*, float))f)(a1, (int32_t)a2, p3, (float)a4);
}

static int32_t qk_w_i32_3142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, void*, float))f)(a1, (int32_t)a2, p3, (float)a4);
}

static float qk_w_f32_3142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, void*, float))f)(a1, (int32_t)a2, p3, (float)a4);
}

static double qk_w_f64_3142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, void*, float))f)(a1, (int32_t)a2, p3, (float)a4);
}

static void* qk_w_ptr_3142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, void*, float))f)(a1, (int32_t)a2, p3, (float)a4);
}

static void qk_w_void_3143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, void*, double))f)(a1, (int32_t)a2, p3, a4);
}

static int32_t qk_w_i32_3143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, void*, double))f)(a1, (int32_t)a2, p3, a4);
}

static float qk_w_f32_3143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, void*, double))f)(a1, (int32_t)a2, p3, a4);
}

static double qk_w_f64_3143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, void*, double))f)(a1, (int32_t)a2, p3, a4);
}

static void* qk_w_ptr_3143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, void*, double))f)(a1, (int32_t)a2, p3, a4);
}

static void qk_w_void_3144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, int32_t, void*, void*))f)(a1, (int32_t)a2, p3, p4);
}

static int32_t qk_w_i32_3144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, int32_t, void*, void*))f)(a1, (int32_t)a2, p3, p4);
}

static float qk_w_f32_3144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, int32_t, void*, void*))f)(a1, (int32_t)a2, p3, p4);
}

static double qk_w_f64_3144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, int32_t, void*, void*))f)(a1, (int32_t)a2, p3, p4);
}

static void* qk_w_ptr_3144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, int32_t, void*, void*))f)(a1, (int32_t)a2, p3, p4);
}

static void qk_w_void_3211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, int32_t, int32_t))f)(a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_3211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, int32_t, int32_t))f)(a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_3211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, int32_t, int32_t))f)(a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_3211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, int32_t, int32_t))f)(a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_3211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, int32_t, int32_t))f)(a1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_3212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, int32_t, float))f)(a1, (float)a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_3212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, int32_t, float))f)(a1, (float)a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_3212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, int32_t, float))f)(a1, (float)a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_3212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, int32_t, float))f)(a1, (float)a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_3212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, int32_t, float))f)(a1, (float)a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_3213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, int32_t, double))f)(a1, (float)a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_3213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, int32_t, double))f)(a1, (float)a2, (int32_t)a3, a4);
}

static float qk_w_f32_3213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, int32_t, double))f)(a1, (float)a2, (int32_t)a3, a4);
}

static double qk_w_f64_3213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, int32_t, double))f)(a1, (float)a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_3213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, int32_t, double))f)(a1, (float)a2, (int32_t)a3, a4);
}

static void qk_w_void_3214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, int32_t, void*))f)(a1, (float)a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_3214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, int32_t, void*))f)(a1, (float)a2, (int32_t)a3, p4);
}

static float qk_w_f32_3214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, int32_t, void*))f)(a1, (float)a2, (int32_t)a3, p4);
}

static double qk_w_f64_3214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, int32_t, void*))f)(a1, (float)a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_3214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, int32_t, void*))f)(a1, (float)a2, (int32_t)a3, p4);
}

static void qk_w_void_3221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, float, int32_t))f)(a1, (float)a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_3221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, float, int32_t))f)(a1, (float)a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_3221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, float, int32_t))f)(a1, (float)a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_3221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, float, int32_t))f)(a1, (float)a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_3221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, float, int32_t))f)(a1, (float)a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_3222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, float, float))f)(a1, (float)a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_3222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, float, float))f)(a1, (float)a2, (float)a3, (float)a4);
}

static float qk_w_f32_3222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, float, float))f)(a1, (float)a2, (float)a3, (float)a4);
}

static double qk_w_f64_3222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, float, float))f)(a1, (float)a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_3222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, float, float))f)(a1, (float)a2, (float)a3, (float)a4);
}

static void qk_w_void_3223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, float, double))f)(a1, (float)a2, (float)a3, a4);
}

static int32_t qk_w_i32_3223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, float, double))f)(a1, (float)a2, (float)a3, a4);
}

static float qk_w_f32_3223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, float, double))f)(a1, (float)a2, (float)a3, a4);
}

static double qk_w_f64_3223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, float, double))f)(a1, (float)a2, (float)a3, a4);
}

static void* qk_w_ptr_3223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, float, double))f)(a1, (float)a2, (float)a3, a4);
}

static void qk_w_void_3224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, float, void*))f)(a1, (float)a2, (float)a3, p4);
}

static int32_t qk_w_i32_3224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, float, void*))f)(a1, (float)a2, (float)a3, p4);
}

static float qk_w_f32_3224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, float, void*))f)(a1, (float)a2, (float)a3, p4);
}

static double qk_w_f64_3224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, float, void*))f)(a1, (float)a2, (float)a3, p4);
}

static void* qk_w_ptr_3224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, float, void*))f)(a1, (float)a2, (float)a3, p4);
}

static void qk_w_void_3231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, double, int32_t))f)(a1, (float)a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_3231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, double, int32_t))f)(a1, (float)a2, a3, (int32_t)a4);
}

static float qk_w_f32_3231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, double, int32_t))f)(a1, (float)a2, a3, (int32_t)a4);
}

static double qk_w_f64_3231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, double, int32_t))f)(a1, (float)a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_3231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, double, int32_t))f)(a1, (float)a2, a3, (int32_t)a4);
}

static void qk_w_void_3232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, double, float))f)(a1, (float)a2, a3, (float)a4);
}

static int32_t qk_w_i32_3232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, double, float))f)(a1, (float)a2, a3, (float)a4);
}

static float qk_w_f32_3232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, double, float))f)(a1, (float)a2, a3, (float)a4);
}

static double qk_w_f64_3232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, double, float))f)(a1, (float)a2, a3, (float)a4);
}

static void* qk_w_ptr_3232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, double, float))f)(a1, (float)a2, a3, (float)a4);
}

static void qk_w_void_3233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, double, double))f)(a1, (float)a2, a3, a4);
}

static int32_t qk_w_i32_3233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, double, double))f)(a1, (float)a2, a3, a4);
}

static float qk_w_f32_3233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, double, double))f)(a1, (float)a2, a3, a4);
}

static double qk_w_f64_3233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, double, double))f)(a1, (float)a2, a3, a4);
}

static void* qk_w_ptr_3233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, double, double))f)(a1, (float)a2, a3, a4);
}

static void qk_w_void_3234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, double, void*))f)(a1, (float)a2, a3, p4);
}

static int32_t qk_w_i32_3234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, double, void*))f)(a1, (float)a2, a3, p4);
}

static float qk_w_f32_3234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, double, void*))f)(a1, (float)a2, a3, p4);
}

static double qk_w_f64_3234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, double, void*))f)(a1, (float)a2, a3, p4);
}

static void* qk_w_ptr_3234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, double, void*))f)(a1, (float)a2, a3, p4);
}

static void qk_w_void_3241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, void*, int32_t))f)(a1, (float)a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_3241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, void*, int32_t))f)(a1, (float)a2, p3, (int32_t)a4);
}

static float qk_w_f32_3241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, void*, int32_t))f)(a1, (float)a2, p3, (int32_t)a4);
}

static double qk_w_f64_3241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, void*, int32_t))f)(a1, (float)a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_3241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, void*, int32_t))f)(a1, (float)a2, p3, (int32_t)a4);
}

static void qk_w_void_3242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, void*, float))f)(a1, (float)a2, p3, (float)a4);
}

static int32_t qk_w_i32_3242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, void*, float))f)(a1, (float)a2, p3, (float)a4);
}

static float qk_w_f32_3242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, void*, float))f)(a1, (float)a2, p3, (float)a4);
}

static double qk_w_f64_3242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, void*, float))f)(a1, (float)a2, p3, (float)a4);
}

static void* qk_w_ptr_3242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, void*, float))f)(a1, (float)a2, p3, (float)a4);
}

static void qk_w_void_3243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, void*, double))f)(a1, (float)a2, p3, a4);
}

static int32_t qk_w_i32_3243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, void*, double))f)(a1, (float)a2, p3, a4);
}

static float qk_w_f32_3243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, void*, double))f)(a1, (float)a2, p3, a4);
}

static double qk_w_f64_3243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, void*, double))f)(a1, (float)a2, p3, a4);
}

static void* qk_w_ptr_3243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, void*, double))f)(a1, (float)a2, p3, a4);
}

static void qk_w_void_3244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, float, void*, void*))f)(a1, (float)a2, p3, p4);
}

static int32_t qk_w_i32_3244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, float, void*, void*))f)(a1, (float)a2, p3, p4);
}

static float qk_w_f32_3244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, float, void*, void*))f)(a1, (float)a2, p3, p4);
}

static double qk_w_f64_3244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, float, void*, void*))f)(a1, (float)a2, p3, p4);
}

static void* qk_w_ptr_3244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, float, void*, void*))f)(a1, (float)a2, p3, p4);
}

static void qk_w_void_3311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, int32_t, int32_t))f)(a1, a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_3311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, int32_t, int32_t))f)(a1, a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_3311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, int32_t, int32_t))f)(a1, a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_3311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, int32_t, int32_t))f)(a1, a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_3311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, int32_t, int32_t))f)(a1, a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_3312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, int32_t, float))f)(a1, a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_3312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, int32_t, float))f)(a1, a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_3312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, int32_t, float))f)(a1, a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_3312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, int32_t, float))f)(a1, a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_3312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, int32_t, float))f)(a1, a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_3313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, int32_t, double))f)(a1, a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_3313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, int32_t, double))f)(a1, a2, (int32_t)a3, a4);
}

static float qk_w_f32_3313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, int32_t, double))f)(a1, a2, (int32_t)a3, a4);
}

static double qk_w_f64_3313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, int32_t, double))f)(a1, a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_3313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, int32_t, double))f)(a1, a2, (int32_t)a3, a4);
}

static void qk_w_void_3314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, int32_t, void*))f)(a1, a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_3314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, int32_t, void*))f)(a1, a2, (int32_t)a3, p4);
}

static float qk_w_f32_3314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, int32_t, void*))f)(a1, a2, (int32_t)a3, p4);
}

static double qk_w_f64_3314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, int32_t, void*))f)(a1, a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_3314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, int32_t, void*))f)(a1, a2, (int32_t)a3, p4);
}

static void qk_w_void_3321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, float, int32_t))f)(a1, a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_3321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, float, int32_t))f)(a1, a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_3321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, float, int32_t))f)(a1, a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_3321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, float, int32_t))f)(a1, a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_3321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, float, int32_t))f)(a1, a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_3322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, float, float))f)(a1, a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_3322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, float, float))f)(a1, a2, (float)a3, (float)a4);
}

static float qk_w_f32_3322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, float, float))f)(a1, a2, (float)a3, (float)a4);
}

static double qk_w_f64_3322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, float, float))f)(a1, a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_3322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, float, float))f)(a1, a2, (float)a3, (float)a4);
}

static void qk_w_void_3323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, float, double))f)(a1, a2, (float)a3, a4);
}

static int32_t qk_w_i32_3323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, float, double))f)(a1, a2, (float)a3, a4);
}

static float qk_w_f32_3323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, float, double))f)(a1, a2, (float)a3, a4);
}

static double qk_w_f64_3323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, float, double))f)(a1, a2, (float)a3, a4);
}

static void* qk_w_ptr_3323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, float, double))f)(a1, a2, (float)a3, a4);
}

static void qk_w_void_3324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, float, void*))f)(a1, a2, (float)a3, p4);
}

static int32_t qk_w_i32_3324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, float, void*))f)(a1, a2, (float)a3, p4);
}

static float qk_w_f32_3324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, float, void*))f)(a1, a2, (float)a3, p4);
}

static double qk_w_f64_3324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, float, void*))f)(a1, a2, (float)a3, p4);
}

static void* qk_w_ptr_3324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, float, void*))f)(a1, a2, (float)a3, p4);
}

static void qk_w_void_3331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, double, int32_t))f)(a1, a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_3331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, double, int32_t))f)(a1, a2, a3, (int32_t)a4);
}

static float qk_w_f32_3331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, double, int32_t))f)(a1, a2, a3, (int32_t)a4);
}

static double qk_w_f64_3331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, double, int32_t))f)(a1, a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_3331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, double, int32_t))f)(a1, a2, a3, (int32_t)a4);
}

static void qk_w_void_3332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, double, float))f)(a1, a2, a3, (float)a4);
}

static int32_t qk_w_i32_3332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, double, float))f)(a1, a2, a3, (float)a4);
}

static float qk_w_f32_3332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, double, float))f)(a1, a2, a3, (float)a4);
}

static double qk_w_f64_3332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, double, float))f)(a1, a2, a3, (float)a4);
}

static void* qk_w_ptr_3332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, double, float))f)(a1, a2, a3, (float)a4);
}

static void qk_w_void_3333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, double, double))f)(a1, a2, a3, a4);
}

static int32_t qk_w_i32_3333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, double, double))f)(a1, a2, a3, a4);
}

static float qk_w_f32_3333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, double, double))f)(a1, a2, a3, a4);
}

static double qk_w_f64_3333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, double, double))f)(a1, a2, a3, a4);
}

static void* qk_w_ptr_3333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, double, double))f)(a1, a2, a3, a4);
}

static void qk_w_void_3334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, double, void*))f)(a1, a2, a3, p4);
}

static int32_t qk_w_i32_3334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, double, void*))f)(a1, a2, a3, p4);
}

static float qk_w_f32_3334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, double, void*))f)(a1, a2, a3, p4);
}

static double qk_w_f64_3334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, double, void*))f)(a1, a2, a3, p4);
}

static void* qk_w_ptr_3334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, double, void*))f)(a1, a2, a3, p4);
}

static void qk_w_void_3341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, void*, int32_t))f)(a1, a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_3341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, void*, int32_t))f)(a1, a2, p3, (int32_t)a4);
}

static float qk_w_f32_3341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, void*, int32_t))f)(a1, a2, p3, (int32_t)a4);
}

static double qk_w_f64_3341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, void*, int32_t))f)(a1, a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_3341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, void*, int32_t))f)(a1, a2, p3, (int32_t)a4);
}

static void qk_w_void_3342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, void*, float))f)(a1, a2, p3, (float)a4);
}

static int32_t qk_w_i32_3342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, void*, float))f)(a1, a2, p3, (float)a4);
}

static float qk_w_f32_3342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, void*, float))f)(a1, a2, p3, (float)a4);
}

static double qk_w_f64_3342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, void*, float))f)(a1, a2, p3, (float)a4);
}

static void* qk_w_ptr_3342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, void*, float))f)(a1, a2, p3, (float)a4);
}

static void qk_w_void_3343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, void*, double))f)(a1, a2, p3, a4);
}

static int32_t qk_w_i32_3343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, void*, double))f)(a1, a2, p3, a4);
}

static float qk_w_f32_3343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, void*, double))f)(a1, a2, p3, a4);
}

static double qk_w_f64_3343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, void*, double))f)(a1, a2, p3, a4);
}

static void* qk_w_ptr_3343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, void*, double))f)(a1, a2, p3, a4);
}

static void qk_w_void_3344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, double, void*, void*))f)(a1, a2, p3, p4);
}

static int32_t qk_w_i32_3344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, double, void*, void*))f)(a1, a2, p3, p4);
}

static float qk_w_f32_3344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, double, void*, void*))f)(a1, a2, p3, p4);
}

static double qk_w_f64_3344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, double, void*, void*))f)(a1, a2, p3, p4);
}

static void* qk_w_ptr_3344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, double, void*, void*))f)(a1, a2, p3, p4);
}

static void qk_w_void_3411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, int32_t, int32_t))f)(a1, p2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_3411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, int32_t, int32_t))f)(a1, p2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_3411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, int32_t, int32_t))f)(a1, p2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_3411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, int32_t, int32_t))f)(a1, p2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_3411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, int32_t, int32_t))f)(a1, p2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_3412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, int32_t, float))f)(a1, p2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_3412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, int32_t, float))f)(a1, p2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_3412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, int32_t, float))f)(a1, p2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_3412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, int32_t, float))f)(a1, p2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_3412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, int32_t, float))f)(a1, p2, (int32_t)a3, (float)a4);
}

static void qk_w_void_3413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, int32_t, double))f)(a1, p2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_3413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, int32_t, double))f)(a1, p2, (int32_t)a3, a4);
}

static float qk_w_f32_3413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, int32_t, double))f)(a1, p2, (int32_t)a3, a4);
}

static double qk_w_f64_3413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, int32_t, double))f)(a1, p2, (int32_t)a3, a4);
}

static void* qk_w_ptr_3413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, int32_t, double))f)(a1, p2, (int32_t)a3, a4);
}

static void qk_w_void_3414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, int32_t, void*))f)(a1, p2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_3414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, int32_t, void*))f)(a1, p2, (int32_t)a3, p4);
}

static float qk_w_f32_3414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, int32_t, void*))f)(a1, p2, (int32_t)a3, p4);
}

static double qk_w_f64_3414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, int32_t, void*))f)(a1, p2, (int32_t)a3, p4);
}

static void* qk_w_ptr_3414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, int32_t, void*))f)(a1, p2, (int32_t)a3, p4);
}

static void qk_w_void_3421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, float, int32_t))f)(a1, p2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_3421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, float, int32_t))f)(a1, p2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_3421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, float, int32_t))f)(a1, p2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_3421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, float, int32_t))f)(a1, p2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_3421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, float, int32_t))f)(a1, p2, (float)a3, (int32_t)a4);
}

static void qk_w_void_3422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, float, float))f)(a1, p2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_3422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, float, float))f)(a1, p2, (float)a3, (float)a4);
}

static float qk_w_f32_3422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, float, float))f)(a1, p2, (float)a3, (float)a4);
}

static double qk_w_f64_3422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, float, float))f)(a1, p2, (float)a3, (float)a4);
}

static void* qk_w_ptr_3422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, float, float))f)(a1, p2, (float)a3, (float)a4);
}

static void qk_w_void_3423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, float, double))f)(a1, p2, (float)a3, a4);
}

static int32_t qk_w_i32_3423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, float, double))f)(a1, p2, (float)a3, a4);
}

static float qk_w_f32_3423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, float, double))f)(a1, p2, (float)a3, a4);
}

static double qk_w_f64_3423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, float, double))f)(a1, p2, (float)a3, a4);
}

static void* qk_w_ptr_3423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, float, double))f)(a1, p2, (float)a3, a4);
}

static void qk_w_void_3424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, float, void*))f)(a1, p2, (float)a3, p4);
}

static int32_t qk_w_i32_3424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, float, void*))f)(a1, p2, (float)a3, p4);
}

static float qk_w_f32_3424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, float, void*))f)(a1, p2, (float)a3, p4);
}

static double qk_w_f64_3424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, float, void*))f)(a1, p2, (float)a3, p4);
}

static void* qk_w_ptr_3424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, float, void*))f)(a1, p2, (float)a3, p4);
}

static void qk_w_void_3431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, double, int32_t))f)(a1, p2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_3431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, double, int32_t))f)(a1, p2, a3, (int32_t)a4);
}

static float qk_w_f32_3431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, double, int32_t))f)(a1, p2, a3, (int32_t)a4);
}

static double qk_w_f64_3431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, double, int32_t))f)(a1, p2, a3, (int32_t)a4);
}

static void* qk_w_ptr_3431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, double, int32_t))f)(a1, p2, a3, (int32_t)a4);
}

static void qk_w_void_3432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, double, float))f)(a1, p2, a3, (float)a4);
}

static int32_t qk_w_i32_3432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, double, float))f)(a1, p2, a3, (float)a4);
}

static float qk_w_f32_3432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, double, float))f)(a1, p2, a3, (float)a4);
}

static double qk_w_f64_3432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, double, float))f)(a1, p2, a3, (float)a4);
}

static void* qk_w_ptr_3432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, double, float))f)(a1, p2, a3, (float)a4);
}

static void qk_w_void_3433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, double, double))f)(a1, p2, a3, a4);
}

static int32_t qk_w_i32_3433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, double, double))f)(a1, p2, a3, a4);
}

static float qk_w_f32_3433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, double, double))f)(a1, p2, a3, a4);
}

static double qk_w_f64_3433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, double, double))f)(a1, p2, a3, a4);
}

static void* qk_w_ptr_3433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, double, double))f)(a1, p2, a3, a4);
}

static void qk_w_void_3434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, double, void*))f)(a1, p2, a3, p4);
}

static int32_t qk_w_i32_3434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, double, void*))f)(a1, p2, a3, p4);
}

static float qk_w_f32_3434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, double, void*))f)(a1, p2, a3, p4);
}

static double qk_w_f64_3434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, double, void*))f)(a1, p2, a3, p4);
}

static void* qk_w_ptr_3434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, double, void*))f)(a1, p2, a3, p4);
}

static void qk_w_void_3441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, void*, int32_t))f)(a1, p2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_3441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, void*, int32_t))f)(a1, p2, p3, (int32_t)a4);
}

static float qk_w_f32_3441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, void*, int32_t))f)(a1, p2, p3, (int32_t)a4);
}

static double qk_w_f64_3441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, void*, int32_t))f)(a1, p2, p3, (int32_t)a4);
}

static void* qk_w_ptr_3441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, void*, int32_t))f)(a1, p2, p3, (int32_t)a4);
}

static void qk_w_void_3442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, void*, float))f)(a1, p2, p3, (float)a4);
}

static int32_t qk_w_i32_3442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, void*, float))f)(a1, p2, p3, (float)a4);
}

static float qk_w_f32_3442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, void*, float))f)(a1, p2, p3, (float)a4);
}

static double qk_w_f64_3442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, void*, float))f)(a1, p2, p3, (float)a4);
}

static void* qk_w_ptr_3442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, void*, float))f)(a1, p2, p3, (float)a4);
}

static void qk_w_void_3443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, void*, double))f)(a1, p2, p3, a4);
}

static int32_t qk_w_i32_3443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, void*, double))f)(a1, p2, p3, a4);
}

static float qk_w_f32_3443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, void*, double))f)(a1, p2, p3, a4);
}

static double qk_w_f64_3443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, void*, double))f)(a1, p2, p3, a4);
}

static void* qk_w_ptr_3443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, void*, double))f)(a1, p2, p3, a4);
}

static void qk_w_void_3444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(double, void*, void*, void*))f)(a1, p2, p3, p4);
}

static int32_t qk_w_i32_3444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(double, void*, void*, void*))f)(a1, p2, p3, p4);
}

static float qk_w_f32_3444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(double, void*, void*, void*))f)(a1, p2, p3, p4);
}

static double qk_w_f64_3444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(double, void*, void*, void*))f)(a1, p2, p3, p4);
}

static void* qk_w_ptr_3444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(double, void*, void*, void*))f)(a1, p2, p3, p4);
}

static void qk_w_void_4111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_4111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_4111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_4111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_4111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_4112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, int32_t, float))f)(p1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_4112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, int32_t, float))f)(p1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_4112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, int32_t, float))f)(p1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_4112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, int32_t, float))f)(p1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_4112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, int32_t, float))f)(p1, (int32_t)a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_4113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, int32_t, double))f)(p1, (int32_t)a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_4113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, int32_t, double))f)(p1, (int32_t)a2, (int32_t)a3, a4);
}

static float qk_w_f32_4113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, int32_t, double))f)(p1, (int32_t)a2, (int32_t)a3, a4);
}

static double qk_w_f64_4113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, int32_t, double))f)(p1, (int32_t)a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_4113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, int32_t, double))f)(p1, (int32_t)a2, (int32_t)a3, a4);
}

static void qk_w_void_4114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, int32_t, void*))f)(p1, (int32_t)a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_4114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, int32_t, void*))f)(p1, (int32_t)a2, (int32_t)a3, p4);
}

static float qk_w_f32_4114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, int32_t, void*))f)(p1, (int32_t)a2, (int32_t)a3, p4);
}

static double qk_w_f64_4114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, int32_t, void*))f)(p1, (int32_t)a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_4114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, int32_t, void*))f)(p1, (int32_t)a2, (int32_t)a3, p4);
}

static void qk_w_void_4121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, float, int32_t))f)(p1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_4121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, float, int32_t))f)(p1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_4121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, float, int32_t))f)(p1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_4121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, float, int32_t))f)(p1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_4121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, float, int32_t))f)(p1, (int32_t)a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_4122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, float, float))f)(p1, (int32_t)a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_4122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, float, float))f)(p1, (int32_t)a2, (float)a3, (float)a4);
}

static float qk_w_f32_4122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, float, float))f)(p1, (int32_t)a2, (float)a3, (float)a4);
}

static double qk_w_f64_4122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, float, float))f)(p1, (int32_t)a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_4122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, float, float))f)(p1, (int32_t)a2, (float)a3, (float)a4);
}

static void qk_w_void_4123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, float, double))f)(p1, (int32_t)a2, (float)a3, a4);
}

static int32_t qk_w_i32_4123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, float, double))f)(p1, (int32_t)a2, (float)a3, a4);
}

static float qk_w_f32_4123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, float, double))f)(p1, (int32_t)a2, (float)a3, a4);
}

static double qk_w_f64_4123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, float, double))f)(p1, (int32_t)a2, (float)a3, a4);
}

static void* qk_w_ptr_4123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, float, double))f)(p1, (int32_t)a2, (float)a3, a4);
}

static void qk_w_void_4124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, float, void*))f)(p1, (int32_t)a2, (float)a3, p4);
}

static int32_t qk_w_i32_4124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, float, void*))f)(p1, (int32_t)a2, (float)a3, p4);
}

static float qk_w_f32_4124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, float, void*))f)(p1, (int32_t)a2, (float)a3, p4);
}

static double qk_w_f64_4124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, float, void*))f)(p1, (int32_t)a2, (float)a3, p4);
}

static void* qk_w_ptr_4124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, float, void*))f)(p1, (int32_t)a2, (float)a3, p4);
}

static void qk_w_void_4131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, double, int32_t))f)(p1, (int32_t)a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_4131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, double, int32_t))f)(p1, (int32_t)a2, a3, (int32_t)a4);
}

static float qk_w_f32_4131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, double, int32_t))f)(p1, (int32_t)a2, a3, (int32_t)a4);
}

static double qk_w_f64_4131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, double, int32_t))f)(p1, (int32_t)a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_4131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, double, int32_t))f)(p1, (int32_t)a2, a3, (int32_t)a4);
}

static void qk_w_void_4132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, double, float))f)(p1, (int32_t)a2, a3, (float)a4);
}

static int32_t qk_w_i32_4132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, double, float))f)(p1, (int32_t)a2, a3, (float)a4);
}

static float qk_w_f32_4132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, double, float))f)(p1, (int32_t)a2, a3, (float)a4);
}

static double qk_w_f64_4132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, double, float))f)(p1, (int32_t)a2, a3, (float)a4);
}

static void* qk_w_ptr_4132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, double, float))f)(p1, (int32_t)a2, a3, (float)a4);
}

static void qk_w_void_4133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, double, double))f)(p1, (int32_t)a2, a3, a4);
}

static int32_t qk_w_i32_4133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, double, double))f)(p1, (int32_t)a2, a3, a4);
}

static float qk_w_f32_4133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, double, double))f)(p1, (int32_t)a2, a3, a4);
}

static double qk_w_f64_4133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, double, double))f)(p1, (int32_t)a2, a3, a4);
}

static void* qk_w_ptr_4133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, double, double))f)(p1, (int32_t)a2, a3, a4);
}

static void qk_w_void_4134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, double, void*))f)(p1, (int32_t)a2, a3, p4);
}

static int32_t qk_w_i32_4134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, double, void*))f)(p1, (int32_t)a2, a3, p4);
}

static float qk_w_f32_4134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, double, void*))f)(p1, (int32_t)a2, a3, p4);
}

static double qk_w_f64_4134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, double, void*))f)(p1, (int32_t)a2, a3, p4);
}

static void* qk_w_ptr_4134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, double, void*))f)(p1, (int32_t)a2, a3, p4);
}

static void qk_w_void_4141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, void*, int32_t))f)(p1, (int32_t)a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_4141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, void*, int32_t))f)(p1, (int32_t)a2, p3, (int32_t)a4);
}

static float qk_w_f32_4141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, void*, int32_t))f)(p1, (int32_t)a2, p3, (int32_t)a4);
}

static double qk_w_f64_4141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, void*, int32_t))f)(p1, (int32_t)a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_4141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, void*, int32_t))f)(p1, (int32_t)a2, p3, (int32_t)a4);
}

static void qk_w_void_4142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, void*, float))f)(p1, (int32_t)a2, p3, (float)a4);
}

static int32_t qk_w_i32_4142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, void*, float))f)(p1, (int32_t)a2, p3, (float)a4);
}

static float qk_w_f32_4142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, void*, float))f)(p1, (int32_t)a2, p3, (float)a4);
}

static double qk_w_f64_4142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, void*, float))f)(p1, (int32_t)a2, p3, (float)a4);
}

static void* qk_w_ptr_4142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, void*, float))f)(p1, (int32_t)a2, p3, (float)a4);
}

static void qk_w_void_4143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, void*, double))f)(p1, (int32_t)a2, p3, a4);
}

static int32_t qk_w_i32_4143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, void*, double))f)(p1, (int32_t)a2, p3, a4);
}

static float qk_w_f32_4143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, void*, double))f)(p1, (int32_t)a2, p3, a4);
}

static double qk_w_f64_4143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, void*, double))f)(p1, (int32_t)a2, p3, a4);
}

static void* qk_w_ptr_4143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, void*, double))f)(p1, (int32_t)a2, p3, a4);
}

static void qk_w_void_4144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, int32_t, void*, void*))f)(p1, (int32_t)a2, p3, p4);
}

static int32_t qk_w_i32_4144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, int32_t, void*, void*))f)(p1, (int32_t)a2, p3, p4);
}

static float qk_w_f32_4144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, int32_t, void*, void*))f)(p1, (int32_t)a2, p3, p4);
}

static double qk_w_f64_4144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, int32_t, void*, void*))f)(p1, (int32_t)a2, p3, p4);
}

static void* qk_w_ptr_4144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, int32_t, void*, void*))f)(p1, (int32_t)a2, p3, p4);
}

static void qk_w_void_4211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, int32_t, int32_t))f)(p1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_4211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, int32_t, int32_t))f)(p1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_4211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, int32_t, int32_t))f)(p1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_4211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, int32_t, int32_t))f)(p1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_4211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, int32_t, int32_t))f)(p1, (float)a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_4212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, int32_t, float))f)(p1, (float)a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_4212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, int32_t, float))f)(p1, (float)a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_4212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, int32_t, float))f)(p1, (float)a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_4212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, int32_t, float))f)(p1, (float)a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_4212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, int32_t, float))f)(p1, (float)a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_4213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, int32_t, double))f)(p1, (float)a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_4213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, int32_t, double))f)(p1, (float)a2, (int32_t)a3, a4);
}

static float qk_w_f32_4213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, int32_t, double))f)(p1, (float)a2, (int32_t)a3, a4);
}

static double qk_w_f64_4213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, int32_t, double))f)(p1, (float)a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_4213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, int32_t, double))f)(p1, (float)a2, (int32_t)a3, a4);
}

static void qk_w_void_4214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, int32_t, void*))f)(p1, (float)a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_4214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, int32_t, void*))f)(p1, (float)a2, (int32_t)a3, p4);
}

static float qk_w_f32_4214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, int32_t, void*))f)(p1, (float)a2, (int32_t)a3, p4);
}

static double qk_w_f64_4214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, int32_t, void*))f)(p1, (float)a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_4214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, int32_t, void*))f)(p1, (float)a2, (int32_t)a3, p4);
}

static void qk_w_void_4221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, float, int32_t))f)(p1, (float)a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_4221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, float, int32_t))f)(p1, (float)a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_4221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, float, int32_t))f)(p1, (float)a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_4221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, float, int32_t))f)(p1, (float)a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_4221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, float, int32_t))f)(p1, (float)a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_4222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, float, float))f)(p1, (float)a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_4222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, float, float))f)(p1, (float)a2, (float)a3, (float)a4);
}

static float qk_w_f32_4222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, float, float))f)(p1, (float)a2, (float)a3, (float)a4);
}

static double qk_w_f64_4222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, float, float))f)(p1, (float)a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_4222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, float, float))f)(p1, (float)a2, (float)a3, (float)a4);
}

static void qk_w_void_4223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, float, double))f)(p1, (float)a2, (float)a3, a4);
}

static int32_t qk_w_i32_4223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, float, double))f)(p1, (float)a2, (float)a3, a4);
}

static float qk_w_f32_4223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, float, double))f)(p1, (float)a2, (float)a3, a4);
}

static double qk_w_f64_4223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, float, double))f)(p1, (float)a2, (float)a3, a4);
}

static void* qk_w_ptr_4223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, float, double))f)(p1, (float)a2, (float)a3, a4);
}

static void qk_w_void_4224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, float, void*))f)(p1, (float)a2, (float)a3, p4);
}

static int32_t qk_w_i32_4224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, float, void*))f)(p1, (float)a2, (float)a3, p4);
}

static float qk_w_f32_4224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, float, void*))f)(p1, (float)a2, (float)a3, p4);
}

static double qk_w_f64_4224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, float, void*))f)(p1, (float)a2, (float)a3, p4);
}

static void* qk_w_ptr_4224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, float, void*))f)(p1, (float)a2, (float)a3, p4);
}

static void qk_w_void_4231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, double, int32_t))f)(p1, (float)a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_4231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, double, int32_t))f)(p1, (float)a2, a3, (int32_t)a4);
}

static float qk_w_f32_4231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, double, int32_t))f)(p1, (float)a2, a3, (int32_t)a4);
}

static double qk_w_f64_4231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, double, int32_t))f)(p1, (float)a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_4231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, double, int32_t))f)(p1, (float)a2, a3, (int32_t)a4);
}

static void qk_w_void_4232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, double, float))f)(p1, (float)a2, a3, (float)a4);
}

static int32_t qk_w_i32_4232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, double, float))f)(p1, (float)a2, a3, (float)a4);
}

static float qk_w_f32_4232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, double, float))f)(p1, (float)a2, a3, (float)a4);
}

static double qk_w_f64_4232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, double, float))f)(p1, (float)a2, a3, (float)a4);
}

static void* qk_w_ptr_4232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, double, float))f)(p1, (float)a2, a3, (float)a4);
}

static void qk_w_void_4233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, double, double))f)(p1, (float)a2, a3, a4);
}

static int32_t qk_w_i32_4233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, double, double))f)(p1, (float)a2, a3, a4);
}

static float qk_w_f32_4233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, double, double))f)(p1, (float)a2, a3, a4);
}

static double qk_w_f64_4233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, double, double))f)(p1, (float)a2, a3, a4);
}

static void* qk_w_ptr_4233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, double, double))f)(p1, (float)a2, a3, a4);
}

static void qk_w_void_4234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, double, void*))f)(p1, (float)a2, a3, p4);
}

static int32_t qk_w_i32_4234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, double, void*))f)(p1, (float)a2, a3, p4);
}

static float qk_w_f32_4234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, double, void*))f)(p1, (float)a2, a3, p4);
}

static double qk_w_f64_4234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, double, void*))f)(p1, (float)a2, a3, p4);
}

static void* qk_w_ptr_4234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, double, void*))f)(p1, (float)a2, a3, p4);
}

static void qk_w_void_4241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, void*, int32_t))f)(p1, (float)a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_4241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, void*, int32_t))f)(p1, (float)a2, p3, (int32_t)a4);
}

static float qk_w_f32_4241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, void*, int32_t))f)(p1, (float)a2, p3, (int32_t)a4);
}

static double qk_w_f64_4241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, void*, int32_t))f)(p1, (float)a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_4241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, void*, int32_t))f)(p1, (float)a2, p3, (int32_t)a4);
}

static void qk_w_void_4242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, void*, float))f)(p1, (float)a2, p3, (float)a4);
}

static int32_t qk_w_i32_4242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, void*, float))f)(p1, (float)a2, p3, (float)a4);
}

static float qk_w_f32_4242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, void*, float))f)(p1, (float)a2, p3, (float)a4);
}

static double qk_w_f64_4242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, void*, float))f)(p1, (float)a2, p3, (float)a4);
}

static void* qk_w_ptr_4242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, void*, float))f)(p1, (float)a2, p3, (float)a4);
}

static void qk_w_void_4243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, void*, double))f)(p1, (float)a2, p3, a4);
}

static int32_t qk_w_i32_4243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, void*, double))f)(p1, (float)a2, p3, a4);
}

static float qk_w_f32_4243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, void*, double))f)(p1, (float)a2, p3, a4);
}

static double qk_w_f64_4243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, void*, double))f)(p1, (float)a2, p3, a4);
}

static void* qk_w_ptr_4243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, void*, double))f)(p1, (float)a2, p3, a4);
}

static void qk_w_void_4244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, float, void*, void*))f)(p1, (float)a2, p3, p4);
}

static int32_t qk_w_i32_4244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, float, void*, void*))f)(p1, (float)a2, p3, p4);
}

static float qk_w_f32_4244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, float, void*, void*))f)(p1, (float)a2, p3, p4);
}

static double qk_w_f64_4244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, float, void*, void*))f)(p1, (float)a2, p3, p4);
}

static void* qk_w_ptr_4244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, float, void*, void*))f)(p1, (float)a2, p3, p4);
}

static void qk_w_void_4311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, int32_t, int32_t))f)(p1, a2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_4311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, int32_t, int32_t))f)(p1, a2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_4311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, int32_t, int32_t))f)(p1, a2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_4311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, int32_t, int32_t))f)(p1, a2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_4311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, int32_t, int32_t))f)(p1, a2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_4312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, int32_t, float))f)(p1, a2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_4312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, int32_t, float))f)(p1, a2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_4312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, int32_t, float))f)(p1, a2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_4312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, int32_t, float))f)(p1, a2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_4312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, int32_t, float))f)(p1, a2, (int32_t)a3, (float)a4);
}

static void qk_w_void_4313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, int32_t, double))f)(p1, a2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_4313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, int32_t, double))f)(p1, a2, (int32_t)a3, a4);
}

static float qk_w_f32_4313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, int32_t, double))f)(p1, a2, (int32_t)a3, a4);
}

static double qk_w_f64_4313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, int32_t, double))f)(p1, a2, (int32_t)a3, a4);
}

static void* qk_w_ptr_4313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, int32_t, double))f)(p1, a2, (int32_t)a3, a4);
}

static void qk_w_void_4314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, int32_t, void*))f)(p1, a2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_4314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, int32_t, void*))f)(p1, a2, (int32_t)a3, p4);
}

static float qk_w_f32_4314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, int32_t, void*))f)(p1, a2, (int32_t)a3, p4);
}

static double qk_w_f64_4314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, int32_t, void*))f)(p1, a2, (int32_t)a3, p4);
}

static void* qk_w_ptr_4314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, int32_t, void*))f)(p1, a2, (int32_t)a3, p4);
}

static void qk_w_void_4321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, float, int32_t))f)(p1, a2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_4321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, float, int32_t))f)(p1, a2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_4321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, float, int32_t))f)(p1, a2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_4321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, float, int32_t))f)(p1, a2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_4321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, float, int32_t))f)(p1, a2, (float)a3, (int32_t)a4);
}

static void qk_w_void_4322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, float, float))f)(p1, a2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_4322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, float, float))f)(p1, a2, (float)a3, (float)a4);
}

static float qk_w_f32_4322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, float, float))f)(p1, a2, (float)a3, (float)a4);
}

static double qk_w_f64_4322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, float, float))f)(p1, a2, (float)a3, (float)a4);
}

static void* qk_w_ptr_4322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, float, float))f)(p1, a2, (float)a3, (float)a4);
}

static void qk_w_void_4323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, float, double))f)(p1, a2, (float)a3, a4);
}

static int32_t qk_w_i32_4323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, float, double))f)(p1, a2, (float)a3, a4);
}

static float qk_w_f32_4323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, float, double))f)(p1, a2, (float)a3, a4);
}

static double qk_w_f64_4323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, float, double))f)(p1, a2, (float)a3, a4);
}

static void* qk_w_ptr_4323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, float, double))f)(p1, a2, (float)a3, a4);
}

static void qk_w_void_4324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, float, void*))f)(p1, a2, (float)a3, p4);
}

static int32_t qk_w_i32_4324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, float, void*))f)(p1, a2, (float)a3, p4);
}

static float qk_w_f32_4324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, float, void*))f)(p1, a2, (float)a3, p4);
}

static double qk_w_f64_4324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, float, void*))f)(p1, a2, (float)a3, p4);
}

static void* qk_w_ptr_4324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, float, void*))f)(p1, a2, (float)a3, p4);
}

static void qk_w_void_4331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, double, int32_t))f)(p1, a2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_4331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, double, int32_t))f)(p1, a2, a3, (int32_t)a4);
}

static float qk_w_f32_4331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, double, int32_t))f)(p1, a2, a3, (int32_t)a4);
}

static double qk_w_f64_4331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, double, int32_t))f)(p1, a2, a3, (int32_t)a4);
}

static void* qk_w_ptr_4331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, double, int32_t))f)(p1, a2, a3, (int32_t)a4);
}

static void qk_w_void_4332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, double, float))f)(p1, a2, a3, (float)a4);
}

static int32_t qk_w_i32_4332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, double, float))f)(p1, a2, a3, (float)a4);
}

static float qk_w_f32_4332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, double, float))f)(p1, a2, a3, (float)a4);
}

static double qk_w_f64_4332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, double, float))f)(p1, a2, a3, (float)a4);
}

static void* qk_w_ptr_4332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, double, float))f)(p1, a2, a3, (float)a4);
}

static void qk_w_void_4333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, double, double))f)(p1, a2, a3, a4);
}

static int32_t qk_w_i32_4333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, double, double))f)(p1, a2, a3, a4);
}

static float qk_w_f32_4333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, double, double))f)(p1, a2, a3, a4);
}

static double qk_w_f64_4333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, double, double))f)(p1, a2, a3, a4);
}

static void* qk_w_ptr_4333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, double, double))f)(p1, a2, a3, a4);
}

static void qk_w_void_4334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, double, void*))f)(p1, a2, a3, p4);
}

static int32_t qk_w_i32_4334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, double, void*))f)(p1, a2, a3, p4);
}

static float qk_w_f32_4334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, double, void*))f)(p1, a2, a3, p4);
}

static double qk_w_f64_4334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, double, void*))f)(p1, a2, a3, p4);
}

static void* qk_w_ptr_4334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, double, void*))f)(p1, a2, a3, p4);
}

static void qk_w_void_4341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, void*, int32_t))f)(p1, a2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_4341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, void*, int32_t))f)(p1, a2, p3, (int32_t)a4);
}

static float qk_w_f32_4341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, void*, int32_t))f)(p1, a2, p3, (int32_t)a4);
}

static double qk_w_f64_4341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, void*, int32_t))f)(p1, a2, p3, (int32_t)a4);
}

static void* qk_w_ptr_4341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, void*, int32_t))f)(p1, a2, p3, (int32_t)a4);
}

static void qk_w_void_4342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, void*, float))f)(p1, a2, p3, (float)a4);
}

static int32_t qk_w_i32_4342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, void*, float))f)(p1, a2, p3, (float)a4);
}

static float qk_w_f32_4342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, void*, float))f)(p1, a2, p3, (float)a4);
}

static double qk_w_f64_4342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, void*, float))f)(p1, a2, p3, (float)a4);
}

static void* qk_w_ptr_4342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, void*, float))f)(p1, a2, p3, (float)a4);
}

static void qk_w_void_4343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, void*, double))f)(p1, a2, p3, a4);
}

static int32_t qk_w_i32_4343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, void*, double))f)(p1, a2, p3, a4);
}

static float qk_w_f32_4343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, void*, double))f)(p1, a2, p3, a4);
}

static double qk_w_f64_4343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, void*, double))f)(p1, a2, p3, a4);
}

static void* qk_w_ptr_4343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, void*, double))f)(p1, a2, p3, a4);
}

static void qk_w_void_4344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, double, void*, void*))f)(p1, a2, p3, p4);
}

static int32_t qk_w_i32_4344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, double, void*, void*))f)(p1, a2, p3, p4);
}

static float qk_w_f32_4344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, double, void*, void*))f)(p1, a2, p3, p4);
}

static double qk_w_f64_4344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, double, void*, void*))f)(p1, a2, p3, p4);
}

static void* qk_w_ptr_4344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, double, void*, void*))f)(p1, a2, p3, p4);
}

static void qk_w_void_4411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, int32_t, int32_t))f)(p1, p2, (int32_t)a3, (int32_t)a4);
}

static int32_t qk_w_i32_4411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, int32_t, int32_t))f)(p1, p2, (int32_t)a3, (int32_t)a4);
}

static float qk_w_f32_4411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, int32_t, int32_t))f)(p1, p2, (int32_t)a3, (int32_t)a4);
}

static double qk_w_f64_4411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, int32_t, int32_t))f)(p1, p2, (int32_t)a3, (int32_t)a4);
}

static void* qk_w_ptr_4411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, int32_t, int32_t))f)(p1, p2, (int32_t)a3, (int32_t)a4);
}

static void qk_w_void_4412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, int32_t, float))f)(p1, p2, (int32_t)a3, (float)a4);
}

static int32_t qk_w_i32_4412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, int32_t, float))f)(p1, p2, (int32_t)a3, (float)a4);
}

static float qk_w_f32_4412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, int32_t, float))f)(p1, p2, (int32_t)a3, (float)a4);
}

static double qk_w_f64_4412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, int32_t, float))f)(p1, p2, (int32_t)a3, (float)a4);
}

static void* qk_w_ptr_4412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, int32_t, float))f)(p1, p2, (int32_t)a3, (float)a4);
}

static void qk_w_void_4413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, int32_t, double))f)(p1, p2, (int32_t)a3, a4);
}

static int32_t qk_w_i32_4413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, int32_t, double))f)(p1, p2, (int32_t)a3, a4);
}

static float qk_w_f32_4413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, int32_t, double))f)(p1, p2, (int32_t)a3, a4);
}

static double qk_w_f64_4413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, int32_t, double))f)(p1, p2, (int32_t)a3, a4);
}

static void* qk_w_ptr_4413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, int32_t, double))f)(p1, p2, (int32_t)a3, a4);
}

static void qk_w_void_4414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, int32_t, void*))f)(p1, p2, (int32_t)a3, p4);
}

static int32_t qk_w_i32_4414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, int32_t, void*))f)(p1, p2, (int32_t)a3, p4);
}

static float qk_w_f32_4414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, int32_t, void*))f)(p1, p2, (int32_t)a3, p4);
}

static double qk_w_f64_4414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, int32_t, void*))f)(p1, p2, (int32_t)a3, p4);
}

static void* qk_w_ptr_4414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, int32_t, void*))f)(p1, p2, (int32_t)a3, p4);
}

static void qk_w_void_4421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, float, int32_t))f)(p1, p2, (float)a3, (int32_t)a4);
}

static int32_t qk_w_i32_4421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, float, int32_t))f)(p1, p2, (float)a3, (int32_t)a4);
}

static float qk_w_f32_4421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, float, int32_t))f)(p1, p2, (float)a3, (int32_t)a4);
}

static double qk_w_f64_4421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, float, int32_t))f)(p1, p2, (float)a3, (int32_t)a4);
}

static void* qk_w_ptr_4421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, float, int32_t))f)(p1, p2, (float)a3, (int32_t)a4);
}

static void qk_w_void_4422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, float, float))f)(p1, p2, (float)a3, (float)a4);
}

static int32_t qk_w_i32_4422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, float, float))f)(p1, p2, (float)a3, (float)a4);
}

static float qk_w_f32_4422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, float, float))f)(p1, p2, (float)a3, (float)a4);
}

static double qk_w_f64_4422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, float, float))f)(p1, p2, (float)a3, (float)a4);
}

static void* qk_w_ptr_4422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, float, float))f)(p1, p2, (float)a3, (float)a4);
}

static void qk_w_void_4423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, float, double))f)(p1, p2, (float)a3, a4);
}

static int32_t qk_w_i32_4423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, float, double))f)(p1, p2, (float)a3, a4);
}

static float qk_w_f32_4423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, float, double))f)(p1, p2, (float)a3, a4);
}

static double qk_w_f64_4423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, float, double))f)(p1, p2, (float)a3, a4);
}

static void* qk_w_ptr_4423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, float, double))f)(p1, p2, (float)a3, a4);
}

static void qk_w_void_4424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, float, void*))f)(p1, p2, (float)a3, p4);
}

static int32_t qk_w_i32_4424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, float, void*))f)(p1, p2, (float)a3, p4);
}

static float qk_w_f32_4424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, float, void*))f)(p1, p2, (float)a3, p4);
}

static double qk_w_f64_4424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, float, void*))f)(p1, p2, (float)a3, p4);
}

static void* qk_w_ptr_4424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, float, void*))f)(p1, p2, (float)a3, p4);
}

static void qk_w_void_4431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, double, int32_t))f)(p1, p2, a3, (int32_t)a4);
}

static int32_t qk_w_i32_4431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, double, int32_t))f)(p1, p2, a3, (int32_t)a4);
}

static float qk_w_f32_4431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, double, int32_t))f)(p1, p2, a3, (int32_t)a4);
}

static double qk_w_f64_4431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, double, int32_t))f)(p1, p2, a3, (int32_t)a4);
}

static void* qk_w_ptr_4431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, double, int32_t))f)(p1, p2, a3, (int32_t)a4);
}

static void qk_w_void_4432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, double, float))f)(p1, p2, a3, (float)a4);
}

static int32_t qk_w_i32_4432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, double, float))f)(p1, p2, a3, (float)a4);
}

static float qk_w_f32_4432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, double, float))f)(p1, p2, a3, (float)a4);
}

static double qk_w_f64_4432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, double, float))f)(p1, p2, a3, (float)a4);
}

static void* qk_w_ptr_4432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, double, float))f)(p1, p2, a3, (float)a4);
}

static void qk_w_void_4433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, double, double))f)(p1, p2, a3, a4);
}

static int32_t qk_w_i32_4433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, double, double))f)(p1, p2, a3, a4);
}

static float qk_w_f32_4433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, double, double))f)(p1, p2, a3, a4);
}

static double qk_w_f64_4433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, double, double))f)(p1, p2, a3, a4);
}

static void* qk_w_ptr_4433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, double, double))f)(p1, p2, a3, a4);
}

static void qk_w_void_4434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, double, void*))f)(p1, p2, a3, p4);
}

static int32_t qk_w_i32_4434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, double, void*))f)(p1, p2, a3, p4);
}

static float qk_w_f32_4434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, double, void*))f)(p1, p2, a3, p4);
}

static double qk_w_f64_4434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, double, void*))f)(p1, p2, a3, p4);
}

static void* qk_w_ptr_4434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, double, void*))f)(p1, p2, a3, p4);
}

static void qk_w_void_4441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, void*, int32_t))f)(p1, p2, p3, (int32_t)a4);
}

static int32_t qk_w_i32_4441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, void*, int32_t))f)(p1, p2, p3, (int32_t)a4);
}

static float qk_w_f32_4441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, void*, int32_t))f)(p1, p2, p3, (int32_t)a4);
}

static double qk_w_f64_4441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, void*, int32_t))f)(p1, p2, p3, (int32_t)a4);
}

static void* qk_w_ptr_4441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, void*, int32_t))f)(p1, p2, p3, (int32_t)a4);
}

static void qk_w_void_4442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, void*, float))f)(p1, p2, p3, (float)a4);
}

static int32_t qk_w_i32_4442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, void*, float))f)(p1, p2, p3, (float)a4);
}

static float qk_w_f32_4442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, void*, float))f)(p1, p2, p3, (float)a4);
}

static double qk_w_f64_4442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, void*, float))f)(p1, p2, p3, (float)a4);
}

static void* qk_w_ptr_4442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, void*, float))f)(p1, p2, p3, (float)a4);
}

static void qk_w_void_4443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, void*, double))f)(p1, p2, p3, a4);
}

static int32_t qk_w_i32_4443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, void*, double))f)(p1, p2, p3, a4);
}

static float qk_w_f32_4443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, void*, double))f)(p1, p2, p3, a4);
}

static double qk_w_f64_4443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, void*, double))f)(p1, p2, p3, a4);
}

static void* qk_w_ptr_4443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, void*, double))f)(p1, p2, p3, a4);
}

static void qk_w_void_4444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    ((void(*)(void*, void*, void*, void*))f)(p1, p2, p3, p4);
}

static int32_t qk_w_i32_4444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int32_t(*)(void*, void*, void*, void*))f)(p1, p2, p3, p4);
}

static float qk_w_f32_4444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((float(*)(void*, void*, void*, void*))f)(p1, p2, p3, p4);
}

static double qk_w_f64_4444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((double(*)(void*, void*, void*, void*))f)(p1, p2, p3, p4);
}

static void* qk_w_ptr_4444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((void*(*)(void*, void*, void*, void*))f)(p1, p2, p3, p4);
}


/* ---- i64（long/64 位整数）返回族：补全 dispatch 表（此前缺失，导致任何返回 long 的
   C 函数在 Windows 上必然报 “ffi call failed”） ---- */
static int64_t qk_w_i64_0(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)())f)();
}
static int64_t qk_w_i64_1(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t))f)((int32_t)a1);
}
static int64_t qk_w_i64_2(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float))f)((float)a1);
}
static int64_t qk_w_i64_3(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double))f)(a1);
}
static int64_t qk_w_i64_4(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*))f)(p1);
}
static int64_t qk_w_i64_11(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t))f)((int32_t)a1, (int32_t)a2);
}
static int64_t qk_w_i64_12(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float))f)((int32_t)a1, (float)a2);
}
static int64_t qk_w_i64_13(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double))f)((int32_t)a1, a2);
}
static int64_t qk_w_i64_14(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*))f)((int32_t)a1, p2);
}
static int64_t qk_w_i64_21(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t))f)((float)a1, (int32_t)a2);
}
static int64_t qk_w_i64_22(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float))f)((float)a1, (float)a2);
}
static int64_t qk_w_i64_23(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double))f)((float)a1, a2);
}
static int64_t qk_w_i64_24(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*))f)((float)a1, p2);
}
static int64_t qk_w_i64_31(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t))f)(a1, (int32_t)a2);
}
static int64_t qk_w_i64_32(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float))f)(a1, (float)a2);
}
static int64_t qk_w_i64_33(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double))f)(a1, a2);
}
static int64_t qk_w_i64_34(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*))f)(a1, p2);
}
static int64_t qk_w_i64_41(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t))f)(p1, (int32_t)a2);
}
static int64_t qk_w_i64_42(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float))f)(p1, (float)a2);
}
static int64_t qk_w_i64_43(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double))f)(p1, a2);
}
static int64_t qk_w_i64_44(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*))f)(p1, p2);
}
static int64_t qk_w_i64_111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3);
}
static int64_t qk_w_i64_112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (float)a3);
}
static int64_t qk_w_i64_113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, a3);
}
static int64_t qk_w_i64_114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, p3);
}
static int64_t qk_w_i64_121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3);
}
static int64_t qk_w_i64_122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, float))f)((int32_t)a1, (float)a2, (float)a3);
}
static int64_t qk_w_i64_123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, double))f)((int32_t)a1, (float)a2, a3);
}
static int64_t qk_w_i64_124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, void*))f)((int32_t)a1, (float)a2, p3);
}
static int64_t qk_w_i64_131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, int32_t))f)((int32_t)a1, a2, (int32_t)a3);
}
static int64_t qk_w_i64_132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, float))f)((int32_t)a1, a2, (float)a3);
}
static int64_t qk_w_i64_133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, double))f)((int32_t)a1, a2, a3);
}
static int64_t qk_w_i64_134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, void*))f)((int32_t)a1, a2, p3);
}
static int64_t qk_w_i64_141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, int32_t))f)((int32_t)a1, p2, (int32_t)a3);
}
static int64_t qk_w_i64_142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, float))f)((int32_t)a1, p2, (float)a3);
}
static int64_t qk_w_i64_143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, double))f)((int32_t)a1, p2, a3);
}
static int64_t qk_w_i64_144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, void*))f)((int32_t)a1, p2, p3);
}
static int64_t qk_w_i64_211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3);
}
static int64_t qk_w_i64_212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, float))f)((float)a1, (int32_t)a2, (float)a3);
}
static int64_t qk_w_i64_213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, double))f)((float)a1, (int32_t)a2, a3);
}
static int64_t qk_w_i64_214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, void*))f)((float)a1, (int32_t)a2, p3);
}
static int64_t qk_w_i64_221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, int32_t))f)((float)a1, (float)a2, (int32_t)a3);
}
static int64_t qk_w_i64_222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, float))f)((float)a1, (float)a2, (float)a3);
}
static int64_t qk_w_i64_223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, double))f)((float)a1, (float)a2, a3);
}
static int64_t qk_w_i64_224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, void*))f)((float)a1, (float)a2, p3);
}
static int64_t qk_w_i64_231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, int32_t))f)((float)a1, a2, (int32_t)a3);
}
static int64_t qk_w_i64_232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, float))f)((float)a1, a2, (float)a3);
}
static int64_t qk_w_i64_233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, double))f)((float)a1, a2, a3);
}
static int64_t qk_w_i64_234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, void*))f)((float)a1, a2, p3);
}
static int64_t qk_w_i64_241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, int32_t))f)((float)a1, p2, (int32_t)a3);
}
static int64_t qk_w_i64_242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, float))f)((float)a1, p2, (float)a3);
}
static int64_t qk_w_i64_243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, double))f)((float)a1, p2, a3);
}
static int64_t qk_w_i64_244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, void*))f)((float)a1, p2, p3);
}
static int64_t qk_w_i64_311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3);
}
static int64_t qk_w_i64_312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, float))f)(a1, (int32_t)a2, (float)a3);
}
static int64_t qk_w_i64_313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, double))f)(a1, (int32_t)a2, a3);
}
static int64_t qk_w_i64_314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, void*))f)(a1, (int32_t)a2, p3);
}
static int64_t qk_w_i64_321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, int32_t))f)(a1, (float)a2, (int32_t)a3);
}
static int64_t qk_w_i64_322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, float))f)(a1, (float)a2, (float)a3);
}
static int64_t qk_w_i64_323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, double))f)(a1, (float)a2, a3);
}
static int64_t qk_w_i64_324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, void*))f)(a1, (float)a2, p3);
}
static int64_t qk_w_i64_331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, int32_t))f)(a1, a2, (int32_t)a3);
}
static int64_t qk_w_i64_332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, float))f)(a1, a2, (float)a3);
}
static int64_t qk_w_i64_333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, double))f)(a1, a2, a3);
}
static int64_t qk_w_i64_334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, void*))f)(a1, a2, p3);
}
static int64_t qk_w_i64_341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, int32_t))f)(a1, p2, (int32_t)a3);
}
static int64_t qk_w_i64_342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, float))f)(a1, p2, (float)a3);
}
static int64_t qk_w_i64_343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, double))f)(a1, p2, a3);
}
static int64_t qk_w_i64_344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, void*))f)(a1, p2, p3);
}
static int64_t qk_w_i64_411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3);
}
static int64_t qk_w_i64_412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, float))f)(p1, (int32_t)a2, (float)a3);
}
static int64_t qk_w_i64_413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, double))f)(p1, (int32_t)a2, a3);
}
static int64_t qk_w_i64_414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, void*))f)(p1, (int32_t)a2, p3);
}
static int64_t qk_w_i64_421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, int32_t))f)(p1, (float)a2, (int32_t)a3);
}
static int64_t qk_w_i64_422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, float))f)(p1, (float)a2, (float)a3);
}
static int64_t qk_w_i64_423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, double))f)(p1, (float)a2, a3);
}
static int64_t qk_w_i64_424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, void*))f)(p1, (float)a2, p3);
}
static int64_t qk_w_i64_431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, int32_t))f)(p1, a2, (int32_t)a3);
}
static int64_t qk_w_i64_432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, float))f)(p1, a2, (float)a3);
}
static int64_t qk_w_i64_433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, double))f)(p1, a2, a3);
}
static int64_t qk_w_i64_434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, void*))f)(p1, a2, p3);
}
static int64_t qk_w_i64_441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, int32_t))f)(p1, p2, (int32_t)a3);
}
static int64_t qk_w_i64_442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, float))f)(p1, p2, (float)a3);
}
static int64_t qk_w_i64_443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, double))f)(p1, p2, a3);
}
static int64_t qk_w_i64_444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, void*))f)(p1, p2, p3);
}
static int64_t qk_w_i64_1111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, int32_t, int32_t))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_1112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, int32_t, float))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_1113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, int32_t, double))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_1114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, int32_t, void*))f)((int32_t)a1, (int32_t)a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_1121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, float, int32_t))f)((int32_t)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_1122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, float, float))f)((int32_t)a1, (int32_t)a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_1123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, float, double))f)((int32_t)a1, (int32_t)a2, (float)a3, a4);
}
static int64_t qk_w_i64_1124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, float, void*))f)((int32_t)a1, (int32_t)a2, (float)a3, p4);
}
static int64_t qk_w_i64_1131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, double, int32_t))f)((int32_t)a1, (int32_t)a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_1132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, double, float))f)((int32_t)a1, (int32_t)a2, a3, (float)a4);
}
static int64_t qk_w_i64_1133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, double, double))f)((int32_t)a1, (int32_t)a2, a3, a4);
}
static int64_t qk_w_i64_1134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, double, void*))f)((int32_t)a1, (int32_t)a2, a3, p4);
}
static int64_t qk_w_i64_1141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, void*, int32_t))f)((int32_t)a1, (int32_t)a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_1142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, void*, float))f)((int32_t)a1, (int32_t)a2, p3, (float)a4);
}
static int64_t qk_w_i64_1143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, void*, double))f)((int32_t)a1, (int32_t)a2, p3, a4);
}
static int64_t qk_w_i64_1144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, int32_t, void*, void*))f)((int32_t)a1, (int32_t)a2, p3, p4);
}
static int64_t qk_w_i64_1211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, int32_t, int32_t))f)((int32_t)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_1212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, int32_t, float))f)((int32_t)a1, (float)a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_1213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, int32_t, double))f)((int32_t)a1, (float)a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_1214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, int32_t, void*))f)((int32_t)a1, (float)a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_1221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, float, int32_t))f)((int32_t)a1, (float)a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_1222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, float, float))f)((int32_t)a1, (float)a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_1223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, float, double))f)((int32_t)a1, (float)a2, (float)a3, a4);
}
static int64_t qk_w_i64_1224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, float, void*))f)((int32_t)a1, (float)a2, (float)a3, p4);
}
static int64_t qk_w_i64_1231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, double, int32_t))f)((int32_t)a1, (float)a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_1232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, double, float))f)((int32_t)a1, (float)a2, a3, (float)a4);
}
static int64_t qk_w_i64_1233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, double, double))f)((int32_t)a1, (float)a2, a3, a4);
}
static int64_t qk_w_i64_1234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, double, void*))f)((int32_t)a1, (float)a2, a3, p4);
}
static int64_t qk_w_i64_1241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, void*, int32_t))f)((int32_t)a1, (float)a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_1242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, void*, float))f)((int32_t)a1, (float)a2, p3, (float)a4);
}
static int64_t qk_w_i64_1243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, void*, double))f)((int32_t)a1, (float)a2, p3, a4);
}
static int64_t qk_w_i64_1244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, float, void*, void*))f)((int32_t)a1, (float)a2, p3, p4);
}
static int64_t qk_w_i64_1311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, int32_t, int32_t))f)((int32_t)a1, a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_1312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, int32_t, float))f)((int32_t)a1, a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_1313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, int32_t, double))f)((int32_t)a1, a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_1314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, int32_t, void*))f)((int32_t)a1, a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_1321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, float, int32_t))f)((int32_t)a1, a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_1322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, float, float))f)((int32_t)a1, a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_1323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, float, double))f)((int32_t)a1, a2, (float)a3, a4);
}
static int64_t qk_w_i64_1324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, float, void*))f)((int32_t)a1, a2, (float)a3, p4);
}
static int64_t qk_w_i64_1331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, double, int32_t))f)((int32_t)a1, a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_1332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, double, float))f)((int32_t)a1, a2, a3, (float)a4);
}
static int64_t qk_w_i64_1333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, double, double))f)((int32_t)a1, a2, a3, a4);
}
static int64_t qk_w_i64_1334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, double, void*))f)((int32_t)a1, a2, a3, p4);
}
static int64_t qk_w_i64_1341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, void*, int32_t))f)((int32_t)a1, a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_1342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, void*, float))f)((int32_t)a1, a2, p3, (float)a4);
}
static int64_t qk_w_i64_1343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, void*, double))f)((int32_t)a1, a2, p3, a4);
}
static int64_t qk_w_i64_1344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, double, void*, void*))f)((int32_t)a1, a2, p3, p4);
}
static int64_t qk_w_i64_1411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, int32_t, int32_t))f)((int32_t)a1, p2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_1412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, int32_t, float))f)((int32_t)a1, p2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_1413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, int32_t, double))f)((int32_t)a1, p2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_1414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, int32_t, void*))f)((int32_t)a1, p2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_1421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, float, int32_t))f)((int32_t)a1, p2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_1422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, float, float))f)((int32_t)a1, p2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_1423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, float, double))f)((int32_t)a1, p2, (float)a3, a4);
}
static int64_t qk_w_i64_1424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, float, void*))f)((int32_t)a1, p2, (float)a3, p4);
}
static int64_t qk_w_i64_1431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, double, int32_t))f)((int32_t)a1, p2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_1432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, double, float))f)((int32_t)a1, p2, a3, (float)a4);
}
static int64_t qk_w_i64_1433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, double, double))f)((int32_t)a1, p2, a3, a4);
}
static int64_t qk_w_i64_1434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, double, void*))f)((int32_t)a1, p2, a3, p4);
}
static int64_t qk_w_i64_1441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, void*, int32_t))f)((int32_t)a1, p2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_1442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, void*, float))f)((int32_t)a1, p2, p3, (float)a4);
}
static int64_t qk_w_i64_1443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, void*, double))f)((int32_t)a1, p2, p3, a4);
}
static int64_t qk_w_i64_1444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(int32_t, void*, void*, void*))f)((int32_t)a1, p2, p3, p4);
}
static int64_t qk_w_i64_2111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, int32_t, int32_t))f)((float)a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_2112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, int32_t, float))f)((float)a1, (int32_t)a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_2113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, int32_t, double))f)((float)a1, (int32_t)a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_2114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, int32_t, void*))f)((float)a1, (int32_t)a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_2121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, float, int32_t))f)((float)a1, (int32_t)a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_2122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, float, float))f)((float)a1, (int32_t)a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_2123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, float, double))f)((float)a1, (int32_t)a2, (float)a3, a4);
}
static int64_t qk_w_i64_2124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, float, void*))f)((float)a1, (int32_t)a2, (float)a3, p4);
}
static int64_t qk_w_i64_2131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, double, int32_t))f)((float)a1, (int32_t)a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_2132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, double, float))f)((float)a1, (int32_t)a2, a3, (float)a4);
}
static int64_t qk_w_i64_2133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, double, double))f)((float)a1, (int32_t)a2, a3, a4);
}
static int64_t qk_w_i64_2134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, double, void*))f)((float)a1, (int32_t)a2, a3, p4);
}
static int64_t qk_w_i64_2141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, void*, int32_t))f)((float)a1, (int32_t)a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_2142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, void*, float))f)((float)a1, (int32_t)a2, p3, (float)a4);
}
static int64_t qk_w_i64_2143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, void*, double))f)((float)a1, (int32_t)a2, p3, a4);
}
static int64_t qk_w_i64_2144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, int32_t, void*, void*))f)((float)a1, (int32_t)a2, p3, p4);
}
static int64_t qk_w_i64_2211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, int32_t, int32_t))f)((float)a1, (float)a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_2212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, int32_t, float))f)((float)a1, (float)a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_2213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, int32_t, double))f)((float)a1, (float)a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_2214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, int32_t, void*))f)((float)a1, (float)a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_2221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, float, int32_t))f)((float)a1, (float)a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_2222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, float, float))f)((float)a1, (float)a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_2223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, float, double))f)((float)a1, (float)a2, (float)a3, a4);
}
static int64_t qk_w_i64_2224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, float, void*))f)((float)a1, (float)a2, (float)a3, p4);
}
static int64_t qk_w_i64_2231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, double, int32_t))f)((float)a1, (float)a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_2232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, double, float))f)((float)a1, (float)a2, a3, (float)a4);
}
static int64_t qk_w_i64_2233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, double, double))f)((float)a1, (float)a2, a3, a4);
}
static int64_t qk_w_i64_2234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, double, void*))f)((float)a1, (float)a2, a3, p4);
}
static int64_t qk_w_i64_2241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, void*, int32_t))f)((float)a1, (float)a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_2242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, void*, float))f)((float)a1, (float)a2, p3, (float)a4);
}
static int64_t qk_w_i64_2243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, void*, double))f)((float)a1, (float)a2, p3, a4);
}
static int64_t qk_w_i64_2244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, float, void*, void*))f)((float)a1, (float)a2, p3, p4);
}
static int64_t qk_w_i64_2311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, int32_t, int32_t))f)((float)a1, a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_2312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, int32_t, float))f)((float)a1, a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_2313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, int32_t, double))f)((float)a1, a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_2314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, int32_t, void*))f)((float)a1, a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_2321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, float, int32_t))f)((float)a1, a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_2322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, float, float))f)((float)a1, a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_2323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, float, double))f)((float)a1, a2, (float)a3, a4);
}
static int64_t qk_w_i64_2324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, float, void*))f)((float)a1, a2, (float)a3, p4);
}
static int64_t qk_w_i64_2331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, double, int32_t))f)((float)a1, a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_2332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, double, float))f)((float)a1, a2, a3, (float)a4);
}
static int64_t qk_w_i64_2333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, double, double))f)((float)a1, a2, a3, a4);
}
static int64_t qk_w_i64_2334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, double, void*))f)((float)a1, a2, a3, p4);
}
static int64_t qk_w_i64_2341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, void*, int32_t))f)((float)a1, a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_2342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, void*, float))f)((float)a1, a2, p3, (float)a4);
}
static int64_t qk_w_i64_2343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, void*, double))f)((float)a1, a2, p3, a4);
}
static int64_t qk_w_i64_2344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, double, void*, void*))f)((float)a1, a2, p3, p4);
}
static int64_t qk_w_i64_2411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, int32_t, int32_t))f)((float)a1, p2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_2412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, int32_t, float))f)((float)a1, p2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_2413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, int32_t, double))f)((float)a1, p2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_2414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, int32_t, void*))f)((float)a1, p2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_2421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, float, int32_t))f)((float)a1, p2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_2422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, float, float))f)((float)a1, p2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_2423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, float, double))f)((float)a1, p2, (float)a3, a4);
}
static int64_t qk_w_i64_2424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, float, void*))f)((float)a1, p2, (float)a3, p4);
}
static int64_t qk_w_i64_2431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, double, int32_t))f)((float)a1, p2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_2432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, double, float))f)((float)a1, p2, a3, (float)a4);
}
static int64_t qk_w_i64_2433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, double, double))f)((float)a1, p2, a3, a4);
}
static int64_t qk_w_i64_2434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, double, void*))f)((float)a1, p2, a3, p4);
}
static int64_t qk_w_i64_2441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, void*, int32_t))f)((float)a1, p2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_2442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, void*, float))f)((float)a1, p2, p3, (float)a4);
}
static int64_t qk_w_i64_2443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, void*, double))f)((float)a1, p2, p3, a4);
}
static int64_t qk_w_i64_2444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(float, void*, void*, void*))f)((float)a1, p2, p3, p4);
}
static int64_t qk_w_i64_3111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, int32_t, int32_t))f)(a1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_3112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, int32_t, float))f)(a1, (int32_t)a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_3113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, int32_t, double))f)(a1, (int32_t)a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_3114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, int32_t, void*))f)(a1, (int32_t)a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_3121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, float, int32_t))f)(a1, (int32_t)a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_3122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, float, float))f)(a1, (int32_t)a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_3123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, float, double))f)(a1, (int32_t)a2, (float)a3, a4);
}
static int64_t qk_w_i64_3124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, float, void*))f)(a1, (int32_t)a2, (float)a3, p4);
}
static int64_t qk_w_i64_3131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, double, int32_t))f)(a1, (int32_t)a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_3132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, double, float))f)(a1, (int32_t)a2, a3, (float)a4);
}
static int64_t qk_w_i64_3133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, double, double))f)(a1, (int32_t)a2, a3, a4);
}
static int64_t qk_w_i64_3134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, double, void*))f)(a1, (int32_t)a2, a3, p4);
}
static int64_t qk_w_i64_3141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, void*, int32_t))f)(a1, (int32_t)a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_3142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, void*, float))f)(a1, (int32_t)a2, p3, (float)a4);
}
static int64_t qk_w_i64_3143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, void*, double))f)(a1, (int32_t)a2, p3, a4);
}
static int64_t qk_w_i64_3144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, int32_t, void*, void*))f)(a1, (int32_t)a2, p3, p4);
}
static int64_t qk_w_i64_3211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, int32_t, int32_t))f)(a1, (float)a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_3212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, int32_t, float))f)(a1, (float)a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_3213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, int32_t, double))f)(a1, (float)a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_3214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, int32_t, void*))f)(a1, (float)a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_3221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, float, int32_t))f)(a1, (float)a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_3222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, float, float))f)(a1, (float)a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_3223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, float, double))f)(a1, (float)a2, (float)a3, a4);
}
static int64_t qk_w_i64_3224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, float, void*))f)(a1, (float)a2, (float)a3, p4);
}
static int64_t qk_w_i64_3231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, double, int32_t))f)(a1, (float)a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_3232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, double, float))f)(a1, (float)a2, a3, (float)a4);
}
static int64_t qk_w_i64_3233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, double, double))f)(a1, (float)a2, a3, a4);
}
static int64_t qk_w_i64_3234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, double, void*))f)(a1, (float)a2, a3, p4);
}
static int64_t qk_w_i64_3241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, void*, int32_t))f)(a1, (float)a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_3242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, void*, float))f)(a1, (float)a2, p3, (float)a4);
}
static int64_t qk_w_i64_3243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, void*, double))f)(a1, (float)a2, p3, a4);
}
static int64_t qk_w_i64_3244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, float, void*, void*))f)(a1, (float)a2, p3, p4);
}
static int64_t qk_w_i64_3311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, int32_t, int32_t))f)(a1, a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_3312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, int32_t, float))f)(a1, a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_3313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, int32_t, double))f)(a1, a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_3314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, int32_t, void*))f)(a1, a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_3321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, float, int32_t))f)(a1, a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_3322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, float, float))f)(a1, a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_3323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, float, double))f)(a1, a2, (float)a3, a4);
}
static int64_t qk_w_i64_3324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, float, void*))f)(a1, a2, (float)a3, p4);
}
static int64_t qk_w_i64_3331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, double, int32_t))f)(a1, a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_3332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, double, float))f)(a1, a2, a3, (float)a4);
}
static int64_t qk_w_i64_3333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, double, double))f)(a1, a2, a3, a4);
}
static int64_t qk_w_i64_3334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, double, void*))f)(a1, a2, a3, p4);
}
static int64_t qk_w_i64_3341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, void*, int32_t))f)(a1, a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_3342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, void*, float))f)(a1, a2, p3, (float)a4);
}
static int64_t qk_w_i64_3343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, void*, double))f)(a1, a2, p3, a4);
}
static int64_t qk_w_i64_3344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, double, void*, void*))f)(a1, a2, p3, p4);
}
static int64_t qk_w_i64_3411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, int32_t, int32_t))f)(a1, p2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_3412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, int32_t, float))f)(a1, p2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_3413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, int32_t, double))f)(a1, p2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_3414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, int32_t, void*))f)(a1, p2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_3421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, float, int32_t))f)(a1, p2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_3422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, float, float))f)(a1, p2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_3423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, float, double))f)(a1, p2, (float)a3, a4);
}
static int64_t qk_w_i64_3424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, float, void*))f)(a1, p2, (float)a3, p4);
}
static int64_t qk_w_i64_3431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, double, int32_t))f)(a1, p2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_3432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, double, float))f)(a1, p2, a3, (float)a4);
}
static int64_t qk_w_i64_3433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, double, double))f)(a1, p2, a3, a4);
}
static int64_t qk_w_i64_3434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, double, void*))f)(a1, p2, a3, p4);
}
static int64_t qk_w_i64_3441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, void*, int32_t))f)(a1, p2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_3442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, void*, float))f)(a1, p2, p3, (float)a4);
}
static int64_t qk_w_i64_3443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, void*, double))f)(a1, p2, p3, a4);
}
static int64_t qk_w_i64_3444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(double, void*, void*, void*))f)(a1, p2, p3, p4);
}
static int64_t qk_w_i64_4111(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, int32_t, int32_t))f)(p1, (int32_t)a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_4112(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, int32_t, float))f)(p1, (int32_t)a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_4113(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, int32_t, double))f)(p1, (int32_t)a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_4114(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, int32_t, void*))f)(p1, (int32_t)a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_4121(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, float, int32_t))f)(p1, (int32_t)a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_4122(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, float, float))f)(p1, (int32_t)a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_4123(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, float, double))f)(p1, (int32_t)a2, (float)a3, a4);
}
static int64_t qk_w_i64_4124(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, float, void*))f)(p1, (int32_t)a2, (float)a3, p4);
}
static int64_t qk_w_i64_4131(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, double, int32_t))f)(p1, (int32_t)a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_4132(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, double, float))f)(p1, (int32_t)a2, a3, (float)a4);
}
static int64_t qk_w_i64_4133(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, double, double))f)(p1, (int32_t)a2, a3, a4);
}
static int64_t qk_w_i64_4134(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, double, void*))f)(p1, (int32_t)a2, a3, p4);
}
static int64_t qk_w_i64_4141(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, void*, int32_t))f)(p1, (int32_t)a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_4142(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, void*, float))f)(p1, (int32_t)a2, p3, (float)a4);
}
static int64_t qk_w_i64_4143(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, void*, double))f)(p1, (int32_t)a2, p3, a4);
}
static int64_t qk_w_i64_4144(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, int32_t, void*, void*))f)(p1, (int32_t)a2, p3, p4);
}
static int64_t qk_w_i64_4211(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, int32_t, int32_t))f)(p1, (float)a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_4212(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, int32_t, float))f)(p1, (float)a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_4213(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, int32_t, double))f)(p1, (float)a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_4214(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, int32_t, void*))f)(p1, (float)a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_4221(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, float, int32_t))f)(p1, (float)a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_4222(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, float, float))f)(p1, (float)a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_4223(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, float, double))f)(p1, (float)a2, (float)a3, a4);
}
static int64_t qk_w_i64_4224(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, float, void*))f)(p1, (float)a2, (float)a3, p4);
}
static int64_t qk_w_i64_4231(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, double, int32_t))f)(p1, (float)a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_4232(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, double, float))f)(p1, (float)a2, a3, (float)a4);
}
static int64_t qk_w_i64_4233(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, double, double))f)(p1, (float)a2, a3, a4);
}
static int64_t qk_w_i64_4234(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, double, void*))f)(p1, (float)a2, a3, p4);
}
static int64_t qk_w_i64_4241(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, void*, int32_t))f)(p1, (float)a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_4242(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, void*, float))f)(p1, (float)a2, p3, (float)a4);
}
static int64_t qk_w_i64_4243(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, void*, double))f)(p1, (float)a2, p3, a4);
}
static int64_t qk_w_i64_4244(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, float, void*, void*))f)(p1, (float)a2, p3, p4);
}
static int64_t qk_w_i64_4311(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, int32_t, int32_t))f)(p1, a2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_4312(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, int32_t, float))f)(p1, a2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_4313(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, int32_t, double))f)(p1, a2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_4314(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, int32_t, void*))f)(p1, a2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_4321(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, float, int32_t))f)(p1, a2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_4322(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, float, float))f)(p1, a2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_4323(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, float, double))f)(p1, a2, (float)a3, a4);
}
static int64_t qk_w_i64_4324(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, float, void*))f)(p1, a2, (float)a3, p4);
}
static int64_t qk_w_i64_4331(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, double, int32_t))f)(p1, a2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_4332(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, double, float))f)(p1, a2, a3, (float)a4);
}
static int64_t qk_w_i64_4333(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, double, double))f)(p1, a2, a3, a4);
}
static int64_t qk_w_i64_4334(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, double, void*))f)(p1, a2, a3, p4);
}
static int64_t qk_w_i64_4341(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, void*, int32_t))f)(p1, a2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_4342(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, void*, float))f)(p1, a2, p3, (float)a4);
}
static int64_t qk_w_i64_4343(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, void*, double))f)(p1, a2, p3, a4);
}
static int64_t qk_w_i64_4344(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, double, void*, void*))f)(p1, a2, p3, p4);
}
static int64_t qk_w_i64_4411(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, int32_t, int32_t))f)(p1, p2, (int32_t)a3, (int32_t)a4);
}
static int64_t qk_w_i64_4412(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, int32_t, float))f)(p1, p2, (int32_t)a3, (float)a4);
}
static int64_t qk_w_i64_4413(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, int32_t, double))f)(p1, p2, (int32_t)a3, a4);
}
static int64_t qk_w_i64_4414(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, int32_t, void*))f)(p1, p2, (int32_t)a3, p4);
}
static int64_t qk_w_i64_4421(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, float, int32_t))f)(p1, p2, (float)a3, (int32_t)a4);
}
static int64_t qk_w_i64_4422(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, float, float))f)(p1, p2, (float)a3, (float)a4);
}
static int64_t qk_w_i64_4423(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, float, double))f)(p1, p2, (float)a3, a4);
}
static int64_t qk_w_i64_4424(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, float, void*))f)(p1, p2, (float)a3, p4);
}
static int64_t qk_w_i64_4431(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, double, int32_t))f)(p1, p2, a3, (int32_t)a4);
}
static int64_t qk_w_i64_4432(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, double, float))f)(p1, p2, a3, (float)a4);
}
static int64_t qk_w_i64_4433(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, double, double))f)(p1, p2, a3, a4);
}
static int64_t qk_w_i64_4434(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, double, void*))f)(p1, p2, a3, p4);
}
static int64_t qk_w_i64_4441(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, void*, int32_t))f)(p1, p2, p3, (int32_t)a4);
}
static int64_t qk_w_i64_4442(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, void*, float))f)(p1, p2, p3, (float)a4);
}
static int64_t qk_w_i64_4443(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, void*, double))f)(p1, p2, p3, a4);
}
static int64_t qk_w_i64_4444(void* f, double a1, double a2, double a3, double a4, void* p1, void* p2, void* p3, void* p4) {
    return ((int64_t(*)(void*, void*, void*, void*))f)(p1, p2, p3, p4);
}

int qk_ffi(void* fn, const int* types, const double* nums, void** ptrs, int n,
           int rettype, int64_t* ret_i, double* ret_f, void* ret_slot) {
    if (!fn) return -1;
    double a1=0,a2=0,a3=0,a4=0; void* p1=0,*p2=0,*p3=0,*p4=0;
    if (n >= 1) { a1 = nums[0]; p1 = ptrs[0]; }
    if (n >= 2) { a2 = nums[1]; p2 = ptrs[1]; }
    if (n >= 3) { a3 = nums[2]; p3 = ptrs[2]; }
    if (n >= 4) { a4 = nums[3]; p4 = ptrs[3]; }
    int64_t code = (int64_t)rettype * 100000; /* 64 位：4 参数时 code 可达 4e9，int 会溢出 */
    int i;
    for (i = 0; i < n; i++) code = code*10 + types[i];
    switch (code) {
        case 0: qk_w_void_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1: qk_w_void_1(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2: qk_w_void_2(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3: qk_w_void_3(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4: qk_w_void_4(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 11: qk_w_void_11(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 12: qk_w_void_12(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 13: qk_w_void_13(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 14: qk_w_void_14(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 21: qk_w_void_21(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 22: qk_w_void_22(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 23: qk_w_void_23(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 24: qk_w_void_24(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 31: qk_w_void_31(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 32: qk_w_void_32(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 33: qk_w_void_33(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 34: qk_w_void_34(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 41: qk_w_void_41(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 42: qk_w_void_42(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 43: qk_w_void_43(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 44: qk_w_void_44(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 111: qk_w_void_111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 112: qk_w_void_112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 113: qk_w_void_113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 114: qk_w_void_114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 121: qk_w_void_121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 122: qk_w_void_122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 123: qk_w_void_123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 124: qk_w_void_124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 131: qk_w_void_131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 132: qk_w_void_132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 133: qk_w_void_133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 134: qk_w_void_134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 141: qk_w_void_141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 142: qk_w_void_142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 143: qk_w_void_143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 144: qk_w_void_144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 211: qk_w_void_211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 212: qk_w_void_212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 213: qk_w_void_213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 214: qk_w_void_214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 221: qk_w_void_221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 222: qk_w_void_222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 223: qk_w_void_223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 224: qk_w_void_224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 231: qk_w_void_231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 232: qk_w_void_232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 233: qk_w_void_233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 234: qk_w_void_234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 241: qk_w_void_241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 242: qk_w_void_242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 243: qk_w_void_243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 244: qk_w_void_244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 311: qk_w_void_311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 312: qk_w_void_312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 313: qk_w_void_313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 314: qk_w_void_314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 321: qk_w_void_321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 322: qk_w_void_322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 323: qk_w_void_323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 324: qk_w_void_324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 331: qk_w_void_331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 332: qk_w_void_332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 333: qk_w_void_333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 334: qk_w_void_334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 341: qk_w_void_341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 342: qk_w_void_342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 343: qk_w_void_343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 344: qk_w_void_344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 411: qk_w_void_411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 412: qk_w_void_412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 413: qk_w_void_413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 414: qk_w_void_414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 421: qk_w_void_421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 422: qk_w_void_422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 423: qk_w_void_423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 424: qk_w_void_424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 431: qk_w_void_431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 432: qk_w_void_432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 433: qk_w_void_433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 434: qk_w_void_434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 441: qk_w_void_441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 442: qk_w_void_442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 443: qk_w_void_443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 444: qk_w_void_444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1111: qk_w_void_1111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1112: qk_w_void_1112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1113: qk_w_void_1113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1114: qk_w_void_1114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1121: qk_w_void_1121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1122: qk_w_void_1122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1123: qk_w_void_1123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1124: qk_w_void_1124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1131: qk_w_void_1131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1132: qk_w_void_1132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1133: qk_w_void_1133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1134: qk_w_void_1134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1141: qk_w_void_1141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1142: qk_w_void_1142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1143: qk_w_void_1143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1144: qk_w_void_1144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1211: qk_w_void_1211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1212: qk_w_void_1212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1213: qk_w_void_1213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1214: qk_w_void_1214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1221: qk_w_void_1221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1222: qk_w_void_1222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1223: qk_w_void_1223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1224: qk_w_void_1224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1231: qk_w_void_1231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1232: qk_w_void_1232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1233: qk_w_void_1233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1234: qk_w_void_1234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1241: qk_w_void_1241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1242: qk_w_void_1242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1243: qk_w_void_1243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1244: qk_w_void_1244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1311: qk_w_void_1311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1312: qk_w_void_1312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1313: qk_w_void_1313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1314: qk_w_void_1314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1321: qk_w_void_1321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1322: qk_w_void_1322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1323: qk_w_void_1323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1324: qk_w_void_1324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1331: qk_w_void_1331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1332: qk_w_void_1332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1333: qk_w_void_1333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1334: qk_w_void_1334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1341: qk_w_void_1341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1342: qk_w_void_1342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1343: qk_w_void_1343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1344: qk_w_void_1344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1411: qk_w_void_1411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1412: qk_w_void_1412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1413: qk_w_void_1413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1414: qk_w_void_1414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1421: qk_w_void_1421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1422: qk_w_void_1422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1423: qk_w_void_1423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1424: qk_w_void_1424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1431: qk_w_void_1431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1432: qk_w_void_1432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1433: qk_w_void_1433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1434: qk_w_void_1434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1441: qk_w_void_1441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1442: qk_w_void_1442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1443: qk_w_void_1443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1444: qk_w_void_1444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2111: qk_w_void_2111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2112: qk_w_void_2112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2113: qk_w_void_2113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2114: qk_w_void_2114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2121: qk_w_void_2121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2122: qk_w_void_2122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2123: qk_w_void_2123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2124: qk_w_void_2124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2131: qk_w_void_2131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2132: qk_w_void_2132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2133: qk_w_void_2133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2134: qk_w_void_2134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2141: qk_w_void_2141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2142: qk_w_void_2142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2143: qk_w_void_2143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2144: qk_w_void_2144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2211: qk_w_void_2211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2212: qk_w_void_2212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2213: qk_w_void_2213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2214: qk_w_void_2214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2221: qk_w_void_2221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2222: qk_w_void_2222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2223: qk_w_void_2223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2224: qk_w_void_2224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2231: qk_w_void_2231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2232: qk_w_void_2232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2233: qk_w_void_2233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2234: qk_w_void_2234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2241: qk_w_void_2241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2242: qk_w_void_2242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2243: qk_w_void_2243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2244: qk_w_void_2244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2311: qk_w_void_2311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2312: qk_w_void_2312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2313: qk_w_void_2313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2314: qk_w_void_2314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2321: qk_w_void_2321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2322: qk_w_void_2322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2323: qk_w_void_2323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2324: qk_w_void_2324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2331: qk_w_void_2331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2332: qk_w_void_2332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2333: qk_w_void_2333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2334: qk_w_void_2334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2341: qk_w_void_2341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2342: qk_w_void_2342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2343: qk_w_void_2343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2344: qk_w_void_2344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2411: qk_w_void_2411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2412: qk_w_void_2412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2413: qk_w_void_2413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2414: qk_w_void_2414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2421: qk_w_void_2421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2422: qk_w_void_2422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2423: qk_w_void_2423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2424: qk_w_void_2424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2431: qk_w_void_2431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2432: qk_w_void_2432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2433: qk_w_void_2433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2434: qk_w_void_2434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2441: qk_w_void_2441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2442: qk_w_void_2442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2443: qk_w_void_2443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2444: qk_w_void_2444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3111: qk_w_void_3111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3112: qk_w_void_3112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3113: qk_w_void_3113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3114: qk_w_void_3114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3121: qk_w_void_3121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3122: qk_w_void_3122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3123: qk_w_void_3123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3124: qk_w_void_3124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3131: qk_w_void_3131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3132: qk_w_void_3132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3133: qk_w_void_3133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3134: qk_w_void_3134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3141: qk_w_void_3141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3142: qk_w_void_3142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3143: qk_w_void_3143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3144: qk_w_void_3144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3211: qk_w_void_3211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3212: qk_w_void_3212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3213: qk_w_void_3213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3214: qk_w_void_3214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3221: qk_w_void_3221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3222: qk_w_void_3222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3223: qk_w_void_3223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3224: qk_w_void_3224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3231: qk_w_void_3231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3232: qk_w_void_3232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3233: qk_w_void_3233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3234: qk_w_void_3234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3241: qk_w_void_3241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3242: qk_w_void_3242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3243: qk_w_void_3243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3244: qk_w_void_3244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3311: qk_w_void_3311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3312: qk_w_void_3312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3313: qk_w_void_3313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3314: qk_w_void_3314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3321: qk_w_void_3321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3322: qk_w_void_3322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3323: qk_w_void_3323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3324: qk_w_void_3324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3331: qk_w_void_3331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3332: qk_w_void_3332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3333: qk_w_void_3333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3334: qk_w_void_3334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3341: qk_w_void_3341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3342: qk_w_void_3342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3343: qk_w_void_3343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3344: qk_w_void_3344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3411: qk_w_void_3411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3412: qk_w_void_3412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3413: qk_w_void_3413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3414: qk_w_void_3414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3421: qk_w_void_3421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3422: qk_w_void_3422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3423: qk_w_void_3423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3424: qk_w_void_3424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3431: qk_w_void_3431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3432: qk_w_void_3432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3433: qk_w_void_3433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3434: qk_w_void_3434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3441: qk_w_void_3441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3442: qk_w_void_3442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3443: qk_w_void_3443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3444: qk_w_void_3444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4111: qk_w_void_4111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4112: qk_w_void_4112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4113: qk_w_void_4113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4114: qk_w_void_4114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4121: qk_w_void_4121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4122: qk_w_void_4122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4123: qk_w_void_4123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4124: qk_w_void_4124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4131: qk_w_void_4131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4132: qk_w_void_4132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4133: qk_w_void_4133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4134: qk_w_void_4134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4141: qk_w_void_4141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4142: qk_w_void_4142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4143: qk_w_void_4143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4144: qk_w_void_4144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4211: qk_w_void_4211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4212: qk_w_void_4212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4213: qk_w_void_4213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4214: qk_w_void_4214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4221: qk_w_void_4221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4222: qk_w_void_4222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4223: qk_w_void_4223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4224: qk_w_void_4224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4231: qk_w_void_4231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4232: qk_w_void_4232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4233: qk_w_void_4233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4234: qk_w_void_4234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4241: qk_w_void_4241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4242: qk_w_void_4242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4243: qk_w_void_4243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4244: qk_w_void_4244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4311: qk_w_void_4311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4312: qk_w_void_4312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4313: qk_w_void_4313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4314: qk_w_void_4314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4321: qk_w_void_4321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4322: qk_w_void_4322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4323: qk_w_void_4323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4324: qk_w_void_4324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4331: qk_w_void_4331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4332: qk_w_void_4332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4333: qk_w_void_4333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4334: qk_w_void_4334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4341: qk_w_void_4341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4342: qk_w_void_4342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4343: qk_w_void_4343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4344: qk_w_void_4344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4411: qk_w_void_4411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4412: qk_w_void_4412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4413: qk_w_void_4413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4414: qk_w_void_4414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4421: qk_w_void_4421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4422: qk_w_void_4422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4423: qk_w_void_4423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4424: qk_w_void_4424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4431: qk_w_void_4431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4432: qk_w_void_4432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4433: qk_w_void_4433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4434: qk_w_void_4434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4441: qk_w_void_4441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4442: qk_w_void_4442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4443: qk_w_void_4443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4444: qk_w_void_4444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000000: *ret_i = qk_w_i32_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001: *ret_i = qk_w_i32_1(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002: *ret_i = qk_w_i32_2(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003: *ret_i = qk_w_i32_3(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004: *ret_i = qk_w_i32_4(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000011: *ret_i = qk_w_i32_11(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000012: *ret_i = qk_w_i32_12(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000013: *ret_i = qk_w_i32_13(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000014: *ret_i = qk_w_i32_14(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000021: *ret_i = qk_w_i32_21(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000022: *ret_i = qk_w_i32_22(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000023: *ret_i = qk_w_i32_23(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000024: *ret_i = qk_w_i32_24(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000031: *ret_i = qk_w_i32_31(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000032: *ret_i = qk_w_i32_32(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000033: *ret_i = qk_w_i32_33(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000034: *ret_i = qk_w_i32_34(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000041: *ret_i = qk_w_i32_41(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000042: *ret_i = qk_w_i32_42(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000043: *ret_i = qk_w_i32_43(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 10000044: *ret_i = qk_w_i32_44(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000111: *ret_i = qk_w_i32_111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000112: *ret_i = qk_w_i32_112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000113: *ret_i = qk_w_i32_113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000114: *ret_i = qk_w_i32_114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000121: *ret_i = qk_w_i32_121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000122: *ret_i = qk_w_i32_122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000123: *ret_i = qk_w_i32_123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000124: *ret_i = qk_w_i32_124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000131: *ret_i = qk_w_i32_131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000132: *ret_i = qk_w_i32_132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000133: *ret_i = qk_w_i32_133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000134: *ret_i = qk_w_i32_134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000141: *ret_i = qk_w_i32_141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000142: *ret_i = qk_w_i32_142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000143: *ret_i = qk_w_i32_143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000144: *ret_i = qk_w_i32_144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000211: *ret_i = qk_w_i32_211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000212: *ret_i = qk_w_i32_212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000213: *ret_i = qk_w_i32_213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000214: *ret_i = qk_w_i32_214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000221: *ret_i = qk_w_i32_221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000222: *ret_i = qk_w_i32_222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000223: *ret_i = qk_w_i32_223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000224: *ret_i = qk_w_i32_224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000231: *ret_i = qk_w_i32_231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000232: *ret_i = qk_w_i32_232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000233: *ret_i = qk_w_i32_233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000234: *ret_i = qk_w_i32_234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000241: *ret_i = qk_w_i32_241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000242: *ret_i = qk_w_i32_242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000243: *ret_i = qk_w_i32_243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000244: *ret_i = qk_w_i32_244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000311: *ret_i = qk_w_i32_311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000312: *ret_i = qk_w_i32_312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000313: *ret_i = qk_w_i32_313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000314: *ret_i = qk_w_i32_314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000321: *ret_i = qk_w_i32_321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000322: *ret_i = qk_w_i32_322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000323: *ret_i = qk_w_i32_323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000324: *ret_i = qk_w_i32_324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000331: *ret_i = qk_w_i32_331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000332: *ret_i = qk_w_i32_332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000333: *ret_i = qk_w_i32_333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000334: *ret_i = qk_w_i32_334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000341: *ret_i = qk_w_i32_341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000342: *ret_i = qk_w_i32_342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000343: *ret_i = qk_w_i32_343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000344: *ret_i = qk_w_i32_344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000411: *ret_i = qk_w_i32_411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000412: *ret_i = qk_w_i32_412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000413: *ret_i = qk_w_i32_413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000414: *ret_i = qk_w_i32_414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000421: *ret_i = qk_w_i32_421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000422: *ret_i = qk_w_i32_422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000423: *ret_i = qk_w_i32_423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000424: *ret_i = qk_w_i32_424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000431: *ret_i = qk_w_i32_431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000432: *ret_i = qk_w_i32_432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000433: *ret_i = qk_w_i32_433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000434: *ret_i = qk_w_i32_434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000441: *ret_i = qk_w_i32_441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000442: *ret_i = qk_w_i32_442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000443: *ret_i = qk_w_i32_443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000444: *ret_i = qk_w_i32_444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001111: *ret_i = qk_w_i32_1111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001112: *ret_i = qk_w_i32_1112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001113: *ret_i = qk_w_i32_1113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001114: *ret_i = qk_w_i32_1114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001121: *ret_i = qk_w_i32_1121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001122: *ret_i = qk_w_i32_1122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001123: *ret_i = qk_w_i32_1123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001124: *ret_i = qk_w_i32_1124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001131: *ret_i = qk_w_i32_1131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001132: *ret_i = qk_w_i32_1132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001133: *ret_i = qk_w_i32_1133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001134: *ret_i = qk_w_i32_1134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001141: *ret_i = qk_w_i32_1141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001142: *ret_i = qk_w_i32_1142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001143: *ret_i = qk_w_i32_1143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001144: *ret_i = qk_w_i32_1144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001211: *ret_i = qk_w_i32_1211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001212: *ret_i = qk_w_i32_1212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001213: *ret_i = qk_w_i32_1213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001214: *ret_i = qk_w_i32_1214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001221: *ret_i = qk_w_i32_1221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001222: *ret_i = qk_w_i32_1222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001223: *ret_i = qk_w_i32_1223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001224: *ret_i = qk_w_i32_1224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001231: *ret_i = qk_w_i32_1231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001232: *ret_i = qk_w_i32_1232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001233: *ret_i = qk_w_i32_1233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001234: *ret_i = qk_w_i32_1234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001241: *ret_i = qk_w_i32_1241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001242: *ret_i = qk_w_i32_1242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001243: *ret_i = qk_w_i32_1243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001244: *ret_i = qk_w_i32_1244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001311: *ret_i = qk_w_i32_1311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001312: *ret_i = qk_w_i32_1312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001313: *ret_i = qk_w_i32_1313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001314: *ret_i = qk_w_i32_1314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001321: *ret_i = qk_w_i32_1321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001322: *ret_i = qk_w_i32_1322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001323: *ret_i = qk_w_i32_1323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001324: *ret_i = qk_w_i32_1324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001331: *ret_i = qk_w_i32_1331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001332: *ret_i = qk_w_i32_1332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001333: *ret_i = qk_w_i32_1333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001334: *ret_i = qk_w_i32_1334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001341: *ret_i = qk_w_i32_1341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001342: *ret_i = qk_w_i32_1342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001343: *ret_i = qk_w_i32_1343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001344: *ret_i = qk_w_i32_1344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001411: *ret_i = qk_w_i32_1411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001412: *ret_i = qk_w_i32_1412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001413: *ret_i = qk_w_i32_1413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001414: *ret_i = qk_w_i32_1414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001421: *ret_i = qk_w_i32_1421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001422: *ret_i = qk_w_i32_1422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001423: *ret_i = qk_w_i32_1423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001424: *ret_i = qk_w_i32_1424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001431: *ret_i = qk_w_i32_1431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001432: *ret_i = qk_w_i32_1432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001433: *ret_i = qk_w_i32_1433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001434: *ret_i = qk_w_i32_1434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001441: *ret_i = qk_w_i32_1441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001442: *ret_i = qk_w_i32_1442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001443: *ret_i = qk_w_i32_1443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000001444: *ret_i = qk_w_i32_1444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002111: *ret_i = qk_w_i32_2111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002112: *ret_i = qk_w_i32_2112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002113: *ret_i = qk_w_i32_2113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002114: *ret_i = qk_w_i32_2114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002121: *ret_i = qk_w_i32_2121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002122: *ret_i = qk_w_i32_2122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002123: *ret_i = qk_w_i32_2123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002124: *ret_i = qk_w_i32_2124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002131: *ret_i = qk_w_i32_2131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002132: *ret_i = qk_w_i32_2132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002133: *ret_i = qk_w_i32_2133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002134: *ret_i = qk_w_i32_2134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002141: *ret_i = qk_w_i32_2141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002142: *ret_i = qk_w_i32_2142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002143: *ret_i = qk_w_i32_2143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002144: *ret_i = qk_w_i32_2144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002211: *ret_i = qk_w_i32_2211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002212: *ret_i = qk_w_i32_2212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002213: *ret_i = qk_w_i32_2213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002214: *ret_i = qk_w_i32_2214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002221: *ret_i = qk_w_i32_2221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002222: *ret_i = qk_w_i32_2222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002223: *ret_i = qk_w_i32_2223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002224: *ret_i = qk_w_i32_2224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002231: *ret_i = qk_w_i32_2231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002232: *ret_i = qk_w_i32_2232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002233: *ret_i = qk_w_i32_2233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002234: *ret_i = qk_w_i32_2234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002241: *ret_i = qk_w_i32_2241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002242: *ret_i = qk_w_i32_2242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002243: *ret_i = qk_w_i32_2243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002244: *ret_i = qk_w_i32_2244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002311: *ret_i = qk_w_i32_2311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002312: *ret_i = qk_w_i32_2312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002313: *ret_i = qk_w_i32_2313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002314: *ret_i = qk_w_i32_2314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002321: *ret_i = qk_w_i32_2321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002322: *ret_i = qk_w_i32_2322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002323: *ret_i = qk_w_i32_2323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002324: *ret_i = qk_w_i32_2324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002331: *ret_i = qk_w_i32_2331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002332: *ret_i = qk_w_i32_2332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002333: *ret_i = qk_w_i32_2333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002334: *ret_i = qk_w_i32_2334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002341: *ret_i = qk_w_i32_2341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002342: *ret_i = qk_w_i32_2342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002343: *ret_i = qk_w_i32_2343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002344: *ret_i = qk_w_i32_2344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002411: *ret_i = qk_w_i32_2411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002412: *ret_i = qk_w_i32_2412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002413: *ret_i = qk_w_i32_2413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002414: *ret_i = qk_w_i32_2414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002421: *ret_i = qk_w_i32_2421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002422: *ret_i = qk_w_i32_2422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002423: *ret_i = qk_w_i32_2423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002424: *ret_i = qk_w_i32_2424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002431: *ret_i = qk_w_i32_2431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002432: *ret_i = qk_w_i32_2432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002433: *ret_i = qk_w_i32_2433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002434: *ret_i = qk_w_i32_2434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002441: *ret_i = qk_w_i32_2441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002442: *ret_i = qk_w_i32_2442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002443: *ret_i = qk_w_i32_2443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000002444: *ret_i = qk_w_i32_2444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003111: *ret_i = qk_w_i32_3111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003112: *ret_i = qk_w_i32_3112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003113: *ret_i = qk_w_i32_3113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003114: *ret_i = qk_w_i32_3114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003121: *ret_i = qk_w_i32_3121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003122: *ret_i = qk_w_i32_3122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003123: *ret_i = qk_w_i32_3123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003124: *ret_i = qk_w_i32_3124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003131: *ret_i = qk_w_i32_3131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003132: *ret_i = qk_w_i32_3132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003133: *ret_i = qk_w_i32_3133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003134: *ret_i = qk_w_i32_3134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003141: *ret_i = qk_w_i32_3141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003142: *ret_i = qk_w_i32_3142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003143: *ret_i = qk_w_i32_3143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003144: *ret_i = qk_w_i32_3144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003211: *ret_i = qk_w_i32_3211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003212: *ret_i = qk_w_i32_3212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003213: *ret_i = qk_w_i32_3213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003214: *ret_i = qk_w_i32_3214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003221: *ret_i = qk_w_i32_3221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003222: *ret_i = qk_w_i32_3222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003223: *ret_i = qk_w_i32_3223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003224: *ret_i = qk_w_i32_3224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003231: *ret_i = qk_w_i32_3231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003232: *ret_i = qk_w_i32_3232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003233: *ret_i = qk_w_i32_3233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003234: *ret_i = qk_w_i32_3234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003241: *ret_i = qk_w_i32_3241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003242: *ret_i = qk_w_i32_3242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003243: *ret_i = qk_w_i32_3243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003244: *ret_i = qk_w_i32_3244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003311: *ret_i = qk_w_i32_3311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003312: *ret_i = qk_w_i32_3312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003313: *ret_i = qk_w_i32_3313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003314: *ret_i = qk_w_i32_3314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003321: *ret_i = qk_w_i32_3321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003322: *ret_i = qk_w_i32_3322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003323: *ret_i = qk_w_i32_3323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003324: *ret_i = qk_w_i32_3324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003331: *ret_i = qk_w_i32_3331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003332: *ret_i = qk_w_i32_3332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003333: *ret_i = qk_w_i32_3333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003334: *ret_i = qk_w_i32_3334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003341: *ret_i = qk_w_i32_3341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003342: *ret_i = qk_w_i32_3342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003343: *ret_i = qk_w_i32_3343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003344: *ret_i = qk_w_i32_3344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003411: *ret_i = qk_w_i32_3411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003412: *ret_i = qk_w_i32_3412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003413: *ret_i = qk_w_i32_3413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003414: *ret_i = qk_w_i32_3414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003421: *ret_i = qk_w_i32_3421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003422: *ret_i = qk_w_i32_3422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003423: *ret_i = qk_w_i32_3423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003424: *ret_i = qk_w_i32_3424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003431: *ret_i = qk_w_i32_3431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003432: *ret_i = qk_w_i32_3432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003433: *ret_i = qk_w_i32_3433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003434: *ret_i = qk_w_i32_3434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003441: *ret_i = qk_w_i32_3441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003442: *ret_i = qk_w_i32_3442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003443: *ret_i = qk_w_i32_3443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000003444: *ret_i = qk_w_i32_3444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004111: *ret_i = qk_w_i32_4111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004112: *ret_i = qk_w_i32_4112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004113: *ret_i = qk_w_i32_4113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004114: *ret_i = qk_w_i32_4114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004121: *ret_i = qk_w_i32_4121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004122: *ret_i = qk_w_i32_4122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004123: *ret_i = qk_w_i32_4123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004124: *ret_i = qk_w_i32_4124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004131: *ret_i = qk_w_i32_4131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004132: *ret_i = qk_w_i32_4132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004133: *ret_i = qk_w_i32_4133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004134: *ret_i = qk_w_i32_4134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004141: *ret_i = qk_w_i32_4141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004142: *ret_i = qk_w_i32_4142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004143: *ret_i = qk_w_i32_4143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004144: *ret_i = qk_w_i32_4144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004211: *ret_i = qk_w_i32_4211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004212: *ret_i = qk_w_i32_4212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004213: *ret_i = qk_w_i32_4213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004214: *ret_i = qk_w_i32_4214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004221: *ret_i = qk_w_i32_4221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004222: *ret_i = qk_w_i32_4222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004223: *ret_i = qk_w_i32_4223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004224: *ret_i = qk_w_i32_4224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004231: *ret_i = qk_w_i32_4231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004232: *ret_i = qk_w_i32_4232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004233: *ret_i = qk_w_i32_4233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004234: *ret_i = qk_w_i32_4234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004241: *ret_i = qk_w_i32_4241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004242: *ret_i = qk_w_i32_4242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004243: *ret_i = qk_w_i32_4243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004244: *ret_i = qk_w_i32_4244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004311: *ret_i = qk_w_i32_4311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004312: *ret_i = qk_w_i32_4312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004313: *ret_i = qk_w_i32_4313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004314: *ret_i = qk_w_i32_4314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004321: *ret_i = qk_w_i32_4321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004322: *ret_i = qk_w_i32_4322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004323: *ret_i = qk_w_i32_4323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004324: *ret_i = qk_w_i32_4324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004331: *ret_i = qk_w_i32_4331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004332: *ret_i = qk_w_i32_4332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004333: *ret_i = qk_w_i32_4333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004334: *ret_i = qk_w_i32_4334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004341: *ret_i = qk_w_i32_4341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004342: *ret_i = qk_w_i32_4342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004343: *ret_i = qk_w_i32_4343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004344: *ret_i = qk_w_i32_4344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004411: *ret_i = qk_w_i32_4411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004412: *ret_i = qk_w_i32_4412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004413: *ret_i = qk_w_i32_4413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004414: *ret_i = qk_w_i32_4414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004421: *ret_i = qk_w_i32_4421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004422: *ret_i = qk_w_i32_4422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004423: *ret_i = qk_w_i32_4423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004424: *ret_i = qk_w_i32_4424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004431: *ret_i = qk_w_i32_4431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004432: *ret_i = qk_w_i32_4432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004433: *ret_i = qk_w_i32_4433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004434: *ret_i = qk_w_i32_4434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004441: *ret_i = qk_w_i32_4441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004442: *ret_i = qk_w_i32_4442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004443: *ret_i = qk_w_i32_4443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 1000004444: *ret_i = qk_w_i32_4444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000000: *ret_f = qk_w_f32_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001: *ret_f = qk_w_f32_1(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002: *ret_f = qk_w_f32_2(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003: *ret_f = qk_w_f32_3(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004: *ret_f = qk_w_f32_4(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000011: *ret_f = qk_w_f32_11(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000012: *ret_f = qk_w_f32_12(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000013: *ret_f = qk_w_f32_13(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000014: *ret_f = qk_w_f32_14(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000021: *ret_f = qk_w_f32_21(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000022: *ret_f = qk_w_f32_22(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000023: *ret_f = qk_w_f32_23(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000024: *ret_f = qk_w_f32_24(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000031: *ret_f = qk_w_f32_31(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000032: *ret_f = qk_w_f32_32(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000033: *ret_f = qk_w_f32_33(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000034: *ret_f = qk_w_f32_34(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000041: *ret_f = qk_w_f32_41(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000042: *ret_f = qk_w_f32_42(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000043: *ret_f = qk_w_f32_43(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 20000044: *ret_f = qk_w_f32_44(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000111: *ret_f = qk_w_f32_111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000112: *ret_f = qk_w_f32_112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000113: *ret_f = qk_w_f32_113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000114: *ret_f = qk_w_f32_114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000121: *ret_f = qk_w_f32_121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000122: *ret_f = qk_w_f32_122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000123: *ret_f = qk_w_f32_123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000124: *ret_f = qk_w_f32_124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000131: *ret_f = qk_w_f32_131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000132: *ret_f = qk_w_f32_132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000133: *ret_f = qk_w_f32_133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000134: *ret_f = qk_w_f32_134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000141: *ret_f = qk_w_f32_141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000142: *ret_f = qk_w_f32_142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000143: *ret_f = qk_w_f32_143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000144: *ret_f = qk_w_f32_144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000211: *ret_f = qk_w_f32_211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000212: *ret_f = qk_w_f32_212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000213: *ret_f = qk_w_f32_213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000214: *ret_f = qk_w_f32_214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000221: *ret_f = qk_w_f32_221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000222: *ret_f = qk_w_f32_222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000223: *ret_f = qk_w_f32_223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000224: *ret_f = qk_w_f32_224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000231: *ret_f = qk_w_f32_231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000232: *ret_f = qk_w_f32_232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000233: *ret_f = qk_w_f32_233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000234: *ret_f = qk_w_f32_234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000241: *ret_f = qk_w_f32_241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000242: *ret_f = qk_w_f32_242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000243: *ret_f = qk_w_f32_243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000244: *ret_f = qk_w_f32_244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000311: *ret_f = qk_w_f32_311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000312: *ret_f = qk_w_f32_312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000313: *ret_f = qk_w_f32_313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000314: *ret_f = qk_w_f32_314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000321: *ret_f = qk_w_f32_321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000322: *ret_f = qk_w_f32_322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000323: *ret_f = qk_w_f32_323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000324: *ret_f = qk_w_f32_324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000331: *ret_f = qk_w_f32_331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000332: *ret_f = qk_w_f32_332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000333: *ret_f = qk_w_f32_333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000334: *ret_f = qk_w_f32_334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000341: *ret_f = qk_w_f32_341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000342: *ret_f = qk_w_f32_342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000343: *ret_f = qk_w_f32_343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000344: *ret_f = qk_w_f32_344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000411: *ret_f = qk_w_f32_411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000412: *ret_f = qk_w_f32_412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000413: *ret_f = qk_w_f32_413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000414: *ret_f = qk_w_f32_414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000421: *ret_f = qk_w_f32_421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000422: *ret_f = qk_w_f32_422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000423: *ret_f = qk_w_f32_423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000424: *ret_f = qk_w_f32_424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000431: *ret_f = qk_w_f32_431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000432: *ret_f = qk_w_f32_432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000433: *ret_f = qk_w_f32_433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000434: *ret_f = qk_w_f32_434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000441: *ret_f = qk_w_f32_441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000442: *ret_f = qk_w_f32_442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000443: *ret_f = qk_w_f32_443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000444: *ret_f = qk_w_f32_444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001111: *ret_f = qk_w_f32_1111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001112: *ret_f = qk_w_f32_1112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001113: *ret_f = qk_w_f32_1113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001114: *ret_f = qk_w_f32_1114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001121: *ret_f = qk_w_f32_1121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001122: *ret_f = qk_w_f32_1122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001123: *ret_f = qk_w_f32_1123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001124: *ret_f = qk_w_f32_1124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001131: *ret_f = qk_w_f32_1131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001132: *ret_f = qk_w_f32_1132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001133: *ret_f = qk_w_f32_1133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001134: *ret_f = qk_w_f32_1134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001141: *ret_f = qk_w_f32_1141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001142: *ret_f = qk_w_f32_1142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001143: *ret_f = qk_w_f32_1143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001144: *ret_f = qk_w_f32_1144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001211: *ret_f = qk_w_f32_1211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001212: *ret_f = qk_w_f32_1212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001213: *ret_f = qk_w_f32_1213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001214: *ret_f = qk_w_f32_1214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001221: *ret_f = qk_w_f32_1221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001222: *ret_f = qk_w_f32_1222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001223: *ret_f = qk_w_f32_1223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001224: *ret_f = qk_w_f32_1224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001231: *ret_f = qk_w_f32_1231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001232: *ret_f = qk_w_f32_1232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001233: *ret_f = qk_w_f32_1233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001234: *ret_f = qk_w_f32_1234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001241: *ret_f = qk_w_f32_1241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001242: *ret_f = qk_w_f32_1242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001243: *ret_f = qk_w_f32_1243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001244: *ret_f = qk_w_f32_1244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001311: *ret_f = qk_w_f32_1311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001312: *ret_f = qk_w_f32_1312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001313: *ret_f = qk_w_f32_1313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001314: *ret_f = qk_w_f32_1314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001321: *ret_f = qk_w_f32_1321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001322: *ret_f = qk_w_f32_1322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001323: *ret_f = qk_w_f32_1323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001324: *ret_f = qk_w_f32_1324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001331: *ret_f = qk_w_f32_1331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001332: *ret_f = qk_w_f32_1332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001333: *ret_f = qk_w_f32_1333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001334: *ret_f = qk_w_f32_1334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001341: *ret_f = qk_w_f32_1341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001342: *ret_f = qk_w_f32_1342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001343: *ret_f = qk_w_f32_1343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001344: *ret_f = qk_w_f32_1344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001411: *ret_f = qk_w_f32_1411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001412: *ret_f = qk_w_f32_1412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001413: *ret_f = qk_w_f32_1413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001414: *ret_f = qk_w_f32_1414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001421: *ret_f = qk_w_f32_1421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001422: *ret_f = qk_w_f32_1422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001423: *ret_f = qk_w_f32_1423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001424: *ret_f = qk_w_f32_1424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001431: *ret_f = qk_w_f32_1431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001432: *ret_f = qk_w_f32_1432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001433: *ret_f = qk_w_f32_1433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001434: *ret_f = qk_w_f32_1434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001441: *ret_f = qk_w_f32_1441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001442: *ret_f = qk_w_f32_1442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001443: *ret_f = qk_w_f32_1443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000001444: *ret_f = qk_w_f32_1444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002111: *ret_f = qk_w_f32_2111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002112: *ret_f = qk_w_f32_2112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002113: *ret_f = qk_w_f32_2113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002114: *ret_f = qk_w_f32_2114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002121: *ret_f = qk_w_f32_2121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002122: *ret_f = qk_w_f32_2122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002123: *ret_f = qk_w_f32_2123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002124: *ret_f = qk_w_f32_2124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002131: *ret_f = qk_w_f32_2131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002132: *ret_f = qk_w_f32_2132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002133: *ret_f = qk_w_f32_2133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002134: *ret_f = qk_w_f32_2134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002141: *ret_f = qk_w_f32_2141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002142: *ret_f = qk_w_f32_2142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002143: *ret_f = qk_w_f32_2143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002144: *ret_f = qk_w_f32_2144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002211: *ret_f = qk_w_f32_2211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002212: *ret_f = qk_w_f32_2212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002213: *ret_f = qk_w_f32_2213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002214: *ret_f = qk_w_f32_2214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002221: *ret_f = qk_w_f32_2221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002222: *ret_f = qk_w_f32_2222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002223: *ret_f = qk_w_f32_2223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002224: *ret_f = qk_w_f32_2224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002231: *ret_f = qk_w_f32_2231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002232: *ret_f = qk_w_f32_2232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002233: *ret_f = qk_w_f32_2233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002234: *ret_f = qk_w_f32_2234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002241: *ret_f = qk_w_f32_2241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002242: *ret_f = qk_w_f32_2242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002243: *ret_f = qk_w_f32_2243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002244: *ret_f = qk_w_f32_2244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002311: *ret_f = qk_w_f32_2311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002312: *ret_f = qk_w_f32_2312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002313: *ret_f = qk_w_f32_2313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002314: *ret_f = qk_w_f32_2314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002321: *ret_f = qk_w_f32_2321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002322: *ret_f = qk_w_f32_2322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002323: *ret_f = qk_w_f32_2323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002324: *ret_f = qk_w_f32_2324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002331: *ret_f = qk_w_f32_2331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002332: *ret_f = qk_w_f32_2332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002333: *ret_f = qk_w_f32_2333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002334: *ret_f = qk_w_f32_2334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002341: *ret_f = qk_w_f32_2341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002342: *ret_f = qk_w_f32_2342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002343: *ret_f = qk_w_f32_2343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002344: *ret_f = qk_w_f32_2344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002411: *ret_f = qk_w_f32_2411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002412: *ret_f = qk_w_f32_2412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002413: *ret_f = qk_w_f32_2413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002414: *ret_f = qk_w_f32_2414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002421: *ret_f = qk_w_f32_2421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002422: *ret_f = qk_w_f32_2422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002423: *ret_f = qk_w_f32_2423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002424: *ret_f = qk_w_f32_2424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002431: *ret_f = qk_w_f32_2431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002432: *ret_f = qk_w_f32_2432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002433: *ret_f = qk_w_f32_2433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002434: *ret_f = qk_w_f32_2434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002441: *ret_f = qk_w_f32_2441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002442: *ret_f = qk_w_f32_2442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002443: *ret_f = qk_w_f32_2443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000002444: *ret_f = qk_w_f32_2444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003111: *ret_f = qk_w_f32_3111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003112: *ret_f = qk_w_f32_3112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003113: *ret_f = qk_w_f32_3113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003114: *ret_f = qk_w_f32_3114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003121: *ret_f = qk_w_f32_3121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003122: *ret_f = qk_w_f32_3122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003123: *ret_f = qk_w_f32_3123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003124: *ret_f = qk_w_f32_3124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003131: *ret_f = qk_w_f32_3131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003132: *ret_f = qk_w_f32_3132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003133: *ret_f = qk_w_f32_3133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003134: *ret_f = qk_w_f32_3134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003141: *ret_f = qk_w_f32_3141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003142: *ret_f = qk_w_f32_3142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003143: *ret_f = qk_w_f32_3143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003144: *ret_f = qk_w_f32_3144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003211: *ret_f = qk_w_f32_3211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003212: *ret_f = qk_w_f32_3212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003213: *ret_f = qk_w_f32_3213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003214: *ret_f = qk_w_f32_3214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003221: *ret_f = qk_w_f32_3221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003222: *ret_f = qk_w_f32_3222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003223: *ret_f = qk_w_f32_3223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003224: *ret_f = qk_w_f32_3224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003231: *ret_f = qk_w_f32_3231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003232: *ret_f = qk_w_f32_3232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003233: *ret_f = qk_w_f32_3233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003234: *ret_f = qk_w_f32_3234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003241: *ret_f = qk_w_f32_3241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003242: *ret_f = qk_w_f32_3242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003243: *ret_f = qk_w_f32_3243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003244: *ret_f = qk_w_f32_3244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003311: *ret_f = qk_w_f32_3311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003312: *ret_f = qk_w_f32_3312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003313: *ret_f = qk_w_f32_3313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003314: *ret_f = qk_w_f32_3314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003321: *ret_f = qk_w_f32_3321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003322: *ret_f = qk_w_f32_3322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003323: *ret_f = qk_w_f32_3323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003324: *ret_f = qk_w_f32_3324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003331: *ret_f = qk_w_f32_3331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003332: *ret_f = qk_w_f32_3332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003333: *ret_f = qk_w_f32_3333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003334: *ret_f = qk_w_f32_3334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003341: *ret_f = qk_w_f32_3341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003342: *ret_f = qk_w_f32_3342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003343: *ret_f = qk_w_f32_3343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003344: *ret_f = qk_w_f32_3344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003411: *ret_f = qk_w_f32_3411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003412: *ret_f = qk_w_f32_3412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003413: *ret_f = qk_w_f32_3413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003414: *ret_f = qk_w_f32_3414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003421: *ret_f = qk_w_f32_3421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003422: *ret_f = qk_w_f32_3422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003423: *ret_f = qk_w_f32_3423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003424: *ret_f = qk_w_f32_3424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003431: *ret_f = qk_w_f32_3431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003432: *ret_f = qk_w_f32_3432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003433: *ret_f = qk_w_f32_3433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003434: *ret_f = qk_w_f32_3434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003441: *ret_f = qk_w_f32_3441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003442: *ret_f = qk_w_f32_3442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003443: *ret_f = qk_w_f32_3443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000003444: *ret_f = qk_w_f32_3444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004111: *ret_f = qk_w_f32_4111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004112: *ret_f = qk_w_f32_4112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004113: *ret_f = qk_w_f32_4113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004114: *ret_f = qk_w_f32_4114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004121: *ret_f = qk_w_f32_4121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004122: *ret_f = qk_w_f32_4122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004123: *ret_f = qk_w_f32_4123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004124: *ret_f = qk_w_f32_4124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004131: *ret_f = qk_w_f32_4131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004132: *ret_f = qk_w_f32_4132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004133: *ret_f = qk_w_f32_4133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004134: *ret_f = qk_w_f32_4134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004141: *ret_f = qk_w_f32_4141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004142: *ret_f = qk_w_f32_4142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004143: *ret_f = qk_w_f32_4143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004144: *ret_f = qk_w_f32_4144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004211: *ret_f = qk_w_f32_4211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004212: *ret_f = qk_w_f32_4212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004213: *ret_f = qk_w_f32_4213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004214: *ret_f = qk_w_f32_4214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004221: *ret_f = qk_w_f32_4221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004222: *ret_f = qk_w_f32_4222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004223: *ret_f = qk_w_f32_4223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004224: *ret_f = qk_w_f32_4224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004231: *ret_f = qk_w_f32_4231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004232: *ret_f = qk_w_f32_4232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004233: *ret_f = qk_w_f32_4233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004234: *ret_f = qk_w_f32_4234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004241: *ret_f = qk_w_f32_4241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004242: *ret_f = qk_w_f32_4242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004243: *ret_f = qk_w_f32_4243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004244: *ret_f = qk_w_f32_4244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004311: *ret_f = qk_w_f32_4311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004312: *ret_f = qk_w_f32_4312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004313: *ret_f = qk_w_f32_4313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004314: *ret_f = qk_w_f32_4314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004321: *ret_f = qk_w_f32_4321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004322: *ret_f = qk_w_f32_4322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004323: *ret_f = qk_w_f32_4323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004324: *ret_f = qk_w_f32_4324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004331: *ret_f = qk_w_f32_4331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004332: *ret_f = qk_w_f32_4332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004333: *ret_f = qk_w_f32_4333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004334: *ret_f = qk_w_f32_4334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004341: *ret_f = qk_w_f32_4341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004342: *ret_f = qk_w_f32_4342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004343: *ret_f = qk_w_f32_4343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004344: *ret_f = qk_w_f32_4344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004411: *ret_f = qk_w_f32_4411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004412: *ret_f = qk_w_f32_4412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004413: *ret_f = qk_w_f32_4413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004414: *ret_f = qk_w_f32_4414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004421: *ret_f = qk_w_f32_4421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004422: *ret_f = qk_w_f32_4422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004423: *ret_f = qk_w_f32_4423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004424: *ret_f = qk_w_f32_4424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004431: *ret_f = qk_w_f32_4431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004432: *ret_f = qk_w_f32_4432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004433: *ret_f = qk_w_f32_4433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004434: *ret_f = qk_w_f32_4434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004441: *ret_f = qk_w_f32_4441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004442: *ret_f = qk_w_f32_4442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004443: *ret_f = qk_w_f32_4443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 2000004444: *ret_f = qk_w_f32_4444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000000: *ret_f = qk_w_f64_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001: *ret_f = qk_w_f64_1(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002: *ret_f = qk_w_f64_2(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003: *ret_f = qk_w_f64_3(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004: *ret_f = qk_w_f64_4(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000011: *ret_f = qk_w_f64_11(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000012: *ret_f = qk_w_f64_12(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000013: *ret_f = qk_w_f64_13(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000014: *ret_f = qk_w_f64_14(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000021: *ret_f = qk_w_f64_21(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000022: *ret_f = qk_w_f64_22(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000023: *ret_f = qk_w_f64_23(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000024: *ret_f = qk_w_f64_24(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000031: *ret_f = qk_w_f64_31(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000032: *ret_f = qk_w_f64_32(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000033: *ret_f = qk_w_f64_33(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000034: *ret_f = qk_w_f64_34(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000041: *ret_f = qk_w_f64_41(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000042: *ret_f = qk_w_f64_42(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000043: *ret_f = qk_w_f64_43(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 30000044: *ret_f = qk_w_f64_44(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000111: *ret_f = qk_w_f64_111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000112: *ret_f = qk_w_f64_112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000113: *ret_f = qk_w_f64_113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000114: *ret_f = qk_w_f64_114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000121: *ret_f = qk_w_f64_121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000122: *ret_f = qk_w_f64_122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000123: *ret_f = qk_w_f64_123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000124: *ret_f = qk_w_f64_124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000131: *ret_f = qk_w_f64_131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000132: *ret_f = qk_w_f64_132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000133: *ret_f = qk_w_f64_133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000134: *ret_f = qk_w_f64_134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000141: *ret_f = qk_w_f64_141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000142: *ret_f = qk_w_f64_142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000143: *ret_f = qk_w_f64_143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000144: *ret_f = qk_w_f64_144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000211: *ret_f = qk_w_f64_211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000212: *ret_f = qk_w_f64_212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000213: *ret_f = qk_w_f64_213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000214: *ret_f = qk_w_f64_214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000221: *ret_f = qk_w_f64_221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000222: *ret_f = qk_w_f64_222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000223: *ret_f = qk_w_f64_223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000224: *ret_f = qk_w_f64_224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000231: *ret_f = qk_w_f64_231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000232: *ret_f = qk_w_f64_232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000233: *ret_f = qk_w_f64_233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000234: *ret_f = qk_w_f64_234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000241: *ret_f = qk_w_f64_241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000242: *ret_f = qk_w_f64_242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000243: *ret_f = qk_w_f64_243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000244: *ret_f = qk_w_f64_244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000311: *ret_f = qk_w_f64_311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000312: *ret_f = qk_w_f64_312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000313: *ret_f = qk_w_f64_313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000314: *ret_f = qk_w_f64_314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000321: *ret_f = qk_w_f64_321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000322: *ret_f = qk_w_f64_322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000323: *ret_f = qk_w_f64_323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000324: *ret_f = qk_w_f64_324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000331: *ret_f = qk_w_f64_331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000332: *ret_f = qk_w_f64_332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000333: *ret_f = qk_w_f64_333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000334: *ret_f = qk_w_f64_334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000341: *ret_f = qk_w_f64_341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000342: *ret_f = qk_w_f64_342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000343: *ret_f = qk_w_f64_343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000344: *ret_f = qk_w_f64_344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000411: *ret_f = qk_w_f64_411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000412: *ret_f = qk_w_f64_412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000413: *ret_f = qk_w_f64_413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000414: *ret_f = qk_w_f64_414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000421: *ret_f = qk_w_f64_421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000422: *ret_f = qk_w_f64_422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000423: *ret_f = qk_w_f64_423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000424: *ret_f = qk_w_f64_424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000431: *ret_f = qk_w_f64_431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000432: *ret_f = qk_w_f64_432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000433: *ret_f = qk_w_f64_433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000434: *ret_f = qk_w_f64_434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000441: *ret_f = qk_w_f64_441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000442: *ret_f = qk_w_f64_442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000443: *ret_f = qk_w_f64_443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000444: *ret_f = qk_w_f64_444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001111: *ret_f = qk_w_f64_1111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001112: *ret_f = qk_w_f64_1112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001113: *ret_f = qk_w_f64_1113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001114: *ret_f = qk_w_f64_1114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001121: *ret_f = qk_w_f64_1121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001122: *ret_f = qk_w_f64_1122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001123: *ret_f = qk_w_f64_1123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001124: *ret_f = qk_w_f64_1124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001131: *ret_f = qk_w_f64_1131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001132: *ret_f = qk_w_f64_1132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001133: *ret_f = qk_w_f64_1133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001134: *ret_f = qk_w_f64_1134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001141: *ret_f = qk_w_f64_1141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001142: *ret_f = qk_w_f64_1142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001143: *ret_f = qk_w_f64_1143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001144: *ret_f = qk_w_f64_1144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001211: *ret_f = qk_w_f64_1211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001212: *ret_f = qk_w_f64_1212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001213: *ret_f = qk_w_f64_1213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001214: *ret_f = qk_w_f64_1214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001221: *ret_f = qk_w_f64_1221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001222: *ret_f = qk_w_f64_1222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001223: *ret_f = qk_w_f64_1223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001224: *ret_f = qk_w_f64_1224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001231: *ret_f = qk_w_f64_1231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001232: *ret_f = qk_w_f64_1232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001233: *ret_f = qk_w_f64_1233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001234: *ret_f = qk_w_f64_1234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001241: *ret_f = qk_w_f64_1241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001242: *ret_f = qk_w_f64_1242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001243: *ret_f = qk_w_f64_1243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001244: *ret_f = qk_w_f64_1244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001311: *ret_f = qk_w_f64_1311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001312: *ret_f = qk_w_f64_1312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001313: *ret_f = qk_w_f64_1313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001314: *ret_f = qk_w_f64_1314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001321: *ret_f = qk_w_f64_1321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001322: *ret_f = qk_w_f64_1322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001323: *ret_f = qk_w_f64_1323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001324: *ret_f = qk_w_f64_1324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001331: *ret_f = qk_w_f64_1331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001332: *ret_f = qk_w_f64_1332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001333: *ret_f = qk_w_f64_1333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001334: *ret_f = qk_w_f64_1334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001341: *ret_f = qk_w_f64_1341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001342: *ret_f = qk_w_f64_1342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001343: *ret_f = qk_w_f64_1343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001344: *ret_f = qk_w_f64_1344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001411: *ret_f = qk_w_f64_1411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001412: *ret_f = qk_w_f64_1412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001413: *ret_f = qk_w_f64_1413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001414: *ret_f = qk_w_f64_1414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001421: *ret_f = qk_w_f64_1421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001422: *ret_f = qk_w_f64_1422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001423: *ret_f = qk_w_f64_1423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001424: *ret_f = qk_w_f64_1424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001431: *ret_f = qk_w_f64_1431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001432: *ret_f = qk_w_f64_1432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001433: *ret_f = qk_w_f64_1433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001434: *ret_f = qk_w_f64_1434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001441: *ret_f = qk_w_f64_1441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001442: *ret_f = qk_w_f64_1442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001443: *ret_f = qk_w_f64_1443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000001444: *ret_f = qk_w_f64_1444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002111: *ret_f = qk_w_f64_2111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002112: *ret_f = qk_w_f64_2112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002113: *ret_f = qk_w_f64_2113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002114: *ret_f = qk_w_f64_2114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002121: *ret_f = qk_w_f64_2121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002122: *ret_f = qk_w_f64_2122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002123: *ret_f = qk_w_f64_2123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002124: *ret_f = qk_w_f64_2124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002131: *ret_f = qk_w_f64_2131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002132: *ret_f = qk_w_f64_2132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002133: *ret_f = qk_w_f64_2133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002134: *ret_f = qk_w_f64_2134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002141: *ret_f = qk_w_f64_2141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002142: *ret_f = qk_w_f64_2142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002143: *ret_f = qk_w_f64_2143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002144: *ret_f = qk_w_f64_2144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002211: *ret_f = qk_w_f64_2211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002212: *ret_f = qk_w_f64_2212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002213: *ret_f = qk_w_f64_2213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002214: *ret_f = qk_w_f64_2214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002221: *ret_f = qk_w_f64_2221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002222: *ret_f = qk_w_f64_2222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002223: *ret_f = qk_w_f64_2223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002224: *ret_f = qk_w_f64_2224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002231: *ret_f = qk_w_f64_2231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002232: *ret_f = qk_w_f64_2232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002233: *ret_f = qk_w_f64_2233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002234: *ret_f = qk_w_f64_2234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002241: *ret_f = qk_w_f64_2241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002242: *ret_f = qk_w_f64_2242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002243: *ret_f = qk_w_f64_2243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002244: *ret_f = qk_w_f64_2244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002311: *ret_f = qk_w_f64_2311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002312: *ret_f = qk_w_f64_2312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002313: *ret_f = qk_w_f64_2313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002314: *ret_f = qk_w_f64_2314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002321: *ret_f = qk_w_f64_2321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002322: *ret_f = qk_w_f64_2322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002323: *ret_f = qk_w_f64_2323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002324: *ret_f = qk_w_f64_2324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002331: *ret_f = qk_w_f64_2331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002332: *ret_f = qk_w_f64_2332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002333: *ret_f = qk_w_f64_2333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002334: *ret_f = qk_w_f64_2334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002341: *ret_f = qk_w_f64_2341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002342: *ret_f = qk_w_f64_2342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002343: *ret_f = qk_w_f64_2343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002344: *ret_f = qk_w_f64_2344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002411: *ret_f = qk_w_f64_2411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002412: *ret_f = qk_w_f64_2412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002413: *ret_f = qk_w_f64_2413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002414: *ret_f = qk_w_f64_2414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002421: *ret_f = qk_w_f64_2421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002422: *ret_f = qk_w_f64_2422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002423: *ret_f = qk_w_f64_2423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002424: *ret_f = qk_w_f64_2424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002431: *ret_f = qk_w_f64_2431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002432: *ret_f = qk_w_f64_2432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002433: *ret_f = qk_w_f64_2433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002434: *ret_f = qk_w_f64_2434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002441: *ret_f = qk_w_f64_2441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002442: *ret_f = qk_w_f64_2442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002443: *ret_f = qk_w_f64_2443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000002444: *ret_f = qk_w_f64_2444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003111: *ret_f = qk_w_f64_3111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003112: *ret_f = qk_w_f64_3112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003113: *ret_f = qk_w_f64_3113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003114: *ret_f = qk_w_f64_3114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003121: *ret_f = qk_w_f64_3121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003122: *ret_f = qk_w_f64_3122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003123: *ret_f = qk_w_f64_3123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003124: *ret_f = qk_w_f64_3124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003131: *ret_f = qk_w_f64_3131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003132: *ret_f = qk_w_f64_3132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003133: *ret_f = qk_w_f64_3133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003134: *ret_f = qk_w_f64_3134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003141: *ret_f = qk_w_f64_3141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003142: *ret_f = qk_w_f64_3142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003143: *ret_f = qk_w_f64_3143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003144: *ret_f = qk_w_f64_3144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003211: *ret_f = qk_w_f64_3211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003212: *ret_f = qk_w_f64_3212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003213: *ret_f = qk_w_f64_3213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003214: *ret_f = qk_w_f64_3214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003221: *ret_f = qk_w_f64_3221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003222: *ret_f = qk_w_f64_3222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003223: *ret_f = qk_w_f64_3223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003224: *ret_f = qk_w_f64_3224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003231: *ret_f = qk_w_f64_3231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003232: *ret_f = qk_w_f64_3232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003233: *ret_f = qk_w_f64_3233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003234: *ret_f = qk_w_f64_3234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003241: *ret_f = qk_w_f64_3241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003242: *ret_f = qk_w_f64_3242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003243: *ret_f = qk_w_f64_3243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003244: *ret_f = qk_w_f64_3244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003311: *ret_f = qk_w_f64_3311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003312: *ret_f = qk_w_f64_3312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003313: *ret_f = qk_w_f64_3313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003314: *ret_f = qk_w_f64_3314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003321: *ret_f = qk_w_f64_3321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003322: *ret_f = qk_w_f64_3322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003323: *ret_f = qk_w_f64_3323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003324: *ret_f = qk_w_f64_3324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003331: *ret_f = qk_w_f64_3331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003332: *ret_f = qk_w_f64_3332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003333: *ret_f = qk_w_f64_3333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003334: *ret_f = qk_w_f64_3334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003341: *ret_f = qk_w_f64_3341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003342: *ret_f = qk_w_f64_3342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003343: *ret_f = qk_w_f64_3343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003344: *ret_f = qk_w_f64_3344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003411: *ret_f = qk_w_f64_3411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003412: *ret_f = qk_w_f64_3412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003413: *ret_f = qk_w_f64_3413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003414: *ret_f = qk_w_f64_3414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003421: *ret_f = qk_w_f64_3421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003422: *ret_f = qk_w_f64_3422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003423: *ret_f = qk_w_f64_3423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003424: *ret_f = qk_w_f64_3424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003431: *ret_f = qk_w_f64_3431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003432: *ret_f = qk_w_f64_3432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003433: *ret_f = qk_w_f64_3433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003434: *ret_f = qk_w_f64_3434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003441: *ret_f = qk_w_f64_3441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003442: *ret_f = qk_w_f64_3442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003443: *ret_f = qk_w_f64_3443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000003444: *ret_f = qk_w_f64_3444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004111: *ret_f = qk_w_f64_4111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004112: *ret_f = qk_w_f64_4112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004113: *ret_f = qk_w_f64_4113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004114: *ret_f = qk_w_f64_4114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004121: *ret_f = qk_w_f64_4121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004122: *ret_f = qk_w_f64_4122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004123: *ret_f = qk_w_f64_4123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004124: *ret_f = qk_w_f64_4124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004131: *ret_f = qk_w_f64_4131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004132: *ret_f = qk_w_f64_4132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004133: *ret_f = qk_w_f64_4133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004134: *ret_f = qk_w_f64_4134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004141: *ret_f = qk_w_f64_4141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004142: *ret_f = qk_w_f64_4142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004143: *ret_f = qk_w_f64_4143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004144: *ret_f = qk_w_f64_4144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004211: *ret_f = qk_w_f64_4211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004212: *ret_f = qk_w_f64_4212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004213: *ret_f = qk_w_f64_4213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004214: *ret_f = qk_w_f64_4214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004221: *ret_f = qk_w_f64_4221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004222: *ret_f = qk_w_f64_4222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004223: *ret_f = qk_w_f64_4223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004224: *ret_f = qk_w_f64_4224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004231: *ret_f = qk_w_f64_4231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004232: *ret_f = qk_w_f64_4232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004233: *ret_f = qk_w_f64_4233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004234: *ret_f = qk_w_f64_4234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004241: *ret_f = qk_w_f64_4241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004242: *ret_f = qk_w_f64_4242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004243: *ret_f = qk_w_f64_4243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004244: *ret_f = qk_w_f64_4244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004311: *ret_f = qk_w_f64_4311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004312: *ret_f = qk_w_f64_4312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004313: *ret_f = qk_w_f64_4313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004314: *ret_f = qk_w_f64_4314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004321: *ret_f = qk_w_f64_4321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004322: *ret_f = qk_w_f64_4322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004323: *ret_f = qk_w_f64_4323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004324: *ret_f = qk_w_f64_4324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004331: *ret_f = qk_w_f64_4331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004332: *ret_f = qk_w_f64_4332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004333: *ret_f = qk_w_f64_4333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004334: *ret_f = qk_w_f64_4334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004341: *ret_f = qk_w_f64_4341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004342: *ret_f = qk_w_f64_4342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004343: *ret_f = qk_w_f64_4343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004344: *ret_f = qk_w_f64_4344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004411: *ret_f = qk_w_f64_4411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004412: *ret_f = qk_w_f64_4412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004413: *ret_f = qk_w_f64_4413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004414: *ret_f = qk_w_f64_4414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004421: *ret_f = qk_w_f64_4421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004422: *ret_f = qk_w_f64_4422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004423: *ret_f = qk_w_f64_4423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004424: *ret_f = qk_w_f64_4424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004431: *ret_f = qk_w_f64_4431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004432: *ret_f = qk_w_f64_4432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004433: *ret_f = qk_w_f64_4433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004434: *ret_f = qk_w_f64_4434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004441: *ret_f = qk_w_f64_4441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004442: *ret_f = qk_w_f64_4442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004443: *ret_f = qk_w_f64_4443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 3000004444: *ret_f = qk_w_f64_4444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000000: *(void**)ret_slot = qk_w_ptr_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001: *(void**)ret_slot = qk_w_ptr_1(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002: *(void**)ret_slot = qk_w_ptr_2(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003: *(void**)ret_slot = qk_w_ptr_3(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004: *(void**)ret_slot = qk_w_ptr_4(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000011: *(void**)ret_slot = qk_w_ptr_11(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000012: *(void**)ret_slot = qk_w_ptr_12(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000013: *(void**)ret_slot = qk_w_ptr_13(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000014: *(void**)ret_slot = qk_w_ptr_14(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000021: *(void**)ret_slot = qk_w_ptr_21(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000022: *(void**)ret_slot = qk_w_ptr_22(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000023: *(void**)ret_slot = qk_w_ptr_23(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000024: *(void**)ret_slot = qk_w_ptr_24(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000031: *(void**)ret_slot = qk_w_ptr_31(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000032: *(void**)ret_slot = qk_w_ptr_32(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000033: *(void**)ret_slot = qk_w_ptr_33(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000034: *(void**)ret_slot = qk_w_ptr_34(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000041: *(void**)ret_slot = qk_w_ptr_41(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000042: *(void**)ret_slot = qk_w_ptr_42(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000043: *(void**)ret_slot = qk_w_ptr_43(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 40000044: *(void**)ret_slot = qk_w_ptr_44(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000111: *(void**)ret_slot = qk_w_ptr_111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000112: *(void**)ret_slot = qk_w_ptr_112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000113: *(void**)ret_slot = qk_w_ptr_113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000114: *(void**)ret_slot = qk_w_ptr_114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000121: *(void**)ret_slot = qk_w_ptr_121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000122: *(void**)ret_slot = qk_w_ptr_122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000123: *(void**)ret_slot = qk_w_ptr_123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000124: *(void**)ret_slot = qk_w_ptr_124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000131: *(void**)ret_slot = qk_w_ptr_131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000132: *(void**)ret_slot = qk_w_ptr_132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000133: *(void**)ret_slot = qk_w_ptr_133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000134: *(void**)ret_slot = qk_w_ptr_134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000141: *(void**)ret_slot = qk_w_ptr_141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000142: *(void**)ret_slot = qk_w_ptr_142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000143: *(void**)ret_slot = qk_w_ptr_143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000144: *(void**)ret_slot = qk_w_ptr_144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000211: *(void**)ret_slot = qk_w_ptr_211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000212: *(void**)ret_slot = qk_w_ptr_212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000213: *(void**)ret_slot = qk_w_ptr_213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000214: *(void**)ret_slot = qk_w_ptr_214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000221: *(void**)ret_slot = qk_w_ptr_221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000222: *(void**)ret_slot = qk_w_ptr_222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000223: *(void**)ret_slot = qk_w_ptr_223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000224: *(void**)ret_slot = qk_w_ptr_224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000231: *(void**)ret_slot = qk_w_ptr_231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000232: *(void**)ret_slot = qk_w_ptr_232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000233: *(void**)ret_slot = qk_w_ptr_233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000234: *(void**)ret_slot = qk_w_ptr_234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000241: *(void**)ret_slot = qk_w_ptr_241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000242: *(void**)ret_slot = qk_w_ptr_242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000243: *(void**)ret_slot = qk_w_ptr_243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000244: *(void**)ret_slot = qk_w_ptr_244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000311: *(void**)ret_slot = qk_w_ptr_311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000312: *(void**)ret_slot = qk_w_ptr_312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000313: *(void**)ret_slot = qk_w_ptr_313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000314: *(void**)ret_slot = qk_w_ptr_314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000321: *(void**)ret_slot = qk_w_ptr_321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000322: *(void**)ret_slot = qk_w_ptr_322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000323: *(void**)ret_slot = qk_w_ptr_323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000324: *(void**)ret_slot = qk_w_ptr_324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000331: *(void**)ret_slot = qk_w_ptr_331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000332: *(void**)ret_slot = qk_w_ptr_332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000333: *(void**)ret_slot = qk_w_ptr_333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000334: *(void**)ret_slot = qk_w_ptr_334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000341: *(void**)ret_slot = qk_w_ptr_341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000342: *(void**)ret_slot = qk_w_ptr_342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000343: *(void**)ret_slot = qk_w_ptr_343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000344: *(void**)ret_slot = qk_w_ptr_344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000411: *(void**)ret_slot = qk_w_ptr_411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000412: *(void**)ret_slot = qk_w_ptr_412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000413: *(void**)ret_slot = qk_w_ptr_413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000414: *(void**)ret_slot = qk_w_ptr_414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000421: *(void**)ret_slot = qk_w_ptr_421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000422: *(void**)ret_slot = qk_w_ptr_422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000423: *(void**)ret_slot = qk_w_ptr_423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000424: *(void**)ret_slot = qk_w_ptr_424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000431: *(void**)ret_slot = qk_w_ptr_431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000432: *(void**)ret_slot = qk_w_ptr_432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000433: *(void**)ret_slot = qk_w_ptr_433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000434: *(void**)ret_slot = qk_w_ptr_434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000441: *(void**)ret_slot = qk_w_ptr_441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000442: *(void**)ret_slot = qk_w_ptr_442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000443: *(void**)ret_slot = qk_w_ptr_443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000444: *(void**)ret_slot = qk_w_ptr_444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001111: *(void**)ret_slot = qk_w_ptr_1111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001112: *(void**)ret_slot = qk_w_ptr_1112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001113: *(void**)ret_slot = qk_w_ptr_1113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001114: *(void**)ret_slot = qk_w_ptr_1114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001121: *(void**)ret_slot = qk_w_ptr_1121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001122: *(void**)ret_slot = qk_w_ptr_1122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001123: *(void**)ret_slot = qk_w_ptr_1123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001124: *(void**)ret_slot = qk_w_ptr_1124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001131: *(void**)ret_slot = qk_w_ptr_1131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001132: *(void**)ret_slot = qk_w_ptr_1132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001133: *(void**)ret_slot = qk_w_ptr_1133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001134: *(void**)ret_slot = qk_w_ptr_1134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001141: *(void**)ret_slot = qk_w_ptr_1141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001142: *(void**)ret_slot = qk_w_ptr_1142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001143: *(void**)ret_slot = qk_w_ptr_1143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001144: *(void**)ret_slot = qk_w_ptr_1144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001211: *(void**)ret_slot = qk_w_ptr_1211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001212: *(void**)ret_slot = qk_w_ptr_1212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001213: *(void**)ret_slot = qk_w_ptr_1213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001214: *(void**)ret_slot = qk_w_ptr_1214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001221: *(void**)ret_slot = qk_w_ptr_1221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001222: *(void**)ret_slot = qk_w_ptr_1222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001223: *(void**)ret_slot = qk_w_ptr_1223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001224: *(void**)ret_slot = qk_w_ptr_1224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001231: *(void**)ret_slot = qk_w_ptr_1231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001232: *(void**)ret_slot = qk_w_ptr_1232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001233: *(void**)ret_slot = qk_w_ptr_1233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001234: *(void**)ret_slot = qk_w_ptr_1234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001241: *(void**)ret_slot = qk_w_ptr_1241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001242: *(void**)ret_slot = qk_w_ptr_1242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001243: *(void**)ret_slot = qk_w_ptr_1243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001244: *(void**)ret_slot = qk_w_ptr_1244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001311: *(void**)ret_slot = qk_w_ptr_1311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001312: *(void**)ret_slot = qk_w_ptr_1312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001313: *(void**)ret_slot = qk_w_ptr_1313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001314: *(void**)ret_slot = qk_w_ptr_1314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001321: *(void**)ret_slot = qk_w_ptr_1321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001322: *(void**)ret_slot = qk_w_ptr_1322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001323: *(void**)ret_slot = qk_w_ptr_1323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001324: *(void**)ret_slot = qk_w_ptr_1324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001331: *(void**)ret_slot = qk_w_ptr_1331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001332: *(void**)ret_slot = qk_w_ptr_1332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001333: *(void**)ret_slot = qk_w_ptr_1333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001334: *(void**)ret_slot = qk_w_ptr_1334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001341: *(void**)ret_slot = qk_w_ptr_1341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001342: *(void**)ret_slot = qk_w_ptr_1342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001343: *(void**)ret_slot = qk_w_ptr_1343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001344: *(void**)ret_slot = qk_w_ptr_1344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001411: *(void**)ret_slot = qk_w_ptr_1411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001412: *(void**)ret_slot = qk_w_ptr_1412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001413: *(void**)ret_slot = qk_w_ptr_1413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001414: *(void**)ret_slot = qk_w_ptr_1414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001421: *(void**)ret_slot = qk_w_ptr_1421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001422: *(void**)ret_slot = qk_w_ptr_1422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001423: *(void**)ret_slot = qk_w_ptr_1423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001424: *(void**)ret_slot = qk_w_ptr_1424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001431: *(void**)ret_slot = qk_w_ptr_1431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001432: *(void**)ret_slot = qk_w_ptr_1432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001433: *(void**)ret_slot = qk_w_ptr_1433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001434: *(void**)ret_slot = qk_w_ptr_1434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001441: *(void**)ret_slot = qk_w_ptr_1441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001442: *(void**)ret_slot = qk_w_ptr_1442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001443: *(void**)ret_slot = qk_w_ptr_1443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000001444: *(void**)ret_slot = qk_w_ptr_1444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002111: *(void**)ret_slot = qk_w_ptr_2111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002112: *(void**)ret_slot = qk_w_ptr_2112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002113: *(void**)ret_slot = qk_w_ptr_2113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002114: *(void**)ret_slot = qk_w_ptr_2114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002121: *(void**)ret_slot = qk_w_ptr_2121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002122: *(void**)ret_slot = qk_w_ptr_2122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002123: *(void**)ret_slot = qk_w_ptr_2123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002124: *(void**)ret_slot = qk_w_ptr_2124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002131: *(void**)ret_slot = qk_w_ptr_2131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002132: *(void**)ret_slot = qk_w_ptr_2132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002133: *(void**)ret_slot = qk_w_ptr_2133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002134: *(void**)ret_slot = qk_w_ptr_2134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002141: *(void**)ret_slot = qk_w_ptr_2141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002142: *(void**)ret_slot = qk_w_ptr_2142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002143: *(void**)ret_slot = qk_w_ptr_2143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002144: *(void**)ret_slot = qk_w_ptr_2144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002211: *(void**)ret_slot = qk_w_ptr_2211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002212: *(void**)ret_slot = qk_w_ptr_2212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002213: *(void**)ret_slot = qk_w_ptr_2213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002214: *(void**)ret_slot = qk_w_ptr_2214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002221: *(void**)ret_slot = qk_w_ptr_2221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002222: *(void**)ret_slot = qk_w_ptr_2222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002223: *(void**)ret_slot = qk_w_ptr_2223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002224: *(void**)ret_slot = qk_w_ptr_2224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002231: *(void**)ret_slot = qk_w_ptr_2231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002232: *(void**)ret_slot = qk_w_ptr_2232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002233: *(void**)ret_slot = qk_w_ptr_2233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002234: *(void**)ret_slot = qk_w_ptr_2234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002241: *(void**)ret_slot = qk_w_ptr_2241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002242: *(void**)ret_slot = qk_w_ptr_2242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002243: *(void**)ret_slot = qk_w_ptr_2243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002244: *(void**)ret_slot = qk_w_ptr_2244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002311: *(void**)ret_slot = qk_w_ptr_2311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002312: *(void**)ret_slot = qk_w_ptr_2312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002313: *(void**)ret_slot = qk_w_ptr_2313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002314: *(void**)ret_slot = qk_w_ptr_2314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002321: *(void**)ret_slot = qk_w_ptr_2321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002322: *(void**)ret_slot = qk_w_ptr_2322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002323: *(void**)ret_slot = qk_w_ptr_2323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002324: *(void**)ret_slot = qk_w_ptr_2324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002331: *(void**)ret_slot = qk_w_ptr_2331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002332: *(void**)ret_slot = qk_w_ptr_2332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002333: *(void**)ret_slot = qk_w_ptr_2333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002334: *(void**)ret_slot = qk_w_ptr_2334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002341: *(void**)ret_slot = qk_w_ptr_2341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002342: *(void**)ret_slot = qk_w_ptr_2342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002343: *(void**)ret_slot = qk_w_ptr_2343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002344: *(void**)ret_slot = qk_w_ptr_2344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002411: *(void**)ret_slot = qk_w_ptr_2411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002412: *(void**)ret_slot = qk_w_ptr_2412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002413: *(void**)ret_slot = qk_w_ptr_2413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002414: *(void**)ret_slot = qk_w_ptr_2414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002421: *(void**)ret_slot = qk_w_ptr_2421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002422: *(void**)ret_slot = qk_w_ptr_2422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002423: *(void**)ret_slot = qk_w_ptr_2423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002424: *(void**)ret_slot = qk_w_ptr_2424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002431: *(void**)ret_slot = qk_w_ptr_2431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002432: *(void**)ret_slot = qk_w_ptr_2432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002433: *(void**)ret_slot = qk_w_ptr_2433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002434: *(void**)ret_slot = qk_w_ptr_2434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002441: *(void**)ret_slot = qk_w_ptr_2441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002442: *(void**)ret_slot = qk_w_ptr_2442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002443: *(void**)ret_slot = qk_w_ptr_2443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000002444: *(void**)ret_slot = qk_w_ptr_2444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003111: *(void**)ret_slot = qk_w_ptr_3111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003112: *(void**)ret_slot = qk_w_ptr_3112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003113: *(void**)ret_slot = qk_w_ptr_3113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003114: *(void**)ret_slot = qk_w_ptr_3114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003121: *(void**)ret_slot = qk_w_ptr_3121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003122: *(void**)ret_slot = qk_w_ptr_3122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003123: *(void**)ret_slot = qk_w_ptr_3123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003124: *(void**)ret_slot = qk_w_ptr_3124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003131: *(void**)ret_slot = qk_w_ptr_3131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003132: *(void**)ret_slot = qk_w_ptr_3132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003133: *(void**)ret_slot = qk_w_ptr_3133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003134: *(void**)ret_slot = qk_w_ptr_3134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003141: *(void**)ret_slot = qk_w_ptr_3141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003142: *(void**)ret_slot = qk_w_ptr_3142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003143: *(void**)ret_slot = qk_w_ptr_3143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003144: *(void**)ret_slot = qk_w_ptr_3144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003211: *(void**)ret_slot = qk_w_ptr_3211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003212: *(void**)ret_slot = qk_w_ptr_3212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003213: *(void**)ret_slot = qk_w_ptr_3213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003214: *(void**)ret_slot = qk_w_ptr_3214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003221: *(void**)ret_slot = qk_w_ptr_3221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003222: *(void**)ret_slot = qk_w_ptr_3222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003223: *(void**)ret_slot = qk_w_ptr_3223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003224: *(void**)ret_slot = qk_w_ptr_3224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003231: *(void**)ret_slot = qk_w_ptr_3231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003232: *(void**)ret_slot = qk_w_ptr_3232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003233: *(void**)ret_slot = qk_w_ptr_3233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003234: *(void**)ret_slot = qk_w_ptr_3234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003241: *(void**)ret_slot = qk_w_ptr_3241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003242: *(void**)ret_slot = qk_w_ptr_3242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003243: *(void**)ret_slot = qk_w_ptr_3243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003244: *(void**)ret_slot = qk_w_ptr_3244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003311: *(void**)ret_slot = qk_w_ptr_3311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003312: *(void**)ret_slot = qk_w_ptr_3312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003313: *(void**)ret_slot = qk_w_ptr_3313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003314: *(void**)ret_slot = qk_w_ptr_3314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003321: *(void**)ret_slot = qk_w_ptr_3321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003322: *(void**)ret_slot = qk_w_ptr_3322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003323: *(void**)ret_slot = qk_w_ptr_3323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003324: *(void**)ret_slot = qk_w_ptr_3324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003331: *(void**)ret_slot = qk_w_ptr_3331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003332: *(void**)ret_slot = qk_w_ptr_3332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003333: *(void**)ret_slot = qk_w_ptr_3333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003334: *(void**)ret_slot = qk_w_ptr_3334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003341: *(void**)ret_slot = qk_w_ptr_3341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003342: *(void**)ret_slot = qk_w_ptr_3342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003343: *(void**)ret_slot = qk_w_ptr_3343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003344: *(void**)ret_slot = qk_w_ptr_3344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003411: *(void**)ret_slot = qk_w_ptr_3411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003412: *(void**)ret_slot = qk_w_ptr_3412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003413: *(void**)ret_slot = qk_w_ptr_3413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003414: *(void**)ret_slot = qk_w_ptr_3414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003421: *(void**)ret_slot = qk_w_ptr_3421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003422: *(void**)ret_slot = qk_w_ptr_3422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003423: *(void**)ret_slot = qk_w_ptr_3423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003424: *(void**)ret_slot = qk_w_ptr_3424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003431: *(void**)ret_slot = qk_w_ptr_3431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003432: *(void**)ret_slot = qk_w_ptr_3432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003433: *(void**)ret_slot = qk_w_ptr_3433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003434: *(void**)ret_slot = qk_w_ptr_3434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003441: *(void**)ret_slot = qk_w_ptr_3441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003442: *(void**)ret_slot = qk_w_ptr_3442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003443: *(void**)ret_slot = qk_w_ptr_3443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000003444: *(void**)ret_slot = qk_w_ptr_3444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004111: *(void**)ret_slot = qk_w_ptr_4111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004112: *(void**)ret_slot = qk_w_ptr_4112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004113: *(void**)ret_slot = qk_w_ptr_4113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004114: *(void**)ret_slot = qk_w_ptr_4114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004121: *(void**)ret_slot = qk_w_ptr_4121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004122: *(void**)ret_slot = qk_w_ptr_4122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004123: *(void**)ret_slot = qk_w_ptr_4123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004124: *(void**)ret_slot = qk_w_ptr_4124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004131: *(void**)ret_slot = qk_w_ptr_4131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004132: *(void**)ret_slot = qk_w_ptr_4132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004133: *(void**)ret_slot = qk_w_ptr_4133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004134: *(void**)ret_slot = qk_w_ptr_4134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004141: *(void**)ret_slot = qk_w_ptr_4141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004142: *(void**)ret_slot = qk_w_ptr_4142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004143: *(void**)ret_slot = qk_w_ptr_4143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004144: *(void**)ret_slot = qk_w_ptr_4144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004211: *(void**)ret_slot = qk_w_ptr_4211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004212: *(void**)ret_slot = qk_w_ptr_4212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004213: *(void**)ret_slot = qk_w_ptr_4213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004214: *(void**)ret_slot = qk_w_ptr_4214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004221: *(void**)ret_slot = qk_w_ptr_4221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004222: *(void**)ret_slot = qk_w_ptr_4222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004223: *(void**)ret_slot = qk_w_ptr_4223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004224: *(void**)ret_slot = qk_w_ptr_4224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004231: *(void**)ret_slot = qk_w_ptr_4231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004232: *(void**)ret_slot = qk_w_ptr_4232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004233: *(void**)ret_slot = qk_w_ptr_4233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004234: *(void**)ret_slot = qk_w_ptr_4234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004241: *(void**)ret_slot = qk_w_ptr_4241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004242: *(void**)ret_slot = qk_w_ptr_4242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004243: *(void**)ret_slot = qk_w_ptr_4243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004244: *(void**)ret_slot = qk_w_ptr_4244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004311: *(void**)ret_slot = qk_w_ptr_4311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004312: *(void**)ret_slot = qk_w_ptr_4312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004313: *(void**)ret_slot = qk_w_ptr_4313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004314: *(void**)ret_slot = qk_w_ptr_4314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004321: *(void**)ret_slot = qk_w_ptr_4321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004322: *(void**)ret_slot = qk_w_ptr_4322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004323: *(void**)ret_slot = qk_w_ptr_4323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004324: *(void**)ret_slot = qk_w_ptr_4324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004331: *(void**)ret_slot = qk_w_ptr_4331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004332: *(void**)ret_slot = qk_w_ptr_4332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004333: *(void**)ret_slot = qk_w_ptr_4333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004334: *(void**)ret_slot = qk_w_ptr_4334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004341: *(void**)ret_slot = qk_w_ptr_4341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004342: *(void**)ret_slot = qk_w_ptr_4342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004343: *(void**)ret_slot = qk_w_ptr_4343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004344: *(void**)ret_slot = qk_w_ptr_4344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004411: *(void**)ret_slot = qk_w_ptr_4411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004412: *(void**)ret_slot = qk_w_ptr_4412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004413: *(void**)ret_slot = qk_w_ptr_4413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004414: *(void**)ret_slot = qk_w_ptr_4414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004421: *(void**)ret_slot = qk_w_ptr_4421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004422: *(void**)ret_slot = qk_w_ptr_4422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004423: *(void**)ret_slot = qk_w_ptr_4423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004424: *(void**)ret_slot = qk_w_ptr_4424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004431: *(void**)ret_slot = qk_w_ptr_4431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004432: *(void**)ret_slot = qk_w_ptr_4432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004433: *(void**)ret_slot = qk_w_ptr_4433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004434: *(void**)ret_slot = qk_w_ptr_4434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004441: *(void**)ret_slot = qk_w_ptr_4441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004442: *(void**)ret_slot = qk_w_ptr_4442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004443: *(void**)ret_slot = qk_w_ptr_4443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 4000004444: *(void**)ret_slot = qk_w_ptr_4444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000: *ret_i = qk_w_i64_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001: *ret_i = qk_w_i64_1(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002: *ret_i = qk_w_i64_2(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003: *ret_i = qk_w_i64_3(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004: *ret_i = qk_w_i64_4(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000011: *ret_i = qk_w_i64_11(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000012: *ret_i = qk_w_i64_12(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000013: *ret_i = qk_w_i64_13(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000014: *ret_i = qk_w_i64_14(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000021: *ret_i = qk_w_i64_21(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000022: *ret_i = qk_w_i64_22(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000023: *ret_i = qk_w_i64_23(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000024: *ret_i = qk_w_i64_24(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000031: *ret_i = qk_w_i64_31(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000032: *ret_i = qk_w_i64_32(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000033: *ret_i = qk_w_i64_33(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000034: *ret_i = qk_w_i64_34(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000041: *ret_i = qk_w_i64_41(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000042: *ret_i = qk_w_i64_42(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000043: *ret_i = qk_w_i64_43(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 60000044: *ret_i = qk_w_i64_44(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000111: *ret_i = qk_w_i64_111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000112: *ret_i = qk_w_i64_112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000113: *ret_i = qk_w_i64_113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000114: *ret_i = qk_w_i64_114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000121: *ret_i = qk_w_i64_121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000122: *ret_i = qk_w_i64_122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000123: *ret_i = qk_w_i64_123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000124: *ret_i = qk_w_i64_124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000131: *ret_i = qk_w_i64_131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000132: *ret_i = qk_w_i64_132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000133: *ret_i = qk_w_i64_133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000134: *ret_i = qk_w_i64_134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000141: *ret_i = qk_w_i64_141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000142: *ret_i = qk_w_i64_142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000143: *ret_i = qk_w_i64_143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000144: *ret_i = qk_w_i64_144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000211: *ret_i = qk_w_i64_211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000212: *ret_i = qk_w_i64_212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000213: *ret_i = qk_w_i64_213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000214: *ret_i = qk_w_i64_214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000221: *ret_i = qk_w_i64_221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000222: *ret_i = qk_w_i64_222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000223: *ret_i = qk_w_i64_223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000224: *ret_i = qk_w_i64_224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000231: *ret_i = qk_w_i64_231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000232: *ret_i = qk_w_i64_232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000233: *ret_i = qk_w_i64_233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000234: *ret_i = qk_w_i64_234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000241: *ret_i = qk_w_i64_241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000242: *ret_i = qk_w_i64_242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000243: *ret_i = qk_w_i64_243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000244: *ret_i = qk_w_i64_244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000311: *ret_i = qk_w_i64_311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000312: *ret_i = qk_w_i64_312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000313: *ret_i = qk_w_i64_313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000314: *ret_i = qk_w_i64_314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000321: *ret_i = qk_w_i64_321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000322: *ret_i = qk_w_i64_322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000323: *ret_i = qk_w_i64_323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000324: *ret_i = qk_w_i64_324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000331: *ret_i = qk_w_i64_331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000332: *ret_i = qk_w_i64_332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000333: *ret_i = qk_w_i64_333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000334: *ret_i = qk_w_i64_334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000341: *ret_i = qk_w_i64_341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000342: *ret_i = qk_w_i64_342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000343: *ret_i = qk_w_i64_343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000344: *ret_i = qk_w_i64_344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000411: *ret_i = qk_w_i64_411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000412: *ret_i = qk_w_i64_412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000413: *ret_i = qk_w_i64_413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000414: *ret_i = qk_w_i64_414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000421: *ret_i = qk_w_i64_421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000422: *ret_i = qk_w_i64_422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000423: *ret_i = qk_w_i64_423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000424: *ret_i = qk_w_i64_424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000431: *ret_i = qk_w_i64_431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000432: *ret_i = qk_w_i64_432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000433: *ret_i = qk_w_i64_433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000434: *ret_i = qk_w_i64_434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000441: *ret_i = qk_w_i64_441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000442: *ret_i = qk_w_i64_442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000443: *ret_i = qk_w_i64_443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 600000444: *ret_i = qk_w_i64_444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001111: *ret_i = qk_w_i64_1111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001112: *ret_i = qk_w_i64_1112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001113: *ret_i = qk_w_i64_1113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001114: *ret_i = qk_w_i64_1114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001121: *ret_i = qk_w_i64_1121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001122: *ret_i = qk_w_i64_1122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001123: *ret_i = qk_w_i64_1123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001124: *ret_i = qk_w_i64_1124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001131: *ret_i = qk_w_i64_1131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001132: *ret_i = qk_w_i64_1132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001133: *ret_i = qk_w_i64_1133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001134: *ret_i = qk_w_i64_1134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001141: *ret_i = qk_w_i64_1141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001142: *ret_i = qk_w_i64_1142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001143: *ret_i = qk_w_i64_1143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001144: *ret_i = qk_w_i64_1144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001211: *ret_i = qk_w_i64_1211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001212: *ret_i = qk_w_i64_1212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001213: *ret_i = qk_w_i64_1213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001214: *ret_i = qk_w_i64_1214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001221: *ret_i = qk_w_i64_1221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001222: *ret_i = qk_w_i64_1222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001223: *ret_i = qk_w_i64_1223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001224: *ret_i = qk_w_i64_1224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001231: *ret_i = qk_w_i64_1231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001232: *ret_i = qk_w_i64_1232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001233: *ret_i = qk_w_i64_1233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001234: *ret_i = qk_w_i64_1234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001241: *ret_i = qk_w_i64_1241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001242: *ret_i = qk_w_i64_1242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001243: *ret_i = qk_w_i64_1243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001244: *ret_i = qk_w_i64_1244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001311: *ret_i = qk_w_i64_1311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001312: *ret_i = qk_w_i64_1312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001313: *ret_i = qk_w_i64_1313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001314: *ret_i = qk_w_i64_1314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001321: *ret_i = qk_w_i64_1321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001322: *ret_i = qk_w_i64_1322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001323: *ret_i = qk_w_i64_1323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001324: *ret_i = qk_w_i64_1324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001331: *ret_i = qk_w_i64_1331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001332: *ret_i = qk_w_i64_1332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001333: *ret_i = qk_w_i64_1333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001334: *ret_i = qk_w_i64_1334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001341: *ret_i = qk_w_i64_1341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001342: *ret_i = qk_w_i64_1342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001343: *ret_i = qk_w_i64_1343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001344: *ret_i = qk_w_i64_1344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001411: *ret_i = qk_w_i64_1411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001412: *ret_i = qk_w_i64_1412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001413: *ret_i = qk_w_i64_1413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001414: *ret_i = qk_w_i64_1414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001421: *ret_i = qk_w_i64_1421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001422: *ret_i = qk_w_i64_1422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001423: *ret_i = qk_w_i64_1423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001424: *ret_i = qk_w_i64_1424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001431: *ret_i = qk_w_i64_1431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001432: *ret_i = qk_w_i64_1432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001433: *ret_i = qk_w_i64_1433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001434: *ret_i = qk_w_i64_1434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001441: *ret_i = qk_w_i64_1441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001442: *ret_i = qk_w_i64_1442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001443: *ret_i = qk_w_i64_1443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000001444: *ret_i = qk_w_i64_1444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002111: *ret_i = qk_w_i64_2111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002112: *ret_i = qk_w_i64_2112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002113: *ret_i = qk_w_i64_2113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002114: *ret_i = qk_w_i64_2114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002121: *ret_i = qk_w_i64_2121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002122: *ret_i = qk_w_i64_2122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002123: *ret_i = qk_w_i64_2123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002124: *ret_i = qk_w_i64_2124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002131: *ret_i = qk_w_i64_2131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002132: *ret_i = qk_w_i64_2132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002133: *ret_i = qk_w_i64_2133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002134: *ret_i = qk_w_i64_2134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002141: *ret_i = qk_w_i64_2141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002142: *ret_i = qk_w_i64_2142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002143: *ret_i = qk_w_i64_2143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002144: *ret_i = qk_w_i64_2144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002211: *ret_i = qk_w_i64_2211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002212: *ret_i = qk_w_i64_2212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002213: *ret_i = qk_w_i64_2213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002214: *ret_i = qk_w_i64_2214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002221: *ret_i = qk_w_i64_2221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002222: *ret_i = qk_w_i64_2222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002223: *ret_i = qk_w_i64_2223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002224: *ret_i = qk_w_i64_2224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002231: *ret_i = qk_w_i64_2231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002232: *ret_i = qk_w_i64_2232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002233: *ret_i = qk_w_i64_2233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002234: *ret_i = qk_w_i64_2234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002241: *ret_i = qk_w_i64_2241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002242: *ret_i = qk_w_i64_2242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002243: *ret_i = qk_w_i64_2243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002244: *ret_i = qk_w_i64_2244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002311: *ret_i = qk_w_i64_2311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002312: *ret_i = qk_w_i64_2312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002313: *ret_i = qk_w_i64_2313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002314: *ret_i = qk_w_i64_2314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002321: *ret_i = qk_w_i64_2321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002322: *ret_i = qk_w_i64_2322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002323: *ret_i = qk_w_i64_2323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002324: *ret_i = qk_w_i64_2324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002331: *ret_i = qk_w_i64_2331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002332: *ret_i = qk_w_i64_2332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002333: *ret_i = qk_w_i64_2333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002334: *ret_i = qk_w_i64_2334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002341: *ret_i = qk_w_i64_2341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002342: *ret_i = qk_w_i64_2342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002343: *ret_i = qk_w_i64_2343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002344: *ret_i = qk_w_i64_2344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002411: *ret_i = qk_w_i64_2411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002412: *ret_i = qk_w_i64_2412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002413: *ret_i = qk_w_i64_2413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002414: *ret_i = qk_w_i64_2414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002421: *ret_i = qk_w_i64_2421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002422: *ret_i = qk_w_i64_2422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002423: *ret_i = qk_w_i64_2423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002424: *ret_i = qk_w_i64_2424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002431: *ret_i = qk_w_i64_2431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002432: *ret_i = qk_w_i64_2432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002433: *ret_i = qk_w_i64_2433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002434: *ret_i = qk_w_i64_2434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002441: *ret_i = qk_w_i64_2441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002442: *ret_i = qk_w_i64_2442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002443: *ret_i = qk_w_i64_2443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000002444: *ret_i = qk_w_i64_2444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003111: *ret_i = qk_w_i64_3111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003112: *ret_i = qk_w_i64_3112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003113: *ret_i = qk_w_i64_3113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003114: *ret_i = qk_w_i64_3114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003121: *ret_i = qk_w_i64_3121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003122: *ret_i = qk_w_i64_3122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003123: *ret_i = qk_w_i64_3123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003124: *ret_i = qk_w_i64_3124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003131: *ret_i = qk_w_i64_3131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003132: *ret_i = qk_w_i64_3132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003133: *ret_i = qk_w_i64_3133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003134: *ret_i = qk_w_i64_3134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003141: *ret_i = qk_w_i64_3141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003142: *ret_i = qk_w_i64_3142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003143: *ret_i = qk_w_i64_3143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003144: *ret_i = qk_w_i64_3144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003211: *ret_i = qk_w_i64_3211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003212: *ret_i = qk_w_i64_3212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003213: *ret_i = qk_w_i64_3213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003214: *ret_i = qk_w_i64_3214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003221: *ret_i = qk_w_i64_3221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003222: *ret_i = qk_w_i64_3222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003223: *ret_i = qk_w_i64_3223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003224: *ret_i = qk_w_i64_3224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003231: *ret_i = qk_w_i64_3231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003232: *ret_i = qk_w_i64_3232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003233: *ret_i = qk_w_i64_3233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003234: *ret_i = qk_w_i64_3234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003241: *ret_i = qk_w_i64_3241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003242: *ret_i = qk_w_i64_3242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003243: *ret_i = qk_w_i64_3243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003244: *ret_i = qk_w_i64_3244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003311: *ret_i = qk_w_i64_3311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003312: *ret_i = qk_w_i64_3312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003313: *ret_i = qk_w_i64_3313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003314: *ret_i = qk_w_i64_3314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003321: *ret_i = qk_w_i64_3321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003322: *ret_i = qk_w_i64_3322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003323: *ret_i = qk_w_i64_3323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003324: *ret_i = qk_w_i64_3324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003331: *ret_i = qk_w_i64_3331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003332: *ret_i = qk_w_i64_3332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003333: *ret_i = qk_w_i64_3333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003334: *ret_i = qk_w_i64_3334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003341: *ret_i = qk_w_i64_3341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003342: *ret_i = qk_w_i64_3342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003343: *ret_i = qk_w_i64_3343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003344: *ret_i = qk_w_i64_3344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003411: *ret_i = qk_w_i64_3411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003412: *ret_i = qk_w_i64_3412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003413: *ret_i = qk_w_i64_3413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003414: *ret_i = qk_w_i64_3414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003421: *ret_i = qk_w_i64_3421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003422: *ret_i = qk_w_i64_3422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003423: *ret_i = qk_w_i64_3423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003424: *ret_i = qk_w_i64_3424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003431: *ret_i = qk_w_i64_3431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003432: *ret_i = qk_w_i64_3432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003433: *ret_i = qk_w_i64_3433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003434: *ret_i = qk_w_i64_3434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003441: *ret_i = qk_w_i64_3441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003442: *ret_i = qk_w_i64_3442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003443: *ret_i = qk_w_i64_3443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000003444: *ret_i = qk_w_i64_3444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004111: *ret_i = qk_w_i64_4111(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004112: *ret_i = qk_w_i64_4112(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004113: *ret_i = qk_w_i64_4113(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004114: *ret_i = qk_w_i64_4114(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004121: *ret_i = qk_w_i64_4121(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004122: *ret_i = qk_w_i64_4122(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004123: *ret_i = qk_w_i64_4123(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004124: *ret_i = qk_w_i64_4124(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004131: *ret_i = qk_w_i64_4131(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004132: *ret_i = qk_w_i64_4132(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004133: *ret_i = qk_w_i64_4133(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004134: *ret_i = qk_w_i64_4134(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004141: *ret_i = qk_w_i64_4141(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004142: *ret_i = qk_w_i64_4142(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004143: *ret_i = qk_w_i64_4143(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004144: *ret_i = qk_w_i64_4144(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004211: *ret_i = qk_w_i64_4211(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004212: *ret_i = qk_w_i64_4212(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004213: *ret_i = qk_w_i64_4213(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004214: *ret_i = qk_w_i64_4214(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004221: *ret_i = qk_w_i64_4221(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004222: *ret_i = qk_w_i64_4222(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004223: *ret_i = qk_w_i64_4223(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004224: *ret_i = qk_w_i64_4224(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004231: *ret_i = qk_w_i64_4231(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004232: *ret_i = qk_w_i64_4232(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004233: *ret_i = qk_w_i64_4233(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004234: *ret_i = qk_w_i64_4234(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004241: *ret_i = qk_w_i64_4241(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004242: *ret_i = qk_w_i64_4242(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004243: *ret_i = qk_w_i64_4243(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004244: *ret_i = qk_w_i64_4244(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004311: *ret_i = qk_w_i64_4311(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004312: *ret_i = qk_w_i64_4312(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004313: *ret_i = qk_w_i64_4313(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004314: *ret_i = qk_w_i64_4314(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004321: *ret_i = qk_w_i64_4321(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004322: *ret_i = qk_w_i64_4322(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004323: *ret_i = qk_w_i64_4323(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004324: *ret_i = qk_w_i64_4324(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004331: *ret_i = qk_w_i64_4331(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004332: *ret_i = qk_w_i64_4332(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004333: *ret_i = qk_w_i64_4333(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004334: *ret_i = qk_w_i64_4334(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004341: *ret_i = qk_w_i64_4341(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004342: *ret_i = qk_w_i64_4342(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004343: *ret_i = qk_w_i64_4343(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004344: *ret_i = qk_w_i64_4344(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004411: *ret_i = qk_w_i64_4411(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004412: *ret_i = qk_w_i64_4412(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004413: *ret_i = qk_w_i64_4413(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004414: *ret_i = qk_w_i64_4414(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004421: *ret_i = qk_w_i64_4421(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004422: *ret_i = qk_w_i64_4422(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004423: *ret_i = qk_w_i64_4423(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004424: *ret_i = qk_w_i64_4424(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004431: *ret_i = qk_w_i64_4431(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004432: *ret_i = qk_w_i64_4432(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004433: *ret_i = qk_w_i64_4433(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004434: *ret_i = qk_w_i64_4434(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004441: *ret_i = qk_w_i64_4441(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004442: *ret_i = qk_w_i64_4442(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004443: *ret_i = qk_w_i64_4443(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 6000004444: *ret_i = qk_w_i64_4444(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 100000: *ret_i = qk_w_i32_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 200000: *ret_f = (double)qk_w_f32_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 300000: *ret_f = qk_w_f64_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
        case 400000: *(void**)ret_slot = qk_w_ptr_0(fn, a1, a2, a3, a4, p1, p2, p3, p4); return 0;
    }
    return -2;
}
#endif
