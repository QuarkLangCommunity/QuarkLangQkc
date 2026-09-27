package lang

// ============ Single-file frontend entry points (shared by tools) ============
//
// The language's own official entry points are Compile / CompileWithImports (they include type checking and import merging).
// Tools (qkcheck / qkdoc / qkrepl / qklsp) often need an AST of **a single file with original coordinates**:
// this does only lexing -> macro expansion -> parsing, no type checking and no import resolution.

// ParseSource does only the frontend: lexing -> macro expansion -> parsing (no import resolution, no type checking).
// Line numbers are file line numbers (for coordinates after import merging see SrcMap).
func ParseSource(src string) (*Program, error) {
	prog, _, err := ParseSourceWithComments(src)
	return prog, err
}

// ParseSourceWithComments is like ParseSource but also returns the comments in the source (in order of appearance).
func ParseSourceWithComments(src string) (*Program, []Comment, error) {
	prog, comments, _, err := ParseSourceAll(src)
	return prog, comments, err
}

// ParseSourceAll additionally returns the **macro definitions** on top of ParseSourceWithComments.
// Macro definitions are cut out before parsing (SplitMacroDefs), so they are not in the AST; qkdoc needs them to generate macro documentation.
func ParseSourceAll(src string) (*Program, []Comment, []*MacroDef, error) {
	toks, comments, err := LexWithComments(src)
	if err != nil {
		return nil, nil, nil, err
	}
	macros, rest, err := SplitMacroDefs(toks)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(macros) > 0 {
		rest, err = ExpandMacros(rest, macros, "explain")
		if err != nil {
			return nil, nil, nil, err
		}
	}
	prog, err := Parse(rest)
	if err != nil {
		return nil, nil, nil, err
	}
	prog.Src = src
	return prog, comments, macros, nil
}
