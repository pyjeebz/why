package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/pyjeebz/why/internal/dig"
	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/render"
	"github.com/pyjeebz/why/internal/target"
)

var logJSON bool

var logCmd = &cobra.Command{
	Use:   "log [FILE[:LINE | :START-END]]",
	Short: "List your notes for this repository",
	Example: `  why log
  why log internal/retry.go
  why log internal/retry.go:120-160 --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		key, err := dig.RepoKey(cwd)
		if err != nil {
			return err
		}

		var ns []notes.Note
		if len(args) == 1 {
			t, perr := target.Parse(args[0])
			if perr != nil {
				return perr
			}
			ns, err = notes.For(key, t)
		} else {
			ns, err = notes.All(key)
		}
		if err != nil {
			return err
		}

		if logJSON {
			if ns == nil {
				ns = []notes.Note{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(ns)
		}
		if len(ns) == 0 {
			fmt.Println(`no notes yet — add one with: why note FILE:LINE -m "..."`)
			return nil
		}
		color := term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == ""
		render.Notes(os.Stdout, ns, color)
		return nil
	},
}

func init() {
	logCmd.Flags().BoolVar(&logJSON, "json", false, "emit the notes as JSON")
	rootCmd.AddCommand(logCmd)
}
