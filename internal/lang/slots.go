package lang

// Variable slot pre-resolution: turns "linear lookup by name" into "direct index access".
//
// Background: a runtime scope is a linear slot table of "parameters + locals" (scope.slots / paramNames),
// and every identifier read/write scans the names (measured at 27% of interpreter time). A slot index is stable within a **function scope**,
// so it can be resolved once at compile time and written back into the AST (Ident.Slot).
//
// Conservative subset (mark only where the index is **provably stable**; everything else stays 0 and takes the original path):
//   - mark only parameters and prefix top-level declarations: the stretch from the start of the function body up to any "declaration inside a nested block".
//     Once a nested declaration appears (`T x` inside if/while/for/for-in/try, a catch variable, a for initializer declaration),
//     later slot numbers may disagree with the static derivation depending on whether the branch ran -> stop marking from that point on.
//   - mark only uses "outside a for-in / catch / C-style for body": those three push a new scope at runtime,
//     so the current scope is not the function scope (if / while push no scope and can be marked).
//
// At runtime paramNames[slot] is still checked against the name before use, so even a wrong inference only falls back to the slow path
// and never reads the wrong variable (safety fallback).

// resolveSlots marks identifier slots for the program's functions and methods.
func resolveSlots(prog *Program) {
	for _, fn := range prog.Funcs {
		resolveFuncSlots(fn)
	}
	for _, im := range prog.Impls {
		for _, m := range im.Methods {
			resolveFuncSlots(m)
		}
	}
}

func resolveFuncSlots(fn *FuncDecl) {
	if fn == nil || fn.Body == nil {
		return
	}
	// 1) Reliable prefix: parameters -> top-level declarations (stop at the first nested declaration)
	slot := map[string]int32{}
	var next int32
	for i := range fn.Params {
		if _, dup := slot[fn.Params[i].Name]; dup {
			continue
		}
		slot[fn.Params[i].Name] = next // 0-based index; +1 when written back to the AST (0 means unresolved)
		next++
	}
	poisoned := false
	for _, st := range fn.Body.Stmts {
		if hasNestedDecl(st) { // this statement (and everything after it) may have unstable slots -> stop extending the prefix
			poisoned = true
			break
		}
		if ds, ok := st.(*DeclStmt); ok {
			if _, dup := slot[ds.Name]; !dup {
				slot[ds.Name] = next
				next++
			}
		}
	}

	// 2) Traversal marking: mark only where "current scope == function scope"
	// Top-level statements: parameters and top-level declarations take effect one by one (forward), guaranteeing "declare before use"
	seen := map[string]bool{}
	for i := range fn.Params {
		seen[fn.Params[i].Name] = true
	}
	for _, st := range fn.Body.Stmts {
		if !poisoned {
			if ds, ok := st.(*DeclStmt); ok {
				seen[ds.Name] = true
			}
		} else {
			// already poisoned: top-level declarations beyond the prefix no longer guarantee an index -> stop marking new names
			if ds, ok := st.(*DeclStmt); ok {
				delete(slot, ds.Name)
			}
		}
		resolveStmtSlotsFiltered(st, 0, slot, seen)
	}
}

// resolveStmtSlotsFiltered walks statements and marks where possible (seen is the set of names already "in effect").
func resolveStmtSlotsFiltered(st Stmt, depth int, slot map[string]int32, seen map[string]bool) {
	mark := func(e Expr) { resolveExprSlots(e, depth, slot, seen) }
	switch t := st.(type) {
	case *ExprStmt:
		mark(t.X)
	case *LogStmt:
		mark(t.X)
	case *DeleteStmt:
		mark(t.X)
	case *ReturnStmt:
		mark(t.X)
	case *DeclStmt:
		mark(t.Init)
	case *AssignStmt:
		mark(t.Target)
		mark(t.X)
	case *IfStmt:
		mark(t.Cond)
		resolveBlockSlots(t.Then, depth, slot, seen) // if pushes no scope
		resolveBlockSlots(t.Else, depth, slot, seen)
	case *WhileStmt:
		mark(t.Cond)
		resolveBlockSlots(t.Body, depth, slot, seen) // while pushes no scope
	case *ForStmt:
		mark(t.Iter) // loop variable and loop body live in a new scope -> depth +1, no more marking
		resolveBlockSlots(t.Body, depth+1, slot, seen)
	case *ForCStmt:
		resolveStmtSlotsFiltered(t.Init, depth+1, slot, seen)
		mark(t.Cond)
		resolveStmtSlotsFiltered(t.Step, depth+1, slot, seen)
		resolveBlockSlots(t.Body, depth+1, slot, seen)
	case *TryStmt:
		resolveBlockSlots(t.Try, depth, slot, seen)
		resolveBlockSlots(t.Catch, depth+1, slot, seen) // catch variable lives in a new scope
	}
}

// resolveBlockSlots walks the statements of a block.
func resolveBlockSlots(b *Block, depth int, slot map[string]int32, seen map[string]bool) {
	if b == nil {
		return
	}
	for _, st := range b.Stmts {
		resolveStmtSlotsFiltered(st, depth, slot, seen)
	}
}

func resolveExprSlots(e Expr, depth int, slot map[string]int32, seen map[string]bool) {
	switch t := e.(type) {
	case nil:
		return
	case *Ident:
		if depth == 0 && t.Slot == 0 && seen[t.Name] {
			if idx, ok := slot[t.Name]; ok {
				t.Slot = idx + 1 // +1: 0 is reserved for "unresolved"
			}
		}
	case *ListLit:
		for _, it := range t.Items {
			resolveExprSlots(it, depth, slot, seen)
		}
	case *StructLit:
		for _, f := range t.Fields {
			resolveExprSlots(f.X, depth, slot, seen)
		}
	case *NewExpr:
		resolveExprSlots(t.Size, depth, slot, seen)
	case *BinOp:
		resolveExprSlots(t.L, depth, slot, seen)
		resolveExprSlots(t.R, depth, slot, seen)
	case *UnOp:
		resolveExprSlots(t.X, depth, slot, seen)
	case *MemberExpr:
		resolveExprSlots(t.X, depth, slot, seen)
	case *IndexExpr:
		resolveExprSlots(t.X, depth, slot, seen)
		resolveExprSlots(t.Idx, depth, slot, seen)
	case *CallExpr:
		resolveExprSlots(t.Fn, depth, slot, seen)
		for _, a := range t.Args {
			resolveExprSlots(a, depth, slot, seen)
		}
		if t.Sign != nil {
			for _, a := range t.Sign.Args {
				resolveExprSlots(a, depth, slot, seen)
			}
		}
	case *ScopeCall:
		for _, a := range t.Args {
			resolveExprSlots(a, depth, slot, seen)
		}
	}
}

// hasNestedDecl reports whether a statement contains a variable declaration in a nested block:
// if so, later slot numbers may disagree with the static derivation (the slot does not exist when the branch is not taken).
func hasNestedDecl(st Stmt) bool {
	switch t := st.(type) {
	case *DeclStmt:
		return false // a top-level declaration is not itself nested
	case *IfStmt:
		return blockHasDecl(t.Then) || blockHasDecl(t.Else)
	case *WhileStmt:
		return blockHasDecl(t.Body)
	case *ForStmt:
		return true // loop variable lives in a new scope, and declarations in the body may not execute
	case *ForCStmt:
		return true
	case *TryStmt:
		return true // catch variable lives in a new scope
	}
	return false
}

func blockHasDecl(b *Block) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		if _, ok := st.(*DeclStmt); ok {
			return true
		}
		if hasNestedDecl(st) {
			return true
		}
	}
	return false
}
