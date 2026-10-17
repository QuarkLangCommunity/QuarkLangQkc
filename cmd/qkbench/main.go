// Command qkbench renders the measured tide report the README and the standard embed.
//
// The report is generated, committed and checked: `go generate ./internal/benchgen` writes
// docs/tide-benchmark.md, and `-check` regenerates it in memory and fails when the committed file
// differs, which is what TestGeneratedTideBenchmarkIsCurrent drives in the ordinary test run.
//
// Usage:
//
//	go generate ./internal/benchgen
//	go run ./cmd/qkbench -check
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/benchgen"
)

// main parses the flags and runs the generator or its drift check.
func main() {
	check := flag.Bool("check", false, "fail when the committed report is out of date instead of writing it")
	root := flag.String("root", ".", "checkout root the default paths are resolved against")
	tsv := flag.String("tsv", "", "measured record (default "+benchgen.DefaultTSV+")")
	template := flag.String("template", "", "authored prose template (default "+benchgen.DefaultTemplate+")")
	out := flag.String("out", "", "generated report (default "+benchgen.DefaultOut+")")
	flag.Parse()
	err := benchgen.Run(benchgen.Options{Root: *root, TSV: *tsv, Template: *template, Out: *out, Check: *check})
	if err != nil {
		fmt.Fprintln(os.Stderr, "qkbench:", err)
		os.Exit(1)
	}
}
