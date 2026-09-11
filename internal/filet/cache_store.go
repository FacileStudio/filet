package filet

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// cacheJSON is the on-disk layout: a mapping from a cache key to the findings
// that inspection of that key's unit produced.
type cacheJSON struct {
	Entries map[string][]Finding `json:"entries"`
}

// cacheRootDir resolves where the cache lives: the configured dir, else
// $XDG_CACHE_HOME/filet, else ~/.cache/filet.
func cacheRootDir(cfg *Config) (string, error) {
	if cfg.Cache.Dir != "" {
		return cfg.Cache.Dir, nil
	}
	if dir := os.Getenv("XDG_CACHE_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, "filet"), nil
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".cache", "filet"), nil
	}
	return "", os.ErrNotExist
}

// cachePath returns the store file for a target. One file per target keeps two
// unrelated projects from clobbering each other in a shared cache dir.
func cachePath(cfg *Config, target string) (string, error) {
	root, err := cacheRootDir(cfg)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, cacheKey(abs)+".json"), nil
}

// loadStore reads and prunes one target's store, keeping only entries issued by
// the current identity. A missing or corrupt file yields an empty store.
func loadStore(cfg *Config, id, target string) map[string][]Finding {
	p, err := cachePath(cfg, target)
	if err != nil {
		return map[string][]Finding{}
	}
	data, rerr := os.ReadFile(p)
	if rerr != nil {
		return map[string][]Finding{}
	}
	var c cacheJSON
	if json.Unmarshal(data, &c) != nil {
		return map[string][]Finding{}
	}
	prefix := id + ":"
	keep := map[string][]Finding{}
	for k, v := range c.Entries {
		if hasPrefix(k, prefix) {
			keep[k] = v
		}
	}
	return keep
}

// saveStore writes a store back, atomically, keeping only the current run's
// entries so the file cannot grow without bound across configs and versions.
func saveStore(cfg *Config, id, target string, store map[string][]Finding) {
	p, err := cachePath(cfg, target)
	if err != nil {
		return
	}
	data, err := json.Marshal(cacheJSON{Entries: pruneByPrefix(store, id)})
	if err != nil {
		return
	}
	writeAtomic(p, data)
}

// pruneByPrefix keeps only entries issued by the current identity.
func pruneByPrefix(store map[string][]Finding, id string) map[string][]Finding {
	pruned := map[string][]Finding{}
	prefix := id + ":"
	for k, v := range store {
		if hasPrefix(k, prefix) {
			pruned[k] = v
		}
	}
	return pruned
}

// writeAtomic writes data to path via a temp file and rename, so a crash
// mid-write leaves the previous store intact rather than a truncated one.
func writeAtomic(path string, data []byte) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, "cache-*.json")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	if err := os.Rename(tmpName, path); err != nil {
		return
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// contentHash is the cache identity of a file's bytes.
func contentHash(src []byte) string {
	return cacheKey(string(src))
}
