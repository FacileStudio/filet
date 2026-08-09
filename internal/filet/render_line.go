package filet

import (
	"fmt"
	"io"
)

// WriteLines renders one finding per line in the GNU error format that editors,
// CI annotators and grep already understand:
//
//	path:line:column: severity: message [rule]
//
// It is never coloured, never grouped and never reordered. Treat it as a stable
// contract: fields are separated by ": " and the rule id is the last bracketed
// token on the line.
func WriteLines(w io.Writer, r Report) error {
	for _, f := range r.Findings {
		column := f.Column
		if column < 1 {
			column = 1
		}
		if _, err := fmt.Fprintf(w, "%s:%d:%d: %s: %s [%s]\n",
			f.File, f.Line, column, f.Severity, f.Message, f.Rule); err != nil {
			return err
		}
	}
	return nil
}
