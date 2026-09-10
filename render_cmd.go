package main

import (
	"os"

	"github.com/FacileStudio/filet/internal/filet"
)

// render picks the output format and writes the report. resolveFormat handles
// "auto"; every other format dispatches straight to its writer. Called from
// emit, with the format string already validated by parseCommon.
func render(report filet.Report, c *commonFlags, roast bool) error {
	switch resolveFormat(c.format) {
	case "json":
		return filet.WriteJSON(os.Stdout, report)
	case "sarif":
		return filet.WriteSARIF(os.Stdout, report, version)
	case "github":
		return filet.WriteGitHub(os.Stdout, report)
	case "lipgloss":
		filet.WriteTextLipgloss(os.Stdout, report, filet.TextOptions{Roast: roast, Quiet: c.quiet})
		return nil
	case "line":
		if c.quiet {
			return nil
		}
		return filet.WriteLines(os.Stdout, report)
	default:
		filet.WriteText(os.Stdout, report, filet.TextOptions{Roast: roast, Quiet: c.quiet})
		return nil
	}
}

// resolveFormat turns "auto" into the grouped human report on a terminal and the
// one-finding-per-line format everywhere else, so a pipe gets something parseable
// without anyone having to pass a flag.
func resolveFormat(format string) string {
	if format != "auto" {
		return format
	}
	if filet.IsTTY(os.Stdout) {
		return "text"
	}
	return "line"
}

func validFormat(format string) bool {
	switch format {
	case "auto", "text", "lipgloss", "line", "json", "sarif", "github":
		return true
	}
	return false
}
