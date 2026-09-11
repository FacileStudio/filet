package filet

import (
	"bytes"
	"os"
)

func readFile(path string) ([]byte, error) { return os.ReadFile(path) }

func countLines(src []byte) int {
	if len(src) == 0 {
		return 0
	}
	return bytes.Count(bytes.TrimRight(src, "\n"), []byte{'\n'}) + 1
}

// Analyze scans target and runs every code and architecture rule over it.
func Analyze(cfg *Config, target string) (Report, error) {
	files, err := Scan(cfg, target)
	if err != nil {
		return Report{}, err
	}

	w := newCacheWorker(cfg, target)
	report := Report{Files: len(files)}

	for _, f := range files {
		n := len(f.Lines)
		if n > 0 && f.Lines[n-1] == "" {
			n--
		}
		report.Lines += n
		report.Findings = append(report.Findings, checkFileCached(w, cfg, f)...)
	}

	report.Findings = append(report.Findings, checkGoCached(w, cfg, files)...)
	report.Findings = append(report.Findings, checkRestCached(w, cfg, files)...)
	w.write()

	SortFindings(report.Findings)
	return report, nil
}

// AnalyzeDockerfiles finds and checks every Dockerfile under target.
func AnalyzeDockerfiles(cfg *Config, target string) (Report, error) {
	paths, err := FindDockerfiles(cfg, target)
	if err != nil {
		return Report{}, err
	}

	report := Report{Files: len(paths)}
	for _, p := range paths {
		src, err := readFile(p)
		if err != nil {
			continue
		}
		report.Lines += countLines(src)
		report.Findings = append(report.Findings, CheckDockerfile(cfg, p, src)...)
	}
	SortFindings(report.Findings)
	return report, nil
}

// FailLevel returns true when the report contains a finding at or above the configured threshold.
func FailLevel(cfg *Config, r Report) bool {
	threshold := ParseSeverity(cfg.FailOn)
	for _, f := range r.Findings {
		if f.Severity >= threshold {
			return true
		}
	}
	return false
}
