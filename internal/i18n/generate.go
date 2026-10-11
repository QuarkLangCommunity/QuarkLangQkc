package i18n

// The catalog (msg_gen.go) is generated from the authored tables under source/ and committed, so a
// build never needs the generator to run. This directive is how it is regenerated; `go run ./cmd/qki18n
// -check`, which TestGeneratedCatalogIsCurrent drives in the ordinary test run, is what fails when the
// committed file no longer matches its source.
//
// It was documented in docs/i18n-design.md §3.4 and in the prototype's run log, but the directive
// itself was never committed, so `go generate ./internal/i18n` silently did nothing and the only way
// to regenerate was to know the command. Step 3 adds it.
//
//go:generate go run ../../cmd/qki18n -source source -out msg_gen.go
