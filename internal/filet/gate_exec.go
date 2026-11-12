package filet

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func suiteStep(phase, name, tool string, s Suite) GateStep {
	return GateStep{
		Phase:   phase,
		Name:    name,
		Tool:    tool,
		Unit:    s.Unit,
		Command: s.Command(),
		Run: func(w io.Writer) (StepStatus, string) {
			return statusOf(RunSuite(s, nil, w))
		},
	}
}

func statusOf(res Result) (StepStatus, string) {
	switch {
	case res.ExitCode == 0:
		return StepPassed, ""
	case res.ExitCode == 127 && res.Err != "":
		return StepSkipped, res.Err
	case res.Err != "":
		return StepFailed, res.Err
	default:
		return StepFailed, fmt.Sprintf("exit %d", res.ExitCode)
	}
}

func gofmtStep(cfg *Config, s Suite) GateStep {
	return GateStep{
		Phase:   "format",
		Name:    "gofmt",
		Tool:    "gofmt",
		Unit:    s.Unit,
		Command: s.Command(),
		Run: func(w io.Writer) (StepStatus, string) {
			var buf bytes.Buffer
			status, detail := statusOf(RunSuite(s, nil, &buf))
			files := unformatted(cfg, s.Dir, buf.String())
			for _, f := range files {
				fmt.Fprintln(w, f)
			}
			if status != StepPassed {
				return status, detail
			}
			if len(files) > 0 {
				return StepFailed, count(len(files), "file needs gofmt", "files need gofmt")
			}
			return StepPassed, ""
		},
	}
}

func unformatted(cfg *Config, dir, listing string) []string {
	var out []string
	for f := range strings.FieldsSeq(listing) {
		if !inSkippedDir(cfg, dir, f) {
			out = append(out, f)
		}
	}
	return out
}

func inSkippedDir(cfg *Config, dir, file string) bool {
	for rel := filepath.Dir(filepath.Clean(file)); rel != "." && rel != string(filepath.Separator); rel = filepath.Dir(rel) {
		if skipUnitDir(cfg, filepath.Join(dir, rel), dir, filepath.Base(rel)) != nil {
			return true
		}
	}
	return false
}
