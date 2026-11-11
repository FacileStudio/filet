package filet

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGateCheckStepHonoursTheBaseline(t *testing.T) {
	cfg := ratchetConfig(t)
	cfg.FailOn = "warn"
	body := "package a\n\nfunc Long() int {\n\tx := 0\n" + strings.Repeat("\tx++\n", 40) + "\treturn x\n}\n"
	if err := os.WriteFile(filepath.Join(cfg.root, "a.go"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	step := checkStep(cfg, cfg.root)
	if status, _ := step.Run(io.Discard); status != StepFailed {
		t.Fatalf("without a baseline the long function must fail the step, got %v", status)
	}
	report, err := Analyze(cfg, cfg.root)
	if err != nil {
		t.Fatal(err)
	}
	ratchetRun(t, cfg, "write", report)
	if status, summary := step.Run(io.Discard); status != StepPassed {
		t.Fatalf("a baselined finding must not fail the gate step, got %v (%s)", status, summary)
	}
}
