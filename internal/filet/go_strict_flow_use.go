package filet

import (
	"go/ast"
	"go/token"
	"go/types"
)

// mentions reports whether s names the variable x anywhere.
func (g *goFile) mentions(s ast.Node, x *types.Var) bool {
	found := false
	ast.Inspect(s, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && g.info.ObjectOf(id) == x {
			found = true
		}
		return !found
	})
	return found
}

// guards reports whether s compares v with nil: the code takes the nil case in
// hand itself.
func (g *goFile) guards(s ast.Node, v *types.Var) bool {
	found := false
	ast.Inspect(s, func(n ast.Node) bool {
		if b, ok := n.(*ast.BinaryExpr); ok && (b.Op == token.EQL || b.Op == token.NEQ) {
			found = found || g.nilCompare(b, v)
		}
		return !found
	})
	return found
}

func (g *goFile) nilCompare(b *ast.BinaryExpr, v *types.Var) bool {
	return (g.varOf(b.X) == v && isNilLiteral(b.Y)) || (g.varOf(b.Y) == v && isNilLiteral(b.X))
}

func (g *goFile) assigns(s ast.Stmt, v *types.Var) bool {
	as, ok := s.(*ast.AssignStmt)
	if !ok {
		return false
	}
	for _, l := range as.Lhs {
		if g.varOf(l) == v {
			return true
		}
	}
	return false
}

// deref returns the first read in s that cannot survive v being nil. A closure
// body counts only under defer or go, where it runs on every path; a closure
// that is merely built may never be called.
func (g *goFile) deref(s ast.Stmt, v *types.Var) ast.Node {
	_, deferred := s.(*ast.DeferStmt)
	_, launched := s.(*ast.GoStmt)
	var at ast.Node
	ast.Inspect(s, func(n ast.Node) bool {
		if _, lit := n.(*ast.FuncLit); lit && !deferred && !launched {
			return false
		}
		if at == nil && g.derefs(n, v) {
			at = n
		}
		return at == nil
	})
	return at
}

func (g *goFile) derefs(n ast.Node, v *types.Var) bool {
	switch t := n.(type) {
	case *ast.StarExpr:
		return g.varOf(t.X) == v
	case *ast.SelectorExpr:
		if g.varOf(t.X) != v {
			return false
		}
		switch obj := g.info.Uses[t.Sel].(type) {
		case *types.Var:
			return obj.IsField() && underIs[*types.Pointer](v.Type())
		case *types.Func:
			return underIs[*types.Interface](v.Type())
		}
	}
	return false
}

func underIs[T types.Type](t types.Type) bool {
	_, ok := t.Underlying().(T)
	return ok
}
