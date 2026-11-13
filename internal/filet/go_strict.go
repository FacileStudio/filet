package filet

import (
	"go/ast"
	"go/types"
	"slices"
)

// strict runs the opt-in strict Go family (style.strict). Test files are
// exempt: they import reflect for DeepEqual, leave responses open against
// httptest servers and assert types they built themselves. The typed rules need
// the package's type information and stay silent without it; the
// go.leak.resource skip notice already names a package that did not resolve.
func (g *goFile) strict() {
	if g.isTest {
		return
	}
	g.strictImports()
	g.strictAsserts()
	if g.info == nil {
		return
	}
	g.strictSwitches()
	g.strictClose()
	g.strictTimeouts()
	g.strictSQL()
	g.strictGoroutines()
	g.strictNilFlow()
}

// namedIs reports whether t is exactly the named type pkg.name, aliases
// resolved and pointers not followed.
func namedIs(t types.Type, pkg, name string) bool {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok || n.Obj().Pkg() == nil {
		return false
	}
	return n.Obj().Pkg().Path() == pkg && n.Obj().Name() == name
}

// derefType returns the type t points to, or t itself when it is no pointer.
func derefType(t types.Type) types.Type {
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

// callee returns the function or method a call invokes, or nil for a
// conversion, a builtin or a call through a func value.
func (g *goFile) callee(call *ast.CallExpr) *types.Func {
	var id *ast.Ident
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		id = fun
	case *ast.SelectorExpr:
		id = fun.Sel
	default:
		return nil
	}
	fn, _ := g.info.Uses[id].(*types.Func)
	return fn
}

// isFunc reports whether fn is one of the package-level functions pkg.names.
func isFunc(fn *types.Func, pkg string, names ...string) bool {
	if fn == nil || fn.Pkg() == nil || fn.Signature().Recv() != nil {
		return false
	}
	return fn.Pkg().Path() == pkg && slices.Contains(names, fn.Name())
}

// varOf returns the variable a bare identifier names, or nil for anything
// else, the blank identifier included.
func (g *goFile) varOf(e ast.Expr) *types.Var {
	id, ok := ast.Unparen(e).(*ast.Ident)
	if !ok || id.Name == "_" {
		return nil
	}
	v, _ := g.info.ObjectOf(id).(*types.Var)
	return v
}

// rootIdent returns the identifier a selector, index or paren chain starts
// from: resp for resp.Body.Close, nil when the chain starts with a call.
func rootIdent(e ast.Expr) *ast.Ident {
	for {
		switch t := e.(type) {
		case *ast.Ident:
			return t
		case *ast.SelectorExpr:
			e = t.X
		case *ast.IndexExpr:
			e = t.X
		case *ast.ParenExpr:
			e = t.X
		default:
			return nil
		}
	}
}

// fieldsOf returns the fields of a possibly absent field list.
func fieldsOf(l *ast.FieldList) []*ast.Field {
	if l == nil {
		return nil
	}
	return l.List
}
