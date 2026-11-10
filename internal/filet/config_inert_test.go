package filet

import (
	"slices"
	"strings"
	"testing"
)

func inertKeys(t *testing.T, body string) []string {
	t.Helper()
	cfg := loadFixture(t, body)
	var keys []string
	for _, f := range InertKeys(cfg, cfg.Root()) {
		if f.Rule != "gen.config.inert" || f.Severity != Warn {
			t.Fatalf("unexpected finding %+v", f)
		}
		key, _, _ := strings.Cut(f.Message, " ")
		keys = append(keys, key)
	}
	return keys
}

func TestInertKeysOnATypeScriptOnlyConfig(t *testing.T) {
	body := "extensions: [.ts]\nlimits:\n  structFields: 12\n  funcLines: 30\n  fileLines: 250\n" +
		"style:\n  banGlobalMutable: true\n  banInit: false\n  banTODO: true\n"
	got := inertKeys(t, body)
	want := []string{"limits.funcLines", "limits.structFields", "style.banGlobalMutable"}
	if !slices.Equal(got, want) {
		t.Fatalf("inert keys = %v, want %v", got, want)
	}
}

func TestInertKeysPointAtTheKeyLine(t *testing.T) {
	cfg := loadFixture(t, "extensions: [.ts]\nstyle:\n  banInit: true\n")
	found := InertKeys(cfg, cfg.Root())
	if len(found) != 1 || found[0].Line != 3 {
		t.Fatalf("want one finding on line 3, got %+v", found)
	}
}

func TestInertKeysStaySilentWhenAKeyCanFire(t *testing.T) {
	for _, body := range []string{
		"extensions: [.go]\nlimits:\n  structFields: 12\nstyle:\n  banInit: true\n",
		"extensions: [.rs]\nlimits:\n  funcLines: 30\n  funcsPerFile: 8\n",
		"extensions: [.ts]\n",
		"extensions: [.ts]\npreset: facile\n",
	} {
		cfg := loadFixture(t, body)
		if found := InertKeys(cfg, cfg.Root()); len(found) != 0 {
			t.Errorf("%q: want no inert key, got %+v", body, found)
		}
	}
}

func TestInertKeysSkipAScopedRun(t *testing.T) {
	cfg := loadFixture(t, "extensions: [.ts]\nstyle:\n  banInit: true\n")
	if found := InertKeys(cfg, cfg.Root()+"/src"); len(found) != 0 {
		t.Fatalf("a scoped run must not repeat config warnings, got %+v", found)
	}
}
