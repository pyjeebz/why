package dig

import (
	"sort"
	"strings"

	"github.com/pyjeebz/why/internal/answer"
	"github.com/pyjeebz/why/internal/githist"
	"github.com/pyjeebz/why/internal/rank"
	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

// maxProbe caps how many candidate regions Survey deep-digs, so a heavily
// edited file costs a bounded number of history walks no matter its size.
const maxProbe = 12

// region is a contiguous run of lines that share the commit that last
// touched them — a candidate worth digging into.
type region struct {
	start, end int
	sha        string
	subject    string
}

// Survey finds the regions of a file worth knowing about. It blames the file
// (one cheap pass) to group lines by the commit that last touched them, deep-
// digs the most promising groups, collapses ones that share a trail, and
// ranks what is left by how load-bearing its history looks. It returns the
// top regions — each with its Answer when answers is set — and the total
// number of candidate regions the file breaks into.
func Survey(dir, path string, top, depth int, answers bool) (regions []trail.Trail, total int, err error) {
	bl, err := githist.Blame(dir, path)
	if err != nil {
		return nil, 0, err
	}
	cands := groupRegions(bl)
	total = len(cands)

	// Probe the most promising candidates first, and cap the cost.
	sort.SliceStable(cands, func(i, j int) bool { return proxyScore(cands[i]) > proxyScore(cands[j]) })
	if len(cands) > maxProbe {
		cands = cands[:maxProbe]
	}

	seen := map[string]bool{}
	for _, rg := range cands {
		tr, digErr := Run(dir, target.Target{Path: path, Start: rg.start, End: rg.end}, depth)
		if digErr != nil {
			continue // a region we cannot read is one we simply skip
		}
		sig := rank.Signature(tr)
		if seen[sig] || !rank.Meaningful(tr) {
			continue // duplicate story, or nothing load-bearing to say
		}
		seen[sig] = true
		if answers {
			tr.Answer = answer.For(tr, false)
		}
		regions = append(regions, tr)
	}

	sort.SliceStable(regions, func(i, j int) bool { return rank.Weight(regions[i]) > rank.Weight(regions[j]) })
	if len(regions) > top {
		regions = regions[:top]
	}
	return regions, total, nil
}

// groupRegions folds blame lines into contiguous same-commit runs, skipping
// lines with uncommitted changes (there is no history to dig for those).
func groupRegions(bl []githist.BlameLine) []region {
	zero := strings.Repeat("0", 40)
	var rs []region
	for _, l := range bl {
		if l.SHA == zero {
			continue
		}
		if n := len(rs); n > 0 && rs[n-1].sha == l.SHA && rs[n-1].end == l.N-1 {
			rs[n-1].end = l.N
			continue
		}
		rs = append(rs, region{start: l.N, end: l.N, sha: l.SHA, subject: l.Subject})
	}
	return rs
}

// proxyScore ranks candidates for probing before their full history is
// known: an incident-flavoured last touch jumps the queue, then larger runs.
func proxyScore(r region) int {
	score := r.end - r.start + 1
	if rank.Incident(r.subject) {
		score += 1000
	}
	return score
}
