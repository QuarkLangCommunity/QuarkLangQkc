#!/usr/bin/env python3
"""Assert one declared P3 ZH fix: the leak a Chinese reader was getting, and the translation that replaces it.

The struct-field pass is the one part of this migration whose proof cannot be byte-identity. A message
written into a struct field never reached a localizer, so `QK_LANG=zh` rendered the English literal
while the catalog already carried its Chinese. Wiring the site changes the Chinese output — that *is*
the fix — and this script is what keeps "it changed" from being a blank cheque. The claim:

  1. the baseline's QK_LANG=zh run was byte-identical to its QK_LANG=en run: the site had no language
     in it at all, so a Chinese reader got English;
  2. this tree's QK_LANG=zh run is the baseline's with that message replaced by the Chinese translation
     the catalog already shipped — same arguments, same order, same verbs, and nothing else moved: not
     the error kind, not the line, not the column, not the exit code;
  3. this tree's QK_LANG=en run is the baseline's, which the caller asserts.

The binding runs on the *Chinese* side, not the English one, and that is deliberate. Both templates have
verbs and the rendering has the arguments substituted, so `%q` and `%s` have to be bound from a
rendering: the Chinese template is the one whose verbs sit between literals that are not also part of
the arguments, so walking it over the prototype fixes the arguments, and the baseline must then be the
same prefix, the English template with those arguments, and the same suffix. Binding the English side
instead splits `no member %q on %s` as `%q` = `"` and `%s` = `int at line 1`, which renders a message
nobody ever saw — the first version of this script did exactly that and reported five false failures.

Usage: check_zh_fix.py <case> <fixes.tsv> <baseline-zh> <baseline-en> <prototype-zh>
"""
import re
import sys

VERB = re.compile(r"%[-+# 0-9.]*([a-zA-Z%])")


def split_template(tpl):
    """The template as alternating literal and verb parts, in order."""
    parts, last = [], 0
    for m in VERB.finditer(tpl):
        parts.append(("lit", tpl[last:m.start()]))
        parts.append(("verb", m.group(1)))
        last = m.end()
    parts.append(("lit", tpl[last:]))
    return parts


def bind(tpl, text):
    """The arguments that make `text` a rendering of `tpl`, or None when it is not one.

    Each verb runs up to the literal that follows it; a verb with no literal after it takes the rest,
    which is what fmt would do. Two adjacent verbs cannot be split and are reported rather than guessed.
    """
    parts = split_template(tpl)
    if parts[0][0] != "lit":
        raise ValueError("template %r starts with a verb, so its rendering cannot be located" % tpl)
    start = text.find(parts[0][1])
    if start < 0:
        return None
    pos, args = start + len(parts[0][1]), []
    for i in range(1, len(parts)):
        kind, val = parts[i]
        if kind == "lit":
            if not text.startswith(val, pos):
                return None
            pos += len(val)
            continue
        nxt = parts[i + 1][1] if i + 1 < len(parts) else ""
        if nxt == "" and i + 1 < len(parts) - 1:
            raise ValueError("template %r has two adjacent verbs, which this checker cannot split" % tpl)
        if nxt == "":
            args.append(text[pos:])
            pos = len(text)
            continue
        j = text.find(nxt, pos)
        if j < 0:
            return None
        args.append(text[pos:j])
        pos = j
    return start, args, pos


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
    case, fixes_path, base_zh_path, base_en_path, proto_zh_path = sys.argv[1:6]
    en = zh = None
    with open(fixes_path, encoding="utf-8") as fh:
        for line in fh:
            if line.strip() == "" or line.startswith("#"):
                continue
            fields = line.rstrip("\n").split("\t")
            if fields[0] == case:
                en, zh = fields[1], fields[2]
                break
    if en is None:
        print("no ZH-fix row for %s" % case, file=sys.stderr)
        return 1

    base_zh = open(base_zh_path, encoding="utf-8").read()
    base_en = open(base_en_path, encoding="utf-8").read()
    proto_zh = open(proto_zh_path, encoding="utf-8").read()

    # 1. the leak: the baseline had no language in this message, so both locales rendered the same text.
    if base_zh != base_en:
        print("the baseline's zh run is not its en run, so this case is not the leak it claims", file=sys.stderr)
        print("  zh: %r" % base_zh, file=sys.stderr)
        print("  en: %r" % base_en, file=sys.stderr)
        return 1

    bound = bind(zh, proto_zh)
    if bound is None:
        print("this tree's zh run does not render the Chinese template %r" % zh, file=sys.stderr)
        print("  prototype: %r" % proto_zh, file=sys.stderr)
        return 1
    start, args, end = bound
    prefix = proto_zh[:start]
    suffix = proto_zh[end:]
    if prefix + render(zh, args) + suffix != proto_zh:
        print("this tree's zh run is not the Chinese template with its arguments", file=sys.stderr)
        print("  prototype: %r" % proto_zh, file=sys.stderr)
        return 1

    want = prefix + render(en, args) + suffix
    if want != base_zh:
        print("the baseline is not this tree's run with the message swapped back to English", file=sys.stderr)
        print("  want: %r" % want, file=sys.stderr)
        print("  got:  %r" % base_zh, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
