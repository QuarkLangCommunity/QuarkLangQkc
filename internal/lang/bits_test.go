package lang

// Tests for the raw-bits family of spec §3: bit, bits<N> (N <= 32) and the uchar view.
//
// They pin the three things the standard fixes: what raw bits ARE (masked storage, unsigned
// printing), what they ADMIT (& | ^ ~ << >> == != and b[i], and nothing else) and what the
// conversion call T(x) MEANS (a free reinterpretation at equal width, a value conversion that
// narrows by keeping the low bits).

import (
	"strings"
	"testing"

	"quarklang/internal/i18n"
)

func runBits(t *testing.T, src string) string {
	t.Helper()
	withLocalizer(t, i18n.New(nil, i18n.EN))
	out, err := runSrc(t, src)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	return out
}

// TestBitsValuesAndMasking: a raw value prints unsigned, and every store keeps only its width.
func TestBitsValuesAndMasking(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"bits<8> prints its unsigned decimal value", `bits<8> a = 0xF0; io.println(a);`, "240\n"},
		{"bits<32> prints past the int range", `bits<32> a = 0x7F454C46; io.println(a);`, "2135247942\n"},
		{"a full 32-bit pattern prints unsigned", `bits<32> a = ~bits<32>(0); io.println(a);`, "4294967295\n"},
		{"bit holds one bit", `bit b = 1; io.println(b);`, "1\n"},
		{"declaring a bit masks to one bit", `bit b = 5; io.println(b);`, "1\n"},
		{"declaring bits<4> masks to four bits", `bits<4> a = 0xFF; io.println(a);`, "15\n"},
		{"assignment masks to the declared width", `bits<8> a = 0; a = 0x1FF; io.println(a);`, "255\n"},
		{"uchar declaration narrows by keeping the low bits", `uchar u = 300; io.println(u);`, "44\n"},
		{"uchar declaration of a negative view", `uchar u = -1; io.println(u);`, "255\n"},
		{"uchar reassignment masks", `uchar u = 1; u = 256; io.println(u);`, "0\n"},
		{"uchar masks a variable assignment", `uchar u = 1; int n = 258; u = n; io.println(u);`, "2\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runBits(t, "program main;\nfn main(IOStream io) { "+tc.src+" }")
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestBitsOperators: the bitwise family keeps the left operand's width, and equality compares bit by bit.
func TestBitsOperators(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"and", `bits<8> a = 0xF0; bits<8> b = 0x3C; io.println(a & b);`, "48\n"},
		{"or", `bits<8> a = 0xF0; bits<8> b = 0x0F; io.println(a | b);`, "255\n"},
		{"xor", `bits<8> a = 0xFF; bits<8> b = 0x0F; io.println(a ^ b);`, "240\n"},
		{"complement keeps the width", `bits<8> a = 0xF0; io.println(~a);`, "15\n"},
		{"complement of bits<32>", `bits<32> a = 0; io.println(~a);`, "4294967295\n"},
		{"shift left drops the bits above the width", `bits<8> a = 0xF0; io.println(a << 2);`, "192\n"},
		{"shift right fills with zeros", `bits<8> a = 0xF0; io.println(a >> 4);`, "15\n"},
		{"a raw right shift is not arithmetic", `bits<8> a = 0x80; io.println(a >> 7);`, "1\n"},
		{"shifting by the width clears the value", `bits<8> a = 0xFF; io.println(a << 8);`, "0\n"},
		{"shifting by more than the width clears the value", `bits<8> a = 0xFF; io.println(a >> 100);`, "0\n"},
		{"a constant adapts to the raw width", `bits<8> a = 0xF0; io.println(a & 0x0F);`, "0\n"},
		{"a runtime shift count", `bits<8> a = 0xF0; int n = 2; io.println(a << n);`, "192\n"},
		{"equality compares the bits", `bits<8> a = 0xF0; bits<8> b = 0xF0; io.println(a == b);`, "true\n"},
		{"inequality", `bits<8> a = 0xF0; bits<8> b = 0x0F; io.println(a != b);`, "true\n"},
		{"storage identity holds for one variable", `bits<8> a = 1; io.println(a === a);`, "true\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runBits(t, "program main;\nfn main(IOStream io) { "+tc.src+" }")
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestBitIndexing: b[i] reads the i-th bit as a bit and b[i] = v writes exactly that bit.
func TestBitIndexing(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"read a set bit", `bits<8> a = 0xF0; io.println(a[7]);`, "1\n"},
		{"read a clear bit", `bits<8> a = 0xF0; io.println(a[3]);`, "0\n"},
		{"read with a runtime index", `bits<8> a = 0xF0; int i = 4; io.println(a[i]);`, "1\n"},
		{"write a bit", `bits<8> a = 0x0F; a[7] = 1; io.println(a);`, "143\n"},
		{"clear a bit", `bits<8> a = 0xFF; a[0] = 0; io.println(a);`, "254\n"},
		{"write through a runtime index", `bits<8> a = 0; int i = 3; a[i] = 1; io.println(a);`, "8\n"},
		{"a written bit is a bit", `bits<8> a = 0; a[0] = 9; io.println(a[0]);`, "1\n"},
		{"write a bit value from another bit", `bits<8> a = 0; bit one = 1; a[5] = one; io.println(a);`, "32\n"},
		{"bits<32> indexing", `bits<32> a = 0x7F454C46; io.println(a[31]);`, "0\n"},
		{"bit reads are stable across writes", `bits<8> a = 1; a[1] = 1; io.println(a[0]); io.println(a[1]);`, "1\n1\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runBits(t, "program main;\nfn main(IOStream io) { "+tc.src+" }")
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestBitsConversions: reinterpretation is free at equal width, a value conversion keeps the low bits.
func TestBitsConversions(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"dropping the interpretation", `uchar u = 0xF0; io.println(bits<8>(u));`, "240\n"},
		{"gaining a view", `bits<8> b = 0xF0; io.println(uchar(b));`, "240\n"},
		{"conversion of bits<32> to the int view", `bits<32> m = 0x7F454C46; io.println(int(m));`, "2135247942\n"},
		{"conversion of the int view to bits<32>", `int n = -1; io.println(bits<32>(n));`, "4294967295\n"},
		{"round trip through the mother is free", `int n = -1; io.println(int(bits<32>(n)));`, "-1\n"},
		{"uchar(300) keeps the low bits", `io.println(uchar(300));`, "44\n"},
		{"uchar(-1) keeps the low bits", `io.println(uchar(-1));`, "255\n"},
		{"uchar of a uchar is the value", `uchar u = 200; io.println(uchar(u));`, "200\n"},
		{"a wider view takes the value", `uchar u = 200; io.println(int(u));`, "200\n"},
		{"a narrower view truncates", `int n = 1000; io.println(uchar(n));`, "232\n"},
		{"an integer constant is materialized at the raw width", `io.println(bits<8>(0xF0));`, "240\n"},
		{"bit is the one-bit view", `io.println(bits<1>(bool(true)));`, "1\n"},
		{"bool over a bit", `bit b = 1; io.println(bool(b));`, "true\n"},
		{"uchar is arithmetic through promotion", `uchar u = 200; io.println(u + 100);`, "300\n"},
		{"uchar ordering uses its value", `uchar u = 200; io.println(u < 250);`, "true\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runBits(t, "program main;\nfn main(IOStream io) { "+tc.src+" }")
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestBitsRefusals: raw bits take the bitwise family, equality and b[i] — every other operator, and
// a width mismatch, is a compile-time error (spec §3.3, §3.4).
//
// The expected substrings are the parts the Chinese catalog value and the English source share (type
// names, numbers, the suggested conversion): the catalog is keyed by its source template, so the two
// languages render different wording around them.
func TestBitsRefusals(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"arithmetic on raw bits", `bits<8> a = 1; io.println(a + 1);`, "int(b)"},
		{"subtraction on raw bits", `bits<8> a = 1; io.println(a - 1);`, "int(b)"},
		{"multiplication on raw bits", `bits<8> a = 1; io.println(a * 2);`, "int(b)"},
		{"ordering on raw bits", `bits<8> a = 1; io.println(a < 2);`, "int(b)"},
		{"greater-or-equal on raw bits", `bits<8> a = 1; io.println(a >= 1);`, "int(b)"},
		{"logic and on raw bits", `bits<8> a = 1; io.println(a && true);`, "requires bool"},
		{"logic or on raw bits", `bits<8> a = 1; io.println(a || true);`, "requires bool"},
		{"logic not on raw bits", `bits<8> a = 1; io.println(!a);`, "requires bool"},
		{"unary minus on raw bits", `bits<8> a = 1; io.println(-a);`, "requires a number"},
		{"bitwise across widths", `bits<8> a = 1; bits<16> b = 1; io.println(a & b);`, "bits<16>"},
		{"bitwise with a view", `bits<8> a = 1; int n = 1; io.println(a | n);`, "bits<8>"},
		{"assignment across raw widths", `bits<8> a = 1; bits<16> b = a;`, "cannot assign bits<8> to bits<16>"},
		{"assignment from a view needs the conversion", `bits<8> a = 1; int n = 1; a = n;`, "cannot assign int to bits<8>"},
		{"narrowing a raw conversion is a width mismatch", `bits<8> a = 1; io.println(int(a));`, "int(bits<8>)"},
		{"widening a view into raw bits is a width mismatch", `int n = 1; io.println(bits<8>(n));`, "bits<8>(int)"},
		{"raw to raw across widths is a width mismatch", `bits<8> a = 1; io.println(bits<16>(a));`, "bits<16>(bits<8>)"},
		{"ordering a uchar against a raw value", `bits<8> a = 1; uchar u = 1; io.println(a > u);`, "int(b)"},
		{"a width above the implementation limit", `bits<33> a = 0;`, "bits<33> is not supported"},
		{"a width of zero", `bits<0> a = 0;`, "N must be at least 1"},
		{"a constant bit index outside the width", `bits<8> a = 1; io.println(a[8]);`, "(0..7)"},
		{"a bit index of the wrong type", `bits<8> a = 1; String s = "0"; io.println(a[s]);`, "String"},
		{"writing a view into one bit", `bits<8> a = 1; int n = 1; a[0] = n;`, "bits<8>[i]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withLocalizer(t, i18n.New(nil, i18n.EN))
			_, err := runSrc(t, "program main;\nfn main(IOStream io) { "+tc.src+" }")
			if err == nil {
				t.Fatalf("expected a compile-time error, got none")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

// TestBitsConversionRefusals: conversions the standard does not define are compile-time errors.
func TestBitsConversionRefusals(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"float has no raw width", `float f = 1.5; io.println(bits<32>(f));`, "bits<32>(float)"},
		{"String is not a view", `String s = "a"; io.println(bits<8>(s));`, "bits<8>(String)"},
		{"bool is not an eight-bit view", `uchar u = 1; bool b = true; io.println(uchar(b));`, "uchar(bool)"},
		{"ordering of raw bits stays refused through a conversion", `bits<8> a = 1; io.println(bits<8>(a) < a);`, "int(b)"},
		{"a bit index out of range at run time", `bits<8> a = 1; int i = 9; io.println(a[i]);`, "bit index 9 is out of range"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withLocalizer(t, i18n.New(nil, i18n.EN))
			_, err := runSrc(t, "program main;\nfn main(IOStream io) { "+tc.src+" }")
			if err == nil {
				t.Fatalf("expected an error, got none")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

// TestBitsCallSpelling: bits<N>(x) parses as a conversion call while a < b > (c) stays comparisons.
func TestBitsCallSpelling(t *testing.T) {
	got := runBits(t, `program main;
fn main(IOStream io) {
  bits<8> a = 0xF0;
  io.println(bits<8>(a));
  io.println(bits<32>(0));
  int x = 1; int y = 2;
  io.println(x < y);
  io.println((x < y) == true);
  io.println(x < y == true);
  io.println(y > (x));
}`)
	want := "240\n0\ntrue\ntrue\ntrue\ntrue\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
