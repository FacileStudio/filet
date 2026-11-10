package filet

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func loadFixture(t *testing.T, body string) *Config {
	t.Helper()
	dir := t.TempDir()
	if err := writeTemp(filepath.Join(dir, "filet.yml"), body); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestFacilePresetEqualsTheSondeConfig(t *testing.T) {
	explicit := loadFixture(t, readFixture(t, "sonde.filet.yml"))
	preset := loadFixture(t, "preset: facile\n")
	pairs := map[string][2]any{
		"limits":        {explicit.Limits, preset.Limits},
		"style":         {explicit.Style, preset.Style},
		"ignore":        {explicit.Ignore, preset.Ignore},
		"extensions":    {explicit.Extensions, preset.Extensions},
		"requiredDirs":  {explicit.Architecture.RequiredDirs, preset.Architecture.RequiredDirs},
		"requiredFiles": {explicit.Architecture.RequiredFiles, preset.Architecture.RequiredFiles},
		"disabled":      {explicit.Disabled, preset.Disabled},
		"failOn":        {explicit.FailOn, preset.FailOn},
	}
	for key, p := range pairs {
		if !reflect.DeepEqual(p[0], p[1]) {
			t.Errorf("%s: Sonde says %v, preset facile says %v", key, p[0], p[1])
		}
	}
}

func TestFacilePresetExceptionRestatesTheGlob(t *testing.T) {
	explicit := loadFixture(t, readFixture(t, "capsule.filet.yml"))
	preset := loadFixture(t, "preset: facile\narchitecture:\n  requiredFiles:\n"+
		"    \"apps/api/modules/*\": [router.go]\n    \"apps/api/modules/docs\": []\n")
	if !reflect.DeepEqual(explicit.Architecture.RequiredFiles, preset.Architecture.RequiredFiles) {
		t.Fatalf("Capsule says %v, preset plus exception says %v",
			explicit.Architecture.RequiredFiles, preset.Architecture.RequiredFiles)
	}
}
