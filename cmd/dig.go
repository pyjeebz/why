package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/pyjeebz/why/internal/answer"
	"github.com/pyjeebz/why/internal/dig"
	"github.com/pyjeebz/why/internal/render"
	"github.com/pyjeebz/why/internal/target"
)

var (
	digDepth int
	digJSON  bool
	digShare bool
	digTrail bool
	digTop   int
)

var digCmd = &cobra.Command{
	Use:   "dig FILE[:LINE | :START-END]",
	Short: "Dig up the decision trail behind a region of code",
	Example: `  why dig internal/retry.go:142
  why dig config/deploy.yaml:20-35
  why dig Makefile`,
	Args: cobra.ExactArgs(1),
	RunE: digRun,
}

func digRun(cmd *cobra.Command, args []string) error {
	t, err := target.Parse(args[0])
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	color := term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == ""

	// A whole-file target is a survey: which regions of this file are
	// worth knowing about, ranked. A line range is a single dig.
	if t.WholeFile() {
		regions, total, err := dig.Survey(cwd, t.Path, digTop, digDepth, !digTrail)
		if err != nil {
			return err
		}
		if digJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{"path": t.Path, "total": total, "regions": regions})
		}
		render.Survey(os.Stdout, t.Path, regions, total, color)
		return nil
	}

	tr, err := dig.Run(cwd, t, digDepth)
	if err != nil {
		return err
	}

	if !digTrail {
		tr.Answer = answer.For(tr, false)
	}

	switch {
	case digJSON:
		return render.JSON(os.Stdout, tr)
	case digShare:
		render.Markdown(os.Stdout, tr)
	default:
		render.Term(os.Stdout, tr, color)
	}
	return nil
}

func addDigFlags(c *cobra.Command) {
	c.Flags().IntVar(&digDepth, "depth", 8, "maximum hops back through history")
	c.Flags().BoolVar(&digJSON, "json", false, "emit the trail as JSON")
	c.Flags().BoolVar(&digShare, "share", false, "emit the trail as markdown for a PR or issue comment")
	c.Flags().BoolVar(&digTrail, "trail", false, "show the receipts only, without the one-line answer")
	c.Flags().IntVar(&digTop, "top", 3, "for a whole-file survey, how many regions to surface")
	c.MarkFlagsMutuallyExclusive("json", "share")
}

func init() {
	addDigFlags(digCmd)
	rootCmd.AddCommand(digCmd)

	// A bare `why FILE[:LINE]` is a dig — no subcommand needed.
	addDigFlags(rootCmd)
	rootCmd.Args = cobra.MaximumNArgs(1)
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return digRun(cmd, args)
	}
}
