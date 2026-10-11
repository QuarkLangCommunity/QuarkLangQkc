#!/usr/bin/env python3
"""Assert one compiled program's English rendering: the leak it replaced, and the two engines agreeing.

The compiled path and the interpreter must render the same message from the same catalog, so the
English a compiled program prints is not free to be anything — it has to be what `quark` prints for the
same program, which is the rendering the tables author. The claims, in the order they can fail:

  1. the baseline's English run carried Han runes. This is what makes the fix a measurement: a compiler
     that printed nothing, or a message that was already English, would pass the rest of this vacuously.
  2. this tree's English run carries no Han rune.
  3. it equals the interpreter's English run, byte for byte. Two engines, one catalog: a difference here
     is a second catalog growing back.
  4. the error kind and the line number did not move — the same prefix and the same `at line N` suffix
     the Chinese rendering carries, so the fix rewrote the sentence and nothing else.

The negative control — the program that does not fail — reaches none of these, and says so rather than
being exempted silently: its English and Chinese runs must be identical to each other and to the
interpreter's, so a harness that reported success for a compiler that printed nothing stays visible.

Usage: check_crt_en.py <case> <baseline-en> <prototype-en> <interpreter-en> <prototype-zh>
"""
import re
import sys

HAN = re.compile(r"[\u4e00-\u9fff]")
# The rendering shape both engines use: "error: <kind>: <message> at line <n>", with the kind and the
# line the parts that must not move.
SHAPE = re.compile(r"^(error: [A-Za-z]+:).*?( at line \d+)$", re.S)


def read(path):
    with open(path, encoding="utf-8") as fh:
        return fh.read()


def shape_of(text):
    """The error kind and the line number, which the migration must not touch."""
    m = SHAPE.match(text.rstrip("\n"))
    if not m:
        return None
    return m.group(1), m.group(2)


def main():
    case = sys.argv[1]
    base_en, proto_en, interp_en, proto_zh = (read(p) for p in sys.argv[2:6])
    bad = 0

    def report(msg):
        nonlocal bad
        print("  %s: %s" % (case, msg), file=sys.stderr)
        bad += 1

    if HAN.search(base_en) is None:
        # Nothing to fix here: the case must be the control, and a control that renders differently in
        # the two languages (or from the interpreter) is a case the corpus is missing, not a pass.
        if proto_en != proto_zh:
            report("the baseline printed no Han rune and the renderings differ, so this case is neither "
                   "a leaked message nor a control:\n    en: %r\n    zh: %r"
                   % (proto_en.strip()[:160], proto_zh.strip()[:160]))
        elif proto_en != interp_en:
            report("the control disagrees with the interpreter:\n    compiled: %r\n    interpreter: %r"
                   % (proto_en.strip()[:160], interp_en.strip()[:160]))
        return 1 if bad else 0

    # From here the baseline leaked: the case exercises one of the runtime's messages.
    if HAN.search(proto_en):
        report("the English run still carries Han runes: %r" % proto_en.strip()[:120])
    if proto_en != interp_en:
        report("the compiled path and the interpreter disagree under en:\n"
               "    compiled:     %r\n    interpreter:  %r"
               % (proto_en.strip()[:160], interp_en.strip()[:160]))
    if proto_en == proto_zh:
        report("the English and Chinese runs are identical, so nothing was translated")
    if shape_of(proto_en) != shape_of(proto_zh):
        report("the error kind or the line number moved: %r vs %r"
               % (shape_of(proto_en), shape_of(proto_zh)))
    if proto_en == base_en:
        report("the English run is unchanged from the baseline, which leaked")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
