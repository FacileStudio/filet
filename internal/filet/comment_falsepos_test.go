package filet

import "testing"

func TestRegexEscapeIsNotInlineComment(t *testing.T) {
	cfg := DefaultConfig()
	for _, c := range []struct {
		body string
		want int
	}{
		{`if (/^https?:\/\//.test(path)) {`, 0},
		{"x = /a\\/\\{/v.test(y) // the real note", 1},
	} {
		f := source(t, "b.ts", c.body)
		if n := countRule(CheckGeneric(cfg, f), "gen.comment.inline"); n != c.want {
			t.Errorf("inline comment on %q: got %d, want %d", c.body, n, c.want)
		}
	}
}

func TestAssignmentInDocProseIsNotCommentedOutCode(t *testing.T) {
	cfg := DefaultConfig()
	for _, c := range []struct {
		body string
		want int
	}{
		{"// on avatar_source = 'upload' quietly drops their picture", 0},
		{"// Enabled=false and no settings row, so a client is the signal", 0},
		{"// keying on avatar_source = 'upload' is a trap", 0},
		{"// total += item.price;", 1},
		{"// x := compute()", 1},
	} {
		f := source(t, "c.go", "package c\n\n"+c.body+"\nvar x = 1\n")
		if n := countRule(CheckGeneric(cfg, f), "gen.commented.code"); n != c.want {
			t.Errorf("commented-out %q: got %d, want %d", c.body, n, c.want)
		}
	}
}

func TestURLIsNotAnInlineComment(t *testing.T) {
	cfg := DefaultConfig()
	for _, c := range []struct {
		ext, body string
		want      int
	}{
		{".svelte", `Run <code class="x">https://mycelium.facile.studio</code> on a machine` + "\n", 0},
		{".svelte", `>mycelium login https://mycelium.facile.studio</code` + "\n", 0},
		{".md", "pair it via https://mycelium.facile.studio/login", 0},
		{".go", "url := \"https://example.com\" // the note", 1},
	} {
		if n := countRule(CheckGeneric(cfg, source(t, "f"+c.ext, c.body)), "gen.comment.inline"); n != c.want {
			t.Errorf("URL inline comment in %s: got %d, want %d", c.ext, n, c.want)
		}
	}
}

func TestPrivateFieldIsNotCommentedOutCode(t *testing.T) {
	cfg := DefaultConfig()
	for _, c := range []struct {
		ext, body string
		want      int
	}{
		{".ts", "#anchor = -1;\n#lookup = $derived(new Set(keys));\nvar x = 1\n", 0},
		{".svelte", "#anchor = -1;\nvar x = 1\n", 0},
		{".go", "#anchor = -1;\nvar x = 1\n", 0},
		{".sh", "# total += item.price;\n# anchor = 1\n", 2},
	} {
		if n := countRule(CheckGeneric(cfg, source(t, "f"+c.ext, c.body)), "gen.commented.code"); n != c.want {
			t.Errorf("private field or # comment in %s: got %d, want %d", c.ext, n, c.want)
		}
	}
}
