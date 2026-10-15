#!/usr/bin/env bash
# The generated review document's own proof: where it is written, and that it cannot be written back
# into the tree it describes.
#
# The document is not committed. It used to be, and the reason was the gate: a staleness check compared
# the committed copy against a fresh rendering, so the file had to exist for the gate to have something
# to compare. The cost was a public repository carrying the Chinese values as data, one row per
# untranslated message. Taking the file out of the tree therefore means proving the three things that
# kept it there are still true of the replacement:
#
#   1. the document is still rendered from the tables, and it is written outside the checkout. A
#      regression here is a document in the tree, which is the defect this harness exists to catch.
#   2. the gate can still go red. A gate that cannot fail proves nothing, so the staleness check is
#      driven against a file that is deliberately not what the tables say, and its message is read.
#   3. `go generate ./internal/i18n` still produces exactly the two artefacts it commits. The document
#      left the directive rather than the tree, so the catalog and the C header must be untouched — and
#      a `-check` run has to agree.
#
# Every claim is measured by running the command and reading its output, not by reading the code. The
# exit status of the command is part of each claim: `-doc` seeded with a path inside the repository has
# to fail, and failing for the right reason is checked separately from failing.
#
# Usage: internal/i18n/proof/generated_document.sh
#
# A failure here means the review document is being written where it must not be, or the gate that
# replaced its staleness check no longer detects a document that does not match the tables.

set -u
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
proof_init QK_DOC_PROOF_WORK "$HOME/.cache/qk-i18n/doc-proof"

bin="$work/qki18n"
rm -rf "$work"
mkdir -p "$work"

# The refusal proofs deliberately seed a path inside the checkout, and a run with the guards removed
# does write it: clean up on every exit, including a failing one, so a red harness cannot leave a
# document holding the Chinese values lying in docs/ for somebody to add.
inside="$root/docs/i18n-backlog.md"
trap 'rm -f "$inside"' EXIT



# The generator itself, built once: the proofs below are about its behaviour, and `go run` would fold
# the build output into every reading.
if ! ( cd "$root" && go build -o "$bin" ./cmd/qki18n ); then
  echo "could not build cmd/qki18n" >&2
  exit 1
fi

# ---- 1. the default location is outside the checkout, and the tool says where it wrote ----
out="$("$bin" -source "$root/internal/i18n/source" -out "$work/msg_gen.go" -root "$root" -doc cache 2>&1)"
rc=$?
written="$(printf '%s\n' "$out" | sed -n 's/^qki18n: wrote //p' | tail -1)"
if [ "$rc" -ne 0 ]; then
  proof_report bad "default location" "the generator failed: $out"
elif [ -z "$written" ]; then
  proof_report bad "default location" "the tool did not print the path it wrote: $out"
elif [ ! -f "$written" ]; then
  proof_report bad "default location" "the tool printed $written, which does not exist"
elif [ ! -s "$written" ]; then
  proof_report bad "default location" "$written is empty"
elif case "$written" in "$root"/*) true ;; *) false ;; esac; then
  proof_report bad "default location" "the document was written inside the checkout: $written"
else
  # And it is a document, not an empty file with the right name.
  if grep -q '^# i18n translation backlog' "$written"; then
    proof_report ok "default location" "written outside the checkout, $(wc -c < "$written") bytes"
  else
    proof_report bad "default location" "$written does not carry the document's title"
  fi
fi
outside="$written"

# ---- 2. the default location cannot be pulled back into the tree ----
#
# Both paths that reach the refusal are driven, because they are two different guards: writing
# (`-doc <in-tree path>`) is refused by cmd/qki18n before the generator is called, and comparing
# (`-check -doc <in-tree path>`) is refused inside runDoc, before it reads the file it was handed. A
# proof that only drove the first would pass with the second deleted.
rm -f "$inside"

# refuse_in_tree <label> <extra args...>
refuse_in_tree() {
  local label="$1"; shift
  local out rc
  out="$("$bin" -source "$root/internal/i18n/source" -out "$work/msg_gen.go" -root "$root" "$@" 2>&1)"
  rc=$?
  total=$((total + 1))
  if [ "$rc" -eq 0 ]; then
    fail=$((fail + 1))
    printf 'FAIL %-22s the generator accepted a path inside the checkout: %s\n' "$label" "$inside" >&2
  elif ! printf '%s' "$out" | grep -q 'inside the repository'; then
    fail=$((fail + 1))
    printf 'FAIL %-22s refused, but not for the reason this proof is about: %s\n' "$label" "$out" >&2
  elif [ -e "$inside" ]; then
    fail=$((fail + 1))
    printf 'FAIL %-22s refused and still created it: %s\n' "$label" "$inside" >&2
  else
    printf 'ok   %-22s refused a path inside the checkout, and wrote nothing\n' "$label"
  fi
}

refuse_in_tree "in-tree refusal"      -doc "$inside"
refuse_in_tree "in-tree check"        -check -doc "$inside"

# ---- 3. the staleness check can still go red ----
#
# A copy that is not what the tables say: the rendered document with its title replaced. The check has
# to fail, and it has to say `out of date` rather than fail for an unrelated reason such as a missing
# file, or the proof would pass on a generator that simply cannot read its own input.
stale="$work/stale.md"
if [ -n "$outside" ] && [ -f "$outside" ]; then
  sed 's/^# i18n translation backlog$/# not the document the tables produce/' "$outside" > "$stale"
  out="$("$bin" -source "$root/internal/i18n/source" -root "$root" -check -out "$root/internal/i18n/msg_gen.go" -doc "$stale" 2>&1)"
  rc=$?
  total=$((total + 1))
  if [ "$rc" -eq 0 ]; then
    fail=$((fail + 1))
    printf 'FAIL %-22s the staleness check accepted a document that is not what the tables say\n' "stale check" >&2
  elif ! printf '%s' "$out" | grep -q 'out of date'; then
    fail=$((fail + 1))
    printf 'FAIL %-22s the check failed for another reason: %s\n' "stale check" "$out" >&2
  else
    printf 'ok   %-22s a document that is not what the tables say is rejected\n' "stale check"
  fi
else
  proof_report bad "stale check" "no document to perturb, so this proof did not run"
fi

# ---- 4. the two committed artefacts are untouched by all of this ----
#
# `go generate` is the command a contributor runs, and the document must not be part of what it writes:
# the tree has to be clean afterwards, and `-check` has to agree with what is committed.
before="$(cd "$root" && git status --porcelain -- internal/i18n/msg_gen.go compiler/internal/rtcat/msg_c.h)"
( cd "$root" && go generate ./internal/i18n ) > "$work/generate.log" 2>&1
rc=$?
after="$(cd "$root" && git status --porcelain -- internal/i18n/msg_gen.go compiler/internal/rtcat/msg_c.h)"
total=$((total + 1))
if [ "$rc" -ne 0 ]; then
  fail=$((fail + 1))
  printf 'FAIL %-22s go generate failed: %s\n' "committed artefacts" "$(cat "$work/generate.log")" >&2
elif [ "$before" != "$after" ]; then
  fail=$((fail + 1))
  printf 'FAIL %-22s go generate changed a committed artefact: %s -> %s\n' "committed artefacts" "$before" "$after" >&2
elif printf '%s' "$after" | grep -q 'docs/i18n-backlog'; then
  fail=$((fail + 1))
  printf 'FAIL %-22s go generate wrote the document back into the tree\n' "committed artefacts" >&2
else
  printf 'ok   %-22s go generate rewrote the catalog and the C header unchanged\n' "committed artefacts"
fi

out="$("$bin" -source "$root/internal/i18n/source" -out "$root/internal/i18n/msg_gen.go" -c-out "$root/compiler/internal/rtcat/msg_c.h" -root "$root" -check 2>&1)"
rc=$?
total=$((total + 1))
if [ "$rc" -ne 0 ]; then
  fail=$((fail + 1))
  printf 'FAIL %-22s -check disagrees with what is committed: %s\n' "generate then check" "$out" >&2
elif printf '%s' "$out" | grep -q 'docs/i18n-backlog'; then
  fail=$((fail + 1))
  printf 'FAIL %-22s -check still compares the document against a committed copy: %s\n' "generate then check" "$out" >&2
else
  printf 'ok   %-22s -check is clean and names only the committed artefacts\n' "generate then check"
fi

proof_finish "generated-document checks" \
  "generated document clean (written outside the checkout, refused inside it; the staleness check still goes red; the committed artefacts are unchanged)"
