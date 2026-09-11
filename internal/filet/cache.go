package filet

import (
	"encoding/hex"
	"hash/fnv"
)

// cacheWorker holds the store for one analyze run and decides, per unit,
// whether cached findings are reusable. Every key seeds on the config identity,
// so changing filet.yml or shipping a new version invalidates all entries and
// the worker simply recomputes — the cache can never serve a stale finding.
type cacheWorker struct {
	id     string
	cfg    *Config
	target string
	store  map[string][]Finding
}

func newCacheWorker(cfg *Config, target string) *cacheWorker {
	w := &cacheWorker{id: cfg.identity(), cfg: cfg, target: target}
	if !cfg.Cache.Enabled {
		return w
	}
	w.store = loadStore(cfg, w.id, target)
	return w
}

// unitKey hashes the parts that identify the inspected unit.
func (w *cacheWorker) unitKey(scope string, parts ...string) string {
	return w.id + ":" + cacheKey(append([]string{scope}, parts...)...)
}

// get returns the cached findings for a key and whether the key was present.
func (w *cacheWorker) get(key string) ([]Finding, bool) {
	if w.store == nil {
		return nil, false
	}
	f, ok := w.store[key]
	if ok {
		restoreSeverity(f)
	}
	return f, ok
}

// put remembers findings for a key.
func (w *cacheWorker) put(key string, f []Finding) {
	if w.store == nil {
		return
	}
	w.store[key] = f
}

// write persists the store when the cache is active.
func (w *cacheWorker) write() {
	if w.store == nil {
		return
	}
	saveStore(w.cfg, w.id, w.target, w.store)
}

// identity names the config that seeds every key: the tool version plus the raw
// config bytes, so any filet.yml edit or new release changes every key and
// silently invalidates whatever the store held.
func (c *Config) identity() string {
	return cacheKey("filet", configSchemaVersion, string(c.raw))
}

// configSchemaVersion distinguishes cache entries across rule or format changes.
// Bump it whenever a rule's behaviour or the cached finding shape changes while
// the on-disk schema stays otherwise compatible; the version is part of every
// key, so a bump silently invalidates the whole store instead of serving
// findings a different rule set would not produce.
const configSchemaVersion = "1"

// cacheKey hashes its parts into a deterministic cache lookup key.
func cacheKey(parts ...string) string {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// restoreSeverity back-fills the Severity field from the persisted Level label,
// because finding.Severity is tagged json:"-" and does not round-trip.
func restoreSeverity(f []Finding) {
	for i := range f {
		if f[i].Level != "" {
			f[i].Severity = ParseSeverity(f[i].Level)
		}
	}
}
