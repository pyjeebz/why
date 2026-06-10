package github

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pyjeebz/why/internal/trail"
)

const prResponse = `{"data":{"repository":{"object":{"associatedPullRequests":{"nodes":[
  {"number":4521,"title":"Raise retry budget","body":"Fixes flakiness.","url":"https://github.com/o/r/pull/4521",
   "closingIssuesReferences":{"nodes":[{"number":4312,"title":"Race in retry loop","url":"https://github.com/o/r/issues/4312"}]}}
]}}}}}`

const emptyResponse = `{"data":{"repository":{"object":{"associatedPullRequests":{"nodes":[]}}}}}`

func hopsFor(shas ...string) []trail.Hop {
	var hops []trail.Hop
	for _, sha := range shas {
		hops = append(hops, trail.Hop{Commit: trail.Commit{SHA: sha, Date: time.Now()}})
	}
	return hops
}

// fakeEnricher returns an enricher whose gh invocations are served by fn.
func fakeEnricher(fn func(oid string) ([]byte, error)) *Enricher {
	e := NewEnricher("o", "r")
	e.skipPathCheck = true
	e.run = func(args ...string) ([]byte, error) {
		for _, a := range args {
			if oid, found := strings.CutPrefix(a, "oid="); found {
				return fn(oid)
			}
		}
		return nil, errors.New("no oid argument")
	}
	return e
}

func TestEnrichFillsPRAndIssues(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	e := fakeEnricher(func(oid string) ([]byte, error) {
		if oid == "aaa" {
			return []byte(prResponse), nil
		}
		return []byte(emptyResponse), nil
	})

	hops := hopsFor("aaa", "bbb")
	if notice := e.Enrich(hops); notice != "" {
		t.Fatalf("unexpected notice %q", notice)
	}

	if hops[0].PR == nil || hops[0].PR.Number != 4521 {
		t.Fatalf("hop 0 PR = %+v, want #4521", hops[0].PR)
	}
	if len(hops[0].Issues) != 1 || hops[0].Issues[0].Number != 4312 {
		t.Fatalf("hop 0 issues = %+v, want #4312", hops[0].Issues)
	}
	if hops[1].PR != nil {
		t.Fatalf("hop 1 PR = %+v, want nil (no PR)", hops[1].PR)
	}
}

func TestEnrichUsesCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	calls := 0
	e := fakeEnricher(func(oid string) ([]byte, error) {
		calls++
		return []byte(prResponse), nil
	})
	if notice := e.Enrich(hopsFor("aaa")); notice != "" {
		t.Fatalf("unexpected notice %q", notice)
	}
	if calls != 1 {
		t.Fatalf("first enrich made %d calls, want 1", calls)
	}

	// A fresh enricher whose run always fails must still succeed from cache.
	e2 := fakeEnricher(func(oid string) ([]byte, error) {
		return nil, errors.New("network down")
	})
	hops := hopsFor("aaa")
	if notice := e2.Enrich(hops); notice != "" {
		t.Fatalf("cached enrich degraded: %q", notice)
	}
	if hops[0].PR == nil || hops[0].PR.Number != 4521 {
		t.Fatalf("cached hop PR = %+v, want #4521", hops[0].PR)
	}
}

func TestEnrichDegradesOnError(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	e := fakeEnricher(func(oid string) ([]byte, error) {
		return nil, fmt.Errorf("gh: HTTP 401 bad credentials")
	})
	hops := hopsFor("aaa", "bbb", "ccc")
	notice := e.Enrich(hops)
	if !strings.Contains(notice, "git-only") || !strings.Contains(notice, "401") {
		t.Fatalf("notice = %q, want degradation mentioning the cause", notice)
	}
	for i, h := range hops {
		if h.PR != nil {
			t.Fatalf("hop %d unexpectedly enriched: %+v", i, h.PR)
		}
	}
}
