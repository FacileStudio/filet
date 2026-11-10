package filet

import (
	"go/ast"
	"slices"
	"strings"
)

func fuzzHints(cfg *Config, pd parsedDir) []Finding {
	if !cfg.Enabled("test.fuzz.missing") || !cfg.hintFuzz() {
		return nil
	}
	if slices.ContainsFunc(pd.files, func(f *ast.File) bool { return len(fuzzFuncs(f)) > 0 }) {
		return nil
	}
	var out []Finding
	for _, file := range pd.files {
		src := pd.byPath[pd.fset.Position(file.Package).Filename]
		if strings.HasSuffix(src.Path, "_test.go") || src.IsGenerated() {
			continue
		}
		for _, fn := range fuzzableFuncs(file) {
			at := pd.fset.Position(fn.Name.Pos())
			finding := newFinding("test.fuzz.missing", src.Display, at.Line, Info,
				fn.Name.Name+" reads untrusted input and its package has no Fuzz target")
			finding.Column = at.Column
			out = append(out, finding)
		}
	}
	return out
}

func fuzzableFuncs(file *ast.File) []*ast.FuncDecl {
	var out []*ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && hasParserName(fn.Name.Name) && takesRawInput(fn) {
			out = append(out, fn)
		}
	}
	return out
}

func hasParserName(name string) bool {
	for _, prefix := range []string{"Parse", "Decode", "Unmarshal"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func takesRawInput(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	switch t := fn.Type.Params.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name == "string"
	case *ast.ArrayType:
		elt, ok := t.Elt.(*ast.Ident)
		return t.Len == nil && ok && elt.Name == "byte"
	case *ast.SelectorExpr:
		pkg, ok := t.X.(*ast.Ident)
		return ok && pkg.Name == "io" && t.Sel.Name == "Reader"
	}
	return false
}
