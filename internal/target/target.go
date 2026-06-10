// Package target parses dig targets like "path/to/file.go:142" or
// "path/to/file.go:120-160" into a structured form.
package target

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Target identifies a region of a file to dig into. Start/End are 1-based
// line numbers; Start == 0 means the whole file.
type Target struct {
	Path  string `json:"path"`
	Start int    `json:"start,omitempty"`
	End   int    `json:"end,omitempty"`
}

// WholeFile reports whether the target covers the entire file rather than
// a line range.
func (t Target) WholeFile() bool { return t.Start == 0 }

func (t Target) String() string {
	switch {
	case t.WholeFile():
		return t.Path
	case t.Start == t.End:
		return fmt.Sprintf("%s:%d", t.Path, t.Start)
	default:
		return fmt.Sprintf("%s:%d-%d", t.Path, t.Start, t.End)
	}
}

var rangeRe = regexp.MustCompile(`^(\d+)(?:-(\d+))?$`)

// Parse interprets arg as FILE, FILE:LINE, or FILE:START-END. A trailing
// colon segment that is not a valid line spec is treated as part of the
// path (files with colons in their names are rare but legal).
func Parse(arg string) (Target, error) {
	if arg == "" {
		return Target{}, fmt.Errorf("empty target")
	}

	path := arg
	var m []string
	if i := strings.LastIndex(arg, ":"); i > 0 {
		if m = rangeRe.FindStringSubmatch(arg[i+1:]); m != nil {
			path = arg[:i]
		}
	}

	t := Target{Path: path}
	if m != nil {
		start, err := strconv.Atoi(m[1])
		if err != nil || start < 1 {
			return Target{}, fmt.Errorf("invalid line number %q", m[1])
		}
		end := start
		if m[2] != "" {
			end, err = strconv.Atoi(m[2])
			if err != nil || end < start {
				return Target{}, fmt.Errorf("invalid range %q: end must be >= start", m[0])
			}
		}
		t.Start, t.End = start, end
	}

	if info, err := os.Stat(t.Path); err != nil {
		return Target{}, fmt.Errorf("cannot read %s: %w", t.Path, err)
	} else if info.IsDir() {
		return Target{}, fmt.Errorf("%s is a directory; dig targets a file", t.Path)
	}
	return t, nil
}
