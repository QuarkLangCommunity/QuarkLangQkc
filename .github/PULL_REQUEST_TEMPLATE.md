## What changed

<!-- One to three lines; link issues with "Closes #123" -->

## Why

<!-- Background/motivation; performance work must include before/after numbers -->

## Checklist

- [ ] `go test ./...` passes
- [ ] `(cd compiler && go test ./...)` passes
- [ ] `(cd compiler && ./testdata/compare.sh)` prints "全部一致" (dual-path parity)
- [ ] Syntax/semantics changes: updated `SYNTAX.md` and the parity cases in `compiler/testdata`
- [ ] Static-analysis changes: updated the labelled cases in `internal/lang/testdata/lintbench/` (the FP/FN gate runs them)
- [ ] Release/CI changes: ran the corresponding script locally as a smoke test

## Notes

<!-- Known limitations, follow-ups, anything a reviewer should pay special attention to -->
