# QuarkLang — Interface Inventory and Composition Map

English | [中文](ARCHITECTURE.zh-CN.md)

Analysed from the code at `67ce88c` (the tip of `main` when this document was written).
Every claim carries a `file:line` reference. A claim that could not be verified from source
or by running a gate is marked **UNVERIFIED** — there are no inferred-but-unchecked statements.

This document describes QuarkLang as a set of **interfaces** and the **compositions** that
consume them. It never says "project X is red" or "the project does X". It says:

> interface `I` promises `P`; under composition `C` that promise does not hold.

Where two places in the repository disagree, both are quoted and the contradiction is stated
rather than silently resolved (§6). The reader is left to decide which side is wrong.

---

## 0. Why this shape

A language implementation is not one program; it is a stack of interfaces that are each verified
under some compositions and unverified under others. The interesting engineering content is
exactly in that asymmetry: the depth cap holds on every OS because it is an explicit counter and
not a stack probe (`internal/lang/limits.go:9-13`), while byte-for-byte engine parity holds for the
committed corpus and is a CI *step* that only one platform runs
(`.github/workflows/ci.yml:118-121`).

Two conventions used throughout:

| Symbol | Meaning |
|---|---|
| `I-k` | an interface in the inventory (§1) |
| `C-k` | a composition in the map (§2) |
| ✔ / ✘ / ⚠ | promise holds / does not hold / holds only under a stated restriction |

The three Go modules in play:

```
┌─────────────────────────────────────────────────────────────────────────┐
│ module github.com/QuarkLangCommunity/QuarkLangQkc            (root)     │
│   go.mod:6-7    requires qkparser @ v0.0.0-20261010115438-20ae746ffdee  │
│   internal/lang        interpreter, typecheck, linter, VM, docs, REPL   │
│   internal/i18n        catalog + localizer                              │
│   cmd/{qkcheck,qkdoc,qkrepl,qklsp,qkstress,quarkwasm}                    │
├─────────────────────────────────────────────────────────────────────────┤
│ module .../QuarkLangQkc/compiler                     (submodule)        │
│   go.mod:5      replace .../QuarkLangQkc => ../                         │
│   go.mod:12     pins qkparser again (indirect)                          │
│   internal/cgen        LLVM IR backend                                  │
├─────────────────────────────────────────────────────────────────────────┤
│ module github.com/QuarkLangCommunity/QuarkLangQkparser    (shared)      │
│   go.mod        requires NOTHING — stdlib only                          │
│   lexer, macro/preprocessor, parser, AST, positions, localizer iface    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 1. Interface inventory

### 1.0 Summary table

| `I-k` | Interface | Location | Verifier | Failure mode |
|---|---|---|---|---|
| I-1 | Lexer | `qkparser/token.go` | `lintcorpus_test.go:148` | `\r` is whitespace; positions stay line-ending independent |
| I-2 | Macro / preprocessor | `qkparser/macro.go` | `eval_test.go:574`, `macroinsert_test.go` | **mode mismatch across engines** (§6.6) |
| I-3 | Parser | `qkparser/parser.go` | `parser_test.go`, `parser_limits_test.go` | refuses nesting > `MaxExprDepth` |
| I-4 | AST + positions | `qkparser/ast.go:4-8` | every downstream test | `Stmt`/`Expr` sealed by unexported methods |
| I-5 | Type checker | `internal/lang/typecheck.go` | typecheck tests in `internal/lang` | explicit depth counter, not a stack probe |
| I-6 | Tree-walk interpreter | `internal/lang/eval.go` | `eval_test.go`, `edges_test.go` | divergent error model for `try` (§6.1) |
| I-7 | Bytecode VM + deopt | `internal/lang/vm.go` | `vm_diff_test.go` | deopts, never errors, on non-int values |
| I-8 | cgen → LLVM IR | `compiler/internal/cgen` | `parity_test.go`, `compare.sh` | refuses unsupported constructs; never miscompiles |
| I-9 | Memory manager | `internal/lang/eval.go:199-354` | `eval_test.go:372,708` | interpreter-only engine surface |
| I-10 | i18n catalog + localizer | `internal/i18n` | `coverage_test.go` (8 gates) | ratcheted, not exhaustively covered |
| I-11 | Module wiring | root `go.mod`, `compiler/go.mod` | `go build ./...` × 3 OS × 2 modules | a local `replace` is honoured only in-checkout |
| I-12 | Tool frontends | `cmd/*` | per-`cmd` `main_test.go` | CLI help says "default Chinese" (§6.5) |
| I-13 | CI gates | `.github/workflows/*.yml` | — | see §2.3 for the job × OS matrix |

### I-1 · Lexer

- **Location**: `qkparser/token.go:170` (`Lex`), `:176` (`LexWithComments`), `:180` (`lex`).
- **Contract**: `Lex(src string) ([]Token, error)`; every token carries `Kind`, `Text`, `Line`, `Col`
  (`token.go:94-102`). Line and column are **line-ending independent**: replacing every `\n` with
  `\r\n` yields the same token count and identical `Kind`/`Line`/`Col` per index.
- **Who composes it**: `internal/lang/compile.go:113` (interpreter front end);
  `compiler/macros.go:51` (the compiler's macro pass); re-exported as `internal/lang.LexWithComments`
  (`internal/lang/syntax.go`); `cmd/qkcheck` through `internal/lang.LintSource`.
- **Verified by**: `internal/lang/lintcorpus_test.go:148` `TestLintCRLFPositions` asserts the whole
  token stream matches token-by-token under CRLF, and `:170-174` asserts the token stream is identical field by field.
  CI runs it as a named step (`.github/workflows/ci.yml:41-42`).
- **Failure modes**: an unterminated string or raw string returns `LexError` (`token.go:104-112`).
  Width beyond the supported set is the parser's check, not the lexer's
  (`parser.go:99` `isBuiltinTypeName`).

### I-2 · Macro / preprocessor — behaviour diverges, see §6.6

- **Location**: `qkparser/macro.go`; entries `SplitMacroDefs` (`:22`), `ExpandMacros` (`:149`),
  `expandBody` (`:211`).
- **Contract (as implemented)**: `ExpandMacros(toks, macros, mode)` rewrites macro calls. `mode`
  selects the `#when` operation time: `expandBody:268` fires a branch when
  `args[0].Text == mode || (mode == "explain" && args[0].Text == "run")`. The refusal message
  advertises the accepted set as `compile|run` (`macro.go:255`, catalogued at
  `internal/i18n/table_macro.go:18`).
- **Who composes it**: the interpreter passes `"explain"` (`internal/lang/compile.go:123`); the
  compiler passes `"compile"` (`compiler/main.go:334`, `:441`) through its own wrapper
  (`compiler/macros.go:44`), which delegates to the same `lang.SplitMacroDefs` / `lang.ExpandMacros`
  (`macros.go:53,60`). The macro *logic* is therefore shared and only the *mode* differs.
- **Verified by**: `internal/lang/eval_test.go:574` `TestMacroWhenCompileDropped` pins the interpreter
  to `run-line` for a macro carrying both branches; `macroinsert_test.go` covers `#insert`/`#ast`;
  `TestMacroErrorDirective` (`eval_test.go:594`) covers `#error`. There is **no committed test of the
  compiler's macro path**: `compiler/macros_test.go` exists, but only as an untracked file in the
  developer's working copy (`git status` in the main checkout reports `?? compiler/macros_test.go`), on
  no branch and in no commit reachable from `main`.
- **Failure modes**: `#error` aborts expansion (`macro.go:279`); an unknown `#` command is refused
  (`macro.go:308`); an argument outside the accepted `#when` set fires no branch and is dropped
  **silently** — measured, §6.6.

### I-3 · Parser

- **Location**: `qkparser/parser.go:30` (`Parse`), `:113` (`parseProgram`); errors are
  `ParseError{Msg, Line, Col}` (`:9-17`).
- **Contract**: a token stream becomes a `*Program`. The recursive descent counts its own nesting and
  refuses the first level past `MaxExprDepth` (`qkparser/limits.go:17`, value `65536`) with a position;
  it never lets the Go stack decide (`limits.go:8-13` gives the reason: a process stack is
  platform-sized, so an implementation that let it decide would answer differently per OS).
- **Who composes it**: `internal/lang/compile.go:128`; `qkparser.ParseSource*` (`frontend.go:11,17,24`),
  re-exported at `internal/lang/syntax.go:195,199`.
- **Verified by**: `qkparser/parser_test.go`, `qkparser/parser_limits_test.go`; the root module
  re-checks the limit end-to-end through the stress case `scale_line_1mb` (`stress/NOTES.md:20-33`).
- **Failure modes**: legal-but-deep constructs are refused rather than crashing — the cap sits 4.5×
  below the measured crash depth (`stress/NOTES.md:31`).

### I-4 · AST and positions

- **Location**: `qkparser/ast.go`; `Pos{Line, Col}` at `:4-8`; `Program` at `:16`.
- **Contract**: `Stmt` and `Expr` are **sealed interfaces** — `isStmt()`/`isExpr()` are unexported
  (`ast.go:130-133`, implementations at `:222-232`), so no package outside `qkparser` can add a node
  kind. Consumers switch over a closed set.
- **Who composes it**: `internal/lang` re-exports the node types as aliases (`internal/lang/syntax.go`,
  e.g. `type Block = qkparser.Block`), so interpreter and linter are clients of the *same* node set;
  `compiler/internal/cgen/lower.go:927` walks the same `*lang.Block`.
- **Verified by**: implicitly by every consumer test; the seal itself is a language-level guarantee.
- **Failure modes**: the seal makes a new node kind an `qkparser`-only change that every engine must
  follow in lockstep — the intended forcing function, and the reason the interpreter's docstring calls
  the AST "the canonical AST" (`cgen.go:4-5`).

### I-5 · Type checker

- **Location**: `internal/lang/typecheck.go`; entry `Typecheck(prog *Program) error` at `:675`, `checkBlock` at
  `:1154`.
- **Contract**: rejects a program before any engine runs it; returns a positioned `CheckError`. It
  recurses once per expression node, so it enforces its own counter against
  `internal/lang.MaxExprDepth` — the *same* constant the parser uses, declared as an alias rather than
  a second number (`internal/lang/limits.go:30`; `:15-29` records that a duplicate value with duplicate
  wording used to drift).
- **Who composes it**: `internal/lang/compile.go:133`, inside `compileSlow`, so **both** engines reach
  it — cgen calls `lang.CompileWithImports` (`compiler/internal/cgen/cgen.go:35-37`).
  `cmd/qkcheck -no-typecheck` opts out (`cmd/qkcheck/main.go:7`).
- **Verified by**: typecheck tests inside `internal/lang`; the depth cap is exercised by the
  `scale_line_1mb` stress case, which reports `REJECT_AGREE` — both engines refuse with a position
  (`stress/NOTES.md:35-37`).
- **Failure modes**: the two depth counters measure *different things* — the parser counts recursion
  levels, the checker counts expression nodes — so a deep `+` chain is refused by the checker and a deep
  parenthesis nest by the parser (`internal/lang/limits.go:27-29`). Documented asymmetry, not a defect.

### I-6 · Tree-walking interpreter

- **Location**: `internal/lang/eval.go`; entries `Run` / `RunDebug` (re-exported in
  `internal/lang/syntax.go`); the interpreter struct at `eval.go:397`, its memory manager initialised at
  `:424`.
- **Contract**: **the semantic reference.** `compiler/internal/cgen/consistency_test.go:8-16` states the
  rule the whole system leans on: when the compiler cannot implement a construct with exactly the
  interpreter's semantics it must refuse the program, never accept it and answer differently.
- **Who composes it**: every `cmd/*` through `internal/lang.Run`; `cmd/qkrepl` through the REPL session
  (`internal/lang/repl.go:106`); `cmd/quarkwasm` — a third backend, the browser
  (`cmd/quarkwasm/main.go:1-5`).
- **Verified by**: `internal/lang/eval_test.go`, `edges_test.go`, `refmove_test.go`,
  `strict_eq_test.go`, plus the differential corpora of §2.1.
- **Failure modes**: a runtime error raised *inside a callee* is caught by the caller's `try` in the
  interpreter but terminates the compiled binary — recorded in `stress/NOTES.md:102-105` as a real,
  pre-existing divergence, reported rather than hidden.

### I-7 · Bytecode VM and its deopt path

- **Location**: `internal/lang/vm.go` (829 lines). Deopt sentinel `errVMDeopt` at `:33`; entry `runVM`
  at `:595`; safety model documented at `:12-18`.
- **Contract**: **an optimisation, never a second semantics** (`vm.go:5-9`). A function is compiled only
  when its *whole body* fits the supported subset; anything else — `try`/`catch`, for-in, method calls
  other than `io.print`/`io.println`, space calls, struct/list/string operations, nested-block
  declarations, shadowing — falls back to the tree-walker (`vm.go:12-15`). The VM reuses the
  interpreter's own primitives (`wrapI32`, `callFunc`, `ioPrintln`) so error values are identical.
  Deopt is signalled, not errored: a function whose runtime value contradicts its statically proven type
  is blacklisted and re-run by the tree-walker (`vm.go:29-34`).
- **Composition switch**: `QUARK_NO_VM=1` disables the VM entirely (`vm.go:16-17`) — the switch the
  differential test flips.
- **Verified by**: `internal/lang/vm_diff_test.go` — `TestVMEqualsTreeWalker` (`:114`) asserts identical
  stdout *and* an identical one-line outcome with the VM on and off; `TestVMEqualsTreeWalkerOnCorpus`
  (`:158`) repeats it over `../../examples` and `../../compiler/testdata/cases`. Crucially,
  `TestVMCompilesHotShapes` (`:132`) asserts `vmExecutions` (`vm.go:25-27`) advanced for three named
  shapes and **did not** advance for `string_ops_fall_back` — the anti-silent-fallback ratchet, without
  which a permanently deoptimising VM would keep every semantic test green while doing nothing.
- **Failure modes**: bytecode implements integers only; a float arriving in an `int`-declared parameter
  is legal in this language and must deopt instead of raising (`vm.go:601-602`). The "VM still engages"
  check is a **4-entry whitelist** (`vm_diff_test.go:137-141`), so unlisted shapes are unverified (§6.2 G-9).

### I-8 · cgen → LLVM IR

- **Location**: `compiler/internal/cgen/`; entry `Transpile(src, filename) (string, error)`
  (`cgen.go:34`). Pipeline documented at `cgen.go:3-9`:
  `lang.CompileWithImports → lowerProgram (lower.go) → emitter (cgen.go)`.
- **Contract**: source in, **LLVM IR text** out. It carries no lexer or parser of its own
  (`cgen.go:5-6`) — one syntax source, `internal/lang`. Value model at `cgen.go:15-22`: `int→i32`,
  `bool→i1`, `float→double`, `String→i8*`; `List<T>` and structs are heap references, matching the
  interpreter's aliasing so writes through any alias are visible. A construct the backend has not
  lowered returns an explicit, position-carrying error — **never a silent miscompile**
  (`cgen.go:11-12`).
- **Who composes it**: `compiler/main.go` (the `qkc` CLI) and the compiler module's tests.
- **Verified by**: `parity_test.go` — `TestParityCaseIRSyntax` (`:35`) pipes every case through LLVM's
  own `llvm-as` and **skips when it is absent** (`:38-40`); `TestParityCaseOutput` (`:69`) compares
  stdout **byte for byte** against a committed `.out` fixture produced by the interpreter and **skips
  when `lli` is absent** (`:71-73`). End-to-end, `compiler/testdata/compare.sh:31` requires
  stdout+stderr *and* exit-code equality.
- **Failure modes**: the two `t.Skip` paths mean the IR and output contracts are simply not checked
  where the LLVM tools are missing (§6.2 G-10). The `.out` fixtures are tracked only because of a
  targeted `.gitignore` negation (`.gitignore:11-13`); without it the parity test fails on a missing
  fixture rather than on a mismatch.

### I-9 · Memory manager

- **Location**: `internal/lang/eval.go:199-354` — `MemBlock` (`:200`), `memHeap` (`:210`),
  `MemoryManager` (`:229`), `Alloc` (`:242`), `ReclaimTask` (`:275`), `Compact` (`:292`),
  `Fragmentation` (`:307`), `BlockCount` (`:324`), `Delete` (`:331`), `Clear` (`:342`). The built-in
  `memory` value is `globalMemory = &Memory{BlockSize: 4096}` (`value.go:754-760`).
- **Contract**: no garbage collector; `delete` is a deterministic release that returns storage to a free
  pool without wiping it (`spec/STANDARD.md:264-269`). Allocation prefers the **least-occupied block
  with room**, else requests a new block (`eval.go:242-271`), which is what lets a tide-shaped workload
  reuse a stable block set; occupancy is tracked by a min-heap keyed on `Used` (`eval.go:210-215`).
- **Who composes it**: the interpreter only — `interp.mem` (`eval.go:397`, initialised `:424`), used for
  the `memory.*` builtins (`:1128`, `:2388-2400`), `new`/pointers (`:741`) and `taskm` (`:3353`,
  `:3374`).
- **Verified by**: `internal/lang/eval_test.go:372` (after `compact`, exactly the persistent thread block
  remains **provided the merged task has already finished** — see §5.4, this one is scheduling-dependent
  and has failed on `windows-latest`) and `:708` (after `delete`+`compact`, zero blocks, no concurrent
  task); benchmarks at `bench_test.go:85,130`.
- **Failure modes**: **the interpreter is the only engine with this manager.** cgen has no
  `MemoryManager`; the compiled binary's storage story is `calloc` plus the runtime it emits. That is a
  boundary rather than a defect, but it means `memory.Fragmentation()` and friends are engine-specific
  surfaces that a parity case must avoid. Releasing storage something still refers to is undefined and
  undetected by design (`spec/STANDARD.md:280-282`). Two doc-comment drifts are recorded in §6.4.

### I-10 · i18n catalog and localizer

- **Location**: `internal/i18n/`; `Detect` (`i18n.go:56`), `Catalog` (`:69`), `Load` (`:82`),
  `Localizer` (`:167`), `T` (`:201`), `SetLocale` (`:268`), miss counter (`miss.go:13-40`).
- **Contract**: `Load` builds an immutable catalog from ten per-domain tables (`i18n.go:148-161`). The
  **direction of an entry is a property of its key**: a key containing Han characters is Chinese-source
  and its value is the English translation; otherwise it is English-source and its value is the Chinese
  translation (`i18n.go:96-102`, `:220-237`). `Localizer.T` looks up **the localizer's own language**
  (`:201-213`) — the fix in `43bbff6`; before it the lookup was gated on `EN`, so 105 English-source
  entries were unreachable under `zh` (§5.2).
- **Who composes it**: `internal/lang/localizer.go:18-23` `SetLocalizer` injects the localizer into
  **both** this package and `qkparser`; `init()` at `:34` wires the parser. Each `cmd/*` sets the
  language from `--lang` / `QK_LANG`.
- **Verified by**: eight gates in `internal/i18n/coverage_test.go` — `TestTemplatesRegistered` (`:124`),
  `TestCatalogIntegrity` (`:170`), `TestVerbSetsMatch` (`:190`), `TestNoSilentFallback` (`:212`),
  `TestDecouplingRatchet` (`:240`), `TestEnglishSourceRatchet` (`:267`),
  `TestTRendersTheCatalogForTheRequestedLanguage` (`:294`),
  `TestCatalogDirectionFollowsTheSourceLanguage` (`:319`) — plus `verborder_test.go:19`.
- **Failure modes**: the substitution-verb multiset must match between the two languages
  (`coverage_test.go:190`). A template absent from the catalog renders in its own source language and
  **is counted, not refused** (`i18n.go:206-207`), so silent fallback is observable but not blocking.
  The wiring scanner walks only this repository (`coverage_test.go:44`), and `skipNestedRepo`
  (`:355-361`) stops at any nested `.git`, so **the pinned parser module's own templates are outside
  every gate** — §5.2 has the probe and its result.

### I-11 · Module wiring

- **Location**: root `go.mod:1-10`, `compiler/go.mod:1-12`.
- **Contract**: `go.mod:5-10` pins the syntax layer to one exact pseudo-version
  (`v0.0.0-20261010115438-20ae746ffdee`) resolved through the module proxy, so a fresh clone needs no
  sibling checkout. The comment at `go.mod:5-9` records the failure this repaired: a relative `replace`
  is honoured only inside the checkout and ignored by importers, "which is exactly how moving the syntax
  layer out broke CI on all three platforms".
- **Submodule**: `compiler/go.mod:7` replaces the root module with `../`; `:12` repeats the parser pin
  because **while the compiler is the main module, a replace declared by the root module does not
  apply** (`compiler/go.mod:9-11`).
- **Verified by**: `go build ./...` + `go test ./...` in each module on three OSes
  (`.github/workflows/ci.yml:28-36`). Locally on 2026-10-10 (linux, Go 1.26.9, clang 22.1.8): root and
  compiler both build and test clean (`ok internal/lang 10.6s`, `ok compiler/internal/cgen 1.1s`).
- **Failure modes**: the shared frontend is **stdlib-only** (`qkparser/go.mod` has no `require` block) —
  that is what lets it be consumed without this repository. The price is that the parser cannot import
  `internal/i18n`, so it defines its own one-method `Localizer` interface and *waits to be injected*
  (`qkparser/localizer.go:8-11`): until `SetLocalizer` runs, every parser message renders verbatim
  through `verbatim{}` (`:15-26`).

### I-12 · Tool frontends

| Tool | Location | Contract |
|---|---|---|
| `qkcheck` (linter) | `cmd/qkcheck/main.go:1-13` | 15 stable codes `QK101`–`QK115` (`internal/lang/lint.go:20-35`); `-json`, `-params`, `-no-typecheck`, `-L`, `-exit0`, `--lang` |
| `qkdoc` | `cmd/qkdoc/main.go:1-12` | `Doc`/`DocItem` from `internal/lang/docrender.go`; Markdown by default, `-html` self-contained |
| `qkrepl` | `cmd/qkrepl/main.go:1-11`; `internal/lang/repl.go:106` | `-e` snippets; interactive `qk> ` prompt; batch mode when stdin is not a terminal, continuing after errors |
| `qklsp` | `cmd/qklsp/main.go:1-8` | diagnostics, definition, completion, hover, symbols; self-implemented JSON-RPC, **zero third-party deps** |
| `quarkwasm` | `cmd/quarkwasm/main.go:1-25` | `//go:build js && wasm`; `quarkCheck`/`quarkRun` on `globalThis`; the browser backend |
| `qkfmt` (formatter) | **not in this repository** | `QuarkLangQkfmt`, module `quarklang/qkfmt`, `go 1.21`, own scanner — see §5.1 |

The linter's design contract is stated at `internal/lang/lint.go:5-9`: scope rules aligned item by item
with the type checker, and it reports **only** what the compiler does not — hard syntax/type errors
belong to `typecheck` and are surfaced separately. That is what keeps the P99 zero-false-positive gate
attainable rather than aspirational.

The four `internal`-fronted tools advertise "default Chinese" (`cmd/qkcheck/main.go:10`,
`cmd/qkdoc/main.go:9`, `cmd/qkrepl/main.go:7`) while `i18n.Detect` returns `EN` as the process default
(`internal/i18n/i18n.go:65`). They are reconciled only by `Detect`'s environment-first order; the
contradiction is recorded in §6.5.

### I-13 · CI gates

`.github/workflows/ci.yml` — 4 jobs, 263 lines, `permissions: contents: read` (`:6-7`), job-level
concurrency that cancels in-progress runs on the same ref (`:10-12`), and every action pinned by SHA
with a version comment (`:24-25`, `:69-70`, `:99`, `:111-115`).

| Job | Matrix | Composition it defines |
|---|---|---|
| `test` (`:16`) | ubuntu, macos, windows | `go build`+`go test` root then `compiler/`; P99 lint gate (`:37-40`); CRLF/perturbation gate (`:41-42`) |
| `stress` (`:58`) | ubuntu, macos, windows | extreme corpus against **both** engines, `-timeout 90s`, two skipped cases, allow-list file |
| `linux-extras` (`:107`) | ubuntu only | dual-path `compare.sh`, toolchain build, editor/LSP, wasm, coverage, benchmarks, release smoke |
| `libraries` (`:182`) | ubuntu, macos, windows | emit-lib → link → **run**, asserting the platform branch value 58/59/61 |

Deliberate platform asymmetries are documented in-workflow rather than implied: RSS sampling degrades
to `-1` off Linux (`:55-57`), and two stress cases are skipped because cost, not a language property,
dominates them (`:49-53`).

---

## 2. Composition map

A composition is a concrete tuple of (interfaces, platform, environment, corpus). The point of this
section is that **an invariant belongs to a composition, never to an interface alone.**

### 2.1 C-1 · Dual-engine parity

```
        ┌──────────────────────────────┐
        │ qkparser: Lex→Macro→Parse    │   I-1, I-2, I-3, I-4
        └───────────────┬──────────────┘
                        │ one AST (I-4 sealed)
        ┌───────────────┴───────────────┐
        ▼                               ▼
┌──────────────────┐          ┌──────────────────────┐
│ I-6 interpreter  │          │ I-8 cgen → LLVM IR   │
│ (+ I-7 VM opt)   │          │ → clang → binary     │
└──────────────────┘          └──────────────────────┘
        │                               │
        └──────── byte-equal? ──────────┘
                        │
         compare.sh (32 cases) + parity_test.go
```

**Made of**: I-1…I-5, I-6, I-7 (optional), I-8, plus an LLVM toolchain that lives outside the repository.

**Invariants that hold for this composition**

| Invariant | Holds for | Evidence |
|---|---|---|
| stdout+stderr byte-equal, exit code equal | the committed corpus: 15 `cases/*.kq` + 17 `cases_run/*.kq` = 32 | `compare.sh:23,31`; **measured green on 2026-10-10** (`all identical`) |
| IR passes LLVM's own syntax check | every case, *when `llvm-as` exists* | `parity_test.go:35-40` |
| runner output equals the committed `.out` | 15 cases, *when `lli` exists* | `parity_test.go:69-73` |
| recursion cap identical, tail calls included | 4 `p_recursion_*` cases | `stress/NOTES.md:68-78` |
| unimplemented construct ⇒ refusal, not miscompile | `===`, permission refs, `&lvalue`, `p++` | `consistency_test.go:1-16` |
| macro expansion preserves the program | programs whose macro bodies contain **no string literal** | `compare.sh` (32/32 measured 2026-10-10) |

**What this composition does NOT promise**

- ✘ Parity for *every* program: the corpus is 32 files, and §6.6 exhibits a program outside it where the
  engines disagree by construction.
- ✘ Parity on any platform but linux **in CI**: `compare.sh` runs in `linux-extras`
  (`.github/workflows/ci.yml:118-121`) on `ubuntu-latest` only, and it hardcodes `/tmp/quark`, `/tmp/qkc`
  (`compare.sh:17-18`) — paths that do not exist on Windows.
- ✘ A caught `try` around a callee's runtime error — interpreter catches, compiled binary terminates
  (`stress/NOTES.md:102-105`).
- ✘ Re-parsing after macro expansion when a macro body contains a **string literal** — see §5.3; the
  interpreter is right and the compiled engine fails or prints nothing, and no corpus case exercises it.
- ⚠ The Go-level parity tests **self-disable**: a missing `llvm-as`/`lli` turns two contracts into skips,
  so a toolchain-free machine reports green while checking nothing (`parity_test.go:38-40,71-73`).

**Platform- and environment-independent parts**: the frontend caps (explicit counters, `limits.go:9-13`
and `qkparser/limits.go:8-13`), the AST contract (I-4), the type checker's rejection semantics, and the
macro *logic* are pure functions over strings and integer counters — the code says so and the design
depends on it. **Environment-dependent**: whether the compiler path is exercised at all (LLVM presence),
peak-RSS measurement (Linux `/proc` only), and `compare.sh`'s absolute paths.

### 2.2 C-2 · Language lookup (per locale)

```
   --lang / QK_LANG ─┐
                     ├─► i18n.Detect() ─► Lang ─► Localizer.T(template, args)
   LC_ALL/LC_MESSAGES/LANG ─┘                         │
                                            ┌─────────┴──────────┐
                                            ▼                    ▼
                                  Lookup(template, lang)   sourceLang(tpl) != lang
                                  → catalog value          → countMiss(tpl)
```

**Invariants per locale**

| Composition | Promise | Status |
|---|---|---|
| `zh` lookup | `T` returns the catalog's Chinese for the requested language | ✔ holds since `43bbff6`; pinned by `coverage_test.go:294` over all 918 pairs |
| `en` lookup | the mirrored promise | ✔ same gate |
| direction | entry direction read off the key, never assumed from the table | ✔ `coverage_test.go:319`, `i18n.go:96-102` |
| verb arity | a translation consumes exactly the source's fmt verbs | ✔ `coverage_test.go:190` |
| coverage | every wired template has a Chinese value | ⚠ **ratchet, not a gate**: ≤124 missing (`coverage_test.go:132`) |
| decoupling | new code carries a `Localizer` | ⚠ ratchet: ≤29 direct `i18n.T(` calls (`coverage_test.go:241`); measured **25** today |
| source direction | new messages are English-source | ⚠ ratchet: ≤318 Chinese-source (`coverage_test.go:268`) |

**"The `zh` composition" is therefore three different things**, and only the first is airtight:

1. `Localizer.T` under an injected `zh` localizer — verified exhaustively across 918 (template,
   language) pairs (`coverage_test.go:294-317`);
2. the *process default* on a machine whose environment says `zh` — verified only by `Detect`'s own
   behaviour;
3. a developer's or CI runner's machine, where the default may be `en` (GitHub's runners are en-US).

The 124-template backlog lives in layer (1)'s data but surfaces only as a count, so the sequence "add an
English-source message, forget its Chinese value" is permitted 124 times.

### 2.3 C-3 · CI job × OS × shell × checkout

| Job | OS | shell | checkout EOL | what it actually enforces |
|---|---|---|---|---|
| `test` | ubuntu/macos/windows | default per OS | native (CRLF on windows) | build, tests, P99 lint = 0, CRLF/perturbation invariance |
| `stress` | ubuntu/macos/windows | **bash** explicit (`ci.yml:65-67`) | native | both engines under a wall-clock bound; RSS bound only on Linux |
| `linux-extras` | ubuntu | bash | LF | dual-path parity, wasm build, editors, release smoke |
| `libraries` | ubuntu/macos/windows | **bash** explicit (`:189-191`) | native | emit→link→run, platform branch 58/59/61 asserted |

Two facts decide whether a step can exist at all in a given composition:

- `.gitattributes` contains **no `text`/`eol` rules** — only a linguist marker for
  `internal/lang/ffi_win.inc` — so a Windows checkout has **native CRLF** working-tree files.
- The two multi-OS jobs that contain POSIX shell (`$(...)`, `[ ... ]`, `command -v`, `case`) request
  `shell: bash` explicitly. `test` does not, and it works because its steps are pure `go` invocations
  that use no shell feature.

**This is the entire content of the Windows-formatting failure in §5.1**: a CI *step* interface
(`qkfmt -l $(git ls-files '*.qk')`) composes `windows × pwsh × CRLF checkout`, and two of those three
factors independently break it.

### 2.4 C-4 · Lint corpus perturbation

**Made of**: I-1 (lexer positions), I-3/I-4, `internal/lang/lint.go`, and a walk over this repository's
own `.qk`/`.kq` files (`lintcorpus_test.go:56-77`), skipping `.git`, `dist`, `dist-ci`, `node_modules`,
`lintbench`, `bench`, and every nested repository.

**Invariants**: a diagnostic set is unchanged by (1) CRLF conversion, (2) a trailing comment on every
line, (3) line shifts (`lintcorpus_test.go:88-146`); plus the statistical gate that the **P99**
false-positive and false-negative counts are exactly 0 over 2000 rounds × 4 cases
(`lintstat_test.go:456,544,547`).

**Does not promise**: invariance under reordering or semantic edits, and nothing about files outside
this repository — a library in a sibling checkout is outside the walk by construction.

### 2.5 C-5 · The formatter (out of tree)

**Made of**: `QuarkLangQkfmt` at `2396137` — a separate module (`quarklang/qkfmt`, `go 1.21`) with its
**own scanner** (`QuarkLangQkfmt/main.go:10-45`) that does not use `qkparser`. Its README claims two properties:
whitespace-only changes with a byte-identical second pass (idempotence), and a CI mode `-l` that exits 1
on any unformatted file.

**Invariants per composition**: see §5.1. The decisive fact is that **nothing in this repository invokes
it**: `qkfmt` appears in `scripts/bench-tools.sh:26,60` (a benchmark row and a sibling-repo build helper)
and nowhere in `ci.yml`, `release.yml` or `bench.yml`; the formatter's own CI is `ubuntu-latest` running
`go build ./... && go vet ./...`. The formatting promise therefore holds under **no CI composition at
all** today.

---

## 3. Boundaries and propagation

### 3.1 Shared vs private

```
 SHARED (published module, consumed by importers we cannot see)
 ┌──────────────────────────────────────────────────────────────┐
 │ QuarkLangQkparser: lexer, macro, parser, AST+Pos, limits,    │
 │ ValidRefPerm/ValidRefScope, Localizer interface, verbatim    │
 │ CONSTRAINT: stdlib only — it cannot import internal/i18n     │
 └──────────────────────────────────────────────────────────────┘
        ▲ injected at runtime                  ▲ pinned by pseudo-version
        │ (SetLocalizer)                       │ (go.mod:7, compiler/go.mod:12)
 PRIVATE (module-internal; importers cannot reach these)
 ┌──────────────────────────────┐   ┌──────────────────────────────────┐
 │ internal/lang                │   │ internal/i18n                    │
 │ internal/lang/stressgen      │   │ (tables + Localizer impl)        │
 │ cmd/*                        │   └──────────────────────────────────┘
 └──────────────────────────────┘   ┌──────────────────────────────────┐
                                    │ compiler/internal/cgen           │
                                    │ separate module; its own         │
                                    │ preproc.go + macros.go           │
                                    └──────────────────────────────────┘
```

`internal/` makes the interpreter, catalog and cgen private by construction: no external consumer can
import them. The only shared surface is the parser module — which is exactly why the *injection*
direction matters. The parser cannot call the catalog, so the root module calls into the parser
(`internal/lang/localizer.go:18-23`), and the parser sees only a one-method interface
(`qkparser/localizer.go:8-11`).

### 3.2 Where a change propagates

| Change | Propagates to | Forced by |
|---|---|---|
| a node kind in `qkparser/ast.go` | interpreter switch, linter switch, cgen `lower.go` | nothing — Go switches are not exhaustive, and the seal only prevents adding kinds *outside* the module |
| `MaxExprDepth` in the parser module | nothing to update: the root reads it by alias (`internal/lang/limits.go:30`) | drift made impossible by construction |
| `MaxCallDepth` (`internal/lang/limits.go:37`) | the interpreter's counter **and** the value baked into generated IR — wording included (`:39-42`) | the `p_recursion_*` parity cases |
| a message template | its catalog entry, in the same commit | `TestTemplatesRegistered` — **but only for templates written in this repository** (§5.2) |
| the pinned parser commit | everything; the pin is one line (`go.mod:7`) repeated in `compiler/go.mod:12` | `go build` on 3 OS × 2 modules |
| a new tool frontend | `cmd/`, `scripts/build-release.sh`, `install.sh`, `ci.yml`'s build loop | manual lists — no generator |

The last row is a real boundary hazard: `ci.yml:124` builds `qkcheck qkdoc qkrepl qklsp` from a
hand-written loop, `ci.yml:160-162` verifies the release artifacts `quark qkc qkcheck qkdoc qkrepl qklsp`,
and `install.sh:32` adds the siblings `QuarkLangQkfmt:qkfmt QuarkLangQktest:qktest QuarkLangQkm:qkm
QuarkLangQkd:qkd`. Four independent lists of "the toolchain", kept in step by hand.

### 3.3 Documented-vs-enforced boundaries

| Boundary | Documented at | Enforced by |
|---|---|---|
| the compiler must refuse what it cannot lower | `consistency_test.go:8-16`, `cgen.go:11-12` | `TestInterpreterOnlyConstructsAreRefusedNotMiscompiled` |
| frontend caps are platform-independent | `internal/lang/limits.go:9-13`, `qkparser/limits.go:8-13` | stress `scale_line_1mb`, `p_recursion_*` |
| one syntax source for both engines | `cgen.go:4-6` | cgen has no lexer; it calls `lang.CompileWithImports` |
| linter and typechecker share scope rules | `internal/lang/lint.go:5-9` | corpus invariance + P99 = 0 |
| the interpreter is the semantic reference | `consistency_test.go:8-16`, `vm.go:5-9` | differential VM test, `compare.sh` |

---

## 4. Invariant and ratchet catalogue

**Ratchet** = a bound that may move in one direction only, so a backlog is forced to shrink and cannot
grow. Each entry names what it pins, the test that enforces it, and the direction it may move.

| # | Name | Pins | Test | Direction |
|---|---|---|---|---|
| R-1 | untranslated templates | wired templates with no catalog entry | `coverage_test.go:132` (`≤124`) | **down only** |
| R-2 | English-source migration | Chinese-source wired templates | `coverage_test.go:268` (`≤318`) | **down only** |
| R-3 | localizer decoupling | direct `i18n.T(` call sites | `coverage_test.go:241` (`≤29`; measured 25) | **down only** |
| R-4 | the VM is actually used | bytecode ran for 3 named shapes, and not for `string_ops_fall_back` | `vm_diff_test.go:132` | boolean, both ways |
| R-5 | engine parity on the corpus | stdout+stderr+exit code, 32 cases | `compare.sh:31`, `parity_test.go:69` | must stay exact |
| R-6 | one recursion cap | same depth, same wording, both engines, tail calls included | `cases_run/p_recursion_*`, `limits.go:37-42` | must stay exact |
| R-7 | frontend depth caps | parser 65536 / checker 65536, one number | `qkparser/limits.go:17`, `internal/lang/limits.go:30` | must stay exact |
| R-8 | linter precision | P99 FP = 0 **and** P99 FN = 0 | `lintstat_test.go:544,547` | must stay 0 |
| R-9 | CRLF invariance | identical tokens and diagnostics under CRLF | `lintcorpus_test.go:148`, `:88` | must stay exact |
| R-10 | refusal over miscompile | an unimplemented construct ⇒ positioned error | `consistency_test.go:24+`, `feature_test.go:647+` | must stay exact |
| R-11 | known-failure allow-list | a listed case needs a reproduction; a fixed case must be delisted | `stress/known-failures.txt:10-18` | **shrinks only** (currently empty) |
| R-12 | compiler-localized wording | cgen's unsupported-construct messages pinned to the catalog's Chinese | `locale_test.go:17-23`, `feature_test.go:647+` | must stay exact |

R-1…R-3 are the only *numeric* ratchets; the rest are exact-value contracts. R-11 is empty on purpose
and the file says so twice (`stress/known-failures.txt:16-18`), which is what makes it a ratchet rather
than a TODO list.

---

## 5. Worked example

Four real failures, decomposed into interface + composition instead of a red project.

### 5.1 The Windows formatting gate — `qkfmt -l` under `windows × pwsh × CRLF`

**The interfaces**

| # | Interface | Promise |
|---|---|---|
| A | the `qkfmt` scanner/formatter | changes whitespace only; a second pass is byte-identical (idempotent); `-l` lists files whose bytes differ from the formatted bytes and exits 1 (`QuarkLangQkfmt/main.go:431-432`) |
| B | the *step* `qkfmt -l $(git ls-files '*.qk')` | a POSIX-shell command substitution, evaluated by the CI runner's shell |
| C | `git checkout` on a Windows runner | produces the working-tree bytes the other two read |

**The composition**: `C = A × windows × pwsh × CRLF checkout`.

**Where each promise fails.** Measured on 2026-10-10 at `QuarkLangQkfmt@2396137`; both are shell/EOL
properties, so they reproduce wherever those factors appear:

| Factor | Observation | Promise that breaks |
|---|---|---|
| A under CRLF | `qkfmt -l` on `fn main(IOStream io) {\r\n    io.println("hi");\r\n}\r\n` prints the path and **exits 1**; the identical text with LF exits 0. `qkfmt` **normalises CRLF → LF** in its output (`od -c`: 47 bytes, `\n` only, for both inputs) | A's "list the files that are not formatted" — in a CRLF working tree every file is reported, because the tool's canonical output is LF |
| B under pwsh | `$(...)` is a PowerShell subexpression, and pwsh does not run `git ls-files` as a command substitution | B's "produce the file list" — the step needs `shell: bash` to work at all |
| C | `.gitattributes` has **no `text`/`eol` rules** | C's "stable checked-out bytes" — nothing makes a Windows checkout match a Linux one |

**The second-order effect that matters more than the first**: `qkfmt -w` on that CRLF file rewrites it to
LF (verified — the recheck then exits 0). So the obvious local fix on Windows, running the formatter's
write mode, converts every `.qk` file to LF and dirties the whole working tree in the opposite
direction. The tool has no `--eol`/`--crlf` switch: its flags are exactly `-w`, `-l`, `-d`
(`QuarkLangQkfmt/main.go:362-366`).

**Status of the promise today**: not violated in CI, because the step **does not exist**. No workflow in
this repository invokes `qkfmt` (only `scripts/bench-tools.sh:26,60` builds and benchmarks it), and the
formatter's own CI runs `go build ./... && go vet ./...` on `ubuntu-latest`. The step in the formatter's
README is a *suggestion* that nothing composes. Stated precisely:

> Interface A promises idempotent, whitespace-only formatting for LF input. Under composition
> `windows × pwsh × CRLF`, A's `-l` mode reports every file as unformatted and A's `-w` mode silently
> converts the tree to LF. No composition in this repository currently runs A, so the defect is
> **latent rather than active**.

### 5.2 The Chinese-locale lookup gap — `Localizer.T` under the `zh` composition

**The interface**: `Localizer.T(template, args...)` promises to render the catalog's entry **for the
localizer's own language**, falling back to the template itself and counting a miss when there is no
entry (`internal/i18n/i18n.go:194-213`).

**The composition**: `C = Localizer.T × zh × {English-source catalog entries}`.

**What failed**: until `43bbff6`, `T` gated the lookup on `L.Lang() == EN`. Every entry migrated to an
English-source template therefore returned its call-site template verbatim under `QK_LANG=zh`. The
catalog held 459 entries and 105 of them carried a Chinese rendering different from the template; all
105 were **unreachable through `T(zh)`**, while `Catalog.Lookup(key, ZH)` answered correctly. The
translation existed and was never reached. Two freshly added depth diagnostics were among them, so
`LANG=zh_CN.UTF-8 ./quark deep_parens.kq` printed English (commit message of `43bbff6`).

**Why the composition was not covered**: the *premise* was encoded in a test. `TestChineseLocalizer
RendersTemplates` asserted that Chinese rendering "needs no translation and no misses" — i.e. the test
pinned the bug as expected behaviour. The direction of each entry was also being read off the *table*
rather than off the *key* (`6a37193`), which put the English value behind the Chinese one for exactly the
migrated entries. The repaired composition is now pinned by
`TestTRendersTheCatalogForTheRequestedLanguage` (`coverage_test.go:294-317`), which renders **all 918
(template, language) pairs** through `T()` and compares each against the catalog — it fails for those 105
flips — and by `TestCatalogDirectionFollowsTheSourceLanguage` (`:319`), which pins direction-per-key.

**The residual gap, stated as interface + composition**: the miss counter is symmetric now
(`i18n.go:206-207`), so an English-source template rendered under `zh` *is* observable — but nothing
makes it fail, because R-1 permits 124 unregistered wired templates. So the promise "under `zh`, a wired
template renders Chinese" holds **per template that has an entry**, and is **not** a property of the `zh`
composition as a whole.

**A second, sharper gap in the same composition — measured, not inferred.** The catalog-validation
scanner walks this repository only and stops at any nested `.git`
(`internal/i18n/coverage_test.go:44`, `:355-361`). The parser module is not in this repository at all — it
resolves from the module cache — so its message templates are outside every gate. I probed it on
2026-10-10 by parsing the pinned module's Go sources with `go/ast` and asking the catalog about each
template:

| Probe | Result |
|---|---|
| literal templates passed to `msg(` in `qkparser` (`macro.go:48-326`, `parser.go:72`, `token.go:197`) | **19 distinct**; all 19 **present** in the catalog |
| their direction | 19 Chinese-source, 0 English-source |
| behaviour if one were *absent* | renders the Chinese template **verbatim to an English reader** and increments the miss counter — verified: `T` under `EN` returned the Chinese text and `MissTotal()==1` |

So the current state is correct and **unverified by any gate**: the parser module's catalog coverage holds
because someone transcribed 19 templates by hand, and a parser-module version bump adding a 20th would
produce Chinese output on an English machine with nothing failing. That is the precise statement, and it
is a different statement from "i18n is broken".


### 5.3 The macro re-parse path — `joinTokens` under `qkc × {program with a macro and a string}`

Both engines expand macros with the *same* function (`lang.ExpandMacros`, shared through the parser
module), so the expansion itself is identical. They differ in how the **result reaches the parser**:
the interpreter keeps it as a token stream and parses that, while the compiler serialises it back to
source text — and that serialisation, not the expansion, is where the program changes.

```
 interpreter (internal/lang/compile.go:118-128)      compiled (compiler/macros.go:44-64)
   Lex → SplitMacroDefs → ExpandMacros("explain")      Lex → SplitMacroDefs → ExpandMacros("compile")
   → Parse(tokens)                          ← tokens   → joinTokens(tokens) → source text → Parse again
```

**The interface that breaks**: `joinTokens` (`compiler/macros.go:13-33`) documents its promise as
"rebuilds the token stream into source text (space between identifiers/numbers/strings, punctuation
stays tight)". It reconstructs each token from `Token.Text` (`macros.go:20,29`). But the lexer stores a
string token's **contents, not its spelling** — `qkparser/token.go:486` builds `Token{Kind: TStr,
Text: string(sb)}` from the decoded bytes, so the quotes are not in `Text` and `joinTokens` cannot put
them back. Measured directly at `compiler/macros.go:13` on 2026-10-10:

```
"io.println(\"FROM-COMPILE\")"  ->  "io.println(FROM-COMPILE)"     // quotes gone
"io.println(\"run-line\")"      ->  "io.println(run-line)"         // quotes gone
"int a = 1;"                     ->  "int a=1;"                      // fine
```

**The composition**: `C = qkc × {program with a #macro} × {string literal in the expanded region}`.

**What it does to real programs** (both engines built from this revision):

| program | interpreter | `qkc -run` |
|---|---|---|
| `examples/macro.qk` (**committed in this repository**) | `42` / `42`, exit 0 | two empty lines, exit 0 |
| `#macro id (x) { #return x }` + `io.println("hello world")` | `hello world`, exit 0 | `ParseError: expected ')', got identifier at line 1, col 39`, exit 1 |
| the same program with no `#macro` anywhere | — | `hello world` (the fast path at `macros.go:46-48` returns the source untouched) |

**Why the corpus cannot see it**: `compare.sh:23` compares `compiler/testdata/cases/*.kq` and
`cases_run/*.kq`, and neither directory contains a `#macro`
(`grep -l '#macro'` matches **only** `examples/macro.qk`, which compare.sh does not run).
`TestVMEqualsTreeWalkerOnCorpus` (`vm_diff_test.go:160`) *does* run `examples/`, but it compares the
interpreter's two engines, so `examples/macro.qk` is green there. The committed example that
demonstrates the feature is the one file that fails, and it fails outside every gate.

**Three interface-level consequences, stated separately from the defect**:

1. `joinTokens`'s promise ("rebuilds the token stream into source text") is **not implementable from
   the information it receives**. `Token` (`qkparser/token.go:94-100`) carries `Kind`, `Text`, `Int`,
   `Flt`, `Line`, `Col`; for `TStr` the spelling is genuinely absent, and for `TInt`/`TFlt` the decimal
   spelling is absent too (`Int`/`Flt` are the parsed values). Any re-parse path built on it is lossy
   for literals, not only for strings — `1_000`, `0x1F` and `1e3` would be re-emitted in canonical form
   by luck rather than by contract. **UNVERIFIED**: I did not test whether a hex or exponent literal in a
   macro argument changes meaning, because the string case already fails first.
2. `expandMacros` **swallows every error** — `macros.go:47,51,55,58,62` each `return src, nil` when
   `Lex`, `SplitMacroDefs` or `ExpandMacros` fails, "let cgen report the error". Combined with (1) this
   is why the observed diagnostics are a `ParseError` about a `)` or an `undeclared identifier
   "COMPILE"` rather than anything that names macro expansion. The failure is reported by an interface
   that is not the one that failed.
3. The two engines differ in **mode** as well as in strategy — `"explain"` vs `"compile"`
   (`internal/lang/compile.go:123`, `compiler/main.go:334,441`), which is §6.6. Two independent
   divergences sit in the same 40 lines.

**Fixed** (this branch): the compiler no longer serialises the stream. `expandMacros` returns
`[]lang.Token` and cgen parses it through `lang.CompileTokensWithImports`, which merges imports at token
level and shares the text path's one import walk (program-declaration rule, line mapping, SrcMap), so
diagnostics did not move; `joinTokens` is gone, and so is `isSymStart`. Consequence 2 is fixed with it:
every macro failure is returned as a macro error (`#error("…")` reports itself, a malformed `#macro`
names the definition, a wrong argument count names the macro), and only a lex failure still defers to the
parser, which reports the identical `LexError`. Consequence 3 is **not** touched — the mode question in
§6.6 stays open.

One correction this work measured, because the table above attributes it to `joinTokens`:
`examples/macro.qk` printing two empty lines under `qkc -run` is the **state** divergence, not the
serialisation. Its body is `#when (run)` only; with a string-free copy of the same macro
(`#macro add (a, b) { #when (run) { #return a + b } }` + `io.println(add(10, 32));`) the interpreter prints
`42` and `qkc -run` prints one empty line, and no `TStr` token exists in that program. Removing the
serialisation cannot change it; only the §6.6 decision can.


### 5.4 `TestMemoryCompactReclaims` — the memory manager's invariant under `{windows runner} × {goroutine scheduling}`

**The interface**: `MemoryManager.Compact()` promises to reclaim every block nobody occupies and return
the reclaimed count (`internal/lang/eval.go:292-305`). **The composition**: the program below, run under
the test harness, on a given OS.

**What the test asserts** (`internal/lang/eval_test.go:348-375`):

```go
// the function block of merge (owner=pid) has been reclaimed by compact;
// the thread's own persistent block (owner=0) is kept → 1 remains
if n := in.mem.BlockCount(); n != 1 {
    t.Fatalf("expected 1 block (persistent thread block) after compact, got %d", n)
}
```

**The race the assertion rests on**: `taskm.spawn()` allocates the thread's block with `owner=0`
(`eval.go:3353`), and each `t.merge(...)` allocates a second block owned by the task's pid
(`eval.go:3374`). That second block is released by `ReclaimTask(t.Pid)` (`eval.go:3384`) — which runs
**inside the task's own goroutine** (`eval.go:3381-3388`), not on the caller's path. The interpreted
program calls `GlobalMemory.compact()` as its last statement. So the block count the test reads is
"1 if the worker goroutine finished before the program did, 2 if it did not" — and nothing in the
program or the harness orders those two events.

**Observed, not hypothesised**: `.github/workflows/ci.yml`'s `tests` job on `windows-latest` failed with

```
--- FAIL: TestMemoryCompactReclaims (0.00s)
    eval_test.go:373: expected 1 block (persistent thread block) after compact, got 2
```

(run [38061536413](https://github.com/QuarkLangCommunity/QuarkLangQkc/actions/runs/38061536413/job/114240627738),
2026-10-10, on a documentation-only pull request — so no code change caused it). The same job passed on
the re-run of the identical SHA, and passes locally: `go test ./internal/lang/ -run
TestMemoryCompactReclaims -count=200` is green on linux. **UNVERIFIED**: I did not measure the
scheduling difference that decides it; the plausible reading is a coarser scheduler quantum on the
runner, but that is a hypothesis, not a measurement.

**Why this belongs in a composition map rather than a flake report**: the promise
"`Compact()` reclaims every unoccupied block" is true. The *test's* claim — "exactly one block remains,
therefore the merge block was reclaimed first" — is a claim about **goroutine scheduling**, and that is
a property of a composition (harness × OS × load), not of the memory manager. `BlockCount()` is
deterministic only after the task has provably ended; nothing in `TestMemoryCompactReclaims` establishes
that. This also refines the interface entry: what `eval_test.go:372` pins is
"`Compact()` + *a completed task* ⇒ 1 block", not "`Compact()` ⇒ 1 block".

The two other memory-manager assertions are not exposed to this race: `eval_test.go:708` uses `delete`
followed by `compact` with no concurrent task, and `bench_test.go:85,130` drive `Alloc` directly.

**Fixed** (this branch): the test's program now waits for the task before it compacts —
`taskm.block(t.pid())` returns only after the goroutine has closed its done channel, which happens after
`ReclaimTask` (`eval.go:3381-3388`), so the wait is ordering rather than a sleep and the strict assertion
(`n != 1`) is unchanged. Reproduced and closed locally with a widened window: a worker that keeps running
after `ch.send` gives `blocks=2` without the wait and `blocks=1` with it, and the committed test is green
under `-count=200` (also `-count=2000`).

---

## 6. Known gaps

### 6.0 Cross-checked against the three primary sources

| Source | State at `67ce88c` |
|---|---|
| `spec/STANDARD.md` §11.1 | `bit`, `bits<N>` (1≤N≤32), `uchar`, the bitwise family, `b[i]` read/write, the three conversion meanings, `===`, permission references, `&lvalue` and `p++` implemented in both engines; `compare.sh` keeps the corpus identical |
| `spec/STANDARD.md` §11.2 | `bits<N>` with N>32 refused by the shared frontend with one identical message; a width takes a decimal literal only; `float`/`double` reinterpretation refused (this implementation's `float` is 64-bit where §3.2 says `float32`); layout of a `bits<N>` whose width is not 8/16/32 rounds up in compiled structs; **call depth capped at 8192** (`lang.MaxCallDepth`) and **expression nesting at 65536** (`lang.MaxExprDepth`), both enforced by explicit counters that are never a stack probe, so the three platforms answer alike |
| `spec/STANDARD.md` §11.3 | **the code does not follow the standard here**: §7.2 macro spelling (`#macro` still accepted, `$` rejected; a conforming change was rolled back); §8 `try`/`catch` still exist although the standard has no exceptions; §9 `List`/`HashTable` still built in, their count frozen by a test |
| `stress/NOTES.md` | all three findings of the first full run are **closed**; the 3 fixed defects are the front-end crash, the quadratic VM compilation, and the engines disagreeing at the recursion limit |
| `stress/known-failures.txt` | empty, with the rules for adding an entry, and the reason a stale entry is itself reported as a warning |

### 6.1 Gaps the sources state themselves

| # | Gap | Where it is stated |
|---|---|---|
| G-1 | a caller's `try` does not catch a runtime error raised inside a callee, in the compiled engine | `stress/NOTES.md:102-105` |
| G-2 | cgen's IR generation is superlinear on one deep chain: 6.0 s @20k, 29.2 s @40k, 175.5 s @80k additions, while the shared frontend is 0.15 s @80k | `stress/NOTES.md:107-112` |
| G-3 | `path_fn_10k_params` sits just inside the 10 s default bound (9.18 s, 1660 MiB peak) — deliberately *not* in the allow-list, and CI skips it | `stress/NOTES.md:116-121`, `ci.yml:49-53` |
| G-4 | macro spelling, exceptions and built-in containers still diverge from the standard | `spec/STANDARD.md` §11.3 |
| G-5 | `bits<N>` layout rounds up rather than aligning to 1 as §3.1 requires; values are exact | `spec/STANDARD.md` §11.2 |

### 6.2 Gaps found by reading the code in this pass

| # | Gap | Evidence |
|---|---|---|
| G-6 | **the two engines disagree on `#when` today**: an interpreter-only macro branch executes `#when(compile)` bodies, so `#when(compile){#return 111}` / `#when(run){#return 222}` prints `222` interpreted and `111` compiled | measured; mechanism is `compile.go:123` (`"explain"`) vs `compiler/main.go:334,441` (`"compile"`) against `macro.go:268` — full write-up in §6.6 |
| G-16 | **`TestMemoryCompactReclaims` is scheduling-dependent and has already failed on `windows-latest`** (`expected 1 block … got 2`) while passing on the re-run of the same SHA and locally | measured; `eval_test.go:373` vs the goroutine at `eval.go:3381-3388`; write-up in §5.4 — **closed** (this branch): the program waits for the task (`taskm.block`) before it compacts, the assertion stayed strict, `-count=200` green |
| G-14 | **`joinTokens` loses string quotes, so `qkc` fails or prints nothing for any program with a `#macro` and a string literal** — including the committed `examples/macro.qk` | measured; `compiler/macros.go:13-33` vs `qkparser/token.go:486`; write-up in §5.3 — **closed** (this branch): `joinTokens` is gone and the compiler parses the expanded tokens; string and escape cases agree in both engines, covered by `compiler/testdata/cases/m_macro_literals.kq` and `compiler/macros_test.go`; the empty output of `examples/macro.qk` is the G-6 state question, not this one |
| G-15 | `expandMacros` swallows all five error paths (`macros.go:47,51,55,58,62`), so a macro-expansion failure is reported by the parser as a confusing `ParseError` | `compiler/macros.go:46-63` — **closed** (this branch): `expandMacros` returns its errors; only a lex failure still defers to the parser, which reports the identical `LexError` |
| G-7 | **the formatter is not in this repository and no gate composes it** | `qkfmt` referenced only by `scripts/bench-tools.sh:26,60`; `QuarkLangQkfmt` CI is `go build && go vet` on ubuntu |
| G-8 | the parser module's templates are outside every i18n gate (correct today, unenforced) | probe in §5.2; scope of `coverage_test.go:44` + `:355-361` |
| G-9 | `TestVMCompilesHotShapes` is a 4-entry whitelist, so "the VM still engages" is checked for 3 shapes only | `vm_diff_test.go:137-141` |
| G-10 | the Go-level parity tests skip themselves when `llvm-as`/`lli` are missing, so a toolchain-free machine reports green | `parity_test.go:38-40,71-73` |
| G-11 | dual-path parity is a linux-only CI step, and its script hardcodes `/tmp` paths | `ci.yml:118-121`, `compare.sh:17-18` |
| G-12 | four hand-maintained lists of "the toolchain" | `ci.yml:124`, `ci.yml:160`, `install.sh:32`, `scripts/build-release.sh` |
| G-13 | a known-failure entry needs a committed reproduction, or a command that materialises the input when it is too large to commit — `stress/repro/` holds exactly one such script | `stress/known-failures.txt:10-18`, `stress/repro/expression_chain_stack_overflow.sh` |

### 6.3 Unverified in this pass

| Item | Why unverified |
|---|---|
| any behaviour on **windows** or **macos** | this analysis ran on linux (Fedora, Go 1.26.9, clang 22.1.8); the CI matrix is the only place those compositions execute |
| that the CI matrix is green at `67ce88c` on all three OSes | no run was triggered for this document; `gh pr checks` on the pull request is the first real evidence |
| the stress corpus results (27 cases, both engines) | the suite is bounded by wall clock and RSS and was not executed here; `stress/REPORT.md` is the committed measurement |
| `bits<N>` compiled layout rounding | stated by §11.2, not re-measured |
| `dist/`, `quark`, `lang.test`, `qkcheck.test` in the main checkout | gitignored build products, not part of the analysed revision |

### 6.4 Contradiction · the memory-manager doc comment vs the standard

`internal/lang/value.go:750-753` says of the `memory` type:

> "block-managed; spec §14. v0.1: **memory is backed by the host Go GC**; compact() returns nothing
> (no value) and BlockSize is the dynamic block-granularity setting (memory.setBlock(n))."

`spec/STANDARD.md:263` says:

> "QuarkLang has **no garbage collector**. Storage is released explicitly and the release is
> deterministic: nothing is collected behind your back…"

These cannot both describe the same mechanism. The code has a real block manager
(`MemoryManager`, `eval.go:229-354`) driven by `memory.setBlock`/`compact`/`Fragmentation`
(`eval.go:1128`, `:2388-2400`), so the standard's model is the one implemented and the "backed by the
host Go GC" sentence is a stale v0.1 note. Related drift: the same comments cite **`spec §14.1` and
`§14.2`** (`eval.go:199,2130,2400`, `internal/lang/value.go:720,751,764`), and `spec/STANDARD.md` has **12 sections** —
its change log records that §11 became §12 and §10 became §11 at version 0.7.2. **Stated, not resolved.**

### 6.5 Contradiction · the CLI help text vs the process default

`cmd/qkcheck/main.go:10`, `cmd/qkdoc/main.go:9` and `cmd/qkrepl/main.go:7` all advertise
"`--lang` Output language zh|en (**default Chinese**; QK_LANG also works)".
`internal/i18n/i18n.go:65` implements `Detect`'s last line as `return EN`, commented
"global default: the documentation is English-first".

Both statements are live, and they are reconcilable only because `Detect` consults `QK_LANG` and then
`LC_ALL`/`LC_MESSAGES`/`LANG` before that default (`i18n.go:56-66`) — so the effective default is
"whatever the environment says, else English", which is what neither statement says. The history shows a
deliberate regime change the help text did not follow: `aba7e20` (2026-09-27) made the default Chinese
and read **only** `QK_LANG`; `981abe1`/`6234e82` (same day) made the language follow the system locale
with English as the default; `65495e4` (same day) rewrote the message layer. **Stated, not resolved.**

### 6.6 Contradiction · `#when` is unspecified, and the two engines disagree about it

`spec/STANDARD.md:211-233` (§7.2) defines the runtime directives as `#expand`, `#run` and the conditional
family `#ifdef`/`#ifndef`/`#if`/`#ifn`/`#endif`, with the worked example using `#ifdef COMPILE` /
`#ifdef EXPLAIN`. **`#when` does not appear anywhere in the standard.** It exists only in the
implementation, where its grammar is fixed by the refusal message `#when (compile|run)`
(`macro.go:255`, catalogued at `internal/i18n/table_macro.go:18`).

Behaviourally, on a macro carrying both branches and returning a different constant from each —
reproduced 2026-10-10 with both engines built from this revision:

```
$ cat when3.qk
#macro which () {
    #when (compile) { #return 111 }
    #when (run) { #return 222 }
}
fn main(IOStream io) { io.println(which()); }

$ ./quark when3.qk        → 222
$ ./qkc -run when3.qk     → 111
```

The mechanism is unambiguous: `expandBody` fires a branch when `args[0].Text == mode`, with
`mode == "explain"` also firing `"run"` (`macro.go:268`); the interpreter passes `"explain"`
(`internal/lang/compile.go:123`) and the compiler passes `"compile"` (`compiler/main.go:334`, `:441`).
So `#when(compile)` selects "not the compiler" rather than "compilation", and the interpreter — which the
rest of the system calls the semantic reference — is the engine that deviates from the message it prints.

Consequences stated as interfaces:

- `compare.sh`'s own promise ("same source, byte for byte", `compare.sh:2`) holds for the committed
  corpus (measured: 32/32 identical) and **not** for this program.
- The corpus cannot see it: `grep` for `#when` finds `#when (run)` in `examples/macro.qk:2` and in
  `eval_test.go`, `macroinsert_test.go`, `stressgen.go` — **no `#when(compile)` anywhere outside
  `eval_test.go:573-590`**. That one interpreter test asserts the *rule* ("`#when(compile)` blocks are
  dropped at run time", `eval_test.go:573`) using a body whose first branch is `#return`-shaped, so when
  it runs it takes the `run` branch and prints the expected `run-line`. The test passes under a mode that
  does not implement the rule it names.
- No engine validates the mode set against anything: an argument outside it fires no branch and is
  dropped **silently** — verified, `#when(bogus){#return 999}` alongside `#when(run){#return 222}`
  prints `222` with no diagnostic.

**An abandoned fix attempt exists, and it has not been carried forward.** The untracked file
`compiler/macros_test.go` in the developer's checkout (mtime 2026-10-08 22:51, ~40 minutes after the
committed tip `d3ee8cc` of that branch) is described in its own header as "the compiler expands macros in
the compile state, the interpreter in the run state" and adds a test named
`TestExpandMacrosCompileStateNoWhenBranchIsHardError`, which would make the compiler **refuse** a macro
whose `#when` branches all miss — turning today's silent divergence into a hard error. It cannot run now:
it imports `quarklang/internal/lang`, and that module path was renamed to the public
`github.com/QuarkLangCommunity/...` one by `6dad203` (2026-10-09 19:47), after the file was written.
`go test ./...` in the compiler module does not see it (untracked), so the suite stays green. So the
intended fix was *diagnosed before this document*; the choice among the three readings below was the part
left open.

**UNVERIFIED**: whether the intended fix is to pass `"compile"` from the interpreter, to retitle the mode
set as `explain|run`, to drop `#when` in favour of the standard's `#ifdef` family, or the abandoned
attempt's "hard error when every branch misses". All are consistent with *some* artefact in the tree,
which is why this is reported as a contradiction rather than a defect with a known fix.

---

## 7. Relationship to the other documents

This document adds a description of interface boundaries and does not restate normative content:
`spec/STANDARD.md` remains the source of truth for the language, and `stress/NOTES.md` /
`stress/REPORT.md` for the measured robustness evidence. Where this document and those disagree, they
have authority and §6 records the disagreement.
