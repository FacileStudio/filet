package filet

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Instruction is one logical Dockerfile instruction, continuations already joined.
type Instruction struct {
	Cmd  string
	Args string
	Line int
}

var (
	secretKeyRe   = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key)`)
	compilerBase  = regexp.MustCompile(`(?i)^(golang|node|rust|maven|gradle|python|openjdk|elixir|ruby)\b`)
	curlPipeShell = regexp.MustCompile(`(?i)(curl|wget)[^|&;]*\|\s*(sudo\s+)?(ba)?sh`)
	depInstallRe  = regexp.MustCompile(`(?i)\b(` + strings.Join([]string{
		"apt-get install", "apk add", "yum install", "dnf install",
		"pip3? install", "poetry install", "npm (ci|install)", "yarn install",
		"pnpm install", "bun install", "go mod download", "cargo fetch",
		"bundle install", "composer install", `mix deps\.get`,
	}, "|") + `)\b`)
)

// ParseDockerfile splits a Dockerfile into instructions, joining backslash continuations.
func ParseDockerfile(src []byte) []Instruction {
	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	var out []Instruction
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		start := i + 1
		joined := trimmed
		for strings.HasSuffix(strings.TrimSpace(joined), "\\") && i+1 < len(lines) {
			i++
			joined = strings.TrimSuffix(strings.TrimSpace(joined), "\\") + " " + strings.TrimSpace(lines[i])
		}
		parts := strings.SplitN(strings.TrimSpace(joined), " ", 2)
		inst := Instruction{Cmd: strings.ToUpper(parts[0]), Line: start}
		if len(parts) > 1 {
			inst.Args = strings.TrimSpace(parts[1])
		}
		out = append(out, inst)
	}
	return out
}

// CheckDockerfile runs every Dockerfile rule and returns the findings.
func CheckDockerfile(cfg *Config, path string, src []byte) []Finding {
	rel := path
	if r, err := filepath.Rel(cfg.root, path); err == nil {
		rel = filepath.ToSlash(r)
	}

	var out []Finding
	add := func(rule string, line int, sev Severity, msg string) {
		if cfg.Enabled(rule) {
			out = append(out, newFinding(rule, rel, line, sev, msg))
		}
	}

	var s dockerScan
	s.installedAfter, s.copyAllLine = -1, -1
	for _, in := range ParseDockerfile(src) {
		s.instruction(in, add)
	}
	s.final(path, add)
	return out
}

type addLine func(rule string, line int, sev Severity, msg string)

type dockerScan struct {
	froms, runs                               int
	hasUser, hasWork, hasHealth               bool
	installedAfter, copyAllLine, lastUserRoot int
	firstFromRef                              string
	firstFromLine                             int
}

// FindDockerfiles returns every Dockerfile under target, ignoring configured directories.
func FindDockerfiles(cfg *Config, target string) ([]string, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{target}, nil
	}
	var out []string
	err = filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return skipDir(cfg, path, target, d.Name())
		}
		if isDockerfile(d.Name()) {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func isDockerfile(name string) bool {
	return name == "Dockerfile" ||
		strings.HasPrefix(name, "Dockerfile.") ||
		strings.HasSuffix(name, ".Dockerfile")
}
