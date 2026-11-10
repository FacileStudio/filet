package filet

import (
	"fmt"
	"io"
	"slices"
)

type impliedLang struct {
	lang     string
	exts     string
	manifest string
	kinds    []string
	rows     [][2]string
}

var impliedLangs = []impliedLang{
	{lang: "go", exts: ".go", manifest: "go.mod", kinds: []string{"go"},
		rows: [][2]string{{"format", "gofmt"}, {"analyze", "vet"}, {"test", "go"}}},
	{lang: "rust", exts: ".rs", manifest: "Cargo.toml", kinds: []string{"cargo"},
		rows: [][2]string{{"format", "cargo-fmt"}, {"analyze", "clippy"}, {"test", "cargo"}}},
	{lang: "node", exts: ".ts/.tsx/.svelte/.vue", manifest: "package.json with a check or test script",
		kinds: nodeSuites, rows: [][2]string{{"analyze", "bun-check"}, {"test", "node"}}},
}

func missingSteps(scan UnitScan) []GateStep {
	var out []GateStep
	for _, l := range impliedLangs {
		n := scan.Files[l.lang]
		if n == 0 || slices.ContainsFunc(scan.Units, l.covers) {
			continue
		}
		detail := fmt.Sprintf("%d %s files but no %s found", n, l.exts, l.manifest)
		for _, row := range l.rows {
			out = append(out, failedStep(row[0], row[1], "look for "+l.manifest, detail))
		}
	}
	return out
}

func (l impliedLang) covers(u Unit) bool {
	return slices.ContainsFunc(u.Kinds, func(k string) bool { return slices.Contains(l.kinds, k) })
}

func failedStep(phase, name, command, detail string) GateStep {
	return GateStep{
		Phase:   phase,
		Name:    name,
		Unit:    ".",
		Command: command,
		Run: func(io.Writer) (StepStatus, string) {
			return StepFailed, detail
		},
	}
}
