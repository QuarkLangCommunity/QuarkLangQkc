# 变更日志

> 由 `scripts/changelog.sh --all` 从 git 历史生成（分组：新功能 / 修复 / 性能 / 重构与清理 / 文档 / 构建与工具链 / 其它）。

## v2.1.0 - 2026-09-26

### 修复

- ci: assert per-platform branch value (58/59/61) instead of a fixed 58（`6d81294`）
- ci: fix cross-target assertion (match add operand, not a literal substring)（`a1c66f8`）
- fix(cross-platform): Windows-safe binary cache filenames (no '|'/spaces) + arch-dispatched pause in thread runtime (arm64 build on macOS)（`9b4bc7f`）

### 重构与清理

- 门面工程：README 第一屏重构 + Demo GIF + 徽章 + 新手友好文件（`4d33c20`）

### 文档

- docs/ci: generic examples only (remove third-party project naming)（`1c3e264`）

### 构建与工具链

- CI/CD 加固（策略层，用户已逐项确认）：SHA 固定 + 发布人工闸门 + dependabot（`ec87a49`）
- CI/CD 权限最小化：工作流级只读，仅 release 作业持有 contents: write（`ab2106a`）

### 其它

- 规范：重申声明形态红线——不存在 let/var/const 关键字式声明（`564aa41`）
- ci: cross-platform library job (emit-lib -> -L merge -> run) on ubuntu/macos/windows（`cd35bca`）
- qkc: portable IR library variants (preprocessor-driven, no .so); -L merges IR at compile time; preprocessor directives (#if os/arch/defined/include)（`8c8d974`）
- qkc: --emit-lib/--obfuscate/-L library artifacts (manifest+shared lib), export ABI normalization, lib mode without main（`41328f8`）

## v2.0.0 - 2026-09-21

### 新功能

- qkcheck 加强：新增 QK108–QK115（7 项检查 + 1 项）并建立误报/漏报基准（`8775272`）
- qklsp + 编辑器支持：LSP（诊断/跳转/补全/悬停/大纲）+ VS Code 扩展 + tree-sitter 语法（`89d315c`）
- qkdoc + qkrepl 首发：API 文档生成 与 交互式求值（`86af00b`）
- qkcheck 首发：静态检查（未使用/不可达/遮蔽/缺返回/接口近失配/void 误用）（`8c8139a`）
- 语言正典化收尾：类型在前/type/impl/space 唯一形态 + 接口按实现严格 + 多项修复（`b52fa9c`）
- qkc 剥离 cleg：渲染/qkstyle/auto/tick 全部移出（qkc 零 cleg 专属代码）；保留通用能力 qkfile/qksignal_emit + 新增 HashTable.keys()（`bc9ee9f`）
- 事件接口族对齐：emit 短名→on 方法名自动映射（clicked→onClicked）+ 可选实现（Partial）（`63b3a78`）
- README：官方项目区新增 cleg GUI 框架（`cc027bc`）
- Operation 协议接口族 + 运算符重载 + 多 impl + 空自我实现 + dynamic（`a12b0d0`）
- README：官方项目区新增 GL / Vulkan 图形库（library FFI 声明集）（`2607273`）
- actions 重试语义：qkexec/qkexecv/qkhttp_get/post 支持可选 retries（默认 1 次重试）（`3d3c1db`）

### 修复

- 测试：槽位预解析语义回归（13 个作用域边界用例 + 槽位名字校验）（`41efa7e`）
- 修复发布脚本：输出目录为绝对路径时 qkc 产物被写错位置（`a4d310f`）
- 发布产物：三平台二进制 + 版本注入 + 变更日志自动化 + 修 macOS/Windows 构建（`1b4fd15`）
- 仓库卫生：ffi_win.inc 标记 linguist-generated（修正 GitHub 语言条误判 C++ 57%）+ 停止跟踪构建产物 compiler/qkc（`c863ef6`）
- 修复循环内重复声明丢失赋值（用户 HTML 剥段 bug 根因）（`a2e1f35`）
- program 声明位置放宽：前后均可（用户确认无先后关系）；删过时测试 + 新回归（`415f9c2`）

### 性能

- 工具链性能优化：qkcheck −80% · qkdoc −60% · qkfmt −29% · 解释器 −31~42% · LSP 补全 −94%（`28c8f75`）
- String 内置文本处理方法集 + 极限优化（`ca39112`）

### 重构与清理

- 仓库卫生：忽略 go test 生成的 *.test 二进制（`7383a9f`）
- 清理：移除临时实验基准文件（BenchmarkPhaseTypecheck）（`d74c002`）

### 文档

- qkdoc 补宏文档（#macro 在解析前被切出 AST，需显式收录）（`5a948bf`）
- README：官方项目 actions 库说明升级为两级结构（`415b3b0`）
- README：官方项目区同步 actions 库（改名）（`8a5b0a4`）

### 构建与工具链

- qkc CLI 完善（-c/-o/--emit-ir/--version/-h）+ CI 工作流（构建/测试/双路径对比）（`af47f02`）
- 递归 import 合并（库文件内的 import 也加载，visited 防环）——cleg→json 传递依赖可用（`812d99e`）
- darwin 无 cgo 构建通过（Mach-O）+ libHandle/libObj 共享文件整理（`d67d0d4`）
- 全跨系统：Windows 交叉构建通过（FFI 自研 ABI wrapper + GDI 屏显）+ macOS 帧输出构建 + 平台 tags 化（`035d486`）

### 其它

- qkcheck 误报/漏报统计（多轮取 P99）+ 跨系统门禁（`4a8eaf1`）
- 记录「何时托管给 C 库」的实测判据 + 前端成本拆分基准（`1d25b45`）
- qkc 第 3 轮：HashTable + List<String>（String.split）+ 指针（T&/new）lowering（compare.sh 全一致）（`5979a7c`）
- qkc：接收者按类型 + interface{}(tAny) lowering（22/22 双路径逐字节一致）（`4900998`）
- 接口签名的接收者也按类型识别（isRecvParam），与 impl 侧/运行期统一（`a884036`）
- 接收者识别改为**按类型**（Self 或该 impl 类型基名），不再按形参名 self（`12cc4cd`）
- qkc：参数全按引用（含 copyd 深拷贝）+ 签名调用 memorize + 匿名类型 + merge 0..4 实参（20/20 双路径一致）（`795fdaa`）
- 编译器：函数重载 + String/List 内建方法 + FFI 链接名映射与 long/pointer（13/13 双路径逐字节一致）（`8f5d0e5`）
- 参数全按引用传递（copyd 才传时拷贝）+ 匿名 struct/interface 可作类型标注（`303362c`）
- qkc lowering 阶段 C+D：泛型单态化 + 接口 vtable 分发 + library FFI + taskm（与解释器逐字节对齐）（`331b4c8`）
- qkc lowering 阶段 A+B：语言基础 + struct/impl/space/运算符重载（与解释器逐字节对齐）（`168aa94`）
- style 表口 QSS 归一（qkstyle_* 旧键回退 bg/size/font）+ 自动重绘模型（qkcleg_auto/tick）（`2c294d6`）
- FreeType 精细渲染：灰度 alpha 混合抗锯齿（默认 on，blendPixel 边缘混合/255 直写快路径）+ hint 模式接口（default/none/light）+ CJK rune 路径统一（`cabc3d0`）
- CJK：字体回退链按字符集选择（Noto Sans CJK/WenQuanYi/Source Han…）+ rune 级光栅（中文实测完整渲染）（`6681f88`）
- org 迁移：URL 全量 QuarkLangCommunity（`fdc90cf`）
- 主仓更名 QuarkLangQkc（语言 + qkc 编译器）；微服务分工：quark/qkd/qkm 独立仓库（`83ccef5`）
- qkc 工具集：quark 调试模式（--bp file:line,... 断点 + c/n/p var/q 交互）（`4114591`）
- raw string 对齐 Go：多行 + 内嵌 \\r 丢弃（`e1ab77b`）
- 字符串纯文本语义：未知转义保留字面（"C:\Windows\Fonts" 原样）；双引号字面 + 反引号含换行（`1551e2d`）
- break：ForStmt 也计入 loopDepth（循环外 break 报错含 for）（`edde563`）
- break 语句：while/for 循环体内跳出（哨兵 + 捕获）+ 循环外编译报错（`02613d9`）
- 组件事件接口族：ClegClickable/ClegCheckable/ClegEditable/ClegValueable/ClegSelectable/ClegItemable/ClegCellable/ClegCloseable/ClegActionable（内置 dynamic 协议）（`dcf252a`）
- expand interface X; 语句语义：块内语句 + dynamic 前缀 + 组合递归收集 + 聚合 conformance（多 impl 满足）（`9298234`）
- Qt 对齐第一波：qkstyle 原语读取 + 圆角 + 列表选中态 + 布局/信号（`2075a7d`）
- QSS 风格 style 层：qkcleg_roundrect（圆角矩形原语）+ 字体链文本 + cleg qss 空间（`4c1257f`）
- 字体回退链（style["font"]）：FreeType 光栅 + 链探测 + 5x7 兜底（跨系统）（`c4927aa`）
- setStyle(string) = 解析 JSON 字符串本体；@styleConfigure(file) = 读文件内容字符串投递给 setStyle（`4c7900a`）
- 重载 fallback：单函数参数类型错误还原 cannot assign 报错（行级落盘）（`304df99`）
- 重载无匹配报错兼容（单函数保留 cannot assign/count 文案）（`9cfe8a3`）
- 四件套：setStyle(file)/反引号字符串/@styleConfigure 签名/file 类型/函数重载（`18fd104`）
- cleg 跨系统屏幕宿主：X11 实测通过 + GDI 分支 + 帧输出回退（`69c8d20`）
- cleg 全面框架（Qt 组件全集 Cleg 化）+ 语言支撑：else if / 标量 toString / 接口动态派发（`bc98064`）
- cleg GUI 框架 v1 + 接口动态派发 + Operation/多 impl 基建（`764c56f`）
- 系统库 FFI（library 声明）：跨系统 dlopen/LoadLibrary + libffi 调用（`9d59426`）
- json.loads 键规整：反序列化的 HashTable 键走 hashKey（与 put/get 一致，loads 后 get("name") 可查）（`656ee0f`）
- json 官方库：qkjson_dumps/qkjson_loads 原语（encoding/json，键还原原生值）+ any 可赋具体类型（`831def9`）
- String 内置文本处理方法集（UTF-8 安全）（`c305785`）
- actions 库网络层原语：qkhttp_get / qkhttp_post（10s 超时、8MiB 响应上限、非2xx 报错）+ typecheck 白名单与参数校验（`63a5ee8`）
- 清除历史残留保留字：in / out 不再是关键字（`4162660`）

## v1.0.0

### 新功能

- 官方库 system 落地：qkexec/qkexecv/qkpopen 原语 + import 合并修正 + CLI 支持 import + README 官方项目区（`0863eb7`）
- SYNTAX.md：整理当前实现语法的逐项清单（供逐项复核）（`aa8ab47`）
- cgen：签名 f(args) @instance(...) 给出明确诊断（编译器后端待实现，不再落为 undefined parse 错误）（`54eee11`）
- 编译器支持 program main;/library;/lib; 顶层声明（main 函数自动挂载为 @main 入口；library 编译为库产物）——program main; 程序编译运行 42 ✓（`bb73a56`）
- README 重写为 v2：特性全（struct/impl/泛型/宏/taskm 编译支持/delete-clear/函数引用/pointer-new）、性能表、跨系统/优化旗标、bench 复现（`c81ac91`）
- 编译器工程化：宏系统共享（复用解释器 token 级宏展开，Lex→SplitMacroDefs→ExpandMacros→joinTokens 重组，一套逻辑两处用）；泛型类型擦除策略（语法完整接受+调用编译）；新增 struct/impl/泛型/try-catch/字符串拼接 编译回归测试；go.mod 链接 quarklang 模块（`a47f5e5`）
- 诚实基准修复：BenchmarkCompile 每次不同源码（绕过缓存，真实编译 12.6µs）、新增 Go 大样本 GC 对照；此前单次采样波动（fib 47-92ms/循环 172-289ms）修正为 20x 稳定值：1M 循环 170ms、fib24 49ms（`9b63dba`）
- 指针与堆申请：pointer 修饰（等价 T&）、new <type>[size] 堆上申请（block 分配，非法大小 badAlloc + log 记录）、空指针解引用 NullPointerError（安全性）；新增 TestPointerAndNew/TestNewBadAlloc；34 测试全绿（`19e6f28`）
- 编译器：多函数支持（参数 i32 寄存器、return、函数调用、递归）——qkc 原生 fib(30)=832040 与 C 持平（均 4ms，LLVM 同优化后端）（`71059a2`）
- xmind 第 6 批：delete variable; 回收内存（List 直接回收、用户类型调 __delete__()、block 消除日志 + clear() 差分清理）；program 预制宏必须写在程序末尾（节点接入顺序）；保留全部原有语法（xmind 未提者不删）；32 测试全绿（`bdc14e8`）
- xmind 第 2 批：type interface<T>{...}(Name)/type struct<T>{...}(Name) 实名语法（type 前缀+括号名）、space {...}(Type) 自我实现空间（匿名实现、泛型目标自动引入类型参数）、.{...} 字面量绑定有名结构体（字段匹配+实例替换）；30 测试全绿（`0181d22`）
- 质量收口：gofmt 统一风格；修复一元负号/字符串转义 bug；支持 % 取模；新增 llvm-as 语法校验与 3 项快速测试（6 项全绿）（`a958822`）
- 实现用户自定义 struct/impl/interface 与用户自定义 Sign（35 项测试全绿）（`a6ac6bc`）

### 修复

- 宏语法修正：#return 是宏返回值指令（用户逐项纠错）（`392ec99`）
- 全量 bug 检查修复：宏编译分支、sum 32 位 wrap、done 布尔语义、IR 缓存版本化（`78e4162`）
- .qlib 库链路三处真 bug 修复 + 往返回归测试（`2ee28ec`）
- 编译器 struct+impl：impl 方法解析（fn name(self T) ret {body}）、receiver struct 值参数、self.a extractvalue、p.sum() 方法调用（load struct + call @Type_method）——Point p=.{3,5} p.sum()=8；修复 impl 循环吞 } bug（`7607c42`）
- 编译器对齐批量：String 拼接（ql_strcat 运行时）、try/catch（除零检查跳 catch + after 块，10/0→caught、10/2→5 正确）、clear/compact（编译=free）；修复 catch 参数空格与 try 内字符串预注册（`edaf460`）
- 编译器 taskm 后端（B 方案：用户态 API + 并行载体）：spawn/merge(单 int 参)/block 编译为 pthread_create/join（LLVM 编号修复：gettimeofday/realloc/join 均分配返回值寄存器）；8 路 x 1e7 并发求和编译路径 1ms vs Go 6ms（快 6 倍，多核满血）；pthread 为 POSIX 载体，Windows 分支待补；done/channel 待补（`58d515c`）
- 编译路径 P99 实测：clock() 内置（编译路径 gettimeofday 微秒）+ List 下标赋值 l[i]=v + 下标 i64 扩展（toI64 修正 SSA 顺序）；fib(20)x1000：qkc 产物 P50=14µs/P99=27µs vs C P50=16µs/P99=25µs——延迟分布与 C 同级（`a68d3de`）
- 位级置换求和：生成器序列周期检测（≤65536 项）→ 周期位统计预计算 + 乘加聚合（sum(n%3,0,1e9)=999999999 仅 2ms，O(P+bits)）；线性闭式加末项校验（修复 n%3 三点差分巧合误判与 int32 环绕误闭式）（`f27ead2`）
- sum 修复：线性探测无条件（3 参默认 step=1 也闭式）——10 亿项求和解释器 2ms（O(1) 乘加闭式），Go 循环 223ms（快 111 倍），C 同级（clang 亦闭式化）（`82e79aa`）
- 跨系统修正：默认 -O3 便携基线（IR 与平台无关，目标平台 clang/llc 生成原生）；-march=native 仅显式 QUARK_CFLAGS 本机极限用（产物仅当前 CPU）；README 跨系统/优化旗标说明（`0857d97`）
- 编译器对齐：List.append(v)（realloc 扩容）；表达式语句回溯修复（l.size(); 完整解析，曾因预读 first 而错）；README 依赖说明（Go≥1.21 零第三方依赖、LLVM 工具链、QUARK_CACHE）（`1139894`）
- 增量编译：解释器 Compile 进程内缓存（源码 sha256 键，重复编译/import 跳过全编译）+ errf 参数修复（`c6a292f`）
- v2 核心：删除 FuncBuffer 语言面（内部 execCtx）；函数必带返回类型、return/log 结束并返回、out 移除；try/catch(名字 类型)；.{...} 匿名结构体；签名 @mb(prefix) → mb.call(prefix)(.{in,out}) 记忆化（原函数在 prefix.fn）；taskm.spawn()→pid / merge(pid,fn,args) / block→void / done=线程空闲；v2 演示全链路通过（旧测试待迁移）（`34fd1cc`）

### 性能

- 编译产物极限优化：纯函数 memory(none) + 线程池唤醒风暴清除 + done 原子化（`0e93509`）
- 编译产物优化：函数属性（norecurse/mustprogress/noundef）+ 默认 thinLTO 降级回退（`fbb3eb6`）
- 极限优化第三轮：调用热路径 -15%（fib24 15.1→12.8ms，累计 27.9→12.8ms）（`9ea1a1d`）
- 宏系统重构 + 签名去固定化 + 解释器热路径优化（`c797674`）
- 编译产物优化：尾调用（return f(x) → LLVM tail call，尾递归栈 O(1)）——gcd(1e9,1) 1 亿次尾递归不爆栈且瞬间完成；fib 非尾递归不受影响（fib35=9227465 正确）（`6ab09d8`）
- 编译器微优化：newReg/newBlock 用 strconv.AppendInt 替代 fmt.Sprintf（编译自身 ~2-5%）（`0add092`）
- 优化实验：编译期 CallExpr.FnIdx 预解析（walk AST）——实测拖慢（分支+回退开销），回滚 eval 分支恢复 fib24 27.6ms/1M 139ms；walk/FnIdx 结构保留（编译期零运行时开销）（`a3183b0`）
- 编译路径突破：List.append 几何增长（realloc len*2，amortized O(1)，消除 O(n²) 逐项拷贝——100 万次 append 编译验证）；发现 PGO 优化点（fib35 profile-use -29%，QUARK_CFLAGS 可传 -fprofile-use）（`0091d35`）
- 性能突破：局部变量入线性槽位（declare 免 map 哈希，get/set 全线性）+ int 比较快路径（< <= > >= 直通 BoolV）——1M 循环 165->135.8ms(-18%)、fib24 47.8->26.5ms(全程-45%)；bench 验证 loopSrc 确为 100 万次（135ns/迭代≈9 指令，解释器真实水平）（`1cf718d`）
- 性能极限：DynamicStackAndHeap 预声明移入全局作用域（outer 链一次可见，每函数调用的 declare+map 分配消除）——fib24 47.8→28.8ms(-40%)、100K 调用 43.9→31.2ms(-29%)、1M 循环 171→165ms（`2b00148`）
- 极限优化：evalExpr int 算术快路径（两侧 IntV 直通，免 binOp 分发；*2^n→shl 内联）——1M 循环 193→172ms(-11%)、fib24 50→47ms(-6%)（`335c362`）
- 函数引用+位运算+sum 优化：① type <类型> 名字; 通用类型别名、function<ret,p1,...> 函数类型、函数作为值传递调用（apply(square,7)=49）② 位移运算符 << >>（解释器+编译器 shl/ashr）+ 乘 2^n→左移优化 ③ sum(generate,begin,stop[,step])：线性生成器算术闭式 O(1)（100 万项瞬间），非线性循环（`4adf907`）
- 函数引用 + sum 优化：① type <类型> 名字; 通用类型别名（type function<int,int> IntFn;）；function<ret,p1,...> 函数类型；函数作为值传递与调用（apply(square,7)=49）② sum(generate,begin,stop[,step]) 内置——线性生成器（二阶差分恒定）算术闭式 O(1) 乘加（100 万项瞬间），非线性退化循环（`2555682`）
- 编译产物极限优化：qkc 默认 -O3 -march=native（fib30 4ms→2ms，2 倍；QUARK_CFLAGS 可覆盖，缓存键含 flags）；IR 形态：单赋值 int 变量 SSA 直通（免 alloca/load/store，tide.ll alloca 大减）（`7bce9c6`）
- 编译器简单优化：递归常量折叠（1+2*3→7 编译期算掉，IR 更小编译更快；除零/取模零不折叠保语义）（`71845b6`）
- 性能优化：参数线性槽位（参数名函数级缓存零分配 + 值复用 ctx.Args，fib 类函数免除 scope map 哈希）——fib24 53→46ms（-14%）；34 测试全绿（`b22f87d`）
- 编译器对齐解释器第一批：List<int> 字面量/下标访问/delete（LLVM malloc/free）；潮汐实证 1 亿轮 QK=1ms 与 C 一致（同 LLVM 优化器，分配消除行为相同），Rust 29ms / Go GC 92ms；Clear 只清空空闲 block（delete 入队不清数据，保障数据安全）（`4a9a2e5`）
- 性能：① 内存线性分配（block 占用度最小优先堆，内部空闲空间复用；内存日志废弃）② execCtx/args 切片对象池（函数调用堆分配大减）③ scope 懒 map ④ 性能基准（Compile 94→20µs -79%，循环 -5%）⑤ 流类型恢复 istream/ostream/ifstream/ofstream/iofstream + 方法 ⑥ import 二进制库互调（.qlib gob 导出/导入、同目录 .qk/.qlib 搜索、program 末尾规则）（`7e5fadc`）

### 重构与清理

- Value 标签联合体重构（方案 A）：24B GC 安全布局，循环 -37% / 调用 -22%（`5d3eb10`）
- xmind 第 3 批：globalMemory.clear() 按修改日志直接清理整个内存、GlobalMemory.mode(DynamicStackAndHeap) 实验栈堆标志（预声明常量）；30 测试全绿（`d19c349`）
- 泛型系统（struct<T>/impl<T>、指针 T&、null）+ Copyd 运行时包装与 .ptr() + 真实内存系统（block 分配/脏标记/协程回收/compact 实际清理）——41 项测试全绿（`0a51347`）

### 文档

- SYNTAX.md 同步 M4/M7（`1a999cb`）
- 全量删兼容：函数关键字统一 fn（func 废弃，token/cgen/解释器/测试/examples/README 全部迁移，无 func 兼容分支）——双路径全测试绿（`0fb028c`）
- README：依赖说明（Go≥1.21 零第三方依赖、LLVM 工具链、QUARK_CACHE、可选对比工具）（`55a41cc`）
- compiler 并入 main：移除 /compiler/ 忽略与 worktree；README Get Started 写清楚解释器与编译器两套用法（`f21128d`）
- README：compiler 分支采用 LLVM 后端（`b2b7d26`）
- main：.gitignore 忽略 compiler worktree；README 同步泛型/指针/Copyd/内存系统与 compiler v0.2（`0713b50`）
- 移除误提交的 docs worktree gitlink（`ec505ae`）
- 移除误提交的 docs worktree gitlink（docs 独立分支，不并入 main）（`b7b9e39`）
- README：taskm 全局变量（.调用）与协程 API 更新（`a162e6e`）
- main：docs 以 worktree 形式回到工作区（仍不并入 main）（`f4dc7c4`）
- docs 移出 main：独立 docs 分支维护（不并入 main）（`80fe879`）
- 添加 MIT 许可证并完善 README（`4fee133`）
- 初始化 QuarkLang：语言设计文档 + Go 解释器 v0.1（`ac72397`）

### 构建与工具链

- xmind 第 4-5 批：预处理 #when(compile/explain)（解释器=explain 态、run 别名兼容）、#insert(名字) 直插捕获、#exec 别名；接口 Self 类型（tTypeVar 通配）、expand interface 组合（递归方法展开+一致性检查）、any=空接口；30 测试全绿（`de4202b`）
- xmind 旨意第一批：① 变量修饰 const/copyd（<decor> <type> <name>，const 禁重赋值、copyd 传时复制）；② 泛型函数 func<T,...>（类型变量 tTypeVar + 调用点实参推断）；③ thread 类：taskm.spawn() 返回 thread（thread.merge/pid/talk），thread 类型注解；④ 内建类型 type-first 声明；30 项测试全绿（`81a2894`）
- 宏系统落地：macro {模式}{主体}（模式禁嵌套大括号）、token 级展开（... 捕获到语句边界）、动态预处理 #when(compile|run)/#insert(#ast(...))/#execute/#error、预制宏 program main;/program library;（运行时报 cannot run a library）/import/pub；30 项测试全绿（+7 宏测试）（`411cb0e`）
- 测试套件迁移到 v2：23 项全绿（return/log 结束/try/catch/.{} 字面量/@mb() 记忆化签名/taskm 线程/struct/泛型/指针/Copyd/内存/严格检查），void=空接口别名，race 干净（`f833e3d`）
- 新模型语法层+运行时：函数必须声明返回类型、return 返回真实值并结束、log 记录并结束函数、out 移除（明确报错）、try/catch(名字 类型)、.{...} 匿名结构体字面量（字段名允许关键字 in/out）；旧测试待迁移（`cfa8aa8`）
- compiler：变量（alloca/store/load）与控制流（if/else/while，基本块+br）、比较/布尔/&&/||/!、格式串按实际类型生成；7 项测试全绿（`9364487`）
- 质量收口：gofmt 统一风格 + 6 项快速单元测试（除零/取模/字符串拼接/布尔/负号/HashTable），47 项测试全绿（`0501f8c`）
- compiler v0.2：后端改为 LLVM IR（不经 C 转译）——qkc 生成 IR，clang 产出原生，-run 编译执行，lli 全链路测试（`87d1566`）
- compiler v0.2：QuarkLang → C 转译器骨架（qkc，hello 子集 + 算术，测试全绿）（`868816a`）

### 其它

- 安全加固：递归深度/通道容量/new 上限 + 嵌套泛型 >> + 回滚不实陈述（`f185fef`）
- 恢复 #insert/#execute/#ast 宏指令（用户纠正：本来就是正确语法）（`34a4631`）
- engineVersion 4：缓存键代次跟进（宏 #return 语法）（`ba463df`）
- 回退直接分配实验（实测 12.8→18.2ms，GC 压力 > 原子池）；确认无锁 CAS 池为解释器 ctx 复用最优；补 ensureLog 防御方法（`af295c4`）
- qthreads 线程池终极形态：票号自旋锁队列（竞争路径零 syscall）（`5a1ff23`）
- GlobalMemory 统一实例点调用（剃刀：删 :: 特判）（`213a8e5`）
- 函数关键字遗患全清：func → fn 无任何残留/兼容分支（`a60e87c`）
- 同步批次：① 解释器与编译器 impl 语法一致（impl <Type> [<Iface>] { fn... }，fn 关键字，分号可选）② .{} 位置形式（.{3,5}，与编译器一致，保留 name: v 命名形式）③ examples 重写为 v2（hello/fib/struct/taskm/sum/macro/trycatch，双路径验证一致）④ bench/ go.mod 隔离（根模块不再扫 .c）⑤ cgen 函数调用字符串拼接改 strings.Builder（免中间分配）（`28d303c`）
- bench/: 跨语言对比源（C/Rust/Go/Erlang fib35 + Makefile 一键复现）（`69acc32`）
- taskm 线程池（qthreads.c POSIX 段）：固定 8 工作线程 + 任务队列（QCAP 65536），merge 投递任务由池执行——大规模并发零线程创建开销；带参 merge + done 编译验证（1/1）；Windows 分支仍每任务线程（待池化）（`9e5723d`）
- 宏快速路径：无 macro 关键字跳过 Lex（编译 18->15.6us）（`0bfe7e1`）
- 编译器对齐全部完成：泛型（func<T> 声明/调用跳过，类型擦除，id<int>(21)=42）、宏（macro 声明跳过）、Windows 线程运行时（_beginthreadex+SRWLOCK，双平台 qthreads.c）——channel/并发/impl/泛型全链路编译验证通过（`82c8775`）
- 编译器 struct：type struct 声明解析、.{v1,v2} 字面量、decl alloca {i32 x N}+字段 store、p.a 字段访问（gep+load）——Point p=.{3,5} p.a+p.b=8 编译正确（`6560ce4`）
- 编译器 taskm 完整收尾（B 方案跨系统运行时）：qthreads.c 用户态线程运行时（pthread 载体，mutex/cond 锁队列，Windows 分支待补）+ go:embed 内嵌 + qkc -run 自动链接 -pthread；spawn/merge(单参)/block/done/channel(send/recv) 全部编译可用：channel 42/7、done=1、8 路并发 2ms；thread=任务对象指针（用户态 API）（`53229e1`）
- sum 对均匀随机生成器（rand 函数引用）用位级置换期望闭式：每列 1 计数 n/2 → Σ(n/2)·2^k = n·2^30 乘加 O(1)（随机不退化；标注为均匀期望近似，精确需循环）；内置函数可作为函数引用传递（callFunc 仅对伪函数分派 builtin，不劫持用户同名方法）（`a605aa4`）
- rand() 内置（LCG 确定性伪随机）+ 编译器 sum 内联循环展开（clang -O3 闭式化线性生成器）；位级置换求和说明：线性生成器用逐位聚合闭式（O(1) 乘加，与位级重排数学等价）（`58dbe17`）
- 编译器对齐：List<int> 结构体化（{i32*,i32} ptr+len）、方法调用解析（l.size()/l.get(i)/l[i] 下标）——编译输出 3/20/30 全链路正确（`01ab1ab`）
- 尾延迟基准：P99/P999 对比（解释器零 GC block 复用 vs Go GC）；P50=12µs/P99=54µs（P99:P50≈4.4x，无 GC 停顿）（`9d3f35d`）
- 内存真实化：delete 先执行 __delete__()（若有）再加入空闲队列；block 固定 BlockSize 内部细分（占用度=内部已用/BlockSize，占用度最小优先复用）——碎片率 0.195%（趋近零）、复用率 99.96%、新 block 收敛为 2；Fragmentation() 指标 + 潮汐基准（`48045b2`）
- 杀手级场景实证：临时数据潮汐——block 线性复用 + delete 即时归队（复用率 99.5%、新 block 收敛、零 GC），比 Go GC 同规模快 5.5 倍；内存管理器加复用统计（AllocCalls/ReusedCount/NewBlocks）（`837ad9f`）
- qkc 增量编译缓存（IR+二进制两级，首次 32ms→增量 2ms）（`cabb4d0`）
- program lib; 简写（program 为语言声明语句，直解无宏开销）（`ea1dae8`）
- gitignore: 忽略 xmind 导图（存放于 design 分支）（`b8f1483`）
- 布局：compiler 文件移入 compiler/ 子目录（准备并入 main）（`7063252`）
- taskm 改为全局变量（.调用）：spawn→pid、block/merge/done(pid)、channel(n) 默认 1024；删除 yield；执行表读写锁（FIFO、读优先）；compact 无返回 + setBlock(n)（`e8d4bd0`）
- compiler 分支占位：编译器工作起点（`69e8d84`）

## 未发布（v1.0.0 之后）

### 新功能

- qklsp + 编辑器支持：LSP（诊断/跳转/补全/悬停/大纲）+ VS Code 扩展 + tree-sitter 语法（`89d315c`）
- qkdoc + qkrepl 首发：API 文档生成 与 交互式求值（`86af00b`）
- qkcheck 首发：静态检查（未使用/不可达/遮蔽/缺返回/接口近失配/void 误用）（`8c8139a`）
- 语言正典化收尾：类型在前/type/impl/space 唯一形态 + 接口按实现严格 + 多项修复（`b52fa9c`）
- qkc 剥离 cleg：渲染/qkstyle/auto/tick 全部移出（qkc 零 cleg 专属代码）；保留通用能力 qkfile/qksignal_emit + 新增 HashTable.keys()（`bc9ee9f`）
- 事件接口族对齐：emit 短名→on 方法名自动映射（clicked→onClicked）+ 可选实现（Partial）（`63b3a78`）
- README：官方项目区新增 cleg GUI 框架（`cc027bc`）
- Operation 协议接口族 + 运算符重载 + 多 impl + 空自我实现 + dynamic（`a12b0d0`）
- README：官方项目区新增 GL / Vulkan 图形库（library FFI 声明集）（`2607273`）
- actions 重试语义：qkexec/qkexecv/qkhttp_get/post 支持可选 retries（默认 1 次重试）（`3d3c1db`）

### 修复

- 仓库卫生：ffi_win.inc 标记 linguist-generated（修正 GitHub 语言条误判 C++ 57%）+ 停止跟踪构建产物 compiler/qkc（`c863ef6`）
- 修复循环内重复声明丢失赋值（用户 HTML 剥段 bug 根因）（`a2e1f35`）
- program 声明位置放宽：前后均可（用户确认无先后关系）；删过时测试 + 新回归（`415f9c2`）

### 性能

- String 内置文本处理方法集 + 极限优化（`ca39112`）

### 文档

- README：官方项目 actions 库说明升级为两级结构（`415b3b0`）
- README：官方项目区同步 actions 库（改名）（`8a5b0a4`）

### 构建与工具链

- qkc CLI 完善（-c/-o/--emit-ir/--version/-h）+ CI 工作流（构建/测试/双路径对比）（`af47f02`）
- 递归 import 合并（库文件内的 import 也加载，visited 防环）——cleg→json 传递依赖可用（`812d99e`）
- darwin 无 cgo 构建通过（Mach-O）+ libHandle/libObj 共享文件整理（`d67d0d4`）
- 全跨系统：Windows 交叉构建通过（FFI 自研 ABI wrapper + GDI 屏显）+ macOS 帧输出构建 + 平台 tags 化（`035d486`）

### 其它

- qkc 第 3 轮：HashTable + List<String>（String.split）+ 指针（T&/new）lowering（compare.sh 全一致）（`5979a7c`）
- qkc：接收者按类型 + interface{}(tAny) lowering（22/22 双路径逐字节一致）（`4900998`）
- 接口签名的接收者也按类型识别（isRecvParam），与 impl 侧/运行期统一（`a884036`）
- 接收者识别改为**按类型**（Self 或该 impl 类型基名），不再按形参名 self（`12cc4cd`）
- qkc：参数全按引用（含 copyd 深拷贝）+ 签名调用 memorize + 匿名类型 + merge 0..4 实参（20/20 双路径一致）（`795fdaa`）
- 编译器：函数重载 + String/List 内建方法 + FFI 链接名映射与 long/pointer（13/13 双路径逐字节一致）（`8f5d0e5`）
- 参数全按引用传递（copyd 才传时拷贝）+ 匿名 struct/interface 可作类型标注（`303362c`）
- qkc lowering 阶段 C+D：泛型单态化 + 接口 vtable 分发 + library FFI + taskm（与解释器逐字节对齐）（`331b4c8`）
- qkc lowering 阶段 A+B：语言基础 + struct/impl/space/运算符重载（与解释器逐字节对齐）（`168aa94`）
- style 表口 QSS 归一（qkstyle_* 旧键回退 bg/size/font）+ 自动重绘模型（qkcleg_auto/tick）（`2c294d6`）
- FreeType 精细渲染：灰度 alpha 混合抗锯齿（默认 on，blendPixel 边缘混合/255 直写快路径）+ hint 模式接口（default/none/light）+ CJK rune 路径统一（`cabc3d0`）
- CJK：字体回退链按字符集选择（Noto Sans CJK/WenQuanYi/Source Han…）+ rune 级光栅（中文实测完整渲染）（`6681f88`）
- org 迁移：URL 全量 QuarkLangCommunity（`fdc90cf`）
- 主仓更名 QuarkLangQkc（语言 + qkc 编译器）；微服务分工：quark/qkd/qkm 独立仓库（`83ccef5`）
- qkc 工具集：quark 调试模式（--bp file:line,... 断点 + c/n/p var/q 交互）（`4114591`）
- raw string 对齐 Go：多行 + 内嵌 \\r 丢弃（`e1ab77b`）
- 字符串纯文本语义：未知转义保留字面（"C:\Windows\Fonts" 原样）；双引号字面 + 反引号含换行（`1551e2d`）
- break：ForStmt 也计入 loopDepth（循环外 break 报错含 for）（`edde563`）
- break 语句：while/for 循环体内跳出（哨兵 + 捕获）+ 循环外编译报错（`02613d9`）
- 组件事件接口族：ClegClickable/ClegCheckable/ClegEditable/ClegValueable/ClegSelectable/ClegItemable/ClegCellable/ClegCloseable/ClegActionable（内置 dynamic 协议）（`dcf252a`）
- expand interface X; 语句语义：块内语句 + dynamic 前缀 + 组合递归收集 + 聚合 conformance（多 impl 满足）（`9298234`）
- Qt 对齐第一波：qkstyle 原语读取 + 圆角 + 列表选中态 + 布局/信号（`2075a7d`）
- QSS 风格 style 层：qkcleg_roundrect（圆角矩形原语）+ 字体链文本 + cleg qss 空间（`4c1257f`）
- 字体回退链（style["font"]）：FreeType 光栅 + 链探测 + 5x7 兜底（跨系统）（`c4927aa`）
- setStyle(string) = 解析 JSON 字符串本体；@styleConfigure(file) = 读文件内容字符串投递给 setStyle（`4c7900a`）
- 重载 fallback：单函数参数类型错误还原 cannot assign 报错（行级落盘）（`304df99`）
- 重载无匹配报错兼容（单函数保留 cannot assign/count 文案）（`9cfe8a3`）
- 四件套：setStyle(file)/反引号字符串/@styleConfigure 签名/file 类型/函数重载（`18fd104`）
- cleg 跨系统屏幕宿主：X11 实测通过 + GDI 分支 + 帧输出回退（`69c8d20`）
- cleg 全面框架（Qt 组件全集 Cleg 化）+ 语言支撑：else if / 标量 toString / 接口动态派发（`bc98064`）
- cleg GUI 框架 v1 + 接口动态派发 + Operation/多 impl 基建（`764c56f`）
- 系统库 FFI（library 声明）：跨系统 dlopen/LoadLibrary + libffi 调用（`9d59426`）
- json.loads 键规整：反序列化的 HashTable 键走 hashKey（与 put/get 一致，loads 后 get("name") 可查）（`656ee0f`）
- json 官方库：qkjson_dumps/qkjson_loads 原语（encoding/json，键还原原生值）+ any 可赋具体类型（`831def9`）
- String 内置文本处理方法集（UTF-8 安全）（`c305785`）
- actions 库网络层原语：qkhttp_get / qkhttp_post（10s 超时、8MiB 响应上限、非2xx 报错）+ typecheck 白名单与参数校验（`63a5ee8`）
- 清除历史残留保留字：in / out 不再是关键字（`4162660`）

