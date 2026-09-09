package filet

import (
	"path/filepath"
	"testing"
)

// goSource writes body into a file named name in dir and returns it as a
// SourceFile. Unlike source(), several calls share the same directory so they
// land in one package and can reference each other.
func goSource(t *testing.T, cfg *Config, dir, name, body string) SourceFile {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := writeTemp(path, body); err != nil {
		t.Fatal(err)
	}
	f, err := readSource(cfg, path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestResourceLeakDetectedAcrossPackageFiles proves the leak rule resolves a
// closer type defined in a sibling file, and still runs when another file of
// the package pulls in an import filet cannot resolve (a third-party module).
// Both failures used to kill the whole type-check and silently no-op the rule.
func TestResourceLeakDetectedAcrossPackageFiles(t *testing.T) {
	cfg := DefaultConfig()
	cfg.root = t.TempDir()
	dir := cfg.root
	files := []SourceFile{
		goSource(t, cfg, dir, "log.go", `package f

import "os"

// Log wraps a file and closes it.
type Log struct{ f *os.File }

// NewLog opens a log.
func NewLog() *Log { return &Log{} }

// Close releases the file.
func (l *Log) Close() error { return l.f.Close() }
`),
		goSource(t, cfg, dir, "use.go", `package f

func worker() {
	l := NewLog()
	_ = l
}
`),
		goSource(t, cfg, dir, "vendor.go", `package f

import _ "github.com/FacileStudio/does-not-resolve"
`),
	}

	got := checkGoFiles(cfg, files)
	if n := countRule(got, "go.leak.resource"); n != 1 {
		t.Fatalf("expected 1 go.leak.resource finding, got %d in %v", n, ruleIDs(got))
	}
}

// TestResourceLeakNotFlaggedWhenPackageCloses proves cross-package detection
// honours a deferred Close in the sibling-typed-closer case.
func TestResourceLeakNotFlaggedWhenPackageCloses(t *testing.T) {
	cfg := DefaultConfig()
	cfg.root = t.TempDir()
	dir := cfg.root
	files := []SourceFile{
		goSource(t, cfg, dir, "log.go", `package f

// Log is a closable handle.
type Log struct{}

// NewLog makes a Log.
func NewLog() *Log { return &Log{} }

// Close releases the handle.
func (l *Log) Close() error { return nil }
`),
		goSource(t, cfg, dir, "use.go", `package f

func worker() {
	l := NewLog()
	defer l.Close()
	_ = l
}
`),
	}
	got := checkGoFiles(cfg, files)
	if n := countRule(got, "go.leak.resource"); n != 0 {
		t.Fatalf("expected 0 go.leak.resource findings, got %d in %v", n, ruleIDs(got))
	}
}
