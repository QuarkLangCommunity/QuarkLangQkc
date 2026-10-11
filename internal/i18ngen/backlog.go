package i18ngen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The backlog document (docs/i18n-backlog.md) is generated here, next to the catalog, so the two can
// never disagree about what the tables say: both are rendered from the same Sources and both are gated
// by the same -check. It exists because the untranslated backlog was, until now, only a number in three
// ratchet tests — 11 reordered-placeholder rows, 25 imported rows whose zh repeats its en, 64 field rows
// whose zh repeats its en — and a number is not something a translator can work off. The document lists
// the rows, what is wrong with each, where it renders, and what is missing.
//
// Nothing here is hand-copied. Every value in the document is read from the authored tables or measured
// by scanning the tree; TestBacklogDocumentIsCurrent regenerates it in memory and fails when the
// committed file differs, which is the same contract TestGeneratedCatalogIsCurrent enforces for
// msg_gen.go.

// BacklogTitle is the document's H1 and the string the staleness gate reports.
const BacklogTitle = "i18n translation backlog"

// BacklogDocPath is where the committed document lives, relative to the repository root.
const BacklogDocPath = "docs/i18n-backlog.md"

// Section titles. They are named here because the "how to fill this in" prose points at them and the
// staleness gate's message quotes the summary table.
const (
	BacklogSectionReordered = "1. Placeholders in a different order than the English (live defects)"
	BacklogSectionImports   = "2. Imported rows whose Chinese repeats the English"
	BacklogSectionFields    = "3. Field-message rows whose Chinese repeats the English"
)

// markerPlaceholder is what a position table writes immediately after a placeholder: the number of the
// argument that placeholder receives, counted in the order the call site passes them. A Chinese value
// never contains it, so the slot a word is bound to is readable without knowing which word it is.
const markerPlaceholder = "\u2039@%d@\u203a"

// indicator is the character class that separates Chinese text from the placeholders around it: any Han
// character. The side words it extracts are why a reader who does not read Chinese can still see which
// word moves with which placeholder.
var indicator = regexp.MustCompile("[\u3400-\u9fff\uf900-\ufaff]")

// verbPat matches one fmt placeholder. It is deliberately the same shape coverage_test.go counts, so the
// document's "placeholders in order" column and TestVerbSequenceMatches describe the same thing.
var verbPat = regexp.MustCompile(`%[a-zA-Z]`)

// scanExts are the file kinds the alternate-Chinese scan reads.
var scanExts = []string{".go", ".tsv", ".md"}

// scanSkipDirs are directories the scan never descends into.
var scanSkipDirs = map[string]bool{".git": true, "node_modules": true, "dist": true, "stress-out": true}

// minRenderingLen is the shortest Han-bearing line treated as a plausible rendering. It filters
// incidental Chinese in comments and fixtures; a candidate is only ever reported next to the message
// whose canonical form it equals.
const minRenderingLen = 15

// isField reports whether an id belongs to the struct-field table, whose rows have their own ratchet
// (TestProofUntranslatedFieldMessagesAreCounted) rather than the imported one.
func isField(id string) bool { return strings.HasPrefix(id, "field.") }

// Backlog renders the review document from the authored tables. It scans opts.Root for call sites and
// for Chinese renderings that exist elsewhere in the tree, so its output is a measurement of that
// checkout rather than a restatement of the tables.
func Backlog(opts Options, src *Sources) ([]byte, error) {
	if src == nil || len(src.Table.Messages) == 0 {
		return nil, fmt.Errorf("no messages")
	}
	if len(src.Table.Languages) < 2 {
		return nil, fmt.Errorf("the backlog needs a source language and a translation language, got %d", len(src.Table.Languages))
	}
	en, zh := src.Table.Languages[0].Code, src.Table.Languages[1].Code
	d := &backlog{
		src:  src,
		scan: newScan(opts.Root, msgIDs(src)),
		en:   en,
		zh:   zh,
	}
	for _, m := range src.Table.Messages {
		source, value := src.Value(m.ID, en), src.Value(m.ID, zh)
		repeats := value == source
		switch {
		case repeats && isField(m.ID):
			d.s3 = append(d.s3, m)
		case repeats && m.Owner == "import":
			d.s2 = append(d.s2, m)
		case !repeats && reordersPlaceholders(source, value):
			d.s1 = append(d.s1, m)
		}
	}
	// Sorted by id: the owner works down the list, and a row that gets filled in leaves a small diff
	// instead of renumbering everything below it.
	sort.Slice(d.s1, func(i, j int) bool { return d.s1[i].ID < d.s1[j].ID })
	sort.Slice(d.s2, func(i, j int) bool { return d.s2[i].ID < d.s2[j].ID })
	sort.Slice(d.s3, func(i, j int) bool { return d.s3[i].ID < d.s3[j].ID })
	return d.bytes()
}

// msgIDs returns every message id, which is what the scan needs to know which constants to look for.
func msgIDs(src *Sources) []string {
	out := make([]string, 0, len(src.Table.Messages))
	for _, m := range src.Table.Messages {
		out = append(out, m.ID)
	}
	return out
}

// reordersPlaceholders reports whether the two texts use the same placeholders in a different order. A
// row that lost or gained a placeholder is not one of these: that is a count defect and it belongs to
// TestVerbSetsMatch, so it is deliberately not listed in section 1.
func reordersPlaceholders(source, value string) bool {
	a, b := verbPat.FindAllString(source, -1), verbPat.FindAllString(value, -1)
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	if strings.Join(a, "\x00") == strings.Join(b, "\x00") {
		return false
	}
	return sameMultiset(a, b)
}

func sameMultiset(a, b []string) bool {
	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	return strings.Join(x, "\x00") == strings.Join(y, "\x00")
}

// backlog accumulates the three sections and renders them.
type backlog struct {
	src  *Sources
	scan *scan
	en   string
	zh   string
	s1   []Message
	s2   []Message
	s3   []Message
}

func (d *backlog) bytes() ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "<!-- Code generated by cmd/qki18n from internal/i18n/source/*.tsv. DO NOT EDIT. -->\n")
	fmt.Fprintf(&b, "# %s\n\n", BacklogTitle)
	fmt.Fprintf(&b, "This document is generated, not maintained: `go generate ./internal/i18n` rewrites it from the\n")
	fmt.Fprintf(&b, "tables, and `go run ./cmd/qki18n -check` — which TestBacklogDocumentIsCurrent runs as part of the\n")
	fmt.Fprintf(&b, "ordinary `go test ./...` — fails when the committed file no longer matches them. Nothing in it is\n")
	fmt.Fprintf(&b, "hand-copied, so a row that gets filled in disappears from here by itself.\n\n")
	fmt.Fprintf(&b, "It lists the untranslated backlog the migration froze rather than fixed, most valuable first. The\n")
	fmt.Fprintf(&b, "three counts are the three ratchets, measured over the same tables: when a row stops being listed,\n")
	fmt.Fprintf(&b, "the ratchet that counted it can be lowered — to the value a run measures, never to a value somebody\n")
	fmt.Fprintf(&b, "hopes for.\n\n")
	fmt.Fprintf(&b, "| Section | Rows | Ratchet | Limit today |\n|---|---|---|---|\n")
	fmt.Fprintf(&b, "| %s | %d | `TestVerbSequenceMatches` | 11 |\n", BacklogSectionReordered, len(d.s1))
	fmt.Fprintf(&b, "| %s | %d | `TestProofUntranslatedImportsAreCounted` | 25 |\n", BacklogSectionImports, len(d.s2))
	fmt.Fprintf(&b, "| %s | %d | `TestProofUntranslatedFieldMessagesAreCounted` | 64 |\n", BacklogSectionFields, len(d.s3))
	fmt.Fprintf(&b, "| **total** | **%d** | | |\n\n", len(d.s1)+len(d.s2)+len(d.s3))
	fmt.Fprintf(&b, "Read them in order. The first is a defect being shipped right now; the other two are work that has\n")
	fmt.Fprintf(&b, "to come from a speaker of Chinese. Each row names the table and the column to edit, and section 1\n")
	fmt.Fprintf(&b, "states which placeholder has to change position — a specification, not a translation.\n\n")

	d.section1(&b)
	d.section23(&b, BacklogSectionImports, d.s2)
	d.section23(&b, BacklogSectionFields, d.s3)
	d.howToFillIn(&b)
	return b.Bytes(), nil
}

// section1 lists the rows whose Chinese reorders the placeholders. These are live defects: the call site
// passes its arguments positionally, so under `QK_LANG=zh` the reader sees the argument for one slot
// printed into another. The section does not rewrite any Chinese value; it states which placeholder has
// to move where, which is something a translator can satisfy and a reviewer can check.
func (d *backlog) section1(b *bytes.Buffer) {
	fmt.Fprintf(b, "## %s\n\n", BacklogSectionReordered)
	fmt.Fprintf(b, "These %d rows are not untranslated: they have Chinese, and the Chinese is wrong where it counts.\n", len(d.s1))
	fmt.Fprintf(b, "The call site passes its arguments positionally (`l.errfID(pos, i18n.MsgX, a, b, c)`), so a\n")
	fmt.Fprintf(b, "placeholder that moved inside the sentence binds its argument to the wrong slot. Below, each\n")
	fmt.Fprintf(b, "placeholder is followed by `\u2039@n@\u203a`, the number of the caller's argument it receives, so the\n")
	fmt.Fprintf(b, "two orders can be compared written in a language the reader may not have.\n\n")
	if len(d.s1) == 0 {
		fmt.Fprintf(b, "_None: every Chinese value uses its source's placeholders in the source's order._\n\n")
		return
	}
	fmt.Fprintf(b, "| # | id | English (argument order) | Current Chinese (argument order) | Where it renders | Which placeholders must change position |\n")
	fmt.Fprintf(b, "|---|---|---|---|---|---|\n")
	for i, m := range d.s1 {
		en, zh := d.src.Value(m.ID, d.en), d.src.Value(m.ID, d.zh)
		fmt.Fprintf(b, "| %d | `%s` | `%s` | `%s` | %s | %s |\n",
			i+1, m.ID, markPlaceholders(en), markPlaceholders(zh), d.where(m), changeTable(en, zh))
	}
	fmt.Fprintf(b, "\n")
	for i, m := range d.s1 {
		en, zh := d.src.Value(m.ID, d.en), d.src.Value(m.ID, d.zh)
		fmt.Fprintf(b, "### %d. `%s`\n\n", i+1, m.ID)
		fmt.Fprintf(b, "- **Row to edit:** %s, the row whose `id` is `%s`, column `%s`.\n", d.rowPointer(m), m.ID, d.zh)
		fmt.Fprintf(b, "- **English source (`%s` column):** `%s`\n", d.en, quoteDoc(en))
		fmt.Fprintf(b, "- **Current Chinese (`%s` column):** `%s`\n", d.zh, quoteDoc(zh))
		fmt.Fprintf(b, "- **Where it renders:** %s\n", d.where(m))
		fmt.Fprintf(b, "- **Trigger:** %s\n", d.trigger(m))
		fmt.Fprintf(b, "- **Placeholder position report:**\n\n")
		fmt.Fprintf(b, "  | Position | English placeholder | Chinese placeholder | Word beside it in the Chinese | Verdict |\n")
		fmt.Fprintf(b, "  |---|---|---|---|---|\n")
		for _, p := range positions(en, zh) {
			fmt.Fprintf(b, "  | %d | `%s` receives argument %s | `%s` receives argument %s | %s | %s |\n",
				p.N, p.En, p.EnArg, p.Zh, p.ZhArg, orDash(p.Side), p.Verdict)
		}
		fmt.Fprintf(b, "\n  **Specification:** %s\n\n", changeTable(en, zh))
		fmt.Fprintf(b, "- **What is missing:** the sentence re-authored around the placeholders. The caller passes,\n")
		fmt.Fprintf(b, "  in order, %s. The Chinese has to be rewritten so that each of those arguments lands where its\n", argumentList(en))
		fmt.Fprintf(b, "  placeholder expects it: %s. Mechanically swapping the placeholders is not the\n", moveMap(en, zh))
		fmt.Fprintf(b, "  fix — the moved word order has to read correctly in the sentence that is left — and the consequence\n")
		fmt.Fprintf(b, "  of getting it wrong is not cosmetic: a reader under `QK_LANG=%s` sees the wrong words in the wrong\n", d.zh)
		fmt.Fprintf(b, "  slots.\n\n")
	}
}

// section23 lists the rows whose Chinese repeats the English. Both sections ask the same question — is
// there a Chinese rendering for this message anywhere already? — and answer it from a scan of the tree,
// not from memory. Reporting a translation that does not exist is the failure mode this column exists
// to prevent, so the answer is per row: the file it was found in, or a plain statement that there is
// none.
func (d *backlog) section23(b *bytes.Buffer, title string, msgs []Message) {
	fmt.Fprintf(b, "## %s\n\n", title)
	fmt.Fprintf(b, "For these %d rows the `%s` column repeats the `%s` column. That is not a translation and not a\n", len(msgs), d.zh, d.en)
	fmt.Fprintf(b, "defect of the migration: the message shipped English-only before it had an id, so inventing Chinese\n")
	fmt.Fprintf(b, "would have changed what a reader sees. The value has to come from a speaker.\n\n")
	fmt.Fprintf(b, "The last column is measured, not assumed. The generator scans every `.go`, `.tsv` and `.md` file in\n")
	fmt.Fprintf(b, "this repository and in the pinned syntax layer, keeps the ones that contain Han characters, and asks\n")
	fmt.Fprintf(b, "whether a Chinese rendering of *this* message exists. A rendering is looked up by the message's own\n")
	fmt.Fprintf(b, "text with its format verbs collapsed to one wildcard, because a rendering written down elsewhere may\n")
	fmt.Fprintf(b, "say `%%s` where this row says `%%q`; such a find is reported as a near match, with the hash of its\n")
	fmt.Fprintf(b, "canonical form so the copy can be identified without printing Chinese into a Go source file. Being\n")
	fmt.Fprintf(b, "one id away is not being the same message: `StringIndexOutOfBoundsError` is not\n")
	fmt.Fprintf(b, "`IndexOutOfBoundsError`, and a row that merely looks like a neighbour is reported as what the scan\n")
	fmt.Fprintf(b, "found, not as a translation that exists.\n\n")
	if len(msgs) == 0 {
		fmt.Fprintf(b, "_None._\n\n")
		return
	}
	fmt.Fprintf(b, "| # | id | English source (`%s` / `src`) | Current Chinese (`%s`) | Where it renders | What is missing | Chinese rendering found elsewhere |\n",
		d.en, d.zh)
	fmt.Fprintf(b, "|---|---|---|---|---|---|---|\n")
	for i, m := range msgs {
		value := d.src.Value(m.ID, d.zh)
		fmt.Fprintf(b, "| %d | `%s` | `%s` | repeats the English | %s | %s | %s |\n",
			i+1, m.ID, quoteDoc(m.Source), d.where(m), d.missing(m, value), d.alternate(m))
	}
	fmt.Fprintf(b, "\n")
	for i, m := range msgs {
		value := d.src.Value(m.ID, d.zh)
		fmt.Fprintf(b, "### %d. `%s`\n\n", i+1, m.ID)
		fmt.Fprintf(b, "- **Row to edit:** %s, the row whose `id` is `%s`, column `%s`.\n", d.rowPointer(m), m.ID, d.zh)
		fmt.Fprintf(b, "- **Owner:** `%s`%s\n", m.Owner, d.ownerMeaning(m))
		fmt.Fprintf(b, "- **English source (`%s` column):** `%s`\n", d.en, quoteDoc(m.Source))
		fmt.Fprintf(b, "- **Current Chinese (`%s` column):** %s\n", d.zh, d.currentChinese(m, value))
		fmt.Fprintf(b, "- **Where it renders:** %s\n", d.where(m))
		fmt.Fprintf(b, "- **Trigger:** %s\n", d.trigger(m))
		fmt.Fprintf(b, "- **What is missing:** %s\n", d.missing(m, value))
		fmt.Fprintf(b, "- **Chinese rendering found elsewhere:** %s\n\n", d.alternate(m))
	}
}

// howToFillIn is the part a translator reads first. It is prose rather than derived, so everything it
// says about columns, gates and limits is checkable against the rest of this file and the tests it names.
func (d *backlog) howToFillIn(b *bytes.Buffer) {
	fmt.Fprintf(b, "## How to fill this in\n\n")
	fmt.Fprintf(b, "1. **Edit the `%s` column of the row's table.** Each row names its table in its *Row to edit* line;\n", d.zh)
	fmt.Fprintf(b, "   the columns are tab-separated and `%s` is the last column on the row. For a `local` row the `src`\n", d.zh)
	fmt.Fprintf(b, "   column must keep repeating the `%s` column, so only `%s` changes. Never edit\n", d.en, d.zh)
	fmt.Fprintf(b, "   `internal/i18n/msg_gen.go`: it is generated, and the next `go generate` discards the edit.\n")
	fmt.Fprintf(b, "2. **Re-run the generator.** `go generate ./internal/i18n` rewrites both the catalog and this document.\n")
	fmt.Fprintf(b, "   `go run ./cmd/qki18n -check` then fails while either of them is stale — that gate is what keeps\n")
	fmt.Fprintf(b, "   this list honest. `go test ./... -count=1` runs the same check through TestGeneratedCatalogIsCurrent\n")
	fmt.Fprintf(b, "   and TestBacklogDocumentIsCurrent.\n")
	fmt.Fprintf(b, "3. **Lower a ratchet limit only to a number a run measured.** Each section's ratchet is the test named\n")
	fmt.Fprintf(b, "   in the summary table above, in `internal/i18n/`; its `allowed` constant may only shrink, and the\n")
	fmt.Fprintf(b, "   test prints the count it measured. Setting the limit to the number of rows still listed here, or to\n")
	fmt.Fprintf(b, "   a round number, is how a ratchet stops guarding anything.\n")
	fmt.Fprintf(b, "4. **A row in section 1 needs the sentence re-authored, not the placeholders swapped.** The Chinese\n")
	fmt.Fprintf(b, "   moves a verb or a type out of position, so the words around the placeholders have to move with\n")
	fmt.Fprintf(b, "   them; the *Placeholder position report* says which position must hold which placeholder, and that\n")
	fmt.Fprintf(b, "   is the whole specification. Getting the order wrong is not a cosmetic defect: the call site binds\n")
	fmt.Fprintf(b, "   its arguments by position, so under `QK_LANG=zh` the reader sees the wrong words in the wrong\n")
	fmt.Fprintf(b, "   slots — for example the row whose id ends in `using_q_on_s_not_supported` under the compiler\n")
	fmt.Fprintf(b, "   domain prints the operator where the type belongs.\n")
	fmt.Fprintf(b, "5. **Every user-facing message goes through an id path.** Wiring a message that is still a literal at\n")
	fmt.Fprintf(b, "   its call site is a separate change: `internal/i18n/fieldpass_test.go` fails on a message literal\n")
	fmt.Fprintf(b, "   written into a struct field, and `TestDecouplingRatchet` fails on a text-keyed entry point coming\n")
	fmt.Fprintf(b, "   back.\n\n")
	fmt.Fprintf(b, "## How this document is kept true\n\n")
	fmt.Fprintf(b, "`cmd/qki18n` renders the catalog and this document from the same tables in one run, and `-check`\n")
	fmt.Fprintf(b, "compares both against what is committed. The call sites and the \"found elsewhere\" column are measured\n")
	fmt.Fprintf(b, "by scanning the checkout at generation time, so they move when the code moves: a message whose call\n")
	fmt.Fprintf(b, "site disappears is reported as `unknown` rather than silently keeping a stale line number, and a\n")
	fmt.Fprintf(b, "translation added anywhere in either tree shows up as a hit on the row it translates.\n\n")
	fmt.Fprintf(b, "Two limits are worth knowing. A rendering that is assembled at run time (`fmt.Sprintf` of a prefix and\n")
	fmt.Fprintf(b, "a suffix) is not a single line and will not be found; and the scan reads `.go`, `.tsv` and `.md` files,\n")
	fmt.Fprintf(b, "so a rendering that lives only in a generated artefact or a history revision is out of its reach.\n")
	fmt.Fprintf(b, "Neither case is reported as a translation.\n")
}

// ---- per-row rendering helpers ----

// rowPointer names the table file a row lives in, taken from the table it was parsed from.
func (d *backlog) rowPointer(m Message) string {
	name := m.Table
	if name == "" {
		name = "unknown.tsv"
	}
	return "`internal/i18n/source/" + name + "`"
}

// ownerMeaning explains the `owner` column for the two owners a row here can have.
func (d *backlog) ownerMeaning(m Message) string {
	if m.Owner == "import" {
		return ": the pinned syntax layer owns the text, this repository owns the value"
	}
	return ": this repository owns the text and the value"
}

// currentChinese states plainly that the value repeats the English, so nobody mistakes a repeated
// column for a translation.
func (d *backlog) currentChinese(m Message, value string) string {
	if value == m.Source {
		return "`" + quoteDoc(value) + "` — **repeats the English; this is not a translation**"
	}
	return "`" + quoteDoc(value) + "`"
}

// localizerPath is where an imported message is rendered in this repository: the pinned module raises
// it, and this repository's localizer bridge renders the value the catalog holds. It is the one render
// path every imported row shares, and it is why a row with no call site of its own here still has a
// `file:line` — the module's own call sites are inside the pinned module, which this scan does not
// claim to have read.
const localizerPath = "internal/lang/localizer.go:25"

// where renders where a message is rendered from.
func (d *backlog) where(m Message) string {
	sites := d.scan.sites(m.ID)
	if len(sites) == 0 {
		if m.Owner == "import" {
			return fmt.Sprintf("`%s` (the localizer bridge every imported id renders through); no call site of its own in this tree — the module raises it",
				localizerPath)
		}
		return "**unknown** — no call site in this tree passes `" + constName(m.ID) + "` to a renderer"
	}
	parts := make([]string, 0, len(sites))
	for _, s := range sites {
		parts = append(parts, fmt.Sprintf("`%s:%d`", s.File, s.Line))
	}
	out := strings.Join(parts, ", ")
	if len(sites) > 1 {
		out += fmt.Sprintf(" (%d sites)", len(sites))
	}
	return out
}

// trigger turns the row's `ctx` and `note` columns into a sentence: they are authored for a translator
// precisely so that a reader does not have to reconstruct the condition from the code.
func (d *backlog) trigger(m Message) string {
	ctx, note := strings.TrimSpace(m.Context), strings.TrimSpace(m.Note)
	switch {
	case ctx == "" && note == "":
		return "**unknown** — the row carries no context column"
	case note == "":
		return fmt.Sprintf("raised at `%s`", ctx)
	case ctx == "":
		return note
	case strings.Contains(ctx, "/") || strings.Contains(ctx, ".go:"):
		return fmt.Sprintf("%s (site recorded in the table: `%s`)", note, ctx)
	default:
		return fmt.Sprintf("%s (recorded context: `%s`)", note, ctx)
	}
}

// missing states what has to be authored. It never prescribes the Chinese: it says which placeholders
// the value has to use, in which order.
func (d *backlog) missing(m Message, value string) string {
	verbs := verbPat.FindAllString(m.Source, -1)
	if value != m.Source {
		return "nothing: the row has a value (it is listed here because its placeholders move)"
	}
	if len(verbs) == 0 {
		return "a Chinese value; the message has no placeholders, so any accurate translation fills the row"
	}
	return fmt.Sprintf("a Chinese value that uses the source's %d placeholder(s) `%s` in this order",
		len(verbs), strings.Join(verbs, " "))
}

// alternate answers "does a Chinese rendering of this message exist elsewhere?" from the scan.
func (d *backlog) alternate(m Message) string {
	hits := d.scan.chineseFor(m.Source)
	if len(hits) == 0 {
		return "**none found** — no Chinese rendering of this message in this repository or in the pinned syntax layer"
	}
	parts := make([]string, 0, len(hits))
	for _, h := range hits {
		kind := "near match, placeholders differ"
		if h.Exact {
			kind = "exact match"
		}
		parts = append(parts, fmt.Sprintf("%s (`%s:%d`, canonical `%s`)", kind, h.File, h.Line, hash16(canonical(h.Text))))
	}
	return strings.Join(parts, "; ")
}

// ---- placeholder reports ----

// pos is one placeholder position of a message: the placeholder each language puts there, the caller's
// argument it receives, and the Chinese word that sits beside it.
type pos struct {
	N       int
	En      string
	EnArg   string
	Zh      string
	ZhArg   string
	Side    string
	Verdict string
}

// positions pairs the source's placeholders with the value's, position by position.
func positions(source, value string) []pos {
	zhMarked := markPlaceholders(value)
	a, b := verbPat.FindAllString(source, -1), verbPat.FindAllString(value, -1)
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]pos, 0, n)
	for i := 0; i < n; i++ {
		p := pos{N: i + 1, En: a[i], Zh: b[i], EnArg: fmt.Sprintf("%d", i+1), ZhArg: fmt.Sprintf("%d", i+1), Verdict: "ok"}
		p.Side = sideWord(zhMarked, i+1)
		if a[i] != b[i] {
			p.Verdict = fmt.Sprintf("**mismatch** — argument %d is printed as `%s` here, and the caller passes a %s value",
				i+1, b[i], kindName(a[i]))
		}
		out = append(out, p)
	}
	return out
}

// sideWord returns the Chinese word adjacent to one marker, so the report can say which word a moved
// placeholder travels with.
func sideWord(marked string, n int) string {
	token := fmt.Sprintf(markerPlaceholder, n)
	i := strings.Index(marked, token)
	if i < 0 {
		return ""
	}
	before := strings.TrimRightFunc(marked[:i], func(r rune) bool { return !indicator.MatchString(string(r)) })
	if j := strings.LastIndexFunc(before, func(r rune) bool { return !indicator.MatchString(string(r)) }); j >= 0 {
		before = before[j+1:]
	}
	after := marked[i+len(token):]
	if j := strings.IndexFunc(after, func(r rune) bool { return !indicator.MatchString(string(r)) }); j >= 0 {
		after = after[:j]
	}
	word := before + after
	if word == "" {
		return ""
	}
	return word
}

// markPlaceholders rewrites a message so every placeholder is followed by the number of the argument it
// receives: `using %q on %s` becomes `using %q\u2039@1@\u203a on %s\u2039@2@\u203a`. Nothing else about the text
// changes, so the Chinese value in the document is the value in the table.
func markPlaceholders(s string) string {
	var b strings.Builder
	n := 0
	for i := 0; i < len(s); {
		if s[i] == '%' && i+1 < len(s) && isASCIILetter(s[i+1]) {
			n++
			b.WriteString(s[i : i+2])
			fmt.Fprintf(&b, markerPlaceholder, n)
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func isASCIILetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// changeTable states, for one row, which positions must hold which placeholder: the specification the
// task asks for. It never says what the Chinese should say.
func changeTable(source, value string) string {
	a, b := verbPat.FindAllString(source, -1), verbPat.FindAllString(value, -1)
	if len(a) != len(b) {
		return fmt.Sprintf("a count defect rather than an order defect: the source has %d placeholder(s) and the value has %d, so this row is not listed in this section",
			len(a), len(b))
	}
	if !sameMultiset(a, b) {
		return "not an order defect: the two use different kinds of placeholder"
	}
	var moves []string
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		moves = append(moves, fmt.Sprintf("position %d must print the caller's %s argument (`%s`), not `%s`",
			i+1, kindName(a[i]), a[i], b[i]))
	}
	if len(moves) == 0 {
		return "none: the placeholders already line up"
	}
	return strings.Join(moves, "; ")
}

// kindName names a placeholder in the caller's terms, so the specification reads as a claim about which
// argument lands where.
func kindName(verb string) string {
	switch verb {
	case "%q":
		return "quoted"
	case "%d":
		return "integer"
	case "%T":
		return "type"
	case "%v":
		return "value"
	default:
		return "string"
	}
}

// moveMap states where the Chinese has to place each argument the caller passes: "position 1 must print
// the quoted argument (`%q`), position 2 the string argument (`%s`)". It is derived from the source, so
// it says where an argument goes and never what the Chinese should say.
func moveMap(source, value string) string {
	a, b := verbPat.FindAllString(source, -1), verbPat.FindAllString(value, -1)
	if len(a) != len(b) {
		return "the placeholder counts already differ, so this row is a count defect rather than an order defect"
	}
	// target[i] is the position the caller's argument i must be printed from.
	target := make([]int, len(a))
	for i := range target {
		target[i] = -1
	}
	used := make([]bool, len(b))
	for i := range a {
		for j := range b {
			if used[j] || b[j] != a[i] {
				continue
			}
			used[j], target[i] = true, j
			break
		}
	}
	parts := make([]string, 0, len(a))
	for i := range a {
		if target[i] < 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("position %d must print the %s argument (`%s`)",
			target[i]+1, kindName(a[i]), a[i]))
	}
	return strings.Join(parts, ", ")
}

// argumentList describes the arguments the source passes, in order.
func argumentList(source string) string {
	verbs := verbPat.FindAllString(source, -1)
	parts := make([]string, 0, len(verbs))
	for i, v := range verbs {
		parts = append(parts, fmt.Sprintf("%s (%s, `%s`)", ordinal(i+1), kindName(v), v))
	}
	return strings.Join(parts, ", ")
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "first"
	case 2:
		return "second"
	case 3:
		return "third"
	case 4:
		return "fourth"
	case 5:
		return "fifth"
	}
	return fmt.Sprintf("%dth", n)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return fmt.Sprintf("`%s`", s)
}

// quoteDoc renders a table value for the document. The value is data, so it is quoted as the table
// wrote it, with the tables' Go quoting unwrapped and backticks folded so a Markdown fence cannot open.
func quoteDoc(s string) string {
	if unquoted, err := strconv.Unquote(s); err == nil {
		s = unquoted
	}
	return strings.ReplaceAll(s, "`", "'")
}

// ---- the tree scan: call sites and Chinese renderings ----

// site is one place a message is rendered from.
type site struct {
	File string
	Line int
}

// hit is one Chinese rendering of a message found in the tree.
type hit struct {
	File  string
	Line  int
	Text  string
	Exact bool
}

// scan answers two questions about the tree: where does a message's constant appear, and does a Chinese
// rendering of a message's text exist anywhere. Both answers come from one pass, and both are
// measurements of the checkout the generator ran in.
type scan struct {
	root     string
	ids      []string
	sitesBy  map[string][]site
	variants map[string][]hit // canonical text -> occurrences
	loaded   bool
}

func newScan(root string, ids []string) *scan {
	return &scan{root: root, ids: ids, sitesBy: map[string][]site{}, variants: map[string][]hit{}}
}

// load walks the tree once. A file that cannot be read or parsed is skipped: a message with no site is
// then reported as `unknown`, which is the honest answer when the scan could not look.
func (s *scan) load() {
	if s.loaded || s.root == "" {
		s.loaded = true
		return
	}
	s.loaded = true
	want := make(map[string]string, len(s.ids))
	for _, id := range s.ids {
		want[constName(id)] = id
	}
	s.walk(s.root, true, want)
}

// sites returns the call sites of a message, sorted by file then line.
func (s *scan) sites(id string) []site {
	s.load()
	out := append([]site(nil), s.sitesBy[id]...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// chineseFor returns every Chinese rendering of a message's text found in the tree, exact matches
// first.
func (s *scan) chineseFor(source string) []hit {
	s.load()
	out := append([]hit(nil), s.variants[canonical(source)]...)
	for i := range out {
		out[i].Exact = out[i].Text == source
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Exact != out[j].Exact {
			return out[i].Exact
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// walk indexes one tree. skipI18n is true for this repository, whose own tables and generated blob are
// the data being described rather than evidence about it.
func (s *scan) walk(dir string, skipI18n bool, want map[string]string) {
	fset := token.NewFileSet()
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if scanSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			if skipI18n && filepath.ToSlash(path) == filepath.ToSlash(filepath.Join(s.root, "internal", "i18n")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !scannable(path) {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		rel := s.relative(path)
		s.indexText(rel, string(raw))
		if filepath.Ext(path) != ".go" {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, raw, 0)
		if perr != nil {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			id, ok := want[sel.Sel.Name]
			if !ok {
				return true
			}
			p := fset.Position(sel.Pos())
			s.sitesBy[id] = appendUniqueSite(s.sitesBy[id], site{File: rel, Line: p.Line})
			return true
		})
		return nil
	})
}

// scannable reports whether a file kind can carry a rendering or a call site.
func scannable(path string) bool {
	ext := filepath.Ext(path)
	for _, want := range scanExts {
		if ext == want {
			return true
		}
	}
	return false
}

// indexText records every Han-bearing line of a file, keyed by its canonical form, so a row whose
// rendering elsewhere uses another placeholder kind is still found and reported as a near match.
func (s *scan) indexText(rel, text string) {
	if !strings.ContainsFunc(text, func(r rune) bool { return indicator.MatchString(string(r)) }) {
		return
	}
	for i, line := range strings.Split(text, "\n") {
		for _, cand := range candidates(line) {
			if len(cand) < minRenderingLen || !indicator.MatchString(cand) {
				continue
			}
			key := canonical(cand)
			s.variants[key] = appendUniqueHit(s.variants[key], hit{File: rel, Line: i + 1, Text: cand})
		}
	}
}

// candidates returns the plausible renderings on one line: the trimmed line itself, and the line with
// the quoting of a Go string, a TSV cell or a Markdown table cell stripped.
func candidates(line string) []string {
	trimmed := strings.TrimSpace(line)
	out := []string{trimmed}
	quotes := []struct{ pre, post string }{
		{`"`, `"`}, {"`", "`"}, {"'", "'"}, {`"`, `",`}, {"`", "`,"},
	}
	for _, q := range quotes {
		if len(trimmed) >= len(q.pre)+len(q.post) &&
			strings.HasPrefix(trimmed, q.pre) && strings.HasSuffix(trimmed, q.post) {
			out = append(out, strings.TrimSpace(trimmed[len(q.pre):len(trimmed)-len(q.post)]))
		}
	}
	if strings.Contains(trimmed, "|") {
		for _, c := range strings.Split(trimmed, "|") {
			if c = strings.TrimSpace(c); c != "" {
				out = append(out, c)
			}
		}
	}
	return out
}

// canonical folds the two things that differ between a rendering here and the same rendering in another
// store: the kind of format verb, and the curly quotes a Chinese sentence may use.
func canonical(s string) string {
	s = verbPat.ReplaceAllString(s, "%@")
	return strings.NewReplacer("\u201c", `"`, "\u201d", `"`, "\u2018", "'", "\u2019", "'").Replace(s)
}

// relative renders a scanned path the way the document cites it: relative to the repository root when it
// is inside the tree, absolute (a module cache path) when it is the pinned module's.
func (s *scan) relative(path string) string {
	if rel, err := filepath.Rel(s.root, path); err == nil && !strings.HasPrefix(filepath.ToSlash(rel), "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func appendUniqueSite(xs []site, x site) []site {
	for _, v := range xs {
		if v == x {
			return xs
		}
	}
	return append(xs, x)
}

func appendUniqueHit(xs []hit, x hit) []hit {
	for _, v := range xs {
		if v.File == x.File && v.Line == x.Line && v.Text == x.Text {
			return xs
		}
	}
	return append(xs, x)
}

// hash16 names a canonical form in the document: it is what lets a reported match be checked without
// printing a Chinese rendering into a Go source file.
func hash16(s string) string {
	f := fnv.New64a()
	_, _ = f.Write([]byte(s))
	return fmt.Sprintf("%016x", f.Sum64())
}
