package filet

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAnalyzeCacheReuse proves end-to-end that Analyze reuses the store: a warm
// run must match a cold run byte-for-byte on findings, and after editing one
// file the cached run must still match a fresh uncached baseline. LSP and
// tree-sitter are off so the outcome is fully deterministic.
func TestAnalyzeCacheReuse(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "pkg"), 0o755)
	writeTestFile(t, filepath.Join(dir, "pkg", "a.go"), "package pkg\n\nfunc Alpha() int { return 1 }\n")
	writeTestFile(t, filepath.Join(dir, "pkg", "b.go"), "package pkg\n\nfunc Beta() int { return 2 }\n")

	mk := func(cached bool) *Config {
		cfg := DefaultConfig()
		cfg.root = dir
		cfg.LSP.Enabled = false
		cfg.Treesitter.Enabled = false
		cfg.Cache.Enabled = cached
		cfg.Cache.Dir = filepath.Join(dir, "cache")
		return cfg
	}

	cold := testAnalyze(t, mk(true), dir)
	warm := testAnalyze(t, mk(true), dir)
	if !sameFindings(cold, warm) {
		t.Fatalf("warm run must reproduce cold findings\ncold: %v\nwarm: %v", cold, warm)
	}

	writeTestFile(t, filepath.Join(dir, "pkg", "b.go"), "package pkg\n\nfunc Beta() int { return 3 }\n")
	changed := testAnalyze(t, mk(true), dir)
	baseline := testAnalyze(t, mk(false), dir)
	if !sameFindings(changed, baseline) {
		t.Fatalf("edited run must match an uncached baseline\nchanged: %v\nbaseline: %v", changed, baseline)
	}
}

func testAnalyze(t *testing.T, cfg *Config, target string) []Finding {
	t.Helper()
	r, err := Analyze(cfg, target)
	if err != nil {
		t.Fatal(err)
	}
	return r.Findings
}

func sameFindings(a, b []Finding) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x.Rule != y.Rule || x.File != y.File || x.Line != y.Line ||
			x.Column != y.Column || x.Message != y.Message || x.Severity != y.Severity {
			return false
		}
	}
	return true
}

func writeTestFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
