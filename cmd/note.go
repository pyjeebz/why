package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/pyjeebz/why/internal/dig"
	"github.com/pyjeebz/why/internal/githist"
	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/target"
)

var (
	noteMessage  string
	noteReviewBy string
)

var noteCmd = &cobra.Command{
	Use:   "note FILE[:LINE | :START-END]",
	Short: "Attach a personal note explaining why a region is the way it is",
	Long: `Notes live in your personal overlay store (XDG data dir), not in the
repository, so you can annotate code you contribute to but don't own.
They surface at the top of later digs of the same region.`,
	Example: `  why note internal/retry.go:142 -m "Jitter avoids thundering herd on cold start"
  why note config/deploy.yaml:20-35 -m "Pinned for the v3 migration" --review-by 2026-12-01`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := target.Parse(args[0])
		if err != nil {
			return err
		}
		if noteReviewBy != "" {
			if _, err := time.Parse("2006-01-02", noteReviewBy); err != nil {
				return fmt.Errorf("--review-by must be a YYYY-MM-DD date")
			}
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		key, err := dig.RepoKey(cwd)
		if err != nil {
			return err
		}

		n, err := notes.Save(key, notes.Note{
			Target:    t,
			Text:      noteMessage,
			Author:    githist.UserName(cwd),
			ReviewBy:  noteReviewBy,
			Source:    "declared",
			AnchorSHA: githist.Head(cwd),
		})
		if err != nil {
			return err
		}
		fmt.Printf("✎ noted %s (%s)\n", t.String(), n.ID)
		return nil
	},
}

func init() {
	noteCmd.Flags().StringVarP(&noteMessage, "message", "m", "", "the note text (required)")
	noteCmd.Flags().StringVar(&noteReviewBy, "review-by", "", "date (YYYY-MM-DD) after which the note needs review")
	_ = noteCmd.MarkFlagRequired("message")
	rootCmd.AddCommand(noteCmd)
}
