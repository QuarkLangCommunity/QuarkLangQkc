# Module consumability: making `compiler` installable from outside the checkout

**Status:** proposal — no `go.mod` and no workflow has been changed. This document records what was
measured, what the options cost, and which one is recommended.

The compiler is the only part of the toolchain a user is expected to install as a Go module. Today it can
only be built from a clone. This note reproduces the failures, explains why the committed `replace` is
there, and proposes the smallest change that makes the module consumable.

## 1. Reproduced failures

Repo state: `origin/main` = `67ce88c`, Go 1.26.8. Published tags: `v1.0.0`, `v2.0.0`, `v2.1.0`. There is
no `v0.0.0`, and no tag carries the `/v2` module-path suffix the two later tags would need.

### 1.1 `go install`

```
$ go install github.com/QuarkLangCommunity/QuarkLangQkc/compiler@latest
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc v1.0.0
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc/compiler v0.0.0-20261010124804-67ce88cd1e52
go: github.com/QuarkLangCommunity/QuarkLangQkc/compiler@latest (in github.com/QuarkLangCommunity/QuarkLangQkc/compiler@v0.0.0-20261010124804-67ce88cd1e52):
	The go.mod file for the module providing named packages contains one or
	more replace directives. It must not contain directives that would cause
	it to be interpreted differently than if it were the main module.
```

### 1.2 `go get`

```
$ go get github.com/QuarkLangCommunity/QuarkLangQkc/compiler@latest
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc v1.0.0
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc/compiler v0.0.0-20261010124804-67ce88cd1e52
go: downloading github.com/QuarkLangCommunity/QuarkLangQkc v0.0.0
go: github.com/QuarkLangCommunity/QuarkLangQkc/compiler imports
	github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18n: reading github.com/QuarkLangCommunity/QuarkLangQkc/go.mod at revision v0.0.0: unknown revision v0.0.0
go: github.com/QuarkLangCommunity/QuarkLangQkc/compiler imports
	github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang: reading github.com/QuarkLangCommunity/QuarkLangQkc/go.mod at revision v0.0.0: unknown revision v0.0.0
```

A third, blunter symptom of the same layout, from the module that *has* been tagged:

```
$ go install github.com/QuarkLangCommunity/QuarkLangQkc/compiler@v1.0.0
go: github.com/QuarkLangCommunity/QuarkLangQkc/compiler@v1.0.0: module github.com/QuarkLangCommunity/QuarkLangQkc@v1.0.0 found, but does not contain package github.com/QuarkLangCommunity/QuarkLangQkc/compiler
```

Both of the first two failures come from the same file, `compiler/go.mod`:

```
module github.com/QuarkLangCommunity/QuarkLangQkc/compiler

go 1.26

require github.com/QuarkLangCommunity/QuarkLangQkc v0.0.0

replace github.com/QuarkLangCommunity/QuarkLangQkc => ../
```

and they have two different causes.

* `go install` fails on the **`replace` directive itself**. A module that is not the main module may not
  carry a `replace`, because it would change how its imports resolve for everyone.
* `go get` fails on the **`require … v0.0.0` that only the replace made meaningful**. A relative replace
  is invisible outside the checkout, so the requirement is left pointing at `v0.0.0`, a revision that was
  never tagged and therefore does not exist on the proxy.

## 2. Why the replace exists, and what it buys

`compiler/go.mod` is a second module inside the repository, so while the compiler is the main module the
root module's own configuration does not apply to it. The replace is what lets the compiler build in a
fresh clone without the root module first having to exist on the proxy — a real benefit: `go build ./...
&& go test ./... -count=1` in `compiler/` passes today with no dependency on a published root module.

The cost is that the replace is *the reason* the compiler cannot be consumed from outside. Measured on
`origin/main`:

| Observation | Command / evidence |
|---|---|
| The root module is tagged, but the published archive strips the compiler entirely | `go mod download …@v1.0.0` succeeds; `unzip -l` reports **30 files**, and the archive contains **0** `compiler/*.go` — the proxy strips nested modules from the zip |
| So the compiler has never been reachable through the proxy | `go install …/compiler@v1.0.0` → `module …@v1.0.0 found, but does not contain package …/compiler` |
| The compiler's own imports are `internal/`, which is not importable from outside | measured: an outside module importing `…/compiler/internal/cgen` is refused with `use of internal package … not allowed` |
| Consumers cannot even require the published version | `go get …@v1.0.0` → `parsing go.mod: module declares its path as: quarklang but was required as: github.com/QuarkLangCommunity/QuarkLangQkc` |
| The module path was corrected only after the last tag | `git log -S'module github.com/QuarkLangCommunity/QuarkLangQkc' -- go.mod` → `6dad203 build: publicly resolvable module paths for the public org`; `git show v1.0.0:go.mod` and `git show v2.1.0:go.mod` both say `module quarklang` |

The compiler imports `internal/lang` and `internal/i18n` from the root module, and `compiler/main.go`
imports `compiler/internal/cgen`. Go's `internal` rule allows such an import only from inside the tree
rooted at the parent of `internal/` — measured, not assumed. That single rule is what rules out option C.

**Consequence for any fix:** because every published tag still declares `module quarklang`, *no existing
version is consumable at all*, and because `v2.x` cannot be a Go module version for this path (see
§3, option A), the fix has to end in a new **`v1.x`** tag with the corrected module path.

## 3. Options

### Option A — fold the compiler into the root module, then tag `v1.x`

Delete `compiler/go.mod` (and `bench/go.mod`, the other nested module) so the repository is one module
again, move the compiler's extra `require` of the parser module into the root `go.mod`, drop the now
pointless replace, and cut a fresh `v1.x` tag.

* **Cost:** small, and concentrated in `go.mod` bookkeeping. `compiler/go.sum` disappears; the duplicated
  parser pin collapses into one.
* **Constraint on the tag:** it must be `v1.x`. A `v2.x` tag is rejected outright, because the module path
  has no `/v2` suffix: `go get …@v2.2.0` → `invalid version: should be v0 or v1, not v2`. Adding `/v2` to
  the module path instead would rewrite every import path in the repository and is out of scope here.
* **CI implications:** removes three duplicated build/test steps and the second `go.mod` that has to be
  kept in step with the root pin. `compiler/testdata/compare.sh` is unaffected — it invokes `go build` by
  path, not across a module boundary.
* **What breaks for existing users:** nothing that works today. `cd compiler && go build` keeps working
  because the directory is still there, and `go install …/compiler@…` starts working for the first time.
* **Everything stays reachable:** the compiler is inside the root module, so `internal/lang`,
  `internal/i18n` and `compiler/internal/cgen` remain legal imports — no re-exporting needed.

### Option B — keep the replace, document that the compiler is checkout-only

Leave both `go.mod` files as they are and state in the README that `qkc` is built from a clone only.

* **Cost:** zero code change; but the module path stays published and broken, so the failing commands keep
  failing for anyone who guesses them.
* **CI implications:** none.
* **What breaks for existing users:** the status quo. The real cost is answering "no" to a reasonable
  expectation, in prose rather than in the module metadata.

### Option C — publish the compiler as its own module path

Give the compiler a path of its own, e.g. `…/QuarkLangQkc/compiler/v2`, depending on a tagged root module.

* **Cost:** high, and it does not currently work. The compiler's own imports are `internal/`
  (`internal/lang`, `internal/i18n`, `compiler/internal/cgen`), and the `internal` rule refuses every one
  of them from outside the root module's tree. The option becomes possible only after those packages are
  re-exported from public paths or moved out of `internal/` — a change to the API surface of the language
  implementation, not a packaging change. It would also need `/v2` in the path and matching tags.
* **CI implications:** a second published module to tag, version and pin, plus a release step that keeps
  the two paths in step.
* **What breaks for existing users:** the module path changes, so any `go install` line written against
  the old path stops resolving.

## 4. Recommendation

**Option A**, with the tag constraint above. It is the smallest change that makes the module consumable,
and it is the only option whose goal is reachable today. Concretely:

1. delete `compiler/go.mod` and `bench/go.mod`, folding both directories into the root module;
2. move the compiler's `require` of the parser module up into the root `go.mod`, and delete
   `compiler/go.sum`;
3. delete the `replace` line — with one module there is nothing left to replace;
4. cut a new **`v1.x`** tag (not a rewrite of `v1.0.0`, and not `v2.x`) and let `release.yml`, which fires
   on `v*`, publish it. Keep the project's own `VERSION`/`v2.x` release numbering out of the Go module's
   version line, or the two will keep colliding.

Measured end to end against a local proxy that serves a proxy-style archive of this repository with
exactly those edits applied, and the root module published as `v1.2.0`:

```
$ go get github.com/QuarkLangCommunity/QuarkLangQkc@v1.2.0
go: added github.com/QuarkLangCommunity/QuarkLangQkc v1.2.0
$ go install github.com/QuarkLangCommunity/QuarkLangQkc/compiler@latest
$ echo $?
0
$ ls -l "$(go env GOPATH)/bin"
-rwxr-xr-x 1 jack jack 8739928 compiler
$ compiler --help | head -1
usage: qkc [options] <file.qk>
$ compiler -run rec.kq
error: StackOverflowError: recursion depth exceeded 8192 at line 3
```

The last line matters for the check itself: the freshly installed binary is exercised on a real program,
not merely started, and it reports the shared call-depth limit that `spec/STANDARD.md` §11.2 documents.

Two smaller notes for whoever executes this:

* Once the module is published at a tag, `@latest` resolves as well — the nested package needs no
  `compiler/vX.Y.Z` tags of its own, because a nested package's version is its module's version. Measured
  with the local proxy, not assumed.
* This document changes no `go.mod` and no workflow. Steps 1–4 are the proposal.
