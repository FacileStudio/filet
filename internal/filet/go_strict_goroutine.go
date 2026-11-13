package filet

import (
	"go/ast"
	"go/types"
)

// strictGoroutines reports a go statement inside an HTTP handler that hands
// the goroutine no context, channel or WaitGroup. Nothing can then stop it,
// wait for it or tell it the request is over, and it outlives the response
// with whatever it captured. A bare context.Background or context.TODO call
// does not count: it is a context in name only.
func (g *goFile) strictGoroutines() {
	reported := map[*ast.GoStmt]bool{}
	ast.Inspect(g.file, func(n ast.Node) bool {
		if body := g.handlerBody(n); body != nil {
			g.unownedGoroutines(body, reported)
		}
		return true
	})
}

func (g *goFile) handlerBody(n ast.Node) *ast.BlockStmt {
	switch t := n.(type) {
	case *ast.FuncDecl:
		if t.Body != nil && g.isHandler(t.Type) {
			return t.Body
		}
	case *ast.FuncLit:
		if g.isHandler(t.Type) {
			return t.Body
		}
	}
	return nil
}

// isHandler reports whether a function takes an http.ResponseWriter and an
// *http.Request, the shape of every net/http handler.
func (g *goFile) isHandler(ft *ast.FuncType) bool {
	var w, r bool
	for _, f := range fieldsOf(ft.Params) {
		t := g.info.TypeOf(f.Type)
		w = w || namedIs(t, "net/http", "ResponseWriter")
		r = r || namedIs(derefType(t), "net/http", "Request")
	}
	return w && r
}

func (g *goFile) unownedGoroutines(body *ast.BlockStmt, reported map[*ast.GoStmt]bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		gs, ok := n.(*ast.GoStmt)
		if ok && !reported[gs] && !g.owned(gs.Call) {
			reported[gs] = true
			g.add("go.strict.goroutine", gs.Pos(), Error,
				"goroutine started in an HTTP handler gets no context, channel or WaitGroup, so nothing can stop it or wait for it")
		}
		return true
	})
}

// owned reports whether the launched call, arguments and closure body
// included, reaches a context, a channel or a WaitGroup.
func (g *goFile) owned(call *ast.CallExpr) bool {
	found := false
	ast.Inspect(call, func(n ast.Node) bool {
		if e, ok := n.(ast.Expr); ok && !found {
			found = g.lifecycle(e)
		}
		return !found
	})
	return found
}

func (g *goFile) lifecycle(e ast.Expr) bool {
	t := g.info.TypeOf(e)
	if t == nil {
		return false
	}
	if _, ok := t.Underlying().(*types.Chan); ok {
		return true
	}
	if namedIs(t, "context", "Context") {
		call, isCall := ast.Unparen(e).(*ast.CallExpr)
		return !isCall || !isFunc(g.callee(call), "context", "Background", "TODO")
	}
	return namedIs(derefType(t), "sync", "WaitGroup")
}
