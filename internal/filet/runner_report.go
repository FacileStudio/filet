package filet

import (
	"fmt"
	"io"
	"time"
)

// WriteHeading announces a suite before its output is streamed.
func WriteHeading(w io.Writer, s Suite) {
	p, g := painter(Colorize(w)), GlyphsFor(w)
	fmt.Fprintf(w, "\n%s %s %s %s\n",
		p(cyan, g.Run), p(bold, s.Name), p(dim, g.Sep), p(grey, s.Command()))
}

// WriteResults prints the per-suite summary and returns the worst exit code.
func WriteResults(w io.Writer, results []Result) int {
	p, g := painter(Colorize(w)), GlyphsFor(w)
	name, worst, passed := 0, 0, 0
	total := time.Duration(0)
	for _, r := range results {
		name = max(name, len(r.Suite))
		total += r.Duration
	}

	fmt.Fprintln(w)
	for _, r := range results {
		if r.ExitCode == 0 {
			passed++
		}
		worst = max(worst, r.ExitCode)
		fmt.Fprintf(w, "  %s %s  %s  %s\n",
			p(resultColor(r), resultMark(g, r)),
			p(bold, pad(r.Suite, name)),
			p(dim, padLeft(r.Duration.Round(time.Millisecond).String(), 7)),
			p(grey, r.Command))
		if r.Err != "" {
			fmt.Fprintf(w, "    %s\n", p(red, r.Err))
		}
	}

	writeSuiteTotals(w, p, g, tally{suites: len(results), passed: passed, took: total})
	return worst
}

type tally struct {
	suites int
	passed int
	took   time.Duration
}

func writeSuiteTotals(w io.Writer, p paintFn, g Glyphs, t tally) {
	sep := " " + p(dim, g.Sep) + " "
	colour := green
	if t.passed < t.suites {
		colour = red
	}
	fmt.Fprintf(w, "\n  %s%s%s%s%s\n",
		p(dim, count(t.suites, "suite", "suites")), sep,
		p(colour, fmt.Sprintf("%d passed", t.passed)), sep,
		p(dim, t.took.Round(time.Millisecond).String()))
}

func resultColor(r Result) string {
	if r.ExitCode == 0 {
		return green
	}
	return red
}

func resultMark(g Glyphs, r Result) string {
	if r.ExitCode == 0 {
		return g.Pass
	}
	return g.Error
}
