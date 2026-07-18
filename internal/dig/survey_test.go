package dig

import (
	"strings"
	"testing"

	"github.com/pyjeebz/why/internal/githist"
)

func TestGroupRegions_contiguousAndSkipsUncommitted(t *testing.T) {
	const a = "1111111111111111111111111111111111111111"
	const b = "2222222222222222222222222222222222222222"
	zero := strings.Repeat("0", 40)

	bl := []githist.BlameLine{
		{N: 1, SHA: a, Subject: "fix"},
		{N: 2, SHA: a, Subject: "fix"},
		{N: 3, SHA: b, Subject: "refactor"},
		{N: 4, SHA: a, Subject: "fix"}, // same commit, non-contiguous → its own region
		{N: 5, SHA: zero},              // uncommitted → skipped
	}

	rs := groupRegions(bl)
	if len(rs) != 3 {
		t.Fatalf("want 3 regions, got %d: %+v", len(rs), rs)
	}
	if rs[0].start != 1 || rs[0].end != 2 || rs[0].sha != a {
		t.Errorf("region 0 = %+v, want lines 1-2 of %s", rs[0], a)
	}
	if rs[2].start != 4 || rs[2].end != 4 {
		t.Errorf("region 2 = %+v, want line 4 alone", rs[2])
	}
}

func TestProxyScore_incidentJumpsQueue(t *testing.T) {
	incident := region{start: 1, end: 1, subject: "fix: race condition"}
	big := region{start: 1, end: 50, subject: "reformat file"}
	if proxyScore(incident) <= proxyScore(big) {
		t.Errorf("an incident region should outrank a large plain one: %d vs %d", proxyScore(incident), proxyScore(big))
	}
}
