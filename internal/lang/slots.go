package lang

// Variable slot pre-resolution: turns "linear lookup by name" into "direct index access".
//
// Background: a runtime scope is a linear slot table of "parameters + locals" (scope.slots / paramNames),
// and every identifier read/write scans the names (measured at 27% of interpreter time). A slot index is stable within a **function scope**,
// so it can be resolved once at compile time and written back into the AST (Ident.Slot).
//
// Marked (everything else stays 0 and takes the lookup-by-name path):
//   - the reliable prefix: parameters followed by the top-level declarations up to the first statement that can
//     skip a declaration (a declaration inside if/while/for/for-in/try, a catch variable, a for initializer).
//     Those names are declared unconditionally in source order before any index can drift, so their slots are stable.
//   - uses of those names anywhere the current runtime scope is the function scope: top-level code and if/while/try
//     bodies (if / while push no scope; a try's try-block runs in the current scope). for-in / catch / C-style for
//     bodies push a new scope and are never marked against the function scope.
//
// The invariant that makes the runtime path a plain slot read (no name check): a marked identifier always executes
// with sc == the function scope, and the type checker rejects re-declaring one of these names in a block that opens
// no scope (if/while/try), so the slot cannot be shadowed between the declaration and the use. REPL chunks never go
// through resolveSlots (they are parsed, not compiled), so they keep Slot == 0.

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
	// 1) Reliable prefix: parameters -> top-level declarations (stop at the first statement that can
	// skip a declaration). Every name in this prefix is declared unconditionally, in this order,
	// before any index can drift, so its slot is stable for the whole function.
	slot := map[string]int32{}
	seen := map[string]bool{}
	var next int32
	add := func(name string) {
		if _, dup := slot[name]; dup {
			return
		}
		slot[name] = next // 0-based index; +1 when written back to the AST (0 means unresolved)
		seen[name] = true
		next++
	}
	for i := range fn.Params {
		add(fn.Params[i].Name)
	}
	for _, st := range fn.Body.Stmts {
		if hasNestedDecl(st) { // this statement (and everything after it) may declare at a different index -> stop extending the prefix
			break
		}
		if ds, ok := st.(*DeclStmt); ok {
			add(ds.Name)
		}
	}

	// 2) Traversal marking: mark only where "current scope == function scope". Names outside the
	// prefix are never marked: a declaration that can be skipped (nested block, loop initializer,
	// catch variable) may or may not have executed, so the index it would take is not knowable
	// statically. Uses of the prefix names are safe anywhere at depth 0 — the prefix is declared
	// before every statement that follows it, and the type checker rejects re-declaring one of
	// these names in a block that opens no scope (if/while/try), so a marked name always denotes
	// the function-scope slot.
	for _, st := range fn.Body.Stmts {
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
		// The executor opens one scope for the loop and runs init, cond, step and body inside it:
		// nothing here may be marked against the function scope (including the condition).
		resolveStmtSlotsFiltered(t.Init, depth+1, slot, seen)
		resolveExprSlots(t.Cond, depth+1, slot, seen)
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
