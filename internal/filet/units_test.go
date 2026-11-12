package filet

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func unitFixture(t *testing.T, name string, tools ...string) string {
	t.Helper()
	proj, bin := t.TempDir(), t.TempDir()
	if err := os.CopyFS(proj, os.DirFS(filepath.Join("..", "..", "testdata", "units", name))); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(proj, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTemp(filepath.Join(proj, "LICENSE"), "MIT License\n"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("GATE_LOG", filepath.Join(bin, "log"))
	for _, tool := range tools {
		script := "#!/bin/sh\necho \"" + tool + " $* @$(pwd -P)\" >> \"$GATE_LOG\"\n"
		if err := os.WriteFile(filepath.Join(bin, tool), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	real, err := filepath.EvalSymlinks(proj)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func planFixture(t *testing.T, name string, tools ...string) (string, GatePlan) {
	t.Helper()
	proj := unitFixture(t, name, tools...)
	cfg, _, err := LoadConfig(proj)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanGate(cfg, proj)
	if err != nil {
		t.Fatal(err)
	}
	return proj, plan
}

func gateCalls(proj string, log []string) string {
	out := make([]string, 0, len(log))
	for _, line := range log {
		if line == "" || strings.HasPrefix(line, "go list ") {
			continue
		}
		cmd, dir, _ := strings.Cut(line, " @")
		rel, _ := filepath.Rel(proj, dir)
		out = append(out, cmd+" @"+filepath.ToSlash(rel))
	}
	return strings.Join(out, "\n")
}

func gateRows(t *testing.T, name string, tools ...string) ([]StepResult, string) {
	t.Helper()
	proj := unitFixture(t, name, tools...)
	results, log := runFakeGate(t, proj, GateOptions{KeepGoing: true})
	return results, gateCalls(proj, log)
}

func TestGateRootModuleRunsAtTheRoot(t *testing.T) {
	results, calls := gateRows(t, "root", "gofmt", "go", "staticcheck", "douane")
	want := "format:gofmt=passed check=passed analyze:vet=passed analyze:staticcheck=passed test:go=passed douane=passed"
	if got := statuses(results); got != want {
		t.Fatalf("statuses:\n got %s\nwant %s", got, want)
	}
	wantCalls := "gofmt -l . @.\ngo vet ./... @.\nstaticcheck ./... @.\ngo test -race ./... @.\n" +
		"douane scan -fail critical -scope prod . @."
	if calls != wantCalls {
		t.Fatalf("calls:\n%s\nwant\n%s", calls, wantCalls)
	}
}

func TestGateSuiteShapeRunsEachUnitInItsDirectory(t *testing.T) {
	results, calls := gateRows(t, "suite", "gofmt", "go", "staticcheck", "bun", "douane")
	want := "format:gofmt apps/api=passed check=passed analyze:vet apps/api=passed " +
		"analyze:staticcheck apps/api=passed analyze:bun-check apps/client=passed " +
		"test:go apps/api=passed test:bun apps/client=passed douane=passed"
	if got := statuses(results); got != want {
		t.Fatalf("statuses:\n got %s\nwant %s", got, want)
	}
	wantCalls := "gofmt -l . @apps/api\ngo vet ./... @apps/api\nstaticcheck ./... @apps/api\n" +
		"bun run check @apps/client\ngo test -race ./... @apps/api\nbun run test @apps/client\n" +
		"douane scan -fail critical -scope prod . @."
	if calls != wantCalls {
		t.Fatalf("calls:\n%s\nwant\n%s", calls, wantCalls)
	}
}

func TestGateCoversSeveralModulesAndACrate(t *testing.T) {
	results, calls := gateRows(t, "multi", "gofmt", "go", "staticcheck", "cargo", "cargo-fmt", "cargo-clippy", "douane")
	want := "format:cargo-fmt crates/core=passed format:gofmt svc/a=passed format:gofmt svc/b=passed check=passed " +
		"analyze:clippy crates/core=passed analyze:vet svc/a=passed analyze:staticcheck svc/a=passed " +
		"analyze:vet svc/b=passed analyze:staticcheck svc/b=passed test:cargo crates/core=passed " +
		"test:go svc/a=passed test:go svc/b=passed fuzz:go svc/a=passed fuzz:rust crates/core=skipped douane=passed"
	if got := statuses(results); got != want {
		t.Fatalf("statuses:\n got %s\nwant %s", got, want)
	}
	for _, call := range []string{"cargo test @crates/core", "go test -run ^FuzzAnswer$ . @svc/a", "go vet ./... @svc/b"} {
		if !strings.Contains(calls, call) {
			t.Fatalf("no %q in calls:\n%s", call, calls)
		}
	}
	if strings.Contains(calls, "legacy") || strings.Contains(calls, "crates/core/fuzz") {
		t.Fatalf("an ignored directory or a nested crate ran:\n%s", calls)
	}
}

func TestGateFailsAPhaseWithFilesButNoUnit(t *testing.T) {
	results, calls := gateRows(t, "orphan", "gofmt", "go", "staticcheck", "douane")
	want := "format:gofmt=failed check=passed analyze:vet=failed test:go=failed douane=passed"
	if got := statuses(results); got != want {
		t.Fatalf("statuses:\n got %s\nwant %s", got, want)
	}
	if d := results[0].Detail; d != "1 .go files but no go.mod found" {
		t.Fatalf("detail = %q", d)
	}
	if calls != "douane scan -fail critical -scope prod . @." {
		t.Fatalf("calls:\n%s", calls)
	}
}

func TestGateJSONCarriesTheUnitOnEveryRow(t *testing.T) {
	_, plan := planFixture(t, "suite", "gofmt", "go", "staticcheck", "bun", "douane")
	if got := UnitsLine(plan.Units); got != "units: apps/api (go), apps/client (bun)" {
		t.Fatalf("units line = %q", got)
	}
	results := RunGate(plan.Steps, GateOptions{KeepGoing: true}, &bytes.Buffer{})
	var buf bytes.Buffer
	if err := WriteJSONValue(&buf, GateReport{Units: plan.Units, Steps: results}); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Units []Unit
		Steps []map[string]any
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Units) != 2 || doc.Units[1].Path != "apps/client" {
		t.Fatalf("units: %s", buf.String())
	}
	for _, row := range doc.Steps {
		if u, ok := row["unit"].(string); !ok || u == "" {
			t.Fatalf("row without a unit: %v", row)
		}
	}
}

func TestEveryGateStepNameIsAValidConfigName(t *testing.T) {
	for _, name := range []string{"suite", "multi", "orphan"} {
		_, plan := planFixture(t, name)
		for _, s := range plan.Steps {
			gc := GateConfig{Disable: []string{s.Phase, s.Name}}
			if s.Tool != "" {
				gc.RequireTools = []string{s.Tool}
			}
			if err := gc.validate(); err != nil {
				t.Fatalf("%s: step %s: %v", name, s.Label(), err)
			}
		}
	}
}

func TestTestSuitesCoverEveryUnit(t *testing.T) {
	proj, plan := planFixture(t, "multi")
	labels := make([]string, 0, 3)
	for _, s := range AllSuites(plan.Units) {
		labels = append(labels, s.Label()+"@"+unitPath(proj, s.Dir))
	}
	if got := strings.Join(labels, ","); got != "cargo crates/core@crates/core,go svc/a@svc/a,go svc/b@svc/b" {
		t.Fatalf("suites = %s", got)
	}
}
