package filet

import (
	"go/ast"
)

// findClosedVars finds every variable used as the receiver of any Close() call
// in the body — deferred (defer f.Close(), a deferred closure) or explicit
// (f.Close(), errors.Join(werr, f.Close())). Presence of a Close() call is
// treated as "the resource is managed". It is path-blind by design; it catches
// resources that are never closed at all, which is the dominant real leak.
func (g *goFile) findClosedVars(body *ast.BlockStmt) map[string]bool {
	closed := make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Close" {
			return true
		}
		if name := closeReceiver(sel.X); name != "" {
			closed[name] = true
		}
		return true
	})
	return closed
}

// closeReceiver returns the root identifier of a method receiver chain: the
// name "file" for a close on file and "resp" for a close on a response body. It
// returns "" when the receiver begins with a call, because no owned variable
// is there.
func closeReceiver(x ast.Expr) string {
	for {
		switch t := x.(type) {
		case *ast.Ident:
			return t.Name
		case *ast.SelectorExpr:
			x = t.X
		case *ast.IndexExpr:
			x = t.X
		default:
			return ""
		}
	}
}
