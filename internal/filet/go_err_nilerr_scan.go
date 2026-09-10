package filet

import (
	"go/ast"
	"go/token"
)

func (g *goFile) scanNilerr(list []ast.Stmt, errIdx int, errName string, gated bool) {
	for _, s := range list {
		g.scanNilerrStmt(s, errIdx, errName, gated)
	}
}

func (g *goFile) scanNilerrStmt(s ast.Stmt, errIdx int, errName string, gated bool) {
	switch t := s.(type) {
	case *ast.ReturnStmt:
		if !gated && errIdx < len(t.Results) {
			if ident, ok := t.Results[errIdx].(*ast.Ident); ok && ident.Name == "nil" {
				g.add("go.err.nilerr", t.Pos(), Warn,
					"function handles an error but returns nil in its place; did you mean to return the error?")
			}
		}
	case *ast.IfStmt:
		g.scanNilerr(t.Body.List, errIdx, errName, gated || sentinelPartition(t.Cond, errName))
		g.scanNilerrElse(t.Else, errIdx, errName)
	case *ast.LabeledStmt:
		g.scanNilerrStmt(t.Stmt, errIdx, errName, gated)
	case *ast.BlockStmt:
		g.scanNilerr(t.List, errIdx, errName, gated)
	default:
		g.scanNilerrContainer(s, errIdx, errName, gated)
	}
}

// scanNilerrElse walks the branch that runs when a gate is false. It is only
// reached when the error is NOT the sentinel, so it is no longer under that gate
// and a nil return there is a swallowed error again.
func (g *goFile) scanNilerrElse(e ast.Stmt, errIdx int, errName string) {
	switch t := e.(type) {
	case *ast.IfStmt:
		g.scanNilerr(t.Body.List, errIdx, errName, sentinelPartition(t.Cond, errName))
		g.scanNilerrElse(t.Else, errIdx, errName)
	case *ast.BlockStmt:
		g.scanNilerr(t.List, errIdx, errName, false)
	}
}

func (g *goFile) scanNilerrContainer(s ast.Stmt, errIdx int, errName string, gated bool) {
	switch t := s.(type) {
	case *ast.ForStmt:
		g.scanNilerr(t.Body.List, errIdx, errName, gated)
	case *ast.RangeStmt:
		g.scanNilerr(t.Body.List, errIdx, errName, gated)
	case *ast.SwitchStmt:
		g.scanNilerrClauses(t.Body.List, errIdx, errName, gated)
	case *ast.TypeSwitchStmt:
		g.scanNilerrClauses(t.Body.List, errIdx, errName, gated)
	case *ast.SelectStmt:
		g.scanNilerrClauses(t.Body.List, errIdx, errName, gated)
	}
}

func (g *goFile) scanNilerrClauses(list []ast.Stmt, errIdx int, errName string, gated bool) {
	for _, s := range list {
		switch t := s.(type) {
		case *ast.CaseClause:
			g.scanNilerr(t.Body, errIdx, errName, gated)
		case *ast.CommClause:
			g.scanNilerr(t.Body, errIdx, errName, gated)
		}
	}
}

// sentinelPartition reports whether cond branches on errName equal to a non-nil
// sentinel — `err == E` or `errors.Is(err, E)`. Distinguishing one known error
// value from the others is a partition of the error space, not a success check,
// so a nil error-slot return under it is an intentional default for that
// sentinel. Traversal mirrors cognitive.go so coverage is complete; a sentinel
// gate on the true branch inherits to everything inside it, while the else is
// reached only when the error is NOT the sentinel and so stays un-gated.
func sentinelPartition(cond ast.Expr, errName string) bool {
	switch c := cond.(type) {
	case *ast.BinaryExpr:
		return c.Op == token.EQL && (isErrName(c.X, errName) && !isNilLiteral(c.Y) ||
			isErrName(c.Y, errName) && !isNilLiteral(c.X))
	case *ast.CallExpr:
		return errorsIsCall(c, errName)
	}
	return false
}

// errorsIsCall reports whether c is errors.Is(errName, sentinel), in either
// argument order.
func errorsIsCall(c *ast.CallExpr, errName string) bool {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Is" {
		return false
	}
	if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "errors" {
		return false
	}
	if len(c.Args) != 2 {
		return false
	}
	return isErrName(c.Args[0], errName) && !isNilLiteral(c.Args[1]) ||
		isErrName(c.Args[1], errName) && !isNilLiteral(c.Args[0])
}

func isErrName(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}
