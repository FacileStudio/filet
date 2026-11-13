package filet

import (
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

var strictFixtures = []string{"assert", "close", "goroutine", "import", "nilflow", "sql", "switch", "timeout"}

func strictConfig() *Config {
	cfg := DefaultConfig()
	cfg.Style.Strict = true
	return cfg
}

func findingLines(f []Finding) []int {
	out := make([]int, 0, len(f))
	for _, x := range f {
		out = append(out, x.Line)
	}
	slices.Sort(out)
	return out
}

func wantLines(body string) []int {
	var out []int
	for i, line := range strings.Split(body, "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), "// want") {
			out = append(out, i+1)
		}
	}
	return out
}

func TestStrictRulesMatchTheirFixtures(t *testing.T) {
	for _, name := range strictFixtures {
		t.Run(name, func(t *testing.T) { checkStrictFixture(t, name) })
	}
}

func checkStrictFixture(t *testing.T, name string) {
	body := readFixture(t, filepath.Join("strict", name+".go"))
	got := ruleFindings(CheckGo(strictConfig(), source(t, name+".go", body)), "go.strict."+name)
	want := wantLines(body)
	if len(want) == 0 {
		t.Fatal("the fixture marks no positive case")
	}
	if lines := findingLines(got); !slices.Equal(lines, want) {
		t.Errorf("findings on lines %v, the fixture marks %v: %v", lines, want, got)
	}
	for _, f := range got {
		if f.Severity != Error {
			t.Errorf("%s reports at %v, want error", f.Rule, f.Severity)
		}
	}
}

func TestStrictRulesAreOptIn(t *testing.T) {
	for _, name := range strictFixtures {
		body := readFixture(t, filepath.Join("strict", name+".go"))
		for _, f := range CheckGo(DefaultConfig(), source(t, name+".go", body)) {
			if strings.HasPrefix(f.Rule, "go.strict.") {
				t.Errorf("%s fired without style.strict: %s", f.Rule, f.Message)
			}
		}
	}
}

func TestStrictRulesSkipTestFiles(t *testing.T) {
	for _, name := range strictFixtures {
		body := readFixture(t, filepath.Join("strict", name+".go"))
		for _, f := range CheckGo(strictConfig(), source(t, name+"_test.go", body)) {
			if strings.HasPrefix(f.Rule, "go.strict.") {
				t.Errorf("%s fired in a test file: %s", f.Rule, f.Message)
			}
		}
	}
}

func TestStrictRunsWithoutTheLeakRule(t *testing.T) {
	dir := t.TempDir()
	cfg := "style:\n  strict: true\ndisabled: [go.leak.resource]\ncache:\n  enabled: false\nlsp:\n  enabled: false\n"
	if err := writeTemp(filepath.Join(dir, "filet.yml"), cfg); err != nil {
		t.Fatal(err)
	}
	if err := writeTemp(filepath.Join(dir, "switch.go"), readFixture(t, filepath.Join("strict", "switch.go"))); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Analyze(loaded, dir)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(report.Findings, "go.strict.switch"); n != 3 {
		t.Fatalf("want 3 go.strict.switch findings through Analyze, got %d in %v", n, ruleIDs(report.Findings))
	}
}

func TestFacileStrictPresetIsFacilePlusStrict(t *testing.T) {
	strict := loadFixture(t, "preset: facile-strict\n")
	facile := loadFixture(t, "preset: facile\n")
	if !strict.Style.Strict || facile.Style.Strict || DefaultConfig().Style.Strict {
		t.Fatalf("strict is %v under facile-strict, %v under facile, %v by default",
			strict.Style.Strict, facile.Style.Strict, DefaultConfig().Style.Strict)
	}
	facile.Style.Strict = true
	pairs := map[string][2]any{
		"limits": {facile.Limits, strict.Limits}, "style": {facile.Style, strict.Style},
		"architecture": {facile.Architecture, strict.Architecture}, "ignore": {facile.Ignore, strict.Ignore},
		"extensions": {facile.Extensions, strict.Extensions}, "disabled": {facile.Disabled, strict.Disabled},
		"failOn": {facile.FailOn, strict.FailOn},
	}
	for key, p := range pairs {
		if !reflect.DeepEqual(p[0], p[1]) {
			t.Errorf("%s: facile says %v, facile-strict says %v", key, p[0], p[1])
		}
	}
}

func TestStrictSwitchListsOnlyConstantsTheCallerCanName(t *testing.T) {
	f := source(t, "tok.go", "package tok\n\nimport \"go/token\"\n\nfunc kind(t token.Token) bool {\n"+
		"\tswitch t {\n\tcase token.ILLEGAL:\n\t\treturn true\n\t}\n\treturn false\n}\n")
	got := ruleFindings(CheckGo(strictConfig(), f), "go.strict.switch")
	if len(got) != 1 {
		t.Fatalf("want one finding, got %v", got)
	}
	if msg := got[0].Message; strings.Contains(msg, "literal_beg") || !strings.Contains(msg, "EOF, COMMENT, IDENT") {
		t.Errorf("the message must list exported constants in declaration order: %s", msg)
	}
}

func TestScaffoldKeepsStrict(t *testing.T) {
	dir := t.TempDir()
	if err := Scaffold(filepath.Join(dir, "filet.yml"), "facile-strict"); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Style.Strict {
		t.Fatal("a config scaffolded from facile-strict lost style.strict")
	}
}
