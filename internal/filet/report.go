package filet

import (
	"encoding/json"
	"io"
	"sort"
)

// Report is everything a command produced, ready to render.
type Report struct {
	Findings []Finding `json:"findings"`
	Files    int       `json:"files"`
	Lines    int       `json:"lines"`
}

// Counts returns how many findings sit at each severity.
func (r Report) Counts() (int, int, int) {
	var info, warn, errs int
	for _, f := range r.Findings {
		switch f.Severity {
		case Info:
			info++
		case Warn:
			warn++
		default:
			errs++
		}
	}
	return info, warn, errs
}

// gradeFloor is the smallest denominator the density is allowed to use.
//
// Without it a small input grades on noise: one info finding on a 37-line
// Dockerfile is 27 findings per thousand lines, which is an F and says nothing
// about the Dockerfile — only that dividing by 37 amplifies everything. The
// floor also removes the division by zero on an empty target.
const gradeFloor = 1000

// Grade scores the code from A to F on findings per thousand lines.
func (r Report) Grade() string {
	density := float64(len(r.Findings)) * 1000 / float64(max(r.Lines, gradeFloor))
	switch {
	case density < 2:
		return "A"
	case density < 6:
		return "B"
	case density < 12:
		return "C"
	case density < 25:
		return "D"
	default:
		return "F"
	}
}

var verdicts = map[string]string{
	"A": "Suspiciously clean. Either this is good code or the linter is broken.",
	"B": "Solid. A few sharp edges nobody will notice until 3am.",
	"C": "It ships. It also has opinions about how it ships.",
	"D": "This codebase is held together by tests it does not have.",
	"F": "Consider this less a review and more an intervention.",
}

type jsonCounts struct {
	Error int `json:"error"`
	Warn  int `json:"warn"`
	Info  int `json:"info"`
}

type jsonReport struct {
	Findings []Finding  `json:"findings"`
	Files    int        `json:"files"`
	Lines    int        `json:"lines"`
	Counts   jsonCounts `json:"counts"`
	Grade    string     `json:"grade"`
}

// WriteJSON renders the report as machine-readable JSON.
//
// The findings array is the report; counts and grade are derived from it and
// emitted anyway, so that a CI step reading this with jq does not reimplement
// the severity tally and the grading curve in shell and get them subtly wrong.
func WriteJSON(w io.Writer, r Report) error {
	if r.Findings == nil {
		r.Findings = []Finding{}
	}
	info, warn, errs := r.Counts()
	payload := jsonReport{
		Findings: r.Findings,
		Files:    r.Files,
		Lines:    r.Lines,
		Counts:   jsonCounts{Error: errs, Warn: warn, Info: info},
		Grade:    r.Grade(),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// WriteJSONValue renders any value as indented JSON.
func WriteJSONValue(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func byFile(findings []Finding) map[string][]Finding {
	out := map[string][]Finding{}
	for _, f := range findings {
		out[f.File] = append(out[f.File], f)
	}
	return out
}

func groupNames(findings []Finding) []string {
	seen := byFile(findings)
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
