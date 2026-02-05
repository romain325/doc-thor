package cmd

import "github.com/spf13/cobra"

var apikeyCmd = &cobra.Command{
	Use:   "apikey",
	Short: "Manage API keys",
}

func init() {
	authCmd.AddCommand(apikeyCmd)
}
