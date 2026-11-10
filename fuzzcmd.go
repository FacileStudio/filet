package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/FacileStudio/filet/internal/filet"
)

type fuzzFlags struct {
	mode     string
	fuzzTime time.Duration
	budget   time.Duration
	list     bool
	format   string
}

func fuzz(args []string) int {
	fs := flag.NewFlagSet("fuzz", flag.ContinueOnError)
	f := fuzzFlags{}
	fs.StringVar(&f.mode, "mode", "seeds", "seeds replays the corpus, smoke fuzzes each target")
	fs.DurationVar(&f.fuzzTime, "fuzztime", filet.DefaultFuzzTime, "smoke mode: fuzzing time per target")
	fs.DurationVar(&f.budget, "budget", 5*time.Minute, "smoke mode: stop starting targets once spent")
	fs.BoolVar(&f.list, "list", false, "print the targets and exit")
	fs.StringVar(&f.format, "format", "text", "summary format: text or json")
	target, err := parseWithTarget(fs, args)
	if err != nil {
		return 2
	}
	opts, err := f.options()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	targets, err := discoverFuzz(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if f.list {
		return listFuzz(targets, f.format == "json")
	}
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "no fuzz target found under "+target)
		return 0
	}
	out := os.Stdout
	if f.format == "json" {
		out = os.Stderr
	}
	return reportSteps(filet.RunFuzz(targets, opts, out), f.format == "json", out)
}

func (f fuzzFlags) options() (filet.FuzzOptions, error) {
	mode, err := filet.ParseFuzzMode(f.mode)
	if err != nil {
		return filet.FuzzOptions{}, err
	}
	if f.fuzzTime <= 0 {
		return filet.FuzzOptions{}, fmt.Errorf("-fuzztime must be positive, got %s", f.fuzzTime)
	}
	if f.format != "text" && f.format != "json" {
		return filet.FuzzOptions{}, fmt.Errorf("-format: unknown format %q (use text or json)", f.format)
	}
	return filet.FuzzOptions{Mode: mode, FuzzTime: f.fuzzTime, Budget: f.budget}, nil
}

func discoverFuzz(target string) ([]filet.FuzzTarget, error) {
	cfg, path, err := filet.LoadConfig(target)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return filet.DiscoverFuzzTargets(cfg, target)
}

func listFuzz(targets []filet.FuzzTarget, asJSON bool) int {
	if asJSON {
		payload := struct {
			Targets []filet.FuzzTarget `json:"targets"`
		}{targets}
		if err := filet.WriteJSONValue(os.Stdout, payload); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		return 0
	}
	for _, t := range targets {
		fmt.Printf("%-5s %-32s %s\n", t.Lang, t.Name, filet.DisplayPath(t.Dir))
	}
	return 0
}
