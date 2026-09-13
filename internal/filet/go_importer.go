package filet

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// importerFor resolves the imports of the package holding f. Inside a module
// it asks go list for the dependency types, which the stdlib importer cannot
// do; anywhere else it degrades to importer.Default, which resolves standard
// library imports only.
func importerFor(fset *token.FileSet, f *ast.File) types.Importer {
	if imp := moduleImporter(filepath.Dir(fset.Position(f.Pos()).Filename)); imp != nil {
		return imp
	}
	return importer.Default()
}

// moduleImporter loads the package in dir with go list and returns an importer
// holding the type of every dependency, or nil when the load fails — outside a
// module, or wherever go list cannot answer.
func moduleImporter(dir string) types.Importer {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedTypes,
		Dir:  dir,
	}, ".")
	if err != nil || len(pkgs) == 0 {
		return nil
	}
	imports := map[string]*types.Package{}
	var walk func(p *packages.Package)
	walk = func(p *packages.Package) {
		if p.Types != nil {
			imports[p.PkgPath] = p.Types
		}
		for _, imp := range p.Imports {
			walk(imp)
		}
	}
	for _, p := range pkgs {
		walk(p)
	}
	if len(imports) == 0 {
		return nil
	}
	return mapImporter(imports)
}

type mapImporter map[string]*types.Package

func (m mapImporter) Import(path string) (*types.Package, error) {
	if p, ok := m[path]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("no type information for import %q", path)
}
