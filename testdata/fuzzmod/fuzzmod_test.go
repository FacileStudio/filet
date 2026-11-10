package fuzzmod

import "testing"

func FuzzPasses(f *testing.F) {
	f.Add("ab")
	f.Fuzz(func(t *testing.T, s string) {
		_ = len(s)
	})
}

func FuzzCrashes(f *testing.F) {
	f.Add("ab")
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 3 {
			t.Fatalf("input longer than three bytes: %q", s)
		}
	})
}

type seeds struct{}

func (seeds) FuzzMethod(f *testing.F) {}
