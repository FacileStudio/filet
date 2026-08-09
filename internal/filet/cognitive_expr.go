package filet

import (
	"go/ast"
	"go/token"
)

type runCounter struct {
	runs int
	last token.Token
}

func logicalRuns(e ast.Expr) int {
	if e == nil {
		return 0
	}
	c := runCounter{last: token.ILLEGAL}
	c.walk(e)
	return c.runs
}

func (c *runCounter) walk(e ast.Expr) {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || (b.Op != token.LAND && b.Op != token.LOR) {
		c.runs += nestedRuns(e)
		return
	}
	c.walk(b.X)
	if b.Op != c.last {
		c.runs++
		c.last = b.Op
	}
	c.walk(b.Y)
}

func nestedRuns(e ast.Expr) int {
	switch t := e.(type) {
	case *ast.ParenExpr:
		return logicalRuns(t.X)
	case *ast.UnaryExpr:
		return logicalRuns(t.X)
	case *ast.CallExpr:
		n := 0
		for _, a := range t.Args {
			n += logicalRuns(a)
		}
		return n
	}
	return 0
}

func statements(body *ast.BlockStmt) int {
	n := 0
	ast.Inspect(body, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause, nil:
		default:
			if _, ok := node.(ast.Stmt); ok {
				n++
			}
		}
		return true
	})
	return n
}
