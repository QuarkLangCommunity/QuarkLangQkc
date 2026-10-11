package i18n

// The catalog (msg_gen.go) is generated from the authored tables under source/ and committed, so a
// build never needs the generator to run. This directive is how it is regenerated; `go run ./cmd/qki18n
// -check`, which TestGeneratedCatalogIsCurrent and TestBacklogDocumentIsCurrent drive in the ordinary
// test run, is what fails when a committed file no longer matches its source.
//
// It was documented in docs/i18n-design.md §3.4 and in the prototype's run log, but the directive
// itself was never committed, so `go generate ./internal/i18n` silently did nothing and the only way
// to regenerate was to know the command. Step 3 adds it.
//
// The same run writes docs/i18n-backlog.md, the review document for the untranslated backlog: one
// generator, one set of tables, two artefacts, one staleness check. -root is the checkout the document
// measures call sites in, and it is given explicitly because `go generate` runs in this directory.
//
//go:generate go run ../../cmd/qki18n -source source -out msg_gen.go -doc ../../docs/i18n-backlog.md -root ../..
