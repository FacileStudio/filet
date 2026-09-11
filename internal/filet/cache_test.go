package filet

import (
	"path/filepath"
	"testing"
)

func TestCacheKeyDeterministic(t *testing.T) {
	k1 := cacheKey("a", "b")
	k2 := cacheKey("a", "b")
	if k1 != k2 {
		t.Fatal("same parts must hash to the same key")
	}
	if cacheKey("a") == cacheKey("b") {
		t.Fatal("different parts must hash to different keys")
	}
	if contentHash([]byte("x")) == contentHash([]byte("y")) {
		t.Fatal("different content must hash differently")
	}
}

func TestConfigIdentityChangesOnRaw(t *testing.T) {
	a := DefaultConfig()
	b := DefaultConfig()
	if a.identity() != b.identity() {
		t.Fatal("identical defaults must share an identity")
	}
	b.raw = []byte("limits:\n  lineLength: 90\n")
	if a.identity() == b.identity() {
		t.Fatal("editing filet.yml must change the identity")
	}
}

func TestWorkerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Cache.Enabled = true
	cfg.Cache.Dir = dir
	f := source(t, "a.dart", "// TODO: fix\n")

	w := newCacheWorker(cfg, filepath.Join(dir, "pkg"))
	key := w.unitKey("file", f.Rel, contentHash(f.Src))
	if got := CheckGeneric(cfg, f); len(got) == 0 {
		t.Fatal("fixture should produce at least one finding")
	}
	w.put(key, CheckGeneric(cfg, f))
	w.write()

	w2 := newCacheWorker(cfg, filepath.Join(dir, "pkg"))
	cached, ok := w2.get(key)
	if !ok {
		t.Fatal("written finding should be loadable")
	}
	if len(cached) == 0 {
		t.Fatal("cached findings must survive a reload")
	}
	if cached[0].Severity != Warn {
		t.Fatalf("severity must be restored from level, got %d", cached[0].Severity)
	}
}

func TestCacheKeyInvalidatedByContent(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Cache.Dir = dir
	w := newCacheWorker(cfg, filepath.Join(dir, "pkg"))

	a := source(t, "a.rb", "todo()\n")
	b := source(t, "b.rb", "called()\n")
	if w.unitKey("file", a.Rel, contentHash(a.Src)) == w.unitKey("file", b.Rel, contentHash(b.Src)) {
		t.Fatal("changed content must change the key")
	}
}

func TestCacheDisabledIsNoop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Cache.Enabled = false
	if w := newCacheWorker(cfg, "."); w.store != nil {
		t.Fatal("disabled cache must not open a store")
	}
}
