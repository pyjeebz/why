package notes

import (
	"strings"
	"testing"
	"time"

	"github.com/pyjeebz/why/internal/target"
)

func TestSaveAndFor(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	key := RepoKey("octo/widgets", "/tmp/x")

	saved, err := Save(key, Note{
		Target: target.Target{Path: "a.go", Start: 5, End: 9},
		Text:   "because thundering herd",
		Source: "declared",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.ID == "" || saved.Created.IsZero() {
		t.Fatalf("Save did not assign ID/Created: %+v", saved)
	}
	if _, err := Save(key, Note{Target: target.Target{Path: "c.go"}, Text: "whole file", Source: "declared"}); err != nil {
		t.Fatalf("Save whole-file: %v", err)
	}

	tests := []struct {
		q    target.Target
		want int
	}{
		{target.Target{Path: "a.go", Start: 7, End: 7}, 1},  // inside range
		{target.Target{Path: "a.go", Start: 9, End: 20}, 1}, // edge overlap
		{target.Target{Path: "a.go", Start: 10, End: 12}, 0},
		{target.Target{Path: "a.go"}, 1}, // whole-file query matches ranged note
		{target.Target{Path: "c.go", Start: 3, End: 3}, 1}, // ranged query matches whole-file note
		{target.Target{Path: "b.go", Start: 7, End: 7}, 0},
	}
	for _, tc := range tests {
		got, err := For(key, tc.q)
		if err != nil {
			t.Fatalf("For(%+v): %v", tc.q, err)
		}
		if len(got) != tc.want {
			t.Errorf("For(%+v) = %d notes, want %d", tc.q, len(got), tc.want)
		}
	}
}

func TestAllEmptyAndOrdering(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	key := RepoKey("", "/some/root")

	if ns, err := All(key); err != nil || len(ns) != 0 {
		t.Fatalf("All on empty store = %v, %v; want none, nil", ns, err)
	}

	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := early.AddDate(0, 3, 0)
	for _, n := range []Note{
		{Target: target.Target{Path: "a.go"}, Text: "second", Created: late, Source: "declared"},
		{Target: target.Target{Path: "a.go"}, Text: "first", Created: early, Source: "declared"},
	} {
		if _, err := Save(key, n); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	ns, err := All(key)
	if err != nil || len(ns) != 2 {
		t.Fatalf("All = %d notes, %v; want 2", len(ns), err)
	}
	if ns[0].Text != "first" || ns[1].Text != "second" {
		t.Fatalf("All not oldest-first: %q, %q", ns[0].Text, ns[1].Text)
	}
}

func TestRepoKey(t *testing.T) {
	if got := RepoKey("octo/widgets", "/r"); got != "octo__widgets" {
		t.Errorf("slug key = %q, want octo__widgets", got)
	}
	a, b := RepoKey("", "/repo/a"), RepoKey("", "/repo/b")
	if !strings.HasPrefix(a, "local__") || a == b {
		t.Errorf("local keys = %q, %q; want distinct local__ keys", a, b)
	}
	if a != RepoKey("", "/repo/a") {
		t.Errorf("local key not deterministic")
	}
}

func TestOverdue(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		reviewBy string
		want     bool
	}{
		{"", false},
		{"2026-06-20", false},
		{"2026-06-11", false}, // due today is not yet overdue
		{"2026-06-01", true},
		{"not-a-date", false},
	}
	for _, tc := range tests {
		n := Note{ReviewBy: tc.reviewBy}
		if got := n.Overdue(now); got != tc.want {
			t.Errorf("Overdue(%q) = %v, want %v", tc.reviewBy, got, tc.want)
		}
	}
}
