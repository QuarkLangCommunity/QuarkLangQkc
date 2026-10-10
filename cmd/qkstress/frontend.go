package main

import (
	"fmt"
	"os"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// cmdFrontend is the run command's in-process acceptance probe for the interpreter: exit 0 means the
// interpreter would attempt to run the file, exit 1 means it would refuse before running anything, and a
// panic leaves Go's own exit status 2 with the crash on stderr. The probe exists so the runner can tell
// "refused before running" from "failed while running" without pattern-matching diagnostic wording: it
// compiles with the shared front end and applies the same entry-point check the interpreter applies when
// it starts, because a file with no main function never reaches the program.
func cmdFrontend(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: qkstress frontend <file.kq>")
		os.Exit(2)
	}
	src, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "qkstress frontend:", err)
		os.Exit(2)
	}
	prog, err := lang.CompileWithImports(string(src), args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if _, ok := prog.FnIndex["main"]; !ok {
		fmt.Fprintln(os.Stderr, "error: CompileError: no main function found (expected: fn main(io IOStream, ...))")
		os.Exit(1)
	}
}
