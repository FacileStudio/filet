package filet

import (
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func runGoChecks(cfg *Config, f SourceFile, fset *token.FileSet, parsed *ast.File) []Finding {
	info := typeInfoFor(cfg, fset, []*ast.File{parsed})
	out := runGoChecksInfo(cfg, f, fset, parsed, info)
	if cfg.Enabled("go.leak.resource") && info == nil {
		out = append(out, leakSkipFinding(f))
	}
	return out
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
	if cfg.Enabled("go.leak.resource") && info != nil {
		g.resourceLeaks()
	}
	return out
}

// leakSkipFinding reports, once per package, that the leak rule could not run.
// It is not a defect of the checked code: the checker failed to resolve the
// package's imports, so it says so instead of staying silent or blaming the
// file.
func leakSkipFinding(f SourceFile) Finding {
	return newFinding("go.leak.resource", f.Display, 1, Info,
		"resource-leak check skipped: package type information could not be resolved")
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
// produced type info, or nil when nothing usable resolved. Module imports are
// resolved through go list (via go/packages); a directory outside a module
// falls back to the stdlib-only default importer, which still types files
// importing nothing but the standard library. Each file of a package is passed
// the same result so cross-file symbols are available to the leak rule.
func typeInfoFor(cfg *Config, fset *token.FileSet, files []*ast.File) *types.Info {
	if !cfg.Enabled("go.leak.resource") || len(files) == 0 {
		return nil
	}
	if goRoot() == "" {
		return nil
	}
	ensureGoRoot()
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := &types.Config{Importer: importerFor(fset, files[0]), Error: func(err error) {}}
	if _, err := conf.Check(files[0].Name.Name, fset, files, info); err != nil && len(info.Types) == 0 {
		return nil
	}
	return info
}

// ensureGoRoot points the shared build context at the resolved toolchain exactly
// once, so the standard-library importer in typeInfoFor can resolve the stdlib
// and so concurrent package checks never race on the global GOROOT slot.
var ensureGoRoot = sync.OnceFunc(func() {
	if build.Default.GOROOT == "" {
		build.Default.GOROOT = goRoot()
	}
})

var goRootFn = sync.OnceValue(func() string {
	if bin, err := exec.LookPath("go"); err == nil {
		if out, err := exec.Command(bin, "env", "GOROOT").Output(); err == nil {
			if g := strings.TrimSpace(string(out)); g != "" {
				return g
			}
		}
	}
	return os.Getenv("GOROOT")
})

// goRoot returns a GOROOT the standard-library importer can use. It is derived
// from `go env GOROOT`, the one answer that stays correct when a snapshot or
// -trimpath binary is copied to a machine where runtime.GOROOT no longer points
// at the toolchain it was built with. The GOROOT environment variable is the
// fallback when no go binary is on the path. An empty result means the stdlib
// cannot be resolved and the leak rule degrades to its visible skip finding
// instead of silently missing leaks.
func goRoot() string { return goRootFn() }
