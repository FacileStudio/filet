package filet

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

type panickingImporter struct{}

func (panickingImporter) Import(string) (*types.Package, error) {
	panic(`cannot decode "context", export data version 4 is greater than maximum supported version 2`)
}

// TestCheckTypesSurvivesAnImporterPanic guards the crash a filet built with an
// older Go hit on a tree whose export data a newer toolchain had written.
func TestCheckTypesSurvivesAnImporterPanic(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "a.go", "package a\n\nimport \"context\"\n\nvar _ context.Context\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	conf := &types.Config{Importer: panickingImporter{}, Error: func(error) {}}
	if checkTypes(conf, fset, []*ast.File{file}, info) {
		t.Fatal("a panicking importer must report no usable type info")
	}
}

// TestCheckTypesReportsUsableInfo keeps the ordinary path honest.
func TestCheckTypesReportsUsableInfo(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "a.go", "package a\n\nvar x = 1\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	conf := &types.Config{Error: func(error) {}}
	if !checkTypes(conf, fset, []*ast.File{file}, info) {
		t.Fatal("a file with no imports must type-check")
	}
}
