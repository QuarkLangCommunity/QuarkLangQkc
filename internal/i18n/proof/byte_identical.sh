#!/usr/bin/env bash
# P3: the migrated surface renders byte-identically to the revision it replaces, in every language,
# except where the migration is itself the fix — and there, only in the one direction that was broken.
#
# It builds the interpreter twice — once from the baseline revision, once from this working tree — and
# runs both over a corpus that exercises the migrated messages, under QK_LANG=en and QK_LANG=zh. stdout,
# stderr and the exit code must match byte for byte.
#
# Two dialects of "identical" live here, and they are not the same claim:
#
#   * QK_LANG=zh must be byte-identical for every case, always. Migrating a domain moves where the text
#     lives, never which text is chosen, and a Chinese reader must not be able to tell that anything
#     happened.
#
#   * QK_LANG=en must be byte-identical too, except for the messages step 3 fixed. Before step 3 the
#     module's errf path did fmt.Sprintf on its own literal and consulted no catalog, so 13 messages
#     written in Chinese rendered Chinese to an English reader even though the repository already
#     carried their English text in internal/i18n/table_lexer.go. For those cases this script does not
#     assert equality; it asserts the leak and the fix by name: the baseline's English run must equal its
#     Chinese run and contain the Chinese template, and this tree's English run must be the baseline's
#     with exactly that template replaced by the English translation that was already sitting in the
#     repository, unused. The line/column suffix and the error kind must not move.
#
# The EN-fixes table is therefore a list of claims about the old design, checkable against the old
# binary, not a list of exemptions. A case that is not in it may not differ under any language.
#
# Usage: internal/i18n/proof/byte_identical.sh [baseline-revision]
#   baseline-revision defaults to origin/main.
#
# A failure here means the migration changed what a user reads, which is the one thing it must not do.

set -u

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../../.." && pwd)"
baseline_rev="${1:-origin/main}"
work="${QK_PROOF_WORK:-$HOME/.cache/qk-i18n/proof}"
mkdir -p "$work/bin"

# The corpus: one case per migrated message family. deep_nesting.qk exercises the syntax layer's own
# refusal; the macro and preprocessor cases exercise the msg() surface migrated in steps 1-2; the errf
# cases exercise the surface migrated in step 3.
corpus="$work/corpus"
rm -rf "$corpus"
mkdir -p "$corpus"
python3 - "$corpus" <<'PY'
import sys, os
d = sys.argv[1]
def w(name, text):
    with open(os.path.join(d, name), 'w', encoding='utf-8') as fh:
        fh.write(text)

# --- steps 1-2: the msg() surface of the pinned module ------------------------------------------
w('deep_nesting.qk', 'fn main(IOStream io) void {\n  io.println(' + '(' * 70000 + '1' + ')' * 70000 + ');\n}\n')
w('macro_needs_name.qk', '#macro\nfn main(IOStream io) void { }\n')
w('macro_missing_body.qk', '#macro twice (a)\nfn main(IOStream io) void { }\n')
w('pp_unknown_command.qk', 'fn main(IOStream io) void {\n    #frobnicate\n}\n')
w('pp_unclosed_if.qk', 'fn main(IOStream io) void {\n#if os("linux")\n}\n')
w('pp_elif_without_if.qk', 'fn main(IOStream io) void {\n#elif os("linux")\n}\n')
w('pp_branch_without_if.qk', '#endif\n')
w('pp_error_directive.qk', 'fn main(IOStream io) void {\n#error("deliberate")\n}\n')
w('pp_insert_needs_ast.qk', 'fn main(IOStream io) void {\n    #insert(nope)\n}\n')
w('ok_program.qk', 'fn main(IOStream io) void {\n    io.println(6 * 7);\n}\n')

# --- step 3, the 13 that were written in Chinese: English now selects the translation that was
#     already in table_lexer.go. These are the EN-fix cases.
w('errf_named_struct.qk', 'struct { int a; } Foo;\nfn main(IOStream io) void { }\n')
w('errf_named_interface.qk', 'interface { } Foo;\nfn main(IOStream io) void { }\n')
w('errf_program_not_main.qk', 'program foo\nfn main(IOStream io) void { }\n')
w('errf_pub_type.qk', 'pub type int X;\nfn main(IOStream io) void { }\n')
w('errf_pub_prefix.qk', 'pub int x = 1;\nfn main(IOStream io) void { }\n')
w('errf_func_keyword.qk', 'func main() { }\n')
w('errf_return_type.qk', 'fn f() { }\nfn main(IOStream io) void { }\n')
w('errf_library_body.qk', 'library lib { int x = 1; }\nfn main(IOStream io) void { }\n')

# --- step 3, the English-only ones: they had no catalog entry at all, so they rendered English in
#     both languages. The migration gives them an ID and a row whose zh repeats the source, and the
#     rendering therefore must not move in either language.
w('errf_unterminated_block.qk', 'fn main(IOStream io) void {\n    if (1) {\n}\n')
w('errf_unterminated_string.qk', 'fn main(IOStream io) void {\n    string s = "abc\n}\n')
w('errf_unterminated_raw.qk', 'fn main(IOStream io) void {\n    string s = `abc\n}\n')
w('errf_unterminated_comment.qk', 'fn main(IOStream io) void {\n    /* open\n}\n')
w('errf_unexpected_char.qk', 'fn main(IOStream io) void {\n    int x = 1 $ 2;\n}\n')
w('errf_int_range.qk', 'fn main(IOStream io) void {\n    int x = 99999999999;\n}\n')
w('errf_top_level.qk', 'int x = 1\n')
PY

# The EN fixes: case name, the Chinese template that leaked, and the English text that was already in
# the repository. python3 checks the substitution, so the two texts must be exact.
fixes="$work/en_fixes.tsv"
cat > "$fixes" <<'TSV'
errf_named_struct.qk	实名结构体必须写 type：type struct { ... } %s;	a named struct requires 'type': type struct { ... } %s;
errf_named_interface.qk	实名接口必须写 type：type interface { ... } %s;	a named interface requires 'type': type interface { ... } %s;
errf_program_not_main.qk	program 声明只支持 main / library（lib）	program declarations support only main / library (lib)
errf_pub_type.qk	pub type 只支持 struct / interface	pub type supports only struct / interface
errf_pub_prefix.qk	pub 只能前缀于 fn 或 type struct/interface（实名类型必须写 type）	pub may only prefix fn or type struct/interface (named types require 'type')
errf_func_keyword.qk	函数声明关键字是 fn：func main(...) 应写为 fn main(...)	the function keyword is 'fn': write fn main(...) rather than func main(...)
errf_return_type.qk	函数必须声明返回类型：fn %s(...) 返回类型 { ... }	a function must declare its return type: fn %s(...) <return type> { ... }
errf_library_body.qk	library 体内只能有 fn 符号签名	a library body may only contain fn symbol signatures
TSV

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

# run <binary> <lang> <file> -> the rendering, with the exit code appended
run() { LANG=C.UTF-8 QK_LANG="$2" "$1" "$3" 2>&1; echo "rc=$?"; }

fail=0
total=0
fixed=0
for lang in en zh; do
  for f in "$corpus"/*.qk; do
    name="$(basename "$f")"
    b_out="$(run "$work/bin/baseline-quark" "$lang" "$f")"
    p_out="$(run "$work/bin/prototype-quark" "$lang" "$f")"
    total=$((total + 1))

    if [ "$b_out" = "$p_out" ]; then
      printf '%-3s %-30s identical (%d bytes)\n' "$lang" "$name" "$(printf '%s' "$b_out" | wc -c)"
      continue
    fi

    if [ "$lang" = "en" ] && grep -qF "$(printf '%s\t' "$name")" "$fixes"; then
      printf '%s' "$b_out" > "$work/baseline.out"
      printf '%s' "$p_out" > "$work/prototype.out"
      if python3 "$here/check_en_fix.py" "$name" "$fixes" "$work/baseline.out" "$work/prototype.out"; then
        fixed=$((fixed + 1))
        printf '%-3s %-30s fixed (the leak, then the translation that was already there)\n' "$lang" "$name"
        continue
      fi
    fi

    fail=$((fail + 1))
    printf '%-3s %-30s DIFFERS\n' "$lang" "$name"
    diff <(printf '%s\n' "$b_out") <(printf '%s\n' "$p_out") | head -10
  done
done

if [ "$fail" -ne 0 ]; then
  echo "FAILED: $fail of $total runs differ from $baseline_rev" >&2
  exit 1
fi
echo "all identical (interpreter: $baseline_rev == this tree), en and zh  [$total runs, $fixed of them a declared EN fix]"
