package filet

import (
	"go/ast"
)

func (g *goFile) calls() {
	panics := !g.isTest && g.cfg.Enabled("go.panic")
	discards := g.cfg.Enabled("go.err.discarded")
	if !panics && !discards {
		return
	}
	cleanup := deferredSpans(g.file)
	ast.Inspect(g.file, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.CallExpr:
			g.panicCall(t, panics)
		case *ast.AssignStmt:
			g.discardedCall(t, discards && !within(cleanup, t.Pos()))
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
	g.add("go.err.discarded", as.Pos(), Info, "every return value discarded into _")
}

func (g *goFile) inBodyComments() {
	if !g.cfg.Style.BanInlineComments || !g.cfg.Enabled("go.comment.inbody") {
		return
	}
	bodies := funcBodies(g.file)
	for _, group := range g.file.Comments {
		if carriesDirective(group) {
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

// carriesDirective reports whether any line of the group speaks to a tool. The
// whole group is then untouchable, since deleting it would delete the directive.
func carriesDirective(group *ast.CommentGroup) bool {
	for _, c := range group.List {
		if IsDirective(c.Text) {
			return true
		}
	}
	return false
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
