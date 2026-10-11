# qkcheck: from hints to corrections

`qkcheck` reports diagnostics. This document is the design of the two capabilities added on top of
that report — telling the user **what the corrected form is**, and applying it with `-fix` — and the
inventory that decides which diagnostics may be corrected mechanically at all.

## 1. The rule that decides fixability

A diagnostic is **auto-fixable** when all three hold:

1. **The correct form is determined by the diagnostic alone** — there is exactly one rewrite, and no
   judgement, intent or design decision enters it;
2. **the rewrite is safe**: it cannot change what the program does, in either engine;
3. **the rewrite is idempotent**: applying it twice produces byte-identical output.

Everything else is **report-only**, and says so in the report. A diagnostic that needs a human
decision is not a defect in the fixer: it is a diagnostic whose correction is the program's design.

## 2. Inventory of QK101–QK115

| Code | Diagnostic | Range carried | Deterministic correct form? | Verdict |
|---|---|---|---|---|
| QK101 | unused local variable | Pos (line+col of the declaration) | Removing a declaration that is never read **does** change the program: the initializer may call a function, and a `delete`-tracked lifetime ends earlier. No mechanical rewrite. | report-only — needs judgement (delete the declaration, or use it, or rename to `_`) |
| QK102 | unused parameter (`-params`) | Pos of the parameter | Same: the parameter is part of the signature, and removing it changes every call site, across files. | report-only — signature change, cross-file |
| QK103 | unreachable code | Pos of the first dead statement | Deleting it looks mechanical, but the choice of *how much* to delete (one statement, the rest of the block, the enclosing branch?) is a design decision, and deletion moves every later line number. | report-only — scope of deletion is judgement |
| QK104 | shadows an outer variable | Pos of the inner declaration | Renaming the inner variable is mechanical only if the new name is free, and the name to use is the author's choice. | report-only — naming |
| QK105 | missing return path | Pos of the function | The missing value is unknown: `return 0`? `return nil`? `panic`? | report-only — the returned value is intent |
| QK106 | interface near miss | Pos of the type | Implementing the missing method is writing new code. | report-only — writes new code |
| QK107 | void result used as a value | Pos of the call | Removing the use, deleting the statement or changing the callee's return type are all defensible; which one is right is intent. | report-only — intent |
| QK108 | self-assignment `x = x;` | Pos of the assignment | `x = x` has no effect; deleting the statement changes nothing observable. | **fixable** — delete the statement line |
| QK109 | constant condition | Pos of the condition | Rewriting `if (false) {…}` to a deletion, or `while (true)` to `loop`, changes structure and line numbers; the branch may be a deliberate placeholder. | report-only — structure |
| QK110 | constant division by zero | Pos of the binary operator | The code is a *deliberate* runtime error outside `try`; the correction (wrap in `try`, change the divisor, return) is intent. | report-only — intent |
| QK111 | unused import | Pos of the `import` keyword | The import contributes no used symbol, so removing the statement cannot change what the program does (imports merge text; they have no side-effect form). | **fixable** — delete the import line, when that line holds exactly that one import |
| QK112 | local shadows a global function | Pos of the declaration | Renaming is the author's choice, as QK104. | report-only — naming |
| QK113 | dead store (overwritten) | Pos of the **earlier** assignment | When the earlier assignment is a plain assignment `x = <pure>;`, deleting it cannot change behaviour. When it is a declaration initializer (`int c = 1;`) deletion would remove the declaration; when the right-hand side calls a function, deletion would drop the call. | **fixable** — only for a pure, non-declaring earlier assignment; otherwise report-only with the reason |
| QK114 | self-comparison | Pos of the binary operator | `x == x` is `true` and `x != x` is `false` for every value the operand can hold, including `NaN` (a variable holds one value, and that value equals itself). | **fixable** — replace with the literal, when the comparison is a standalone statement whose operand evaluation cannot be dropped |
| QK115 | uncalled function | Pos of the function | The function may be intended as API, or about to be wired up. | report-only — intent |

The `-exit0` gate, the exit codes, and every diagnostic's severity are unchanged: an unfixed
diagnostic is still reported, and still fails the gate.

## 3. How a fix is expressed

The front end carries a **start position only** (line and column); the AST has no end position for a
statement or an expression. A fix is therefore expressed against the source text:

```go
type Fix struct {
    Line int    // 1-based line the fix anchors on
    From int    // byte offset in that line
    To   int    // byte offset in that line
    Text string // replacement (From==To = pure insertion)
    Delete bool // From:To is a whole-line span (From==0, To==line length): remove the line
    Note string // why this fix is applicable / not applicable, for the report
}
```

Every fix this tool applies rewrites **within one line**, and either deletes that whole line or
replaces the tail of it that the statement occupies. Two properties follow:

* the file's own line endings survive: a CRLF checkout stays CRLF (the line separator is never part
  of a rewrite, only of a deletion, and a deletion removes the separator of the line it deletes);
* line numbers of untouched lines never move, so a diagnostic elsewhere in the file keeps its
  position.

### 3.1 Applicability gates

A candidate rewrite is attached to a diagnostic only when every gate passes:

| Gate | Question | Failure means |
|---|---|---|
| shape | Does the line hold exactly this statement: leading whitespace, the statement, an optional `;`, then only whitespace and an optional comment? | the statement shares its line with something else — deleting or replacing the line would eat a neighbour: report-only |
| purity | Does the removed/replaced text consist only of identifier and member reads, with no call, assignment, index read or `delete`? | removing it could drop a side effect: report-only |
| liveness (QK114) | Is the operand's root identifier read somewhere else in the file? | rewriting the only read of a variable would turn a correct program into a QK101 finding: report-only |
| single (QK111) | Does the import line hold exactly one import? | rewriting a shared import line is possible but not needed today: report-only |
| declaration (QK113) | Is the earlier assignment a plain assignment rather than a declaration initializer? | deleting it would remove the declaration: report-only |

When two fixes target the same line, the higher-priority one is applied and the other is dropped
with that as its reason. Priority: `QK113` (delete the dead write) > `QK108` (delete the no-op
statement) > `QK111` (delete the import) > `QK114` (replace).

## 4. Modes

| Invocation | Behaviour | Exit code with findings |
|---|---|---|
| `qkcheck file` | reports every diagnostic; a fixable one prints its proposed change beneath it | 1, unchanged |
| `qkcheck -check file` | reports, **writes nothing**, exits non-zero when a fix is pending (CI form; the same as `-dry-run`, and `-dry-run` is accepted as an alias) | 3 |
| `qkcheck -fix file` | applies every applicable fix in place, then re-checks the result and reports what remains | 1 |
| `qkcheck -fix -exit0 file` | as above, always exit 0 (the report-only form) | 0 |

Exit codes: `0` no findings · `1` findings remain · `2` usage or I/O error · `3` fixes pending and
nothing was written.

`-fix` re-checks the file it just rewrote and repeats while a fix still applies, because a fix can
expose another one (deleting `x = x;` between two writes can make the mark it carried dead too).
The rewrite is bounded by a small pass limit; reaching it is reported rather than looped on.

## 5. Why behaviour preservation is provable here

The fixable set was chosen so that each rewrite removes or replaces an expression whose *observable*
effect is provably nil:

* **QK108 / QK113** remove a store or an assignment whose value is, by the diagnostic's own analysis,
  read by nobody before being overwritten. The purity gate guarantees the removed right-hand side has
  no call, so no side effect disappears with it. Both engines' observable output is unchanged.
* **QK111** removes an import that contributes no symbol the file uses. Imports merge text into the
  compiled unit; they are not executed, so nothing runs at import time and nothing else changes.
* **QK114** replaces a comparison of a value with itself. `==` on one value against itself is `true`
  (and `!=` is `false`) for every type the language has, including a `NaN` float and a reference,
  because the operand is read once per side from the same storage. The purity and liveness gates make
  sure the rewrite neither drops a read nor creates an unused variable.

The corpus proof in `internal/lang/lintfix_test.go` and `cmd/qkcheck/fixcorpus_test.go` checks all
three claims empirically over the stress corpus, `examples/` and `testdata/`: the second `-fix` run is
a no-op, every fixed program produces byte-identical output under `quark` and `qkc -run` to its
unfixed original, and no line other than an intended one changes.
