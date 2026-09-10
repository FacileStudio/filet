package filet

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func noiseLines(f SourceFile, parsed *ast.File, fset *token.FileSet) map[int]bool {
	noise := map[int]bool{}
	for i, l := range f.Lines {
		if strings.TrimSpace(l) == "" {
			noise[i+1] = true
		}
	}
	for _, group := range parsed.Comments {
		markComments(group, f, fset, noise)
	}
	return noise
}

func markComments(group *ast.CommentGroup, f SourceFile, fset *token.FileSet, noise map[int]bool) {
	for _, c := range group.List {
		start, end := fset.Position(c.Pos()), fset.Position(c.End())
		for n := start.Line; n <= end.Line; n++ {
			if n != start.Line || ownsLine(f.Lines, n, start.Column) {
				noise[n] = true
			}
		}
	}
}

func ownsLine(lines []string, n, column int) bool {
	line := cut(lines, n)
	end := min(max(0, column-1), len(line))
	return strings.TrimSpace(line[:end]) == ""
}

// inBodyCommentLines returns the 1-based line numbers of the comments inside a
// Go function body that go.comment.inbody would flag: comment groups that sit
// inside a top-level function body and carry no tool directive. clean removes
// exactly the lines such a comment owns, so a cleaned file stops reporting the
// finding. It is best-effort exactly like formatting: a file the Go parser
// cannot read is left for the check to surface and clean does not fail.
func inBodyCommentLines(cfg *Config, f SourceFile) map[int]bool {
	drop := map[int]bool{}
	if !cfg.Style.BanInlineComments || !cfg.Enabled("go.comment.inbody") || f.Ext != ".go" {
		return drop
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, f.Path, f.Src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return drop
	}
	bodies := funcBodies(parsed)
	for _, group := range parsed.Comments {
		if carriesDirective(group) || !insideAny(group, bodies) {
			continue
		}
		markComments(group, f, fset, drop)
	}
	return drop
}
