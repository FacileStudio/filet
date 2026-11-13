package filet

import (
	"go/ast"
	"go/token"
	"go/types"
)

// closeTargets are the handles go.strict.close follows, as package path, type
// name and the name a message gives them.
var closeTargets = [][3]string{
	{"net/http", "Response", "*http.Response"},
	{"database/sql", "Rows", "*sql.Rows"},
	{"os", "File", "*os.File"},
}

// strictClose reports handles nothing can close: a call whose *http.Response,
// *sql.Rows or *os.File result is dropped on the floor, and an *http.Response
// whose Body is never closed, returned, stored or handed to a closer. A kept
// *sql.Rows or *os.File without a Close is go.leak.resource's finding, so it
// is not reported twice.
func (g *goFile) strictClose() {
	ast.Inspect(g.file, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.ExprStmt:
			g.discardedHandle(t.X, nil)
		case *ast.AssignStmt:
			if len(t.Rhs) == 1 {
				g.discardedHandle(t.Rhs[0], t.Lhs)
			}
		case *ast.FuncDecl:
			g.openBodies(t)
		}
		return true
	})
}

// discardedHandle reports a call whose handle result lands nowhere: the call
// is a statement of its own, or the handle's slot on the left is blank.
func (g *goFile) discardedHandle(e ast.Expr, lhs []ast.Expr) {
	call, ok := ast.Unparen(e).(*ast.CallExpr)
	if !ok {
		return
	}
	for i, t := range resultTypes(g.info.TypeOf(call)) {
		name := closeTarget(t)
		if name == "" || (lhs != nil && !blankAt(lhs, i)) {
			continue
		}
		g.add("go.strict.close", call.Pos(), Error, "the "+name+" this call returns is discarded, so nothing closes it")
		return
	}
}

func resultTypes(t types.Type) []types.Type {
	tuple, ok := t.(*types.Tuple)
	if !ok {
		return []types.Type{t}
	}
	var out []types.Type
	for v := range tuple.Variables() {
		out = append(out, v.Type())
	}
	return out
}

func closeTarget(t types.Type) string {
	if t == nil {
		return ""
	}
	for _, c := range closeTargets {
		if namedIs(derefType(t), c[0], c[1]) {
			return c[2]
		}
	}
	return ""
}

func blankAt(lhs []ast.Expr, i int) bool {
	if i >= len(lhs) {
		return false
	}
	id, ok := lhs[i].(*ast.Ident)
	return ok && id.Name == "_"
}

// openBodies reports each *http.Response variable of a function that the body
// acquires from a call and then never closes nor lets go of. A named result is
// handed to the caller by any return, naked ones included, so it never counts.
func (g *goFile) openBodies(fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}
	open := g.responseVars(fn.Body)
	for _, f := range fieldsOf(fn.Type.Results) {
		for _, name := range f.Names {
			delete(open, g.varOf(name))
		}
	}
	if len(open) == 0 {
		return
	}
	u := &bodyUse{g: g, open: open, handled: map[*types.Var]bool{}}
	ast.Inspect(fn.Body, u.visit)
	for v, pos := range open {
		if !u.handled[v] {
			g.add("go.strict.close", pos, Error, v.Name()+" is an *http.Response whose Body is never closed")
		}
	}
}

func (g *goFile) responseVars(body *ast.BlockStmt) map[*types.Var]token.Pos {
	out := map[*types.Var]token.Pos{}
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || !isAcquisition(as) {
			return true
		}
		for _, lhs := range as.Lhs {
			v := g.varOf(lhs)
			if _, seen := out[v]; v != nil && !seen && namedIs(derefType(v.Type()), "net/http", "Response") {
				out[v] = lhs.Pos()
			}
		}
		return true
	})
	return out
}

// hasClose reports whether t has a Close method, the mark of a value that
// takes ownership of what it wraps.
func hasClose(t types.Type) bool {
	if t == nil {
		return false
	}
	obj, _, _ := types.LookupFieldOrMethod(t, true, nil, "Close")
	_, ok := obj.(*types.Func)
	return ok
}
