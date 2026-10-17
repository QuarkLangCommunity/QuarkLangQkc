/* Tide control, dead-code floor: the same loop without the volatile sink, so clang may remove the
 * malloc/free pair. The checksum keeps the loop itself alive. */
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <time.h>

/* now_us returns a monotonic timestamp in microseconds. */
static long long now_us(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (long long)ts.tv_sec * 1000000LL + ts.tv_nsec / 1000;
}

int main(int argc, char **argv) {
    long long n = argc > 1 ? atoll(argv[1]) : 100000000LL;
    long long acc = 0;
    long long t0 = now_us();
    for (long long i = 0; i < n; i++) {
        int32_t *p = malloc(3 * sizeof(int32_t));
        p[0] = (int32_t)(i % 7);
        p[1] = (int32_t)(i % 11);
        p[2] = (int32_t)(i % 13);
        acc += p[0] + p[2];
        free(p);
    }
    long long t1 = now_us();
    printf("%lld\n%lld\n", t1 - t0, acc);
    return 0;
}
