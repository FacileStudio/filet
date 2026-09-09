package filet

import (
	"slices"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

var blockKinds = []string{"block", "unsafe_block", "async_block", "try_block"}

// ncount is a node's named child count as a native int.
func ncount(n *tree_sitter.Node) int { return int(n.NamedChildCount()) }

func childAt(n *tree_sitter.Node, i int) *tree_sitter.Node { return n.NamedChild(uint(i)) }

// childKind finds the first direct named child with the given kind, or nil.
func childKind(n *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := 0; i < ncount(n); i++ {
		if c := childAt(n, i); c.Kind() == kind {
			return c
		}
	}
	return nil
}

// isBlock reports whether a node kind opens a nested block scope.
func isBlock(kind string) bool {
	return slices.Contains(blockKinds, kind)
}

// isComment reports whether a node kind is a comment of any flavour.
func isComment(kind string) bool {
	switch kind {
	case "line_comment", "block_comment", "doc_comment":
		return true
	}
	return false
}

// fnName returns the name text of a function declaration and its line.
func fnName(fn *tree_sitter.Node, src []byte) (string, int) {
	if name := fn.ChildByFieldName("name"); name != nil {
		return name.Utf8Text(src), 1 + int(name.StartPosition().Row)
	}
	for i := 0; i < ncount(fn); i++ {
		if c := childAt(fn, i); c.Kind() == "identifier" {
			return c.Utf8Text(src), 1 + int(c.StartPosition().Row)
		}
	}
	return "?", 1 + int(fn.StartPosition().Row)
}
