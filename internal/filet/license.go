package filet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const licenseRule = "gen.license.required"

var licenseFiles = []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING"}

var licenseTexts = []struct{ marker, id string }{
	{"GNU AFFERO GENERAL PUBLIC LICENSE", "AGPL-3.0-only"},
	{"Permission is hereby granted, free of charge", "MIT"},
}

var cargoLicense = regexp.MustCompile(`(?m)^\s*license(\.workspace)?\s*=\s*("([^"]*)"|true)`)

// CheckLicense warns when the config root has no LICENSE, LICENSE.md,
// LICENSE.txt or COPYING, and when a package.json or Cargo.toml there leaves
// its license field empty or names a different licence than the LICENSE text.
// Only AGPL-3.0-only and MIT are recognised by their text.
func CheckLicense(cfg *Config) []Finding {
	if !cfg.Enabled(licenseRule) {
		return nil
	}
	path, text := readLicense(cfg.root)
	if path == "" {
		return []Finding{newFinding(licenseRule, DisplayPath(filepath.Join(cfg.root, "LICENSE")), 1, Warn,
			"no LICENSE, LICENSE.md, LICENSE.txt or COPYING at the repository root")}
	}
	want := licenseID(text)
	var out []Finding
	for _, m := range []struct {
		name string
		read func([]byte) (string, bool)
	}{{"package.json", packageLicense}, {"Cargo.toml", cargoManifestLicense}} {
		out = append(out, manifestLicense(filepath.Join(cfg.root, m.name), want, m.read)...)
	}
	return out
}

func readLicense(root string) (string, string) {
	for _, name := range licenseFiles {
		path := filepath.Join(root, name)
		if src, err := os.ReadFile(path); err == nil {
			return path, string(src)
		}
	}
	return "", ""
}

func licenseID(text string) string {
	for _, l := range licenseTexts {
		if strings.Contains(text, l.marker) {
			return l.id
		}
	}
	return ""
}

func manifestLicense(path, want string, read func([]byte) (string, bool)) []Finding {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	got, inherited := read(src)
	display := DisplayPath(path)
	switch {
	case inherited:
		return nil
	case got == "":
		return []Finding{newFinding(licenseRule, display, 1, Warn, "license field is not set")}
	case want != "" && !sameLicense(want, got):
		return []Finding{newFinding(licenseRule, display, 1, Warn,
			"license field is "+got+" but LICENSE is "+want)}
	}
	return nil
}

func sameLicense(want, got string) bool {
	if want == "AGPL-3.0-only" {
		return slices.Contains([]string{"AGPL-3.0-only", "AGPL-3.0-or-later", "AGPL-3.0"}, got)
	}
	return want == got
}

func packageLicense(src []byte) (string, bool) {
	var pkg struct {
		License any `json:"license"`
	}
	if json.Unmarshal(src, &pkg) != nil {
		return "", true
	}
	s, _ := pkg.License.(string)
	return strings.TrimSpace(s), false
}

func cargoManifestLicense(src []byte) (string, bool) {
	m := cargoLicense.FindSubmatch(src)
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(string(m[3])), len(m[1]) > 0
}
