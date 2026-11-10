package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/FacileStudio/filet/internal/filet"
)

func gate(args []string) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	keepGoing := fs.Bool("keep-going", false, "run every step even after one fails")
	format := fs.String("format", "text", "summary format: text or json")
	target, err := parseWithTarget(fs, args)
	if err != nil {
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "-format: unknown format %q (use text or json)\n", *format)
		return 2
	}
	plan, gc, err := planGate(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	out := os.Stdout
	if *format == "json" {
		out = os.Stderr
	}
	results := filet.RunGate(plan.Steps, filet.GateOptions{KeepGoing: *keepGoing, Config: gc}, out)
	filet.WriteUnits(out, plan.Units)
	if *format == "json" {
		return emitGate(results, plan.Units)
	}
	return reportSteps(results, false, out)
}

func emitGate(results []filet.StepResult, units []filet.Unit) int {
	doc := filet.GateReport{Passed: !filet.AnyFailed(results), Units: units, Steps: results}
	if err := filet.WriteJSONValue(os.Stdout, doc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if !doc.Passed {
		return 1
	}
	return 0
}

func planGate(target string) (filet.GatePlan, filet.GateConfig, error) {
	cfg, path, err := filet.LoadConfig(target)
	if err != nil {
		return filet.GatePlan{}, filet.GateConfig{}, fmt.Errorf("%s: %w", path, err)
	}
	gc, err := cfg.GateSettings()
	if err != nil {
		return filet.GatePlan{}, filet.GateConfig{}, fmt.Errorf("%s: gate: %w", path, err)
	}
	plan, err := filet.PlanGate(cfg, target)
	return plan, gc, err
}

func reportSteps(results []filet.StepResult, asJSON bool, out *os.File) int {
	if asJSON {
		if err := filet.WriteJSONValue(os.Stdout, filet.Summarize(results)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	} else {
		filet.WriteSteps(out, results)
	}
	if filet.AnyFailed(results) {
		return 1
	}
	return 0
}

func parseWithTarget(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() == 0 {
		return ".", nil
	}
	target := fs.Arg(0)
	if err := fs.Parse(fs.Args()[1:]); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "takes at most one path")
		return "", flag.ErrHelp
	}
	return target, nil
}
