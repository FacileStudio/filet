package filet

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

func suiteStep(phase, name, tool string, s Suite) GateStep {
	return GateStep{
		Phase:   phase,
		Name:    name,
		Tool:    tool,
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

func gofmtStep(dir string) GateStep {
	s := Suite{Name: "gofmt", Dir: dir, Cmd: "gofmt", Args: []string{"-l", "."}}
	return GateStep{
		Phase:   "format",
		Name:    "gofmt",
		Tool:    "gofmt",
		Command: s.Command(),
		Run: func(w io.Writer) (StepStatus, string) {
			var buf bytes.Buffer
			status, detail := statusOf(RunSuite(s, nil, io.MultiWriter(w, &buf)))
			if status != StepPassed {
				return status, detail
			}
			if files := strings.Fields(buf.String()); len(files) > 0 {
				return StepFailed, count(len(files), "file needs gofmt", "files need gofmt")
			}
			return StepPassed, ""
		},
	}
}
