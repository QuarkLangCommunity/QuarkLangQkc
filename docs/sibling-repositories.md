# Sibling repositories: roles, hash evidence, and reconciliation

**Status:** measured and reconciled on 2026-10-17 against `origin/main` `ad7a98c` and the default branch
of every sibling repository. No sibling repository was deleted, archived, or force-pushed.

The organisation has ten standalone repositories whose names overlap with this repository's toolchain.
This note answers one question per pair: is the sibling a **mirror** (same code), an **older home** of the
same implementation (same lineage, later superseded), or an **independent implementation / different
tool**? Only a mirror would justify keeping two copies in sync; the measurement says there are none.

## 1. What was compared

For every repository listed in the task, the whole tracked tree was compared with the candidate path under
`cmd/` in this repository. Content identity means the Git blob SHA-1 of the file is equal, not that the
file name is similar. For the one source file both sides share, the line-level similarity is the
`difflib.SequenceMatcher` ratio. The Go-code digest is SHA-256 over `path\0blob\0` for tracked `.go` files,
sorted by path; it is the value frozen in `scripts/sibling-repositories.tsv`.

Only the repository name, the `cmd/` path and the measured values are recorded. Paths are relative; no
machine location is part of the result.

## 2. Per-pair result

| Pair | Sibling HEAD | Sibling files (Go) | Main files (Go) | Identical blobs | `main.go` blob (sibling / main) | Lines (sib/main) | Line similarity | Role |
|---|---|---|---|---|---|---|---|---|
| `QuarkLangQkcheck` ↔ `cmd/qkcheck` | `94e71e2` | 6 (1) | 6 | 0 of 1 compared | `55771bb2aad7eb1dcfd57c2848bd724a81bc8956` / `8f3144d2848d3737efa1ee67380e2dd4c264ca77` | 435 / 496 | 0.097 | independent implementation (superseded standalone home) |
| `QuarkLangQkdoc` ↔ `cmd/qkdoc` | `d80c880` | 6 (1) | 3 | 0 of 1 compared | `9aa85fdf477ead3a7b98fb70ba0eb341b85ac287` / `20cffd2a47b7ebbd69284e49c346d468a3c769fa` | 312 / 141 | 0.137 | independent implementation (superseded standalone home) |
| `QuarkLangQklsp` ↔ `cmd/qklsp` | `92f129f` | 9 (1) | 7 | 0 of 1 compared | `432baee488e06a53e3fee6522b77499f6a13e18e` / `6bcb69b42cfaa0d48229a77ae967ba52d03e567b` | 250 / 439 | 0.113 | independent implementation (superseded standalone home) |
| `QuarkLangQkrepl` ↔ `cmd/qkrepl` | `597e37b` | 6 (1) | 2 | 0 of 1 compared | `b0636508dae10c796cef09ca3f8f4888fa6e975b` / `78d761ea1feaea3a06a3fdb5b7660c7794d6cb19` | 170 / 204 | 0.128 | independent implementation (superseded standalone home) |
| `QuarkLangQkm` ↔ (no `cmd/qkm`) | `973a029` | 27 (5) | — | — | — | — | — | independent tool; no counterpart in this repository |
| `QuarkLangQkd` ↔ (no `cmd/qkd`) | `546b805` | 4 (1) | — | — | — | — | — | independent tool; no counterpart in this repository |
| `QuarkLangPlayground` ↔ (no `cmd/playground`) | `1acd2ee` | 21 (0) | — | — | — | — | — | independent frontend; consumes `cmd/quarkwasm`, does not duplicate it |
| `QuarkLangQuark` ↔ (no `cmd/quark`) | `10a556a` | 3 (0) | — | — | — | — | — | independent runtime service; builds from this source tree |
| `QuarkLangQkfmt` ↔ (no `cmd/qkfmt`) | `d89824d` | 9 (3) | — | — | — | — | — | independent tool; no counterpart in this repository |
| `QuarkLangQktest` ↔ (no `cmd/qktest`) | `08ac62f` | 10 (1) | — | — | — | — | — | independent tool; no counterpart in this repository |

The four legacy HEADs are the commits at measurement. The reconciliation PRs add exactly one docs-only
commit on top of each (the README pointer), so the current sibling HEAD is that merge commit; the Go-code
digests below are unchanged because those commits touch only `README.md`.

The premise that this tree carries `cmd/qkm` (and the other unmatched names) does not hold at the measured
commit: `git ls-tree -r origin/main cmd/` contains `qkcheck`, `qkdoc`, `qklsp`, `qkrepl`, `quarkwasm`,
`qki18n` and `qkstress` only.

### Content hashes for the declared pairs

```
QuarkLangQkcheck  Go digest  1cb87a8d980ab4a68cf744231fdd1c06cc9696ecc85b842bdf648d7750f2f930
                  cmd/qkcheck bc1fe00f3dc48e071dc16fde6dfe9fd700e5c7f02f1fef0e574f0e5b2a360521
QuarkLangQkdoc    Go digest  cdb12c2aaa6b93c25f43602ed35d17f6e6d6ced515faaad0981a56a33ec2afb0
                  cmd/qkdoc   8947deb3065bfc8852442bc9e2d2e4df04ab93c82d135cc66c040efcdf6a70e5
QuarkLangQklsp    Go digest  13c38e9406be0506d94fef03c91bc83b7f22baba762c4a5f3dcaa7c94a44bf24
                  cmd/qklsp   61074978b6219f1dc7b692244b3916063d3b0f267d67b5ae3de7103628232936
QuarkLangQkrepl   Go digest  75d6f7fad71df4711c7f836364d0601feb0c1329252e08fe363e337504765806
                  cmd/qkrepl  9fcdedbfd95eafa28990f3cfca4f96bfd52b97db6d47d7eae402cd7888147893
```

## 3. What the numbers say

- **Mirrors: none.** Not one tracked Go blob in the four same-named repositories exists in this
  repository's tree or history. The only common blob anywhere is the generic `LICENSE` file in
  `QuarkLangPlayground`; that is not tool code.
- **Older homes: four.** `QuarkLangQkcheck`, `QuarkLangQkdoc`, `QuarkLangQklsp` and `QuarkLangQkrepl`
  are standalone implementations of tools that `cmd/` now implements better. They share the tool name and
  the same front end (`quarklang/internal/lang` via a relative `replace`), but not the code: zero identical
  blobs, 9.7%–13.7% line similarity on the only shared source file, and no sibling blob anywhere in this
  repository's history. They are independent implementations, not stale copies; their READMEs now say so.
- **Different tools: six.** `QuarkLangQkm`, `QuarkLangQkd`, `QuarkLangPlayground`, `QuarkLangQuark`,
  `QuarkLangQkfmt` and `QuarkLangQktest` have no `cmd/` counterpart. The first two and `QuarkLangQuark`
  describe themselves as separate microservices; `QuarkLangPlayground` is a browser frontend over
  `cmd/quarkwasm`; `QuarkLangQkfmt` and `QuarkLangQktest` are standalone tools this repository only calls
  out in documentation.

## 4. Reconciliation applied

Deleting a repository was out of scope, and deleting code from the four legacy repositories would destroy
history that users may still build. The smaller honest change is to make the relationship explicit:

- Four PRs add one banner to the top of each legacy README, pointing at the maintained `cmd/<tool>` path:
  [Qkcheck #1](https://github.com/QuarkLangCommunity/QuarkLangQkcheck/pull/1),
  [Qkdoc #1](https://github.com/QuarkLangCommunity/QuarkLangQkdoc/pull/1),
  [Qklsp #1](https://github.com/QuarkLangCommunity/QuarkLangQklsp/pull/1),
  [Qkrepl #1](https://github.com/QuarkLangCommunity/QuarkLangQkrepl/pull/1).
  Code and history are untouched; no repository was archived or deleted.
- The six independent repositories were left untouched: there is no duplication to reconcile.
- This repository is the single source for the four tools under `cmd/`.

## 5. The drift check

`scripts/sibling-repositories.tsv` declares each role, the `cmd/` path for a pair, and the frozen Go-code
digest for a legacy snapshot. `scripts/check_sibling_drift.py` enforces the declaration:

| Role | Check | Failure means |
|---|---|---|
| `legacy` | Go digest equals the recorded digest; README carries the exact `Superseded. The maintained implementation` pointer to `cmd/<tool>` | someone edited the frozen copy, or the pointer was removed/changed |
| `independent` | no `cmd/<name>` directory exists | a counterpart appeared; the pair must be reclassified |
| `mirror` | every Go file is byte-identical to its source | real mirror drift (the check has no rows today; `--self-test` proves the comparator) |

Run it against a directory of sibling checkouts:

```sh
python3 scripts/check_sibling_drift.py --root <sibling-root> --require-all
python3 scripts/check_sibling_drift.py --self-test
```

`.github/workflows/sibling-drift.yml` runs the same check weekly and on demand, cloning only the four
legacy repositories. The check is read-only and needs no token beyond the default checkout access.

## 6. Reproduce the measurement

```sh
# Inventory and blob hashes of a sibling
git -C QuarkLangQkcheck ls-tree -r HEAD

# The candidate source in this repository
git ls-tree -r HEAD -- cmd/qkcheck

# One file's content identity
git -C QuarkLangQkcheck rev-parse HEAD:main.go
git rev-parse HEAD:cmd/qkcheck/main.go
```

The line count and similarity values come from a `difflib.SequenceMatcher` comparison of the two
`main.go` files; the Go digest is recomputed by the checker from the same rule documented in §1.
