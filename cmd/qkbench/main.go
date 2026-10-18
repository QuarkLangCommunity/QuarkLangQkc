// Command qkbench renders the measured benchmark reports the README and the standard embed.
//
// The reports are generated, committed and checked: `go generate ./internal/benchgen` writes
// docs/tide-benchmark.md and docs/performance-claims.md, and `-check` regenerates them in memory and
// fails when a committed file differs, which is what the drift tests drive in the ordinary test run.
//
// Usage:
//
//	go generate ./internal/benchgen
//	go run ./cmd/qkbench -check
//	go run ./cmd/qkbench -check -suite claims
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/benchgen"
)

// main parses the flags and runs the generator or its drift check.
func main() {
	check := flag.Bool("check", false, "fail when a committed report is out of date instead of writing it")
	root := flag.String("root", ".", "checkout root the default paths are resolved against")
	suite := flag.String("suite", "all", "report suite: tide, claims or all")
	tsv := flag.String("tsv", "", "measured tide record (default "+benchgen.DefaultTSV+")")
	template := flag.String("template", "", "authored tide template (default "+benchgen.DefaultTemplate+")")
	out := flag.String("out", "", "generated tide report (default "+benchgen.DefaultOut+")")
	flag.Parse()
	var err error
	switch *suite {
	case "tide":
		err = benchgen.Run(benchgen.Options{Root: *root, TSV: *tsv, Template: *template, Out: *out, Check: *check})
	case "claims":
		if *tsv != "" || *template != "" || *out != "" {
			err = fmt.Errorf("-tsv, -template and -out apply to the tide suite")
		} else {
			err = benchgen.RunSuite(*root, benchgen.ClaimsSuite, *check)
		}
	case "all":
		if *tsv != "" || *template != "" || *out != "" {
			err = fmt.Errorf("-tsv, -template and -out apply to the tide suite; select -suite tide to use them")
		} else {
			for _, s := range benchgen.Suites() {
				if err = benchgen.RunSuite(*root, s, *check); err != nil {
					break
				}
			}
		}
	default:
		err = fmt.Errorf("unknown -suite %q: use tide, claims or all", *suite)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "qkbench:", err)
		os.Exit(1)
	}
}
