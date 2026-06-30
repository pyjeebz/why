package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

func TestDropCommits_excludesTheChangeUnderReview(t *testing.T) {
	tr := trail.Trail{Hops: []trail.Hop{
		{Commit: trail.Commit{SHA: "pr1"}},
		{Commit: trail.Commit{SHA: "old1"}},
		{Commit: trail.Commit{SHA: "pr2"}},
		{Commit: trail.Commit{SHA: "old2"}},
	}}
	dropCommits(&tr, map[string]bool{"pr1": true, "pr2": true})

	if len(tr.Hops) != 2 || tr.Hops[0].Commit.SHA != "old1" || tr.Hops[1].Commit.SHA != "old2" {
		t.Fatalf("expected only the prior history to remain, got %+v", tr.Hops)
	}
}

func TestExpand_widensAndClampsToFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("a\nb\nc\nd\ne\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A one-line change in the middle, widened by 2, stays inside the file.
	got := expand(dir, []target.Target{{Path: "f.txt", Start: 3, End: 3}}, 2)
	if r := got[0]; r.Start != 1 || r.End != 5 {
		t.Fatalf("expected 1-5 after clamp, got %d-%d", r.Start, r.End)
	}

	// A change at the very end clamps to the last line, not past it.
	got = expand(dir, []target.Target{{Path: "f.txt", Start: 5, End: 5}}, 3)
	if r := got[0]; r.Start != 2 || r.End != 5 {
		t.Fatalf("expected 2-5 after clamp, got %d-%d", r.Start, r.End)
	}
}
