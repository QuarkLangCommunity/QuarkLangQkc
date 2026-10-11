#!/usr/bin/env bash
# P3: the migrated domain renders byte-identically to the design it replaces.
#
# It builds the interpreter twice — once from the pre-migration revision, once from this working tree —
# and runs both over a corpus that exercises the migrated messages (the parser-module nesting refusal,
# the macro and preprocessor diagnostics), under QK_LANG=en and QK_LANG=zh. stdout, stderr and the exit
# code must match byte for byte.
#
# Usage: internal/i18n/proof/byte_identical.sh [baseline-revision]
#   baseline-revision defaults to origin/main.
#
# A failure here means the migration changed what a user reads, which is the one thing it must not do:
# migrating a domain moves where the text lives, never which text is chosen.

set -u

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../../.." && pwd)"
baseline_rev="${1:-origin/main}"
work="${QK_PROOF_WORK:-$HOME/.cache/qk-i18n/proof}"
mkdir -p "$work/bin"

# The corpus: one case per migrated message family. deep_nesting.qk exercises the syntax layer's own
# refusal (the message whose catalog direction was inverted twice in one day); the macro and
# preprocessor cases exercise the msg() surface of the same pinned module.
corpus="$work/corpus"
mkdir -p "$corpus"
python3 - "$corpus" <<'PY'
import sys, os
d = sys.argv[1]
def w(name, text):
    open(os.path.join(d, name), 'w', encoding='utf-8').write(text)
w('deep_nesting.qk', 'fn main(IOStream io) void {\n  io.println(' + '(' * 70000 + '1' + ')' * 70000 + ');\n}\n')
w('macro_needs_name.qk', '#macro\nfn main(IOStream io) void { }\n')
w('macro_missing_body.qk', '#macro twice (a)\nfn main(IOStream io) void { }\n')
w('pp_unknown_command.qk', 'fn main(IOStream io) void {\n    #frobnicate\n}\n')
w('pp_unclosed_if.qk', 'fn main(IOStream io) void {\n#if os("linux")\n}\n')
w('pp_elif_without_if.qk', 'fn main(IOStream io) void {\n#elif os("linux")\n}\n')
w('pp_error_directive.qk', 'fn main(IOStream io) void {\n#error("deliberate")\n}\n')
w('pp_insert_needs_ast.qk', 'fn main(IOStream io) void {\n    #insert(nope)\n}\n')
w('ok_program.qk', 'fn main(IOStream io) void {\n    io.println(6 * 7);\n}\n')
PY

build() { # build <revision-or-worktree> <output>
  local src="$1" out="$2"
  ( cd "$src" && go build -o "$out" . ) || { echo "build failed: $src" >&2; exit 1; }
}

# The baseline: a detached worktree at the pre-migration revision, so the comparison is against the
# code as it was, not against a description of it.
base_tree="$work/baseline"
if [ ! -d "$base_tree" ]; then
  git -C "$root" worktree add -q --detach "$base_tree" "$baseline_rev" || exit 1
fi
git -C "$base_tree" checkout -q --detach "$baseline_rev" || exit 1

build "$base_tree" "$work/bin/baseline-quark"
build "$root" "$work/bin/prototype-quark"

fail=0
total=0
for lang in en zh; do
  for f in "$corpus"/*.qk; do
    name="$(basename "$f")"
    b_out="$(LANG=C.UTF-8 QK_LANG=$lang "$work/bin/baseline-quark" "$f" 2>&1; echo "rc=$?")"
    p_out="$(LANG=C.UTF-8 QK_LANG=$lang "$work/bin/prototype-quark" "$f" 2>&1; echo "rc=$?")"
    total=$((total + 1))
    if [ "$b_out" = "$p_out" ]; then
      printf '%-3s %-24s identical (%d bytes)\n' "$lang" "$name" "$(printf '%s' "$b_out" | wc -c)"
    else
      fail=$((fail + 1))
      printf '%-3s %-24s DIFFERS\n' "$lang" "$name"
      diff <(printf '%s\n' "$b_out") <(printf '%s\n' "$p_out") | head -10
    fi
  done
done

if [ "$fail" -ne 0 ]; then
  echo "FAILED: $fail of $total runs differ from $baseline_rev" >&2
  exit 1
fi
echo "all identical (interpreter: $baseline_rev == this tree), en and zh  [$total runs]"
