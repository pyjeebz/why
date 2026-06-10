// Package cache is a JSON file cache under the user cache directory.
// Cached GitHub objects (commits, merged PRs) are immutable, so entries
// never expire.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// dir returns the why cache root, honoring XDG_CACHE_HOME via
// os.UserCacheDir.
func dir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "why"), nil
}

// Get loads the entry at key (a relative path like "github/o__r/sha.json")
// into v, reporting whether it existed and parsed.
func Get(key string, v any) bool {
	root, err := dir()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(filepath.Join(root, key))
	if err != nil {
		return false
	}
	return json.Unmarshal(data, v) == nil
}

// Put stores v as JSON at key, creating parent directories as needed.
// Cache writes are best-effort; failures only cost a future refetch.
func Put(key string, v any) error {
	root, err := dir()
	if err != nil {
		return err
	}
	path := filepath.Join(root, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
