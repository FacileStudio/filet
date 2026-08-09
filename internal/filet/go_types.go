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
			if d.Tok == token.VAR && g.cfg.Style.BanGlobalMutable && !g.isTest && !mustInitialised(s) {
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

func (g *goFile) flagGlobals(s *ast.ValueSpec, written map[string]bool) {
	for _, n := range s.Names {
		if n.Name == "_" || (!n.IsExported() && !written[n.Name]) {
			continue
		}
		g.add("go.global.mutable", n.Pos(), Warn, "package-level var "+n.Name+" is shared mutable state")
	}
}
