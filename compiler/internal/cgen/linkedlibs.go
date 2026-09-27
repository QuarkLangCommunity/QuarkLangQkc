package cgen

import "strings"

// linkedlibs.go — libraries provided via qkc -L (their object files already take part in linking):
// these `library <name>` declarations must **not** also produce a -l<name> link flag, or the linker will look for a library that does not exist.

var objectProvidedLibs = map[string]bool{}

// SetObjectProvidedLibs records library names "provided by object files" (compared in lowercase)
func SetObjectProvidedLibs(names []string) {
	for _, n := range names {
		objectProvidedLibs[strings.ToLower(n)] = true
	}
}

// isObjectProvidedLib reports whether the library is provided by an object file
func isObjectProvidedLib(name string) bool { return objectProvidedLibs[strings.ToLower(name)] }
