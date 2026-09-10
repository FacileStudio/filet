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
// swallowed error, so the rule must not fire on it. The block is walked by the
// scanner in go_err_nilerr_scan.go.
func (g *goFile) checkBlockForNilerr(body *ast.BlockStmt, errIdx int, errName string) {
	if body == nil {
		return
	}
	g.scanNilerr(body.List, errIdx, errName, false)
}
