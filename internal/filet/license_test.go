package filet

import (
	"path/filepath"
	"testing"
)

const mitText = "MIT License\n\nPermission is hereby granted, free of charge, to any person obtaining a copy\n"

const agplText = "GNU AFFERO GENERAL PUBLIC LICENSE\nVersion 3, 19 November 2007\n"

func licenseFindings(t *testing.T, files map[string]string) []Finding {
	t.Helper()
	if _, ok := files["filet.yml"]; !ok {
		files["filet.yml"] = "style:\n  requireLicense: true\n"
	}
	proj := t.TempDir()
	for name, body := range files {
		if err := writeTemp(filepath.Join(proj, name), body); err != nil {
			t.Fatal(err)
		}
	}
	cfg, _, err := LoadConfig(proj)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Analyze(cfg, proj)
	if err != nil {
		t.Fatal(err)
	}
	var out []Finding
	for _, f := range report.Findings {
		if f.Rule == licenseRule {
			out = append(out, f)
		}
	}
	return out
}

func TestLicensePresentWithMatchingManifests(t *testing.T) {
	cases := map[string]map[string]string{
		"mit":     {"LICENSE": mitText, "package.json": `{"license": "MIT"}`},
		"agpl":    {"LICENSE.md": agplText, "Cargo.toml": "[package]\nname = \"x\"\nlicense = \"AGPL-3.0-only\"\n"},
		"bare":    {"COPYING": "custom terms\n"},
		"inherit": {"LICENSE": mitText, "Cargo.toml": "[package]\nlicense.workspace = true\n"},
	}
	for name, files := range cases {
		if got := licenseFindings(t, files); len(got) != 0 {
			t.Errorf("%s: unexpected findings %+v", name, got)
		}
	}
}

func TestLicenseMissingWarns(t *testing.T) {
	got := licenseFindings(t, map[string]string{"main.go": "package main\n"})
	if len(got) != 1 || got[0].Severity != Warn || filepath.Base(got[0].File) != "LICENSE" {
		t.Fatalf("want one warning on LICENSE, got %+v", got)
	}
}

func TestLicenseManifestMismatchOrUnsetWarns(t *testing.T) {
	got := licenseFindings(t, map[string]string{
		"LICENSE":      agplText,
		"package.json": `{"license": "MIT"}`,
		"Cargo.toml":   "[package]\nname = \"x\"\n",
	})
	want := map[string]string{
		"package.json": "license field is MIT but LICENSE is AGPL-3.0-only",
		"Cargo.toml":   "license field is not set",
	}
	if len(got) != len(want) {
		t.Fatalf("want %d findings, got %+v", len(want), got)
	}
	for _, f := range got {
		if want[filepath.Base(f.File)] != f.Message || f.Severity != Warn {
			t.Errorf("unexpected finding %+v", f)
		}
	}
}

func TestLicenseSilentUnlessEnabled(t *testing.T) {
	for name, cfg := range map[string]string{"default": "kind: cli\n", "facile": "preset: facile\n"} {
		if got := licenseFindings(t, map[string]string{"filet.yml": cfg, "main.go": "package main\n"}); len(got) != 0 {
			t.Errorf("%s: rule fired without style.requireLicense: %+v", name, got)
		}
	}
}
