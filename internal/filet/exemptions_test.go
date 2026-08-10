package filet

import "testing"

// The rules below all fire on a shape the language does not offer an
// alternative to, which is the bar an exemption has to clear: not "this is
// noisy" but "there is no other way to write it".

// TestGroupDocFormOnlyAppliesToItsOwnDeclaration pins both halves: one comment
// above a `type ( ... )` block documents every name in it and cannot open with
// all of them, while a comment above a lone declaration is that declaration's
// own and the form rule still applies to it. The second half is the regression
// risk — Go's parser hangs a lone type's comment off the GenDecl, not the spec,
// so "came from the group" is not on its own a reason to skip the check.
func TestGroupDocFormOnlyAppliesToItsOwnDeclaration(t *testing.T) {
	cfg := DefaultConfig()
	body := `package alias

// The registry types, aliased so callers never import the renderer.
type (
	Response = int
	Module   = int
)

// A single type that does not open with its own name.
type Lonely struct{}
`

	got := CheckGo(cfg, source(t, "alias.go", body))
	if n := countRule(got, "go.doc.form"); n != 1 {
		t.Fatalf("want only Lonely flagged, got %d findings: %v", n, ruleIDs(got))
	}
	if n := countRule(got, "go.doc.missing"); n != 0 {
		t.Fatalf("the group comment documents every alias, got %d missing", n)
	}
}

func TestEmbeddedFilesystemsAreNotSharedMutableState(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Style.BanGlobalMutable = true
	body := `package assets

import "embed"

// FS holds the migrations.
//
//go:embed *.sql
var FS embed.FS

var (
	//go:embed banner.txt
	Banner string

	Registry = map[string]int{}
)
`

	got := CheckGo(cfg, source(t, "assets.go", body))
	if n := countRule(got, "go.global.mutable"); n != 1 {
		t.Fatalf("want only Registry flagged, got %d findings: %v", n, ruleIDs(got))
	}
}
