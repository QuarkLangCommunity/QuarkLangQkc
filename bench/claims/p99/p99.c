#include <stdio.h>
#include <stdlib.h>
#include <time.h>

static int fib(int n) { return n < 2 ? n : fib(n - 1) + fib(n - 2); }

// countLE returns how many of rounds fib(20) calls finish within limit microseconds.
static int countLE(int limit, int rounds) {
    int c = 0;
    for (int i = 0; i < rounds; i++) {
        struct timespec a, b;
        clock_gettime(CLOCK_MONOTONIC, &a);
        int r = fib(20);
        clock_gettime(CLOCK_MONOTONIC, &b);
        long us = (b.tv_sec - a.tv_sec) * 1000000L + (b.tv_nsec - a.tv_nsec) / 1000;
        if (us <= limit) {
            c++;
        }
        (void)r;
    }
    return c;
}

int main(int argc, char **argv) {
    int target = argc > 1 ? atoi(argv[1]) : 500;
    int rounds = argc > 2 ? atoi(argv[2]) : 1000;
    int lo = 0, hi = 1000000;
    while (lo < hi) {
        int mid = (lo + hi) / 2;
        if (countLE(mid, rounds) >= target) {
            hi = mid;
        } else {
            lo = mid + 1;
        }
    }
    int checksum = fib(20);
    printf("%d\n%d\n", lo, checksum);
    return 0;
}
