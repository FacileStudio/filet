package filet

import (
	"path/filepath"
	"testing"
)

func writeScopeTree(t *testing.T, root string) {
	t.Helper()
	for _, f := range []string{"main.go", "README.md", "notes.txt"} {
		if err := writeTemp(filepath.Join(root, f), "package main\n"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExplicitPathRespectsExtensions(t *testing.T) {
	root := t.TempDir()
	writeScopeTree(t, root)

	cfg := DefaultConfig()
	cfg.root = root
	cfg.Extensions = []string{".go"}

	for _, name := range []string{"README.md", "notes.txt"} {
		got, err := Scan(cfg, filepath.Join(root, name))
		if err != nil {
			t.Fatalf("Scan(%s): %v", name, err)
		}
		if len(got) != 0 {
			t.Errorf("an explicit %s is outside extensions, want it unscanned, got %d files", name, len(got))
		}
	}

	got, err := Scan(cfg, filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatalf("Scan(main.go): %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("an explicit .go path must still be scanned, got %d files", len(got))
	}
	if base := filepath.Base(got[0].Path); base != "main.go" {
		t.Errorf("scanned %s, want main.go", base)
	}
}

func TestExplicitPathMatchesTheDirectorySweep(t *testing.T) {
	root := t.TempDir()
	writeScopeTree(t, root)

	cfg := DefaultConfig()
	cfg.root = root
	cfg.Extensions = []string{".go"}

	sweep, err := Scan(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"main.go"} {
		single, err := Scan(cfg, filepath.Join(root, want))
		if err != nil {
			t.Fatal(err)
		}
		if len(single) != 1 {
			t.Errorf("a file the sweep keeps must survive an explicit path: %s", want)
		}
	}
	if len(sweep) != 1 {
		t.Errorf("the sweep must keep exactly the .go file, got %d files", len(sweep))
	}
}

func TestScanStillFailsOnAMissingPath(t *testing.T) {
	root := t.TempDir()
	cfg := DefaultConfig()
	cfg.root = root

	if _, err := Scan(cfg, filepath.Join(root, "nope.go")); err == nil {
		t.Error("a path that does not exist must still be an error, not an empty clean result")
	}
}
