package filet

import "testing"

func TestNilerrNotFlaggedForEqualitySentinelGate(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "sent.go", `package sent

const ErrNoRows = (error)

func Load(id string) (*Item, error) {
	item, err := fetch(id)
	if err != nil {
		if err == ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func fetch(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 0 {
		t.Fatalf("expected 0 go.err.nilerr findings for an equality sentinel gate, got %d in %v", n, ruleIDs(got))
	}
}

func TestNilerrNotFlaggedForErrorsIsSentinelGate(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "is.go", `package is

const ErrNoRows = (error)

func Load(id string) (*Item, error) {
	item, err := fetch(id)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func fetch(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 0 {
		t.Fatalf("expected 0 go.err.nilerr findings for an errors.Is sentinel gate, got %d in %v", n, ruleIDs(got))
	}
}

func TestNilerrStillFlaggedInSentinelElse(t *testing.T) {
	cfg := DefaultConfig()
	f := source(t, "else.go", `package elset

const ErrNoRows = (error)

func Load(id string) (*Item, error) {
	item, err := fetch(id)
	if err != nil {
		if err == ErrNoRows {
			return nil, nil
		} else {
			return nil, nil
		}
	}
	return item, nil
}

func fetch(id string) (*Item, error) { return nil, nil }
`)
	got := CheckGo(cfg, f)
	if n := countRule(got, "go.err.nilerr"); n != 1 {
		t.Fatalf("expected 1 go.err.nilerr finding in the sentinel else, got %d in %v", n, ruleIDs(got))
	}
}
