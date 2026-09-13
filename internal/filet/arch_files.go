package filet

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strconv"
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
func bestPattern[V any](rules map[string]V, dir string) (string, bool) {
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

// filesPerDir flags every directory holding more tracked source files than
// architecture.maxFilesPerDir allows, one finding per directory.
func filesPerDir(cfg *Config, counts map[string]int) []Finding {
	if !cfg.Enabled("arch.dir.files") {
		return nil
	}
	var out []Finding
	for dir, n := range counts {
		limit, ok := cfg.Architecture.MaxFilesPerDir.limitFor(dir)
		if !ok || n <= limit {
			continue
		}
		path := filepath.Join(cfg.root, filepath.FromSlash(dir))
		out = append(out, newFinding("arch.dir.files", DisplayPath(path), 1, Warn,
			"directory holds "+strconv.Itoa(n)+" files (limit "+strconv.Itoa(limit)+")"))
	}
	return out
}

// limitFor returns the limit for dir: the most specific matching glob wins
// over the general limit, mirroring requiredFiles. A limit of zero never
// limits, matching maxDepth, so a glob can opt a directory out entirely.
func (d DirFileLimits) limitFor(dir string) (int, bool) {
	if len(d.Globs) > 0 {
		if pattern, ok := bestPattern(d.Globs, dir); ok {
			return d.Globs[pattern], d.Globs[pattern] > 0
		}
	}
	if d.General > 0 {
		return d.General, true
	}
	return 0, false
}

// UnmarshalYAML accepts either a scalar limit for every directory or a map
// from glob to limit.
func (d *DirFileLimits) UnmarshalYAML(ctx context.Context, decode func(any) error) error {
	var n int
	if err := decode(&n); err == nil {
		d.General = n
		return nil
	}
	return decode(&d.Globs)
}
