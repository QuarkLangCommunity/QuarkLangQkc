package lang

// 变量槽位预解析：把「按名字线性查找」降为「直接下标访问」。
//
// 背景：运行时作用域是「参数 + 局部变量」的线性槽位表（scope.slots / paramNames），
// 每次读/写标识符都要扫名字（实测占解释器 27% 时间）。槽位下标在**函数作用域**内是稳定的，
// 因此可以在编译期解析一次、把下标写回 AST（Ident.Slot）。
//
// 保守子集（只在**可证明下标稳定**处写标记，其余留 0 走原路径）：
//   - 只标参数与前缀顶层声明：从函数体开头起、在遇到任何「嵌套块内声明」之前的那一段。
//     一旦出现嵌套声明（if/while/for/for-in/try 体内的 `T x`、catch 变量、for 初始化声明），
//     后续槽位序号可能因分支是否执行而与静态推导不一致 → 从该点起停止标注。
//   - 只标「不在 for-in / catch / C 风格 for 体内」的使用点：这三处运行时会压入新作用域，
//     此时当前作用域不是函数作用域（if / while 不压作用域，可标）。
//
// 运行时还会校验 paramNames[slot] == 名字后才使用，即使推断有误也只会退回慢路径，
// 不会读到错误的变量（安全兜底）。

// resolveSlots 为程序里的函数与方法标注标识符槽位。
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
	// 1) 可靠前缀：参数 → 顶层声明（遇到嵌套声明即停止）
	slot := map[string]int32{}
	var next int32
	for i := range fn.Params {
		if _, dup := slot[fn.Params[i].Name]; dup {
			continue
		}
		slot[fn.Params[i].Name] = next // 0 基下标；写回 AST 时 +1（0 表示未解析）
		next++
	}
	poisoned := false
	for _, st := range fn.Body.Stmts {
		if hasNestedDecl(st) { // 该语句（含其后）槽位可能不稳定 → 停止扩充前缀
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

	// 2) 遍历标注：只在「当前作用域 = 函数作用域」处标注
	// 顶层语句：参数与顶层声明逐个生效（前向），保证「先声明后使用」
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
			// 已 poisoned：前缀之外的顶层声明不再保证下标 → 不再标注新名字
			if ds, ok := st.(*DeclStmt); ok {
				delete(slot, ds.Name)
			}
		}
		resolveStmtSlotsFiltered(st, 0, slot, seen)
	}
}

// resolveStmtSlotsFiltered 遍历语句并在可用处标注（seen 为「已生效」的名字集合）。
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
		resolveBlockSlots(t.Then, depth, slot, seen) // if 不压作用域
		resolveBlockSlots(t.Else, depth, slot, seen)
	case *WhileStmt:
		mark(t.Cond)
		resolveBlockSlots(t.Body, depth, slot, seen) // while 不压作用域
	case *ForStmt:
		mark(t.Iter) // 循环变量与循环体在新作用域内 → 深度 +1，不再标注
		resolveBlockSlots(t.Body, depth+1, slot, seen)
	case *ForCStmt:
		resolveStmtSlotsFiltered(t.Init, depth+1, slot, seen)
		mark(t.Cond)
		resolveStmtSlotsFiltered(t.Step, depth+1, slot, seen)
		resolveBlockSlots(t.Body, depth+1, slot, seen)
	case *TryStmt:
		resolveBlockSlots(t.Try, depth, slot, seen)
		resolveBlockSlots(t.Catch, depth+1, slot, seen) // catch 变量在新作用域
	}
}

// resolveBlockSlots 遍历块内语句。
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
				t.Slot = idx + 1 // +1：0 保留给「未解析」
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

// hasNestedDecl 判断语句内部（嵌套块中）是否存在变量声明：
// 有则其后槽位序号可能与静态推导不一致（分支未执行时该槽位不存在）。
func hasNestedDecl(st Stmt) bool {
	switch t := st.(type) {
	case *DeclStmt:
		return false // 顶层声明本身不算嵌套
	case *IfStmt:
		return blockHasDecl(t.Then) || blockHasDecl(t.Else)
	case *WhileStmt:
		return blockHasDecl(t.Body)
	case *ForStmt:
		return true // 循环变量在新作用域，且体内声明可能不执行
	case *ForCStmt:
		return true
	case *TryStmt:
		return true // catch 变量在新作用域
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
