package filet

import (
	"io"
	"os"
	"strings"
)

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	grey   = "\033[90m"
	green  = "\033[32m"
)

type paintFn func(color, s string) string

// IsTTY reports whether w is an interactive terminal rather than a pipe or file.
func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// Colorize reports whether ANSI output should be used for w.
func Colorize(w io.Writer) bool {
	return os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && IsTTY(w)
}

// GlyphsFor picks the richest character set the environment claims to handle.
func GlyphsFor(w io.Writer) Glyphs {
	if !IsTTY(w) || !utf8Locale() {
		return asciiGlyphs
	}
	return unicodeGlyphs
}

func utf8Locale() bool {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := os.Getenv(key); v != "" {
			up := strings.ToUpper(v)
			return strings.Contains(up, "UTF-8") || strings.Contains(up, "UTF8")
		}
	}
	return false
}

func painter(enabled bool) paintFn {
	return func(color, s string) string {
		if !enabled {
			return s
		}
		return color + s + reset
	}
}

func severityColor(s Severity) string {
	switch s {
	case Warn:
		return yellow
	case Error:
		return red
	default:
		return blue
	}
}

func gradeColor(g string) string {
	switch g {
	case "A", "B":
		return green
	case "C":
		return yellow
	default:
		return red
	}
}
