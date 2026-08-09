package filet

import (
	"go/ast"
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
