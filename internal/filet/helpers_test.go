package filet

import (
	"os"
	"path/filepath"
	"testing"
)

func ruleIDs(f []Finding) []string {
	out := make([]string, 0, len(f))
	for _, x := range f {
		out = append(out, x.Rule)
	}
	return out
}

func countRule(f []Finding, rule string) int {
	n := 0
	for _, x := range f {
		if x.Rule == rule {
			n++
		}
	}
	return n
}

func ruleFindings(f []Finding, rule string) []Finding {
	var out []Finding
	for _, x := range f {
		if x.Rule == rule {
			out = append(out, x)
		}
	}
	return out
}

func writeTemp(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}

func source(t *testing.T, name, body string) SourceFile {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := writeTemp(path, body); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.root = dir
	f, err := readSource(cfg, path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
