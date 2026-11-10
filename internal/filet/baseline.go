package filet

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// BaselineName is the ratchet file filet reads and writes at the config root,
// unless filet.yml names another one with `baseline:`.
const BaselineName = ".filet-baseline.json"

const staleRule = "gen.baseline.stale"

type counts map[string]map[string]int

func (c counts) add(rule, path string, n int) {
	if c[rule] == nil {
		c[rule] = map[string]int{}
	}
	c[rule][path] += n
}

// ratchet holds one run's baseline next to the counts the run produced. scope
// is the checked target relative to the config root: entries outside it are
// neither compared nor rewritten, so a scoped run cannot erase the rest.
type ratchet struct {
	cfg     *Config
	path    string
	scope   string
	base    counts
	current counts
}

// Ratchet applies the baseline to r. Mode "" compares against the baseline
// file when one exists, "write" records the current counts, and "update"
// lowers recorded counts to the current ones without ever raising one. It
// returns the report to render and the report to gate on: a finding the
// baseline accounts for renders as info with a "(baseline)" suffix and is left
// out of the gate, and a count that dropped adds a gen.baseline.stale error.
func Ratchet(cfg *Config, target, mode string, r Report) (Report, Report, error) {
	SortFindings(r.Findings)
	if mode != "" && mode != "write" && mode != "update" {
		return r, r, fmt.Errorf("-baseline: unknown mode %q (use write or update)", mode)
	}
	rc, found, err := openRatchet(cfg, target, mode)
	if err != nil || (!found && mode != "write") {
		return r, r, err
	}
	rc.current = gatingCounts(cfg, r.Findings)
	switch mode {
	case "write":
		rc.base = rc.merged()
	case "update":
		rc.base = rc.shrunk()
	}
	if mode != "" {
		if err := writeBaseline(rc.path, rc.base); err != nil {
			return r, r, err
		}
	}
	shown, gate := rc.apply(r)
	return shown, gate, nil
}

func gatingCounts(cfg *Config, findings []Finding) counts {
	threshold := ParseSeverity(cfg.FailOn)
	c := counts{}
	for _, f := range findings {
		if f.Severity >= threshold {
			c.add(f.Rule, RootRelative(cfg.root, f.File), 1)
		}
	}
	return c
}

func (rc *ratchet) inScope(path string) bool {
	return rc.scope == "." || path == rc.scope || strings.HasPrefix(path, rc.scope+"/")
}

func (rc *ratchet) merged() counts {
	out := counts{}
	for rule, paths := range rc.base {
		for path, n := range paths {
			if !rc.inScope(path) {
				out.add(rule, path, n)
			}
		}
	}
	for rule, paths := range rc.current {
		for path, n := range paths {
			out.add(rule, path, n)
		}
	}
	return out
}

func (rc *ratchet) shrunk() counts {
	out := counts{}
	for rule, paths := range rc.base {
		for path, n := range paths {
			switch m := rc.current[rule][path]; {
			case !rc.inScope(path):
				out.add(rule, path, n)
			case m > 0:
				out.add(rule, path, min(n, m))
			}
		}
	}
	return out
}

func (rc *ratchet) apply(r Report) (Report, Report) {
	threshold := ParseSeverity(rc.cfg.FailOn)
	shown, gate := r, r
	shown.Findings, gate.Findings = nil, nil
	for _, f := range r.Findings {
		path := RootRelative(rc.cfg.root, f.File)
		if f.Severity >= threshold && rc.current[f.Rule][path] <= rc.base[f.Rule][path] {
			f.Severity, f.Level, f.Message = Info, Info.String(), f.Message+" (baseline)"
			shown.Findings = append(shown.Findings, f)
			continue
		}
		shown.Findings = append(shown.Findings, f)
		gate.Findings = append(gate.Findings, f)
	}
	stale := rc.stale()
	shown.Findings = append(shown.Findings, stale...)
	gate.Findings = append(gate.Findings, stale...)
	SortFindings(shown.Findings)
	return shown, gate
}

func (rc *ratchet) stale() []Finding {
	if !rc.cfg.Enabled(staleRule) {
		return nil
	}
	var out []Finding
	for _, rule := range slices.Sorted(maps.Keys(rc.base)) {
		for _, path := range slices.Sorted(maps.Keys(rc.base[rule])) {
			n, m := rc.base[rule][path], rc.current[rule][path]
			if rc.inScope(path) && m < n {
				out = append(out, newFinding(staleRule, DisplayPath(rc.path), 1, Error,
					fmt.Sprintf("%s on %s dropped from %d to %d; run filet check -baseline update", rule, path, n, m)))
			}
		}
	}
	return out
}
