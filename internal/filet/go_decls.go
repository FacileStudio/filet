package filet

import (
	"go/ast"
	"strings"
)

func (g *goFile) calls() {
	panics := !g.isTest && g.cfg.Enabled("go.panic")
	discards := g.cfg.Enabled("go.err.discarded")
	if !panics && !discards {
		return
	}
	ast.Inspect(g.file, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.CallExpr:
			g.panicCall(t, panics)
		case *ast.AssignStmt:
			g.discardedCall(t, discards)
		}
		return true
	})
}

func (g *goFile) panicCall(call *ast.CallExpr, enabled bool) {
	id, ok := call.Fun.(*ast.Ident)
	if !enabled || !ok || id.Name != "panic" {
		return
	}
	g.add("go.panic", call.Pos(), Warn, "panic() in library code turns a caller's bug into everyone's crash")
}

func (g *goFile) discardedCall(as *ast.AssignStmt, enabled bool) {
	if !enabled || !allBlank(as) {
		return
	}
	g.add("go.err.discarded", as.Pos(), Warn, "every return value discarded into _")
}

func (g *goFile) inBodyComments() {
	if !g.cfg.Style.BanInlineComments || !g.cfg.Enabled("go.comment.inbody") {
		return
	}
	bodies := funcBodies(g.file)
	for _, group := range g.file.Comments {
		if strings.HasPrefix(group.List[0].Text, "//go:") {
			continue
		}
		if insideAny(group, bodies) {
			g.add("go.comment.inbody", group.Pos(), Info, "comment inside a function body")
		}
	}
}

func funcBodies(file *ast.File) []*ast.BlockStmt {
	var out []*ast.BlockStmt
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
			out = append(out, fn.Body)
		}
	}
	return out
}

func insideAny(group *ast.CommentGroup, bodies []*ast.BlockStmt) bool {
	for _, body := range bodies {
		if group.Pos() > body.Pos() && group.End() < body.End() {
			return true
		}
	}
	return false
}

func cut(lines []string, n int) string {
	if n < 1 || n > len(lines) {
		return ""
	}
	return lines[n-1]
}
