package filet

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeModule(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeTemp(path, body); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}
}

func moduleSources(t *testing.T, dir string) []SourceFile {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.root = dir
	var out []SourceFile
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".go" {
			continue
		}
		f, err := readSource(cfg, filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, f)
	}
	return out
}

func TestLeakRuleResolvesModuleImports(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, map[string]string{
		"go.mod":     "module example.com/probe\n\ngo 1.24\n",
		"sub/sub.go": "package sub\n\ntype Handle struct{}\n\nfunc (h *Handle) Close() error { return nil }\n\nfunc Open() *Handle { return &Handle{} }\n",
		"main.go":    "package main\n\nimport \"example.com/probe/sub\"\n\nfunc main() {\n\th := sub.Open()\n\t_ = h\n}\n",
	})
	got := checkGoFiles(DefaultConfig(), moduleSources(t, dir))
	if n := countRule(got, "go.leak.resource"); n != 1 {
		t.Fatalf("expected 1 go.leak.resource finding for a leaked handle from a module import, got %d in %v", n, ruleIDs(got))
	}
}

func TestLeakSkipReportedOncePerPackage(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"go.mod": "module example.com/probe\n\ngo 1.24\n",
		"a.go":   "package main\n\nimport \"example.com/probe/missing\"\n\nfunc main() {\n\tmissing.Run()\n}\n",
		"b.go":   "package main\n\nfunc helper() {}\n",
	})
	got := checkGoFiles(DefaultConfig(), moduleSources(t, dir))
	if n := countRule(got, "go.leak.resource"); n != 1 {
		t.Fatalf("expected 1 skip finding for the whole package, got %d in %v", n, ruleIDs(got))
	}
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeTemp(path, body); err != nil {
			t.Fatal(err)
		}
	}
}
