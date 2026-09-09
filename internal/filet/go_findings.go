package filet

import (
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"strings"
)

func runGoChecks(cfg *Config, f SourceFile, fset *token.FileSet, parsed *ast.File) []Finding {
	return runGoChecksInfo(cfg, f, fset, parsed, typeInfoFor(cfg, fset, []*ast.File{parsed}))
}

// runGoChecksInfo runs the AST rules over one parsed file, carrying the type
// info produced for its whole package. checkGoFiles calls it once per file of a
// package with the same info so go.leak.resource sees symbols from sibling
// files. It is partial-safe: go/types keeps filling info even when some import
// cannot be resolved, so the leak rule runs on whatever did resolve.
func runGoChecksInfo(cfg *Config, f SourceFile, fset *token.FileSet, parsed *ast.File, info *types.Info) []Finding {
	var out []Finding
	g := &goFile{
		cfg:    cfg,
		file:   parsed,
		isTest: strings.HasSuffix(f.Rel, "_test.go"),
		line:   func(p token.Pos) int { return fset.Position(p).Line },
		noise:  noiseLines(f, parsed, fset),
		info:   info,
		add:    makeGoFindingsAdder(cfg, f, fset, &out),
	}
	g.decls()
	g.inBodyComments()
	g.calls()
	g.nilerrCheck()
	if cfg.Enabled("go.leak.resource") {
		if info == nil {
			out = append(out, newFinding("go.leak.resource", f.Display, 1, Info,
				"could not run the resource-leak check: type information is unavailable for this file"))
		} else {
			g.resourceLeaks()
		}
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

// typeInfoFor type-checks the given files as one package and returns the
// produced type info, or nil when nothing usable resolved. go/types fills
// info with everything it can even when Check returns an error (an
// unresolvable third-party import, an undefined name) — only a check that
// resolves nothing yields nil. Each file of a package is passed the same
// result so cross-file symbols are available to the leak rule.
func typeInfoFor(cfg *Config, fset *token.FileSet, files []*ast.File) *types.Info {
	if !cfg.Enabled("go.leak.resource") || len(files) == 0 {
		return nil
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := &types.Config{Importer: importer.Default(), Error: func(err error) {}}
	if _, err := conf.Check(files[0].Name.Name, fset, files, info); err != nil && len(info.Types) == 0 {
		return nil
	}
	return info
}
