package filet

import (
	"go/ast"
	"strings"
)

func mustInitialised(s *ast.ValueSpec) bool {
	if len(s.Values) == 0 || len(s.Values) != len(s.Names) {
		return false
	}
	for _, v := range s.Values {
		call, ok := v.(*ast.CallExpr)
		if !ok {
			return false
		}
		name := ""
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			name = fn.Sel.Name
		case *ast.Ident:
			name = fn.Name
		}
		if !strings.HasPrefix(name, "Must") && !strings.HasPrefix(name, "Once") {
			return false
		}
	}
	return true
}

func countFields(l *ast.FieldList) int {
	if l == nil {
		return 0
	}
	n := 0
	for _, f := range l.List {
		if len(f.Names) == 0 {
			n++
			continue
		}
		n += len(f.Names)
	}
	return n
}

func named(l *ast.FieldList) bool {
	if l == nil {
		return false
	}
	for _, f := range l.List {
		if len(f.Names) > 0 {
			return true
		}
	}
	return false
}

func receiverType(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return receiverType(t.X)
	case *ast.IndexExpr:
		return receiverType(t.X)
	case *ast.IndexListExpr:
		return receiverType(t.X)
	case *ast.Ident:
		return t.Name
	}
	return "?"
}

func allBlank(as *ast.AssignStmt) bool {
	if len(as.Rhs) != 1 {
		return false
	}
	if _, isCall := as.Rhs[0].(*ast.CallExpr); !isCall {
		return false
	}
	for _, lhs := range as.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok || id.Name != "_" {
			return false
		}
	}
	return true
}
