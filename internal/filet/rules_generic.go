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
	//
	// The marker is chosen per language. `#` only starts a comment in a braceless
	// language (.py/.rb/.sh); everywhere else a leading `#` is live code — a
	// TypeScript/Svelte private field or a Go preprocessor-less token — so only
	// `//` is ever a comment there.
	commentedCodeCC    = regexp.MustCompile(`^\s*// ?(` + commentedCodeBody + `)`)
	commentedCodeSharp = regexp.MustCompile(`^\s*(//|#) ?(` + commentedCodeBody + `)`)

	// commentedCodeBody is the statement-shaped body shared by both markers.
	commentedCodeBody = `(func|def|const|let|var|import|export)(\s|$)` +
		// A package clause is exactly two tokens. Without that anchor, any
		// wrapped sentence in a package doc whose line happens to begin with
		// the word "package" reads as commented-out code.
		`|package\s+\w+\s*$` +
		`|(if|for|while|switch|case|class|else|elif)(\s|$).*[(){};=]` +
		`|return\s+\S+\s*$` +
		`|(break|continue|fallthrough|pass)\s*;?\s*$` +
		// An assignment is a statement only when the value runs to the end of
		// the line (or a terminal `;`) — gofmt omits the semicolon a Go
		// statement would otherwise carry. When prose continues after the
		// value ("avatar_source = 'upload' quietly drops"), it is a sentence,
		// not commented-out code.
		`|[\w.]+\s*[-+*/|&^]?:?=\s*[^=\s;][^\s;]*\s*;?\s*$` +
		`|[\w.]+\([^;]*\)\s*[;{]?\s*$` +
		`|[});]\s*$`
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

// brace tracks block depth one character at a time, and scores the line on the
// depth that is still open once the line ends.
//
// Two things fall out of that, both deliberate. Counting the braces on a line
// and applying every close before every open gets the wrong answer whenever a
// line closes deeper than it is and then reopens: the clamp at zero swallows
// the underflow and the opens land on top of the floor, leaving the counter too
// high for the rest of the file — which is what the Go table-driven idiom
// `}{{"a"}, {"b"}}` does. And a brace pair that opens and closes on the same
// line nests nothing a reader has to hold in their head, so `[]string{"a"}` or
// `if err != nil { return }` must not score: this rule is about blocks, and a
// composite literal is not one.
func (s *lineScan) brace(n int, stripped string) {
	for _, c := range codeOnly(stripped) {
		switch c {
		case '{':
			s.depth++
		case '}':
			s.depth = max(0, s.depth-1)
		}
	}
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
