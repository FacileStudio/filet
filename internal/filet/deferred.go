package filet

import (
	"go/ast"
)

// findDeferredCloses finds all variables that are closed via defer,
// including Close() calls inside anonymous functions passed to defer.
func (g *goFile) findDeferredCloses(body *ast.BlockStmt) map[string]bool {
	closes := make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}
		g.findClosingInDefer(deferStmt.Call.Fun, closes)
		return true
	})
	return closes
}

// findClosingInDefer searches the deferred expression for Close() calls on
// variables. It descends into anonymous functions passed to defer.
func (g *goFile) findClosingInDefer(e ast.Expr, closes map[string]bool) {
	switch t := e.(type) {
	case *ast.SelectorExpr:
		if t.Sel.Name == "Close" {
			if ident, ok := t.X.(*ast.Ident); ok {
				closes[ident.Name] = true
			}
		}
	case *ast.CallExpr:
		g.findClosingInDefer(t.Fun, closes)
		for _, arg := range t.Args {
			g.findClosingInDefer(arg, closes)
		}
	case *ast.FuncLit:
		g.findClosingInDeferInBody(t.Body, closes)
	}
}

func (g *goFile) findClosingInDeferInBody(body *ast.BlockStmt, closes map[string]bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "Close" {
			if ident, ok := sel.X.(*ast.Ident); ok {
				closes[ident.Name] = true
			}
		}
		return true
	})
}
