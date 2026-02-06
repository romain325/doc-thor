package cmd

import "github.com/spf13/cobra"

var integrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "Manage VCS integrations",
	Long:  `Manage VCS integrations (GitLab, GitHub, Gitea) for webhook and discovery.`,
}

func init() {
	rootCmd.AddCommand(integrationCmd)
}
