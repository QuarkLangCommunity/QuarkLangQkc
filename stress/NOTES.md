## Notes

### What this slice contains

Slice 1 of the robustness suite: the extreme-input generator, the bounded runner, the measured report
and the known-failure list above. Go fuzz targets and a CI job that runs this suite are slice 2 and are
deliberately absent here. Nothing in this slice changes language behaviour: it adds inputs and
measurements, and it documents what it finds rather than adjusting a check until it passes.

### The cost curve behind `scale_line_1mb`

Measured by hand on the machine above, with the same binaries and `QK_LANG=en`, on
`fn main(IOStream io) { io.println(0+1+1+…); }` (`+1` repeated):

| chained additions | front end only (parse + type check) | `quark` end to end |
|---:|---:|---:|
| 20 000 | — | 2.23 s |
| 40 000 | — | 4.68 s |
| 50 000 | 0.13 s | — |
| 80 000 | — | 24.06 s |
| 100 000 | 0.26 s | — |
| 120 000 | — | > 30 s (killed) |
| 160 000 | — | > 30 s (killed) |
| 320 000 | — | exit 2, Go stack overflow after 0.70 s |

The front end is linear in the chain length while the *run* is not, so there are two separate defects
behind this one case:

1. **quadratic VM compilation.** A stack sample taken during the 80 000-term run shows goroutine 1 in
   `internal/lang.(*vmCompiler).typeOf` (vm.go:411, recursing through `expr`/`binary` at vm.go:426): the
   VM compiler re-derives the type of every subexpression by walking the subtree it belongs to, and a
   left-deep chain makes that walk cost O(n) per node.
2. **stack overflow instead of a diagnostic.** Past roughly 300 000 chained binary nodes the shared type
   checker (`internal/lang/typecheck.go`, `(*checker).inferBin` → `infer`, one frame per node) exhausts
   the Go stack, and *both* engines die with `runtime: goroutine stack exceeds 1000000000-byte limit` /
   `fatal error: stack overflow` (exit 2). The same crash is reached by the compiler while generating IR,
   because the type checker is shared. This is the loudest finding of the slice: an input the size of a
   single source line takes the whole process down instead of producing a source-position diagnostic.

### Near misses and non-failures worth knowing

| case | measurement | reading |
|---|---|---|
| `path_fn_10k_params` | `qkc -run` 8.80–9.20 s across runs, 1658 MiB peak RSS | 10 000 parameters sits *just* inside the 10 s bound. On a slower or busier machine this case will be reported as `TIMEOUT`, and it is not in the known-failure list on purpose: that report would be a real finding about the bound, not a flake to be silenced |
| `res_few_mb` | interpreter 0.42 s, `qkc -run` 3.73 s, 740 MiB peak | a 4 MB valid program is comfortably within the bounds |
| `scale_call_chain` | `qkc` IR generation 3.17 s | 10 000 chained calls are fine at the front end, costly at codegen |
| `path_exp_macro` | rejected by both engines in under 1 ms | macro expansion is single pass (M1), so the doubling chain cannot blow up; the case measures a design property, not a defect |
| `path_recursive_macro` | rejected by both engines | mutual macro recursion terminates; only the wording differs |
| `scale_nested_parens`, `scale_nested_blocks` | 10 000 levels accepted by both engines | nesting depth is not what breaks the parser — the left-deep *chain* shape is |
| `bad_nul_byte`, `bad_crlf_mixed` | accepted by both engines, outputs identical | a NUL inside a string literal and line endings mixed between CRLF and LF are handled |
| `bad_bom` | rejected by both engines, identical wording | a UTF-8 byte-order mark is a lex error ("unexpected character"), which is worth knowing because Windows editors add one |
| `bad_invalid_utf8` | rejected by both engines, identical wording | invalid UTF-8 in an identifier or a string literal is a clean lex error, not a crash |

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
