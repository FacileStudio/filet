package filet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func module(t *testing.T, root string, name string, files ...string) {
	t.Helper()
	dir := filepath.Join(root, "apps", "api", "modules", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := writeTemp(filepath.Join(dir, f), "package "+name+"\n"); err != nil {
			t.Fatal(err)
		}
	}
}

func archFindings(t *testing.T, cfg *Config, root string) []Finding {
	t.Helper()
	files, err := Scan(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	return CheckArchitecture(cfg, files)
}

func TestRequiredFilesPerDirectoryGlob(t *testing.T) {
	root := t.TempDir()
	module(t, root, "auth", "router.go", "service.go")
	module(t, root, "billing", "service.go")
	module(t, root, "docs", "generate.go")

	cfg := DefaultConfig()
	cfg.root = root
	cfg.Architecture.RequiredFiles = map[string][]string{
		"apps/api/modules/*":    {"router.go"},
		"apps/api/modules/docs": {},
	}

	got := archFindings(t, cfg, root)
	if n := countRule(got, "arch.file.missing"); n != 1 {
		t.Fatalf("only billing lacks a router, got %d in %v", n, ruleIDs(got))
	}
	for _, f := range got {
		if f.Rule != "arch.file.missing" {
			continue
		}
		if !strings.Contains(f.File, "billing") {
			t.Fatalf("expected the finding on billing, got %q", f.File)
		}
		if f.Severity != Error {
			t.Fatal("a declared architecture contract is an error, not a warning")
		}
	}
}

func TestPathsFollowTheConfigRootNotTheWorkingDirectory(t *testing.T) {
	base := t.TempDir()
	proj := filepath.Join(base, "myproj")
	module(t, proj, "billing", "service.go")
	cfg := "extensions: [.go]\narchitecture:\n  requiredFiles:\n    \"apps/api/modules/*\": [router.go]\n"
	if err := writeTemp(filepath.Join(proj, ".filet.yml"), cfg); err != nil {
		t.Fatal(err)
	}

	t.Chdir(base)
	loaded, _, err := LoadConfig("myproj")
	if err != nil {
		t.Fatal(err)
	}
	files, err := Scan(loaded, "myproj")
	if err != nil {
		t.Fatal(err)
	}

	assertPaths(t, files[0])

	got := CheckArchitecture(loaded, files)
	if n := countRule(got, "arch.file.missing"); n != 1 {
		t.Fatalf("the glob must still match from a parent directory, got %d in %v", n, ruleIDs(got))
	}
	if !strings.HasPrefix(got[0].File, "myproj/") {
		t.Fatalf("the reported path must be followable from here, got %q", got[0].File)
	}
}

func assertPaths(t *testing.T, f SourceFile) {
	t.Helper()
	if f.Rel != "apps/api/modules/billing/service.go" {
		t.Fatalf("Rel must be relative to the config root, got %q", f.Rel)
	}
	if f.Display != "myproj/apps/api/modules/billing/service.go" {
		t.Fatalf("Display must be relative to the working directory, got %q", f.Display)
	}
}
