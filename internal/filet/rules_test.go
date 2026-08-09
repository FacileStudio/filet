package filet

import (
	"slices"
	"testing"
)

func TestGoRulesCatchTheObviousSins(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Limits.Params = 2
	cfg.Limits.Complexity = 2

	f := source(t, "bad.go", `package bad

var Shared = 1

func init() {}

func Tangle(a, b, c int) (n int, err error) {
	if a > 0 && b > 0 || c > 0 {
		for i := 0; i < a; i++ {
			if i == b {
				return
			}
		}
	}
	_ = doThing()
	panic("nope")
}

func doThing() error { return nil }
`)

	got := ruleIDs(CheckGo(cfg, f))
	for _, want := range []string{
		"go.global.mutable", "go.init", "go.func.params", "go.func.complexity",
		"go.return.naked", "go.err.discarded", "go.panic", "go.doc.missing",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("missing rule %s in %v", want, got)
		}
	}
}

func TestUnwrittenLookupTableIsNotMutableState(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "table.go", `package table

var lookup = map[string]int{"a": 1}

var Exported = map[string]int{"a": 1}

func get(k string) int { return lookup[k] }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.global.mutable"); n != 1 {
		t.Fatalf("only the exported var should be flagged, got %d in %v", n, ruleIDs(got))
	}
}

func TestCleanGoFileIsClean(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "good.go", `package good

// Add returns the sum of a and b.
func Add(a, b int) int { return a + b }
`)
	if got := CheckGo(cfg, f); len(got) != 0 {
		t.Fatalf("expected no findings, got %v", ruleIDs(got))
	}
}

func TestFuncLinesIgnoresCommentsAndBlanks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Limits.FuncLines = 3
	f := source(t, "long.go", `package long

func f() {
	a()

	// a comment that should not count
	// nor should this one

	b()
	c()
}
`)
	if n := countRule(CheckGo(cfg, f), "go.func.long"); n != 0 {
		t.Fatal("3 statements padded with comments and blanks must not trip a 3-line limit")
	}

	cfg.Limits.FuncLines = 2
	if n := countRule(CheckGo(cfg, f), "go.func.long"); n != 1 {
		t.Fatal("3 lines of actual code must trip a 2-line limit")
	}
}
