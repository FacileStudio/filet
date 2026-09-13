package filet

import (
	"os"
	"path/filepath"
	"testing"
)

var ignoreTree = []string{
	"main.go",
	"gen/manual.go",
	"data/train.go",
	"data/training/samples.go",
	"data/training/deep/samples.go",
	"gen/notes.gen.go",
	"vendor/x/a.go",
}

func writeIgnoreTree(t *testing.T, root string) {
	t.Helper()
	for _, f := range ignoreTree {
		p := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeTemp(p, "package main\n"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIgnorePathPatterns(t *testing.T) {
	root := t.TempDir()
	writeIgnoreTree(t, root)

	cfg := DefaultConfig()
	cfg.root = root
	cfg.Ignore = append(cfg.Ignore, "data/training/**", "gen/*.gen.go", "vendor/*")

	got, err := Scan(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range got {
		seen[filepath.ToSlash(f.Rel)] = true
	}
	for _, want := range ignoreTree[:3] {
		if !seen[want] {
			t.Errorf("%s must survive the ignore patterns", want)
		}
	}
	for _, banned := range ignoreTree[3:] {
		if seen[banned] {
			t.Errorf("%s must be ignored", banned)
		}
	}
}
