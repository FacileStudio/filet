package filet

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// githubData escapes the message half of a workflow command. A raw newline ends
// the command, so a multi-line message would silently truncate the annotation
// and leave the rest of the text on the log as if it were output.
var githubData = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")

// githubProperty escapes a property value. Properties are separated by commas
// and their values by equals signs, so a message or a path containing a comma
// or a colon has to be escaped further than the message half does — filet
// messages routinely contain both.
var githubProperty = strings.NewReplacer(
	"%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")

// WriteGitHub renders each finding as a GitHub Actions workflow command, which
// the runner turns into an annotation on the file and line inside the pull
// request diff.
//
// This is the format to use on a private repository: annotations cost nothing
// and work everywhere, while a SARIF upload needs code scanning, which private
// repositories only get with GitHub Advanced Security.
func WriteGitHub(w io.Writer, r Report) error {
	for _, f := range r.Findings {
		_, err := fmt.Fprintf(w, "::%s file=%s,line=%d,col=%d,title=%s::%s\n",
			githubLevel(f.Severity),
			githubProperty.Replace(filepath.ToSlash(f.File)),
			max(f.Line, 1),
			max(f.Column, 1),
			githubProperty.Replace(f.Rule),
			githubData.Replace(f.Message))
		if err != nil {
			return err
		}
	}
	return nil
}

// githubLevel maps a severity onto the three commands the runner understands.
// There is no "info" command; "notice" is its name on the other side.
func githubLevel(s Severity) string {
	switch s {
	case Error:
		return "error"
	case Warn:
		return "warning"
	default:
		return "notice"
	}
}
