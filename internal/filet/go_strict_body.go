package filet

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// bodyUse walks a function body for the ways an *http.Response stops being
// the function's problem: its Body is closed; the response or its Body is
// returned, stored, sent or wrapped in something closable; the response is
// handed to another function; or its Body goes to a deferred call or to a
// function named for closing or draining it. Reading the Body with io.ReadAll
// or a decoder is none of these.
type bodyUse struct {
	g       *goFile
	open    map[*types.Var]token.Pos
	handled map[*types.Var]bool
}

func (u *bodyUse) visit(n ast.Node) bool {
	switch t := n.(type) {
	case *ast.SelectorExpr:
		if id := rootIdent(t.X); id != nil && t.Sel.Name == "Close" {
			u.handled[u.g.varOf(id)] = true
		}
	case *ast.ReturnStmt:
		u.keep(t.Results)
	case *ast.AssignStmt:
		u.keep(t.Rhs)
	case *ast.CompositeLit:
		u.keep(t.Elts)
	case *ast.SendStmt:
		u.keep([]ast.Expr{t.Value})
	case *ast.DeferStmt:
		u.deferred(t.Call.Args)
	case *ast.CallExpr:
		u.call(t)
	}
	return true
}

func (u *bodyUse) keep(exprs []ast.Expr) {
	for _, e := range exprs {
		u.handled[u.owned(e)] = true
	}
}

// owned returns the response an expression carries away: the response itself,
// its Body, either behind & or a composite key, or a closable wrapper of them.
func (u *bodyUse) owned(e ast.Expr) *types.Var {
	switch t := ast.Unparen(e).(type) {
	case *ast.Ident:
		return u.g.varOf(t)
	case *ast.UnaryExpr:
		return u.owned(t.X)
	case *ast.KeyValueExpr:
		return u.owned(t.Value)
	case *ast.SelectorExpr:
		return u.bodyOf(t)
	case *ast.CallExpr:
		return u.wrapped(t)
	}
	return nil
}

func (u *bodyUse) bodyOf(sel *ast.SelectorExpr) *types.Var {
	if sel.Sel.Name != "Body" {
		return nil
	}
	return u.g.varOf(sel.X)
}

// wrapped returns the response whose Body a call wraps into a value that has
// its own Close, such as io.NopCloser(resp.Body): the wrapper now owns it.
func (u *bodyUse) wrapped(call *ast.CallExpr) *types.Var {
	if !hasClose(u.g.info.TypeOf(call)) {
		return nil
	}
	for _, a := range call.Args {
		if v := u.owned(a); v != nil {
			return v
		}
	}
	return nil
}

// call marks a response passed whole to any function, and a Body passed to a
// function whose name says it closes or drains it.
func (u *bodyUse) call(call *ast.CallExpr) {
	closer := closerName(call.Fun)
	for _, a := range call.Args {
		sel, isSel := ast.Unparen(a).(*ast.SelectorExpr)
		switch {
		case !isSel:
			u.handled[u.owned(a)] = true
		case closer:
			u.handled[u.bodyOf(sel)] = true
		}
	}
}

func (u *bodyUse) deferred(args []ast.Expr) {
	for _, a := range args {
		if sel, ok := ast.Unparen(a).(*ast.SelectorExpr); ok {
			u.handled[u.bodyOf(sel)] = true
		}
	}
}

func closerName(fun ast.Expr) bool {
	var name string
	switch f := ast.Unparen(fun).(type) {
	case *ast.Ident:
		name = f.Name
	case *ast.SelectorExpr:
		name = f.Sel.Name
	}
	name = strings.ToLower(name)
	return strings.Contains(name, "close") || strings.Contains(name, "drain")
}
