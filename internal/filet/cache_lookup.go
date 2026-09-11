package filet

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// checkFileCached returns a file's generic and tree-sitter findings, reusing a
// cached copy keyed by content hash when the config and bytes are unchanged.
// Go files get the generic rules here and their package-scoped rules via
// checkGoCached; the two never overlap in rules or findings.
func checkFileCached(w *cacheWorker, cfg *Config, f SourceFile) []Finding {
	key := w.unitKey("file", f.Rel, contentHash(f.Src))
	if cached, ok := w.get(key); ok {
		return cached
	}
	out := CheckGeneric(cfg, f)
	if cfg.UsesTreesitter(f.Ext) {
		out = append(out, CheckTree(cfg, f)...)
	}
	w.put(key, out)
	return out
}

// checkGoCached checks every Go directory as one package-scoped unit, reusing
// per-directory findings when none of the directory's files changed. A change
// in one file re-checks its whole directory, never a sibling directory.
func checkGoCached(w *cacheWorker, cfg *Config, files []SourceFile) []Finding {
	var out []Finding
	for dir, group := range groupByDir(files) {
		key := w.unitKey("go", dir, dirHash(group))
		if found, ok := w.get(key); ok {
			out = append(out, found...)
			continue
		}
		found := checkGoDir(cfg, group)
		w.put(key, found)
		out = append(out, found...)
	}
	return out
}

// groupByDir buckets go files by their directory, deterministically ordered.
func groupByDir(files []SourceFile) map[string][]SourceFile {
	byDir := map[string][]SourceFile{}
	for _, f := range files {
		if f.Ext != ".go" {
			continue
		}
		d := filepath.Dir(f.Path)
		byDir[d] = append(byDir[d], f)
	}
	return byDir
}

// dirHash hashes a directory's files as one unit: sorted relative paths plus
// their content hashes, so any change to any member invalidates the whole dir.
func dirHash(group []SourceFile) string {
	sorted := append([]SourceFile(nil), group...)
	slices.SortFunc(sorted, func(a, b SourceFile) int { return cmp.Compare(a.Rel, b.Rel) })
	var parts []string
	for _, f := range sorted {
		parts = append(parts, f.Rel, contentHash(f.Src))
	}
	return cacheKey(parts...)
}

// checkRestCached runs the architecture and LSP tiers together, so a run where
// no file content changed skips re-reading them entirely. The key covers the
// whole file set, their hashes and the language servers' locations, so a new
// repo layout or a server upgrade invalidates it exactly when rerunning would
// produce a different verdict.
func checkRestCached(w *cacheWorker, cfg *Config, files []SourceFile) []Finding {
	key := w.unitKey("rest", runHash(files), lspBinHash(cfg))
	if found, ok := w.get(key); ok {
		return found
	}
	out := CheckArchitecture(cfg, files)
	out = append(out, CheckLSP(cfg, files)...)
	w.put(key, out)
	return out
}

// runHash hashes the whole scan in stable order: every file's relative path and
// content hash, plus the file count so adding or removing a file registers.
func runHash(files []SourceFile) string {
	sorted := append([]SourceFile(nil), files...)
	slices.SortFunc(sorted, func(a, b SourceFile) int { return cmp.Compare(a.Rel, b.Rel) })
	var parts []string
	for _, f := range sorted {
		parts = append(parts, f.Rel, contentHash(f.Src))
	}
	parts = append(parts, fmt.Sprint(len(files)))
	return cacheKey(parts...)
}

// lspBinHash fingerprints every configured language server's resolved binary, so
// a server that is moved or upgraded invalidates cached LSP findings even when
// no source file changed since the last run. Resolving misses the version, but
// a server installed elsewhere or swapped out is caught, which is the common
// cause of a stale diagnostic.
func lspBinHash(cfg *Config) string {
	if !cfg.LSP.Enabled {
		return cacheKey("lsp-off")
	}
	bins := make([]string, 0, len(cfg.LSP.Servers))
	seen := map[string]bool{}
	for _, srv := range cfg.LSP.Servers {
		bin := findServerBin(srv.Server)
		if bin == "" {
			continue
		}
		if s, err := os.Stat(bin); err == nil {
			bin = fmt.Sprintf("%s@%d", bin, s.Size())
		}
		if !seen[bin] {
			seen[bin] = true
			bins = append(bins, bin)
		}
	}
	slices.Sort(bins)
	return cacheKey("lsp", cacheKey(bins...))
}
