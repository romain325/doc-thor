package cmd

import (
	"fmt"

	"github.com/romain325/doc-thor/cli/internal/client"
	"github.com/romain325/doc-thor/cli/internal/ui"
	"github.com/spf13/cobra"
)

var apikeyLabel string

var apikeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := c.CreateAPIKey(client.APIKeyCreateRequest{Label: apikeyLabel})
		if err != nil {
			return err
		}
		if ui.JSON {
			return ui.PrintJSON(key)
		}
		ui.Success("API key created.")
		fmt.Println(ui.WarningStyle.Render("This key is shown only once — store it safely."))
		ui.DetailCard("API Key", [][]string{
			{"Label", key.Label},
			{"Key", key.Key},
		})
		return nil
	},
}

func init() {
	apikeyCmd.AddCommand(apikeyCreateCmd)
	apikeyCreateCmd.Flags().StringVar(&apikeyLabel, "label", "", "human-readable label")
}
