package filet

import (
	"go/ast"
	"go/token"
)

func writtenIdents(file *ast.File) map[string]bool {
	written := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		markWrites(n, written)
		return true
	})
	return written
}

func markWrites(n ast.Node, written map[string]bool) {
	switch t := n.(type) {
	case *ast.AssignStmt:
		for _, lhs := range t.Lhs {
			markBase(lhs, written)
		}
	case *ast.IncDecStmt:
		markBase(t.X, written)
	case *ast.UnaryExpr:
		if t.Op == token.AND {
			markBase(t.X, written)
		}
	case *ast.CallExpr:
		markBuiltin(t, written)
	}
}

func markBuiltin(call *ast.CallExpr, written map[string]bool) {
	id, ok := call.Fun.(*ast.Ident)
	if !ok || len(call.Args) == 0 {
		return
	}
	if id.Name == "append" || id.Name == "delete" || id.Name == "clear" {
		markBase(call.Args[0], written)
	}
}

func markBase(e ast.Expr, written map[string]bool) {
	for {
		switch t := e.(type) {
		case *ast.Ident:
			written[t.Name] = true
			return
		case *ast.IndexExpr:
			e = t.X
		case *ast.SelectorExpr:
			e = t.X
		case *ast.StarExpr:
			e = t.X
		default:
			return
		}
	}
}
