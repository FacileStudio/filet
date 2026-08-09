package filet

import (
	"go/ast"
)

type cognitiveScorer struct {
	score int
	name  string
}

// Cognitive returns the cognitive complexity of a function body, following the
// SonarSource metric: a switch costs one point rather than one per case, and every
// construct is penalised by how deeply it is nested.
func Cognitive(fn *ast.FuncDecl) int {
	if fn.Body == nil {
		return 0
	}
	c := cognitiveScorer{name: fn.Name.Name}
	c.stmts(fn.Body.List, 0)
	return c.score
}

func (c *cognitiveScorer) stmts(list []ast.Stmt, depth int) {
	for _, s := range list {
		c.stmt(s, depth)
	}
}

func (c *cognitiveScorer) stmt(s ast.Stmt, depth int) {
	switch t := s.(type) {
	case *ast.IfStmt:
		c.score += 1 + depth + logicalRuns(t.Cond)
		c.stmts(t.Body.List, depth+1)
		c.elseBranch(t.Else, depth)
	case *ast.BranchStmt:
		if t.Label != nil {
			c.score++
		}
	case *ast.LabeledStmt:
		c.stmt(t.Stmt, depth)
	case *ast.BlockStmt:
		c.stmts(t.List, depth)
	default:
		if !c.loopOrSwitch(s, depth) {
			c.walkExprs(s, depth)
		}
	}
}

func (c *cognitiveScorer) loopOrSwitch(s ast.Stmt, depth int) bool {
	switch t := s.(type) {
	case *ast.ForStmt:
		c.score += 1 + depth + logicalRuns(t.Cond)
		c.stmts(t.Body.List, depth+1)
	case *ast.RangeStmt:
		c.score += 1 + depth
		c.stmts(t.Body.List, depth+1)
	case *ast.SwitchStmt:
		c.score += 1 + depth + logicalRuns(t.Tag)
		c.clauses(t.Body.List, depth+1)
	case *ast.TypeSwitchStmt:
		c.score += 1 + depth
		c.clauses(t.Body.List, depth+1)
	case *ast.SelectStmt:
		c.score += 1 + depth
		c.clauses(t.Body.List, depth+1)
	default:
		return false
	}
	return true
}

func (c *cognitiveScorer) elseBranch(s ast.Stmt, depth int) {
	switch t := s.(type) {
	case *ast.IfStmt:
		c.score += 1 + logicalRuns(t.Cond)
		c.stmts(t.Body.List, depth+1)
		c.elseBranch(t.Else, depth)
	case *ast.BlockStmt:
		c.score++
		c.stmts(t.List, depth+1)
	}
}

func (c *cognitiveScorer) clauses(list []ast.Stmt, depth int) {
	for _, s := range list {
		switch t := s.(type) {
		case *ast.CaseClause:
			c.stmts(t.Body, depth)
		case *ast.CommClause:
			c.stmts(t.Body, depth)
		}
	}
}

func (c *cognitiveScorer) walkExprs(n ast.Node, depth int) {
	ast.Inspect(n, func(node ast.Node) bool {
		switch t := node.(type) {
		case *ast.FuncLit:
			c.stmts(t.Body.List, depth+1)
			return false
		case *ast.CallExpr:
			if id, ok := t.Fun.(*ast.Ident); ok && id.Name == c.name {
				c.score++
			}
		}
		return true
	})
}
