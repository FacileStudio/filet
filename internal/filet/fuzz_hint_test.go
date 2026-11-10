package filet

import (
	"maps"
	"os"
	"path/filepath"
	"testing"
)

const hintSource = `package p

import "io"

// ParseName reads a name.
func ParseName(s string) string { return s }

// DecodeBlob reads a blob.
func DecodeBlob(b []byte) []byte { return b }

// UnmarshalFrom reads from r.
func UnmarshalFrom(r io.Reader) error { return nil }

// ParseCount takes no raw input.
func ParseCount(n int) int { return n }

// Render is not a parser.
func Render(s string) string { return s }
`

func hintFindings(t *testing.T, config string, extra map[string]string) []Finding {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{"filet.yml": config, "go.mod": "module p\n\ngo 1.26\n", "p.go": hintSource}
	maps.Copy(files, extra)
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := writeTemp(filepath.Join(dir, name), body); err != nil {
			t.Fatal(err)
		}
	}
	cfg, _, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Disabled = append(cfg.Disabled, "go.leak.resource")
	src, err := Scan(cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	return ruleFindings(checkGoFiles(cfg, src), "test.fuzz.missing")
}

func TestFuzzHintFlagsRawInputParsersWhenEnabled(t *testing.T) {
	got := hintFindings(t, "style:\n  hintFuzz: true\n", nil)
	if len(got) != 3 {
		t.Fatalf("want ParseName, DecodeBlob and UnmarshalFrom, got %+v", got)
	}
	if got[0].Severity != Info {
		t.Fatalf("test.fuzz.missing is a hint, got severity %s", got[0].Severity)
	}
}

func TestFuzzHintIsOffByDefault(t *testing.T) {
	if got := hintFindings(t, "failOn: info\n", nil); len(got) != 0 {
		t.Fatalf("hint fired without style.hintFuzz: %+v", got)
	}
}

func TestFuzzHintIsSatisfiedByAFuzzTarget(t *testing.T) {
	fuzz := "package p\n\nimport \"testing\"\n\nfunc FuzzParseName(f *testing.F) {\n\tf.Fuzz(func(t *testing.T, s string) { ParseName(s) })\n}\n"
	if got := hintFindings(t, "style:\n  hintFuzz: true\n", map[string]string{"p_test.go": fuzz}); len(got) != 0 {
		t.Fatalf("package with a Fuzz target still hinted: %+v", got)
	}
}
