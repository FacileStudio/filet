package filet

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// The line format is a contract other tools depend on. If this regex has to
// change, that is a breaking change, not a cosmetic one.
var lineFormat = regexp.MustCompile(`^([^:]+):(\d+):(\d+): (info|warn|error): (.+) \[([a-z.]+)\]$`)

func TestLineFormatIsStableAndParseable(t *testing.T) {
	r := Report{Findings: []Finding{
		{Rule: "go.panic", File: "internal/a.go", Line: 12, Column: 4, Severity: Warn, Message: "panic in library code"},
		{Rule: "docker.secret", File: "Dockerfile", Line: 2, Severity: Error, Message: "secret baked in"},
	}}

	var buf bytes.Buffer
	if err := WriteLines(&buf, r); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected one line per finding, got %d", len(lines))
	}

	m := lineFormat.FindStringSubmatch(lines[0])
	if m == nil {
		t.Fatalf("line does not match the documented format: %q", lines[0])
	}
	if m[1] != "internal/a.go" || m[2] != "12" || m[3] != "4" || m[4] != "warn" || m[6] != "go.panic" {
		t.Fatalf("fields parsed wrong: %#v", m[1:])
	}

	if m := lineFormat.FindStringSubmatch(lines[1]); m == nil || m[3] != "1" {
		t.Fatalf("a finding with no column must still emit one: %q", lines[1])
	}
	if strings.Contains(buf.String(), "\033[") {
		t.Fatal("the line format must never be coloured")
	}
}

func TestTextFormatIsNeverColouredOffATerminal(t *testing.T) {
	r := Report{Findings: []Finding{
		{Rule: "go.panic", File: "a.go", Line: 1, Severity: Error, Message: "boom", Roast: "ouch"},
	}, Files: 1, Lines: 10}

	var buf bytes.Buffer
	WriteText(&buf, r, TextOptions{Roast: true})
	out := buf.String()
	if strings.Contains(out, "\033[") {
		t.Fatal("a bytes.Buffer is not a terminal, so no escape codes may be emitted")
	}
	for _, want := range []string{"a.go", "go.panic", "boom", "ouch", "1 error", "grade"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
}

func TestAsciiFallbackHasNoBoxDrawing(t *testing.T) {
	var buf bytes.Buffer
	WriteText(&buf, Report{
		Findings: []Finding{{Rule: "go.panic", File: "a.go", Line: 1, Severity: Warn, Message: "m"}},
		Files:    1, Lines: 2,
	}, TextOptions{})

	for _, glyph := range []string{unicodeGlyphs.File, unicodeGlyphs.Warn, unicodeGlyphs.Sep} {
		if strings.Contains(buf.String(), glyph) {
			t.Fatalf("non-terminal output must fall back to ASCII, found %q", glyph)
		}
	}
}
