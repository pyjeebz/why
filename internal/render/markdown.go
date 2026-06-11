package render

import (
	"fmt"
	"io"
	"time"

	"github.com/pyjeebz/why/internal/trail"
)

// Markdown writes the trail as GitHub-flavored markdown shaped for
// pasting into a PR or issue comment: compact, linked, self-contained.
func Markdown(w io.Writer, t trail.Trail) {
	fmt.Fprintf(w, "### why · `%s` — %s, newest first\n\n", t.Target.String(), nhops(len(t.Hops)))

	for _, n := range t.Notes {
		fmt.Fprintf(w, "> ✎ %s\n> <sub>%s</sub>\n\n", n.Text, noteMeta(n, time.Now()))
	}

	for _, h := range t.Hops {
		sha := "`" + h.Commit.ShortSHA() + "`"
		if t.Repo != "" {
			sha = fmt.Sprintf("[%s](https://github.com/%s/commit/%s)", sha, t.Repo, h.Commit.SHA)
		}
		fmt.Fprintf(w, "- %s **%s** — %s, %s\n", sha, h.Commit.Subject, h.Commit.Author, h.Commit.Date.Format("2006-01-02"))
		if h.PR != nil {
			fmt.Fprintf(w, "  - PR [#%d](%s): %s\n", h.PR.Number, h.PR.URL, h.PR.Title)
		}
		for _, is := range h.Issues {
			fmt.Fprintf(w, "  - closes [#%d](%s): %s\n", is.Number, is.URL, is.Title)
		}
	}

	if t.Notice != "" {
		fmt.Fprintf(w, "\n> %s\n", t.Notice)
	}
	fmt.Fprintf(w, "\n<sub>dug up with [`why`](https://github.com/pyjeebz/why)</sub>\n")
}

// nhops formats a hop count with its noun.
func nhops(n int) string {
	if n == 1 {
		return "1 hop"
	}
	return fmt.Sprintf("%d hops", n)
}
