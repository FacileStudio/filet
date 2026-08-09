package filet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// LoadConfig walks up from dir looking for a config file and merges it over the defaults.
func LoadConfig(dir string) (*Config, string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", err
	}

	path, raw := findConfig(abs)
	cfg := DefaultConfig()
	if path == "" {
		cfg.root = abs
		return cfg, "", cfg.compile()
	}

	if err := applyPreset(cfg, raw); err != nil {
		return nil, path, err
	}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, path, err
	}
	cfg.root = filepath.Dir(path)
	if cfg.FailOn == "" {
		cfg.FailOn = "error"
	}
	if !ValidFailOn(cfg.FailOn) {
		return nil, path, fmt.Errorf("failOn: unknown severity %q (use info, warn, error or never)", cfg.FailOn)
	}
	return cfg, path, cfg.compile()
}

func findConfig(start string) (string, []byte) {
	for cur := start; ; {
		if path, raw := configIn(cur); path != "" {
			return path, raw
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", nil
		}
		cur = parent
	}
}

func configIn(dir string) (string, []byte) {
	for _, name := range ConfigNames() {
		candidate := filepath.Join(dir, name)
		if raw, err := os.ReadFile(candidate); err == nil {
			return candidate, raw
		}
	}
	return "", nil
}

func applyPreset(cfg *Config, raw []byte) error {
	var head struct {
		Preset string `yaml:"preset"`
	}
	if err := yaml.Unmarshal(raw, &head); err != nil {
		return err
	}
	if head.Preset == "" {
		return nil
	}
	apply, ok := Preset(head.Preset)
	if !ok {
		return fmt.Errorf("unknown preset %q (known: %s)", head.Preset, strings.Join(PresetNames(), ", "))
	}
	apply(cfg)
	return nil
}
