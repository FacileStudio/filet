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
		usageSection{title: "Commands", rows: []usageRow{
			usageRow{cmd: "filet check", detail: "[flags] [path]", desc: "Run style, quality and architecture rules."},
			usageRow{cmd: "filet roast", detail: "[flags] [path]", desc: "Run the same checks, with punchlines."},
			usageRow{cmd: "filet docker", detail: "[flags] [path]", desc: "Roast every Dockerfile found under path."},
			usageRow{cmd: "filet test", detail: "[flags] [path]", desc: "Detect and run the project's test suites."},
			usageRow{cmd: "filet init", detail: "[flags] [path]", desc: "Scaffold filet.yml (-preset relaxed|epitech)."},
			usageRow{cmd: "filet rules", detail: "", desc: "List every rule id."},
			usageRow{cmd: "filet version", detail: "", desc: "Print the version."},
		}},
		usageSection{title: "Droast", rows: []usageRow{
			usageRow{cmd: "droast", detail: "", desc: "Alias for \"filet docker\"."},
		}},
		usageSection{title: "Flags", rows: []usageRow{
			usageRow{cmd: "-format", detail: "auto|text|lipgloss|line|json|sarif|github", desc: "Output format. Default: auto."},
			usageRow{cmd: "-fail", detail: "info|warn|error|never", desc: "Severity that makes the command fail."},
			usageRow{cmd: "-quiet", detail: "", desc: "Print the summary only."},
		}},
		usageSection{title: "Examples", rows: []usageRow{
			usageRow{cmd: "filet check .", detail: "", desc: ""},
			usageRow{cmd: "filet roast ./apps ./pkg", detail: "", desc: ""},
			usageRow{cmd: "filet docker .", detail: "", desc: ""},
			usageRow{cmd: "filet test ./apps/api", detail: "", desc: ""},
			usageRow{cmd: "filet init -preset relaxed", detail: "", desc: ""},
		}},
	}
}

func usageBold(r *lipgloss.Renderer) lipgloss.Style {
	return r.NewStyle().Bold(true)
}

func usageDim(r *lipgloss.Renderer) lipgloss.Style {
	return r.NewStyle().Foreground(lipgloss.Color("grey")).Faint(true)
}
