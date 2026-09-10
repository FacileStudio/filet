package filet

import (
	"fmt"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// CheckTree runs the parse-tree shape rules over one file. It replaces the
// brace-counting approximations in CheckGeneric for any language that has a
// grammar, so a finding is anchored to real syntax rather than character counts.
func CheckTree(cfg *Config, f SourceFile) []Finding {
	g, ok := GrammarFor(f.Ext)
	if !ok {
		return nil
	}
	t := ParseTS(g, f.Src)
	if t == nil {
		return nil
	}
	defer t.Close()
	root := t.Root()

	ts := &tsFile{cfg: cfg, f: f}
	ts.nesting(root)
	if ts.max > cfg.Limits.Nesting && cfg.Enabled("ts.nesting") {
		ts.add("ts.nesting", ts.line, Warn,
			fmt.Sprintf("nesting reaches depth %d (limit %d)", ts.max, cfg.Limits.Nesting))
	}
	ts.walkFuncs(root)
	return ts.out
}

type tsFile struct {
	cfg     *Config
	f       SourceFile
	out     []Finding
	depth   int
	max     int
	line    int
	comment map[int]bool
}

// nesting records the deepest run of block nodes in a file and flags when it
// exceeds the limit. Counting block nodes, not braces, ignores braces inside
// strings and composite literals that the text scanner has to special-case.
func (s *tsFile) nesting(n *tree_sitter.Node) {
	kind := n.Kind()
	if isBlock(kind) {
		s.depth++
		if s.depth > s.max {
			s.max = s.depth
			s.line = 1 + int(n.StartPosition().Row)
		}
	}
	for i := 0; i < ncount(n); i++ {
		s.nesting(childAt(n, i))
	}
	if isBlock(kind) {
		s.depth--
	}
}

// walkFuncs walks the tree and runs the shape rules on every function body.
func (s *tsFile) walkFuncs(n *tree_sitter.Node) {
	if n.Kind() == "function_item" {
		s.function(n)
		return
	}
	for i := 0; i < ncount(n); i++ {
		s.walkFuncs(childAt(n, i))
	}
}

func (s *tsFile) function(fn *tree_sitter.Node) {
	body := childKind(fn, "block")
	if body == nil {
		return
	}
	name, _ := fnName(fn, s.f.Src)
	lim := s.cfg.Limits
	line := 1 + int(fn.StartPosition().Row)

	if lim.FuncLines > 0 && s.linesOf(body) > lim.FuncLines {
		s.add("ts.func.long", line, Warn,
			fmt.Sprintf("%s is %d lines of code (limit %d)", name, s.linesOf(body), lim.FuncLines))
	}
	if lim.FuncStatements > 0 && ncount(body) > lim.FuncStatements {
		s.add("ts.func.statements", line, Warn,
			fmt.Sprintf("%s has %d statements (limit %d)", name, ncount(body), lim.FuncStatements))
	}
	if lim.Params > 0 && parmsCount(fn) > lim.Params {
		s.add("ts.func.params", line, Warn,
			fmt.Sprintf("%s takes %d parameters (limit %d)", name, parmsCount(fn), lim.Params))
	}
	if lim.Complexity > 0 {
		if nc := CognitiveTS(body, s.f.Src); nc > lim.Complexity {
			s.add("ts.func.complexity", line, Warn,
				fmt.Sprintf("%s has cognitive complexity %d (limit %d)", name, nc, lim.Complexity))
		}
	}
}

// parmsCount is the number of a function's parameters, or 0 when it has none.
func parmsCount(fn *tree_sitter.Node) int {
	if params := childKind(fn, "parameters"); params != nil {
		return ncount(params)
	}
	return 0
}

// linesOf counts the lines of a body that a reader holds in mind: non-blank,
// non-comment lines between the body's first and last line.
func (s *tsFile) linesOf(body *tree_sitter.Node) int {
	first, last := 1+int(body.StartPosition().Row), 1+int(body.EndPosition().Row)
	s.comment = map[int]bool{}
	s.markComments(body)
	n := 0
	for l := first + 1; l < last; l++ {
		if s.comment[l] || strings.TrimSpace(s.f.Lines[l-1]) == "" {
			continue
		}
		n++
	}
	return n
}

func (s *tsFile) markComments(n *tree_sitter.Node) {
	if isComment(n.Kind()) {
		start, end := int(n.StartPosition().Row), int(n.EndPosition().Row)
		for l := start; l <= end; l++ {
			s.comment[1+l] = true
		}
	}
	for i := 0; i < ncount(n); i++ {
		s.markComments(childAt(n, i))
	}
}

func (s *tsFile) add(rule string, line int, sev Severity, msg string) {
	if s.cfg.Enabled(rule) {
		s.out = append(s.out, newFinding(rule, s.f.Display, line, sev, msg))
	}
}
