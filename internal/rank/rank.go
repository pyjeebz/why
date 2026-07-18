// Package rank scores and identifies decision trails: how load-bearing a
// region's history looks (Weight), whether it is worth surfacing on its own
// (Meaningful), and what history shaped it (Signature). Both the PR-comment
// digest and the whole-file survey lean on these, so they live apart from
// any one renderer.
package rank

import (
	"strings"

	"github.com/pyjeebz/why/internal/trail"
)

// Signature keys a trail by the commits (and any note IDs) behind it, so
// two regions shaped by the same history collapse and a noted region never
// folds into an un-noted one.
func Signature(t trail.Trail) string {
	var b strings.Builder
	for _, h := range t.Hops {
		b.WriteString(h.Commit.SHA)
		b.WriteByte(',')
	}
	for _, n := range t.Notes {
		b.WriteString("n:")
		b.WriteString(n.ID)
		b.WriteByte(',')
	}
	return b.String()
}

// Weight scores how load-bearing a trail's history looks. Notes and linked
// issues count most (a human deliberately recorded something), PRs and
// incident-flavored commits next, depth last.
func Weight(t trail.Trail) int {
	w := 3 * len(t.Notes)
	for _, h := range t.Hops {
		w++
		if h.PR != nil {
			w += 2
		}
		w += 2 * len(h.Issues)
		if Incident(h.Commit.Subject) {
			w += 2
		}
	}
	return w
}

// Meaningful reports whether a trail is worth surfacing on its own rather
// than folding into a tally: it carries a note, more than one commit, or a
// linked PR/issue.
func Meaningful(t trail.Trail) bool {
	if len(t.Notes) > 0 || len(t.Hops) >= 2 {
		return true
	}
	for _, h := range t.Hops {
		if h.PR != nil || len(h.Issues) > 0 {
			return true
		}
	}
	return false
}

var incidentWords = []string{
	"revert", "rollback", "hotfix", "regression", "race", "deadlock",
	"leak", "security", "vuln", "cve", "incident", "outage", "panic", "corrupt",
}

// Incident reports whether a commit subject smells like it was driven by an
// incident — the history most worth putting in front of someone.
func Incident(subject string) bool {
	s := strings.ToLower(subject)
	for _, word := range incidentWords {
		if strings.Contains(s, word) {
			return true
		}
	}
	return false
}
