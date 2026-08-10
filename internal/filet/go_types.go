package filet

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func (g *goFile) genDecl(d *ast.GenDecl) {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			g.typeSpec(s, d.Doc, len(d.Specs) > 1)
		case *ast.ValueSpec:
			if d.Tok == token.VAR && g.cfg.Style.BanGlobalMutable && !g.isTest &&
				!mustInitialised(s) && !sentinelError(s) && !embedded(s, d.Doc) {
				g.flagGlobals(s, writtenIdents(g.file))
			}
		}
	}
}

func (g *goFile) typeSpec(s *ast.TypeSpec, groupDoc *ast.CommentGroup, grouped bool) {
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
	doc, shared := s.Doc, false
	if doc == nil {
		doc, shared = groupDoc, grouped
	}
	g.checkDoc(docTarget{doc: doc, name: s.Name.Name, kind: "type", pos: s.Pos(), shared: shared})
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

// embedded reports whether this spec is the target of a //go:embed directive.
//
// The directive only applies to a package-level var — embed rejects a const, and
// there is no function form — so flagging one asks for a shape the language does
// not offer. The value also comes from the build rather than from the program,
// which is the opposite of the shared mutable state the rule is looking for.
func embedded(s *ast.ValueSpec, groupDoc *ast.CommentGroup) bool {
	for _, doc := range []*ast.CommentGroup{s.Doc, groupDoc} {
		if doc == nil {
			continue
		}
		for _, c := range doc.List {
			if strings.HasPrefix(strings.TrimPrefix(c.Text, "//"), "go:embed") {
				return true
			}
		}
	}
	return false
}

func (g *goFile) flagGlobals(s *ast.ValueSpec, written map[string]bool) {
	for _, n := range s.Names {
		if n.Name == "_" || (!n.IsExported() && !written[n.Name]) {
			continue
		}
		g.add("go.global.mutable", n.Pos(), Warn, "package-level var "+n.Name+" is shared mutable state")
	}
}
