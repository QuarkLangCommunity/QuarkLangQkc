module github.com/QuarkLangCommunity/QuarkLangQkc

go 1.26

require github.com/QuarkLangCommunity/QuarkLangQkparser v0.0.0

// Local development points at the sibling checkout, exactly as the compiler module replaces this one.
replace github.com/QuarkLangCommunity/QuarkLangQkparser => ./QuarkLangQkparser
