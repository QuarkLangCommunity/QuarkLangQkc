// Package stressgen builds the extreme inputs of the QuarkLang robustness suite: every case is a
// deterministic byte string, so a measured run can be reproduced from the generator alone.
//
// The corpus is deliberately adversarial, not a feature showcase: the point of a case is to make the
// lexer, the parser, the type checker or a back end hit a size or shape it was never tuned for. A case
// that the implementation rejects cleanly is a pass; a panic, a hang or a disagreement between the two
// engines is a defect to be written down, never hidden.
package stressgen

import (
	"fmt"
	"strconv"
	"strings"
)

// Categories of the corpus: what a case is meant to stress.
const (
	CategoryScale        = "scale"        // sheer size: lexer, parser, symbol table, code generator
	CategoryPathological = "pathological" // legal-looking shapes that are adversarial to compile or run
	CategoryMalformed    = "malformed"    // inputs no front end may ever accept
	CategoryResource     = "resource"     // the multi-megabyte parse / compile / link cost case
)

// Corpus sizes, exported so the generator test can assert the documented magnitudes.
const (
	IdentifierChars = 1_000_000 // scale_ident_1m: characters in the single identifier
	TopLevelDecls   = 100_000   // scale_decls_100k: top-level declarations
	LineBytes       = 1 << 20   // scale_line_1mb: bytes on the one source line
	NestedBlocks    = 10_000    // scale_nested_blocks: nested block levels inside main
	NestedParens    = 10_000    // scale_nested_parens: nested parentheses in one expression
	GenericDepth    = 1_000     // scale_nested_generics: nested generic type arguments
	CallChainLen    = 10_000    // scale_call_chain: chained instance calls
	FuncParams      = 10_000    // path_fn_10k_params: parameters of one function
	StructFields    = 10_000    // path_struct_10k_fields: fields of one struct
	RecursionDepth  = 1_000_000 // path_recursion_*: recursion levels attempted at run time
	MacroDupDepth   = 24        // path_exp_macro: nesting depth of the duplicating macro chain
	ResourceFuncs   = 50_000    // res_few_mb: functions in the multi-megabyte program
)

// Case is one generated input: a file name, the category it belongs to and the exact bytes to write.
type Case struct {
	Name     string // file name stem (the harness writes <Name>.kq), unique across the corpus
	Category string // one of the Category* constants
	Doc      string // one line: what the case stresses and what a clean outcome looks like
	Source   []byte // the exact file content
}

// All returns the whole corpus in a stable order, one entry per documented stress case.
func All() []Case {
	ctor := func(name, category, doc string, src []byte) Case {
		return Case{Name: name, Category: category, Doc: doc, Source: src}
	}
	return []Case{
		// ── scale: sheer size ──
		ctor("scale_ident_1m", CategoryScale, "one identifier of "+itoa(IdentifierChars)+" characters", identifierMillion()),
		ctor("scale_decls_100k", CategoryScale, itoa(TopLevelDecls)+" top-level type declarations", topLevelDecls()),
		ctor("scale_line_1mb", CategoryScale, "a single source line of at least "+itoa(LineBytes)+" bytes", oneMegabyteLine()),
		ctor("scale_nested_blocks", CategoryScale, itoa(NestedBlocks)+" nested block levels", nestedBlocks()),
		ctor("scale_nested_parens", CategoryScale, itoa(NestedParens)+" nested parentheses in one expression", nestedParens()),
		ctor("scale_nested_generics", CategoryScale, itoa(GenericDepth)+" nested generic type arguments", nestedGenericArgs()),
		ctor("scale_call_chain", CategoryScale, itoa(CallChainLen)+" chained instance calls", longCallChain()),
		// ── pathological: adversarial shapes ──
		ctor("path_recursion_stack", CategoryPathological, "non-tail recursion "+itoa(RecursionDepth)+" levels deep: must fail cleanly, never crash the host", deepRecursion()),
		ctor("path_recursion_tail", CategoryPathological, "tail recursion "+itoa(RecursionDepth)+" levels deep: bounded work, so it must finish", tailRecursion()),
		ctor("path_int_beyond_64", CategoryPathological, "an integer literal far beyond 64 bits", hugeIntLiteral()),
		ctor("path_bits33", CategoryPathological, "a bits<33> declaration, one bit past the supported width", bits33()),
		ctor("path_fn_10k_params", CategoryPathological, "a function with "+itoa(FuncParams)+" parameters", manyParams()),
		ctor("path_struct_10k_fields", CategoryPathological, "a struct with "+itoa(StructFields)+" fields", manyFields()),
		ctor("path_recursive_macro", CategoryPathological, "two macros that expand into each other", recursiveMacro()),
		ctor("path_exp_macro", CategoryPathological, "a macro chain whose substitution would grow exponentially if it were re-scanned", exponentialMacro()),
		// ── malformed: inputs no front end may accept ──
		ctor("bad_empty", CategoryMalformed, "a zero-byte file", emptyFile()),
		ctor("bad_only_comments", CategoryMalformed, "a file that holds nothing but comments", onlyComments()),
		ctor("bad_unterminated_string", CategoryMalformed, "a string literal that never closes", unterminatedString()),
		ctor("bad_unterminated_block", CategoryMalformed, "a function body that never closes", unterminatedBlock()),
		ctor("bad_unterminated_paren", CategoryMalformed, "a parenthesis that never closes", unterminatedParen()),
		ctor("bad_invalid_utf8", CategoryMalformed, "invalid UTF-8 bytes in an identifier and a string literal", invalidUTF8()),
		ctor("bad_nul_byte", CategoryMalformed, "a NUL byte inside a string literal", nulByte()),
		ctor("bad_bom", CategoryMalformed, "a valid program behind a UTF-8 byte-order mark", bomFile()),
		ctor("bad_crlf_mixed", CategoryMalformed, "a program whose line endings alternate between CRLF and LF", crlfMixed()),
		ctor("bad_only_directives", CategoryMalformed, "a file that holds preprocessor directives and no code", onlyDirectives()),
		ctor("bad_unbalanced_ifdef", CategoryMalformed, "an #ifdef that is never closed", unbalancedIfdef()),
		// ── resource: the measured multi-megabyte case ──
		ctor("res_few_mb", CategoryResource, "a valid multi-megabyte program ("+itoa(ResourceFuncs)+" functions)", fewMegabyteProgram()),
	}
}

// program assembles a source file from top-level declarations plus the body of the canonical entry point.
func program(decls, body string) []byte {
	var b strings.Builder
	if decls != "" {
		b.WriteString(decls)
		b.WriteString("\n")
	}
	b.WriteString("fn main(IOStream io) {\n")
	b.WriteString(body)
	b.WriteString("\n}\n")
	return []byte(b.String())
}

// itoa is the small spelling of strconv.Itoa used to build the case descriptions.
func itoa(n int) string { return strconv.Itoa(n) }

// identifierMillion declares one variable whose name is IdentifierChars characters long.
func identifierMillion() []byte {
	return program("", "  int "+strings.Repeat("a", IdentifierChars)+" = 7;\n  io.println(1);")
}

// topLevelDecls declares TopLevelDecls types, so the symbol table and the code generator see one
// declaration per line for a hundred thousand lines.
func topLevelDecls() []byte {
	var b strings.Builder
	for i := 0; i < TopLevelDecls; i++ {
		fmt.Fprintf(&b, "type struct { int v%d; } T%d;\n", i, i)
	}
	return program(b.String(), "  T0 t;\n  io.println(t.v0);")
}

// oneMegabyteLine emits a program whose single source line carries at least LineBytes bytes of
// chained additions, so the lexer cannot amortise a line break.
func oneMegabyteLine() []byte {
	const head, tail = "fn main(IOStream io) { io.println(0", "); }"
	var b strings.Builder
	b.WriteString(head)
	for b.Len()+len(tail) < LineBytes {
		b.WriteString("+1")
	}
	b.WriteString(tail)
	b.WriteString("\n")
	return []byte(b.String())
}

// nestedBlocks nests NestedBlocks if-blocks, so every recursive descent into a statement list is
// NestedBlocks frames deep.
func nestedBlocks() []byte {
	var b strings.Builder
	b.WriteString("fn main(IOStream io) {\n")
	b.WriteString(strings.Repeat("if (true) { ", NestedBlocks))
	b.WriteString("io.println(1);")
	b.WriteString(strings.Repeat(" }", NestedBlocks))
	b.WriteString("\n}\n")
	return []byte(b.String())
}

// nestedParens nests NestedParens parentheses around one integer literal.
func nestedParens() []byte {
	expr := strings.Repeat("(", NestedParens) + "1" + strings.Repeat(")", NestedParens)
	return program("", "  io.println("+expr+");")
}

// nestedGenericArgs nests the type argument of a generic struct GenericDepth levels deep, which also
// exercises the lexer's '>>' closing-run handling.
func nestedGenericArgs() []byte {
	typ := strings.Repeat("Box<", GenericDepth) + "int" + strings.Repeat(">", GenericDepth)
	return program("type struct<T> { T v; } Box;", "  "+typ+" deep;\n  io.println(\"generics\");")
}

// longCallChain chains CallChainLen instance calls on one receiver.
func longCallChain() []byte {
	chain := strings.Repeat(".id()", CallChainLen)
	return program("type struct { int n; } Chain;\nimpl { fn id(Chain self) Chain { return self; } } Chain;",
		"  Chain c;\n  c.n = 7;\n  io.println(c"+chain+".n);")
}

// deepRecursion recurses RecursionDepth levels without a tail call, so the run must exhaust whatever
// stack the engine has and report it instead of taking the host down.
func deepRecursion() []byte {
	return program("fn down(int n) int {\n  if (n == 0) { return 0; }\n  return n + down(n - 1);\n}",
		"  io.println(down("+itoa(RecursionDepth)+"));")
}

// tailRecursion recurses RecursionDepth levels as a tail call, so the work is bounded and a back end
// that reuses the frame finishes while a depth-limited interpreter reports its limit.
func tailRecursion() []byte {
	return program("fn up(int n, int acc) int {\n  if (n == 0) { return acc; }\n  return up(n - 1, acc + 1);\n}",
		"  io.println(up("+itoa(RecursionDepth)+", 0));")
}

// hugeIntLiteral assigns 2^128, which no 64-bit integer can hold.
func hugeIntLiteral() []byte {
	return program("", "  long big = 340282366920938463463374607431768211456;\n  io.println(big);")
}

// bits33 declares raw bits one bit past the documented 32-bit limit.
func bits33() []byte {
	return program("", "  bits<33> w = 0;\n  io.println(w);")
}

// manyParams declares a function with FuncParams parameters and calls it with FuncParams arguments.
func manyParams() []byte {
	var decl, call strings.Builder
	decl.WriteString("fn many(")
	call.WriteString("  io.println(many(")
	for i := 0; i < FuncParams; i++ {
		if i > 0 {
			decl.WriteString(", ")
			call.WriteString(", ")
		}
		fmt.Fprintf(&decl, "int p%d", i)
		call.WriteString(itoa(i))
	}
	decl.WriteString(") int {\n  return p" + itoa(FuncParams-1) + ";\n}")
	call.WriteString("));")
	return program(decl.String(), call.String())
}

// manyFields declares a struct with StructFields fields and touches the last one.
func manyFields() []byte {
	last := itoa(StructFields - 1)
	var b strings.Builder
	b.WriteString("type struct {\n")
	for i := 0; i < StructFields; i++ {
		fmt.Fprintf(&b, "  int f%d;\n", i)
	}
	b.WriteString("} Wide;")
	return program(b.String(), "  Wide w;\n  w.f"+last+" = 42;\n  io.println(w.f"+last+");")
}

// recursiveMacro defines two macros that expand into each other and calls one of them, so an
// expander that re-scans its own output never terminates.
func recursiveMacro() []byte {
	return []byte(`#macro ping (x) {
  #when (run) {
    #return pong(x)
  }
}
#macro pong (x) {
  #when (run) {
    #return ping(x)
  }
}
fn main(IOStream io) {
  ping(io.println("mutual"));
}
`)
}

// exponentialMacro doubles MacroDupDepth times, so an expander that re-scans substituted bodies would
// see 2^MacroDupDepth copies of the leaf.
func exponentialMacro() []byte {
	expr := `io.println("leaf")`
	for i := 0; i < MacroDupDepth; i++ {
		expr = "dup(" + expr + ")"
	}
	return []byte("#macro dup (x) { x x }\nfn main(IOStream io) {\n  " + expr + ";\n}\n")
}

// emptyFile is the zero-byte input.
func emptyFile() []byte { return []byte{} }

// onlyComments holds a line comment and a block comment and no code at all.
func onlyComments() []byte {
	return []byte("// only a line comment\n/* and a block comment */\n")
}

// unterminatedString opens a string literal that never closes.
func unterminatedString() []byte {
	return []byte("fn main(IOStream io) {\n  io.println(\"never closed);\n}\n")
}

// unterminatedBlock opens a function body that never closes.
func unterminatedBlock() []byte {
	return []byte("fn main(IOStream io) {\n  io.println(\"missing brace\");\n")
}

// unterminatedParen opens a parenthesis that never closes.
func unterminatedParen() []byte {
	return []byte("fn main(IOStream io) {\n  io.println((1 + 2);\n}\n")
}

// invalidUTF8 puts bytes that are not valid UTF-8 in an identifier and in a string literal.
func invalidUTF8() []byte {
	return []byte("fn main(IOStream io) {\n  int \xff\xfe = 1;\n  String s = \"\xff\xfe\";\n  io.println(\"ascii\");\n}\n")
}

// nulByte puts a NUL byte inside a string literal of an otherwise valid program.
func nulByte() []byte {
	return []byte("fn main(IOStream io) {\n  String s = \"a\x00b\";\n  io.println(\"ascii\");\n}\n")
}

// bomFile prefixes an otherwise valid program with a UTF-8 byte-order mark.
func bomFile() []byte {
	src := []byte("fn main(IOStream io) {\n  io.println(\"bom\");\n}\n")
	return append([]byte{0xEF, 0xBB, 0xBF}, src...)
}

// crlfMixed alternates CRLF and LF line endings inside a valid program.
func crlfMixed() []byte {
	lines := []string{
		"fn main(IOStream io) {",
		"  int a = 1;",
		"  int b = 2;",
		"  io.println(a + b);",
		"}",
	}
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(line)
		if i%2 == 0 {
			b.WriteString("\r\n")
		} else {
			b.WriteString("\n")
		}
	}
	return []byte(b.String())
}

// onlyDirectives holds preprocessor directives and no code whatsoever.
func onlyDirectives() []byte {
	return []byte("#define EMPTY\n#ifdef EMPTY\n#endif\n")
}

// unbalancedIfdef opens an #ifdef that is never closed.
func unbalancedIfdef() []byte {
	return []byte("#define GUARD\n#ifdef GUARD\nfn main(IOStream io) {\n  io.println(1);\n}\n")
}

// fewMegabyteProgram is the resource case: a valid program of a few megabytes whose parse, type
// check, code generation, link and run are all measured.
func fewMegabyteProgram() []byte {
	var b strings.Builder
	for i := 0; i < ResourceFuncs; i++ {
		fmt.Fprintf(&b, "fn g%d(int x) int {\n  int a = x + %d;\n  int c = a * 2;\n  return c - %d;\n}\n", i, i, i)
	}
	return program(b.String(), "  io.println(g0(1));")
}
