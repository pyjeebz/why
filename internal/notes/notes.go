// Package notes stores personal provenance notes in an overlay outside
// the repository, so contributors can annotate upstream code they
// cannot commit to. One JSON file per note, grouped by repository.
package notes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pyjeebz/why/internal/target"
)

// Note is one declared or inferred reason attached to a region of code.
// ReviewBy is a YYYY-MM-DD date after which the note needs review: notes
// are claims that expire, not permanent truth.
type Note struct {
	ID        string        `json:"id"`
	Target    target.Target `json:"target"`
	Text      string        `json:"text"`
	Author    string        `json:"author,omitempty"`
	Created   time.Time     `json:"created"`
	ReviewBy  string        `json:"review_by,omitempty"`
	Source    string        `json:"source"` // declared | inferred
	AnchorSHA string        `json:"anchor_sha,omitempty"`
}

// Overdue reports whether the note's review-by date has passed.
func (n Note) Overdue(now time.Time) bool {
	if n.ReviewBy == "" {
		return false
	}
	d, err := time.Parse("2006-01-02", n.ReviewBy)
	return err == nil && now.After(d.AddDate(0, 0, 1))
}

// RepoKey identifies a repository in the overlay store: the GitHub slug
// when known, otherwise a hash of the local root path.
func RepoKey(slug, root string) string {
	if slug != "" {
		return strings.ReplaceAll(slug, "/", "__")
	}
	sum := sha256.Sum256([]byte(root))
	return "local__" + hex.EncodeToString(sum[:6])
}

// Save persists a note for the repository, assigning ID and Created.
func Save(repoKey string, n Note) (Note, error) {
	d, err := dir(repoKey)
	if err != nil {
		return Note{}, err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return Note{}, err
	}

	if n.Created.IsZero() {
		n.Created = time.Now()
	}
	for id := n.Created.UnixNano(); ; id++ {
		n.ID = fmt.Sprintf("%d", id)
		if _, err := os.Stat(filepath.Join(d, n.ID+".json")); os.IsNotExist(err) {
			break
		}
	}

	data, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return Note{}, err
	}
	if err := os.WriteFile(filepath.Join(d, n.ID+".json"), data, 0o644); err != nil {
		return Note{}, err
	}
	return n, nil
}

// All returns every note for the repository, oldest first. A repository
// with no notes yet is not an error.
func All(repoKey string) ([]Note, error) {
	d, err := dir(repoKey)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(d)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var ns []Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(d, e.Name()))
		if err != nil {
			continue
		}
		var n Note
		if err := json.Unmarshal(data, &n); err != nil {
			continue
		}
		ns = append(ns, n)
	}
	sort.Slice(ns, func(i, j int) bool { return ns[i].Created.Before(ns[j].Created) })
	return ns, nil
}

// For returns the repository's notes overlapping the target region,
// oldest first.
func For(repoKey string, t target.Target) ([]Note, error) {
	all, err := All(repoKey)
	if err != nil {
		return nil, err
	}
	var ns []Note
	for _, n := range all {
		if overlaps(n.Target, t) {
			ns = append(ns, n)
		}
	}
	return ns, nil
}

func overlaps(a, b target.Target) bool {
	if a.Path != b.Path {
		return false
	}
	if a.WholeFile() || b.WholeFile() {
		return true
	}
	return a.Start <= b.End && b.Start <= a.End
}

func dir(repoKey string) (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "why", "notes", repoKey), nil
}
