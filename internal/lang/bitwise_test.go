package lang

// Tests for the bitwise operator family: & | ^ on int views and ~ as a unary complement.
//
// The family is the precondition for the standard's bit/bits<N> layer (§3.3 of spec/STANDARD.md):
// raw bits admit bitwise operations and nothing else, and the operators had to exist first.
// Precedence follows Go, not C: the bitwise operators bind tighter than comparison, so `a & b == 0`
// reads as `(a & b) == 0`, and they bind looser than + - * / %.

import (
	"strings"
	"testing"

	"quarklang/internal/i18n"
)

func TestBitwiseOperators(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"or", `program main;
fn main(IOStream io) { int a = 240; int b = 15; io.println(a | b); }`, "255\n"},
		{"and", `program main;
fn main(IOStream io) { int a = 240; int b = 15; io.println(a & b); }`, "0\n"},
		{"xor", `program main;
fn main(IOStream io) { int a = 240; int b = 15; io.println(a ^ b); }`, "255\n"},
		{"xor of equal values", `program main;
fn main(IOStream io) { int a = 37; io.println(a ^ a); }`, "0\n"},
		{"complement", `program main;
fn main(IOStream io) { io.println(~240); }`, "-241\n"},
		{"complement is an involution", `program main;
fn main(IOStream io) { int a = 12345; io.println(~~a); }`, "12345\n"},
		{"complement of zero and minus one", `program main;
fn main(IOStream io) { io.println(~0); io.println(~-1); }`, "-1\n0\n"},
		{"hex literals", `program main;
fn main(IOStream io) { io.println(0xF0 & 0x0F); io.println(0xF0 | 0x0F); }`, "0\n255\n"},
		{"precedence: comparison is looser than &", `program main;
fn main(IOStream io) { int a = 240; int b = 15; io.println(a & b == 0); }`, "true\n"},
		{"precedence: division is tighter than &", `program main;
fn main(IOStream io) { io.println(4 & 12 / 3); }`, "4\n"},
		{"precedence: | is looser than & and ^", `program main;
fn main(IOStream io) { io.println(1 | 2 ^ 3 & 1); }`, "3\n"},
		{"shifts still work beside the family", `program main;
fn main(IOStream io) { io.println((1 << 4) | 1); io.println(0xFF >> 4); }`, "17\n15\n"},
		{"assignment forms and variables", `program main;
fn main(IOStream io) { int m = 0; m = m | 4; m = m | 1; io.println(m & 5); }`, "5\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withLocalizer(t, i18n.New(nil, i18n.EN))
			got, err := runSrc(t, tc.src)
			if err != nil {
				t.Fatalf("run error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestBitwiseRejectsNonInts pins the type rule: the family is defined for int views only until
// bit/bits<N> land, and the interpreter refuses instead of coercing.
func TestBitwiseRejectsNonInts(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"float operand on &", `program main;
fn main(IOStream io) { float f = 1.5; io.println(f & 1); }`},
		{"float operand on |", `program main;
fn main(IOStream io) { float f = 1.5; io.println(1 | f); }`},
		{"float operand on ^", `program main;
fn main(IOStream io) { float f = 1.5; io.println(f ^ 1); }`},
		{"string operand on |", `program main;
fn main(IOStream io) { String s = "a"; io.println(s | 1); }`},
		{"complement of a float", `program main;
fn main(IOStream io) { float f = 1.5; io.println(~f); }`},
		{"complement of a string", `program main;
fn main(IOStream io) { String s = "a"; io.println(~s); }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withLocalizer(t, i18n.New(nil, i18n.EN))
			_, err := runSrc(t, tc.src)
			if err == nil {
				t.Fatalf("expected a type error, got none")
			}
			if !strings.Contains(err.Error(), "int") {
				t.Fatalf("expected the error to mention int operands, got %v", err)
			}
		})
	}
}
