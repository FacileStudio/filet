package filet

import (
	"strings"
	"testing"
)

// These tests cover clean's go.comment.inbody self-fix: a comment that sits
// inside a function body and carries no tool directive is removed whole, while
// code sharing its line and in-body directives survive. Every fixture is
// parseable Go (a bare `func` is rejected by the parser), because the body scan
// is best-effort: an unparseable file is reported by the check, not rewritten.

func TestCleanRemovesInBodySentenceComment(t *testing.T) {
	f := cleanSource(t, "clean.go", "package main\n\nfunc f() {\n\t// this does a thing.\n\tx := 1\n}\n")
	res := CleanFile(cleanCfg(), f)
	got := strings.Join(res.Lines, "\n")
	if strings.Contains(got, "this does a thing.") {
		t.Fatalf("in-body sentence comment survived:\n%s", got)
	}
	if !strings.Contains(got, "x := 1") {
		t.Fatalf("real code was removed with the comment:\n%s", got)
	}
	if res.Stats.InBodyComment != 1 {
		t.Fatalf("expected one in-body fix, got %d", res.Stats.InBodyComment)
	}
}

func TestCleanRemovesInBodyBlockComment(t *testing.T) {
	f := cleanSource(t, "clean.go", "package main\n\nfunc f() {\n\t/* one\n\t   two */\n\tx := 1\n}\n")
	res := CleanFile(cleanCfg(), f)
	got := strings.Join(res.Lines, "\n")
	if strings.Contains(got, "/*") || strings.Contains(got, "two */") {
		t.Fatalf("in-body block comment survived:\n%s", got)
	}
	if !strings.Contains(got, "x := 1") {
		t.Fatalf("code after block comment lost:\n%s", got)
	}
	if res.Stats.InBodyComment != 2 {
		t.Fatalf("expected two in-body line fixes, got %d", res.Stats.InBodyComment)
	}
}

func TestCleanKeepsCodeTrailingInBodyComment(t *testing.T) {
	f := cleanSource(t, "clean.go", "package main\n\nfunc f() {\n\tx := 1 // note\n\ty := 2\n}\n")
	res := CleanFile(cleanCfg(), f)
	got := strings.Join(res.Lines, "\n")
	for _, keep := range []string{"x := 1", "y := 2"} {
		if !strings.Contains(got, keep) {
			t.Fatalf("code sharing a line with a body comment was dropped: %s\n%s", keep, got)
		}
	}
	if strings.Contains(got, "// note") {
		t.Fatalf("trailing in-body comment survived:\n%s", got)
	}
	if res.Stats.InBodyComment != 0 || res.Stats.InlineComment != 1 {
		t.Fatalf("expected inline fix only, got %d in-body %d inline", res.Stats.InBodyComment, res.Stats.InlineComment)
	}
}

func TestCleanKeepsInBodyDirective(t *testing.T) {
	f := cleanSource(t, "clean.go", "package main\n\nfunc f() {\n\t//nolint:complexity\n\tx := 1\n}\n")
	res := CleanFile(cleanCfg(), f)
	got := strings.Join(res.Lines, "\n")
	if !strings.Contains(got, "//nolint:complexity") {
		t.Fatalf("in-body directive was stripped:\n%s", got)
	}
	if res.Stats.InBodyComment != 0 {
		t.Fatalf("no in-body fix expected for a directive, got %d", res.Stats.InBodyComment)
	}
}
