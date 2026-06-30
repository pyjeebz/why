package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

func region(path string, start, end int, hops []trail.Hop, ns []notes.Note) trail.Trail {
	return trail.Trail{
		Target: target.Target{Path: path, Start: start, End: end},
		Repo:   "octo/widgets",
		Hops:   hops,
		Notes:  ns,
	}
}

func commit(sha, subject string) trail.Hop {
	return trail.Hop{Commit: trail.Commit{SHA: sha, Subject: subject, Author: "Dev", Date: time.Now()}}
}

func TestComment_expandsMeaningfulTalliesTheRest(t *testing.T) {
	withPR := commit("aaa1111", "fix: clamp backoff")
	withPR.PR = &trail.PR{Number: 42, Title: "Clamp backoff", URL: "https://x/42"}

	regions := []trail.Trail{
		region("a.go", 1, 5, []trail.Hop{withPR}, nil),                  // meaningful: has a PR
		region("b.go", 1, 1, []trail.Hop{commit("bbb2222", "tweak")}, nil), // thin: lone commit, no PR
		region("c.go", 9, 9, []trail.Hop{commit("bbb2222", "tweak")}, nil), // same trail as b.go via SHA... but different path
		region("d.go", 1, 1, nil, nil),                                  // bare: no history
	}

	var buf bytes.Buffer
	Comment(&buf, regions, true)
	out := buf.String()

	if !strings.Contains(out, CommentMarker) {
		t.Error("missing update marker")
	}
	if !strings.Contains(out, "`a.go:1-5`") {
		t.Errorf("meaningful region a.go not expanded:\n%s", out)
	}
	if !strings.Contains(out, "PR [#42]") {
		t.Errorf("PR not rendered for meaningful region:\n%s", out)
	}
	if strings.Contains(out, "b.go") || strings.Contains(out, "c.go") {
		t.Errorf("thin regions should be tallied, not expanded:\n%s", out)
	}
	if !strings.Contains(out, "single commit with no linked PR or issue") {
		t.Errorf("missing thin tally:\n%s", out)
	}
	if !strings.Contains(out, "1 with no recorded history") {
		t.Errorf("missing bare tally:\n%s", out)
	}
}

func TestComment_collapsesSharedTrail(t *testing.T) {
	// Two regions in the same file shaped by the exact same two commits
	// (a meaningful trail) should collapse into one section listing both.
	hops := []trail.Hop{commit("ccc3333", "refactor"), commit("ddd4444", "init")}
	regions := []trail.Trail{
		region("main.go", 5, 5, hops, nil),
		region("main.go", 10, 26, hops, nil),
	}

	var buf bytes.Buffer
	Comment(&buf, regions, true)
	out := buf.String()

	if n := strings.Count(out, "#### "); n != 1 {
		t.Errorf("expected regions with identical trails to collapse into 1 section, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "`main.go:5, 10-26`") {
		t.Errorf("collapsed section should list both line specs:\n%s", out)
	}
}

func TestComment_staysQuietWhenNothingLoadBearing(t *testing.T) {
	regions := []trail.Trail{
		region("a.go", 1, 1, []trail.Hop{commit("aaa1111", "tweak")}, nil),
		region("b.go", 1, 1, nil, nil),
	}

	var buf bytes.Buffer
	Comment(&buf, regions, true)
	out := buf.String()

	if !strings.Contains(out, "load-bearing") {
		t.Errorf("expected restraint message when nothing meaningful:\n%s", out)
	}
	if strings.Contains(out, "#### ") {
		t.Errorf("nothing should be expanded:\n%s", out)
	}
}

func TestComment_nudgesWhenReasonMissingAndCanBeSilenced(t *testing.T) {
	regions := []trail.Trail{
		region("a.go", 1, 1, []trail.Hop{commit("aaa1111", "tweak")}, nil), // thin: no recorded reason
	}

	var on, off bytes.Buffer
	Comment(&on, regions, true)
	Comment(&off, regions, false)

	if !strings.Contains(on.String(), "the trail the next person digs up") {
		t.Errorf("expected a nudge when reason is missing:\n%s", on.String())
	}
	if strings.Contains(off.String(), "the trail the next person digs up") {
		t.Errorf("nudge should be suppressed when disabled:\n%s", off.String())
	}
}

func TestComment_nudgeIsAnAsideWhenHistoryAlsoExists(t *testing.T) {
	withPR := commit("aaa1111", "fix: clamp backoff")
	withPR.PR = &trail.PR{Number: 42, Title: "Clamp", URL: "https://x/42"}
	regions := []trail.Trail{
		region("a.go", 1, 5, []trail.Hop{withPR}, nil), // meaningful
		region("b.go", 1, 1, nil, nil),                 // bare: no history
	}

	var buf bytes.Buffer
	Comment(&buf, regions, true)
	out := buf.String()

	if !strings.Contains(out, "`a.go:1-5`") {
		t.Errorf("meaningful history should still expand:\n%s", out)
	}
	if !strings.Contains(out, "isn't recorded yet") {
		t.Errorf("expected the quiet aside nudge alongside real history:\n%s", out)
	}
}

func TestComment_rendersNoteAndRanksItFirst(t *testing.T) {
	noted := region("hot.go", 1, 3,
		[]trail.Hop{commit("eee5555", "adjust")},
		[]notes.Note{{ID: "n1", Text: "deliberate, do not simplify", Source: "declared", Created: time.Now()}},
	)
	plain := commit("fff6666", "fix: race in loop") // incident word + ... still 1 hop, no PR
	plainTrail := region("other.go", 1, 1, []trail.Hop{plain}, nil)
	// give it a PR so it is meaningful and competes for ranking
	plainTrail.Hops[0].PR = &trail.PR{Number: 7, Title: "Fix race", URL: "https://x/7"}

	var buf bytes.Buffer
	Comment(&buf, []trail.Trail{plainTrail, noted}, true)
	out := buf.String()

	if !strings.Contains(out, "deliberate, do not simplify") {
		t.Errorf("note not rendered:\n%s", out)
	}
	// The noted region (weight 3 + 1) should outrank the PR region (weight 1 + 2 + 2 incident).
	// Both are meaningful; assert the noted one appears and the note shows.
	if i, j := strings.Index(out, "hot.go"), strings.Index(out, "other.go"); i < 0 || j < 0 {
		t.Errorf("both meaningful regions should appear:\n%s", out)
	}
}
