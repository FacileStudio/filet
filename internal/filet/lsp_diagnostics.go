package filet

import "fmt"

// lspDiagnostic is one diagnostic as the LSP protocol describes it.
type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"`
	Code     any      `json:"code"`
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
}

type lspPosition struct {
	Line int `json:"line"`
}

// lspFinding maps one server diagnostic to a filet finding. LSP severities are
// 1=error, 2=warning, 3=info and 4=hint; the finding's rule is the mapped
// lsp.error/lsp.warn/lsp.info class and its message carries source and code.
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
		msg = "[" + code + "] " + msg
	}
	return newFinding(rule, f.Display, d.Range.Start.Line+1, sev, msg)
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
