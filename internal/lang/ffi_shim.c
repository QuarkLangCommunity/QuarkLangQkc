#ifdef _WIN32
/* Windows FFI：自研 wrapper 见 ffi_win_gen.c（免 libffi） */
#else
#include <stdint.h>
#include <stddef.h>
#include <string.h>

#ifdef _WIN32
#include <windows.h>
#define QK_EXPORT __declspec(dllexport)
#else
#include <dlfcn.h>
#define QK_EXPORT
#endif

#if defined(__APPLE__)
#include <ffi/ffi.h>   /* macOS：头文件在 SDK 的 usr/include/ffi/ 下 */
#else
#include <ffi.h>       /* Linux：libffi-dev 装在 /usr/include/ffi.h */
#endif

/* 跨系统库加载：POSIX=dlopen/dlsym（Linux/macOS），Windows=LoadLibrary/GetProcAddress */
QK_EXPORT void* qk_dlopen(const char* n) {
#ifdef _WIN32
    return (void*)LoadLibraryA(n);
#else
    return dlopen(n, RTLD_NOW | RTLD_GLOBAL);
#endif
}

QK_EXPORT void* qk_dlsym(void* h, const char* s) {
#ifdef _WIN32
    return (void*)GetProcAddress((HMODULE)h, s);
#else
    return dlsym(h, s);
#endif
}

QK_EXPORT const char* qk_dlerror(void) {
#ifdef _WIN32
    return "LoadLibrary/GetProcAddress failed (win32)";
#else
    return dlerror();
#endif
}

typedef union { int32_t i; float f; double d; void* p; } qk_arg;

/* 类型码：0=void 1=int32 2=f32 3=f64 4=ptr 5=bool 6=int64 */
QK_EXPORT int qk_ffi(void* fn, const int* types, const double* nums, void** ptrs,
                     int n, int rettype,
                     int64_t* ret_i, double* ret_f, void* ret_p_slot) {
    if (!fn) return -1;
    if (n < 0 || n > 32) return -2;
    ffi_cif cif;
    ffi_type* at[32];
    qk_arg av[32];
    memset(av, 0, sizeof(av));
    for (int k = 0; k < n; k++) {
        switch (types[k]) {
        case 1: case 5: at[k] = &ffi_type_sint32; av[k].i = (int32_t)nums[k]; break;
        case 2: at[k] = &ffi_type_float;  av[k].f = (float)nums[k]; break;
        case 3: at[k] = &ffi_type_double; av[k].d = nums[k]; break;
        case 6: at[k] = &ffi_type_sint64; av[k].i = (int32_t)nums[k]; /* 高位截断由 Go 侧规避 */ break;
        default: at[k] = &ffi_type_pointer; av[k].p = ptrs[k]; break;
        }
    }
    void* avp[32];
    for (int k = 0; k < n; k++) avp[k] = &av[k];
    ffi_type* rt = &ffi_type_pointer;
    switch (rettype) {
    case 0: rt = &ffi_type_void; break;
    case 1: case 5: rt = &ffi_type_sint32; break;
    case 2: rt = &ffi_type_float; break;
    case 3: rt = &ffi_type_double; break;
    case 6: rt = &ffi_type_sint64; break;
    }
    if (ffi_prep_cif(&cif, FFI_DEFAULT_ABI, n, rt, at) != FFI_OK) return -3;
    ffi_arg rv;
    ffi_call(&cif, FFI_FN(fn), &rv, avp);
    switch (rettype) {
    case 1: case 5: *ret_i = (int32_t)rv; break;
    case 2: { float fv; memcpy(&fv, &rv, 4); *ret_f = fv; break; }
    case 3: *(double*)ret_f = *(double*)&rv; break;
    case 6: *ret_i = (int64_t)rv; break;
    case 4: *(void**)ret_p_slot = (void*)rv; break;
    }
    return 0;
}
#endif
