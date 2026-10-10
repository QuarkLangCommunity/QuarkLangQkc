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
	case "run":
		err = cmdRun(args[1:])
	case "frontend":
		cmdFrontend(args[1:]) // exits on its own: the exit status is the acceptance signal
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
  run   execute every case against both engines and print the measured verdicts

  go run ./cmd/qkstress gen -out stress-out/cases
  go run ./cmd/qkstress run -cases stress-out/cases

flags:
  gen:  -out stress-out/cases   output directory (wiped and rewritten)
  run:  -cases stress-out/cases -interp stress-out/bin/quark -compiler stress-out/bin/qkc
        -work stress-out -timeout 10s -mem 2048 -known stress/known-failures.txt
        -json stress-out/results.json -report stress-out/report.md -filter <regexp>

Run from the repository root: every default path is relative to it.
`)
}

// newFlagSet builds a flag set that prints the tool's own usage on a bad invocation.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = usage
	return fs
}
