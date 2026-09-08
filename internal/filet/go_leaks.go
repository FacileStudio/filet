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
		ident := g.closingIdentFromAssign(n)
		if ident == nil {
			return true
		}
		obj := g.info.ObjectOf(ident)
		if obj == nil {
			return true
		}
		v, ok := obj.(*types.Var)
		if ok && g.implementsCloser(v.Type()) {
			closerVars[v] = n
		}
		return true
	}
}

func (g *goFile) closingIdentFromAssign(n ast.Node) *ast.Ident {
	assign, ok := n.(*ast.AssignStmt)
	if !ok {
		return nil
	}
	for _, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			return ident
		}
	}
	return nil
}

// reportUnclosedVars reports variables that implement io.Closer but are not deferred.
func (g *goFile) reportUnclosedVars(closerVars map[*types.Var]ast.Node, deferredCloses map[string]bool) {
	for v, assignNode := range closerVars {
		if deferredCloses[v.Name()] {
			continue
		}
		g.add("go.leak.resource", assignNode.Pos(), Warn,
			fmt.Sprintf("%s implements io.Closer but has no deferred Close() call", v.Name()))
	}
}
