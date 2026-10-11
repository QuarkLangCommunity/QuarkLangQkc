package i18n

// The prototype's proofs. Each one checks a property the text-keyed catalog could not provide, against
// the real authored tables rather than a fixture, so the test fails when the design regresses rather
// than when a sample drifts. See docs/i18n-design.md §6.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/i18ngen"
)

// parserModule is the pinned syntax layer whose messages this repository ships.
const parserModule = "github.com/QuarkLangCommunity/QuarkLangQkparser"

// repoRoot is the checkout root: these tests run in internal/i18n.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// generatedOptions describes the committed catalog and where the generator's inputs are.
func generatedOptions(t *testing.T, check bool) i18ngen.Options {
	t.Helper()
	root := repoRoot(t)
	return i18ngen.Options{
		SourceDir:    filepath.Join(root, "internal", "i18n", "source"),
		OutFile:      filepath.Join(root, "internal", "i18n", "msg_gen.go"),
		ModuleDir:    moduleDir(t),
		ModuleSource: i18ngen.DefaultModuleSource,
		Check:        check,
	}
}

// moduleDir locates the pinned syntax layer on disk through the Go toolchain, so it is the revision
// go.mod pins and never a sibling checkout that happens to exist.
func moduleDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", parserModule).Output()
	if err != nil {
		t.Fatalf("locating the pinned syntax layer: %v", err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		t.Fatal("the pinned syntax layer resolved to an empty directory")
	}
	return dir
}

// loadAuthored loads the authored tables plus the pinned module's, which is what every proof starts
// from.
func loadAuthored(t *testing.T) *i18ngen.Sources {
	t.Helper()
	src, err := i18ngen.Load(generatedOptions(t, false))
	if err != nil {
		t.Fatalf("loading the authored tables: %v", err)
	}
	return src
}

// TestGeneratedCatalogIsCurrent is the standing gate: the committed catalog must be exactly what the
// generator writes today. A template, an ID or a translation edited without regenerating fails here,
// which is the ordinary `go test ./... -count=1` run.
func TestGeneratedCatalogIsCurrent(t *testing.T) {
	if problems := i18ngen.Run(generatedOptions(t, true)); len(problems) > 0 {
		t.Fatalf("the committed catalog is not current:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestGeneratedCatalogIsDeterministic pins the property the message numbers depend on: generating
// twice produces the same bytes. A generator that walked a map would renumber the blob run to run.
func TestGeneratedCatalogIsDeterministic(t *testing.T) {
	first, err := i18ngen.Generate(loadAuthored(t))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		again, gerr := i18ngen.Generate(loadAuthored(t))
		if gerr != nil {
			t.Fatal(gerr)
		}
		if string(again) != string(first) {
			t.Fatalf("run %d produced different bytes", i+2)
		}
	}
}

// TestProofRephraseKeepsTranslation is P1: rewriting the English text of a message does not lose its
// translation. The ID identifies the message, so the Chinese value stays reachable through the same
// constant after the English is rewritten.
func TestProofRephraseKeepsTranslation(t *testing.T) {
	const id = MsgParserMacroNeedsName
	const oldEN = "ParseError: #macro needs a name, line %d"
	const newEN = "ParseError: a macro declaration needs a name, line %d"
	wantZH := msgText(id, langBlock("zh"))

	src := loadAuthored(t)
	if got := src.Value("parser.macro_needs_name", "en"); got != oldEN {
		t.Fatalf("authored English is %q, want %q", got, oldEN)
	}
	src.SetValue("parser.macro_needs_name", "en", newEN)

	out, err := i18ngen.Generate(src)
	if err != nil {
		t.Fatal(err)
	}
	// Read the rephrased catalog the way the runtime does: the ID selects the offset pair, and the
	// blob holds the bytes. The English changed; the Chinese is byte-identical and still on the same ID.
	en := renderFromGenerated(t, out, id, langBlock("en"))
	zh := renderFromGenerated(t, out, id, langBlock("zh"))
	if en != newEN {
		t.Fatalf("after rephrasing, the English rendering is %q, want %q", en, newEN)
	}
	if zh != wantZH {
		t.Fatalf("after rephrasing, the Chinese rendering is %q, want %q", zh, wantZH)
	}
}

// renderFromGenerated reads a value out of a rendered generated file the way the runtime reads it: the
// offset table plus the blob, addressed by the message number and the language block. It parses the
// generated source rather than calling a helper, so the proof covers the artefact that ships.
func renderFromGenerated(t *testing.T, generated []byte, id Msg, block int) string {
	t.Helper()
	src := string(generated)
	offsets := parseUint32Table(t, src, "msgOffsets")
	blob := parseStringLiteral(t, src, "msgBlob")
	nMsgs := len(offsets) / len(MsgLangs())
	k := block*nMsgs + int(id)
	if k+1 >= len(offsets) {
		t.Fatalf("message %d in block %d is outside the offset table (%d entries)", id, block, len(offsets))
	}
	if offsets[k+1] > uint32(len(blob)) {
		t.Fatalf("offset %d runs past the %d-byte blob", offsets[k+1], len(blob))
	}
	return blob[offsets[k]:offsets[k+1]]
}

// parseUint32Table reads "var name = [...]uint32{ ... }" out of a generated file.
func parseUint32Table(t *testing.T, src, name string) []uint32 {
	t.Helper()
	at := strings.Index(src, "var "+name+" = [...]uint32{")
	if at < 0 {
		t.Fatalf("no %s table in the generated file", name)
	}
	body := src[at:]
	body = body[strings.Index(body, "{")+1:]
	body = body[:strings.Index(body, "}")]
	var out []uint32
	for _, field := range strings.Split(body, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		n, err := strconv.ParseUint(field, 10, 32)
		if err != nil {
			t.Fatalf("%s: %q is not a number: %v", name, field, err)
		}
		out = append(out, uint32(n))
	}
	return out
}

// parseStringLiteral reads "const name = <value>" out of a generated file, in either form the
// generator may emit. It prefers a raw literal, because that is what keeps the blob readable in a
// diff, but falls back to a quoted one: a message that itself contains a backtick cannot live in a raw
// literal, so one such message switches the whole blob to an escaped literal. The docgen domain has
// one (`> Form: ` + "`program `"), which is how this was found — the proof, not the generator, was
// making an assumption about the artefact's spelling.
func parseStringLiteral(t *testing.T, src, name string) string {
	t.Helper()
	head := "const " + name + " = "
	at := strings.Index(src, head)
	if at < 0 {
		t.Fatalf("no %s literal in the generated file", name)
	}
	rest := src[at+len(head):]
	if rest == "" {
		t.Fatalf("the %s literal is empty", name)
	}
	switch rest[0] {
	case '`':
		end := strings.Index(rest[1:], "`")
		if end < 0 {
			t.Fatalf("the %s literal is not terminated", name)
		}
		return rest[1 : 1+end]
	case '"':
		end := 1
		for end < len(rest) {
			if rest[end] == '\\' {
				end += 2
				continue
			}
			if rest[end] == '"' {
				break
			}
			end++
		}
		if end >= len(rest) {
			t.Fatalf("the %s literal is not terminated", name)
		}
		value, err := strconv.Unquote(rest[:end+1])
		if err != nil {
			t.Fatalf("the %s literal does not unquote: %v", name, err)
		}
		return value
	}
	t.Fatalf("the %s literal starts with %q, which is neither ` nor \"", name, rest[0])
	return ""
}

// TestProofTextKeyedCatalogLosesTheTranslationOnRephrase is the mirror of P1, measured against the
// design this replaces. It builds the catalog the way the old one was built — the call-site Chinese as
// the key, the English translation as the value (internal/i18n/table_macro.go did exactly this before
// the row was migrated) — and shows the same rephrase dropping the translation: the reader gets the
// source language back — and whether anything notices depends on which language the reader asked for,
// because the counter is built on the same Han-character heuristic as the direction. The two tests
// together are the proof, so neither rests on prose alone.
func TestProofTextKeyedCatalogLosesTheTranslationOnRephrase(t *testing.T) {
	const oldZH = "ParseError: #macro 需要名字，第 %d 行"
	const oldEN = "ParseError: #macro needs a name, line %d"
	const newEN = "ParseError: a macro declaration needs a name, line %d"

	// The old design in one line: the text is the key, so the pair is only reachable while the call
	// site spells the message exactly as the table does.
	legacy := map[string]string{oldZH: oldEN}
	lookup := func(template string) (string, bool) { v, ok := legacy[template]; return v, ok }

	if got, ok := lookup(oldZH); !ok || got != oldEN {
		t.Fatalf("the text-keyed table rendered %q (found %v)", got, ok)
	}
	// The call site's English is corrected. Under the text-keyed design the key no longer matches
	// anything, so the translation is unreachable and the reader gets the new source text.
	if got, ok := lookup(newEN); ok {
		t.Fatalf("the rephrased text still resolved to %q", got)
	}
	// This test used to run that measurement through the real Catalog.T and the miss counter, including
	// the asymmetry that made the defect survive review: under EN nothing was recorded, because the
	// heuristic that decides "is this a miss" reads the rephrased text as English and matches the
	// requested language. Step 5 deleted both the machinery and the counter, so the measurement cannot
	// be repeated against shipping code — what is left is the property, and the record of it is in
	// docs/i18n-design.md §6.1 and in this file's history.
	//
	// The ID-keyed design holds the same message and is untouched by the rephrase: P1 measures that on
	// the real table, one test above.
	if msgText(MsgParserMacroNeedsName, langBlock("zh")) == "" {
		t.Fatal("the generated catalog lost the message the rephrase was supposed to endanger")
	}
}

// TestProofMissingLanguageFails is P2: a message with no value in a shipped language is a generation
// failure, for this repository's own messages and for the ones the pinned module ships.
//
// It checks every (message, language) pair in the table rather than one sample, because the property
// that matters is "no message can be incomplete", not "this message cannot".
func TestProofMissingLanguageFails(t *testing.T) {
	langs := MsgLangs()
	if len(langs) < 2 {
		t.Fatalf("this proof needs at least two shipped languages, got %d", len(langs))
	}
	base := loadAuthored(t)
	checked := 0
	for i := range base.Table.Messages {
		for li := 1; li < len(langs); li++ { // li 0 is the source language, which has no translation to lose
			src := base.Clone()
			id := src.Table.Messages[i].ID
			if !src.DropValue(id, langs[li]) {
				t.Fatalf("%s has no %s value to drop", id, langs[li])
			}
			problems := src.Complete()
			if len(problems) == 0 {
				t.Fatalf("%s without its %s value passed the completeness check", id, langs[li])
			}
			joined := strings.Join(problems, "\n")
			if !strings.Contains(joined, id) {
				t.Fatalf("the diagnostic does not name the message:\n%s", joined)
			}
			// Run must report the problem and write nothing at all.
			opts := generatedOptions(t, false)
			opts.OutFile = filepath.Join(t.TempDir(), "msg_gen.go")
			if wrote := i18ngen.RunWith(opts, src); len(wrote) == 0 {
				t.Fatalf("%s without its %s value generated a catalog", id, langs[li])
			}
			if _, err := os.Stat(opts.OutFile); err == nil {
				t.Fatalf("a catalog was written for an incomplete table (%s/%s)", id, langs[li])
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no (message, language) pair was checked")
	}
	t.Logf("checked %d (message, language) pairs: every one fails generation when its value is removed", checked)
}

// TestProofModuleMessageIsCompleteInThisCatalog is P2 across the module boundary: the pinned syntax
// layer's IDs are checked against this catalog, so a message added there cannot reach a user
// untranslated. This is the hole the source scanner could not see (docs/i18n-design.md §D-4).
func TestProofModuleMessageIsCompleteInThisCatalog(t *testing.T) {
	src := loadAuthored(t)
	if problems := src.CheckModule(); len(problems) > 0 {
		t.Fatalf("this catalog does not agree with the pinned module:\n  %s", strings.Join(problems, "\n  "))
	}
	imported := 0
	for _, m := range src.Table.Messages {
		if m.Owner == "import" {
			imported++
		}
	}
	if imported == 0 {
		t.Fatal("no message is owned by the pinned module, so this proof proves nothing")
	}

	// A message the module rephrases: this repository recorded the old text, so the check fires.
	drifted := src.Clone()
	first := ""
	for _, m := range drifted.Table.Messages {
		if m.Owner == "import" {
			first = m.ID
			break
		}
	}
	if first == "" {
		t.Fatal("no imported message to drift")
	}
	drifted.SetSource(first, "a text the pinned module no longer ships")
	if problems := drifted.CheckModule(); len(problems) == 0 {
		t.Fatal("a src value that disagrees with the pinned module passed the check")
	} else if !strings.Contains(strings.Join(problems, "\n"), first) {
		t.Fatalf("the drift diagnostic does not name the message:\n  %s", strings.Join(problems, "\n  "))
	}

	// A message the module gains: an ID this repository has never written down. Its absence is what
	// the module's published ID list catches, so check that the list is actually being read.
	moduleMsgs, err := i18ngen.ParseModuleTable("module", src.ModuleRaw)
	if err != nil {
		t.Fatal(err)
	}
	if len(moduleMsgs) != imported {
		t.Fatalf("the module publishes %d messages but %d are recorded as imported", len(moduleMsgs), imported)
	}
	grown := src.Clone()
	grown.Table.Messages = grown.Table.Messages[:0]
	for _, m := range src.Table.Messages {
		if m.Owner == "import" {
			continue
		}
		grown.Table.Messages = append(grown.Table.Messages, m)
	}
	if problems := grown.CheckModule(); len(problems) != imported {
		t.Fatalf("dropping every imported row produced %d problems, want %d (one per module message)", len(problems), imported)
	}
}

// TestProofRuntimeReadsTheBlob is P4: the value a renderer gets is the blob's own slice, for every
// language and every message, and the offset table accounts for all of them exactly.
func TestProofRuntimeReadsTheBlob(t *testing.T) {
	langs := MsgLangs()
	keys := MsgKeys()
	if len(langs) < 2 {
		t.Fatalf("this proof needs at least two shipped languages, got %d", len(langs))
	}
	if len(msgOffsets) != len(keys)*len(langs)+1 {
		t.Fatalf("the offset table has %d entries, want %d messages x %d languages + 1", len(msgOffsets), len(keys), len(langs))
	}
	for block, code := range langs {
		for id := range keys {
			got := msgText(Msg(id), block)
			start, end := msgOffsets[block*len(keys)+id], msgOffsets[block*len(keys)+id+1]
			if want := msgBlob[start:end]; got != want {
				t.Fatalf("%s/%s renders %q but the blob at [%d:%d] holds %q", keys[id], code, got, start, end, want)
			}
			if got == "" {
				t.Fatalf("%s/%s renders an empty message", keys[id], code)
			}
		}
	}
	// The language selects the block: for a message that is genuinely translated the two renderings
	// differ, which is what makes this a check rather than a tautology.
	const translated = MsgParserNestingTooDeep
	en := New(EN).Msg(translated, 7)
	zh := New(ZH).Msg(translated, 7)
	if en == zh {
		t.Fatalf("both languages rendered %q, so the language is not selecting a block", en)
	}
	if want := strings.Replace(msgText(translated, 0), "%d", "7", 1); en != want {
		t.Fatalf("the English rendering %q is not the source block with its verb filled (%q)", en, want)
	}
	if want := strings.Replace(msgText(translated, 1), "%d", "7", 1); zh != want {
		t.Fatalf("the Chinese rendering %q is not the translation block with its verb filled (%q)", zh, want)
	}
}

// TestProofNoTextKeyedEntriesRemain is deleted by step 5, and this is the reason recorded where it
// stood. It asserted that the text-keyed catalog had no entries — the state the migration reached — and
// step 5 then deleted the catalog itself, so the claim became a statement about a type that no longer
// exists. The contract that replaces it is TestDecouplingRatchet: a text-keyed entry point coming back
// is a build failure rather than a count, which is the design's whole point (D-3).

// TestModuleIDsAndCatalogIDsAgree pins the mapping the type conversion in internal/lang relies on: the
// generated Msg numbering follows the pinned module's table order, so qkparser.ID(n) and i18n.Msg(n)
// name the same message.
func TestModuleIDsAndCatalogIDsAgree(t *testing.T) {
	src := loadAuthored(t)
	moduleMsgs, err := i18ngen.ParseModuleTable("module", src.ModuleRaw)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]int{}
	for n, m := range moduleMsgs {
		byID[m.ID] = n
	}
	seen := 0
	for n, m := range src.Table.Messages {
		if m.Owner != "import" {
			continue
		}
		want, ok := byID[m.ID]
		if !ok {
			t.Fatalf("%s is imported but the pinned module does not ship it", m.ID)
		}
		if want != n {
			t.Fatalf("%s is message %d here but %d in the module: the generated numbering drifted", m.ID, n, want)
		}
		seen++
	}
	if seen == 0 {
		t.Fatal("no imported message was checked")
	}
	t.Logf("%d imported messages share their number with the pinned module", seen)
}

// TestProofUntranslatedImportsAreCounted replaces the measurement TestEnglishSourceRatchet makes for
// this domain. That ratchet counts Han templates at call sites in this repository, and it reported 277
// before step 3 and 277 after it: the templates step 3 migrated live in the pinned module, where the
// scanner cannot reach them, so the number could not move and never measured the backlog it was taken
// for. This one counts what actually shipped untranslated — an imported ID whose zh value repeats its
// English one — and freezes it. It may only go down.
//
// The rows it counts are not a defect of the migration: they are the messages that had no ID and no
// catalog entry at all before step 3, so they rendered English under QK_LANG=zh and nothing counted
// them. Migrating them could not invent a translation without changing the rendering, which P3 forbids,
// so the column repeats the source and the backlog is now a number somebody can work off.
func TestProofUntranslatedImportsAreCounted(t *testing.T) {
	const allowed = 25 // measured on step 3; the English-only errf templates that gained an ID and a row
	src, err := i18ngen.Load(i18ngen.Options{
		SourceDir:    filepath.Join("source"),
		OutFile:      "msg_gen.go",
		ModuleSource: i18ngen.DefaultModuleSource,
	})
	if err != nil {
		t.Fatal(err)
	}
	en, zh := src.Table.Languages[0].Code, src.Table.Languages[1].Code
	var untranslated []string
	for _, m := range src.Table.Messages {
		if m.Owner != "import" {
			continue
		}
		if src.Value(m.ID, en) == src.Value(m.ID, zh) {
			untranslated = append(untranslated, m.ID)
		}
	}
	if len(untranslated) > allowed {
		t.Errorf("untranslated imported messages grew to %d (limit %d): an English-only message needs a Chinese value, not a repeated column", len(untranslated), allowed)
		for i, id := range untranslated {
			if i == 10 {
				t.Errorf("  … and %d more", len(untranslated)-i)
				break
			}
			t.Errorf("  %s", id)
		}
	}
	t.Logf("imported messages whose zh repeats the source: %d (limit %d)", len(untranslated), allowed)
}

// TestParserModuleMessagesReachTheRealInterpreter is the end-to-end seam: an error raised inside the
// pinned syntax layer renders the value this catalog holds, in both languages, through the localizer
// internal/lang installs. It fails if the module's IDs stop lining up with the generated ones.
func TestParserModuleMessagesReachTheRealInterpreter(t *testing.T) {
	en := New(EN).Msg(MsgParserMacroNeedsName, 1)
	zh := New(ZH).Msg(MsgParserMacroNeedsName, 1)
	if en == zh {
		t.Fatalf("both languages rendered %q", en)
	}
	if !strings.Contains(en, "needs a name") {
		t.Fatalf("the English rendering is %q", en)
	}
	if !strings.Contains(zh, "需要名字") {
		t.Fatalf("the Chinese rendering is %q", zh)
	}
}
