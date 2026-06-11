package cmd

import (
	"bytes"
	"os"

	"github.com/spf13/cobra"

	"github.com/pyjeebz/why/internal/dig"
	"github.com/pyjeebz/why/internal/mcpserver"
	"github.com/pyjeebz/why/internal/render"
	"github.com/pyjeebz/why/internal/target"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Serve why over MCP (stdio) so agents can dig before they change code",
	Long: `Exposes one tool, why_dig, over the Model Context Protocol. Point your
agent at it and it can recover the trail behind any region before
touching it. Example .mcp.json entry:

  {"mcpServers": {"why": {"command": "why", "args": ["mcp"]}}}`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		return mcpserver.Serve(os.Stdin, os.Stdout, func(path string, start, end, depth int) (string, error) {
			t := target.Target{Path: path, Start: start, End: end}
			if err := target.Check(t); err != nil {
				return "", err
			}
			tr, err := dig.Run(cwd, t, depth)
			if err != nil {
				return "", err
			}
			var buf bytes.Buffer
			render.Markdown(&buf, tr)
			return buf.String(), nil
		})
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
