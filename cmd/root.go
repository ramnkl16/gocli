// Package cmd wires up the cobra command tree for gocli.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	verboseFlag bool

	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

// SetBuildInfo lets main.go inject ldflags-baked version metadata.
func SetBuildInfo(version, commit, date string) {
	buildVersion, buildCommit, buildDate = version, commit, date
}

var rootCmd = &cobra.Command{
	Use:   "gocli",
	Short: "One CLI for Jira, GitHub, Copilot PR review, deployments, and Teams.",
	Long: `gocli is a unified developer cockpit.

It bundles Jira issue tracking, GitHub PR / issue work, AI-assisted PR
reviews via GitHub Copilot / GitHub Models, pluggable deployment
pipelines (scripts, Docker, Kubernetes), and optional Microsoft Teams
webhook notifications into one fast static binary so you stop context-switching between tools.

Run ` + "`gocli auth login`" + ` to get started.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the entry point called from main.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(jiraCmd)
	rootCmd.AddCommand(ghCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(teamsCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gocli %s (commit %s, built %s)\n", buildVersion, buildCommit, buildDate)
	},
}
