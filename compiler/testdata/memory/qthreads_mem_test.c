/* Allocation-accounting harness for compiler/qthreads.c.txt.
 *
 * It is compiled by compiler/memory_runtime_test.go together with the actual embedded runtime
 * (runtimeSource(): the generated message catalog followed by qthreads.c.txt), under
 * -DQL_MEM_DEBUG=1 and again without it. The debug build turns the allocator's assertions on; both
 * builds call ql_mem_check() after every batch, so the invariant walk is exercised too.
 *
 * The loop is the design's T4 shape: delete/reallocate under load with several size classes and the
 * large path, live objects kept across iterations, and a full check at the end. It is intentionally
 * deterministic; a failure prints the harness line that saw it.
 */
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void* ql_alloc(size_t size, int owner);
void* ql_calloc(size_t count, size_t size, int owner);
void* ql_realloc(void* ptr, size_t size);
void ql_free(void* ptr);
int ql_mem_set_block(int n);
void ql_mem_compact(void);
void ql_mem_clear(void);
void ql_mem_task_begin(int owner);
void ql_mem_task_end(int owner);
double ql_mem_fragmentation(void);
double ql_mem_occupancy(void);
long long ql_mem_block_count(void);
long long ql_mem_live_bytes(void);
long long ql_mem_used_bytes(void);
long long ql_mem_alloc_calls(void);
long long ql_mem_free_calls(void);
void ql_mem_stats(long long* reuse, long long* new_blocks, long long* block_count);
int ql_mem_check(void);

/* A routed runtime helper (D1): the harness calls it directly to prove its allocation is counted. */
char* ql_strcat(const char* a, const char* b);

/* qthreads' task pool: used here only to put the allocator on its multi-threaded lock path. */
int ql_spawn(void);
void ql_merge(int id, void* fn, long long a0, long long a1, long long a2, long long a3,
              long long a4, long long a5, long long a6, long long a7);
void ql_block(int id);

/* The generated IR owns these in a real compiled program; the allocator test never calls them. */
char* ql_int_to_str(int v);
char* ql_float_to_str(double v);
char* ql_int_to_str(int v) { (void)v; return 0; }
char* ql_float_to_str(double v) { (void)v; return 0; }

static uint64_t rng_state = 0x9e3779b97f4a7c15ull;

static uint32_t rng(void) {
    rng_state ^= rng_state << 13;
    rng_state ^= rng_state >> 7;
    rng_state ^= rng_state << 17;
    return (uint32_t)rng_state;
}

#define CHECK(cond) do { \
    if (!(cond)) { \
        fprintf(stderr, "qthreads_mem_test:%d: %s\n", __LINE__, #cond); \
        return 1; \
    } \
} while (0)

static int basic_sizes(void) {
    static const size_t sizes[] = {0, 1, 8, 16, 17, 48, 100, 512, 1024, 1025, 4096, 50000};
    void* p[sizeof sizes / sizeof sizes[0]];
    size_t i;
    for (i = 0; i < sizeof sizes / sizeof sizes[0]; i++) {
        p[i] = ql_alloc(sizes[i], 7);
        CHECK(p[i] != 0);
        CHECK(((uintptr_t)p[i] % 16) == 0);
        memset(p[i], (int)(i + 1), sizes[i]);
    }
    CHECK(ql_mem_check() == 0);
    for (i = 0; i < sizeof sizes / sizeof sizes[0]; i++) ql_free(p[i]);
    CHECK(ql_mem_check() == 0);
    ql_mem_compact();
    CHECK(ql_mem_block_count() == 0);
    CHECK(ql_mem_live_bytes() == 0);
    CHECK(ql_mem_used_bytes() == 0);
    CHECK(ql_mem_fragmentation() == 0.0);
    return 0;
}

static int calloc_and_realloc(void) {
    unsigned char* p = (unsigned char*)ql_calloc(64, 1, 0);
    int i;
    CHECK(p != 0);
    for (i = 0; i < 64; i++) CHECK(p[i] == 0);
    for (i = 0; i < 64; i++) p[i] = (unsigned char)(i + 1);
    p = (unsigned char*)ql_realloc(p, 4096);
    CHECK(p != 0);
    for (i = 0; i < 64; i++) CHECK(p[i] == (unsigned char)(i + 1));
    p = (unsigned char*)ql_realloc(p, 16);
    CHECK(p != 0);
    for (i = 0; i < 16; i++) CHECK(p[i] == (unsigned char)(i + 1));
    ql_free(p);
    p = (unsigned char*)ql_realloc(0, 16);
    CHECK(p != 0);
    p = (unsigned char*)ql_realloc(p, 0);
    CHECK(p == 0);
    ql_free(0);
    CHECK(ql_mem_check() == 0);
    return 0;
}

/* D3(a): release does not wipe; the next same-class allocation may see the old payload. calloc is the
 * operation that zeroes, and the allocator must not pretend otherwise. */
static int payload_is_preserved(void) {
    unsigned char* p = (unsigned char*)ql_alloc(32, 0);
    unsigned char* q;
    int i;
    CHECK(p != 0);
    for (i = 0; i < 32; i++) p[i] = (unsigned char)(0xA0 + i);
    ql_free(p);
    q = (unsigned char*)ql_alloc(32, 0);
    CHECK(q != 0);
    for (i = 0; i < 32; i++) CHECK(q[i] == (unsigned char)(0xA0 + i));
    ql_free(q);
    q = (unsigned char*)ql_calloc(1, 32, 0);
    CHECK(q != 0);
    for (i = 0; i < 32; i++) CHECK(q[i] == 0);
    ql_free(q);
    CHECK(ql_mem_check() == 0);
    return 0;
}

static int block_size(void) {
    void* a;
    CHECK(ql_mem_set_block(0) == 0);
    CHECK(ql_mem_set_block((1 << 20) + 1) == 0);
    CHECK(ql_mem_set_block(1) == 1);
    a = ql_alloc(64, 0);
    CHECK(a != 0);
    CHECK(((uintptr_t)a % 16) == 0);
    CHECK(ql_mem_check() == 0);
    ql_free(a);
    CHECK(ql_mem_set_block(4096) == 1);
    ql_mem_compact();
    CHECK(ql_mem_block_count() == 0);
    return 0;
}

/* A tide that allocates and releases the same class reuses one slot and one slab after the first
 * iteration; this is the property §10 calls "reuse is the point". */
static int tide_reuses(void) {
    long long r0, b0, c0, r1, b1, c1;
    int i;
    ql_mem_compact();
    ql_mem_stats(&r0, &b0, &c0);
    for (i = 0; i < 1000; i++) {
        void* p = ql_alloc(64, 0);
        CHECK(p != 0);
        ql_free(p);
    }
    ql_mem_stats(&r1, &b1, &c1);
    CHECK(r1 - r0 == 999);
    CHECK(b1 - b0 == 1);
    CHECK(c1 - c0 == 1);
    CHECK(ql_mem_check() == 0);
    ql_mem_compact();
    CHECK(ql_mem_block_count() == 0);
    return 0;
}

static int fragmentation_and_occupancy(void) {
    void* p = ql_alloc(32, 0);
    void* q = ql_alloc(2000, 0);
    double f, o;
    CHECK(p != 0 && q != 0);
    f = ql_mem_fragmentation();
    o = ql_mem_occupancy();
    CHECK(f >= 0.0 && f <= 1.0);
    CHECK(o >= 0.0 && o <= 1.0);
    CHECK(f + o > 0.999 && f + o < 1.001);
    CHECK(ql_mem_used_bytes() > 0);
    ql_free(p);
    ql_free(q);
    ql_mem_compact();
    CHECK(ql_mem_fragmentation() == 0.0);
    CHECK(ql_mem_occupancy() == 0.0);
    CHECK(ql_mem_block_count() == 0);
    return 0;
}

/* A small list literal is one counted-slab block: 16 bytes of %List header followed by the elements.
 * The point of the co-allocation is that the accounting sees one allocation and one release, not two
 * of each; this is the C half of the IR tests (the generated release/grow helpers themselves are
 * exercised by cases_run/z_list_coalloc.kq through the real runtime). */
static int combined_list_is_one_counted_block(void) {
    long long a0, f0, u0;
    struct ql_list_like { void* buf; int head; int tail; }* l;
    int* buf;
    ql_mem_compact();
    a0 = ql_mem_alloc_calls();
    f0 = ql_mem_free_calls();
    u0 = ql_mem_used_bytes();
    l = (struct ql_list_like*)ql_alloc(16 + 3 * sizeof(int), 0);
    CHECK(l != 0);
    CHECK(((uintptr_t)l % 16) == 0);
    buf = (int*)((char*)l + 16);
    l->buf = buf;
    l->head = 0;
    l->tail = 3;
    buf[0] = 7;
    buf[1] = 11;
    buf[2] = 13;
    CHECK(ql_mem_alloc_calls() == a0 + 1);
    CHECK(ql_mem_free_calls() == f0);
    CHECK(ql_mem_used_bytes() == u0 + 16 + 3 * sizeof(int));
    CHECK(ql_mem_check() == 0);
    ql_free(l);
    CHECK(ql_mem_free_calls() == f0 + 1);
    CHECK(ql_mem_used_bytes() == u0);
    CHECK(ql_mem_check() == 0);
    ql_mem_compact();
    return 0;
}

/* The task pool enables the allocator lock before its workers start. Run one worker and the main
 * thread concurrently through alloc/free/realloc; the debug build's assertions and the final
 * ql_mem_check() then cover the lock path, not just the single-threaded fast path. */
static void mem_worker(long long iterations, long long a1, long long a2, long long a3,
                       long long a4, long long a5, long long a6, long long a7) {
    long long i;
    (void)a1; (void)a2; (void)a3; (void)a4; (void)a5; (void)a6; (void)a7;
    for (i = 0; i < iterations; i++) {
        size_t size = (size_t)((i * 37) % 5000 + 1);
        void* p = ql_alloc(size, 1);
        if (!p) abort();
        ((unsigned char*)p)[0] = (unsigned char)i;
        p = ql_realloc(p, (size_t)((i * 53) % 5000 + 1));
        if (!p) abort();
        ql_free(p);
    }
}

static int multithreaded_stress(void) {
    int id = ql_spawn();
    long long i;
    CHECK(id > 0);
    ql_merge(id, (void*)mem_worker, 30000, 0, 0, 0, 0, 0, 0, 0);
    for (i = 0; i < 30000; i++) {
        void* p = ql_alloc((size_t)((i * 29) % 3000 + 1), 0);
        CHECK(p != 0);
        ql_free(p);
    }
    ql_block(id);
    CHECK(ql_mem_check() == 0);
    ql_mem_compact();
    CHECK(ql_mem_check() == 0);
    /* D1: the task object ql_spawn allocated is a routed allocation and stays live (the runtime keeps
     * task handles in its table), so the metric does not return to zero. What compact released is the
     * workers' own empties; the spawned object's slab is not one of them. */
    CHECK(ql_mem_block_count() > 0);
    CHECK(ql_mem_live_bytes() > 0);
    return 0;
}

/* T4: deterministic delete/reallocate under load across classes and the large path. */
static int delete_reallocate_stress(void) {
    enum { SLOTS = 512 };
    void* live[SLOTS] = {0};
    size_t size[SLOTS] = {0};
    int n = 0;
    int iter;
    ql_mem_compact();
    for (iter = 0; iter < 200000; iter++) {
        uint32_t r = rng();
        int slot = (int)((r >> 2) % SLOTS);
        if (n == 0 || (r & 3) == 0) {
            size_t sz = (r >> 8) % 3000;
            if (live[slot]) {
                ql_free(live[slot]);
                live[slot] = 0;
                n--;
            }
            live[slot] = ql_alloc(sz, (int)(r & 7));
            CHECK(live[slot] != 0);
            size[slot] = sz;
            memset(live[slot], (int)(iter & 0xff), sz < 32 ? sz : 32);
            n++;
        } else if ((r & 3) == 1) {
            if (live[slot]) {
                ql_free(live[slot]);
                live[slot] = 0;
                n--;
            }
        } else if ((r & 3) == 2) {
            if (live[slot]) {
                size_t sz = 1 + (r >> 8) % 6000;
                unsigned char tag = ((unsigned char*)live[slot])[0];
                live[slot] = ql_realloc(live[slot], sz);
                CHECK(live[slot] != 0);
                if (size[slot] > 0) CHECK(((unsigned char*)live[slot])[0] == tag);
                size[slot] = sz;
            }
        } else {
            if (live[slot]) {
                size_t sz = size[slot];
                ((unsigned char*)live[slot])[sz ? sz - 1 : 0] = (unsigned char)(iter & 0xff);
                CHECK(ql_mem_check() == 0);
            }
        }
        if ((iter % 5000) == 0) CHECK(ql_mem_check() == 0);
    }
    for (iter = 0; iter < SLOTS; iter++) {
        if (live[iter]) {
            ql_free(live[iter]);
            live[iter] = 0;
        }
    }
    CHECK(ql_mem_check() == 0);
    ql_mem_compact();
    CHECK(ql_mem_check() == 0);
    CHECK(ql_mem_block_count() == 0);
    CHECK(ql_mem_live_bytes() == 0);
    CHECK(ql_mem_used_bytes() == 0);
    CHECK(ql_mem_fragmentation() == 0.0);
    return 0;
}

/* §10's clear is the derived wipe: release keeps the payload (D3), clear removes it from the region it
 * releases. A partially occupied slab stays cached, so its free slot can be read back by the next
 * same-class allocation; a live object in the same slab must not be touched. */
static int clear_wipes_released_region(void) {
    unsigned char* a;
    unsigned char* live;
    unsigned char* b;
    int i;
    ql_mem_compact();
    a = (unsigned char*)ql_alloc(16, 0);
    live = (unsigned char*)ql_alloc(16, 0);
    CHECK(a != 0 && live != 0);
    for (i = 0; i < 16; i++) {
        a[i] = 0xA5;
        live[i] = 0x5A;
    }
    ql_free(a);
    b = (unsigned char*)ql_alloc(16, 0);
    CHECK(b != 0);
    for (i = 0; i < 16; i++) CHECK(b[i] == 0xA5); /* release does not wipe */
    ql_free(b);
    ql_mem_clear();
    CHECK(ql_mem_check() == 0);
    for (i = 0; i < 16; i++) CHECK(live[i] == 0x5A); /* live payload untouched */
    b = (unsigned char*)ql_alloc(16, 0);
    CHECK(b != 0);
    for (i = 0; i < 16; i++) CHECK(b[i] == 0); /* the released region was wiped */
    ql_free(b);
    ql_free(live);
    ql_mem_clear();
    CHECK(ql_mem_check() == 0);
    CHECK(ql_mem_block_count() == 0); /* clear released the empty slab */
    CHECK(ql_mem_live_bytes() == 0);
    CHECK(ql_mem_used_bytes() == 0);
    CHECK(ql_mem_fragmentation() == 0.0);
    return 0;
}

/* D1: a runtime helper's allocation is visible to the same accounting the language allocations use. */
static int runtime_allocations_are_counted(void) {
    long long allocs;
    long long used;
    char* s;
    ql_mem_compact();
    allocs = ql_mem_alloc_calls();
    used = ql_mem_used_bytes();
    s = ql_strcat("ab", "cd");
    CHECK(s != 0);
    CHECK(strcmp(s, "abcd") == 0);
    CHECK(ql_mem_alloc_calls() == allocs + 1);
    CHECK(ql_mem_used_bytes() == used + 5);
    ql_free(s);
    CHECK(ql_mem_used_bytes() == used);
    CHECK(ql_mem_check() == 0);
    return 0;
}

/* D5: a block allocated by a task belongs to it; ending the task releases it, while the main
 * thread's block stays. */
static int task_owned_blocks_are_reclaimed(void) {
    long long before_used;
    long long before_blocks;
    void* keep;
    void* task;
    ql_mem_compact();
    before_used = ql_mem_used_bytes();
    before_blocks = ql_mem_block_count();
    keep = ql_alloc(32, 0);
    CHECK(keep != 0);
    ql_mem_task_begin(11);
    task = ql_alloc(32, 0); /* owner 0 inside a task means "this task" */
    CHECK(task != 0);
    CHECK(ql_alloc(50000, 0) != 0); /* the large path carries the owner in its metadata too */
    CHECK(ql_mem_used_bytes() == before_used + 64 + 50000);
    ql_mem_task_end(11);
    CHECK(ql_mem_used_bytes() == before_used + 32);
    CHECK(ql_mem_block_count() == before_blocks + 1);
    CHECK(ql_mem_check() == 0);
    ql_free(keep);
    ql_mem_compact();
    CHECK(ql_mem_used_bytes() == 0);
    CHECK(ql_mem_block_count() == 0);
    return 0;
}

/* The worker allocates and frees nothing: the task's end is what must reclaim its storage. */
static void reclaim_worker(long long iterations, long long a1, long long a2, long long a3,
                           long long a4, long long a5, long long a6, long long a7) {
    long long i;
    (void)a1; (void)a2; (void)a3; (void)a4; (void)a5; (void)a6; (void)a7;
    for (i = 0; i < iterations; i++) {
        size_t size = (i % 2) ? 5000 : 64; /* both the slab and the large path */
        void* p = ql_alloc(size, 0);
        if (!p) abort();
        ((unsigned char*)p)[0] = 1;
    }
}

/* The integration half of D5: the real task pool runs a worker that allocates without releasing, and
 * ql_block observes the allocator back at the pre-merge used count because the worker reclaimed at the
 * task boundary before publishing done. */
static int task_end_reclaims_its_allocations(void) {
    long long before_used;
    int id = ql_spawn();
    CHECK(id > 0);
    before_used = ql_mem_used_bytes();
    ql_merge(id, (void*)reclaim_worker, 1000, 0, 0, 0, 0, 0, 0, 0);
    ql_block(id);
    CHECK(ql_mem_used_bytes() == before_used);
    CHECK(ql_mem_check() == 0);
    return 0;
}

int main(void) {
    long long reuse = 0, blocks = 0, count = 0;
    CHECK(basic_sizes() == 0);
    CHECK(calloc_and_realloc() == 0);
    CHECK(payload_is_preserved() == 0);
    CHECK(clear_wipes_released_region() == 0);
    CHECK(runtime_allocations_are_counted() == 0);
    CHECK(block_size() == 0);
    CHECK(tide_reuses() == 0);
    CHECK(fragmentation_and_occupancy() == 0);
    CHECK(combined_list_is_one_counted_block() == 0);
    CHECK(delete_reallocate_stress() == 0);
    /* The task tests spawn, and ql_spawn leaves a live, routed task object behind, so everything from
     * here on expects a non-zero block count after its own allocations have been released (D1). */
    CHECK(task_owned_blocks_are_reclaimed() == 0);
    CHECK(task_end_reclaims_its_allocations() == 0);
    CHECK(multithreaded_stress() == 0);
    ql_mem_stats(&reuse, &blocks, &count);
    printf("ok reuse=%lld new_blocks=%lld block_count=%lld alloc_calls=%lld free_calls=%lld\n",
           reuse, blocks, count, ql_mem_alloc_calls(), ql_mem_free_calls());
    return 0;
}
