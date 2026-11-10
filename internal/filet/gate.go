package filet

import (
	"io"
	"os/exec"
	"slices"
	"time"
)

// StepStatus is the verdict of one gate or fuzz step. A step whose tool is
// missing is skipped, never passed, so a green table means every row ran.
type StepStatus string

// The four verdicts a step can carry.
const (
	StepPassed  StepStatus = "passed"
	StepFailed  StepStatus = "failed"
	StepSkipped StepStatus = "skipped"
	StepNotRun  StepStatus = "not-run"
)

// StepResult is one row of the gate or fuzz summary table.
type StepResult struct {
	Step     string        `json:"step"`
	Status   StepStatus    `json:"status"`
	Command  string        `json:"command,omitempty"`
	Detail   string        `json:"detail,omitempty"`
	Millis   int64         `json:"durationMs"`
	Duration time.Duration `json:"-"`
}

// GateStep is one unit of work the gate runs. Phase and Name are both valid
// targets for gate.disable; Tool, when set, must be on PATH for Run to start.
type GateStep struct {
	Phase   string
	Name    string
	Tool    string
	Command string
	Run     func(w io.Writer) (StepStatus, string)
}

// Label is the row name the summary prints: the phase, then the step name
// when the two differ.
func (s GateStep) Label() string {
	if s.Name == "" || s.Name == s.Phase {
		return s.Phase
	}
	return s.Phase + ":" + s.Name
}

// GateOptions tunes one gate run.
type GateOptions struct {
	KeepGoing bool
	Config    GateConfig
}

// RunGate runs steps in order, streaming their output to w. It stops at the
// first failed step unless KeepGoing is set; the steps it never started are
// returned as not-run so the table still shows them.
func RunGate(steps []GateStep, opts GateOptions, w io.Writer) []StepResult {
	results := make([]StepResult, 0, len(steps))
	stopped := false
	for _, s := range steps {
		if stopped {
			results = append(results, StepResult{Step: s.Label(), Status: StepNotRun, Command: s.Command,
				Detail: "an earlier step failed"})
			continue
		}
		r := runGateStep(s, opts.Config, w)
		results = append(results, r)
		stopped = r.Status == StepFailed && !opts.KeepGoing
	}
	return results
}

func runGateStep(s GateStep, cfg GateConfig, w io.Writer) StepResult {
	res := StepResult{Step: s.Label(), Command: s.Command}
	if cfg.disables(s) {
		res.Status, res.Detail = StepSkipped, "disabled by gate.disable"
		return res
	}
	if s.Tool != "" {
		if _, err := exec.LookPath(s.Tool); err != nil {
			res.Status, res.Detail = StepSkipped, s.Tool+" not found"
			if slices.Contains(cfg.RequireTools, s.Tool) {
				res.Status, res.Detail = StepFailed, s.Tool+" not found and listed in gate.requireTools"
			}
			return res
		}
	}
	WriteStepHeading(w, s)
	start := time.Now()
	res.Status, res.Detail = s.Run(w)
	res.Duration = time.Since(start)
	res.Millis = res.Duration.Milliseconds()
	return res
}

// GateSummary is the JSON document `filet gate -format json` prints.
type GateSummary struct {
	Passed bool         `json:"passed"`
	Steps  []StepResult `json:"steps"`
}

// Summarize wraps results into the JSON document, passed only when no row
// failed.
func Summarize(results []StepResult) GateSummary {
	return GateSummary{Passed: !AnyFailed(results), Steps: results}
}

// AnyFailed reports whether any row carries the failed status.
func AnyFailed(results []StepResult) bool {
	return slices.ContainsFunc(results, func(r StepResult) bool { return r.Status == StepFailed })
}
