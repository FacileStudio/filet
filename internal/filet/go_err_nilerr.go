package filet

import (
	"go/ast"
	"go/token"
	"strings"
)

// nilerrCheck finds functions that compare an error against nil but then
// return nil in the error's place — the classic nilerr bug that compiles clean
// and ships to production as a silent success.
func (g *goFile) nilerrCheck() {
	ast.Inspect(g.file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		if g.isTest && strings.HasPrefix(fn.Name.Name, "Test") {
			return true
		}
		g.checkNilerr(fn)
		return true
	})
}

func (g *goFile) checkNilerr(fn *ast.FuncDecl) {
	errIdx := errorReturnIndex(fn.Type.Results)
	if errIdx < 0 {
		return
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if !isNilComparison(ifStmt.Cond) {
			return true
		}
		g.checkBlockForNilerr(ifStmt.Body, errIdx)
		return true
	})
}

// errorReturnIndex returns the index of the error return value, or -1 if the
// function does not return the built-in error type.
func errorReturnIndex(results *ast.FieldList) int {
	if results == nil || len(results.List) == 0 {
		return -1
	}
	for i, f := range results.List {
		if ident, ok := f.Type.(*ast.Ident); ok && ident.Name == "error" {
			return i
		}
	}
	return -1
}

// isNilComparison reports whether e is a comparison of an identifier against
// nil: `x != nil` or `nil != x`.
func isNilComparison(e ast.Expr) bool {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return false
	}
	return (isIdent(b.X) && isNilLiteral(b.Y)) || (isNilLiteral(b.X) && isIdent(b.Y))
}

func isIdent(e ast.Expr) bool {
	_, ok := e.(*ast.Ident)
	return ok
}

func isNilLiteral(e ast.Expr) bool {
	ident, ok := e.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func (g *goFile) checkBlockForNilerr(body *ast.BlockStmt, errIdx int) {
	if body == nil {
		return
	}
	ast.Inspect(body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		if errIdx >= len(ret.Results) {
			return true
		}
		if ident, ok := ret.Results[errIdx].(*ast.Ident); ok && ident.Name == "nil" {
			g.add("go.err.nilerr", ret.Pos(), Warn,
				"function handles an error but returns nil in its place; did you mean to return the error?")
		}
		return true
	})
}