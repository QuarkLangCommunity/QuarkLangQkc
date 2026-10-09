package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// version is the release version: injected at build time with -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-V") {
		fmt.Println("quark " + version)
		return
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: quark <file.qk> [args...] [--bp file:line,...]")
		os.Exit(2)
	}
	// the file may appear anywhere (flags may precede it): the first non-flag argument
	file := ""
	rest := []string{}
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		if a == "--bp" && i+1 < len(os.Args) {
			rest = append(rest, a, os.Args[i+1])
			i++
			continue
		}
		if file == "" && !strings.HasPrefix(a, "-") {
			file = a
			continue
		}
		rest = append(rest, a)
	}
	if file == "" {
		fmt.Fprintln(os.Stderr, "usage: quark <file.qk> [args...] [--bp file:line,...]")
		os.Exit(2)
	}
	src, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	prog, err := lang.CompileWithImports(string(src), file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	err = runMain(prog, file, rest, os.Stdin, os.Stdout)
	if err != nil {
		lang.ReportError(err, os.Stderr)
		os.Exit(1)
	}
}

// runMain runs in normal or debug mode (--bp "file:line,..." breakpoints).
func runMain(prog *lang.Program, file string, args []string, stdin io.Reader, stdout io.Writer) error {
	bps := []string{}
	for i, a := range args {
		if a == "--bp" && i+1 < len(args) {
			bps = strings.Split(args[i+1], ",")
			args = append(args[:i], args[i+2:]...)
			break
		}
	}
	if len(bps) > 0 {
		return lang.RunDebug(prog, file, args, stdin, stdout, bps)
	}
	return lang.Run(prog, file, args, stdin, stdout)
}
