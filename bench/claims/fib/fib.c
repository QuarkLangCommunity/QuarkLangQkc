#include <stdio.h>
#include <stdlib.h>
#include <time.h>

static int fib(int n) { return n < 2 ? n : fib(n - 1) + fib(n - 2); }

int main(int argc, char **argv) {
    volatile int n = argc > 1 ? atoi(argv[1]) : 35;
    struct timespec a, b;
    clock_gettime(CLOCK_MONOTONIC, &a);
    int r = fib(n);
    clock_gettime(CLOCK_MONOTONIC, &b);
    long us = (b.tv_sec - a.tv_sec) * 1000000L + (b.tv_nsec - a.tv_nsec) / 1000;
    printf("%ld\n%d\n", us, r);
    return 0;
}
