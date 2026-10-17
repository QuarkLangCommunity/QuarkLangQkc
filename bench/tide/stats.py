#!/usr/bin/env python3
"""Turn one workload's run times into the canonical measurement row.

The values arrive on stdin, one integer microsecond value per line. The output is one tab-separated
row in the schema spec/bench/tide-template.md documents. The script is deliberately small: the
statistics are the point, and the harness (run.sh) owns the build and the commands.
"""

import argparse
import statistics
import sys


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--key", required=True)
    parser.add_argument("--label", required=True)
    parser.add_argument("--language", required=True)
    parser.add_argument("--rounds", required=True, type=int)
    parser.add_argument("--runs", required=True, type=int)
    parser.add_argument("--unit", default="us")
    parser.add_argument("--checksum", default="-")
    parser.add_argument("--note", required=True)
    args = parser.parse_args()

    values = [int(line) for line in sys.stdin.read().split()]
    if len(values) != args.runs:
        print(f"stats.py: expected {args.runs} values, got {len(values)}", file=sys.stderr)
        return 2
    if any(v < 0 for v in values):
        print("stats.py: negative duration", file=sys.stderr)
        return 2
    median = statistics.median(values)
    spread = 0.0
    if median > 0 and len(values) > 1:
        spread = (max(values) - min(values)) / median * 100.0
    stdev = statistics.stdev(values) if len(values) > 1 else 0.0
    for field in (args.label, args.note, args.language, args.key):
        if "\t" in field or "\n" in field:
            print("stats.py: a field contains a tab or newline", file=sys.stderr)
            return 2
    print(
        "\t".join(
            [
                "timing",
                args.key,
                args.label,
                args.language,
                str(args.rounds),
                str(args.runs),
                f"{median:.0f}",
                args.unit,
                str(min(values)),
                str(max(values)),
                f"{stdev:.0f}",
                f"{spread:.2f}",
                str(args.checksum),
                args.note,
            ]
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
