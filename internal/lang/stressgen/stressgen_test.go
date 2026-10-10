package stressgen

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// findCase returns the case with this name, or false when the corpus does not carry it.
func findCase(name string) (Case, bool) {
	for _, c := range All() {
		if c.Name == name {
			return c, true
		}
	}
	return Case{}, false
}

// caseByName indexes the corpus so a test can assert one documented case at a time.
func caseByName(t *testing.T, name string) Case {
	t.Helper()
	c, ok := findCase(name)
	if !ok {
		t.Fatalf("case %q is missing from the corpus", name)
	}
	return c
}

// TestCorpusShape checks the invariants every case must satisfy, whatever it stresses.
func TestCorpusShape(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range All() {
		if seen[c.Name] {
			t.Errorf("duplicate case name %q", c.Name)
		}
		seen[c.Name] = true
		if filepathUnsafe(c.Name) {
			t.Errorf("%s: name is not a plain file name", c.Name)
		}
		if strings.Contains(c.Name, ".") {
			t.Errorf("%s: the harness appends the .kq extension itself", c.Name)
		}
		if strings.TrimSpace(c.Doc) == "" {
			t.Errorf("%s: no description", c.Name)
		}
		switch c.Category {
		case CategoryScale, CategoryPathological, CategoryMalformed, CategoryResource:
		default:
			t.Errorf("%s: unknown category %q", c.Name, c.Category)
		}
	}
	if len(All()) < 26 {
		t.Errorf("corpus shrunk to %d cases", len(All()))
	}
}

// filepathUnsafe reports whether a name could escape the case directory.
func filepathUnsafe(name string) bool {
	return name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".."
}

// TestCategoriesCoverEveryRequiredStress pins the requirement list to case names, so dropping a case
// breaks the test instead of quietly shrinking the suite.
func TestCategoriesCoverEveryRequiredStress(t *testing.T) {
	required := map[string]string{
		"one million character identifier": "scale_ident_1m",
		"100k top-level declarations":      "scale_decls_100k",
		"one megabyte line":                "scale_line_1mb",
		"10k nested blocks":                "scale_nested_blocks",
		"10k nested parentheses":           "scale_nested_parens",
		"deeply nested generics":           "scale_nested_generics",
		"very long call chain":             "scale_call_chain",
		"stack-exhausting recursion":       "path_recursion_stack",
		"integer literal beyond 64 bits":   "path_int_beyond_64",
		"bits<33> declaration":             "path_bits33",
		"function with 10k parameters":     "path_fn_10k_params",
		"struct with 10k fields":           "path_struct_10k_fields",
		"recursive macro":                  "path_recursive_macro",
		"exponentially expanding macro":    "path_exp_macro",
		"empty file":                       "bad_empty",
		"only comments":                    "bad_only_comments",
		"unterminated string":              "bad_unterminated_string",
		"unterminated block":               "bad_unterminated_block",
		"unterminated parenthesis":         "bad_unterminated_paren",
		"invalid UTF-8":                    "bad_invalid_utf8",
		"NUL byte":                         "bad_nul_byte",
		"BOM":                              "bad_bom",
		"CRLF mixed with LF":               "bad_crlf_mixed",
		"only preprocessor directives":     "bad_only_directives",
		"unbalanced #ifdef":                "bad_unbalanced_ifdef",
		"few-megabyte valid program":       "res_few_mb",
	}
	for what, name := range required {
		if _, ok := findCase(name); !ok {
			t.Errorf("requirement %q is not covered: the case %q is missing", what, name)
		}
	}
}

// TestSizesMatchTheDocumentedMagnitudes checks that each case really carries the magnitude its name
// and description promise, which is what makes the measurement comparable across runs.
func TestSizesMatchTheDocumentedMagnitudes(t *testing.T) {
	ident := caseByName(t, "scale_ident_1m")
	if want := strings.Repeat("a", IdentifierChars); !bytes.Contains(ident.Source, []byte(want)) {
		t.Errorf("scale_ident_1m: no %d-character identifier", IdentifierChars)
	}
	if len(ident.Source) < IdentifierChars {
		t.Errorf("scale_ident_1m: only %d bytes", len(ident.Source))
	}

	decls := caseByName(t, "scale_decls_100k")
	if got := strings.Count(string(decls.Source), "type struct { int v"); got != TopLevelDecls {
		t.Errorf("scale_decls_100k: %d declarations, want %d", got, TopLevelDecls)
	}

	line := caseByName(t, "scale_line_1mb")
	lines := strings.Split(strings.TrimSuffix(string(line.Source), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("scale_line_1mb: %d lines, want a single line", len(lines))
	}
	if len(lines[0]) < LineBytes {
		t.Errorf("scale_line_1mb: the line is %d bytes, want at least %d", len(lines[0]), LineBytes)
	}

	blocks := caseByName(t, "scale_nested_blocks")
	if got := strings.Count(string(blocks.Source), "if (true) {"); got != NestedBlocks {
		t.Errorf("scale_nested_blocks: %d levels, want %d", got, NestedBlocks)
	}

	parens := caseByName(t, "scale_nested_parens")
	if want := strings.Repeat("(", NestedParens) + "1"; !strings.Contains(string(parens.Source), want) {
		t.Errorf("scale_nested_parens: no %d-deep parenthesis nest", NestedParens)
	}

	generics := caseByName(t, "scale_nested_generics")
	if want := strings.Repeat("Box<", GenericDepth) + "int" + strings.Repeat(">", GenericDepth); !strings.Contains(string(generics.Source), want) {
		t.Errorf("scale_nested_generics: no %d-deep type argument nest", GenericDepth)
	}

	chain := caseByName(t, "scale_call_chain")
	if got := strings.Count(string(chain.Source), ".id()"); got != CallChainLen {
		t.Errorf("scale_call_chain: %d calls, want %d", got, CallChainLen)
	}

	params := caseByName(t, "path_fn_10k_params")
	if got := strings.Count(string(params.Source), "int p"); got != FuncParams {
		t.Errorf("path_fn_10k_params: %d parameters, want %d", got, FuncParams)
	}

	fields := caseByName(t, "path_struct_10k_fields")
	if got := strings.Count(string(fields.Source), "  int f"); got != StructFields {
		t.Errorf("path_struct_10k_fields: %d fields, want %d", got, StructFields)
	}

	macro := caseByName(t, "path_exp_macro")
	if got := strings.Count(string(macro.Source), "dup("); got != MacroDupDepth {
		t.Errorf("path_exp_macro: %d doublings, want %d", got, MacroDupDepth)
	}

	resource := caseByName(t, "res_few_mb")
	if len(resource.Source) < 2<<20 {
		t.Errorf("res_few_mb: %d bytes, want at least 2 MiB", len(resource.Source))
	}
}

// TestMalformedBytesAreIntact checks the byte-level malformations, which is the only place where the
// exact bytes matter more than the program they spell.
func TestMalformedBytesAreIntact(t *testing.T) {
	if got := len(caseByName(t, "bad_empty").Source); got != 0 {
		t.Errorf("bad_empty: %d bytes, want 0", got)
	}
	if src := caseByName(t, "bad_bom").Source; !bytes.HasPrefix(src, []byte{0xEF, 0xBB, 0xBF}) {
		t.Errorf("bad_bom: no UTF-8 byte-order mark")
	}
	if src := caseByName(t, "bad_nul_byte").Source; !bytes.Contains(src, []byte{0}) {
		t.Errorf("bad_nul_byte: no NUL byte")
	}
	if src := caseByName(t, "bad_invalid_utf8").Source; utf8.Valid(src) {
		t.Errorf("bad_invalid_utf8: the bytes are valid UTF-8")
	}
	crlf := string(caseByName(t, "bad_crlf_mixed").Source)
	if !strings.Contains(crlf, "\r\n") || !strings.Contains(strings.ReplaceAll(crlf, "\r\n", ""), "\n") {
		t.Errorf("bad_crlf_mixed: CRLF and LF are not both present")
	}
}

// TestCorpusIsDeterministic checks that two builds of the corpus are byte-identical, so a report can
// be tied to the generator revision alone.
func TestCorpusIsDeterministic(t *testing.T) {
	if !reflect.DeepEqual(All(), All()) {
		t.Fatal("two All() calls produced different corpora")
	}
}
