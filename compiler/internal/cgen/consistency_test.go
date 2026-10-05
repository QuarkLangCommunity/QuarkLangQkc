package cgen

// Cross-engine consistency contract for the features that are interpreter-only today:
// === (storage identity), permission-carrying references, address-of and the reference move ++.
//
// The rule these tests encode is the project's policy: when the compiler cannot implement a construct
// with exactly the interpreter's semantics, it must refuse the program — never accept it and produce a
// silently different answer. That is what makes "the interpreter is the semantic reference" true
// rather than aspirational, and it is checked here so a future change cannot quietly relax it.
//
// For === there is a concrete reason it is still refused rather than implemented with a naive pointer
// comparison: parameters are passed by reference, so two distinct parameter variables can carry the
// same address. The interpreter compares the storage the operands themselves occupy (so such a pair is
// false), while a pointer comparison on the callee's registers would say true — a divergence that
// would only show up in aliasing cases, which is exactly the kind of bug a hard error prevents.

import (
	"strings"
	"testing"
)

func TestInterpreterOnlyConstructsAreRefusedNotMiscompiled(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string // substring of the required error
	}{
		{
			name: "storage identity === on variables",
			src:  "fn main(IOStream io) {\n    int x = 5;\n    io.println(x === x);\n}\n",
			want: "=== storage identity is implemented in the interpreter only",
		},
		{
			name: "storage identity === on two variables",
			src:  "fn main(IOStream io) {\n    int x = 5;\n    int y = 5;\n    io.println(x === y);\n}\n",
			want: "=== storage identity is implemented in the interpreter only",
		},
		{
			name: "permission reference declaration",
			src:  "fn main(IOStream io) {\n    int x = 5;\n    &rw u int p = &x;\n    p = 7;\n    io.println(x);\n}\n",
			want: "compiling a permission reference",
		},
		{
			name: "address-of",
			src:  "fn main(IOStream io) {\n    int x = 5;\n    int& p = &x;\n    io.println(p);\n}\n",
			want: "compiling & (address-of) is not supported yet",
		},
		{
			name: "reference move",
			// A bare reference declaration compiles, so the refusal below comes from ++ itself;
			// with a permissioned declaration the permission error would (correctly) fire first.
			src:  "fn main(IOStream io) {\n    int& q = new int;\n    q++;\n}\n",
			want: "compiling ++ (reference move) is not supported yet",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := lowerErr(t, tc.src)
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
			// The refusal must be explicit and actionable, not an internal failure.
			if strings.Contains(err.Error(), "internal:") || strings.Contains(err.Error(), "panic") {
				t.Fatalf("refusal looks like an internal failure: %v", err)
			}
		})
	}
}
