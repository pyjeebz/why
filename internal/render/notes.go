package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pyjeebz/why/internal/notes"
)

// Notes writes a list of overlay notes, oldest first, for why log.
func Notes(w io.Writer, ns []notes.Note, color bool) {
	d, c, y, r := dim, cyan, yellow, reset
	if !color {
		d, c, y, r = "", "", "", ""
	}
	for _, n := range ns {
		fmt.Fprintf(w, "%s✎%s %s%s%s\n", y, r, c, n.Target.String(), r)
		fmt.Fprintf(w, "  %s\n", n.Text)
		fmt.Fprintf(w, "  %s%s%s\n\n", d, noteMeta(n, time.Now()), r)
	}
}

// noteMeta is the shared meta line: author · source · created, plus the
// review state when the note carries a review-by date.
func noteMeta(n notes.Note, now time.Time) string {
	var parts []string
	if n.Author != "" {
		parts = append(parts, n.Author)
	}
	parts = append(parts, n.Source, n.Created.Format("2006-01-02"))
	switch {
	case n.Overdue(now):
		parts = append(parts, "⚠ review overdue ("+n.ReviewBy+")")
	case n.ReviewBy != "":
		parts = append(parts, "review by "+n.ReviewBy)
	}
	return strings.Join(parts, " · ")
}
