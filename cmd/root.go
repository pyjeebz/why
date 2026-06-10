// Package cmd wires up the why CLI.
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "why",
	Short: "Recover the reasons behind code",
	Long: `why answers "why is this code the way it is?" by walking the history
of a file region and assembling the decision trail behind it: commits,
the pull requests that introduced them, and the issues behind those.

Read-only, zero setup. Works on any repo you have cloned.`,
	SilenceUsage: true,
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}
