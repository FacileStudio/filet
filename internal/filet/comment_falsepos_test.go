package filet

import "testing"

func TestShellExpansionIsNotAnInlineComment(t *testing.T) {
	cfg := DefaultConfig()
	flagged := func(body string) bool {
		f := source(t, "s.sh", body)
		return countRule(CheckGeneric(cfg, f), "gen.comment.inline") > 0
	}

	// $# (argument count) and other $ expansions are shell parameter expansions
	// not comments. These lines must not be flagged.
	for _, ok := range []string{
		`[ $# -eq 1 ] && pane_id="$1"`,
		`echo "got $# args"`,
		`total=$(( $# + 1 ))`,
		`case "$1" in "$#") true ;; esac`,
	} {
		if flagged(ok) {
			t.Errorf("shell parameter expansion should not flag inline comment: %q", ok)
		}
	}

	// Real inline comments should still be flagged.
	for _, bad := range []string{
		`echo hello # this is a real comment`,
		`x=1 # another real comment`,
	} {
		if !flagged(bad) {
			t.Errorf("real inline comment should flag: %q", bad)
		}
	}
}