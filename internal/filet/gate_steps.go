package filet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

// GateSteps builds the ordered step list for target from the suites
// DetectSuites finds there: format, check, analyzers, tests, fuzz seeds and
// douane. A step that does not apply to the project is left out; one that
// applies but whose tool is missing stays in and is reported as skipped.
func GateSteps(cfg *Config, target string) ([]GateStep, error) {
	suites := DetectSuites(target)
	fuzzRows, err := fuzzSteps(cfg, target)
	if err != nil {
		return nil, err
	}
	steps := formatSteps(target, suites)
	steps = append(steps, checkStep(cfg, target))
	steps = append(steps, analyzeSteps(target, suites)...)
	steps = append(steps, testSteps(suites)...)
	steps = append(steps, fuzzRows...)
	return append(steps, douaneStep(target)), nil
}

func formatSteps(target string, suites []Suite) []GateStep {
	var out []GateStep
	if hasSuite(suites, "go") {
		out = append(out, gofmtStep(target))
	}
	if hasSuite(suites, "cargo") {
		out = append(out, suiteStep("format", "cargo-fmt", "cargo-fmt",
			Suite{Name: "cargo-fmt", Dir: target, Cmd: "cargo", Args: []string{"fmt", "--check"}}))
	}
	return out
}

func analyzeSteps(target string, suites []Suite) []GateStep {
	var out []GateStep
	if hasSuite(suites, "go") {
		out = append(out,
			suiteStep("analyze", "vet", "go", Suite{Name: "vet", Dir: target, Cmd: "go", Args: []string{"vet", "./..."}}),
			suiteStep("analyze", "staticcheck", "staticcheck",
				Suite{Name: "staticcheck", Dir: target, Cmd: "staticcheck", Args: []string{"./..."}}))
	}
	if hasSuite(suites, "cargo") {
		out = append(out, suiteStep("analyze", "clippy", "cargo-clippy",
			Suite{Name: "clippy", Dir: target, Cmd: "cargo", Args: []string{"clippy", "--", "-D", "warnings"}}))
	}
	if hasScript(target, "check") {
		out = append(out, suiteStep("analyze", "bun-check", "bun",
			Suite{Name: "bun-check", Dir: target, Cmd: "bun", Args: []string{"run", "check"}}))
	}
	return out
}

func testSteps(suites []Suite) []GateStep {
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
	return suiteStep("douane", "douane", "douane", Suite{Name: "douane", Dir: target, Cmd: "douane",
		Args: []string{"scan", "-fail", "critical", "-scope", "prod", "."}})
}

func hasSuite(suites []Suite, name string) bool {
	return slices.ContainsFunc(suites, func(s Suite) bool { return s.Name == name })
}

func hasScript(dir, name string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return false
	}
	_, ok := pkg.Scripts[name]
	return ok
}
