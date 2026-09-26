# 贡献指南（CONTRIBUTING）

感谢你愿意为 QuarkLang 花时间。本文只讲**怎么做能最快被合入**，其余细节都在代码与文档里。

## 最快的三条路径

| 你想做 | 从哪里开始 |
|---|---|
| 报 bug / 提需求 | [新建 Issue](https://github.com/QuarkLangCommunity/QuarkLangQkc/issues/new/choose)（用模板，附最小复现） |
| 第一次贡献 | 挑 [`good first issue`](https://github.com/QuarkLangCommunity/QuarkLangQkc/labels/good%20first%20issue)（文档、示例、报错文案、测试用例，都不需要懂编译器内部） |
| 改代码 | 见下方流程；**main 受保护，必须走 PR 且 CI 全绿** |

## 提 PR 的流程

```sh
git clone https://github.com/QuarkLangCommunity/QuarkLangQkc && cd QuarkLangQkc
git switch -c fix/短描述            # 分支名：fix/… feat/… docs/… ci/…
# …改代码…
go test ./... && (cd compiler && go test ./...)      # ① 两层测试
(cd compiler && ./testdata/compare.sh)               # ② 双路径一致性（期望「全部一致」）
git commit -m "简短标题（中文，一行说清改了什么）"      # ③ 提交信息：中文、动词开头、必要时补正文
git push -u origin HEAD && gh pr create --fill       # ④ 开 PR；CI 会跑三平台矩阵
```

PR 合入条件（自动化，不需要人工点 Approve）：

| 门禁 | 说明 |
|---|---|
| 7 项必需检查 | 三平台 `测试（os）`、`Linux 专项`、三平台 `库与预处理器` |
| 分支保护 | `main` 禁止直推 / 禁 force-push / 禁删除 |

## 提交前自检清单（本机即可跑）

```sh
go test ./...                                    # 解释器 + 工具链（含 qkcheck 误报/漏报统计门禁）
(cd compiler && go test ./...)                   # 编译器
(cd compiler && ./testdata/compare.sh)           # 双路径一致性（同一份源码两条后端输出必须一致）
go build -o qkcheck ./cmd/qkcheck && ./qkcheck examples/ compiler/testdata   # 静态检查自举
go run ./scripts/... 2>/dev/null || true         # 发布/基准脚本改动时，跑一次冒烟
```

新增语言特性时，**必须同时更新**：

1. `SYNTAX.md`（唯一正典语法清单，逐编号 M/K/E/S/O/T/P）；
2. 解释器与编译器**两条路径**（共享前端，但后端语义要对齐）；
3. `compiler/testdata/compare.sh` 的对比用例（保证两条路径一致）；
4. 涉及静态检查时，`internal/lang/testdata/lintbench/` 的标注用例（误报/漏报门禁会跑它）。

## 代码风格

- **语法一律正典形态**：类型在前（`<修饰> <类型> <名字>`）；`impl { … } 名字;`；`space { … } 名字;`；不存在 `let`/`var` 声明。
- Go 侧：`gofmt` 干净、注释用中文说明「为什么」而不是「是什么」；零第三方依赖（新增依赖需在 PR 里说明理由）。
- 性能相关改动请附**前后实测**（`scripts/bench-tools.sh` 或 `go test -bench`），并说明是否先 profile。
- 报错文案面向使用者：说清「哪里错了 + 应该怎么写」，尽量给正典写法示例。

## 行为准则

参与本项目即表示你同意 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

## 联系方式

- Issues / PR 里 @ [@Enoch-199811](https://github.com/Enoch-199811)（维护者）。
- 安全问题请**不要**开公开 Issue，直接用 GitHub 的私密报告（Security → Report a vulnerability）。
