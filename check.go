package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/saravenpi/filet/internal/filet"
)

type commonFlags struct {
	format string
	fail   string
	quiet  bool
	target string
}

func parseCommon(name string, args []string) (*commonFlags, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	c := &commonFlags{}
	fs.StringVar(&c.format, "format", "auto", "output format: auto, text, line or json")
	fs.StringVar(&c.fail, "fail", "", "severity that makes the command exit 1")
	fs.BoolVar(&c.quiet, "quiet", false, "only print the summary")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if !validFormat(c.format) {
		return nil, fmt.Errorf("-format: unknown format %q (use auto, text, line or json)", c.format)
	}
	c.target = "."
	if fs.NArg() > 0 {
		c.target = fs.Arg(0)
		if err := fs.Parse(fs.Args()[1:]); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func loadConfig(c *commonFlags) (*filet.Config, error) {
	cfg, path, err := filet.LoadConfig(c.target)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if c.fail != "" {
		if !filet.ValidFailOn(c.fail) {
			return nil, fmt.Errorf("-fail: unknown severity %q (use info, warn, error or never)", c.fail)
		}
		cfg.FailOn = c.fail
	}
	return cfg, nil
}

func analyze(name string, args []string, roast bool) int {
	c, err := parseCommon(name, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	cfg, err := loadConfig(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	report, err := filet.Analyze(cfg, c.target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return emit(cfg, report, c, roast)
}

func docker(args []string) int {
	c, err := parseCommon("docker", args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	cfg, err := loadConfig(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	report, err := filet.AnalyzeDockerfiles(cfg, c.target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if report.Files == 0 {
		fmt.Fprintln(os.Stderr, "no Dockerfile found under "+c.target)
		return 0
	}
	return emit(cfg, report, c, true)
}

func emit(cfg *filet.Config, report filet.Report, c *commonFlags, roast bool) int {
	if roast {
		filet.Roast(report.Findings)
	}
	if err := render(report, c, roast); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if strings.EqualFold(cfg.FailOn, "never") {
		return 0
	}
	if filet.FailLevel(cfg, report) {
		return 1
	}
	return 0
}

func render(report filet.Report, c *commonFlags, roast bool) error {
	switch resolveFormat(c.format) {
	case "json":
		return filet.WriteJSON(os.Stdout, report)
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
	case "auto", "text", "line", "json":
		return true
	}
	return false
}
