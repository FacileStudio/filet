package filet

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// CheckArchitecture validates the project layout against the architecture config.
func CheckArchitecture(cfg *Config, files []SourceFile) []Finding {
	var out []Finding
	a := cfg.Architecture

	out = append(out, missingDirs(cfg)...)

	seenDir := map[string]bool{}
	var dirs []string
	for _, f := range files {
		dir := filepath.ToSlash(filepath.Dir(f.Rel))
		if !seenDir[dir] {
			seenDir[dir] = true
			dirs = append(dirs, dir)
			out = append(out, forbiddenDir(cfg, dir)...)
		}
		out = append(out, depth(cfg, f), filename(cfg, f))
		if cfg.Enabled("arch.import.forbidden") && len(a.ForbiddenImports) > 0 {
			out = append(out, checkImports(a.ForbiddenImports, f)...)
		}
	}
	out = append(out, requiredFiles(cfg, dirs)...)
	return slices.DeleteFunc(out, func(f Finding) bool { return f.Rule == "" })
}

func missingDirs(cfg *Config) []Finding {
	if !cfg.Enabled("arch.dir.missing") {
		return nil
	}
	var out []Finding
	for _, dir := range cfg.Architecture.RequiredDirs {
		if info, err := os.Stat(filepath.Join(cfg.root, dir)); err != nil || !info.IsDir() {
			out = append(out, newFinding("arch.dir.missing",
				DisplayPath(filepath.Join(cfg.root, dir)), 1, Error, "required directory is missing"))
		}
	}
	return out
}

func forbiddenDir(cfg *Config, dir string) []Finding {
	if !cfg.Enabled("arch.dir.forbidden") {
		return nil
	}
	var out []Finding
	for _, bad := range cfg.Architecture.ForbiddenDirs {
		if dir == bad || strings.HasPrefix(dir, strings.TrimSuffix(bad, "/")+"/") {
			out = append(out, newFinding("arch.dir.forbidden", dir, 1, Error,
				"directory matches forbidden pattern "+bad))
		}
	}
	return out
}

func depth(cfg *Config, f SourceFile) Finding {
	maxDepth := cfg.Architecture.MaxDepth
	d := strings.Count(f.Rel, "/")
	if !cfg.Enabled("arch.depth") || maxDepth <= 0 || d <= maxDepth {
		return Finding{}
	}
	return newFinding("arch.depth", f.Display, 1, Warn,
		"nested "+strconv.Itoa(d)+" directories deep (limit "+strconv.Itoa(maxDepth)+")")
}

func filename(cfg *Config, f SourceFile) Finding {
	pattern := cfg.Architecture.FileNamePattern
	if !cfg.Enabled("arch.filename") || pattern == "" {
		return Finding{}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return Finding{}
	}
	if re.MatchString(filepath.Base(f.Rel)) {
		return Finding{}
	}
	return newFinding("arch.filename", f.Display, 1, Warn,
		"filename does not match "+pattern)
}

func checkImports(rules map[string][]string, f SourceFile) []Finding {
	var out []Finding
	for prefix, banned := range rules {
		if !strings.HasPrefix(f.Rel, strings.TrimSuffix(prefix, "/")) {
			continue
		}
		out = append(out, bannedImports(prefix, banned, f)...)
	}
	return out
}

func bannedImports(prefix string, banned []string, f SourceFile) []Finding {
	var out []Finding
	for _, imp := range imports(f) {
		if !matchesBan(imp, banned) {
			continue
		}
		out = append(out, newFinding("arch.import.forbidden", f.Display, 1, Error,
			prefix+" must not import "+imp))
	}
	return out
}

func matchesBan(imp string, banned []string) bool {
	for _, b := range banned {
		if imp == b || strings.HasPrefix(imp, strings.TrimSuffix(b, "/")+"/") {
			return true
		}
	}
	return false
}
