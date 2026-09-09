package filet

import "strings"

// defaultRecipes maps a file extension to the language server filet uses for it
// by default. Every entry can be overridden or extended in filet.yml's
// lsp.servers. A recipe's server is resolved via PATH, then the standard mason
// install dir, so a Neovim-managed LSP fleet is found with zero config.
func defaultRecipes() map[string]Server {
	return map[string]Server{
		".go":     {Server: "gopls"},
		".rs":     {Server: "rust-analyzer"},
		".ts":     {Server: "typescript-language-server", Args: []string{"--stdio"}},
		".tsx":    {Server: "typescript-language-server", Args: []string{"--stdio"}},
		".js":     {Server: "typescript-language-server", Args: []string{"--stdio"}},
		".jsx":    {Server: "typescript-language-server", Args: []string{"--stdio"}},
		".svelte": {Server: "svelte-language-server", Args: []string{"--stdio"}},
	}
}

// defaultLSP returns the on-by-default language-server tier. Fail stays off so
// lsp.* findings never gate a run unless a repo opts in.
func defaultLSP() LSP {
	return LSP{Enabled: true, Fail: false, Servers: defaultRecipes()}
}

// CheckLSP runs the language-server tier over the scanned files: one server per
// recipe, every diagnostic folded into an lsp.* finding. A recipe whose server
// cannot be resolved or fails to start or speak surfaces a single lsp.unavailable
// info finding — never a hard crash, never silence. When lsp.fail is set every
// lsp.* finding is promoted to error so the tier becomes part of the repo's gate.
func CheckLSP(cfg *Config, files []SourceFile) []Finding {
	if !cfg.LSP.Enabled || len(files) == 0 {
		return nil
	}
	var out []Finding
	for _, grp := range lspGroups(cfg, files) {
		out = append(out, checkLSPExtension(cfg, grp.files, grp.srv)...)
	}
	if cfg.LSP.Fail {
		for i := range out {
			out[i].Severity = Error
			out[i].Level = "error"
		}
	}
	return keepEnabled(cfg, out)
}

// lspGroups buckets files by server recipe, one group per distinct binary and
// argument list, so a single type server serves all of .ts/.tsx/.js/.jsx.
func lspGroups(cfg *Config, files []SourceFile) []lspGroup {
	idx := map[string]int{}
	var groups []lspGroup
	for _, f := range files {
		srv, ok := cfg.LSP.Servers[f.Ext]
		if !ok || srv.Server == "" {
			continue
		}
		key := lspRecipeKey(srv)
		if i, ok := idx[key]; ok {
			groups[i].files = append(groups[i].files, f)
			continue
		}
		idx[key] = len(groups)
		groups = append(groups, lspGroup{srv: srv, files: []SourceFile{f}})
	}
	return groups
}

// checkLSPExtension runs one server over its file group and reports the leaks.
func checkLSPExtension(cfg *Config, group []SourceFile, srv Server) []Finding {
	bin := findServerBin(srv.Server)
	if bin == "" {
		return lspUnable(group, srv.Server+" is not installed; install it or set a server in lsp.servers")
	}
	s, err := lspStart(bin, srv.Args, cfg.root)
	if err != nil {
		return lspUnable(group, srv.Server+": "+err.Error())
	}
	got, missing := lspCollect(s, group)
	s.close()
	out := lspFileDiagnostics(group, got)
	if len(missing) > 0 {
		out = append(out, lspUnavailable(group[0],
			srv.Server+": no diagnostics for "+strings.Join(missing, ", ")))
	}
	return out
}

// lspFileDiagnostics maps every collected diagnostic to an lsp.* finding.
func lspFileDiagnostics(group []SourceFile, got map[string][]lspDiagnostic) []Finding {
	var out []Finding
	for _, f := range group {
		for _, d := range got[f.Rel] {
			out = append(out, lspFinding(f, d))
		}
	}
	return out
}

func lspUnable(group []SourceFile, msg string) []Finding {
	return []Finding{lspUnavailable(group[0], msg)}
}

// keepEnabled drops findings whose rule is in the repo's disabled list.
func keepEnabled(cfg *Config, findings []Finding) []Finding {
	kept := findings[:0]
	for _, f := range findings {
		if cfg.Enabled(f.Rule) {
			kept = append(kept, f)
		}
	}
	return kept
}
