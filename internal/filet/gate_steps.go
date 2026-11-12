package filet

import (
	"slices"
)

// GatePlan is what one gate run covers: the units DiscoverUnits found under
// the target and the ordered steps built from them.
type GatePlan struct {
	Units []Unit
	Steps []GateStep
}

// PlanGate builds the ordered step list for target from the units under it:
// format, check, analyzers, tests, fuzz seeds and douane, each per-unit step
// run from its unit's directory. A step that does not apply is left out; one
// that applies but whose tool is missing stays in and is reported as skipped.
// A language with source files but no unit gets failed rows, never none.
func PlanGate(cfg *Config, target string) (GatePlan, error) {
	scan, err := DiscoverUnits(cfg, target)
	if err != nil {
		return GatePlan{}, err
	}
	fuzzRows, err := fuzzSteps(cfg, target, scan.Units)
	if err != nil {
		return GatePlan{}, err
	}
	missing := missingSteps(scan)
	steps := inPhase("format", formatSteps(cfg, scan.Units), missing)
	steps = append(steps, checkStep(cfg, target))
	steps = append(steps, inPhase("analyze", analyzeSteps(scan.Units), missing)...)
	steps = append(steps, inPhase("test", testSteps(scan.Units), missing)...)
	steps = append(steps, fuzzRows...)
	steps = append(steps, douaneStep(target))
	return GatePlan{Units: scan.Units, Steps: steps}, nil
}

func inPhase(phase string, steps, missing []GateStep) []GateStep {
	for _, s := range missing {
		if s.Phase == phase {
			steps = append(steps, s)
		}
	}
	return steps
}

func formatSteps(cfg *Config, units []Unit) []GateStep {
	var out []GateStep
	for _, u := range units {
		if hasSuite(u.Suites, "go") {
			out = append(out, gofmtStep(cfg, u.suite("gofmt", "gofmt", "-l", ".")))
		}
		if hasSuite(u.Suites, "cargo") {
			out = append(out, suiteStep("format", "cargo-fmt", "cargo-fmt", u.suite("cargo-fmt", "cargo", "fmt", "--check")))
		}
	}
	return out
}

func analyzeSteps(units []Unit) []GateStep {
	var out []GateStep
	for _, u := range units {
		if hasSuite(u.Suites, "go") {
			out = append(out,
				suiteStep("analyze", "vet", "go", u.suite("vet", "go", "vet", "./...")),
				suiteStep("analyze", "staticcheck", "staticcheck", u.suite("staticcheck", "staticcheck", "./...")))
		}
		if hasSuite(u.Suites, "cargo") {
			out = append(out, suiteStep("analyze", "clippy", "cargo-clippy",
				u.suite("clippy", "cargo", "clippy", "--", "-D", "warnings")))
		}
		if u.Check {
			out = append(out, suiteStep("analyze", "bun-check", "bun", u.suite("bun-check", "bun", "run", "check")))
		}
	}
	return out
}

func testSteps(units []Unit) []GateStep {
	suites := AllSuites(units)
	out := make([]GateStep, 0, len(suites))
	for _, s := range suites {
		if s.Name == "go" {
			s.Args = []string{"test", "-race", "./..."}
		}
		out = append(out, suiteStep("test", s.Name, s.Cmd, s))
	}
	return out
}

func douaneStep(target string) GateStep {
	return suiteStep("douane", "douane", "douane", Suite{Name: "douane", Dir: target, Unit: ".", Cmd: "douane",
		Args: []string{"scan", "-fail", "critical", "-scope", "prod", "."}})
}

func hasSuite(suites []Suite, name string) bool {
	return slices.ContainsFunc(suites, func(s Suite) bool { return s.Name == name })
}
