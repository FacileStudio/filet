package filet

import "fmt"

// lspDiagnostic is one diagnostic as the LSP protocol describes it. Server
// diagnostics carry a lot more than the message line: the start character pins
// the exact column, codeDescription links the rule's docs, and tags mark
// deprecated or unnecessary code. lspFinding folds all of it into the finding.
type lspDiagnostic struct {
	Range           lspRange            `json:"range"`
	Severity        int                 `json:"severity"`
	Code            any                 `json:"code"`
	Source          string              `json:"source"`
	Message         string              `json:"message"`
	Tags            []int               `json:"tags"`
	CodeDescription *lspCodeDescription `json:"codeDescription"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
}

type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// lspCodeDescription carries the optional "how to fix / learn more" link a
// server attaches to a diagnostic code (TypeScript, clangd and Pyright set it).
type lspCodeDescription struct {
	Href string `json:"href"`
}

// lspFinding maps one server diagnostic to a filet finding. LSP severities are
// 1=error, 2=warning, 3=info and 4=hint; the finding's rule is the mapped
// lsp.error/lsp.warn/lsp.info class. The message carries source and code with
// no decorative brackets, plus the deprecated/unnecessary tag when the server
// flags one. A codeDescription href becomes the finding's Docs link so a
// terminal renderer can style it muted instead of burying it in the message.
func lspFinding(f SourceFile, d lspDiagnostic) Finding {
	sev := Info
	if d.Severity == 2 {
		sev = Warn
	}
	if d.Severity == 1 {
		sev = Error
	}
	rule := "lsp.info"
	if sev == Warn {
		rule = "lsp.warn"
	}
	if sev == Error {
		rule = "lsp.error"
	}
	msg := d.Message
	if d.Source != "" {
		msg = d.Source + ": " + msg
	}
	if code := codeText(d.Code); code != "" {
		msg = code + " " + msg
	}
	msg = appendTagMarkers(msg, d.Tags)
	fd := newFinding(rule, f.Display, d.Range.Start.Line+1, sev, msg)
	fd.Column = d.Range.Start.Character + 1
	if d.CodeDescription != nil && d.CodeDescription.Href != "" {
		fd.Docs = d.CodeDescription.Href
	}
	return fd
}

// appendTagMarkers suffixes deprecated/unnecessary markers a server flags with
// LSP tag 1 (unnecessary) and 2 (deprecated), so a dead or doomed line says so.
func appendTagMarkers(msg string, tags []int) string {
	if tags == nil {
		return msg
	}
	for _, t := range tags {
		switch t {
		case 1:
			msg = msg + " (unnecessary)"
		case 2:
			msg = msg + " (deprecated)"
		}
	}
	return msg
}

// codeText renders a diagnostic code for the message line; codes are often
// numbers that arrive as JSON numbers, so the union is handled here.
func codeText(code any) string {
	switch v := code.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%g", v)
	}
	return ""
}
