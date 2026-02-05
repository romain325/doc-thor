package cmd

import "github.com/spf13/cobra"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Manage published versions",
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
