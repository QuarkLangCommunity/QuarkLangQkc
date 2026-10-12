package i18n

// The authored tables record where each message is raised, and the generated artefacts repeat those
// sites: all of them are committed and gate-checked, so what a row cites has to be true on a machine
// that never saw the checkout the row was written in. The two tests below are that claim kept as a
// gate, and the generator refuses an absolute site while parsing (checkSiteContext in
// internal/i18ngen), so a table cannot carry one back in.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// machineMarkers are the substrings a committed artefact must not contain.
var machineMarkers = []string{"/home/", "/Users/", "/root/", "/tmp/", "/private/", "file://"}

// isMachinePath reports whether a table cell is an absolute POSIX path, a Windows drive or UNC path, or a
// file URI.
func isMachinePath(cell string) bool {
	f := strings.TrimSpace(cell)
	if strings.HasPrefix(f, "/") || strings.HasPrefix(f, `\\`) || strings.HasPrefix(f, "file://") {
		return true
	}
	if len(f) > 2 && f[1] == ':' && (f[2] == '\\' || f[2] == '/') {
		c := f[0]
		return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
	}
	return false
}

// TestAuthoredTablesCarryNoMachinePath refuses a machine path in any cell of any authored table, which is
// what the site column of fields.tsv held in 93 rows before it was made repository-relative.
func TestAuthoredTablesCarryNoMachinePath(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("source", "*"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no authored tables under source/: %v", err)
	}
	rows := 0
	for _, file := range files {
		raw, rerr := os.ReadFile(file)
		if rerr != nil {
			t.Fatal(rerr)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			for _, cell := range strings.Split(line, "\t") {
				if isMachinePath(cell) {
					t.Errorf("%s:%d: cell %q names a machine's filesystem; cite the site relative to the repository root", file, i+1, strings.TrimSpace(cell))
				}
			}
			if strings.TrimSpace(line) != "" {
				rows++
			}
		}
	}
	if rows < 100 {
		t.Fatalf("only %d authored rows read; the scan is not looking at the tables", rows)
	}
}

// TestGeneratedArtefactsCarryNoMachinePath refuses a machine path in the committed artefacts, which the
// drift gate compares byte for byte against a fresh generation and would otherwise pass along unchanged.
// The review document is not in the list: it is generated outside the repository, so there is no
// committed copy of it to carry anything, and cmd/qki18n refuses to write one inside the checkout.
func TestGeneratedArtefactsCarryNoMachinePath(t *testing.T) {
	artefacts := []string{
		"msg_gen.go",
		filepath.Join("..", "..", "compiler", "internal", "rtcat", "msg_c.h"),
	}
	for _, file := range artefacts {
		raw, err := os.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				t.Logf("%s is not in the tree; nothing committed to check", file)
				continue
			}
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, marker := range machineMarkers {
				if strings.Contains(line, marker) {
					t.Errorf("%s:%d: contains %q, a machine's own path", file, i+1, marker)
				}
			}
		}
	}
}
