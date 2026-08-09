package filet

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (g *goFile) genDecl(d *ast.GenDecl) {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			g.typeSpec(s, d.Doc)
		case *ast.ValueSpec:
			if d.Tok == token.VAR && g.cfg.Style.BanGlobalMutable && !g.isTest &&
				!mustInitialised(s) && !sentinelError(s) {
				g.flagGlobals(s, writtenIdents(g.file))
			}
		}
	}
}

func (g *goFile) typeSpec(s *ast.TypeSpec, groupDoc *ast.CommentGroup) {
	lim := g.cfg.Limits
	switch t := s.Type.(type) {
	case *ast.StructType:
		if n := countFields(t.Fields); n > lim.StructFields {
			g.add("go.struct.fields", s.Pos(), Warn,
				fmt.Sprintf("struct %s has %d fields (limit %d)", s.Name.Name, n, lim.StructFields))
		}
	case *ast.InterfaceType:
		if n := countFields(t.Methods); n > lim.InterfaceMethods {
			g.add("go.interface.big", s.Pos(), Warn,
				fmt.Sprintf("interface %s has %d methods (limit %d)", s.Name.Name, n, lim.InterfaceMethods))
		}
	}
	doc := s.Doc
	if doc == nil {
		doc = groupDoc
	}
	g.checkDoc(docTarget{doc: doc, name: s.Name.Name, kind: "type", pos: s.Pos()})
}

// sentinelError reports whether this spec declares errors the way Go declares
// them: `var ErrNotFound = errors.New("not found")`.
//
// They are package-level, exported and technically assignable, which is exactly
// what the rule looks for — but flagging them means flagging io.EOF and
// sql.ErrNoRows. The standard library declares sentinels this way and callers
// compare against them with errors.Is, so there is no other shape available.
func sentinelError(s *ast.ValueSpec) bool {
	if len(s.Values) == 0 {
		return false
	}
	for _, v := range s.Values {
		if !errorConstructor(v) {
			return false
		}
	}
	return true
}

func errorConstructor(v ast.Expr) bool {
	call, ok := v.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return (pkg.Name == "errors" && sel.Sel.Name == "New") ||
		(pkg.Name == "fmt" && sel.Sel.Name == "Errorf")
}

func (g *goFile) flagGlobals(s *ast.ValueSpec, written map[string]bool) {
	for _, n := range s.Names {
		if n.Name == "_" || (!n.IsExported() && !written[n.Name]) {
			continue
		}
		g.add("go.global.mutable", n.Pos(), Warn, "package-level var "+n.Name+" is shared mutable state")
	}
}
