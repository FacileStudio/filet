package filet

import (
	"go/ast"
	"go/types"
	"strconv"
)

// strictBans names the imports style.strict refuses, with the reason each
// finding gives.
var strictBans = map[string]string{
	"unsafe":  "steps outside the type system and the memory model",
	"C":       "links C through cgo, out of reach of the race detector and the Go toolchain's checks",
	"reflect": "moves type errors from compile time to run time",
}

func (g *goFile) strictImports() {
	for _, spec := range g.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if why, banned := strictBans[path]; err == nil && banned {
			g.add("go.strict.import", spec.Pos(), Error, "import "+strconv.Quote(path)+" "+why)
		}
	}
}

// strictAsserts reports a single-value type assertion. x.(T) panics when x
// holds anything else, a nil interface included; the comma-ok form and a type
// switch hand that case back to the code.
func (g *goFile) strictAsserts() {
	checked := commaOK(g.file)
	ast.Inspect(g.file, func(n ast.Node) bool {
		ta, ok := n.(*ast.TypeAssertExpr)
		if ok && ta.Type != nil && !checked[ta] {
			g.add("go.strict.assert", ta.Pos(), Error,
				"x.("+types.ExprString(ta.Type)+") panics on any other type; use the v, ok form or a type switch")
		}
		return true
	})
}

// commaOK collects the assertions written as v, ok := x.(T) or
// var v, ok = x.(T), the forms that cannot panic.
func commaOK(file *ast.File) map[*ast.TypeAssertExpr]bool {
	checked := map[*ast.TypeAssertExpr]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		var lhs int
		var rhs []ast.Expr
		switch t := n.(type) {
		case *ast.AssignStmt:
			lhs, rhs = len(t.Lhs), t.Rhs
		case *ast.ValueSpec:
			lhs, rhs = len(t.Names), t.Values
		}
		if lhs != 2 || len(rhs) != 1 {
			return true
		}
		if ta, ok := ast.Unparen(rhs[0]).(*ast.TypeAssertExpr); ok {
			checked[ta] = true
		}
		return true
	})
	return checked
}
