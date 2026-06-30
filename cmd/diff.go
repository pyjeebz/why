package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pyjeebz/why/internal/dig"
	"github.com/pyjeebz/why/internal/github"
	"github.com/pyjeebz/why/internal/githist"
	"github.com/pyjeebz/why/internal/render"
	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

var (
	diffDepth   int
	diffMax     int
	diffComment bool
	diffPR      int
	diffRepo    string
	diffDryRun  bool
	diffNudge   bool
	diffContext int
)

var diffCmd = &cobra.Command{
	Use:   "diff [BASE]",
	Short: "Explain the history behind every region a change touches",
	Long: `Reads a diff, then digs the decision trail behind each region it touches
and renders them as a single comment — the why behind the code you are
about to change, before you change it.

With no argument it inspects your working tree against HEAD (run it before
you push). Given a BASE ref it inspects BASE...HEAD — the change a pull
request introduces — which is how the GitHub Action drives it in CI.`,
	Example: `  why diff                 # working-tree changes vs HEAD
  why diff origin/main     # everything this branch changes since main`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		base := ""
		rng := "HEAD"
		if len(args) == 1 {
			base = args[0]
			rng = base + "...HEAD"
		}
		regions, err := changedRegions(cwd, rng, diffContext)
		if err != nil {
			return err
		}
		if len(regions) == 0 {
			fmt.Fprintln(os.Stderr, "why · no changed regions to dig")
			return nil
		}

		exclude := prCommits(cwd, base)
		trails, considered, skipped := collectTrails(cwd, regions, exclude, diffDepth, diffMax)
		if skipped > 0 {
			fmt.Fprintf(os.Stderr, "why · skipped %d of %d region(s) whose history could not be read\n", skipped, considered)
		}
		// All considered regions failing to dig is a read failure, not an
		// absence of history — say so loudly rather than posting a comment
		// that misreports it as "no recorded history".
		if considered > 0 && len(trails) == 0 {
			return fmt.Errorf("could not read history for any of the %d changed region(s)", considered)
		}

		var buf bytes.Buffer
		render.Comment(&buf, trails, diffNudge)
		if diffComment {
			return postComment(cwd, buf.String(), diffPR, diffRepo, diffDryRun)
		}
		fmt.Print(buf.String())
		return nil
	},
}

var hunkRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// changedRegions parses `git diff` for the line ranges a change adds or
// modifies, one Target per hunk, each widened by ctx lines of surrounding
// context. It reads the new side of each hunk (the lines present after the
// change), so the targets line up with the working tree the dig will walk.
// The context margin is what lets an insertion — whose own new line has no
// past — pick up the history of the code it lands among. Pure deletions,
// which leave nothing to point at, are skipped.
func changedRegions(dir, rng string, ctx int) ([]target.Target, error) {
	out, err := exec.Command("git", "-C", dir, "diff", "--unified=0", "--no-color", rng).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff %s: %w", rng, err)
	}

	var regions []target.Target
	var file string
	for line := range strings.SplitSeq(string(out), "\n") {
		if p, ok := strings.CutPrefix(line, "+++ "); ok {
			if p == "/dev/null" {
				file = ""
			} else {
				file = strings.TrimPrefix(p, "b/")
			}
			continue
		}
		if file == "" {
			continue
		}
		m := hunkRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		start, _ := strconv.Atoi(m[1])
		count := 1
		if m[2] != "" {
			count, _ = strconv.Atoi(m[2])
		}
		if count == 0 {
			continue // pure deletion: no lines remain in the new tree
		}
		regions = append(regions, target.Target{Path: file, Start: start, End: start + count - 1})
	}
	return coalesce(expand(dir, regions, ctx)), nil
}

// runDig is a seam over dig.Run so the gather loop can be tested without a
// repository.
var runDig = dig.Run

// collectTrails digs each region (up to max), excluding the change's own
// commits, and reports how many regions were considered and how many had to
// be skipped because their history could not be read. Separating the counts
// lets the caller tell "no history" apart from "could not read it".
func collectTrails(dir string, regions []target.Target, exclude map[string]bool, depth, max int) (trails []trail.Trail, considered, skipped int) {
	for i, rg := range regions {
		if i >= max {
			break
		}
		considered++
		tr, err := runDig(dir, rg, depth)
		if err != nil {
			skipped++
			continue
		}
		dropCommits(&tr, exclude)
		trails = append(trails, tr)
	}
	return trails, considered, skipped
}

// expand widens each region by ctx lines on both sides, clamped to the file
// so a dig near the end of a file never runs off it.
func expand(dir string, regions []target.Target, ctx int) []target.Target {
	if ctx <= 0 {
		return regions
	}
	lines := map[string]int{}
	for i := range regions {
		r := &regions[i]
		if r.WholeFile() {
			continue
		}
		if r.Start -= ctx; r.Start < 1 {
			r.Start = 1
		}
		n, ok := lines[r.Path]
		if !ok {
			n = lineCount(dir, r.Path)
			lines[r.Path] = n
		}
		if r.End += ctx; n > 0 && r.End > n {
			r.End = n
		}
	}
	return regions
}

// lineCount returns the number of lines in a working-tree file, or 0 if it
// cannot be read.
func lineCount(dir, path string) int {
	b, err := os.ReadFile(filepath.Join(dir, path))
	if err != nil || len(b) == 0 {
		return 0
	}
	n := bytes.Count(b, []byte{'\n'})
	if b[len(b)-1] != '\n' {
		n++
	}
	return n
}

// prCommits returns the set of commit SHAs the change under review
// introduces (base..HEAD), so the trail can exclude the very commits being
// reviewed and never report a change back at its own author. Empty when
// there is no base — working-tree mode has no committed change to exclude.
func prCommits(dir, base string) map[string]bool {
	set := map[string]bool{}
	if base == "" {
		return set
	}
	out, err := exec.Command("git", "-C", dir, "rev-list", base+"..HEAD").Output()
	if err != nil {
		return set
	}
	for sha := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if sha != "" {
			set[sha] = true
		}
	}
	return set
}

// dropCommits removes hops whose commit is in the exclude set, in place.
func dropCommits(tr *trail.Trail, exclude map[string]bool) {
	if len(exclude) == 0 {
		return
	}
	kept := tr.Hops[:0]
	for _, h := range tr.Hops {
		if !exclude[h.Commit.SHA] {
			kept = append(kept, h)
		}
	}
	tr.Hops = kept
}

// coalesceGap is how many unchanged lines two hunks in the same file may be
// apart before why treats them as one region rather than two.
const coalesceGap = 5

// coalesce merges hunks in the same file that sit within coalesceGap lines
// of each other, so a cluster of small edits reads as one region instead of
// fragmenting into several near-identical sections. Hunks arrive in file
// order from git diff, so a single forward pass suffices.
func coalesce(regions []target.Target) []target.Target {
	out := regions[:0:0]
	for _, r := range regions {
		if n := len(out); n > 0 {
			last := &out[n-1]
			if last.Path == r.Path && r.Start <= last.End+coalesceGap+1 {
				if r.End > last.End {
					last.End = r.End
				}
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// postComment posts the body as a comment on a pull request, updating why's
// own previous comment in place — found by its marker — instead of adding a
// new one on every push. The repository and PR number are taken from flags,
// then the CI environment, then the git remote, so the same command works at
// a desk and in a workflow.
func postComment(dir, body string, prFlag int, repoFlag string, dryRun bool) error {
	slug, ok := resolveRepo(dir, repoFlag)
	if !ok {
		return fmt.Errorf("could not determine repository; pass --repo owner/name or set GITHUB_REPOSITORY")
	}
	pr := resolvePR(prFlag)
	if pr <= 0 {
		return fmt.Errorf("could not determine pull request number; pass --pr N")
	}

	listPath := fmt.Sprintf("repos/%s/issues/%d/comments", slug, pr)
	jq := fmt.Sprintf(`.[] | select(.body | contains("%s")) | .id`, render.CommentMarker)

	if dryRun {
		fmt.Fprintf(os.Stderr, "why · dry run — would comment on %s#%d (%d bytes)\n", slug, pr, len(body))
		fmt.Fprintf(os.Stderr, "  find:   gh api %s --paginate --jq '%s'\n", listPath, jq)
		fmt.Fprintf(os.Stderr, "  update: gh api repos/%s/issues/comments/<id> --method PATCH --input <body>\n", slug)
		fmt.Fprintf(os.Stderr, "  create: gh api %s --method POST --input <body>\n\n", listPath)
		fmt.Print(body)
		return nil
	}

	existing, err := runGH("api", listPath, "--paginate", "--jq", jq)
	if err != nil {
		return fmt.Errorf("listing PR comments: %w", err)
	}

	tmp, err := writeCommentBody(body)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)

	if id := firstLine(existing); id != "" {
		if _, err := runGH("api", fmt.Sprintf("repos/%s/issues/comments/%s", slug, id), "--method", "PATCH", "--input", tmp); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "why · updated comment on %s#%d\n", slug, pr)
		return nil
	}
	if _, err := runGH("api", listPath, "--method", "POST", "--input", tmp); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "why · posted comment on %s#%d\n", slug, pr)
	return nil
}

// resolveRepo determines the owner/name slug: an explicit flag wins, then
// the GITHUB_REPOSITORY the Actions runner sets, then the git origin remote.
func resolveRepo(dir, flag string) (string, bool) {
	if flag != "" {
		return flag, strings.Count(flag, "/") == 1
	}
	if env := os.Getenv("GITHUB_REPOSITORY"); env != "" {
		return env, true
	}
	if owner, repo, ok := github.ParseRemote(githist.RemoteURL(dir)); ok {
		return owner + "/" + repo, true
	}
	return "", false
}

// resolvePR determines the PR number: an explicit flag wins, otherwise the
// GITHUB_REF a pull_request workflow sets (refs/pull/<n>/merge).
func resolvePR(flag int) int {
	if flag > 0 {
		return flag
	}
	if rest, ok := strings.CutPrefix(os.Getenv("GITHUB_REF"), "refs/pull/"); ok {
		if i := strings.IndexByte(rest, '/'); i > 0 {
			if n, err := strconv.Atoi(rest[:i]); err == nil {
				return n
			}
		}
	}
	return 0
}

// writeCommentBody writes a {"body": ...} payload to a temp file for gh's
// --input, so the markdown is JSON-escaped exactly once and never has to
// survive a shell.
func writeCommentBody(body string) (string, error) {
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "why-comment-*.json")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(payload); err != nil {
		f.Close()
		return "", err
	}
	return f.Name(), f.Close()
}

func runGH(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

func init() {
	diffCmd.Flags().IntVar(&diffDepth, "depth", 8, "maximum hops back through history per region")
	diffCmd.Flags().IntVar(&diffMax, "max-regions", 25, "cap on how many changed regions to dig")
	diffCmd.Flags().BoolVar(&diffComment, "comment", false, "post (or update) the trail as a comment on a pull request")
	diffCmd.Flags().IntVar(&diffPR, "pr", 0, "pull request number to comment on (default: inferred from CI env)")
	diffCmd.Flags().StringVar(&diffRepo, "repo", "", "owner/name of the repository (default: inferred from remote or CI env)")
	diffCmd.Flags().BoolVar(&diffDryRun, "dry-run", false, "with --comment, print the plan and body instead of calling gh")
	diffCmd.Flags().BoolVar(&diffNudge, "nudge", true, "when some touched code has no recorded reason, invite the author to record it")
	diffCmd.Flags().IntVar(&diffContext, "context", 3, "lines of surrounding context to dig around each change")
	rootCmd.AddCommand(diffCmd)
}
