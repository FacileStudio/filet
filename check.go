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
	fs.StringVar(&c.format, "format", "text", "output format: text or json")
	fs.StringVar(&c.fail, "fail", "", "severity that makes the command exit 1")
	fs.BoolVar(&c.quiet, "quiet", false, "only print the summary")
	if err := fs.Parse(args); err != nil {
		return nil, err
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
	switch c.format {
	case "json":
		if err := filet.WriteJSON(os.Stdout, report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	default:
		filet.WriteText(os.Stdout, report, filet.TextOptions{Roast: roast, Quiet: c.quiet})
	}
	if strings.EqualFold(cfg.FailOn, "never") {
		return 0
	}
	if filet.FailLevel(cfg, report) {
		return 1
	}
	return 0
}
