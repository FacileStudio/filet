package filet

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type usageRow struct {
	cmd    string
	detail string
	desc   string
}

type usageSection struct {
	title string
	rows  []usageRow
}

// RenderUsage prints the help text, styled through lipgloss like the rest of
// the output. The Ascii color profile keeps it plain on non-terminal writers.
func RenderUsage(w io.Writer) {
	r := newRenderer(w)
	fmt.Fprintln(w, lipgloss.JoinHorizontal(lipgloss.Top,
		usageBold(r).Render("filet"),
		usageDim(r).Render(" · style checker, code roaster and test runner")))
	fmt.Fprintln(w)
	for _, sec := range usageSections() {
		writeUsageSection(w, r, sec)
	}
}

func writeUsageSection(w io.Writer, r *lipgloss.Renderer, s usageSection) {
	title := usageBold(r).Foreground(lipgloss.Color("cyan"))
	fmt.Fprintln(w, title.Render(s.title))
	width := usageWidth(s.rows)
	for _, row := range s.rows {
		left := usageLeft(r, row)
		if row.desc == "" {
			fmt.Fprintln(w, "  "+left)
			continue
		}
		pad := width + 2 - plainLeft(row)
		fmt.Fprintf(w, "  %s%s\n", left, strings.Repeat(" ", pad)+row.desc)
	}
	fmt.Fprintln(w)
}

func usageLeft(r *lipgloss.Renderer, row usageRow) string {
	left := usageBold(r).Render(row.cmd)
	if row.detail != "" {
		left += " " + usageDim(r).Render(row.detail)
	}
	return left
}

func plainLeft(row usageRow) int {
	if row.detail == "" {
		return len(row.cmd)
	}
	return len(row.cmd) + 1 + len(row.detail)
}

func usageWidth(rows []usageRow) int {
	width := 0
	for _, row := range rows {
		if plainLeft(row) > width {
			width = plainLeft(row)
		}
	}
	return width
}

func usageSections() []usageSection {
	return []usageSection{
		{title: "Commands", rows: []usageRow{
			{cmd: "filet check", detail: "[flags] [path]", desc: "Run style, quality and architecture rules."},
			{cmd: "filet roast", detail: "[flags] [path]", desc: "Run the same checks, with punchlines."},
			{cmd: "filet clean", detail: "[-dry-run] [path]", desc: "Apply the auto-fixable checks in place."},
			{cmd: "filet docker", detail: "[flags] [path]", desc: "Roast every Dockerfile found under path."},
			{cmd: "filet test", detail: "[flags] [path]", desc: "Detect and run the project's test suites."},
			{cmd: "filet fuzz", detail: "[flags] [path]", desc: "Replay or smoke-fuzz Go and Rust fuzz targets."},
			{cmd: "filet gate", detail: "[flags] [path]", desc: "Run format, check, analyzers, tests, fuzz seeds, douane."},
			{cmd: "filet init", detail: "[flags] [path]", desc: "Scaffold filet.yml (-preset relaxed|epitech)."},
			{cmd: "filet rules", detail: "", desc: "List every rule id."},
			{cmd: "filet version", detail: "", desc: "Print the version."},
		}},
		{title: "Droast", rows: []usageRow{
			{cmd: "droast", detail: "", desc: "Alias for \"filet docker\"."},
		}},
		{title: "Flags", rows: []usageRow{
			{cmd: "-format", detail: "auto|text|lipgloss|line|json|sarif|github", desc: "Output format. Default: auto."},
			{cmd: "-fail", detail: "info|warn|error|never", desc: "Severity that makes the command fail."},
			{cmd: "-quiet", detail: "", desc: "Print the summary only."},
			{cmd: "-baseline", detail: "write|update", desc: "check only: write or lower the ratchet baseline."},
		}},
		{title: "Examples", rows: usageExamples()},
	}
}

func usageBold(r *lipgloss.Renderer) lipgloss.Style {
	return r.NewStyle().Bold(true)
}

func usageDim(r *lipgloss.Renderer) lipgloss.Style {
	return r.NewStyle().Foreground(lipgloss.Color("grey")).Faint(true)
}
