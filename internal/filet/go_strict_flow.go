package filet

import (
	"go/ast"
	"go/types"
)

// strictNilFlow reports a pointer or interface result read before the error
// returned beside it is looked at. `resp, err := c.Do(req)` followed by
// `defer resp.Body.Close()` panics on the failure path, where resp is nil.
// Only reads that cannot survive nil count: a field through a pointer, a method
// of an interface, an explicit *p. A pointer method call does not, since the
// method may accept a nil receiver. The scan stops at the first statement that
// mentions the error, compares the value with nil or assigns it again.
func (g *goFile) strictNilFlow() {
	ast.Inspect(g.file, func(n ast.Node) bool {
		list := stmtList(n)
		for i, s := range list {
			g.resultsBeforeErr(s, list[i+1:])
		}
		return true
	})
}

func stmtList(n ast.Node) []ast.Stmt {
	switch t := n.(type) {
	case *ast.BlockStmt:
		return t.List
	case *ast.CaseClause:
		return t.Body
	case *ast.CommClause:
		return t.Body
	}
	return nil
}

// resultsBeforeErr looks at `a, b, err := call()` and follows each nilable
// result through the statements after it.
func (g *goFile) resultsBeforeErr(s ast.Stmt, rest []ast.Stmt) {
	as, ok := s.(*ast.AssignStmt)
	if !ok || len(as.Rhs) != 1 || len(as.Lhs) < 2 {
		return
	}
	if _, isCall := ast.Unparen(as.Rhs[0]).(*ast.CallExpr); !isCall {
		return
	}
	errVar := g.varOf(as.Lhs[len(as.Lhs)-1])
	if errVar == nil || !types.Identical(errVar.Type(), errorType) {
		return
	}
	for _, lhs := range as.Lhs[:len(as.Lhs)-1] {
		if v := g.varOf(lhs); v != nil && nilable(v.Type()) {
			g.firstUse(v, errVar, rest)
		}
	}
}

func (g *goFile) firstUse(v, errVar *types.Var, rest []ast.Stmt) {
	for _, s := range rest {
		if g.mentions(s, errVar) || g.guards(s, v) || g.assigns(s, v) {
			return
		}
		if at := g.deref(s, v); at != nil {
			g.add("go.strict.nilflow", at.Pos(), Error,
				v.Name()+" is read before "+errVar.Name()+" is checked, and it is nil when the call fails")
			return
		}
	}
}

func nilable(t types.Type) bool {
	if _, param := t.(*types.TypeParam); param || types.Identical(t, errorType) {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Interface:
		return true
	}
	return false
}

// errorType is the predeclared error interface.
var errorType = types.Universe.Lookup("error").Type()
