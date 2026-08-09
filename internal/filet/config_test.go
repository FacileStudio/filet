package filet

import (
	"path/filepath"
	"testing"
)

func TestPresetIsOverriddenByExplicitKeys(t *testing.T) {
	dir := t.TempDir()
	body := "preset: epitech\nlimits:\n  lineLength: 100\n"
	if err := writeTemp(filepath.Join(dir, ".filet.yml"), body); err != nil {
		t.Fatal(err)
	}

	cfg, path, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != ".filet.yml" {
		t.Fatalf("expected .filet.yml, got %q", path)
	}
	if cfg.Limits.FuncLines != 20 || cfg.Limits.Params != 4 || cfg.Limits.Nesting != 3 {
		t.Fatalf("preset not applied: %+v", cfg.Limits)
	}
	if cfg.Limits.LineLength != 100 {
		t.Fatalf("explicit key must win over the preset, got %d", cfg.Limits.LineLength)
	}
	if cfg.Limits.StructFields != DefaultConfig().Limits.StructFields {
		t.Fatal("keys touched by neither preset nor file must keep the default")
	}
	if cfg.Root() != dir {
		t.Fatalf("root must be the config's directory, got %q", cfg.Root())
	}
}

func TestUnknownPresetIsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := writeTemp(filepath.Join(dir, ".filet.yml"), "preset: fortran77\n"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadConfig(dir); err == nil {
		t.Fatal("expected an error for an unknown preset")
	}
}

func TestScaffoldRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".filet.yml")
	if err := Scaffold(path, "epitech"); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("scaffolded config does not parse: %v", err)
	}
	if cfg.Limits.FuncLines != 20 || cfg.Limits.LineLength != 80 {
		t.Fatalf("scaffolded epitech config lost its limits: %+v", cfg.Limits)
	}
}

func TestUndottedConfigWinsAndTheDottedOneStillLoads(t *testing.T) {
	dir := t.TempDir()
	if err := writeTemp(filepath.Join(dir, ".filet.yml"), "limits:\n  funcLines: 7\n"); err != nil {
		t.Fatal(err)
	}

	cfg, path, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Limits.FuncLines != 7 {
		t.Errorf("a repository still on the dotted name must keep working, got funcLines %d", cfg.Limits.FuncLines)
	}

	if err := writeTemp(filepath.Join(dir, "filet.yml"), "limits:\n  funcLines: 9\n"); err != nil {
		t.Fatal(err)
	}
	cfg, path, err = LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if filepath.Base(path) != "filet.yml" || cfg.Limits.FuncLines != 9 {
		t.Errorf("filet.yml must win over .filet.yml, got %q with funcLines %d", path, cfg.Limits.FuncLines)
	}
}

func TestInitWritesTheUndottedName(t *testing.T) {
	if got := ConfigNames()[0]; got != "filet.yml" {
		t.Errorf("filet init writes ConfigNames()[0] = %q, want filet.yml", got)
	}
}
