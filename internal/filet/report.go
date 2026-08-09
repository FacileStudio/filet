package filet

import (
	"encoding/json"
	"io"
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

// Grade scores the code from A to F on findings per thousand lines.
func (r Report) Grade() string {
	if r.Lines == 0 {
		return "A"
	}
	density := float64(len(r.Findings)) * 1000 / float64(r.Lines)
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

// WriteJSON renders the report as machine-readable JSON.
func WriteJSON(w io.Writer, r Report) error {
	if r.Findings == nil {
		r.Findings = []Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteJSONValue renders any value as indented JSON.
func WriteJSONValue(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
