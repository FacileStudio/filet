package filet

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDockerfileRules(t *testing.T) {
	cfg := DefaultConfig()
	cfg.root = t.TempDir()
	path := filepath.Join(cfg.root, "Dockerfile")

	src := `FROM golang:latest
ENV API_TOKEN=hunter2
COPY . .
RUN apt-get update && apt-get install -y curl
RUN go build -o app .
CMD ./app
`
	got := ruleIDs(CheckDockerfile(cfg, path, []byte(src)))
	for _, want := range []string{
		"docker.from.latest", "docker.secret", "docker.apt.recommends", "docker.apt.cleanup",
		"docker.copy.cachebust", "docker.exec.form", "docker.user.root",
		"docker.multistage.missing", "docker.dockerignore.missing",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("missing rule %s in %v", want, got)
		}
	}
}

func TestDockerfileJoinsContinuations(t *testing.T) {
	insts := ParseDockerfile([]byte("RUN apt-get update \\\n  && apt-get install -y curl\nUSER app\n"))
	if len(insts) != 2 {
		t.Fatalf("expected 2 instructions, got %d: %+v", len(insts), insts)
	}
	if !strings.Contains(insts[0].Args, "install -y curl") {
		t.Fatalf("continuation not joined: %q", insts[0].Args)
	}
	if insts[1].Line != 3 {
		t.Fatalf("expected USER on line 3, got %d", insts[1].Line)
	}
}

func TestRoastIsDeterministic(t *testing.T) {
	f := []Finding{{Rule: "go.panic", File: "a.go", Line: 12}}
	g := []Finding{{Rule: "go.panic", File: "a.go", Line: 12}}
	if Roast(f)[0].Roast != Roast(g)[0].Roast {
		t.Fatal("same finding must produce the same punchline")
	}
	if Roast(f)[0].Roast == "" {
		t.Fatal("expected a punchline")
	}
}

func TestQuietKeepsTheSummaryAndTheVerdict(t *testing.T) {
	r := Report{
		Findings: []Finding{newFinding("docker.secret", "Dockerfile", 2, Error, "boom")},
		Files:    1,
		Lines:    6,
	}
	var buf bytes.Buffer
	WriteText(&buf, r, TextOptions{Quiet: true, Roast: true})

	out := buf.String()
	if strings.Contains(out, "nothing to complain about") {
		t.Fatalf("quiet must never report a clean tree when findings exist:\n%s", out)
	}
	if !strings.Contains(out, "1 error") {
		t.Fatalf("quiet must still print the summary counts:\n%s", out)
	}
	if strings.Contains(out, "boom") {
		t.Fatalf("quiet must not print individual findings:\n%s", out)
	}

	cfg := DefaultConfig()
	if !FailLevel(cfg, r) {
		t.Fatal("an error-severity finding must fail the gate whatever the output mode")
	}
}

func TestFailOnValidation(t *testing.T) {
	for _, ok := range []string{"info", "warn", "warning", "error", "never", "ERROR"} {
		if !ValidFailOn(ok) {
			t.Errorf("%q should be a valid failOn", ok)
		}
	}
	for _, bad := range []string{"banana", "nevr", "", "warnings"} {
		if ValidFailOn(bad) {
			t.Errorf("%q should not be a valid failOn", bad)
		}
	}

	dir := t.TempDir()
	if err := writeTemp(filepath.Join(dir, ".filet.yml"), "failOn: banana\n"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadConfig(dir); err == nil {
		t.Fatal("a bogus failOn must be rejected, not silently gated on error")
	}
}
