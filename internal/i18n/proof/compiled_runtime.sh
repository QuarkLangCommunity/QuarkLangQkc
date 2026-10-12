#!/usr/bin/env bash
# The compiled path's own i18n proof: the text a *compiled* program prints.
#
# byte_identical.sh proves the interpreter, and that is exactly why this gap survived the migration:
# these strings live in C, inside the runtime qkc embeds (compiler/qthreads.c.txt) and compiles into the
# binary, with no Go code in reach at run time. The README carried `越界` as the evidence.
#
# Three claims, each measured against a revision or against the other engine rather than asserted:
#
#   1. the baseline leaked. Under QK_LANG=en the pre-change compiled binary must print Han runes, or
#      this harness would pass vacuously on a compiler that prints nothing.
#   2. this tree does not. Under QK_LANG=en no program in the corpus may print a Han rune, and the
#      rendering must equal the interpreter's for the same program — one catalog, two engines, so they
#      are not allowed to disagree — with the error kind and the line number unmoved.
#   3. the Chinese did not move. Under QK_LANG=zh every case must be byte-identical to the baseline:
#      migrating a message moves where the text lives, never which text is chosen, and a Chinese
#      reader must not be able to tell that anything happened on this path either.
#
# Claim 3 has one class of exception, and it is a consequence of claim 2 rather than an exemption: a
# parse error whose VALUE contains a control character. The baseline's C built `ParseError: "%s" …`
# around the raw value, so a value holding a tab printed the tab; the catalog row spells the same
# sentence with Go's %q, which escapes it (`\t`, `\\`, …), and the interpreter has always printed the
# escaped form. Preserving the baseline byte-for-byte there would make the two engines render the same
# program differently — the defect this harness exists to prevent — so the escaped form wins, and
# zh_escapes.tsv below names each such case with both renderings instead of leaving it to be
# rediscovered. Nothing outside that table may differ.
#
# The corpus is one program per message the runtime can reach, plus a program that does not fail at all
# (the negative control: a harness reporting "no Han rune" for a compiler that printed nothing would
# otherwise be invisible). source/c_runtime.txt is the authority on how many messages that is, and the
# last section fails when a listed message no longer reaches a case.
#
# Usage: internal/i18n/proof/compiled_runtime.sh [baseline-revision]
#   baseline-revision defaults to the pre-change revision pinned below. The harness refuses to run when it
#   resolves to this checkout: a revision compared with itself measures nothing. Needs clang, because qkc
#   links the runtime with it.
#
# A failure here means a compiled program's text disagrees with the interpreter's, which is the defect
# this harness exists to catch.

set -u

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../../.." && pwd)"

# The revision the three claims are measured against: main as it stood when the runtime's text moved into
# the catalog — the first parent of the merge that landed PR #101, which is the revision `origin/main`
# named when this harness was written. It is pinned rather than named by a branch because `origin/main`
# named that revision only until the change merged, and named the change itself afterwards: the default
# then compared a revision with itself, which is how this gate came to report six failures whose text
# blamed the corpus. A baseline that moves is a gate that turns green into vacuous and red into
# unexplained.
baseline_default="a3c4cd4"
baseline_rev="${1:-$baseline_default}"

# Refuse that comparison here, before spending a build on it. Claim 1 asks whether the baseline leaked Han
# runes, and a tree cannot leak differently from itself, so no case below could be measured and every
# message case would fail for a reason that names the corpus rather than the invocation. The exit code is
# 2, not 1: nothing was measured, so this is a gate that cannot run rather than a gate that went red.
head_commit="$(git -C "$root" rev-parse 'HEAD^{commit}')"
if ! base_commit="$(git -C "$root" rev-parse --verify --quiet "$baseline_rev^{commit}")"; then
  echo "cannot run: the baseline revision '$baseline_rev' does not resolve to a commit in $root" >&2
  exit 2
fi
if [ "$base_commit" = "$head_commit" ] ||
   [ "$(git -C "$root" rev-parse 'HEAD^{tree}')" = "$(git -C "$root" rev-parse "$base_commit^{tree}")" ]; then
  echo "refusing to run: the baseline revision $baseline_rev is this checkout ($head_commit)," >&2
  echo "so claim 1 (the baseline leaked Han runes) cannot hold, and a revision compared with itself measures" >&2
  echo "nothing. Pass the pre-change revision instead, e.g. $baseline_default." >&2
  exit 2
fi

work="${QK_CRT_PROOF_WORK:-$HOME/.cache/qk-i18n/crt-proof}"
mkdir -p "$work/bin"

# The corpus: one program per message the embedded runtime renders, each failing in exactly one way.
corpus="$work/corpus"
rm -rf "$corpus"
mkdir -p "$corpus"
cat > "$corpus/substring.qk" <<'QK'
fn main(IOStream io) void {
    String s = "abc";
    io.println(s.substring(1, 9));
}
QK
cat > "$corpus/charat.qk" <<'QK'
fn main(IOStream io) void {
    String s = "abc";
    io.println(s.charAt(9));
}
QK
cat > "$corpus/toint.qk" <<'QK'
fn main(IOStream io) void {
    String s = "nope";
    io.println(s.toInt());
}
QK
cat > "$corpus/tofloat.qk" <<'QK'
fn main(IOStream io) void {
    String s = "nope";
    io.println(s.toFloat());
}
QK
cat > "$corpus/ok.qk" <<'QK'
fn main(IOStream io) void {
    io.println(6 * 7);
}
QK
# The control-character cases: the values whose escape rendering is the one declared exception to the
# byte-identical rule above. They are part of the corpus rather than a side table so the claim and the
# code that measures it live in one place.
printf 'fn main(IOStream io) void {\n    String s = "a\\tb";\n    io.println(s.toInt());\n}\n' > "$corpus/esc.qk"
printf 'fn main(IOStream io) void {\n    String s = "a\\\\b";\n    io.println(s.toInt());\n}\n' > "$corpus/back.qk"

build_qkc() { ( cd "$1/compiler" && go build -o "$2" . ) || { echo "qkc build failed: $1" >&2; exit 1; }; }
build_quark() { ( cd "$1" && go build -o "$2" . ) || { echo "quark build failed: $1" >&2; exit 1; }; }

# The baseline: a detached worktree at the pre-change revision, so the comparison is against the code
# as it was rather than against a description of it.
base_tree="$work/baseline"
if [ ! -d "$base_tree" ]; then
  git -C "$root" worktree add -q --detach "$base_tree" "$baseline_rev" || exit 1
fi
git -C "$base_tree" checkout -q --detach "$baseline_rev" || exit 1

build_qkc "$base_tree" "$work/bin/baseline-qkc"
build_qkc "$root" "$work/bin/prototype-qkc"
build_quark "$base_tree" "$work/bin/baseline-quark"
build_quark "$root" "$work/bin/prototype-quark"

# run_compiled <binary> <lang> <file> -> the rendering with the exit code appended, as one line that
# ends with the `rc=` field. Command substitution drops a trailing newline, so the declared renderings in
# the table below end there too: the claim is about the bytes a user would see, and the exit code.
run_compiled() {
  local bin="$1" lang="$2" file="$3" cache="$work/cache"
  rm -rf "$cache"; mkdir -p "$cache"
  ( cd "$corpus" && LANG=C.UTF-8 QK_LANG="$lang" QUARK_CACHE="$cache" "$bin" -run "$file" 2>&1 )
  echo "rc=$?"
}
# run_interp <lang> <binary> <file>
run_interp() { ( cd "$corpus" && LANG=C.UTF-8 QK_LANG="$1" "$2" "$3" 2>&1 ); echo "rc=$?"; }

fail=0
total=0
fixed=0
cases=0
baseline_leaked=0

# The declared control-character exceptions to claim 3: case name, the rendering the baseline produced,
# and the rendering this tree produces — each including the `rc=` line the harness appends, because the
# exit code is part of what must not move. Both texts are written out, so the table is a claim about the
# two revisions that a reviewer can check rather than a list of names that may differ. The texts are
# JSON strings because one of them holds a literal tab.
zh_escapes="$work/zh_escapes.tsv"
cat > "$zh_escapes" <<'TSV'
esc.qk	"error: ParseError: \"a\tb\" \u4e0d\u662f\u5408\u6cd5\u6574\u6570 at line 3\nrc=1"	"error: ParseError: \"a\\tb\" \u4e0d\u662f\u5408\u6cd5\u6574\u6570 at line 3\nrc=1"
back.qk	"error: ParseError: \"a\\b\" \u4e0d\u662f\u5408\u6cd5\u6574\u6570 at line 3\nrc=1"	"error: ParseError: \"a\\\\b\" \u4e0d\u662f\u5408\u6cd5\u6574\u6570 at line 3\nrc=1"
TSV

for f in "$corpus"/*.qk; do
  name="$(basename "$f")"
  cases=$((cases + 1))

  # ---- 3. QK_LANG=zh is byte-identical to the revision this replaces ----
  b_zh="$(run_compiled "$work/bin/baseline-qkc" zh "$name")"
  p_zh="$(run_compiled "$work/bin/prototype-qkc" zh "$name")"
  total=$((total + 1))
  if [ "$b_zh" = "$p_zh" ]; then
    printf 'zh  %-12s identical to %s (%d bytes)\n' "$name" "$baseline_rev" "$(printf '%s' "$p_zh" | wc -c)"
  elif python3 "$here/check_crt_zh.py" "$name" "$zh_escapes" <(printf '%s' "$b_zh") <(printf '%s' "$p_zh"); then
    # A declared escape case: the value's control character renders escaped, the way the interpreter has
    # always rendered it. The English side of the same case is checked below like every other.
    fixed=$((fixed + 1))
    printf 'zh  %-12s declared exception (the value holds a control character)\n' "$name"
  else
    fail=$((fail + 1))
    printf 'zh  %-12s DIFFERS (a Chinese reader can see the migration)\n' "$name"
    diff <(printf '%s\n' "$b_zh") <(printf '%s\n' "$p_zh") | head -8
  fi

  # ---- 1 and 2. the English leak, the fix, and the two engines agreeing ----
  b_en="$(run_compiled "$work/bin/baseline-qkc" en "$name")"
  p_en="$(run_compiled "$work/bin/prototype-qkc" en "$name")"
  i_en="$(run_interp en "$work/bin/prototype-quark" "$name")"
  # Claim 1, counted as well as checked: whether the baseline leaked at all is what tells a wrong baseline
  # apart from a corpus that stopped failing, and the report below says which one this run was.
  if printf '%s' "$b_en" | grep -qP '[\x{4e00}-\x{9fff}]'; then
    baseline_leaked=$((baseline_leaked + 1))
  fi
  total=$((total + 2))
  printf '%s' "$b_en" > "$work/baseline.en"
  printf '%s' "$p_en" > "$work/prototype.en"
  printf '%s' "$i_en" > "$work/interp.en"
  printf '%s' "$p_zh" > "$work/prototype.zh"
  if python3 "$here/check_crt_en.py" "$name" "$work/baseline.en" "$work/prototype.en" "$work/interp.en" "$work/prototype.zh"; then
    printf 'en  %-12s %s (%d bytes)\n' "$name" \
      "$([ "$b_en" = "$p_en" ] && echo identical || echo fixed)" "$(printf '%s' "$p_en" | wc -c)"
  else
    fail=$((fail + 1))
    printf 'en  %-12s FAILED\n' "$name"
  fi
done

# ---- the runtime's list is covered ----
#
# Every id source/c_runtime.txt lists must be a message a case actually prints, or the list has grown
# past what this harness proves. The check runs on the Chinese rendering of the corpus, because that is
# the text both revisions share, and it reads the authored tables rather than a copy of the strings.
missing=0
declared="$work/reached.zh"
: > "$declared"
for f in "$corpus"/*.qk; do
  name="$(basename "$f")"
  run_compiled "$work/bin/prototype-qkc" zh "$name" >> "$declared"
done
while IFS= read -r id; do
  case "$id" in ''|'#'*) continue ;; esac
  zh="$(python3 - "$root/internal/i18n/source" "$id" <<'PY'
import os, sys, re, json
src, want = sys.argv[1], sys.argv[2]
for name in sorted(os.listdir(src)):
    if not name.endswith('.tsv'):
        continue
    for line in open(os.path.join(src, name), encoding='utf-8'):
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        fields = line.split('\t')
        if fields[0] != want:
            continue
        # columns: id, owner, ctx, note, src, then one per language; zh is the last (see languages.tsv)
        print(json.loads(fields[-1]))
        sys.exit(0)
sys.exit(1)
PY
)" || { echo "source/c_runtime.txt names $id, which no authored table carries" >&2; missing=$((missing + 1)); continue; }
  # The message as it renders: each fmt verb stands for an argument one of the corpus programs fills
  # in, so the fixed text either side of it is what has to appear. Literal comparison, not a pattern:
  # a message carries brackets, parentheses and percent signs of its own.
  if ! python3 - "$zh" "$declared" <<'PY'
import re, sys
zh, reached = sys.argv[1], open(sys.argv[2], encoding='utf-8').read()
parts = [p for p in re.split(r"%[-+# 0-9.]*[a-zA-Z]", zh) if p.strip()]
sys.exit(0 if parts and all(p in reached for p in parts) else 1)
PY
  then
    echo "source/c_runtime.txt lists $id, but no corpus program prints it" >&2
    missing=$((missing + 1))
  fi
done < "$root/internal/i18n/source/c_runtime.txt"
total=$((total + 1))
if [ "$missing" -ne 0 ]; then
  fail=$((fail + missing))
else
  printf '%s  %-12s every id in source/c_runtime.txt is reached by the corpus\n' "--" "list"
fi

if [ "$fail" -ne 0 ]; then
  # One cause deserves naming rather than leaving it to be read off six case failures: a baseline that
  # leaked in no case is not a pre-change revision, so claim 1 was never measured and the run says nothing
  # about the product either way. It stays a failure — a gate that called this clean would be a gate that
  # cannot run — but it is not a red gate, and the note says so.
  if [ "$baseline_leaked" -eq 0 ]; then
    echo "note: the baseline ($baseline_rev) leaked a Han rune in 0 of $cases cases, so claim 1 was never" >&2
    echo "measured: that baseline is not a pre-change revision, and the case failures above all report this" >&2
    echo "one cause. Pass the pre-change revision (default $baseline_default) to measure the fix." >&2
  fi
  echo "FAILED: $fail of $total compiled-path checks" >&2
  exit 1
fi

# ---- what the corpus actually demonstrated ----
#
# A proof that cannot fail proves nothing, so the counts are asserted rather than left to the reader.
# Two things have to be true of the run above, and neither is implied by "the script exited 0":
#
#   * every message the runtime can render is reached by a case that fails, so a case that quietly
#     stopped failing (a compiler that answered the bounds check at compile time, say) shows up as a
#     missing message rather than as a proof that shrank without saying so;
#   * the two declared escape cases are reached, so that table is a claim about a path the corpus really
#     runs instead of a comment.
#
# The counts are derived from the corpus and the list, not hardcoded: the escape cases exist only for
# the exception above, and every other case must correspond to one listed message.
leaked=0
escaped=0
for f in "$corpus"/*.qk; do
  name="$(basename "$f")"
  if grep -qF "$(printf '%s\t' "$name")" "$zh_escapes"; then
    escaped=$((escaped + 1))
    continue
  fi
  zh="$(run_compiled "$work/bin/prototype-qkc" zh "$name")"
  if printf '%s' "$zh" | grep -qP '[\x{4e00}-\x{9fff}]'; then
    leaked=$((leaked + 1))
  fi
done
expected=$(grep -c '^[a-z]' "$root/internal/i18n/source/c_runtime.txt")
total=$((total + 1))
if [ "$leaked" -ne "$expected" ] || [ "$escaped" -ne 2 ]; then
  echo "the corpus reached $leaked of $expected listed messages, and $escaped of 2 escape cases" >&2
  fail=$((fail + 1))
else
  printf '%s  %-12s %d listed messages reached, %d declared escape cases exercised\n' \
    "--" "self-check" "$leaked" "$escaped"
fi

if [ "$fail" -ne 0 ]; then
  echo "FAILED: $fail of $total compiled-path checks" >&2
  exit 1
fi
echo "compiled path clean (qkc: $baseline_rev == this tree on the Chinese; no Han rune under en; the interpreter agrees; every listed message reached)  [$total checks]"
