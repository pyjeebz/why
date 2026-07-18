package rank

import (
	"testing"

	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/trail"
)

func hop(sha, subject string) trail.Hop {
	return trail.Hop{Commit: trail.Commit{SHA: sha, Subject: subject}}
}

func TestMeaningful(t *testing.T) {
	lone := trail.Trail{Hops: []trail.Hop{hop("a", "tweak")}}
	if Meaningful(lone) {
		t.Error("a lone plain commit should not be meaningful")
	}

	withPR := trail.Trail{Hops: []trail.Hop{{Commit: trail.Commit{SHA: "a"}, PR: &trail.PR{Number: 1}}}}
	if !Meaningful(withPR) {
		t.Error("a linked PR should make a trail meaningful")
	}

	two := trail.Trail{Hops: []trail.Hop{hop("a", "x"), hop("b", "y")}}
	if !Meaningful(two) {
		t.Error("two commits should be meaningful")
	}

	noted := trail.Trail{Notes: []notes.Note{{ID: "n1"}}}
	if !Meaningful(noted) {
		t.Error("a note should make a trail meaningful even with no hops")
	}
}

func TestWeight_liftsLoadBearingAbovePlain(t *testing.T) {
	plain := trail.Trail{Hops: []trail.Hop{hop("a", "refactor")}}
	withIssue := trail.Trail{Hops: []trail.Hop{{
		Commit: trail.Commit{SHA: "a", Subject: "fix"},
		PR:     &trail.PR{Number: 1},
		Issues: []trail.Issue{{Number: 2}},
	}}}
	noted := trail.Trail{Notes: []notes.Note{{ID: "n1"}}, Hops: []trail.Hop{hop("a", "x")}}

	// Both a human note and a linked PR+issue outrank a bare commit. The
	// note-vs-issue tiebreak is intentionally left unasserted — it is an open
	// tuning question for whole-file ranking, not settled here.
	if Weight(noted) <= Weight(plain) {
		t.Errorf("a note should outweigh a bare commit: %d vs %d", Weight(noted), Weight(plain))
	}
	if Weight(withIssue) <= Weight(plain) {
		t.Errorf("a linked PR+issue should outweigh a bare commit: %d vs %d", Weight(withIssue), Weight(plain))
	}
}

func TestWeight_incidentBoost(t *testing.T) {
	plain := trail.Trail{Hops: []trail.Hop{hop("a", "update deps")}}
	incidenty := trail.Trail{Hops: []trail.Hop{hop("a", "fix: race in retry loop")}}
	if Weight(incidenty) <= Weight(plain) {
		t.Errorf("an incident-flavored subject should score higher: %d vs %d", Weight(incidenty), Weight(plain))
	}
}

func TestSignature_collapsesSharedHistoryButNotNotes(t *testing.T) {
	// Two independently-built trails with the same history collapse.
	a := trail.Trail{Hops: []trail.Hop{hop("a", "x"), hop("b", "y")}}
	b := trail.Trail{Hops: []trail.Hop{hop("a", "x"), hop("b", "y")}}
	if Signature(a) != Signature(b) {
		t.Error("identical history should share a signature")
	}

	// The same history with a note attached must not collapse into it.
	noted := trail.Trail{Hops: []trail.Hop{hop("a", "x"), hop("b", "y")}, Notes: []notes.Note{{ID: "n1"}}}
	if Signature(noted) == Signature(a) {
		t.Error("a noted region should not collapse into an un-noted one")
	}
}
