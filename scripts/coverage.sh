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

# How much of a failing test's own output to print: enough for the failing test's log lines and the
# package's FAIL summary, without burying the rest of the CI log.
failure_tail=120

report_test_failure() { # usage: report_test_failure <label> <log>
  local label="$1" log="$2" failed
  # `go test ./...` prints one package's output per package, so the failing test is not guaranteed to sit
  # in the tail: name every failing test first, then show the tail for the detail around them.
  failed="$(grep -E '^--- FAIL: ' "$log" | sort -u || true)"
  if [ -n "$failed" ]; then
    echo "✘ $label: go test failed — failing test(s):" >&2
    printf '%s\n' "$failed" >&2
    echo "--- last $failure_tail lines of the test output ---" >&2
  else
    echo "✘ $label: go test failed — last $failure_tail lines of the test output:" >&2
  fi
  tail -n "$failure_tail" "$log" >&2
}

run_tests() { # usage: run_tests <module dir> <profile> <label>
  local dir="$1" profile="$2" label="$3" status=0
  local log="$profile.log" # the profile path is already unique per module, so it names the log too
  # The output is kept in a file rather than discarded: when the gate fails, this log is the only thing
  # that names the test that failed it. The step used to send it to /dev/null, so a red "Coverage report"
  # step (linux extras, run 38053298015) could not be told apart from a regression without a rerun and a
  # local reproduction. Nothing else about the gate changes: the same packages run with the same flags,
  # and the failure still stops the script with go test's own exit status.
  (cd "$dir" && go test ./... -coverprofile="$profile" -covermode=atomic -count=1) >"$log" 2>&1 || status=$?
  if [ "$status" -ne 0 ]; then
    report_test_failure "$label" "$log"
    return "$status"
  fi
}

echo '→ root module: go test ./... -covermode=atomic' >&2
run_tests "." "$tmp/root.out" "root module"
echo '→ compiler module: go test ./... -covermode=atomic' >&2
run_tests "compiler" "$tmp/compiler.out" "compiler module"

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
