package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

func sampleTrail() trail.Trail {
	date := time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC)
	return trail.Trail{
		Target: target.Target{Path: "internal/retry.go", Start: 142, End: 142},
		Repo:   "octo/widgets",
		Hops: []trail.Hop{
			{
				Commit: trail.Commit{SHA: "aaaa111bbbb222cccc333", Author: "Jane Doe", Date: date, Subject: "fix: add jitter to retry backoff", Body: "Thundering herd on cold start.\n\nDetails."},
				PR:     &trail.PR{Number: 4521, Title: "Raise retry budget", URL: "https://github.com/octo/widgets/pull/4521"},
				Issues: []trail.Issue{{Number: 4312, Title: "Race in retry loop", URL: "https://github.com/octo/widgets/issues/4312"}},
			},
			{
				Commit: trail.Commit{SHA: "dddd444eeee555ffff666", Author: "Sam Roe", Date: date.AddDate(0, -2, 0), Subject: "feat: retry layer"},
			},
		},
	}
}

func TestMarkdownLinksAndFooter(t *testing.T) {
	var buf bytes.Buffer
	Markdown(&buf, sampleTrail())
	out := buf.String()

	for _, want := range []string{
		"### why · `internal/retry.go:142` — 2 hops, newest first",
		"[`aaaa111`](https://github.com/octo/widgets/commit/aaaa111bbbb222cccc333)",
		"**fix: add jitter to retry backoff** — Jane Doe, 2026-05-14",
		"- PR [#4521](https://github.com/octo/widgets/pull/4521): Raise retry budget",
		"- closes [#4312](https://github.com/octo/widgets/issues/4312): Race in retry loop",
		"<sub>dug up with [`why`](https://github.com/pyjeebz/why)</sub>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("markdown missing %q in:\n%s", want, out)
		}
	}
}

func TestMarkdownWithoutSlugOrEnrichment(t *testing.T) {
	tr := sampleTrail()
	tr.Repo = ""
	tr.Hops = tr.Hops[1:]
	tr.Notice = "trail is git-only: no GitHub origin remote"

	var buf bytes.Buffer
	Markdown(&buf, tr)
	out := buf.String()

	if !strings.Contains(out, "— 1 hop, newest first") {
		t.Fatalf("singular hop count missing in:\n%s", out)
	}
	if strings.Contains(out, "github.com/octo") {
		t.Fatalf("commit link rendered without a repo slug:\n%s", out)
	}
	if !strings.Contains(out, "- `dddd444` **feat: retry layer**") {
		t.Fatalf("plain sha missing in:\n%s", out)
	}
	if !strings.Contains(out, "> trail is git-only: no GitHub origin remote") {
		t.Fatalf("notice blockquote missing in:\n%s", out)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, sampleTrail()); err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	var got trail.Trail
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got.Repo != "octo/widgets" || len(got.Hops) != 2 {
		t.Fatalf("round-trip lost data: repo=%q hops=%d", got.Repo, len(got.Hops))
	}
	if got.Hops[0].PR == nil || got.Hops[0].PR.Number != 4521 {
		t.Fatalf("round-trip PR = %+v, want #4521", got.Hops[0].PR)
	}

	out := buf.String()
	if strings.Contains(out, `"notice"`) {
		t.Fatalf("empty notice serialized:\n%s", out)
	}
	if strings.Count(out, `"pr"`) != 1 {
		t.Fatalf("nil PR should be omitted, got:\n%s", out)
	}
}

func TestTermPlainOutput(t *testing.T) {
	var buf bytes.Buffer
	tr := sampleTrail()
	tr.Notice = "trail is partly git-only: HTTP 401"
	Term(&buf, tr, false)
	out := buf.String()

	for _, want := range []string{
		"why · trail for internal/retry.go:142 — 2 hops, newest first",
		"● aaaa111  2026-05-14  Jane Doe",
		"  fix: add jitter to retry backoff",
		"  └ PR #4521: Raise retry budget",
		"    https://github.com/octo/widgets/pull/4521",
		"  └ closes #4312: Race in retry loop",
		"  Thundering herd on cold start.",
		"· trail is partly git-only: HTTP 401",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("term output missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\033[") {
		t.Fatalf("escape codes leaked into plain output:\n%s", out)
	}
}

func TestTermColorOutput(t *testing.T) {
	var buf bytes.Buffer
	Term(&buf, sampleTrail(), true)
	if !strings.Contains(buf.String(), yellow+"aaaa111"+reset) {
		t.Fatalf("colored sha missing in:\n%s", buf.String())
	}
}
