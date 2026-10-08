package lang

import (
	"strings"
	"testing"
)

// The standard spelling: macro name($a $b) { body } — `macro` keyword, `$`-marked parameters, a braced body (STANDARD 6.6).
func TestMacroDefinitionUsesTheStandardSpelling(t *testing.T) {
	out, err := runSrc(t, `macro add($a $b) {
    #when (run) {
        #return $a + $b
    }
}

fn main(IOStream io) {
    io.println(add(10, 32));
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "42\n" {
		t.Fatalf("got %q", out)
	}
}

// The `$` sigil is what marks a parameter: $x is replaced by the whole argument token sequence.
func TestDollarSubstitutesTheBoundArgument(t *testing.T) {
	out, err := runSrc(t, `macro twice($x) {
    #when (run) {
        #return $x * 2
    }
}

macro pick($first $second) {
    #return $second + $first
}

fn main(IOStream io) {
    io.println(twice(21));
    io.println(pick(10 + 1, 31));
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "42\n42\n" {
		t.Fatalf("got %q", out)
	}
}

// A bare name is not a parameter reference: `x` in the body stays the identifier x, so it resolves where the call sits.
func TestBareNameIsNotASubstitution(t *testing.T) {
	out, err := runSrc(t, `macro use($x) {
    #when (run) {
        #return x
    }
}

fn main(IOStream io) {
    int x = 5;
    io.println(use(1));
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "5\n" {
		t.Fatalf("a bare name must not be substituted by the parameter $x, got %q", out)
	}
}

// #insert(#ast(...)) names a parameter: `$name` is the standard form and a bare name is the legacy form, still accepted.
func TestMacroAstAcceptsTheSigilAndTheLegacyBareName(t *testing.T) {
	for _, ref := range []string{"$a", "a"} {
		t.Run(ref, func(t *testing.T) {
			out, err := runSrc(t, `macro five($a) {
    #when (run) {
        #return #insert(#ast(`+ref+`))
    }
}

fn main(IOStream io) {
    io.println(five(5));
}`)
			if err != nil {
				t.Fatal(err)
			}
			if out != "5\n" {
				t.Fatalf("got %q", out)
			}
		})
	}
}

// A definition with #macro is not the standard spelling and must be rejected with a migration hint.
func TestLegacyHashMacroSpellingIsRejected(t *testing.T) {
	_, err := runSrc(t, `#macro twice($x) {
    #return $x + $x
}

fn main(IOStream io) {
    io.println(twice(21));
}`)
	if err == nil {
		t.Fatal("the #macro spelling must be rejected")
	}
	for _, want := range []string{"#macro", "STANDARD 6.6", "macro name($a $b)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the rejection should mention %q, got %v", want, err)
		}
	}
}

// $ lexes as one token kind (TDollar) so the parser can see the parameter sigil.
func TestDollarLexesAsTheParameterSigil(t *testing.T) {
	toks, err := Lex("macro f($a) { #return $a }")
	if err != nil {
		t.Fatal(err)
	}
	var got []TokenKind
	for _, tok := range toks {
		got = append(got, tok.Kind)
	}
	want := []TokenKind{TMacro, TIdent, TLParen, TDollar, TIdent, TRParen, TLBrace, TSharp, TReturn, TDollar, TIdent, TRBrace, TEOF}
	if len(got) != len(want) {
		t.Fatalf("got %d tokens %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d is %v, want %v (all: %v)", i, got[i], want[i], got)
		}
	}
	if got := TDollar.String(); got != "'$'" {
		t.Errorf("TDollar renders as %q, want %q", got, "'$'")
	}
}
