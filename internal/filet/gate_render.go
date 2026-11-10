package filet

import (
	"fmt"
	"io"
	"time"
)

// WriteStepHeading announces a step before its output is streamed.
func WriteStepHeading(w io.Writer, s GateStep) {
	p, g := painter(Colorize(w)), GlyphsFor(w)
	fmt.Fprintf(w, "\n%s %s %s %s\n",
		p(cyan, g.Run), p(bold, s.Label()), p(dim, g.Sep), p(grey, s.Command))
}

// WriteSteps prints the summary table: one row per step with its status,
// duration and the reason a row did not pass.
func WriteSteps(w io.Writer, results []StepResult) {
	p, g := painter(Colorize(w)), GlyphsFor(w)
	width := 0
	total := time.Duration(0)
	for _, r := range results {
		width = max(width, len(r.Step))
		total += r.Duration
	}
	fmt.Fprintln(w)
	for _, r := range results {
		fmt.Fprintf(w, "  %s %s  %s  %s",
			p(stepColor(r.Status), stepMark(g, r.Status)),
			p(bold, pad(r.Step, width)),
			p(stepColor(r.Status), pad(string(r.Status), 7)),
			p(dim, padLeft(r.Duration.Round(time.Millisecond).String(), 8)))
		if r.Detail != "" {
			fmt.Fprintf(w, "  %s", p(grey, r.Detail))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "\n  %s %s %s\n", p(dim, count(len(results), "step", "steps")), p(dim, g.Sep),
		p(dim, total.Round(time.Millisecond).String()))
}

func stepColor(s StepStatus) string {
	switch s {
	case StepPassed:
		return green
	case StepFailed:
		return red
	case StepSkipped:
		return yellow
	default:
		return grey
	}
}

func stepMark(g Glyphs, s StepStatus) string {
	switch s {
	case StepPassed:
		return g.Pass
	case StepFailed:
		return g.Error
	case StepSkipped:
		return g.Warn
	default:
		return g.Info
	}
}
