package filet

import "strings"

// Glyphs are the separators and severity marks, with an ASCII fallback for
// terminals that cannot be trusted with box drawing.
type Glyphs struct {
	File  string
	Error string
	Warn  string
	Info  string
	Arrow string
	Sep   string
	Rule  string
	Run   string
	Pass  string
}

var (
	unicodeGlyphs = Glyphs{
		File: "◆", Error: "✗", Warn: "▲", Info: "·",
		Arrow: "↳", Sep: "·", Rule: "─", Run: "▶", Pass: "✓",
	}
	asciiGlyphs = Glyphs{
		File: ">", Error: "x", Warn: "!", Info: "-",
		Arrow: "->", Sep: "|", Rule: "-", Run: ">", Pass: "+",
	}
)

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func padLeft(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat(" ", n-len(s)) + s
}
