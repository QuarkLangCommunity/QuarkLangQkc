# 库制品（.qklib）—— 编译出库 / 混淆 / 引用库

> 状态：**已打通的最终形态 —— 不依赖 .so**。库以**可移植 IR 多平台变体**分发，
> 使用方在**编译期合并**（预处理命令决定平台分支），跨系统一致；无共享库、无 dlopen、无平台二进制依赖。
> 相关实现：`compiler/lib.go`、`compiler/internal/cgen/libmode.go`、`compiler/main.go`

## 1. 为什么要有"库"

- **语言本体**（qkc / 运行时 / 语言核心）与**具体项目扩展**必须分开：项目扩展只是**一个库**，
  用 QuarkLang 写、编译成二进制，供其它程序**引用**而不必交出源码（商业机密友好）。
- Mod / 玩法扩展由此变成"**纯二进制制品**"：发布方给 `.qklib` 或 `.game_ext`，接收方只链接/执行。

## 2. 命令

```sh
# 出库：一份制品内含多个平台变体（由 #if os()/arch() 决定各变体内容）
qkc --emit-lib --obfuscate --lib-name game_ext --lib-version 0.2.0 \
    --lib-targets linux-x86_64,windows-x86_64,darwin-arm64,darwin-x86_64 \
    -o game-ext-0.2.0.qklib src/game_ext.qk

# 引用：按使用方目标平台挑变体，把 IR 合并进自己的模块（选项写在源文件之前）
qkc -c -L game-ext-0.2.0.qklib -o app app.qk
qkc -c --target-os windows -L game-ext-0.2.0.qklib -o app.exe app.qk   # 交叉构建
```

### 2.1 预处理指令（跨系统的关键工具）

```qk
#if os("windows")        // linux / darwin / windows / any
    int bias = 40;
#elif os("darwin")
    int bias = 38;
#else
    int bias = 37;
#endif
#if arch("x86_64") && defined(FEATURE_X)   // arch(): x86_64 / arm64 / any
...
#endif
#include "other.qk"       // 相对当前文件
#error 不该走到这里         // 条件不满足即编译失败
```
预处理在**使用方编译期**解析；`--target-os/--target-arch` 可交叉预处理（并参与缓存键）。

## 2. 旧命令（历史）

```sh
# 出库（可混淆）：产出 .qklib = 清单(JSON) + lib<名>.so
qkc --emit-lib --obfuscate --lib-name game_ext --lib-version 1.0.0 -o game-ext.qklib src/game_ext.qk

# 引用库：自动注入 library 声明 + 把 lib<名>.so 装到输出目录（-o 的目录）
qkc -c -L game-ext.qklib -o app app.qk     # 注意：选项写在源文件之前
```

调用语法（实测）：`库名.函数(...)`，例如 `game_ext.scale_cap(21)`。

## 3. 容器格式（.qklib，大端）

```
"QKLB" | ver(1) | manifestLen(uint32) | manifest(JSON) | lib<名>.so
manifest: {name, version, abi, soname, obfuscated, exports[{name, sig, ql_sig, params, ret}], sha256_obj, built_at, qkc}
```

## 4. 混淆（--obfuscate）

- IR 层：内部符号改名 `o_<sha256前12>`（**导出名保持不变**，否则无法引用）；
- 抹掉 `source_filename`、调试元数据（不泄漏源码路径/局部名）；
- 链接后 `strip --strip-debug`；
- **导出面由版本脚本精确控制**：`global: <导出清单>; local: *;`
  （比 `--strip-all`/`--strip-unneeded` 可靠——实测后者会把共享库的导出符号一起剥掉）。

## 5. 库模式（lib mode）

- 前端：`cgen.SetLibMode(true)` → **豁免 main 入口要求**，且**不发射 `main`**
  （库带 `main` 会在动态链接时顶替宿主入口 → 实测段错误）；
- IR 缓存按模式分文件（`<hash>-lib.ll`），避免"改了模式仍命中旧 IR"。

## 6. 未完成（下一步）

**症状**：使用方带 `-L` 链接时报
`undefined reference to 'ql_int_to_str' / 'ql_float_to_str'`（来自运行时对象 `qthreads.c`）。

**已定位的根因**：库的 IR 里**也发射了运行时辅助函数**（`ql_int_to_str` 等），
而版本脚本把它们标为 `local`（隐藏）。使用方的运行时对象引用这些名字时，
解析到的是库里的**隐藏副本** → 链接器报"未定义"。

**已修**：库模式把运行时辅助函数发射为 `declare`（外部），并在**与库链接时让宿主程序发射全部运行时辅助函数**
（`cgen.SetForceRuntimeHelpers(true)`）——链接期已通过（`qkc: 已构建 use`）。

### 6.1 已解决：导出函数的按引用/按值 ABI 不一致（**导出面 ABI 规范化**）

**做法**（`normalizeExportedABI`）：只改**导出函数**——把指针形参 `i32* %p0` 改为值形参 `i32 %p0.val`，
并在入口处插入 `%p0 = alloca i32` + `store i32 %p0.val, i32* %p0`，
使函数体原有的 `%p0` 用法**完全不变**（非导出函数不动，库内部约定不受影响）。
返回本就按值，无需处理。清单里的签名同步更新为规范化后的值签名（如 `(i32)`）。

实测：`qkc -c -L game-ext-0.1.0.qklib -o use use.qk && ./use` → **58**；
`readelf -d use` → `NEEDED libgame_ext.so` + `RUNPATH $ORIGIN`（链接期动态链接）；
`nm -D use | grep -c dlopen` → **0**（**不使用 dlopen**）。

gdb 实测（库调用崩在函数第一条指令）：
```
Program received signal SIGSEGV
#0  scale_cap () at libgame_ext.so
=> 0x...f0 <scale_cap>: mov (%rdi),%eax      ← 库把参数当**指针**解引用
rdi = 0x15                                       ← 调用方传的是**值** 21
```
- 库侧 IR：`define i32 @scale_cap(i32* noundef %p0)`（按引用）
- 调用侧 IR：`call i32 @scale_cap(i32 21)`（按值）
⇒ **必须让库导出函数采用与调用方一致的按值 ABI**（或显式约定 C-ABI 值类型边界）。

**下一步修法（二选一）**：
1. **按值 ABI**：修库模式的按引用分析（`lowerFunc`/`scanByRefArgs` 在无 main 的程序上判定退化），
   使 `fn f(int) int` 在库与程序中发射同一签名；
2. **显式 C-ABI 导出边界**：库只导出 `int/long/float/bool/char*/指针` 值类型（跨系统最稳，
   也避免 GC 逃逸分析差异），QL 侧用 `library X { fn f(int) int; }` 直接对应。

**未完成（旧记录）**：库模式下这些**运行时辅助函数应发射为 `declare`（外部）**，
让库显式依赖宿主（程序）提供的运行时——这正是动态链接应有的依赖方向：
`程序(含运行时) ← dlopen/链接 → 库(只含业务符号)`。

**顺带修掉的坑（已生效）**：
- 不要加 `-rdynamic`（实测触发同样的 `ql_*` 未定义）；
- 选项必须写在源文件之前（`qkc -c -L x.qklib -o app app.qk`）；
- `-c` 新鲜构建时也要把产物拷到 `-o` 路径（此前只在缓存命中时拷贝）。
