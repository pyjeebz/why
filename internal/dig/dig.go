// Package dig orchestrates a dig: walk git history, enrich from GitHub,
// and overlay personal notes.
package dig

import (
	"github.com/pyjeebz/why/internal/githist"
	"github.com/pyjeebz/why/internal/github"
	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

// Run digs the target region in the repository containing dir.
func Run(dir string, t target.Target, depth int) (trail.Trail, error) {
	root, err := githist.RepoRoot(dir)
	if err != nil {
		return trail.Trail{}, err
	}

	commits, err := githist.Walk(dir, t, depth)
	if err != nil {
		return trail.Trail{}, err
	}

	tr := trail.Trail{Target: t}
	for _, c := range commits {
		tr.Hops = append(tr.Hops, trail.Hop{Commit: c})
	}

	var slug string
	if owner, repo, ok := github.ParseRemote(githist.RemoteURL(dir)); ok {
		slug = owner + "/" + repo
		tr.Repo = slug
		tr.Notice = github.NewEnricher(owner, repo).Enrich(tr.Hops)
	} else {
		tr.Notice = "trail is git-only: no GitHub origin remote"
	}

	if ns, err := notes.For(notes.RepoKey(slug, root), t); err == nil {
		tr.Notes = ns
	}
	return tr, nil
}

// RepoKey returns the overlay-store key for the repository containing dir.
func RepoKey(dir string) (string, error) {
	root, err := githist.RepoRoot(dir)
	if err != nil {
		return "", err
	}
	var slug string
	if owner, repo, ok := github.ParseRemote(githist.RemoteURL(dir)); ok {
		slug = owner + "/" + repo
	}
	return notes.RepoKey(slug, root), nil
}
