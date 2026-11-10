package filet

import (
	"fmt"
	"io"
	"strings"
)

func checkStep(cfg *Config, target string) GateStep {
	return GateStep{
		Phase:   "check",
		Name:    "check",
		Command: "filet check " + target,
		Run: func(w io.Writer) (StepStatus, string) {
			report, err := Analyze(cfg, target)
			if err != nil {
				return StepFailed, err.Error()
			}
			if err := WriteLines(w, report); err != nil {
				return StepFailed, err.Error()
			}
			found := count(len(report.Findings), "finding", "findings")
			if !strings.EqualFold(cfg.FailOn, "never") && FailLevel(cfg, report) {
				return StepFailed, found
			}
			return StepPassed, found
		},
	}
}

func fuzzSteps(cfg *Config, target string) ([]GateStep, error) {
	targets, err := DiscoverFuzzTargets(cfg, target)
	if err != nil {
		return nil, err
	}
	var out []GateStep
	for _, lang := range []string{"go", "rust"} {
		if picked := targetsIn(targets, lang); len(picked) > 0 {
			out = append(out, fuzzGroupStep(lang, picked))
		}
	}
	return out, nil
}

func targetsIn(targets []FuzzTarget, lang string) []FuzzTarget {
	var out []FuzzTarget
	for _, t := range targets {
		if t.Lang == lang {
			out = append(out, t)
		}
	}
	return out
}

func fuzzGroupStep(lang string, targets []FuzzTarget) GateStep {
	return GateStep{
		Phase:   "fuzz",
		Name:    lang,
		Tool:    fuzzTool(lang),
		Command: "filet fuzz -mode seeds",
		Run: func(w io.Writer) (StepStatus, string) {
			return foldFuzz(RunFuzz(targets, FuzzOptions{Mode: FuzzSeeds}, w))
		},
	}
}

func foldFuzz(results []StepResult) (StepStatus, string) {
	tally := map[StepStatus]int{}
	for _, r := range results {
		tally[r.Status]++
	}
	switch {
	case tally[StepFailed] > 0:
		return StepFailed, fmt.Sprintf("%d of %d targets failed", tally[StepFailed], len(results))
	case tally[StepPassed] == 0:
		return StepSkipped, "no target ran"
	case tally[StepSkipped] > 0:
		return StepSkipped, fmt.Sprintf("%d of %d targets skipped", tally[StepSkipped], len(results))
	default:
		return StepPassed, count(len(results), "target", "targets")
	}
}
