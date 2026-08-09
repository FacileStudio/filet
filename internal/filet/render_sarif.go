package filet

import (
	"encoding/json"
	"io"
	"path/filepath"
	"slices"
)

const (
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
	sarifVersion = "2.1.0"
	sarifHomeURI = "https://github.com/FacileStudio/filet"
)

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ShortDescription sarifText `json:"shortDescription"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	RuleIndex int             `json:"ruleIndex"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

// WriteSARIF renders the report as SARIF 2.1.0, the format GitHub code scanning
// and most review tools ingest.
//
// Paths are emitted relative to the working directory with forward slashes,
// because a code scanning upload matches results against repository paths and
// silently shows nothing for a URI it cannot resolve. Run filet from the
// repository root in CI, or the annotations land on no file at all.
func WriteSARIF(w io.Writer, r Report, version string) error {
	log := sarifLog{Schema: sarifSchema, Version: sarifVersion, Runs: []sarifRun{sarifRunOf(r, version)}}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}

func sarifRunOf(r Report, version string) sarifRun {
	fired := firedRules(r.Findings)
	driver := sarifDriver{
		Name:           "filet",
		Version:        version,
		InformationURI: sarifHomeURI,
		Rules:          sarifRules(fired),
	}
	return sarifRun{Tool: sarifTool{Driver: driver}, Results: sarifResults(r.Findings, fired)}
}

// firedRules lists the rule ids present in the findings, sorted, so that a
// rule's index in the driver's rule table is stable between two runs.
func firedRules(findings []Finding) []string {
	ids := make([]string, 0, len(registry))
	for _, f := range findings {
		if !slices.Contains(ids, f.Rule) {
			ids = append(ids, f.Rule)
		}
	}
	slices.Sort(ids)
	return ids
}

func sarifRules(ids []string) []sarifRule {
	described := make(map[string]string, len(registry))
	for _, rule := range Rules() {
		described[rule.ID] = rule.Description
	}

	rules := make([]sarifRule, 0, len(ids))
	for _, id := range ids {
		text := described[id]
		if text == "" {
			text = id
		}
		rules = append(rules, sarifRule{ID: id, Name: id, ShortDescription: sarifText{Text: text}})
	}
	return rules
}

func sarifResults(findings []Finding, fired []string) []sarifResult {
	results := make([]sarifResult, 0, len(findings))
	for _, f := range findings {
		results = append(results, sarifResult{
			RuleID:    f.Rule,
			RuleIndex: slices.Index(fired, f.Rule),
			Level:     sarifLevel(f.Severity),
			Message:   sarifText{Text: f.Message},
			Locations: []sarifLocation{sarifLocationOf(f)},
		})
	}
	return results
}

func sarifLocationOf(f Finding) sarifLocation {
	physical := sarifPhysical{
		ArtifactLocation: sarifArtifact{URI: filepath.ToSlash(f.File)},
		Region:           sarifRegion{StartLine: max(f.Line, 1), StartColumn: max(f.Column, 1)},
	}
	return sarifLocation{PhysicalLocation: physical}
}

// sarifLevel maps a severity onto the four levels SARIF defines. Info becomes
// "note" rather than "none", since "none" is for results that carry no judgement
// at all and would be hidden by every consumer.
func sarifLevel(s Severity) string {
	switch s {
	case Error:
		return "error"
	case Warn:
		return "warning"
	default:
		return "note"
	}
}
