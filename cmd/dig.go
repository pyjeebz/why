package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/pyjeebz/why/internal/githist"
	"github.com/pyjeebz/why/internal/github"
	"github.com/pyjeebz/why/internal/render"
	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

var (
	digDepth int
	digJSON  bool
	digShare bool
)

var digCmd = &cobra.Command{
	Use:   "dig FILE[:LINE | :START-END]",
	Short: "Dig up the decision trail behind a region of code",
	Example: `  why dig internal/retry.go:142
  why dig config/deploy.yaml:20-35
  why dig Makefile`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := target.Parse(args[0])
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if _, err := githist.RepoRoot(cwd); err != nil {
			return err
		}

		commits, err := githist.Walk(cwd, t, digDepth)
		if err != nil {
			return err
		}

		tr := trail.Trail{Target: t}
		for _, c := range commits {
			tr.Hops = append(tr.Hops, trail.Hop{Commit: c})
		}

		if owner, repo, ok := github.ParseRemote(githist.RemoteURL(cwd)); ok {
			tr.Repo = owner + "/" + repo
			tr.Notice = github.NewEnricher(owner, repo).Enrich(tr.Hops)
		} else {
			tr.Notice = "trail is git-only: no GitHub origin remote"
		}

		switch {
		case digJSON:
			return render.JSON(os.Stdout, tr)
		case digShare:
			render.Markdown(os.Stdout, tr)
		default:
			color := term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == ""
			render.Term(os.Stdout, tr, color)
		}
		return nil
	},
}

func init() {
	digCmd.Flags().IntVar(&digDepth, "depth", 8, "maximum hops back through history")
	digCmd.Flags().BoolVar(&digJSON, "json", false, "emit the trail as JSON")
	digCmd.Flags().BoolVar(&digShare, "share", false, "emit the trail as markdown for a PR or issue comment")
	digCmd.MarkFlagsMutuallyExclusive("json", "share")
	rootCmd.AddCommand(digCmd)
}
