package filet

import (
	"os"
	"path/filepath"
	"testing"
)

func repo(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestConfigOutsideTheRepositoryIsIgnored(t *testing.T) {
	outer := t.TempDir()
	app := repo(t, filepath.Join(outer, "app"))
	if err := writeTemp(filepath.Join(outer, ".filet.yml"), "limits:\n  funcLines: 7\n"); err != nil {
		t.Fatal(err)
	}

	cfg, path, err := LoadConfig(app)
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Fatalf("a config above the repository must not apply, got %q", path)
	}
	if cfg.Limits.FuncLines != DefaultConfig().Limits.FuncLines {
		t.Fatal("it leaked into the limits, so a local run and CI would disagree")
	}
	if cfg.Root() != app {
		t.Fatalf("root should fall back to the target, got %q", cfg.Root())
	}
}

func TestConfigAtTheRepositoryRootIsFoundFromWithin(t *testing.T) {
	app := repo(t, t.TempDir())
	want := filepath.Join(app, ".filet.yml")
	if err := writeTemp(want, "limits:\n  funcLines: 9\n"); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(app, "apps", "api", "modules", "auth")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, path, err := LoadConfig(deep)
	if err != nil {
		t.Fatal(err)
	}
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
	if cfg.Limits.FuncLines != 9 || cfg.Root() != app {
		t.Fatalf("root=%q funcLines=%d", cfg.Root(), cfg.Limits.FuncLines)
	}
}
