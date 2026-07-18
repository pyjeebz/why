package githist

import (
	"fmt"
	"regexp"
	"strings"
)

// BlameLine pairs a file line with the commit that last touched it and that
// commit's subject — enough to survey a whole file cheaply, without walking
// any single line's full history.
type BlameLine struct {
	N       int
	SHA     string
	Subject string
}

// Blame returns one BlameLine per line of path, in file order. It is the
// cheap survey pass: a single `git blame` yields the whole file's provenance.
func Blame(dir, path string) ([]BlameLine, error) {
	out, err := run(dir, "blame", "--line-porcelain", "--", path)
	if err != nil {
		return nil, err
	}
	lines := parseBlame(out)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no blame history for %s", path)
	}
	return lines, nil
}

var blameHeader = regexp.MustCompile(`^([0-9a-f]{40}) \d+ \d+`)

// parseBlame reads git blame --line-porcelain: each line is preceded by a
// header (sha, source and result line numbers) and a repeated commit block,
// then the content line itself, prefixed by a tab.
func parseBlame(out string) []BlameLine {
	var lines []BlameLine
	var sha, subject string
	for line := range strings.SplitSeq(out, "\n") {
		if m := blameHeader.FindStringSubmatch(line); m != nil {
			sha, subject = m[1], ""
			continue
		}
		if s, ok := strings.CutPrefix(line, "summary "); ok {
			subject = s
			continue
		}
		if strings.HasPrefix(line, "\t") {
			lines = append(lines, BlameLine{N: len(lines) + 1, SHA: sha, Subject: subject})
		}
	}
	return lines
}
