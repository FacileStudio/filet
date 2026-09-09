package filet

import (
	"slices"
	"strings"
	"testing"
)

func tsConfig(t *testing.T, limits Limits) *Config {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Treesitter.Enabled = true
	cfg.Limits = limits
	return cfg
}

func TestTreesitterNestingAndComplexity(t *testing.T) {
	cfg := tsConfig(t, Limits{
		Nesting: 2, Complexity: 5, FuncLines: 20, FuncStatements: 25, Params: 2,
	})
	f := source(t, "deep.rs", strings.Join([]string{
		"fn outer(a: i32, b: i32) -> i32 {",
		"    if a > 0 {",
		"        if a > 1 {",
		"            if a > 2 {",
		"                return a + b;",
		"            }",
		"        }",
		"    }",
		"    return b;",
		"}",
	}, "\n"))

	got := CheckTree(cfg, f)
	if !slices.Contains(ruleIDs(got), "ts.nesting") {
		t.Fatal("expected ts.nesting for depth 4")
	}
	if !slices.Contains(ruleIDs(got), "ts.func.complexity") {
		t.Fatal("expected ts.func.complexity for dense nesting")
	}
	for _, x := range got {
		if x.Rule == "ts.nesting" && x.Line != 4 {
			t.Fatalf("ts.nesting should anchor at the deepest block, got line %d", x.Line)
		}
	}
}

func TestTreesitterFuncParamsAndStatements(t *testing.T) {
	cfg := tsConfig(t, Limits{Nesting: 6, Complexity: 20, FuncLines: 20, FuncStatements: 2, Params: 2})
	f := source(t, "wide.rs", strings.Join([]string{
		"fn take(a: i32, b: i32, c: i32) -> i32 {",
		"    let x = a + b;",
		"    let y = b + c;",
		"    let z = x + y;",
		"    return c;",
		"}",
	}, "\n"))

	got := CheckTree(cfg, f)
	if !slices.Contains(ruleIDs(got), "ts.func.params") {
		t.Fatal("expected ts.func.params for 3 parameters past a limit of 2")
	}
	if !slices.Contains(ruleIDs(got), "ts.func.statements") {
		t.Fatal("expected ts.func.statements for 4 statements past a limit of 2")
	}
}

func TestTreesitterOwnsNestingWhenEnabled(t *testing.T) {
	cfg := tsConfig(t, Limits{Nesting: 2, Complexity: 5, FuncLines: 20, FuncStatements: 25, Params: 2})
	f := source(t, "deep.rs", strings.Join([]string{
		"fn outer() {",
		"    if a {",
		"        if b {",
		"            run();",
		"        }",
		"    }",
		"}",
	}, "\n"))

	cfg.Treesitter.Enabled = true
	generic := CheckGeneric(cfg, f)
	if slices.Contains(ruleIDs(generic), "gen.nesting") {
		t.Fatal("with treesitter on, gen.nesting must defer to ts.nesting, not double-count")
	}

	cfg.Treesitter.Enabled = false
	generic = CheckGeneric(cfg, f)
	if !slices.Contains(ruleIDs(generic), "gen.nesting") {
		t.Fatal("with treesitter off, the brace-based gen.nesting must come back")
	}
}

func TestTreesitterOnByDefault(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Treesitter.Enabled {
		t.Fatal("treesitter is deterministic and offline, so it must default on")
	}
	if !cfg.UsesTreesitter(".rs") {
		t.Fatal("with the tier on, a vendored-grammar file must route through the parse tree")
	}
	if cfg.UsesTreesitter(".ts") {
		t.Fatal("a language without a vendored grammar must keep the brace-counting rules")
	}
}
