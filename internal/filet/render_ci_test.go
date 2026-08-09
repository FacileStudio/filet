package filet

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func ciReport() Report {
	return Report{
		Findings: []Finding{
			newFinding("go.doc.missing", "apps/api/main.go", 12, Warn, "exported Run has no doc comment"),
			newFinding("docker.user.root", "Dockerfile", 0, Error, "no USER instruction"),
			newFinding("go.doc.missing", "apps/api/run.go", 3, Warn, "exported Do has no doc comment"),
		},
		Files: 3,
		Lines: 400,
	}
}

func TestJSONCarriesTheDerivedCountsAndGrade(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, ciReport()); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var payload jsonReport
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if payload.Counts != (jsonCounts{Error: 1, Warn: 2, Info: 0}) {
		t.Errorf("counts = %+v, want 1 error and 2 warnings", payload.Counts)
	}
	if payload.Grade != ciReport().Grade() {
		t.Errorf("grade = %q, want %q", payload.Grade, ciReport().Grade())
	}
	if len(payload.Findings) != 3 {
		t.Errorf("findings = %d, want 3", len(payload.Findings))
	}
}

func TestGradeDoesNotCollapseOnASmallTarget(t *testing.T) {
	small := Report{
		Findings: []Finding{newFinding("docker.healthcheck.missing", "Dockerfile", 1, Info, "no HEALTHCHECK")},
		Files:    1,
		Lines:    37,
	}
	if got := small.Grade(); got != "A" {
		t.Errorf("one info finding on a 37-line file grades %q, want A", got)
	}
	if got := (Report{}).Grade(); got != "A" {
		t.Errorf("an empty report grades %q, want A", got)
	}

	bad := Report{Findings: make([]Finding, 40), Lines: 1000}
	if got := bad.Grade(); got != "F" {
		t.Errorf("40 findings per 1000 lines grades %q, want F", got)
	}
}

func TestSARIFIndexesEveryResultIntoTheRuleTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, ciReport(), "0.2.0"); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if log.Version != "2.1.0" || len(log.Runs) != 1 {
		t.Fatalf("want one 2.1.0 run, got version %q and %d runs", log.Version, len(log.Runs))
	}

	run := log.Runs[0]
	if run.Tool.Driver.Version != "0.2.0" {
		t.Errorf("driver version = %q, want 0.2.0", run.Tool.Driver.Version)
	}
	if len(run.Tool.Driver.Rules) != 2 {
		t.Fatalf("want 2 fired rules in the table, got %d", len(run.Tool.Driver.Rules))
	}
	for _, result := range run.Results {
		if result.RuleIndex < 0 || result.RuleIndex >= len(run.Tool.Driver.Rules) {
			t.Fatalf("ruleIndex %d is outside the rule table", result.RuleIndex)
		}
		if got := run.Tool.Driver.Rules[result.RuleIndex].ID; got != result.RuleID {
			t.Errorf("ruleIndex points at %q, want %q", got, result.RuleID)
		}
	}
}

func TestSARIFNeverEmitsLineZero(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, ciReport(), "0.2.0"); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	for _, result := range log.Runs[0].Results {
		region := result.Locations[0].PhysicalLocation.Region
		if region.StartLine < 1 || region.StartColumn < 1 {
			t.Errorf("%s: region %d:%d, both must be 1-based",
				result.RuleID, region.StartLine, region.StartColumn)
		}
	}
}

var sarifLevelCases = []struct {
	severity Severity
	want     string
}{{Error, "error"}, {Warn, "warning"}, {Info, "note"}}

func TestSARIFLevels(t *testing.T) {
	for _, tc := range sarifLevelCases {
		if got := sarifLevel(tc.severity); got != tc.want {
			t.Errorf("sarifLevel(%s) = %q, want %q", tc.severity, got, tc.want)
		}
	}
}

func TestGitHubAnnotationCarriesFileLineAndRule(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteGitHub(&buf, ciReport()); err != nil {
		t.Fatalf("WriteGitHub: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("want one annotation per finding, got %d", len(lines))
	}
	want := "::warning file=apps/api/main.go,line=12,col=1,title=go.doc.missing::" +
		"exported Run has no doc comment"
	if lines[0] != want {
		t.Errorf("annotation =\n  %s\nwant\n  %s", lines[0], want)
	}
	if !strings.HasPrefix(lines[1], "::error file=Dockerfile,line=1,") {
		t.Errorf("a line-0 finding must still anchor at line 1, got %s", lines[1])
	}
}

func TestGitHubEscapesWhatWouldTruncateTheCommand(t *testing.T) {
	report := Report{Findings: []Finding{
		newFinding("gen.todo", "a,b:c.go", 1, Info, "100% wrong\nsecond line"),
	}}

	var buf bytes.Buffer
	if err := WriteGitHub(&buf, report); err != nil {
		t.Fatalf("WriteGitHub: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "::notice file=a%2Cb%3Ac.go,line=1,col=1,title=gen.todo::100%25 wrong%0Asecond line"
	if got != want {
		t.Errorf("annotation =\n  %s\nwant\n  %s", got, want)
	}
	if strings.Count(got, "\n") != 0 {
		t.Error("a newline survived into the command, which truncates the annotation")
	}
}
