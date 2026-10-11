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

# --- the struct-field pass, and the one case in it that changes ENGLISH: a message written in Chinese
#     inside a struct field. `new T[size]` with a bad size rendered `badAlloc: new int 申请大小非法` in
#     BOTH languages before the pass, because the site concatenated its own literal and never consulted
#     a localizer. The pass gives it an ID whose English source is authored here — nothing in either
#     repository has an English text for it — and whose Chinese value is the shipped text, so the
#     Chinese reader sees exactly what they saw and the English reader stops seeing Chinese.
w('badalloc.qk', 'fn main(IOStream io) void {\n    pointer int p = new int[-1];\n    io.println(1);\n}\n')
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
badalloc.qk	badAlloc: new %s 申请大小非法	badAlloc: new %s has an invalid allocation size
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

# run <binary> <lang> [file] -> the rendering, with the exit code appended
run() { LANG=C.UTF-8 QK_LANG="$2" "$1" ${3:+"$3"} 2>&1; echo "rc=$?"; }

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

# ---- the struct-field pass: the messages no callee-based gate could see ------------------------------
#
# This is the one section where QK_LANG=zh is NOT asserted byte-identical, and the exception is the
# point of the pass. A message written into a struct field (`&RunError{Msg: fmt.Sprintf("...", x)}`) is
# not an argument to any callee, so it never reached a localizer: the site rendered its own English
# literal in both languages while the catalog already carried the Chinese. Wiring the site is what
# changes the Chinese rendering, so equality cannot be the claim. The claim is the leak and the fix,
# checked by name, in the mirror of step 3's EN fixes:
#
#   * the baseline's QK_LANG=zh run must equal its QK_LANG=en run -- there was no language in the site;
#   * this tree's QK_LANG=zh run must be the baseline's with that English template replaced by the
#     Chinese the catalog already shipped, everything else byte for byte;
#   * this tree's QK_LANG=en run must equal the baseline's, because the pass moves where a message
#     lives and never which English text is chosen.
#
# The REPL is the driver rather than a .qk file on purpose: `quark file.qk` runs the static checker
# first, and the checker has its own copy of many of these messages, so the interpreter's site is never
# reached. A dynamically typed program (`Any a = 5; log a.x`) is what gets past the checker and into the
# evaluator, which is where these sites are.
field_fixes="$work/zh_fixes.tsv"
cat > "$field_fixes" <<'TSV'
field_forin	TypeError: for-in requires a List or a container with size() int and get(int), got %s	TypeError: for-in 需要 List，或具备 size() int 与 get(int) 的容器，得到 %s
field_assign_member	TypeError: cannot assign member of %s	TypeError: 不能给 %s 的成员赋值
field_no_member	TypeError: no member %q on %s	TypeError: 成员 %q 在 %s 上不存在
field_unary_minus	TypeError: unary '-' requires a number, got %s	TypeError: 一元 '-' 需要数字，得到 %s
field_order_compare	TypeError: cannot order-compare %s and %s	TypeError: 无法对 %s 和 %s 做大小比较
field_no_method	TypeError: no method %q on %s	TypeError: 方法 %q 在 %s 上不存在
field_not_callable	TypeError: this expression is not callable	TypeError: 这个表达式不可调用
field_appendall	TypeError: appendAll requires a List, got %s	TypeError: appendAll 需要 List，得到 %s
field_taskm_done	TypeError: taskm.done requires a pid (int), got %s	TypeError: taskm.done 需要 pid（int），得到 %s
field_io_static	TypeError: IO has no static method %q	TypeError: IO 没有静态方法 %q
field_list_static	TypeError: List has no static method %q	TypeError: List 没有静态方法 %q
field_hash_static	TypeError: HashTable has no static method %q	TypeError: HashTable 没有静态方法 %q
field_memorize_static	TypeError: memorize has no static method %q	TypeError: memorize 没有静态方法 %q
field_arity_call	CompileError: %s() expects %d args, got %d	CompileError: %s() 需要 %d 个参数，得到 %d
field_arity_call2	CompileError: %s() expects %d args, got %d	CompileError: %s() 需要 %d 个参数，得到 %d
field_undeclared_fn	CompileError: undeclared function %q	CompileError: 未声明的函数 %q
field_star_deref	TypeError: '*' requires a List or a container handle with get(), got %s	TypeError: '*' 需要 List 或带 get() 的容器句柄，得到 %s
field_assign_target	TypeError: unsupported assignment target	TypeError: 不支持的赋值目标
field_sig_wrap	TypeError: a signature can only wrap a direct function call	TypeError: 签名只能包装直接的函数调用
field_assign_undecl	CompileError: assignment to undeclared identifier %q	CompileError: 对未声明的标识符 %q 赋值
field_undecl_read	CompileError: undeclared identifier %q	CompileError: 未声明的标识符 %q
field_unknown_scope	CompileError: unknown scope %q	CompileError: 未知的作用域 %q
typecheck_condition	TypeError: condition must be bool, got %s	TypeError: 条件必须是 bool，得到 %s
typecheck_condition_while	TypeError: condition must be bool, got %s	TypeError: 条件必须是 bool，得到 %s
TSV

build_field() { ( cd "$1" && go build -o "$2" ./cmd/qkrepl ) || { echo "qkrepl build failed: $1" >&2; exit 1; }; }
build_field "$base_tree" "$work/bin/baseline-field-repl"
build_field "$root" "$work/bin/prototype-field-repl"

# run_snippet <binary> <lang> <snippet> -> the rendering with the exit code appended
run_snippet() {
  LANG=C.UTF-8 QK_LANG="$2" "$1" -q -e "$3" 2>&1; echo "rc=$?"
}

field_fail=0
field_case() { # field_case <name> <snippet>
  local name="$1" snippet="$2"
  local b_zh p_zh b_en p_en
  b_zh="$(run_snippet "$work/bin/baseline-field-repl" zh "$snippet")"
  p_zh="$(run_snippet "$work/bin/prototype-field-repl" zh "$snippet")"
  b_en="$(run_snippet "$work/bin/baseline-field-repl" en "$snippet")"
  p_en="$(run_snippet "$work/bin/prototype-field-repl" en "$snippet")"
  total=$((total + 2))

  printf '%s' "$b_zh" > "$work/baseline.zh"
  printf '%s' "$b_en" > "$work/baseline.en"
  printf '%s' "$p_zh" > "$work/prototype.zh"
  printf '%s' "$p_en" > "$work/prototype.en"

  if [ "$b_en" != "$p_en" ]; then
    field_fail=$((field_fail + 1))
    printf '%-3s %-30s DIFFERS (the English text moved)\n' "en" "field/$name"
    diff <(printf '%s\n' "$b_en") <(printf '%s\n' "$p_en") | head -10
  else
    printf '%-3s %-30s identical (%d bytes)\n' "en" "field/$name" "$(printf '%s' "$b_en" | wc -c)"
  fi

  if [ "$b_zh" = "$p_zh" ]; then
    printf '%-3s %-30s identical (%d bytes)\n' "zh" "field/$name" "$(printf '%s' "$b_zh" | wc -c)"
  elif python3 "$here/check_zh_fix.py" "$name" "$field_fixes" "$work/baseline.zh" "$work/baseline.en" "$work/prototype.zh"; then
    fixed=$((fixed + 1))
    printf '%-3s %-30s fixed (English leaked to a Chinese reader, then the shipped translation)\n' "zh" "field/$name"
  else
    field_fail=$((field_fail + 1))
    printf '%-3s %-30s DIFFERS\n' "zh" "field/$name"
    diff <(printf '%s\n' "$b_zh") <(printf '%s\n' "$p_zh") | head -10
  fi
}

field_case "field_forin" 'Any a = 5; for (int x : a) { }'
field_case "field_assign_member" 'Any a = 5; a.x = 1'
field_case "field_no_member" 'Any a = 5; log a.x'
field_case "field_unary_minus" 'Any a = "s"; -a'
field_case "field_order_compare" 'Any a = 1; Any b = "s"; log a < b'
field_case "field_no_method" 'Any a = 5; log a.nope()'
field_case "field_not_callable" 'Any l = [1]; log l.get(0)()'
field_case "field_appendall" 'Any a = 5; List<int> l = [1]; l.appendAll(a)'
field_case "field_taskm_done" 'Any a = "x"; taskm.done(a)'
field_case "field_io_static" 'IO::nope()'
field_case "field_list_static" 'List::nope()'
field_case "field_hash_static" 'HashTable::nope()'
field_case "field_memorize_static" 'memorize::nope()'
field_case "field_arity_call" 'IO::setIn(5)'
field_case "field_arity_call2" 'IO::setOut(5)'
field_case "field_undeclared_fn" 'log nope()'
field_case "field_star_deref" 'Any a = 5; log *a'
field_case "field_assign_target" 'Any a = 5; *a = 1'
field_case "field_sig_wrap" 'Any l = [1]; l.get(0) @nope(x)'
field_case "field_assign_undecl" 'nope = 1'
field_case "field_undecl_read" 'log nope'
field_case "field_unknown_scope" 'log Nope::f()'

# --- the typecheck domain's one declared ZH fix. It is the same shape as the struct-field pass: the
#     literal never reached a localizer, so the catalog's Chinese for it was unreachable and a Chinese
#     reader got English. The checker has the message too, but a dynamically typed condition reaches the
#     evaluator's copy first, which is why this snippet is the one that shows it.
field_case "typecheck_condition" 'Any a = 1; if (a) { log 1; }'
field_case "typecheck_condition_while" 'Any a = "s"; while (a) { }'

fail=$((fail + field_fail))

if [ "$fail" -ne 0 ]; then
  echo "FAILED: $fail of $total runs differ from $baseline_rev" >&2
  exit 1
fi

# ---- the docgen domain: a rendered document, not a diagnostic ------------------------------------
#
# A document is not one message, so the per-message fix table above does not fit it. Two claims do, and
# they are the two that matter:
#
#   * under QK_LANG=zh the document is byte-identical, like every other case here;
#   * under QK_LANG=en it contains no Han rune, and the baseline's did. The docgen domain had nine
#     Chinese labels hardcoded in the renderer (kindLabel's 函数/结构体/接口/… and the section titles),
#     never handed to a localizer at all, so an English reader got a Chinese document and no scanner
#     could see it: a hardcoded message is invisible to a tool that looks for literals passed to an
#     i18n boundary. This is the check that would have caught it.
#
# The EN rendering is allowed to differ, and saying so explicitly is the point: for this domain the
# difference is the fix, and the Han check is what keeps "it differs" from being a blank cheque.

doc_corpus="$work/docgen"
rm -rf "$doc_corpus"
mkdir -p "$doc_corpus"
cat > "$doc_corpus/all.qk" <<'QK'
/* A documented module. */

pub type struct {
    int x; /* the x coordinate */
    int y; /* the y coordinate */
} Point;

pub type interface {
    fn area() int; /* the area */
} Shape;

pub fn origin() int {
    return 1;
}
QK
cat > "$doc_corpus/empty.qk" <<'QK'
/* Nothing public here. */
fn hidden() int {
    return 1;
}
QK

build_qkdoc() { ( cd "$1" && go build -o "$2" ./cmd/qkdoc ) || { echo "qkdoc build failed: $1" >&2; exit 1; }; }
build_qkdoc "$base_tree" "$work/bin/baseline-qkdoc"
build_qkdoc "$root" "$work/bin/prototype-qkdoc"

doc_fail=0
for f in "$doc_corpus"/*.qk; do
  name="$(basename "$f")"
  b_zh="$(run "$work/bin/baseline-qkdoc" zh "$f")"
  p_zh="$(run "$work/bin/prototype-qkdoc" zh "$f")"
  b_en="$(run "$work/bin/baseline-qkdoc" en "$f")"
  p_en="$(run "$work/bin/prototype-qkdoc" en "$f")"
  total=$((total + 3))

  if [ "$b_zh" != "$p_zh" ]; then
    doc_fail=$((doc_fail + 1))
    printf 'zh  doc/%-26s DIFFERS\n' "$name"
    diff <(printf '%s\n' "$b_zh") <(printf '%s\n' "$p_zh") | head -10
  else
    printf '%-3s %-30s identical (%d bytes)\n' "zh" "doc/$name" "$(printf '%s' "$b_zh" | wc -c)"
  fi

  printf '%s' "$b_en" > "$work/baseline.out"
  printf '%s' "$p_en" > "$work/prototype.out"
  if python3 "$here/check_docgen_en.py" "$work/baseline.out" "$work/prototype.out"; then
    fixed=$((fixed + 1))
    printf '%-3s %-30s fixed (Chinese labels, then English)\n' "en" "doc/$name"
  else
    doc_fail=$((doc_fail + 1))
    printf '%-3s %-30s DIFFERS\n' "en" "doc/$name"
  fi
done
fail=$((fail + doc_fail))

# ---- the shared domain: the compiler's own usage text ---------------------------------------------
#
# `qkc` with no arguments is the only way to reach this message, and it is the repository's own (domain
# `shared`, one row). It is a Chinese-source row, and the one place the migration is interesting is that
# the old design did NOT leak it: the English value was selected, so `QK_LANG=en qkc` already printed
# English. Both languages are therefore asserted byte-identical, and there is no declared EN fix here.
#
#   $ QK_LANG=zh qkc    usage: qkc [options] <file.qk> … （默认） …     (before and after)
#
# The Chinese help is deliberately NOT extended with the `--lang` row the English text carries (the
# English text is 20 lines, the Chinese 19, and that predates this commit). Migrating a domain moves
# where the text lives, never which text is chosen, so the Chinese rendering stays exactly what it was;
# inventing a translation to make the two columns look parallel is the retyping the migration forbids.
# The asymmetry is reported in the commit message and §7.4, not papered over.
build_qkc() { ( cd "$1/compiler" && go build -o "$2" . ) || { echo "qkc build failed: $1" >&2; exit 1; }; }
build_qkc "$base_tree" "$work/bin/baseline-qkc"
build_qkc "$root" "$work/bin/prototype-qkc"

shared_fail=0
for lang in en zh; do
  b="$(run "$work/bin/baseline-qkc" "$lang")"
  p="$(run "$work/bin/prototype-qkc" "$lang")"
  total=$((total + 1))
  if [ "$b" != "$p" ]; then
    shared_fail=$((shared_fail + 1))
    printf '%-3s %-30s DIFFERS\n' "$lang" "shared/usage"
    diff <(printf '%s\n' "$b") <(printf '%s\n' "$p") | head -10
  else
    printf '%-3s %-30s identical (%d bytes)\n' "$lang" "shared/usage" "$(printf '%s' "$b" | wc -c)"
  fi
done
fail=$((fail + shared_fail))

# ---- the repl domain: the interactive session's echo -------------------------------------------------
#
# The session's messages are not reachable from a process-level corpus: `qkrepl` reads stdin, and the
# echo comes back through the session API. check_repl.py writes one test program, builds it against both
# revisions (it has to sit next to the package it tests, because it uses an internal import) and requires
# byte-identical output under both languages. Both languages are identical, so nothing is declared.
repl_fail=0
if python3 "$here/check_repl.py" "$base_tree" "$root" "$work"; then
  total=$((total + 2))
else
  repl_fail=1
fi
fail=$((fail + repl_fail))

# ---- the lint domain: qkcheck's diagnostics ---------------------------------------------------------
#
# Every one of the domain's 22 rows reaches `warn`/`warnID`, and `qkcheck` prints the result, so a
# process-level corpus does reach them. Both languages are asserted byte-identical, and this domain
# declares no EN fix. That is a measurement, not an assumption: the migration's reading found
# `lintKindName` returning `msg("形参")`/`msg("变量")`, which looks like the docgen class of leak — the
# labels are not in the lint table — but it is not one, because this domain's labels resolve through the
# shared `compiler` table's `"形参": "parameter"` row. An English reader got "parameter x shadows …"
# before and gets it now. The rows are the compiler domain's, and they are deleted when that domain
# migrates, which is where the label lineage has to be checked; this harness is the half of that check
# that lives here.
#
# The label rows the migration DID move (loop variable, catch variable, and the two with no catalog entry
# at all) are pinned by TestLintKindLabelsAreMessages in internal/lang, which renders them in both
# languages, because no corpus reaches all four kinds.
lint_corpus="$work/lint"
rm -rf "$lint_corpus"
mkdir -p "$lint_corpus"
cat > "$lint_corpus/kinds.qk" <<'QK'
fn used(int a) int {
    int shadow = a;
    if (true) {
        int a = shadow;
        io.println(a);
    }
    return shadow;
}
fn main(IOStream io) void {
    int dead = used(1);
}
QK
cat > "$lint_corpus/mixed.qk" <<'QK'
/* Nothing public here. */
fn hidden(int p) int {
    int q = p;
    return q;
}
fn main(IOStream io) void {
    int unused = 1;
    io.println(2);
}
QK

build_qkcheck() { ( cd "$1" && go build -o "$2" ./cmd/qkcheck ) || { echo "qkcheck build failed: $1" >&2; exit 1; }; }
build_qkcheck "$base_tree" "$work/bin/baseline-qkcheck"
build_qkcheck "$root" "$work/bin/prototype-qkcheck"

lint_fail=0
for f in "$lint_corpus"/*.qk; do
  name="$(basename "$f")"
  for lang in en zh; do
    b="$(run "$work/bin/baseline-qkcheck" "$lang" "$f")"
    p="$(run "$work/bin/prototype-qkcheck" "$lang" "$f")"
    total=$((total + 1))
    if [ "$b" != "$p" ]; then
      lint_fail=$((lint_fail + 1))
      printf '%-3s %-30s DIFFERS\n' "$lang" "lint/$name"
      diff <(printf '%s\n' "$b") <(printf '%s\n' "$p") | head -10
    else
      printf '%-3s %-30s identical (%d bytes)\n' "$lang" "lint/$name" "$(printf '%s' "$b" | wc -c)"
    fi
  done
done
fail=$((fail + lint_fail))

# ---- the cli domain: the four tools' own messages ----------------------------------------------------
#
# These are the messages the tools print for a bad invocation, an unknown command and an unimplemented
# LSP method. The domain migrated through a localizer each cmd/ package now owns, and two of its 26 rows
# were reached through a *variable* — `replyErr(id, code, format, …)` and `i18n.T(helpText)` — which is
# the third way a message hides from a literal scan and the reason the call sites had to be rewritten
# rather than the rows moved.
#
# Two of this domain's messages never reached an i18n boundary at all and are the declared EN fixes:
# qklsp's JSON error was built with fmt.Errorf on its own literal, and `:help` printed a Chinese raw
# literal. In both, the baseline rendered Chinese to an English reader; check_en_leak.py proves the fix
# by name, and the ZH run is asserted byte-identical like every other case.
cli_corpus="$work/cli"
rm -rf "$cli_corpus"
mkdir -p "$cli_corpus"
printf 'fn main(IOStream io) void {\n    io.println(1);\n}\n' > "$cli_corpus/ok.qk"
printf 'fn hidden() int {\n    return 1;\n}\n' > "$cli_corpus/hidden.qk"

# one leak table per case, because check_en_leak.py measures the baseline against the table it is given
leaks_repl_help="$work/cli_repl_help_leaks.tsv"
cat > "$leaks_repl_help" <<'TSV'
命令：
  :help            显示本帮助
  :quit / :exit    退出
  :load <file.qk>  加载文件：登记其中的函数/类型/实现（不自动执行 main；可随后调用 main(io)）
说明：
  - 表达式直接回显其值；语句以 ; 结尾（省略时自动补）
  - 块未闭合会自动续行；变量/函数/类型在后续输入中持续可见
  - log 的记录、io.println 的输出会即时显示	
TSV
leaks_lsp_json="$work/cli_lsp_json_leaks.tsv"
# the leaked text as it RENDERS, not as it is written in the source: the message is a %w verb, so the
# baseline prints "qklsp: JSON 解析失败: invalid character …" and the table has to name that prefix.
printf 'qklsp: JSON 解析失败:\tqklsp: JSON parse failed:\n' > "$leaks_lsp_json"

build_tool() { ( cd "$1" && go build -o "$2" "./cmd/$3" ) || { echo "$3 build failed: $1" >&2; exit 1; }; }
for tool in qkcheck qkdoc qkrepl; do
  build_tool "$base_tree" "$work/bin/baseline-$tool" "$tool"
  build_tool "$root" "$work/bin/prototype-$tool" "$tool"
done

# run_argv <binary> <lang> <args...> -> output with the exit code appended, at a fixed working directory
run_argv() {
  local bin="$1" lang="$2"; shift 2
  ( cd "$work/cli" && LANG=C.UTF-8 QK_LANG="$lang" "$bin" "$@" 2>&1 ); echo "rc=$?"
}

cli_fail=0
cli_case() { # cli_case <name> <tool> <args...>
  local name="$1" tool="$2"; shift 2
  local b p
  b="$(run_argv "$work/bin/baseline-$tool" zh "$@")"
  p="$(run_argv "$work/bin/prototype-$tool" zh "$@")"
  total=$((total + 1))
  if [ "$b" != "$p" ]; then
    cli_fail=$((cli_fail + 1))
    printf 'zh  %-30s DIFFERS\n' "cli/$name"
    diff <(printf '%s\n' "$b") <(printf '%s\n' "$p") | head -10
  else
    printf '%-3s %-30s identical (%d bytes)\n' "zh" "cli/$name" "$(printf '%s' "$b" | wc -c)"
  fi
  b="$(run_argv "$work/bin/baseline-$tool" en "$@")"
  p="$(run_argv "$work/bin/prototype-$tool" en "$@")"
  total=$((total + 1))
  printf '%s' "$b" > "$work/baseline.out"
  printf '%s' "$p" > "$work/prototype.out"
  if [ "$b" = "$p" ]; then
    printf '%-3s %-30s identical (%d bytes)\n' "en" "cli/$name" "$(printf '%s' "$b" | wc -c)"
  elif python3 "$here/check_en_leak.py" "cli/$name" "$work/baseline.out" "$work/prototype.out" "$leaks_repl_help"; then
    fixed=$((fixed + 1))
    printf '%-3s %-30s fixed (a hardcoded Chinese literal, then the translation)\n' "en" "cli/$name"
  else
    cli_fail=$((cli_fail + 1))
    printf '%-3s %-30s DIFFERS\n' "en" "cli/$name"
  fi
}

cli_case "check_usage"   qkcheck
cli_case "check_bad_flag" qkcheck -nope
cli_case "check_no_files" qkcheck -L .
cli_case "check_ok"      qkcheck ok.qk
cli_case "doc_usage"     qkdoc
cli_case "doc_bad_flag"  qkdoc -nope
cli_case "doc_report"    qkdoc -o out.md ok.qk
cli_case "repl_usage"    qkrepl
cli_case "repl_bad_flag" qkrepl -nope

# The REPL's own commands come from stdin, not from -e: `-e` is evaluated as an expression, so `:help`
# there is a parse error and the message under test never renders. The first harness version used -e and
# silently measured "both binaries reject :help identically", which is true and proves nothing.
run_stdin() { # run_stdin <binary> <lang> <input> <args...>
  local bin="$1" lang="$2" input="$3"; shift 3
  ( cd "$work/cli" && printf '%s' "$input" | LANG=C.UTF-8 QK_LANG="$lang" "$bin" "$@" 2>&1 ); echo "rc=$?"
}
cli_stdin_case() { # cli_stdin_case <name> <input> <args...>
  local name="$1" input="$2"; shift 2
  local b p
  b="$(run_stdin "$work/bin/baseline-qkrepl" zh "$input" "$@")"
  p="$(run_stdin "$work/bin/prototype-qkrepl" zh "$input" "$@")"
  total=$((total + 1))
  if [ "$b" != "$p" ]; then
    cli_fail=$((cli_fail + 1))
    printf 'zh  %-30s DIFFERS\n' "cli/$name"
    diff <(printf '%s\n' "$b") <(printf '%s\n' "$p") | head -10
  else
    printf '%-3s %-30s identical (%d bytes)\n' "zh" "cli/$name" "$(printf '%s' "$b" | wc -c)"
  fi
  b="$(run_stdin "$work/bin/baseline-qkrepl" en "$input" "$@")"
  p="$(run_stdin "$work/bin/prototype-qkrepl" en "$input" "$@")"
  total=$((total + 1))
  printf '%s' "$b" > "$work/baseline.out"
  printf '%s' "$p" > "$work/prototype.out"
  if [ "$b" = "$p" ]; then
    printf '%-3s %-30s identical (%d bytes)\n' "en" "cli/$name" "$(printf '%s' "$b" | wc -c)"
  elif python3 "$here/check_en_leak.py" "cli/$name" "$work/baseline.out" "$work/prototype.out" "$leaks_repl_help"; then
    fixed=$((fixed + 1))
    printf '%-3s %-30s fixed (a hardcoded Chinese literal, then the translation)\n' "en" "cli/$name"
  else
    cli_fail=$((cli_fail + 1))
    printf '%-3s %-30s DIFFERS\n' "en" "cli/$name"
  fi
}

cli_stdin_case "repl_help"    ':help
:quit
'
cli_stdin_case "repl_bad_cmd" ':nope
:quit
'
cli_stdin_case "repl_load"    ':load nope.qk
:quit
'

# qklsp speaks JSON-RPC on stdio, so its framing errors are reachable with a pipe. A Content-Length that
# is not a number and a body that is not JSON are the two paths; the second is the declared EN fix,
# because the baseline built it with fmt.Errorf on its own Chinese literal.
run_lsp() { # run_lsp <binary> <lang> <frame>
  local bin="$1" lang="$2" frame="$3"
  ( printf '%b' "$frame" | LANG=C.UTF-8 QK_LANG="$lang" timeout 10 "$bin" 2>&1 ); echo "rc=$?"
}
lsp_case() { # lsp_case <name> <frame>
  local name="$1" frame="$2"
  local b p
  b="$(run_lsp "$work/bin/baseline-qklsp" zh "$frame")"
  p="$(run_lsp "$work/bin/prototype-qklsp" zh "$frame")"
  total=$((total + 1))
  if [ "$b" != "$p" ]; then
    cli_fail=$((cli_fail + 1))
    printf 'zh  %-30s DIFFERS\n' "cli/$name"
    diff <(printf '%s\n' "$b") <(printf '%s\n' "$p") | head -10
  else
    printf '%-3s %-30s identical (%d bytes)\n' "zh" "cli/$name" "$(printf '%s' "$b" | wc -c)"
  fi
  b="$(run_lsp "$work/bin/baseline-qklsp" en "$frame")"
  p="$(run_lsp "$work/bin/prototype-qklsp" en "$frame")"
  total=$((total + 1))
  printf '%s' "$b" > "$work/baseline.out"
  printf '%s' "$p" > "$work/prototype.out"
  if [ "$b" = "$p" ]; then
    printf '%-3s %-30s identical (%d bytes)\n' "en" "cli/$name" "$(printf '%s' "$b" | wc -c)"
  elif python3 "$here/check_en_leak.py" "cli/$name" "$work/baseline.out" "$work/prototype.out" "$leaks_lsp_json"; then
    fixed=$((fixed + 1))
    printf '%-3s %-30s fixed (a hardcoded Chinese literal, then the translation)\n' "en" "cli/$name"
  else
    cli_fail=$((cli_fail + 1))
    printf '%-3s %-30s DIFFERS\n' "en" "cli/$name"
  fi
}

build_tool "$base_tree" "$work/bin/baseline-qklsp" qklsp
build_tool "$root" "$work/bin/prototype-qklsp" qklsp
lsp_case "lsp_bad_length" 'Content-Length: xyz\r\n\r\n'
lsp_case "lsp_bad_json"   'Content-Length: 5\r\n\r\nnotjs'
fail=$((fail + cli_fail))

# ---- the runtime domain: the interpreter's own diagnostics -------------------------------------------
#
# These are the messages the evaluator raises while a program runs, so a corpus reaches them by running
# programs that fail in the specific ways. Both languages are asserted byte-identical: the domain's rows
# were already wired through msg(), so nothing here is a declared fix.
#
# Two of the 65 rows had NO msg() site, and that is the domain's finding: their call sites were
# `&RunError{Msg: "literal"}` — a message written into a struct field, which no callee-based scanner can
# see. Both are wired here rather than in a follow-up, because leaving them would mean the catalog kept
# a Chinese translation that the code never asked for.
#
# The corpus is restricted to programs the LINTER accepts, because a message raised before it is a
# compile error never runs: the first version of this corpus used `string` (the language's type is
# `String`) and measured 57 identical bytes of "unknown type", which proves nothing about the domain.
runtime_corpus="$work/runtime"
rm -rf "$runtime_corpus"
mkdir -p "$runtime_corpus"
cat > "$runtime_corpus/index.qk" <<'QK'
fn main(IOStream io) void {
    List<int> xs = [1, 2];
    io.println(xs[9]);
}
QK
cat > "$runtime_corpus/substr.qk" <<'QK'
fn main(IOStream io) void {
    string s = "abc";
    io.println(s.substring(1, 9));
}
QK
cat > "$runtime_corpus/bitwise.qk" <<'QK'
fn main(IOStream io) void {
    string s = "a";
    io.println(s & s);
}
QK
cat > "$runtime_corpus/complement.qk" <<'QK'
fn main(IOStream io) void {
    string s = "a";
    io.println(~s);
}
QK
cat > "$runtime_corpus/divzero.qk" <<'QK'
fn main(IOStream io) void {
    int z = 0;
    io.println(1 / z);
}
QK
cat > "$runtime_corpus/libload.qk" <<'QK'
library nope;
fn main(IOStream io) void { }
QK
cat > "$runtime_corpus/stringargs.qk" <<'QK'
fn main(IOStream io) void {
    string s = "abc";
    io.println(s.contains(1));
}
QK

runtime_fail=0
for f in "$runtime_corpus"/*.qk; do
  name="$(basename "$f")"
  for lang in en zh; do
    b="$(run "$work/bin/baseline-quark" "$lang" "$f")"
    p="$(run "$work/bin/prototype-quark" "$lang" "$f")"
    total=$((total + 1))
    if [ "$b" != "$p" ]; then
      runtime_fail=$((runtime_fail + 1))
      printf '%-3s %-30s DIFFERS\n' "$lang" "runtime/$name"
      diff <(printf '%s\n' "$b") <(printf '%s\n' "$p") | head -10
    else
      printf '%-3s %-30s identical (%d bytes)\n' "$lang" "runtime/$name" "$(printf '%s' "$b" | wc -c)"
    fi
  done
done
fail=$((fail + runtime_fail))
if [ "$fail" -ne 0 ]; then
  echo "FAILED: $fail of $total runs differ from $baseline_rev" >&2
  exit 1
fi
# The count is of cases that took a fix branch, and which domains those are depends on what the baseline
# already carries: the fixes of a domain that has merged are in origin/main, so its cases are identical
# here and only the newly migrated ones are counted. That is why the number moves down as PRs land.
echo "all identical (interpreter, qkdoc, qkc, repl, qkcheck: $baseline_rev == this tree), en and zh  [$total runs, $fixed of them a declared fix]"
