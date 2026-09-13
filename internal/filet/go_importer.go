package filet

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/packages"
)

// importerFor resolves the imports of the package holding f. Inside a module
// it resolves dependency types from export data, which the stdlib importer
// cannot do; anywhere else it degrades to importer.Default, which resolves
// standard library imports only.
func importerFor(fset *token.FileSet, f *ast.File) types.Importer {
	if imp := moduleImporter(filepath.Dir(fset.Position(f.Pos()).Filename)); imp != nil {
		return imp
	}
	return importer.Default()
}

// moduleImporters memoizes one importer per module root. A whole-repo check
// walks hundreds of directories of the same module; without the cache each
// directory would pay a full go list and export-data load again. Values are
// either types.Importer or a failedMarker (load attempted and failed).
var moduleImporters sync.Map

var moduleImportMu sync.Mutex

type failedMarker struct{}

// moduleImporter returns the module's importer, loading it once per module
// root. It returns nil when dir is outside a module or the load fails, so the
// caller falls back to the stdlib-only importer.
func moduleImporter(dir string) types.Importer {
	root := moduleRoot(dir)
	if root == "" {
		return nil
	}
	if v, ok := moduleImporters.Load(root); ok {
		if _, failed := v.(failedMarker); failed {
			return nil
		}
		return v.(types.Importer)
	}
	moduleImportMu.Lock()
	defer moduleImportMu.Unlock()
	if v, ok := moduleImporters.Load(root); ok {
		if _, failed := v.(failedMarker); failed {
			return nil
		}
		return v.(types.Importer)
	}
	imp := loadModuleImporter(root)
	if imp == nil {
		moduleImporters.Store(root, failedMarker{})
	} else {
		moduleImporters.Store(root, imp)
	}
	return imp
}

// moduleRoot walks up from dir to the nearest go.mod, or "" when dir sits
// outside any module.
func moduleRoot(dir string) string {
	for {
		if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// loadModuleImporter asks go list for the module's package graph once and
// decodes every dependency's export data. Export data is what makes this
// cheap: go/packages would otherwise type-check the whole dependency graph
// from source, which takes minutes on a real module.
func loadModuleImporter(root string) types.Importer {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps |
			packages.NeedExportFile | packages.NeedTypesSizes,
		Dir: root,
	}, "./...")
	if err != nil || len(pkgs) == 0 {
		return nil
	}
	imports := map[string]*types.Package{}
	fset := token.NewFileSet()
	for _, p := range reachable(pkgs) {
		readExport(imports, fset, p)
	}
	if len(imports) == 0 {
		return nil
	}
	return mapImporter(imports)
}

// reachable collects every package in the loaded graph, deduplicated and in
// depth-first order.
func reachable(pkgs []*packages.Package) []*packages.Package {
	seen := map[*packages.Package]bool{}
	var out []*packages.Package
	var walk func(*packages.Package)
	walk = func(p *packages.Package) {
		if seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
		for _, imp := range p.Imports {
			walk(imp)
		}
	}
	for _, p := range pkgs {
		walk(p)
	}
	return out
}

// readExport decodes one package's export data into imports, sharing the map
// gcexportdata needs so references between packages stay consistent.
func readExport(imports map[string]*types.Package, fset *token.FileSet, p *packages.Package) {
	if p.ExportFile == "" || p.PkgPath == "" {
		return
	}
	if _, ok := imports[p.PkgPath]; ok {
		return
	}
	file, err := os.Open(p.ExportFile)
	if err != nil {
		return
	}
	defer file.Close()
	t, err := gcexportdata.Read(file, fset, imports, p.PkgPath)
	if err == nil && t != nil {
		imports[p.PkgPath] = t
	}
}

type mapImporter map[string]*types.Package

func (m mapImporter) Import(path string) (*types.Package, error) {
	if p, ok := m[path]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("no type information for import %q", path)
}
