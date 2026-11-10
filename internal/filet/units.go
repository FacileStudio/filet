package filet

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// Unit is one project root under a gate, fuzz or test target: a directory with
// a go.mod, a Cargo.toml, a package.json carrying a check or test script, or
// another manifest DetectSuites knows. Path is relative to the target, "."
// for the target itself.
type Unit struct {
	Path   string   `json:"path"`
	Kinds  []string `json:"kinds"`
	Dir    string   `json:"-"`
	Suites []Suite  `json:"-"`
	Check  bool     `json:"-"`
}

// UnitScan is what DiscoverUnits found: the units in walk order, and how many
// source files of each language (go, rust, node) sit under the target, so a
// caller can tell a language that has files but no unit.
type UnitScan struct {
	Units []Unit
	Files map[string]int
}

var unitSkipDirs = []string{"node_modules", "vendor", "dist", "build", "target", ".git", "testdata"}

var nodeSuites = []string{"bun", "pnpm", "yarn", "npm", "deno"}

// DiscoverUnits walks target, honouring the config's ignore list and never
// entering node_modules, vendor, dist, build, target, .git or testdata, and
// returns every unit it finds. A crate under another crate is left to the
// outer one, which is what cargo does with workspace members. Below the target
// a package.json counts only when it has a check or test script; at the target
// any package.json does, as DetectSuites has always treated it.
func DiscoverUnits(cfg *Config, target string) (UnitScan, error) {
	scan := UnitScan{Units: []Unit{}, Files: map[string]int{}}
	var crates []string
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case !d.IsDir():
			scan.Files[sourceLang(d.Name())]++
			return nil
		}
		if skip := skipUnitDir(cfg, path, target, d.Name()); skip != nil {
			return skip
		}
		if u, ok := unitAt(path, target, crates); ok {
			scan.Units = append(scan.Units, u)
			if hasSuite(u.Suites, "cargo") {
				crates = append(crates, path)
			}
		}
		return nil
	})
	delete(scan.Files, "")
	return scan, err
}

func skipUnitDir(cfg *Config, path, target, name string) error {
	if path != target && slices.Contains(unitSkipDirs, name) {
		return filepath.SkipDir
	}
	return skipDir(cfg, path, target, name)
}

func sourceLang(name string) string {
	switch filepath.Ext(name) {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".ts", ".tsx", ".svelte", ".vue":
		return "node"
	default:
		return ""
	}
}

func unitAt(dir, target string, crates []string) (Unit, bool) {
	scripts := packageScripts(dir)
	_, check := scripts["check"]
	path := unitPath(target, dir)
	var suites []Suite
	for _, s := range DetectSuites(dir) {
		if keepSuite(s, dir == target, scripts, underAny(dir, crates)) {
			s.Unit = path
			suites = append(suites, s)
		}
	}
	if len(suites) == 0 && !check {
		return Unit{}, false
	}
	return Unit{Path: path, Dir: dir, Suites: suites, Check: check, Kinds: unitKinds(suites, check)}, true
}

func keepSuite(s Suite, atTarget bool, scripts map[string]string, inCrate bool) bool {
	switch {
	case s.Name == "cargo":
		return !inCrate
	case s.Name == "deno" || !slices.Contains(nodeSuites, s.Name):
		return true
	default:
		_, ok := scripts["test"]
		return atTarget || ok
	}
}

func underAny(dir string, roots []string) bool {
	return slices.ContainsFunc(roots, func(r string) bool {
		rel, err := filepath.Rel(r, dir)
		return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
	})
}

func unitPath(target, dir string) string {
	rel, err := filepath.Rel(target, dir)
	if err != nil {
		return filepath.ToSlash(dir)
	}
	return filepath.ToSlash(rel)
}

func unitKinds(suites []Suite, check bool) []string {
	kinds := make([]string, 0, len(suites)+1)
	for _, s := range suites {
		kinds = append(kinds, s.Name)
	}
	if check && !slices.ContainsFunc(suites, func(s Suite) bool { return slices.Contains(nodeSuites, s.Name) }) {
		kinds = append(kinds, "bun")
	}
	return kinds
}
