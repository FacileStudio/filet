package filet

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
)

// checkGoFiles runs the Go rules over goFiles, type-checking each package as
// one unit (shared FileSet, shared types.Info) so a file can resolve names its
// siblings define. Without this, single-file type checks hit "undefined: X"
// the moment any symbol comes from another file in the package.
func checkGoFiles(cfg *Config, goFiles []SourceFile) []Finding {
	byDir := map[string][]SourceFile{}
	var dirs []string
	for _, f := range goFiles {
		d := filepath.Dir(f.Path)
		if _, ok := byDir[d]; !ok {
			dirs = append(dirs, d)
		}
		byDir[d] = append(byDir[d], f)
	}
	sort.Strings(dirs)

	var out []Finding
	for _, dir := range dirs {
		out = append(out, checkGoDir(cfg, byDir[dir])...)
	}
	return out
}

// checkGoDir parses the files of one directory, types each package present and
// runs the per-file Go rules with the package's shared type info. Parse errors
// are reported and the offending file skipped.
func checkGoDir(cfg *Config, files []SourceFile) []Finding {
	pd := parseGoFiles(files)
	out := pd.findings
	for _, grp := range groupByPackage(pd.files) {
		info := typeInfoFor(cfg, pd.fset, grp.files)
		out = append(out, checkGoGroup(cfg, pd.fset, pd.byPath, grp, info)...)
	}
	return out
}

func checkGoGroup(cfg *Config, fset *token.FileSet, byPath map[string]SourceFile, grp pkgGroup, info *types.Info) []Finding {
	var out []Finding
	for _, parsed := range grp.files {
		f := byPath[fset.Position(parsed.Name.Pos()).Filename]
		if f.IsGenerated() {
			continue
		}
		out = append(out, runGoChecksInfo(cfg, f, fset, parsed, info)...)
	}
	return out
}

func parseGoFiles(files []SourceFile) parsedDir {
	pd := parsedDir{
		byPath: map[string]SourceFile{},
		fset:   token.NewFileSet(),
	}
	for _, f := range files {
		pd.byPath[f.Path] = f
		parsed, err := parser.ParseFile(pd.fset, f.Path, f.Src, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			pd.findings = append(pd.findings, newFinding("go.parse", f.Display, 1, Error, "cannot parse: "+err.Error()))
			continue
		}
		pd.files = append(pd.files, parsed)
	}
	return pd
}

// parsedDir is the result of parsing every file in one directory: the shared
// FileSet the type-checker needs plus the parsed files and per-file parse
// errors, keyed by path.
type parsedDir struct {
	fset     *token.FileSet
	byPath   map[string]SourceFile
	files    []*ast.File
	findings []Finding
}

// pkgGroup is one distinct package in a directory. A directory can hold several
// (a *_test external package); each must be typed under its own name.
type pkgGroup struct {
	files []*ast.File
}

// groupByPackage partitions parsed files by declared package name, ordered
// deterministically by name.
func groupByPackage(all []*ast.File) []pkgGroup {
	byName := map[string][]*ast.File{}
	var names []string
	for _, p := range all {
		name := p.Name.Name
		if _, ok := byName[name]; !ok {
			names = append(names, name)
		}
		byName[name] = append(byName[name], p)
	}
	sort.Strings(names)
	groups := make([]pkgGroup, 0, len(names))
	for _, name := range names {
		groups = append(groups, pkgGroup{byName[name]})
	}
	return groups
}
