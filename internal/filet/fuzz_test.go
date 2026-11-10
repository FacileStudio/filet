package filet

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fuzzFixture(t *testing.T) (*Config, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "fuzzmod")
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("..", "..", "testdata", "fuzzmod"))); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTemp(filepath.Join(dir, "go.mod"), "module fuzzmod\n\ngo 1.26\n"); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, dir
}

func TestDiscoverFuzzTargetsFindsOnlyTestingFTargets(t *testing.T) {
	cfg, dir := fuzzFixture(t)
	rust := filepath.Join(dir, "crate", "fuzz", "fuzz_targets")
	if err := os.MkdirAll(rust, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeTemp(filepath.Join(rust, "parse.rs"), "fn main() {}\n"); err != nil {
		t.Fatal(err)
	}
	targets, err := DiscoverFuzzTargets(cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tg := range targets {
		got = append(got, tg.Lang+":"+tg.Name)
	}
	if strings.Join(got, ",") != "go:FuzzCrashes,go:FuzzPasses,rust:parse" {
		t.Fatalf("targets = %v", got)
	}
	if targets[2].Dir != filepath.Join(dir, "crate") {
		t.Fatalf("rust target runs from %s, want the crate root", targets[2].Dir)
	}
}

func TestFuzzSeedsPassAndSmokeReportsTheCrashInput(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go test -fuzz")
	}
	cfg, dir := fuzzFixture(t)
	targets, err := DiscoverFuzzTargets(cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range RunFuzz(targets, FuzzOptions{Mode: FuzzSeeds}, io.Discard) {
		if r.Status != StepPassed {
			t.Fatalf("seed replay of %s: %s %s", r.Step, r.Status, r.Detail)
		}
	}
	smoke := RunFuzz(targets, FuzzOptions{Mode: FuzzSmoke, FuzzTime: 5 * time.Second}, io.Discard)
	crash := smoke[0]
	if crash.Status != StepFailed || !strings.Contains(crash.Detail, filepath.Join(dir, "testdata", "fuzz", "FuzzCrashes")) {
		t.Fatalf("crashing target: %s %q", crash.Status, crash.Detail)
	}
	if _, err := os.Stat(strings.TrimPrefix(crash.Detail, "failing input: ")); err != nil {
		t.Fatalf("reported crash input does not exist: %v", err)
	}
}

func TestFuzzBudgetStopsStartingTargets(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	targets := []FuzzTarget{{Lang: "go", Name: "FuzzA", Dir: t.TempDir()}, {Lang: "go", Name: "FuzzB", Dir: t.TempDir()}}
	results := RunFuzz(targets, FuzzOptions{Mode: FuzzSmoke, FuzzTime: time.Second, Budget: time.Nanosecond}, io.Discard)
	if results[0].Status != StepPassed || results[1].Status != StepSkipped {
		t.Fatalf("budget: %+v", results)
	}
	if !strings.HasPrefix(results[1].Detail, "budget of 1ns spent") {
		t.Fatalf("skip reason = %q", results[1].Detail)
	}
}

func TestFuzzWithoutGoIsSkippedNeverPassed(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	r := RunFuzz([]FuzzTarget{{Lang: "go", Name: "FuzzA", Dir: t.TempDir()}}, FuzzOptions{Mode: FuzzSeeds}, io.Discard)
	if r[0].Status != StepSkipped || r[0].Detail != "go not found" {
		t.Fatalf("go target: %+v", r[0])
	}
}

func TestRustTargetWithoutCargoFuzzIsSkipped(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	r := RunFuzz([]FuzzTarget{{Lang: "rust", Name: "parse", Dir: t.TempDir()}}, FuzzOptions{Mode: FuzzSeeds}, io.Discard)
	if r[0].Status != StepSkipped || r[0].Detail != "cargo-fuzz not found" {
		t.Fatalf("rust target: %+v", r[0])
	}
}
