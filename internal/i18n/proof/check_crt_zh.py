#!/usr/bin/env python3
"""Assert one declared exception to "the Chinese rendering is byte-identical to the baseline".

The rule for this harness is that QK_LANG=zh renders exactly what it rendered before, and there is one
class of case where that cannot hold: a parse error whose value contains a control character. The old C
built `ParseError: "%s" …` around the raw value, so a tab printed as a tab; the catalog row the two
engines now share spells the value with Go's %q, which escapes it, and the interpreter has always
printed the escaped form. Keeping the baseline byte-for-byte there would make the compiled path and the
interpreter disagree about the same program, which is the defect this harness exists to catch, so the
escaped form wins and the divergence is declared here instead of being tolerated silently.

The declaration is checked rather than trusted: the case must be named in the table, the table must
record both renderings verbatim, and those renderings must be the two the run produced. An entry that
stops matching — because the text moved again, or because a case was renamed — fails.

Usage: check_crt_zh.py <case> <zh_escapes.tsv> <baseline-rendering> <new-rendering>
  zh_escapes.tsv: case<TAB>baseline rendering<TAB>new rendering, one case per line, the two renderings
  written as JSON strings so a literal tab in one of them cannot be mistaken for a column separator.
"""
import json
import sys


def read(path):
    with open(path, encoding="utf-8") as fh:
        return fh.read()


def declared(table, case):
    """The declared pair for a case as (baseline, new), or None when it is not declared."""
    for line in read(table).splitlines():
        if not line.strip() or line.startswith("#"):
            continue
        fields = line.split("\t")
        if len(fields) == 3 and fields[0] == case:
            return json.loads(fields[1]), json.loads(fields[2])
    return None


def main():
    case, table = sys.argv[1], sys.argv[2]
    base, new = (read(p) for p in sys.argv[3:5])
    pair = declared(table, case)
    if pair is None:
        print("  %s: differs from the baseline and is not declared in %s" % (case, table),
              file=sys.stderr)
        return 1
    want_base, want_new = pair
    bad = 0
    if base != want_base:
        print("  %s: the baseline rendering is %r, the table declares %r"
              % (case, base.strip()[:160], want_base.strip()[:160]), file=sys.stderr)
        bad += 1
    if new != want_new:
        print("  %s: this tree renders %r, the table declares %r"
              % (case, new.strip()[:160], want_new.strip()[:160]), file=sys.stderr)
        bad += 1
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
