// Package github recovers PR and issue context for commits using the
// GitHub API via the gh CLI. Everything here is optional: when gh or a
// GitHub remote is missing, trails degrade to git-only.
package github

import "strings"

// ParseRemote extracts owner and repo from a GitHub remote URL in https,
// ssh, or scp-like form. ok is false for non-GitHub remotes.
func ParseRemote(url string) (owner, repo string, ok bool) {
	rest := strings.TrimSpace(url)
	switch {
	case strings.HasPrefix(rest, "https://github.com/"):
		rest = strings.TrimPrefix(rest, "https://github.com/")
	case strings.HasPrefix(rest, "http://github.com/"):
		rest = strings.TrimPrefix(rest, "http://github.com/")
	case strings.HasPrefix(rest, "git@github.com:"):
		rest = strings.TrimPrefix(rest, "git@github.com:")
	case strings.HasPrefix(rest, "ssh://git@github.com/"):
		rest = strings.TrimPrefix(rest, "ssh://git@github.com/")
	default:
		return "", "", false
	}
	rest = strings.TrimSuffix(rest, "/")
	rest = strings.TrimSuffix(rest, ".git")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
