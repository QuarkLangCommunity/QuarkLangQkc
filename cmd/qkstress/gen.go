package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang/stressgen"
)

// cmdGen materialises the whole stress corpus as <out>/<case>.kq and prints what it wrote.
func cmdGen(args []string) error {
	fs := newFlagSet("gen")
	out := fs.String("out", ".stress/cases", "output directory (wiped and rewritten)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("-out must not be empty")
	}
	if err := wipeDir(*out); err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "case\tcategory\tbytes")
	total := 0
	for _, c := range stressgen.All() {
		path := filepath.Join(*out, c.Name+".kq")
		if err := os.WriteFile(path, c.Source, 0o644); err != nil {
			return err
		}
		total += len(c.Source)
		fmt.Fprintf(tw, "%s\t%s\t%d\n", c.Name, c.Category, len(c.Source))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Printf("%d cases, %d bytes in %s\n", len(stressgen.All()), total, *out)
	return nil
}

// wipeDir removes a generated directory after refusing the paths whose removal would destroy something
// a human made: the filesystem root, the current directory, or the home directory.
func wipeDir(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if abs == string(filepath.Separator) {
		return fmt.Errorf("refusing to clean the filesystem root %s", abs)
	}
	if wd, werr := os.Getwd(); werr == nil && abs == wd {
		return fmt.Errorf("refusing to clean the working directory %s", abs)
	}
	if home, herr := os.UserHomeDir(); herr == nil && abs == home {
		return fmt.Errorf("refusing to clean the home directory %s", abs)
	}
	return os.RemoveAll(abs)
}
