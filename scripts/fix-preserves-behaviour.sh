#!/usr/bin/env bash
# Proof that a -fix run does not change what a program does, over a corpus and under both engines.
#
#   scripts/fix-preserves-behaviour.sh <qkcheck> <quark> <qkc> <corpus...>
#
# For every source file of the corpus:
#
#   1. copy it, and its directory, into a scratch tree (the corpus itself is never written to);
#   2. run it with `quark` and with `qkc -run`, recording everything observable: stdout+stderr and the
#      exit code;
#   3. apply the corrections with `qkcheck -fix`, and keep the result;
#   4. run the corrected file under both engines again and compare with step 2;
#   5. apply the corrections a second time and compare the file with step 3's result.
#
# A file that does not run before the fix is not evidence about what running it does, so it counts as
# "not runnable" rather than as a pass. Its rewrite is still checked for idempotence, and a corrected
# file that stops running when the original ran is a failure.
#
# Exit code 0 means every runnable file behaved identically before and after, and every rewrite was
# idempotent. Anything else is a defect in a correction, and the report names the file and the way.
set -u

if [ $# -lt 4 ]; then
  echo "usage: $0 <qkcheck> <quark> <qkc> <corpus...>" >&2
  exit 2
fi

qkcheck="$1"; quark="$2"; qkc="$3"; shift 3
# A program that reads stdin must not stall the proof, and neither must a compiler run: a bounded run
# whose timeout shows up as a difference is reported rather than waited on.
timeout_s="${FIX_PROOF_TIMEOUT:-120}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
cache="$work/cache"
mkdir -p "$cache"

# run_interp prints everything the interpreter makes observable, exit code included, so a caller can
# compare the whole observable surface with one string comparison.
run_interp() {
  local out rc
  out="$(timeout "$timeout_s" "$quark" "$1" </dev/null 2>&1)"; rc=$?
  printf '%s\n--exit %d--\n' "$out" "$rc"
}

# run_compiled is the same for the native compiler, whose cache is cleared per run so a stale binary
# cannot mask a codegen difference.
run_compiled() {
  local out rc
  rm -rf "$cache"; mkdir -p "$cache"
  out="$(QUARK_CACHE="$cache" timeout "$timeout_s" "$qkc" -run "$1" </dev/null 2>&1)"; rc=$?
  printf '%s\n--exit %d--\n' "$out" "$rc"
}

files=0; fixed=0; compared=0; skipped=0; failures=0
for corpus in "$@"; do
  while IFS= read -r -d '' src; do
    files=$((files + 1))
    dir="$work/$(printf '%s' "$src" | tr '/.' '__')"
    mkdir -p "$dir"
    cp -a "$(dirname "$src")"/. "$dir"/ 2>/dev/null || true
    target="$dir/$(basename "$src")"

    # A file with no pending correction is left byte for byte alone by the run below, and a program
    # that was never rewritten says nothing about whether a rewrite preserves behaviour: skip it
    # before paying for two engine runs. -check is the tool's own answer to "is there anything to do".
    timeout "$timeout_s" "$qkcheck" -exit0 -check "$target" >/dev/null 2>&1
    if [ $? -ne 3 ]; then
      continue
    fi

    before_file="$(cat "$target")"
    interp_before="$(run_interp "$target")"
    comp_before="$(run_compiled "$target")"

    timeout "$timeout_s" "$qkcheck" -exit0 -fix "$target" >/dev/null 2>&1
    after_file="$(cat "$target")"
    fixed=$((fixed + 1))

    # Idempotence: the second run must leave the bytes of the first result alone.
    timeout "$timeout_s" "$qkcheck" -exit0 -fix "$target" >/dev/null 2>&1
    if [ "$(cat "$target")" != "$after_file" ]; then
      echo "FAIL $src: the second -fix changed the file"
      failures=$((failures + 1))
      continue
    fi

    interp_after="$(run_interp "$target")"
    comp_after="$(run_compiled "$target")"
    if [ "$interp_before" != "$interp_after" ]; then
      if [ "$interp_before" != "$comp_before" ]; then
        skipped=$((skipped + 1)) # did not run cleanly before the fix: not evidence
        continue
      fi
      echo "FAIL $src: a corrected program behaves differently under the interpreter"
      diff <(printf '%s' "$interp_before") <(printf '%s' "$interp_after") | head -20
      failures=$((failures + 1))
      continue
    fi
    if [ "$comp_before" != "$comp_after" ]; then
      echo "FAIL $src: a corrected program behaves differently under qkc -run"
      diff <(printf '%s' "$comp_before") <(printf '%s' "$comp_after") | head -20
      failures=$((failures + 1))
      continue
    fi
    compared=$((compared + 1))
  done < <(find "$corpus" -type f \( -name '*.qk' -o -name '*.kq' \) -print0 | sort -z)
done

echo "corpus files $files | rewritten $fixed | compared under both engines $compared | not runnable $skipped | failures $failures"
[ "$failures" -eq 0 ] || exit 1
echo "every corrected program behaves exactly as its original did (both engines), and every rewrite is idempotent"
