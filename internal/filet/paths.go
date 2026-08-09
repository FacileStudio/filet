package filet

import (
	"os"
	"path/filepath"
	"strings"
)

// RootRelative reports where path sits inside root. Architecture patterns are
// written against the project layout, so they have to match the same way
// wherever filet happened to be invoked from.
func RootRelative(root, path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// DisplayPath is what a finding shows: relative to the working directory, so an
// editor or a terminal can follow it. A path outside the working directory is
// shown absolute rather than as a pile of "..".
func DisplayPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(abs)
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}
