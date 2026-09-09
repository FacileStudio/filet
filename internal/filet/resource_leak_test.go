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

func TestResourceLeakNotFlaggedForBorrowedHandle(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "borrowed.go", `package borrowed

import (
	"os"
)

func writeAll(w *os.File) error {
	out := os.Stdout
	_, err := out.Write([]byte("x"))
	_ = w
	return err
}
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.leak.resource"); n != 0 {
		t.Fatalf("expected 0 go.leak.resource findings for a borrowed handle, got %d in %v", n, ruleIDs(got))
	}
}

func TestResourceLeakNotFlaggedWhenExplicitClose(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "closeexpr.go", `package closeexpr

import (
	"errors"
	"os"
)

func appendLine(path string, line []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_RDONLY, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.Write(append(line, '\n'))
	return errors.Join(werr, f.Close())
}
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.leak.resource"); n != 0 {
		t.Fatalf("expected 0 go.leak.resource findings for an explicit Close, got %d in %v", n, ruleIDs(got))
	}
}
