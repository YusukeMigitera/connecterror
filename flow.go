package connecterror

import (
	"go/ast"
	"go/token"
	"go/types"
)

// This file lets the checker take the order of statements into account, so
// that an error reassigned right before a return, or converted by a deferred
// closure, is judged by the value actually returned.

// scanFlow records what the order-aware checks need: the parent of each
// node, the assignments made inside closures, whether the body jumps, and
// the deferred conversion of the named error result.
func (c *checker) scanFlow() {
	c.parents = map[ast.Node]ast.Node{}
	c.closureAssigns = map[*types.Var][]ast.Expr{}

	var stack []ast.Node
	ast.Inspect(c.body, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			c.parents[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)

		switch n := n.(type) {
		case *ast.BranchStmt:
			if n.Tok == token.GOTO || n.Tok == token.FALLTHROUGH {
				c.jumps = true
			}
		case *ast.FuncLit:
			c.scanClosure(n)
		}
		return true
	})

	c.conversion = c.findConversion()
}

// scanClosure records the assignments made inside lit, including those in
// nested closures.
func (c *checker) scanClosure(lit *ast.FuncLit) {
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				v := c.localVar(lhs)
				if v == nil {
					continue
				}
				switch {
				case len(n.Lhs) == len(n.Rhs):
					c.closureAssigns[v] = append(c.closureAssigns[v], n.Rhs[i])
				case isLastResultOfCall(n.Lhs, n.Rhs, i):
					c.closureAssigns[v] = append(c.closureAssigns[v], n.Rhs[0])
				default:
					// Recorded as untracked by newChecker.
					c.closureAssigns[v] = append(c.closureAssigns[v], nil)
				}
			}
		case *ast.RangeStmt:
			for _, x := range []ast.Expr{n.Key, n.Value} {
				if v := c.localVar(x); v != nil {
					c.closureAssigns[v] = append(c.closureAssigns[v], nil)
				}
			}
		}
		return true
	})
}

// namedError returns the named error result of the function, or nil.
func (c *checker) namedError() *types.Var {
	if c.results.Len() == 0 {
		return nil
	}

	v := c.results.At(c.results.Len() - 1)
	if v.Name() == "" || v.Name() == "_" {
		return nil
	}
	return v
}

// findConversion returns a defer statement at the top level of the body
// whose closure converts the named error result to a Connect error:
//
//	defer func() {
//		if err != nil {
//			err = connect.NewError(connect.CodeInternal, err)
//		}
//	}()
//
// The deferred closure runs after the result is set by any later return, so
// those returns always return nil or a Connect error. It is accepted only if
// every closure in the body assigns the error only nil or Connect errors,
// so that no other deferred closure can undo the conversion, and the
// converting closure does not return before converting.
func (c *checker) findConversion() *ast.DeferStmt {
	v := c.namedError()
	if v == nil || c.untracked[v] {
		return nil
	}

	for _, value := range c.closureAssigns[v] {
		if value == nil || !c.isNilOrConnectError(value) {
			return nil
		}
	}

	for _, stmt := range c.body.List {
		d, ok := stmt.(*ast.DeferStmt)
		if !ok {
			continue
		}

		lit, ok := ast.Unparen(d.Call.Fun).(*ast.FuncLit)
		if !ok || containsReturn(lit.Body) {
			continue
		}

		if c.convertsError(lit.Body.List, v) {
			return d
		}
	}

	return nil
}

// convertsError reports whether stmts contain, at the top level, either an
// assignment of a Connect error to v or such an assignment guarded by
// `if v != nil`.
func (c *checker) convertsError(stmts []ast.Stmt, v *types.Var) bool {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *ast.AssignStmt:
			if c.assignsConnectError(stmt, v) {
				return true
			}
		case *ast.IfStmt:
			if stmt.Init != nil || stmt.Else != nil || !c.isNotNilCheck(stmt.Cond, v) {
				continue
			}
			for _, inner := range stmt.Body.List {
				if assign, ok := inner.(*ast.AssignStmt); ok && c.assignsConnectError(assign, v) {
					return true
				}
			}
		}
	}

	return false
}

// assignsConnectError reports whether assign is `v = e` with e a Connect
// error.
func (c *checker) assignsConnectError(assign *ast.AssignStmt, v *types.Var) bool {
	return assign.Tok == token.ASSIGN &&
		len(assign.Lhs) == 1 && len(assign.Rhs) == 1 &&
		c.isVar(assign.Lhs[0], v) &&
		c.isConnectError(assign.Rhs[0])
}

// isNotNilCheck reports whether cond is `v != nil` or `nil != v`.
func (c *checker) isNotNilCheck(cond ast.Expr, v *types.Var) bool {
	bin, ok := ast.Unparen(cond).(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}

	return (c.isVar(bin.X, v) && isNil(c.pass, bin.Y)) ||
		(isNil(c.pass, bin.X) && c.isVar(bin.Y, v))
}

func (c *checker) isVar(expr ast.Expr, v *types.Var) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && c.pass.TypesInfo.ObjectOf(ident) == v
}

// containsReturn reports whether body has a return statement outside nested
// closures.
func containsReturn(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			found = true
		}
		return !found
	})
	return found
}

// resultAt returns the i-th result of ret: the expression for an explicit
// result, or the named result variable for a bare return.
func (c *checker) resultAt(ret *ast.ReturnStmt, i int) (ast.Expr, *types.Var) {
	if len(ret.Results) == 0 {
		return nil, c.results.At(i)
	}

	expr := ret.Results[i]
	return expr, c.localVar(expr)
}

// returnsNil reports whether the i-th result of ret is nil.
func (c *checker) returnsNil(ret *ast.ReturnStmt, i int) bool {
	expr, v := c.resultAt(ret, i)
	if v == nil {
		return c.isNil(expr)
	}

	if value, ok := c.valueAt(v, ret); ok {
		return value == nil || c.isNil(value)
	}
	return c.isNilVar(v)
}

// returnsConnectErrorAt reports whether the error that ret returns is a
// Connect error.
func (c *checker) returnsConnectErrorAt(ret *ast.ReturnStmt) bool {
	if c.conversion != nil && c.conversion.End() <= ret.Pos() {
		return true
	}

	expr, v := c.resultAt(ret, c.results.Len()-1)
	if v == nil {
		return c.isConnectError(expr)
	}

	if value, ok := c.valueAt(v, ret); ok {
		return value != nil && c.isConnectError(value)
	}
	return c.isConnectErrorVar(v)
}

// valueAt returns the value that v holds when ret runs, found by walking
// back from ret to the last assignment to v. A nil value with ok means the
// zero value. It reports false when the value cannot be decided this way,
// such as across loops or when closures assign to v; the caller then falls
// back to all the values assigned to v.
func (c *checker) valueAt(v *types.Var, ret *ast.ReturnStmt) (value ast.Expr, ok bool) {
	if c.jumps || c.untracked[v] || len(c.closureAssigns[v]) > 0 {
		return nil, false
	}

	var node ast.Node = ret
	for {
		parent := c.parents[node]

		var before []ast.Stmt
		switch p := parent.(type) {
		case *ast.BlockStmt:
			// In the body of a switch, the other clauses do not run before
			// node.
			if _, isClause := node.(*ast.CaseClause); !isClause {
				before = stmtsBefore(p.List, node)
			}
		case *ast.CaseClause:
			before = stmtsBefore(p.Body, node)
		case *ast.IfStmt:
			// node is the body or the else branch.
			if p.Init != nil {
				before = []ast.Stmt{p.Init}
			}
		case *ast.SwitchStmt:
			if p.Init != nil {
				before = []ast.Stmt{p.Init}
			}
		case *ast.TypeSwitchStmt:
			if p.Init != nil {
				before = []ast.Stmt{p.Init}
			}
		default:
			// Loops, select, labeled statements and anything else.
			return nil, false
		}

		for i := len(before) - 1; i >= 0; i-- {
			if value, found, ok := c.assignedBy(before[i], v); found {
				return value, ok
			}
		}

		if parent == c.body {
			// Named results start as zero values. Other variables are
			// declared in the body, so reaching the top means v was given
			// a value in a way not followed here, as in a type switch.
			return nil, c.isNamedResult(v)
		}

		node = parent
	}
}

// stmtsBefore returns the statements in list that precede node.
func stmtsBefore(list []ast.Stmt, node ast.Node) []ast.Stmt {
	for i, stmt := range list {
		if stmt == node {
			return list[:i]
		}
	}
	return nil
}

// assignedBy reports whether stmt assigns to v and, if it does, the value it
// assigns. A nil value with ok means the zero value. ok is false when stmt
// assigns to v in a way that cannot be followed, such as inside a branch.
func (c *checker) assignedBy(stmt ast.Stmt, v *types.Var) (value ast.Expr, found, ok bool) {
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		for i, lhs := range stmt.Lhs {
			if !c.isVar(lhs, v) {
				continue
			}
			switch {
			case stmt.Tok != token.ASSIGN && stmt.Tok != token.DEFINE:
				return nil, true, false
			case len(stmt.Lhs) == len(stmt.Rhs):
				return stmt.Rhs[i], true, true
			case isLastResultOfCall(stmt.Lhs, stmt.Rhs, i):
				return stmt.Rhs[0], true, true
			default:
				return nil, true, false
			}
		}
	case *ast.DeclStmt:
		gen, isGen := stmt.Decl.(*ast.GenDecl)
		if !isGen {
			break
		}
		for _, spec := range gen.Specs {
			vs, isValue := spec.(*ast.ValueSpec)
			if !isValue {
				continue
			}
			for i, name := range vs.Names {
				if c.pass.TypesInfo.Defs[name] != v {
					continue
				}
				switch {
				case len(vs.Values) == 0:
					return nil, true, true
				case len(vs.Names) == len(vs.Values):
					return vs.Values[i], true, true
				case isLastResultOfCall(identExprs(vs.Names), vs.Values, i):
					return vs.Values[0], true, true
				default:
					return nil, true, false
				}
			}
		}
		return nil, false, false
	}

	if c.assignsWithin(stmt, v) {
		return nil, true, false
	}
	return nil, false, false
}

// assignsWithin reports whether any statement nested in node assigns to v.
func (c *checker) assignsWithin(node ast.Node, v *types.Var) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range n.Lhs {
				if c.isVar(lhs, v) {
					found = true
				}
			}
		case *ast.RangeStmt:
			if n.Tok == token.ASSIGN && (c.isVar(n.Key, v) || c.isVar(n.Value, v)) {
				found = true
			}
		}
		return !found
	})
	return found
}
