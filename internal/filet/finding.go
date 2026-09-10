package filet

import (
	"cmp"
	"slices"
	"strings"
)

// Severity ranks a finding from cosmetic to actively harmful.
type Severity int

const (
	Info Severity = iota
	Warn
	Error
)

// ValidFailOn reports whether s names a severity gate accepted by failOn.
func ValidFailOn(s string) bool {
	switch strings.ToLower(s) {
	case "info", "warn", "warning", "error", "never":
		return true
	}
	return false
}

// ParseSeverity maps a config string to a Severity, defaulting to Error.
func ParseSeverity(s string) Severity {
	switch strings.ToLower(s) {
	case "info":
		return Info
	case "warn", "warning":
		return Warn
	default:
		return Error
	}
}

// String returns the lowercase label used in output and JSON.
func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warn:
		return "warn"
	default:
		return "error"
	}
}

// Finding is a single rule violation anchored at a file and line.
type Finding struct {
	Rule     string   `json:"rule"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Column   int      `json:"column,omitempty"`
	Message  string   `json:"message"`
	Docs     string   `json:"docs,omitempty"`
	Severity Severity `json:"-"`
	Level    string   `json:"severity"`
	Roast    string   `json:"roast,omitempty"`
}

func newFinding(rule, file string, line int, sev Severity, msg string) Finding {
	return Finding{Rule: rule, File: file, Line: line, Severity: sev, Level: sev.String(), Message: msg}
}

// docsSuffix returns the "docs: <url>" tail for a finding that carries a docs
// link, or "" otherwise. Colour renderers style a finding's Docs field directly;
// flat renderers (line, SARIF, GitHub) append this so the link is not lost.
func docsSuffix(f Finding) string {
	if f.Docs == "" {
		return ""
	}
	return " docs: " + f.Docs
}

// SortFindings orders findings by file, then line, then rule so output is stable.
func SortFindings(f []Finding) {
	slices.SortStableFunc(f, func(a, b Finding) int {
		if c := cmp.Compare(a.File, b.File); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Line, b.Line); c != 0 {
			return c
		}
		return cmp.Compare(a.Rule, b.Rule)
	})
}
