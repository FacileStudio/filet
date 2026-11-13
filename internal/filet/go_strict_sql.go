package filet

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
)

// sqlMethods are the query entry points of database/sql, pgx, sqlx and GORM's
// Raw. Each takes the statement text as its first string parameter.
var sqlMethods = map[string]bool{
	"Query": true, "QueryContext": true, "QueryRow": true, "QueryRowContext": true,
	"Queryx": true, "QueryRowx": true, "Exec": true, "ExecContext": true, "MustExec": true,
	"Prepare": true, "PrepareContext": true, "Raw": true,
}

// sqlScan judges the statement text of one file's query calls. assigns holds
// every value the file assigns to each variable, so a query built in steps
// (q := "..."; q += x) is traced back to its parts.
type sqlScan struct {
	g       *goFile
	assigns map[*types.Var][]ast.Expr
	seen    map[*types.Var]bool
}

// strictSQL reports a query method called with statement text the code builds
// at run time, by + or fmt.Sprintf, out of a string that is neither a
// constant, a number, a quoted identifier nor built from those: the shape of
// SQL injection. Text that only passes through, such as a wrapper's parameter,
// is not reported, because nothing in sight builds it.
func (g *goFile) strictSQL() {
	q := &sqlScan{g: g, assigns: g.assignments(), seen: map[*types.Var]bool{}}
	ast.Inspect(g.file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			q.call(call)
		}
		return true
	})
}

func (q *sqlScan) call(call *ast.CallExpr) {
	fn := q.g.callee(call)
	if fn == nil || !sqlMethods[fn.Name()] || fn.Signature().Recv() == nil {
		return
	}
	i := firstStringParam(fn.Signature())
	if i < 0 || i >= len(call.Args) || !q.built(call.Args[i]) {
		return
	}
	q.g.add("go.strict.sql", call.Args[i].Pos(), Error,
		fn.Name()+" runs SQL assembled from a run-time string; pass the value as a query argument")
}

func firstStringParam(sig *types.Signature) int {
	for i := range sig.Params().Len() {
		if b, ok := sig.Params().At(i).Type().Underlying().(*types.Basic); ok && b.Kind() == types.String {
			return i
		}
	}
	return -1
}

// built reports whether e assembles SQL in sight, by + or fmt.Sprintf or in a
// variable assigned that way, with a part that is not safe.
func (q *sqlScan) built(e ast.Expr) bool {
	if q.constant(e) {
		return false
	}
	switch t := ast.Unparen(e).(type) {
	case *ast.BinaryExpr:
		return t.Op == token.ADD && !(q.safe(t.X) && q.safe(t.Y))
	case *ast.CallExpr:
		return isFunc(q.g.callee(t), "fmt", "Sprintf") && !q.safeArgs(t.Args)
	case *ast.Ident:
		return q.builtVar(q.g.varOf(t))
	}
	return false
}

func (q *sqlScan) builtVar(v *types.Var) bool {
	if v == nil || q.seen[v] {
		return false
	}
	q.seen[v] = true
	defer delete(q.seen, v)
	return slices.ContainsFunc(q.assigns[v], q.built)
}

// assignments maps each variable to the values the file gives it. q += x is
// recorded as q + x, so the existing text and the addition are both judged.
func (g *goFile) assignments() map[*types.Var][]ast.Expr {
	out := map[*types.Var][]ast.Expr{}
	ast.Inspect(g.file, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.AssignStmt:
			g.recordAssign(out, t)
		case *ast.ValueSpec:
			g.recordValues(out, identExprs(t.Names), t.Values)
		}
		return true
	})
	return out
}

func (g *goFile) recordAssign(out map[*types.Var][]ast.Expr, as *ast.AssignStmt) {
	if as.Tok == token.ADD_ASSIGN && len(as.Lhs) == 1 && len(as.Rhs) == 1 {
		g.recordValues(out, as.Lhs, []ast.Expr{&ast.BinaryExpr{X: as.Lhs[0], Op: token.ADD, Y: as.Rhs[0]}})
		return
	}
	g.recordValues(out, as.Lhs, as.Rhs)
}

func (g *goFile) recordValues(out map[*types.Var][]ast.Expr, lhs, rhs []ast.Expr) {
	if len(lhs) != len(rhs) {
		return
	}
	for i, l := range lhs {
		if v := g.varOf(l); v != nil {
			out[v] = append(out[v], rhs[i])
		}
	}
}
