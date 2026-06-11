// Package githist walks git history for a target region using git's
// line-log (git log -L), which follows the region across renames and
// rewrites, falling back to --follow for whole-file targets.
package githist

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
	format    = "%H" + fieldSep + "%an" + fieldSep + "%ae" + fieldSep + "%aI" + fieldSep + "%s" + fieldSep + "%b" + recordSep
)

// RepoRoot returns the top-level directory of the repository containing
// dir, or an error if dir is not inside a git work tree.
func RepoRoot(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository (%w)", err)
	}
	return strings.TrimSpace(out), nil
}

// RemoteURL returns the origin remote URL, or "" when there is none.
func RemoteURL(dir string) string {
	out, err := run(dir, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// UserName returns the configured git author name, or "" when unset.
func UserName(dir string) string {
	out, err := run(dir, "config", "user.name")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// Head returns the current HEAD commit SHA, or "" when unresolvable.
func Head(dir string) string {
	out, err := run(dir, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// Walk returns the commits that shaped the target region, newest first,
// capped at maxHops.
func Walk(dir string, t target.Target, maxHops int) ([]trail.Commit, error) {
	args := []string{"log", "--no-patch", "--format=" + format, "--max-count=" + strconv.Itoa(maxHops)}
	if t.WholeFile() {
		args = append(args, "--follow", "--", t.Path)
	} else {
		args = append(args, fmt.Sprintf("-L%d,%d:%s", t.Start, t.End, t.Path))
	}

	out, err := run(dir, args...)
	if err != nil {
		return nil, friendlyLogError(err, t)
	}
	return parseLog(out)
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}

func parseLog(out string) ([]trail.Commit, error) {
	var commits []trail.Commit
	for _, rec := range strings.Split(out, recordSep) {
		rec = strings.TrimLeft(rec, "\n")
		if strings.TrimSpace(rec) == "" {
			continue
		}
		fields := strings.SplitN(rec, fieldSep, 6)
		if len(fields) != 6 {
			return nil, fmt.Errorf("unexpected git log record: %q", rec)
		}
		date, err := time.Parse(time.RFC3339, fields[3])
		if err != nil {
			return nil, fmt.Errorf("bad commit date %q: %w", fields[3], err)
		}
		commits = append(commits, trail.Commit{
			SHA:     fields[0],
			Author:  fields[1],
			Email:   fields[2],
			Date:    date,
			Subject: fields[4],
			Body:    strings.TrimSpace(fields[5]),
		})
	}
	if len(commits) == 0 {
		return nil, fmt.Errorf("no history found for target")
	}
	return commits, nil
}

// friendlyLogError rewrites the most common git log -L failures into
// messages phrased in why's terms.
func friendlyLogError(err error, t target.Target) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "has only"):
		return fmt.Errorf("%s: requested lines %d-%d are past the end of the file", t.Path, t.Start, t.End)
	case strings.Contains(msg, "exists on disk, but not in"),
		strings.Contains(msg, "There is no path"):
		return fmt.Errorf("%s is not tracked by git (new or ignored file)", t.Path)
	default:
		return fmt.Errorf("git log failed: %s", msg)
	}
}
