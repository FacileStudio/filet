package filet

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"math"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// FuzzMode selects how RunFuzz exercises a target.
type FuzzMode string

// The two modes: seeds replays the committed corpus only, smoke fuzzes each
// target for a bounded time.
const (
	FuzzSeeds FuzzMode = "seeds"
	FuzzSmoke FuzzMode = "smoke"
)

// DefaultFuzzTime is how long smoke mode fuzzes one target when no time is given.
const DefaultFuzzTime = 30 * time.Second

// FuzzOptions bounds a fuzz run. FuzzTime applies per target in smoke mode;
// Budget stops RunFuzz from starting a new target once that much time is spent.
type FuzzOptions struct {
	Mode     FuzzMode
	FuzzTime time.Duration
	Budget   time.Duration
}

var crashLine = regexp.MustCompile(`(?:Failing input|Test unit) written to (\S+)`)

// Label is the row name the summary prints for the target.
func (t FuzzTarget) Label() string {
	return t.Lang + " " + t.Name + " " + DisplayPath(t.Dir)
}

// RunFuzz runs targets one at a time, because `go test -fuzz` accepts a single
// target, and returns one row per target. A target whose tool is missing is
// skipped with the reason, never passed.
func RunFuzz(targets []FuzzTarget, opts FuzzOptions, w io.Writer) []StepResult {
	results := make([]StepResult, 0, len(targets))
	spent := time.Duration(0)
	for _, t := range targets {
		if opts.Mode == FuzzSmoke && opts.Budget > 0 && spent >= opts.Budget {
			results = append(results, StepResult{Step: t.Label(), Status: StepSkipped,
				Detail: "budget of " + opts.Budget.String() + " spent"})
			continue
		}
		r := runFuzzTarget(t, opts, w)
		spent += r.Duration
		results = append(results, r)
	}
	return results
}

func runFuzzTarget(t FuzzTarget, opts FuzzOptions, w io.Writer) StepResult {
	s := fuzzSuite(t, opts)
	res := StepResult{Step: t.Label(), Command: s.Command()}
	if _, err := exec.LookPath(fuzzTool(t.Lang)); err != nil {
		res.Status, res.Detail = StepSkipped, fuzzTool(t.Lang)+" not found"
		return res
	}
	WriteHeading(w, s)
	var buf bytes.Buffer
	run := RunSuite(s, nil, io.MultiWriter(w, &buf))
	res.Duration, res.Millis = run.Duration, run.Millis
	res.Status, res.Detail = statusOf(run)
	if m := crashLine.FindStringSubmatch(buf.String()); res.Status == StepFailed && m != nil {
		res.Detail = "failing input: " + filepath.Join(t.Dir, m[1])
	}
	return res
}

func fuzzTool(lang string) string {
	if lang == "rust" {
		return "cargo-fuzz"
	}
	return "go"
}

func fuzzSuite(t FuzzTarget, opts FuzzOptions) Suite {
	fuzzTime := cmp.Or(opts.FuzzTime, DefaultFuzzTime)
	s := Suite{Name: t.Name, Dir: t.Dir, Cmd: "go"}
	switch {
	case t.Lang == "rust" && opts.Mode == FuzzSmoke:
		secs := int(math.Ceil(fuzzTime.Seconds()))
		s.Cmd, s.Args = "cargo", []string{"fuzz", "run", t.Name, "--", fmt.Sprintf("-max_total_time=%d", secs)}
	case t.Lang == "rust":
		s.Cmd, s.Args = "cargo", []string{"fuzz", "run", t.Name, "--", "-runs=0"}
	case opts.Mode == FuzzSmoke:
		s.Args = []string{"test", "-run", "^$", "-fuzz", "^" + t.Name + "$", "-fuzztime", fuzzTime.String(), "."}
	default:
		s.Args = []string{"test", "-run", "^" + t.Name + "$", "."}
	}
	return s
}

// ParseFuzzMode maps a -mode flag value to a FuzzMode.
func ParseFuzzMode(s string) (FuzzMode, error) {
	switch m := FuzzMode(strings.ToLower(s)); m {
	case FuzzSeeds, FuzzSmoke:
		return m, nil
	default:
		return "", fmt.Errorf("-mode: unknown mode %q (use seeds or smoke)", s)
	}
}
