// Package answer computes the one-line why for a region — the reason a
// reader wants before the receipts. It walks a provenance ladder and never
// invents: a human's declared note wins; then a saved draft; then a
// deterministic headline restated from the strongest artifact already in the
// trail. Every rung is labelled with where it came from.
package answer

import (
	"fmt"

	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/trail"
)

// For returns the answer for a trail, or nil when there is nothing to say
// (no notes, no history). allowLLM is reserved for the drafted rung, which
// synthesises the trail into prose; it is not wired yet, so a trail with no
// note falls straight through to the deterministic headline.
func For(t trail.Trail, allowLLM bool) *trail.Answer {
	if a := fromNotes(t.Notes); a != nil {
		return a
	}
	// TODO(drafted rung): when allowLLM and an LLM is configured, synthesise
	// the trail into prose, cache it on the trail signature, and return it
	// with Source "drafted" before falling back to the headline.
	return headline(t)
}

// fromNotes prefers a human's declared note (most recent), then a saved
// inferred draft. The note's own text becomes the answer verbatim.
func fromNotes(ns []notes.Note) *trail.Answer {
	var declared, inferred *notes.Note
	for i := range ns {
		switch n := &ns[i]; n.Source {
		case "declared":
			if declared == nil || n.Created.After(declared.Created) {
				declared = n
			}
		case "inferred":
			if inferred == nil || n.Created.After(inferred.Created) {
				inferred = n
			}
		}
	}
	switch {
	case declared != nil:
		return &trail.Answer{Text: declared.Text, Source: "declared", Author: declared.Author}
	case inferred != nil:
		return &trail.Answer{Text: inferred.Text, Source: "drafted", Author: inferred.Author}
	default:
		return nil
	}
}

// headline restates the strongest artifact in the trail as one deterministic
// sentence. An issue is the why-behind-the-why, so it leads; then a PR; then,
// with nothing linked, the commit itself. It only restates text already
// present — it never synthesises.
func headline(t trail.Trail) *trail.Answer {
	for _, h := range t.Hops {
		if len(h.Issues) > 0 {
			is := h.Issues[0]
			text := fmt.Sprintf("Exists because of #%d (%s)", is.Number, is.Title)
			if h.PR != nil {
				text += fmt.Sprintf(", via PR #%d", h.PR.Number)
			}
			return &trail.Answer{Text: text + ".", Source: "headline"}
		}
	}
	for _, h := range t.Hops {
		if h.PR != nil {
			return &trail.Answer{
				Text:   fmt.Sprintf("Introduced by PR #%d (%s).", h.PR.Number, h.PR.Title),
				Source: "headline",
			}
		}
	}
	if len(t.Hops) > 0 {
		h := t.Hops[0]
		return &trail.Answer{
			Text:   fmt.Sprintf("Last shaped by %q — %s, %s.", h.Commit.Subject, h.Commit.ShortSHA(), h.Commit.Date.Format("2006-01-02")),
			Source: "headline",
		}
	}
	return nil
}
