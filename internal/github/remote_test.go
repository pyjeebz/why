package github

import "testing"

func TestParseRemote(t *testing.T) {
	tests := []struct {
		url         string
		owner, repo string
		ok          bool
	}{
		{"https://github.com/pyjeebz/why.git", "pyjeebz", "why", true},
		{"https://github.com/pyjeebz/why", "pyjeebz", "why", true},
		{"git@github.com:opentofu/opentofu.git", "opentofu", "opentofu", true},
		{"ssh://git@github.com/cli/cli.git", "cli", "cli", true},
		{"https://github.com/pyjeebz/why/", "pyjeebz", "why", true},
		{"https://gitlab.com/group/proj.git", "", "", false},
		{"git@bitbucket.org:o/r.git", "", "", false},
		{"https://github.com/justowner", "", "", false},
		{"", "", "", false},
	}
	for _, tc := range tests {
		owner, repo, ok := ParseRemote(tc.url)
		if owner != tc.owner || repo != tc.repo || ok != tc.ok {
			t.Errorf("ParseRemote(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.url, owner, repo, ok, tc.owner, tc.repo, tc.ok)
		}
	}
}
