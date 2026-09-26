## 改了什么

<!-- 一到三行说清；关联 Issue 用 "Closes #123" -->

## 为什么

<!-- 背景/动机；性能类改动请附前后实测数据 -->

## 自检

- [ ] `go test ./...` 通过
- [ ] `(cd compiler && go test ./...)` 通过
- [ ] `(cd compiler && ./testdata/compare.sh)` 输出「全部一致」
- [ ] 若改了语法/语义：同步更新 `SYNTAX.md` 与 `compiler/testdata` 对比用例
- [ ] 若改了静态检查：更新 `internal/lang/testdata/lintbench/` 标注用例（误报/漏报门禁会跑）
- [ ] 若改了发布/CI：本地跑一次对应脚本冒烟

## 备注

<!-- 已知限制、后续计划、需要 reviewer 特别关注的点 -->
