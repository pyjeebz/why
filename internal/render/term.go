// Package render turns a trail into output for humans and machines.
package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pyjeebz/why/internal/trail"
)

const (
	bold   = "\033[1m"
	dim    = "\033[2m"
	cyan   = "\033[36m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

// Term writes the trail for a terminal, newest hop first. When color is
// false (piped output, NO_COLOR), plain text is emitted.
func Term(w io.Writer, t trail.Trail, color bool) {
	b, d, c, y, r := bold, dim, cyan, yellow, reset
	if !color {
		b, d, c, y, r = "", "", "", "", ""
	}

	if t.Answer != nil {
		fmt.Fprintf(w, "%swhy%s · %s%s%s\n\n", b, r, c, t.Target.String(), r)
		fmt.Fprintf(w, "  %s%s%s\n", b, t.Answer.Text, r)
		fmt.Fprintf(w, "  %s— %s%s\n\n", d, provenance(t.Answer), r)
		if len(t.Hops) > 0 {
			fmt.Fprintf(w, "%sreceipts%s · %s, newest first\n\n", d, r, nhops(len(t.Hops)))
		}
	} else {
		fmt.Fprintf(w, "%swhy%s · trail for %s%s%s — %s, newest first\n\n", b, r, c, t.Target.String(), r, nhops(len(t.Hops)))
	}

	for _, n := range t.Notes {
		if t.Answer != nil && n.Text == t.Answer.Text {
			continue // already shown as the answer
		}
		fmt.Fprintf(w, "%s✎%s %s\n", y, r, n.Text)
		fmt.Fprintf(w, "  %s%s%s\n\n", d, noteMeta(n, time.Now()), r)
	}

	for _, h := range t.Hops {
		fmt.Fprintf(w, "%s●%s %s%s%s  %s%s  %s%s\n", c, r, y, h.Commit.ShortSHA(), r, d, h.Commit.Date.Format("2006-01-02"), h.Commit.Author, r)
		fmt.Fprintf(w, "  %s%s%s\n", b, h.Commit.Subject, r)
		if h.PR != nil {
			fmt.Fprintf(w, "  └ PR #%d: %s\n", h.PR.Number, h.PR.Title)
			fmt.Fprintf(w, "    %s%s%s\n", d, h.PR.URL, r)
		}
		for _, is := range h.Issues {
			fmt.Fprintf(w, "  └ closes #%d: %s\n", is.Number, is.Title)
			fmt.Fprintf(w, "    %s%s%s\n", d, is.URL, r)
		}
		if body := firstLine(h.Commit.Body); body != "" {
			fmt.Fprintf(w, "  %s%s%s\n", d, body, r)
		}
		fmt.Fprintln(w)
	}

	if t.Notice != "" {
		fmt.Fprintf(w, "%s· %s%s\n", d, t.Notice, r)
	}
}

// provenance renders an Answer's source as a short human phrase for the line
// beneath it, so a reader always knows whether the why was written by a
// person, drafted from the trail, or restated from an artifact.
func provenance(a *trail.Answer) string {
	switch a.Source {
	case "declared":
		if a.Author != "" {
			return "recorded by " + a.Author
		}
		return "a recorded note"
	case "drafted":
		if a.Author != "" {
			return "drafted from the trail, saved by " + a.Author
		}
		return "drafted from the trail"
	case "headline":
		return "from the trail"
	default:
		return "no reason recorded"
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
