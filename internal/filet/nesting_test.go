package filet

import (
	"slices"
	"strings"
	"testing"
)

func TestNestingUsesBraceDepthNotStringContents(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Limits.Nesting = 2
	f := source(t, "nest.ts", strings.Join([]string{
		`const brace = "{{{{{{";`,
		`function a() {`,
		`  if (x) {`,
		`    while (y) {`,
		`      z();`,
		`    }`,
		`  }`,
		`}`,
	}, "\n"))

	if !slices.Contains(ruleIDs(CheckGeneric(cfg, f)), "gen.nesting") {
		t.Fatal("expected gen.nesting for depth 3")
	}

	cfg.Limits.Nesting = 3
	if slices.Contains(ruleIDs(CheckGeneric(cfg, f)), "gen.nesting") {
		t.Fatal("depth 3 must not trip a limit of 3")
	}
}

func TestRawStringBracesDoNotCount(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Limits.Nesting = 2
	body := strings.Join([]string{
		"package raw",
		"",
		"var tpl = `",
		"if a {",
		"\tif b {",
		"\t\tif c {",
		"\t\t}",
		"\t}",
		"}",
		"`",
	}, "\n")

	if n := countRule(CheckGeneric(cfg, source(t, "raw.go", body)), "gen.nesting"); n != 0 {
		t.Fatal("braces inside a multi-line raw string must not count as nesting")
	}
}

func TestClosingDeeperThanTheLineOpensDoesNotInflateDepth(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Limits.Nesting = 4
	body := strings.Join([]string{
		"package table",
		"",
		"var cases = []struct {",
		"\ta string",
		"\tb string",
		"}{{\"x\", \"y\"}, {\"z\", \"w\"}}",
		"",
		"func f() {",
		"\tfor _, c := range cases {",
		"\t\tif c.a != \"\" {",
		"\t\t\tprintln(c.b)",
		"\t\t}",
		"\t}",
		"}",
	}, "\n")

	if n := countRule(CheckGeneric(cfg, source(t, "table.go", body)), "gen.nesting"); n != 0 {
		t.Fatal("a table-driven literal must not leave the depth counter raised for the rest of the file")
	}
}

func TestBracePairsClosedOnTheirOwnLineAreNotNesting(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Limits.Nesting = 3
	body := strings.Join([]string{
		"package lit",
		"",
		"func f(ok bool) []string {",
		"\tswitch {",
		"\tcase ok:",
		"\t\tif ok {",
		"\t\t\treturn []string{\"a\", \"b\"}",
		"\t\t}",
		"\t}",
		"\treturn nil",
		"}",
	}, "\n")

	if n := countRule(CheckGeneric(cfg, source(t, "lit.go", body)), "gen.nesting"); n != 0 {
		t.Fatal("a composite literal opened and closed on one line is data, not a block")
	}
}

func TestFuncsPerFileGo(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Limits.FuncsPerFile = 2
	f := source(t, "many.go", `package many

func a() {}
func b() {}
func c() {}
func d() {}
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.file.funcs"); n != 1 {
		t.Fatalf("expected exactly one go.file.funcs finding, got %d in %v", n, ruleIDs(got))
	}
	for _, x := range got {
		if x.Rule == "go.file.funcs" && x.Line != 5 {
			t.Fatalf("expected the finding on the third function, got line %d", x.Line)
		}
	}

	cfg.Limits.FuncsPerFile = 0
	if n := countRule(CheckGo(cfg, f), "go.file.funcs"); n != 0 {
		t.Fatal("funcsPerFile 0 must disable the rule")
	}
}

func TestFuncsPerFileGeneric(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Limits.FuncsPerFile = 2
	f := source(t, "many.ts", `export function a() {}
const b = async (x: number) => x;
function c() {}
`)
	if n := countRule(CheckGeneric(cfg, f), "gen.file.funcs"); n != 1 {
		t.Fatalf("expected gen.file.funcs for the third TS function, got %d", n)
	}

	cfg.Limits.FuncsPerFile = 0
	if n := countRule(CheckGeneric(cfg, f), "gen.file.funcs"); n != 0 {
		t.Fatal("funcsPerFile 0 must disable the rule")
	}
}
