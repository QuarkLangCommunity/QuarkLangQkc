// Command qkstatus renders the implementation-status report the standard's §11 cites.
//
// The report is generated, committed and checked: `go generate ./internal/statusgen` writes
// docs/implementation-status.md, and `-check` regenerates in memory and fails when the committed file
// differs, which is what TestGeneratedStatusIsCurrent drives in the ordinary test run.
//
// Usage:
//
//	go generate ./internal/statusgen
//	go run ./cmd/qkstatus -check
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/statusgen"
)

// main parses the flags and runs the generator or its drift check.
func main() {
	check := flag.Bool("check", false, "fail when the committed report is out of date instead of writing it")
	root := flag.String("root", ".", "checkout root the default paths are resolved against")
	claims := flag.String("claims", "", "claims table (default "+statusgen.DefaultClaims+")")
	template := flag.String("template", "", "authored prose template (default "+statusgen.DefaultTemplate+")")
	out := flag.String("out", "", "generated report (default "+statusgen.DefaultOut+")")
	flag.Parse()
	err := statusgen.Run(statusgen.Options{Root: *root, Claims: *claims, Template: *template, Out: *out, Check: *check})
	if err != nil {
		fmt.Fprintln(os.Stderr, "qkstatus:", err)
		os.Exit(1)
	}
}
