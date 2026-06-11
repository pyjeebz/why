package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pyjeebz/why/internal/dig"
	"github.com/pyjeebz/why/internal/githist"
	"github.com/pyjeebz/why/internal/notes"
	"github.com/pyjeebz/why/internal/render"
	"github.com/pyjeebz/why/internal/target"
)

var (
	summarizeDepth int
	summarizeSave  bool
)

const summarizePrompt = `You are explaining why a region of code is the way it is.
Below is its decision trail: commits, pull requests, and issues, newest first.
Write a concise note (3-6 sentences) capturing the why — the constraints,
incidents, and decisions behind the current shape. Plain text only, no
preamble, written as if it will sit next to the code.

`

var summarizeCmd = &cobra.Command{
	Use:   "summarize FILE[:LINE | :START-END]",
	Short: "Draft a why-note from the trail using an LLM",
	Long: `Runs dig, then pipes the trail through an LLM to draft the reason in
prose. Uses the claude CLI by default; set WHY_LLM to any shell command
that reads a prompt on stdin and writes the answer to stdout.`,
	Example: `  why summarize internal/retry.go:142
  why summarize config/deploy.yaml:20-35 --save
  WHY_LLM="ollama run llama3" why summarize Makefile`,
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
		tr, err := dig.Run(cwd, t, summarizeDepth)
		if err != nil {
			return err
		}

		var trailMD bytes.Buffer
		render.Markdown(&trailMD, tr)

		llm := os.Getenv("WHY_LLM")
		if llm == "" {
			llm = "claude -p"
		}
		c := exec.Command("sh", "-c", llm)
		c.Stdin = strings.NewReader(summarizePrompt + trailMD.String())
		c.Stderr = os.Stderr
		var out bytes.Buffer
		c.Stdout = &out
		if err := c.Run(); err != nil {
			return fmt.Errorf("LLM command %q failed: %w (set WHY_LLM to a command that reads a prompt on stdin)", llm, err)
		}
		text := strings.TrimSpace(out.String())
		fmt.Println(text)

		if summarizeSave {
			key, err := dig.RepoKey(cwd)
			if err != nil {
				return err
			}
			n, err := notes.Save(key, notes.Note{
				Target:    t,
				Text:      text,
				Author:    githist.UserName(cwd),
				Source:    "inferred",
				AnchorSHA: githist.Head(cwd),
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "✎ saved as inferred note %s\n", n.ID)
		}
		return nil
	},
}

func init() {
	summarizeCmd.Flags().IntVar(&summarizeDepth, "depth", 8, "maximum hops back through history")
	summarizeCmd.Flags().BoolVar(&summarizeSave, "save", false, "save the draft as an inferred note in your overlay")
	rootCmd.AddCommand(summarizeCmd)
}
