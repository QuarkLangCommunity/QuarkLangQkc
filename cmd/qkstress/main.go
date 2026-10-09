// qkstress is the QuarkLang stress and robustness harness: `gen` materialises the extreme corpus and
// `run` executes every case against the interpreter and the compiler under a hard time and memory bound.
package main

import (
	"flag"
	"fmt"
	"os"
)

// main dispatches the subcommands and turns a returned error into a non-zero exit status.
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	var err error
	switch args[0] {
	case "gen":
		err = cmdGen(args[1:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "qkstress: unknown command %q\n", args[0])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "qkstress:", err)
		os.Exit(1)
	}
}

// usage prints the documented one-liners for the commands.
func usage() {
	fmt.Fprint(os.Stderr, `usage: qkstress <command> [flags]

  gen   write the extreme corpus into a gitignored directory

  go run ./cmd/qkstress gen -out .stress/cases

flags:
  gen:  -out .stress/cases   output directory (wiped and rewritten)

Run from the repository root: every default path is relative to it.
`)
}

// newFlagSet builds a flag set that prints the tool's own usage on a bad invocation.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = usage
	return fs
}
