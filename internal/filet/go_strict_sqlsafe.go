package filet

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// safe reports whether a piece of statement text cannot carry an injection: a
// constant, a number or boolean, a strconv formatting of one, an identifier
// quoted by its driver (pq.QuoteIdentifier, pgx.Identifier.Sanitize), or a
// concatenation, Sprintf or local variable made only of those. A parameter or
// a field read is not safe: nothing in the file says where it came from.
func (q *sqlScan) safe(e ast.Expr) bool {
	if q.constant(e) || numeric(q.g.info.TypeOf(e)) {
		return true
	}
	switch t := ast.Unparen(e).(type) {
	case *ast.BinaryExpr:
		return t.Op == token.ADD && q.safe(t.X) && q.safe(t.Y)
	case *ast.CallExpr:
		return q.safeCall(t)
	case *ast.Ident:
		return q.safeVar(q.g.varOf(t))
	}
	return false
}

func (q *sqlScan) safeCall(call *ast.CallExpr) bool {
	fn := q.g.callee(call)
	if fn == nil {
		return false
	}
	name := strings.ToLower(fn.Name())
	switch {
	case fn.Pkg() != nil && fn.Pkg().Path() == "strconv":
		return strings.HasPrefix(fn.Name(), "Format") || fn.Name() == "Itoa"
	case strings.Contains(name, "quote") || strings.Contains(name, "sanitize"):
		return true
	case isFunc(fn, "fmt", "Sprintf"):
		return q.safeArgs(call.Args)
	case isFunc(fn, "strings", "Repeat"):
		return len(call.Args) == 2 && q.safe(call.Args[0])
	}
	return false
}

// safeVar reports whether every value the file assigns to v is safe. A
// variable the file never assigns, such as a parameter, is not.
func (q *sqlScan) safeVar(v *types.Var) bool {
	if v == nil || len(q.assigns[v]) == 0 {
		return false
	}
	if q.seen[v] {
		return true
	}
	q.seen[v] = true
	defer delete(q.seen, v)
	for _, rhs := range q.assigns[v] {
		if !q.safe(rhs) {
			return false
		}
	}
	return true
}

func (q *sqlScan) safeArgs(args []ast.Expr) bool {
	for _, a := range args {
		if !q.safe(a) {
			return false
		}
	}
	return true
}

func (q *sqlScan) constant(e ast.Expr) bool {
	tv, ok := q.g.info.Types[e]
	return ok && tv.Value != nil
}

func numeric(t types.Type) bool {
	if t == nil {
		return false
	}
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&(types.IsNumeric|types.IsBoolean) != 0
}
