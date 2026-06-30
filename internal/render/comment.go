package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

// CommentMarker is an invisible HTML tag stamped on the PR comment so the
// poster can find and update its own comment instead of spamming a new one
// on every push.
const CommentMarker = "<!-- why-trail -->"

// maxSections caps how many regions the comment expands; the rest are
// tallied so even a sprawling change yields a readable comment.
const maxSections = 10

// section is one rendered block: a trail plus every changed region that
// shares it. Regions whose history is identical collapse into one section
// so a single commit is never printed twice.
type section struct {
	targets []target.Target
	trail   trail.Trail
	weight  int
}

// Comment renders the trails behind the regions a change touches as a
// single PR comment. It collapses regions that share a trail, ranks what
// remains by how load-bearing its history looks, and stays quiet when
// nothing the change touches has a story worth telling — the restraint is
// the point, and what keeps it from being noise.
//
// When nudge is set and some touched code has no recorded reason, it closes
// the loop: rather than only reading history, it asks the author to record
// the why now — in the commit or PR, where the next dig will recover it.
func Comment(w io.Writer, regions []trail.Trail, nudge bool) {
	sections, thin, bare := digest(regions)

	fmt.Fprintf(w, "%s\n### why · context for this change\n\n", CommentMarker)

	if len(sections) == 0 {
		if thin+bare > 0 {
			fmt.Fprintf(w, "Nothing the history flags as load-bearing — %s changed with no recorded reason.\n\n", nregions(thin+bare))
			if nudge {
				fmt.Fprint(w, nudgeText)
			}
		} else {
			fmt.Fprintf(w, "No recorded history behind the regions this change touches.\n\n")
		}
		fmt.Fprint(w, commentFooter)
		return
	}

	fmt.Fprintf(w, "This change touches code with recorded history. Here is why it is the way it is:\n\n")

	shown, overflow := sections, 0
	if len(shown) > maxSections {
		overflow = len(shown) - maxSections
		shown = shown[:maxSections]
	}
	for _, s := range shown {
		fmt.Fprintf(w, "#### %s\n\n", joinTargets(s.targets))
		for _, n := range s.trail.Notes {
			noteQuote(w, n)
		}
		for _, h := range s.trail.Hops {
			hopBullet(w, s.trail.Repo, h)
		}
		fmt.Fprintln(w)
	}

	var tally []string
	if overflow > 0 {
		tally = append(tally, fmt.Sprintf("%d more region(s) with history", overflow))
	}
	if thin > 0 {
		tally = append(tally, fmt.Sprintf("%d traced to a single commit with no linked PR or issue", thin))
	}
	if bare > 0 {
		tally = append(tally, fmt.Sprintf("%d with no recorded history", bare))
	}
	if len(tally) > 0 {
		fmt.Fprintf(w, "<sub>Also: %s.</sub>\n\n", strings.Join(tally, "; "))
	}
	if nudge && thin+bare > 0 {
		fmt.Fprint(w, nudgeLine)
	}
	fmt.Fprint(w, commentFooter)
}

const commentFooter = "<sub>dug up with [`why`](https://github.com/pyjeebz/why)</sub>\n"

// nudgeText is the whole message when nothing the change touches carries a
// recorded reason; nudgeLine is the quieter aside when some regions do have
// history and others do not. Both point the author at the commit or PR,
// because that is what a future dig reads back.
const (
	nudgeText = "If you know why it is this way, a line in the commit message — or this PR's description — becomes the trail the next person digs up.\n\n"
	nudgeLine = "<sub>↳ The reason for those isn't recorded yet — a line in the commit or PR body will be here next time.</sub>\n\n"
)

// digest groups regions into sections by shared trail, splits off the
// regions whose history is thin (a lone commit, no PR/issue/note) or bare
// (no history at all), and returns the meaningful sections ranked by
// weight, heaviest first.
func digest(regions []trail.Trail) (sections []section, thin, bare int) {
	bySig := map[string]*section{}
	var order []string
	for _, t := range regions {
		if len(t.Hops) == 0 && len(t.Notes) == 0 {
			bare++
			continue
		}
		sig := signature(t)
		s, ok := bySig[sig]
		if !ok {
			s = &section{trail: t, weight: weight(t)}
			bySig[sig] = s
			order = append(order, sig)
		}
		s.targets = append(s.targets, t.Target)
	}

	for _, sig := range order {
		s := bySig[sig]
		if meaningful(s.trail) {
			sections = append(sections, *s)
		} else {
			thin += len(s.targets)
		}
	}
	sort.SliceStable(sections, func(i, j int) bool {
		return sections[i].weight > sections[j].weight
	})
	return sections, thin, bare
}

// signature keys a trail by the commits (and any note IDs) behind it, so
// two regions shaped by the same history collapse and a noted region never
// folds into an un-noted one.
func signature(t trail.Trail) string {
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

// weight scores how load-bearing a trail's history looks. Notes and linked
// issues count most (a human deliberately recorded something), PRs and
// incident-flavored commits next, depth last.
func weight(t trail.Trail) int {
	w := 3 * len(t.Notes)
	for _, h := range t.Hops {
		w++
		if h.PR != nil {
			w += 2
		}
		w += 2 * len(h.Issues)
		if incident(h.Commit.Subject) {
			w += 2
		}
	}
	return w
}

// meaningful reports whether a trail is worth expanding rather than
// tallying: it carries a note, more than one commit, or a linked PR/issue.
func meaningful(t trail.Trail) bool {
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

func incident(subject string) bool {
	s := strings.ToLower(subject)
	for _, word := range incidentWords {
		if strings.Contains(s, word) {
			return true
		}
	}
	return false
}

// joinTargets renders a section's regions as code spans, grouping line
// specs under their file: `main.go:5, 10-26`.
func joinTargets(ts []target.Target) string {
	var order []string
	spec := map[string][]string{}
	for _, t := range ts {
		if _, ok := spec[t.Path]; !ok {
			order = append(order, t.Path)
		}
		spec[t.Path] = append(spec[t.Path], lineSpec(t))
	}
	parts := make([]string, 0, len(order))
	for _, p := range order {
		parts = append(parts, fmt.Sprintf("`%s:%s`", p, strings.Join(spec[p], ", ")))
	}
	return strings.Join(parts, ", ")
}

func lineSpec(t target.Target) string {
	switch {
	case t.WholeFile():
		return "all"
	case t.Start == t.End || t.End == 0:
		return fmt.Sprintf("%d", t.Start)
	default:
		return fmt.Sprintf("%d-%d", t.Start, t.End)
	}
}

// nregions formats a region count with its noun.
func nregions(n int) string {
	if n == 1 {
		return "1 region"
	}
	return fmt.Sprintf("%d regions", n)
}
