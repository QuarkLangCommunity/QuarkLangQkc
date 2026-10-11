#!/usr/bin/env python3
"""Assert one declared P3 EN fix, against the two renderings the proof just produced.

The claim under test is not "these differ" but "they differ in exactly one way, and it is the one way
step 3 set out to change":

  1. the baseline's English run carried the Chinese template. That text is what a Chinese reader gets
     as well — the script's QK_LANG=zh pass is what proves the zh rendering is the same before and
     after, so the English run was handing an English reader the Chinese message;
  2. this tree's English run is the baseline's English run with that message replaced by the English
     text that already sat in internal/i18n/table_lexer.go, never selected;
  3. nothing else moved: not the error kind, not the line, not the column, not the exit code.

Both sides are templates with fmt verbs, and the renderings have the arguments substituted, so the
comparison runs on the templates: the Chinese one matches the baseline and its capture groups are the
arguments, those arguments are substituted into the English one, and the answer has to be the
prototype's rendering, byte for byte. A verb count that disagreed between the two languages would fail
here, which is the same thing TestVerbSetsMatch checks on the table.

Usage: check_en_fix.py <case> <fixes.tsv> <baseline-rendering> <prototype-rendering>
"""
import re
import sys

VERB = re.compile(r"%[-+# 0-9.]*([a-zA-Z%])")


def template_regex(tpl):
    """The template as a regex, one capture group per verb, in order."""
    out, last, n = [], 0, 0
    for m in VERB.finditer(tpl):
        out.append(re.escape(tpl[last:m.start()]))
        if m.group(1) == "%":
            out.append("%")
        else:
            out.append("(.+?)")
            n += 1
        last = m.end()
    out.append(re.escape(tpl[last:]))
    return re.compile("".join(out)), n


def render(tpl, args):
    """The template with the given arguments substituted, the way fmt would."""
    out, last, i = [], 0, 0
    for m in VERB.finditer(tpl):
        out.append(tpl[last:m.start()])
        if m.group(1) == "%":
            out.append("%")
        else:
            out.append(args[i])
            i += 1
        last = m.end()
    out.append(tpl[last:])
    if i != len(args):
        raise ValueError("%d arguments for %d verbs" % (len(args), i))
    return "".join(out)


def main():
    case, fixes_path, base_path, proto_path = sys.argv[1:5]
    zh = en = None
    with open(fixes_path, encoding="utf-8") as fh:
        for line in fh:
            if line.strip() == "" or line.startswith("#"):
                continue
            fields = line.rstrip("\n").split("\t")
            if fields[0] == case:
                zh, en = fields[1], fields[2]
                break
    if zh is None:
        print("no fix row for %s" % case, file=sys.stderr)
        return 1

    base = open(base_path, encoding="utf-8").read()
    proto = open(proto_path, encoding="utf-8").read()

    rx, nverbs = template_regex(zh)
    _, nverbsen = template_regex(en)
    if nverbs != nverbsen:
        print("the two languages bind a different number of verbs: %d vs %d" % (nverbs, nverbsen), file=sys.stderr)
        return 1

    m = rx.search(base)
    if m is None:
        print("the baseline's English run does not carry the Chinese template %r" % zh, file=sys.stderr)
        print("  baseline: %r" % base, file=sys.stderr)
        return 1

    expected = render(en, m.groups())
    want = base[:m.start()] + expected + base[m.end():]
    if want != proto:
        print("the fix changed more than the message", file=sys.stderr)
        print("  want: %r" % want, file=sys.stderr)
        print("  got:  %r" % proto, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
