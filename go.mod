module github.com/QuarkLangCommunity/QuarkLangQkc

go 1.26

// The syntax layer lives in its own public repository. The pseudo-version pins one exact commit
// (that module has no tag yet), so a fresh clone — and every consumer of this module — resolves the
// parser from the module proxy with no sibling checkout on disk. A relative replace is honoured
// only inside this checkout and is ignored by importers, which is exactly how moving the syntax
// layer out broke CI on all three platforms.
require github.com/QuarkLangCommunity/QuarkLangQkparser v0.0.0-20261009120357-9d2483d39193
