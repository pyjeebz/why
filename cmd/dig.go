package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/pyjeebz/why/internal/dig"
	"github.com/pyjeebz/why/internal/render"
	"github.com/pyjeebz/why/internal/target"
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

		tr, err := dig.Run(cwd, t, digDepth)
		if err != nil {
			return err
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
