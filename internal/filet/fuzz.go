package filet

import (
	"cmp"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// FuzzTarget is one fuzz entry point: a Go `func Fuzz*(f *testing.F)` in the
// package at Dir, or a cargo-fuzz target in the crate rooted at Dir.
type FuzzTarget struct {
	Lang string `json:"lang"`
	Name string `json:"name"`
	Dir  string `json:"dir"`
}

// DiscoverFuzzTargets walks target, honouring the config's ignore list, and
// returns every Go and Rust fuzz target it finds, sorted by language, directory
// and name.
func DiscoverFuzzTargets(cfg *Config, target string) ([]FuzzTarget, error) {
	out := []FuzzTarget{}
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case d.IsDir():
			return skipDir(cfg, path, target, d.Name())
		case strings.HasSuffix(path, "_test.go"):
			out = append(out, goFuzzTargets(path)...)
		case isRustFuzzTarget(path):
			crate := filepath.Dir(filepath.Dir(filepath.Dir(path)))
			out = append(out, FuzzTarget{Lang: "rust", Name: strings.TrimSuffix(d.Name(), ".rs"), Dir: crate})
		}
		return nil
	})
	slices.SortFunc(out, func(a, b FuzzTarget) int {
		return cmp.Or(cmp.Compare(a.Lang, b.Lang), cmp.Compare(a.Dir, b.Dir), cmp.Compare(a.Name, b.Name))
	})
	return out, err
}

func isRustFuzzTarget(path string) bool {
	parent := filepath.Dir(path)
	return filepath.Ext(path) == ".rs" &&
		filepath.Base(parent) == "fuzz_targets" &&
		filepath.Base(filepath.Dir(parent)) == "fuzz"
}

func goFuzzTargets(path string) []FuzzTarget {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	var out []FuzzTarget
	for _, name := range fuzzFuncs(file) {
		out = append(out, FuzzTarget{Lang: "go", Name: name, Dir: filepath.Dir(path)})
	}
	return out
}

func fuzzFuncs(file *ast.File) []string {
	var out []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Fuzz") && takesTestingF(fn) {
			out = append(out, fn.Name.Name)
		}
	}
	return out
}

func takesTestingF(fn *ast.FuncDecl) bool {
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) > 1 {
		return false
	}
	star, ok := params[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "testing" && sel.Sel.Name == "F"
}
