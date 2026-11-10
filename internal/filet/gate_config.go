package filet

import (
	"fmt"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

// GateConfig is the gate: block of filet.yml. Disable names steps the gate
// skips, by phase (format, check, analyze, test, fuzz, douane) or by step name
// (gofmt, vet, staticcheck, ...). RequireTools turns a missing tool from a
// skipped row into a failed one.
type GateConfig struct {
	Disable      []string `yaml:"disable"`
	RequireTools []string `yaml:"requireTools"`
}

var gateStepNames = []string{
	"format", "check", "analyze", "test", "fuzz", "douane",
	"gofmt", "cargo-fmt", "vet", "staticcheck", "clippy", "bun-check",
	"go", "cargo", "deno", "bun", "pnpm", "yarn", "npm", "pytest", "rust", "node",
}

var gateToolNames = []string{
	"gofmt", "go", "staticcheck", "cargo", "cargo-fmt", "cargo-clippy", "cargo-fuzz",
	"bun", "deno", "pnpm", "yarn", "npm", "pytest", "douane",
}

func (g GateConfig) validate() error {
	if err := knownNames("disable", "step", g.Disable, gateStepNames); err != nil {
		return err
	}
	return knownNames("requireTools", "tool", g.RequireTools, gateToolNames)
}

func knownNames(key, what string, names, valid []string) error {
	for _, n := range names {
		if !slices.Contains(valid, n) {
			sorted := slices.Sorted(slices.Values(valid))
			return fmt.Errorf("%s: unknown %s %q (valid: %s)", key, what, n, strings.Join(sorted, ", "))
		}
	}
	return nil
}

func (g GateConfig) disables(s GateStep) bool {
	return slices.Contains(g.Disable, s.Phase) || slices.Contains(g.Disable, s.Name)
}

type configExtras struct {
	Gate  GateConfig `yaml:"gate"`
	Style struct {
		HintFuzz bool `yaml:"hintFuzz"`
	} `yaml:"style"`
}

func (c *Config) extras() (configExtras, error) {
	var e configExtras
	if len(c.raw) == 0 {
		return e, nil
	}
	err := decodeYAML(c.raw, &e)
	return e, err
}

// decodeYAML unmarshals raw into v and turns a decoder panic into an error.
// goccy/go-yaml v1.19.2 dereferences nil on a tagged scalar decoded into a
// slice (`ignore: !x`), which FuzzLoadConfig found; a malformed filet.yml must
// be reported, not crash the run.
func decodeYAML(raw []byte, v any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("malformed YAML: %v", r)
		}
	}()
	return yaml.Unmarshal(raw, v)
}

// GateSettings returns the gate: block of the loaded config file, or the zero
// value when the project has no config file. A name in disable or
// requireTools that no step or tool carries is an error naming the valid ones,
// so a typo cannot quietly loosen the gate.
func (c *Config) GateSettings() (GateConfig, error) {
	e, err := c.extras()
	if err != nil {
		return GateConfig{}, err
	}
	return e.Gate, e.Gate.validate()
}

func (c *Config) hintFuzz() bool {
	e, err := c.extras()
	return err == nil && e.Style.HintFuzz
}
