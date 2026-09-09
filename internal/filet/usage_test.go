package filet

import (
	"bytes"
	"strings"
	"testing"
)

// The help screen renders through the same newRenderer machinery as reports, so
// it degrades to plain text off a terminal; a buffer must never come back with
// ANSI escapes in it.
func TestUsageIsPlainOffTerminal(t *testing.T) {
	var buf bytes.Buffer
	RenderUsage(&buf)
	text := buf.String()
	if strings.Contains(text, "\033[") {
		t.Fatal("help must not be coloured off a terminal")
	}
	for _, heading := range []string{"Commands", "Droast", "Flags", "Examples"} {
		if !strings.Contains(text, heading) {
			t.Fatalf("help missing section %q", heading)
		}
	}
}

// Command descriptions line up in one column so the help is scannable. If
// usageWidth or the padding math drifts, every described row in the section
// shifts, so a single mismatched column is enough to fail.
func TestUsageAlignsCommandDescriptions(t *testing.T) {
	var buf bytes.Buffer
	RenderUsage(&buf)
	lines := strings.Split(buf.String(), "\n")
	col := -1
	for _, line := range lines {
		if !strings.HasPrefix(line, "  filet ") {
			continue
		}
		c := usageDescCol(line)
		if c < 0 {
			continue
		}
		if col < 0 {
			col = c
		}
		if c != col {
			t.Fatalf("description column drifted: %q at %d, expected %d", line, c, col)
		}
	}
	if col < 0 {
		t.Fatal("expected at least one described command row")
	}
}

// usageDescCol returns the 0-based column where a rendered row's description
// starts, or -1 when the row carries none. Detail cells contain only single
// spaces, so the first run of 2+ spaces is the padding before the description.
func usageDescCol(line string) int {
	s := strings.TrimLeft(line, " ")
	base := len(line) - len(s)
	for i := 0; i < len(s); i++ {
		if i+1 >= len(s) || s[i] != ' ' || s[i+1] != ' ' {
			continue
		}
		j := i
		for j < len(s) && s[j] == ' ' {
			j++
		}
		if j >= len(s) {
			return -1
		}
		return base + j
	}
	return -1
}
