/*
 * The Windows FFI call shim: LoadLibrary/GetProcAddress plus one universal x64 call thunk.
 *
 * The old shim generated one C wrapper per combination of four argument types and at most four
 * arguments, which is why a call with five arguments could not be expressed at all. This shim instead
 * classifies each argument at run time and hands the values to a small thunk that loads the argument
 * registers and copies the rest of the arguments to the stack in the order the calling convention
 * requires. The number of arguments is therefore bounded by the frame this shim allocates, not by a
 * combinatorial table: the x64 convention itself has four integer and four vector argument registers
 * and passes every later argument on the stack.
 *
 * The two x64 conventions assign those registers differently. On Windows the register index is the
 * argument's position: argument i at i < 4 uses the i-th integer register (RCX/RDX/R8/R9) or the i-th
 * vector register (XMM0-XMM3) according to its class, and every argument from position 4 on travels on
 * the stack. On System V the classes are counted separately: the sixth integer argument and the ninth
 * vector argument, wherever they appear, are the first stack arguments.
 *
 * The same dispatcher is compiled on a POSIX host for the local regression test, with the System V
 * register file (six integer, eight vector registers, no shadow space) and the same classification
 * code, so the logic that decides register versus stack can be exercised without Windows. The thunk
 * itself is necessarily ABI-specific; the Windows CI leg is what proves the Microsoft x64 one.
 */
#ifdef _WIN32
#include <windows.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>

void* qk_dlopen(const char* n) { return (void*)LoadLibraryA(n); }
void* qk_dlsym(void* h, const char* s) { return (void*)GetProcAddress((HMODULE)h, s); }
const char* qk_dlerror(void) { return "LoadLibrary/GetProcAddress failed (win32)"; }

#define QK_CALL_IREGS 4
#define QK_CALL_FREGS 4
#define QK_CALL_SHADOW 32
#define QK_CALL_WIN64 1
#elif defined(__x86_64__)
#include <stdint.h>
#include <stddef.h>
#include <string.h>

#define QK_CALL_IREGS 6
#define QK_CALL_FREGS 8
#define QK_CALL_SHADOW 0
#define QK_CALL_SYSV64 1
#else
#error "the local FFI shim test build requires an x86-64 host"
#endif

/* The most arguments one call may carry; the same bound the libffi backends use. */
#define QK_CALL_MAX_ARGS 32

/* The Microsoft x64 rule maps each of the first four argument positions to the same-index register. */
#define QK_CALL_WIN_REGS 4

/* One prepared call: the classifier fills the register file and the stack slots, and the thunk runs it. */
typedef struct {
    void* fn;
    int n_stack;
    uint64_t ireg[6];
    uint64_t freg[8];
    const uint64_t* stack;
    uint64_t* out;
} qk_call;

typedef char qk_call_size_is_stable[(sizeof(qk_call) == 144) ? 1 : -1];
typedef char qk_call_ireg_offset_is_stable[(offsetof(qk_call, ireg) == 16) ? 1 : -1];
typedef char qk_call_freg_offset_is_stable[(offsetof(qk_call, freg) == 64) ? 1 : -1];
typedef char qk_call_stack_offset_is_stable[(offsetof(qk_call, stack) == 128) ? 1 : -1];

#ifdef QK_CALL_WIN64
/* qk_call_invoke runs the prepared call with the Microsoft x64 convention and stores RAX/XMM0 in out. */
__attribute__((naked)) static void qk_call_invoke(qk_call* c) {
    __asm__ __volatile__(
        "push %rbx\n\t"
        "push %rbp\n\t"
        "push %rsi\n\t"
        "push %rdi\n\t"
        "sub $8, %rsp\n\t"
        "mov %rcx, %rbx\n\t"
        "mov 136(%rbx), %rbp\n\t"
        "mov 8(%rbx), %eax\n\t"
        "lea 32(,%rax,8), %r11\n\t"
        "add $15, %r11\n\t"
        "and $-16, %r11\n\t"
        "sub %r11, %rsp\n\t"
        "mov 128(%rbx), %rsi\n\t"
        "lea 32(%rsp), %rdi\n\t"
        "mov 8(%rbx), %rcx\n\t"
        "cld\n\t"
        "rep movsq\n\t"
        "mov 16(%rbx), %rcx\n\t"
        "mov 24(%rbx), %rdx\n\t"
        "mov 32(%rbx), %r8\n\t"
        "mov 40(%rbx), %r9\n\t"
        "movsd 64(%rbx), %xmm0\n\t"
        "movsd 72(%rbx), %xmm1\n\t"
        "movsd 80(%rbx), %xmm2\n\t"
        "movsd 88(%rbx), %xmm3\n\t"
        "call *(%rbx)\n\t"
        "mov %rax, 0(%rbp)\n\t"
        "movsd %xmm0, 8(%rbp)\n\t"
        "mov 8(%rbx), %eax\n\t"
        "lea 32(,%rax,8), %r11\n\t"
        "add $15, %r11\n\t"
        "and $-16, %r11\n\t"
        "add %r11, %rsp\n\t"
        "add $8, %rsp\n\t"
        "pop %rdi\n\t"
        "pop %rsi\n\t"
        "pop %rbp\n\t"
        "pop %rbx\n\t"
        "ret\n\t"
    );
}
#else
/* qk_call_invoke runs the prepared call with the System V x64 convention and stores RAX/XMM0 in out. */
__attribute__((naked)) static void qk_call_invoke(qk_call* c) {
    __asm__ __volatile__(
        "push %rbx\n\t"
        "push %rbp\n\t"
        "sub $8, %rsp\n\t"
        "mov %rdi, %rbx\n\t"
        "mov 136(%rbx), %rbp\n\t"
        "mov 8(%rbx), %eax\n\t"
        "lea 8(,%rax,8), %r11\n\t"
        "add $15, %r11\n\t"
        "and $-16, %r11\n\t"
        "sub %r11, %rsp\n\t"
        "mov 128(%rbx), %rsi\n\t"
        "mov %rsp, %rdi\n\t"
        "mov 8(%rbx), %rcx\n\t"
        "cld\n\t"
        "rep movsq\n\t"
        "mov 16(%rbx), %rdi\n\t"
        "mov 24(%rbx), %rsi\n\t"
        "mov 32(%rbx), %rdx\n\t"
        "mov 40(%rbx), %rcx\n\t"
        "mov 48(%rbx), %r8\n\t"
        "mov 56(%rbx), %r9\n\t"
        "movsd 64(%rbx), %xmm0\n\t"
        "movsd 72(%rbx), %xmm1\n\t"
        "movsd 80(%rbx), %xmm2\n\t"
        "movsd 88(%rbx), %xmm3\n\t"
        "movsd 96(%rbx), %xmm4\n\t"
        "movsd 104(%rbx), %xmm5\n\t"
        "movsd 112(%rbx), %xmm6\n\t"
        "movsd 120(%rbx), %xmm7\n\t"
        "call *(%rbx)\n\t"
        "mov %rax, 0(%rbp)\n\t"
        "movsd %xmm0, 8(%rbp)\n\t"
        "mov 8(%rbx), %eax\n\t"
        "lea 8(,%rax,8), %r11\n\t"
        "add $15, %r11\n\t"
        "and $-16, %r11\n\t"
        "add %r11, %rsp\n\t"
        "add $8, %rsp\n\t"
        "pop %rbp\n\t"
        "pop %rbx\n\t"
        "ret\n\t"
    );
}
#endif

/* qk_is_float reports whether an FFI type code occupies a vector register. */
static int qk_is_float(int type) { return type == 2 || type == 3; }

/* qk_lay_out reports where argument k goes (0 integer register, 1 vector register, 2 stack) and its slot; win64 selects the assignment rule. */
static int qk_lay_out(int win64, int k, int is_float, int *ni, int *nf, int *ns, int *slot) {
    if (win64) {
        if (k < QK_CALL_WIN_REGS) { *slot = k; return is_float; }
    } else if (is_float) {
        if (*nf < QK_CALL_FREGS) { *slot = (*nf)++; return 1; }
    } else if (*ni < QK_CALL_IREGS) {
        *slot = (*ni)++;
        return 0;
    }
    *slot = (*ns)++;
    return 2;
}

/* qk_ffi classifies the declared signature, runs the call through the thunk and decodes the return. */
int qk_ffi(void* fn, const int* types, const double* nums, void** ptrs, int n,
           int rettype, int64_t* ret_i, double* ret_f, void* ret_slot) {
    qk_call call;
    uint64_t stack[QK_CALL_MAX_ARGS];
    uint64_t out[2] = {0, 0};
    int ni = 0, nf = 0, ns = 0;
    int k;
    if (!fn) return -1;
    if (n < 0 || n > QK_CALL_MAX_ARGS) return -2;
    memset(&call, 0, sizeof(call));
    for (k = 0; k < n; k++) {
        uint64_t v = 0;
        int is_float = 0;
        switch (types[k]) {
        case 1: /* int32 */
        case 5: /* bool */
            v = (uint64_t)(int64_t)(int32_t)nums[k];
            break;
        case 6: /* int64 */
            v = (uint64_t)(int64_t)nums[k];
            break;
        case 2: { /* f32: the low 32 bits of the slot are the float (a float register or stack slot) */
            float f = (float)nums[k];
            uint32_t b = 0;
            memcpy(&b, &f, sizeof(b));
            v = b;
            is_float = 1;
            break;
        }
        case 3: { /* f64 */
            double d = nums[k];
            memcpy(&v, &d, sizeof(v));
            is_float = 1;
            break;
        }
        default: /* pointer / String */
            v = (uint64_t)ptrs[k];
            break;
        }
        {
            int slot = 0;
            int cls;
#ifdef QK_CALL_WIN64
            cls = qk_lay_out(1, k, is_float, &ni, &nf, &ns, &slot);
#else
            cls = qk_lay_out(0, k, is_float, &ni, &nf, &ns, &slot);
#endif
            if (cls == 0) call.ireg[slot] = v;
            else if (cls == 1) call.freg[slot] = v;
            else stack[slot] = v;
        }
    }
    call.fn = fn;
    call.n_stack = ns;
    call.stack = stack;
    call.out = out;
    qk_call_invoke(&call);
    switch (rettype) {
    case 0: /* void */
        break;
    case 1: /* int32 */
    case 5: /* bool */
        *ret_i = (int64_t)(int32_t)(uint32_t)out[0];
        break;
    case 2: { /* f32 */
        uint32_t b = (uint32_t)out[1];
        float f = 0;
        memcpy(&f, &b, sizeof(f));
        *ret_f = (double)f;
        break;
    }
    case 3: { /* f64 */
        double d = 0;
        memcpy(&d, &out[1], sizeof(d));
        *ret_f = d;
        break;
    }
    case 4: /* pointer */
        *(void**)ret_slot = (void*)(uintptr_t)out[0];
        break;
    case 6: /* int64 */
        *ret_i = (int64_t)out[0];
        break;
    default:
        return -3;
    }
    return 0;
}
