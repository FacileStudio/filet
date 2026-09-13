package filet

import (
	"maps"
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

func TestMaxFilesPerDirFlagsDirectoriesOverTheLimit(t *testing.T) {
	root := t.TempDir()
	module(t, root, "auth", "router.go", "service.go", "store.go")
	module(t, root, "billing", "service.go")
	for _, name := range []string{"main.go", "cli.go", "server.go"} {
		if err := writeTemp(filepath.Join(root, name), "package main\n"); err != nil {
			t.Fatal(err)
		}
	}
	cfg := DefaultConfig()
	cfg.root = root
	cfg.Architecture.MaxFilesPerDir = DirFileLimits{General: 2}

	got := ruleFindings(archFindings(t, cfg, root), "arch.dir.files")
	if len(got) != 2 {
		t.Fatalf("the root and auth exceed the limit, got %d findings in %v", len(got), ruleIDs(got))
	}
	sawRoot := false
	for _, f := range got {
		sawRoot = sawRoot || f.File == DisplayPath(root)
		if f.Severity != Warn {
			t.Fatal("a directory over its file budget is a warning, not an error")
		}
		if f.Message != "directory holds 3 files (limit 2)" {
			t.Fatalf("unexpected message %q", f.Message)
		}
	}
	if !sawRoot {
		t.Fatal("the root directory counts too")
	}
}

func TestMaxFilesPerDirGlobOverridesGeneral(t *testing.T) {
	root := t.TempDir()
	module(t, root, "auth", "router.go", "service.go", "store.go")
	module(t, root, "billing", "router.go", "service.go", "store.go")

	cfg := DefaultConfig()
	cfg.root = root
	cfg.Architecture.MaxFilesPerDir = DirFileLimits{
		General: 1,
		Globs: map[string]int{
			"apps/api/modules/billing": 5,
			"apps/api/modules/*":       2,
		},
	}

	got := ruleFindings(archFindings(t, cfg, root), "arch.dir.files")
	if len(got) != 1 {
		t.Fatalf("only auth is over its glob limit, got %d in %v", len(got), ruleIDs(got))
	}
	if !strings.Contains(got[0].File, "auth") {
		t.Fatalf("billing is covered by the exact glob and must stay quiet, got %q", got[0].File)
	}
	if got[0].Message != "directory holds 3 files (limit 2)" {
		t.Fatalf("the wildcard limit applies, got %q", got[0].Message)
	}
}

func TestMaxFilesPerDirOffByDefault(t *testing.T) {
	root := t.TempDir()
	module(t, root, "auth", "router.go", "service.go", "store.go")

	cfg := DefaultConfig()
	cfg.root = root
	if got := archFindings(t, cfg, root); countRule(got, "arch.dir.files") != 0 {
		t.Fatal("unset maxFilesPerDir must stay silent")
	}

	cfg.Architecture.MaxFilesPerDir = DirFileLimits{General: 1}
	cfg.Disabled = []string{"arch.dir.files"}
	if got := archFindings(t, cfg, root); countRule(got, "arch.dir.files") != 0 {
		t.Fatal("disabled: [arch.dir.files] must silence the rule")
	}

	cfg.Disabled = nil
	cfg.Architecture.MaxFilesPerDir = DirFileLimits{General: 1, Globs: map[string]int{"apps/api/modules/auth": 0}}
	if got := archFindings(t, cfg, root); countRule(got, "arch.dir.files") != 0 {
		t.Fatal("a zero glob must opt the directory out, not fall back to the general limit")
	}
}

func TestMaxFilesPerDirAcceptsScalarAndMap(t *testing.T) {
	dir := t.TempDir()
	scalar := "architecture:\n  maxFilesPerDir: 12\n"
	if err := writeTemp(filepath.Join(dir, ".filet.yml"), scalar); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Architecture.MaxFilesPerDir.General != 12 || len(cfg.Architecture.MaxFilesPerDir.Globs) != 0 {
		t.Fatalf("the scalar form must set the general limit, got %+v", cfg.Architecture.MaxFilesPerDir)
	}

	mapped := "architecture:\n  maxFilesPerDir:\n    \"apps/*\": 40\n    \".\": 10\n"
	if err := writeTemp(filepath.Join(dir, ".filet.yml"), mapped); err != nil {
		t.Fatal(err)
	}
	cfg, _, err = LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"apps/*": 40, ".": 10}
	if cfg.Architecture.MaxFilesPerDir.General != 0 || !maps.Equal(cfg.Architecture.MaxFilesPerDir.Globs, want) {
		t.Fatalf("the map form must set per-glob limits, got %+v", cfg.Architecture.MaxFilesPerDir)
	}
}
