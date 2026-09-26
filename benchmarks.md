# QuarkLang benchmarks (reproducible)

**English** · [简体中文](benchmarks.zh-CN.md)

> This file records every number used in the promo video / articles, together with how to reproduce it.
> Environment: Linux x86-64, Go 1.26, clang/LLVM 22, Rust 1.x, Erlang/OTP, QuarkLang v2 (qkc compiled path).

## Reproduction commands

```sh
# Interpreter benchmarks (fib/loop/calls/memory/fragmentation/P99)
cd QuarkLang && go test ./internal/lang/ -bench=. -benchtime=20x -run=NONE
# Compiler (LLVM backend)
cd compiler && go build -o qkc .
./qkc -run examples/fib.qk
# Cross-language comparison (C/Rust/Go/Erlang sources in bench/)
cd bench && make all
```

## Compute-heavy (fib, same LLVM -O3 backend, same machine)

| Benchmark | QuarkLang (compiled) | C | Rust | Go | Erlang (BEAM) |
|---|---|---|---|---|---|
| fib(30), 1.66M calls | 3ms | 3ms | 3ms | 6ms | 1106ms |
| fib(35), 29.86M calls | 21ms | 22ms | 17ms | 36ms | — |
| 100M-round tide | 1ms | 1ms | 29ms | 92ms | — |

## High concurrency (8 tasks, user-space model)

| Implementation | 8-way × 1e7 summation |
|---|---|
| QuarkLang taskm (thread pool) | 1ms |
| Go goroutine | 6ms |
| Erlang process | 61s (1e5 items) |

## P99 / P999 (fib(20) × 1000, same machine, same sampling)

| Quantile | QuarkLang (compiled) | C |
|---|---|---|
| P50 | 14µs | 16µs |
| P99 | 27µs | 25µs |

## Memory (tide scenario)

| Metric | QuarkLang | Go |
|---|---|---|
| block reuse rate | 99.96% | — |
| fragmentation | 0.195% | — |
| GC pauses | none | yes |

## sum (bit-level permutation / closed form)

| 1e9 terms | Time |
|---|---|
| QuarkLang linear closed form | 2ms |
| Go loop | 223ms |

## Fairness statement

- Same compiler (LLVM -O3), same machine, multiple samples over 20x rounds;
- Full P50/P99/P999 reported, not a single best run;
- Compiled-path comparisons (qkc output vs native); interpreter numbers are labelled separately.
