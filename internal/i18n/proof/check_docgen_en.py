#!/usr/bin/env python3
"""Assert the docgen domain's English document: a rendered document, checked by outcome rather than by
byte-equality against the baseline.

A document is not one message, so the per-message fix table in byte_identical.sh does not fit it. Two
admissible outcomes, and neither is a blank cheque:

  * the baseline's English document carried Han runes — the domain had nine Chinese labels hardcoded in
    the renderer (kindLabel's 函数 / 结构体 / 接口 / … and the section titles), never handed to a
    localizer, so an English reader got a Chinese document and no gate could see it. Then this tree's
    document must carry none, and must still be a document: the same non-blank line count, the same table
    skeleton, the same headings.
  * the baseline's English document was already Han-free. That is the case for a domain whose leak has
    already merged, because the baseline revision is origin/main and origin/main then carries the fix.
    Then this tree's document must be byte-identical to the baseline's: a migration must not disturb a
    document that was already right, and this is the half that catches it.

Usage: check_docgen_en.py <baseline-rendering> <prototype-rendering>
"""
import re
import sys

HAN = re.compile(r"[\u4e00-\u9fff]")


def main():
    base = open(sys.argv[1], encoding="utf-8").read()
    proto = open(sys.argv[2], encoding="utf-8").read()

    if not HAN.search(base):
        if base != proto:
            print("the baseline document was already Han-free, so this tree must reproduce it exactly",
                  file=sys.stderr)
            a, b = base.split("\n"), proto.split("\n")
            for i in range(max(len(a), len(b))):
                x = a[i] if i < len(a) else "<missing>"
                y = b[i] if i < len(b) else "<missing>"
                if x != y:
                    print("  line %d: %r -> %r" % (i + 1, x, y), file=sys.stderr)
            return 1
        return 0

    bad = HAN.findall(proto)
    if bad:
        print("the English document still contains %d Han rune(s): %s" % (len(bad), "".join(sorted(set(bad)))), file=sys.stderr)
        return 1

    want_lines = [l for l in base.split("\n") if l.strip() != ""]
    got_lines = [l for l in proto.split("\n") if l.strip() != ""]
    # The document structure must survive: the same path line, the same table skeleton, the same headings
    # count. Only the labels were allowed to change.
    if len(want_lines) != len(got_lines):
        print("the document changed shape: %d non-blank lines became %d" % (len(want_lines), len(got_lines)), file=sys.stderr)
        return 1
    for i, (w, g) in enumerate(zip(want_lines, got_lines)):
        if w.count("|") != g.count("|"):
            print("line %d changed its table shape: %r -> %r" % (i + 1, w, g), file=sys.stderr)
            return 1
        if w.lstrip().startswith("#") and not g.lstrip().startswith("#"):
            print("line %d stopped being a heading: %r -> %r" % (i + 1, w, g), file=sys.stderr)
            return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
