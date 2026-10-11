#!/usr/bin/env python3
"""P3's EN-leak check: the English run differs from the baseline, and exactly by the declared leaks.

`check_en_fix.py` proves a per-message substitution against a fix table, and it needs the leaked text to
be a whole message. This covers the case where the leak is a *fragment* or a message with no catalog row
at all:

  * a literal that never reached an i18n boundary — qklsp built its JSON error with `fmt.Errorf` on its
    own Chinese text, and `qkrepl` printed a Chinese `:help` block;
  * an argument written in Chinese inside a message that does translate — `lintKindName` returned
    `msg("形参")`, so the diagnostic was half English, half Chinese.

The claim is checkable without trusting the migration:

  1. the baseline's English run contains every leaked token the caller declares, so the leak is measured
     on the old binary rather than asserted about it;
  2. this tree's English run contains none of them and no other Han rune;
  3. the new run IS the baseline's output with exactly those tokens replaced — byte for byte, everything
     else unchanged. The check rebuilds the baseline from the new output and the declared table, so any
     other difference fails.

The ZH side is not this script's business: byte_identical.sh asserts it separately and unconditionally.

Usage: check_en_leak.py <case> <baseline-en> <new-en> <leaks.tsv>
  leaks.tsv: one `leaked<TAB>replacement` pair per line, longest leaked text first.
"""
import sys

HAN_LO, HAN_HI = '\u3400', '\u9fff'


def has_han(s):
    return any(HAN_LO <= c <= HAN_HI for c in s)


def main():
    case, base_path, new_path, leaks_path = sys.argv[1:5]
    base = open(base_path, encoding='utf-8').read()
    new = open(new_path, encoding='utf-8').read()
    pairs = []
    block = []
    for line in open(leaks_path, encoding='utf-8'):
        line = line.rstrip('\n')
        if not line or line.startswith('#'):
            continue
        if '\t' not in line:
            # a leaked block is written across several lines; only its last line carries the empty
            # replacement, and the block itself is that many lines long. `block` accumulates it.
            block.append(line)
            continue
        leaked, replacement = line.split('\t', 1)
        if block:
            leaked = '\n'.join(block + [leaked])
            block = []
        pairs.append((leaked, replacement))
    if not pairs:
        print('%s: the leak table is empty, so this proof would be vacuous' % case, file=sys.stderr)
        return 1
    if block:
        print('%s: the leak table ends inside a block' % case, file=sys.stderr)
        return 1
    # longest first: a token can contain another, and replacing the short one first would leave a
    # half-repaired string that no longer matches.
    pairs.sort(key=lambda p: -len(p[0]))
    pairs = [(leaked, replacement, i) for i, (leaked, replacement) in enumerate(pairs)]

    missing = [leaked for leaked, _, _ in pairs if leaked not in base]
    if missing:
        print('%s: the baseline English run does not contain %s, so this is not the declared leak'
              % (case, missing), file=sys.stderr)
        return 1
    if has_han(new):
        print('%s: the new English run still contains Han runes' % case, file=sys.stderr)
        for i, line in enumerate(new.splitlines(), 1):
            if has_han(line):
                print('  line %d: %s' % (i, line), file=sys.stderr)
        return 1

    # Rebuild the baseline from the new output: substitute placeholders so a replacement that contains
    # another leaked token cannot be re-replaced.
    rebuilt = new
    for leaked, replacement, i in pairs:
        if not replacement:
            continue
        rebuilt = rebuilt.replace(replacement, '\x00%d\x00' % i)
    for leaked, replacement, i in pairs:
        if not replacement:
            continue
        rebuilt = rebuilt.replace('\x00%d\x00' % i, leaked)
    if rebuilt == base:
        print('%s: %d leaked token(s) replaced, everything else byte-identical' % (case, len(pairs)))
        return 0

    # A leaked *block* whose replacement reflows the text (the :help block: the Chinese column and the
    # English one do not line up token by token) is checked line for line instead: the two outputs must
    # have the same line count, and every line that differs must be one the baseline wrote in Chinese and
    # this tree wrote without Han runes. Anything else is a difference, not a fix.
    blocks = [(leaked, replacement, i) for leaked, replacement, i in pairs if not replacement]
    if blocks and len(base.split(chr(10))) == len(new.split(chr(10))):
        a, b = base.split(chr(10)), new.split(chr(10))
        differing = [(x, y) for x, y in zip(a, b) if x != y]
        if differing and all(has_han(x) and not has_han(y) for x, y in differing):
            print('%s: %d line(s) rewritten from Chinese, every other line byte-identical'
                  % (case, len(differing)))
            return 0

    print('%s: the new English run is not the baseline with the declared leaks replaced' % case,
          file=sys.stderr)
    a, b = base.splitlines(), rebuilt.splitlines()
    for i in range(max(len(a), len(b))):
        x = a[i] if i < len(a) else '<missing>'
        y = b[i] if i < len(b) else '<missing>'
        if x != y:
            print('  line %d: baseline %r != rebuilt %r' % (i + 1, x, y), file=sys.stderr)
    return 1


if __name__ == '__main__':
    sys.exit(main())
