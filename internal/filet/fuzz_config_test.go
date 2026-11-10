package filet

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoadConfig(f *testing.F) {
	if seed, err := os.ReadFile(filepath.Join("..", "..", "filet.yml")); err == nil {
		f.Add(seed)
	}
	f.Add([]byte("preset: relaxed\nfailOn: warn\n"))
	f.Add([]byte("architecture:\n  maxFilesPerDir:\n    .: 20\n    'internal/*': 0\n"))
	f.Add([]byte("architecture:\n  maxFilesPerDir: 7\n  fileNamePattern: '['\n"))
	f.Add([]byte("gate:\n  disable: [douane]\n  requireTools: [staticcheck]\nstyle:\n  hintFuzz: true\n"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "filet.yml"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
		cfg, _, err := LoadConfig(dir)
		if err != nil {
			return
		}
		if !ValidFailOn(cfg.FailOn) {
			t.Fatalf("loaded a config with failOn %q", cfg.FailOn)
		}
		if gate, err := cfg.GateSettings(); err == nil && gate.disables(GateStep{}) {
			t.Fatalf("an unnamed step must never match gate.disable %v", gate.Disable)
		}
	})
}
