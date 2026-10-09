# QuarkLang stress / robustness suite (抗压测试)

Extreme inputs, a bounded runner and a measured report: the suite answers "what does the implementation
do when the input is absurd?" for the interpreter (`quark`) and the native compiler (`qkc -run`) at once.

Two rules shape everything here:

1. **Nothing is weakened to make a case pass.** A panic, a hang, a runaway allocation or a disagreement
   between the two engines is a *finding*: it is written down in [known-failures.txt](known-failures.txt)
   and in [REPORT.md](REPORT.md), never deleted or skipped.
2. **Nothing is generated into the repository.** Cases, binaries, caches and intermediate output land in
   the gitignored `/stress-out/` tree; only the generator, the runner, the report and the known-failure
   list are tracked.

## Tracked vs generated

| Path | Tracked | Contents |
|---|---|---|
| `internal/lang/stressgen/` | yes | the corpus: 27 deterministic cases, one function per case |
| `cmd/qkstress/` | yes | `gen` (materialise the corpus) and `run` (bounded execution + report) |
| `stress/REPORT.md` | yes | the measured report of a real run |
| `stress/known-failures.txt` | yes | documented defects the suite knows about, with minimal repros |
| `stress/repro/` | yes | the smallest source that still shows a known failure |
| `/stress-out/` | no (gitignored) | generated cases, binaries, caches, temp files, JSON results |

## Running it

```sh
# 1. build both engines into the generated tree
go build -o stress-out/bin/quark .
(cd compiler && go build -o ../stress-out/bin/qkc .)

# 2. materialise the extreme corpus (gitignored, ~10 MB)
go run ./cmd/qkstress gen -out stress-out/cases

# 3. execute every case against both engines under a hard bound and print the verdicts
go run ./cmd/qkstress run -cases stress-out/cases
```

`gen` alone is enough to see the inputs:

```sh
go run ./cmd/qkstress gen -out stress-out/cases && ls stress-out/cases
```

Both commands are run from the repository root: every default path is relative to it. Nothing writes to
`/tmp`: `run` points `TMPDIR` and `QUARK_CACHE` at `stress-out/`, and each compiler case gets a cold
cache, so the measured compiler time is a full cold compile, link and run.

## The corpus

| Category | Cases | What it stresses |
|---|---|---|
| `scale` | 7 | size: a 1,000,000-character identifier, 100,000 top-level declarations, one 1 MiB line, 10,000 nested blocks, 10,000 nested parentheses, 1,000 nested generic arguments, a 10,000-call chain |
| `pathological` | 8 | adversarial shapes: stack-exhausting recursion, tail recursion, an integer literal beyond 64 bits, `bits<33>`, 10,000 parameters, 10,000 struct fields, macros that expand into each other, a doubling macro chain |
| `malformed` | 11 | empty file, comments only, unterminated string/block/parenthesis, invalid UTF-8, NUL byte, BOM, mixed CRLF, directives only, unbalanced `#ifdef` |
| `resource` | 1 | a valid ~4 MB program, for parse / codegen / link time and peak memory |

## What the runner measures

For every case it runs four bounded phases — interpreter front end (in-process acceptance probe),
interpreter run, compiler IR generation, `qkc -run` — and records exit status, wall time, peak RSS of the
whole process group, whether stderr carries a source position, and whether the two engines agree.

The verdict vocabulary:

| Verdict | Meaning | Fails the run |
|---|---|---|
| `AGREE` | both engines accepted the source and produced identical output and exit status | no |
| `REJECT_AGREE` | both engines rejected the source with identical diagnostics | no |
| `REJECT_DIFF` | both rejected it, with different wording (information only) | no |
| `DIVERGE` | at least one engine accepted it and the run outcome differs | yes |
| `ACCEPT_MISMATCH` | one front end accepted the source and the other rejected it | yes |
| `PANIC` | a Go panic or fatal error surfaced in either engine | yes |
| `TIMEOUT` | a phase exceeded `-timeout` (default 10 s) and was killed | yes |
| `MEMCAP` | a phase exceeded `-mem` (default 2048 MiB) and was killed | yes |

A "yes" is loud: the run prints the finding and exits non-zero — unless the case is listed in
`known-failures.txt`, in which case it is printed as `KNOWN` and does not change the exit status. A
defect that is not in that file always fails the run, so the list cannot be used to hide anything new.
An entry that no longer fails is reported as stale (⚠) but does not fail the run: a fixed case should be
removed from the list by the person who fixed it.

## Known failures

`stress/known-failures.txt` is both the machine-readable allow-list (`<case-name> <reason>` per line,
`#` starts a comment) and its own documentation; `stress/repro/` holds the minimal source of each entry.
The runner embeds the file verbatim in the report.
