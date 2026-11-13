package filet

import (
	"go/ast"
	"go/token"
	"go/types"
)

// timed is what an `x.Timeout = d` anywhere in the file covers: the objects
// whose client Timeout is assigned, and the client expressions bound to them.
type timed struct {
	objs  map[types.Object]bool
	exprs map[ast.Expr]bool
}

func (g *goFile) timedClients() timed {
	set := timed{objs: map[types.Object]bool{}, exprs: map[ast.Expr]bool{}}
	ast.Inspect(g.file, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok {
			g.timeoutWrites(as.Lhs, set)
		}
		return true
	})
	ast.Inspect(g.file, func(n ast.Node) bool {
		switch b := n.(type) {
		case *ast.AssignStmt:
			g.bindings(b.Lhs, b.Rhs, set)
		case *ast.ValueSpec:
			g.bindings(identExprs(b.Names), b.Values, set)
		}
		return true
	})
	return set
}

func (g *goFile) timeoutWrites(lhs []ast.Expr, set timed) {
	for _, l := range lhs {
		sel, ok := l.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Timeout" || !namedIs(derefType(g.info.TypeOf(sel.X)), "net/http", "Client") {
			continue
		}
		if id := rootIdent(sel.X); id != nil {
			set.objs[g.info.ObjectOf(id)] = true
		}
	}
}

// bindings records the client expressions assigned to something whose Timeout
// the file sets later: c := &http.Client{} followed by c.Timeout = d.
func (g *goFile) bindings(lhs, rhs []ast.Expr, set timed) {
	if len(lhs) != len(rhs) {
		return
	}
	for i, l := range lhs {
		if id := rootIdent(l); id != nil && set.objs[g.info.ObjectOf(id)] {
			set.exprs[clientExpr(rhs[i])] = true
		}
	}
}

// clientExpr strips the parentheses and the & around a client expression.
func clientExpr(e ast.Expr) ast.Expr {
	e = ast.Unparen(e)
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		return ast.Unparen(u.X)
	}
	return e
}

func identExprs(ids []*ast.Ident) []ast.Expr {
	out := make([]ast.Expr, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
