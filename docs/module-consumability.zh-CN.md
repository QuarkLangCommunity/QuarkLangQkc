# 模块可用性：让 `compiler` 能在检出目录之外安装

**状态：提案。** 未改动任何 `go.mod`、未改动任何 workflow。本文只记录实测结果、各方案的代价，以及推荐项。

编译器是工具链里唯一预期以 Go 模块方式安装给用户的部分。目前它只能从 clone 里构建。本文复现失败、
解释那个已提交的 `replace` 为何存在，并给出让它可被消费的最小改动。

## 1. 已复现的失败

仓库状态：`origin/main` = `67ce88c`，Go 1.26.8。已发布标签：`v1.0.0`、`v2.0.0`、`v2.1.0`。
没有 `v0.0.0`，也没有任何标签带上后两个标签所要求的 `/v2` 模块路径后缀。

### 1.1 `go install`

```
$ go install github.com/QuarkLangCommunity/QuarkLangQkc/compiler@latest
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc v1.0.0
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc/compiler v0.0.0-20261010124804-67ce88cd1e52
go: github.com/QuarkLangCommunity/QuarkLangQkc/compiler@latest (in github.com/QuarkLangCommunity/QuarkLangQkc/compiler@v0.0.0-20261010124804-67ce88cd1e52):
	The go.mod file for the module providing named packages contains one or
	more replace directives. It must not contain directives that would cause
	it to be interpreted differently than if it were the main module.
```

### 1.2 `go get`

```
$ go get github.com/QuarkLangCommunity/QuarkLangQkc/compiler@latest
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc v1.0.0
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc/compiler v0.0.0-20261010124804-67ce88cd1e52
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc v0.0.0
go: github.com/QuarkLangCommunity/QuarkLangQkc/compiler imports
	github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n: reading github.com/QuarkLangCommunity/QuarkLangQkc/go.mod at revision v0.0.0: unknown revision v0.0.0
go: github.com/QuarkLangCommunity/QuarkLangQkc/compiler imports
	github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang: reading github.com/QuarkLangCommunity/QuarkLangQkc/go.mod at revision v0.0.0: unknown revision v0.0.0
```

同一布局还有第三种更直白的症状，出自那个确实打过标签的模块：

```
$ go install github.com/QuarkLangCommunity/QuarkLangQkc/compiler@v1.0.0
go: github.com/QuarkLangCommunity/QuarkLangQkc/compiler@v1.0.0: module github.com/QuarkLangCommunity/QuarkLangQkc@v1.0.0 found, but does not contain package github.com/QuarkLangCommunity/QuarkLangQkc/compiler
```

前两处失败同源，都出自 `compiler/go.mod` 里的 `replace github.com/QuarkLangCommunity/QuarkLangQkc => ../`，
但成因不同：

* `go install` 卡在 **`replace` 指令本身**。非主模块不允许带 `replace`，因为它会改变所有人的导入解析。
* `go get` 卡在 **只被该 replace 撑着的 `require … v0.0.0`**。相对路径 replace 在检出目录之外不可见，
  于是这个 require 就指向了从未打过标签的 `v0.0.0`。

## 2. replace 为何存在，它买到了什么

`compiler/go.mod` 是仓库内的第二个模块；当编译器作为主模块时，根模块自己的配置对它不生效。这个 replace
买到的正是「新 clone 无需根模块先发布到 proxy 即可构建」，而且确有实效：今天在 `compiler/` 里
`go build ./... && go test ./... -count=1` 通过，不依赖任何已发布的根模块。

代价是：这个 replace 本身就是编译器无法被外部消费的原因。在 `origin/main` 上的实测：

| 观测 | 命令 / 证据 |
|---|---|
| 根模块打过标签，但发布归档把编译器整个剔除了 | `go mod download …@v1.0.0` 成功；`unzip -l` 显示 **30 个文件**，其中 `compiler/*.go` 为 **0** —— proxy 会剔除嵌套模块 |
| 因此编译器从未能通过 proxy 取到 | `go install …/compiler@v1.0.0` → `module …@v1.0.0 found, but does not contain package …/compiler` |
| 编译器自身的导入是 `internal/`，外部不可导入 | 实测：外部模块导入 `…/compiler/internal/cgen` 被拒，报 `use of internal package … not allowed` |
| 消费者连 require 已发布版本都做不到 | `go get …@v1.0.0` → `parsing go.mod: module declares its path as: quarklang but was required as: github.com/QuarkLangCommunity/QuarkLangQkc` |
| 模块路径是在最后一个标签之后才更正的 | `git log -S'module github.com/QuarkLangCommunity/QuarkLangQkc' -- go.mod` → `6dad203`；`git show v1.0.0:go.mod` 与 `git show v2.1.0:go.mod` 都仍写着 `module quarklang` |

**对任何修法的影响：** 由于所有已发布标签都还写着 `module quarklang`，*现存版本没有一个可被消费*；
又由于 `v2.x` 在该路径下不可能是合法的 Go 模块版本（见 §3 方案 A），修法最终必须落在一个新的 **`v1.x`**
标签上，且带上已更正的模块路径。

## 3. 方案

### 方案 A —— 把编译器收进根模块，然后打 `v1.x` 标签

删除 `compiler/go.mod`（以及另一个嵌套模块 `bench/go.mod`），让仓库重新成为单一模块；把编译器对 parser
模块的额外 `require` 上移到根 `go.mod`；删掉已无意义的 replace；重新打一个 `v1.x` 标签。

* **代价**：小，且集中在 `go.mod` 记账上。`compiler/go.sum` 消失，重复的 parser pin 合并为一处。
* **标签约束**：必须是 `v1.x`。`v2.x` 会被直接拒绝，因为模块路径没有 `/v2` 后缀：
  `go get …@v2.2.0` → `invalid version: should be v0 or v1, not v2`。改走「给模块路径加 `/v2`」则会重写
  仓库里每一处导入路径，超出本提案范围。
* **CI 影响**：去掉三处重复的 build/test 步骤，也去掉一份需与根 pin 保持同步的第二 `go.mod`。
  `compiler/testdata/compare.sh` 不受影响（按路径调用 `go build`，不跨模块边界）。
* **对现有用户的影响**：今天能用的都不受影响。`cd compiler && go build` 照旧（目录仍在），
  而 `go install …/compiler@…` 首次变得可用。
* **可达性不变**：编译器在根模块树内，`internal/lang`、`internal/i18n`、`compiler/internal/cgen`
  仍是合法导入，无需重导出。

### 方案 B —— 保留 replace，写明编译器只能在检出目录内构建

两个 `go.mod` 都不动，只在 README 里说明 `qkc` 仅能从 clone 构建。

* **代价**：零代码改动；但模块路径仍然发布着且仍然坏着，猜到这两条命令的人依旧失败。
* **CI 影响**：无。
* **对现有用户的影响**：维持现状。真正的代价是对一个合理期待回答「不行」，而且答案写在散文里而不是模块元数据里。

### 方案 C —— 把编译器发布成独立模块路径

给编译器独立路径，例如 `…/QuarkLangQkc/compiler/v2`，让它依赖已打标签的根模块。

* **代价**：高，且当前根本走不通。编译器自己的导入都是 `internal/`（`internal/lang`、`internal/i18n`、
  `compiler/internal/cgen`），`internal` 规则会拒绝来自根模块树之外的每一次导入。要走这条路，必须先把这些
  包从公开路径重导出、或搬离 `internal/` —— 那是语言实现的 API 面变更，不是打包变更。该路径还需要 `/v2`
  以及与之匹配的标签。
* **CI 影响**：多一个需要打标签、定版本、定 pin 的已发布模块，发布流程要保证两条路径同步。
* **对现有用户的影响**：模块路径变更，按旧路径写的 `go install` 全部失效。

## 4. 推荐

**方案 A**，并遵守上面的标签约束。它是让模块可被消费的最小改动，也是唯一今天就可达成的方案。具体：

1. 删除 `compiler/go.mod` 与 `bench/go.mod`，把两个目录收进根模块；
2. 把编译器对 parser 模块的 `require` 上移到根 `go.mod`，删除 `compiler/go.sum`；
3. 删掉 `replace` —— 只剩一个模块时没有东西可 replace；
4. 打一个新的 **`v1.x`** 标签（不改写 `v1.0.0`，也不用 `v2.x`），由监听 `v*` 的 `release.yml` 发布。
   把项目自身的 `VERSION`/`v2.x` 发布编号与 Go 模块的版本线分开，否则两者会持续冲突。

在与真实 proxy 同构的本地 proxy 上（按上述编辑生成的归档，根模块以 `v1.2.0` 发布）端到端实测：

```
$ go get github.com/QuarkLangCommunity/QuarkLangQkc@v1.2.0
go: added github.com/QuarkLangCommunity/QuarkLangQkc v1.2.0
$ go install github.com/QuarkLangCommunity/QuarkLangQkc/compiler@latest
$ echo $?
0
$ ls -l "$(go env GOPATH)/bin"
-rwxr-xr-x 1 jack jack 8739928 compiler
$ compiler --help | head -1
usage: qkc [options] <file.qk>
$ compiler -run rec.kq
error: StackOverflowError: recursion depth exceeded 8192 at line 3
```

最后一行对验证本身有意义：新装出来的二进制是在真实程序上跑过的，不只是启动过；它报出的正是
`spec/STANDARD.md` §11.2 记录的那条共享调用深度上限。

另外两点供执行者参考：

* 模块一旦按标签发布，`@latest` 也能解析 —— 嵌套包不需要自己的 `compiler/vX.Y.Z` 标签，
  因为嵌套包的版本就是其所属模块的版本。此点同样是实测而非推断。
* 本文不改任何 `go.mod`、不改任何 workflow；第 1–4 步即提案内容。
