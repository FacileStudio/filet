package filet

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// commentedCode looks for a commented-out *statement*. Keywords that also
	// open an English sentence — for, if, class, case — have to be backed by code
	// punctuation before they count. A single space may follow the marker: a tab
	// or a wider indent means a godoc or markdown code block, which is
	// documentation and must survive.
	commentedCode = regexp.MustCompile(`^\s*(//|#) ?(` +
		`(func|def|const|let|var|import|export|package)(\s|$)` +
		`|(if|for|while|switch|case|class|else|elif)(\s|$).*[(){};=]` +
		`|return\s+\S+\s*$` +
		`|(break|continue|fallthrough|pass)\s*;?\s*$` +
		`|[\w.]+\s*[-+*/|&^]?:?=[^=]` +
		`|[\w.]+\([^;]*\)\s*[;{]?\s*$` +
		`|[});]\s*$` +
		`)`)
	braceless = map[string]bool{".py": true, ".rb": true, ".sh": true}

	funcDeclRe = map[string]*regexp.Regexp{
		".ts":     jsFuncRe,
		".tsx":    jsFuncRe,
		".js":     jsFuncRe,
		".jsx":    jsFuncRe,
		".svelte": jsFuncRe,
		".rs":     regexp.MustCompile(`^\s*(pub(\([^)]*\))?\s+)?(const\s+|async\s+|unsafe\s+|extern\s+"[^"]*"\s+)*fn\s+\w+`),
		".py":     regexp.MustCompile(`^\s*(async\s+)?def\s+\w+`),
		".rb":     regexp.MustCompile(`^\s*def\s+\w+`),
		".java":   regexp.MustCompile(`^\s*(public|private|protected|static|final|\s)+[\w<>\[\].]+\s+\w+\s*\([^;]*\)\s*\{`),
		".sh":     regexp.MustCompile(`^\s*(function\s+)?\w+\s*\(\)\s*\{`),
	}
)

var jsFuncRe = regexp.MustCompile(`^\s*(export\s+)?(default\s+)?(async\s+)?(` +
	`function\s*\*?\s+\w+` +
	`|(const|let|var)\s+\w+\s*=\s*(async\s+)?(function\b|\([^)]*\)\s*(:[^=]+)?=>|\w+\s*=>)` +
	`)`)

// CheckGeneric runs the language-agnostic rules over a single file.
func CheckGeneric(cfg *Config, f SourceFile) []Finding {
	if f.IsGenerated() {
		return nil
	}
	s := &lineScan{cfg: cfg, f: f, funcRe: funcDeclRe[f.Ext]}
	s.fileLength()
	for i, raw := range f.Lines {
		s.line(i+1, raw)
	}
	s.nesting()
	return s.out
}

type lineScan struct {
	cfg          *Config
	f            SourceFile
	funcRe       *regexp.Regexp
	funcs        int
	depth        int
	maxDepth     int
	maxDepthLine int
	state        lineState
	out          []Finding
}

func (s *lineScan) fileLength() {
	n := len(s.f.Lines)
	if n > 0 && s.f.Lines[n-1] == "" {
		n--
	}
	if s.cfg.Enabled("gen.file.long") && n > s.cfg.Limits.FileLines {
		s.out = append(s.out, newFinding("gen.file.long", s.f.Display, 1, Warn,
			fmt.Sprintf("file is %d lines (limit %d)", n, s.cfg.Limits.FileLines)))
	}
}

func (s *lineScan) line(n int, raw string) {
	stripped, state := stripLine(raw, s.state)
	s.state = state
	s.out = append(s.out, checkLine(s.cfg, s.f, n, raw, stripped)...)
	s.countFunc(n, raw)
	if !braceless[s.f.Ext] {
		s.brace(n, stripped)
	}
}

func (s *lineScan) countFunc(n int, raw string) {
	limit := s.cfg.Limits.FuncsPerFile
	if s.funcRe == nil || limit <= 0 || !s.cfg.Enabled("gen.file.funcs") {
		return
	}
	if !s.funcRe.MatchString(raw) {
		return
	}
	s.funcs++
	if s.funcs == limit+1 {
		s.out = append(s.out, newFinding("gen.file.funcs", s.f.Display, n, Warn,
			fmt.Sprintf("function %d in this file (limit %d)", s.funcs, limit)))
	}
}

func (s *lineScan) brace(n int, stripped string) {
	code := codeOnly(stripped)
	if closes := strings.Count(code, "}"); closes > 0 {
		s.depth = max(0, s.depth-closes)
	}
	opens := strings.Count(code, "{")
	if opens == 0 {
		return
	}
	s.depth += opens
	if s.depth > s.maxDepth {
		s.maxDepth, s.maxDepthLine = s.depth, n
	}
}

func (s *lineScan) nesting() {
	if !s.cfg.Enabled("gen.nesting") || s.maxDepth <= s.cfg.Limits.Nesting {
		return
	}
	s.out = append(s.out, newFinding("gen.nesting", s.f.Display, s.maxDepthLine, Warn,
		fmt.Sprintf("nesting reaches depth %d (limit %d)", s.maxDepth, s.cfg.Limits.Nesting)))
}

func inlineComment(cfg *Config, f SourceFile, line string) bool {
	if !cfg.Style.BanInlineComments || !cfg.Enabled("gen.comment.inline") {
		return false
	}
	return trailingComment(f.Ext, line) && !IsDirective(commentText(f.Ext, line))
}

func leftoverMarker(cfg *Config, f SourceFile, line string) string {
	if !cfg.Style.BanTODO || !cfg.Enabled("gen.todo") {
		return ""
	}
	return strings.ToUpper(todoMarker(commentText(f.Ext, line)))
}
