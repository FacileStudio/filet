package filet

import (
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type lipglossCtx struct {
	renderer *lipgloss.Renderer
	p        paintFn
	g        Glyphs
}

// WriteTextLipgloss renders the same report using lipgloss styles and a
// lightweight table-like layout for findings. It is opt-in; WriteText remains
// the default path.
func WriteTextLipgloss(w io.Writer, r Report, opts TextOptions) {
	ctx := lipglossCtx{renderer: newRenderer(w), p: painter(Colorize(w)), g: GlyphsFor(w)}
	if len(r.Findings) == 0 {
		fmt.Fprintf(w, "%s %s, %s, nothing to complain about.\n",
			severityStyle(ctx.renderer, Info).Render("clean"),
			count(r.Files, "file", "files"),
			count(r.Lines, "line", "lines"))
		if opts.Roast {
			fmt.Fprintf(w, "%s %s\n",
				gradeStyle(ctx.renderer, "A").Render("grade A"),
				severityStyle(ctx.renderer, Info).Render(verdicts["A"]))
		}
		return
	}
	if !opts.Quiet {
		grouped := byFile(r.Findings)
		for _, name := range groupNames(r.Findings) {
			header := lipgloss.JoinHorizontal(lipgloss.Top,
				ctx.renderer.NewStyle().Foreground(lipgloss.Color("cyan")).Render(ctx.g.File),
				" ",
				dimStyle(ctx.renderer).Render(path.Dir(name)),
				boldStyle(ctx.renderer).Render(path.Base(name)),
			)
			fmt.Fprintf(w, "\n%s\n", header)
			writeGroupLipgloss(w, ctx, grouped[name], opts.Roast)
		}
		fmt.Fprintln(w)
	}
	writeSummaryLipgloss(w, ctx, r, opts.Roast)
}

func writeGroupLipgloss(w io.Writer, ctx lipglossCtx, findings []Finding, roast bool) {
	at, rule := groupWidths(findings)
	for _, f := range findings {
		pos := padLeft(position(f), at)
		body := f.Message
		if f.Docs != "" {
			body = body + dimStyle(ctx.renderer).Render(docsSuffix(f))
		}
		line := lipgloss.JoinHorizontal(lipgloss.Top,
			dimStyle(ctx.renderer).Render(pos),
			"  ",
			markLipgloss(ctx.renderer, ctx.g, f.Severity),
			" ",
			severityStyle(ctx.renderer, f.Severity).Render(pad(f.Severity.String(), 5)),
			"  ",
			dimStyle(ctx.renderer).Render(pad(f.Rule, rule)),
			"  ",
			body,
		)
		fmt.Fprintln(w, "  "+line)
		if roast && f.Roast != "" {
			indent := strings.Repeat(" ", at+2)
			roastLine := dimStyle(ctx.renderer).Render(ctx.g.Arrow + " " + f.Roast)
			fmt.Fprintln(w, "  "+indent+roastLine)
		}
	}
}

func markLipgloss(renderer *lipgloss.Renderer, g Glyphs, s Severity) string {
	m := mark(g, s)
	switch s {
	case Warn:
		return severityStyle(renderer, Warn).Render(m)
	case Error:
		return severityStyle(renderer, Error).Render(m)
	default:
		return severityStyle(renderer, Info).Render(m)
	}
}

func writeSummaryLipgloss(w io.Writer, ctx lipglossCtx, r Report, roast bool) {
	info, warn, errs := r.Counts()
	sep := " " + dimStyle(ctx.renderer).Render(ctx.g.Sep) + " "
	fmt.Fprintf(w, "  %s%s%s\n",
		dimStyle(ctx.renderer).Render(count(r.Files, "file", "files")), sep,
		dimStyle(ctx.renderer).Render(count(r.Lines, "line", "lines")))
	fmt.Fprintf(w, "  %s%s%s%s%s\n",
		severityStyle(ctx.renderer, Error).Render(count(errs, "error", "errors")), sep,
		severityStyle(ctx.renderer, Warn).Render(count(warn, "warning", "warnings")), sep,
		severityStyle(ctx.renderer, Info).Render(count(info, "info", "info")))
	if roast {
		grade := r.Grade()
		fmt.Fprintf(w, "  %s%s%s\n",
			gradeStyle(ctx.renderer, grade).Render("grade "+grade), sep,
			dimStyle(ctx.renderer).Render(verdicts[grade]))
	}
}

func severityStyle(r *lipgloss.Renderer, s Severity) lipgloss.Style {
	switch s {
	case Warn:
		return r.NewStyle().Foreground(lipgloss.Color("yellow"))
	case Error:
		return r.NewStyle().Foreground(lipgloss.Color("red"))
	default:
		return r.NewStyle().Foreground(lipgloss.Color("blue"))
	}
}

func gradeStyle(r *lipgloss.Renderer, g string) lipgloss.Style {
	switch g {
	case "A", "B":
		return r.NewStyle().Foreground(lipgloss.Color("green"))
	case "C":
		return r.NewStyle().Foreground(lipgloss.Color("yellow"))
	default:
		return r.NewStyle().Foreground(lipgloss.Color("red"))
	}
}

func dimStyle(r *lipgloss.Renderer) lipgloss.Style {
	return r.NewStyle().Foreground(lipgloss.Color("grey")).Faint(true)
}

func boldStyle(r *lipgloss.Renderer) lipgloss.Style {
	return r.NewStyle().Bold(true)
}
