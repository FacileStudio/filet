package filet

import (
	"encoding/json"
	"net/url"
	"path/filepath"
)

// fileURI builds a protocol-compliant file:// URI, escaping the path once so it
// can be interpolated into JSON without breaking it.
func fileURI(p string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(p)}).String()
}

// jsonInitParams is the initialize request body. processId stays null because
// filet is a single-shot client.
func jsonInitParams(root string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"processId":    nil,
		"rootUri":      fileURI(root),
		"clientInfo":   map[string]any{"name": "filet"},
		"capabilities": map[string]any{},
	})
}
