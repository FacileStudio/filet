package filet

import (
	"fmt"
	"strings"
	"testing"
)

func nestingDepth(t *testing.T, name string, lines ...string) int {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Limits.Nesting = 0
	found := CheckGeneric(cfg, source(t, name, strings.Join(lines, "\n")))
	for _, f := range found {
		if f.Rule == "gen.nesting" {
			var d, limit int
			if _, err := fmt.Sscanf(f.Message, "nesting reaches depth %d (limit %d)", &d, &limit); err != nil {
				t.Fatalf("unexpected message %q", f.Message)
			}
			return d
		}
	}
	return 0
}

func TestCommentBracesAreNotNesting(t *testing.T) {
	got := nestingDepth(t, "c.go",
		"package c",
		"",
		"func shallow() {",
		"\t// a comment with braces { { { {",
		"\tprintln(\"flat\")",
		"}",
	)
	if got != 1 {
		t.Fatalf("braces inside a line comment must not nest: got depth %d, want 1", got)
	}
}

func TestBlockCommentBracesAreNotNesting(t *testing.T) {
	got := nestingDepth(t, "c.js",
		"/**",
		" * @param {Object} opts { unbalanced",
		" */",
		"function flat() {",
		"\tif (a) {",
		"\t\tb();",
		"\t}",
		"}",
	)
	if got != 2 {
		t.Fatalf("braces inside a block comment must not nest: got depth %d, want 2", got)
	}
}

func TestUnpairedBacktickInCommentDoesNotSwallowTheFile(t *testing.T) {
	got := nestingDepth(t, "b.go",
		"package b",
		"",
		"// run it with `go test to see the output",
		"func deep() {",
		"\tif a {",
		"\t\tif b {",
		"\t\t\tprintln(1)",
		"\t\t}",
		"\t}",
		"}",
	)
	if got != 3 {
		t.Fatalf("a stray backtick in a comment must not hide the code after it: got depth %d, want 3", got)
	}
}

func TestRawStringSpansLines(t *testing.T) {
	got := nestingDepth(t, "r.go",
		"package r",
		"",
		"var tpl = `",
		"if a {",
		"\tif b {",
		"\t}",
		"}",
		"`",
		"",
		"func f() {",
		"\tprintln(1)",
		"}",
	)
	if got != 1 {
		t.Fatalf("braces inside a multi-line raw string must not nest: got depth %d, want 1", got)
	}
}

func TestRustRawStringDoesNotLookLikeInlineComment(t *testing.T) {
	cfg := DefaultConfig()
	for _, tc := range []struct {
		ext, body string
		want      int
	}{
		{".rs", "src = r\"// still literal\n\";\n", 0},
		{".rs", "src = r##\"// not a comment either\n\"##;\n", 0},
		{".rs", "let s = \"// still literal\";\n", 0},
		{".rs", "// HEADER\nstruct S {\n    a: i32,\n    // standalone comment\n    b: u64, // inline\n}\n", 1},
	} {
		f := source(t, "f"+tc.ext, tc.body)
		if n := countRule(CheckGeneric(cfg, f), "gen.comment.inline"); n != tc.want {
			t.Errorf("Rust inline-comment %q: got %d, want %d", tc.body, n, tc.want)
		}
	}
}

func TestStripLineKeepsCodeAndDropsLiterals(t *testing.T) {
	cases := []struct{ in, want string }{
		{`a := "he{llo"`, "a := "},
		{"x := `raw{`", "x := "},
		{`if c == '}' {`, "if c ==  {"},
		{`s := "esc\"aped{" + t`, "s :=  + t"},
		{`code() // trailing {`, "code() // trailing {"},
	}
	for _, tc := range cases {
		got, st := stripLine(tc.in, lineState{})
		if got != tc.want {
			t.Errorf("stripLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if st.raw || st.block {
			t.Errorf("stripLine(%q) left state open: %+v", tc.in, st)
		}
	}
}

func TestStripLineCarriesStateAcrossLines(t *testing.T) {
	_, st := stripLine("x := `open", lineState{})
	if !st.raw {
		t.Fatal("an unterminated raw string must carry over")
	}
	_, st = stripLine("still string` + y", st)
	if st.raw {
		t.Fatal("the closing backtick must end the raw string")
	}

	_, st = stripLine("/* open", lineState{})
	if !st.block {
		t.Fatal("an unterminated block comment must carry over")
	}
	_, st = stripLine("still comment */ code()", st)
	if st.block {
		t.Fatal("*/ must end the block comment")
	}
}

func TestTodoNeedsMarkerFormNotProse(t *testing.T) {
	cfg := DefaultConfig()
	flagged := func(line string) bool {
		f := source(t, "t.go", "package t\n\nfunc f() {\n\t"+line+"\n}\n")
		return countRule(CheckGeneric(cfg, f), "gen.todo") > 0
	}

	for _, want := range []string{
		"// TODO: split this",
		"// TODO split this",
		"// FIXME(bob): broken",
		"x() // HACK: works around the driver",
		"// XXX do not ship",
	} {
		if !flagged(want) {
			t.Errorf("expected a marker for %q", want)
		}
	}

	for _, prose := range []string{
		"// this is a hack to work around the driver",
		"// so TODO and FIXME rules still see the text",
		"// nothing to fix here",
		`x := "TODO: inside a string literal"`,
	} {
		if flagged(prose) {
			t.Errorf("prose must not be a marker: %q", prose)
		}
	}
}

func TestCommentedCodeIsAStatementNotProse(t *testing.T) {
	cfg := DefaultConfig()
	flagged := func(line string) bool {
		f := source(t, "c.go", "package c\n\n"+line+"\nvar x = 1\n")
		return countRule(CheckGeneric(cfg, f), "gen.commented.code") > 0
	}

	for _, code := range []string{
		"// return nil", "// x := compute()", "// doThing()", "// }",
		"// total += item.price;", "// let total = 0;", "// break",
	} {
		if !flagged(code) {
			t.Errorf("expected commented-out code: %q", code)
		}
	}

	for _, keep := range []string{
		"//\tapiref.Mount(router, apiref.Config{",
		"//\t\tTitle: \"Sablier API\",",
		"// style; failing at startup is the only safe reading of it.",
		"// Handler() does the thing before the request lands",
		"// Loop through all items",
		"// class is recorded.",
		"// for the whole run, so a pool capped at one deadlocks against itself.",
		"// if the request is a POST, return 405 instead",
		"// package-level concurrency instead of being forced through -p 1",
	} {
		if flagged(keep) {
			t.Errorf("prose or a godoc code block must survive: %q", keep)
		}
	}
}

func TestProseBeginningWithPackageIsNotCommentedOutCode(t *testing.T) {
	cfg := DefaultConfig()
	body := `// Package auth authenticates dashboard callers.
//
// When the suite's unified auth
// package arrives, it replaces Service as the Authenticator and supplies its own
// login routes.
package auth
`

	if n := countRule(CheckGeneric(cfg, source(t, "auth.go", body)), "gen.commented.code"); n != 0 {
		t.Error("a wrapped sentence in a package doc is prose, not a package clause")
	}

	commented := "// package auth\nfunc f() {}\n"
	if n := countRule(CheckGeneric(cfg, source(t, "c.go", commented)), "gen.commented.code"); n != 1 {
		t.Error("an actual commented-out package clause must still be caught")
	}
}
