package filet

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCloseReceiver(t *testing.T) {
	cases := []struct {
		expr string
		want string
	}{
		{"f.Close", "f"},
		{"resp.Body.Close", "resp"},
		{"x.y[0].Close", "x"},
		{"x[0].z.Close", "x"},
		{"get().Close", ""},
	}
	for _, c := range cases {
		t.Run(c.expr, func(t *testing.T) {
			got := closeReceiver(closeSel(t, c.expr))
			if got != c.want {
				t.Fatalf("closeReceiver(%s) = %q, want %q", c.expr, got, c.want)
			}
		})
	}
}

// closeSel parses expr as a package-level var and returns the Close SelectorExpr.
func closeSel(t *testing.T, expr string) ast.Expr {
	t.Helper()
	src := "package p\n\nvar _ = " + expr + "()\n"
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "t.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sel *ast.SelectorExpr
	ast.Inspect(parsed, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok && s.Sel.Name == "Close" {
			sel = s
			return false
		}
		return true
	})
	if sel == nil {
		t.Fatalf("no Close selector in %s", expr)
	}
	return sel.X
}
