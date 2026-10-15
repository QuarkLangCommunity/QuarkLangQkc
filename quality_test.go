package main

import (
	"go/ast"
	"strconv"
	"strings"
	"testing"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/ratchet"
)

// This file is the standing ratchet against the two shapes a growing codebase takes when nobody is
// measuring it: a function that keeps absorbing branches, and an exported name whose last caller left.
// Both are counted rather than forbidden, because the tree already carries them and a ratchet that
// demands a rewrite on its first run is a gate somebody switches off. The counts may only be lowered,
// and there are none to lower for the unused exports: that rule allows zero.

// qualityRules is this gate's rule set: the length a function may not exceed, and the exported names
// nothing calls. counts is the tree's identifier index, which the export rule reads.
func qualityRules(counts map[string]int) []ratchet.Rule {
	return []ratchet.Rule{
		// Sixty is where length stops being a style question: the tree carries 185 functions over forty
		// lines, so a forty-line budget would be a list of ninety numbers rather than a guard.
		longFunction{limit: 60},
		unusedExport{counts: counts},
	}
}

// qualitySpec is this gate over the tree: the identifier index is built once, because "no caller
// anywhere" is not a fact about one file.
func qualitySpec(t *testing.T) ratchet.Spec {
	return ratchet.Spec{
		Name:      "hand-written-shape",
		Reason:    "a count here may only be lowered: split the function, or delete the name nothing calls",
		Allowlist: "internal/quality/allowlist.tsv",
		Proofs:    "internal/ratchet/testdata/quality-rules.tsv",
		Rules:     qualityRules(nameCounts(t)),
	}
}

// longFunction counts the functions of one file longer than its limit.
type longFunction struct {
	// limit is measured from the signature to the closing brace, so a long explanation above a short
	// function is not itself a long function.
	limit int
}

func (r longFunction) Name() string { return "function-over-" + strconv.Itoa(r.limit) }

func (r longFunction) Reads(path string) bool { return strings.HasSuffix(path, ".go") }

func (r longFunction) Scan(f *ratchet.File) []ratchet.Hit {
	if f.Go == nil {
		return nil
	}
	var out []ratchet.Hit
	for _, d := range f.Go.File.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		length := f.Go.Fset.Position(fn.End()).Line - f.Go.Fset.Position(fn.Pos()).Line + 1
		if length > r.limit {
			out = append(out, ratchet.Hit{Rule: r.Name(), Path: f.Path})
		}
	}
	return out
}

// unusedExport counts the exported functions of one file that no file in the tree names twice: once to
// declare them, and once to call them.
type unusedExport struct{ counts map[string]int }

func (r unusedExport) Name() string { return "unused-export" }

func (r unusedExport) Reads(path string) bool { return strings.HasSuffix(path, ".go") }

func (r unusedExport) Scan(f *ratchet.File) []ratchet.Hit {
	if f.Go == nil {
		return nil
	}
	var out []ratchet.Hit
	for _, d := range f.Go.File.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !fn.Name.IsExported() {
			continue
		}
		// main and the testing entry points are called by the toolchain rather than by this tree, and an
		// interface method is called through the interface, which no name index can see.
		name := fn.Name.Name
		if name == "main" || isEntryPoint(name) {
			continue
		}
		if r.counts[name] <= 1 {
			out = append(out, ratchet.Hit{Rule: r.Name(), Path: f.Path})
		}
	}
	return out
}

// isEntryPoint reports whether a name is called by the Go toolchain rather than by this tree.
func isEntryPoint(name string) bool {
	return strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") || strings.HasPrefix(name, "Example")
}

// nameCounts counts how many times each identifier occurs in every tracked Go file, which is the index
// the unused-export rule reads: an exported name occurring once occurs only at its declaration.
func nameCounts(t *testing.T) map[string]int {
	t.Helper()
	root := ratchet.RepoRoot(t)
	counts := map[string]int{}
	for _, path := range ratchet.TrackedFiles(t, root) {
		file := ratchet.NewFile(path, ratchet.ReadText(root, path))
		if file.Go == nil {
			continue
		}
		ast.Inspect(file.Go.File, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				counts[id.Name]++
			}
			return true
		})
	}
	if len(counts) == 0 {
		t.Fatal("no identifiers were counted, so the unused-export rule would report every export")
	}
	return counts
}

// TestQualityRatchet is the ratchet itself, over every tracked Go file.
func TestQualityRatchet(t *testing.T) { ratchet.Verify(t, qualitySpec(t)) }

// TestQualityAllowlistIsWellFormed keeps the ratchet's own input honest: a path git does not track, or a
// count of zero, is a permission that guards nothing.
func TestQualityAllowlistIsWellFormed(t *testing.T) { ratchet.CheckAllowlist(t, qualitySpec(t)) }

// TestQualityRulesMatch is the proof that neither rule here is vacuous: a sixty-one line function is
// caught and a sixty-line one is left, and the export rule is shown both a name nothing calls and a
// method with the same name, which it must leave because a name index cannot see an interface call.
func TestQualityRulesMatch(t *testing.T) {
	// The proof supplies its own index, so what it shows does not depend on what the tree happens to
	// contain: one name nothing calls, one name that is called.
	proof := ratchet.Spec{
		Proofs: "internal/ratchet/testdata/quality-rules.tsv",
		Rules:  qualityRules(map[string]int{"NobodyCallsMe": 1, "EveryBodyCallsMe": 2}),
	}
	ratchet.ProveRules(t, proof, func(r ratchet.Rule, f *ratchet.File) bool { return len(r.Scan(f)) > 0 })
}
