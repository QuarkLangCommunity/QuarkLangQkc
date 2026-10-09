package lang

// Permission-carrying references (R2, modelled on BioLang's typed smart references).
//
// A reference type is written "&<perm> <scope> <base>" at the declaration, for example:
//
//	&r   u int p = &x;      // read-only, program scope
//	&w   u int p = &a[0];   // write-only
//	&rw  u int s = &obj.f;  // read-write
//	&rm  f int q = &a[1];   // read + move (function scope)
//
// perm is a stack of r (read) / w (write) / m (move) and scope is u (program) / f (function) /
// a (area) / t (thread) — the same 7 x 4 matrix as BioLang's 28 reference types.
//
// Internally the permission travels inside the type string as a suffix, so every existing type
// comparison keeps working: "int&rw:u" is the type above, and a bare "int&" keeps the pre-R2 meaning
// (full permission, program scope). The bare form is what older code and the compiler already handle,
// which is why nothing regresses.

import (
	"strings"

	"github.com/QuarkLangCommunity/QuarkLangQkparser"
)

// validRefPerm reports whether s is a permission stack made of r/w/m.
func validRefPerm(s string) bool { return qkparser.ValidRefPerm(s) }

// validRefScope reports whether s is one of the four follow layers.
func validRefScope(s string) bool { return qkparser.ValidRefScope(s) }

// refBaseType turns a base type into the reference type declared by "&perm scope base".
func refBaseType(base string) string { return qkparser.RefBaseType(base) }

// refDefaultPerm is the permission of a bare "T&" declaration: everything, program scope — the
// behaviour T& had before permissions existed, so old code keeps working.
const (
	refDefaultPerm  = "rwm"
	refDefaultScope = "u"
)

// isRefType reports whether a type is a reference (T& / pointer T).
func isRefKind(t *Type) bool { return t != nil && t.Kind == tPtr }

// refAllows reports whether a permission stack permits an operation (one of r/w/m).
func refAllows(perm, need string) bool {
	return strings.Contains(perm, need)
}

// refPermCovers reports whether a source permission grants everything the destination needs.
func refPermCovers(src, dst string) bool {
	for _, need := range []string{"r", "w", "m"} {
		if strings.Contains(dst, need) && !strings.Contains(src, need) {
			return false
		}
	}
	return true
}

// refPermWidth ranks scopes by how long a reference stays valid: a reference may be stored in a
// variable whose scope is equal or narrower, never wider (a function-scoped reference must not
// outlive the function by landing in a program-scoped variable).
func refScopeRank(scope string) int {
	switch scope {
	case "u":
		return 3
	case "t":
		return 2
	case "a":
		return 1
	case "f":
		return 0
	}
	return -1
}

// refScopeName renders a scope for error messages.
func refScopeName(scope string) string {
	switch scope {
	case "u":
		return "program"
	case "t":
		return "thread"
	case "a":
		return "area"
	case "f":
		return "function"
	}
	return scope
}
