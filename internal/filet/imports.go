package filet

import (
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
)

var jsImportRe = regexp.MustCompile(`(?m)(?:from\s+|require\(\s*|import\(\s*)['"]([^'"]+)['"]`)

func imports(f SourceFile) []string {
	if f.Ext == ".go" {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f.Path, f.Src, parser.ImportsOnly|parser.SkipObjectResolution)
		if err != nil {
			return nil
		}
		out := make([]string, 0, len(file.Imports))
		for _, i := range file.Imports {
			if p, err := strconv.Unquote(i.Path.Value); err == nil {
				out = append(out, p)
			}
		}
		return out
	}
	var out []string
	for _, m := range jsImportRe.FindAllStringSubmatch(string(f.Src), -1) {
		out = append(out, m[1])
	}
	return out
}
