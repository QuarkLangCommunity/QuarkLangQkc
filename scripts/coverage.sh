#!/usr/bin/env bash
# Coverage report for both modules (root: interpreter + toolchain; compiler: qkc + LLVM IR emitter), as markdown.
#
#   scripts/coverage.sh [-o report.md]       # markdown to stdout, or to a file with -o
#
# Reproducible by design: the report carries the commit, the date and the Go version, so a number
# pasted into an issue can be traced back to the tree that produced it.
set -euo pipefail
cd "$(dirname "$0")/.."

out=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) out="${2:?-o needs a path}"; shift 2 ;;
    -h|--help) sed -n '2,8p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo '→ root module: go test ./... -covermode=atomic' >&2
go test ./... -coverprofile="$tmp/root.out" -covermode=atomic -count=1 >/dev/null
echo '→ compiler module: go test ./... -covermode=atomic' >&2
(cd compiler && go test ./... -coverprofile="$tmp/compiler.out" -covermode=atomic -count=1 >/dev/null)

module_table() { # usage: module_table <title> <profile> <module dir>
  # the cover tool resolves package paths through the module it runs in, so each profile is read
  # from its own module directory (the profile itself is an absolute path)
  echo "### $1"
  echo
  echo '| Package | Coverage |'
  echo '|---|---|'
  (cd "$3" && go tool cover -func="$2") | awk '{split($1,a,":"); pkg=a[1]; cov[pkg]=$NF} END {for (p in cov) printf "| %s | %s |\n", p, cov[p]}' | sort
  echo
  echo "**Total: $( (cd "$3" && go tool cover -func="$2") | tail -1 | awk '{print $NF}')**"
  echo
}

{
  echo '# Test coverage report'
  echo
  echo "- Commit: $(git rev-parse --short HEAD)"
  echo "- Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "- Go: $(go version | awk '{print $3}')"
  echo
  module_table 'root module (interpreter + toolchain)' "$tmp/root.out" "."
  module_table 'compiler module (qkc + LLVM IR emitter)' "$tmp/compiler.out" "compiler"
} > "$tmp/report.md"

if [ -n "$out" ]; then
  cp "$tmp/report.md" "$out"
  echo "coverage report written to $out" >&2
else
  cat "$tmp/report.md"
fi
