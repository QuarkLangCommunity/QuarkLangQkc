# QuarkLang — Primitives, and Everything Derived From Them

**Status.** Analysis and proposal. This document is *not* the standard. `spec/STANDARD.md` is the
owner's document and is untouched by this work: where this document proposes a change to the standard
it says so and leaves the ruling to the owner (§8).

**Analysed at** `923784b` (tip of `main` when this document was written), against
`spec/STANDARD.md` **v0.8.2** (`spec/STANDARD.md:3`), and against
`github.com/QuarkLangCommunity/QuarkLangQkparser v0.0.0-20261011044519-5316b66c4d5b`, the syntax module
both engines consume (root `go.mod`, the `require` line). The implementation moves; every `file:line` is
a claim about *that* revision.

**The instruction this document answers.** *Distil everything into a standard with very few primitives —
every feature should either be a primitive or be derived from the primitives by a stated rule, instead of
being enumerated as its own case.*

**Conventions.**

| Symbol | Meaning |
|---|---|
| `P-k` | a proposed primitive (§2.1) |
| `D-k` | a derivation rule (§2.2) |
| ✔ | read or run at the cited site |
| ⚠ | verified for some cases only, or verified from the standard rather than the code |
| ✘ | **UNVERIFIED** — marked rather than asserted |

`qkparser/…` means a path inside the pinned syntax module, whose root is the module cache directory
`$(go env GOMODCACHE)/github.com/!quark!lang!community/!quark!lang!qkparser@v0.0.0-20261011044519-5316b66c4d5b`.
It is a separate repository, so a reader cannot resolve those paths from this checkout alone; that is
stated here once instead of at every citation.

**How the counts were obtained.** Every count in this document is the output of one command, and the
command is given with the count so it can be re-run. A count that is a *list* is given as the list, so
the reader can count it again rather than trust a number.

---

## 1. Inventory — the primitives as they stand

### 1.0 Summary

| Kind | Standard | Implementation site | Cases enumerated now |
|---|---|---|---|
| storage atoms | §3.1 (`:49-69`) | `internal/lang/typecheck.go:43-44,85,88-117` | **2** spellings |
| type views | §3.2 (`:71-96`) | `internal/lang/typecheck.go:16-45,121-131,183-190,355-400` | **28** `tKind`s; **13** standard rows |
| operators | §5 (`:164-183`) | `qkparser/token.go:45-68`, `qkparser/parser.go:1319-1332`, `internal/lang/typecheck.go:597-625` | **21** tokens, **10** levels, **13** methods |
| statements & declarations | §2 (`:33-45`), §7.1 (`:201-209`) | `qkparser/ast.go:130-232,355-371`, `qkparser/parser.go:324,976,1211` | **12** statements, **16** expressions, **16** keywords |
| directives | §7.2 (`:211-232`) | `qkparser/preproc.go`, `qkparser/macro.go` | **15** implemented spellings, **8** standard spellings |
| permission surface | §4 (`:143-162`) | `qkparser/refperm.go:10-25`, `internal/lang/refperm.go:26-90`, `internal/lang/typecheck.go:1373,1376,1411,1423,1426,1665,1674` | **7** perms × **4** scopes = **28**; **7** enforcement sites, **4** rules |
| memory operations | §10 (`:261-283`) | `internal/lang/eval.go:229-350,723-731,2386-2405` | **4** standard primitives, **4** implemented, **1** missing on each side |
| error constructs | §8 (`:234-243`) | `internal/lang/eval.go:26,2933`, `internal/lang/typecheck.go:479`, `internal/i18n/source/*.tsv` | **17** reason prefixes; **2** Go error types |

### 1.1 Storage atoms — 2 spellings

The standard states one storage atom and one aggregate family (`spec/STANDARD.md:51-57`): `bit` (values
`0`/`1`) and `bits<N>` for `N ≥ 1`, with `bits<1>` *being* `bit`. Layout is `⌈N/8⌉` bytes, byte-aligned,
bits past `N` zero (`:68-69`).

Implementation ✔ — the family is real and both engines have it:

| Site | What it is |
|---|---|
| `internal/lang/typecheck.go:43` | `tBits // raw bits: bit (width 1) and bits<N> (spec §3.1)` |
| `internal/lang/typecheck.go:55` | `Bits int // tBits: the width in bits (>= 1); 0 for every other kind` |
| `internal/lang/typecheck.go:88,91` | `isRawBits`, `rawBitsOf(n)` — one constructor, width-parameterised |
| `internal/lang/typecheck.go:85` | `bitsWidthLimit = 32` — the implementation's ceiling (§11.2 states 32) |
| `internal/lang/typecheck.go:97-117` | `bitsWidthOf`, `rawBitsNameWidth` — width parsing and validation |
| `internal/lang/typecheck.go:94` | `isBitsTypeName`: `s == "bit" \|\| strings.HasPrefix(s, "bits<")` |

Count: **2 spellings, 1 constructor.** `bit` is not a second kind of storage — `rawBitsOf(1)` is the same
value as the `bit` spelling (`:113-115`), exactly as the standard says `bits<1>` *is* `bit`.

Measured ✔ (`bit`/`bits<N>` accepted by both engines; the historical `byte`/`bytes<N>` spellings are
refused by both — they were renamed in standard 0.7, `spec/STANDARD.md:345`):

| Program fragment | interpreter | compiler |
|---|---|---|
| `uchar u = uchar(1);` | exit 0 | exit 0 |
| `bits<4> q = bits<4>(0);` | exit 0 | exit 0 |
| `byte b = byte(1);` | exit 1 `CompileError: unknown type "byte"` | exit 1, same message |
| `bytes<4> m = …` | exit 1 `unknown type "bytes<4>"` | exit 1, same message |

### 1.2 Type views — 28 `tKind`s against 13 standard rows

The standard gives one table of views (`spec/STANDARD.md:73-80`): `bool`, `char`, `uchar`,
`int32`/`int64`/`int128`/`int256`, `float32`/`float64`, `&T`/`*T`. Read as *names* rather than rows that
is **11** (`1+1+1+4+2+2`) in 6 rows; adding the two degenerate interpretations registered by the type
system itself (`void`, `any`, `:89-90`) makes **13**. The implementation's own enumeration is larger — a 28-entry `tKind` enum:

`internal/lang/typecheck.go:16-45` — `tInt tFloat tString tBool tNil tAny tList tHashTable tFuncBuffer
tIOStream tInputStream tOutputStream tChannel tTask tMemorize tMemory tFunc tStruct tTaskm tPtr tCopyd
tNull tTypeVar tLib tInterface tFile tBits tUchar` (**28**).

Three further enumerations of the *same* fact exist in the same file ✔:

| Site | Enumeration |
|---|---|
| `internal/lang/typecheck.go:121-131` | `viewWidth`: `bool`→1, `uchar`→8, `int`→32; **3** cases, everything else 0 |
| `internal/lang/typecheck.go:183-190` | `kindName`: a name→kind map for error text |
| `internal/lang/typecheck.go:355-401` | `parseTypeStr`: the name→`Type` decision — ≥13 `case` arms from `:363`, one per accepted spelling (`int/long/char`, `float/double`, `bool`, `uchar`, `bit`, `bits<N>`, `String`, `void`, `FuncBuffer`, `IOStream`, the three input-stream spellings, the three output-stream spellings) |
| `internal/lang/typecheck.go:134,137` | `isIntView` = `tInt|tUchar`; `isSignedView` = `tInt` |

Measured ✔ — which standard rows actually exist: `uchar` and `bits<N>` are implemented in both engines;
`int32`, `char`, `float64` are **not** (both engines: `CompileError: unknown type "int32"`). That is
consistent with §11.1, which lists only `bit`, `bits<N>` (1≤N≤32) and `uchar` as implemented in both
engines (`spec/STANDARD.md:293-295`) — so §3.2's table is partly **aspirational**, and §11.1 is what says
which rows are real. The gap is declared, not hidden; see §1.2b for the rows that are *not* declared.

#### 1.2b Two rows the status section does not declare

`char` and `double` are the standard's legacy spellings (`spec/STANDARD.md:255-257`: "Legacy spellings
(`long`, `double`, and `char` used as a narrow integer) are implementation compatibility, not standard").
Measured ✔, the two engines do not accept the same language for them:

| Program | interpreter | compiler |
|---|---|---|
| `char c = 65; io.println(c);` | exit 0, prints `65` | exit 1, `char` variable type not lowered |
| `char c = char(65);` | exit 0 | exit 1, same refusal |
| `fn f(char x) int {…}` | exit 0 | exit 1, `char` parameter type not lowered |
| `double d = 1;` | exit 0, prints `1` | exit 1, `CompileError: unknown type "double"` |
| `long n = 5;` | exit 0, prints `5` | exit 0, prints `5` — agrees |

The compiler's refusal is a **hard error, not a silent difference**, so §1.4 (`:26-28`) is honoured in
letter. But the *accepted language* differs, and the corpus cannot cover these rows: no case under
`compiler/testdata/cases/` or `cases_run/` declares a `char` or `double` variable, which is why
`compare.sh` still prints `all identical`. §11.1 does not list `char`/`double` as implemented in both
engines, §11.2 does not list them as limits, and §11.3's three divergences (`:321-329`) do not mention
them. This is a **fourth class of divergence that the status section does not declare** — reported here,
not resolved.

### 1.3 Operators — 21 tokens, 10 levels, 13 methods

Standard §5 (`:164-183`) names the operator vocabulary, the precedence order, `===`, and the operation
methods. Implementation ✔:

| Site | Count | Cases |
|---|---|---|
| `qkparser/token.go:45-68` | **21** | `TAssign TPlus TMinus TStar TSlash TPercent TEq TEqStrict TNe TLt TLe TGt TGe TAnd TOr TShl TShr TCaret TTilde TPipe TAmper` |
| `qkparser/parser.go:1319-1332` | **10** levels | see the ladder below |
| `internal/lang/typecheck.go:597-625` | **13** | `__add__ __sub__ __mul__ __div__ __mod__ __eq__ __ne__ __lt__ __le__ __gt__ __ge__ __neg__` |
| `internal/lang/typecheck.go:2322,3175,3199,3203`; `internal/lang/eval.go:731` | **4** | `__sort__`, `__literal__`, `__element__`, `__delete__` |
| `qkparser/token.go:54`, `:342-347` | **1** | `TEqStrict` (`===`), lexed as its own token |

The precedence ladder as implemented, loosest first (`qkparser/parser.go:1319-1332`) ✔:

```
parseOr    → ||                                    (loosest)
parseAnd   → &&
parseCmp   → == != < <= > >=
parseBitOr → |
parseBitXor→ ^
parseBitAnd→ &
parseAdd   → + -
parseShift → << >>
parseMul   → * / %
parseUnary → - * & ! ~                            (tightest)
```

**Measured divergence from the standard's stated order.** §5 (`:172-174`) says: "Precedence, loosest
first: `||`, `&&`, `== != < <= > >=`, `|`, `^`, `&`, `<< >>`, `+ -`, `* / %`". Read as a list, shifts are
*looser* than additive — C's grouping. The implementation puts `parseShift` **inside** `parseAdd`
(`:1330`), so shifts bind *tighter* than additive — Go's grouping. Measured ✔, both engines agreeing
with each other and disagreeing with the standard:

| Expression | Both engines | Standard's order |
|---|---|---|
| `1 + 2 << 3` | **17** (`1 + (2<<3)`) | **24** (`(1+2)<<3`) |
| `1 << 2 + 3` | **7** (`(1<<2)+3`) | **32** (`1<<(2+3)`) |
| `10 - 2 >> 1` | **9** (`10-(2>>1)`) | **4** (`(10-2)>>1`) |
| `1 & 2 == 0` | `true` — `(1&2)==0`, so `&` is tighter than `==` | same — agrees with §5 |
| `1 \| 2 ^ 3` | `1` — `1\|(2^3)`, so `^` is tighter than `\|` | same — agrees with §5 |

The standard's own worked consequence (`a & b == c` is `(a & b) == c`, `:173-174`) **does** hold ✔ — the
divergence is confined to the shift level. This is a case the parity gate structurally cannot catch: both
engines agree, so `compare.sh` is green while both disagree with the document they implement.

### 1.4 Statements and declarations — 12 + 16 nodes, 16 keywords

Standard §2 (`:33-45`) fixes the declaration form (`<modifier> <type> <name> [= init];`), the modifier
set (`const`, `copyd`), `type` aliases, named types, `impl`, and comments. §7.1 (`:201-209`) adds
`program`.

Implementation ✔ — `qkparser/ast.go`, counted by the types implementing the sealed marker interfaces
(`Stmt` at `:130`, `Expr` at `:133`):

- **12 statement nodes**: `BreakStmt ExprStmt LogStmt DeleteStmt TryStmt ReturnStmt IfStmt WhileStmt
  ForStmt ForCStmt DeclStmt AssignStmt` (`:154,222-232`).
- **16 expression nodes**: `IntLit FloatLit StrLit BoolLit NullLit Ident ListLit StructLit NewExpr BinOp
  IncExpr UnOp CallExpr MemberExpr ScopeCall IndexExpr` (`:355-371`).
- **16 reserved keywords**: `fn struct impl interface return if else while for true false log null try
  catch macro` (`qkparser/token.go:115-130`).
- Declaration/modifier spellings recognised as *contextual identifiers*, not keywords ✔:
  `type` (`qkparser/parser.go:324`), `const`/`copyd` (`:449`, `:1211`), `delete` (`:976`), the
  `&perm scope` reference form (`:1177-1201`), plus `program`, `import`, `pub`, `library` in the
  top-level dispatch — the parser's own message enumerates seven of them
  (`expected a top-level declaration (fn/struct/impl/interface/program/import/pub)`).

`try`/`catch` (`:128-129`, `TryStmt` at `ast.go:225`) is a statement the standard has no place for:
§1.3 and §8 (`:236`) say there is no `try`/`catch`, and §11.3 (`:326-328`) records that it still exists
and is scheduled for removal. Measured ✔ — `try { … } catch (void e) { … }` compiles and runs in both
engines today, so the *removal is not done*.

### 1.5 Directives — 15 implemented spellings against 8 standard ones

Standard §7.2 (`:211-232`) names 8 spellings in 4 rows: `#expand`, `#run`, the conditional family
(`#ifdef` `#ifndef` `#if` `#ifn` `#endif`), and `macro x($a $b) { … }`.

Implementation ✔ — the directive surface is **three layers**, which the standard's single table does not
distinguish:

| Layer | Site | Spellings | Count |
|---|---|---|---|
| line level (decides which lines exist, before lexing) | `qkparser/preproc.go:12-19` (the documented set), dispatch at `:113,143-192,193` | `#define #undef #ifdef #ifndef #if #elif #else #endif #include #error` | **10** |
| macro-body level (token splicing) | `qkparser/macro.go:221` `macroCommand` | `#return #error #insert #execute` (+ `#ast` inside `#insert`, `:330`) | **4** |
| definition level | `qkparser/macro.go:9-46` | `#macro name (params) { body }` | **1** |

Total implemented: **15** spellings. The standard's conditional family is 5 spellings; the implementation
implements **4** of them.

Because there is **one** preprocessor in the syntax layer and both engines call it (`preproc.go:6-10`),
"the two engines agree" is structural for this layer rather than a coincidence: any line that reaches the
preprocessor is decided once. The table below therefore separates what was **run** (`probe`, both engines
invoked) from what was **read** (the shared implementation's dispatch, which no engine can bypass):

Measured ✔ (`probe`), with both engines agreeing on every line:

| Spelling | Source | Result in both engines |
|---|---|---|
| `#if` + `#endif`, `#ifdef`, `#ifndef`, `#else` | probe | work — the guarded branch is the one that survives |
| `#define NAME`, `#undef NAME` | probe | work — the name is defined / no longer defined |
| `#define NAME 0` (a value after the name) | probe | **silently ignored**: the name stays undefined and nothing is reported — the guard `reDefineNm.MatchString(nm)` at `preproc.go:164` rejects `"X 0"` with no `else` |
| `#elif` | read | the same selection code as `#else`: `preproc.go:129-140` (`#elif`) and `:143-160` (`#else`/`#endif`) |
| `#include "f.qk"` | read | `preproc.go:193-221`, include depth cap 16 at `:101` |
| `#error <msg>` | read | `preproc.go:185-187` |
| `#ifn` | probe | **not implemented** — `#ifn` matches neither the `if|elif` regex (`preproc.go:335`) nor the dispatch switch (`:143-192`), so it opens no group and the following `#endif` is refused: `#endif without a matching #if` |
| `#run main` | probe | **not implemented** — `ParseError: expected a top-level declaration … got unknown token` |
| `#expand main in program` | probe | **not implemented** — same `ParseError` |
| `macro x($a $b) { … }` (the standard's spelling) | probe | **not implemented** — `ParseError: expected a top-level declaration … got unknown token` at line 1 |
| `#macro pick (x) { #return 7 }` | probe | works, prints `7` |
| `#when (run) { … }` (deleted) | probe | refused by name: `macro pick failed to expand: line 2: #when is not a QuarkLang preprocessor directive` |

Three of the standard's spellings (`#expand`, `#run`, `#ifn`) exist in **neither** engine, and the macro
spelling differs by one character (`macro` vs `#macro`). The `#expand`/`#run`/`#ifn` gaps are already
recorded in `docs/ARCHITECTURE.md` §6.7–§6.8, which this document does not re-derive; §5 below uses them
only as a *precedent* for how a spelling leaves the language.

### 1.6 The permission surface — 7 × 4 = 28, from the checker

Counted from the code, not remembered ✔. The permission a declaration carries is a **stack spelling**
enumerated literally in the syntax module, and the scope is a second enumeration:

```go
// qkparser/refperm.go:10-16
func ValidRefPerm(s string) bool {
	switch s {
	case "r", "w", "m", "rw", "rm", "wm", "rwm":
		return true
	}
	return false
}
// qkparser/refperm.go:19-25
func ValidRefScope(s string) bool {
	switch s { case "u", "f", "a", "t": return true }
	return false
}
```

| Quantity | Value | Site |
|---|---|---|
| permission spellings | **7** (`r w m rw rm wm rwm`) | `qkparser/refperm.go:12` |
| scopes | **4** (`u f a t`) | `qkparser/refperm.go:21` |
| reference types | **28** | `spec/STANDARD.md:150`; produced at `internal/lang/typecheck.go:1306-1310` |
| bare-`T&` default | `rwm` / `u` | `internal/lang/refperm.go:38-39`, applied at `typecheck.go:1156-1160,1308` |
| deciding predicates | **2** | `refAllows` = `strings.Contains` (`refperm.go:46`); `refPermCovers` = subset test over `{r,w,m}` (`:51-58`) |
| scope predicate | **1** ordinal | `refScopeRank`: `u`=3, `t`=2, `a`=1, `f`=0 (`:63-75`) |
| enforcement sites | **7** | `typecheck.go:1373,1376,1411,1423,1426,1665,1674` |

The 7 enforcement sites collapse to **4 distinct rules**:

| Rule | Predicate | Sites |
|---|---|---|
| writing needs `w` | `refAllows(perm,"w")` | `typecheck.go:1411` |
| moving needs `m` | `refAllows(perm,"m")` | `typecheck.go:1665` |
| reading needs `r` | `refAllows(perm,"r")` | `typecheck.go:1674` |
| assign a reference: `src ⊇ dst` **and** `rank(src) ≥ rank(dst)` | `refPermCovers` + `refScopeRank` | `typecheck.go:1373+1376` (declaration init), `1423+1426` (assignment) |

This matches §4's enforcement paragraph (`:160-162`) rule for rule ✔. **The algebra is already smaller
than the enumeration**: `refAllows` is a membership test and `refPermCovers` is `src ⊇ dst`, so the
checker never compares spellings — it compares *sets*. The 7 spellings are a hard-coded expansion of
2³ − 1 (every non-empty subset of `{r,w,m}`; the empty set is unrepresentable).

### 1.7 Memory operations — 4 planned, 4 implemented, not the same 4

Standard §10 (`:261-283`) gives a table of **4** primitives — `delete x;`, `memory.setBlock(n)`,
`memory.compact()`, `memory.Fragmentation()` — and names a fifth thing in prose, `clear` ("the payload is
**not** wiped — wiping is `clear`'s job", `:271`).

Implementation ✔:

| Site | What it is |
|---|---|
| `internal/lang/eval.go:229-350` | `MemoryManager`: `Alloc(size, owner)` `:244`, `MarkDirty` `:272`, `ReclaimTask(pid)` `:281`, `Compact()` `:294`, `Fragmentation()` `:309`, `BlockCount()` `:326`, `Delete(id)` `:333`, `Clear()` `:344` |
| `internal/lang/eval.go:723-731` | the `delete` statement: runs `__delete__()` if the type declares it, then pushes onto the free queue — data kept, wiping is `clear`'s job (`:723` comment) |
| `internal/lang/eval.go:2386,2398,2405` | the `memory` object's methods: `clear`, `compact`, `setBlock` |
| `internal/lang/typecheck.go:2477,2487,2492` | the checker's mirror of the same three |

So the *block policy* of §10 is implemented, in the interpreter, as a logical layer. The two sides do not
agree on membership:

| Primitive | In §10 | Implemented | Measured result |
|---|---|---|---|
| `delete x;` | ✔ `:271` | interpreter ✔; compiler only for `List` variables | `int x = 1; delete x;` runs interpreted; compiler: `delete "x"` not supported |
| `memory.setBlock(n)` | ✔ `:272` | interpreter ✔ | interpreted runs; compiler: "method setBlock … interpreter-only" |
| `memory.compact()` | ✔ `:273` | interpreter ✔ | interpreted runs; compiler: interpreter-only |
| `memory.Fragmentation()` | ✔ `:274` | **✘ neither engine** | both: `TypeError: method "Fragmentation" does not exist on memory` |
| `memory.clear` | prose only `:271` | interpreter ✔ | (not in §10's table) |

**Measured:** §10's own scope sentence (`:266-267`: "This section describes the contract of the
**compiled artifact**") is not met — the compiler lowers none of the four, and `Fragmentation` exists
nowhere. This is not declared in §11.2 or §11.3.

### 1.8 Error constructs — 17 reason prefixes, 2 Go types

Standard §8 (`:234-243`) names 5 runtime errors (`DivisionByZeroError`, `IndexOutOfBoundsError`,
`IndexOutOfRangeError`, `NullPointerError`, `PanicError`), `panic(msg)` as the core primitive, and
`Option<T>`/`Result<T, E>` as **library** types.

Implementation ✔ — two Go carrier types (`RunError` at `internal/lang/eval.go:26`, `CheckError` at
`internal/lang/typecheck.go:479`), the `panic` builtin at `internal/lang/eval.go:2933` (checked at
`typecheck.go:2940`), and the reason carried as a *catalog id*, not a Go type. The reason prefixes
measured in `internal/i18n/source/*.tsv` (`grep -rhoE '[A-Za-z]+Error:' internal/i18n/source/ | sort | uniq -c`):

| Prefix | Occurrences | Prefix | Occurrences |
|---|---|---|---|
| `TypeError:` | 540 | `IndexOutOfRangeError:` | 6 |
| `CompileError:` | 129 | `FFIError:` | 6 |
| `IOError:` | 33 | `RunError:` | 3 |
| `ParseError:` | 30 | `PanicError:` | 3 |
| `RuntimeError:` | 12 | `IndexOutOfBoundsError:` | 3 |
| `LibraryError:` | 12 | `ImportError:` | 3 |
| `DivisionByZeroError:` | 9 | `HTTPError:` | 3 |
| `StringIndexOutOfBoundsError:` | 6 | `JSONError:` | 6 |
| `NullPointerError:` | 6 | | |

**17** distinct prefixes. `StackOverflowError` is raised in code (`internal/lang/limits.go:42`) and named
in §11.2, but no line of the catalog carries that prefix — ✘ **UNVERIFIED** whether it reaches the user
through the catalog or is composed in the compiler.

Two of the 17 name **different conditions**, and the names stay distinct: `IndexOutOfBoundsError` is a
container subscript — a `List` read or write outside `0 … size-1` (`internal/lang/value.go:462,479`,
catalog row `field.indexoutofboundserror_index_d_out_range_0`) — while `IndexOutOfRangeError` is a bit
subscript outside the width (`internal/lang/eval.go:3527`) and a reference moved past the end
(`internal/i18n/source/runtime.tsv:61`). The compiler's runtime carries one helper per condition
(`compiler/internal/cgen/cgen.go:5404,5421`). When this document was written §8 named only
`IndexOutOfRangeError`; the standard now defines both and says where each is raised (§8, 0.8.6), and
`internal/lang/error_names_test.go` pins one case of each and asserts the other name is absent. Measured
interpreter message for a list index: `IndexOutOfBoundsError: index 5 out of range [0,1)`; for a bit
index: `IndexOutOfRangeError: bit index 9 is out of range for bits<8> (0..7)`.

`Option`/`Result` are correctly library, not core ✔: `stdlib/option.qk`, `stdlib/result.qk` (with
`pub fn<T, E> ok(...)` at `stdlib/result.qk:12`), and §8's list of methods matches those files. No core
implementation of either was found.

---

## 2. The minimal set, earned

The primitive set below is stated as **8 primitives + 1 restriction**. It is not a wish list: §2.2 shows
for each inventory row above whether it is a primitive or a derivation, and names the site that makes the
derivation checkable.

### 2.1 The primitives

| | Primitive | What it is | Standard today |
|---|---|---|---|
| `P-1` | **storage** | a bit sequence of constant width `N`; `bit` is `N = 1` | §3.1 `bit` + `bits<N>` (**2** spellings → **1** constructor) |
| `P-2` | **view** | an interpretation of a fixed width laid over storage: same storage, never a wrapper, never adds a field | §3.2's table (**13** rows → **1** constructor + a naming rule) |
| `P-3` | **address** | a value denoting storage; the only thing that moves | §4's `*T` and `p++` |
| `P-4` | **permission** *(restriction on P-3)* | a subset of `{r,w,m}` plus a lifetime rank, attached to a name for storage | §4 (**28** reference types → **3** atoms + **1** ordinal) |
| `P-5` | **block** | the allocation unit: a size, an occupancy, an owner; release is explicit | §10 (**4** primitives → **2**) |
| `P-6` | **bind and apply** | bind a name to a value; apply a function to arguments | §2's declaration form; `T(x)` in §3.4 |
| `P-7` | **abort** | terminate carrying a reason id and a position | §8's `panic(msg)`; the 4 named errors become reason literals |
| `P-8` | **preprocess-time selection** | one conditional over a boolean symbol store, evaluated before lexing | §7.2's conditional family (**5** spellings → **1**) |
| `P-9` | **inclusion** | splice another file's lines in | §7.2 does not list `#include` at all, though both engines implement it (`preproc.go:193`) |

### 2.2 The derivation table

Every row is a feature that exists today, with the rule that produces it from §2.1, and the site a reader
can check the rule against. "Current" quotes the standard's or the implementation's own enumeration.

| # | Current feature | Primitive or rule | Checkable site |
|---|---|---|---|
| D-1 | `bit` (width 1) | **P-1** at `N=1`; not a separate kind | `typecheck.go:113-115` (`rawBitsNameWidth` returns 1), §3.1 `:57` (`bits<1>` *is* `bit`) |
| D-2 | `bits<N>` | **P-1** with the width as a parameter | `typecheck.go:91` `rawBitsOf(n)` |
| D-3 | `bool` | **P-2**: view `(N=1, two values: 0 = false, 1 = true, nothing else)` | `typecheck.go:123-124` `viewWidth(tBool)==1`; §3.2 `:75` |
| D-4 | `uchar` | **P-2**: view `(N=8, unsigned)` | `typecheck.go:125-126`; §3.2 `:77` |
| D-5 | `int` | **P-2** + **default rule**: `int` *is* `int32` = `bits<32>` | `typecheck.go:127-128` `viewWidth(tInt)==32` with the comment "int is int32 in this implementation"; §3.2 `:89` "`int` means `int32`" |
| D-6 | `int32/64/128/256`, `float32/64`, `char` | **P-2** rows: the name is a *function* of interpretation and width — §3.2's own sentence "A view's name states its own width" (`:82-83`) — so the table is data, not cases | §3.2's table `:73-80`; the rows are not yet in code (§11.1) |
| D-7 | `void`, `any` | **P-2**'s two degenerate rows: the empty and the full interpretation, "registered by the type system itself" | §3.2 `:89-90`; code has `tAny` (`typecheck.go:22`) and maps `void`→`tAnyV` (`:391`) |
| D-8 | the conversion call `T(x)` | **P-6** (application of a view constructor) + one rule: *if the operand is raw bits of the target width, the bits are reinterpreted; if the operand is itself a view, the value is computed* | `typecheck.go:2677-2679` (`isRawBits(src) && src.Bits == dst.Bits`) + `:101` width refusal; §3.4's three rows `:118-130` are the two branches plus the width rule |
| D-9 | narrowing "keeps the low bits" | **P-2** + **P-6**: the target view's width decides the mask | `typecheck.go:139-149` `storeWidth`; §3.4 `:129` |
| D-10 | `&T` (reference) | **P-3** restricted by **P-4**: an address that carries a permission and a lifetime rank | `typecheck.go:1306-1310`; §4 `:146-147` |
| D-11 | the 28 reference types | **P-4** applied twice: `2^{r,w,m} − ∅ = 7` spellings × `4` ranks. The implementation *already* treats them as sets | `qkparser/refperm.go:12,21` (the two enumerations); `refperm.go:46,51` (membership and subset tests) |
| D-12 | bare `T&` | **P-4**'s top element and default | `internal/lang/refperm.go:38-39` |
| D-13 | "writing needs `w`, reading `r`, moving `m`" | **P-4**'s three membership rules | `typecheck.go:1411,1674,1665` |
| D-14 | "not to a wider scope", "not widened" | **P-4**'s one ordinal rule applied twice | `typecheck.go:1376,1426` (`refScopeRank`) |
| D-15 | `*T` and `p++` | **P-3** without **P-4**'s restriction | §4 `:155-158` |
| D-16 | `===` storage identity | **derived from P-3**: identity of the storage an address denotes. Not an operator *kind* — which is why "not overloadable" is a consequence, not a rule: identity is not a method on a type | `qkparser/token.go:54` (its own token); §5 `:176-179` |
| D-17 | `==` value comparison | **P-2/P-6** + **one dispatch rule**: an operator on a struct resolves to the method whose name the operator's canonical spelling generates | `typecheck.go:597-625` (operator→name), `:639-650` (the interface mirror) |
| D-18 | the **13** operation methods | **D-17**: 13 names generated from the operator table by one naming rule | `typecheck.go:597-617` |
| D-19 | `[a,b,c]`, `for x : c`, `*h` | **D-17** applied to protocols: resolved by method name (`__literal__`, `__element__`, `size`/`get`) | `typecheck.go:3175,3199,3203`; §6 `:190-194` |
| D-20 | `bit`/`bits<N>` admission rules | **P-1** + one rule: raw bits admit exactly the operations that need no arithmetic — i.e. the width-preserving, bit-local ones (`& \| ^ ~ << >>`, `== !=`, `b[i]`, `===`) | The refusal *is* one id per family, each raised at one site: `typecheck.go:1931` (`MsgTypecheckTypeerrorRawBitsAdmitOnlyB`), so the four rows of §3.3 `:102-107` are one rule with four spellings |
| D-21 | shift keeps the left operand's width | **P-1**: the result width is the left operand's | §3.3 `:109`; `compiler/internal/cgen/cgen.go:4383-4385` `emitBitsShift` |
| D-22 | `b[i]` read/write | **P-1** + **P-3**: an address at bit offset `i` | §3.3 `:105`; constant-index refusal `typecheck.go:175-181` |
| D-23 | the 12 statements | **P-6** + control: binding (`DeclStmt`), assignment (`AssignStmt`), application as statement (`ExprStmt`), and the branch/loop forms | `qkparser/ast.go:222-232` |
| D-24 | `const`, `copyd` | **P-6** modifiers, not declarations — the standard already derives these | §2 `:36-37`; `qkparser/parser.go:449,1211` |
| D-25 | `type` alias, `type struct`, `type interface`, `impl` | **P-6** + **P-2**: a named binding of a type expression; `impl` binds methods by name | §2 `:38-42`; `qkparser/parser.go:324` |
| D-26 | `program` / `main` | **P-6** + a *capability declaration*, never mandatory | §7.1 `:201-209` |
| D-27 | `delete x;` | **P-5**'s release; `__delete__()` is **D-17** (a method resolved by name) before the release | `internal/lang/eval.go:723-731` |
| D-28 | `memory.setBlock(n)` | **P-5**'s granularity knob | `internal/lang/eval.go:2405`; §10 `:272` |
| D-29 | `memory.compact()` | **derived from P-5**: reclaim the blocks no live value occupies — a function of occupancy, not a new capability | `internal/lang/eval.go:294` `Compact()`; §10 `:273` |
| D-30 | `memory.Fragmentation()` | **derived from P-5**: a pure function of block occupancy (unused space ÷ live space) | `internal/lang/eval.go:309` exists as a Go method; §10 `:274`. The *language* binding is missing in both engines (§1.7) |
| D-31 | `memory.clear` | **derived**: overwrite storage with zero — an ordinary **P-1** store, not a memory primitive | `internal/lang/eval.go:2386`; §10 `:271` names it in prose |
| D-32 | `panic(msg)` | **P-7** | `internal/lang/eval.go:2933`; §8 `:241` |
| D-33 | `DivisionByZeroError`, `NullPointerError`, `IndexOutOfRangeError`, `StackOverflowError`, … | **P-7** with a *reason literal*: the 17 prefixes are reasons, not constructs | `internal/i18n/source/*.tsv`; §8 `:242-243` |
| D-34 | `IndexOutOfBoundsError` vs `IndexOutOfRangeError` | **D-33**: two reason literals for one condition — a reduction, not a derivation | both measured in §1.8 |
| D-35 | `Option<T>`, `Result<T, E>` | **not primitives and not core**: they are library types over **P-2** + **P-5** | `stdlib/option.qk`, `stdlib/result.qk`; §8 `:237-239`, §9 `:251` |
| D-36 | `#if` / `#elif` / `#else` / `#endif` | **P-8**: one conditional, four spellings of its own syntax | `qkparser/preproc.go:113-159` |
| D-37 | `#ifdef X` | **P-8** with a derived test: `#if defined(X)` | `preproc.go:176-178` vs `:401-404` — both read the same `c.Defines` map |
| D-38 | `#ifndef X` | **P-8**: `#if !defined(X)` | `preproc.go:178-180` (`cond = !cond`) |
| D-39 | `#define` / `#undef` | **P-8**'s symbol store = *assignment to a boolean*; `#undef` = assign false. That the store holds no values is *proved by the code*: `#define X 0` is silently ignored (§1.5), because the value has nowhere to go | `preproc.go:162-175` (`c.Defines[nm] = true`, `delete(c.Defines, nm)`); reader at `:404`; measured in §1.5 |
| D-40 | the `defined(NAME)` / `os(…)` / `arch(…)` tests and `! && \|\|` | **P-8**'s expression algebra: one boolean expression evaluated before lexing | `preproc.go:335-341` (the regexes), `:352-404` (`evalAnd`/`evalUnary`/literals) |
| D-41 | the two engines differing only by symbol | **P-8** + one rule: the engine is a *predefined symbol* (`COMPILE`/`EXPLAIN`), decided before lexing, so no directive branches on the engine | `preproc.go:8-10`; §7.2's example `:222-230` |
| D-42 | `#include` | **P-9** | `preproc.go:193-221` |
| D-43 | `#error` | **P-7** at preprocess time: abort with a reason | `preproc.go:185-187` |
| D-44 | `#macro` + `#return`/`#error`/`#insert`/`#execute` | **not derived**: token-level substitution is its own mechanism. **P-6** at token level is the closest honest description, and it is weaker than the thing itself | `qkparser/macro.go:9-46,221,240-328`; §7.2 `:218` |
| D-45 | `#expand` / `#run` | **not derived from P-1…P-9**: naming a *node* and placing or executing it needs a node namespace that does not exist | `docs/ARCHITECTURE.md` §6.8; measured in §1.5 |

### 2.3 What cannot be derived (the honest residue)

Four things resist, and saying so is part of the proposal:

1. **The operator vocabulary itself** (D-17). `+` cannot be derived from `&`. What *is* derived is
   everything *around* it: the 13 method names (one naming rule), `===` (identity, D-16), and the
   admission rules (D-20). Reducing the operator count is a semantic change, not a distillation, and is
   not proposed.
2. **`#include`** (D-42) — splicing another file's lines is not a boolean operation.
3. **Macro token substitution** (D-44) — a token-level rewrite with parameter substitution; it is
   parameterised (`macro.go:231-253`), not generated.
4. **Node-level `#expand`/`#run`** (D-45) — unimplemented, and not derivable from anything above without
   first deciding what a "node" is (§8, Q-3).

---

## 3. Count: before and after

### 3.1 Per kind

| Kind | Before (measured) | After (proposed) | Rules stated for what stopped being primitive |
|---|---|---|---|
| storage atoms | **2** spellings | **1** (`P-1`) | `bit` = `P-1(N=1)` — D-1. *No change in meaning; the standard already says `bits<1>` is `bit`.* |
| type views | **28** `tKind`s / **13** standard rows | **1** constructor (`P-2`) + a naming rule | D-3…D-7; the rows become *data* (interpretation, width) |
| operators | **21** tokens, **10** levels, **13** methods | **13** methods → **1** rule (D-17); `===` → **0** (D-16) | tokens and levels stay: irreducible vocabulary (§2.3) |
| statements/declarations | **12** + **16** nodes, **16** keywords | **1** mechanism (`P-6`) + control forms | D-23…D-26 |
| directives | **15** implemented spellings, **8** standard spellings | **3** (`P-8` conditional, `P-9` include, `P-7`-at-preprocess = `#error`) + macro (not derived) | D-36…D-43; `#ifdef`/`#ifndef`/`#ifn` → `#if defined()` (D-37/D-38), `#define`/`#undef` → one store (D-39), the 5 conditional spellings → 1 construct (D-36) |
| permission surface | **7** perms × **4** scopes = **28** | **3** atoms + **1** ordinal (`P-4`) | D-11: the 7 spellings are the 7 non-empty subsets; the checker already tests sets, never spellings |
| memory operations | **4** planned / **4** implemented / **1** missing each way | **2** (`P-5`: release + block) | D-29 `compact`, D-30 `Fragmentation`, D-31 `clear` become derived |
| error constructs | **17** reason prefixes, **2** carrier types | **1** primitive (`P-7`) + reason literals | D-33; D-34 collapses two prefixes into one |

Totals. Counting the "Before" column as components rather than as a single figure, so the arithmetic can
be checked: storage **2**; views **28**; operators **21 + 10 + 13 = 44**; statements and declarations
**12 + 16 + 16 = 44**; directives **15**; permissions **28**; memory **5** (`delete`, `setBlock`,
`compact`, `Fragmentation`, `clear`); errors **17**. That is **183** enumerated cases across the eight
kinds, covered after the change by **8 primitives + 1 restriction** plus the residue of §2.3.

### 3.2 Reductions that are only renames — stated, not counted

The instruction is explicit that a rename must be declared rather than counted as a reduction. Three
candidates in this list are **renames**, and the count above does not credit them:

1. **`byte`/`bytes<N>` → `bit`/`bits<N>`** — already done in standard 0.7 (`spec/STANDARD.md:345`). It
   changed the unit from bytes to bits and made each view's name state its width; the number of storage
   *constructors* stayed 1. Not a reduction.
2. **`bits<N>` as "derived from `bit`"** (D-2) — `bits<N>` is `bit` with a width parameter, which is what
   the standard already says. Calling it a derivation of `bit` adds no economy: the constructor was
   always width-parameterised (`rawBitsOf(n)`, `typecheck.go:91`). It is restatement.
3. **`#ifdef` → `#if defined(X)`** (D-37) — removes **one keyword from the language**, which is a real
   reduction in *surface to implement identically in two engines*, but it removes **no concept**: the
   symbol store, the test and the selection all remain. Counted as **1 fewer spelling**, not as a
   smaller primitive set.

And one candidate is **not** a reduction and is not proposed:

4. **Collapsing `&T` and `*T` into one constructor.** §4 (`:155`) forbids reading them as "two spellings
   of one", and the implementation enforces the split (`isRefKind`, `internal/lang/refperm.go:43`).
   §2.1 models a reference as **`P-3` + `P-4`** — an address carrying a permission and a rank — which
   *keeps* the distinction as a restriction instead of erasing it. Whether that model is faithful is a
   design question for the owner, not a counting result: see Q-1.

---

## 4. The permission surface

This is the largest single enumeration in the language — **28** reference types named by the standard
(`spec/STANDARD.md:150`), **7 + 4** spellings enumerated literally in the parser
(`qkparser/refperm.go:12,21`), and 7 enforcement sites in the checker (§1.6).

### 4.1 What the checker actually enforces

Measured from the code, and it is **less** than the 28 names suggest — 4 rules over 2 predicates:

```
permission p ⊆ {r,w,m}          refAllows(p, "r"|"w"|"m")   = need ∈ p        refperm.go:46
                                refPermCovers(src, dst)     = src ⊇ dst       refperm.go:51-58
lifetime  rank(u)=3 > rank(t)=2 > rank(a)=1 > rank(f)=0     refScopeRank      refperm.go:63-75

R1  write through a reference            needs w ∈ p             typecheck.go:1411
R2  read through a reference             needs r ∈ p             typecheck.go:1674
R3  move a reference (p++)               needs m ∈ p             typecheck.go:1665
R4  store a reference in a reference     src ⊇ dst ∧ rank(src) ≥ rank(dst)
                                                                typecheck.go:1373+1376, 1423+1426
```

No fifth predicate exists: `grep -n "refPerm\|refScope" internal/lang/*.go` returns exactly these sites
plus the two declarations (§1.6).

### 4.2 The smallest algebra that covers it

**Two sorts, four operations.**

- A **permission set** `p ⊆ {r,w,m}` — 8 values, of which the language can spell 7 (the empty set has no
  spelling, `qkparser/refperm.go:12`).
- A **lifetime rank** `n ∈ {0,1,2,3}` with the names `f < a < t < u` — one ordinal, not a lattice, because
  `refScopeRank` is the only consumer and it only ever compares (`typecheck.go:1376,1426`).
- `∈` (R1–R3), `⊇` (R4a), `≥` (R4b).

`P-4` is that pair. Every one of the 28 reference types is `(p, n)` for a spelled `p` and a named `n`.

### 4.3 Which current kinds become compositions

| Today | Becomes | Why it is safe |
|---|---|---|
| `r`, `w`, `m`, `rw`, `rm`, `wm`, `rwm` (7) | the 7 non-empty subsets of `{r,w,m}` | the checker never compares the spelling — `refAllows` is `strings.Contains` and `refPermCovers` is a subset test (`refperm.go:46,51`) |
| `u`, `f`, `a`, `t` (4) | 4 names for one ordinal | `refScopeRank` maps them to `3,2,1,0` and nothing else reads the name except error text (`refScopeName`, `:78-89`) |
| bare `T&` | `(rwm, u)` — the top element | `refDefaultPerm`/`refDefaultScope` (`refperm.go:38-39`) *are* exactly that pair |
| "a more restrictive reference may not be widened" | `src ⊇ dst` | R4a |
| "a reference may not be stored in a wider-scoped variable" | `rank(src) ≥ rank(dst)` | R4b |

### 4.4 What resists reduction, and why

1. **The empty permission set.** 8 subsets exist; the language spells 7. Reducing "7 to 8 by derivation"
   would require an *addition* (a spelling for a reference that may do nothing), not a reduction. The
   asymmetry is unexplained in §4 and is not resolvable by counting — see Q-2.
2. **`m` doing two jobs.** §4 (`:155-158`) says a reference "never moves" and that moving is what makes a
   pointer a pointer; the checker says a reference with `m` **may** be moved (R3 at `typecheck.go:1665`,
   and the interpreter's own message for a missing `m` is `++ needs a reference (permission m)`). So the
   `m` permission is *already* the movability bit, and "reference vs pointer" is a restriction on `P-3`
   rather than a second kind. Whether that is the intended reading is a design question — Q-1.
3. **Where the rank comes from.** `refScopeRank` hard-codes `3,2,1,0`; nothing in the standard says what
   an "area" is or how a rank is computed for a value that is not a declaration. The algebra covers what
   is *enforced*; it does not cover what §4 *means* by `a` (area) — the standard never defines it
   (`spec/STANDARD.md:150` names it in one line). Q-2.

---

## 5. The directive surface

### 5.1 The `#when` precedent, as a method

`#when` is the worked example the instruction points at, and the shape of its removal is the argument for
every other reduction in this section. From `git log -S'#when'` and commit `40ad832`
(`lang+compiler: one shared preprocessor, and #when is gone`):

| Step | What was done | Evidence |
|---|---|---|
| 1 | recognise it as a **duplicate of the `#if` family**: "the implementation's own spelling for an operation-time switch the standard never defines" | `40ad832` message |
| 2 | **delete the branch** in the shared expander, not add a feature flag | `40ad832`; `qkparser/macro.go:262-266` keeps the *refusal* |
| 3 | **make the refusal identical in both engines** — a `#` spelling that is neither a macro command nor a line-level directive is refused **by name** | `qkparser/macro.go:260-267`, message `#when is not a QuarkLang preprocessor directive` |
| 4 | **keep the refusal as a test**, with the measured divergence in the file | `compiler/testdata/when_rejected.qk.rejected`; `compiler/macros_test.go` |
| 5 | **migrate the users** and say so | `40ad832` message lists `examples/macro.qk:2`, the macro tests, the stress generator, `SYNTAX.md`, both READMEs, three i18n entries |
| 6 | **state what is left over** instead of inventing it | `docs/ARCHITECTURE.md` §6.7 (ruling) and §6.8 (`#expand`/`#run`/`#ifn` still missing) |

Measured today ✔: `#when` is refused by both engines with the same message at the same line (§1.5), and
`#ifdef`/`#if`/`#elif`/`#else`/`#endif` all work in both. The step that made this cheap is step 3: the
refusal lives in **one** shared place, so "both engines implement it identically" is a property of the
architecture rather than a second implementation.

### 5.2 The smallest form both engines can implement identically

Apply the precedent's step-1 question — *is this spelling derivable from one already there?* — to every
directive that exists:

| Spelling | Derivable? | Rule | Verdict |
|---|---|---|---|
| `#ifdef X` | ✔ | `#if defined(X)` — both read the same `c.Defines` map (`preproc.go:176` vs `:404`) | **derived**, remove the spelling |
| `#ifndef X` | ✔ | `#if !defined(X)` (`preproc.go:178`) | **derived**, remove |
| `#if` / `#elif` / `#else` / `#endif` | ✘ | the selection construct itself | **primitive** (`P-8`) |
| `#ifn` | ✔ (if it means "not") | `#if !(…)` | **derived**, and it is *unimplemented*, so removing it costs nothing: the generated status report already leaves it undefined (Q-4) |
| `#define` / `#undef` | ✔ | assignment to a boolean in one symbol store: `#undef X` ≡ `X := false` | **one construct** with two spellings (`P-8`'s store) |
| `#include "f.qk"` | ✘ | line splicing from another file | **primitive** (`P-9`) |
| `#error <msg>` | ✔ | `P-7` (abort) at preprocess time | **derived** from the abort primitive |
| `#macro` + `#return`/`#error`/`#insert`/`#execute` | ✘ | token substitution | **separate mechanism** (§2.3); the standard's spelling (`macro x($a $b)`) is not implemented (§1.5) |
| `#run` / `#expand` | ✘ | needs a node namespace | **not implementable before Q-3 is answered** |

So the smallest directive surface is **three primitives**: one conditional (`P-8`), one inclusion
(`P-9`), one abort-at-preprocess-time (`P-7`) — plus the macro mechanism, which is not a directive but a
rewrite stage, and which the standard states in a row of the same table (`spec/STANDARD.md:218`).

Two consequences worth stating plainly:

- The **"one preprocessor for both engines" architecture is already the distillation**. `preproc.go:6-10`
  says it: one implementation in the syntax layer, no mode parameter, no per-engine branch, the engines
  differing only in a predefined symbol. Any directive reduction is therefore a *deletion in one place*,
  which is why §5.3's order is cheap.
- **`#ifn` can be removed for free.** It is in §7.2's table and in the macro expander's
  `lineDirective` list (`qkparser/macro.go:212`), and no layer implements it (§1.5). Removing it from
  both lists is a correction of a factual error, which §12 (`:333`) permits at any time.

### 5.3 What resists

The macro mechanism (D-44) and `#expand`/`#run` (D-45). Both need a decision the standard has not made —
see Q-3 and Q-5.

---

## 6. The memory surface

### 6.1 Does §10's semantics allow "release + block granularity"?

§10 states four primitives and one prose-mentioned operation (`:269-274`), and its own scope sentence
(`:266-267`). Read against §1.7's measurements:

| §10 primitive | Reducible to release + block? | Why |
|---|---|---|
| `delete x;` | **is** the release | `internal/lang/eval.go:723-731`; runs `__delete__()` then frees the block |
| `memory.setBlock(n)` | **is** the block granularity | `internal/lang/eval.go:2405`; §10 `:272` defines it as exactly "the unit the runtime allocates and reuses" |
| `memory.compact()` | ✔ derived | reclaiming blocks nobody occupies is a function of occupancy — `MemoryManager.Compact()` (`eval.go:294`) is a sweep, and `ReclaimTask(pid)` (`:281`) is the same policy on a task boundary (`:280` "When a task ends, the blocks it owns become reclaimable") |
| `memory.Fragmentation()` | ✔ derived | a *measurement*: unused space inside live blocks ÷ live blocks. `eval.go:309` is a pure Go method over the same state; a language-level observation point, not a capability |
| `memory.clear` (prose, `:271`) | ✔ derived | overwriting storage with zero is an ordinary `P-1` store — the `clear`/`delete` split exists only because §10 separates "free" from "wipe" |

**So yes, the semantics allow it**: §10's normative content is one allocation policy (a block, reused,
least-occupied-first, `:276-280`) plus one release operation. The other three names are operations on
that state — a sweep, a measurement, and a wipe.

### 6.2 What the reduction would look like

**Two primitives**: `delete` (release) and `setBlock` (granularity), with `compact`, `Fragmentation` and
`clear` derived as above. This is exactly the shape the top-level instruction asks for, and the
implementation already has it in Go: `MemoryManager` is 120 lines (`eval.go:229-350`) with eight methods,
of which only `Alloc` + `Delete` touch the policy and the rest are bookkeeping, sweeps and measurements.

### 6.3 What would be lost — and what is lost already

Three things are genuinely at risk in the reduction, and all three must be stated rather than assumed:

1. **`Fragmentation()` as a language-level promise.** §10 calls it "(inspection)". Removing it from the
   standard removes the only *stated* way for a program to observe the block policy. It is derived, but it
   is not *replaceable*: a program cannot compute block occupancy from anything else in the surface.
   Recommendation: keep it as a derived-but-normative observation point, or move it to the toolchain.
2. **`delete`'s "does not wipe" guarantee** (`:271`). This is a *security-relevant* sentence: "a released
   block never hands another value's bytes to the next user". It survives the reduction only if the
   release primitive keeps it; if `delete` were redefined as release-plus-wipe, the guarantee becomes a
   property of `clear` instead. Either choice is expressible; the standard must say which — Q-6.
3. **The compiled artifact's contract.** §10's scope sentence says the section describes the compiled
   artifact, and §1.7 measured that the compiler lowers **none** of the four primitives (with a hard
   error each time, so §1.4 holds). Distilling §10 to two primitives does not fix that; it makes the gap
   *smaller and therefore louder*: two primitives to lower instead of four. What is lost by not fixing it
   is the whole section — a reader can write a conforming program that no compiler accepts.

---

## 7. What would leave the standard, and what each removal costs

### 7.0 The separation rule

Every item below is classified as one of two things, because §12 (`:331-336`) permits only one of them
without the owner's ruling:

- **Pure restatement** — the change removes a duplication, or corrects a factual error, and no program
  changes meaning. §12 `:333` allows this at any time ("Corrections of factual errors … are allowed at
  any time").
- **Semantic change** — some program changes meaning, or surface is added. Waits for the review date and
  a version bump (§12 `:336`).

### 7.1 §11 Implementation status — **pure restatement**, and it should leave

§11 says of itself: "This section is **not normative**" (`:287`). It is a status report inside a frozen
specification, and §1.2b/§1.7 measured that it is **incomplete**: `char`/`double` diverge between the
engines, `memory.Fragmentation` exists in neither, the compiler lowers none of §10 — none of which §11
records.

| What leaves | What it costs |
|---|---|
| §11.1–§11.3 as *specification* text | the only written record of the three declared divergences (macro spelling, `try`/`catch`, built-in containers) and the two user-visible limits (call depth 8192, nesting 65536) |
| the four unrecorded divergences above | nothing, if they are not in §11 to begin with — but the *omission* is why they went unnoticed |
| the invariant "§11 is checked by tests" | §11.2's numbers are enforced by `lang.MaxCallDepth`/`lang.MaxExprDepth`; nothing enforces that §11's *list* is complete |

**Recommendation (order: first). Done, 2026-10-17.** §11 is now a citation: the report is generated by
`cmd/qkstatus` from `spec/status/claims.tsv` and `spec/status/template.md`, committed as
`docs/implementation-status.md`, and `TestGeneratedStatusIsCurrent` regenerates it in memory and fails
when it is stale, in the same style as `cmd/qki18n -check`. The generator resolves every evidence
reference against the tree and refuses a claim that cites a test or corpus case that does not exist; the
report corrects the old §11.1 list, which had grouped four interpreter-only constructs with the
implemented ones. A generated status document cannot silently omit a divergence it is built from, and
the standard stays a statement of what the language *is*.

### 7.2 The tools — **pure restatement**, but they carry surface

Five frontends exist (`cmd/qkcheck`, `cmd/qkdoc`, `cmd/qkrepl`, `cmd/qklsp`, `cmd/qkstress`), and one
more lives outside this repository (`qkfmt`, referenced only by `scripts/bench-tools.sh:26,60` —
`docs/ARCHITECTURE.md` §6.2 G-7).

| What would leave | What it costs |
|---|---|
| nothing in the standard today names the tools — this is a *boundary* question, not a removal | the linter defines **15 codes** `QK101`–`QK115` (`grep -rhoE 'QK1[0-9][0-9]' internal/lang/lint.go \| sort -u`), i.e. a body of rules about the language that the standard does not mention. A reader who only reads the standard does not know they exist |

**Recommendation (order: third).** Do not remove the tools; **record the boundary** — one sentence in the
standard saying that diagnostics are not part of the language surface, and that a conforming program is
one the *engines* accept. The cost of *not* doing it is that the lint code set silently becomes the
de-facto standard for style and correctness.

### 7.3 The test corpus — **pure restatement**, and it is load-bearing

`compiler/testdata/cases/*.kq` (**21**, **23** after the two cases §7.6 item 2 added) and
`cases_run/*.kq` (**17**) = **40** parity cases today, plus `examples/*.qk` (**8**).

| What would leave | What it costs |
|---|---|
| the corpus is not *in* the standard; §1.4 says behaviour is defined by the interpreter (`:26-28`) | the corpus is the only place a *divergence* is pinned. §1.2b showed the corpus cannot pin `char`/`double` because it does not contain them; §1.3 showed it cannot pin shift precedence because **both engines agree** with each other and disagree with the standard. Two blind spots, both structural |

**Recommendation (order: second).** Keep the corpus, and add the two missing *kinds* of case: (a) one case
per §3.2 view row that the standard claims is core, so an unimplemented row fails loudly; (b) a case that
pins each **precedence level** against the standard's stated order rather than against the other engine.
Cost: two small files; benefit: the parity gate stops being the only gate.

### 7.4 The parity gate — **pure restatement**, keep the citation

`spec/STANDARD.md:10-11` names the gate: "`compiler/testdata/compare.sh` must keep printing `all
identical (interpreter == compiler)`".

| What would leave | What it costs |
|---|---|
| the citation in the standard | the standard's §1.4 ("the native compiler must match it bit for bit") would have no named instrument. This document's measurements show the gate is **necessary and not sufficient**: it is blind to (i) a feature both engines omit, (ii) a rule both engines implement the other way round, (iii) surface both engines refuse identically (§1.2b, §1.3, §1.5) |

**Recommendation (order: fourth).** Keep the citation, and make it precise: the gate pins
*interpreter == compiler*, not *code == standard*. One sentence in §1.4 stating which of the two the gate
proves would have made §1.3's divergence visible without a measurement.

### 7.5 Downstream users — **semantic change**, needs a ruling

| Downstream | Evidence | What it costs |
|---|---|---|
| `stdlib/option.qk`, `stdlib/result.qk`, `stdlib/vec.qk` | §9 `:249-253` promises `Vec<T>` and `Option`/`Result` as library | `stdlib/vec.qk:9` declares `pointer T data;` — a spelling §3.2 (`:91-92`) says does not exist ("no `pointer T` spelling: a pointer is `*T`"). Measured ✔: `pointer int p = null;` compiles and runs in **both** engines. So the standard library is written in surface the standard disclaims |
| `examples/*.qk` (8 files) | the corpus the READMEs point at | they are also the migration target of the `#when` removal (`40ad832`) |
| `QuarkLangLibs-*` (7 sibling repositories), `QuarkLangNpm`, `QuarkLangPlayground`, `QuarkLangQk*` | root `.gitignore` names them as separate repositories that "must not be tracked here" | every reduction above changes the surface they compile against; none of them is gated by this repository's CI |

**Recommendation (order: last).** Fix `stdlib/vec.qk`'s `pointer T` spelling first — it is either a
factual error in the standard (the spelling exists) or a defect in the library (it should be `*T`), and
either way it is **one line** and it is measured. The rest of this row is a semantic change and belongs
to the review date.

### 7.6 Recommended order, smallest first

| Order | Change | Class | Size | Cost if skipped |
|---|---|---|---|---|
| 1 | Generate §11 outside the standard; keep the citation to the generated file | pure restatement | doc + generator wiring | a status section that is measurably incomplete stays normative-looking |
| 2 | Add the two missing corpus *kinds* (§7.3) | pure restatement | 2 files | the gate keeps its two blind spots |
| 3 | Record the tool/lint boundary in one sentence | pure restatement | 1 sentence | 15 lint codes remain an unwritten standard |
| 4 | State what `compare.sh` proves, in §1.4 | pure restatement | 1 sentence | §1.3's divergence stays invisible |
| 5 | `stdlib/vec.qk`: `pointer T` → the standard's spelling, or §3.2 corrected | pure restatement (one of the two) | 1 line | the standard library contradicts the standard |
| 6 | Delete `#ifdef`/`#ifndef` in favour of `#if defined(…)`; delete `#ifn` from §7.2 and `lineDirective` | pure restatement | 2 spellings + 2 lists | continues the `#when` precedent one step short |
| 7 | Collapse `IndexOutOfBoundsError`/`IndexOutOfRangeError` to one reason id | pure restatement, but it changes a user-visible message | 2 catalog rows | one condition keeps two names |
| 8 | Reduce §4's 28 reference types to the `(set, rank)` algebra | pure restatement of what the checker does | §4 reworded | the largest enumeration in the language stays enumerated |
| 9 | Reduce §10 to release + block granularity | pure restatement of §10's own policy | §10 reworded | three of five memory entries stay primitive by listing |
| 10 | `char`/`double`: declare them interpreter-only in §11.2, or lower them in the compiler | semantic change | either | undeclared engine divergence (§1.2b) |
| 11 | Shift precedence: change the parser to the standard's order, or the standard to the parser's | **semantic change** — programs change meaning | 2 lines in the parser, or 1 line in §5 | both engines keep disagreeing with the document they implement |
| 12 | `#expand`/`#run`/`macro x($a $b)`: implement or delete from §7.2 | semantic change | §7.2 is a whole section | §7.2's example does not compile in either engine |

Items 1–9 are pure restatement: no program changes meaning. Items 10–12 change meaning or add surface,
and §12 puts them behind the review date.

Disposition, 2026-10-15: items 10–12 **await the owner's ruling** and are left as they are here. Item 10 was
closed without a ruling because §7.6 itself offers the choice and one branch changes nothing a program
computes: the compiler now lowers `char`/`double` onto the views the interpreter already used (§11.1 of the
standard). Item 11 is a real semantic change — the two-line syntax-layer fix is validated but not applied,
and the standard records the divergence in §11.3 with the reason it cannot be repaired in this repository.
Item 12 has no definition to implement against (§8's Q-3, Q-5). The generated report's §11.4 lists which of items
1–9 this revision adopted and which of them are not restatements after all.

Disposition, 2026-10-17, under the standard-is-truth rule:

- **Item 1 is completed as specified.** The standard's §11 is a citation to the generated report
  (`docs/implementation-status.md`); the claim/evidence table and the authored prose live in
  `spec/status/`, and `TestGeneratedStatusIsCurrent` is the drift gate. The generated table corrects the
  old hand-written list where it was wrong: `===`, permission references, `&lvalue` and `p++` are
  interpreter-only, and the compiler refuses all four.
- **Item 6 is completed as a record, not a deletion.** §7.2 names `#ifn` but defines it nowhere, and the
  shared preprocessor gives it no meaning. Defining it would invent surface; deleting it is surface
  removal; both wait for the owner. The generated status report's §11.3 records the gap and the safer reading
  (`#if !(condition)`, the complement of `#ifndef`), and `TestIfnHasNoDefinedMeaning` makes the
  implementation refuse it rather than accept an invented meaning. The consequence is a weak diagnostic
  (`#endif without a matching #if`) and no `#ifn` form until the owner rules.
- **Item 7 is completed as the opposite of the proposal.** The two names describe different conditions —
  a container subscript and a bit subscript (or a reference moved past the end) — so collapsing them
  would change a user-visible message and conflate the conditions. §8 now defines each and says where it
  is raised; the catalog rows are unchanged; `internal/lang/error_names_test.go` pins the distinction.
- **Item 8 is completed as the reconciliation, not the rewrite.** §4 now says what the checker does: the
  referent does not move while the variable holding a reference may be re-bound with `m` (Q-1); the four
  scopes are one lifetime order `u > t > a > f`, `a` has no behaviour beyond that rank, and the empty
  permission set is not spellable because it would permit nothing (Q-2). No new surface; the 7 × 4
  enumeration is correct as it stands. Reducing it to the `(set, rank)` algebra remains a later
  simplification, not a correction.
- **Item 9 is completed as the reconciliation, not the rewrite.** §10 now names `memory.clear()` as the
  derived wiping operation (release plus zero-fill) that the `delete` row already referred to (Q-6), so
  release does not wipe and the four primitive rows stay the surface. Reducing the table to release plus
  block granularity remains a later simplification; the generated report's §11.3 keeps the per-engine status of every row.

---

## 8. Open questions

Each question is stated with the consequence of each answer. **None was resolved when this document was
written**; the resolutions dated 2026-10-17 record which questions the standard has since answered, and
which remain open.

**Q-1 · Is a reference an address with a permission, or a second kind of thing?**
§4 (`:155-158`) says a reference "never moves" and that moving is what distinguishes a pointer; the
checker says a reference with `m` **may** be moved (`internal/lang/typecheck.go:1665`), and `p++` is
documented as both a reference move (D-15) and a pointer operation.
*If §4's reading is right:* `m` on a reference is meaningless and should be refused for references —
which removes one of the 7 permission spellings' effects and makes `P-3` + `P-4` the honest model.
*If the checker's reading is right:* §4's sentence "a reference never moves" is wrong and a reference *is*
an address with a permission — which is the model in §2.1, and `m` becomes the movability bit.
*Consequence either way:* `D-11`(28 reference types) becomes a *composition*, and §4's prose needs one
sentence changed.

**Resolved, 2026-10-17 — the checker's reading, without new surface.** §4 now distinguishes the referent
from the variable: a reference's storage does not move, while `p++` re-binds the variable holding a pointer
or a reference that names a container element when it has `m`, so `m` is the movability bit. The implementation already did this
(`internal/lang/typecheck.go:1647-1666` for the type check, `internal/lang/eval.go:1626` for `moveRef`),
and no syntax or type was added. The consequence stands: `D-11` is a composition; the 28-type enumeration
in §4 is the surface and is correct.

**Q-2 · What is an "area" (`a`), and why is the empty permission set unspellable?**
§4 (`:150`) names four scopes in one line and never defines `a`. `refScopeRank`
(`internal/lang/refperm.go:63-75`) orders it `u > t > a > f` with no other consumer.
*If `a` has a definition:* it is a fifth rank or a real region, and the ordinal in `P-4` may need to
become a lattice.
*If not:* the four scopes are four names for one ordinal, and `a` should be removed or defined.
*On the empty set:* the parser enumerates 7 of the 8 subsets (`qkparser/refperm.go:12`). If "no
permission" is meaningful, the enumeration is incomplete; if it is not, §4 should say the empty set is
refused *and why*.

**Resolved as far as the surface allows, 2026-10-17.** §4 now says the four scopes are one lifetime order,
`u > t > a > f`, and that `a` is an area scope with no behaviour beyond that rank — which is exactly what
the implementation does (`internal/lang/refperm.go:63-75`; `internal/lang/typecheck.go:1367,1417` enforce
the order). The empty set is now stated as not a permission, because a reference carrying it could neither
read, write, nor move; the parser's seven spellings (`qkparser/refperm.go:10-16`) are therefore complete,
not truncated. No new surface. A real region model for `a` would be new surface and waits for the review
date; its consequence today is that `a` is a rank only, indistinguishable in behaviour from any other
scope of the same order.

**Q-3 · What is a "node" for `#expand`/`#run`?**
§7.2's example (`:222-230`) is the only specification. `docs/ARCHITECTURE.md` §6.8 records that the AST
has no addressable node metadata and that `#run` collides with the automatic entry point.
*If a node namespace is declaration names:* `#expand`/`#run` become derivable from `P-6` (bind and
apply) and `P-8` (selection).
*If it is arbitrary structure:* they need a new primitive, and §7.2's example must be extended before
either engine can implement it identically.

**Q-4 · Does `#ifn` mean "not", and if so is it `!` applied to what?**
§7.2 (`:217`) lists it in the conditional family and defines it nowhere; no engine implements it (§1.5);
`docs/ARCHITECTURE.md` §6.8 already reports the gap.
*If it means `#if !(…)`:* it is derived (D-40), deletable, and costs nothing to remove.
*If it means something else:* §7.2 needs a definition and one parity case.

**Resolved as a record, not a definition, 2026-10-17.** Neither answer is adopted: defining `#ifn` would
invent surface, and deleting it would remove surface, so both wait for the owner. The generated status report's
§11.3 now records that the semantics are undecided and that the safer reading is `#if !(condition)`, the complement
of `#ifndef`, because it is the only reading derivable from the existing conditional family. The
implementation gives it no meaning: the shared preprocessor leaves the line unresolved, so the following
`#endif` is refused with `#endif without a matching #if` in both engines, and `TestIfnHasNoDefinedMeaning`
pins that refusal. Consequence: a program using `#ifn` gets that diagnostic rather than a defined branch,
and no parity case exists until the owner rules.

**Q-5 · Is the macro a directive or a rewrite stage, and which spelling is the language's?**
§7.2 (`:218`) puts `macro x($a $b) { … }` in the directives table; the implementation has `#macro name
(p1, p2) { body }` and rejects `$` (§11.3 `:323-325`). Measured: the standard's spelling is a
`ParseError` in both engines (§1.5).
*If a directive:* `P-8` must cover it and the table's single row is right.
*If a rewrite stage:* it belongs in §2 or §6, not §7.2, and the standard should say which spelling the
language has — currently it says one and the code has the other.

**Q-6 · Does release wipe?**
§10 (`:271`) says `delete` does **not** wipe and that wiping is `clear`'s job — a guarantee about what a
released block may hand to the next user. `internal/lang/eval.go:723` implements exactly that ("data
kept; only clear wipes it"), and `clear` is not in §10's primitive table.
*If release does not wipe:* `clear` is normative surface and should be in the table (as a derived
operation, D-31), and the security sentence stays in `delete`'s row.
*If it should wipe:* the sentence moves and `delete` becomes release + zero-fill.
*Consequence:* this decides whether the memory surface is 2 or 3 entries, and it is the one place where
"derived" and "normative" must both be said of the same operation.

**Resolved, 2026-10-17 — release does not wipe; `clear` is derivable.** §10 now names `memory.clear()` as
the derived wiping operation (release plus zero-fill) that the `delete` row already referred to, and keeps
the four primitive rows as the surface. The implementation matches: `delete` keeps the data
(`internal/lang/eval.go` `DeleteStmt`), `clear` wipes it (`MemoryManager.Clear`), and the interpreter
exposes `memory.clear()`. No new primitive was added; the consequence is that the memory surface is four
primitives plus one derived operation, not five primitives, and the reduction to release plus block
granularity remains a later simplification.

---

## Appendix A — Measurement protocol

Every measured claim above came from one of these, run in the analysed checkout with both engines built
from that revision (`go build -o /tmp/quark .` and `cd compiler && go build -o /tmp/qkc .`).

**Probe harness** (each row: write a one-program file, run it in both engines, compare the exit code and
the whole output; a difference is reported as `DIVERGE`, sameness as `SAME`):

```bash
run() { printf '%s\n' "$2" > p_t.kq
  i=$(/tmp/quark p_t.kq 2>&1); ri=$?
  c=$(QUARK_CACHE=$(mktemp -d) /tmp/qkc -run p_t.kq 2>&1); rc=$?
  if [ "$i" = "$c" ] && [ $ri -eq $rc ]; then echo "SAME exit=$ri"; else
    echo "DIVERGE interp($ri)=$i | qkc($rc)=$c"; fi; }
```

Programs probed: the type spellings of §1.1/§1.2b, the four precedence expressions of §1.3, the six
directive forms of §1.5, the five memory operations of §1.7, and `div by zero` / `index out of range` /
`panic` / `try-catch` for §1.8. The probes ran under the default locale of this machine
(`LANG=zh_CN.UTF-8`), so the *wording* of a message is the Chinese column of the catalog; the message
*id* is the same in both locales, and the message paths are the ids (`i18n.Msg…`).

**Counts** came from `grep` over the pinned sources; each is stated with its command in §1 so it can be
re-derived:

| Count | Command |
|---|---|
| `tKind` = 28 | `sed -n '17,44p' internal/lang/typecheck.go \| grep -c '^\tt'` |
| statements 12 / expressions 16 | the types implementing `isStmt()`/`isExpr()` in `qkparser/ast.go` |
| operator tokens 21 | `qkparser/token.go:45-68` |
| precedence levels 10 | `qkparser/parser.go:1319-1332` |
| operation methods 13 | `internal/lang/typecheck.go:597-625` |
| permissions 7 / scopes 4 | `qkparser/refperm.go:12,21` |
| enforcement sites 7 | `grep -n 'refPerm\|refScope' internal/lang/*.go` |
| error prefixes 17 | `grep -rhoE '[A-Za-z]+Error:' internal/i18n/source/ \| sort \| uniq -c` |
| corpus 21 + 17 | `ls compiler/testdata/cases/*.kq`, `ls compiler/testdata/cases_run/*.kq` |
| lint codes 15 | `grep -rhoE 'QK1[0-9][0-9]' internal/lang/lint.go \| sort -u` |

## Appendix B — Gates run for this change

This is a documentation-only change; the gates still ran, on the branch that carries this file, at
`923784b` + this document.

| Gate | Command | Result |
|---|---|---|
| build + test, root and `compiler`, `LANG=C.UTF-8` | `go build ./... && go test ./... -count=1` | ✔ all packages `ok`, exit 0 |
| build + test, both modules, `LANG=zh_CN.UTF-8` | same | ✔ all packages `ok`, exit 0 |
| dual-path parity | `bash compiler/testdata/compare.sh` | ✔ `all identical (interpreter == compiler)` |
| formatting | `gofmt -l internal compiler cmd` | ✔ empty |
| generated artefacts current | `go generate ./internal/i18n/` then the `-check` run | ✔ no diff, `qki18n: checked …` ×3 |
| stress subset | `go run ./cmd/qkstress run … -timeout 90s -skip 'path_fn_10k_params\|res_few_mb'` | ✔ exit 0, "no loud findings"; 28 cases AGREE 14 / REJECT_AGREE 11 / REJECT_DIFF 3, the three being the pre-existing wording-only `no main function found` differences |

The three `REJECT_DIFF` cases are *wording-only* disagreements between the two engines about the message
for "no main function found". They are non-fatal by the suite's own rule and predate this change; they are
noted here because they are the same class of fact as §1.2b — the two engines agree on behaviour and
differ on words.

**One gate-order interaction, measured while running these gates.** The stress gate writes its corpus
into `stress-out/cases/` (30 programs, 10 MB, gitignored — `ci.yml`'s stress job materialises it with
`go run ./cmd/qkstress gen -out stress-out/cases`). The test gate's corpus walker starts at the repository
root and skips only `.git`, `dist`, `dist-ci`, `node_modules`, `lintbench` and `bench`
(`internal/lang/lintcorpus_test.go:56-70`, `corpusFiles`); `stress-out` is **not** in that list. So running
the stress gate *before* the test gate in one checkout makes the test gate lint the extreme corpus and die
with `fatal error: stack overflow` (`runtime: goroutine stack exceeds 1000000000-byte limit`) — measured,
under both locales, and it disappears the moment `stress-out/` is removed (`internal/lang` then reports
`ok` in 10.1 s / 11.7 s). CI never sees it because the two gates run on separate runners. This is not
caused by this document (a markdown file is not linted as a program); it is recorded here because the
gate order above is the order a reader will run, and the failure looks like a regression.

**A second observation from the same gate run: a committed artefact carried machine-specific absolute
paths.** The generated-artefact gate is green, and it was green *while* the artefact contained another
worktree's paths: `internal/i18n/source/fields.tsv` held **93** occurrences of one machine's home
directory in its source-site column (e.g. row 95 cited an absolute path ending
`/internal/lang/eval.go:3563`, where its sibling row in `typecheck.tsv:57` cites the same kind of site
relatively, as `internal/lang/typecheck.go:1926`), and the generated backlog review document that
repeated those columns carried **64** of them. The gate compares the committed artefact against a fresh
generation, so it could not see this: both sides were equally absolute. It is recorded here because it is
the same shape as §1.3 — a gate that verifies *consistency* rather than *correctness* — and because
§7.2's boundary question is exactly about what the catalogue and the tools are allowed to put into a
committed file. **Fixed since:**
every authored site is written relative to the repository root, the generator refuses an absolute site
while reading a table (`checkSiteContext` in `internal/i18ngen`), and `internal/i18n/sites_test.go` keeps a
machine path out of the authored tables and the committed artefacts.
