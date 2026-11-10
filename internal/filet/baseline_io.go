package filet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

const baselineVersion = 1

// baselineFile is the on-disk shape of the ratchet: a format version and, for
// each rule, the count of findings per root-relative path. Counts, not line
// numbers, so an edit elsewhere in a file never invalidates it.
type baselineFile struct {
	Version int    `json:"version"`
	Counts  counts `json:"counts"`
}

// openRatchet finds and reads the baseline for cfg. It reports found false when
// there is no file, which is an error only when the run asks to update it or
// filet.yml names one: a configured ratchet that silently stops ratcheting is
// the one outcome worse than a failure.
func openRatchet(cfg *Config, target, mode string) (*ratchet, bool, error) {
	path, named, err := baselinePath(cfg)
	if err != nil {
		return nil, false, err
	}
	base, found, err := readBaseline(path)
	if err != nil {
		return nil, false, err
	}
	if !found && mode != "write" && (named || mode == "update") {
		return nil, false, fmt.Errorf("%s: no baseline; run filet check -baseline write", DisplayPath(path))
	}
	rc := &ratchet{cfg: cfg, path: path, scope: RootRelative(cfg.root, target), base: base}
	return rc, found, nil
}

func baselinePath(cfg *Config) (string, bool, error) {
	var head struct {
		Baseline string `yaml:"baseline"`
	}
	if len(cfg.raw) > 0 {
		if err := yaml.Unmarshal(cfg.raw, &head); err != nil {
			return "", false, err
		}
	}
	switch {
	case head.Baseline == "":
		return filepath.Join(cfg.root, BaselineName), false, nil
	case filepath.IsAbs(head.Baseline):
		return head.Baseline, true, nil
	}
	return filepath.Join(cfg.root, head.Baseline), true, nil
}

func readBaseline(path string) (counts, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return counts{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var file baselineFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, false, fmt.Errorf("%s: %w", DisplayPath(path), err)
	}
	if file.Version != baselineVersion {
		return nil, false, fmt.Errorf("%s: unsupported baseline version %d", DisplayPath(path), file.Version)
	}
	if file.Counts == nil {
		file.Counts = counts{}
	}
	return file.Counts, true, nil
}

func writeBaseline(path string, c counts) error {
	raw, err := json.MarshalIndent(baselineFile{Version: baselineVersion, Counts: c}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
