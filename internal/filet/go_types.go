package filet

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (g *goFile) genDecl(d *ast.GenDecl) {
	documented := d.Doc != nil
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			g.typeSpec(s, documented)
		case *ast.ValueSpec:
			if d.Tok == token.VAR && g.cfg.Style.BanGlobalMutable && !g.isTest && !mustInitialised(s) {
				g.flagGlobals(s, writtenIdents(g.file))
			}
		}
	}
}

func (g *goFile) typeSpec(s *ast.TypeSpec, documented bool) {
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
	if s.Name.IsExported() && g.cfg.Style.RequireDocComments && !documented && s.Doc == nil && !g.isTest {
		g.add("go.doc.missing", s.Pos(), Info, "exported type "+s.Name.Name+" has no doc comment")
	}
}

func (g *goFile) flagGlobals(s *ast.ValueSpec, written map[string]bool) {
	for _, n := range s.Names {
		if n.Name == "_" || (!n.IsExported() && !written[n.Name]) {
			continue
		}
		g.add("go.global.mutable", n.Pos(), Warn, "package-level var "+n.Name+" is shared mutable state")
	}
}
