package filet

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gofmtFixture(t *testing.T, files map[string]string) (StepStatus, string) {
	t.Helper()
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skip("gofmt not on PATH")
	}
	proj := t.TempDir()
	files["go.mod"] = "module p\n\ngo 1.26\n"
	files["main.go"] = "package main\n\nfunc main() {}\n"
	for name, body := range files {
		path := filepath.Join(proj, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeTemp(path, body); err != nil {
			t.Fatal(err)
		}
	}
	cfg, _, err := LoadConfig(proj)
	if err != nil {
		t.Fatal(err)
	}
	u := Unit{Path: ".", Dir: proj}
	return gofmtStep(cfg, u.suite("gofmt", "gofmt", "-l", ".")).Run(io.Discard)
}

const badlyFormatted = "package x\nfunc   F( ) {  }\n"

func TestGofmtStepSkipsVendor(t *testing.T) {
	status, detail := gofmtFixture(t, map[string]string{"vendor/example.com/x/x.go": badlyFormatted})
	if status != StepPassed {
		t.Fatalf("vendor file failed the step: %s %s", status, detail)
	}
}

func TestGofmtStepFailsOutsideVendor(t *testing.T) {
	status, detail := gofmtFixture(t, map[string]string{
		"vendor/example.com/x/x.go": badlyFormatted,
		"internal/x/x.go":           badlyFormatted,
	})
	if status != StepFailed || detail != "1 file needs gofmt" {
		t.Fatalf("got %s %q, want failed with one file", status, detail)
	}
}
