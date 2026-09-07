package filet

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
)

// resourceLeaks finds io.Closer values that are not closed with a deferred call
// in the same function scope.
func (g *goFile) resourceLeaks() {
	if g.info == nil {
		return
	}

	ast.Inspect(g.file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		// Skip test functions - they often open files for testing
		if g.isTest && strings.HasPrefix(fn.Name.Name, "Test") {
			return true
		}
		g.checkFunctionForLeaks(fn)
		return true
	})
}

// checkFunctionForLeaks checks a single function for unclosed io.Closer values.
func (g *goFile) checkFunctionForLeaks(fn *ast.FuncDecl) {
	closerVars := g.findCloserVariables(fn.Body)

	if len(closerVars) == 0 {
		return
	}

	deferredCloses := g.findDeferredCloses(fn.Body)
	g.reportUnclosedVars(closerVars, deferredCloses)
}

// findCloserVariables finds all variables of types that implement io.Closer.
func (g *goFile) findCloserVariables(body *ast.BlockStmt) map[*types.Var]ast.Node {
	closerVars := make(map[*types.Var]ast.Node)
	ast.Inspect(body, g.makeCloserVarInspector(closerVars))
	return closerVars
}

// makeCloserVarInspector returns an inspector that finds closer variables.
func (g *goFile) makeCloserVarInspector(closerVars map[*types.Var]ast.Node) func(ast.Node) bool {
	return func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i := range assign.Lhs {
			if i >= len(assign.Lhs) {
				continue
			}
			if ident, ok := assign.Lhs[i].(*ast.Ident); ok {
				if obj := g.info.ObjectOf(ident); obj != nil {
					if v, ok := obj.(*types.Var); ok {
						if g.implementsCloser(v.Type()) {
							closerVars[v] = n
						}
					}
				}
			}
		}
		return true
	}
}

// findDeferredCloses finds all variables that are closed with defer.
func (g *goFile) findDeferredCloses(body *ast.BlockStmt) map[string]bool {
	closes := make(map[string]bool)
	ast.Inspect(body, g.makeDeferredCloseInspector(closes))
	return closes
}

// makeDeferredCloseInspector returns an inspector that finds deferred close calls.
func (g *goFile) makeDeferredCloseInspector(closes map[string]bool) func(ast.Node) bool {
	return func(n ast.Node) bool {
		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}
		call, ok := deferStmt.Call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if call.Sel.Name == "Close" {
			if ident, ok := call.X.(*ast.Ident); ok {
				closes[ident.Name] = true
			}
		}
		return true
	}
}

// reportUnclosedVars reports variables that implement io.Closer but are not deferred.
func (g *goFile) reportUnclosedVars(closerVars map[*types.Var]ast.Node, deferredCloses map[string]bool) {
	for v, assignNode := range closerVars {
		if !deferredCloses[v.Name()] {
			g.add("go.leak.resource", assignNode.Pos(), Warn,
				fmt.Sprintf("%s implements io.Closer but has no deferred Close() call", v.Name()))
		}
	}
}

// implementsCloser checks if a type implements io.Closer by checking if
// it has a Close() error method.
func (g *goFile) implementsCloser(t types.Type) bool {
	named := g.getNamedType(t)
	if named == nil {
		return false
	}
	return g.hasCloseMethod(named)
}

// getNamedType gets the named type from a pointer or interface.
func (g *goFile) getNamedType(t types.Type) *types.Named {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return nil
	}
	return named
}

// hasCloseMethod checks if a named type has a Close method that returns error.
func (g *goFile) hasCloseMethod(named *types.Named) bool {
	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		if m.Name() != "Close" {
			continue
		}
		if g.returnsError(m.Type()) {
			return true
		}
	}
	return false
}

// returnsError checks if a signature returns exactly one error value.
func (g *goFile) returnsError(typ types.Type) bool {
	sig, ok := typ.(*types.Signature)
	if !ok {
		return false
	}
	results := sig.Results()
	if results == nil || results.Len() != 1 {
		return false
	}
	result := results.At(0)
	return g.isErrorType(result.Type())
}

// isErrorType checks if a type is the built-in error type or an empty interface.
func (g *goFile) isErrorType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name() == "error"
	}
	if iface, ok := t.(*types.Interface); ok {
		return iface.NumMethods() == 0 || iface.Empty()
	}
	return false
}