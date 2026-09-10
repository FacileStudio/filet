package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/FacileStudio/filet/internal/filet"
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
	fs.StringVar(&c.format, "format", "auto", "output format: auto, text, lipgloss, line, json, sarif or github")
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

	if !validFormat(c.format) {
		return nil, fmt.Errorf("-format: unknown format %q (use auto, text, lipgloss, line, json, sarif or github)", c.format)
	}
	return c, nil
}

// loadConfig resolves the project's rules, and says so on stderr when it found
// none.
//
// The notice exists because the alternative is a silent disagreement: a config
// that is not found — wrong filename, a directory above the repository root, a
// binary too old to know the name — still produces a full report, and a report
// against the defaults reads exactly like a report against the project's own
// rules. It goes to stderr so that no output format changes.
func loadConfig(c *commonFlags) (*filet.Config, error) {
	cfg, path, err := filet.LoadConfig(c.target)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if path == "" {
		fmt.Fprintf(os.Stderr, "filet: no %s under %s, checking against the defaults\n",
			filet.ConfigNames()[0], c.target)
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

// clean applies every line-level auto-fix the project's rules ask for, writing
// each changed file back in place. It is the write-mode sibling of check: check
// reports a trailing comment, clean removes it.
func clean(args []string) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.Usage = func() {}
	dryRun := fs.Bool("dry-run", false, "report what would change without writing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "filet clean takes at most one path")
		return 2
	}
	target := "."
	if fs.NArg() == 1 {
		target = fs.Arg(0)
	}
	cfg, err := loadConfig(&commonFlags{target: target})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	report, err := filet.Clean(cfg, target, *dryRun)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return printClean(report, *dryRun)
}

func printClean(report filet.CleanReport, dryRun bool) int {
	for _, res := range report.PerFile {
		if !res.Changed {
			continue
		}
		verb := "cleaned "
		if dryRun {
			verb = "would clean "
		}
		fmt.Printf("%s%s (%s)\n", verb, res.Path, cleanDetail(res.Stats))
	}
	verb := "files changed"
	if dryRun {
		verb = "files would change"
	}
	fmt.Printf("%d %s, %d fixes (%s)\n",
		report.Changed, verb,
		report.Stats.CommentedCode+report.Stats.InlineComment+report.Stats.InBodyComment+report.Stats.TrailingSpace+report.Stats.Formatting,
		cleanDetail(report.Stats))
	return 0
}

func cleanDetail(s filet.CleanStats) string {
	var parts []string
	if s.CommentedCode > 0 {
		parts = append(parts, fmt.Sprintf("%d commented-code", s.CommentedCode))
	}
	if s.InlineComment > 0 {
		parts = append(parts, fmt.Sprintf("%d inline-comment", s.InlineComment))
	}
	if s.InBodyComment > 0 {
		parts = append(parts, fmt.Sprintf("%d in-body", s.InBodyComment))
	}
	if s.TrailingSpace > 0 {
		parts = append(parts, fmt.Sprintf("%d trailing-space", s.TrailingSpace))
	}
	if s.Formatting > 0 {
		parts = append(parts, fmt.Sprintf("%d formatted", s.Formatting))
	}
	if len(parts) == 0 {
		return "no fixes"
	}
	return strings.Join(parts, ", ")
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
