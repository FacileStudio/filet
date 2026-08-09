package filet

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// TextOptions controls how a report renders as text.
type TextOptions struct {
	Roast bool
	Quiet bool
}

// WriteText renders the report for humans, grouped by file.
func WriteText(w io.Writer, r Report, opts TextOptions) {
	p := painter(Colorize(w))
	if len(r.Findings) == 0 {
		writeClean(w, p, r, opts.Roast)
		return
	}
	if !opts.Quiet {
		for _, name := range groupNames(r.Findings) {
			fmt.Fprintf(w, "\n%s\n", p(bold, name))
			writeFile(w, p, byFile(r.Findings)[name], opts.Roast)
		}
	}
	writeSummary(w, p, r, opts.Roast)
}

func writeClean(w io.Writer, p paintFn, r Report, roast bool) {
	fmt.Fprintf(w, "%s %d files, %d lines, nothing to complain about.\n",
		p(green, "clean"), r.Files, r.Lines)
	if roast {
		fmt.Fprintf(w, "%s %s — %s\n", p(bold, "grade  "), p(green, "A"), verdicts["A"])
	}
}

func byFile(findings []Finding) map[string][]Finding {
	out := map[string][]Finding{}
	for _, f := range findings {
		out[f.File] = append(out[f.File], f)
	}
	return out
}

func groupNames(findings []Finding) []string {
	seen := byFile(findings)
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func writeFile(w io.Writer, p paintFn, findings []Finding, roast bool) {
	for _, f := range findings {
		num := fmt.Sprint(f.Line)
		fmt.Fprintf(w, "  %s:%s %s %s %s\n",
			p(grey, num),
			strings.Repeat(" ", max(0, 4-len(num))),
			p(severityColor(f.Severity), pad(f.Severity.String(), 5)),
			f.Message,
			p(dim, "["+f.Rule+"]"))
		if roast && f.Roast != "" {
			fmt.Fprintf(w, "        %s\n", p(dim, "↳ "+f.Roast))
		}
	}
}

func writeSummary(w io.Writer, p paintFn, r Report, roast bool) {
	info, warn, errs := r.Counts()
	fmt.Fprintf(w, "\n%s %d error, %d warn, %d info across %d files (%d lines)\n",
		p(bold, "summary"), errs, warn, info, r.Files, r.Lines)
	if !roast {
		return
	}
	grade := r.Grade()
	fmt.Fprintf(w, "%s %s — %s\n", p(bold, "grade  "), p(gradeColor(grade), grade), verdicts[grade])
}
