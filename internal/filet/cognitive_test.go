package filet

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func cognitiveOf(t *testing.T, body string) int {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", "package x\nfunc f(a, b, c bool) {\n"+body+"\n}\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	return Cognitive(f.Decls[0].(*ast.FuncDecl))
}

func TestCognitiveScoresNestingAndForgivesDispatch(t *testing.T) {
	cases := []struct {
		name string
		want int
		body string
	}{
		{"flat if", 1, "if a { g() }"},
		{"if else", 2, "if a { g() } else { h() }"},
		{"else if chain", 3, "if a { g() } else if b { h() } else { i() }"},
		{"nested if costs more", 3, "if a {\nif b { g() }\n}"},
		{"same operator is one run", 2, "if a && b && c { g() }"},
		{"mixed operators are two runs", 3, "if a && b || c { g() }"},
		{"switch is one point not one per case", 1,
			"switch {\ncase a:\ng()\ncase b:\nh()\ncase c:\ni()\n}"},
		{"nested loop and if", 6, "for a {\nfor b {\nif c { g() }\n}\n}"},
	}
	for _, tc := range cases {
		if got := cognitiveOf(t, tc.body); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestCognitiveRanksDispatchBelowNesting(t *testing.T) {
	dispatch := cognitiveOf(t, "switch {\ncase a:\ng()\ncase b:\nh()\ncase c:\ni()\ncase !a:\nj()\n}")
	nested := cognitiveOf(t, "if a {\nif b {\nif c { g() }\n}\n}")
	if dispatch >= nested {
		t.Fatalf("a 4-case switch (%d) must score below three nested ifs (%d)", dispatch, nested)
	}
}
