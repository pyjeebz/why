package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

func TestCollectTrails_countsSkippedSeparatelyFromKept(t *testing.T) {
	orig := runDig
	defer func() { runDig = orig }()
	runDig = func(_ string, tg target.Target, _ int) (trail.Trail, error) {
		if tg.Path == "bad.go" {
			return trail.Trail{}, errors.New("git -L failed")
		}
		return trail.Trail{Target: tg, Hops: []trail.Hop{{Commit: trail.Commit{SHA: "abc"}}}}, nil
	}

	regions := []target.Target{
		{Path: "ok.go", Start: 1, End: 2},
		{Path: "bad.go", Start: 1, End: 2},
	}
	trails, considered, skipped := collectTrails("", regions, nil, 8, 25)

	if considered != 2 || skipped != 1 || len(trails) != 1 {
		t.Fatalf("considered=%d skipped=%d kept=%d; want 2/1/1", considered, skipped, len(trails))
	}
	if trails[0].Target.Path != "ok.go" {
		t.Fatalf("kept the wrong region: %s", trails[0].Target.Path)
	}
}

func TestCollectTrails_stopsAtMax(t *testing.T) {
	orig := runDig
	defer func() { runDig = orig }()
	calls := 0
	runDig = func(string, target.Target, int) (trail.Trail, error) {
		calls++
		return trail.Trail{}, nil
	}

	regions := []target.Target{{Path: "a"}, {Path: "b"}, {Path: "c"}}
	_, considered, _ := collectTrails("", regions, nil, 8, 2)

	if considered != 2 || calls != 2 {
		t.Fatalf("considered=%d calls=%d; want 2/2", considered, calls)
	}
}

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
