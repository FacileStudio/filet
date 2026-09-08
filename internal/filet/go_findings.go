package filet

import (
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"strings"
)

func runGoChecks(cfg *Config, f SourceFile, fset *token.FileSet, parsed *ast.File) []Finding {
	var out []Finding
	g := &goFile{
		cfg:    cfg,
		file:   parsed,
		isTest: strings.HasSuffix(f.Rel, "_test.go"),
		line:   func(p token.Pos) int { return fset.Position(p).Line },
		noise:  noiseLines(f, parsed, fset),
		info:   typeInfoFor(cfg, parsed, fset),
		add:    makeGoFindingsAdder(cfg, f, fset, &out),
	}
	g.decls()
	g.inBodyComments()
	g.calls()
	g.nilerrCheck()
	if g.info != nil {
		g.resourceLeaks()
	}
	return out
}

func makeGoFindingsAdder(cfg *Config, f SourceFile, fset *token.FileSet, out *[]Finding) addFn {
	return func(rule string, pos token.Pos, sev Severity, msg string) {
		if !cfg.Enabled(rule) {
			return
		}
		at := fset.Position(pos)
		finding := newFinding(rule, f.Display, at.Line, sev, msg)
		finding.Column = at.Column
		*out = append(*out, finding)
	}
}

func typeInfoFor(cfg *Config, parsed *ast.File, fset *token.FileSet) *types.Info {
	if !cfg.Enabled("go.leak.resource") {
		return nil
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := &types.Config{Importer: importer.Default(), Error: func(err error) {}}
	if _, err := conf.Check("", fset, []*ast.File{parsed}, info); err != nil {
		return nil
	}
	return info
}
