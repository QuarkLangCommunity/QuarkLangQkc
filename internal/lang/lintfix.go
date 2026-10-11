package lang

// qkcheck corrections: what a fixable diagnostic's corrected form is, and how it is applied.
//
// The design, the inventory of QK101-QK115 and the applicability gates are in cmd/qkcheck/README.md.
// In one sentence: a fix exists only where the corrected form is determined by the diagnostic alone,
// cannot change what the program does, and is idempotent.
//
// The front end carries a start position only (line and column), so a fix is expressed against the
// source text: it deletes a whole line, or replaces the tail of a line that one statement occupies.
// Rewriting inside a line is what makes CRLF checkouts safe - a line separator is never rewritten,
// only removed together with the line it terminates.

import (
	"fmt"
	"sort"
	"strings"
)

// Fix is one mechanical correction of a diagnostic: replace Text at [From,To) of Line.
//
// A whole-line span (From==0, To==len(line)) with an empty Text and Delete set removes the line
// together with its terminator. Deleting a line never touches any other line's bytes, so a file's own
// line endings and every untouched line number survive the rewrite.
type Fix struct {
	Line     int    `json:"line"`
	From     int    `json:"from"`
	To       int    `json:"to"`
	Text     string `json:"text"`
	Delete   bool   `json:"delete,omitempty"`
	Why      string `json:"why,omitempty"`      // what makes this correction right, in a few words
	Original string `json:"original,omitempty"` // the line as it is now, for the before/after report
}

// Replacement is the line as it becomes, or "" when the fix deletes the line. It is what a report
// shows as the corrected form, and what applying the fix writes.
func (f *Fix) Replacement() string {
	if f.Delete {
		return ""
	}
	return f.Original[:f.From] + f.Text + f.Original[f.To:]
}

// HasFix reports whether the diagnostic carries a mechanical correction.
func (d Diag) HasFix() bool { return d.Fix != nil }

// Brief renders the correction in one line, for the default report.
func (f *Fix) Brief() string {
	if f == nil {
		return ""
	}
	if f.Delete {
		return fmt.Sprintf("delete line %d (%s)", f.Line, f.Why)
	}
	return fmt.Sprintf("line %d: %s (%s)", f.Line, f.Text, f.Why)
}

// FixPlan is what applying corrections to one file's diagnostics would do.
type FixPlan struct {
	Fixes []AppliedFix // one per diagnostic that carries a correction, in line order
	Lines []LineChange // the rewrites to make, one per line, in line order
	Skip  []SkipReason // corrections withheld, and why (never silently dropped)
}

// AppliedFix binds one diagnostic index to the correction proposed for it.
type AppliedFix struct {
	DiagIndex int
	Code      string
	Fix       Fix
}

// LineChange is one line of the file after the plan is applied.
type LineChange struct {
	Line int
	Old  string // the line without its terminator, as it is now
	New  string // the line without its terminator, as it becomes
}

// SkipReason records a diagnostic whose correction was withheld, and why.
type SkipReason struct {
	Code   string
	Line   int
	Reason string
}

// Pending reports how many lines the plan would rewrite.
func (p *FixPlan) Pending() int { return len(p.Lines) }

// Apply writes the plan's rewrites into src, preserving every line separator of the original.
func (p *FixPlan) Apply(src string) string {
	if p == nil || len(p.Lines) == 0 {
		return src
	}
	sf := splitSource(src)
	for _, c := range p.Lines {
		if c.Line < 1 || c.Line > len(sf.lines) {
			continue
		}
		sf.lines[c.Line-1].text = c.New
	}
	return sf.join()
}

// ---------- the source, split so that each line keeps its own terminator ----------

// srcLine is one line with the terminator it was written with, so a rewrite can never normalise a
// CRLF file to LF or the other way round.
type srcLine struct {
	text string // without the terminator
	sep  string // "\n", "\r\n" or "" for a last line that does not end in one
}

type srcLines struct{ lines []srcLine }

func splitSource(src string) srcLines {
	var out srcLines
	for len(src) > 0 {
		i := strings.IndexByte(src, '\n')
		if i < 0 {
			out.lines = append(out.lines, srcLine{text: src})
			break
		}
		text, sep := src[:i], "\n"
		if strings.HasSuffix(text, "\r") {
			text, sep = text[:len(text)-1], "\r\n"
		}
		out.lines = append(out.lines, srcLine{text: text, sep: sep})
		src = src[i+1:]
	}
	return out
}

func (s srcLines) join() string {
	var b strings.Builder
	for _, l := range s.lines {
		b.WriteString(l.text)
		b.WriteString(l.sep)
	}
	return b.String()
}

// ---------- line-shape gates ----------

// stmtOnOwnLine reports whether nothing but a semicolon and a comment may follow the statement that
// starts at col: leading whitespace, then the statement, then at most `;` and a trailing comment. This
// is the gate that makes a whole-line rewrite safe - the statement shares its line with nothing else.
//
// The text between col and the semicolon is the statement, which this gate cannot check (the front end
// carries no end position); what it does check is that everything after the statement's terminator is
// whitespace or a comment, and that nothing precedes col but indentation. Callers pair it with their
// own knowledge of the statement's text, so a mismatch costs the fix rather than the file.
func stmtOnOwnLine(line string, col int) bool {
	from := col - 1
	if from < 0 || from > len(line) {
		return false
	}
	if strings.TrimSpace(line[:from]) != "" {
		return false
	}
	return onlyCommentAfterStmt(strings.TrimRight(line[from:], " \t"))
}

// onlyCommentAfterStmt reports whether the tail of a line, from the statement to its end, is a
// statement plus an optional `;` and an optional trailing comment - and nothing else.
func onlyCommentAfterStmt(s string) bool {
	if s == "" {
		return true
	}
	if c := commentStart(s); c >= 0 {
		if c == 0 {
			return true // the line holds a comment and nothing before it
		}
		s = strings.TrimRight(s[:c], " \t")
		s = strings.TrimSuffix(s, ";")
		return strings.TrimSpace(s) != "" && !strings.HasSuffix(s, ";")
	}
	s = strings.TrimSuffix(strings.TrimRight(s, " \t"), ";")
	return !strings.ContainsAny(s, ";{}")
}

// commentStart is the offset where a comment begins in s, or -1. A `//`, `#` or `/*` inside a string
// literal is not a comment, which matters because a statement's text is passed through here.
func commentStart(s string) int {
	inStr, raw, esc := false, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case esc:
			esc = false
		case c == '\\':
			esc = true
		case raw:
			if c == '`' {
				raw = false
			}
		case inStr:
			if c == '"' {
				inStr = false
			}
		case c == '`':
			raw = true
		case c == '"':
			inStr = true
		case c == '/' && i+1 < len(s) && (s[i+1] == '/' || s[i+1] == '*'):
			return i
		case c == '#':
			return i
		}
	}
	return -1
}

func isComment(s string) bool {
	return strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") || strings.HasPrefix(s, "#")
}

// importAloneOnLine reports whether the line holds exactly this one import and nothing else: the
// keyword, the path, a `;` and at most a trailing comment. An import that shares its line with another
// import or with code is not corrected - removing it would remove the rest of the line too.
func importAloneOnLine(line string, path string) bool {
	rest := strings.TrimSpace(line)
	if !strings.HasPrefix(rest, "import") {
		return false
	}
	rest = rest[len("import"):]
	if rest == "" || (rest[0] != ' ' && rest[0] != '\t' && rest[0] != '"') {
		return false // "imports" and other identifiers that merely start with the keyword
	}
	rest = strings.TrimLeft(rest, " \t")
	quoted, ok := cutImportPath(rest)
	if !ok || unquoteKq(quoted) != path {
		return false // a different path on this line: not this diagnostic's import
	}
	rest = strings.TrimLeft(rest[len(quoted):], " \t")
	if !strings.HasPrefix(rest, ";") {
		return false
	}
	rest = strings.TrimLeft(rest[1:], " \t")
	return rest == "" || isComment(rest)
}

// cutImportPath returns the quoted or bare import path at the head of s.
func cutImportPath(s string) (string, bool) {
	if strings.HasPrefix(s, `"`) {
		for i := 1; i < len(s); i++ {
			if s[i] == '\\' {
				i++
				continue
			}
			if s[i] == '"' {
				return s[:i+1], true
			}
		}
		return "", false
	}
	for i := 0; i < len(s); i++ {
		if !isIdentByte(s[i]) {
			if i == 0 {
				return "", false
			}
			return s[:i], true
		}
	}
	return s, s != ""
}

// isIdentByte reports whether c can appear in a bare import path.
func isIdentByte(c byte) bool {
	return c == '_' || c == '.' || c == '/' || c == '-' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// unquoteKq decodes a quoted import path, or returns a bare path unchanged.
func unquoteKq(s string) string {
	if len(s) < 2 || s[0] != '"' {
		return s
	}
	var b strings.Builder
	for i := 1; i < len(s)-1; i++ {
		if s[i] == '\\' && i+1 < len(s)-1 {
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(s[i])
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// exprStart is the position of the leftmost token of an expression: the head of a member or index
// chain rather than its operator. posOf is the position a diagnostic wants (qkparser records an index
// expression at its "[" and an assignment at its "="), while a whole-line rewrite has to start where
// the statement starts.
func exprStart(e Expr) Pos {
	switch t := e.(type) {
	case nil:
		return Pos{}
	case *MemberExpr:
		return exprStart(t.X)
	case *IndexExpr:
		return exprStart(t.X)
	}
	return posOf(e)
}

// ---------- purity: may this expression be dropped? ----------

// pureExpr reports whether evaluating e can do nothing but read. Reads of a variable, a field or a
// list element are allowed - dropping a read is observable only through the use counts the linter
// keeps, which the liveness gate covers; calls, assignments, ++ and delete are not.
func pureExpr(e Expr) bool {
	switch t := e.(type) {
	case nil:
		return true
	case *Ident, *IntLit, *FloatLit, *StrLit, *BoolLit, *NullLit:
		return true
	case *ListLit:
		for _, it := range t.Items {
			if !pureExpr(it) {
				return false
			}
		}
		return true
	case *StructLit:
		for _, f := range t.Fields {
			if !pureExpr(f.X) {
				return false
			}
		}
		return true
	case *MemberExpr:
		return pureExpr(t.X)
	case *IndexExpr:
		return pureExpr(t.X) && pureExpr(t.Idx)
	case *UnOp:
		return pureExpr(t.X)
	case *BinOp:
		return pureExpr(t.L) && pureExpr(t.R)
	}
	return false // *CallExpr, *ScopeCall, *NewExpr, *IncExpr and whatever is added later
}

// rootIdent names the variable an expression's reads depend on, and counts those reads: the count
// that decides whether a rewrite may drop them.
func rootIdent(e Expr, reads *int) string {
	switch t := e.(type) {
	case *Ident:
		*reads++
		return t.Name
	case *MemberExpr:
		return rootIdent(t.X, reads)
	case *IndexExpr:
		n := rootIdent(t.X, reads)
		inner := 0
		if id := rootIdent(t.Idx, &inner); id != "" {
			return id // an index that is itself a variable read: keep the leftmost root
		}
		return n
	}
	return ""
}

// ---------- attaching corrections to diagnostics ----------

// fixCandidate is a correction found during the lint walk. It is decided at the end of that walk
// because two gates (liveness of a read, and a second correction on the same line) need the whole
// file.
type fixCandidate struct {
	at     Pos    // the diagnostic's own position, which is how the candidate finds its diagnostic
	pos    Pos    // the start of the statement the correction rewrites (differs from at for QK114)
	code   string // the diagnostic code whose correction this is
	line   int    // the line the correction rewrites (the line the fix deletes or replaces)
	why    string // what makes the correction right, shown in the report
	need   string // variable whose liveness gates this candidate ("" = no liveness gate)
	needN  int    // how many reads the rewrite would drop
	reason string // why the correction is withheld when the shape gate fails
	build  func(sf srcLines) (Fix, bool)
}

// attachFixes decides, for every diagnostic, whether it carries a mechanical correction, and returns
// the diagnostics in their final (position-sorted) order. A diagnostic whose correction is withheld
// keeps its message and gains no fix: nothing is dropped.
//
// A candidate finds its diagnostic by (position, code) rather than by index: one position can carry
// two codes (a self-assignment between two writes is a QK108 and a QK113 at once), and the order in
// which diagnostics and candidates were recorded is not the order they are reported in.
func attachFixes(diags []Diag, cands []fixCandidate, sf srcLines, reads map[string]int) ([]Diag, *FixPlan) {
	plan := &FixPlan{}
	type key struct {
		pos  Pos
		code string
	}
	byKey := map[key][]int{}
	for i, d := range diags {
		k := key{d.Pos, d.Code}
		byKey[k] = append(byKey[k], i)
	}
	for _, c := range cands {
		k := key{c.at, c.code}
		list := byKey[k]
		if len(list) == 0 {
			continue // the diagnostic this correction belongs to is not in this result (e.g. filtered out)
		}
		idx := list[0]
		byKey[k] = list[1:] // one candidate per diagnostic, in registration order
		if c.need != "" && reads[c.need] <= c.needN {
			plan.Skip = append(plan.Skip, SkipReason{Code: c.code, Line: c.line, Reason: fmt.Sprintf(
				"rewriting it would drop the last read of %s and leave a new QK101 finding behind", c.need)})
			continue
		}
		if c.build == nil {
			plan.Skip = append(plan.Skip, SkipReason{Code: c.code, Line: c.line, Reason: c.reason})
			continue // decisively not applicable (a declaration dead store, an impure right-hand side)
		}
		f, ok := c.build(sf)
		if !ok {
			plan.Skip = append(plan.Skip, SkipReason{Code: c.code, Line: c.line, Reason: c.reason})
			continue
		}
		f.Why = c.why
		f.Original = lineTextAt(sf, c.line)
		diags[idx].Fix = &Fix{Line: f.Line, From: f.From, To: f.To, Text: f.Text, Delete: f.Delete, Why: f.Why, Original: f.Original}
		plan.Fixes = append(plan.Fixes, AppliedFix{DiagIndex: idx, Code: c.code, Fix: f})
	}
	plan.merge()
	sort.SliceStable(diags, func(i, j int) bool {
		a, b := diags[i], diags[j]
		if a.Pos.Line != b.Pos.Line {
			return a.Pos.Line < b.Pos.Line
		}
		if a.Pos.Col != b.Pos.Col {
			return a.Pos.Col < b.Pos.Col
		}
		return a.Code < b.Code
	})
	return diags, plan
}

// lineTextAt returns the line without its terminator, or "" when the line does not exist.
func lineTextAt(sf srcLines, line int) string {
	if line < 1 || line > len(sf.lines) {
		return ""
	}
	return sf.lines[line-1].text
}

// merge turns the per-diagnostic rewrites into one rewrite per line, in line order. Two diagnostics
// on one line are either the same correction (a self-assignment is also a dead store) or a conflict;
// a conflict keeps the higher-priority correction and reports the other as withheld.
func (p *FixPlan) merge() {
	sort.SliceStable(p.Fixes, func(i, j int) bool {
		if p.Fixes[i].Fix.Line != p.Fixes[j].Fix.Line {
			return p.Fixes[i].Fix.Line < p.Fixes[j].Fix.Line
		}
		return fixPriority(p.Fixes[i].Code) < fixPriority(p.Fixes[j].Code)
	})
	for _, af := range p.Fixes {
		if n := len(p.Lines); n > 0 {
			if prev := p.Lines[n-1]; prev.Line == af.Fix.Line {
				if prev.Old != af.Fix.Original || prev.New != af.Fix.Replacement() {
					p.Skip = append(p.Skip, SkipReason{Code: af.Code, Line: af.Fix.Line, Reason: fmt.Sprintf(
						"another fix rewrites the same line instead (%q becomes %q)", prev.Old, prev.New)})
				}
				continue // two diagnostics, one correction
			}
		}
		p.Lines = append(p.Lines, LineChange{Line: af.Fix.Line, Old: af.Fix.Original, New: af.Fix.Replacement()})
	}
}

// fixPriority orders the corrections that can land on one line; the lowest is applied.
func fixPriority(code string) int {
	switch code {
	case CodeDeadStore:
		return 0
	case CodeSelfAssign:
		return 1
	case CodeUnusedImport:
		return 2
	default:
		return 3
	}
}

// ---------- per-code applicability ----------

// selfAssignFix corrects QK108 by deleting the statement: `x = x;` has no effect.
// A self-assignment shares its diagnostic position with a QK113 dead store exactly when it is the
// statement that carried the pending write, and both delete the same line, so the two agree.
func selfAssignFix(at, start Pos) fixCandidate {
	return fixCandidate{
		at:   at,    // the diagnostic reports the assignment operator
		pos:  start, // the rewrite starts where the statement starts
		code: CodeSelfAssign,
		line: start.Line,
		why:  "a self-assignment has no effect",
		reason: "the statement shares its line with other code, so deleting the line would remove " +
			"something else (the linter only rewrites a whole line)",
		build: func(sf srcLines) (Fix, bool) {
			return deleteLineIf(sf, start.Line, start.Col, stmtOnOwnLine)
		},
	}
}

// deadStoreFix corrects QK113 by deleting the earlier assignment whose value nobody reads.
// It is withheld for a declaration initializer (deleting it would delete the declaration) and for a
// right-hand side that can call something (deleting it would drop the call).
func deadStoreFix(d Diag, prev *pendingWrite) fixCandidate {
	c := fixCandidate{
		at:   d.Pos,
		pos:  prev.Start,
		code: CodeDeadStore,
		line: prev.Start.Line,
		why:  "the value it computes is overwritten before any read",
	}
	if prev.Decl {
		c.reason = "the overwritten write is the declaration initializer, and deleting it would delete the declaration itself"
		return c
	}
	if !pureExpr(prev.RHS) {
		c.reason = "the overwritten right-hand side calls or writes something, so deleting the assignment would drop that effect"
		return c
	}
	c.reason = "the assignment shares its line with other code, so deleting the line would remove something else"
	c.build = func(sf srcLines) (Fix, bool) {
		return deleteLineIf(sf, prev.Start.Line, prev.Start.Col, stmtOnOwnLine)
	}
	return c
}

// unusedImportFix corrects QK111 by deleting the import line, which is safe because no symbol of the
// imported file is used. Withheld when the line holds more than that one import.
func unusedImportFix(pos Pos, path string) fixCandidate {
	return fixCandidate{
		at:   pos,
		pos:  pos,
		code: CodeUnusedImport,
		line: pos.Line,
		why:  "no symbol of the imported file is used",
		reason: "the line holds more than this one import (or something else follows it), and the " +
			"linter only rewrites a whole line",
		build: func(sf srcLines) (Fix, bool) {
			l := lineTextAt(sf, pos.Line)
			if !importAloneOnLine(l, path) {
				return Fix{}, false
			}
			return Fix{Line: pos.Line, From: 0, To: len(l), Text: "", Delete: true}, true
		},
	}
}

// selfCompareFix corrects QK114 by replacing the standalone comparison with the constant it always
// yields: `x == x` is true and `x != x` is false for every value x can hold, including NaN, because
// both sides read the same storage. It is withheld unless the comparison is a whole statement (so
// nothing but the comparison is lost) and the operand is a pure read that the file reads elsewhere
// (so the rewrite cannot turn the program into a new QK101 finding).
func selfCompareFix(d Diag, op string, operand Expr) fixCandidate {
	var reads int
	root := rootIdent(operand, &reads)
	c := fixCandidate{
		at:   d.Pos,              // the diagnostic reports the operator ...
		pos:  exprStart(operand), // ... while the rewrite starts at the operand
		code: CodeSelfCompare,
		line: exprStart(operand).Line,
		why:  "a value compared with itself is a constant",
	}
	if !pureExpr(operand) {
		c.reason = "the operand calls or writes something, so evaluating it could have an effect the constant would drop"
		return c
	}
	c.need, c.needN = root, reads
	if root == "" {
		c.reason = "the operand is not a plain read, so its value cannot be shown to be constant here"
		return c
	}
	c.reason = "the comparison shares its line with other code, so replacing the line would remove something else"
	start := exprStart(operand)
	c.build = func(sf srcLines) (Fix, bool) {
		line := lineTextAt(sf, start.Line)
		if !stmtOnOwnLine(line, start.Col) {
			return Fix{}, false
		}
		// Replace the comparison only, and keep the terminator and any trailing comment: the fix
		// changes what the expression evaluates to, not the shape of the statement around it.
		from, to := start.Col-1, stmtEnd(line, start.Col-1)
		if to < from {
			return Fix{}, false
		}
		text := "false"
		if op == "==" {
			text = "true"
		}
		return Fix{Line: start.Line, From: from, To: to, Text: text}, true
	}
	return c
}

// stmtEnd is the offset where the statement starting at from ends (exclusive): at its `;` or at the
// comment that follows it, with the whitespace before either trimmed. Everything from there on belongs
// to the line rather than to the statement, so a rewrite must leave it alone.
func stmtEnd(line string, from int) int {
	if from < 0 || from > len(line) {
		return -1
	}
	end := len(line)
	if c := commentStart(line[from:]); c >= 0 {
		end = from + c
	}
	if semi := strings.IndexByte(line[from:end], ';'); semi >= 0 {
		end = from + semi
	}
	for end > from && (line[end-1] == ' ' || line[end-1] == '\t') {
		end--
	}
	return end
}

// deleteLineIf builds a whole-line deletion for a statement that passes the shape gate. The span
// covers the line's text; the terminator goes with it, which is what keeps the file's own line
// endings intact.
func deleteLineIf(sf srcLines, line, col int, ok func(string, int) bool) (Fix, bool) {
	l := lineTextAt(sf, line)
	if !ok(l, col) {
		return Fix{}, false
	}
	return Fix{Line: line, From: 0, To: len(l), Text: "", Delete: true}, true
}
