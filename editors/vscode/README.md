# QuarkLang for VS Code

QuarkLang 语言支持扩展（工具链成员）：**语法高亮**（TextMate）+ **语言服务**（诊断/跳转/补全/悬停/大纲）。

## 安装（本地开发）

```sh
# 1) 构建语言服务器（仓库根目录）
go build -o qklsp ./cmd/qklsp

# 2) 让扩展找到它
export PATH="$PWD:$PATH"      # 或把 qklsp 放到 PATH；或在设置里指定：
#   quarklang.serverPath = /绝对路径/qklsp

# 3) 开发宿主里加载扩展
code --extensionDevelopmentPath=editors/vscode <某个 .qk 工程>
```

打包安装（需 `npx @vscode/vsce`，可选）：

```sh
cd editors/vscode && npx @vscode/vsce package     # 产出 quarklang-0.1.0.vsix
code --install-extension quarklang-0.1.0.vsix
```

## 能力

| 能力 | 提供者 | 说明 |
|---|---|---|
| 语法高亮 | `syntaxes/quarklang.tmLanguage.json` | 注释/字符串（含反引号原始串）/数字/关键字/内置类型/函数名/宏 `#…`/签名调用 `@mb`/运算符 |
| 片段 | `snippets/quarklang.json` | `fnmain` `fn` `struct` `interface` `impl` `space` `library` `for` `forin` `try` `macro` |
| 诊断 | `qklsp` | 解析/类型错误（`qkc`）+ 静态检查（`qkcheck`：未使用变量、不可达、遮蔽、缺返回、接口未实现、void 误用） |
| 跳转定义 | `qklsp` | 同一文件内的符号（含局部变量/形参/字段） |
| 补全 | `qklsp` | 关键字 + 内置类型 + 内置方法 + 当前文件的函数/类型/空间/局部变量 |
| 悬停 | `qklsp` | 签名 + 声明上方文档注释 |
| 大纲 | `qklsp` | `documentSymbol`：函数/结构体/接口/实现/空间/系统库 |

## 设置

| 设置 | 默认 | 说明 |
|---|---|---|
| `quarklang.serverPath` | `qklsp` | 语言服务器可执行文件（PATH 或绝对路径） |
| `quarklang.libDirs` | `[]` | 额外 import 搜索目录（传给 `qklsp -L`，多个可重复） |
| `quarklang.trace.server` | `false` | 把收发报文打到「输出 → QuarkLang」通道，排查用 |

## 实现说明（零 npm 依赖）

- `src/protocol.js`：JSON-RPC over stdio 的帧编解码与请求/通知生命周期，**不 import vscode**，
  因此可用 Node 直接联调（见下）。
- `src/extension.js`：VS Code 粘合层（DiagnosticCollection、Definition/Completion/Hover/DocumentSymbol Provider）。
- 不依赖 `vscode-languageclient`：扩展只需 vscode API 与 Node 内置模块，克隆即可用。

## 测试

```sh
go build -o /tmp/qklsp ./cmd/qklsp
QKLSP=/tmp/qklsp node editors/vscode/test/protocol.test.js
```

实测输出（八项全过）：

```
✓ initialize 返回能力          ✓ didOpen 收到诊断（含 QK101 未使用变量）
✓ 诊断带范围与消息             ✓ 定义跳转到第 2 行
✓ 补全含 double / io / while   ✓ 悬停显示签名与文档
✓ didChange 后诊断清空         ✓ 服务端优雅退出
```

> tree-sitter 语法另见 `editors/tree-sitter-quarklang/`（Neovim/Helix/Emacs 等可直接复用）。
