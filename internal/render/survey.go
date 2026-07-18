package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

// Survey renders a whole-file overview: the regions worth knowing about,
// heaviest history first, each led by its one-line answer and a compact
// receipt. total is how many candidate regions the file breaks into, so the
// footer can be honest that only the most load-bearing are shown.
func Survey(w io.Writer, path string, regions []trail.Trail, total int, color bool) {
	b, d, c, r := bold, dim, cyan, reset
	if !color {
		b, d, c, r = "", "", "", ""
	}

	if len(regions) == 0 {
		fmt.Fprintf(w, "%swhy%s · %s%s%s\n\n", b, r, c, path, r)
		fmt.Fprintf(w, "%sNo regions with load-bearing history — this file's past is git-only, or its reasons were never recorded.%s\n", d, r)
		return
	}

	fmt.Fprintf(w, "%swhy%s · %s%s%s — %s worth knowing about\n\n", b, r, c, path, r, nregions(len(regions)))
	for _, tr := range regions {
		fmt.Fprintf(w, "%s▸%s %s%s%s\n", c, r, b, regionLabel(tr.Target), r)
		if tr.Answer != nil {
			fmt.Fprintf(w, "  %s\n", tr.Answer.Text)
		}
		fmt.Fprintf(w, "  %s%s%s\n\n", d, receiptLine(tr), r)
	}

	if rest := total - len(regions); rest > 0 {
		fmt.Fprintf(w, "%sscanned %d change-groups; showing the %d with the most load-bearing history.%s\n", d, total, len(regions), r)
	}
}

func regionLabel(t target.Target) string {
	if t.Start == t.End {
		return fmt.Sprintf("line %d", t.Start)
	}
	return fmt.Sprintf("lines %d-%d", t.Start, t.End)
}

// receiptLine is the compact one-liner beneath a surveyed region's answer:
// where the answer came from, then the strongest artifact behind it.
func receiptLine(tr trail.Trail) string {
	var parts []string
	if tr.Answer != nil {
		parts = append(parts, provenance(tr.Answer))
	}
	if len(tr.Hops) > 0 {
		h := tr.Hops[0]
		parts = append(parts, h.Commit.ShortSHA())
		if h.PR != nil {
			parts = append(parts, fmt.Sprintf("PR #%d", h.PR.Number))
		}
		for _, is := range h.Issues {
			parts = append(parts, fmt.Sprintf("closes #%d", is.Number))
		}
	}
	return strings.Join(parts, " · ")
}
