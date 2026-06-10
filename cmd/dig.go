package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/pyjeebz/why/internal/githist"
	"github.com/pyjeebz/why/internal/render"
	"github.com/pyjeebz/why/internal/target"
	"github.com/pyjeebz/why/internal/trail"
)

var digDepth int

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
		root, err := githist.RepoRoot(cwd)
		if err != nil {
			return err
		}

		commits, err := githist.Walk(cwd, t, digDepth)
		if err != nil {
			return err
		}

		tr := trail.Trail{Target: t, Repo: root}
		for _, c := range commits {
			tr.Hops = append(tr.Hops, trail.Hop{Commit: c})
		}

		color := term.IsTerminal(int(os.Stdout.Fd()))
		render.Term(os.Stdout, tr, color)
		return nil
	},
}

func init() {
	digCmd.Flags().IntVar(&digDepth, "depth", 8, "maximum hops back through history")
	rootCmd.AddCommand(digCmd)
}
