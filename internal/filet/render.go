package filet

import (
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

// TextOptions controls how a report renders as text.
type TextOptions struct {
	Roast bool
	Quiet bool
}

// WriteText renders the report for humans, grouped by file. It is meant to be
// read, not parsed; use WriteLines or WriteJSON to feed another tool.
func WriteText(w io.Writer, r Report, opts TextOptions) {
	p, g := painter(Colorize(w)), GlyphsFor(w)
	if len(r.Findings) == 0 {
		writeClean(w, p, r, opts.Roast)
		return
	}
	if !opts.Quiet {
		grouped := byFile(r.Findings)
		for _, name := range groupNames(r.Findings) {
			fmt.Fprintf(w, "\n%s %s\n", p(cyan, g.File), filePath(p, name))
			writeGroup(w, p, g, grouped[name], opts.Roast)
		}
		fmt.Fprintln(w)
	}
	writeSummary(w, p, g, r, opts.Roast)
}

func filePath(p paintFn, rel string) string {
	dir, base := path.Split(rel)
	return p(dim, dir) + p(bold, base)
}

func writeGroup(w io.Writer, p paintFn, g Glyphs, findings []Finding, roast bool) {
	at, rule := 0, 0
	for _, f := range findings {
		if len(position(f)) > at {
			at = len(position(f))
		}
		if len(f.Rule) > rule {
			rule = len(f.Rule)
		}
	}
	for _, f := range findings {
		colour := severityColor(f.Severity)
		docs := ""
		if f.Docs != "" {
			docs = p(dim, docsSuffix(f))
		}
		fmt.Fprintf(w, "  %s  %s %s  %s  %s%s\n",
			p(grey, padLeft(position(f), at)),
			p(colour, mark(g, f.Severity)),
			p(colour, pad(f.Severity.String(), 5)),
			p(dim, pad(f.Rule, rule)),
			f.Message, docs)
		if roast && f.Roast != "" {
			fmt.Fprintf(w, "  %s    %s\n", strings.Repeat(" ", at), p(dim, g.Arrow+" "+f.Roast))
		}
	}
}

func position(f Finding) string {
	if f.Column > 0 {
		return strconv.Itoa(f.Line) + ":" + strconv.Itoa(f.Column)
	}
	return strconv.Itoa(f.Line)
}

func mark(g Glyphs, s Severity) string {
	switch s {
	case Warn:
		return g.Warn
	case Error:
		return g.Error
	default:
		return g.Info
	}
}

func writeSummary(w io.Writer, p paintFn, g Glyphs, r Report, roast bool) {
	info, warn, errs := r.Counts()
	sep := " " + p(dim, g.Sep) + " "
	fmt.Fprintf(w, "  %s%s%s\n",
		p(dim, count(r.Files, "file", "files")), sep, p(dim, count(r.Lines, "line", "lines")))
	fmt.Fprintf(w, "  %s%s%s%s%s\n",
		p(severityColor(Error), count(errs, "error", "errors")), sep,
		p(severityColor(Warn), count(warn, "warning", "warnings")), sep,
		p(severityColor(Info), count(info, "info", "info")))
	if roast {
		grade := r.Grade()
		fmt.Fprintf(w, "  %s%s%s\n", p(gradeColor(grade), "grade "+grade), sep, p(dim, verdicts[grade]))
	}
}

func writeClean(w io.Writer, p paintFn, r Report, roast bool) {
	fmt.Fprintf(w, "%s %s, %s, nothing to complain about.\n",
		p(green, "clean"), count(r.Files, "file", "files"), count(r.Lines, "line", "lines"))
	if roast {
		fmt.Fprintf(w, "%s %s\n", p(green, "grade A"), p(dim, verdicts["A"]))
	}
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
