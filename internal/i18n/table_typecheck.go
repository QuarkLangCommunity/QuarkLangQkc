package i18n

// The typecheck domain moved to source/typecheck.tsv: every row is a generated ID now, and the checker's
// call sites render through (*checker).errfID. The table is kept, empty, because i18n.Load assembles the
// domains from this list and the file is deleted at step 5 with the rest of the text-keyed machinery.
//
// The one row that was not a message went with it: a `"%s"` pass-through, used by the call sites that
// hand `errf` an already-rendered message value. It said nothing to a translator and could not be
// translated, so it was deleted rather than migrated, and the pass-through is now named
// (*checker).errfRendered.
var typecheckTable = map[string]string{}
