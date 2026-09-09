package filet

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// CognitiveTS scores the cognitive complexity of a function body on the parse
// tree, mirroring the SonarSource metric used for Go in cognitive.go: a switch
// costs one point rather than one per arm, and every construct is penalised by
// how deeply it is nested.
func CognitiveTS(body *tree_sitter.Node, src []byte) int {
	if body == nil {
		return 0
	}
	c := &tsScorer{src: src}
	c.walk(body, 0)
	return c.score
}

type tsScorer struct {
	score int
	src   []byte
}

func (c *tsScorer) walk(n *tree_sitter.Node, depth int) {
	switch n.Kind() {
	case "if_expression":
		c.score += 1 + depth + c.runs(conditionOf(n))
		c.walkIfBody(n, depth+1)
		if e := childKind(n, "else_clause"); e != nil {
			c.elseBranch(e, depth)
		}
	case "for_expression", "while_expression", "loop_expression", "match_expression":
		c.score += 1 + depth
		c.children(n, depth+1)
	default:
		c.children(n, depth)
	}
}

// walkIfBody descends into the if's own branch, leaving the else to elseBranch
// so an else-if chain does not pay for its own nesting twice.
func (c *tsScorer) walkIfBody(n *tree_sitter.Node, depth int) {
	for i := 0; i < ncount(n); i++ {
		if child := childAt(n, i); child.Kind() != "else_clause" {
			c.walk(child, depth)
		}
	}
}

func (c *tsScorer) elseBranch(e *tree_sitter.Node, depth int) {
	if first := childAt(e, 0); first != nil && first.Kind() == "if_expression" {
		c.score += 1 + c.runs(conditionOf(first))
		c.walkIfBody(first, depth+1)
		if nested := childKind(first, "else_clause"); nested != nil {
			c.elseBranch(nested, depth)
		}
		return
	}
	c.score++
	c.children(e, depth+1)
}

func (c *tsScorer) children(n *tree_sitter.Node, depth int) {
	for i := 0; i < ncount(n); i++ {
		c.walk(childAt(n, i), depth)
	}
}

// conditionOf returns the expression guarding an if: the first named child that
// is not its own block or an else clause.
func conditionOf(n *tree_sitter.Node) *tree_sitter.Node {
	for i := 0; i < ncount(n); i++ {
		if c := childAt(n, i); !isBlock(c.Kind()) && c.Kind() != "else_clause" {
			return c
		}
	}
	return n
}

// runs counts the maximal runs of && and || in a condition: a && b && c is one
// run, a && b || c is two, matching the metric's treatment of logical runs.
func (c *tsScorer) runs(n *tree_sitter.Node) int {
	var count int
	var prev string
	stack := []*tree_sitter.Node{n}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if op := binOp(cur, c.src); op == "&&" || op == "||" {
			if op != prev {
				count++
			}
			prev = op
		} else {
			prev = ""
		}
		for i := 0; i < ncount(cur); i++ {
			stack = append(stack, childAt(cur, i))
		}
	}
	return count
}

// binOp returns the && or || operator of a binary_expression, or "".
func binOp(n *tree_sitter.Node, src []byte) string {
	for i := 0; i < int(n.ChildCount()); i++ {
		if t := n.Child(uint(i)).Utf8Text(src); t == "&&" || t == "||" {
			return t
		}
	}
	return ""
}
