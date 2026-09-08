package filet

import (
	"testing"
)

func TestNilerrCaught(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "nilerr.go", `package nilerr

func Get(id string) (*Item, error) {
	item, err := fetch(id)
	if err != nil {
		return nil, nil
	}
	return item, nil
}

func fetch(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 1 {
		t.Fatalf("expected 1 go.err.nilerr finding, got %d in %v", n, ruleIDs(got))
	}
}

func TestNilerrNotFlaggedWhenReturningError(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "good.go", `package good

func Get(id string) (*Item, error) {
	item, err := fetch(id)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func fetch(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 0 {
		t.Fatalf("expected 0 go.err.nilerr findings, got %d in %v", n, ruleIDs(got))
	}
}

func TestNilerrNotFlaggedWhenNoErrorReturn(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "noret.go", `package noret

func Get(id string) {
	item, err := fetch(id)
	if err != nil {
		return
	}
	_ = item
}

func fetch(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 0 {
		t.Fatalf("expected 0 go.err.nilerr findings, got %d in %v", n, ruleIDs(got))
	}
}

func TestNilerrNotFlaggedInTestFunctions(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "nilerr_test.go", `package nilerr_test

func TestGet(t *testing.T) {
	item, err := fetch("1")
	if err != nil {
		t.Fatal("unexpected error")
	}
	if item == nil {
		return nil, nil
	}
	_ = item
}

func fetch(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 0 {
		t.Fatalf("expected 0 go.err.nilerr findings in test, got %d in %v", n, ruleIDs(got))
	}
}

func TestNilerrReversedComparison(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "rev.go", `package rev

func Get(id string) (*Item, error) {
	item, err := fetch(id)
	if nil != err {
		return nil, nil
	}
	return item, nil
}

func fetch(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 1 {
		t.Fatalf("expected 1 go.err.nilerr finding for reversed nil!=err, got %d in %v", n, ruleIDs(got))
	}
}

func TestNilerrMultipleBlocks(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "multi.go", `package multi

func Get(id string) (*Item, error) {
	item, err := fetch(id)
	if err != nil {
		return nil, nil
	}
	resp, err := fetchResp(id)
	if err != nil {
		return nil, nil
	}
	return item, resp
}

func fetch(id string) (*Item, error)       { return nil, nil }
func fetchResp(id string) (*Item, error)  { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 2 {
		t.Fatalf("expected 2 go.err.nilerr findings, got %d in %v", n, ruleIDs(got))
	}
}

func TestNilerrSafeMultipleValues(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "safe.go", `package safe

func Get(id string) (string, *Item, error) {
	item, err := fetch(id)
	if err != nil {
		return "", nil, err
	}
	resp, err := fetchResp(id)
	if err != nil {
		return "", nil, err
	}
	return "ok", item, nil
}

func fetch(id string) (*Item, error)      { return nil, nil }
func fetchResp(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 0 {
		t.Fatalf("expected 0 go.err.nilerr findings, got %d in %v", n, ruleIDs(got))
	}
}
