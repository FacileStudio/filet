package filet

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fakeTools(t *testing.T, exits map[string]int) string {
	t.Helper()
	bin, proj := t.TempDir(), t.TempDir()
	t.Setenv("PATH", bin)
	t.Setenv("GATE_LOG", filepath.Join(bin, "log"))
	for tool, code := range exits {
		script := "#!/bin/sh\necho \"" + tool + " $*\" >> \"$GATE_LOG\"\nexit " + strconv.Itoa(code) + "\n"
		if err := os.WriteFile(filepath.Join(bin, tool), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(proj, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTemp(filepath.Join(proj, "go.mod"), "module p\n\ngo 1.26\n"); err != nil {
		t.Fatal(err)
	}
	return proj
}

func runFakeGate(t *testing.T, proj string, opts GateOptions) ([]StepResult, []string) {
	t.Helper()
	cfg, _, err := LoadConfig(proj)
	if err != nil {
		t.Fatal(err)
	}
	steps, err := GateSteps(cfg, proj)
	if err != nil {
		t.Fatal(err)
	}
	results := RunGate(steps, opts, io.Discard)
	raw, _ := os.ReadFile(os.Getenv("GATE_LOG"))
	return results, strings.Split(strings.TrimSpace(string(raw)), "\n")
}

func statuses(results []StepResult) string {
	parts := make([]string, 0, len(results))
	for _, r := range results {
		parts = append(parts, r.Step+"="+string(r.Status))
	}
	return strings.Join(parts, " ")
}

func allTools(override map[string]int) map[string]int {
	exits := map[string]int{"gofmt": 0, "go": 0, "staticcheck": 0, "douane": 0}
	maps.Copy(exits, override)
	return exits
}

func TestGateRunsStepsInOrder(t *testing.T) {
	results, log := runFakeGate(t, fakeTools(t, allTools(nil)), GateOptions{})
	want := "format:gofmt=passed check=passed analyze:vet=passed analyze:staticcheck=passed test:go=passed douane=passed"
	if got := statuses(results); got != want {
		t.Fatalf("statuses:\n got %s\nwant %s", got, want)
	}
	calls := strings.Join(log, "|")
	if calls != "gofmt -l .|go vet ./...|staticcheck ./...|go test -race ./...|douane scan -fail critical -scope prod ." {
		t.Fatalf("tool calls = %s", calls)
	}
	if AnyFailed(results) {
		t.Fatal("an all-green gate reported a failure")
	}
}

func TestGateStopsAtTheFirstFailure(t *testing.T) {
	results, log := runFakeGate(t, fakeTools(t, allTools(map[string]int{"staticcheck": 1})), GateOptions{})
	want := "format:gofmt=passed check=passed analyze:vet=passed analyze:staticcheck=failed test:go=not-run douane=not-run"
	if got := statuses(results); got != want {
		t.Fatalf("statuses:\n got %s\nwant %s", got, want)
	}
	if strings.Contains(strings.Join(log, "|"), "douane") {
		t.Fatalf("douane ran after a failed step: %v", log)
	}
}

func TestGateKeepGoingRunsEveryStep(t *testing.T) {
	results, log := runFakeGate(t, fakeTools(t, allTools(map[string]int{"staticcheck": 1})), GateOptions{KeepGoing: true})
	if !strings.HasSuffix(statuses(results), "test:go=passed douane=passed") || len(log) != 5 {
		t.Fatalf("keep-going: %s / %v", statuses(results), log)
	}
	if !AnyFailed(results) {
		t.Fatal("keep-going must still fail the gate")
	}
}

func TestGateMissingToolIsSkippedUnlessRequired(t *testing.T) {
	exits := allTools(nil)
	delete(exits, "staticcheck")
	proj := fakeTools(t, exits)
	results, _ := runFakeGate(t, proj, GateOptions{})
	if r := results[3]; r.Status != StepSkipped || r.Detail != "staticcheck not found" {
		t.Fatalf("missing staticcheck: %+v", r)
	}
	if AnyFailed(results) {
		t.Fatal("a skipped row failed the gate")
	}
	results, _ = runFakeGate(t, proj, GateOptions{Config: GateConfig{RequireTools: []string{"staticcheck"}}})
	if r := results[3]; r.Status != StepFailed {
		t.Fatalf("required staticcheck: %+v", r)
	}
}

func TestGateDisableSkipsByPhaseOrName(t *testing.T) {
	cfg := GateConfig{Disable: []string{"analyze", "douane"}}
	results, log := runFakeGate(t, fakeTools(t, allTools(map[string]int{"staticcheck": 1})), GateOptions{Config: cfg})
	want := "format:gofmt=passed check=passed analyze:vet=skipped analyze:staticcheck=skipped test:go=passed douane=skipped"
	if got := statuses(results); got != want || len(log) != 2 {
		t.Fatalf("statuses %s, calls %v", got, log)
	}
}

func TestGateJSONShape(t *testing.T) {
	results, _ := runFakeGate(t, fakeTools(t, allTools(map[string]int{"douane": 1})), GateOptions{})
	var buf bytes.Buffer
	if err := WriteJSONValue(&buf, Summarize(results)); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Passed *bool `json:"passed"`
		Steps  []map[string]any
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Passed == nil || *doc.Passed || len(doc.Steps) != len(results) {
		t.Fatalf("summary: %s", buf.String())
	}
	for _, key := range []string{"step", "status", "durationMs"} {
		if _, ok := doc.Steps[0][key]; !ok {
			t.Fatalf("step row lacks %q: %v", key, doc.Steps[0])
		}
	}
}

func TestGateSettingsReadsTheGateBlock(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "failOn: info\ngate:\n  disable: [douane]\n  requireTools: [staticcheck]\n"
	if err := writeTemp(filepath.Join(dir, "filet.yml"), body); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	gc, err := cfg.GateSettings()
	if err != nil || strings.Join(gc.Disable, ",") != "douane" || strings.Join(gc.RequireTools, ",") != "staticcheck" {
		t.Fatalf("gate block = %+v, %v", gc, err)
	}
}
