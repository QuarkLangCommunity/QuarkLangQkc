#!/usr/bin/env bash
# The three i18n proof harnesses' common shape: where the checkout is, the baseline revision they are
# measured against, the build they do, and the pass/fail ledger they report through. Each harness keeps
# its own claims and its own corpus; this file is only what all three do the same way, so a change to the
# refusal or the ledger is one edit rather than three.

# proof_init <env-var-name> <default-work-dir>: locate the repository, choose the scratch directory, and
# require the interpreter the checkers are written in.
proof_init() {
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  root="$(cd "$here/../../.." && pwd)"
  work="${!1:-$2}"
  mkdir -p "$work/bin"
  command -v python3 >/dev/null 2>&1 || { echo "cannot run: python3 is required by the checkers" >&2; exit 2; }
}

# proof_baseline <default-revision> <why-this-refusal-matters>: resolve the revision the claims are
# measured against, and refuse to compare this checkout with itself. The exit code is 2, not 1: nothing
# was measured, so this is a gate that cannot run rather than a gate that went red.
proof_baseline() {
  baseline_default="$1"
  baseline_rev="${2:-}"
  local why="$3"
  [ -n "$baseline_rev" ] || baseline_rev="$baseline_default"
  head_commit="$(git -C "$root" rev-parse 'HEAD^{commit}')"
  if ! base_commit="$(git -C "$root" rev-parse --verify --quiet "$baseline_rev^{commit}")"; then
    echo "cannot run: the baseline revision '$baseline_rev' does not resolve to a commit in $root" >&2
    exit 2
  fi
  if [ "$base_commit" = "$head_commit" ] ||
     [ "$(git -C "$root" rev-parse 'HEAD^{tree}')" = "$(git -C "$root" rev-parse "$base_commit^{tree}")" ]; then
    echo "refusing to run: the baseline revision $baseline_rev (commit $base_commit) carries this checkout's" >&2
    echo "content (commit $head_commit), so $why. A revision compared with" >&2
    echo "itself measures nothing. Pass the pre-change revision instead, e.g. $baseline_default." >&2
    exit 2
  fi
}

# proof_worktree <path>: a detached worktree at the baseline revision, so the comparison is against the
# code as it was rather than against a description of it.
proof_worktree() {
  if [ ! -d "$1" ]; then
    git -C "$root" worktree add -q --detach "$1" "$baseline_rev" || exit 1
  fi
  git -C "$1" checkout -q --detach "$baseline_rev" || exit 1
}

# proof_build <engine: quark|qkc> <source-dir> <output>: build one engine from a checkout, and say which
# one failed to build.
proof_build() {
  local engine="$1" dir="$2" out="$3" where="$2"
  [ "$engine" = qkc ] && where="$dir/compiler"
  ( cd "$where" && go build -o "$out" . ) || { echo "$engine build failed: $dir" >&2; exit 1; }
}

# ---- the ledger ----
fail=0
total=0

# proof_report <ok|bad> <name> <detail>
proof_report() {
  total=$((total + 1))
  if [ "$1" = ok ]; then
    printf 'ok   %-22s %s\n' "$2" "$3"
  else
    fail=$((fail + 1))
    printf 'FAIL %-22s %s\n' "$2" "$3" >&2
  fi
}

# proof_finish <noun> <clean-message>: exit non-zero when any claim failed, and say what was proved.
proof_finish() {
  if [ "$fail" -ne 0 ]; then
    echo "FAILED: $fail of $total $1" >&2
    exit 1
  fi
  echo "$2  [$total checks]"
}
