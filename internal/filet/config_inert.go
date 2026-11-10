package filet

import (
	"fmt"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

// keyReach names a config key whose rules run on a bounded set of languages,
// and reports whether a file extension is one of them. Keys that drive a
// language-agnostic rule are absent: they can always fire.
type keyReach struct {
	key   string
	fires func(*Config, string) bool
}

var keyReaches = []keyReach{
	{"limits.funcsPerFile", goOrFuncPattern},
	{"limits.funcLines", goOrGrammar},
	{"limits.funcStatements", goOrGrammar},
	{"limits.params", goOrGrammar},
	{"limits.complexity", goOrGrammar},
	{"limits.returns", goOnly},
	{"limits.structFields", goOnly},
	{"limits.interfaceMethods", goOnly},
	{"style.requireDocComments", goOnly},
	{"style.banGlobalMutable", goOnly},
	{"style.banInit", goOnly},
	{"style.format", goOnly},
}

func goOnly(_ *Config, ext string) bool { return ext == ".go" }

func goOrGrammar(c *Config, ext string) bool { return ext == ".go" || c.UsesTreesitter(ext) }

func goOrFuncPattern(_ *Config, ext string) bool { return ext == ".go" || funcDeclRe[ext] != nil }

// InertKeys reports, as gen.config.inert warnings on the config file, every
// key filet.yml switches on that cannot fire for any extension in
// `extensions:`. A config is a statement about the code; one that bans global
// mutables in a TypeScript-only repository states a rule nothing enforces.
// Only keys written in the file count, and only on a run over the config root,
// so a scoped run on one file does not repeat a warning about the whole config.
func InertKeys(cfg *Config, target string) []Finding {
	if len(cfg.raw) == 0 || !cfg.Enabled("gen.config.inert") || RootRelative(cfg.root, target) != "." {
		return nil
	}
	set, err := writtenKeys(cfg.raw)
	if err != nil {
		return nil
	}
	path, _ := configIn(cfg.root)
	var out []Finding
	for _, k := range keyReaches {
		reached := slices.ContainsFunc(cfg.Extensions, func(ext string) bool { return k.fires(cfg, ext) })
		if set[k.key] && !reached {
			out = append(out, newFinding("gen.config.inert", DisplayPath(path), keyLine(cfg.raw, k.key), Warn,
				fmt.Sprintf("%s is set but no extension in [%s] runs a rule it drives",
					k.key, strings.Join(cfg.Extensions, ", "))))
		}
	}
	return out
}

func writtenKeys(raw []byte) (map[string]bool, error) {
	var head struct {
		Limits map[string]int  `yaml:"limits"`
		Style  map[string]bool `yaml:"style"`
	}
	if err := yaml.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for k, v := range head.Limits {
		set["limits."+k] = v > 0
	}
	for k, v := range head.Style {
		set["style."+k] = v
	}
	return set, nil
}

func keyLine(raw []byte, key string) int {
	leaf := key[strings.IndexByte(key, '.')+1:]
	for i, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), leaf+":") {
			return i + 1
		}
	}
	return 1
}
