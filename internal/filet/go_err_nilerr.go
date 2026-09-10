package filet

import (
	"go/ast"
	"go/token"
	"strings"
)

// nilerrCheck finds functions that compare an error against nil but then
// return nil in the error's place — the classic nilerr bug that compiles clean
// and ships to production as a silent success.
func (g *goFile) nilerrCheck() {
	ast.Inspect(g.file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		if g.isTest && strings.HasPrefix(fn.Name.Name, "Test") {
			return true
		}
		g.checkNilerr(fn)
		return true
	})
}

func (g *goFile) checkNilerr(fn *ast.FuncDecl) {
	errIdx := errorReturnIndex(fn.Type.Results)
	if errIdx < 0 {
		return
	}
	errs := g.functionErrorVars(fn.Body)
	if len(errs) == 0 {
		return
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		compared := comparedIdent(ifStmt.Cond)
		if compared == nil || !errs[compared.Name] {
			return true
		}
		g.checkBlockForNilerr(ifStmt.Body, errIdx, compared.Name)
		return true
	})
}

// errorReturnIndex returns the index of the error return value, or -1 if the
// function does not return the built-in error type.
func errorReturnIndex(results *ast.FieldList) int {
	if results == nil || len(results.List) == 0 {
		return -1
	}
	for i, f := range results.List {
		if ident, ok := f.Type.(*ast.Ident); ok && ident.Name == "error" {
			return i
		}
	}
	return -1
}

// functionErrorVars collects the names of the variables in body that are bound
// as the error slot of an error-returning call. A comparison against one of
// these is an error guard; a comparison against any other name is not, and the
// nilerr rule must not fire on it — `if item != nil` guards a value, not an
// error, and returning nil there is a normal "not found", not a swallowed
// failure.
func (g *goFile) functionErrorVars(body *ast.BlockStmt) map[string]bool {
	errs := make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Rhs) != 1 {
			return true
		}
		call, ok := as.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		idx := g.errorSlotOf(call, as.Lhs)
		if idx < 0 || idx >= len(as.Lhs) {
			return true
		}
		if id, ok := as.Lhs[idx].(*ast.Ident); ok {
			errs[id.Name] = true
		}
		return true
	})
	return errs
}

// errorSlotOf returns the index, within lhs, of the target bound from the
// callee's error slot. It resolves the callee when it is declared in this file;
// otherwise it falls back to the error-is-last convention, where the trailing
// target of a multi-value assignment is the only shape that reads as an error.
func (g *goFile) errorSlotOf(call *ast.CallExpr, lhs []ast.Expr) int {
	name := ""
	switch t := call.Fun.(type) {
	case *ast.Ident:
		name = t.Name
	case *ast.SelectorExpr:
		name = t.Sel.Name
	}
	if name != "" {
		for _, decl := range g.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Name.Name == name {
				return errorReturnIndex(fn.Type.Results)
			}
		}
		if len(lhs) >= 2 {
			return len(lhs) - 1
		}
	}
	return -1
}

// comparedIdent returns the identifier compared against nil — `err != nil` or
// `nil != err` — or nil when e is not a null-equality comparison.
func comparedIdent(e ast.Expr) *ast.Ident {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || b.Op != token.NEQ {
		return nil
	}
	if id, ok := b.X.(*ast.Ident); ok && isNilLiteral(b.Y) {
		return id
	}
	if id, ok := b.Y.(*ast.Ident); ok && isNilLiteral(b.X) {
		return id
	}
	return nil
}

func isNilLiteral(e ast.Expr) bool {
	ident, ok := e.(*ast.Ident)
	return ok && ident.Name == "nil"
}

// checkBlockForNilerr flags a return that puts nil in the error slot inside an
// error-handling block, unless the return sits under a sentinel gate that has
// already partitioned the error space. `if err == fs.ErrNotExist { return 0, nil }`
// is a deliberate "this flavour of failure is a normal result" decision, not a
// swallowed error, so the rule must not fire on it. Traversal mirrors
// cognitive.go so coverage is complete; a sentinel gate on the true branch
// inherits to everything inside it, while the else is reached only when the
// error is NOT the sentinel and so stays un-gated.
func (g *goFile) checkBlockForNilerr(body *ast.BlockStmt, errIdx int, errName string) {
	if body == nil {
		return
	}
	g.scanNilerr(body.List, errIdx, errName, false)
}

func (g *goFile) scanNilerr(list []ast.Stmt, errIdx int, errName string, gated bool) {
	for _, s := range list {
		g.scanNilerrStmt(s, errIdx, errName, gated)
	}
}

func (g *goFile) scanNilerrStmt(s ast.Stmt, errIdx int, errName string, gated bool) {
	switch t := s.(type) {
	case *ast.ReturnStmt:
		if !gated && errIdx < len(t.Results) {
			if ident, ok := t.Results[errIdx].(*ast.Ident); ok && ident.Name == "nil" {
				g.add("go.err.nilerr", t.Pos(), Warn,
					"function handles an error but returns nil in its place; did you mean to return the error?")
			}
		}
	case *ast.IfStmt:
		g.scanNilerr(t.Body.List, errIdx, errName, gated || sentinelPartition(t.Cond, errName))
		g.scanNilerrElse(t.Else, errIdx, errName)
	case *ast.LabeledStmt:
		g.scanNilerrStmt(t.Stmt, errIdx, errName, gated)
	case *ast.BlockStmt:
		g.scanNilerr(t.List, errIdx, errName, gated)
	default:
		g.scanNilerrContainer(s, errIdx, errName, gated)
	}
}

// scanNilerrElse walks the branch that runs when a gate is false. It is only
// reached when the error is NOT the sentinel, so it is no longer under that gate
// and a nil return there is a swallowed error again.
func (g *goFile) scanNilerrElse(e ast.Stmt, errIdx int, errName string) {
	switch t := e.(type) {
	case *ast.IfStmt:
		g.scanNilerr(t.Body.List, errIdx, errName, sentinelPartition(t.Cond, errName))
		g.scanNilerrElse(t.Else, errIdx, errName)
	case *ast.BlockStmt:
		g.scanNilerr(t.List, errIdx, errName, false)
	}
}

func (g *goFile) scanNilerrContainer(s ast.Stmt, errIdx int, errName string, gated bool) {
	switch t := s.(type) {
	case *ast.ForStmt:
		g.scanNilerr(t.Body.List, errIdx, errName, gated)
	case *ast.RangeStmt:
		g.scanNilerr(t.Body.List, errIdx, errName, gated)
	case *ast.SwitchStmt:
		g.scanNilerrClauses(t.Body.List, errIdx, errName, gated)
	case *ast.TypeSwitchStmt:
		g.scanNilerrClauses(t.Body.List, errIdx, errName, gated)
	case *ast.SelectStmt:
		g.scanNilerrClauses(t.Body.List, errIdx, errName, gated)
	}
}

func (g *goFile) scanNilerrClauses(list []ast.Stmt, errIdx int, errName string, gated bool) {
	for _, s := range list {
		switch t := s.(type) {
		case *ast.CaseClause:
			g.scanNilerr(t.Body, errIdx, errName, gated)
		case *ast.CommClause:
			g.scanNilerr(t.Body, errIdx, errName, gated)
		}
	}
}

// sentinelPartition reports whether cond branches on errName equal to a non-nil
// sentinel — `err == E` or `errors.Is(err, E)`. Distinguishing one known error
// value from the others is a partition of the error space, not a success check,
// so a nil error-slot return under it is an intentional default for that
// sentinel.
func sentinelPartition(cond ast.Expr, errName string) bool {
	switch c := cond.(type) {
	case *ast.BinaryExpr:
		if c.Op == token.EQL {
			return isErrName(c.X, errName) && !isNilLiteral(c.Y) ||
				isErrName(c.Y, errName) && !isNilLiteral(c.X)
		}
	case *ast.CallExpr:
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Is" {
			return false
		}
		if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "errors" {
			return false
		}
		if len(c.Args) != 2 {
			return false
		}
		return isErrName(c.Args[0], errName) && !isNilLiteral(c.Args[1]) ||
			isErrName(c.Args[1], errName) && !isNilLiteral(c.Args[0])
	}
	return false
}

func isErrName(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}
