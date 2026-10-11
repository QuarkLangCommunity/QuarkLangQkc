#!/usr/bin/env python3
"""P3's EN-leak check: a case where the migration fixes a message that never reached a boundary.

`check_en_fix.py` covers the case where a message had catalog entries on both sides and the old code
picked the Chinese one for an English reader. This covers the worse case the docgen domain turned up:
a literal that never reached an i18n boundary at all, so EN and ZH rendered the same bytes and no
ratchet could see it — `QK_LANG=en qkc` printed a Chinese usage text.

The claim is not "EN may differ". It is three checkable statements about the two binaries:

  1. the baseline rendered the same bytes under `en` and `zh` — the leak, measured on the old binary;
  2. the baseline's `en` run carried Han runes and this tree's does not;
  3. the two runs are the same text apart from the leaked and repaired lines: every line that differs
     must be one the baseline wrote in Chinese and this tree wrote without Han runes, and the number,
     order and non-Han content of the lines must be unchanged.

Anything else is a difference, not a fix, and fails.

Usage: check_en_leak.py <case> <baseline-en> <baseline-zh> <new-en>
"""
import json
import re
import sys

HAN = re.compile(r'[\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff\U00020000-\U0002ffff]')


def main():
    name, base_en_path, base_zh_path, new_en_path = sys.argv[1:5]
    base_en = open(base_en_path, encoding='utf-8').read()
    base_zh = open(base_zh_path, encoding='utf-8').read()
    new_en = open(new_en_path, encoding='utf-8').read()

    # 1) the leak: on the baseline the English run was the Chinese run.
    if base_en != base_zh:
        print('%s: the baseline did not leak (en and zh already differ), so this is not a declared fix'
              % name, file=sys.stderr)
        return 1
    if not HAN.search(base_en):
        print('%s: the baseline English output carries no Han rune, so there is nothing to fix' % name,
              file=sys.stderr)
        return 1

    # 2) the fix: no Han rune reaches an English reader now.
    if HAN.search(new_en):
        print('%s: the migration still renders Han runes under QK_LANG=en' % name, file=sys.stderr)
        for i, line in enumerate(new_en.splitlines(), 1):
            if HAN.search(line):
                print('  line %d: %s' % (i, line), file=sys.stderr)
        return 1

    # 3) same shape, and every difference is a leaked line now written in English.
    a, b = base_en.splitlines(), new_en.splitlines()
    if len(a) != len(b):
        print('%s: the output changed shape (%d lines before, %d after)' % (name, len(a), len(b)),
              file=sys.stderr)
        return 1
    fixed = []
    for i, (x, y) in enumerate(zip(a, b), 1):
        if x == y:
            continue
        if not HAN.search(x):
            print('%s: line %d changed but the baseline line was not Chinese: %r -> %r'
                  % (name, i, x, y), file=sys.stderr)
            return 1
        fixed.append(i)
    if not fixed:
        print('%s: nothing changed, so nothing was fixed' % name, file=sys.stderr)
        return 1
    print(json.dumps({'case': name, 'leaked_lines': len(fixed), 'line_numbers': fixed[:8]}))
    return 0


if __name__ == '__main__':
    sys.exit(main())
