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
