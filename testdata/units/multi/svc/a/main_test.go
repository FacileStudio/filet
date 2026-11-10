package main

import "testing"

func FuzzAnswer(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string) {})
}
