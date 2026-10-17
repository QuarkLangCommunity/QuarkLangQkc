/* Checked-in allocator workload for docs/compiled-memory-stage1.md.
 *
 * Compile with -DUSE_QL_ALLOC=1 and the embedded runtime (qthreads.c.txt) to measure the counted
 * slabs; compile with -DUSE_QL_ALLOC=0 to measure the libc path the compiled artifact used before
 * stage 1. The volatile sink stores every returned pointer and the payload is touched, so the
 * optimizer cannot delete the allocation in either build (the design's R1/T5 sink).
 *
 * Usage: alloc_bench <malloc|calloc> <bytes> <iterations>
 * Output: allocator,op,size,iterations,total_ns,ns_per_op
 */
#define _POSIX_C_SOURCE 199309L
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#if USE_QL_ALLOC
void* ql_alloc(size_t size, int owner);
void* ql_calloc(size_t count, size_t size, int owner);
void ql_free(void* ptr);
/* The generated IR owns these in a real compiled program; this workload never calls them. */
char* ql_int_to_str(int v);
char* ql_float_to_str(double v);
char* ql_int_to_str(int v) { (void)v; return 0; }
char* ql_float_to_str(double v) { (void)v; return 0; }
#define ALLOC(n) ql_alloc((n), 0)
#define CALLOC(n, s) ql_calloc((n), (s), 0)
#define FREE(p) ql_free(p)
#define ALLOCATOR "counted-slab"
#else
#define ALLOC(n) malloc(n)
#define CALLOC(n, s) calloc((n), (s))
#define FREE(p) free(p)
#define ALLOCATOR "libc"
#endif

static volatile void* sink;

int main(int argc, char** argv) {
    int calloc_mode;
    size_t size;
    long long iterations;
    long long i;
    struct timespec t0, t1;
    long long total_ns;
    void* p;
    if (argc != 4) {
        fprintf(stderr, "usage: %s <malloc|calloc> <bytes> <iterations>\n", argv[0]);
        return 2;
    }
    calloc_mode = strcmp(argv[1], "calloc") == 0;
    if (!calloc_mode && strcmp(argv[1], "malloc") != 0) {
        fprintf(stderr, "unknown operation: %s\n", argv[1]);
        return 2;
    }
    size = (size_t)strtoull(argv[2], 0, 10);
    iterations = strtoll(argv[3], 0, 10);
    if (size == 0 || iterations <= 0) {
        fprintf(stderr, "size and iterations must be positive\n");
        return 2;
    }
    sink = 0;
    clock_gettime(CLOCK_MONOTONIC, &t0);
    for (i = 0; i < iterations; i++) {
        if (calloc_mode) p = CALLOC(1, size);
        else p = ALLOC(size);
        if (!p) {
            fprintf(stderr, "allocation failed at iteration %lld\n", i);
            return 3;
        }
        ((unsigned char*)p)[0] = (unsigned char)i;
        ((unsigned char*)p)[size - 1] = (unsigned char)(i >> 8);
        sink = p;
        FREE(p);
    }
    clock_gettime(CLOCK_MONOTONIC, &t1);
    total_ns = (long long)(t1.tv_sec - t0.tv_sec) * 1000000000LL + (t1.tv_nsec - t0.tv_nsec);
    printf("%s,%s,%zu,%lld,%lld,%.3f\n", ALLOCATOR, calloc_mode ? "calloc" : "malloc",
           size, iterations, total_ns, (double)total_ns / (double)iterations);
    return 0;
}
