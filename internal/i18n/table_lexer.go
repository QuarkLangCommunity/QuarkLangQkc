package i18n

// The lexer domain is empty and stays: every message the syntax layer raises is now an ID in
// internal/i18n/source/parser.tsv, resolved by the localizer, so there is no text left to key here.
//
// The last 13 rows moved in step 3 of docs/i18n-design.md, and the move was the point: those
// translations were unreachable. The call sites live in the pinned qkparser module, whose errf path
// did fmt.Sprintf on its own literal and consulted no catalog, so the Chinese template rendered under
// QK_LANG=en and these English values were never selected. The IDs are parser.named_struct_requires_type
// and its siblings; TestProofNoTextKeyedEntriesRemain fails if one comes back.
var lexerTable = map[string]string{}
