module github.com/QuarkLangCommunity/QuarkLangQkc/compiler

go 1.26

require github.com/QuarkLangCommunity/QuarkLangQkc v0.0.0

replace github.com/QuarkLangCommunity/QuarkLangQkc => ../

// cgen never names the parser module directly — the AST types arrive through internal/lang's
// re-exports — but the module graph still has to resolve it, and a replace only applies from the
// main module, so the sibling checkout must be named here too.
require github.com/QuarkLangCommunity/QuarkLangQkparser v0.0.0

replace github.com/QuarkLangCommunity/QuarkLangQkparser => ../QuarkLangQkparser
