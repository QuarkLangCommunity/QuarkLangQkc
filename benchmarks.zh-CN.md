# QuarkLang 基准（可复现）

**简体中文** · [English](benchmarks.md)

> 本文件记录宣传视频/专栏中的全部数据，给出复现方法。环境：Linux x86-64，Go 1.26，clang/LLVM 22，Rust 1.x，Erlang/OTP，QuarkLang v2（qkc 编译路径）。

## 复现命令
```sh
# 解释器基准（fib/循环/调用/内存/碎片/P99）
cd QuarkLang && go test ./internal/lang/ -bench=. -benchtime=20x -run=NONE
# 编译器（LLVM 后端）
cd compiler && go build -o qkc .
./qkc -run examples/fib.qk
# 跨语言对比（C/Rust/Go/Erlang 源见 bench/）
cd bench && make all
# 同一程序跑两条后端（解释器 vs 原生 qkc），并断言输出一致
./scripts/bench-report.sh 3
# 双模块逐包测试覆盖率
./scripts/coverage.sh -o coverage.md
```

## 高计算（fib，LLVM -O3 同后端同机）
| 基准 | QuarkLang(编译) | C | Rust | Go | Erlang(BEAM) |
|---|---|---|---|---|---|
| fib(30) 1.66M 调用 | 3ms | 3ms | 3ms | 6ms | 1106ms |
| fib(35) 29.86M 调用 | 21ms | 22ms | 17ms | 36ms | — |
| 1 亿轮潮汐 | 1ms | 1ms | 29ms | 92ms | — |

## 高并发（8 路任务，用户态模型）
| 实现 | 8 路 × 1e7 求和 |
|---|---|
| QuarkLang taskm（线程池） | 1ms |
| Go goroutine | 6ms |
| Erlang process | 61s（10 万项） |

## P99 / P999（fib(20) × 1000 次，同机同采样）
| 分位 | QuarkLang(编译) | C |
|---|---|---|
| P50 | 14µs | 16µs |
| P99 | 27µs | 25µs |

## 内存（潮汐场景）
| 指标 | QuarkLang | Go |
|---|---|---|
| block 复用率 | 99.96% | — |
| 碎片率 | 0.195% | — |
| GC 停顿 | 零 | 有 |

## sum（位级置换/闭式优化）
| 10 亿项 | 时间 |
|---|---|
| QuarkLang 线性闭式 | 2ms |
| Go 循环 | 223ms |

## 公正性声明
- 同编译器（LLVM -O3）、同机、同轮次多次采样（20x）；
- P50/P99/P999 完整报告，非单次最优；
- 编译路径对比（qkc 产物 vs 原生）；解释器单独标注；
- Erlang 为 BEAM 虚拟机（架构不同，标注对比）。