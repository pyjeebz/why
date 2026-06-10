package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/pyjeebz/why/internal/cache"
	"github.com/pyjeebz/why/internal/trail"
)

// query resolves a commit to its PR and that PR's closing issues in a
// single API call.
const query = `query($owner:String!,$name:String!,$oid:GitObjectID!){
  repository(owner:$owner,name:$name){
    object(oid:$oid){
      ... on Commit{
        associatedPullRequests(first:1){
          nodes{
            number title body url
            closingIssuesReferences(first:5){nodes{number title url}}
          }
        }
      }
    }
  }
}`

// Enrichment is the recovered context for one commit. PR is nil when the
// commit reached the default branch without a pull request.
type Enrichment struct {
	PR     *trail.PR     `json:"pr"`
	Issues []trail.Issue `json:"issues,omitempty"`
}

// Enricher fills PR/issue context for trail hops, cache-first.
type Enricher struct {
	Owner, Repo string

	// run executes gh with the given arguments; swapped out in tests,
	// along with skipPathCheck.
	run           func(args ...string) ([]byte, error)
	skipPathCheck bool

	mu     sync.Mutex
	reason string // first failure; once set, no further API calls are made
}

// NewEnricher returns an enricher for the given GitHub repository.
func NewEnricher(owner, repo string) *Enricher {
	return &Enricher{
		Owner: owner,
		Repo:  repo,
		run: func(args ...string) ([]byte, error) {
			out, err := exec.Command("gh", args...).Output()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
					return nil, fmt.Errorf("%s", firstLine(string(ee.Stderr)))
				}
				return nil, err
			}
			return out, nil
		},
	}
}

// Enrich fills hops in place and returns a notice describing any
// degradation ("" when fully enriched).
func (e *Enricher) Enrich(hops []trail.Hop) string {
	if !e.skipPathCheck {
		if _, err := exec.LookPath("gh"); err != nil {
			return "trail is git-only: gh is not installed"
		}
	}

	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i := range hops {
		wg.Add(1)
		go func(h *trail.Hop) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			enr, ok := e.lookup(h.Commit.SHA)
			if !ok {
				return
			}
			h.PR = enr.PR
			h.Issues = enr.Issues
		}(&hops[i])
	}
	wg.Wait()

	if e.reason != "" {
		return "trail is partly git-only: " + e.reason
	}
	return ""
}

// lookup returns the enrichment for one commit, consulting the cache
// before the API.
func (e *Enricher) lookup(sha string) (Enrichment, bool) {
	key := fmt.Sprintf("github/%s__%s/%s.json", e.Owner, e.Repo, sha)

	var enr Enrichment
	if cache.Get(key, &enr) {
		return enr, true
	}

	e.mu.Lock()
	disabled := e.reason != ""
	e.mu.Unlock()
	if disabled {
		return Enrichment{}, false
	}

	out, err := e.run("api", "graphql",
		"-f", "query="+query,
		"-f", "owner="+e.Owner,
		"-f", "name="+e.Repo,
		"-f", "oid="+sha)
	if err != nil {
		e.mu.Lock()
		if e.reason == "" {
			e.reason = err.Error()
		}
		e.mu.Unlock()
		return Enrichment{}, false
	}

	enr = parseResponse(out)
	_ = cache.Put(key, enr)
	return enr, true
}

func parseResponse(data []byte) Enrichment {
	var resp struct {
		Data struct {
			Repository struct {
				Object struct {
					AssociatedPullRequests struct {
						Nodes []struct {
							Number                  int    `json:"number"`
							Title                   string `json:"title"`
							Body                    string `json:"body"`
							URL                     string `json:"url"`
							ClosingIssuesReferences struct {
								Nodes []trail.Issue `json:"nodes"`
							} `json:"closingIssuesReferences"`
						} `json:"nodes"`
					} `json:"associatedPullRequests"`
				} `json:"object"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return Enrichment{}
	}
	nodes := resp.Data.Repository.Object.AssociatedPullRequests.Nodes
	if len(nodes) == 0 {
		return Enrichment{}
	}
	n := nodes[0]
	return Enrichment{
		PR:     &trail.PR{Number: n.Number, Title: n.Title, Body: n.Body, URL: n.URL},
		Issues: n.ClosingIssuesReferences.Nodes,
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
