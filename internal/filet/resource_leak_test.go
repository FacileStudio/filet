package filet

import (
	"testing"
)

func TestResourceLeakCaught(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "leak.go", `package leak

import (
	"os"
)

func handler() {
	f, _ := os.Open("file.txt")
	_ = f
}
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.leak.resource"); n != 1 {
		t.Fatalf("expected 1 go.leak.resource finding, got %d in %v", n, ruleIDs(got))
	}
}

func TestResourceLeakNotFlaggedWhenDeferredClose(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "noleak.go", `package noleak

import (
	"os"
)

func handler() {
	f, _ := os.Open("file.txt")
	defer f.Close()
	_ = f
}
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.leak.resource"); n != 0 {
		t.Fatalf("expected 0 go.leak.resource findings, got %d in %v", n, ruleIDs(got))
	}
}

func TestResourceLeakNotFlaggedWhenDeferredAnonymousClose(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "appendline.go", `package appendline

import (
	"os"
)

func appendLine(path string, line []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.leak.resource"); n != 0 {
		t.Fatalf("expected 0 go.leak.resource findings (deferred in anonymous func), got %d in %v", n, ruleIDs(got))
	}
}

func TestResourceLeakNotFlaggedInTestFiles(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "handler_test.go", `package leak_test

import (
	"os"
)

func TestHandler() {
	f, _ := os.Open("file.txt")
	_ = f
}
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.leak.resource"); n != 0 {
		t.Fatalf("expected 0 go.leak.resource findings in test file, got %d in %v", n, ruleIDs(got))
	}
}
