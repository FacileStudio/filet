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
	if err := applyGlobalLSP(cfg); err != nil {
		return nil, "", err
	}
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
		cfg.FailOn = "info"
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
		if isRepoRoot(cur) || parent == cur {
			return "", nil
		}
		cur = parent
	}
}

// isRepoRoot reports whether dir holds a repository marker. The search for a
// config stops there: a config living outside the repository would apply on a
// developer's machine and vanish in CI, where only the repository is checked
// out, and the two runs would disagree with nothing to show for it.
func isRepoRoot(dir string) bool {
	for _, marker := range []string{".git", ".hg", ".svn"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
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

// applyGlobalLSP overlays the user's server recipes from a global config file
// ($XDG_CONFIG_HOME/filet/filet.yml, else ~/.config/filet/filet.yml) onto the
// defaults. The layer touches only lsp.servers: gate switches like enabled,
// fail and failOn stay per-repository, so a user pointing filet at their own
// servers cannot silently change anyone else's verdict. A malformed global file
// is ignored rather than failing the run.
func applyGlobalLSP(cfg *Config) error {
	raw := globalServerRaw()
	if raw == nil {
		return nil
	}
	var layer struct {
		LSP struct {
			Servers map[string]Server `yaml:"servers"`
		} `yaml:"lsp"`
	}
	if err := yaml.Unmarshal(raw, &layer); err != nil {
		return nil
	}
	for lang, srv := range layer.LSP.Servers {
		cfg.LSP.Servers[lang] = srv
	}
	return nil
}

// globalServerRaw returns the raw bytes of a global lsp.servers layer, if any.
func globalServerRaw() []byte {
	for _, dir := range globalConfigDirs() {
		_, raw := configIn(dir)
		if raw != nil {
			return raw
		}
	}
	return nil
}

func globalConfigDirs() []string {
	var dirs []string
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		dirs = append(dirs, filepath.Join(dir, "filet"))
	}
	if home := os.Getenv("HOME"); home != "" {
		dirs = append(dirs, filepath.Join(home, ".config", "filet"))
	}
	return dirs
}
