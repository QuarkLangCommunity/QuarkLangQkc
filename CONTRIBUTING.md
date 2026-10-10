# Contributing to QuarkLang

**English** · [简体中文](CONTRIBUTING.zh-CN.md)

Thanks for spending time on QuarkLang. This document only covers **how to get your change merged fastest**;
everything else lives in the code and the docs.

## The three lowest-friction paths

| Goal | Start here |
|---|---|
| Report a bug / request a feature | [Open an issue](https://github.com/QuarkLangCommunity/QuarkLangQkc/issues/new/choose) (use the template, attach a minimal reproduction) |
| First contribution | Pick a [`good first issue`](https://github.com/QuarkLangCommunity/QuarkLangQkc/labels/good%20first%20issue) (docs, examples, error messages, tests — no compiler internals needed) |
| Change code | Follow the flow below; **`main` is protected: PR required, CI must be green** |

## Pull request flow

```sh
git clone https://github.com/QuarkLangCommunity/QuarkLangQkc && cd QuarkLangQkc
git switch -c fix/short-description    # branch names: fix/… feat/… docs/… ci/…
# …make your change…
go test ./... && (cd compiler && go test ./...)      # ① both test layers
(cd compiler && ./testdata/compare.sh)               # ② dual-path parity (must print "all identical")
git commit -m "Short title (English or Chinese, one line)"   # ③ commit message: imperative, add a body when useful
git push -u origin HEAD && gh pr create --fill       # ④ open the PR; CI runs the three-platform matrix
```

Merge requirements (automated — no human approval needed today):

| Gate | What it means |
|---|---|
| 7 required checks | `tests (os)` on all three platforms, `linux extras (dual-path / editors / release)`, `libraries & preprocessor (os)` on all three |
| Branch protection | `main` rejects direct pushes, force-pushes and deletions |

## Pre-push checklist (runs locally)

```sh
go test ./...                                    # interpreter + toolchain (includes the lint FP/FN gates)
(cd compiler && go test ./...)                   # compiler
(cd compiler && ./testdata/compare.sh)           # dual-path parity (both backends must agree)
go build -o qkcheck ./cmd/qkcheck && ./qkcheck examples/ compiler/testdata   # dogfood the linter
```

### The syntax layer is a separate repository

`go.mod` pins [`QuarkLangQkparser`](https://github.com/QuarkLangCommunity/QuarkLangQkparser) to one exact
commit, so a fresh clone needs **nothing extra**: `go test ./...` downloads it like any other dependency,
and so does anyone importing this module.

To change the parser and this repository together, keep the sibling checkout the `.gitignore` already
expects (`QuarkLangQkparser/`) and wire it in with a per-machine Go workspace; the pin stays untouched, so
CI keeps building the published revision:

```sh
git clone https://github.com/QuarkLangCommunity/QuarkLangQkparser   # → ./QuarkLangQkparser
go work init . ./compiler ./QuarkLangQkparser                       # go.work is per-machine, not tracked
```

When a parser change lands, bump the pin in both modules: `go get github.com/QuarkLangCommunity/QuarkLangQkparser@<commit>`.

When you add a language feature, **all four of these must be updated together**:

1. `SYNTAX.md` — the single canonical syntax reference (English; Chinese: `SYNTAX.zh-CN.md`; numbered M/K/E/S/O/T/P entries);
2. **both backends** (interpreter and compiler — they share the front end, but semantics must match);
3. `compiler/testdata/compare.sh` — the parity cases;
4. for static-analysis changes, the labelled cases under `internal/lang/testdata/lintbench/` (the FP/FN gate runs them).

## Style

- **Canonical syntax only**: type first (`<modifier> <type> <name>`); `impl { … } Name;`; `space { … } name;`;
  there is no `let`/`var`-style declaration.
- Go code: `gofmt`-clean; comments explain **why**, not what; zero third-party dependencies
  (a new dependency needs a justification in the PR).
- Performance changes must ship **before/after measurements** (`scripts/bench-tools.sh` or `go test -bench`),
  and state whether you profiled first.
- Error messages are user-facing: state what is wrong **and** how to write it correctly, preferably with a
  canonical-syntax example.
- **Bilingual messages**: wrap new user-facing Chinese text with `i18n.T(...)` and register the English in
  `internal/i18n/table.go`; `go test ./internal/i18n/` fails until you do.

## Code of conduct

By participating you agree to the [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Contact

- Mention [@Enoch-199811](https://github.com/Enoch-199811) (maintainer) in an issue or PR.
- For security issues, please **do not** open a public issue — use GitHub's private
  [Report a vulnerability](https://github.com/QuarkLangCommunity/QuarkLangQkc/security/advisories/new) flow.
