# why

**`git blame` tells you who. `why` tells you why.**

`why` recovers the decision trail behind any region of code: the commits
that shaped it, the pull requests that introduced them, and the issues
behind those — newest first, in your terminal, with zero setup.

```
$ why dig internal/retry.go:142

why · trail for internal/retry.go:142 — 3 hops, newest first

● 49647fb  2026-03-02  Jane Doe
  fix: add jitter to retry backoff
  └ PR #4521: Raise retry budget
    https://github.com/octo/widgets/pull/4521
  └ closes #4312: Race in retry loop
    https://github.com/octo/widgets/issues/4312
  Thundering herd on cold start.
...
```

## Install

```
go install github.com/pyjeebz/why@latest
```

Or grab a static binary from the [releases page](../../releases).

`why` needs `git`. PR/issue enrichment uses the [`gh` CLI](https://cli.github.com)
when it is installed and authenticated; without it you still get the
full commit trail, just git-only.

## Usage

```
why dig FILE[:LINE | :START-END]    # the decision trail behind a region
    --share                         #   as markdown, for pasting into a PR/issue comment
    --json                          #   machine-readable
    --depth N                       #   hops back through history (default 8)

why diff [BASE]                     # the trail behind every region a change touches
    --comment --pr N                #   post it as one PR comment (updates in place)
    --dry-run                       #   print the comment instead of posting

why note FILE:LINE -m "..."         # attach a personal note to a region
    --review-by YYYY-MM-DD          #   date after which the note needs review
why log [FILE[:LINE | :START-END]]  # list your notes for this repository

why summarize FILE:LINE             # LLM-draft the why from the trail
    --save                          #   keep the draft as an inferred note

why mcp                             # MCP server so agents can dig too
```

## On every pull request

`why diff` reads a change and digs the decision trail behind every region
it touches — the why behind the code you are about to change, before you
change it. Run it against your working tree before you push, or add the
Action and it comments the trail on each pull request automatically:

```yaml
# .github/workflows/why.yml
name: why
on: pull_request
permissions:
  contents: read
  pull-requests: write
jobs:
  context:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }   # why walks line history
      - uses: pyjeebz/why@v1
```

It stays quiet by design: regions that share a trail collapse into one, what
remains ranks by how load-bearing its history is, and it says nothing when
there is nothing to say. When a change touches code whose reasoning was
never recorded, it invites you to record it — in the commit or PR, where the
next dig will find it.

## Your notes live with you, not the repo

You contribute to repositories you don't own. Notes are stored in your
personal overlay (XDG data dir, keyed by repository), so you can annotate
upstream code freely — no commit, no PR, no permission needed. They
surface at the top of later digs of the same region, and a `--review-by`
date marks a note as overdue once it expires: notes are claims that age,
not permanent truth.

## For agents

Your agent knows what the code says; `why` tells it how it got that way.
Add the MCP server and agents can check the trail before changing a
constant they don't understand:

```json
{"mcpServers": {"why": {"command": "why", "args": ["mcp"]}}}
```

## How it works

`git log -L` walks the region's history (following renames), then one
GraphQL call per commit via `gh` recovers the associated PR and its
closing issues — four-way parallel, cached in `~/.cache/why`, so a warm
dig is instant. Everything degrades gracefully: no `gh`, no auth, or no
GitHub remote just means a git-only trail with a notice saying so.

## License

Apache-2.0
