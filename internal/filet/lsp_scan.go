package filet

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// lspGroup is the set of files sharing one server recipe.
type lspGroup struct {
	srv   Server
	files []SourceFile
}

// lspRecipeKey identifies a server recipe by binary and argument list.
func lspRecipeKey(srv Server) string {
	return srv.Server + "\x00" + strings.Join(srv.Args, "\x00")
}

// lspURI is the document URI a server keys diagnostics on, stable with the one
// filet sent in didOpen.
func lspURI(s *lsp, f SourceFile) string {
	return fileURI(filepath.Join(s.rootAbs, f.Rel))
}

// findServerBin resolves a server on PATH, then in the standard mason install
// dir, returning a runnable path or "".
func findServerBin(name string) string {
	if _, err := exec.LookPath(name); err == nil {
		return name
	}
	cand := filepath.Join(masonBinDir(), name)
	if _, err := os.Stat(cand); err == nil {
		return cand
	}
	return ""
}

func masonBinDir() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "nvim", "mason", "bin")
}

// lspUnavailable emits the honest, visible finding for a language filet could
// not serve through its server.
func lspUnavailable(f SourceFile, msg string) Finding {
	return newFinding("lsp.unavailable", f.Display, 0, Info, msg)
}
