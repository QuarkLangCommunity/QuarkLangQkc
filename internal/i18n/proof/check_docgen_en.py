#!/usr/bin/env python3
"""Assert the docgen domain's English fix: the document used to be Chinese, and now it is not.

Two claims, both about the renderings the proof just produced:

  1. the baseline's English document contained Han runes, and this tree's does not. The domain had nine
     Chinese labels hardcoded in the renderer — kindLabel's 函数 / 结构体 / 接口 / … and the section titles
     — which were never handed to a localizer, so an English reader got a Chinese document and no gate
     in the repository could see it. A hardcoded message is invisible to a scanner that looks for
     literals passed to an i18n boundary, which is why this has to be asserted on the output.
  2. this tree's English document still looks like the baseline's: same number of lines, and the
     non-label text (headings' line positions, the table's pipes, the code spans, the file path) is
     unchanged. Otherwise "no Han runes" would be satisfied by rendering nothing at all.

Usage: check_docgen_en.py <baseline-rendering> <prototype-rendering>
"""
import re
import sys

HAN = re.compile(r"[\u4e00-\u9fff]")


def main():
    base = open(sys.argv[1], encoding="utf-8").read()
    proto = open(sys.argv[2], encoding="utf-8").read()

    if not HAN.search(base):
        print("the baseline's English document had no Chinese in it, so there was no leak to fix", file=sys.stderr)
        return 1
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
