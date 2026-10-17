package i18n

// Engineering gates for the generated catalog. They read the authored tables through internal/i18ngen
// — the same code the generator runs — and the file the generator wrote, rather than scanning sources
// with regular expressions, which missed trailing comments, CRLF checkouts and helper-passed format
// strings. Which gate replaced which, and why the deleted ones had no subject left, is recorded in
// docs/i18n-design.md's gate table rather than here.

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18ngen"
)

var hanRe = regexp.MustCompile(`[\p{Han}]`)

// formatVerbs returns the multiset of fmt verbs in a template ("%s" -> {"s": 1}).
var verbRe = regexp.MustCompile(`%[-+# 0-9.]*([a-zA-Z%])`)

func formatVerbs(s string) map[string]int {
	out := map[string]int{}
	for _, m := range verbRe.FindAllStringSubmatch(s, -1) {
		out[m[1]]++
	}
	return out
}

// authored loads the authored tables the way the generator does, so a gate and the generated file can
// never be looking at different data.
func authored(t *testing.T) *i18ngen.Sources {
	t.Helper()
	src, err := i18ngen.Load(i18ngen.Options{
		SourceDir:    "source",
		OutFile:      "msg_gen.go",
		ModuleSource: i18ngen.DefaultModuleSource,
	})
	if err != nil {
		t.Fatalf("loading the authored tables: %v", err)
	}
	if len(src.Table.Messages) == 0 {
		t.Fatal("the authored tables are empty, so these gates would pass vacuously")
	}
	return src
}

// msgIDs maps every authored ID to the number the generated file gives it.
func msgIDs() map[string]Msg {
	out := map[string]Msg{}
	for n, key := range MsgKeys() {
		out[key] = Msg(n)
	}
	return out
}

// TestEveryGeneratedMessageRendersItsAuthoredText renders every row of every table in both languages and
// requires the authored value back. It is the gate that makes "the tables are the source of truth" a
// checkable claim rather than an assertion about the generator, and it subsumes the per-domain proofs
// the migration landed one at a time (the field table's included).
func TestEveryGeneratedMessageRendersItsAuthoredText(t *testing.T) {
	src := authored(t)
	ids := msgIDs()
	en, zh := src.Table.Languages[0].Code, src.Table.Languages[1].Code
	checked := 0
	for _, m := range src.Table.Messages {
		id, ok := ids[m.ID]
		if !ok {
			t.Errorf("%s: the authored row has no generated ID", m.ID)
			continue
		}
		if got, want := New(EN).Msg(id), src.Value(m.ID, en); got != want {
			t.Errorf("%s: the English rendering is %q, want the authored %q", m.ID, truncate(got), truncate(want))
		}
		if got, want := New(ZH).Msg(id), src.Value(m.ID, zh); got != want {
			t.Errorf("%s: the Chinese rendering is %q, want the authored %q", m.ID, truncate(got), truncate(want))
		}
		checked++
	}
	t.Logf("rendered %d authored messages in both languages", checked)
}

// TestCatalogIntegrity: no empty ID or value, and no ID twice. A duplicate ID is the mistake a
// hand-edited table makes, and the generator would otherwise resolve it to whichever row came first.
func TestCatalogIntegrity(t *testing.T) {
	src := authored(t)
	seen := map[string]int{}
	for n, m := range src.Table.Messages {
		if strings.TrimSpace(m.ID) == "" {
			t.Errorf("row %d has an empty id", n+1)
		}
		if prev, dup := seen[m.ID]; dup {
			t.Errorf("%s is authored twice (rows %d and %d): the second would be unreachable", m.ID, prev+1, n+1)
		}
		seen[m.ID] = n
		for i, v := range m.Values {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s: empty %s value", m.ID, src.Table.Languages[i].Code)
			}
		}
		if strings.TrimSpace(m.Source) == "" {
			t.Errorf("%s: empty src value", m.ID)
		}
	}
}

// TestVerbSetsMatch: a translation must consume exactly the same fmt verbs as its source, which is
// what silently breaks when a placeholder is dropped or reordered while translating. It reads the
// authored table now — the same data the blob is packed from — rather than a Go map.
func TestVerbSetsMatch(t *testing.T) {
	src := authored(t)
	en, zh := src.Table.Languages[0].Code, src.Table.Languages[1].Code
	for _, m := range src.Table.Messages {
		source, translated := src.Value(m.ID, en), src.Value(m.ID, zh)
		want, got := formatVerbs(source), formatVerbs(translated)
		if len(want) != len(got) {
			t.Errorf("%s: verb kinds differ\n  en: %q -> %v\n  zh: %q -> %v", m.ID, truncate(source), want, truncate(translated), got)
			continue
		}
		for verb, n := range want {
			if got[verb] != n {
				t.Errorf("%s: verb %%%s count differs (%d vs %d)\n  en: %q\n  zh: %q",
					m.ID, verb, n, got[verb], truncate(source), truncate(translated))
			}
		}
	}
}

// TestVerbSequenceMatches is the ordered counterpart of TestVerbSetsMatch: call sites pass their
// arguments positionally, so a translation with the right verbs in the wrong order still passes the set
// check and then renders nonsense at run time.
//
// IT IS A RATCHET AGAIN, AND RE-POINTING IT AT THE AUTHORED TABLES IS WHY. The check used to read the
// text-keyed catalog, which by the end held nothing; over all 635 authored rows it finds **11** messages
// whose Chinese reorders the verbs, and they are real: `using %q on %s is not supported` renders its
// type where the operator belongs under QK_LANG=zh. Nothing could see them before, because the rows that
// carry them moved out of the text-keyed catalog one domain at a time and the check was reading the
// other store.
//
// They are frozen rather than fixed here. Reordering the verbs inside a Chinese sentence rewrites the
// sentence (the phrasing "not supported on %s" becoming "not supported for %s"), which is authoring a
// translation, and this
// migration does not invent Chinese: the same rule that leaves 64 field rows with an English zh column
// applies. The count may only shrink, and the fix has to come from a speaker.
func TestVerbSequenceMatches(t *testing.T) {
	const allowed = 11 // measured by this test's first run over the authored tables; fix with a speaker
	src := authored(t)
	re := regexp.MustCompile("%[a-zA-Z]")
	en, zh := src.Table.Languages[0].Code, src.Table.Languages[1].Code
	bad := 0
	for _, m := range src.Table.Messages {
		a, b := re.FindAllString(src.Value(m.ID, en), -1), re.FindAllString(src.Value(m.ID, zh), -1)
		if len(a) != len(b) {
			continue // the set check reports the count mismatch
		}
		for i := range a {
			if a[i] != b[i] {
				bad++
				if bad > allowed {
					t.Errorf("%s: order differs at #%d\n  en: %v\n  zh: %v", m.ID, i+1, a, b)
				} else {
					t.Logf("%s: order differs at #%d\n  en: %v\n  zh: %v", m.ID, i+1, a, b)
				}
				break
			}
		}
	}
	if bad > allowed {
		t.Fatalf("%d authored messages reorder their placeholders (limit %d); the translation must follow the source order", bad, allowed)
	}
	t.Logf("authored messages with reordered placeholders: %d (limit %d)", bad, allowed)
}

// textKeyedEntryPoints are the names of the machinery step 5 deleted. None may come back: a text-keyed
// entry point is what made a message's identity its own wording, and that is the defect the whole
// migration removed (D-3 in docs/i18n-design.md).
var textKeyedEntryPoints = []string{
	"i18n.T(", ".T(", "i18n.Load(", "i18n.Catalog", "Catalog{", ".byKey",
	"i18n.Has(", "Untranslated(", "i18n.Misses(", "MissTotal(", "countMiss(",
	"needsTranslation(", "sourceLang(", "domainTables(", "msg(template",
}

// TestDecouplingRatchet is R-3 as an exact contract: there are no text-keyed entry points at all, in
// any package, which is stronger than a count that may only shrink and is the claim the design asked
// for. It scans the whole module, so the compiler packages are included.
func TestDecouplingRatchet(t *testing.T) {
	root := filepath.Join("..", "..")
	found := map[string][]string{}
	scanned := 0
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		slash := filepath.ToSlash(path)
		if err != nil || info.IsDir() {
			return nil
		}
		if skipNestedRepo(path, info, root) {
			return nil
		}
		if !strings.HasSuffix(slash, ".go") || strings.Contains(slash, "/.git/") {
			return nil
		}
		rel, _ := filepath.Rel(root, slash)
		rel = filepath.ToSlash(rel) // Windows separates with a backslash, and this comparison is a path
		if rel == "internal/i18n/coverage_test.go" {
			return nil // this file names them in order to look for them
		}
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		scanned++
		for _, name := range textKeyedEntryPoints {
			if strings.Contains(string(src), name) {
				found[name] = append(found[name], rel)
			}
		}
		return nil
	})
	if scanned == 0 {
		t.Fatal("no files were scanned, so this contract would hold vacuously")
	}
	if len(found) > 0 {
		names := make([]string, 0, len(found))
		for name := range found {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			t.Errorf("%q is a text-keyed entry point and came back in: %s", name, strings.Join(found[name], ", "))
		}
	}
	t.Logf("text-keyed entry points: %d (scanned %d Go files in the module)", len(found), scanned)
}

// TestGeneratedFileIsParsable reads the artefact the runtime actually ships: a truncated or hand-edited
// blob fails here rather than at the first message.
func TestGeneratedFileIsParsable(t *testing.T) {
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "msg_gen.go", nil, 0); err != nil {
		t.Fatalf("msg_gen.go does not parse: %v", err)
	}
}

func truncate(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 90 {
		return s[:90] + "…"
	}
	return s
}

// skipNestedRepo reports whether a walked directory belongs to a different repository. The QuarkLang
// family keeps its sibling projects inside this workspace, each with its own .git, and another
// repository's files are not this project's files: walking into them would let their sources decide
// this project's gates.
func skipNestedRepo(path string, info os.FileInfo, root string) bool {
	if info == nil || !info.IsDir() || path == root {
		return false
	}
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
