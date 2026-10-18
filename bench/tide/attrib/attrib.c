/* Attribution controls for the observable tide path. The phases are separated so each one can be
 * timed on its own; ql_list_int_str is extracted verbatim from compiler/qthreads.c.txt by
 * bench/tide/run.sh, so the "render" phase times the runtime function the language actually calls,
 * not a copy of it.
 *
 * The phases are controls, not language surface: every number printed here is microseconds for one
 * phase, and the harness subtracts nothing inside this program. */
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "ql_list_int_str.inc"

/* The counted-slab runtime the compiled tide links. */
void *ql_alloc(size_t size, int owner);
void *ql_calloc(size_t count, size_t size, int owner);
void ql_free(void *ptr);

/* The module-generated helpers the runtime's interface{} dispatch refers to. The attribution
 * phases never take that path, but the runtime object needs the symbols to link. */
char *ql_int_to_str(int v) {
    static char b[16];
    snprintf(b, sizeof b, "%d", v);
    return b;
}

char *ql_float_to_str(double v) {
    static char b[40];
    snprintf(b, sizeof b, "%.17g", v);
    return b;
}

static void *volatile sink;
static long long volatile acc;

/* now_us returns a monotonic timestamp in microseconds. */
static long long now_us(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (long long)ts.tv_sec * 1000000LL + ts.tv_nsec / 1000;
}

/* base is the loop and checksum with no allocation, string or sink. */
static void phase_base(long long n) {
    for (long long i = 0; i < n; i++) acc += (i % 7) + (i % 13);
}

/* allocpair is the two counted-slab allocations a List<int> literal performs and their releases,
 * with none of the header or element stores. */
static void phase_allocpair(long long n) {
    for (long long i = 0; i < n; i++) {
        void *header = ql_calloc(1, 16, 0);
        void *buffer = ql_alloc(12, 0);
        sink = header;
        acc += ((int *)buffer)[0];
        ql_free(buffer);
        ql_free(header);
    }
}

/* header is allocpair plus the %List field and element stores a literal emits, and the two reads
 * the checksum performs. Its difference from allocpair is the List header/length bookkeeping. */
static void phase_header(long long n) {
    for (long long i = 0; i < n; i++) {
        int *buffer = (int *)ql_alloc(12, 0);
        int *header = (int *)ql_calloc(1, 16, 0);
        buffer[0] = (int)(i % 7);
        buffer[1] = (int)(i % 11);
        buffer[2] = (int)(i % 13);
        header[0] = (int)(intptr_t)buffer; /* buf */
        header[1] = 0;                     /* head */
        header[2] = 3;                     /* tail */
        sink = buffer;
        acc += buffer[0] + buffer[2] + header[2];
        ql_free(buffer);
        ql_free(header);
    }
}

/* string_alloc is the one host malloc/free ql_list_int_str performs per call, with the same touch
 * the helper makes on the returned buffer. */
static void phase_string_alloc(long long n) {
    for (long long i = 0; i < n; i++) {
        char *s = (char *)malloc(16);
        s[0] = '[';
        sink = s;
        acc += s[0];
        free(s);
    }
}

/* int_fmt is the three snprintf("%d") conversions per call, without the buffer around them. */
static void phase_int_fmt(long long n) {
    for (long long i = 0; i < n; i++) {
        char nb[16];
        int n0 = snprintf(nb, sizeof nb, "%d", (int)(i % 7));
        acc += nb[0] + n0;
        int n1 = snprintf(nb, sizeof nb, "%d", (int)(i % 11));
        acc += nb[0] + n1;
        int n2 = snprintf(nb, sizeof nb, "%d", (int)(i % 13));
        acc += nb[0] + n2;
    }
}

/* fast_fmt is the same three conversions through ql_itoa, the formatter List.toString uses now. */
static void phase_fast_fmt(long long n) {
    for (long long i = 0; i < n; i++) {
        char nb[16];
        int n0 = ql_itoa((int)(i % 7), nb);
        acc += nb[0] + n0;
        int n1 = ql_itoa((int)(i % 11), nb);
        acc += nb[0] + n1;
        int n2 = ql_itoa((int)(i % 13), nb);
        acc += nb[0] + n2;
    }
}

/* bookkeeping is the helper's buffer framing with fixed-width digits already in hand: one malloc,
 * the brackets and separators, the element copies, and one free. */
static void phase_bookkeeping(long long n) {
    for (long long i = 0; i < n; i++) {
        size_t cap = 16, o = 0;
        char *r = (char *)malloc(cap);
        r[o++] = '[';
        for (int e = 0; e < 3; e++) {
            const char *nb = "12";
            size_t len = 2;
            if (e > 0) {
                if (o + 2 > cap) { cap *= 2; r = (char *)realloc(r, cap); }
                r[o++] = ','; r[o++] = ' ';
            }
            if (o + len + 2 > cap) { cap = (o + len + 2) * 2; r = (char *)realloc(r, cap); }
            memcpy(r + o, nb, len);
            o += len;
        }
        if (o + 2 > cap) { cap += 2; r = (char *)realloc(r, cap); }
        r[o++] = ']'; r[o] = 0;
        sink = r;
        acc += r[0];
        free(r);
    }
}

/* render times the real helper on the benchmark's three values and releases its result, exactly as
 * the C sink does. */
static void phase_render(long long n) {
    for (long long i = 0; i < n; i++) {
        int buf[3];
        buf[0] = (int)(i % 7);
        buf[1] = (int)(i % 11);
        buf[2] = (int)(i % 13);
        char *s = ql_list_int_str(buf, 0, 3);
        sink = s;
        acc += s[0];
        free(s);
    }
}

int main(int argc, char **argv) {
    if (argc < 3) {
        fprintf(stderr, "attrib: usage: attrib phase rounds\n");
        return 2;
    }
    const char *phase = argv[1];
    long long n = atoll(argv[2]);
    long long t0 = now_us();
    if (strcmp(phase, "base") == 0) phase_base(n);
    else if (strcmp(phase, "allocpair") == 0) phase_allocpair(n);
    else if (strcmp(phase, "header") == 0) phase_header(n);
    else if (strcmp(phase, "string_alloc") == 0) phase_string_alloc(n);
    else if (strcmp(phase, "int_fmt") == 0) phase_int_fmt(n);
    else if (strcmp(phase, "fast_fmt") == 0) phase_fast_fmt(n);
    else if (strcmp(phase, "bookkeeping") == 0) phase_bookkeeping(n);
    else if (strcmp(phase, "render") == 0) phase_render(n);
    else {
        fprintf(stderr, "attrib: unknown phase: %s\n", phase);
        return 2;
    }
    long long t1 = now_us();
    if (acc == 0) fprintf(stderr, "attrib: checksum collapsed\n");
    printf("%lld\n", t1 - t0);
    return 0;
}
