package filet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ratchetConfig(t *testing.T) *Config {
	t.Helper()
	cfg := DefaultConfig()
	cfg.root = t.TempDir()
	return cfg
}

func findingsAt(cfg *Config, rule string, paths ...string) Report {
	var r Report
	for i, p := range paths {
		r.Findings = append(r.Findings, newFinding(rule, filepath.Join(cfg.root, p), i+1, Warn, "boom"))
	}
	return r
}

func ratchetRun(t *testing.T, cfg *Config, mode string, r Report) (Report, Report) {
	t.Helper()
	shown, gate, err := Ratchet(cfg, cfg.root, mode, r)
	if err != nil {
		t.Fatal(err)
	}
	return shown, gate
}

func readBaselineText(t *testing.T, cfg *Config) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(cfg.root, BaselineName))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestBaselineFileFormatIsGolden(t *testing.T) {
	cfg := ratchetConfig(t)
	ratchetRun(t, cfg, "write", findingsAt(cfg, "go.func.long", "b.go", "a/x.go", "b.go"))
	want := "{\n  \"version\": 1,\n  \"counts\": {\n    \"go.func.long\": {\n" +
		"      \"a/x.go\": 1,\n      \"b.go\": 2\n    }\n  }\n}\n"
	if got := readBaselineText(t, cfg); got != want {
		t.Fatalf("baseline file:\n%s\nwant:\n%s", got, want)
	}
}

func TestBaselineExcusesKnownFindings(t *testing.T) {
	cfg := ratchetConfig(t)
	r := findingsAt(cfg, "go.func.long", "a.go", "a.go")
	ratchetRun(t, cfg, "write", r)
	shown, gate := ratchetRun(t, cfg, "", findingsAt(cfg, "go.func.long", "a.go", "a.go"))
	if len(gate.Findings) != 0 {
		t.Fatalf("baselined findings must not gate: %+v", gate.Findings)
	}
	for _, f := range shown.Findings {
		if f.Severity != Info || !strings.HasSuffix(f.Message, "(baseline)") {
			t.Fatalf("baselined finding must render as info (baseline): %+v", f)
		}
	}
}

func TestBaselineFailsOnGrowthAndNewPaths(t *testing.T) {
	cfg := ratchetConfig(t)
	ratchetRun(t, cfg, "write", findingsAt(cfg, "go.func.long", "a.go"))
	_, gate := ratchetRun(t, cfg, "", findingsAt(cfg, "go.func.long", "a.go", "a.go"))
	if len(gate.Findings) != 2 {
		t.Fatalf("a grown count must gate every finding of that path, got %d", len(gate.Findings))
	}
	_, gate = ratchetRun(t, cfg, "", findingsAt(cfg, "go.func.long", "a.go", "new.go"))
	if ids := ruleIDs(gate.Findings); len(ids) != 1 || gate.Findings[0].File != filepath.Join(cfg.root, "new.go") {
		t.Fatalf("a new path must gate on its own, got %+v", gate.Findings)
	}
}

func TestBaselineDropIsStaleUntilUpdated(t *testing.T) {
	cfg := ratchetConfig(t)
	ratchetRun(t, cfg, "write", findingsAt(cfg, "go.func.long", "a.go", "a.go"))
	_, gate := ratchetRun(t, cfg, "", findingsAt(cfg, "go.func.long", "a.go"))
	if countRule(gate.Findings, staleRule) != 1 || !FailLevel(cfg, gate) {
		t.Fatalf("a dropped count must fail as %s, got %+v", staleRule, gate.Findings)
	}
	_, gate = ratchetRun(t, cfg, "update", findingsAt(cfg, "go.func.long", "a.go"))
	if len(gate.Findings) != 0 {
		t.Fatalf("update must clear the stale entry, got %+v", gate.Findings)
	}
	if !strings.Contains(readBaselineText(t, cfg), "\"a.go\": 1") {
		t.Fatalf("update must lower the count:\n%s", readBaselineText(t, cfg))
	}
}

func TestBaselineUpdateNeverRaises(t *testing.T) {
	cfg := ratchetConfig(t)
	ratchetRun(t, cfg, "write", findingsAt(cfg, "go.func.long", "a.go"))
	_, gate := ratchetRun(t, cfg, "update", findingsAt(cfg, "go.func.long", "a.go", "a.go", "b.go"))
	if len(gate.Findings) != 3 {
		t.Fatalf("growth must still gate after update, got %+v", gate.Findings)
	}
	if text := readBaselineText(t, cfg); !strings.Contains(text, "\"a.go\": 1") || strings.Contains(text, "b.go") {
		t.Fatalf("update raised or added an entry:\n%s", text)
	}
}

func TestBaselineRenameIsOneStaleAndOneNew(t *testing.T) {
	cfg := ratchetConfig(t)
	ratchetRun(t, cfg, "write", findingsAt(cfg, "go.func.long", "old.go"))
	_, gate := ratchetRun(t, cfg, "", findingsAt(cfg, "go.func.long", "renamed.go"))
	if countRule(gate.Findings, staleRule) != 1 || countRule(gate.Findings, "go.func.long") != 1 {
		t.Fatalf("a rename must be one stale entry plus one new one, got %+v", gate.Findings)
	}
}

func TestBaselineScopedRunKeepsTheRest(t *testing.T) {
	cfg := ratchetConfig(t)
	ratchetRun(t, cfg, "write", findingsAt(cfg, "go.func.long", "api/a.go", "cli/b.go"))
	sub := filepath.Join(cfg.root, "api")
	_, gate, err := Ratchet(cfg, sub, "write", findingsAt(cfg, "go.func.long", "api/a.go"))
	if err != nil || len(gate.Findings) != 0 {
		t.Fatalf("scoped write: err %v, gate %+v", err, gate.Findings)
	}
	if !strings.Contains(readBaselineText(t, cfg), "cli/b.go") {
		t.Fatalf("a scoped write erased an entry outside its scope:\n%s", readBaselineText(t, cfg))
	}
}

func TestBaselineRefusesBadModesAndMissingFiles(t *testing.T) {
	cfg := ratchetConfig(t)
	if _, _, err := Ratchet(cfg, cfg.root, "rewrite", Report{}); err == nil {
		t.Fatal("an unknown mode must be an error")
	}
	if _, _, err := Ratchet(cfg, cfg.root, "update", Report{}); err == nil {
		t.Fatal("update without a baseline must be an error")
	}
	cfg.raw = []byte("baseline: ratchet.json\n")
	if _, _, err := Ratchet(cfg, cfg.root, "", Report{}); err == nil {
		t.Fatal("a baseline named in filet.yml but absent must be an error")
	}
	if _, gate := ratchetRun(t, ratchetConfig(t), "", findingsAt(cfg, "go.func.long", "a.go")); len(gate.Findings) != 1 {
		t.Fatal("no baseline file means no ratchet: every finding gates")
	}
}
