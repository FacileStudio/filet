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
	grey   = "\033[90m"
	green  = "\033[32m"
)

type paintFn func(color, s string) string

// Colorize reports whether ANSI output should be used for w.
func Colorize(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
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

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
