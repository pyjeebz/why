package answer

import (
	"testing"
	"time"

	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/trail"
)

func TestFor_declaredNoteWinsVerbatim(t *testing.T) {
	tr := trail.Trail{
		Notes: []notes.Note{{Text: "kept at 2 to avoid double charges", Source: "declared", Author: "Jane", Created: time.Now()}},
		Hops:  []trail.Hop{{Commit: trail.Commit{SHA: "a"}, PR: &trail.PR{Number: 9, Title: "x"}}},
	}
	a := For(tr, false)
	if a == nil || a.Source != "declared" || a.Text != "kept at 2 to avoid double charges" || a.Author != "Jane" {
		t.Fatalf("a declared note should win verbatim, got %+v", a)
	}
}

func TestFor_inferredNoteIsLabelledDrafted(t *testing.T) {
	tr := trail.Trail{Notes: []notes.Note{{Text: "drafted reason", Source: "inferred", Created: time.Now()}}}
	if a := For(tr, false); a == nil || a.Source != "drafted" {
		t.Fatalf("an inferred note should be labelled drafted, got %+v", a)
	}
}

func TestFor_headlineLeadsWithIssue(t *testing.T) {
	tr := trail.Trail{Hops: []trail.Hop{{
		Commit: trail.Commit{SHA: "a", Subject: "fix"},
		PR:     &trail.PR{Number: 380, Title: "Cap retries"},
		Issues: []trail.Issue{{Number: 214, Title: "Double charge on 502 retries"}},
	}}}
	a := For(tr, false)
	want := "Exists because of #214 (Double charge on 502 retries), via PR #380."
	if a == nil || a.Source != "headline" || a.Text != want {
		t.Fatalf("headline = %+v, want %q", a, want)
	}
}

func TestFor_headlineFallsBackToPRThenCommit(t *testing.T) {
	prOnly := trail.Trail{Hops: []trail.Hop{{Commit: trail.Commit{SHA: "a"}, PR: &trail.PR{Number: 7, Title: "Add cache"}}}}
	if a := For(prOnly, false); a == nil || a.Text != "Introduced by PR #7 (Add cache)." {
		t.Fatalf("PR headline = %+v", a)
	}

	date := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	commitOnly := trail.Trail{Hops: []trail.Hop{{Commit: trail.Commit{SHA: "abcdef1234", Subject: "tweak backoff", Date: date}}}}
	if a := For(commitOnly, false); a == nil || a.Text != `Last shaped by "tweak backoff" — abcdef1, 2026-03-02.` {
		t.Fatalf("commit headline = %+v", a)
	}
}

func TestFor_emptyTrailHasNoAnswer(t *testing.T) {
	if a := For(trail.Trail{}, false); a != nil {
		t.Fatalf("an empty trail should have no answer, got %+v", a)
	}
}
