// Package render turns a trail into output for humans and machines.
package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/pyjeebz/why/internal/trail"
)

const (
	bold  = "\033[1m"
	dim   = "\033[2m"
	cyan  = "\033[36m"
	reset = "\033[0m"
)

// Term writes the trail for a terminal, newest hop first. When color is
// false (piped output), plain text is emitted.
func Term(w io.Writer, t trail.Trail, color bool) {
	b, d, c, r := bold, dim, cyan, reset
	if !color {
		b, d, c, r = "", "", "", ""
	}

	fmt.Fprintf(w, "%swhy%s · trail for %s%s%s — %d hop(s), newest first\n\n", b, r, c, t.Target.String(), r, len(t.Hops))

	for _, h := range t.Hops {
		fmt.Fprintf(w, "%s●%s %s%s%s  %s  %s\n", c, r, b, h.Commit.ShortSHA(), r, h.Commit.Date.Format("2006-01-02"), h.Commit.Author)
		fmt.Fprintf(w, "  %s\n", h.Commit.Subject)
		if h.PR != nil {
			fmt.Fprintf(w, "  %s└ PR #%d: %s%s\n", d, h.PR.Number, h.PR.Title, r)
			fmt.Fprintf(w, "  %s  %s%s\n", d, h.PR.URL, r)
		}
		for _, is := range h.Issues {
			fmt.Fprintf(w, "  %s└ issue #%d: %s — %s%s\n", d, is.Number, is.Title, is.URL, r)
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

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
