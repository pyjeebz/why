// Package trail defines the data model for a dig result: the ordered set
// of history hops that shaped a region of code.
package trail

import (
	"time"

	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/target"
)

// Commit is one git commit touching the target region.
type Commit struct {
	SHA     string    `json:"sha"`
	Author  string    `json:"author"`
	Email   string    `json:"email"`
	Date    time.Time `json:"date"`
	Subject string    `json:"subject"`
	Body    string    `json:"body,omitempty"`
}

// ShortSHA returns the abbreviated commit hash.
func (c Commit) ShortSHA() string {
	if len(c.SHA) > 7 {
		return c.SHA[:7]
	}
	return c.SHA
}

// PR is the pull request that introduced a commit. Populated by the
// GitHub enrichment layer; nil when unavailable.
type PR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body,omitempty"`
	URL    string `json:"url"`
}

// Issue is an issue linked to a PR via closing references.
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// Hop is one step back through history: a commit plus whatever context
// could be recovered around it.
type Hop struct {
	Commit Commit  `json:"commit"`
	PR     *PR     `json:"pr,omitempty"`
	Issues []Issue `json:"issues,omitempty"`
}

// Trail is the full dig result, hops ordered newest first. Repo is the
// "owner/repo" slug when a GitHub origin remote was recognized — empty
// otherwise — and is what renderers use to build commit links. Notice
// describes how much context could be recovered when the trail is
// degraded (git-only, partial enrichment).
type Trail struct {
	Target target.Target `json:"target"`
	Repo   string        `json:"repo,omitempty"`
	Notes  []notes.Note  `json:"notes,omitempty"`
	Hops   []Hop         `json:"hops"`
	Notice string        `json:"notice,omitempty"`
}
