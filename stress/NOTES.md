## Notes

### What this slice contains

Slice 1 of the robustness suite: the extreme-input generator, the bounded runner, the measured report,
the known-failure list and — since the first full run of the suite — a CI job that runs a bounded subset
of it on linux, macos and windows. The three defects that run found are fixed, so the list of known
failures is empty on purpose, and the section below records what each one was and how it was closed.

### Closed findings

The first full run left three entries in `known-failures.txt`. All three are fixed, their reproductions
under `stress/repro/` now pass, and the corresponding corpus cases run as ordinary cases. What follows is
the measured evidence each fix rests on, on the machine in the report above.

**`scale_line_1mb` — the front end crashed instead of reporting a position.** The type checker recurses
once per expression node, so a single source line of ~524 000 chained additions exhausted the Go stack
(`runtime: goroutine stack exceeds 1000000000-byte limit`, exit 2, in both engines). Measured on the base
commit with the front-end-only probe: 290 000 nodes survive, 320 000 die — the input, not a policy,
decided when the process went down.

Both front ends now count their own nesting and refuse one level past `internal/lang.MaxExprDepth`
(65 536, one constant for every platform and both engines):

- the type checker refuses an over-deep *expression* (`CompileError: expression nesting is deeper than
  65536 levels …` at the node's line);
- the parser refuses an over-deep *nest*, which the checker never sees: 300 000 nested parentheses used
  to crash the same way through the recursive descent (`ParseError: nesting is deeper than 65536 levels
  …`).

65 536 is deliberately far below the measured crash depth (a 4.5x margin, so reaching the limit is always
possible) and far above anything a program writes — the largest expression in this corpus is 10 000
chained calls, and ten thousand nested parentheses and blocks still parse and type check.

The case now reports `REJECT_AGREE`: both engines refuse the 1 MiB line in ~0.34 s with a source position
and exit 1, instead of taking the process down after ~0.7 s.

**The quadratic VM compilation behind it.** `(*vmCompiler).typeOf` re-derived a node's type by walking
its whole subtree, and the VM compiler calls it once per node, so a left-deep chain cost O(n²) visits.
The answer for a node cannot change while one function is compiled, so it is cached per node now. Chain
of N additions in one expression, interpreter end to end, both builds with the front-end cap raised so
that the larger sizes are reachable:

| chained additions | before | after (per-node type cache) |
|---:|---:|---:|
| 20 000 | 1.35 s | 0.075 s |
| 40 000 | 4.11 s | 0.134 s |
| 80 000 | 18.42 s | 0.271 s |
| 120 000 | 52.07 s | 0.348 s |
| 160 000 | 109.80 s | 0.504 s |

Before: 8x the input costs 81x the time (quadratic). After: 8x the input costs 6.7x (linear), and the
160 000-term chain is 218x faster. With the shipped cap, 80 000 and above are refused in ~0.15–0.19 s,
which is the intended answer for an input that size. `BenchmarkVMCompileChain` (20 000/60 000 additions)
keeps the linearity under watch: 39 ms / 74 ms here.

**`path_recursion_stack` and `path_recursion_tail` — the engines disagreed at the recursion limit.** The
interpreter refuses a call made from a frame 8192 levels deep (`StackOverflowError: recursion depth
exceeded 8192`); the compiled binary ran 1 000 000 levels of non-tail recursion to completion and, worse,
turned a self tail call into a loop so that the depth never grew at all.

The generated code now counts the same levels with a thread-local counter
(`internal/lang.MaxCallDepth`, `@ql_depth_enter` / `@ql_depth_leave`), reports the shared message at the
call site the caller recorded (the interpreter reports the refused *call*'s position, not the callee's),
and exits 1. The LLVM tail call is gone on purpose: a reused native frame cannot carry the counter, so
keeping the optimisation would have left the tail case bypassing the limit. Recursion is bounded by the
cap now, so the native stack saving was worth nothing.

Measured at the boundary, both engines, byte for byte:

| program | interpreter | compiled binary |
|---|---|---|
| `down(8191)` | `33550336`, exit 0 | `33550336`, exit 0 |
| `down(8192)` | `error: StackOverflowError: recursion depth exceeded 8192 at line 3`, exit 1 | identical |
| `up(8192, 0)` (tail) | `error: StackOverflowError: recursion depth exceeded 8192 at line 3`, exit 1 | identical |
| `up(9000, 0)` (tail, the reproduction) | `error: … at line 12`, exit 1 | identical |

Both corpus cases now report `AGREE`, and `compiler/testdata/compare.sh` fails if the two engines ever
disagree about the limit again (`cases_run/p_recursion_*.kq`).

One measured detail behind the guard's shape: a function that cannot call carries no counter, and that is
exact — a frame that makes no call is always the innermost one and has returned before the next call, so it
can never be the *caller* of a refused call, and the counter holds the same value at every refusal point
with or without it. The exemption matters for cost: the `res_few_mb` corpus case is 50 000 arithmetic-only
functions, and guarding each of them makes every one of them a non-leaf function for clang. Measured on
`res_few_mb` (IR generation, then clang `-O3 -flto=thin`, then `qkc -run`):

| build | IR bytes | clang | `qkc -run` | peak RSS |
|---|---:|---:|---:|---:|
| before the guard | 10 147 430 | 3.46 s | 4.51 s | 739 MiB |
| guard on every function | 13 149 471 | 6.05 s | 7.29 s | 1133 MiB |
| guard except call-free bodies (shipped) | 10 149 471 | 3.40 s | 3.96 s | 738 MiB |

The predicate is a whitelist over the lowered body — arithmetic, comparisons, short-circuit logic, reads,
assignments, declarations and branches over those; a call, an interface dispatch, an index, a table, a
conversion, a print, a loop, a merge or anything the walk does not recognise counts as a call — so a node
kind added later can only widen the guard, never narrow it. `cases_run/p_recursion_with_leaf_call.kq` is the
parity case that keeps it honest: the compiled counter must refuse the mixed shape at exactly the depth the
interpreter refuses.

**What is still different, and why it is not this fix.** A runtime error raised *inside a callee* is not
caught by the caller's `try` in the compiled engine: `try { g(); } catch (void e) { … }` is caught by the
interpreter and terminates the compiled binary. That is a property of the compiled error model — `fn g()
int { return 10 / 0; }` behaves the same way — and the interpreter is the language reference, so it is a
real (pre-existing) divergence, reported as one rather than hidden here.

**Still open, measured while fixing this.** `qkc`'s own IR generation is superlinear on one deep chain:
6.0 s at 20 000 additions, 29.2 s at 40 000 and 175.5 s at 80 000 (the shared front end is 0.15 s at
80 000), so the cost is in lowering, not in the checker. This fix does not touch it; the 1 MiB case no
longer reaches it, because the front end refuses the input first. Out of scope here, reported to the
maintainer as a follow-up finding.

### Near misses and non-failures worth knowing

| case | measurement | reading |
|---|---|---|
| `path_fn_10k_params` | `qkc -run` 9.18 s across runs, 1660 MiB peak RSS | 10 000 parameters sits *just* inside the 10 s default bound. On a slower or busier machine this case will be reported as `TIMEOUT`, and it is not in the known-failure list on purpose: that report would be a real finding about the bound, not a flake to silence. It is also why the CI job passes an explicit `-timeout 90s` and skips this case (see `.github/workflows/ci.yml`), and why the committed report was produced with `-timeout 60s` |
| `res_few_mb` | interpreter 0.42 s, `qkc -run` 3.96 s, 738 MiB peak | a 4 MB valid program is comfortably within the bounds; it is skipped in CI because compile-and-link cost, not a language property, dominates it (see the table above for what guarding every one of its 50 000 functions would cost) |
| `scale_call_chain` | `qkc` IR generation 3.17 s | 10 000 chained calls are fine at the front end, costly at codegen |
| `path_exp_macro` | rejected by both engines in under 1 ms | macro expansion is single pass (M1), so the doubling chain cannot blow up; the case measures a design property, not a defect |
| `path_recursive_macro` | rejected by both engines | mutual macro recursion terminates; only the wording differs |
| `scale_nested_parens`, `scale_nested_blocks` | 10 000 levels accepted by both engines | nesting is not what breaks the parser — but it *was* what broke the Go stack at 300 000 levels, which is why the parser now carries the same nesting limit as the checker |
| `bad_nul_byte`, `bad_crlf_mixed` | accepted by both engines, outputs identical | a NUL inside a string literal and line endings mixed between CRLF and LF are handled |
| `bad_bom` | rejected by both engines, identical wording | a UTF-8 byte-order mark is a lex error ("unexpected character"), which is worth knowing because Windows editors add one |
| `bad_invalid_utf8` | rejected by both engines, identical wording | invalid UTF-8 in an identifier or a string literal is a clean lex error, not a crash |

### Where the suite runs in CI

`.github/workflows/ci.yml` has one `stress` job on `ubuntu-latest`, `macos-latest` and
`windows-latest`. It builds both engines into `stress-out/`, materialises the corpus and runs
`qkstress run` with an explicit `-timeout 90s`, `-skip 'path_fn_10k_params|res_few_mb'` (the two cases
whose cost is dominated by size: 9.2 s against a 10 s default bound with a 1.6 GiB peak, and a 4 MB
program) and the tracked known-failure list. Every other case runs on every platform, and a case that
reports `PANIC`, `TIMEOUT`, `MEMCAP`, `ACCEPT_MISMATCH` or `DIVERGE` fails the job.

The peak-RSS probe reads `/proc` on Linux only; on macOS and Windows `processGroupRSSKB` reports -1
("unavailable", rendered as `-`) and the memory ceiling simply cannot fire. The job is built to pass
without `/proc`: it asserts exit statuses, output parity and the wall-clock bound, which are the
platform-independent properties the suite is about.

### Interaction with the existing whole-repo lint gate

`cmd/qkcheck`'s corpus gate (`TestCLICorpusNoFalsePositives`) walks the repository and lints every `.qk`
file it finds, *including untracked and gitignored directories*. With this suite's generated corpus on
disk that walk reaches `stress-out/cases/scale_line_1mb.kq` and used to kill the linter with the same
type-checker stack overflow the engines died of. Two changes came out of that:

- the walk now skips `stress-out/`, next to the `dist/`, `dist-ci/`, `lintbench/` and `bench/` entries it
  already skips, because generated adversarial inputs are not part of the reviewed corpus;
- the linter itself survives that input since the front-end depth limits landed, so the file set is no
  longer the only thing standing between the gate and a crash.

### Found while building the corpus, not a corpus case

`List<List<int>>` is accepted by the interpreter and refused by the compiler (`variable type … is not
supported`, an explicit limitation of the compiler's type lowering). `scale_nested_generics` therefore
nests `Box<…>` arguments: the case is meant to measure depth, not to trip over an unrelated unsupported
type. Recorded here because it is a real front-end difference that the corpus deliberately avoids
depending on.

### Reproducing the numbers

Everything in this report comes from the run quoted above; the JSON beside it
(`stress-out/results.json`) carries the per-phase exit statuses, wall times, peak RSS figures and capped
stderr for every case. Regenerating overwrites the report with new timings — the machine line and the
commit are there precisely so two reports can be told apart.

The cost curves were measured by hand with the same binaries and `QK_LANG=en`, on
`fn main(IOStream io) { io.println(0+1+1+…); }` (`+1` repeated), with `MaxExprDepth` raised to
100 000 000 in both the "before" and the "after" build so that every size is reachable and the two curves
compare the same work; the shipped cap and its effect are reported separately above.
