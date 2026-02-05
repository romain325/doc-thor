package cmd

import "github.com/spf13/cobra"

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server connection commands",
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
