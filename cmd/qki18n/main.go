// Command qki18n generates internal/i18n's message catalog from the authored tables under
// internal/i18n/source.
//
// The catalog is generated at build time and committed, so `go build ./...` never needs this command
// to run. `-check` is what the ordinary `go test ./... -count=1` gate uses: it regenerates in memory
// and fails when the committed file differs, which is how a template, an ID or a translation that was
// edited without regenerating becomes a test failure rather than a runtime fallback.
//
// It also reads the message table of the pinned qkparser module, so an ID that the syntax layer ships
// — or a text it changed — is caught here, in this repository's build, before it can ship untranslated.
//
// `-doc` renders the review document for the untranslated backlog, which is not committed: it carries
// the Chinese values as data, so it is written outside the checkout, on request. `-doc cache` puts it
// at the per-user default and prints where that is, and the generator refuses a path inside the
// repository. `-check -doc <path>` compares that file with a fresh rendering, and `-check` alone still
// renders the document in memory so the gate covers the code path a reviewer depends on.
//
// Usage:
//
//	go generate ./internal/i18n
//	go run ./cmd/qki18n -check
//	go run ./cmd/qki18n -doc cache
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18ngen"
)

// parserModule is the syntax layer whose messages this repository ships. Its authored table is read
// at the revision go.mod pins, so a new message there is a generation failure here.
const parserModule = "github.com/QuarkLangCommunity/QuarkLangQkparser"

// pinnedModuleDir asks the Go toolchain where the pinned revision of the module is on disk. It reads
// the module graph only: the revision is already required by go.mod, so this needs no network.
func pinnedModuleDir() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", parserModule).Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return "", fmt.Errorf("locating %s: %s", parserModule, strings.TrimSpace(string(exit.Stderr)))
		}
		return "", fmt.Errorf("locating %s: %w", parserModule, err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("%s resolved to an empty directory (is it required by go.mod?)", parserModule)
	}
	return dir, nil
}

func main() {
	check := flag.Bool("check", false, "fail when the committed catalog (or document) is out of date instead of writing it")
	source := flag.String("source", filepath.Join("internal", "i18n", "source"), "authored message tables")
	out := flag.String("out", filepath.Join("internal", "i18n", "msg_gen.go"), "generated catalog")
	doc := flag.String("doc", "", "write the backlog review document to this path, outside the repository; `cache` means the per-user default (empty: no document)")
	cout := flag.String("c-out", "", "generated C message catalog for the embedded runtime, e.g. compiler/internal/rtcat/msg_c.h (empty: do not generate one)")
	root := flag.String("root", "", "checkout the call sites and the alternate-Chinese scan are measured in (empty: the working directory)")
	module := flag.String("module", "", "pinned qkparser source directory (default: resolved from go.mod)")
	flag.Parse()

	moduleDir := *module
	if moduleDir == "" {
		dir, err := pinnedModuleDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "qki18n:", err)
			os.Exit(1)
		}
		moduleDir = dir
	}

	scanRoot := *root
	if scanRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "qki18n:", err)
			os.Exit(1)
		}
		scanRoot = wd
	}

	// The review document is not committed, so a bare `-check` has no file to compare it with. It is
	// still rendered, in memory, so the gate covers the code path a reviewer depends on: a table the
	// document cannot be rendered from fails the gate even though nothing is written. `-doc` names a
	// file outside the repository to write, or with `-check` to compare against a fresh rendering.
	docFile, err := docTarget(*doc, scanRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "qki18n:", err)
		os.Exit(1)
	}

	problems := i18ngen.Run(i18ngen.Options{
		SourceDir:    *source,
		OutFile:      *out,
		DocFile:      docFile,
		COutFile:     *cout,
		Root:         scanRoot,
		ModuleDir:    moduleDir,
		ModuleSource: i18ngen.DefaultModuleSource,
		Check:        *check,
	})
	if len(problems) > 0 {
		fmt.Fprintf(os.Stderr, "qki18n: %d problem(s):\n", len(problems))
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "  "+p)
		}
		os.Exit(1)
	}
	verb := "wrote"
	if *check {
		verb = "checked"
	}
	fmt.Printf("qki18n: %s %s\n", verb, *out)
	if *cout != "" {
		fmt.Printf("qki18n: %s %s\n", verb, *cout)
	}
	if docFile != "" {
		fmt.Printf("qki18n: %s %s\n", verb, docFile)
	}
}

// docTarget resolves the -doc flag into the path the document is written to or compared with, or ""
// when this run should not touch the document at all. `cache` names the per-user default; a path
// inside the checkout is refused, because the document carries the Chinese values as data and this
// repository keeps no such document. A bare `-check` returns "", and renders the document in memory
// instead of on disk, so the gate covers the rendering without writing anything.
func docTarget(value, checkout string) (string, error) {
	switch {
	case value == i18ngen.BacklogRequest:
		value = i18ngen.BacklogPath()
	case value == "":
		return "", nil
	}
	if i18ngen.InRepository(checkout, value) {
		return "", fmt.Errorf("-doc %s is inside the repository: the review document is generated outside the checkout, because it carries the Chinese values as data", value)
	}
	return value, nil
}
