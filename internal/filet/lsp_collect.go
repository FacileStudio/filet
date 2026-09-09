package filet

import (
	"encoding/json"
)

// lspCollect opens every file with the server, then collects the diagnostics it
// pushes in textDocument/publishDiagnostics until every open file has reported
// or the budget runs out. Files that never report are listed as missing, but
// the diagnostics that did arrive are still returned so a slow file cannot sink
// the whole group.
func lspCollect(s *lsp, group []SourceFile) (map[string][]lspDiagnostic, []string) {
	uriToRel := make(map[string]string, len(group))
	left := make(map[string]bool, len(group))
	for _, f := range group {
		uri := lspURI(s, f)
		uriToRel[uri] = f.Rel
		left[f.Rel] = true
		if openWithServer(s, uri, f) != nil {
			return nil, []string{f.Rel}
		}
	}
	got := map[string][]lspDiagnostic{}
	s.bounded(lspCollectTimeout, func() error {
		return consumeDiagnostics(s, left, got, uriToRel)
	})
	if len(left) == 0 {
		return got, nil
	}
	return got, missingRels(left)
}

// collectedDiags is one publishDiagnostics decoupled from the wire envelope.
type collectedDiags struct {
	uri   string
	items []lspDiagnostic
}

// parseDiags decodes a publishDiagnostics notification, or nil for any other
// frame.
func parseDiags(body []byte) *collectedDiags {
	var m publishDiags
	if json.Unmarshal(body, &m) != nil || m.Method != "textDocument/publishDiagnostics" {
		return nil
	}
	if m.Params.Diagnostics == nil {
		m.Params.Diagnostics = []lspDiagnostic{}
	}
	return &collectedDiags{uri: m.Params.URI, items: m.Params.Diagnostics}
}

// consumeDiagnostics reads server frames, filling got for each file that
// reports until every open file has or the pipe closes.
func consumeDiagnostics(s *lsp, left map[string]bool, got map[string][]lspDiagnostic, uriToRel map[string]string) error {
	for len(left) > 0 {
		body, err := readFrame(s.reader)
		if err != nil {
			return err
		}
		diags := parseDiags(body)
		if diags == nil {
			continue
		}
		rel, ok := uriToRel[diags.uri]
		if !ok || !left[rel] {
			continue
		}
		got[rel] = diags.items
		delete(left, rel)
	}
	return nil
}

// missingRels lists the files that never reported diagnostics.
func missingRels(left map[string]bool) []string {
	var missing []string
	for rel := range left {
		missing = append(missing, rel)
	}
	return missing
}

// openWithServer tells the server a file is open so it starts pushing
// diagnostics for it.
func openWithServer(s *lsp, uri string, f SourceFile) error {
	params, _ := json.Marshal(didOpenParams{TextDocument: textDocumentInfo{
		URI:        uri,
		LanguageID: lspLanguage(f.Ext),
		Version:    1,
		Text:       string(f.Src),
	}})
	return lspNotify(s, "textDocument/didOpen", params)
}

// lspLanguage maps a file extension to an LSP languageId.
func lspLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".ts", ".tsx", ".js", ".jsx":
		return "typescript"
	case ".svelte":
		return "svelte"
	default:
		return "plaintext"
	}
}

// didOpenParams is the body of textDocument/didOpen.
type didOpenParams struct {
	TextDocument textDocumentInfo `json:"textDocument"`
}

type textDocumentInfo struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// publishDiags is a textDocument/publishDiagnostics notification.
type publishDiags struct {
	Method string           `json:"method"`
	Params publishDiagsBody `json:"params"`
}

type publishDiagsBody struct {
	URI         string          `json:"uri"`
	Diagnostics []lspDiagnostic `json:"diagnostics"`
}
