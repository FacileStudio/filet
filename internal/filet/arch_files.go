package filet

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// requiredFiles checks the per-directory contract: every directory matching a
// glob must contain the files that glob demands. A directory covered by an
// exact pattern uses that one, so a convention can carry its own exceptions.
func requiredFiles(cfg *Config, dirs []string) []Finding {
	rules := cfg.Architecture.RequiredFiles
	if !cfg.Enabled("arch.file.missing") || len(rules) == 0 {
		return nil
	}

	var out []Finding
	for _, dir := range dirs {
		pattern, ok := bestPattern(rules, dir)
		if !ok {
			continue
		}
		for _, name := range rules[pattern] {
			full := filepath.Join(cfg.root, filepath.FromSlash(dir), name)
			if _, err := os.Stat(full); err == nil {
				continue
			}
			out = append(out, newFinding("arch.file.missing", DisplayPath(full), 1, Error,
				dir+" matches "+pattern+" so it must declare "+name))
		}
	}
	return out
}

// bestPattern returns the most specific glob covering dir: an exact match wins
// over a wildcard, and a longer literal prefix wins over a shorter one.
func bestPattern(rules map[string][]string, dir string) (string, bool) {
	best, found := "", false
	for pattern := range rules {
		ok, err := path.Match(pattern, dir)
		if err != nil || !ok {
			continue
		}
		if !found || specificity(pattern) > specificity(best) {
			best, found = pattern, true
		}
	}
	return best, found
}

func specificity(pattern string) int {
	if i := strings.IndexAny(pattern, "*?["); i >= 0 {
		return i
	}
	return len(pattern) + 1
}
