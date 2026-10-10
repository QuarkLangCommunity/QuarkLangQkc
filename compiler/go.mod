module github.com/QuarkLangCommunity/QuarkLangQkc/compiler

go 1.26

require github.com/QuarkLangCommunity/QuarkLangQkc v0.0.0

replace github.com/QuarkLangCommunity/QuarkLangQkc => ../

// cgen never names the parser module directly — the AST types arrive through internal/lang's
// re-exports — but the module graph still has to resolve it, so the pin is repeated here: while the
// compiler is the main module, a replace declared by the root module does not apply.
require github.com/QuarkLangCommunity/QuarkLangQkparser v0.0.0-20261010115438-20ae746ffdee // indirect
