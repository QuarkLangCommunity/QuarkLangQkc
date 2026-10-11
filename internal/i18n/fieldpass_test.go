package i18n

// The struct-field pass (docs/i18n-design.md 7.5) is the one part of this migration that has no callee
// to scan. A message written into a struct field —
//
//	&RunError{Msg: fmt.Sprintf("CompileError: duplicate declaration %q", name)}
//
// — is not an argument to anything, so wiredTemplates (coverage_test.go) never sees it, the ratchets
// never count it, and the catalog's Chinese for it is never reached. The gates in this file scan for
// that shape instead, and they are the mechanical form of the claim the pass makes:
//
//   - no message literal is written into a Msg field any more (the pass is complete, and a new message
//     cannot be added that way: it fails here rather than rendering English to a Chinese reader);
//   - every row of the field table renders its authored English and Chinese, so a row that lost a
//     language cannot hide behind the count;
//   - the number of rows whose zh repeats their en — the backlog whose Chinese has to come from a
//     speaker — is frozen and may only shrink.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18ngen"
)

// fieldMessageFields are the struct fields this project writes user-facing messages into. `Msg` is the
// one that carries them today; the others are listed so that a message moved into one of them is
// caught here rather than by a reader.
var fieldMessageFields = map[string]bool{"Msg": true, "Message": true}

// fieldLiterals parses every Go source outside this package and returns the message literals written
// straight into a message field, keyed by literal with the call sites that use it.
func fieldLiterals(t *testing.T) map[string][]string {
	t.Helper()
	root := filepath.Join("..", "..")
	out := map[string][]string{}
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if skipNestedRepo(path, info, root) {
			return filepath.SkipDir
		}
		if err != nil || info.IsDir() {
			return nil
		}
		slash := filepath.ToSlash(path)
		if !strings.HasSuffix(slash, ".go") || strings.HasSuffix(slash, "_test.go") ||
			strings.Contains(slash, "/internal/i18n/") || strings.Contains(slash, "/.git/") {
			return nil
		}
		rel, _ := filepath.Rel(root, slash)
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Errorf("parse %s: %v", rel, perr)
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			kv, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || !fieldMessageFields[key.Name] {
				return true
			}
			lit, ok := fieldLiteral(kv.Value)
			if !ok {
				return true
			}
			out[lit] = append(out[lit], rel)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// fieldLiteral reports whether a field value is a message literal, and returns it: the literal itself,
// fmt.Sprintf's format argument, or the concatenation of a literal with operands (which becomes a
// %s template). A value that is a call to msgID, or a variable, is what a wired site looks like.
func fieldLiteral(e ast.Expr) (string, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(v.Value)
		if err != nil {
			return "", false
		}
		return s, true
	case *ast.CallExpr:
		sel, ok := v.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Sprintf" {
			return "", false
		}
		if len(v.Args) == 0 {
			return "", false
		}
		return fieldLiteral(v.Args[0])
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return "", false
		}
		left, okL := fieldLiteral(v.X)
		right, okR := fieldLiteral(v.Y)
		switch {
		case okL && okR:
			return left + right, true
		case okL:
			return left + "%s", true
		case okR:
			return "%s" + right, true
		}
		return "", false
	}
	return "", false
}

// TestProofNoMessageLiteralIsWrittenIntoAField is the pass's structural gate: the shape that hid 94
// literals from every other gate is empty.
//
// It landed with the value this file's first run measured and was lowered to 0 by the commit that
// closed the pass, which is the only way a ratchet may move here. The failure it prevents is concrete:
// a message written into a field renders its own text in both languages, so a reader who asked for
// Chinese gets English, and nothing in this repository notices.
func TestProofNoMessageLiteralIsWrittenIntoAField(t *testing.T) {
	const allowed = 0 // lowered from 65: the struct-field pass wired every one of the 94 literals
	literals := fieldLiterals(t)
	if len(literals) > allowed {
		t.Errorf("%d message literals are written straight into a struct field (limit %d): give the message an ID in source/fields.tsv and render it with msgID/errfID/a localizer", len(literals), allowed)
		i := 0
		for lit, files := range literals {
			if i == 15 {
				t.Errorf("  … and %d more", len(literals)-i)
				break
			}
			i++
			t.Errorf("  %q   <- %s", truncate(lit), strings.Join(files, ", "))
		}
	}
	t.Logf("message literals written into a struct field: %d (limit %d)", len(literals), allowed)
}

// TestProofFieldMessagesRenderBothLanguages walks the field table and renders every row in both
// languages. A row is a claim about what a reader sees, and an ID that does not resolve is a message
// that disappears, so the table is checked rather than trusted.
func TestProofFieldMessagesRenderBothLanguages(t *testing.T) {
	src, err := i18ngen.Load(i18ngen.Options{
		SourceDir:    "source",
		OutFile:      "msg_gen.go",
		ModuleSource: i18ngen.DefaultModuleSource,
	})
	if err != nil {
		t.Fatal(err)
	}
	en, zh := src.Table.Languages[0].Code, src.Table.Languages[1].Code
	ids := map[string]Msg{}
	for n, key := range MsgKeys() {
		ids[key] = Msg(n) // a message's number is its position in the concatenated tables
	}
	checked := 0
	for _, m := range src.Table.Messages {
		if !strings.HasPrefix(m.ID, "field.") {
			continue
		}
		id, ok := ids[m.ID]
		if !ok {
			t.Errorf("%s: the authored row has no generated ID", m.ID)
			continue
		}
		if got, want := New(stdCatalog, EN).Msg(id), src.Value(m.ID, en); got != want {
			t.Errorf("%s: the English rendering is %q, want the authored %q", m.ID, truncate(got), truncate(want))
		}
		if got, want := New(stdCatalog, ZH).Msg(id), src.Value(m.ID, zh); got != want {
			t.Errorf("%s: the Chinese rendering is %q, want the authored %q", m.ID, truncate(got), truncate(want))
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("the field table has no rows, so this proof is vacuous")
	}
	t.Logf("rendered %d field messages in both languages", checked)
}

// TestProofFieldMessagesAreReachableFromASite is the "never migrate a dead row" gate, mechanised: a
// row whose constant no call site names is a row that can never render, and it would sit in the table
// looking like work that was done.
func TestProofFieldMessagesAreReachableFromASite(t *testing.T) {
	used := identifiersUsed(t)
	src, err := i18ngen.Load(i18ngen.Options{
		SourceDir:    "source",
		OutFile:      "msg_gen.go",
		ModuleSource: i18ngen.DefaultModuleSource,
	})
	if err != nil {
		t.Fatal(err)
	}
	dead := 0
	for _, m := range src.Table.Messages {
		if !strings.HasPrefix(m.ID, "field.") {
			continue
		}
		if !used[msgConstName(m.ID)] {
			dead++
			t.Errorf("%s: no call site names %s, so this row cannot render", m.ID, msgConstName(m.ID))
		}
	}
	if dead > 0 {
		t.Errorf("%d field rows have no call site", dead)
	}
}

// identifiersUsed returns every identifier used in this tree outside this package, so a generated
// constant can be looked up without repeating the generator's naming rule per row.
func identifiersUsed(t *testing.T) map[string]bool {
	t.Helper()
	root := filepath.Join("..", "..")
	out := map[string]bool{}
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if skipNestedRepo(path, info, root) {
			return filepath.SkipDir
		}
		if err != nil || info.IsDir() {
			return nil
		}
		slash := filepath.ToSlash(path)
		if !strings.HasSuffix(slash, ".go") || strings.HasSuffix(slash, "_test.go") ||
			strings.Contains(slash, "/internal/i18n/") || strings.Contains(slash, "/.git/") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				out[id.Name] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("no identifiers were collected, so this gate would pass vacuously")
	}
	return out
}

// msgConstName mirrors i18ngen's rule for turning an ID into the constant the generated file
// declares ("field.for_in" -> "MsgFieldForIn"). It is written out here rather than exported because
// the failure mode is the safe one: if the two disagree, this gate finds no call site and fails loudly
// instead of passing.
func msgConstName(id string) string {
	var b strings.Builder
	b.WriteString("Msg")
	for _, part := range strings.Split(id, ".") {
		for _, w := range strings.Split(part, "_") {
			if w == "" {
				continue
			}
			b.WriteString(strings.ToUpper(w[:1]))
			b.WriteString(w[1:])
		}
	}
	return b.String()
}

// TestProofUntranslatedFieldMessagesAreCounted freezes the backlog this pass could not translate.
//
// The rows it counts are not a defect of the migration: they are messages that shipped English-only
// with no ID and no catalog entry, so they rendered English under QK_LANG=zh and nothing counted them.
// Giving them an ID could not invent a translation without changing what a reader sees, which the
// design forbids, so the zh column repeats the English source and the backlog is a number somebody can
// work off. It may only go down: 64 rows are the English-only literals this pass froze, and the Chinese
// for every one of them has to come from a speaker.
func TestProofUntranslatedFieldMessagesAreCounted(t *testing.T) {
	const allowed = 64 // the English-only field literals: measured by this test's first run, and the backlog
	src, err := i18ngen.Load(i18ngen.Options{
		SourceDir:    "source",
		OutFile:      "msg_gen.go",
		ModuleSource: i18ngen.DefaultModuleSource,
	})
	if err != nil {
		t.Fatal(err)
	}
	en, zh := src.Table.Languages[0].Code, src.Table.Languages[1].Code
	var untranslated []string
	for _, m := range src.Table.Messages {
		if !strings.HasPrefix(m.ID, "field.") {
			continue
		}
		if src.Value(m.ID, en) == src.Value(m.ID, zh) {
			untranslated = append(untranslated, m.ID)
		}
	}
	if len(untranslated) > allowed {
		t.Errorf("%d field messages have no Chinese translation (limit %d): the zh column repeats the source, and the Chinese has to come from a speaker", len(untranslated), allowed)
		for i, id := range untranslated {
			if i == 10 {
				t.Errorf("  … and %d more", len(untranslated)-i)
				break
			}
			t.Errorf("  %s", id)
		}
	}
	t.Logf("field messages whose zh repeats the source: %d (limit %d)", len(untranslated), allowed)
}
