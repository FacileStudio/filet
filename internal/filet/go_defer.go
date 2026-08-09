package filet

import (
	"go/ast"
	"go/token"
)

// deferredSpans locates every deferred call. Suppressing an error there —
// `defer func() { _ = f.Close() }()` — is the documented Go idiom, not a defect.
func deferredSpans(file *ast.File) [][2]token.Pos {
	var out [][2]token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		if t, ok := n.(*ast.DeferStmt); ok {
			out = append(out, [2]token.Pos{t.Pos(), t.End()})
		}
		return true
	})
	return out
}

func within(spans [][2]token.Pos, pos token.Pos) bool {
	for _, s := range spans {
		if pos >= s[0] && pos < s[1] {
			return true
		}
	}
	return false
}
